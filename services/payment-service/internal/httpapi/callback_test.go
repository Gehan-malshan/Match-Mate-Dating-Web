package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/payment-service/internal/application"
	"github.com/gehan-malshan/matchmate/payment-service/internal/domain"
	"github.com/gehan-malshan/matchmate/payment-service/internal/payhere"
)

type testVerifier struct{}

func (testVerifier) Verify(string) (domain.Principal, error) { return domain.Principal{}, nil }

type adminVerifier struct{ roles []string }

func (v adminVerifier) Verify(string) (domain.Principal, error) {
	return domain.Principal{Subject: "account-1", Roles: v.roles}, nil
}

type testRepository struct{ applied int }

func (*testRepository) Ping(context.Context) error { return nil }

func (*testRepository) Initiate(context.Context, domain.Payment, string, string) (domain.Payment, bool, error) {
	return domain.Payment{}, false, nil
}
func (*testRepository) FindByBooking(context.Context, string, string) (domain.Payment, error) {
	return domain.Payment{}, nil
}
func (r *testRepository) ApplyCallback(context.Context, payhere.Notification, string, time.Time) (domain.Payment, bool, error) {
	r.applied++
	return domain.Payment{}, false, nil
}
func (*testRepository) ListReview(context.Context, string, int, string) (application.ReviewPage, error) {
	return application.ReviewPage{Items: []application.ReviewItem{}}, nil
}

func TestCallbackRejectsInvalidOrAmbiguousForm(t *testing.T) {
	repo := &testRepository{}
	cfg := payhere.Config{MerchantID: "123", MerchantSecret: "secret"}
	h := New(application.New(repo, nil, cfg), testVerifier{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	n := payhere.Notification{MerchantID: "123", OrderID: "order-1", PaymentID: "p-1", Amount: "1000.00", Currency: "LKR", StatusCode: "2"}
	n.Signature = payhere.NotificationHash(n, cfg.MerchantSecret)
	form := url.Values{"merchant_id": {n.MerchantID}, "order_id": {n.OrderID}, "payment_id": {n.PaymentID}, "payhere_amount": {n.Amount}, "payhere_currency": {n.Currency}, "status_code": {n.StatusCode}, "md5sig": {n.Signature}}
	tests := []struct {
		name        string
		form        url.Values
		contentType string
		want        int
	}{
		{"valid", form, "application/x-www-form-urlencoded; charset=UTF-8", 200},
		{"invalid signature", cloneWith(form, "md5sig", "00000000000000000000000000000000"), "application/x-www-form-urlencoded", 400},
		{"duplicate status", cloneWith(form, "status_code", "2", "-2"), "application/x-www-form-urlencoded", 400},
		{"missing payment id", cloneWith(form, "payment_id"), "application/x-www-form-urlencoded", 400},
		{"wrong content type", form, "application/json", 415},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/payhere/callback", strings.NewReader(tc.form.Encode()))
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("got %d want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
	if repo.applied != 1 {
		t.Fatalf("expected only valid callback to reach repository, got %d", repo.applied)
	}
}

func cloneWith(in url.Values, key string, values ...string) url.Values {
	out := url.Values{}
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	if len(values) == 0 {
		out.Del(key)
	} else {
		out[key] = values
	}
	return out
}

func TestReconciliationRequiresAdmin(t *testing.T) {
	repo := &testRepository{}
	svc := application.New(repo, nil, payhere.Config{})
	for _, tc := range []struct {
		roles []string
		limit string
		want  int
	}{
		{[]string{"member"}, "", 403},
		{[]string{"admin"}, "101", 400},
		{[]string{"admin"}, "25", 200},
	} {
		h := New(svc, adminVerifier{roles: tc.roles}, slog.New(slog.NewTextHandler(io.Discard, nil)))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/payments/reconciliation?limit="+tc.limit, nil))
		if w.Code != tc.want {
			t.Fatalf("roles=%v limit=%q got=%d want=%d", tc.roles, tc.limit, w.Code, tc.want)
		}
	}
}
