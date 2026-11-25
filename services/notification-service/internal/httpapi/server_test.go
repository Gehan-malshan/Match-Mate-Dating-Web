package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gehan-malshan/matchmate/notification-service/internal/application"
	"github.com/gehan-malshan/matchmate/notification-service/internal/auth"
	"github.com/google/uuid"
)

type readinessStub struct{ err error }

func (s readinessStub) Ready(context.Context) error { return s.err }

type verifierStub struct {
	principal auth.Principal
	err       error
}

func (s verifierStub) Verify(string) (auth.Principal, error) { return s.principal, s.err }

type feedStub struct {
	page           application.FeedPage
	listAccount    string
	markAccount    string
	markID         string
	markAllAccount string
	unread         int
	err            error
}

func (s *feedStub) List(_ context.Context, accountID string, _ int, _ string) (application.FeedPage, error) {
	s.listAccount = accountID
	return s.page, s.err
}
func (s *feedStub) UnreadCount(context.Context, string) (int, error) { return s.unread, s.err }
func (s *feedStub) MarkRead(_ context.Context, accountID, notificationID string) error {
	s.markAccount, s.markID = accountID, notificationID
	return s.err
}
func (s *feedStub) MarkAllRead(_ context.Context, accountID string) (int64, error) {
	s.markAllAccount = accountID
	return 2, s.err
}

func TestHealthEndpoints(t *testing.T) {
	for _, test := range []struct {
		name, path string
		readyErr   error
		want       int
	}{
		{"live", "/health/live", nil, http.StatusNoContent},
		{"ready", "/health/ready", nil, http.StatusNoContent},
		{"not-ready", "/health/ready", errors.New("database unavailable"), http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			New(readinessStub{err: test.readyErr}, &feedStub{}, verifierStub{}).ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d", response.Code, test.want)
			}
		})
	}
}

func TestNotificationEndpointsRequireAuthenticationAndUseTokenSubject(t *testing.T) {
	accountID := uuid.NewString()
	feed := &feedStub{page: application.FeedPage{UnreadCount: 1}}
	handler := New(readinessStub{}, feed, verifierStub{principal: auth.Principal{Subject: accountID, Roles: []string{"member"}}})

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil))
	if unauthorized.Code != http.StatusUnauthorized || !strings.Contains(unauthorized.Body.String(), "AUTHENTICATION_REQUIRED") {
		t.Fatalf("unexpected unauthorized response: %d %s", unauthorized.Code, unauthorized.Body.String())
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=5", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || feed.listAccount != accountID {
		t.Fatalf("status=%d account=%q body=%s", response.Code, feed.listAccount, response.Body.String())
	}
}

func TestMarkReadCannotSelectAnotherAccount(t *testing.T) {
	accountID := uuid.NewString()
	notificationID := uuid.NewString()
	feed := &feedStub{}
	handler := New(readinessStub{}, feed, verifierStub{principal: auth.Principal{Subject: accountID}})
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+notificationID+"/read", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || feed.markAccount != accountID || feed.markID != notificationID {
		t.Fatalf("status=%d account=%q id=%q", response.Code, feed.markAccount, feed.markID)
	}
}

func TestInvalidPaginationUsesProblemDetails(t *testing.T) {
	handler := New(readinessStub{}, &feedStub{}, verifierStub{principal: auth.Principal{Subject: uuid.NewString()}})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=invalid", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || response.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status=%d type=%q body=%s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
}
