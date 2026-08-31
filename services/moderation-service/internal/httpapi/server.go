package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/application"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/config"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/domain"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type verifier interface {
	Verify(string) (domain.Principal, error)
}
type Server struct {
	app  *application.Service
	auth verifier
	cfg  config.Config
	log  *slog.Logger
	mux  *http.ServeMux
}
type principalKey struct{}
type correlationKey struct{}
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func New(app *application.Service, auth verifier, cfg config.Config, log *slog.Logger) http.Handler {
	s := &Server{app: app, auth: auth, cfg: cfg, log: log, mux: http.NewServeMux()}
	s.routes()
	return s.middleware(s.mux)
}
func (s *Server) routes() {
	s.mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })
	s.mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		if s.app.Ready(r.Context()) != nil {
			problem(w, r, 503, "DEPENDENCY_UNAVAILABLE", "Database is unavailable", nil)
			return
		}
		w.WriteHeader(204)
	})
	s.mux.Handle("POST /api/v1/reports", s.protected(http.HandlerFunc(s.createReport)))
	s.mux.Handle("GET /api/v1/reports/mine", s.protected(http.HandlerFunc(s.listMine)))
	s.mux.Handle("GET /api/v1/moderation/cases", s.protected(http.HandlerFunc(s.listCases)))
	s.mux.Handle("GET /api/v1/moderation/cases/{caseId}", s.protected(http.HandlerFunc(s.getCase)))
	s.mux.Handle("POST /api/v1/moderation/cases/{caseId}/assign", s.protected(http.HandlerFunc(s.assign)))
	s.mux.Handle("POST /api/v1/moderation/cases/{caseId}/status", s.protected(http.HandlerFunc(s.caseStatus)))
	s.mux.Handle("POST /api/v1/moderation/cases/{caseId}/actions", s.protected(http.HandlerFunc(s.action)))
	s.mux.Handle("POST /api/v1/moderation/actions/{actionId}/appeals", s.protected(http.HandlerFunc(s.appeal)))
	s.mux.Handle("POST /api/v1/moderation/appeals/{appealId}/decision", s.protected(http.HandlerFunc(s.decision)))
}
func (s *Server) createReport(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateReportInput
	if !decode(w, r, &in) {
		return
	}
	value, err := s.app.CreateReport(r.Context(), principal(r), in, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 201, value)
}
func pagination(r *http.Request) (string, int) {
	size, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return r.URL.Query().Get("cursor"), size
}
func (s *Server) listMine(w http.ResponseWriter, r *http.Request) {
	cursor, size := pagination(r)
	value, err := s.app.ListMine(r.Context(), principal(r), cursor, size)
	if handle(w, r, err) {
		return
	}
	write(w, 200, value)
}
func (s *Server) listCases(w http.ResponseWriter, r *http.Request) {
	cursor, size := pagination(r)
	value, err := s.app.ListCases(r.Context(), principal(r), cursor, size)
	if handle(w, r, err) {
		return
	}
	write(w, 200, value)
}
func (s *Server) getCase(w http.ResponseWriter, r *http.Request) {
	value, err := s.app.GetCase(r.Context(), principal(r), r.PathValue("caseId"), correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, value)
}
func (s *Server) assign(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AssigneeID string `json:"assigneeId"`
		Reason     string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	value, err := s.app.Assign(r.Context(), principal(r), r.PathValue("caseId"), in.AssigneeID, in.Reason, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, value)
}
func (s *Server) caseStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	value, err := s.app.UpdateCaseStatus(r.Context(), principal(r), r.PathValue("caseId"), in.Status, in.Reason, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, value)
}
func (s *Server) action(w http.ResponseWriter, r *http.Request) {
	var in domain.Action
	if !decode(w, r, &in) {
		return
	}
	value, err := s.app.ApplyAction(r.Context(), principal(r), r.PathValue("caseId"), in, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 201, value)
}
func (s *Server) appeal(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	value, err := s.app.Appeal(r.Context(), principal(r), r.PathValue("actionId"), in.Reason, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 201, value)
}
func (s *Server) decision(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	value, err := s.app.Decide(r.Context(), principal(r), r.PathValue("appealId"), in.Decision, in.Reason, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, value)
}
func (s *Server) protected(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			problem(w, r, 401, "AUTHENTICATION_REQUIRED", "Bearer access token is required", nil)
			return
		}
		p, err := s.auth.Verify(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
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
	value, _ := r.Context().Value(correlationKey{}).(string)
	return value
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		response := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		id := strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
		if id == "" || len(id) > 100 {
			id = uuid.NewString()
		}
		response.Header().Set("X-Correlation-ID", id)
		origin := r.Header.Get("Origin")
		for _, allowed := range s.cfg.AllowedOrigins {
			if origin == allowed {
				response.Header().Set("Access-Control-Allow-Origin", origin)
				response.Header().Set("Vary", "Origin")
			}
		}
		response.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Correlation-ID")
		response.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			response.WriteHeader(204)
			return
		}
		request := r.WithContext(context.WithValue(r.Context(), correlationKey{}, id))
		next.ServeHTTP(response, request)
		route := request.Pattern
		if route == "" {
			route = "unmatched"
		}
		s.log.Info("http_request", "method", r.Method, "route", route, "result_code", response.status, "correlation_id", id, "duration_ms", time.Since(started).Milliseconds())
	})
}
func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil {
		problem(w, r, 400, "INVALID_JSON", "Request body is invalid", nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
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
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://matchmate.example/problems/" + strings.ToLower(strings.ReplaceAll(code, "_", "-")), "title": code, "status": status, "code": code, "detail": detail, "instance": r.URL.Path, "traceId": correlation(r), "fieldErrors": fields})
}
