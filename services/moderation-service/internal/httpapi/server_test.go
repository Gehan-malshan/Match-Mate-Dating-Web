package httpapi

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/moderation-service/internal/application"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/config"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/domain"
	"github.com/google/uuid"
)

type verifierStub struct{ principal domain.Principal }

func (s verifierStub) Verify(string) (domain.Principal, error) { return s.principal, nil }

type repoStub struct{}

func (repoStub) Ready(context.Context) error { return nil }
func (repoStub) CreateReport(_ context.Context, p domain.Principal, in domain.CreateReportInput, _ string, now time.Time) (domain.Report, error) {
	return domain.Report{ID: uuid.NewString(), CaseID: uuid.NewString(), TargetType: in.TargetType, TargetID: in.TargetID, Category: in.Category, Status: domain.ReportOpen, Severity: domain.SeverityMedium, CreatedAt: now, UpdatedAt: now}, nil
}
func (repoStub) ListOwnReports(context.Context, string, int, time.Time, string) (domain.Page[domain.Report], error) {
	return domain.Page[domain.Report]{Items: []domain.Report{}, Limit: 20}, nil
}
func (repoStub) ListCases(context.Context, int, time.Time, string) (domain.Page[domain.Case], error) {
	return domain.Page[domain.Case]{Items: []domain.Case{}, Limit: 20}, nil
}
func (repoStub) GetCase(context.Context, string) (domain.Case, error) { return domain.Case{}, nil }
func (repoStub) RecordCaseView(context.Context, string, domain.Principal, string, time.Time) error {
	return nil
}
func (repoStub) AssignCase(context.Context, string, string, domain.Principal, string, string, time.Time) (domain.Case, error) {
	return domain.Case{}, nil
}
func (repoStub) UpdateCaseStatus(context.Context, string, domain.ReportStatus, domain.Principal, string, string, time.Time) (domain.Case, error) {
	return domain.Case{}, nil
}
func (repoStub) CreateAction(context.Context, string, domain.Action, domain.Principal, string, time.Time) (domain.Action, error) {
	return domain.Action{}, nil
}
func (repoStub) CreateAppeal(context.Context, string, string, domain.Principal, string, time.Time) (domain.Appeal, error) {
	return domain.Appeal{}, nil
}
func (repoStub) DecideAppeal(context.Context, string, string, domain.Principal, string, string, time.Time) (domain.Appeal, error) {
	return domain.Appeal{}, nil
}
func (repoStub) ExpireActions(context.Context, time.Time, int) (int, error) { return 0, nil }
func handler(principal domain.Principal) http.Handler {
	return New(application.New(repoStub{}, 5), verifierStub{principal}, config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
func TestAuthenticationAndRoleIsolation(t *testing.T) {
	unauthorized := httptest.NewRecorder()
	handler(domain.Principal{}).ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/reports/mine", nil))
	if unauthorized.Code != 401 {
		t.Fatalf("unauthorized status=%d", unauthorized.Code)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/moderation/cases", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler(domain.Principal{Subject: uuid.NewString(), Roles: []string{"organizer"}}).ServeHTTP(response, request)
	if response.Code != 403 {
		t.Fatalf("organizer case status=%d body=%s", response.Code, response.Body.String())
	}
}
func TestMemberCanSubmitValidReport(t *testing.T) {
	target := uuid.NewString()
	body := `{"targetType":"ACCOUNT","targetId":"` + target + `","category":"SAFETY","description":"A concrete safety concern."}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/reports", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler(domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}).ServeHTTP(response, request)
	if response.Code != 201 || strings.Contains(response.Body.String(), "reporter") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRequestBodyRejectsUnknownAndTrailingJSON(t *testing.T) {
	principal := domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}
	target := uuid.NewString()
	valid := `{"targetType":"ACCOUNT","targetId":"` + target + `","category":"SAFETY","description":"A concrete safety concern."}`
	for name, body := range map[string]string{
		"unknown field":  strings.TrimSuffix(valid, "}") + `,"reporterId":"` + uuid.NewString() + `"}`,
		"trailing value": valid + `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/reports", strings.NewReader(body))
			request.Header.Set("Authorization", "Bearer valid")
			response := httptest.NewRecorder()
			handler(principal).ServeHTTP(response, request)
			if response.Code != 400 || response.Header().Get("Content-Type") != "application/problem+json" || !strings.Contains(response.Body.String(), `"code":"INVALID_JSON"`) {
				t.Fatalf("status=%d content-type=%s body=%s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
			}
		})
	}
}

func TestModeratorCanStartInvestigation(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/moderation/cases/"+uuid.NewString()+"/status", strings.NewReader(`{"status":"INVESTIGATING","reason":"Documented triage decision"}`))
	request.Header.Set("Authorization", "Bearer valid")
	request.Header.Set("X-Correlation-ID", "moderation-test")
	response := httptest.NewRecorder()
	handler(domain.Principal{Subject: uuid.NewString(), Roles: []string{"moderator"}}).ServeHTTP(response, request)
	if response.Code != 200 || response.Header().Get("X-Correlation-ID") != "moderation-test" {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestMemberCannotChangeCaseStatus(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/moderation/cases/"+uuid.NewString()+"/status", strings.NewReader(`{"status":"DISMISSED","reason":"Documented dismissal reason"}`))
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler(domain.Principal{Subject: uuid.NewString(), Roles: []string{"member"}}).ServeHTTP(response, request)
	if response.Code != 403 || !strings.Contains(response.Body.String(), `"code":"AUTHORIZATION_DENIED"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRequestLogUsesRoutePatternAndResultCode(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	server := New(application.New(repoStub{}, 5), verifierStub{domain.Principal{Subject: uuid.NewString(), Roles: []string{"moderator"}}}, config.Config{}, logger)
	caseID := uuid.NewString()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/moderation/cases/"+caseID, nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	logLine := output.String()
	if !strings.Contains(logLine, `"route":"GET /api/v1/moderation/cases/{caseId}"`) || !strings.Contains(logLine, `"result_code":200`) || strings.Contains(logLine, caseID) {
		t.Fatalf("unsafe or incomplete request log: %s", logLine)
	}
}
