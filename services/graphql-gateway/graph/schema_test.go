package graph

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/transport"
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
		if (r.URL.Path != "/admin/events/event-id/registrations" && r.URL.Path != "/admin/registrations") || r.URL.Query().Get("limit") != "25" {
			t.Errorf("unexpected booking path: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"bookingId":"booking-id","accountId":"member-id","eventId":"event-id","state":"CONFIRMED","entryEligible":false,"paymentMethod":"AT_VENUE","amount":"2500.00","currency":"LKR","createdAt":"2026-09-01T12:00:00Z"}],"hasMore":false}`))
	}))
	defer booking.Close()
	client := upstream.New(upstream.Services{Account: account.URL, Booking: booking.URL})
	server := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{Upstream: client}}))
	call := func(query string) map[string]any {
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
	queries := []string{
		`{"query":"query { eventRegistrations(eventId: \"event-id\") { items { nickname email state entryEligible paymentMethod } hasMore } }"}`,
		`{"query":"query { adminRegistrations { items { nickname email state entryEligible paymentMethod eventId } hasMore } }"}`,
	}
	for _, query := range queries {
		adminResult := call(query)
		if adminResult["errors"] != nil || !bytes.Contains([]byte(toJSON(adminResult)), []byte(`"nickname":"Asha"`)) || !bytes.Contains([]byte(toJSON(adminResult)), []byte(`"entryEligible":false`)) {
			t.Fatalf("admin result: %v", adminResult)
		}
	}
	member = true
	for _, query := range queries {
		memberResult := call(query)
		if memberResult["errors"] == nil || bookingCalls != 2 {
			t.Fatalf("member was not denied before booking access: %v, calls %d", memberResult, bookingCalls)
		}
	}
}

func toJSON(value any) string { body, _ := json.Marshal(value); return string(body) }

func TestVenueStaffMutationsRequireAdmin(t *testing.T) {
	role:="member"
	account:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Header().Set("Content-Type","application/json");_,_=w.Write([]byte(`{"account":{"id":"actor","email":"a@example.test","status":"ACTIVE","verification":"VERIFIED","roles":["`+role+`"]}}`))}))
	defer account.Close()
	calls:=0
	booking:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){calls++;if r.Header.Get("Idempotency-Key")!="key-12345678"{t.Errorf("missing idempotency key")};w.Header().Set("Content-Type","application/json");_,_=w.Write([]byte(`{"bookingId":"booking-id","eventId":"event-id","state":"CONFIRMED","paymentMethod":"AT_VENUE","amount":"2500.00","currency":"LKR","policyVersion":1,"expiresAt":"2026-11-22T12:00:00Z","version":2,"createdAt":"2026-11-22T10:00:00Z"}`))}))
	defer booking.Close()
	server:=handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers:&Resolver{Upstream:upstream.New(upstream.Services{Account:account.URL,Booking:booking.URL})}}))
	query:=`{"query":"mutation { collectVenuePayment(bookingId: \"booking-id\", method: \"CASH\", idempotencyKey: \"key-12345678\") { bookingId } }"}`
	call:=func()[]byte{w:=httptest.NewRecorder();r:=httptest.NewRequest(http.MethodPost,"/graphql",bytes.NewBufferString(query));r.Header.Set("Content-Type","application/json");server.ServeHTTP(w,r);return w.Body.Bytes()}
	if result:=call();!bytes.Contains(result,[]byte(`"errors"`))||calls!=0{t.Fatalf("member reached staff command: %s",result)}
	role="admin"
	if result:=call();bytes.Contains(result,[]byte(`"errors"`))||calls!=1{t.Fatalf("admin command failed: %s calls=%d",result,calls)}
}

func TestAppealOnlyLoginAndOwnerAppealThroughGateway(t *testing.T) {
	meCalls := 0
	account := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/login", "/auth/refresh":
			_, _ = w.Write([]byte(`{"accessToken":"appeal-token","roles":["appeal"]}`))
		case "/users/me":
			meCalls++
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Errorf("unexpected account path: %s", r.URL.Path)
		}
	}))
	defer account.Close()
	moderationCalls := 0
	moderation := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		moderationCalls++
		if r.Header.Get("Authorization") != "Bearer appeal-token" {
			t.Errorf("appeal authorization was not forwarded")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/moderation/actions/mine":
			_, _ = w.Write([]byte(`{"items":[{"actionId":"action-id","actionClass":"ACCOUNT_SUSPENSION","scope":"ALL","effectiveAt":"2026-11-21T00:00:00Z"}]}`))
		case "/moderation/actions/action-id/appeals":
			if r.Method != http.MethodPost {
				t.Errorf("appeal method: %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"appealId":"appeal-id","actionId":"action-id","state":"OPEN"}`))
		default:
			t.Errorf("unexpected moderation path: %s", r.URL.Path)
		}
	}))
	defer moderation.Close()
	client := upstream.New(upstream.Services{Account: account.URL, Moderation: moderation.URL})
	server := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{Upstream: client}}))
	call := func(query, token string) []byte {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(query))
		request.Header.Set("Content-Type", "application/json")
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		server.ServeHTTP(recorder, request.WithContext(transport.WithHTTP(request.Context(), request, recorder)))
		return recorder.Body.Bytes()
	}
	for _, query := range []string{
		`{"query":"mutation { login(email: \"a@example.test\", password: \"test\") { accessToken roles me { account { id } } } }"}`,
		`{"query":"mutation { refreshSession { accessToken roles me { account { id } } } }"}`,
	} {
		result := call(query, "")
		if bytes.Contains(result, []byte(`"errors"`)) || !bytes.Contains(result, []byte(`"roles":["appeal"]`)) || !bytes.Contains(result, []byte(`"me":null`)) {
			t.Fatalf("restricted authentication failed: %s", result)
		}
	}
	if meCalls != 0 {
		t.Fatalf("restricted login fetched normal profile %d times", meCalls)
	}
	list := call(`{"query":"query { myModerationActions { items { actionId actionClass } } }"}`, "appeal-token")
	if bytes.Contains(list, []byte(`"errors"`)) || !bytes.Contains(list, []byte(`"actionId":"action-id"`)) {
		t.Fatalf("owner action list failed: %s", list)
	}
	appeal := call(`{"query":"mutation { appealModerationAction(actionId: \"action-id\", reason: \"Please review this action\") { appealId state } }"}`, "appeal-token")
	if bytes.Contains(appeal, []byte(`"errors"`)) || !bytes.Contains(appeal, []byte(`"appealId":"appeal-id"`)) {
		t.Fatalf("owner appeal failed: %s", appeal)
	}
	if moderationCalls != 2 {
		t.Fatalf("expected only owner calls, got %d", moderationCalls)
	}
	me := call(`{"query":"query { me { account { id } } }"}`, "appeal-token")
	if !bytes.Contains(me, []byte(`"errors"`)) || meCalls != 1 {
		t.Fatalf("ordinary member route was not denied: %s", me)
	}
}

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

func TestArchiveEventRequiresAdminAndForwardsVersion(t *testing.T) {
	member := false
	account := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := "admin"
		if member {
			role = "member"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"account":{"id":"actor","email":"admin@example.test","status":"ACTIVE","verification":"VERIFIED","roles":["` + role + `"]}}`))
	}))
	defer account.Close()
	archiveCalls := 0
	events := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		archiveCalls++
		if r.URL.Path != "/events/event-id/archive" || r.Method != http.MethodPost {
			t.Errorf("unexpected event request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			ExpectedVersion int `json:"expectedVersion"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ExpectedVersion != 3 {
			t.Errorf("unexpected archive body: %+v %v", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"eventId":"event-id","version":4,"archivedAt":"2026-09-01T12:00:00Z"}`))
	}))
	defer events.Close()
	server := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{Upstream: upstream.New(upstream.Services{Account: account.URL, Event: events.URL})}}))
	call := func() []byte {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString(`{"query":"mutation { archiveEvent(eventId: \"event-id\", expectedVersion: 3) { eventId version archivedAt } }"}`))
		request.Header.Set("Content-Type", "application/json")
		server.ServeHTTP(recorder, request)
		return recorder.Body.Bytes()
	}
	if response := call(); bytes.Contains(response, []byte(`"errors"`)) || !bytes.Contains(response, []byte(`"archivedAt"`)) {
		t.Fatalf("admin archive failed: %s", response)
	}
	member = true
	if response := call(); !bytes.Contains(response, []byte(`"errors"`)) || archiveCalls != 1 {
		t.Fatalf("member archive was not denied: %s calls=%d", response, archiveCalls)
	}
}
