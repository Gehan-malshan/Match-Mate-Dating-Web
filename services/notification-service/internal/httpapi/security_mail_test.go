package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
)

type securitySender struct{ recipient, body string }

func (s *securitySender) Send(_ context.Context, _ domain.Delivery, msg domain.RenderedMessage, recipient string) (string, error) {
	s.recipient = recipient
	s.body = msg.Body
	return "test", nil
}

func TestSecurityMailRequiresServiceTokenAndUsesOnlyApprovedTemplate(t *testing.T) {
	sender := &securitySender{}
	handler := SecurityMail(http.NotFoundHandler(), sender, "internal-only", "https://app.example.test")
	request := func(auth, purpose string) *httptest.ResponseRecorder {
		body := `{"email":"member@example.test","purpose":"` + purpose + `","token":"` + strings.Repeat("a", 43) + `"}`
		r := httptest.NewRequest(http.MethodPost, "/api/v1/internal/security-email", strings.NewReader(body))
		if auth != "" {
			r.Header.Set("X-MatchMate-Internal-Token", auth)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := request("", "reset"); w.Code != http.StatusUnauthorized {
		t.Fatal("anonymous delivery accepted")
	}
	if w := request("internal-only", "other"); w.Code != http.StatusBadRequest {
		t.Fatal("unapproved template accepted")
	}
	if w := request("internal-only", "reset"); w.Code != http.StatusNoContent {
		body, _ := io.ReadAll(w.Body)
		t.Fatalf("delivery failed %d %s", w.Code, body)
	}
	if sender.recipient != "member@example.test" || !strings.Contains(sender.body, "https://app.example.test/reset-password?token=") {
		t.Fatal("recovery message incorrect")
	}
}
