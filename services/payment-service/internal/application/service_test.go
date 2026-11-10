package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/payment-service/internal/domain"
	"github.com/gehan-malshan/matchmate/payment-service/internal/payhere"
)

type callbackRepo struct{ calls int }

func (*callbackRepo) Ping(context.Context) error { return nil }

func (*callbackRepo) Initiate(context.Context, domain.Payment, string, string) (domain.Payment, bool, error) {
	return domain.Payment{}, false, nil
}
func (*callbackRepo) FindByBooking(context.Context, string, string) (domain.Payment, error) {
	return domain.Payment{}, nil
}
func (r *callbackRepo) ApplyCallback(context.Context, payhere.Notification, string, time.Time) (domain.Payment, bool, error) {
	r.calls++
	return domain.Payment{}, false, nil
}
func (*callbackRepo) ListReview(context.Context, string, int, string) (ReviewPage, error) {
	return ReviewPage{}, nil
}

func TestInvalidCallbackCannotChangePaymentState(t *testing.T) {
	repo := &callbackRepo{}
	svc := New(repo, nil, payhere.Config{MerchantID: "123", MerchantSecret: "secret"})
	n := payhere.Notification{MerchantID: "123", OrderID: "order-1", PaymentID: "p-1", Amount: "1000.00", Currency: "LKR", StatusCode: "2", Signature: "invalid"}
	if err := svc.Callback(context.Background(), n); !errors.Is(err, ErrInvalidCallback) || repo.calls != 0 {
		t.Fatalf("untrusted callback reached repository: err=%v calls=%d", err, repo.calls)
	}
	n.Signature = payhere.NotificationHash(n, "secret")
	if err := svc.Callback(context.Background(), n); err != nil || repo.calls != 1 {
		t.Fatalf("valid callback was not applied: err=%v calls=%d", err, repo.calls)
	}
}
