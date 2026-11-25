package main

import (
	"context"
	"github.com/gehan-malshan/matchmate/payment-service/internal/config"
	"github.com/gehan-malshan/matchmate/payment-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, e := config.Load()
	if e != nil {
		panic(e)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, e := pgxpool.New(ctx, cfg.DatabaseURL)
	if e != nil {
		panic(e)
	}
	defer pool.Close()
	repo := postgres.New(pool)
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			n, err := repo.OpenPendingReconciliation(ctx, now.UTC().Add(-30*time.Minute), now.UTC())
			if err != nil {
				log.Error("payment_reconciliation_scan_failed", "error", err)
			} else if n > 0 {
				log.Warn("payment_reconciliation_items_opened", "count", n)
			}
		}
	}
}
