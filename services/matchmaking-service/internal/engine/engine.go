package engine

import (
	"errors"
	"math"
	"sort"
	"strings"

	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/domain"
)

const (
	OptimizerVersion     = "hungarian-v1"
	maxPrototypePeople   = 10_000
	cardinalityEdgeBonus = int64(maxPrototypePeople*100 + 1)
)

var componentOrder = []string{"relationship", "personality", "interests", "lifestyle", "values", "language_location"}

func ValidateRuleset(r domain.Ruleset) error {
	if strings.TrimSpace(r.Version) == "" || r.MinimumScore < 0 || r.MinimumScore > 100 || r.MissingDataPolicy != "IGNORE_AND_RENORMALIZE" {
		return errors.New("invalid ruleset")
	}
	if len(r.Weights) != len(componentOrder) {
		return errors.New("ruleset contains an unknown or missing component weight")
	}
	total := 0
	for _, name := range componentOrder {
		weight, ok := r.Weights[name]
		if !ok || weight < 0 {
			return errors.New("ruleset is missing a component weight")
		}
		total += weight
	}
	if total != 100 {
		return errors.New("ruleset weights must sum to 100")
	}
	return nil
}

func Generate(participants []domain.Participant, ruleset domain.Ruleset) (domain.EngineResult, error) {
	if err := ValidateRuleset(ruleset); err != nil {
		return domain.EngineResult{}, err
	}
	if err := validateParticipants(participants); err != nil {
		return domain.EngineResult{}, err
	}
	people := append([]domain.Participant(nil), participants...)
	sort.Slice(people, func(i, j int) bool { return people[i].AccountID < people[j].AccountID })
	groups := map[string][]domain.Participant{}
	for _, p := range people {
		groups[p.Group] = append(groups[p.Group], p)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) != 2 {
		return domain.EngineResult{}, errors.New("prototype ruleset requires exactly two participant groups")
	}

	left, right := groups[keys[0]], groups[keys[1]]
	result := domain.EngineResult{}
	edges := map[string]domain.Candidate{}
	for _, a := range left {
		for _, b := range right {
			candidate := Evaluate(a, b, ruleset)
			result.Candidates = append(result.Candidates, candidate)
			if candidate.Eligible && candidate.TotalScore >= ruleset.MinimumScore {
				edges[pairKey(a.AccountID, b.AccountID)] = candidate
			}
		}
	}
	result.Pairings = optimize(left, right, edges)
	paired := map[string]bool{}
	for _, pair := range result.Pairings {
		paired[pair.ParticipantA], paired[pair.ParticipantB] = true, true
	}
	for _, p := range people {
		if paired[p.AccountID] {
			continue
		}
		reason := unmatchedReason(p.AccountID, result.Candidates, ruleset.MinimumScore)
		result.Unmatched = append(result.Unmatched, domain.Unmatched{ParticipantID: p.AccountID, Code: p.ParticipantCode, Reason: reason})
	}
	return result, nil
}

func Evaluate(a, b domain.Participant, ruleset domain.Ruleset) domain.Candidate {
	candidate := domain.Candidate{ParticipantA: a.AccountID, ParticipantB: b.AccountID}
	candidate.RejectionCodes = hardRules(a, b, ruleset)
	if len(candidate.RejectionCodes) > 0 {
		return candidate
	}
	candidate.Eligible = true
	components := map[string]int{}
	if a.RelationshipIntent != "" && b.RelationshipIntent != "" {
		if a.RelationshipIntent == b.RelationshipIntent {
			components["relationship"] = 100
		} else {
			components["relationship"] = 60
		}
	}
	if score, ok := personality(a.Personality, b.Personality); ok {
		components["personality"] = score
	}
	if score, ok := jaccard(a.Interests, b.Interests); ok {
		components["interests"] = score
	}
	if score, ok := mapSimilarity(a.Lifestyle, b.Lifestyle); ok {
		components["lifestyle"] = score
	}
	if score, ok := jaccard(a.Values, b.Values); ok {
		components["values"] = score
	}
	if score, ok := languageLocation(a, b); ok {
		components["language_location"] = score
	}
	candidate.Components = components
	candidate.TotalScore = weighted(components, ruleset.Weights)
	candidate.SafeReasons = explain(components)
	return candidate
}

func hardRules(a, b domain.Participant, r domain.Ruleset) []string {
	reasons := []string{}
	if a.AccountID == b.AccountID || a.Group == b.Group {
		reasons = append(reasons, "EVENT_POLICY_MISMATCH")
	}
	if !a.Active || !b.Active || !a.Verified || !b.Verified {
		reasons = append(reasons, "ACCOUNT_INELIGIBLE")
	}
	if !a.ProfileApproved || !b.ProfileApproved {
		reasons = append(reasons, "PROFILE_INCOMPLETE")
	}
	if !a.BookingConfirmed || !b.BookingConfirmed {
		reasons = append(reasons, "BOOKING_NOT_CONFIRMED")
	}
	if !contains(a.AcceptedGroups, b.Group) || !contains(b.AcceptedGroups, a.Group) {
		reasons = append(reasons, "PARTNER_PREFERENCE_MISMATCH")
	}
	if b.Age < a.MinimumAge || b.Age > a.MaximumAge || a.Age < b.MinimumAge || a.Age > b.MaximumAge {
		reasons = append(reasons, "AGE_RANGE_MISMATCH")
	}
	if contains(a.BlockedAccountIDs, b.AccountID) || contains(b.BlockedAccountIDs, a.AccountID) {
		reasons = append(reasons, "BLOCKED_RELATIONSHIP")
	}
	if a.SafetyExcluded || b.SafetyExcluded {
		reasons = append(reasons, "SAFETY_EXCLUSION")
	}
	if breaks(a, b) || breaks(b, a) {
		reasons = append(reasons, "DEAL_BREAKER_MISMATCH")
	}
	if !r.AllowRepeatPairings && (contains(a.PriorPartnerIDs, b.AccountID) || contains(b.PriorPartnerIDs, a.AccountID)) {
		reasons = append(reasons, "REPEAT_PAIR_NOT_ALLOWED")
	}
	return unique(reasons)
}

func validateParticipants(participants []domain.Participant) error {
	if len(participants) < 2 {
		return errors.New("at least two participants are required")
	}
	if len(participants) > maxPrototypePeople {
		return errors.New("prototype participant limit exceeded")
	}
	accountIDs := make(map[string]struct{}, len(participants))
	codes := make(map[string]struct{}, len(participants))
	for _, p := range participants {
		if strings.TrimSpace(p.AccountID) == "" || strings.TrimSpace(p.ParticipantCode) == "" || strings.TrimSpace(p.Group) == "" {
			return errors.New("participant accountId, participantCode, and group are required")
		}
		if _, exists := accountIDs[p.AccountID]; exists {
			return errors.New("participant accountId must be unique")
		}
		accountIDs[p.AccountID] = struct{}{}
		if _, exists := codes[p.ParticipantCode]; exists {
			return errors.New("participantCode must be unique")
		}
		codes[p.ParticipantCode] = struct{}{}
		if p.Age < 18 || p.Age > 120 || p.MinimumAge < 18 || p.MaximumAge > 120 || p.MinimumAge > p.MaximumAge {
			return errors.New("participant age preferences are invalid")
		}
		for _, value := range p.Personality {
			if value < 1 || value > 5 {
				return errors.New("personality values must be between 1 and 5")
			}
		}
	}
	return nil
}

func breaks(owner, other domain.Participant) bool {
	for key, disallowed := range owner.DealBreakers {
		if other.Lifestyle[key] == disallowed {
			return true
		}
	}
	return false
}
func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
func unique(items []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

func personality(a, b map[string]int) (int, bool) {
	total, count := 0, 0
	for key, av := range a {
		if bv, ok := b[key]; ok {
			total += int(math.Abs(float64(av - bv)))
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return clamp(100 - (total * 25 / count)), true
}
func mapSimilarity(a, b map[string]string) (int, bool) {
	same, total := 0, 0
	for key, av := range a {
		if bv, ok := b[key]; ok {
			total++
			if av == bv {
				same++
			}
		}
	}
	if total == 0 {
		return 0, false
	}
	return same * 100 / total, true
}
func jaccard(a, b []string) (int, bool) {
	if len(a) == 0 || len(b) == 0 {
		return 0, false
	}
	set := map[string]int{}
	for _, v := range a {
		set[strings.ToLower(v)] |= 1
	}
	for _, v := range b {
		set[strings.ToLower(v)] |= 2
	}
	union, shared := 0, 0
	for _, mask := range set {
		union++
		if mask == 3 {
			shared++
		}
	}
	return int(math.Round(float64(shared) * 100 / float64(union))), true
}
func languageLocation(a, b domain.Participant) (int, bool) {
	language, ok := jaccard(a.Languages, b.Languages)
	locationKnown := a.BroadLocation != "" && b.BroadLocation != ""
	if !ok && !locationKnown {
		return 0, false
	}
	if !ok {
		if a.BroadLocation == b.BroadLocation {
			return 100, true
		}
		return 40, true
	}
	score := language * 70 / 100
	if locationKnown {
		if a.BroadLocation == b.BroadLocation {
			score += 30
		} else {
			score += 10
		}
	} else {
		score = language
	}
	return clamp(score), true
}
func weighted(scores, weights map[string]int) int {
	sum, active := 0, 0
	for name, score := range scores {
		weight := weights[name]
		sum += score * weight
		active += weight
	}
	if active == 0 {
		return 0
	}
	return int(math.Round(float64(sum) / float64(active)))
}
func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
func explain(c map[string]int) []string {
	out := []string{}
	if c["relationship"] >= 90 {
		out = append(out, "Same approved relationship intention")
	}
	if c["interests"] >= 40 {
		out = append(out, "Meaningful shared interests")
	}
	if c["lifestyle"] >= 70 {
		out = append(out, "Compatible lifestyle preferences")
	}
	if c["values"] >= 40 {
		out = append(out, "Several shared values")
	}
	if c["language_location"] >= 60 {
		out = append(out, "Shared language or broad location")
	}
	if len(out) == 0 {
		out = append(out, "Passed all reciprocal eligibility rules")
	}
	return out
}
func pairKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func optimize(left, right []domain.Participant, edges map[string]domain.Candidate) []domain.Pairing {
	size := len(left)
	if len(right) > size {
		size = len(right)
	}
	if size == 0 {
		return nil
	}
	weights := make([][]int64, size)
	for i := range weights {
		weights[i] = make([]int64, size)
	}
	for i, a := range left {
		for j, b := range right {
			if c, ok := edges[pairKey(a.AccountID, b.AccountID)]; ok {
				// One extra eligible pairing is worth more than the maximum total
				// score of every participant. Sorted inputs plus the stable
				// Hungarian scan make equal objectives deterministic.
				weights[i][j] = cardinalityEdgeBonus + int64(c.TotalScore)
			}
		}
	}
	assignment := hungarianMax(weights)
	out := []domain.Pairing{}
	for i, j := range assignment {
		if i >= len(left) || j < 0 || j >= len(right) {
			continue
		}
		c, ok := edges[pairKey(left[i].AccountID, right[j].AccountID)]
		if !ok {
			continue
		}
		out = append(out, domain.Pairing{ParticipantA: left[i].AccountID, ParticipantB: right[j].AccountID, ParticipantACode: left[i].ParticipantCode, ParticipantBCode: right[j].ParticipantCode, Score: c.TotalScore, SafeReasons: c.SafeReasons, Source: "ALGORITHM"})
	}
	sort.Slice(out, func(i, j int) bool {
		return pairKey(out[i].ParticipantA, out[i].ParticipantB) < pairKey(out[j].ParticipantA, out[j].ParticipantB)
	})
	return out
}

func hungarianMax(weight [][]int64) []int {
	n := len(weight)
	max := int64(0)
	for i := range weight {
		for j := range weight[i] {
			if weight[i][j] > max {
				max = weight[i][j]
			}
		}
	}
	u := make([]int64, n+1)
	v := make([]int64, n+1)
	p := make([]int, n+1)
	way := make([]int, n+1)
	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		minv := make([]int64, n+1)
		used := make([]bool, n+1)
		for j := 1; j <= n; j++ {
			minv[j] = math.MaxInt64
		}
		for {
			used[j0] = true
			i0 := p[j0]
			delta := int64(math.MaxInt64)
			j1 := 0
			for j := 1; j <= n; j++ {
				if used[j] {
					continue
				}
				cur := (max - weight[i0-1][j-1]) - u[i0] - v[j]
				if cur < minv[j] {
					minv[j] = cur
					way[j] = j0
				}
				if minv[j] < delta {
					delta = minv[j]
					j1 = j
				}
			}
			for j := 0; j <= n; j++ {
				if used[j] {
					u[p[j]] += delta
					v[j] -= delta
				} else {
					minv[j] -= delta
				}
			}
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for {
			j1 := way[j0]
			p[j0] = p[j1]
			j0 = j1
			if j0 == 0 {
				break
			}
		}
	}
	answer := make([]int, n)
	for i := range answer {
		answer[i] = -1
	}
	for j := 1; j <= n; j++ {
		if p[j] > 0 {
			answer[p[j]-1] = j - 1
		}
	}
	return answer
}
func unmatchedReason(id string, candidates []domain.Candidate, minimum int) string {
	eligible, above := false, false
	for _, c := range candidates {
		if c.ParticipantA != id && c.ParticipantB != id {
			continue
		}
		if c.Eligible {
			eligible = true
			if c.TotalScore >= minimum {
				above = true
			}
		}
	}
	if !eligible {
		return "NO_ELIGIBLE_CANDIDATE"
	}
	if !above {
		return "BELOW_MINIMUM_SCORE"
	}
	return "GROUP_CAPACITY_IMBALANCE"
}
