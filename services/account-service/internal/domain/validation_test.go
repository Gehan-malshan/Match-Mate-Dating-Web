package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAdultBoundary(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	if err := ValidateAdult("2008-08-28", 18, now); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAdult("2008-08-29", 18, now); err == nil {
		t.Fatal("minor accepted")
	}
}
func TestProfileRejectsContactDetails(t *testing.T) {
	if err := ValidateProfileText("Gehan", "Colombo", "message me on instagram: @example"); err == nil {
		t.Fatal("contact information accepted")
	}
}
func TestCommunityProjectionCannotSerializePrivateFields(t *testing.T) {
	body, _ := json.Marshal(CommunityProfile{ProfileID: "p1", Nickname: "Member", AgeBand: "25-29"})
	text := string(body)
	for _, restricted := range []string{"email", "dateOfBirth", "dealBreakers", "interestedIn"} {
		if strings.Contains(text, restricted) {
			t.Fatalf("community projection leaked %s", restricted)
		}
	}
}
