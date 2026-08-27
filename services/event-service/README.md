# Event Service

Owns the event catalog, organizer ownership, event lifecycle, venue, schedule, ticket price, configured capacity, registration windows, and event discovery.

The Booking Service remains authoritative for consumed capacity and active ticket holds.

## Responsibilities

- Create, edit, publish, open/close registration, cancel, complete, and archive events.
- Manage assigned organizer, venue/broad location, start/end/time zone, registration window, price/currency, configured capacity and matching-policy reference.
- Provide public/member discovery with approved visibility, filters, cursor pagination, and optional nearby search.
- Version capacity and matching policies and publish lifecycle facts.
- Expose authoritative event configuration to Booking/Matchmaking through versioned APIs/events.

## Does not own

Consumed seats, ticket holds, confirmed bookings, payments, matching results, notification delivery, or account profiles.

## Proposed API

```text
GET    /api/v1/events
GET    /api/v1/events/{eventId}
POST   /api/v1/events
PATCH  /api/v1/events/{eventId}
POST   /api/v1/events/{eventId}/publish
POST   /api/v1/events/{eventId}/open-registration
POST   /api/v1/events/{eventId}/close-registration
POST   /api/v1/events/{eventId}/cancel
PUT    /api/v1/events/{eventId}/capacity-policy
PUT    /api/v1/events/{eventId}/matching-policy
```

Mutations require assigned organizer/admin scope, optimistic concurrency, reason/audit where appropriate, and valid state transitions.

## Proposed data

`event`, `event_location`, `event_capacity_policy`, `event_matching_policy`, and `outbox`.

Key invariants:

- Start is before end; registration closes before event start.
- Price uses fixed precision and explicit currency.
- Published price/policy changes create versions and cannot mutate existing Booking snapshots.
- Configured limits cannot be reduced below current authoritative allocation without Booking validation and approved policy.
- Events with business history are cancelled/archived, not hard-deleted.

## State

```text
DRAFT -> PUBLISHED -> REGISTRATION_OPEN -> REGISTRATION_CLOSED
      -> MATCHING -> ONGOING -> COMPLETED
Approved states -> CANCELLED
```

## Events

Produces `EventCreated`, `EventUpdated`, `EventPublished`, `EventCapacityChanged`, `EventRegistrationOpened/Closed`, `EventCancelled`, and `EventCompleted` as approved by AsyncAPI.

May consume safe booking allocation facts for a non-authoritative discovery projection. Commands still validate through Booking where required.

## Required tests

- All lifecycle transitions and invalid transitions.
- Organizer/admin scope and unrelated organizer denial.
- Optimistic concurrency/stale update.
- Date/time-zone/registration boundaries.
- Decimal price/currency and policy versioning.
- Discovery visibility, filters, pagination, and optional PostGIS bounds.
- Capacity reduction race and Booking validation failure.
- Event cancellation event/outbox idempotency.

## Completion criteria

An organizer can manage an event through approved states and a member can discover only eligible published events. Configuration is versioned, authorized, auditable, observable, and safe for Booking/Matchmaking consumers.

Update this README, architecture/data docs, contracts, tests, and change history whenever behavior changes.

