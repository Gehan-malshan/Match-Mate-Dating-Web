package config

import (
	"os"
	"strings"

	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/upstream"
)

type Config struct {
	Address        string
	AllowedOrigins map[string]bool
	Services       upstream.Services
}

func Load() Config {
	return Config{
		Address:        value("GRAPHQL_HTTP_ADDRESS", ":8080"),
		AllowedOrigins: origins(value("GRAPHQL_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		Services: upstream.Services{
			Account:      value("ACCOUNT_API_URL", "http://account-api:8081/api/v1"),
			Event:        value("EVENT_API_URL", "http://event-api:8082/api/v1"),
			Matchmaking:  value("MATCHMAKING_API_URL", "http://matchmaking-api:8083/api/v1"),
			Payment:      value("PAYMENT_API_URL", "http://payment-api:8084/api/v1"),
			Booking:      value("BOOKING_API_URL", "http://booking-api:8085/api/v1"),
			Notification: value("NOTIFICATION_API_URL", "http://notification-api:8086/api/v1"),
		},
	}
}

func value(key, fallback string) string {
	if result := strings.TrimSpace(os.Getenv(key)); result != "" {
		return result
	}
	return fallback
}

func origins(raw string) map[string]bool {
	result := map[string]bool{}
	for _, item := range strings.Split(raw, ",") {
		if origin := strings.TrimSpace(item); origin != "" {
			result[origin] = true
		}
	}
	return result
}
