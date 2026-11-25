package application

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
)

const (
	eventID   = "10000000-0000-4000-8000-000000000001"
	accountID = "20000000-0000-4000-8000-000000000001"
	bookingID = "30000000-0000-4000-8000-000000000001"
	eventRef  = "40000000-0000-4000-8000-000000000001"
)

func envelope(eventType, aggregate string, payload any) domain.EventEnvelope {
	body, _ := json.Marshal(payload)
	return domain.EventEnvelope{EventID: eventID, EventType: eventType, SchemaVersion: 1, OccurredAt: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC), AggregateID: aggregate, CorrelationID: "test", Payload: body}
}

func TestRouteSupportedEvents(t *testing.T) {
	router := NewRouter("en-LK")
	tests := []struct {
		name, eventType, aggregate, template string
		payload                              any
		action                               domain.Action
	}{
		{"welcome", "AccountRegistered", accountID, "account-welcome", map[string]any{"accountId": accountID}, domain.ActionDeliver},
		{"verified", "AccountVerified", accountID, "account-verified", map[string]any{"accountId": accountID}, domain.ActionDeliver},
		{"confirmed", "BookingConfirmed", bookingID, "booking-confirmed", map[string]any{"bookingId": bookingID, "eventId": eventRef, "accountId": accountID, "state": "CONFIRMED"}, domain.ActionDeliver},
		{"deactivated", "AccountDeactivated", accountID, "", map[string]any{"accountId": accountID}, domain.ActionSuppress},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, err := router.Route(envelope(test.eventType, test.aggregate, test.payload))
			if err != nil {
				t.Fatal(err)
			}
			if plan.Action != test.action || plan.TemplateKey != test.template || plan.RecipientAccountID != accountID {
				t.Fatalf("unexpected plan: %+v", plan)
			}
			if plan.Action == domain.ActionDeliver && plan.BusinessKey == "" {
				t.Fatal("delivery plan must have a business idempotency key")
			}
		})
	}
}

func TestRouteRejectsMismatchedAggregateAndUnsupportedVersion(t *testing.T) {
	router := NewRouter("en-LK")
	_, err := router.Route(envelope("AccountVerified", bookingID, map[string]any{"accountId": accountID}))
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("expected invalid event, got %v", err)
	}
	event := envelope("AccountVerified", accountID, map[string]any{"accountId": accountID})
	event.SchemaVersion = 2
	_, err = router.Route(event)
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("expected unsupported version to fail, got %v", err)
	}
}

func TestRouteIgnoresUnknownEventWithoutCopyingPayload(t *testing.T) {
	plan, err := NewRouter("en-LK").Route(envelope("UnknownFact", accountID, map[string]any{"email": "must-not-copy@example.test"}))
	if err != nil || plan.Action != domain.ActionIgnore || len(plan.Variables) != 0 || plan.RecipientAccountID != "" {
		t.Fatalf("unexpected ignored plan: %+v, %v", plan, err)
	}
}
