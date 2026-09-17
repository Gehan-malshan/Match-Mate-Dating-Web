package upstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/transport"
)

func TestDoForwardsIdentityAndRewritesRefreshCookiePath(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" || r.Header.Get("Origin") != "http://localhost:5173" || r.Header.Get("Cookie") != "matchmate_refresh=refresh" {
			t.Fatal("identity headers were not forwarded")
		}
		http.SetCookie(w, &http.Cookie{Name: "matchmate_refresh", Value: "next", Path: "/api/v1/auth", HttpOnly: true})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer service.Close()

	incoming := httptest.NewRequest(http.MethodPost, "http://gateway/graphql", nil)
	incoming.Header.Set("Authorization", "Bearer token")
	incoming.Header.Set("Origin", "http://localhost:5173")
	incoming.Header.Set("Cookie", "matchmate_refresh=refresh")
	recorder := httptest.NewRecorder()
	ctx := transport.WithHTTP(context.Background(), incoming, recorder)
	var result map[string]bool
	if err := New(Services{}).Do(ctx, service.URL, "", http.MethodGet, nil, &result, nil); err != nil {
		t.Fatal(err)
	}
	if cookie := recorder.Header().Get("Set-Cookie"); cookie == "" || !contains(cookie, "Path=/") || contains(cookie, "Path=/api/v1/auth") {
		t.Fatalf("unexpected rewritten cookie: %s", cookie)
	}
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
