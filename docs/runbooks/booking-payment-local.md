# Booking and Payment Local Runbook

## Safe initial policy

Booking holds last 15 minutes by default and are configurable from 1 minute to 1 hour. There is no waitlist or automatic refund. A verified payment arriving after expiry enters `PAYMENT_REVIEW` and does not reclaim capacity.

## Start order

1. Start the root development stack with `docker compose up --build -d`; it starts Account, RabbitMQ, Event, Matchmaking, Booking, and Payment together. Do not also start any included Compose file separately, because that would create duplicate databases/APIs and host-port conflicts.
2. Confirm Account JWKS is reachable at `http://127.0.0.1:8081/.well-known/jwks.json`; Booking and Payment cache its ES256 public key.
3. The local Compose file defaults to non-functional sandbox placeholders so Booking can be tested without PayHere. Override `PAYHERE_MERCHANT_ID`, `PAYHERE_MERCHANT_SECRET`, and a public HTTPS `PAYHERE_NOTIFY_URL` in the same terminal for a real PayHere Sandbox test.
4. Validate `docker compose -f infrastructure/compose/booking-payment.compose.yml config`.
5. Verify the complete topology with `docker compose ps -a`. It includes Booking/Payment databases, migrations, APIs, expiry/reconciliation workers, both outbox relays, and the Booking Payment consumer.
6. Do not also run an included Compose file separately; duplicate projects will compete for ports `5433` through `5437` and `8081` through `8085`.

Create an Event in `REGISTRATION_OPEN`, then create a Booking with an `Idempotency-Key`. Initiate Payment using the returned booking ID. The browser return is never success evidence; poll Booking/Payment status after the verified callback.

Monitor held/confirmed counts, expired holds, inbox duplicates, outbox age, callback reviews, and payment-to-confirmation latency. Preserve both databases during rollback.
