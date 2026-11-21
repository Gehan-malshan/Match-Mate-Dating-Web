package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/safety"
)

type safetyStub struct {
	decision safety.Decision
	err      error
}

func (s *safetyStub) Check(context.Context, string, string) (safety.Decision, error) {
	return s.decision, s.err
}

func TestMatchingSafetyAndReversal(t *testing.T) {
	store := &Store{}
	guard := &safetyStub{decision: safety.Decision{MatchmakingExcluded: true}}
	store.SetSafety(guard)
	if err := store.checkSafety(context.Background(), "member", "event", false); err == nil {
		t.Fatal("excluded member entered matching")
	}
	guard.decision = safety.Decision{RevealPrevented: true}
	if err := store.checkSafety(context.Background(), "member", "event", true); err == nil {
		t.Fatal("reveal prevention was ignored")
	}
	if err := store.checkSafety(context.Background(), "member", "event", false); err != nil {
		t.Fatalf("reveal-only action blocked safe matching: %v", err)
	}
	guard.decision = safety.Decision{ProfileHidden: true}
	if err := store.checkSafety(context.Background(), "member", "event", false); err == nil {
		t.Fatal("hidden profile entered matching")
	}
	guard.decision = safety.Decision{}
	if err := store.checkSafety(context.Background(), "member", "event", true); err != nil {
		t.Fatalf("reversed appeal still restricts member: %v", err)
	}
	guard.err = errors.New("moderation unavailable")
	if err := store.checkSafety(context.Background(), "member", "event", false); err == nil {
		t.Fatal("moderation outage failed open")
	}
}
