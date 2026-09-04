package devsink

import (
	"context"
	"log/slog"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
)

// Sink proves the delivery workflow without resolving or logging member contact
// information. Production environments must use an approved provider adapter and
// constrained Account contact-resolution contract instead.
type Sink struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Sink { return &Sink{log: log} }

func (s *Sink) Send(_ context.Context, delivery domain.Delivery, _ domain.RenderedMessage, _ string) (string, error) {
	reference := "dev-sink:" + delivery.ID
	s.log.Info("notification_dev_sink_delivered",
		"delivery_id", delivery.ID,
		"template_key", delivery.Template.Key,
		"source_event_type", delivery.SourceEventType,
		"provider_reference", reference,
	)
	return reference, nil
}
