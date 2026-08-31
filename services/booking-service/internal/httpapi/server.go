package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gehan-malshan/matchmate/booking-service/internal/application"
	"github.com/gehan-malshan/matchmate/booking-service/internal/auth"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type Verifier interface {
	Verify(string) (auth.Principal, error)
}
type key int

const principalKey key = 1

type Server struct {
	app  *application.Service
	auth Verifier
	log  *slog.Logger
}

func New(app *application.Service, v Verifier, log *slog.Logger) http.Handler {
	s := &Server{app: app, auth: v, log: log}
	m := http.NewServeMux()
	m.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	m.Handle("POST /api/v1/bookings", s.protect(http.HandlerFunc(s.create)))
	m.Handle("GET /api/v1/bookings/{bookingId}", s.protect(http.HandlerFunc(s.get)))
	m.Handle("GET /api/v1/bookings", s.protect(http.HandlerFunc(s.list)))
	m.Handle("POST /api/v1/bookings/{bookingId}/cancel", s.protect(http.HandlerFunc(s.cancel)))
	m.Handle("GET /internal/v1/bookings/{bookingId}/payment-snapshot", s.protect(http.HandlerFunc(s.snapshot)))
	return m
}
func (s *Server) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		p, err := s.auth.Verify(raw)
		if err != nil {
			problem(w, 401, "AUTHENTICATION_REQUIRED", "A valid access token is required.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, p)))
	})
}
func principal(r *http.Request) auth.Principal {
	return r.Context().Value(principalKey).(auth.Principal)
}
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EventID string `json:"eventId"`
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil {
		problem(w, 400, "INVALID_REQUEST", "Request body is invalid.")
		return
	}
	b, err := s.app.Create(r.Context(), principal(r).Subject, in.EventID, r.Header.Get("Idempotency-Key"))
	if handle(w, err) {
		return
	}
	write(w, 201, b)
}
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	b, err := s.app.Get(r.Context(), principal(r).Subject, r.PathValue("bookingId"))
	if handle(w, err) {
		return
	}
	write(w, 200, b)
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	items, err := s.app.List(r.Context(), principal(r).Subject, 50)
	if handle(w, err) {
		return
	}
	write(w, 200, map[string]any{"items": items})
}
func (s *Server) cancel(w http.ResponseWriter, r *http.Request) {
	b, err := s.app.Cancel(r.Context(), principal(r).Subject, r.PathValue("bookingId"), r.Header.Get("Idempotency-Key"))
	if handle(w, err) {
		return
	}
	write(w, 200, b)
}
func (s *Server) snapshot(w http.ResponseWriter, r *http.Request) {
	b, err := s.app.Snapshot(r.Context(), principal(r).Subject, r.PathValue("bookingId"))
	if handle(w, err) {
		return
	}
	write(w, 200, map[string]any{"bookingId": b.ID, "accountId": principal(r).Subject, "amount": b.Amount, "currency": b.Currency, "status": b.State, "expiresAt": b.ExpiresAt})
}
func handle(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	status := 500
	code := "INTERNAL_ERROR"
	detail := "An unexpected error occurred."
	if errors.Is(err, application.ErrNotFound) {
		status = 404
		code = "BOOKING_NOT_FOUND"
		detail = "Booking was not found."
	}
	if errors.Is(err, application.ErrCapacity) {
		status = 409
		code = "BOOKING_CAPACITY_EXHAUSTED"
		detail = "Event capacity is no longer available."
	}
	if errors.Is(err, application.ErrConflict) {
		status = 409
		code = "BOOKING_STATE_CONFLICT"
		detail = "Booking cannot be changed in its current state."
	}
	if errors.Is(err, application.ErrInvalid) {
		status = 400
		code = "INVALID_REQUEST"
		detail = "The request is invalid."
	}
	if errors.Is(err, application.ErrDependency) {
		status = 503
		code = "DEPENDENCY_UNAVAILABLE"
		detail = "A required service is unavailable."
	}
	problem(w, status, code, detail)
	return true
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://matchmate.example/problems/" + strings.ToLower(strings.ReplaceAll(code, "_", "-")), "title": code, "status": status, "code": code, "detail": detail, "traceId": uuid.NewString()})
}
