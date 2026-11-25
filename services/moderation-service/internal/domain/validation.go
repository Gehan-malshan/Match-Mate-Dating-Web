package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var sha256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func ValidateReport(in CreateReportInput) map[string]string {
	errors := map[string]string{}
	if _, err := uuid.Parse(in.TargetID); err != nil {
		errors["targetId"] = "must be a UUID"
	}
	validTarget := map[TargetType]bool{TargetAccount: true, TargetProfile: true, TargetEvent: true, TargetBooking: true, TargetPairing: true}
	if !validTarget[in.TargetType] {
		errors["targetType"] = "is unsupported"
	}
	validCategory := map[ReportCategory]bool{CategoryHarassment: true, CategorySafety: true, CategoryImpersonation: true, CategoryContactSharing: true, CategoryInappropriateContent: true, CategoryFraud: true, CategoryOther: true}
	if !validCategory[in.Category] {
		errors["category"] = "is unsupported"
	}
	description := strings.TrimSpace(in.Description)
	if len(description) < 10 || len(description) > 2000 {
		errors["description"] = "must contain 10 to 2000 characters"
	}
	if in.EventID != "" {
		if _, err := uuid.Parse(in.EventID); err != nil {
			errors["eventId"] = "must be a UUID"
		}
	}
	if len(in.Evidence) > 5 {
		errors["evidence"] = "may contain at most 5 references"
	}
	for _, e := range in.Evidence {
		if len(strings.TrimSpace(e.Reference)) < 3 || len(e.Reference) > 500 || len(e.MediaType) > 100 || !sha256Pattern.MatchString(e.SHA256) {
			errors["evidence"] = "contains invalid reference metadata"
			break
		}
	}
	return errors
}
func ValidAction(class ActionClass, scope, reason string, effective time.Time, expires *time.Time) map[string]string {
	errors := map[string]string{}
	valid := map[ActionClass]bool{ActionContentHide: true, ActionProfileRestriction: true, ActionEventExclusion: true, ActionAccountSuspension: true, ActionMatchmakingExclusion: true, ActionPairingInvalidation: true, ActionRevealPrevention: true}
	if !valid[class] {
		errors["actionClass"] = "is unsupported"
	}
	if len(strings.TrimSpace(scope)) < 1 || len(scope) > 200 {
		errors["scope"] = "is required and limited to 200 characters"
	}
	if len(strings.TrimSpace(reason)) < 10 || len(reason) > 1000 {
		errors["reason"] = "must contain 10 to 1000 characters"
	}
	if effective.IsZero() {
		errors["effectiveAt"] = "is required"
	}
	if expires != nil && !expires.After(effective) {
		errors["expiresAt"] = "must be after effectiveAt"
	}
	return errors
}

func ValidTarget(value TargetType) bool {
	return map[TargetType]bool{TargetAccount: true, TargetProfile: true, TargetEvent: true, TargetBooking: true, TargetPairing: true}[value]
}
