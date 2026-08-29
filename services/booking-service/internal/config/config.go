package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddress, DatabaseURL, EventAPIURL, JWTPublicKeyPEM, AccountJWKSURL, JWTIssuer, JWTAudience, RabbitMQURL, EventExchange, PaymentQueue string
	HoldDuration                                                                                                                             time.Duration
	AllowedOrigins                                                                                                                           []string
}

func value(k, f string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return f
}
func Load() (Config, error) {
	duration, err := time.ParseDuration(value("BOOKING_HOLD_DURATION", "15m"))
	if err != nil || duration < time.Minute || duration > time.Hour {
		return Config{}, errors.New("BOOKING_HOLD_DURATION must be between 1m and 1h")
	}
	c := Config{HTTPAddress: value("BOOKING_HTTP_ADDRESS", ":8085"), DatabaseURL: os.Getenv("BOOKING_DATABASE_URL"), EventAPIURL: value("BOOKING_EVENT_API_URL", "http://127.0.0.1:8082/api/v1/events/{eventId}"), JWTPublicKeyPEM: os.Getenv("BOOKING_JWT_PUBLIC_KEY_PEM"), AccountJWKSURL: value("ACCOUNT_JWKS_URL", "http://127.0.0.1:8081/.well-known/jwks.json"), JWTIssuer: value("JWT_ISSUER", "matchmate-account"), JWTAudience: value("JWT_AUDIENCE", "matchmate-api"), RabbitMQURL: value("BOOKING_RABBITMQ_URL", "amqp://matchmate:matchmate@localhost:5672/"), EventExchange: value("BOOKING_EVENT_EXCHANGE", "matchmate.events"), PaymentQueue: value("BOOKING_PAYMENT_QUEUE", "booking.payment.v1"), HoldDuration: duration, AllowedOrigins: strings.Split(value("ALLOWED_ORIGINS", "http://127.0.0.1:5173,http://localhost:5173"), ",")}
	if c.DatabaseURL == "" {
		return c, errors.New("BOOKING_DATABASE_URL is required")
	}
	return c, nil
}
