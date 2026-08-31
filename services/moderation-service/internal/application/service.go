package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/domain"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/store"
	"github.com/google/uuid"
	"strings"
	"sync"
	"time"
)

func parseCursor(value string) (time.Time, string, error) {
	if value == "" {
		return time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC), "ffffffff-ffff-ffff-ffff-ffffffffffff", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, "", &domain.ProblemError{Status: 400, Code: "INVALID_CURSOR", Detail: "Cursor is invalid"}
	}
	var cursor struct {
		T  time.Time `json:"t"`
		ID string    `json:"id"`
	}
	if json.Unmarshal(raw, &cursor) != nil {
		return time.Time{}, "", &domain.ProblemError{Status: 400, Code: "INVALID_CURSOR", Detail: "Cursor is invalid"}
	}
	if _, err = uuid.Parse(cursor.ID); err != nil {
		return time.Time{}, "", &domain.ProblemError{Status: 400, Code: "INVALID_CURSOR", Detail: "Cursor is invalid"}
	}
	return cursor.T, cursor.ID, nil
}

type Service struct {
	repo          store.Repository
	now           func() time.Time
	mu            sync.Mutex
	reportWindows map[string][]time.Time
	rateLimit     int
}

func New(repo store.Repository, rateLimit int) *Service {
	return &Service{repo: repo, now: time.Now, reportWindows: map[string][]time.Time{}, rateLimit: rateLimit}
}
func (s *Service) Ready(ctx context.Context) error { return s.repo.Ready(ctx) }
func hasAny(p domain.Principal, roles ...string) bool {
	for _, role := range roles {
		if p.HasRole(role) {
			return true
		}
	}
	return false
}
func forbidden() *domain.ProblemError {
	return &domain.ProblemError{Status: 403, Code: "AUTHORIZATION_DENIED", Detail: "The authenticated role cannot perform this moderation operation"}
}
func (s *Service) allowReport(account string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := now.Add(-time.Hour)
	current := s.reportWindows[account]
	filtered := current[:0]
	for _, at := range current {
		if at.After(cutoff) {
			filtered = append(filtered, at)
		}
	}
	if len(filtered) >= s.rateLimit {
		s.reportWindows[account] = filtered
		return false
	}
	s.reportWindows[account] = append(filtered, now)
	return true
}
func (s *Service) CreateReport(ctx context.Context, p domain.Principal, in domain.CreateReportInput, correlation string) (domain.Report, error) {
	if !hasAny(p, "member", "organizer", "moderator", "admin") {
		return domain.Report{}, forbidden()
	}
	if fields := domain.ValidateReport(in); len(fields) > 0 {
		return domain.Report{}, &domain.ProblemError{Status: 400, Code: "REPORT_INVALID", Detail: "Report fields are invalid", Fields: fields}
	}
	now := s.now().UTC()
	if !s.allowReport(p.Subject, now) {
		return domain.Report{}, &domain.ProblemError{Status: 429, Code: "REPORT_RATE_LIMITED", Detail: "Too many reports were submitted; retry later"}
	}
	return s.repo.CreateReport(ctx, p, in, correlation, now)
}
func limit(value int) int {
	if value < 1 {
		return 20
	}
	if value > 50 {
		return 50
	}
	return value
}
func (s *Service) ListMine(ctx context.Context, p domain.Principal, cursor string, size int) (domain.Page[domain.Report], error) {
	at, id, err := parseCursor(cursor)
	if err != nil {
		return domain.Page[domain.Report]{}, err
	}
	return s.repo.ListOwnReports(ctx, p.Subject, limit(size), at, id)
}
func (s *Service) ListCases(ctx context.Context, p domain.Principal, cursor string, size int) (domain.Page[domain.Case], error) {
	if !hasAny(p, "moderator", "admin") {
		return domain.Page[domain.Case]{}, forbidden()
	}
	at, id, err := parseCursor(cursor)
	if err != nil {
		return domain.Page[domain.Case]{}, err
	}
	return s.repo.ListCases(ctx, limit(size), at, id)
}
func (s *Service) GetCase(ctx context.Context, p domain.Principal, id, correlation string) (domain.Case, error) {
	if !hasAny(p, "moderator", "admin") {
		return domain.Case{}, forbidden()
	}
	if _, err := uuid.Parse(id); err != nil {
		return domain.Case{}, &domain.ProblemError{Status: 404, Code: "MODERATION_CASE_NOT_FOUND", Detail: "Moderation case was not found"}
	}
	if err := s.repo.RecordCaseView(ctx, id, p, correlation, s.now().UTC()); err != nil {
		return domain.Case{}, err
	}
	return s.repo.GetCase(ctx, id)
}
func (s *Service) Assign(ctx context.Context, p domain.Principal, caseID, assignee, reason, correlation string) (domain.Case, error) {
	if !hasAny(p, "moderator", "admin") {
		return domain.Case{}, forbidden()
	}
	if _, err := uuid.Parse(caseID); err != nil {
		return domain.Case{}, &domain.ProblemError{Status: 404, Code: "MODERATION_CASE_NOT_FOUND", Detail: "Moderation case was not found"}
	}
	if _, err := uuid.Parse(assignee); err != nil || len(strings.TrimSpace(reason)) < 10 || len(strings.TrimSpace(reason)) > 1000 {
		return domain.Case{}, &domain.ProblemError{Status: 400, Code: "ASSIGNMENT_INVALID", Detail: "Assignee and audit reason are required"}
	}
	return s.repo.AssignCase(ctx, caseID, assignee, p, strings.TrimSpace(reason), correlation, s.now().UTC())
}
func (s *Service) UpdateCaseStatus(ctx context.Context, p domain.Principal, caseID, status, reason, correlation string) (domain.Case, error) {
	if !hasAny(p, "moderator", "admin") {
		return domain.Case{}, forbidden()
	}
	if _, err := uuid.Parse(caseID); err != nil {
		return domain.Case{}, &domain.ProblemError{Status: 404, Code: "MODERATION_CASE_NOT_FOUND", Detail: "Moderation case was not found"}
	}
	next := domain.ReportStatus(strings.ToUpper(strings.TrimSpace(status)))
	reason = strings.TrimSpace(reason)
	if (next != domain.ReportInvestigating && next != domain.ReportDismissed) || len(reason) < 10 || len(reason) > 1000 {
		return domain.Case{}, &domain.ProblemError{Status: 400, Code: "CASE_STATUS_INVALID", Detail: "Status and audit reason are invalid"}
	}
	return s.repo.UpdateCaseStatus(ctx, caseID, next, p, reason, correlation, s.now().UTC())
}
func (s *Service) ApplyAction(ctx context.Context, p domain.Principal, caseID string, a domain.Action, correlation string) (domain.Action, error) {
	if !hasAny(p, "moderator", "admin") {
		return a, forbidden()
	}
	if _, err := uuid.Parse(caseID); err != nil {
		return a, &domain.ProblemError{Status: 404, Code: "MODERATION_CASE_NOT_FOUND", Detail: "Moderation case was not found"}
	}
	if _, err := uuid.Parse(a.TargetID); err != nil || !domain.ValidTarget(a.TargetType) {
		return a, &domain.ProblemError{Status: 400, Code: "ACTION_INVALID", Detail: "Action target is invalid"}
	}
	if fields := domain.ValidAction(a.Class, a.Scope, a.Reason, a.EffectiveAt, a.ExpiresAt); len(fields) > 0 {
		return a, &domain.ProblemError{Status: 400, Code: "ACTION_INVALID", Detail: "Action fields are invalid", Fields: fields}
	}
	return s.repo.CreateAction(ctx, caseID, a, p, correlation, s.now().UTC())
}
func (s *Service) Appeal(ctx context.Context, p domain.Principal, actionID, reason, correlation string) (domain.Appeal, error) {
	if _, err := uuid.Parse(actionID); err != nil {
		return domain.Appeal{}, &domain.ProblemError{Status: 404, Code: "ACTION_NOT_APPEALABLE", Detail: "Active action was not found"}
	}
	reason = strings.TrimSpace(reason)
	if len(reason) < 10 || len(reason) > 1000 {
		return domain.Appeal{}, &domain.ProblemError{Status: 400, Code: "APPEAL_INVALID", Detail: "Appeal reason must contain 10 to 1000 characters"}
	}
	return s.repo.CreateAppeal(ctx, actionID, reason, p, correlation, s.now().UTC())
}
func (s *Service) Decide(ctx context.Context, p domain.Principal, appealID, decision, reason, correlation string) (domain.Appeal, error) {
	if !hasAny(p, "moderator", "admin") {
		return domain.Appeal{}, forbidden()
	}
	if _, err := uuid.Parse(appealID); err != nil {
		return domain.Appeal{}, &domain.ProblemError{Status: 404, Code: "APPEAL_NOT_PENDING", Detail: "Pending appeal was not found"}
	}
	decision = strings.ToUpper(strings.TrimSpace(decision))
	reason = strings.TrimSpace(reason)
	if (decision != "UPHELD" && decision != "REVERSED") || len(reason) < 10 || len(reason) > 1000 {
		return domain.Appeal{}, &domain.ProblemError{Status: 400, Code: "APPEAL_DECISION_INVALID", Detail: "Decision and audit reason are invalid"}
	}
	return s.repo.DecideAppeal(ctx, appealID, decision, p, reason, correlation, s.now().UTC())
}
