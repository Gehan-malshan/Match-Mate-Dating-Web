# Payment and PayHere Runbook

## Current scope

The Payment slice supports authenticated checkout initiation, verified form-encoded PayHere callbacks, member-owned status reads, audit records, transactional outbox facts, a publisher-confirmed RabbitMQ relay, Booking snapshot/confirmation integration, and pending-payment reconciliation classification after 30 minutes. Invalid callbacks return 400 without changing payment state. Verified mismatches/contradictory results enter `REVIEW`. Administrators can read a cursor-paged manual review queue at `GET /api/v1/payments/reconciliation`. The current product policy is **no automatic refunds**; finance must review disputed/late/duplicate/charged-back outcomes and follow an approved exception process. Provider retrieval/resolution, gateway rate limiting, live dependency evidence, and PayHere approval remain deployment blockers.

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

An invalid merchant or signature is rejected before state change. A verified wrong amount/currency or unknown order remains non-completed and enters review. A late success must not recreate capacity; Booking owns the seat and review decision. Preserve the database and callback/audit evidence during rollback.

For an incident, first check `GET /health/ready`, then inspect the administrator review queue. A `PENDING_PROVIDER_RESULT` means a payment has remained pending beyond the worker threshold; `CALLBACK_MISMATCH` or `CONFLICTING_PROVIDER_RESULT` requires finance investigation against PayHere records. `REVIEW_UNKNOWN_ORDER` means a verified callback did not match a local order. Do not change a booking or payment directly in SQL. The queue is intentionally read-only because provider retrieval, resolution authorization, and exception policy are not approved. Escalate unresolved items to the payment owner and preserve audit/callback fingerprints. Never treat a browser return URL as payment proof.

Before enabling online payment, run `PAYMENT_TEST_DATABASE_URL` component tests against an isolated PostgreSQL database, RabbitMQ outage/redelivery checks, and PayHere sandbox callback tests through a public HTTPS URL. Docker Desktop must be running for the local Compose stack; the real provider cannot notify localhost.

## Moving to live PayHere

Sandbox is a separate deployment and cannot be converted. Apply for a new live PayHere Merchant Account after the production website and sale/service information are complete. In the live portal, add the exact production domain under Domains and Credentials, wait for approval, and obtain the domain-specific live Merchant Secret.

Before switching:

- Complete business/bank approval and confirm permitted MatchMate event sales and refund terms.
- Publish HTTPS return, cancel, and public callback URLs on the approved domain.
- Store live Merchant ID/Secret in the production secret manager, separate from sandbox.
- Set `PAYHERE_ENVIRONMENT=live`; the service then selects `https://www.payhere.lk/pay/checkout`.
- Verify firewall/WAF, callback request size/rate controls, monitoring, reconciliation, finance access, backups, and rollback.
- Run a low-value real transaction with finance approval and verify callback signature, exact money, outbox, Booking confirmation, settlement/retrieval, cancellation, and the published no-refunds terms.
- Keep sandbox available for regression testing. Never copy sandbox orders, secrets, or callbacks into production.

Automated retrieval requires separate PayHere Business App credentials and live server-IP whitelisting where PayHere requires it. Do not enable it until provider reconciliation is approved and tested. No refund API is planned under the current policy; legally required or provider-driven exceptions require a separate reviewed process.
