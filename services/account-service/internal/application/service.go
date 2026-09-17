package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/account-service/internal/auth"
	"github.com/gehan-malshan/matchmate/account-service/internal/domain"
	"github.com/gehan-malshan/matchmate/account-service/internal/store"
	"github.com/google/uuid"
)

type Service struct {
	store                                  store.Repository
	tokens                                 *auth.Manager
	accessTTL, refreshTTL, verificationTTL time.Duration
	minimumAge                             int
	consentVersion                         string
	exposeVerification                     bool
	now                                    func() time.Time
}

func New(repo store.Repository, tokens *auth.Manager, access, refresh, verification time.Duration, minimumAge int, consentVersion string, expose bool) *Service {
	return &Service{store: repo, tokens: tokens, accessTTL: access, refreshTTL: refresh, verificationTTL: verification, minimumAge: minimumAge, consentVersion: consentVersion, exposeVerification: expose, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Register(ctx context.Context, in domain.RegisterInput, correlation string) (domain.Registration, error) {
	var err error
	in.Email, err = domain.NormalizeEmail(in.Email)
	if err != nil {
		return domain.Registration{}, err
	}
	in.Nickname = strings.TrimSpace(in.Nickname)
	if fields := domain.ValidateRegistration(in, s.minimumAge, s.now()); len(fields) > 0 {
		return domain.Registration{}, problem(422, "VALIDATION_FAILED", "Registration is invalid", fields)
	}
	if in.ConsentVersion != s.consentVersion {
		return domain.Registration{}, problem(422, "CONSENT_VERSION_OUTDATED", "Reload and accept the current privacy terms", map[string]string{"consentVersion": "outdated"})
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return domain.Registration{}, err
	}
	raw, tokenHash, err := auth.NewOpaqueToken()
	if err != nil {
		return domain.Registration{}, err
	}
	me, err := s.store.Register(ctx, in, hash, tokenHash, s.now().Add(s.verificationTTL), event("AccountRegistered", correlation, ""))
	if errors.Is(err, store.ErrConflict) {
		return domain.Registration{}, problem(409, "EMAIL_ALREADY_REGISTERED", "An account already uses this email", nil)
	}
	if err != nil {
		return domain.Registration{}, err
	}
	if !s.exposeVerification {
		raw = ""
	}
	return domain.Registration{Me: me, VerificationToken: raw}, nil
}

func (s *Service) Verify(ctx context.Context, raw, correlation string) (domain.Account, error) {
	if strings.TrimSpace(raw) == "" {
		return domain.Account{}, problem(422, "TOKEN_REQUIRED", "Verification token is required", nil)
	}
	a, err := s.store.VerifyEmail(ctx, auth.HashOpaqueToken(raw), s.now(), event("AccountVerified", correlation, ""))
	if errors.Is(err, store.ErrInvalidToken) {
		return a, problem(400, "INVALID_OR_EXPIRED_TOKEN", "Verification token is invalid or expired", nil)
	}
	return a, err
}

func (s *Service) Login(ctx context.Context, email, password string) (domain.TokenPair, error) {
	normalized, normErr := domain.NormalizeEmail(email)
	if normErr != nil {
		return domain.TokenPair{}, problem(401, "INVALID_CREDENTIALS", "Email or password is incorrect", nil)
	}
	c, err := s.store.CredentialsByEmail(ctx, normalized)
	valid := false
	if err == nil {
		valid, _ = auth.VerifyPassword(password, c.PasswordHash)
	}
	if err != nil || !valid {
		return domain.TokenPair{}, problem(401, "INVALID_CREDENTIALS", "Email or password is incorrect", nil)
	}
	if c.Account.Status != domain.AccountActive {
		return domain.TokenPair{}, problem(403, "ACCOUNT_UNAVAILABLE", "Account is not active", nil)
	}
	if c.Account.Verification != domain.VerificationVerified {
		return domain.TokenPair{}, problem(403, "EMAIL_NOT_VERIFIED", "Verify your email before signing in", nil)
	}
	c.Account.TokenVersion = c.TokenVersion
	return s.newSession(ctx, c.Account, "", s.now())
}

// LoginWithGoogle signs in an existing verified MatchMate account after the
// Account HTTP adapter has verified Google's OAuth identity response. Google is
// an additional authentication method; registration still collects the DOB and
// consent records required by MatchMate's safety rules.
func (s *Service) LoginWithGoogle(ctx context.Context, email string) (domain.TokenPair, error) {
	normalized, err := domain.NormalizeEmail(email)
	if err != nil {
		return domain.TokenPair{}, problem(401, "GOOGLE_IDENTITY_INVALID", "Google identity is invalid", nil)
	}
	a, err := s.store.AccountByEmail(ctx, normalized)
	if errors.Is(err, store.ErrNotFound) {
		return domain.TokenPair{}, problem(403, "GOOGLE_ACCOUNT_NOT_FOUND", "Create and verify a MatchMate profile before using Google sign in", nil)
	}
	if err != nil {
		return domain.TokenPair{}, err
	}
	if a.Status != domain.AccountActive {
		return domain.TokenPair{}, problem(403, "ACCOUNT_UNAVAILABLE", "Account is not active", nil)
	}
	if a.Verification != domain.VerificationVerified {
		return domain.TokenPair{}, problem(403, "EMAIL_NOT_VERIFIED", "Verify your MatchMate email before using Google sign in", nil)
	}
	return s.newSession(ctx, a, "", s.now())
}

// NotificationRecipient returns a verified active email only to the narrowly
// authorized internal delivery adapter. It is never emitted in public events.
func (s *Service) NotificationRecipient(ctx context.Context, id string) (domain.Account, error) {
	a, err := s.store.AccountByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return domain.Account{}, problem(404, "NOTIFICATION_RECIPIENT_UNAVAILABLE", "Recipient is unavailable", nil)
	}
	if err != nil {
		return domain.Account{}, err
	}
	if a.Status != domain.AccountActive || a.Verification != domain.VerificationVerified {
		return domain.Account{}, problem(404, "NOTIFICATION_RECIPIENT_UNAVAILABLE", "Recipient is unavailable", nil)
	}
	return a, nil
}

func (s *Service) Refresh(ctx context.Context, raw string) (domain.TokenPair, error) {
	if raw == "" {
		return domain.TokenPair{}, problem(401, "REFRESH_REQUIRED", "Refresh session is required", nil)
	}
	nextRaw, nextHash, err := auth.NewOpaqueToken()
	if err != nil {
		return domain.TokenPair{}, err
	}
	now := s.now()
	next := domain.Session{ID: uuid.NewString(), TokenHash: nextHash, ExpiresAt: now.Add(s.refreshTTL)}
	a, err := s.store.RotateSession(ctx, auth.HashOpaqueToken(raw), next, now)
	if errors.Is(err, store.ErrRefreshReuse) {
		return domain.TokenPair{}, problem(401, "SESSION_REUSE_DETECTED", "Session family was revoked", nil)
	}
	if errors.Is(err, store.ErrInvalidToken) {
		return domain.TokenPair{}, problem(401, "INVALID_SESSION", "Session is invalid or expired", nil)
	}
	if err != nil {
		return domain.TokenPair{}, err
	}
	if a.Status != domain.AccountActive {
		return domain.TokenPair{}, problem(403, "ACCOUNT_UNAVAILABLE", "Account is not active", nil)
	}
	access, accessExpiry, err := s.tokens.Issue(a.ID, a.Roles, a.TokenVersion, now)
	if err != nil {
		return domain.TokenPair{}, err
	}
	return domain.TokenPair{AccessToken: access, RefreshToken: nextRaw, AccessExpiresAt: accessExpiry, RefreshExpiresAt: next.ExpiresAt}, nil
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.store.RevokeSession(ctx, auth.HashOpaqueToken(raw), s.now())
}
func (s *Service) Authenticate(ctx context.Context, raw string) (*auth.Claims, error) {
	c, err := s.tokens.Parse(raw)
	if err != nil {
		return nil, problem(401, "INVALID_ACCESS_TOKEN", "Access token is invalid or expired", nil)
	}
	a, err := s.store.AccountByID(ctx, c.Subject)
	if err != nil || a.Status != domain.AccountActive || a.TokenVersion != c.TokenVersion {
		return nil, problem(401, "INVALID_ACCESS_TOKEN", "Access token is no longer valid", nil)
	}
	return c, nil
}
func (s *Service) JWK() map[string]any { return s.tokens.JWK() }
func (s *Service) GetMe(ctx context.Context, id string) (domain.Me, error) {
	return s.store.GetMe(ctx, id)
}
func (s *Service) UpdateProfile(ctx context.Context, id, correlation string, p domain.ProfilePatch) (domain.Profile, error) {
	if f := domain.ValidateProfilePatch(p, s.minimumAge, s.now()); len(f) > 0 {
		return domain.Profile{}, problem(422, "VALIDATION_FAILED", "Profile is invalid", f)
	}
	if p.Nickname != nil {
		v := strings.TrimSpace(*p.Nickname)
		p.Nickname = &v
	}
	if p.BroadLocation != nil {
		v := strings.TrimSpace(*p.BroadLocation)
		p.BroadLocation = &v
	}
	if p.Bio != nil {
		v := strings.TrimSpace(*p.Bio)
		p.Bio = &v
	}
	if p.Interests != nil {
		v, err := domain.NormalizeList(*p.Interests, 20, 50)
		if err != nil {
			return domain.Profile{}, err
		}
		p.Interests = &v
	}
	v, err := s.store.UpdateProfile(ctx, id, p, event("ProfileUpdated", correlation, id))
	if errors.Is(err, store.ErrConflict) {
		return v, problem(409, "PROFILE_VERSION_CONFLICT", "Profile changed; reload and retry", nil)
	}
	return v, err
}
func (s *Service) ReplacePreferences(ctx context.Context, id string, p domain.PreferenceInput) (domain.Preferences, error) {
	if p.MinAge < s.minimumAge || p.MaxAge < p.MinAge || p.MaxAge > 120 {
		return domain.Preferences{}, problem(422, "INVALID_AGE_RANGE", "Preference age range is invalid", nil)
	}
	var err error
	if p.Intentions, err = domain.NormalizeList(p.Intentions, 10, 50); err != nil {
		return domain.Preferences{}, err
	}
	if p.InterestedIn, err = domain.NormalizeList(p.InterestedIn, 10, 50); err != nil {
		return domain.Preferences{}, err
	}
	if p.Languages, err = domain.NormalizeList(p.Languages, 20, 50); err != nil {
		return domain.Preferences{}, err
	}
	if p.DealBreakers, err = domain.NormalizeList(p.DealBreakers, 20, 100); err != nil {
		return domain.Preferences{}, err
	}
	return s.store.ReplacePreferences(ctx, id, p)
}
func (s *Service) ListCommunity(ctx context.Context, id, cursor string, limit int) ([]domain.CommunityProfile, string, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.store.ListCommunity(ctx, id, cursor, limit)
}
func (s *Service) GetCommunity(ctx context.Context, viewer, id string) (domain.CommunityProfile, error) {
	v, e := s.store.GetCommunity(ctx, viewer, id)
	if errors.Is(e, store.ErrNotFound) {
		return v, problem(404, "PROFILE_NOT_FOUND", "Profile is unavailable", nil)
	}
	return v, e
}
func (s *Service) Block(ctx context.Context, actor, target, correlation string) error {
	return s.store.Block(ctx, actor, target, event("MemberBlocked", correlation, actor))
}
func (s *Service) Unblock(ctx context.Context, actor, target, correlation string) error {
	return s.store.Unblock(ctx, actor, target, event("MemberUnblocked", correlation, actor))
}
func (s *Service) Deactivate(ctx context.Context, id, correlation string) error {
	return s.store.Deactivate(ctx, id, event("AccountDeactivated", correlation, id))
}
func (s *Service) Moderate(ctx context.Context, actor, target, decision, reason, correlation string) error {
	if decision != domain.ApprovalApproved && decision != domain.ApprovalHidden {
		return problem(422, "INVALID_DECISION", "Decision must be APPROVED or HIDDEN", nil)
	}
	return s.store.SetProfileDecision(ctx, actor, target, decision, strings.TrimSpace(reason), event("Profile"+strings.Title(strings.ToLower(decision)), correlation, actor))
}
func (s *Service) Ready(ctx context.Context) error { return s.store.Ping(ctx) }

func (s *Service) newSession(ctx context.Context, a domain.Account, family string, now time.Time) (domain.TokenPair, error) {
	raw, hash, err := auth.NewOpaqueToken()
	if err != nil {
		return domain.TokenPair{}, err
	}
	if family == "" {
		family = uuid.NewString()
	}
	session := domain.Session{ID: uuid.NewString(), FamilyID: family, AccountID: a.ID, TokenHash: hash, ExpiresAt: now.Add(s.refreshTTL)}
	if err = s.store.CreateSession(ctx, session); err != nil {
		return domain.TokenPair{}, err
	}
	access, exp, err := s.tokens.Issue(a.ID, a.Roles, a.TokenVersion, now)
	return domain.TokenPair{AccessToken: access, RefreshToken: raw, AccessExpiresAt: exp, RefreshExpiresAt: session.ExpiresAt}, err
}
func event(kind, correlation, actor string) domain.Event {
	if correlation == "" {
		correlation = uuid.NewString()
	}
	return domain.Event{EventID: uuid.NewString(), EventType: kind, SchemaVersion: 1, OccurredAt: time.Now().UTC(), CorrelationID: correlation, ActorID: actor}
}
func problem(status int, code, detail string, fields map[string]string) *domain.ProblemError {
	return &domain.ProblemError{Status: status, Code: code, Title: code, Detail: detail, Fields: fields}
}
