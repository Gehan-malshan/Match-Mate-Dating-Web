package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/auth"
	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/config"
	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/httpapi"
	"github.com/gehan-malshan/matchmate/matchmaking-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	server := &http.Server{Addr: cfg.HTTPAddress, Handler: httpapi.New(postgres.New(pool), verifier, cfg, logger), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("matchmaking_api_started", "address", cfg.HTTPAddress)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("matchmaking_api_stopped", "error", serveErr)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
