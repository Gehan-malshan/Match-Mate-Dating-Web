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
	for _, file := range []string{"000001_init.up.sql", "000002_cancellation.up.sql", "000003_payment_method.up.sql", "000004_event_registrations.up.sql"} {
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
	first := makeBooking(uuid.NewString(), "AT_VENUE")
	created, replay, err := repo.Create(ctx, first, "venue-key", eventID+":AT_VENUE", 1)
	if err != nil || replay || created.State != domain.Confirmed {
		t.Fatalf("venue creation: %+v %t %v", created, replay, err)
	}
	registrations, err := repo.ListForEvent(ctx, eventID, 25, 0)
	if err != nil || len(registrations) != 1 || registrations[0].AccountID != first.AccountID {
		t.Fatalf("event registrations: %+v %v", registrations, err)
	}
	_, replay, err = repo.Create(ctx, makeBooking(first.AccountID, "AT_VENUE"), "venue-key", eventID+":AT_VENUE", 1)
	if err != nil || !replay {
		t.Fatalf("idempotency replay: %t %v", replay, err)
	}
	var held, confirmed int
	if err = pool.QueryRow(ctx, `SELECT held_count,confirmed_count FROM capacity_allocation WHERE event_id=$1`, eventID).Scan(&held, &confirmed); err != nil || held != 0 || confirmed != 1 {
		t.Fatalf("allocation %d/%d: %v", held, confirmed, err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := makeBooking(uuid.NewString(), "AT_VENUE")
			_, _, e := repo.Create(ctx, b, uuid.NewString(), eventID+":AT_VENUE", 1)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	for e := range results {
		if !errors.Is(e, application.ErrCapacity) {
			t.Fatalf("expected capacity conflict, got %v", e)
		}
	}
	var facts int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE event_type='BookingConfirmed'`).Scan(&facts); err != nil || facts != 1 {
		t.Fatalf("confirmed facts=%d: %v", facts, err)
	}
}
