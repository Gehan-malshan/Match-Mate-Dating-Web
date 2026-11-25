package config

import "testing"

func TestLivePayHereRejectsLocalReturnAndCallbackURLs(t *testing.T) {
	for key, value := range map[string]string{
		"PAYHERE_ENVIRONMENT":          "live",
		"PAYMENT_DATABASE_URL":         "postgres://example.test/payment",
		"PAYMENT_BOOKING_SNAPSHOT_URL": "https://private.example.test/bookings/{bookingId}",
		"PAYHERE_MERCHANT_ID":          "merchant",
		"PAYHERE_MERCHANT_SECRET":      "secret",
		"PAYHERE_RETURN_URL":           "https://matchmate.lk/bookings",
		"PAYHERE_CANCEL_URL":           "https://matchmate.lk/bookings",
		"PAYHERE_NOTIFY_URL":           "http://localhost:8084/api/v1/payments/payhere/callback",
	} {
		t.Setenv(key, value)
	}
	if _, err := Load(); err == nil {
		t.Fatal("live mode accepted a localhost callback")
	}
	t.Setenv("PAYHERE_NOTIFY_URL", "https://payments.matchmate.lk/api/v1/payments/payhere/callback")
	if _, err := Load(); err != nil {
		t.Fatalf("valid live URL configuration rejected: %v", err)
	}
	t.Setenv("PAYHERE_NOTIFY_URL", "https://example.test/api/v1/payments/payhere/callback")
	if _, err := Load(); err == nil {
		t.Fatal("live mode accepted a placeholder callback domain")
	}
}
