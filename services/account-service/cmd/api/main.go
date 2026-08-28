package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gehan-malshan/matchmate/account-service/internal/application"
	"github.com/gehan-malshan/matchmate/account-service/internal/auth"
	"github.com/gehan-malshan/matchmate/account-service/internal/config"
	"github.com/gehan-malshan/matchmate/account-service/internal/httpapi"
	"github.com/gehan-malshan/matchmate/account-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	tokens, err := auth.NewManager(cfg.JWTPrivateKeyPEM, cfg.JWTKeyID, cfg.Issuer, cfg.Audience, cfg.AccessTTL)
	if err != nil {
		panic(err)
	}
	repo := postgres.New(pool)
	app := application.New(repo, tokens, cfg.AccessTTL, cfg.RefreshTTL, cfg.VerificationTTL, cfg.MinimumAge, cfg.CurrentConsentVersion, cfg.DevExposeVerificationToken)
	server := &http.Server{Addr: cfg.HTTPAddress, Handler: httpapi.New(app, cfg, log), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Info("account_api_started", "address", cfg.HTTPAddress)
		if e := server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Error("account_api_stopped", "error", e)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
