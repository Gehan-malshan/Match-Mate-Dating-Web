package engine

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/domain"
)

func rules() domain.Ruleset {
	return domain.Ruleset{Version: "prototype-v1", MinimumScore: 45, MissingDataPolicy: "IGNORE_AND_RENORMALIZE", Weights: map[string]int{"relationship": 25, "personality": 20, "interests": 20, "lifestyle": 15, "values": 10, "language_location": 10}}
}
func person(id, group string) domain.Participant {
	return domain.Participant{AccountID: id, ParticipantCode: id, Group: group, AcceptedGroups: []string{map[string]string{"A": "B", "B": "A"}[group]}, Age: 30, MinimumAge: 25, MaximumAge: 40, Active: true, Verified: true, ProfileApproved: true, BookingConfirmed: true, RelationshipIntent: "long-term", Interests: []string{"books", "travel"}, Personality: map[string]int{"social": 3}, Lifestyle: map[string]string{"smoking": "no", "activity": "active"}, Values: []string{"kindness", "growth"}, Languages: []string{"English"}, BroadLocation: "Colombo"}
}

func TestHardRulesCannotBeOutscored(t *testing.T) {
	a, b := person("a", "A"), person("b", "B")
	a.BlockedAccountIDs = []string{"b"}
	candidate := Evaluate(a, b, rules())
	if candidate.Eligible || !reflect.DeepEqual(candidate.RejectionCodes, []string{"BLOCKED_RELATIONSHIP"}) {
		t.Fatalf("unexpected candidate: %#v", candidate)
	}
}

func TestEveryHardRule(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		mutate func(*domain.Participant, *domain.Participant, *domain.Ruleset)
	}{
		{"account", "ACCOUNT_INELIGIBLE", func(a, _ *domain.Participant, _ *domain.Ruleset) { a.Active = false }},
		{"profile", "PROFILE_INCOMPLETE", func(a, _ *domain.Participant, _ *domain.Ruleset) { a.ProfileApproved = false }},
		{"booking", "BOOKING_NOT_CONFIRMED", func(_, b *domain.Participant, _ *domain.Ruleset) { b.BookingConfirmed = false }},
		{"group preference", "PARTNER_PREFERENCE_MISMATCH", func(a, _ *domain.Participant, _ *domain.Ruleset) { a.AcceptedGroups = []string{"C"} }},
		{"age", "AGE_RANGE_MISMATCH", func(_, b *domain.Participant, _ *domain.Ruleset) { b.Age = 50 }},
		{"block", "BLOCKED_RELATIONSHIP", func(a, _ *domain.Participant, _ *domain.Ruleset) { a.BlockedAccountIDs = []string{"b"} }},
		{"safety", "SAFETY_EXCLUSION", func(_, b *domain.Participant, _ *domain.Ruleset) { b.SafetyExcluded = true }},
		{"deal breaker", "DEAL_BREAKER_MISMATCH", func(a, _ *domain.Participant, _ *domain.Ruleset) { a.DealBreakers = map[string]string{"smoking": "no"} }},
		{"repeat", "REPEAT_PAIR_NOT_ALLOWED", func(a, _ *domain.Participant, _ *domain.Ruleset) { a.PriorPartnerIDs = []string{"b"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b, r := person("a", "A"), person("b", "B"), rules()
			tt.mutate(&a, &b, &r)
			candidate := Evaluate(a, b, r)
			if candidate.Eligible || !reflect.DeepEqual(candidate.RejectionCodes, []string{tt.code}) {
				t.Fatalf("wanted %s, got %#v", tt.code, candidate)
			}
		})
	}
}

func TestSameGroupViolatesEventPolicy(t *testing.T) {
	a, b := person("a", "A"), person("b", "A")
	a.AcceptedGroups, b.AcceptedGroups = []string{"A"}, []string{"A"}
	candidate := Evaluate(a, b, rules())
	if !reflect.DeepEqual(candidate.RejectionCodes, []string{"EVENT_POLICY_MISMATCH"}) {
		t.Fatalf("unexpected rejection: %#v", candidate.RejectionCodes)
	}
}

func TestPerfectComponentScores(t *testing.T) {
	candidate := Evaluate(person("a", "A"), person("b", "B"), rules())
	if !candidate.Eligible || candidate.TotalScore != 100 {
		t.Fatalf("expected perfect score: %#v", candidate)
	}
	for name, score := range candidate.Components {
		if score < 0 || score > 100 {
			t.Fatalf("component %s is outside 0..100: %d", name, score)
		}
	}
}
func TestMissingComponentsAreRenormalized(t *testing.T) {
	a, b := person("a", "A"), person("b", "B")
	a.Personality = nil
	b.Personality = nil
	a.Lifestyle = nil
	b.Lifestyle = nil
	c := Evaluate(a, b, rules())
	if !c.Eligible || c.TotalScore < 90 {
		t.Fatalf("missing optional data became a negative: %#v", c)
	}
}
func TestOptimizerFindsGlobalResultAndIsDeterministic(t *testing.T) {
	people := []domain.Participant{person("a1", "A"), person("a2", "A"), person("b1", "B"), person("b2", "B")}
	people[0].Interests = []string{"music"}
	people[1].Interests = []string{"books"}
	people[2].Interests = []string{"books"}
	people[3].Interests = []string{"music"}
	first, err := Generate(people, rules())
	if err != nil {
		t.Fatal(err)
	}
	second, _ := Generate(people, rules())
	if !reflect.DeepEqual(first, second) {
		t.Fatal("identical inputs produced different output")
	}
	if len(first.Pairings) != 2 || first.Pairings[0].ParticipantB != "b2" || first.Pairings[1].ParticipantB != "b1" {
		t.Fatalf("unexpected optimum: %#v", first.Pairings)
	}
}

func TestOptimizerBeatsGreedyChoice(t *testing.T) {
	left := []domain.Participant{person("a1", "A"), person("a2", "A")}
	right := []domain.Participant{person("b1", "B"), person("b2", "B")}
	edges := map[string]domain.Candidate{}
	for key, score := range map[string]int{
		pairKey("a1", "b1"): 100,
		pairKey("a1", "b2"): 99,
		pairKey("a2", "b1"): 98,
		pairKey("a2", "b2"): 1,
	} {
		edges[key] = domain.Candidate{Eligible: true, TotalScore: score}
	}
	got := optimize(left, right, edges)
	if len(got) != 2 || got[0].ParticipantB != "b2" || got[1].ParticipantB != "b1" || got[0].Score+got[1].Score != 197 {
		t.Fatalf("expected global score 197 instead of greedy score 101: %#v", got)
	}
}

func TestEqualObjectivesAreDeterministic(t *testing.T) {
	people := []domain.Participant{person("a2", "A"), person("b2", "B"), person("a1", "A"), person("b1", "B")}
	first, err := Generate(people, rules())
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Pairings) != 2 || first.Pairings[0].ParticipantA != "a1" || first.Pairings[0].ParticipantB != "b1" || first.Pairings[1].ParticipantA != "a2" || first.Pairings[1].ParticipantB != "b2" {
		t.Fatalf("unexpected account-ID tie break: %#v", first.Pairings)
	}
	for i := 0; i < 20; i++ {
		got, err := Generate(people, rules())
		if err != nil || !reflect.DeepEqual(got.Pairings, first.Pairings) {
			t.Fatalf("run %d was not deterministic: %#v, %v", i, got.Pairings, err)
		}
	}
}

func TestOptimizerPrioritizesPairCountBeforeScore(t *testing.T) {
	left := []domain.Participant{person("a1", "A"), person("a2", "A")}
	right := []domain.Participant{person("b1", "B"), person("b2", "B")}
	edges := map[string]domain.Candidate{
		pairKey("a1", "b1"): {Eligible: true, TotalScore: 100},
		pairKey("a1", "b2"): {Eligible: true, TotalScore: 0},
		pairKey("a2", "b1"): {Eligible: true, TotalScore: 0},
	}
	got := optimize(left, right, edges)
	if len(got) != 2 {
		t.Fatalf("expected two eligible pairs even though one-pair score is higher: %#v", got)
	}
}
func TestUnmatchedReasons(t *testing.T) {
	a, b := person("a", "A"), person("b", "B")
	b.BookingConfirmed = false
	result, err := Generate([]domain.Participant{a, b}, rules())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Pairings) != 0 || len(result.Unmatched) != 2 || result.Unmatched[0].Reason != "NO_ELIGIBLE_CANDIDATE" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestBelowThresholdAndGroupImbalanceReasons(t *testing.T) {
	a, b := person("a", "A"), person("b", "B")
	a.RelationshipIntent = "casual"
	r := rules()
	r.MinimumScore = 100
	result, err := Generate([]domain.Participant{a, b}, r)
	if err != nil || len(result.Unmatched) != 2 || result.Unmatched[0].Reason != "BELOW_MINIMUM_SCORE" {
		t.Fatalf("unexpected below-threshold result: %#v, %v", result, err)
	}

	result, err = Generate([]domain.Participant{person("a1", "A"), person("a2", "A"), person("b1", "B")}, rules())
	if err != nil || len(result.Pairings) != 1 || len(result.Unmatched) != 1 || result.Unmatched[0].Reason != "GROUP_CAPACITY_IMBALANCE" {
		t.Fatalf("unexpected imbalance result: %#v, %v", result, err)
	}
}

func TestParticipantValidation(t *testing.T) {
	tests := []struct {
		name   string
		people []domain.Participant
		want   string
	}{
		{"too few", []domain.Participant{person("a", "A")}, "at least two"},
		{"duplicate account", []domain.Participant{person("a", "A"), person("a", "B")}, "accountId must be unique"},
		{"duplicate code", func() []domain.Participant {
			a, b := person("a", "A"), person("b", "B")
			b.ParticipantCode = a.ParticipantCode
			return []domain.Participant{a, b}
		}(), "participantCode must be unique"},
		{"invalid age", func() []domain.Participant {
			a, b := person("a", "A"), person("b", "B")
			a.MinimumAge = 50
			a.MaximumAge = 20
			return []domain.Participant{a, b}
		}(), "age preferences"},
		{"invalid personality", func() []domain.Participant {
			a, b := person("a", "A"), person("b", "B")
			a.Personality["social"] = 8
			return []domain.Participant{a, b}
		}(), "personality values"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Generate(tt.people, rules())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("wanted error containing %q, got %v", tt.want, err)
			}
		})
	}
}
func TestRulesetWeightsMustSumToOneHundred(t *testing.T) {
	r := rules()
	r.Weights["values"] = 9
	if ValidateRuleset(r) == nil {
		t.Fatal("invalid ruleset accepted")
	}
}

func TestRulesetRejectsUnknownWeight(t *testing.T) {
	r := rules()
	r.Weights["unknown"] = 0
	if ValidateRuleset(r) == nil {
		t.Fatal("unknown component weight was accepted")
	}
}
