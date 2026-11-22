package config

import "testing"

func TestLoadDevelopmentDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("NOTIFICATION_DATABASE_URL", "postgres://example.test/notification")
	t.Setenv("NOTIFICATION_PROVIDER", "dev-sink")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddress != ":8086" || cfg.MaxAttempts != 5 || cfg.Locale != "en-LK" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsDevelopmentSinkInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("NOTIFICATION_DATABASE_URL", "postgres://example.test/notification")
	t.Setenv("NOTIFICATION_PROVIDER", "dev-sink")
	if _, err := Load(); err == nil {
		t.Fatal("production must not start with development sink")
	}
}

func TestLoadRequiresVerifiedSenderForProductionSMTP(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("NOTIFICATION_DATABASE_URL", "postgres://example.test/notification")
	t.Setenv("NOTIFICATION_PROVIDER", "smtp")
	t.Setenv("SMTP_HOST", "smtp.example.test")
	t.Setenv("SMTP_USERNAME", "user")
	t.Setenv("SMTP_PASSWORD", "password")
	t.Setenv("SMTP_FROM", "MatchMate <updates@example.test>")
	t.Setenv("NOTIFICATION_ACCOUNT_CONTACT_URL", "https://account.example.test/api/v1/internal/notification-recipients")
	t.Setenv("INTERNAL_SERVICE_TOKEN", "internal")
	t.Setenv("SECURITY_MAIL_TOKEN", "security")
	t.Setenv("SECURITY_PUBLIC_URL", "https://app.example.test")
	t.Setenv("MODERATION_API_URL", "https://moderation.example.test")
	t.Setenv("MODERATION_SERVICE_TOKEN", "safety")
	if _, err := Load(); err == nil {
		t.Fatal("production SMTP must require verified sender attestation")
	}
	t.Setenv("SMTP_SENDER_VERIFIED", "true")
	if _, err := Load(); err != nil {
		t.Fatalf("valid production SMTP config rejected: %v", err)
	}
}

func TestLoadRejectsUnsupportedPreferenceCategories(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("NOTIFICATION_DATABASE_URL", "postgres://example.test/notification")
	t.Setenv("NOTIFICATION_PROVIDER", "dev-sink")
	t.Setenv("NOTIFICATION_PREFERENCE_CATEGORIES", "MARKETING")
	if _, err := Load(); err == nil {
		t.Fatal("unsupported preference categories must be rejected")
	}
}
