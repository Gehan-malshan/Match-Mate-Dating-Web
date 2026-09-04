package store

import (
	"context"
	"errors"
	"time"

	"github.com/gehan-malshan/matchmate/account-service/internal/domain"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrInvalidToken = errors.New("invalid token")
var ErrRefreshReuse = errors.New("refresh reuse")

type Credential struct {
	Account      domain.Account
	PasswordHash string
	TokenVersion int64
}
type OutboxRecord struct {
	ID         string
	RoutingKey string
	Body       []byte
}

type Repository interface {
	Ping(context.Context) error
	Register(context.Context, domain.RegisterInput, string, []byte, time.Time, domain.Event) (domain.Me, error)
	VerifyEmail(context.Context, []byte, time.Time, domain.Event) (domain.Account, error)
	CredentialsByEmail(context.Context, string) (Credential, error)
	AccountByEmail(context.Context, string) (domain.Account, error)
	AccountByID(context.Context, string) (domain.Account, error)
	CreateSession(context.Context, domain.Session) error
	RotateSession(context.Context, []byte, domain.Session, time.Time) (domain.Account, error)
	RevokeSession(context.Context, []byte, time.Time) error
	GetMe(context.Context, string) (domain.Me, error)
	UpdateProfile(context.Context, string, domain.ProfilePatch, domain.Event) (domain.Profile, error)
	ReplacePreferences(context.Context, string, domain.PreferenceInput) (domain.Preferences, error)
	ListCommunity(context.Context, string, string, int) ([]domain.CommunityProfile, string, error)
	GetCommunity(context.Context, string, string) (domain.CommunityProfile, error)
	Block(context.Context, string, string, domain.Event) error
	Unblock(context.Context, string, string, domain.Event) error
	Deactivate(context.Context, string, domain.Event) error
	SetProfileDecision(context.Context, string, string, string, string, domain.Event) error
	ClaimOutbox(context.Context, int) ([]OutboxRecord, error)
	MarkOutboxPublished(context.Context, string, time.Time) error
}
