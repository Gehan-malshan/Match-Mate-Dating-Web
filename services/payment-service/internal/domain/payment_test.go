package domain

import (
	"testing"
	"time"
)

func TestValidateSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)
	valid := BookingSnapshot{BookingID: "b1", AccountID: "a1", Amount: "5000.00", Currency: "LKR", Status: "PENDING_PAYMENT", ExpiresAt: now.Add(time.Minute)}
	if err := ValidateSnapshot(valid, "a1", now); err != nil {
		t.Fatal(err)
	}
	valid.Amount = "5000"
	if err := ValidateSnapshot(valid, "a1", now); err == nil {
		t.Fatal("expected invalid money")
	}
}

func TestCallbackState(t *testing.T) {
	if CallbackState("2") != Completed || CallbackState("0") != Pending || CallbackState("-2") != Failed {
		t.Fatal("unexpected mapping")
	}
}
