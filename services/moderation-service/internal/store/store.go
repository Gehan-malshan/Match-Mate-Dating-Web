package store

import (
	"context"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/domain"
	"time"
)

type Repository interface {
	Ready(context.Context) error
	CreateReport(context.Context, domain.Principal, domain.CreateReportInput, string, time.Time) (domain.Report, error)
	ListOwnReports(context.Context, string, int, time.Time, string) (domain.Page[domain.Report], error)
	ListCases(context.Context, int, time.Time, string) (domain.Page[domain.Case], error)
	GetCase(context.Context, string) (domain.Case, error)
	RecordCaseView(context.Context, string, domain.Principal, string, time.Time) error
	AssignCase(context.Context, string, string, domain.Principal, string, string, time.Time) (domain.Case, error)
	UpdateCaseStatus(context.Context, string, domain.ReportStatus, domain.Principal, string, string, time.Time) (domain.Case, error)
	CreateAction(context.Context, string, domain.Action, domain.Principal, string, time.Time) (domain.Action, error)
	CreateAppeal(context.Context, string, string, domain.Principal, string, time.Time) (domain.Appeal, error)
	DecideAppeal(context.Context, string, string, domain.Principal, string, string, time.Time) (domain.Appeal, error)
	ExpireActions(context.Context, time.Time, int) (int, error)
}
type OutboxRecord struct {
	ID, RoutingKey string
	Body           []byte
}
