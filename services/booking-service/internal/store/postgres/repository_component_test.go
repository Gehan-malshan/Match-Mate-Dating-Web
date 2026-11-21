package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/booking-service/internal/application"
	"github.com/gehan-malshan/matchmate/booking-service/internal/domain"
	storepg "github.com/gehan-malshan/matchmate/booking-service/internal/store/postgres"
	"github.com/gehan-malshan/matchmate/booking-service/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestVenueCapacityAndIdempotency(t *testing.T) {
	url := os.Getenv("BOOKING_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set BOOKING_TEST_DATABASE_URL for PostgreSQL component tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "booking_test_" + strings.ReplaceAll(uuid.NewString(), "-", "_")
	if _, err = admin.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA %s`, schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), fmt.Sprintf(`DROP SCHEMA %s CASCADE`, schema)) })
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, file := range []string{"000001_init.up.sql", "000002_cancellation.up.sql", "000003_payment_method.up.sql", "000004_event_registrations.up.sql", "000005_registration_feed.up.sql", "000006_venue_attendance.up.sql"} {
		body, e := migrations.Files.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = pool.Exec(ctx, string(body)); e != nil {
			t.Fatal(e)
		}
	}
	repo := storepg.New(pool)
	eventID := uuid.NewString()
	makeBooking := func(accountID, method string) domain.Booking {
		now := time.Now().UTC()
		b := domain.Booking{ID: uuid.NewString(), AccountID: accountID, EventID: eventID, State: domain.Confirmed, PaymentMethod: method, Amount: "2500.00", Currency: "LKR", PolicyVersion: 1, ExpiresAt: now.Add(15 * time.Minute), Version: 1, CreatedAt: now, ConfirmedAt: &now}
		return b
	}
	var wg sync.WaitGroup
	type createResult struct {
		booking domain.Booking
		err     error
	}
	results := make(chan createResult, 8)
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := makeBooking(uuid.NewString(), "AT_VENUE")
			<-start
			created, _, e := repo.Create(ctx, b, uuid.NewString(), eventID+":AT_VENUE", 1)
			results <- createResult{created, e}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	var first domain.Booking
	winners := 0
	for result := range results {
		if result.err == nil {
			first = result.booking
			winners++
		} else if !errors.Is(result.err, application.ErrCapacity) {
			t.Fatalf("unexpected final-seat result: %v", result.err)
		}
	}
	if winners != 1 || first.State != domain.Confirmed {
		t.Fatalf("expected exactly one confirmed final-seat booking, got %d: %+v", winners, first)
	}
	registrations, err := repo.ListForEvent(ctx, eventID, 25, 0)
	if err != nil || len(registrations) != 1 || registrations[0].AccountID != first.AccountID {
		t.Fatalf("event registrations: %+v %v", registrations, err)
	}
	allRegistrations, err := repo.ListRegistrations(ctx, 25, 0)
	if err != nil || len(allRegistrations) != 1 || allRegistrations[0].EventID != eventID {
		t.Fatalf("all registrations: %+v %v", allRegistrations, err)
	}
	var held, confirmed int
	if err = pool.QueryRow(ctx, `SELECT held_count,confirmed_count FROM capacity_allocation WHERE event_id=$1`, eventID).Scan(&held, &confirmed); err != nil || held != 0 || confirmed != 1 {
		t.Fatalf("allocation %d/%d: %v", held, confirmed, err)
	}
	var facts int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE event_type='BookingConfirmed'`).Scan(&facts); err != nil || facts != 1 {
		t.Fatalf("confirmed facts=%d: %v", facts, err)
	}
	staffID := uuid.NewString()
	if _, err = repo.CheckIn(ctx, staffID, first.ID, "early-checkin", time.Now().UTC()); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("unpaid venue booking checked in: %v", err)
	}
	collected, err := repo.CollectVenue(ctx, staffID, first.ID, "collect-1", "CASH", time.Now().UTC())
	if err != nil || collected.VenueCollectedAt == nil || collected.VenueCollectionMethod == nil || *collected.VenueCollectionMethod != "CASH" {
		t.Fatalf("collection failed: %+v %v", collected, err)
	}
	if _, err = repo.CollectVenue(ctx, staffID, first.ID, "collect-1", "CASH", time.Now().UTC()); err != nil {
		t.Fatalf("collection replay failed: %v", err)
	}
	if _, err = repo.CollectVenue(ctx, staffID, first.ID, "collect-2", "CASH", time.Now().UTC()); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("duplicate collection accepted: %v", err)
	}
	checked, err := repo.CheckIn(ctx, staffID, first.ID, "checkin-1", time.Now().UTC())
	if err != nil || checked.CheckedInAt == nil {
		t.Fatalf("check-in failed: %+v %v", checked, err)
	}
	if _, err = repo.CheckIn(ctx, staffID, first.ID, "checkin-1", time.Now().UTC()); err != nil {
		t.Fatalf("check-in replay failed: %v", err)
	}
	if _, _, err = repo.Cancel(ctx, first.AccountID, first.ID, "cancel-venue", time.Now().UTC()); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("confirmed venue cancellation accepted: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT confirmed_count FROM capacity_allocation WHERE event_id=$1`, eventID).Scan(&confirmed); err != nil || confirmed != 1 {
		t.Fatalf("capacity changed after collection/checkin: %d %v", confirmed, err)
	}
	onlineEvent := uuid.NewString()
	old := time.Now().UTC().Add(-time.Hour)
	pending := domain.Booking{ID: uuid.NewString(), AccountID: uuid.NewString(), EventID: onlineEvent, State: domain.Pending, PaymentMethod: "ONLINE", Amount: "2500.00", Currency: "LKR", PolicyVersion: 1, ExpiresAt: old.Add(time.Minute), Version: 1, CreatedAt: old}
	if _, _, err = repo.Create(ctx, pending, "online-key", onlineEvent+":ONLINE", 1); err != nil {
		t.Fatal(err)
	}
	if n, e := repo.Expire(ctx, time.Now().UTC(), 10); e != nil || n != 1 {
		t.Fatalf("expiry %d %v", n, e)
	}
	if n, e := repo.Expire(ctx, time.Now().UTC(), 10); e != nil || n != 0 {
		t.Fatalf("duplicate expiry %d %v", n, e)
	}
	paymentEventID := uuid.NewString()
	if err = repo.ApplyPaymentEvent(ctx, paymentEventID, "PaymentCompleted", uuid.NewString(), pending.ID, time.Now().UTC()); err != nil {
		t.Fatalf("late payment: %v", err)
	}
	if err = repo.ApplyPaymentEvent(ctx, paymentEventID, "PaymentCompleted", uuid.NewString(), pending.ID, time.Now().UTC()); err != nil {
		t.Fatalf("duplicate event: %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT held_count,confirmed_count FROM capacity_allocation WHERE event_id=$1`, onlineEvent).Scan(&held, &confirmed); err != nil || held != 0 || confirmed != 0 {
		t.Fatalf("expired capacity not recovered: %d/%d %v", held, confirmed, err)
	}
	var state string
	if err = pool.QueryRow(ctx, `SELECT state FROM booking WHERE id=$1`, pending.ID).Scan(&state); err != nil || state != "PAYMENT_REVIEW" {
		t.Fatalf("late payment state=%s err=%v", state, err)
	}
}
