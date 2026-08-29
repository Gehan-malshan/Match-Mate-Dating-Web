# Payment Service

Owns PayHere payment initiation, provider order references, trusted price verification, callback validation, payment state, reconciliation, refunds, and payment audit records.

Amounts and currency must come from an immutable server-side booking price snapshot, never from client input.

## Implementation status

The first executable slice includes a Go API, ES256 access-token validation, constrained Booking snapshot client, PayHere sandbox/live checkout adapter, initiation idempotency, callback verification and replay fingerprints, member-owned status reads, migration 1, audit records, and transactional outbox facts. The Booking service now supplies the owner-authorized immutable snapshot and consumes completion/review facts.

The Payment outbox relay publishes persistent events with publisher confirms and crash-safe claims. A scheduled worker opens restricted reconciliation items for payments still pending after 30 minutes; provider retrieval and automatic resolution remain disabled. Still required before release: real PostgreSQL/RabbitMQ tests, gateway callback rate limits, approved refund/provider-resolution policy, and PayHere sandbox/live evidence. See `docs/runbooks/payment-payhere.md`.

Local verification:

```powershell
go mod tidy
go test ./...
go vet ./...
$env:PAYMENT_DATABASE_URL = "postgres://..."
go run ./cmd/migrate
go run ./cmd/api
go run ./cmd/outbox-relay
go run ./cmd/reconciliation-worker
```

## Responsibilities

- Initiate payment for an authorized eligible booking.
- Retrieve/verify immutable booking amount and currency through the approved internal contract.
- Generate unique order ID and PayHere request/hash in a provider adapter.
- Validate callback signature, merchant, order, exact amount/currency, expected booking, state transition, and replay.
- Persist callback fingerprint/provider reference before state transition.
- Publish payment facts through outbox.
- Reconcile pending, late, mismatched, failed, and refunded transactions.
- Manage refund state/provider references according to approved finance policy.
- Provide restricted support/finance views and tamper-evident audit.

## Does not own

Ticket capacity, booking eligibility rules, event price definition, account profile, matching, or notification delivery.

## Proposed API

```text
POST /api/v1/payments/initiate
POST /api/v1/payments/payhere/callback
GET  /api/v1/bookings/{bookingId}/payment
POST /api/v1/payments/{paymentId}/refund          finance/admin
GET  /api/v1/payments/reconciliation             restricted support/finance
```

Initiation accepts booking ID, not authoritative amount/currency. Initiation/refund require idempotency keys. Callback is provider-authenticated through exact verification, not member JWT.

## Proposed data

`payment`, `provider_callback`, `refund`, `reconciliation_item`, `payment_audit`, `inbox`, and `outbox`.

Key invariants:

- Order/provider IDs and callback fingerprints are unique.
- Stored amount/currency exactly match Booking snapshot and verified callback.
- One callback can cause at most one state transition and completion event.
- Failed/mismatched verification never changes payment to completed.
- Refund total cannot exceed captured amount.
- Provider credentials and card data are never stored/logged.

## State

```text
PENDING -> COMPLETED | FAILED
COMPLETED -> REFUND_PENDING -> REFUNDED
Verification/booking mismatch -> REVIEW
```

## Events

Produces `PaymentInitiated`, `PaymentCompleted`, `PaymentFailed`, `PaymentReviewRequired`, `RefundRequested`, and `PaymentRefunded` as approved.

Consumes booking cancellation/expiry facts when needed for review/refund coordination. Avoid circular synchronous dependencies; document any internal booking eligibility query.

## PayHere callback checklist

1. Apply TLS/WAF/request-size/rate controls.
2. Parse without logging raw sensitive payload.
3. Verify required fields and signature exactly as provider specifies.
4. Compare merchant ID, order ID, fixed-precision amount, currency, provider state, and booking relation.
5. Insert unique callback receipt/fingerprint.
6. Apply allowed idempotent transition and append audit/outbox in one transaction.
7. Acknowledge valid duplicate safely.
8. Reject/quarantine mismatch and alert/reconcile.

## Required tests

- Valid callback and every individual mismatch.
- Invalid/malformed signature and timing-safe comparison where applicable.
- Duplicate/reordered callbacks and concurrent delivery.
- Client amount tampering attempt.
- Provider timeout/unknown result and polling/reconciliation.
- Hold expiry before late success.
- Broker outage, outbox retry, Booking consumer outage.
- Refund limits, duplicate refund, provider failure.
- Decimal precision, audit immutability, authorization, PII/log sanitation.
- PayHere sandbox end-to-end verification before release.

## Completion criteria

No client can influence authoritative charge values; callbacks are replay-safe and fully verified; successful payment produces at most one completion; unknown/late/mismatch states are visible and recoverable; finance audit/reconciliation is complete.

Update this README, architecture/data/testing docs, OpenAPI/AsyncAPI, payment runbook, and change history whenever behavior changes.
