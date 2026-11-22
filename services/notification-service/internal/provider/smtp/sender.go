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
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
)

// Sender delivers UTF-8 plain-text email over SMTP with required STARTTLS.
// Credentials are injected through the process environment, never the database.
type Sender struct{ host, port, username, password, from string }

func New(host, port, username, password, from string) *Sender {
	return &Sender{host: host, port: port, username: username, password: password, from: from}
}

// Preflight proves that the configured SMTP endpoint accepts TLS, credentials,
// and the configured sender envelope. It deliberately does not add a recipient
// or message body, so it cannot send mail. Provider-dashboard domain ownership
// verification remains an operator responsibility before setting the production
// SMTP_SENDER_VERIFIED deployment gate.
func (s *Sender) Preflight(ctx context.Context) error {
	from, client, closeClient, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer closeClient()
	if err = client.Mail(from.Address); err != nil {
		return classify("SMTP_SENDER_REJECTED", err)
	}
	if err = client.Reset(); err != nil {
		return retryable("SMTP_SEND_FAILED", err)
	}
	return nil
}

func (s *Sender) Send(ctx context.Context, delivery domain.Delivery, message domain.RenderedMessage, recipient string) (string, error) {
	if _, err := mail.ParseAddress(recipient); err != nil {
		return "", permanent("INVALID_DESTINATION", err)
	}
	from, client, closeClient, err := s.connect(ctx)
	if err != nil {
		return "", err
	}
	defer closeClient()
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

func (s *Sender) connect(ctx context.Context) (*mail.Address, *smtp.Client, func(), error) {
	from, err := mail.ParseAddress(s.from)
	if err != nil {
		return nil, nil, func() {}, permanent("SMTP_FROM_INVALID", err)
	}
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(s.host, s.port))
	if err != nil {
		return nil, nil, func() {}, retryable("PROVIDER_UNAVAILABLE", err)
	}
	deadline := time.Now().Add(8 * time.Second)
	if requestDeadline, ok := ctx.Deadline(); ok && requestDeadline.Before(deadline) {
		deadline = requestDeadline
	}
	_ = connection.SetDeadline(deadline)
	client, err := smtp.NewClient(connection, s.host)
	if err != nil {
		connection.Close()
		return nil, nil, func() {}, retryable("PROVIDER_UNAVAILABLE", err)
	}
	closeClient := func() { _ = client.Quit(); _ = connection.Close() }
	if ok, _ := client.Extension("STARTTLS"); !ok {
		closeClient()
		return nil, nil, func() {}, permanent("SMTP_TLS_REQUIRED", errors.New("SMTP server does not support STARTTLS"))
	}
	if err = client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
		closeClient()
		return nil, nil, func() {}, retryable("PROVIDER_TLS_FAILED", err)
	}
	if s.username != "" {
		if err = client.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
			closeClient()
			return nil, nil, func() {}, classify("SMTP_AUTH_FAILED", err)
		}
	}
	return from, client, closeClient, nil
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
