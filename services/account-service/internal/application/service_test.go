package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/account-service/internal/auth"
	"github.com/gehan-malshan/matchmate/account-service/internal/domain"
	"github.com/gehan-malshan/matchmate/account-service/internal/store"
)

type fakeRepo struct {
	store.Repository
	credential        store.Credential
	registered        domain.RegisterInput
	passwordHash      string
	session           domain.Session
	account           domain.Account
	resetHash         []byte
	resetPasswordHash string
	resetUsed         bool
	mfaCode           string
	deactivated       bool
}

func (f *fakeRepo) Register(_ context.Context, in domain.RegisterInput, hash string, _ []byte, _ time.Time, _ domain.Event) (domain.Me, error) {
	f.registered = in
	f.passwordHash = hash
	return domain.Me{Account: domain.Account{ID: "a1", Email: in.Email, Status: domain.AccountActive, Verification: domain.VerificationPending, Roles: []string{"member"}, TokenVersion: 1}, Profile: domain.Profile{AccountID: "a1", Nickname: in.Nickname, DateOfBirth: in.DateOfBirth, Visibility: domain.VisibilityPrivate, Approval: domain.ApprovalPending, Version: 1}}, nil
}
func (f *fakeRepo) CredentialsByEmail(context.Context, string) (store.Credential, error) {
	return f.credential, nil
}
func (f *fakeRepo) AccountByEmail(context.Context, string) (domain.Account, error) {
	return f.account, nil
}
func (f *fakeRepo) CreateSession(_ context.Context, s domain.Session) error {
	f.session = s
	return nil
}
func (f *fakeRepo) AccountByID(context.Context, string) (domain.Account, error) {
	return f.account, nil
}
func (f *fakeRepo) AllowRate(context.Context, []byte, time.Time, int) (bool, error) { return true, nil }
func (f *fakeRepo) CreatePasswordReset(_ context.Context, _ string, hash []byte, _ time.Time) error {
	f.resetHash = hash
	return nil
}
func (f *fakeRepo) ConsumePasswordReset(_ context.Context, hash []byte, passwordHash string, _ time.Time) error {
	if f.resetUsed || string(hash) != string(f.resetHash) {
		return store.ErrInvalidToken
	}
	f.resetUsed = true
	f.resetPasswordHash = passwordHash
	return nil
}
func (f *fakeRepo) VerifyAdminTOTP(_ context.Context, _ string, code string, _ time.Time, _ []byte) error {
	f.mfaCode = code
	if code != "123456" {
		return store.ErrInvalidToken
	}
	return nil
}
func (f *fakeRepo) Deactivate(context.Context, string, domain.Event) error {
	f.deactivated = true
	return nil
}

func testService(t *testing.T, repo store.Repository) *Service {
	t.Helper()
	tokens, err := auth.NewManager("", "test", "issuer", "audience", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	s := New(repo, tokens, time.Minute, time.Hour, time.Hour, 18, "privacy-2026-08", true)
	s.now = func() time.Time { return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC) }
	return s
}
func TestAdminMemberIdentityInputIsBounded(t *testing.T) {
	s := testService(t, &fakeRepo{})
	if _, err := s.AdminMemberIdentities(context.Background(), []string{"not-a-uuid"}); err == nil {
		t.Fatal("expected invalid account ID")
	}
	ids := make([]string, 51)
	if _, err := s.AdminMemberIdentities(context.Background(), ids); err == nil {
		t.Fatal("expected batch limit")
	}
}
func TestRegisterCreatesPrivatePendingProfileAndHash(t *testing.T) {
	repo := &fakeRepo{}
	s := testService(t, repo)
	result, err := s.Register(context.Background(), domain.RegisterInput{Email: " MEMBER@Example.com ", Password: "correct horse battery staple", Nickname: "Member", DateOfBirth: "1995-01-01", ConsentVersion: "privacy-2026-08"}, "correlation")
	if err != nil {
		t.Fatal(err)
	}
	if repo.registered.Email != "member@example.com" {
		t.Fatalf("email not normalized: %s", repo.registered.Email)
	}
	if !strings.HasPrefix(repo.passwordHash, "$argon2id$") {
		t.Fatal("password was not Argon2id hashed")
	}
	if result.Me.Profile.Visibility != domain.VisibilityPrivate || result.Me.Profile.Approval != domain.ApprovalPending {
		t.Fatal("registration exposed the profile")
	}
	if result.VerificationToken == "" {
		t.Fatal("development verification token missing")
	}
}
func TestLoginRequiresVerificationThenCreatesRotatingSession(t *testing.T) {
	hash, _ := auth.HashPassword("correct horse battery staple")
	repo := &fakeRepo{credential: store.Credential{Account: domain.Account{ID: "a1", Status: domain.AccountActive, Verification: domain.VerificationPending, Roles: []string{"member"}}, PasswordHash: hash, TokenVersion: 1}}
	s := testService(t, repo)
	if _, err := s.Login(context.Background(), "member@example.com", "correct horse battery staple"); err == nil {
		t.Fatal("unverified account logged in")
	}
	repo.credential.Account.Verification = domain.VerificationVerified
	pair, err := s.Login(context.Background(), "member@example.com", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" || len(repo.session.TokenHash) == 0 {
		t.Fatal("session tokens were not created")
	}
	if string(repo.session.TokenHash) == pair.RefreshToken {
		t.Fatal("refresh token stored in plaintext")
	}
}

func TestLoginWithGoogleRequiresExistingVerifiedActiveAccount(t *testing.T) {
	repo := &fakeRepo{account: domain.Account{ID: "a1", Email: "member@example.test", Status: domain.AccountActive, Verification: domain.VerificationVerified, Roles: []string{"member"}, TokenVersion: 1}}
	s := testService(t, repo)
	pair, err := s.LoginWithGoogle(context.Background(), " MEMBER@EXAMPLE.TEST ")
	if err != nil || pair.AccessToken == "" || repo.session.AccountID != "a1" {
		t.Fatalf("google login pair=%+v session=%+v err=%v", pair, repo.session, err)
	}
	repo.account.Verification = domain.VerificationPending
	if _, err = s.LoginWithGoogle(context.Background(), "member@example.test"); err == nil {
		t.Fatal("unverified account logged in with Google")
	}
}

func TestRecoveryIsSingleUseAndDoesNotExposeAccountExistence(t *testing.T) {
	repo := &fakeRepo{account: domain.Account{ID: "a1", Email: "member@example.test", Status: domain.AccountActive, Verification: domain.VerificationVerified}}
	s := testService(t, repo)
	var raw string
	s.SetSecurityMailer(func(_ context.Context, email, purpose, token string) error {
		if email != "member@example.test" || purpose != "reset" {
			t.Fatal("unexpected delivery")
		}
		raw = token
		return nil
	})
	if err := s.RequestPasswordReset(context.Background(), "member@example.test"); err != nil {
		t.Fatal(err)
	}
	if raw == "" || string(repo.resetHash) == raw {
		t.Fatal("reset token was not separated from hash")
	}
	if err := s.ResetPassword(context.Background(), raw, "new secure password here"); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword(context.Background(), raw, "new secure password here"); err == nil {
		t.Fatal("reset replay accepted")
	}
	valid, err := auth.VerifyPassword("new secure password here", repo.resetPasswordHash)
	if err != nil || !valid {
		t.Fatal("new password not hashed")
	}
	repo.account = domain.Account{}
}

func TestAdminRequiresMFAAndGoogleCannotBypass(t *testing.T) {
	hash, _ := auth.HashPassword("correct horse battery staple")
	account := domain.Account{ID: "a1", Email: "admin@example.test", Status: domain.AccountActive, Verification: domain.VerificationVerified, Roles: []string{"admin"}, TokenVersion: 1}
	repo := &fakeRepo{account: account, credential: store.Credential{Account: account, PasswordHash: hash, TokenVersion: 1}}
	s := testService(t, repo)
	s.SetAdminMFAKey(make([]byte, 32))
	if _, err := s.Login(context.Background(), account.Email, "correct horse battery staple"); err == nil {
		t.Fatal("admin logged in without MFA")
	}
	if _, err := s.Login(context.Background(), account.Email, "correct horse battery staple", "123456"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoginWithGoogle(context.Background(), account.Email); err == nil {
		t.Fatal("Google bypassed admin MFA")
	}
}
func TestDeletionRequiresPasswordAndRejectsAdmin(t *testing.T) {
	hash, _ := auth.HashPassword("correct horse battery staple")
	repo := &fakeRepo{account: domain.Account{ID: "a1", Email: "member@example.test", Status: domain.AccountActive, Roles: []string{"member"}}, credential: store.Credential{PasswordHash: hash}}
	s := testService(t, repo)
	if err := s.DeleteAccount(context.Background(), "a1", "wrong", "correlation"); err == nil || repo.deactivated {
		t.Fatal("deletion accepted wrong password")
	}
	if err := s.DeleteAccount(context.Background(), "a1", "correct horse battery staple", "correlation"); err != nil || !repo.deactivated {
		t.Fatal("verified deletion failed")
	}
	repo.deactivated = false
	repo.account.Roles = []string{"admin"}
	if err := s.DeleteAccount(context.Background(), "a1", "correct horse battery staple", "correlation"); err == nil || repo.deactivated {
		t.Fatal("admin self-deletion accepted")
	}
}
