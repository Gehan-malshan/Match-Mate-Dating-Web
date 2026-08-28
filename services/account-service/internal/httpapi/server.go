package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gehan-malshan/matchmate/account-service/internal/application"
	"github.com/gehan-malshan/matchmate/account-service/internal/auth"
	"github.com/gehan-malshan/matchmate/account-service/internal/config"
	"github.com/gehan-malshan/matchmate/account-service/internal/domain"
	"github.com/google/uuid"
)

type Server struct {
	app     *application.Service
	cfg     config.Config
	log     *slog.Logger
	mux     *http.ServeMux
	limits  map[string]rateWindow
	limitMu sync.Mutex
}
type rateWindow struct {
	Started time.Time
	Count   int
}
type principalKey struct{}
type correlationKey struct{}

func New(app *application.Service, cfg config.Config, log *slog.Logger) http.Handler {
	s := &Server{app: app, cfg: cfg, log: log, mux: http.NewServeMux(), limits: map[string]rateWindow{}}
	s.routes()
	return s.middleware(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	s.mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := s.app.Ready(r.Context()); err != nil {
			problem(w, r, 503, "DEPENDENCY_UNAVAILABLE", "Database is unavailable", nil)
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	})
	s.mux.HandleFunc("GET /.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) { write(w, 200, s.app.JWK()) })
	s.mux.Handle("POST /api/v1/auth/register", s.authLimited(http.HandlerFunc(s.register)))
	s.mux.HandleFunc("POST /api/v1/auth/verify-email", s.verify)
	s.mux.Handle("POST /api/v1/auth/login", s.authLimited(http.HandlerFunc(s.login)))
	s.mux.Handle("POST /api/v1/auth/refresh", s.authLimited(http.HandlerFunc(s.refresh)))
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	s.mux.Handle("GET /api/v1/users/me", s.protected(http.HandlerFunc(s.me)))
	s.mux.Handle("PATCH /api/v1/users/me/profile", s.protected(http.HandlerFunc(s.updateProfile)))
	s.mux.Handle("PUT /api/v1/users/me/matching-preferences", s.protected(http.HandlerFunc(s.preferences)))
	s.mux.Handle("DELETE /api/v1/users/me", s.protected(http.HandlerFunc(s.deactivate)))
	s.mux.Handle("GET /api/v1/community/profiles", s.protected(http.HandlerFunc(s.community)))
	s.mux.Handle("GET /api/v1/community/profiles/{accountId}", s.protected(http.HandlerFunc(s.communityOne)))
	s.mux.Handle("POST /api/v1/users/me/blocks", s.protected(http.HandlerFunc(s.block)))
	s.mux.Handle("DELETE /api/v1/users/me/blocks/{accountId}", s.protected(http.HandlerFunc(s.unblock)))
	s.mux.Handle("POST /api/v1/admin/profiles/{accountId}/decision", s.protected(s.roles("moderator", "admin", http.HandlerFunc(s.moderate))))
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in domain.RegisterInput
	if !decode(w, r, &in) {
		return
	}
	v, err := s.app.Register(r.Context(), in, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 201, v)
}
func (s *Server) verify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := s.app.Verify(r.Context(), in.Token, correlation(r))
	if handle(w, r, err) {
		return
	}
	write(w, 200, v)
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := s.app.Login(r.Context(), in.Email, in.Password)
	if handle(w, r, err) {
		return
	}
	s.setSession(w, v)
	write(w, 200, map[string]any{"accessToken": v.AccessToken, "accessExpiresAt": v.AccessExpiresAt})
}
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	if !s.trustedOrigin(r) {
		problem(w, r, 403, "ORIGIN_NOT_ALLOWED", "Request origin is not allowed", nil)
		return
	}
	c, _ := r.Cookie("matchmate_refresh")
	raw := ""
	if c != nil {
		raw = c.Value
	}
	v, err := s.app.Refresh(r.Context(), raw)
	if handle(w, r, err) {
		s.clearSession(w)
		return
	}
	s.setSession(w, v)
	write(w, 200, map[string]any{"accessToken": v.AccessToken, "accessExpiresAt": v.AccessExpiresAt})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !s.trustedOrigin(r) {
		problem(w, r, 403, "ORIGIN_NOT_ALLOWED", "Request origin is not allowed", nil)
		return
	}
	c, _ := r.Cookie("matchmate_refresh")
	if c != nil {
		_ = s.app.Logout(r.Context(), c.Value)
	}
	s.clearSession(w)
	w.WriteHeader(204)
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	v, err := s.app.GetMe(r.Context(), principal(r).Subject)
	if handle(w, r, err) {
		return
	}
	write(w, 200, v)
}
func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var in domain.ProfilePatch
	if !decode(w, r, &in) {
		return
	}
	v, err := s.app.UpdateProfile(r.Context(), principal(r).Subject, correlation(r), in)
	if handle(w, r, err) {
		return
	}
	write(w, 200, v)
}
func (s *Server) preferences(w http.ResponseWriter, r *http.Request) {
	var in domain.PreferenceInput
	if !decode(w, r, &in) {
		return
	}
	v, err := s.app.ReplacePreferences(r.Context(), principal(r).Subject, in)
	if handle(w, r, err) {
		return
	}
	write(w, 200, v)
}
func (s *Server) deactivate(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Deactivate(r.Context(), principal(r).Subject, correlation(r)); handle(w, r, err) {
		return
	}
	s.clearSession(w)
	w.WriteHeader(204)
}
func (s *Server) community(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, next, err := s.app.ListCommunity(r.Context(), principal(r).Subject, r.URL.Query().Get("cursor"), limit)
	if handle(w, r, err) {
		return
	}
	write(w, 200, map[string]any{"items": items, "nextCursor": next})
}
func (s *Server) communityOne(w http.ResponseWriter, r *http.Request) {
	v, err := s.app.GetCommunity(r.Context(), principal(r).Subject, r.PathValue("accountId"))
	if handle(w, r, err) {
		return
	}
	write(w, 200, v)
}
func (s *Server) block(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AccountID string `json:"accountId"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.app.Block(r.Context(), principal(r).Subject, in.AccountID, correlation(r)); handle(w, r, err) {
		return
	}
	w.WriteHeader(204)
}
func (s *Server) unblock(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Unblock(r.Context(), principal(r).Subject, r.PathValue("accountId"), correlation(r)); handle(w, r, err) {
		return
	}
	w.WriteHeader(204)
}
func (s *Server) moderate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.app.Moderate(r.Context(), principal(r).Subject, r.PathValue("accountId"), strings.ToUpper(in.Decision), in.Reason, correlation(r)); handle(w, r, err) {
		return
	}
	w.WriteHeader(204)
}

func (s *Server) protected(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			problem(w, r, 401, "AUTHENTICATION_REQUIRED", "Bearer access token is required", nil)
			return
		}
		c, err := s.app.Authenticate(r.Context(), raw)
		if handle(w, r, err) {
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, c)))
	})
}
func (s *Server) authLimited(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.RemoteAddr
		if i := strings.LastIndex(host, ":"); i > 0 {
			host = host[:i]
		}
		key := host + "|" + r.URL.Path
		now := time.Now()
		s.limitMu.Lock()
		window := s.limits[key]
		if window.Started.IsZero() || now.Sub(window.Started) >= time.Minute {
			window = rateWindow{Started: now}
		}
		window.Count++
		s.limits[key] = window
		s.limitMu.Unlock()
		if window.Count > 20 {
			w.Header().Set("Retry-After", "60")
			problem(w, r, 429, "RATE_LIMITED", "Too many authentication attempts", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) roles(a, b string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, v := range principal(r).Roles {
			if v == a || v == b {
				next.ServeHTTP(w, r)
				return
			}
		}
		problem(w, r, 403, "FORBIDDEN", "Required role is missing", nil)
	})
}
func principal(r *http.Request) *auth.Claims { return r.Context().Value(principalKey{}).(*auth.Claims) }
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
		if s.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Correlation-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		ctx := context.WithValue(r.Context(), correlationKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
		s.log.Info("http_request", "method", r.Method, "path", r.URL.Path, "correlation_id", id, "duration_ms", time.Since(started).Milliseconds())
	})
}
func (s *Server) originAllowed(origin string) bool {
	for _, v := range s.cfg.AllowedOrigins {
		if origin == v {
			return true
		}
	}
	return false
}
func (s *Server) trustedOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return (origin == "" && s.cfg.Environment != "production") || s.originAllowed(origin)
}
func (s *Server) setSession(w http.ResponseWriter, v domain.TokenPair) {
	http.SetCookie(w, &http.Cookie{Name: "matchmate_refresh", Value: v.RefreshToken, Path: "/api/v1/auth", HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, Expires: v.RefreshExpiresAt, MaxAge: int(time.Until(v.RefreshExpiresAt).Seconds())})
}
func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "matchmate_refresh", Value: "", Path: "/api/v1/auth", HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
}
func decode(w http.ResponseWriter, r *http.Request, d any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(d); err != nil {
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
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://docs.matchmate.local/problems/" + strings.ToLower(code), "title": code, "status": status, "detail": detail, "code": code, "correlationId": correlation(r), "fields": fields})
}
