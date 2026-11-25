package main

import (
	"context"
	"github.com/gehan-malshan/matchmate/booking-service/internal/application"
	"github.com/gehan-malshan/matchmate/booking-service/internal/config"
	eventclient "github.com/gehan-malshan/matchmate/booking-service/internal/event"
	"github.com/gehan-malshan/matchmate/booking-service/internal/store/postgres"
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
	app := application.New(postgres.New(pool), eventclient.New(cfg.EventAPIURL), cfg.HoldDuration)
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := app.Expire(ctx, 100)
			if err != nil {
				log.Error("booking_expiry_failed", "error", err)
			} else if n > 0 {
				log.Info("booking_holds_expired", "count", n)
			}
		}
	}
}
