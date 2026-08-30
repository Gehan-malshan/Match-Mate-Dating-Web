package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"
)

type Action string

const (
	ActionDeliver  Action = "DELIVER"
	ActionSuppress Action = "SUPPRESS"
	ActionIgnore   Action = "IGNORE"
)

type DeliveryState string

const (
	DeliveryPending           DeliveryState = "PENDING"
	DeliveryProcessing        DeliveryState = "PROCESSING"
	DeliveryRetryScheduled    DeliveryState = "RETRY_SCHEDULED"
	DeliveryDelivered         DeliveryState = "DELIVERED"
	DeliverySuppressed        DeliveryState = "SUPPRESSED"
	DeliveryPermanentlyFailed DeliveryState = "PERMANENTLY_FAILED"
	DeliveryDeadLettered      DeliveryState = "DEAD_LETTERED"
)

type EventEnvelope struct {
	EventID       string          `json:"eventId"`
	EventType     string          `json:"eventType"`
	SchemaVersion int             `json:"schemaVersion"`
	OccurredAt    time.Time       `json:"occurredAt"`
	AggregateID   string          `json:"aggregateId"`
	CorrelationID string          `json:"correlationId"`
	CausationID   *string         `json:"causationId"`
	ActorID       string          `json:"actorId"`
	Payload       json.RawMessage `json:"payload"`
}

type Plan struct {
	Action             Action
	TemplateKey        string
	Locale             string
	Channel            string
	Category           string
	RecipientAccountID string
	BusinessKey        string
	SourceAggregateID  string
	SuppressionReason  string
	Variables          map[string]string
}

type Template struct {
	ID               string
	Key              string
	Version          int
	Locale           string
	Channel          string
	Category         string
	SubjectTemplate  string
	BodyTemplate     string
	AllowedVariables []string
}

type Delivery struct {
	ID                 string
	BusinessKey        string
	RecipientAccountID string
	SourceEventID      string
	SourceEventType    string
	SourceAggregateID  string
	Template           Template
	Variables          map[string]string
	State              DeliveryState
	AttemptCount       int
	MaxAttempts        int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type FeedCursor struct {
	CreatedAt time.Time
	ID        string
}

type FeedRecord struct {
	ID              string
	SourceEventType string
	Category        string
	Template        Template
	Variables       map[string]string
	ReadAt          *time.Time
	CreatedAt       time.Time
}

type FeedItem struct {
	ID              string     `json:"notificationId"`
	SourceEventType string     `json:"sourceEventType"`
	Category        string     `json:"category"`
	Title           string     `json:"title"`
	Message         string     `json:"message"`
	ActionPath      string     `json:"actionPath"`
	ReadAt          *time.Time `json:"readAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type RenderedMessage struct {
	Subject string
	Body    string
}

func Render(t Template, variables map[string]string) (RenderedMessage, error) {
	allowed := make(map[string]struct{}, len(t.AllowedVariables))
	for _, key := range t.AllowedVariables {
		allowed[key] = struct{}{}
	}
	for key := range variables {
		if _, ok := allowed[key]; !ok {
			return RenderedMessage{}, fmt.Errorf("template variable %q is not allowed", key)
		}
	}
	for _, key := range t.AllowedVariables {
		if _, ok := variables[key]; !ok {
			return RenderedMessage{}, fmt.Errorf("template variable %q is required", key)
		}
	}

	render := func(name, source string) (string, error) {
		parsed, err := template.New(name).Option("missingkey=error").Parse(source)
		if err != nil {
			return "", err
		}
		var output bytes.Buffer
		if err = parsed.Execute(&output, variables); err != nil {
			return "", err
		}
		return strings.TrimSpace(output.String()), nil
	}
	subject, err := render("subject", t.SubjectTemplate)
	if err != nil {
		return RenderedMessage{}, err
	}
	body, err := render("body", t.BodyTemplate)
	if err != nil {
		return RenderedMessage{}, err
	}
	if subject == "" || body == "" || strings.ContainsAny(subject, "\r\n") {
		return RenderedMessage{}, errors.New("rendered notification subject/body is invalid")
	}
	return RenderedMessage{Subject: subject, Body: body}, nil
}

type FailureKind string

const (
	FailureRetryable FailureKind = "RETRYABLE"
	FailurePermanent FailureKind = "PERMANENT"
)

type SendFailure struct {
	Kind FailureKind
	Code string
	Err  error
}

func (e *SendFailure) Error() string {
	if e.Err == nil {
		return e.Code
	}
	return e.Err.Error()
}

func (e *SendFailure) Unwrap() error { return e.Err }

func RetryDelay(base time.Duration, attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	shift := attempt - 1
	if shift > 6 {
		shift = 6
	}
	return base * time.Duration(1<<shift)
}
