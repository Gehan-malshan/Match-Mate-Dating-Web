package main

import (
	"context"
	"encoding/json"
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

type envelope struct {
	EventID    string    `json:"eventId"`
	EventType  string    `json:"eventType"`
	OccurredAt time.Time `json:"occurredAt"`
	Payload    struct {
		PaymentID string `json:"paymentId"`
		BookingID string `json:"bookingId"`
	} `json:"payload"`
}

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
	q, e := ch.QueueDeclare(cfg.PaymentQueue, true, false, false, false, nil)
	if e != nil {
		panic(e)
	}
	for _, routingKey := range []string{"payment.PaymentCompleted", "payment.PaymentReviewRequired"} {
		if e = ch.QueueBind(q.Name, routingKey, cfg.EventExchange, false, nil); e != nil {
			panic(e)
		}
	}
	messages, e := ch.Consume(q.Name, "booking-payment-consumer", false, false, false, false, nil)
	if e != nil {
		panic(e)
	}
	repo := postgres.New(pool)
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			var event envelope
			if json.Unmarshal(msg.Body, &event) != nil || event.EventID == "" || event.Payload.BookingID == "" {
				_ = msg.Nack(false, false)
				continue
			}
			if err := repo.ApplyPaymentEvent(ctx, event.EventID, event.EventType, event.Payload.PaymentID, event.Payload.BookingID, event.OccurredAt); err != nil {
				log.Error("booking_payment_event_failed", "event_id", event.EventID, "error", err)
				_ = msg.Nack(false, true)
				continue
			}
			_ = msg.Ack(false)
		}
	}
}
