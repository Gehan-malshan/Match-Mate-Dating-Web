package application

import (
	"context"
	"errors"
	"github.com/gehan-malshan/matchmate/booking-service/internal/domain"
	"github.com/gehan-malshan/matchmate/booking-service/internal/safety"
	"sync/atomic"
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
func (f *fakeRepo) ListRegistrations(context.Context, int, int) ([]domain.Booking, error) {
	return []domain.Booking{f.created}, nil
}
func (f *fakeRepo) Cancel(_ context.Context, _ string, _ string, _ string, at time.Time) (domain.Booking, bool, error) {
	f.created.State = domain.Cancelled
	f.created.CancelledAt = &at
	return f.created, false, nil
}
func (f *fakeRepo) Expire(context.Context, time.Time, int) (int, error) { return 0, nil }
func (f *fakeRepo) GetForStaff(context.Context, string) (domain.Booking, error) {
	return f.created, nil
}
func (f *fakeRepo) CollectVenue(_ context.Context, _, _, _, _ string, at time.Time) (domain.Booking, error) {
	f.created.VenueCollectedAt = &at
	return f.created, nil
}
func (f *fakeRepo) CheckIn(_ context.Context, _, _, _ string, at time.Time) (domain.Booking, error) {
	f.created.CheckedInAt = &at
	return f.created, nil
}
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

type safetyStub struct {
	excluded  atomic.Bool
	suspended atomic.Bool
	calls     atomic.Int32
	entered   chan struct{}
	release   chan struct{}
}

func (s *safetyStub) Check(_ context.Context, _, _ string) (safety.Decision, error) {
	if s.entered != nil && s.calls.Add(1) == 2 {
		s.entered <- struct{}{}
		<-s.release
	}
	return safety.Decision{EventExcluded: s.excluded.Load(), Suspended: s.suspended.Load()}, nil
}
func TestSuspendedActorCannotUseBookingRoutes(t *testing.T) {
	svc := New(&fakeRepo{}, fakeEvents{}, time.Minute)
	guard := &safetyStub{}
	svc.SetSafety(guard)
	guard.suspended.Store(true)
	if err := svc.CheckActor(context.Background(), "member"); !errors.Is(err, ErrExcluded) {
		t.Fatalf("suspended JWT holder was admitted: %v", err)
	}
	guard.suspended.Store(false)
	if err := svc.CheckActor(context.Background(), "member"); err != nil {
		t.Fatalf("reversed suspension still blocked actor: %v", err)
	}
}
func TestBookingSafetyRaceAndAppealReversal(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeRepo{}
	svc := New(repo, fakeEvents{domain.EventSnapshot{EventID: "event", Status: "REGISTRATION_OPEN", Price: "2500.00", Currency: "LKR", PaymentOptions: "AT_VENUE", ConfiguredCapacity: 2, RegistrationClosesAt: now.Add(time.Hour)}}, time.Minute)
	guard := &safetyStub{entered: make(chan struct{}, 1), release: make(chan struct{})}
	svc.SetSafety(guard)
	done := make(chan error, 1)
	go func() {
		_, err := svc.Create(context.Background(), "member", "event", "AT_VENUE", "first-key")
		done <- err
	}()
	<-guard.entered
	guard.excluded.Store(true) // Moderation acts after the first check, before the final admission check.
	close(guard.release)
	if err := <-done; !errors.Is(err, ErrExcluded) || repo.created.ID != "" {
		t.Fatalf("excluded booking was created: %v %+v", err, repo.created)
	}
	eligible, err := svc.EntryEligible(context.Background(), domain.Booking{AccountID: "member", EventID: "event", State: domain.Confirmed})
	if err != nil || eligible {
		t.Fatalf("excluded confirmed booking still admitted: eligible=%v err=%v", eligible, err)
	}
	guard.entered = nil
	guard.excluded.Store(false) // An approved appeal reverses the current decision.
	if _, err := svc.Create(context.Background(), "member", "event", "AT_VENUE", "second-key"); err != nil {
		t.Fatalf("reversed action still blocks booking: %v", err)
	}
	eligible, err = svc.EntryEligible(context.Background(), repo.created)
	if err != nil || !eligible {
		t.Fatalf("reversed exclusion still blocks entry: eligible=%v err=%v", eligible, err)
	}
}

func TestStaffActionsRespectCurrentEligibilityAndEventCancellation(t *testing.T) {
	repo := &fakeRepo{created: domain.Booking{ID: "booking", AccountID: "member", EventID: "event", State: domain.Confirmed, PaymentMethod: "AT_VENUE"}}
	events := fakeEvents{domain.EventSnapshot{EventID: "event", Status: "PUBLISHED"}}
	svc := New(repo, events, time.Minute)
	guard := &safetyStub{}
	svc.SetSafety(guard)
	guard.excluded.Store(true)
	if _, err := svc.CollectVenue(context.Background(), "admin", "booking", "collection-key", "CASH"); !errors.Is(err, ErrExcluded) || repo.created.VenueCollectedAt != nil {
		t.Fatalf("excluded member's collection accepted: %v", err)
	}
	if _, err := svc.CheckIn(context.Background(), "admin", "booking", "checkin-key"); !errors.Is(err, ErrExcluded) || repo.created.CheckedInAt != nil {
		t.Fatalf("excluded member checked in: %v", err)
	}
	guard.excluded.Store(false)
	if _, err := svc.CollectVenue(context.Background(), "admin", "booking", "collection-key", "CASH"); err != nil {
		t.Fatalf("appeal reversal still blocks collection: %v", err)
	}
	svc.events = fakeEvents{domain.EventSnapshot{EventID: "event", Status: "CANCELLED"}}
	if _, err := svc.CheckIn(context.Background(), "admin", "booking", "checkin-key"); !errors.Is(err, ErrConflict) || repo.created.CheckedInAt != nil {
		t.Fatalf("cancelled event admitted member: %v", err)
	}
}
