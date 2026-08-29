package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/gehan-malshan/matchmate/booking-service/internal/domain"
	"github.com/google/uuid"
	"time"
)

var (
	ErrConflict   = errors.New("conflict")
	ErrNotFound   = errors.New("not found")
	ErrCapacity   = errors.New("event capacity exhausted")
	ErrInvalid    = errors.New("invalid request")
	ErrDependency = errors.New("dependency unavailable")
)

type EventReader interface {
	Get(context.Context, string) (domain.EventSnapshot, error)
}
type Repository interface {
	Create(context.Context, domain.Booking, string, string, int) (domain.Booking, bool, error)
	Get(context.Context, string, string) (domain.Booking, error)
	List(context.Context, string, int) ([]domain.Booking, error)
	Cancel(context.Context, string, string, string, time.Time) (domain.Booking, bool, error)
	Expire(context.Context, time.Time, int) (int, error)
}
type Service struct {
	repo   Repository
	events EventReader
	hold   time.Duration
	now    func() time.Time
}

func New(repo Repository, events EventReader, hold time.Duration) *Service {
	return &Service{repo: repo, events: events, hold: hold, now: time.Now}
}
func (s *Service) Create(ctx context.Context, actor, eventID, key string) (domain.Booking, error) {
	if eventID == "" || key == "" {
		return domain.Booking{}, fmt.Errorf("%w: eventId and Idempotency-Key are required", ErrInvalid)
	}
	e, err := s.events.Get(ctx, eventID)
	if err != nil {
		return domain.Booking{}, fmt.Errorf("%w: Event service", ErrDependency)
	}
	now := s.now().UTC()
	if err = domain.ValidateEvent(e, now); err != nil {
		return domain.Booking{}, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	b := domain.Booking{ID: uuid.NewString(), AccountID: actor, EventID: e.EventID, State: domain.Pending, Amount: domain.NormalizeMoney(e.Price), Currency: e.Currency, PolicyVersion: e.CapacityPolicyVersion, ExpiresAt: now.Add(s.hold), Version: 1, CreatedAt: now}
	b, _, err = s.repo.Create(ctx, b, key, eventID, e.ConfiguredCapacity)
	return b, err
}
func (s *Service) Get(ctx context.Context, actor, id string) (domain.Booking, error) {
	return s.repo.Get(ctx, actor, id)
}
func (s *Service) List(ctx context.Context, actor string, limit int) ([]domain.Booking, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, actor, limit)
}
func (s *Service) Cancel(ctx context.Context, actor, id, key string) (domain.Booking, error) {
	if id == "" || key == "" {
		return domain.Booking{}, fmt.Errorf("%w: bookingId and Idempotency-Key are required", ErrInvalid)
	}
	b, _, err := s.repo.Cancel(ctx, actor, id, key, s.now().UTC())
	return b, err
}
func (s *Service) Snapshot(ctx context.Context, actor, id string) (domain.Booking, error) {
	b, err := s.repo.Get(ctx, actor, id)
	if err != nil {
		return b, err
	}
	if b.State != domain.Pending {
		return b, fmt.Errorf("%w: booking is not pending payment", ErrConflict)
	}
	if !b.ExpiresAt.After(s.now().UTC()) {
		return b, fmt.Errorf("%w: booking hold expired", ErrConflict)
	}
	return b, nil
}
func (s *Service) Expire(ctx context.Context, limit int) (int, error) {
	return s.repo.Expire(ctx, s.now().UTC(), limit)
}
