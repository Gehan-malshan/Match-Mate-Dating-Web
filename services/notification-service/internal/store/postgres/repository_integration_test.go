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
	for _, name := range []string{"000001_init.up.sql", "000002_member_feed.up.sql", "000003_email_templates.up.sql", "000004_safety_delivery.up.sql", "000005_delivery_replay.up.sql"} {
		migration, readErr := migrations.Files.ReadFile(name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = pool.Exec(ctx, string(migration)); err != nil {
			t.Fatal(err)
		}
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
	feed, hasMore, err := repository.ListFeed(ctx, accountID, 20, nil)
	if err != nil || hasMore || len(feed) != 1 || feed[0].SourceEventType != "AccountVerified" {
		t.Fatalf("feed=%+v hasMore=%v err=%v", feed, hasMore, err)
	}
	unread, err := repository.UnreadCount(ctx, accountID)
	if err != nil || unread != 1 {
		t.Fatalf("unread=%d err=%v", unread, err)
	}
	updated, err := repository.MarkRead(ctx, accountID, feed[0].ID, now.Add(2*time.Second))
	if err != nil || !updated {
		t.Fatalf("mark read updated=%v err=%v", updated, err)
	}
	otherUpdated, err := repository.MarkRead(ctx, uuid.NewString(), feed[0].ID, now.Add(2*time.Second))
	if err != nil || otherUpdated {
		t.Fatalf("cross-account mark updated=%v err=%v", otherUpdated, err)
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
	preference, err := repository.SetPreference(ctx, accountID, "EMAIL", "ACCOUNT", false, "MEMBER_SELF_SERVICE", now)
	if err != nil || preference.Allowed || preference.Category != "ACCOUNT" {
		t.Fatalf("preference=%+v err=%v", preference, err)
	}
	preferences, err := repository.ListPreferences(ctx, accountID, "EMAIL")
	if err != nil || len(preferences) != 1 || preferences[0].Allowed {
		t.Fatalf("preferences=%+v err=%v", preferences, err)
	}

	if _, err = pool.Exec(ctx, `UPDATE notification_delivery SET state='DEAD_LETTERED',attempt_count=3 WHERE id=$1`, delivery.ID); err != nil {
		t.Fatal(err)
	}
	replayed, err := repository.Replay(ctx, delivery.ID, "operator@example.test", "SMTP sender verification was corrected", 3, now.Add(3*time.Second))
	if err != nil || !replayed {
		t.Fatalf("replayed=%v err=%v", replayed, err)
	}
	var replayCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery_replay WHERE delivery_id=$1`, delivery.ID).Scan(&replayCount); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT state FROM notification_delivery WHERE id=$1`, delivery.ID).Scan(&state); err != nil || state != domain.DeliveryRetryScheduled || replayCount != 1 {
		t.Fatalf("state=%s replays=%d err=%v", state, replayCount, err)
	}
}
