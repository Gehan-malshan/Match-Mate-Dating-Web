package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/config"
	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/domain"
	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/store/postgres"
	"github.com/google/uuid"
)

type tokenVerifier interface {
	Verify(string) (domain.Principal, error)
}
type service interface {
	Ready(context.Context) error
	Generate(context.Context, domain.Principal, string, string, string) (domain.Run, error)
	List(context.Context, domain.Principal, string) ([]domain.Run, error)
	Get(context.Context, domain.Principal, string) (domain.Run, error)
	Review(context.Context, domain.Principal, string, int64, string) (domain.Run, error)
	Override(context.Context, domain.Principal, string, string, string, string, string, string, string, int64) (domain.Run, error)
	Lock(context.Context, domain.Principal, string, int64, string, string) (domain.Run, error)
	Publish(context.Context, domain.Principal, string, int64, string) (domain.Run, error)
	Mine(context.Context, domain.Principal) ([]postgres.MemberMatch, error)
	Respond(context.Context, domain.Principal, string, string, string, string, string) error
	Consent(context.Context, domain.Principal, string, string, string, string, string) error
	Feedback(context.Context, domain.Principal, string, int, int, bool, string, string) error
}
type Server struct {
	app  service
	auth tokenVerifier
	cfg  config.Config
	log  *slog.Logger
	mux  *http.ServeMux
}
type principalKey struct{}
type correlationKey struct{}

func New(app service, auth tokenVerifier, cfg config.Config, log *slog.Logger) http.Handler {
	s := &Server{app: app, auth: auth, cfg: cfg, log: log, mux: http.NewServeMux()}
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
	s.mux.Handle("GET /api/v1/events/{eventId}/matching-runs", s.protected(http.HandlerFunc(s.list)))
	s.mux.Handle("POST /api/v1/events/{eventId}/matching-runs", s.protected(http.HandlerFunc(s.generate)))
	s.mux.Handle("GET /api/v1/matching-runs/{runId}", s.protected(http.HandlerFunc(s.get)))
	s.mux.Handle("POST /api/v1/matching-runs/{runId}/review", s.protected(http.HandlerFunc(s.review)))
	s.mux.Handle("POST /api/v1/matching-runs/{runId}/overrides", s.protected(http.HandlerFunc(s.override)))
	s.mux.Handle("POST /api/v1/matching-runs/{runId}/lock", s.protected(http.HandlerFunc(s.lock)))
	s.mux.Handle("POST /api/v1/matching-runs/{runId}/publish", s.protected(http.HandlerFunc(s.publish)))
	s.mux.Handle("GET /api/v1/matches/mine", s.protected(http.HandlerFunc(s.mine)))
	s.mux.Handle("POST /api/v1/matches/{matchId}/response", s.protected(http.HandlerFunc(s.respond)))
	s.mux.Handle("POST /api/v1/matches/{matchId}/reveal-consent", s.protected(http.HandlerFunc(s.consent)))
	s.mux.Handle("POST /api/v1/matches/{matchId}/feedback", s.protected(http.HandlerFunc(s.feedback)))
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	items, err := s.app.List(r.Context(), principal(r), r.PathValue("eventId"))
	if handle(w, r, err) {
		return
	}
	write(w, 200, map[string]any{"items": items})
}
func (s *Server) generate(w http.ResponseWriter, r *http.Request) {
	run, err := s.app.Generate(r.Context(), principal(r), r.PathValue("eventId"), r.Header.Get("Idempotency-Key"), correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 201, run)
}
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	run, err := s.app.Get(r.Context(), principal(r), r.PathValue("runId"))
	if handle(w, r, err) {
		return
	}
	write(w, 200, run)
}
func (s *Server) review(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	if !decode(w, r, &in) {
		return
	}
	run, err := s.app.Review(r.Context(), principal(r), r.PathValue("runId"), in.ExpectedVersion, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, run)
}
func (s *Server) override(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExpectedVersion   int64  `json:"expectedVersion"`
		RemoveSelectionID string `json:"removeSelectionId"`
		ParticipantA      string `json:"participantA"`
		ParticipantB      string `json:"participantB"`
		Reason            string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	run, err := s.app.Override(r.Context(), principal(r), r.PathValue("runId"), in.RemoveSelectionID, in.ParticipantA, in.ParticipantB, in.Reason, r.Header.Get("Idempotency-Key"), correlation(r), in.ExpectedVersion)
	if handle(w, r, err) {
		return
	}
	write(w, 200, run)
}
func (s *Server) lock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	if !decode(w, r, &in) {
		return
	}
	run, err := s.app.Lock(r.Context(), principal(r), r.PathValue("runId"), in.ExpectedVersion, r.Header.Get("Idempotency-Key"), correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, run)
}
func (s *Server) publish(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
		problem(w, r, 400, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key is required", nil)
		return
	}
	var in struct {
		ExpectedVersion int64 `json:"expectedVersion"`
	}
	if !decode(w, r, &in) {
		return
	}
	run, err := s.app.Publish(r.Context(), principal(r), r.PathValue("runId"), in.ExpectedVersion, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, run)
}
func (s *Server) mine(w http.ResponseWriter, r *http.Request) {
	items, err := s.app.Mine(r.Context(), principal(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, map[string]any{"items": items})
}
func (s *Server) respond(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Response        string `json:"response"`
		QuestionVersion string `json:"questionVersion"`
	}
	if !decode(w, r, &in) {
		return
	}
	err := s.app.Respond(r.Context(), principal(r), r.PathValue("matchId"), in.Response, in.QuestionVersion, r.Header.Get("Idempotency-Key"), correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, map[string]string{"status": "recorded"})
}
func (s *Server) consent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Decision      string `json:"decision"`
		PolicyVersion string `json:"policyVersion"`
	}
	if !decode(w, r, &in) {
		return
	}
	err := s.app.Consent(r.Context(), principal(r), r.PathValue("matchId"), in.Decision, in.PolicyVersion, r.Header.Get("Idempotency-Key"), correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, map[string]string{"status": "recorded"})
}
func (s *Server) feedback(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ComfortRating int  `json:"comfortRating"`
		QualityRating int  `json:"qualityRating"`
		SafetyConcern bool `json:"safetyConcern"`
	}
	if !decode(w, r, &in) {
		return
	}
	err := s.app.Feedback(r.Context(), principal(r), r.PathValue("matchId"), in.ComfortRating, in.QualityRating, in.SafetyConcern, r.Header.Get("Idempotency-Key"), correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, map[string]string{"status": "recorded"})
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
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		id := strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
		if id == "" || len(id) > 100 {
			id = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), correlationKey{}, id)
		w.Header().Set("X-Correlation-ID", id)
		origin := r.Header.Get("Origin")
		for _, allowed := range s.cfg.AllowedOrigins {
			if origin == allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Correlation-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
		s.log.Info("http_request", "service", "matchmaking-service", "method", r.Method, "path", r.URL.Path, "correlation_id", id, "duration_ms", time.Since(started).Milliseconds())
	})
}
func principal(r *http.Request) domain.Principal {
	return r.Context().Value(principalKey{}).(domain.Principal)
}
func correlation(r *http.Request) string {
	value, _ := r.Context().Value(correlationKey{}).(string)
	return value
}
func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		problem(w, r, 400, "INVALID_JSON", "Request body is invalid", nil)
		return false
	}
	return true
}
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
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
func parseInt(value string) int { number, _ := strconv.Atoi(value); return number }
