package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/application"
	"github.com/gehan-malshan/matchmate/notification-service/internal/auth"
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
	verifier, err := auth.New(cfg.JWTPublicKeyPEM, cfg.AccountJWKSURL, cfg.JWTIssuer, cfg.JWTAudience)
	if err != nil {
		panic(err)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "notification-api", "environment", cfg.AppEnv)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	repository := postgres.New(pool)
	feed := application.NewFeedService(repository)
	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           cors(httpapi.New(repository, feed, verifier), cfg.AllowedOrigins),
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

func cors(next http.Handler, allowed []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, candidate := range allowed {
			if strings.TrimSpace(candidate) == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
				break
			}
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PATCH, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
