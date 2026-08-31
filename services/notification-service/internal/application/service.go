package application

import (
	"context"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
)

type EventRepository interface {
	ApplyEvent(context.Context, domain.EventEnvelope, domain.Plan, time.Time, int) (bool, error)
}

type Service struct {
	router      *Router
	repository  EventRepository
	maxAttempts int
	now         func() time.Time
}

func New(router *Router, repository EventRepository, maxAttempts int) *Service {
	return &Service{router: router, repository: repository, maxAttempts: maxAttempts, now: time.Now}
}

func (s *Service) Handle(ctx context.Context, event domain.EventEnvelope) (bool, error) {
	plan, err := s.router.Route(event)
	if err != nil {
		return false, err
	}
	return s.repository.ApplyEvent(ctx, event, plan, s.now().UTC(), s.maxAttempts)
}
