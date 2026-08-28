package domain

import "time"

type Status string

const (
	Draft              Status = "DRAFT"
	Published          Status = "PUBLISHED"
	RegistrationOpen   Status = "REGISTRATION_OPEN"
	RegistrationClosed Status = "REGISTRATION_CLOSED"
	Cancelled          Status = "CANCELLED"
)

type Event struct {
	ID                     string    `json:"eventId"`
	OrganizerID            string    `json:"organizerId,omitempty"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	VenueName              string    `json:"venueName,omitempty"`
	BroadLocation          string    `json:"broadLocation"`
	TimeZone               string    `json:"timeZone"`
	StartsAt               time.Time `json:"startsAt"`
	EndsAt                 time.Time `json:"endsAt"`
	RegistrationOpensAt    time.Time `json:"registrationOpensAt"`
	RegistrationClosesAt   time.Time `json:"registrationClosesAt"`
	Price                  string    `json:"price"`
	Currency               string    `json:"currency"`
	ConfiguredCapacity     int       `json:"configuredCapacity"`
	CapacityPolicyVersion  int64     `json:"capacityPolicyVersion"`
	MatchingRulesetVersion string    `json:"matchingRulesetVersion"`
	Status                 Status    `json:"status"`
	Version                int64     `json:"version"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type CreateInput struct {
	OrganizerID            string    `json:"organizerId"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	VenueName              string    `json:"venueName"`
	BroadLocation          string    `json:"broadLocation"`
	TimeZone               string    `json:"timeZone"`
	StartsAt               time.Time `json:"startsAt"`
	EndsAt                 time.Time `json:"endsAt"`
	RegistrationOpensAt    time.Time `json:"registrationOpensAt"`
	RegistrationClosesAt   time.Time `json:"registrationClosesAt"`
	Price                  string    `json:"price"`
	Currency               string    `json:"currency"`
	ConfiguredCapacity     int       `json:"configuredCapacity"`
	MatchingRulesetVersion string    `json:"matchingRulesetVersion"`
}
type UpdateInput struct {
	CreateInput
	ExpectedVersion int64 `json:"expectedVersion"`
}
type Page struct {
	Items      []Event `json:"items"`
	NextCursor string  `json:"nextCursor,omitempty"`
	Limit      int     `json:"limit"`
}
type PublicEvent struct {
	ID                     string    `json:"eventId"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	BroadLocation          string    `json:"broadLocation"`
	TimeZone               string    `json:"timeZone"`
	StartsAt               time.Time `json:"startsAt"`
	EndsAt                 time.Time `json:"endsAt"`
	RegistrationOpensAt    time.Time `json:"registrationOpensAt"`
	RegistrationClosesAt   time.Time `json:"registrationClosesAt"`
	Price                  string    `json:"price"`
	Currency               string    `json:"currency"`
	ConfiguredCapacity     int       `json:"configuredCapacity"`
	MatchingRulesetVersion string    `json:"matchingRulesetVersion"`
	Status                 Status    `json:"status"`
	Version                int64     `json:"version"`
}

func (e Event) Public() PublicEvent {
	return PublicEvent{e.ID, e.Name, e.Description, e.BroadLocation, e.TimeZone, e.StartsAt, e.EndsAt, e.RegistrationOpensAt, e.RegistrationClosesAt, e.Price, e.Currency, e.ConfiguredCapacity, e.MatchingRulesetVersion, e.Status, e.Version}
}

type Principal struct {
	Subject string
	Roles   []string
}

func (p Principal) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type Fact struct {
	EventID, EventType                  string
	SchemaVersion                       int
	OccurredAt                          time.Time
	AggregateID, CorrelationID, ActorID string
	Payload                             map[string]any
}
type ProblemError struct {
	Status       int
	Code, Detail string
	Fields       map[string]string
}

func (e *ProblemError) Error() string { return e.Code }
