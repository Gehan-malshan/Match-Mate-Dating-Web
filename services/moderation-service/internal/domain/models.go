package domain

import "time"

type Principal struct {
	Subject string
	Roles   []string
}

func (p Principal) HasRole(want string) bool {
	for _, role := range p.Roles {
		if role == want {
			return true
		}
	}
	return false
}

type ReportStatus string

const (
	ReportOpen          ReportStatus = "OPEN"
	ReportTriaged       ReportStatus = "TRIAGED"
	ReportInvestigating ReportStatus = "INVESTIGATING"
	ReportActioned      ReportStatus = "ACTIONED"
	ReportDismissed     ReportStatus = "DISMISSED"
)

type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

type TargetType string

const (
	TargetAccount TargetType = "ACCOUNT"
	TargetProfile TargetType = "PROFILE"
	TargetEvent   TargetType = "EVENT"
	TargetBooking TargetType = "BOOKING"
	TargetPairing TargetType = "PAIRING"
)

type ReportCategory string

const (
	CategoryHarassment           ReportCategory = "HARASSMENT"
	CategorySafety               ReportCategory = "SAFETY"
	CategoryImpersonation        ReportCategory = "IMPERSONATION"
	CategoryContactSharing       ReportCategory = "CONTACT_SHARING"
	CategoryInappropriateContent ReportCategory = "INAPPROPRIATE_CONTENT"
	CategoryFraud                ReportCategory = "FRAUD"
	CategoryOther                ReportCategory = "OTHER"
)

type ActionClass string

const (
	ActionContentHide          ActionClass = "CONTENT_HIDE"
	ActionProfileRestriction   ActionClass = "PROFILE_RESTRICTION"
	ActionEventExclusion       ActionClass = "EVENT_EXCLUSION"
	ActionAccountSuspension    ActionClass = "ACCOUNT_SUSPENSION"
	ActionMatchmakingExclusion ActionClass = "MATCHMAKING_EXCLUSION"
	ActionPairingInvalidation  ActionClass = "PAIRING_INVALIDATION"
	ActionRevealPrevention     ActionClass = "REVEAL_PREVENTION"
)

type EvidenceInput struct {
	Reference   string     `json:"reference"`
	MediaType   string     `json:"mediaType"`
	SHA256      string     `json:"sha256"`
	RetainUntil *time.Time `json:"retainUntil,omitempty"`
}
type CreateReportInput struct {
	TargetType  TargetType      `json:"targetType"`
	TargetID    string          `json:"targetId"`
	Category    ReportCategory  `json:"category"`
	Description string          `json:"description"`
	EventID     string          `json:"eventId,omitempty"`
	Evidence    []EvidenceInput `json:"evidence,omitempty"`
}
type Report struct {
	ID          string         `json:"reportId"`
	CaseID      string         `json:"caseId"`
	TargetType  TargetType     `json:"targetType"`
	TargetID    string         `json:"targetId"`
	Category    ReportCategory `json:"category"`
	Description string         `json:"description,omitempty"`
	Status      ReportStatus   `json:"status"`
	Severity    Severity       `json:"severity"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}
type Case struct {
	ID         string       `json:"caseId"`
	Status     ReportStatus `json:"status"`
	Severity   Severity     `json:"severity"`
	AssigneeID string       `json:"assigneeId,omitempty"`
	SLAAt      *time.Time   `json:"slaAt,omitempty"`
	Version    int64        `json:"version"`
	CreatedAt  time.Time    `json:"createdAt"`
	UpdatedAt  time.Time    `json:"updatedAt"`
	Reports    []Report     `json:"reports,omitempty"`
	Actions    []Action     `json:"actions,omitempty"`
	Appeals    []Appeal     `json:"appeals,omitempty"`
}
type Action struct {
	ID          string      `json:"actionId"`
	CaseID      string      `json:"caseId"`
	TargetType  TargetType  `json:"targetType"`
	TargetID    string      `json:"targetId"`
	Class       ActionClass `json:"actionClass"`
	Scope       string      `json:"scope"`
	Reason      string      `json:"reason"`
	Version     int64       `json:"version"`
	EffectiveAt time.Time   `json:"effectiveAt"`
	ExpiresAt   *time.Time  `json:"expiresAt,omitempty"`
	State       string      `json:"state"`
	CreatedAt   time.Time   `json:"createdAt"`
}
type Appeal struct {
	ID             string     `json:"appealId"`
	ActionID       string     `json:"actionId"`
	AppellantID    string     `json:"appellantId,omitempty"`
	Reason         string     `json:"reason,omitempty"`
	State          string     `json:"state"`
	DecisionReason string     `json:"decisionReason,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	DecidedAt      *time.Time `json:"decidedAt,omitempty"`
}
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	Limit      int    `json:"limit"`
}
type ProblemError struct {
	Status       int
	Code, Detail string
	Fields       map[string]string
}

func (e *ProblemError) Error() string { return e.Code }

type Fact struct {
	EventID, EventType                  string
	SchemaVersion                       int
	OccurredAt                          time.Time
	AggregateID, CorrelationID, ActorID string
	Payload                             map[string]any
}
