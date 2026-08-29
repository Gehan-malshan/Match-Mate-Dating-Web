# Payment and PayHere Runbook

## Current scope

The first Payment slice supports authenticated checkout initiation, verified form-encoded PayHere callbacks, member-owned status reads, audit records, transactional outbox facts, a publisher-confirmed RabbitMQ relay, Booking snapshot/confirmation integration, and pending-payment reconciliation classification after 30 minutes. Provider retrieval/resolution, refunds, gateway rate limiting, live dependency evidence, and PayHere approval remain deployment blockers.

## Sandbox setup

1. Create a separate PayHere sandbox merchant account.
2. Configure an ignored local environment from `services/payment-service/.env.example`.
3. Expose the callback through an approved HTTPS development tunnel or deployed development host. PayHere cannot notify localhost.
4. Set `PAYHERE_ENVIRONMENT=sandbox`; never put credentials in Git.
5. Run migration `go run ./cmd/migrate`, then `go run ./cmd/api` from the service directory.
6. Create a valid Booking `PENDING_PAYMENT` hold and call initiation with an `Idempotency-Key`.
7. POST the returned `fields` to the returned `actionUrl` using a browser form.
8. After redirect, query `GET /api/v1/bookings/{bookingId}/payment`; never trust the redirect as proof of success.

## Signals and incident handling

Alert on callback verification mismatch, `REVIEW` payments, old `PENDING` payments, callback error rate, database failures, outbox age, and payment-completed-to-booking-confirmed latency. Never log raw callback bodies, secrets, checkout contact fields, or card metadata.

A wrong amount, currency, merchant, order, or signature must remain non-completed and enter review. A late success must not recreate capacity; Booking owns the review/refund decision. Preserve the database and callback/audit evidence during rollback.

## Moving to live PayHere

Sandbox is a separate deployment and cannot be converted. Apply for a new live PayHere Merchant Account after the production website and sale/service information are complete. In the live portal, add the exact production domain under Domains and Credentials, wait for approval, and obtain the domain-specific live Merchant Secret.

Before switching:

- Complete business/bank approval and confirm permitted MatchMate event sales and refund terms.
- Publish HTTPS return, cancel, and public callback URLs on the approved domain.
- Store live Merchant ID/Secret in the production secret manager, separate from sandbox.
- Set `PAYHERE_ENVIRONMENT=live`; the service then selects `https://www.payhere.lk/pay/checkout`.
- Verify firewall/WAF, callback request size/rate controls, monitoring, reconciliation, finance access, backups, and rollback.
- Run a low-value real transaction with finance approval and verify callback signature, exact money, outbox, Booking confirmation, settlement/retrieval, cancellation, and approved refund behavior.
- Keep sandbox available for regression testing. Never copy sandbox orders, secrets, or callbacks into production.

Automated retrieval/refund APIs require separate PayHere Business App credentials and live server-IP whitelisting where PayHere requires it. Do not enable them until the refund/reconciliation policy is approved and tested.
