package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var moneyPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,2})?$`)

type State string

const (
	Pending       State = "PENDING_PAYMENT"
	Confirmed     State = "CONFIRMED"
	Expired       State = "EXPIRED"
	Cancelled     State = "CANCELLED"
	PaymentReview State = "PAYMENT_REVIEW"
)

type Booking struct {
	ID            string     `json:"bookingId"`
	AccountID     string     `json:"-"`
	EventID       string     `json:"eventId"`
	State         State      `json:"state"`
	Amount        string     `json:"amount"`
	Currency      string     `json:"currency"`
	PolicyVersion int64      `json:"policyVersion"`
	ExpiresAt     time.Time  `json:"expiresAt"`
	Version       int64      `json:"version"`
	CreatedAt     time.Time  `json:"createdAt"`
	ConfirmedAt   *time.Time `json:"confirmedAt,omitempty"`
	CancelledAt   *time.Time `json:"cancelledAt,omitempty"`
}
type EventSnapshot struct {
	EventID               string    `json:"eventId"`
	Status                string    `json:"status"`
	Price                 string    `json:"price"`
	Currency              string    `json:"currency"`
	ConfiguredCapacity    int       `json:"configuredCapacity"`
	CapacityPolicyVersion int64     `json:"capacityPolicyVersion"`
	RegistrationClosesAt  time.Time `json:"registrationClosesAt"`
}

func ValidateEvent(e EventSnapshot, now time.Time) error {
	if e.EventID == "" || e.Status != "REGISTRATION_OPEN" {
		return errors.New("event registration is not open")
	}
	if !e.RegistrationClosesAt.After(now) {
		return errors.New("event registration has closed")
	}
	if e.ConfiguredCapacity < 1 || !moneyPattern.MatchString(e.Price) || len(e.Currency) != 3 || e.Currency != strings.ToUpper(e.Currency) {
		return errors.New("event configuration is invalid")
	}
	return nil
}

func NormalizeMoney(value string) string {
	if !strings.Contains(value, ".") {
		return value + ".00"
	}
	parts := strings.SplitN(value, ".", 2)
	if len(parts[1]) == 1 {
		return value + "0"
	}
	return value
}
