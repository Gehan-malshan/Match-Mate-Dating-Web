package store

import (
	"context"
	"errors"
	"github.com/gehan-malshan/matchmate/event-service/internal/domain"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type Repository interface {
	Ping(context.Context) error
	Create(context.Context, domain.Event, domain.Fact) (domain.Event, error)
	Get(context.Context, string) (domain.Event, error)
	Update(context.Context, string, domain.UpdateInput, domain.Fact) (domain.Event, error)
	Transition(context.Context, string, int64, domain.Status, string, domain.Fact) (domain.Event, error)
	ListDiscoverable(context.Context, string, int, time.Time) (domain.Page, error)
	ListManaged(context.Context, string, bool, string, int) (domain.Page, error)
}

type OutboxRecord struct {
	ID, RoutingKey string
	Body           []byte
}
