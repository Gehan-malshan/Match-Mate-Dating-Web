package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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
	s.mux.HandleFunc("GET /api/v1/auth/google/start", s.googleStart)
	s.mux.HandleFunc("GET /api/v1/auth/google/callback", s.googleCallback)
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
	s.mux.HandleFunc("GET /api/v1/internal/notification-recipients/{accountId}", s.notificationRecipient)
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
}
type googleUserInfo struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func (s *Server) googleConfigured() bool {
	return s.cfg.GoogleClientID != "" && s.cfg.GoogleClientSecret != "" && s.cfg.GoogleRedirectURL != ""
}

func (s *Server) googleStart(w http.ResponseWriter, r *http.Request) {
	if !s.googleConfigured() {
		s.googleFailure(w, r, "GOOGLE_LOGIN_NOT_CONFIGURED")
		return
	}
	state, err := randomState()
	if err != nil {
		problem(w, r, 500, "INTERNAL_ERROR", "Could not start Google sign in", nil)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "matchmate_google_state", Value: state, Path: "/api/v1/auth/google", HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	query := url.Values{"client_id": {s.cfg.GoogleClientID}, "redirect_uri": {s.cfg.GoogleRedirectURL}, "response_type": {"code"}, "scope": {"openid email profile"}, "state": {state}, "prompt": {"select_account"}}
	http.Redirect(w, r, "https://accounts.google.com/o/oauth2/v2/auth?"+query.Encode(), http.StatusFound)
}

func (s *Server) googleCallback(w http.ResponseWriter, r *http.Request) {
	if !s.googleConfigured() {
		s.googleFailure(w, r, "GOOGLE_LOGIN_NOT_CONFIGURED")
		return
	}
	cookie, err := r.Cookie("matchmate_google_state")
	if err != nil || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(r.URL.Query().Get("state"))) != 1 {
		s.googleFailure(w, r, "GOOGLE_STATE_INVALID")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "matchmate_google_state", Value: "", Path: "/api/v1/auth/google", HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	if r.URL.Query().Get("error") != "" {
		s.googleFailure(w, r, "GOOGLE_LOGIN_CANCELLED")
		return
	}
	user, err := s.googleUser(r.Context(), r.URL.Query().Get("code"))
	if err != nil || !user.EmailVerified || user.Email == "" || user.Subject == "" {
		s.googleFailure(w, r, "GOOGLE_IDENTITY_INVALID")
		return
	}
	pair, err := s.app.LoginWithGoogle(r.Context(), user.Email)
	if err != nil {
		var p *domain.ProblemError
		if errors.As(err, &p) {
			s.googleFailure(w, r, p.Code)
			return
		}
		s.googleFailure(w, r, "GOOGLE_LOGIN_FAILED")
		return
	}
	s.setSession(w, pair)
	http.Redirect(w, r, s.cfg.GoogleSuccessRedirectURL, http.StatusFound)
}

func (s *Server) googleUser(ctx context.Context, code string) (googleUserInfo, error) {
	if strings.TrimSpace(code) == "" {
		return googleUserInfo{}, errors.New("authorization code is required")
	}
	form := url.Values{"code": {code}, "client_id": {s.cfg.GoogleClientID}, "client_secret": {s.cfg.GoogleClientSecret}, "redirect_uri": {s.cfg.GoogleRedirectURL}, "grant_type": {"authorization_code"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return googleUserInfo{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return googleUserInfo{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return googleUserInfo{}, errors.New("google token exchange failed")
	}
	var token googleTokenResponse
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&token); err != nil || token.AccessToken == "" {
		return googleUserInfo{}, errors.New("google token response invalid")
	}
	profileRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return googleUserInfo{}, err
	}
	profileRequest.Header.Set("Authorization", "Bearer "+token.AccessToken)
	profile, err := client.Do(profileRequest)
	if err != nil {
		return googleUserInfo{}, err
	}
	defer profile.Body.Close()
	if profile.StatusCode != http.StatusOK {
		return googleUserInfo{}, errors.New("google userinfo request failed")
	}
	var user googleUserInfo
	if err = json.NewDecoder(io.LimitReader(profile.Body, 1<<20)).Decode(&user); err != nil {
		return googleUserInfo{}, err
	}
	return user, nil
}

func (s *Server) googleFailure(w http.ResponseWriter, r *http.Request, code string) {
	separator := "?"
	if strings.Contains(s.cfg.GoogleSuccessRedirectURL, "?") {
		separator = "&"
	}
	http.Redirect(w, r, s.cfg.GoogleSuccessRedirectURL+separator+"googleError="+url.QueryEscape(code), http.StatusFound)
}

func randomState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (s *Server) notificationRecipient(w http.ResponseWriter, r *http.Request) {
	if s.cfg.InternalServiceToken == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-MatchMate-Internal-Token")), []byte(s.cfg.InternalServiceToken)) != 1 {
		problem(w, r, 401, "INTERNAL_AUTH_REQUIRED", "Internal service authorization is required", nil)
		return
	}
	a, err := s.app.NotificationRecipient(r.Context(), r.PathValue("accountId"))
	if handle(w, r, err) {
		return
	}
	write(w, 200, map[string]string{"email": a.Email})
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
