# Booking Service

Owns ticket holds, bookings, event allocation, capacity consumption, hold expiry, cancellation, attendance, and check-in status.

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

