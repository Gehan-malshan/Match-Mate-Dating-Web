// notification-provider-preflight validates SMTP transport and the configured
// sender envelope without sending an email. Run it in the target deployment
// environment after the sender domain/address is verified in the provider.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/config"
	"github.com/gehan-malshan/matchmate/notification-service/internal/provider/smtp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	if cfg.Provider != "smtp" {
		fmt.Fprintln(os.Stderr, "NOTIFICATION_PROVIDER=smtp is required for provider preflight")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = smtp.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom).Preflight(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "SMTP sender preflight failed")
		os.Exit(1)
	}
	fmt.Println("SMTP sender preflight succeeded")
}
