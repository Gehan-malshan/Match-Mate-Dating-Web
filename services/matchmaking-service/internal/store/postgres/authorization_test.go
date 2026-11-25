package postgres

import (
	"context"
	"testing"

	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

type queryMustNotRun struct{}

func (queryMustNotRun) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("authorization queried storage before rejecting a non-admin")
}

func TestMatchingRunManagementRequiresAdministrator(t *testing.T) {
	store := &Store{}
	for _, principal := range []domain.Principal{
		{Subject: "member", Roles: []string{"member"}},
		{Subject: "organizer", Roles: []string{"member", "organizer"}},
	} {
		err := store.authorizeEvent(context.Background(), queryMustNotRun{}, principal, "event-id")
		problem, ok := err.(*domain.ProblemError)
		if !ok || problem.Code != "MATCHMAKING_ADMIN_REQUIRED" || problem.Status != 403 {
			t.Fatalf("principal %s received unexpected authorization result: %v", principal.Subject, err)
		}
	}
}
