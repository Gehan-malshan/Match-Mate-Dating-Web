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
