package smtp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
)

// Sender delivers UTF-8 plain-text email over SMTP with required STARTTLS.
// Credentials are injected through the process environment, never the database.
type Sender struct{ host, port, username, password, from string }

func New(host, port, username, password, from string) *Sender {
	return &Sender{host: host, port: port, username: username, password: password, from: from}
}

func (s *Sender) Send(ctx context.Context, delivery domain.Delivery, message domain.RenderedMessage, recipient string) (string, error) {
	if _, err := mail.ParseAddress(recipient); err != nil {
		return "", permanent("INVALID_DESTINATION", err)
	}
	from, err := mail.ParseAddress(s.from)
	if err != nil {
		return "", permanent("SMTP_FROM_INVALID", err)
	}
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(s.host, s.port))
	if err != nil {
		return "", retryable("PROVIDER_UNAVAILABLE", err)
	}
	client, err := smtp.NewClient(connection, s.host)
	if err != nil {
		connection.Close()
		return "", retryable("PROVIDER_UNAVAILABLE", err)
	}
	defer client.Quit()
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return "", permanent("SMTP_TLS_REQUIRED", errors.New("SMTP server does not support STARTTLS"))
	}
	if err = client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
		return "", retryable("PROVIDER_TLS_FAILED", err)
	}
	if s.username != "" {
		if err = client.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
			return "", classify("SMTP_AUTH_FAILED", err)
		}
	}
	if err = client.Mail(from.Address); err != nil {
		return "", classify("SMTP_SEND_FAILED", err)
	}
	if err = client.Rcpt(recipient); err != nil {
		return "", classify("INVALID_DESTINATION", err)
	}
	writer, err := client.Data()
	if err != nil {
		return "", classify("SMTP_SEND_FAILED", err)
	}
	body := "From: " + from.String() + "\r\nTo: " + recipient + "\r\nSubject: " + message.Subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + strings.ReplaceAll(message.Body, "\n", "\r\n")
	if _, err = writer.Write([]byte(body)); err != nil {
		_ = writer.Close()
		return "", retryable("SMTP_SEND_FAILED", err)
	}
	if err = writer.Close(); err != nil {
		return "", classify("SMTP_SEND_FAILED", err)
	}
	return "smtp:" + delivery.ID, nil
}

func permanent(code string, err error) error {
	return &domain.SendFailure{Kind: domain.FailurePermanent, Code: code, Err: err}
}
func retryable(code string, err error) error {
	return &domain.SendFailure{Kind: domain.FailureRetryable, Code: code, Err: err}
}
func classify(code string, err error) error {
	var smtpError *textproto.Error
	if errors.As(err, &smtpError) && smtpError.Code >= 500 {
		return permanent(code, fmt.Errorf("smtp rejected request: %w", err))
	}
	return retryable(code, err)
}
