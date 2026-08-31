package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Environment, HTTPAddress, DatabaseURL string
	JWTIssuer, JWTAudience                string
	JWTPublicKeyPEM, AccountJWKSURL       string
	AllowedOrigins                        []string
}

func Load() (Config, error) {
	c := Config{Environment: value("APP_ENV", "development"), HTTPAddress: value("HTTP_ADDRESS", ":8083"), DatabaseURL: os.Getenv("DATABASE_URL"), JWTIssuer: value("JWT_ISSUER", "matchmate-account"), JWTAudience: value("JWT_AUDIENCE", "matchmate-api"), JWTPublicKeyPEM: os.Getenv("JWT_PUBLIC_KEY_PEM"), AccountJWKSURL: value("ACCOUNT_JWKS_URL", "http://localhost:8081/.well-known/jwks.json"), AllowedOrigins: split(value("ALLOWED_ORIGINS", "http://127.0.0.1:5173,http://localhost:5173"))}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is required")
	}
	return c, nil
}
func value(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
func split(v string) []string {
	out := []string{}
	for _, item := range strings.Split(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
