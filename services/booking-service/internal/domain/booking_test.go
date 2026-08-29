package domain

import (
	"testing"
	"time"
)

func TestValidateEvent(t *testing.T) {
	now := time.Now()
	e := EventSnapshot{EventID: "e", Status: "REGISTRATION_OPEN", Price: "5000.00", Currency: "LKR", ConfiguredCapacity: 1, RegistrationClosesAt: now.Add(time.Hour)}
	if err := ValidateEvent(e, now); err != nil {
		t.Fatal(err)
	}
	e.Status = "CANCELLED"
	if ValidateEvent(e, now) == nil {
		t.Fatal("expected rejection")
	}
}

func TestNormalizeMoney(t *testing.T) {
	for in, want := range map[string]string{"0": "0.00", "4500": "4500.00", "12.3": "12.30", "12.34": "12.34"} {
		if got := NormalizeMoney(in); got != want {
			t.Fatalf("%s: got %s want %s", in, got, want)
		}
	}
}
