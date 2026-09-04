package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gehan-malshan/matchmate/notification-service/internal/application"
	"github.com/gehan-malshan/matchmate/notification-service/internal/config"
	"github.com/gehan-malshan/matchmate/notification-service/internal/domain"
	"github.com/gehan-malshan/matchmate/notification-service/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

var routingKeys = []string{
	"account.AccountRegistered",
	"account.AccountVerified",
	"account.ProfileApproved",
	"account.ProfileHidden",
	"account.AccountDeactivated",
	"booking.BookingPending",
	"booking.BookingConfirmed",
	"booking.BookingCancelled",
	"booking.HoldExpired",
	"booking.BookingPaymentReviewRequired",
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "notification-consumer", "environment", cfg.AppEnv)
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	connection, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		panic(err)
	}
	defer connection.Close()
	channel, err := connection.Channel()
	if err != nil {
		panic(err)
	}
	defer channel.Close()
	if err = declareTopology(channel, cfg); err != nil {
		panic(err)
	}
	if err = channel.Qos(20, 0, false); err != nil {
		panic(err)
	}
	messages, err := channel.Consume(cfg.QueueName, "notification-business-v1", false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	notificationChannel := "DEVELOPMENT"
	if cfg.Provider == "smtp" {
		notificationChannel = "EMAIL"
	}
	service := application.New(application.NewRouter(cfg.Locale, notificationChannel), postgres.New(pool), cfg.MaxAttempts)
	log.Info("notification_consumer_started", "queue", cfg.QueueName)
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-messages:
			if !ok {
				return
			}
			if len(message.Body) > 1<<20 {
				log.Warn("notification_event_rejected", "reason", "PAYLOAD_TOO_LARGE", "message_id", message.MessageId)
				_ = message.Nack(false, false)
				continue
			}
			var event domain.EventEnvelope
			if err = json.Unmarshal(message.Body, &event); err != nil {
				log.Warn("notification_event_rejected", "reason", "INVALID_JSON", "message_id", message.MessageId)
				_ = message.Nack(false, false)
				continue
			}
			created, handleErr := service.Handle(ctx, event)
			if errors.Is(handleErr, application.ErrInvalidEvent) {
				log.Warn("notification_event_rejected", "reason", "INVALID_EVENT", "event_id", event.EventID, "event_type", event.EventType)
				_ = message.Nack(false, false)
				continue
			}
			if handleErr != nil {
				log.Error("notification_event_processing_failed", "event_id", event.EventID, "event_type", event.EventType, "error", handleErr)
				select {
				case <-time.After(time.Second):
				case <-ctx.Done():
					return
				}
				_ = message.Nack(false, true)
				continue
			}
			_ = message.Ack(false)
			log.Info("notification_event_processed", "event_id", event.EventID, "event_type", event.EventType, "delivery_created", created)
		}
	}
}

func declareTopology(channel *amqp.Channel, cfg config.Config) error {
	if err := channel.ExchangeDeclare(cfg.EventExchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := channel.ExchangeDeclare(cfg.DeadLetterExch, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(cfg.DeadLetterQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := channel.QueueBind(cfg.DeadLetterQueue, "", cfg.DeadLetterExch, false, nil); err != nil {
		return err
	}
	args := amqp.Table{"x-dead-letter-exchange": cfg.DeadLetterExch}
	if _, err := channel.QueueDeclare(cfg.QueueName, true, false, false, false, args); err != nil {
		return err
	}
	for _, routingKey := range routingKeys {
		if err := channel.QueueBind(cfg.QueueName, routingKey, cfg.EventExchange, false, nil); err != nil {
			return err
		}
	}
	return nil
}
