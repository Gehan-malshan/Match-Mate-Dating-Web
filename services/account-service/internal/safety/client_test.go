package safety

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSafetyClientFailsClosedOnMissingFieldsAndUnauthorized(t *testing.T) {
	body := `{"suspended":true,"profileHidden":true,"eventExcluded":false,"matchmakingExcluded":true,"revealPrevented":true}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Moderation-Service-Token") != "test-token" {
			w.WriteHeader(401)
			return
		}
		if strings.Contains(r.URL.RawQuery, "eventId=bad") {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	decision, err := New(server.URL, "test-token").Check(context.Background(), "account", "event")
	if err != nil || !decision.Suspended || !decision.ProfileHidden || !decision.MatchmakingExcluded {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	if _, err = New(server.URL, "wrong").Check(context.Background(), "account", "event"); err == nil {
		t.Fatal("unauthorized safety response was accepted")
	}
	if _, err = New(server.URL, "test-token").Check(context.Background(), "account", "bad"); err == nil {
		t.Fatal("missing safety fields were treated as safe")
	}
}
