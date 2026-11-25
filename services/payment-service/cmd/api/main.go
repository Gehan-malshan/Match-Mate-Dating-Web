package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gehan-malshan/matchmate/payment-service/internal/application"
	"github.com/gehan-malshan/matchmate/payment-service/internal/auth"
	"github.com/gehan-malshan/matchmate/payment-service/internal/booking"
	"github.com/gehan-malshan/matchmate/payment-service/internal/config"
	"github.com/gehan-malshan/matchmate/payment-service/internal/httpapi"
	"github.com/gehan-malshan/matchmate/payment-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, e := config.Load()
	if e != nil {
		panic(e)
	}
	v, e := auth.New(cfg.JWTPublicKeyPEM, cfg.AccountJWKSURL, cfg.JWTIssuer, cfg.JWTAudience)
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
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	svc := application.New(postgres.New(pool), booking.New(cfg.BookingSnapshotURL), cfg.PayHere).WithCheckoutURLs(cfg.ReturnURL, cfg.CancelURL, cfg.NotifyURL)
	server := &http.Server{Addr: cfg.HTTPAddress, Handler: cors(httpapi.New(svc, v, log), cfg.AllowedOrigins), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Info("payment_api_started", "address", cfg.HTTPAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("payment_api_stopped", "error", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
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
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
