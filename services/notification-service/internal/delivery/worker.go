package delivery

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
	"github.com/gehan-malshan/matchmate/notification-service/internal/provider"
	"github.com/gehan-malshan/matchmate/notification-service/internal/store/postgres"
)

type Worker struct {
	repository Repository
	sender     provider.Sender
	log        *slog.Logger
	lease      time.Duration
	retryBase  time.Duration
	now        func() time.Time
}

type Repository interface {
	ClaimDue(context.Context, time.Time, time.Duration) (domain.Delivery, bool, error)
	CompleteAttempt(context.Context, domain.Delivery, postgres.AttemptResult) (domain.DeliveryState, error)
}

func NewWorker(repository Repository, sender provider.Sender, log *slog.Logger, lease, retryBase time.Duration) *Worker {
	return &Worker{repository: repository, sender: sender, log: log, lease: lease, retryBase: retryBase, now: time.Now}
}

func (w *Worker) RunOnce(ctx context.Context) (bool, error) {
	started := w.now().UTC()
	delivery, found, err := w.repository.ClaimDue(ctx, started, w.lease)
	if err != nil || !found {
		return found, err
	}

	message, renderErr := domain.Render(delivery.Template, delivery.Variables)
	result := postgres.AttemptResult{StartedAt: started, CompletedAt: w.now().UTC()}
	if renderErr != nil {
		result.PermanentFailure = true
		result.ErrorCode = "TEMPLATE_RENDER_INVALID"
	} else {
		reference, sendErr := w.sender.Send(ctx, delivery, message)
		result.CompletedAt = w.now().UTC()
		result.ProviderReference = reference
		if sendErr == nil {
			result.Delivered = true
		} else {
			result.ErrorCode = "PROVIDER_UNAVAILABLE"
			var failure *domain.SendFailure
			if errors.As(sendErr, &failure) {
				result.PermanentFailure = failure.Kind == domain.FailurePermanent
				if failure.Code != "" {
					result.ErrorCode = failure.Code
				}
			}
		}
	}
	if !result.Delivered && !result.PermanentFailure {
		result.RetryAt = result.CompletedAt.Add(domain.RetryDelay(w.retryBase, delivery.AttemptCount))
	}
	state, err := w.repository.CompleteAttempt(ctx, delivery, result)
	if err != nil {
		return true, err
	}
	w.log.Info("notification_delivery_processed",
		"delivery_id", delivery.ID,
		"source_event_type", delivery.SourceEventType,
		"template_key", delivery.Template.Key,
		"attempt", delivery.AttemptCount,
		"state", state,
		"result_code", result.ErrorCode,
	)
	return true, nil
}
