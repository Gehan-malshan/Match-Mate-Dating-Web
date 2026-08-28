package application

import (
	"context"
	"github.com/gehan-malshan/matchmate/event-service/internal/domain"
	"github.com/gehan-malshan/matchmate/event-service/internal/store"
	"testing"
	"time"
)

type fakeRepo struct{ event domain.Event }

func (f *fakeRepo) Ping(context.Context) error { return nil }
func (f *fakeRepo) Create(_ context.Context, e domain.Event, _ domain.Fact) (domain.Event, error) {
	f.event = e
	return e, nil
}
func (f *fakeRepo) Get(context.Context, string) (domain.Event, error) {
	if f.event.ID == "" {
		return domain.Event{}, store.ErrNotFound
	}
	return f.event, nil
}
func (f *fakeRepo) Update(_ context.Context, _ string, in domain.UpdateInput, _ domain.Fact) (domain.Event, error) {
	if in.ExpectedVersion != f.event.Version {
		return domain.Event{}, store.ErrConflict
	}
	f.event.Name = in.Name
	f.event.Version++
	return f.event, nil
}
func (f *fakeRepo) Transition(_ context.Context, _ string, v int64, to domain.Status, _ string, _ domain.Fact) (domain.Event, error) {
	if v != f.event.Version {
		return domain.Event{}, store.ErrConflict
	}
	f.event.Status = to
	f.event.Version++
	return f.event, nil
}
func (f *fakeRepo) ListDiscoverable(context.Context, string, int, time.Time) (domain.Page, error) {
	return domain.Page{}, nil
}
func (f *fakeRepo) ListManaged(context.Context, string, bool, string, int) (domain.Page, error) {
	return domain.Page{}, nil
}
func appInput() domain.CreateInput {
	start := time.Now().UTC().Add(10 * 24 * time.Hour)
	return domain.CreateInput{OrganizerID: "org-1", Name: "Colombo Social", BroadLocation: "Colombo", TimeZone: "Asia/Colombo", StartsAt: start, EndsAt: start.Add(2 * time.Hour), RegistrationOpensAt: start.Add(-9 * 24 * time.Hour), RegistrationClosesAt: start.Add(-time.Hour), Price: "2500.00", Currency: "LKR", ConfiguredCapacity: 40, MatchingRulesetVersion: "v1"}
}
func TestOrganizerCannotCreateForAnotherOrganizer(t *testing.T) {
	s := New(&fakeRepo{})
	_, err := s.Create(context.Background(), domain.Principal{Subject: "org-2", Roles: []string{"organizer"}}, appInput(), "")
	p, ok := err.(*domain.ProblemError)
	if !ok || p.Code != "EVENT_ORGANIZER_SCOPE" {
		t.Fatalf("unexpected error: %v", err)
	}
}
func TestAdminCanCreateAndPublish(t *testing.T) {
	r := &fakeRepo{}
	s := New(r)
	e, err := s.Create(context.Background(), domain.Principal{Subject: "admin", Roles: []string{"admin"}}, appInput(), "c")
	if err != nil {
		t.Fatal(err)
	}
	e, err = s.Transition(context.Background(), domain.Principal{Subject: "admin", Roles: []string{"admin"}}, e.ID, e.Version, domain.Published, "", "c")
	if err != nil || e.Status != domain.Published {
		t.Fatalf("publish failed: %v %#v", err, e)
	}
}
func TestStaleTransitionReturnsStableConflict(t *testing.T) {
	r := &fakeRepo{}
	s := New(r)
	e, _ := s.Create(context.Background(), domain.Principal{Subject: "admin", Roles: []string{"admin"}}, appInput(), "c")
	_, err := s.Transition(context.Background(), domain.Principal{Subject: "admin", Roles: []string{"admin"}}, e.ID, 99, domain.Published, "", "c")
	p, ok := err.(*domain.ProblemError)
	if !ok || p.Code != "EVENT_VERSION_CONFLICT" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishRejectsExpiredRegistrationWindow(t *testing.T) {
	r := &fakeRepo{}
	s := New(r)
	principal := domain.Principal{Subject: "admin", Roles: []string{"admin"}}
	event, _ := s.Create(context.Background(), principal, appInput(), "c")
	r.event.RegistrationClosesAt = time.Now().UTC().Add(-time.Minute)
	_, err := s.Transition(context.Background(), principal, event.ID, event.Version, domain.Published, "", "c")
	p, ok := err.(*domain.ProblemError)
	if !ok || p.Code != "EVENT_PUBLICATION_WINDOW_EXPIRED" {
		t.Fatalf("unexpected error: %v", err)
	}
}
