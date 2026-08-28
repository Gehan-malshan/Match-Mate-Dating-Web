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
	credential   store.Credential
	registered   domain.RegisterInput
	passwordHash string
	session      domain.Session
	account      domain.Account
}

func (f *fakeRepo) Register(_ context.Context, in domain.RegisterInput, hash string, _ []byte, _ time.Time, _ domain.Event) (domain.Me, error) {
	f.registered = in
	f.passwordHash = hash
	return domain.Me{Account: domain.Account{ID: "a1", Email: in.Email, Status: domain.AccountActive, Verification: domain.VerificationPending, Roles: []string{"member"}, TokenVersion: 1}, Profile: domain.Profile{AccountID: "a1", Nickname: in.Nickname, DateOfBirth: in.DateOfBirth, Visibility: domain.VisibilityPrivate, Approval: domain.ApprovalPending, Version: 1}}, nil
}
func (f *fakeRepo) CredentialsByEmail(context.Context, string) (store.Credential, error) {
	return f.credential, nil
}
func (f *fakeRepo) CreateSession(_ context.Context, s domain.Session) error {
	f.session = s
	return nil
}
func (f *fakeRepo) AccountByID(context.Context, string) (domain.Account, error) {
	return f.account, nil
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
