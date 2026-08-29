# Booking and Payment Local Runbook

## Safe initial policy

Booking holds last 15 minutes by default and are configurable from 1 minute to 1 hour. There is no waitlist or automatic refund. A verified payment arriving after expiry enters `PAYMENT_REVIEW` and does not reclaim capacity.

## Start order

1. Start Account/RabbitMQ using `account-profile.compose.yml` and Event using `event.compose.yml`.
2. Confirm Account JWKS is reachable at `http://127.0.0.1:8081/.well-known/jwks.json`; Booking and Payment cache its ES256 public key.
3. Export sandbox `PAYHERE_MERCHANT_ID`, `PAYHERE_MERCHANT_SECRET`, and a public HTTPS `PAYHERE_NOTIFY_URL`.
4. Validate `docker compose -f infrastructure/compose/booking-payment.compose.yml config`.
5. Start APIs with `docker compose -f infrastructure/compose/booking-payment.compose.yml up -d --build`.
6. Start workers by adding `--profile messaging`.

Create an Event in `REGISTRATION_OPEN`, then create a Booking with an `Idempotency-Key`. Initiate Payment using the returned booking ID. The browser return is never success evidence; poll Booking/Payment status after the verified callback.

Monitor held/confirmed counts, expired holds, inbox duplicates, outbox age, callback reviews, and payment-to-confirmation latency. Preserve both databases during rollback.
