package postgres_test

import (
	"context"
	"errors"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/domain"
	store "github.com/gehan-malshan/matchmate/moderation-service/internal/store/postgres"
	"github.com/gehan-malshan/matchmate/moderation-service/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
	"time"
)

func TestReportActionAppealAndOutbox(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("MODERATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("MODERATION_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "moderation_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE") })
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sql, err := migrations.Files.ReadFile("000001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(sql)); err != nil {
		t.Fatal(err)
	}
	repo := store.New(pool)
	now := time.Now().UTC()
	member := domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}
	report, err := repo.CreateReport(ctx, member, domain.CreateReportInput{TargetType: domain.TargetAccount, TargetID: member.Subject, Category: domain.CategorySafety, Description: "A concrete safety concern."}, "correlation", now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateReport(ctx, member, domain.CreateReportInput{TargetType: domain.TargetAccount, TargetID: member.Subject, Category: domain.CategorySafety, Description: "A concrete safety concern."}, "correlation", now)
	requireProblem(t, err, 409, "DUPLICATE_REPORT")
	owned, err := repo.ListOwnReports(ctx, member.Subject, 20, time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC), "ffffffff-ffff-ffff-ffff-ffffffffffff")
	if err != nil || len(owned.Items) != 1 || owned.Items[0].Description != "" {
		t.Fatalf("owner-safe reports=%+v err=%v", owned, err)
	}
	moderator := domain.Principal{Subject: uuid.NewString(), Roles: []string{"moderator"}}
	if err = repo.RecordCaseView(ctx, report.CaseID, moderator, "case-view", now); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateCaseStatus(ctx, report.CaseID, domain.ReportInvestigating, moderator, "Start investigation after triage", "correlation", now); err == nil {
		t.Fatal("case moved from OPEN directly to INVESTIGATING")
	}
	if _, err = repo.AssignCase(ctx, report.CaseID, moderator.Subject, moderator, "Assign case for safety review", "correlation", now); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateCaseStatus(ctx, report.CaseID, domain.ReportInvestigating, moderator, "Start investigation after triage", "correlation", now); err != nil {
		t.Fatal(err)
	}
	action, err := repo.CreateAction(ctx, report.CaseID, domain.Action{TargetType: domain.TargetAccount, TargetID: member.Subject, Class: domain.ActionRevealPrevention, Scope: "ALL", Reason: "Prevent reveal during investigation", EffectiveAt: now}, moderator, "correlation", now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateAction(ctx, report.CaseID, domain.Action{TargetType: domain.TargetAccount, TargetID: member.Subject, Class: domain.ActionRevealPrevention, Scope: "ALL", Reason: "Prevent reveal during investigation", EffectiveAt: now}, moderator, "correlation", now)
	requireProblem(t, err, 409, "ACTION_ALREADY_ACTIVE")
	_, err = repo.CreateAppeal(ctx, action.ID, "Please review this safety decision", domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}, "correlation", now)
	requireProblem(t, err, 403, "APPEAL_FORBIDDEN")
	appeal, err := repo.CreateAppeal(ctx, action.ID, "Please review this safety decision", member, "correlation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.DecideAppeal(ctx, appeal.ID, "REVERSED", moderator, "Evidence supports reversing action", "correlation", now); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.DecideAppeal(ctx, appeal.ID, "UPHELD", moderator, "Second decision must be rejected", "correlation", now); err == nil {
		t.Fatal("appeal was decided twice")
	}
	expiringMember := domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}
	expiringReport, err := repo.CreateReport(ctx, expiringMember, domain.CreateReportInput{TargetType: domain.TargetAccount, TargetID: expiringMember.Subject, Category: domain.CategoryHarassment, Description: "A separate harassment report for expiry."}, "expiry-correlation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.AssignCase(ctx, expiringReport.CaseID, moderator.Subject, moderator, "Assign expiring action review", "expiry-correlation", now); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateCaseStatus(ctx, expiringReport.CaseID, domain.ReportInvestigating, moderator, "Investigate expiring action report", "expiry-correlation", now); err != nil {
		t.Fatal(err)
	}
	expiresAt := now.Add(time.Minute)
	if _, err = repo.CreateAction(ctx, expiringReport.CaseID, domain.Action{TargetType: domain.TargetAccount, TargetID: expiringMember.Subject, Class: domain.ActionMatchmakingExclusion, Scope: "ALL", Reason: "Temporary exclusion during investigation", EffectiveAt: now, ExpiresAt: &expiresAt}, moderator, "expiry-correlation", now); err != nil {
		t.Fatal(err)
	}
	if count, expireErr := repo.ExpireActions(ctx, now.Add(2*time.Minute), 20); expireErr != nil || count != 1 {
		t.Fatalf("expired=%d err=%v", count, expireErr)
	}
	if count, expireErr := repo.ExpireActions(ctx, now.Add(2*time.Minute), 20); expireErr != nil || count != 0 {
		t.Fatalf("expiry was not idempotent: expired=%d err=%v", count, expireErr)
	}
	dismissedMember := domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}
	dismissedReport, err := repo.CreateReport(ctx, dismissedMember, domain.CreateReportInput{TargetType: domain.TargetAccount, TargetID: dismissedMember.Subject, Category: domain.CategoryOther, Description: "A separate report that will be dismissed."}, "dismiss-correlation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.AssignCase(ctx, dismissedReport.CaseID, moderator.Subject, moderator, "Assign dismissal review case", "dismiss-correlation", now); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateCaseStatus(ctx, dismissedReport.CaseID, domain.ReportInvestigating, moderator, "Investigate report before decision", "dismiss-correlation", now); err != nil {
		t.Fatal(err)
	}
	dismissed, err := repo.UpdateCaseStatus(ctx, dismissedReport.CaseID, domain.ReportDismissed, moderator, "Dismiss after documented investigation", "dismiss-correlation", now)
	if err != nil || dismissed.Status != domain.ReportDismissed || len(dismissed.Reports) != 1 || dismissed.Reports[0].Status != domain.ReportDismissed {
		t.Fatalf("dismissed case=%+v err=%v", dismissed, err)
	}
	records, err := repo.ClaimOutbox(ctx, 20)
	if err != nil || len(records) != 9 {
		t.Fatalf("outbox=%d err=%v", len(records), err)
	}
	for _, record := range records {
		body := strings.ToLower(string(record.Body))
		if strings.Contains(body, "reporter") || strings.Contains(body, "description") || strings.Contains(body, "evidence") {
			t.Fatalf("private report data leaked in outbox event: %s", record.Body)
		}
	}
	claimedAgain, err := repo.ClaimOutbox(ctx, 20)
	if err != nil || len(claimedAgain) != 0 {
		t.Fatalf("claimed outbox records were immediately reclaimed: count=%d err=%v", len(claimedAgain), err)
	}
	if err = repo.MarkOutboxPublished(ctx, records[0].ID, now); err != nil {
		t.Fatal(err)
	}
	var published int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE published_at IS NOT NULL`).Scan(&published); err != nil || published != 1 {
		t.Fatalf("published=%d err=%v", published, err)
	}
	var auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM moderation_audit WHERE case_id=$1`, report.CaseID).Scan(&auditCount); err != nil || auditCount != 7 {
		t.Fatalf("audit count=%d err=%v", auditCount, err)
	}
}

func requireProblem(t *testing.T, err error, status int, code string) {
	t.Helper()
	var problem *domain.ProblemError
	if !errors.As(err, &problem) || problem.Status != status || problem.Code != code {
		t.Fatalf("problem=%#v want status=%d code=%s", err, status, code)
	}
}
