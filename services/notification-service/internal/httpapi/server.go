package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/application"
	"github.com/gehan-malshan/matchmate/notification-service/internal/auth"
	"github.com/google/uuid"
)

type Readiness interface {
	Ready(context.Context) error
}

type Feed interface {
	List(context.Context, string, int, string) (application.FeedPage, error)
	UnreadCount(context.Context, string) (int, error)
	MarkRead(context.Context, string, string) error
	MarkAllRead(context.Context, string) (int64, error)
}

type Verifier interface {
	Verify(string) (auth.Principal, error)
}

type principalKey struct{}

type Server struct {
	readiness Readiness
	feed      Feed
	verifier  Verifier
}

func New(readiness Readiness, feed Feed, verifier Verifier) http.Handler {
	server := &Server{readiness: readiness, feed: feed, verifier: verifier}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /health/ready", server.ready)
	mux.Handle("GET /api/v1/notifications", server.protect(http.HandlerFunc(server.list)))
	mux.Handle("GET /api/v1/notifications/unread-count", server.protect(http.HandlerFunc(server.unreadCount)))
	mux.Handle("PATCH /api/v1/notifications/{notificationId}/read", server.protect(http.HandlerFunc(server.markRead)))
	mux.Handle("POST /api/v1/notifications/read-all", server.protect(http.HandlerFunc(server.markAllRead)))
	return mux
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.readiness.Ready(ctx); err != nil {
		problem(w, r, http.StatusServiceUnavailable, "NOTIFICATION_NOT_READY", "Notification service is not ready.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			problem(w, r, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "A valid access token is required.")
			return
		}
		requestPrincipal, err := s.verifier.Verify(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			problem(w, r, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "A valid access token is required.")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, requestPrincipal)))
	})
}

func principal(r *http.Request) auth.Principal {
	return r.Context().Value(principalKey{}).(auth.Principal)
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			problem(w, r, http.StatusBadRequest, "INVALID_PAGINATION", "Notification pagination is invalid.")
			return
		}
		limit = parsed
	}
	page, err := s.feed.List(r.Context(), principal(r).Subject, limit, r.URL.Query().Get("cursor"))
	if errors.Is(err, application.ErrInvalidCursor) {
		problem(w, r, http.StatusBadRequest, "INVALID_PAGINATION", "Notification pagination is invalid.")
		return
	}
	if err != nil {
		problem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Notifications could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) unreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := s.feed.UnreadCount(r.Context(), principal(r).Subject)
	if err != nil {
		problem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Unread notifications could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"unreadCount": count})
}

func (s *Server) markRead(w http.ResponseWriter, r *http.Request) {
	err := s.feed.MarkRead(r.Context(), principal(r).Subject, r.PathValue("notificationId"))
	if errors.Is(err, application.ErrNotificationMissing) {
		problem(w, r, http.StatusNotFound, "NOTIFICATION_NOT_FOUND", "Notification was not found.")
		return
	}
	if err != nil {
		problem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Notification could not be updated.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) markAllRead(w http.ResponseWriter, r *http.Request) {
	count, err := s.feed.MarkAllRead(r.Context(), principal(r).Subject)
	if err != nil {
		problem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Notifications could not be updated.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"updatedCount": count})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func problem(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":     "https://matchmate.example/problems/" + strings.ToLower(strings.ReplaceAll(code, "_", "-")),
		"title":    code,
		"status":   status,
		"code":     code,
		"detail":   detail,
		"instance": r.URL.Path,
		"traceId":  uuid.NewString(),
	})
}
