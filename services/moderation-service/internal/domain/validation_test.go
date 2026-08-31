package domain

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestReportValidation(t *testing.T) {
	valid := CreateReportInput{TargetType: TargetAccount, TargetID: uuid.NewString(), Category: CategorySafety, Description: "A concrete safety concern."}
	if got := ValidateReport(valid); len(got) != 0 {
		t.Fatalf("valid report rejected: %v", got)
	}
	valid.Description = "short"
	if ValidateReport(valid)["description"] == "" {
		t.Fatal("short description accepted")
	}
}
func TestReportValidationRejectsUnsafeMetadata(t *testing.T) {
	valid := CreateReportInput{TargetType: TargetAccount, TargetID: uuid.NewString(), Category: CategorySafety, Description: "A concrete safety concern."}
	tests := map[string]func(*CreateReportInput){
		"target type": func(in *CreateReportInput) { in.TargetType = "UNKNOWN" },
		"target id":   func(in *CreateReportInput) { in.TargetID = "not-a-uuid" },
		"category":    func(in *CreateReportInput) { in.Category = "UNKNOWN" },
		"event id":    func(in *CreateReportInput) { in.EventID = "not-a-uuid" },
		"evidence count": func(in *CreateReportInput) {
			in.Evidence = make([]EvidenceInput, 6)
		},
		"evidence hash": func(in *CreateReportInput) {
			in.Evidence = []EvidenceInput{{Reference: "object/ref", MediaType: "image/png", SHA256: "bad"}}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			in := valid
			mutate(&in)
			if len(ValidateReport(in)) == 0 {
				t.Fatal("invalid report accepted")
			}
		})
	}
}
func TestActionValidation(t *testing.T) {
	now := time.Now()
	before := now.Add(-time.Hour)
	if ValidAction(ActionRevealPrevention, "ALL", "Documented safety reason", now, &before)["expiresAt"] == "" {
		t.Fatal("invalid expiry accepted")
	}
}
func TestActionValidationRejectsMissingFields(t *testing.T) {
	if got := ValidAction("UNKNOWN", "", "short", time.Time{}, nil); len(got) != 4 {
		t.Fatalf("field errors=%v", got)
	}
}
