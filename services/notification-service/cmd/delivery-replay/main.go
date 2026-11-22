// notification-delivery-replay is an owner-service recovery command.
// It never publishes a raw RabbitMQ payload or creates a new notification:
// it reopens exactly one terminal delivery using the original delivery ID.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/config"
	"github.com/gehan-malshan/matchmate/notification-service/internal/store/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	deliveryID := flag.String("delivery-id", "", "terminal notification delivery UUID")
	actor := flag.String("actor", "", "operator identity for the replay audit record")
	reason := flag.String("reason", "", "why the root cause is fixed (8-500 characters)")
	additionalAttempts := flag.Int("additional-attempts", 0, "new attempts allowed after replay (1-20; default configured maximum)")
	confirm := flag.String("confirm", "", "must equal REPLAY")
	flag.Parse()

	if *confirm != "REPLAY" || strings.TrimSpace(*actor) == "" || len(strings.TrimSpace(*reason)) < 8 || len(strings.TrimSpace(*reason)) > 500 {
		fmt.Fprintln(os.Stderr, "usage: notification-delivery-replay -delivery-id <uuid> -actor <operator> -reason <8-500 chars> -confirm REPLAY [-additional-attempts 5]")
		os.Exit(2)
	}
	if _, err := uuid.Parse(*deliveryID); err != nil {
		fmt.Fprintln(os.Stderr, "delivery-id must be a UUID")
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	if *additionalAttempts == 0 {
		*additionalAttempts = cfg.MaxAttempts
	}
	if *additionalAttempts < 1 || *additionalAttempts > 20 {
		fmt.Fprintln(os.Stderr, "additional-attempts must be between 1 and 20")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	replayed, err := postgres.New(pool).Replay(ctx, *deliveryID, strings.TrimSpace(*actor), strings.TrimSpace(*reason), *additionalAttempts, time.Now().UTC())
	if err != nil {
		panic(err)
	}
	if !replayed {
		fmt.Fprintln(os.Stderr, "delivery was not terminal, was suppressed, or was not found; no replay was performed")
		os.Exit(1)
	}
	fmt.Println("notification delivery replay scheduled")
}
