package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

type State string

const (
	Pending   State = "PENDING"
	Completed State = "COMPLETED"
	Failed    State = "FAILED"
	Review    State = "REVIEW"
)

var moneyPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.[0-9]{2}$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type Payment struct {
	ID                string     `json:"paymentId"`
	BookingID         string     `json:"bookingId"`
	AccountID         string     `json:"-"`
	OrderID           string     `json:"orderId"`
	Amount            string     `json:"amount"`
	Currency          string     `json:"currency"`
	Provider          string     `json:"provider"`
	ProviderPaymentID string     `json:"-"`
	State             State      `json:"state"`
	Version           int64      `json:"version"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
}

type BookingSnapshot struct {
	BookingID, AccountID, Amount, Currency, Status string
	ExpiresAt                                      time.Time
}

type Principal struct {
	Subject string
	Roles   []string
}

func ValidateSnapshot(s BookingSnapshot, actorID string, now time.Time) error {
	if s.BookingID == "" || s.AccountID != actorID {
		return errors.New("booking is not accessible")
	}
	if s.Status != "PENDING_PAYMENT" {
		return errors.New("booking is not pending payment")
	}
	if !s.ExpiresAt.After(now) {
		return errors.New("booking hold expired")
	}
	if !moneyPattern.MatchString(s.Amount) || !currencyPattern.MatchString(s.Currency) {
		return errors.New("invalid booking price snapshot")
	}
	return nil
}

func CallbackState(status string) State {
	switch strings.TrimSpace(status) {
	case "2":
		return Completed
	case "-1", "-2", "-3":
		return Failed
	default:
		return Pending
	}
}
