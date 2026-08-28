package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gehan-malshan/matchmate/event-service/internal/application"
	"github.com/gehan-malshan/matchmate/event-service/internal/config"
	"github.com/gehan-malshan/matchmate/event-service/internal/domain"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type tokenVerifier interface {
	Verify(string) (domain.Principal, error)
}
type Server struct {
	app  *application.Service
	auth tokenVerifier
	cfg  config.Config
	log  *slog.Logger
	mux  *http.ServeMux
}
type principalKey struct{}
type correlationKey struct{}

func New(app *application.Service, a tokenVerifier, cfg config.Config, log *slog.Logger) http.Handler {
	s := &Server{app: app, auth: a, cfg: cfg, log: log, mux: http.NewServeMux()}
	s.routes()
	return s.middleware(s.mux)
}
func (s *Server) routes() {
	s.mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	s.mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		if s.app.Ready(r.Context()) != nil {
			problem(w, r, 503, "DEPENDENCY_UNAVAILABLE", "Database is unavailable", nil)
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	})
	s.mux.HandleFunc("GET /api/v1/events", s.list)
	s.mux.HandleFunc("GET /api/v1/events/{eventId}", s.get)
	s.mux.Handle("GET /api/v1/organizer/events", s.protected(http.HandlerFunc(s.managed)))
	s.mux.Handle("POST /api/v1/events", s.protected(http.HandlerFunc(s.create)))
	s.mux.Handle("PATCH /api/v1/events/{eventId}", s.protected(http.HandlerFunc(s.update)))
	for path, status := range map[string]domain.Status{"publish": domain.Published, "open-registration": domain.RegistrationOpen, "close-registration": domain.RegistrationClosed, "cancel": domain.Cancelled} {
		to := status
		s.mux.Handle("POST /api/v1/events/{eventId}/"+path, s.protected(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.transition(w, r, to) })))
	}
}
func (s *Server) managed(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	p, err := s.app.ListManaged(r.Context(), principal(r), r.URL.Query().Get("cursor"), limit)
	if handle(w, r, err) {
		return
	}
	write(w, 200, p)
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	p, err := s.app.List(r.Context(), r.URL.Query().Get("cursor"), limit)
	if handle(w, r, err) {
		return
	}
	items := make([]domain.PublicEvent, 0, len(p.Items))
	for _, e := range p.Items {
		items = append(items, e.Public())
	}
	write(w, 200, map[string]any{"items": items, "nextCursor": p.NextCursor, "limit": p.Limit})
}
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	e, err := s.app.Get(r.Context(), r.PathValue("eventId"))
	if handle(w, r, err) {
		return
	}
	if e.Status != domain.Published && e.Status != domain.RegistrationOpen && e.Status != domain.RegistrationClosed {
		problem(w, r, 404, "EVENT_NOT_FOUND", "Event was not found", nil)
		return
	}
	write(w, 200, e.Public())
}
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateInput
	if !decode(w, r, &in) {
		return
	}
	e, err := s.app.Create(r.Context(), principal(r), in, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 201, e)
}
func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	var in domain.UpdateInput
	if !decode(w, r, &in) {
		return
	}
	e, err := s.app.Update(r.Context(), principal(r), r.PathValue("eventId"), in, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, e)
}
func (s *Server) transition(w http.ResponseWriter, r *http.Request, to domain.Status) {
	var in struct {
		ExpectedVersion int64  `json:"expectedVersion"`
		Reason          string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	e, err := s.app.Transition(r.Context(), principal(r), r.PathValue("eventId"), in.ExpectedVersion, to, in.Reason, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, e)
}
func (s *Server) protected(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if raw == "" {
			problem(w, r, 401, "AUTHENTICATION_REQUIRED", "Bearer access token is required", nil)
			return
		}
		p, err := s.auth.Verify(raw)
		if err != nil {
			problem(w, r, 401, "INVALID_ACCESS_TOKEN", "Access token is invalid or expired", nil)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
	})
}
func principal(r *http.Request) domain.Principal {
	return r.Context().Value(principalKey{}).(domain.Principal)
}
func correlation(r *http.Request) string {
	v, _ := r.Context().Value(correlationKey{}).(string)
	return v
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		id := strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
		if id == "" || len(id) > 100 {
			id = uuid.NewString()
		}
		w.Header().Set("X-Correlation-ID", id)
		origin := r.Header.Get("Origin")
		for _, o := range s.cfg.AllowedOrigins {
			if origin == o {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Correlation-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), correlationKey{}, id)))
		s.log.Info("http_request", "service", "event-service", "method", r.Method, "path", r.URL.Path, "correlation_id", id, "duration_ms", time.Since(started).Milliseconds())
	})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		problem(w, r, 400, "INVALID_JSON", "Request body is invalid", nil)
		return false
	}
	return true
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func handle(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	var p *domain.ProblemError
	if errors.As(err, &p) {
		problem(w, r, p.Status, p.Code, p.Detail, p.Fields)
	} else {
		problem(w, r, 500, "INTERNAL_ERROR", "An unexpected error occurred", nil)
	}
	return true
}
func problem(w http.ResponseWriter, r *http.Request, status int, code, detail string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://docs.matchmate.local/problems/" + strings.ToLower(code), "title": code, "status": status, "code": code, "detail": detail, "instance": r.URL.Path, "traceId": correlation(r), "fieldErrors": fields})
}
