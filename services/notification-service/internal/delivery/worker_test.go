package delivery

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
	"github.com/gehan-malshan/matchmate/notification-service/internal/store/postgres"
)

type repositoryStub struct {
	delivery domain.Delivery
	found    bool
	result   postgres.AttemptResult
	state    domain.DeliveryState
}

func (s *repositoryStub) ClaimDue(context.Context, time.Time, time.Duration) (domain.Delivery, bool, error) {
	return s.delivery, s.found, nil
}

func (s *repositoryStub) CompleteAttempt(_ context.Context, _ domain.Delivery, result postgres.AttemptResult) (domain.DeliveryState, error) {
	s.result = result
	return s.state, nil
}

type senderStub struct{ err error }

func (s senderStub) Send(context.Context, domain.Delivery, domain.RenderedMessage, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "provider:one", nil
}

func testDelivery() domain.Delivery {
	return domain.Delivery{
		ID:           "delivery-one",
		AttemptCount: 2,
		MaxAttempts:  5,
		Template: domain.Template{
			Key:             "booking-confirmed",
			SubjectTemplate: "Confirmed",
			BodyTemplate:    "Your booking is confirmed.",
		},
		Variables: map[string]string{},
	}
}

func TestWorkerClassifiesSuccessfulRetryableAndPermanentResults(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		sendErr     error
		state       domain.DeliveryState
		delivered   bool
		permanent   bool
		errorCode   string
		expectRetry bool
	}{
		{"success", nil, domain.DeliveryDelivered, true, false, "", false},
		{"retryable", &domain.SendFailure{Kind: domain.FailureRetryable, Code: "PROVIDER_TIMEOUT", Err: errors.New("timeout")}, domain.DeliveryRetryScheduled, false, false, "PROVIDER_TIMEOUT", true},
		{"permanent", &domain.SendFailure{Kind: domain.FailurePermanent, Code: "INVALID_DESTINATION", Err: errors.New("invalid")}, domain.DeliveryPermanentlyFailed, false, true, "INVALID_DESTINATION", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &repositoryStub{delivery: testDelivery(), found: true, state: test.state}
			worker := NewWorker(repository, senderStub{err: test.sendErr}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), 30*time.Second, time.Minute)
			worker.now = func() time.Time { return now }
			processed, err := worker.RunOnce(context.Background())
			if err != nil || !processed {
				t.Fatalf("processed=%v err=%v", processed, err)
			}
			if repository.result.Delivered != test.delivered || repository.result.PermanentFailure != test.permanent || repository.result.ErrorCode != test.errorCode {
				t.Fatalf("unexpected attempt result: %+v", repository.result)
			}
			if test.expectRetry != !repository.result.RetryAt.IsZero() {
				t.Fatalf("retry time mismatch: %+v", repository.result)
			}
		})
	}
}

func TestWorkerTreatsInvalidTemplateAsPermanent(t *testing.T) {
	delivery := testDelivery()
	delivery.Template.SubjectTemplate = ""
	repository := &repositoryStub{delivery: delivery, found: true, state: domain.DeliveryPermanentlyFailed}
	worker := NewWorker(repository, senderStub{}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second, time.Minute)
	worker.now = time.Now
	if _, err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !repository.result.PermanentFailure || repository.result.ErrorCode != "TEMPLATE_RENDER_INVALID" {
		t.Fatalf("unexpected attempt result: %+v", repository.result)
	}
}
