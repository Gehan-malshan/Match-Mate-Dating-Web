package application

import (
	"context"
	"errors"
	"github.com/gehan-malshan/matchmate/booking-service/internal/domain"
	"testing"
	"time"
)

type fakeEvents struct{ event domain.EventSnapshot }

func (f fakeEvents) Get(context.Context, string) (domain.EventSnapshot, error) { return f.event, nil }

type fakeRepo struct {
	created  domain.Booking
	capacity int
}

func (f *fakeRepo) Create(_ context.Context, b domain.Booking, _, _ string, capacity int) (domain.Booking, bool, error) {
	f.created = b
	f.capacity = capacity
	return b, false, nil
}
func (f *fakeRepo) Get(context.Context, string, string) (domain.Booking, error) {
	return f.created, nil
}
func (f *fakeRepo) List(context.Context, string, int) ([]domain.Booking, error) {
	return []domain.Booking{f.created}, nil
}
func (f *fakeRepo) ListForEvent(context.Context, string, int, int) ([]domain.Booking, error) {
	return []domain.Booking{f.created}, nil
}
func (f *fakeRepo) Cancel(_ context.Context, _ string, _ string, _ string, at time.Time) (domain.Booking, bool, error) {
	f.created.State = domain.Cancelled
	f.created.CancelledAt = &at
	return f.created, false, nil
}
func (f *fakeRepo) Expire(context.Context, time.Time, int) (int, error) { return 0, nil }
func TestCreateUsesEventOwnedPrice(t *testing.T) {
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepo{}
	svc := New(repo, fakeEvents{domain.EventSnapshot{EventID: "9fc263e0-3972-47c0-bf79-d9fa5e9d1201", Status: "REGISTRATION_OPEN", Price: "4500.00", Currency: "LKR", ConfiguredCapacity: 25, CapacityPolicyVersion: 3, RegistrationClosesAt: now.Add(time.Hour)}}, 15*time.Minute)
	svc.now = func() time.Time { return now }
	b, err := svc.Create(context.Background(), "30e5dc52-29ea-4312-85fb-9bc519a3e92f", "event", "ONLINE", "key-12345678")
	if err != nil {
		t.Fatal(err)
	}
	if b.Amount != "4500.00" || b.Currency != "LKR" || repo.capacity != 25 {
		t.Fatalf("unexpected snapshot: %+v", b)
	}
	if !b.ExpiresAt.Equal(now.Add(15 * time.Minute)) {
		t.Fatal("wrong expiry")
	}
}

func TestPayAtVenueConfirmsWithoutHold(t *testing.T) {
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepo{}
	svc := New(repo, fakeEvents{domain.EventSnapshot{EventID: "event", Status: "REGISTRATION_OPEN", Price: "2500.00", Currency: "LKR", PaymentOptions: "BOTH", ConfiguredCapacity: 2, RegistrationClosesAt: now.Add(time.Hour)}}, 15*time.Minute)
	svc.now = func() time.Time { return now }
	b, err := svc.Create(context.Background(), "member", "event", "AT_VENUE", "key-venue")
	if err != nil || b.State != domain.Confirmed || b.PaymentMethod != "AT_VENUE" || b.ConfirmedAt == nil {
		t.Fatalf("unexpected venue reservation: %+v, %v", b, err)
	}
	if _, err = svc.Snapshot(context.Background(), "member", b.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("venue booking must not start PayHere: %v", err)
	}
}

func TestDisallowedPaymentMethod(t *testing.T) {
	now := time.Now().UTC()
	svc := New(&fakeRepo{}, fakeEvents{domain.EventSnapshot{EventID: "event", Status: "REGISTRATION_OPEN", Price: "2500.00", Currency: "LKR", PaymentOptions: "AT_VENUE", ConfiguredCapacity: 2, RegistrationClosesAt: now.Add(time.Hour)}}, 15*time.Minute)
	if _, err := svc.Create(context.Background(), "member", "event", "ONLINE", "key"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("online must be forbidden: %v", err)
	}
}

func TestCancelRequiresIdempotencyKey(t *testing.T) {
	svc := New(&fakeRepo{}, fakeEvents{}, 15*time.Minute)
	if _, err := svc.Cancel(context.Background(), "member", "booking", ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}
