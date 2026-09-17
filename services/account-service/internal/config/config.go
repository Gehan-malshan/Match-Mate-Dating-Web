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
	Issuer, Audience, JWTPrivateKeyPEM, JWTKeyID                      string
	GoogleClientID, GoogleClientSecret, GoogleRedirectURL             string
	GoogleSuccessRedirectURL, InternalServiceToken                    string
	CurrentConsentVersion                                             string
	AllowedOrigins                                                    []string
	AccessTTL, RefreshTTL, VerificationTTL                            time.Duration
	MinimumAge                                                        int
	CookieSecure, DevExposeVerificationToken                          bool
}

func Load() (Config, error) {
	c := Config{
		Environment: env("APP_ENV", "development"), HTTPAddress: env("HTTP_ADDRESS", ":8081"),
		DatabaseURL: os.Getenv("DATABASE_URL"), RabbitMQURL: env("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:5672/"), EventExchange: env("EVENT_EXCHANGE", "matchmate.events"),
		Issuer: env("JWT_ISSUER", "matchmate-account"), Audience: env("JWT_AUDIENCE", "matchmate-api"), JWTPrivateKeyPEM: os.Getenv("JWT_PRIVATE_KEY_PEM"), JWTKeyID: env("JWT_KEY_ID", "account-dev-1"),
		GoogleClientID: os.Getenv("GOOGLE_OAUTH_CLIENT_ID"), GoogleClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"), GoogleRedirectURL: os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
		GoogleSuccessRedirectURL: env("GOOGLE_OAUTH_SUCCESS_REDIRECT_URL", "http://localhost:5173/login?google=success"), InternalServiceToken: os.Getenv("INTERNAL_SERVICE_TOKEN"),
		CurrentConsentVersion: env("CURRENT_CONSENT_VERSION", "privacy-2026-08"),
		AllowedOrigins:        split(env("ALLOWED_ORIGINS", "http://127.0.0.1:5173,http://localhost:5173")),
		AccessTTL:             duration("ACCESS_TOKEN_TTL", 10*time.Minute), RefreshTTL: duration("REFRESH_TOKEN_TTL", 30*24*time.Hour), VerificationTTL: duration("VERIFICATION_TOKEN_TTL", 24*time.Hour),
		MinimumAge: integer("MINIMUM_AGE", 18), CookieSecure: boolean("COOKIE_SECURE", false), DevExposeVerificationToken: boolean("DEV_EXPOSE_VERIFICATION_TOKEN", false),
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if c.Environment == "production" && c.JWTPrivateKeyPEM == "" {
		return Config{}, errors.New("JWT_PRIVATE_KEY_PEM is required in production")
	}
	if c.Environment == "production" && !c.CookieSecure {
		return Config{}, errors.New("COOKIE_SECURE must be true in production")
	}
	googleValues := []string{c.GoogleClientID, c.GoogleClientSecret, c.GoogleRedirectURL}
	configuredGoogle := 0
	for _, value := range googleValues {
		if strings.TrimSpace(value) != "" {
			configuredGoogle++
		}
	}
	if configuredGoogle != 0 && configuredGoogle != len(googleValues) {
		return Config{}, errors.New("GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET, and GOOGLE_OAUTH_REDIRECT_URL must be configured together")
	}
	if c.Environment == "production" && c.InternalServiceToken == "" {
		return Config{}, errors.New("INTERNAL_SERVICE_TOKEN is required in production")
	}
	return c, nil
}

func env(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
func split(v string) []string {
	var out []string
	for _, x := range strings.Split(v, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}
func duration(k string, d time.Duration) time.Duration {
	v, err := time.ParseDuration(env(k, d.String()))
	if err != nil {
		return d
	}
	return v
}
func integer(k string, d int) int {
	v, err := strconv.Atoi(env(k, strconv.Itoa(d)))
	if err != nil {
		return d
	}
	return v
}
func boolean(k string, d bool) bool {
	v, err := strconv.ParseBool(env(k, strconv.FormatBool(d)))
	if err != nil {
		return d
	}
	return v
}
