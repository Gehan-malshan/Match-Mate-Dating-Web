package application

import (
	"context"
	"errors"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/domain"
	"github.com/google/uuid"
	"testing"
	"time"
)

type repoStub struct {
	created   int
	views     int
	status    domain.ReportStatus
	principal domain.Principal
}

func (s *repoStub) Ready(context.Context) error { return nil }
func (s *repoStub) CreateReport(_ context.Context, _ domain.Principal, in domain.CreateReportInput, _ string, now time.Time) (domain.Report, error) {
	s.created++
	return domain.Report{ID: uuid.NewString(), TargetID: in.TargetID, CreatedAt: now}, nil
}
func (s *repoStub) ListOwnReports(context.Context, string, int, time.Time, string) (domain.Page[domain.Report], error) {
	return domain.Page[domain.Report]{}, nil
}
func (s *repoStub) ListCases(context.Context, int, time.Time, string) (domain.Page[domain.Case], error) {
	return domain.Page[domain.Case]{}, nil
}
func (s *repoStub) GetCase(context.Context, string) (domain.Case, error) { return domain.Case{}, nil }
func (s *repoStub) RecordCaseView(context.Context, string, domain.Principal, string, time.Time) error {
	s.views++
	return nil
}
func (s *repoStub) AssignCase(context.Context, string, string, domain.Principal, string, string, time.Time) (domain.Case, error) {
	return domain.Case{}, nil
}
func (s *repoStub) UpdateCaseStatus(_ context.Context, _ string, status domain.ReportStatus, principal domain.Principal, _ string, _ string, _ time.Time) (domain.Case, error) {
	s.status = status
	s.principal = principal
	return domain.Case{}, nil
}
func (s *repoStub) CreateAction(context.Context, string, domain.Action, domain.Principal, string, time.Time) (domain.Action, error) {
	return domain.Action{}, nil
}
func (s *repoStub) CreateAppeal(context.Context, string, string, domain.Principal, string, time.Time) (domain.Appeal, error) {
	return domain.Appeal{}, nil
}
func (s *repoStub) DecideAppeal(context.Context, string, string, domain.Principal, string, string, time.Time) (domain.Appeal, error) {
	return domain.Appeal{}, nil
}
func (s *repoStub) ExpireActions(context.Context, time.Time, int) (int, error) { return 0, nil }
func TestReportAuthorizationValidationAndRateLimit(t *testing.T) {
	repo := &repoStub{}
	service := New(repo, 1)
	input := domain.CreateReportInput{TargetType: domain.TargetAccount, TargetID: uuid.NewString(), Category: domain.CategorySafety, Description: "A concrete safety concern."}
	if _, err := service.CreateReport(context.Background(), domain.Principal{Subject: uuid.NewString()}, input, "c"); err == nil {
		t.Fatal("unprivileged report accepted")
	}
	p := domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}
	if _, err := service.CreateReport(context.Background(), p, input, "c"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateReport(context.Background(), p, input, "c"); err == nil {
		t.Fatal("rate limit not enforced")
	}
}
func TestOrganizerCannotReadCases(t *testing.T) {
	service := New(&repoStub{}, 5)
	if _, err := service.ListCases(context.Background(), domain.Principal{Subject: uuid.NewString(), Roles: []string{"organizer"}}, "", 20); err == nil {
		t.Fatal("organizer accessed cases")
	}
}

func requireProblem(t *testing.T, err error, status int, code string) {
	t.Helper()
	var problem *domain.ProblemError
	if !errors.As(err, &problem) || problem.Status != status || problem.Code != code {
		t.Fatalf("problem=%#v want status=%d code=%s", err, status, code)
	}
}

func TestCaseViewAuthorizationAndAudit(t *testing.T) {
	repo := &repoStub{}
	service := New(repo, 5)
	caseID := uuid.NewString()
	moderator := domain.Principal{Subject: uuid.NewString(), Roles: []string{"moderator"}}
	if _, err := service.GetCase(context.Background(), moderator, caseID, "correlation"); err != nil {
		t.Fatal(err)
	}
	if repo.views != 1 {
		t.Fatalf("privileged views=%d want=1", repo.views)
	}
	_, err := service.GetCase(context.Background(), moderator, "not-a-uuid", "correlation")
	requireProblem(t, err, 404, "MODERATION_CASE_NOT_FOUND")
	if repo.views != 1 {
		t.Fatal("invalid case ID was audited or sent to the repository")
	}
	_, err = service.GetCase(context.Background(), domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}, caseID, "correlation")
	requireProblem(t, err, 403, "AUTHORIZATION_DENIED")
}

func TestCaseStatusAuthorizationAndValidation(t *testing.T) {
	repo := &repoStub{}
	service := New(repo, 5)
	caseID := uuid.NewString()
	moderator := domain.Principal{Subject: uuid.NewString(), Roles: []string{"moderator"}}
	_, err := service.UpdateCaseStatus(context.Background(), domain.Principal{Subject: uuid.NewString(), Roles: []string{"organizer"}}, caseID, "INVESTIGATING", "Documented triage reason", "correlation")
	requireProblem(t, err, 403, "AUTHORIZATION_DENIED")
	_, err = service.UpdateCaseStatus(context.Background(), moderator, caseID, "ACTIONED", "Documented triage reason", "correlation")
	requireProblem(t, err, 400, "CASE_STATUS_INVALID")
	_, err = service.UpdateCaseStatus(context.Background(), moderator, caseID, "investigating", "Documented triage reason", "correlation")
	if err != nil {
		t.Fatal(err)
	}
	if repo.status != domain.ReportInvestigating || repo.principal.Subject != moderator.Subject {
		t.Fatalf("status=%s principal=%s", repo.status, repo.principal.Subject)
	}
}

func TestInvalidResourceIDsDoNotReachRepository(t *testing.T) {
	service := New(&repoStub{}, 5)
	moderator := domain.Principal{Subject: uuid.NewString(), Roles: []string{"moderator"}}
	member := domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}
	_, err := service.Assign(context.Background(), moderator, "bad", uuid.NewString(), "Documented assignment reason", "correlation")
	requireProblem(t, err, 404, "MODERATION_CASE_NOT_FOUND")
	_, err = service.ApplyAction(context.Background(), moderator, "bad", domain.Action{}, "correlation")
	requireProblem(t, err, 404, "MODERATION_CASE_NOT_FOUND")
	_, err = service.Appeal(context.Background(), member, "bad", "Documented appeal reason", "correlation")
	requireProblem(t, err, 404, "ACTION_NOT_APPEALABLE")
	_, err = service.Decide(context.Background(), moderator, "bad", "UPHELD", "Documented decision reason", "correlation")
	requireProblem(t, err, 404, "APPEAL_NOT_PENDING")
}

func TestInvalidCursorIsRejected(t *testing.T) {
	service := New(&repoStub{}, 5)
	_, err := service.ListMine(context.Background(), domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}, "not-base64", 20)
	requireProblem(t, err, 400, "INVALID_CURSOR")
}
