package graph

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/upstream"
)

func TestEventRegistrationsJoinAndRoleGuard(t *testing.T) {
	bookingCalls := 0
	member := false
	account := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/me":
			role := "admin"
			if member {
				role = "member"
			}
			_, _ = w.Write([]byte(`{"account":{"id":"admin-id","email":"admin@example.test","status":"ACTIVE","verification":"VERIFIED","roles":["` + role + `"]}}`))
		case "/admin/member-identities":
			_, _ = w.Write([]byte(`{"items":[{"accountId":"member-id","nickname":"Asha","email":"asha@example.test"}]}`))
		default:
			t.Errorf("unexpected account path: %s", r.URL.Path)
		}
	}))
	defer account.Close()
	booking := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bookingCalls++
		if r.URL.Path != "/admin/events/event-id/registrations" || r.URL.Query().Get("limit") != "25" {
			t.Errorf("unexpected booking path: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"bookingId":"booking-id","accountId":"member-id","eventId":"event-id","state":"CONFIRMED","paymentMethod":"AT_VENUE","amount":"2500.00","currency":"LKR","createdAt":"2026-09-01T12:00:00Z"}],"hasMore":false}`))
	}))
	defer booking.Close()
	client := upstream.New(upstream.Services{Account: account.URL, Booking: booking.URL})
	server := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{Upstream: client}}))
	query := `{"query":"query { eventRegistrations(eventId: \"event-id\") { items { nickname email state paymentMethod } hasMore } }"}`
	call := func() map[string]any {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(query))
		request.Header.Set("Content-Type", "application/json")
		server.ServeHTTP(recorder, request)
		var payload map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	adminResult := call()
	if adminResult["errors"] != nil || !bytes.Contains([]byte(toJSON(adminResult)), []byte(`"nickname":"Asha"`)) {
		t.Fatalf("admin result: %v", adminResult)
	}
	member = true
	memberResult := call()
	if memberResult["errors"] == nil || bookingCalls != 1 {
		t.Fatalf("member was not denied before booking access: %v, calls %d", memberResult, bookingCalls)
	}
}

func toJSON(value any) string { body, _ := json.Marshal(value); return string(body) }

func TestPublicEventsQueryUsesEventService(t *testing.T) {
	events := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" || r.URL.Query().Get("limit") != "12" {
			t.Fatalf("unexpected event request: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"limit":12}`))
	}))
	defer events.Close()

	client := upstream.New(upstream.Services{Event: events.URL})
	server := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{Upstream: client}}))
	request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(`{"query":"query { events(limit: 12) { limit items { eventId } } }"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || bytes.Contains(recorder.Body.Bytes(), []byte(`"errors"`)) {
		t.Fatalf("unexpected GraphQL response: %d %s", recorder.Code, recorder.Body.String())
	}
}
