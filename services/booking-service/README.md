# Booking Service

Owns ticket holds, bookings, event allocation, capacity consumption, hold expiry, cancellation, attendance, and check-in status.

## Implementation status

The executable member slice provides authenticated hold creation/read/list, idempotent cancellation of unpaid online holds, a constrained Payment snapshot, Event-owned price/currency lookup, atomic capacity allocation, configurable expiry, Payment event inbox deduplication, confirmation/late-review transitions, transactional outbox facts, and a publisher-confirmed RabbitMQ relay. The safe initial policy is a configurable 15-minute hold, no waitlist, no refunds, and manual review for late payment. Confirmed venue bookings cannot be cancelled.

Migration 6 adds administrator-only venue collection (`CASH` or `CARD_AT_VENUE`) and check-in. Booking records the immutable price snapshot, actor, method, and UTC time; this is a staff declaration, not PayHere settlement. Venue check-in requires collection first. Both commands require `Idempotency-Key`, use a booking row lock, and emit outbox facts. Moderation eligibility and Event cancellation are rechecked before the action. See [ADR-0006](../../docs/adr/0006-venue-collection-and-checkin.md).

The local API uses port `8085` because the upstream Matchmaking prototype owns port `8083`.

Booking now checks current Moderation state before creating reservations and exposing a pending online booking to Payment. A suspended or event-excluded member is denied; a Moderation outage fails closed. Administrative registration rows include `entryEligible`, computed from current restrictions and confirmed state. A confirmed historical booking remains recorded, but `entryEligible=false` means it must not be treated as permission to enter. Automated cancellation, capacity release, and refund treatment for excluded paid members require Product/Finance/Safety policy. Production requires `MODERATION_API_URL` and `MODERATION_SERVICE_TOKEN`.

Every authenticated Booking route also rechecks the actor's global suspension so an older signed member or administrator token cannot continue using the service. The administrator registration UI displays the current entry decision separately from booking status.

Administrators can list an event's booking records with `GET /api/v1/admin/events/{eventId}/registrations?limit=25&offset=0`. The endpoint enforces the admin role from the signed token, returns account IDs only to authorized callers, and includes `hasMore` for pagination. Migration 4 indexes the event/date ordering. The GraphQL gateway's `eventRegistrations` query joins these booking records to Account-owned nickname/email via an admin-only batch lookup. The booking rows include collection and check-in timestamps; `CONFIRMED` + `AT_VENUE` alone means the seat is reserved, not paid.

The dedicated admin registrations page also uses `GET /api/v1/admin/registrations?limit=25&offset=0` to page across all events. It has the same role check and privacy boundary; migration 5 indexes global booking order. The selected-event filter continues to use the event-scoped endpoint. Neither endpoint changes member-owned `GET /bookings` behavior.

Migration 3 adds the immutable `paymentMethod` booking snapshot. A member can select `AT_VENUE` only when Event permits it. That reservation atomically increments confirmed capacity and publishes `BookingConfirmed` without starting PayHere. Its confirmed status means the seat is reserved, **not that payment was collected**. `ONLINE` retains the expiring hold and verified-Payment confirmation flow. Existing bookings default to `ONLINE`. `BOOKING_TEST_DATABASE_URL` enables the disposable-schema PostgreSQL capacity, idempotency, expiry, duplicate-event, collection, and attendance component test.

It must prevent overselling through transactional allocation controls and idempotent commands.

## Responsibilities

- Validate event registration/policy and member eligibility for booking.
- Create one time-limited hold using atomic capacity allocation.
- Store immutable event price, currency, and relevant policy version.
- Expire/release holds idempotently.
- Confirm valid bookings after a trusted payment-completed fact.
- Permit cancellation and expiry of unpaid online holds; retain confirmed venue reservations under the no-cancellation policy.
- Record administrator venue collection and check-in. A future policy must define event-scoped staff access and no-show handling.
- Provide authoritative allocation/usage checks to Event and matching eligibility to Matchmaking.

## Does not own

Event configuration, PayHere/provider state, account credentials/profiles, matchmaking scores, or notification delivery.

## Implemented API

```text
POST   /api/v1/bookings
GET    /api/v1/bookings/{bookingId}
GET    /api/v1/bookings
POST   /api/v1/bookings/{bookingId}/cancel
POST   /api/v1/admin/bookings/{bookingId}/venue-collection admin
POST   /api/v1/admin/bookings/{bookingId}/check-in admin
GET    /api/v1/admin/events/{eventId}/registrations admin
GET    /api/v1/admin/registrations admin
GET    /internal/v1/bookings/{bookingId}/payment-snapshot constrained Payment integration
```

Create/cancel/venue-collection/check-in require idempotency keys. Member booking access uses authenticated subject ownership.

## Data

Implemented tables include `booking`, `capacity_allocation`, `idempotency_record`, `booking_staff_action`, `inbox`, and `outbox`. Venue collection and check-in are recorded on `booking`.

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
CONFIRMED venue bookings remain reserved; collection and check-in add timestamps without changing booking state
Late/mismatched payment -> PAYMENT_REVIEW
```

## Events

Produces `BookingPending`, `BookingConfirmed`, `BookingCancelled`, `HoldExpired`, `VenuePaymentCollected`, and `BookingCheckedIn` facts.

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
- Administrator-only attendance and collection, current eligibility, and idempotent action audit.
- Migration, outbox/inbox, metrics, and sensitive-log behavior.

## Completion criteria

Concurrency tests prove no overselling; every hold reaches a controlled terminal/review state; confirmed booking is authoritative for event participation; duplicate commands/events are safe; operations can observe and recover failures.

Update this README, architecture/data/testing docs, contracts, and runbooks whenever behavior changes.

