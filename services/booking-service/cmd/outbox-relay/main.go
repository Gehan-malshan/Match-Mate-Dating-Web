package main

import (
	"context"
	"github.com/gehan-malshan/matchmate/booking-service/internal/config"
	"github.com/gehan-malshan/matchmate/booking-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
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
	conn, e := amqp.Dial(cfg.RabbitMQURL)
	if e != nil {
		panic(e)
	}
	defer conn.Close()
	ch, e := conn.Channel()
	if e != nil {
		panic(e)
	}
	defer ch.Close()
	if e = ch.ExchangeDeclare(cfg.EventExchange, "topic", true, false, false, false, nil); e != nil {
		panic(e)
	}
	if e = ch.Confirm(false); e != nil {
		panic(e)
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	repo := postgres.New(pool)
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			records, err := repo.ClaimOutbox(ctx, 50)
			if err != nil {
				log.Error("booking_outbox_claim_failed", "error", err)
				continue
			}
			for _, record := range records {
				err = ch.PublishWithContext(ctx, cfg.EventExchange, record.RoutingKey, false, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", MessageId: record.ID, Timestamp: time.Now().UTC(), Body: record.Body})
				if err != nil {
					break
				}
				select {
				case confirm := <-confirms:
					if !confirm.Ack {
						continue
					}
				case <-time.After(5 * time.Second):
					continue
				case <-ctx.Done():
					return
				}
				_ = repo.MarkOutboxPublished(ctx, record.ID, time.Now().UTC())
			}
		}
	}
}
