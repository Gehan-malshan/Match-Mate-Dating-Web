package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/config"
	"github.com/gehan-malshan/matchmate/notification-service/internal/delivery"
	"github.com/gehan-malshan/matchmate/notification-service/internal/provider"
	"github.com/gehan-malshan/matchmate/notification-service/internal/provider/devsink"
	mailprovider "github.com/gehan-malshan/matchmate/notification-service/internal/provider/smtp"
	"github.com/gehan-malshan/matchmate/notification-service/internal/recipient"
	"github.com/gehan-malshan/matchmate/notification-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "notification-worker", "environment", cfg.AppEnv)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	var sender provider.Sender = devsink.New(log)
	var resolver delivery.RecipientResolver
	if cfg.Provider == "smtp" {
		sender = mailprovider.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
		resolver = recipient.NewAccount(cfg.AccountContactURL, cfg.InternalServiceToken)
	}
	worker := delivery.NewWorker(postgres.New(pool), sender, resolver, log, cfg.LeaseDuration, cfg.RetryBase)
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	log.Info("notification_worker_started", "provider", cfg.Provider)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for processed := true; processed; {
				processed, err = worker.RunOnce(ctx)
				if err != nil {
					log.Error("notification_delivery_worker_failed", "error", err)
					break
				}
			}
		}
	}
}
