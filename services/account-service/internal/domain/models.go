package domain

import "time"

const (
	AccountActive        = "ACTIVE"
	AccountDeactivated   = "DEACTIVATED"
	AccountSuspended     = "SUSPENDED"
	VerificationPending  = "PENDING"
	VerificationVerified = "VERIFIED"
	VisibilityPrivate    = "PRIVATE"
	VisibilityCommunity  = "COMMUNITY"
	VisibilityHidden     = "HIDDEN"
	ApprovalPending      = "PENDING"
	ApprovalApproved     = "APPROVED"
	ApprovalHidden       = "HIDDEN"
)

type Account struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Status       string    `json:"status"`
	Verification string    `json:"verification"`
	Roles        []string  `json:"roles"`
	TokenVersion int64     `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
type Profile struct {
	AccountID     string    `json:"accountId"`
	Nickname      string    `json:"nickname"`
	DateOfBirth   string    `json:"dateOfBirth"`
	BroadLocation string    `json:"broadLocation"`
	Bio           string    `json:"bio"`
	Visibility    string    `json:"visibility"`
	Approval      string    `json:"approval"`
	Interests     []string  `json:"interests"`
	Version       int64     `json:"version"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
type Preferences struct {
	MinAge       int       `json:"minAge"`
	MaxAge       int       `json:"maxAge"`
	Intentions   []string  `json:"intentions"`
	InterestedIn []string  `json:"interestedIn"`
	Languages    []string  `json:"languages"`
	DealBreakers []string  `json:"dealBreakers"`
	Version      int64     `json:"version"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
type Me struct {
	Account     Account      `json:"account"`
	Profile     Profile      `json:"profile"`
	Preferences *Preferences `json:"preferences,omitempty"`
}
type CommunityProfile struct {
	ProfileID     string   `json:"profileId"`
	Nickname      string   `json:"nickname"`
	AgeBand       string   `json:"ageBand"`
	BroadLocation string   `json:"broadLocation"`
	Bio           string   `json:"bio"`
	Interests     []string `json:"interests"`
}
type RegisterInput struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	Nickname       string `json:"nickname"`
	DateOfBirth    string `json:"dateOfBirth"`
	ConsentVersion string `json:"consentVersion"`
}
type ProfilePatch struct {
	Nickname        *string   `json:"nickname"`
	DateOfBirth     *string   `json:"dateOfBirth"`
	BroadLocation   *string   `json:"broadLocation"`
	Bio             *string   `json:"bio"`
	Visibility      *string   `json:"visibility"`
	Interests       *[]string `json:"interests"`
	ExpectedVersion int64     `json:"expectedVersion"`
}
type PreferenceInput struct {
	MinAge       int      `json:"minAge"`
	MaxAge       int      `json:"maxAge"`
	Intentions   []string `json:"intentions"`
	InterestedIn []string `json:"interestedIn"`
	Languages    []string `json:"languages"`
	DealBreakers []string `json:"dealBreakers"`
}
type Session struct {
	ID, FamilyID, AccountID string
	TokenHash               []byte
	ExpiresAt               time.Time
}
type TokenPair struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"-"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshExpiresAt time.Time `json:"-"`
}
type Registration struct {
	Me                Me     `json:"me"`
	VerificationToken string `json:"verificationToken,omitempty"`
}
type Event struct {
	EventID, EventType                               string
	SchemaVersion                                    int
	OccurredAt                                       time.Time
	AggregateID, CorrelationID, CausationID, ActorID string
	Payload                                          map[string]any
}

type ProblemError struct {
	Status              int
	Code, Title, Detail string
	Fields              map[string]string
}

func (e *ProblemError) Error() string { return e.Code }
