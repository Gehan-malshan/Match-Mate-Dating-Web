package application

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
	"github.com/google/uuid"
)

var ErrInvalidEvent = errors.New("invalid notification event")

type Router struct {
	locale  string
	channel string
}

func NewRouter(locale string, channels ...string) *Router {
	channel := "DEVELOPMENT"
	if len(channels) > 0 && channels[0] != "" {
		channel = channels[0]
	}
	return &Router{locale: locale, channel: channel}
}

type accountPayload struct {
	AccountID string `json:"accountId"`
}

type bookingPayload struct {
	BookingID string `json:"bookingId"`
	EventID   string `json:"eventId"`
	AccountID string `json:"accountId"`
	State     string `json:"state"`
}

func (r *Router) Route(event domain.EventEnvelope) (domain.Plan, error) {
	if _, err := uuid.Parse(event.EventID); err != nil || event.EventType == "" || event.SchemaVersion != 1 || event.OccurredAt.IsZero() || event.AggregateID == "" {
		return domain.Plan{}, ErrInvalidEvent
	}

	plan := domain.Plan{Action: domain.ActionIgnore, Locale: r.locale, Channel: r.channel, Variables: map[string]string{}}
	switch event.EventType {
	case "AccountRegistered", "AccountVerified", "ProfileApproved", "ProfileHidden", "AccountDeactivated":
		var payload accountPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil || !validAccountID(payload.AccountID) || payload.AccountID != event.AggregateID {
			return domain.Plan{}, fmt.Errorf("%w: account payload", ErrInvalidEvent)
		}
		plan.RecipientAccountID = payload.AccountID
		plan.SourceAggregateID = event.AggregateID
		plan.Category = "ACCOUNT"
		if event.EventType == "AccountDeactivated" {
			plan.Action = domain.ActionSuppress
			plan.SuppressionReason = "ACCOUNT_DEACTIVATED"
			return plan, nil
		}
		plan.Action = domain.ActionDeliver
		plan.TemplateKey = map[string]string{
			"AccountRegistered": "account-welcome",
			"AccountVerified":   "account-verified",
			"ProfileApproved":   "profile-approved",
			"ProfileHidden":     "profile-hidden",
		}[event.EventType]
	case "BookingPending", "BookingConfirmed", "BookingCancelled", "HoldExpired", "BookingPaymentReviewRequired":
		var payload bookingPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil || !validAccountID(payload.AccountID) {
			return domain.Plan{}, fmt.Errorf("%w: booking payload", ErrInvalidEvent)
		}
		if _, err := uuid.Parse(payload.BookingID); err != nil || payload.BookingID != event.AggregateID {
			return domain.Plan{}, fmt.Errorf("%w: booking aggregate", ErrInvalidEvent)
		}
		if _, err := uuid.Parse(payload.EventID); err != nil {
			return domain.Plan{}, fmt.Errorf("%w: event id", ErrInvalidEvent)
		}
		plan.Action = domain.ActionDeliver
		plan.RecipientAccountID = payload.AccountID
		plan.SourceAggregateID = payload.BookingID
		plan.Category = "BOOKING"
		plan.TemplateKey = map[string]string{
			"BookingPending":               "booking-pending",
			"BookingConfirmed":             "booking-confirmed",
			"BookingCancelled":             "booking-cancelled",
			"HoldExpired":                  "booking-hold-expired",
			"BookingPaymentReviewRequired": "booking-payment-review",
		}[event.EventType]
	default:
		return plan, nil
	}
	plan.BusinessKey = fmt.Sprintf("notification:v1:%s:%s:%s", event.EventID, plan.TemplateKey, plan.RecipientAccountID)
	return plan, nil
}

func validAccountID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
