package postgres_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/application"
	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
	store "github.com/gehan-malshan/matchmate/notification-service/internal/store/postgres"
	"github.com/gehan-malshan/matchmate/notification-service/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryEventDeduplicationDeliveryAndSuppression(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("NOTIFICATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("NOTIFICATION_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "notification_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE") })

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	migration, err := migrations.Files.ReadFile("000001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}

	repository := store.New(pool)
	router := application.NewRouter("en-LK")
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	accountID := uuid.NewString()
	event := domain.EventEnvelope{EventID: uuid.NewString(), EventType: "AccountVerified", SchemaVersion: 1, OccurredAt: now, AggregateID: accountID, CorrelationID: uuid.NewString()}
	event.Payload, _ = json.Marshal(map[string]string{"accountId": accountID})
	plan, err := router.Route(event)
	if err != nil {
		t.Fatal(err)
	}
	created, err := repository.ApplyEvent(ctx, event, plan, now, 3)
	if err != nil || !created {
		t.Fatalf("first event created=%v err=%v", created, err)
	}
	created, err = repository.ApplyEvent(ctx, event, plan, now.Add(time.Second), 3)
	if err != nil || created {
		t.Fatalf("duplicate event created=%v err=%v", created, err)
	}

	delivery, found, err := repository.ClaimDue(ctx, now, 30*time.Second)
	if err != nil || !found || delivery.AttemptCount != 1 {
		t.Fatalf("claim found=%v delivery=%+v err=%v", found, delivery, err)
	}
	state, err := repository.CompleteAttempt(ctx, delivery, store.AttemptResult{Delivered: true, ProviderReference: "test:1", StartedAt: now, CompletedAt: now.Add(time.Second)})
	if err != nil || state != domain.DeliveryDelivered {
		t.Fatalf("complete state=%s err=%v", state, err)
	}

	deactivated := domain.EventEnvelope{EventID: uuid.NewString(), EventType: "AccountDeactivated", SchemaVersion: 1, OccurredAt: now.Add(2 * time.Second), AggregateID: accountID, CorrelationID: uuid.NewString()}
	deactivated.Payload, _ = json.Marshal(map[string]string{"accountId": accountID})
	suppressPlan, err := router.Route(deactivated)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.ApplyEvent(ctx, deactivated, suppressPlan, now.Add(2*time.Second), 3); err != nil {
		t.Fatal(err)
	}
	var inboxCount, deliveryCount, suppressionCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM notification_inbox`).Scan(&inboxCount); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery`).Scan(&deliveryCount); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM notification_suppression`).Scan(&suppressionCount); err != nil {
		t.Fatal(err)
	}
	if inboxCount != 2 || deliveryCount != 1 || suppressionCount != 1 {
		t.Fatalf("inbox=%d delivery=%d suppression=%d", inboxCount, deliveryCount, suppressionCount)
	}
}
