package application

import (
	"context"
	"errors"
	"github.com/gehan-malshan/matchmate/event-service/internal/domain"
	"github.com/gehan-malshan/matchmate/event-service/internal/store"
	"github.com/google/uuid"
	"strings"
	"time"
)

type Service struct {
	repo store.Repository
	now  func() time.Time
}

func New(r store.Repository) *Service {
	return &Service{repo: r, now: func() time.Time { return time.Now().UTC() }}
}
func (s *Service) Create(ctx context.Context, p domain.Principal, in domain.CreateInput, corr string) (domain.Event, error) {
	if !p.HasRole("admin") && !p.HasRole("organizer") {
		return domain.Event{}, problem(403, "EVENT_FORBIDDEN", "Organizer or admin role is required", nil)
	}
	if p.HasRole("organizer") && !p.HasRole("admin") && in.OrganizerID != p.Subject {
		return domain.Event{}, problem(403, "EVENT_ORGANIZER_SCOPE", "An organizer can only create their own event", nil)
	}
	if f := domain.Validate(in); len(f) > 0 {
		return domain.Event{}, problem(422, "EVENT_VALIDATION_FAILED", "Event configuration is invalid", f)
	}
	now := s.now()
	e := domain.Event{ID: uuid.NewString(), OrganizerID: strings.TrimSpace(in.OrganizerID), Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description), VenueName: strings.TrimSpace(in.VenueName), BroadLocation: strings.TrimSpace(in.BroadLocation), TimeZone: in.TimeZone, StartsAt: in.StartsAt.UTC(), EndsAt: in.EndsAt.UTC(), RegistrationOpensAt: in.RegistrationOpensAt.UTC(), RegistrationClosesAt: in.RegistrationClosesAt.UTC(), Price: in.Price, Currency: in.Currency, ConfiguredCapacity: in.ConfiguredCapacity, CapacityPolicyVersion: 1, MatchingRulesetVersion: strings.TrimSpace(in.MatchingRulesetVersion), Status: domain.Draft, Version: 1, CreatedAt: now, UpdatedAt: now}
	return s.repo.Create(ctx, e, fact("EventCreated", e, p, corr))
}
func (s *Service) Get(ctx context.Context, id string) (domain.Event, error) {
	e, err := s.repo.Get(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return e, problem(404, "EVENT_NOT_FOUND", "Event was not found", nil)
	}
	return e, err
}
func (s *Service) Update(ctx context.Context, p domain.Principal, id string, in domain.UpdateInput, corr string) (domain.Event, error) {
	e, err := s.authorize(ctx, p, id)
	if err != nil {
		return domain.Event{}, err
	}
	if e.Status != domain.Draft {
		return domain.Event{}, problem(409, "EVENT_STATE_CONFLICT", "Only draft events can be edited", nil)
	}
	if in.OrganizerID == "" {
		in.OrganizerID = e.OrganizerID
	}
	if p.HasRole("organizer") && !p.HasRole("admin") && in.OrganizerID != p.Subject {
		return domain.Event{}, problem(403, "EVENT_ORGANIZER_SCOPE", "Organizer assignment cannot be changed", nil)
	}
	if f := domain.Validate(in.CreateInput); len(f) > 0 {
		return domain.Event{}, problem(422, "EVENT_VALIDATION_FAILED", "Event configuration is invalid", f)
	}
	v, err := s.repo.Update(ctx, id, in, fact("EventUpdated", e, p, corr))
	if errors.Is(err, store.ErrConflict) {
		return v, problem(409, "EVENT_VERSION_CONFLICT", "Event changed; reload and retry", nil)
	}
	return v, err
}
func (s *Service) Transition(ctx context.Context, p domain.Principal, id string, expected int64, to domain.Status, reason, corr string) (domain.Event, error) {
	e, err := s.authorize(ctx, p, id)
	if err != nil {
		return e, err
	}
	if !domain.CanTransition(e.Status, to) {
		return e, problem(409, "EVENT_TRANSITION_INVALID", "The requested lifecycle transition is not allowed", nil)
	}
	if to == domain.Published && !s.now().Before(e.RegistrationClosesAt) {
		return e, problem(409, "EVENT_PUBLICATION_WINDOW_EXPIRED", "An event cannot be published after registration has closed", nil)
	}
	if to == domain.Cancelled && strings.TrimSpace(reason) == "" {
		return e, problem(422, "EVENT_CANCELLATION_REASON_REQUIRED", "A cancellation reason is required", map[string]string{"reason": "required"})
	}
	if to == domain.RegistrationOpen {
		now := s.now()
		if now.Before(e.RegistrationOpensAt) || !now.Before(e.RegistrationClosesAt) {
			return e, problem(409, "EVENT_REGISTRATION_WINDOW_INACTIVE", "Registration can open only during its configured window", nil)
		}
	}
	name := map[domain.Status]string{domain.Published: "EventPublished", domain.RegistrationOpen: "EventRegistrationOpened", domain.RegistrationClosed: "EventRegistrationClosed", domain.Cancelled: "EventCancelled"}[to]
	v, err := s.repo.Transition(ctx, id, expected, to, reason, fact(name, e, p, corr))
	if errors.Is(err, store.ErrConflict) {
		return v, problem(409, "EVENT_VERSION_CONFLICT", "Event changed; reload and retry", nil)
	}
	return v, err
}
func (s *Service) List(ctx context.Context, cursor string, limit int) (domain.Page, error) {
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		return domain.Page{}, problem(422, "EVENT_LIMIT_INVALID", "Limit must be between 1 and 100", map[string]string{"limit": "out_of_range"})
	}
	return s.repo.ListDiscoverable(ctx, cursor, limit, s.now())
}
func (s *Service) ListManaged(ctx context.Context, p domain.Principal, cursor string, limit int) (domain.Page, error) {
	if !p.HasRole("admin") && !p.HasRole("organizer") {
		return domain.Page{}, problem(403, "EVENT_FORBIDDEN", "Organizer or admin role is required", nil)
	}
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return domain.Page{}, problem(422, "EVENT_LIMIT_INVALID", "Limit must be between 1 and 100", map[string]string{"limit": "out_of_range"})
	}
	return s.repo.ListManaged(ctx, p.Subject, p.HasRole("admin"), cursor, limit)
}
func (s *Service) Ready(ctx context.Context) error { return s.repo.Ping(ctx) }
func (s *Service) authorize(ctx context.Context, p domain.Principal, id string) (domain.Event, error) {
	e, err := s.Get(ctx, id)
	if err != nil {
		return e, err
	}
	if p.HasRole("admin") || (p.HasRole("organizer") && e.OrganizerID == p.Subject) {
		return e, nil
	}
	return e, problem(403, "EVENT_FORBIDDEN", "Only the assigned organizer or an admin can change this event", nil)
}
func fact(kind string, e domain.Event, p domain.Principal, corr string) domain.Fact {
	if corr == "" {
		corr = uuid.NewString()
	}
	return domain.Fact{EventID: uuid.NewString(), EventType: kind, SchemaVersion: 1, OccurredAt: time.Now().UTC(), AggregateID: e.ID, CorrelationID: corr, ActorID: p.Subject, Payload: map[string]any{"eventId": e.ID, "status": e.Status, "version": e.Version}}
}
func problem(status int, code, detail string, fields map[string]string) *domain.ProblemError {
	return &domain.ProblemError{Status: status, Code: code, Detail: detail, Fields: fields}
}
