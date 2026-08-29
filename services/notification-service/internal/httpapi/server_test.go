package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type readinessStub struct{ err error }

func (s readinessStub) Ready(context.Context) error { return s.err }

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
			New(readinessStub{err: test.readyErr}).ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d", response.Code, test.want)
			}
		})
	}
}
