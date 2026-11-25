package main

import (
	"context"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/config"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "moderation-expiry-worker")
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	repo := postgres.New(pool)
	ticker := time.NewTicker(cfg.ExpiryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, e := repo.ExpireActions(ctx, time.Now().UTC(), 100)
			if e != nil {
				log.Error("moderation_expiry_failed", "error", e)
			} else if count > 0 {
				log.Info("moderation_actions_expired", "count", count)
			}
		}
	}
}
