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

## Implemented API

```text
GET    /api/v1/events
GET    /api/v1/events/{eventId}
POST   /api/v1/events
PATCH  /api/v1/events/{eventId}
POST   /api/v1/events/{eventId}/publish
POST   /api/v1/events/{eventId}/open-registration
POST   /api/v1/events/{eventId}/close-registration
POST   /api/v1/events/{eventId}/cancel
```

Mutations require assigned organizer/admin scope, optimistic concurrency, reason/audit where appropriate, and valid state transitions.

## Implemented data

The first migration creates `event`, `event_audit`, and `outbox`. Location, capacity policy, and matching policy are versioned on the event aggregate for this first slice; separate policy history tables are deferred until published-policy replacement and Booking validation are implemented.

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

Update this README, architecture/data docs, contracts, and tests whenever behavior changes. Record the before/after impact in the pull request; this repository deliberately does not maintain a merge-prone shared change-log file.

## Current implementation boundary

The executable Go service uses `net/http`, `pgx`, and a service-owned PostgreSQL database. Draft creation/replacement, assigned-organizer listing, publish, open registration, close registration, cancellation, public discovery, and public detail are implemented. Every update uses `expectedVersion`; stale writes return `EVENT_VERSION_CONFLICT`. Mutation authorization validates an ES256 access token using Account Service JWK discovery or a configured static public key, then permits only the assigned organizer or an administrator. Lifecycle changes create audit and transactional outbox records. `cmd/outbox-relay` claims unpublished records safely, publishes persistent messages with RabbitMQ publisher confirms, and marks them published only after acknowledgement.

Discovery intentionally omits organizer IDs and the exact venue name. It returns only future events in `PUBLISHED`, `REGISTRATION_OPEN`, or `REGISTRATION_CLOSED` state. Configured capacity is a catalog value and is not an availability promise; Booking remains authoritative for holds and consumed seats.

The first organizer UI is in `frontend/apps/admin`; member discovery/detail is in `frontend/apps/web`. A real-PostgreSQL component suite is available when `EVENT_TEST_DATABASE_URL` is set, and `tests/e2e/event-service-smoke.ps1` verifies the local authenticated create/publish/public-discovery path.

Still open for Phase 2: separate policy history/replacement endpoints, Booking validation before capacity reduction, later lifecycle states, production gateway integration, automated browser E2E, RabbitMQ failure/DLQ tests, and production operational evidence. The service is not production complete until these items and related policy decisions are resolved.

## Beginner verification

Run the fast tests from this directory:

```powershell
go test ./...
go vet ./...
```

Start the complete local baseline (Account/Profile, Event, their databases, and four shared development events) from the repository root:

```powershell
docker compose up --build -d
```

`event-seed` runs only with `APP_ENV=development`, after migrations. It inserts or refreshes four fixed-ID upcoming catalogue fixtures so all developers see the same event names and fields; it never runs in a production topology and does not create bookings or payments.

For the Event stack alone:

```powershell
docker compose -f infrastructure/compose/event.compose.yml up -d --build
$env:DATABASE_URL='postgres://matchmate:matchmate@localhost:5434/matchmate_event?sslmode=disable'
go run ./services/event-service/cmd/migrate
```

By default, the API loads Account Service signing keys from `http://localhost:8081/.well-known/jwks.json`. Start Account Service before testing organizer commands. A deployment may instead inject a matching static public key through `JWT_PUBLIC_KEY_PEM`.

```powershell
$env:HTTP_ADDRESS=':8082'
$env:JWT_ISSUER='matchmate-account'
$env:JWT_AUDIENCE='matchmate-api'
$env:ACCOUNT_JWKS_URL='http://localhost:8081/.well-known/jwks.json'
go run ./services/event-service/cmd/api
```

These checks need no token:

```powershell
Invoke-RestMethod http://localhost:8082/health/live
Invoke-RestMethod http://localhost:8082/health/ready
Invoke-RestMethod 'http://localhost:8082/api/v1/events?limit=10'
```

Use `contracts/openapi/event-v1.yaml` for request bodies. Organizer commands require `Authorization: Bearer <access-token>`; never add a local authentication bypass.

After Account and Event APIs are running, execute the safe development smoke test from the repository root:

```powershell
./tests/e2e/event-service-smoke.ps1
```

The smoke test checks both `/health/ready` endpoints first and stops with a startup hint when either API or its PostgreSQL dependency is unavailable.

When your terminal is already inside `services/event-service`, the equivalent convenience command is:

```powershell
./tests/e2e/event-service-smoke.ps1
```

To run the real-PostgreSQL component suite against the local Event database:

```powershell
$env:EVENT_TEST_DATABASE_URL='postgres://matchmate:matchmate@localhost:5434/matchmate_event?sslmode=disable'
go test ./services/event-service/internal/store/postgres -count=1
```

Start the relay after the Account development Compose stack has started RabbitMQ:

```powershell
docker compose -f infrastructure/compose/event.compose.yml --profile messaging up -d --build event-outbox-relay
```
