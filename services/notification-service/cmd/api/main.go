package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/config"
	"github.com/gehan-malshan/matchmate/notification-service/internal/httpapi"
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
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "notification-api", "environment", cfg.AppEnv)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	repository := postgres.New(pool)
	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           httpapi.New(repository),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Info("notification_api_started", "address", cfg.HTTPAddress)
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
