package main

import (
	"context"
	"github.com/gehan-malshan/matchmate/event-service/internal/config"
	"github.com/gehan-malshan/matchmate/event-service/internal/store/postgres"
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
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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
	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()
	if err = ch.ExchangeDeclare(cfg.EventExchange, "topic", true, false, false, false, nil); err != nil {
		panic(err)
	}
	if err = ch.Confirm(false); err != nil {
		panic(err)
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
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
				e = ch.PublishWithContext(ctx, cfg.EventExchange, record.RoutingKey, false, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", MessageId: record.ID, Timestamp: time.Now().UTC(), Body: record.Body})
				if e != nil {
					log.Error("outbox_publish_failed", "event_id", record.ID, "error", e)
					break
				}
				select {
				case confirmation := <-confirms:
					if !confirmation.Ack {
						log.Error("outbox_publish_not_acknowledged", "event_id", record.ID)
						continue
					}
				case <-time.After(5 * time.Second):
					log.Error("outbox_publish_confirm_timeout", "event_id", record.ID)
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
