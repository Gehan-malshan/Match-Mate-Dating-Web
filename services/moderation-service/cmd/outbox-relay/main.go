package main

import (
	"context"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/config"
	"github.com/gehan-malshan/matchmate/moderation-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"log/slog"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "moderation-outbox-relay")
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	channel, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer channel.Close()
	if err = channel.ExchangeDeclare(cfg.EventExchange, "topic", true, false, false, false, nil); err != nil {
		panic(err)
	}
	if err = channel.Confirm(false); err != nil {
		panic(err)
	}
	confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	repo := postgres.New(pool)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			records, e := repo.ClaimOutbox(ctx, 50)
			if e != nil {
				log.Error("outbox_claim_failed", "error", e)
				continue
			}
			for _, record := range records {
				e = channel.PublishWithContext(ctx, cfg.EventExchange, record.RoutingKey, false, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", MessageId: record.ID, Timestamp: time.Now().UTC(), Body: record.Body})
				if e != nil {
					log.Error("outbox_publish_failed", "event_id", record.ID, "error", e)
					break
				}
				select {
				case confirmation := <-confirms:
					if !confirmation.Ack {
						continue
					}
				case <-time.After(5 * time.Second):
					continue
				case <-ctx.Done():
					return
				}
				if e = repo.MarkOutboxPublished(ctx, record.ID, time.Now().UTC()); e != nil {
					log.Error("outbox_mark_failed", "event_id", record.ID, "error", e)
				}
			}
		}
	}
}
