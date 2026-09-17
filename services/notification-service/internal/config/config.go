package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv               string
	HTTPAddress          string
	DatabaseURL          string
	RabbitMQURL          string
	EventExchange        string
	QueueName            string
	DeadLetterExch       string
	DeadLetterQueue      string
	Provider             string
	SMTPHost             string
	SMTPPort             string
	SMTPUsername         string
	SMTPPassword         string
	SMTPFrom             string
	AccountContactURL    string
	InternalServiceToken string
	Locale               string
	MaxAttempts          int
	PollInterval         time.Duration
	LeaseDuration        time.Duration
	RetryBase            time.Duration
	JWTPublicKeyPEM      string
	AccountJWKSURL       string
	JWTIssuer            string
	JWTAudience          string
	AllowedOrigins       []string
}

func value(key, fallback string) string {
	if current := strings.TrimSpace(os.Getenv(key)); current != "" {
		return current
	}
	return fallback
}

func positiveInt(key string, fallback int) (int, error) {
	raw := value(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 {
		return 0, errors.New(key + " must be a positive integer")
	}
	return parsed, nil
}

func positiveDuration(key, fallback string) (time.Duration, error) {
	parsed, err := time.ParseDuration(value(key, fallback))
	if err != nil || parsed <= 0 {
		return 0, errors.New(key + " must be a positive duration")
	}
	return parsed, nil
}

func Load() (Config, error) {
	maxAttempts, err := positiveInt("NOTIFICATION_MAX_ATTEMPTS", 5)
	if err != nil {
		return Config{}, err
	}
	poll, err := positiveDuration("NOTIFICATION_POLL_INTERVAL", "1s")
	if err != nil {
		return Config{}, err
	}
	lease, err := positiveDuration("NOTIFICATION_LEASE_DURATION", "30s")
	if err != nil {
		return Config{}, err
	}
	retryBase, err := positiveDuration("NOTIFICATION_RETRY_BASE", "1m")
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:               strings.ToLower(value("APP_ENV", "development")),
		HTTPAddress:          value("NOTIFICATION_HTTP_ADDRESS", ":8086"),
		DatabaseURL:          strings.TrimSpace(os.Getenv("NOTIFICATION_DATABASE_URL")),
		RabbitMQURL:          value("NOTIFICATION_RABBITMQ_URL", "amqp://matchmate:matchmate@localhost:5672/"),
		EventExchange:        value("NOTIFICATION_EVENT_EXCHANGE", "matchmate.events"),
		QueueName:            value("NOTIFICATION_QUEUE", "notification.business.v1"),
		DeadLetterExch:       value("NOTIFICATION_DEAD_LETTER_EXCHANGE", "matchmate.notification.dlx"),
		DeadLetterQueue:      value("NOTIFICATION_DEAD_LETTER_QUEUE", "notification.business.v1.dlq"),
		Provider:             strings.ToLower(value("NOTIFICATION_PROVIDER", "dev-sink")),
		SMTPHost:             strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:             value("SMTP_PORT", "587"),
		SMTPUsername:         strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:         os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:             strings.TrimSpace(os.Getenv("SMTP_FROM")),
		AccountContactURL:    strings.TrimSpace(os.Getenv("NOTIFICATION_ACCOUNT_CONTACT_URL")),
		InternalServiceToken: os.Getenv("INTERNAL_SERVICE_TOKEN"),
		Locale:               value("NOTIFICATION_DEFAULT_LOCALE", "en-LK"),
		MaxAttempts:          maxAttempts,
		PollInterval:         poll,
		LeaseDuration:        lease,
		RetryBase:            retryBase,
		JWTPublicKeyPEM:      strings.TrimSpace(os.Getenv("NOTIFICATION_JWT_PUBLIC_KEY_PEM")),
		AccountJWKSURL:       value("ACCOUNT_JWKS_URL", "http://127.0.0.1:8081/.well-known/jwks.json"),
		JWTIssuer:            value("JWT_ISSUER", "matchmate-account"),
		JWTAudience:          value("JWT_AUDIENCE", "matchmate-api"),
		AllowedOrigins:       strings.Split(value("ALLOWED_ORIGINS", "http://127.0.0.1:5173,http://localhost:5173"), ","),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("NOTIFICATION_DATABASE_URL is required")
	}
	if cfg.Provider == "dev-sink" {
		if cfg.AppEnv != "development" && cfg.AppEnv != "test" {
			return cfg, errors.New("dev-sink notification provider is allowed only in development or test")
		}
		return cfg, nil
	}
	if cfg.Provider != "smtp" {
		return cfg, errors.New("NOTIFICATION_PROVIDER must be dev-sink or smtp")
	}
	if cfg.SMTPHost == "" || cfg.SMTPUsername == "" || cfg.SMTPPassword == "" || cfg.SMTPFrom == "" {
		return cfg, errors.New("SMTP_HOST, SMTP_USERNAME, SMTP_PASSWORD, and SMTP_FROM are required for smtp")
	}
	if cfg.AccountContactURL == "" || cfg.InternalServiceToken == "" {
		return cfg, errors.New("NOTIFICATION_ACCOUNT_CONTACT_URL and INTERNAL_SERVICE_TOKEN are required for smtp")
	}
	if cfg.AppEnv == "production" && cfg.Provider != "smtp" {
		return cfg, errors.New("smtp notification provider is required in production")
	}
	return cfg, nil
}
