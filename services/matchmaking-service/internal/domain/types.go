package domain

import "time"

type Principal struct {
	Subject string
	Roles   []string
}

func (p Principal) HasRole(want ...string) bool {
	for _, actual := range p.Roles {
		for _, role := range want {
			if actual == role {
				return true
			}
		}
	}
	return false
}

type Participant struct {
	AccountID          string            `json:"accountId"`
	ParticipantCode    string            `json:"participantCode"`
	Group              string            `json:"group"`
	AcceptedGroups     []string          `json:"acceptedGroups"`
	Age                int               `json:"age"`
	MinimumAge         int               `json:"minimumAge"`
	MaximumAge         int               `json:"maximumAge"`
	Active             bool              `json:"active"`
	Verified           bool              `json:"verified"`
	ProfileApproved    bool              `json:"profileApproved"`
	BookingConfirmed   bool              `json:"bookingConfirmed"`
	SafetyExcluded     bool              `json:"safetyExcluded"`
	BlockedAccountIDs  []string          `json:"blockedAccountIds"`
	RelationshipIntent string            `json:"relationshipIntent"`
	Interests          []string          `json:"interests"`
	Personality        map[string]int    `json:"personality"`
	Lifestyle          map[string]string `json:"lifestyle"`
	Values             []string          `json:"values"`
	Languages          []string          `json:"languages"`
	BroadLocation      string            `json:"broadLocation"`
	DealBreakers       map[string]string `json:"dealBreakers"`
	PriorPartnerIDs    []string          `json:"priorPartnerIds"`
}

type Ruleset struct {
	Version             string         `json:"version"`
	Weights             map[string]int `json:"weights"`
	MinimumScore        int            `json:"minimumScore"`
	AllowRepeatPairings bool           `json:"allowRepeatPairings"`
	MissingDataPolicy   string         `json:"missingDataPolicy"`
}

type Candidate struct {
	ParticipantA   string         `json:"participantA"`
	ParticipantB   string         `json:"participantB"`
	Eligible       bool           `json:"eligible"`
	RejectionCodes []string       `json:"rejectionCodes,omitempty"`
	Components     map[string]int `json:"components,omitempty"`
	TotalScore     int            `json:"totalScore,omitempty"`
	SafeReasons    []string       `json:"safeReasons,omitempty"`
}

type Pairing struct {
	PairingID        string   `json:"pairingId,omitempty"`
	ParticipantA     string   `json:"participantA"`
	ParticipantB     string   `json:"participantB"`
	ParticipantACode string   `json:"participantACode,omitempty"`
	ParticipantBCode string   `json:"participantBCode,omitempty"`
	Score            int      `json:"score"`
	SafeReasons      []string `json:"safeReasons"`
	Source           string   `json:"source,omitempty"`
}

type Unmatched struct {
	ParticipantID string `json:"participantId"`
	Code          string `json:"participantCode,omitempty"`
	Reason        string `json:"reason"`
}

type EngineResult struct {
	Candidates []Candidate `json:"candidates"`
	Pairings   []Pairing   `json:"pairings"`
	Unmatched  []Unmatched `json:"unmatched"`
}

type Run struct {
	RunID             string      `json:"runId"`
	EventID           string      `json:"eventId"`
	RunVersion        int         `json:"runVersion"`
	Version           int64       `json:"version"`
	Status            string      `json:"status"`
	RulesetVersion    string      `json:"rulesetVersion"`
	OptimizerVersion  string      `json:"optimizerVersion"`
	TieBreakPolicy    string      `json:"tieBreakPolicy"`
	ParticipantCount  int         `json:"participantCount"`
	EligiblePairCount int         `json:"eligiblePairCount"`
	CreatedBy         string      `json:"createdBy"`
	CreatedAt         time.Time   `json:"createdAt"`
	UpdatedAt         time.Time   `json:"updatedAt"`
	Suggestions       []Pairing   `json:"suggestions,omitempty"`
	Selections        []Pairing   `json:"selections,omitempty"`
	Unmatched         []Unmatched `json:"unmatched,omitempty"`
	Candidates        []Candidate `json:"candidates,omitempty"`
}

type ProblemError struct {
	Status int
	Code   string
	Detail string
	Fields map[string]string
}

func (e *ProblemError) Error() string { return e.Code }
