# Booking and Payment Local Runbook

## Venue collection and check-in

From the repository root, apply Booking migration 6 with `docker compose up --build -d booking-migrate booking-api graphql-api` (or start the full stack). Sign in as an administrator in the single web app, open **Registrations**, and refresh the table before admission. For an eligible confirmed `AT_VENUE` booking, record **cash** or **card at venue** only after money is actually received. Booking uses its stored amount/currency and records the administrator and UTC time. Then select **Check in**. An online confirmed booking can be checked in without a venue collection record. Confirmed venue bookings cannot be cancelled; pending online holds can still expire or be cancelled. Collection is a staff declaration and must be reconciled with cash or the external card terminal separately.

To run the capacity and recovery component test against a disposable migrated PostgreSQL database, set `BOOKING_TEST_DATABASE_URL` and run `go -C services/booking-service test -count=1 ./internal/store/postgres`. The test creates and removes its own schema. It exercises last-seat contention, expiry, duplicate Payment events, collection retries, check-in retries, and no venue cancellation. Keep the database available during the run. Do not drop migration 6 columns after recording real collections.

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
