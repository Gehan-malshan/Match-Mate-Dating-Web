package domain

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

func validInput() CreateInput {
	start := time.Date(2026, 12, 20, 10, 0, 0, 0, time.UTC)
	return CreateInput{OrganizerID: "organizer-1", Name: "Colombo Winter Social", BroadLocation: "Colombo", TimeZone: "Asia/Colombo", StartsAt: start, EndsAt: start.Add(3 * time.Hour), RegistrationOpensAt: start.Add(-30 * 24 * time.Hour), RegistrationClosesAt: start.Add(-24 * time.Hour), Price: "4500.00", Currency: "LKR", ConfiguredCapacity: 80, MatchingRulesetVersion: "rules-v1"}
}
func TestValidateAcceptsValidEvent(t *testing.T) {
	if f := Validate(validInput()); len(f) > 0 {
		t.Fatalf("unexpected fields: %v", f)
	}
}
func TestValidateRejectsDatesMoneyAndCapacity(t *testing.T) {
	in := validInput()
	in.EndsAt = in.StartsAt
	in.RegistrationClosesAt = in.StartsAt
	in.Price = "45.999"
	in.Currency = "lkr"
	in.ConfiguredCapacity = 0
	f := Validate(in)
	for _, k := range []string{"endsAt", "registrationClosesAt", "price", "currency", "configuredCapacity"} {
		if _, ok := f[k]; !ok {
			t.Errorf("expected %s error", k)
		}
	}
}
func TestLifecycleTransitions(t *testing.T) {
	valid := [][2]Status{{Draft, Published}, {Published, RegistrationOpen}, {RegistrationOpen, RegistrationClosed}, {Draft, Cancelled}}
	for _, v := range valid {
		if !CanTransition(v[0], v[1]) {
			t.Errorf("expected %s -> %s", v[0], v[1])
		}
	}
	if CanTransition(Draft, RegistrationOpen) {
		t.Fatal("draft must not open registration directly")
	}
	if CanTransition(Cancelled, Published) {
		t.Fatal("cancelled event must be terminal")
	}
}
func TestPublicEventOmitsOperationalFields(t *testing.T) {
	raw, _ := json.Marshal(Event{OrganizerID: "private-organizer", VenueName: "exact venue"}.Public())
	if regexp.MustCompile(`organizer|venue`).Match(raw) {
		t.Fatalf("public event leaked operational fields: %s", raw)
	}
}
