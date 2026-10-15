package postgres_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/event-service/internal/application"
	"github.com/gehan-malshan/matchmate/event-service/internal/domain"
	"github.com/gehan-malshan/matchmate/event-service/internal/store"
	storepg "github.com/gehan-malshan/matchmate/event-service/internal/store/postgres"
	"github.com/gehan-malshan/matchmate/event-service/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryMigrationLifecycleAndOutbox(t *testing.T) {
	url := os.Getenv("EVENT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set EVENT_TEST_DATABASE_URL to run PostgreSQL component tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "event_test_" + uuid.NewString()
	schema = strings.ReplaceAll(schema, "-", "_")
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
	if _, err = pool.Exec(ctx, migrations.Up); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, migrations.PaymentOptionsUp); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, migrations.EventImageUp); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, migrations.EventArchiveUp); err != nil {
		t.Fatal(err)
	}
	// Simulate a database created before Event auditing was removed.
	if _, err = pool.Exec(ctx, `CREATE TABLE event_audit (audit_id integer PRIMARY KEY); INSERT INTO event_audit VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, migrations.RemoveEventAuditUp); err != nil {
		t.Fatal(err)
	}
	var auditTable *string
	if err = pool.QueryRow(ctx, `SELECT to_regclass('event_audit')::text`).Scan(&auditTable); err != nil || auditTable != nil {
		t.Fatalf("event audit table was not removed: %v %v", auditTable, err)
	}
	repo := storepg.New(pool)
	app := application.New(repo)
	start := time.Now().UTC().Add(7 * 24 * time.Hour)
	input := domain.CreateInput{OrganizerID: "organizer-1", Name: "Component Test Social", Description: "Fictional event", VenueName: "Private venue", BroadLocation: "Colombo", TimeZone: "Asia/Colombo", StartsAt: start, EndsAt: start.Add(2 * time.Hour), RegistrationOpensAt: start.Add(-6 * 24 * time.Hour), RegistrationClosesAt: start.Add(-time.Hour), Price: "3000.00", Currency: "LKR", ConfiguredCapacity: 30, MatchingRulesetVersion: "rules-v1"}
	principal := domain.Principal{Subject: "organizer-1", Roles: []string{"admin"}}
	event, err := app.Create(ctx, principal, input, "component-correlation")
	if err != nil {
		t.Fatal(err)
	}
	imageBytes := []byte{0xff, 0xd8, 0xff, 0xd9}
	withImage, err := repo.SaveImage(ctx, event.ID, imageBytes)
	if err != nil || withImage.ImageVersion != 1 {
		t.Fatalf("draft image save: %+v %v", withImage, err)
	}
	if _, err = repo.PublicImage(ctx, event.ID); err == nil {
		t.Fatal("draft image leaked publicly")
	}
	event, err = app.Transition(ctx, principal, event.ID, event.Version, domain.Published, "component-correlation")
	if err != nil {
		t.Fatal(err)
	}
	publicImage, err := repo.PublicImage(ctx, event.ID)
	if err != nil || !bytes.Equal(publicImage, imageBytes) {
		t.Fatalf("published image unavailable: %v", err)
	}
	if _, err = repo.SaveImage(ctx, event.ID, imageBytes); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("published image was mutable: %v", err)
	}
	var outbox int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE aggregate_id=$1`, event.ID).Scan(&outbox); err != nil {
		t.Fatal(err)
	}
	if outbox != 2 {
		t.Fatalf("expected two outbox rows, got %d", outbox)
	}
	records, err := repo.ClaimOutbox(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected two claimed records, got %d", len(records))
	}
	if err = repo.MarkOutboxPublished(ctx, records[0].ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	var published bool
	if err = pool.QueryRow(ctx, `SELECT published_at IS NOT NULL FROM outbox WHERE event_id=$1`, records[0].ID).Scan(&published); err != nil || !published {
		t.Fatalf("expected published marker: %v", err)
	}
	archiveInput := input
	archiveInput.Name = "Archived Draft"
	draft, err := app.Create(ctx, principal, archiveInput, "archive-test")
	if err != nil {
		t.Fatal(err)
	}
	draft, err = app.SetArchived(ctx, principal, draft.ID, draft.Version, true, "archive-test")
	if err != nil || draft.ArchivedAt == nil {
		t.Fatalf("archive draft: %+v %v", draft, err)
	}
	active, err := app.ListManaged(ctx, principal, false, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range active.Items {
		if item.ID == draft.ID {
			t.Fatal("deleted draft visible in active list")
		}
	}
	deleted, err := app.ListManaged(ctx, principal, true, "", 100)
	if err != nil || len(deleted.Items) != 1 || deleted.Items[0].ID != draft.ID {
		t.Fatalf("deleted list: %+v %v", deleted, err)
	}
	if _, err = app.SetArchived(ctx, principal, draft.ID, draft.Version-1, false, "archive-test"); err == nil {
		t.Fatal("stale restore accepted")
	}
	draft, err = app.SetArchived(ctx, principal, draft.ID, draft.Version, false, "archive-test")
	if err != nil || draft.ArchivedAt != nil {
		t.Fatalf("restore draft: %+v %v", draft, err)
	}
}
