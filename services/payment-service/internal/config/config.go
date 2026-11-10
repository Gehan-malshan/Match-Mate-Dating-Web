package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/gehan-malshan/matchmate/payment-service/internal/payhere"
)

type Config struct {
	HTTPAddress, DatabaseURL, BookingSnapshotURL, JWTPublicKeyPEM, AccountJWKSURL, JWTIssuer, JWTAudience, ReturnURL, CancelURL, NotifyURL string
	RabbitMQURL, EventExchange                                                                                                             string
	PayHere                                                                                                                                payhere.Config
	AllowedOrigins                                                                                                                         []string
}

func value(k, f string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return f
}
func Load() (Config, error) {
	env := value("PAYHERE_ENVIRONMENT", "sandbox")
	checkout := payhere.SandboxCheckoutURL
	if env == "live" {
		checkout = payhere.LiveCheckoutURL
	} else if env != "sandbox" {
		return Config{}, errors.New("PAYHERE_ENVIRONMENT must be sandbox or live")
	}
	c := Config{HTTPAddress: value("PAYMENT_HTTP_ADDRESS", ":8084"), DatabaseURL: os.Getenv("PAYMENT_DATABASE_URL"), BookingSnapshotURL: os.Getenv("PAYMENT_BOOKING_SNAPSHOT_URL"), JWTPublicKeyPEM: os.Getenv("PAYMENT_JWT_PUBLIC_KEY_PEM"), AccountJWKSURL: value("ACCOUNT_JWKS_URL", "http://127.0.0.1:8081/.well-known/jwks.json"), JWTIssuer: value("JWT_ISSUER", "matchmate-account"), JWTAudience: value("JWT_AUDIENCE", "matchmate-api"), ReturnURL: os.Getenv("PAYHERE_RETURN_URL"), CancelURL: os.Getenv("PAYHERE_CANCEL_URL"), NotifyURL: os.Getenv("PAYHERE_NOTIFY_URL"), RabbitMQURL: value("PAYMENT_RABBITMQ_URL", "amqp://matchmate:matchmate@localhost:5672/"), EventExchange: value("PAYMENT_EVENT_EXCHANGE", "matchmate.events"), PayHere: payhere.Config{MerchantID: os.Getenv("PAYHERE_MERCHANT_ID"), MerchantSecret: os.Getenv("PAYHERE_MERCHANT_SECRET"), CheckoutURL: checkout}, AllowedOrigins: strings.Split(value("ALLOWED_ORIGINS", "http://127.0.0.1:5173,http://localhost:5173"), ",")}
	if c.DatabaseURL == "" || c.BookingSnapshotURL == "" || c.PayHere.MerchantID == "" || c.PayHere.MerchantSecret == "" || c.ReturnURL == "" || c.CancelURL == "" || c.NotifyURL == "" {
		return c, errors.New("payment database, booking snapshot, PayHere credentials, return, cancel, and notify configuration are required")
	}
	if env == "live" {
		for _, raw := range []string{c.ReturnURL, c.CancelURL, c.NotifyURL} {
			u, err := url.Parse(raw)
			if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
				return c, errors.New("live PayHere URLs must be public HTTPS URLs without user info or fragments")
			}
			host := strings.ToLower(u.Hostname())
			ip := net.ParseIP(host)
			if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".test") || strings.HasSuffix(host, ".example") || host == "example.com" || ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified()) {
				return c, errors.New("live PayHere URLs require a public approved domain")
			}
		}
	}
	return c, nil
}
