package postgres_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/event-service/internal/application"
	"github.com/gehan-malshan/matchmate/event-service/internal/domain"
	storepg "github.com/gehan-malshan/matchmate/event-service/internal/store/postgres"
	"github.com/gehan-malshan/matchmate/event-service/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryMigrationLifecycleAuditAndOutbox(t *testing.T) {
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
	repo := storepg.New(pool)
	app := application.New(repo)
	start := time.Now().UTC().Add(7 * 24 * time.Hour)
	input := domain.CreateInput{OrganizerID: "organizer-1", Name: "Component Test Social", Description: "Fictional event", VenueName: "Private venue", BroadLocation: "Colombo", TimeZone: "Asia/Colombo", StartsAt: start, EndsAt: start.Add(2 * time.Hour), RegistrationOpensAt: start.Add(-6 * 24 * time.Hour), RegistrationClosesAt: start.Add(-time.Hour), Price: "3000.00", Currency: "LKR", ConfiguredCapacity: 30, MatchingRulesetVersion: "rules-v1"}
	principal := domain.Principal{Subject: "organizer-1", Roles: []string{"organizer"}}
	event, err := app.Create(ctx, principal, input, "component-correlation")
	if err != nil {
		t.Fatal(err)
	}
	event, err = app.Transition(ctx, principal, event.ID, event.Version, domain.Published, "", "component-correlation")
	if err != nil {
		t.Fatal(err)
	}
	var audits, outbox int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM event_audit WHERE event_id=$1`, event.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE aggregate_id=$1`, event.ID).Scan(&outbox); err != nil {
		t.Fatal(err)
	}
	if audits != 1 || outbox != 2 {
		t.Fatalf("expected one audit and two outbox rows, got audit=%d outbox=%d", audits, outbox)
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
}
