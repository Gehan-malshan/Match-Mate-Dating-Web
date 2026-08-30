package provider

import (
	"context"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
)

type Sender interface {
	Send(context.Context, domain.Delivery, domain.RenderedMessage) (string, error)
}
