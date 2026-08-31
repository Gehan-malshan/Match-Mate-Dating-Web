package graph

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/upstream"
)

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
