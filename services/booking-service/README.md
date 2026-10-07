# Booking Service

Owns ticket holds, bookings, event allocation, capacity consumption, hold expiry, cancellation, attendance, and check-in status.

## Implementation status

The executable member slice provides authenticated hold creation/read/list, idempotent cancellation of unpaid holds, a constrained Payment snapshot, Event-owned price/currency lookup, atomic capacity allocation, configurable expiry, Payment event inbox deduplication, confirmation/late-review transitions, transactional outbox facts, and a publisher-confirmed RabbitMQ relay. The safe initial policy is a configurable 15-minute hold, no waitlist, no automatic refund, and manual review for late payment. Confirmed bookings cannot be cancelled until refund policy is approved. Attendance, account/moderation consumers, and approved refund coordination remain later slices.

The local API uses port `8085` because the upstream Matchmaking prototype owns port `8083`.

Administrators can list an event's booking records with `GET /api/v1/admin/events/{eventId}/registrations?limit=25&offset=0`. The endpoint enforces the admin role from the signed token, returns account IDs only to authorized callers, and includes `hasMore` for pagination. Migration 4 indexes the event/date ordering. The GraphQL gateway's `eventRegistrations` query joins these booking records to Account-owned nickname/email via an admin-only batch lookup. This is a booking list, not an attendance or payment-collection ledger: `CONFIRMED` + `AT_VENUE` means the seat is reserved, not paid.

Migration 3 adds the immutable `paymentMethod` booking snapshot. A member can select `AT_VENUE` only when Event permits it. That reservation atomically increments confirmed capacity and publishes `BookingConfirmed` without starting PayHere. Its confirmed status means the seat is reserved, **not that payment was collected**. `ONLINE` retains the expiring hold and verified-Payment confirmation flow. Existing bookings default to `ONLINE`. Confirmed venue cancellations, staff collection, and check-in are not implemented pending policy; do not present those as available actions. `BOOKING_TEST_DATABASE_URL` enables the disposable-schema PostgreSQL last-seat/idempotency component test.

It must prevent overselling through transactional allocation controls and idempotent commands.

## Responsibilities

- Validate event registration/policy and member eligibility for booking.
- Create one time-limited hold using atomic capacity allocation.
- Store immutable event price, currency, and relevant policy version.
- Expire/release holds idempotently.
- Confirm valid bookings after a trusted payment-completed fact.
- Handle cancellation/refund-request coordination according to approved policy.
- Record attendance/check-in/no-show with organizer scope and audit.
- Provide authoritative allocation/usage checks to Event and matching eligibility to Matchmaking.

## Does not own

Event configuration, PayHere/provider state, account credentials/profiles, matchmaking scores, or notification delivery.

## Proposed API

```text
POST   /api/v1/bookings
GET    /api/v1/bookings/{bookingId}
GET    /api/v1/bookings?mine=true
POST   /api/v1/bookings/{bookingId}/cancel
POST   /api/v1/bookings/{bookingId}/check-in       organizer/admin
GET    /internal/v1/events/{eventId}/usage         constrained internal
GET    /internal/v1/bookings/{bookingId}/payment-snapshot constrained Payment integration
```

Create/cancel/check-in require idempotency keys. Member booking access uses authenticated subject ownership.

## Proposed data

`booking`, `seat_hold`, `capacity_allocation`, `attendance`, `inbox`, and `outbox`.

Key invariants:

- At most one active booking per account/event.
- Held plus confirmed allocation cannot exceed approved capacity.
- Last-seat decisions are atomic; never count then insert without locking/constraint.
- Price/currency/policy snapshots cannot change after hold creation.
- Only an allowed pending hold can become confirmed.
- Expiry, cancellation, and payment event redelivery are idempotent.

## State

```text
PENDING_PAYMENT -> CONFIRMED
PENDING_PAYMENT -> EXPIRED | CANCELLED
CONFIRMED -> CANCELLED under policy
Late/mismatched payment -> PAYMENT_REVIEW
```

## Events

Produces `BookingPending`, `BookingConfirmed`, `BookingCancelled`, `HoldExpired`, and attendance facts.

Consumes Event policy/lifecycle updates, `PaymentCompleted`/payment review facts, and account/moderation restrictions as approved. Consumer updates and inbox insertion occur atomically.

## Failure behavior

- Payment completes before expiry: confirm exactly once.
- Payment completes after expiry: do not reallocate; enter review and trigger reconciliation/refund policy.
- Broker outage: state/outbox remain committed for later relay.
- Event capacity becomes restrictive: use policy version and explicit conflict/re-evaluation behavior.
- Concurrent final-seat requests: one succeeds, others receive stable capacity conflict.

## Required tests

- High-contention last-seat scenario.
- Idempotency-key replay and conflict.
- Duplicate booking and account/event eligibility.
- Hold expiry/release worker crash/retry.
- Payment event duplicate/reorder/late behavior.
- Cancellation authorization, window, allocation release, refund coordination.
- Capacity policy version race.
- Attendance organizer scope and audit.
- Migration, outbox/inbox, metrics, and sensitive-log behavior.

## Completion criteria

Concurrency tests prove no overselling; every hold reaches a controlled terminal/review state; confirmed booking is authoritative for event participation; duplicate commands/events are safe; operations can observe and recover failures.

Update this README, architecture/data/testing docs, contracts, runbooks, and change history whenever behavior changes.

