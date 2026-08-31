package domain

import (
	"strings"
	"testing"
	"time"
)

func TestRenderValidatesAllowListAndSubject(t *testing.T) {
	tpl := Template{SubjectTemplate: "Booking {{.state}}", BodyTemplate: "Current status: {{.state}}", AllowedVariables: []string{"state"}}
	message, err := Render(tpl, map[string]string{"state": "confirmed"})
	if err != nil || message.Subject != "Booking confirmed" {
		t.Fatalf("unexpected render: %+v, %v", message, err)
	}
	if _, err = Render(tpl, map[string]string{"state": "confirmed", "email": "private@example.test"}); err == nil {
		t.Fatal("unexpected variable should be rejected")
	}
	tpl.SubjectTemplate = "Unsafe\nsubject"
	if _, err = Render(tpl, map[string]string{"state": "confirmed"}); err == nil {
		t.Fatal("newline in subject should be rejected")
	}
}

func TestRetryDelayIsBounded(t *testing.T) {
	base := time.Minute
	if got := RetryDelay(base, 1); got != time.Minute {
		t.Fatalf("first retry = %s", got)
	}
	if got := RetryDelay(base, 20); got != 64*time.Minute {
		t.Fatalf("bounded retry = %s", got)
	}
}

func TestSendFailureDoesNotExposeWrappedFormatting(t *testing.T) {
	failure := &SendFailure{Kind: FailurePermanent, Code: "INVALID_DESTINATION"}
	if !strings.Contains(failure.Error(), "INVALID_DESTINATION") {
		t.Fatal("stable provider code should be available")
	}
}
