package domain

import (
	"regexp"
	"strings"
	"time"
)

var money = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,2})?$`)
var currency = regexp.MustCompile(`^[A-Z]{3}$`)

func Validate(in CreateInput) map[string]string {
	f := map[string]string{}
	if strings.TrimSpace(in.OrganizerID) == "" {
		f["organizerId"] = "required"
	}
	if n := len(strings.TrimSpace(in.Name)); n < 3 || n > 120 {
		f["name"] = "must contain 3 to 120 characters"
	}
	if len(strings.TrimSpace(in.Description)) > 2000 {
		f["description"] = "must not exceed 2000 characters"
	}
	if strings.TrimSpace(in.BroadLocation) == "" || len(in.BroadLocation) > 120 {
		f["broadLocation"] = "required and at most 120 characters"
	}
	if _, err := time.LoadLocation(in.TimeZone); err != nil {
		f["timeZone"] = "must be an IANA time zone"
	}
	if in.StartsAt.IsZero() || !in.EndsAt.After(in.StartsAt) {
		f["endsAt"] = "must be after startsAt"
	}
	if in.RegistrationOpensAt.IsZero() || !in.RegistrationClosesAt.After(in.RegistrationOpensAt) {
		f["registrationClosesAt"] = "must be after registrationOpensAt"
	}
	if !in.RegistrationClosesAt.Before(in.StartsAt) {
		f["registrationClosesAt"] = "must be before startsAt"
	}
	if !money.MatchString(in.Price) {
		f["price"] = "must be a non-negative decimal with at most two fraction digits"
	}
	if !currency.MatchString(in.Currency) {
		f["currency"] = "must be an uppercase ISO 4217 code"
	}
	if in.ConfiguredCapacity < 1 || in.ConfiguredCapacity > 10000 {
		f["configuredCapacity"] = "must be between 1 and 10000"
	}
	if strings.TrimSpace(in.MatchingRulesetVersion) == "" {
		f["matchingRulesetVersion"] = "required"
	}
	return f
}
func CanTransition(from, to Status) bool {
	allowed := map[Status][]Status{Draft: {Published, Cancelled}, Published: {RegistrationOpen, Cancelled}, RegistrationOpen: {RegistrationClosed, Cancelled}, RegistrationClosed: {Cancelled}}
	for _, v := range allowed[from] {
		if v == to {
			return true
		}
	}
	return false
}
