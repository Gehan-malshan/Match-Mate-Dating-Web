# ADR-0001: Event payment choice and venue reservations

- **Status:** Accepted
- **Date:** 2026-10-07
- **Owners:** MatchMate project team

## Context

Previously every booking was a time-limited online payment hold. Omitting Payment would therefore expire every reservation. Launch needs members to reserve a seat online and buy the ticket at the entrance, while retaining PayHere for later use.

## Decision

Event owns a draft-only `paymentOptions` value: `ONLINE`, `AT_VENUE`, or `BOTH`. Existing events and omitted API values default to `ONLINE`. Booking owns the member's immutable `paymentMethod` snapshot: `ONLINE` or `AT_VENUE`. For `BOTH`, the member chooses at reservation time. Booking validates the choice against Event and derives price/currency from Event, never the browser.

Online reservations use the existing expiring `PENDING_PAYMENT` hold. Venue reservations atomically consume confirmed capacity, create `CONFIRMED` immediately, and emit `BookingConfirmed` once. This means **seat confirmed, payment due at venue**, not a paid ticket. PayHere initiation remains restricted to pending online bookings. No venue payment collection, attendance, or check-in ledger is claimed by this change.

## Consequences and compatibility

Two additive migrations introduce defaulted, constrained columns. Existing rows become online-only/online bookings. GraphQL, REST, and booking facts gain additive fields. Old clients can book existing online-only events; venue/both events require an explicit method. Event payment options change only while a draft; existing booking snapshots never change. Rollback is application-first; retain columns until no writer depends on them. Never drop booked venue data as a rollback.

## Safety and operations

Booking's conditional PostgreSQL allocation remains capacity authority. The one-active-booking constraint applies to both methods. The frontend must not call Payment for a venue reservation. Confirmed venue cancellation, collection recording, check-in, no-show, production feature flags, and a payment-free deployment topology require separate approved policy/design. Existing PayHere development flow is not thereby certified for production.

## Verification

Domain/service tests cover choices, venue confirmation, rejection of disallowed methods, and PayHere snapshot denial. A disposable-schema PostgreSQL component test covers idempotency and final-seat contention. Rollout requires both migrations before new binaries, then contract and browser E2E checks for all three event options.
