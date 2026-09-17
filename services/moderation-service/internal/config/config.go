package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment, HTTPAddress, DatabaseURL, RabbitMQURL, EventExchange string
	JWTIssuer, JWTAudience, JWTPublicKeyPEM, AccountJWKSURL           string
	AllowedOrigins                                                    []string
	ReportRateLimit                                                   int
	ExpiryInterval                                                    time.Duration
}

func value(key, fallback string) string {
	if current := strings.TrimSpace(os.Getenv(key)); current != "" {
		return current
	}
	return fallback
}
func Load() (Config, error) {
	rate, err := strconv.Atoi(value("MODERATION_REPORT_RATE_LIMIT_PER_HOUR", "5"))
	if err != nil || rate < 1 {
		return Config{}, errors.New("MODERATION_REPORT_RATE_LIMIT_PER_HOUR must be positive")
	}
	interval, err := time.ParseDuration(value("MODERATION_EXPIRY_INTERVAL", "30s"))
	if err != nil || interval <= 0 {
		return Config{}, errors.New("MODERATION_EXPIRY_INTERVAL must be positive")
	}
	cfg := Config{Environment: value("APP_ENV", "development"), HTTPAddress: value("HTTP_ADDRESS", ":8087"), DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")), RabbitMQURL: value("RABBITMQ_URL", "amqp://matchmate:matchmate@localhost:5672/"), EventExchange: value("EVENT_EXCHANGE", "matchmate.events"), JWTIssuer: value("JWT_ISSUER", "matchmate-account"), JWTAudience: value("JWT_AUDIENCE", "matchmate-api"), JWTPublicKeyPEM: strings.TrimSpace(os.Getenv("JWT_PUBLIC_KEY_PEM")), AccountJWKSURL: value("ACCOUNT_JWKS_URL", "http://localhost:8081/.well-known/jwks.json"), AllowedOrigins: split(value("ALLOWED_ORIGINS", "http://127.0.0.1:5173,http://localhost:5173")), ReportRateLimit: rate, ExpiryInterval: interval}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}
func split(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
