package main

import (
	"context"
	"errors"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/application"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/auth"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/config"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/httpapi"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
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
	verifier, err := auth.New(cfg.JWTPublicKeyPEM, cfg.AccountJWKSURL, cfg.JWTIssuer, cfg.JWTAudience)
	if err != nil {
		panic(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "moderation-api", "environment", cfg.Environment)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	server := &http.Server{Addr: cfg.HTTPAddress, Handler: httpapi.New(application.New(postgres.New(pool), cfg.ReportRateLimit), verifier, cfg, log), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		if e := server.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			log.Error("moderation_api_stopped", "error", e)
			stop()
		}
	}()
	log.Info("moderation_api_started", "address", cfg.HTTPAddress)
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
