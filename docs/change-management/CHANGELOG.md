# MatchMate Change Log

### CHG-20260829-003 — Complete the member booking and PayHere checkout slice

- **Before:** Members could discover events, while Booking and Payment were backend-only first slices without booking list/cancellation APIs or a web purchase journey.
- **After:** Members can create an atomic event hold, submit transient checkout contact fields to PayHere, inspect authoritative booking/payment status in `/app/bookings`, and idempotently cancel an unpaid hold with immediate capacity release. Confirmed cancellation remains blocked until refund policy is approved.
- **Contracts/data:** Booking OpenAPI adds list and cancel operations; Booking AsyncAPI adds `BookingCancelled`; Booking migration 2 adds `cancelled_at`. Existing Payment v1 contracts remain compatible.
- **Security/privacy:** Access tokens remain memory-only; Booking enforces subject ownership. Checkout contact fields are sent to Payment/PayHere and are not stored in the frontend or Payment database. Browser redirects are never treated as payment success.
- **Deployment/rollback:** Apply Booking migrations through version 2 before deploying the new API. Rolling back the UI/API leaves the additive nullable column intact; preserve Booking and Payment databases for reconciliation.
- **Verification:** Booking and Payment Go tests/vet pass. Member web typecheck, 17 tests, and production build pass. Real PostgreSQL/RabbitMQ contention, PayHere sandbox, callback rate limiting, refund/provider resolution, attendance, and moderation consumers remain release blockers.

## Unreleased

### CHG-20260829-002 — Add the first executable Booking and Payment-confirmation slice

- **Status:** In progress
- **Date:** 2026-08-29
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Booking Service, Booking PostgreSQL schema, Booking REST/events, Payment-to-Booking messaging, local Compose
- **Decision/ADR:** Not required; implements documented Booking ownership and asynchronous Payment confirmation

#### Before

Booking was documentation-only, so Payment could not retrieve an authoritative price snapshot or confirm participation. No service owned consumed capacity at runtime.

#### After

Booking creates authenticated, idempotent 15-minute holds from Event-owned price/currency and atomically reserves Booking-owned capacity. It exposes an owner-authorized Payment snapshot, expires holds with capacity release, consumes Payment completion/review facts with inbox deduplication, confirms valid holds exactly once, routes late success to review without reallocating capacity, and publishes Booking facts through a transactional outbox relay. Local ports avoid the existing Matchmaking topology. Cancellation, waitlist, attendance, automatic refunds, and production policy remain incomplete.

#### Compatibility and migration

This additive slice introduces Booking migration 1 and v1 REST/event contracts. No Booking production data exists to backfill. Rollback stops Booking/Payment processes but retains both databases for financial and allocation reconciliation.

#### Security, privacy, and moderation impact

Member endpoints validate Account-issued ES256 tokens and enforce subject ownership. Amount/currency come from Event and are snapshotted by Booking; clients cannot set them. Events contain scalar IDs and minimum operational money fields, not profiles or private preferences.

#### Deployment and rollback

Start Account/RabbitMQ, Event, Booking/Payment databases and migrations, APIs, expiry worker, relays, and consumer in that order. Monitor capacity invariants, expiry, inbox/outbox age, reviews, and confirmation latency. Preserve databases on rollback and reconcile before restart.

#### Verification

Booking and Payment unit tests/vet pass. Compose validation, real PostgreSQL contention/migration, RabbitMQ redelivery/outage, and PayHere sandbox E2E evidence are required before release.

#### Documentation updated

Booking README; Booking OpenAPI/AsyncAPI; architecture, data, security, testing, implementation, Compose and local runbook guides; this change log.

### CHG-20260829-001 — Add the first executable Payment Service slice

- **Status:** In progress
- **Date:** 2026-08-29
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Payment Service, Payment PostgreSQL schema, REST/payment-event contracts, PayHere configuration, security and operations
- **Decision/ADR:** Not required; implements the existing Payment ownership and PayHere adapter decision

#### Before

Payment was documentation-only. No executable initiation, callback verification, payment persistence, audit, replay protection, status API, PayHere adapter, or versioned Payment contracts existed. Booking and its immutable price snapshot are also not implemented.

#### After

The first Payment slice provides authenticated initiation using only an eligible Booking-owned snapshot for authoritative money, domain-specific PayHere sandbox/live configuration and protocol hashes, a form-encoded callback endpoint, constant-time checksum comparison, exact amount/currency verification, callback fingerprints, idempotent state transitions, member-owned status reads, audit records, transactional outbox facts, a publisher-confirmed RabbitMQ relay with crash-safe claims, and pending-payment reconciliation classification. Invalid, unknown, or mismatched outcomes cannot complete a payment. Checkout contact fields are returned transiently to PayHere and are not persisted. The slice remains in progress until provider reconciliation/refunds, component tests, and PayHere sandbox/live evidence exist.

#### Reason

Phase 4 requires verified, replay-safe PayHere processing without trusting browser-controlled price or success state.

#### Compatibility and migration

This is additive and introduces Payment migration 1 plus v1 OpenAPI/AsyncAPI contracts. There is no existing Payment data to backfill. Retain the isolated database on rollback for financial evidence and forward recovery.

#### Security, privacy, and moderation impact

ES256 authentication and ownership protect initiation/status. Provider callbacks use PayHere verification rather than member JWT. No secrets, card data, raw callbacks, checkout contact fields, or internal account identifiers are returned or logged. Callback mismatches enter review and remain auditable.

#### Deployment and rollback

Deploy only after Booking exposes its constrained snapshot and configure database, JWT public key, approved URLs, and secret-managed PayHere credentials. Start with sandbox and a public HTTPS callback. Monitor callback failures/reviews, pending age, outbox age, and confirmation latency. Rollback stops the service but preserves its database; rotate any exposed provider secret.

#### Verification

Go unit tests cover money/snapshot validation, provider status mapping, checkout hashing, notification verification, and tampering. Compilation/vet and live component/sandbox evidence are recorded when run. Booking/RabbitMQ integration, concurrency, recovery, and production verification remain pending.

#### Documentation updated

Payment README; architecture, data, security, testing, and implementation guides; OpenAPI/AsyncAPI READMEs and v1 contracts; PayHere runbook; this change log.

### CHG-20260828-001 — Add the first executable Event Service slice

- **Status:** In review
- **Date:** 2026-08-28
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Event Service, Event PostgreSQL schema, REST/event contracts, local development, architecture/data/implementation documentation
- **Decision/ADR:** Not required; implements the already documented Event Service boundary and platform conventions

#### Before

The Event Service contained documentation only. There was no executable API, service-owned migration, event lifecycle implementation, discovery query, audit record, transactional outbox, container build, or local test command. Event REST and RabbitMQ contracts were proposed but not versioned specifications.

#### After

The repository has an executable Event Service with REST endpoints for scoped organizer listing, draft creation/replacement, publication, registration open/close, cancellation, future-event discovery, and public detail. Event owns migration version 1 with the `event`, `event_audit`, and `outbox` tables. Domain validation covers schedule/registration boundaries, fixed-precision price strings, currency, capacity, lifecycle rules, organizer scope, and optimistic concurrency. Public DTOs omit assigned organizer IDs and exact venue names. Lifecycle state and its outbox fact are committed in one database transaction, and the relay publishes claimed facts with RabbitMQ publisher confirms.

Member event discovery/detail and the first organizer create/edit/lifecycle workspace are implemented. Account JWK discovery provides reproducible local authentication without a bypass. A real-PostgreSQL component harness and an authenticated PowerShell smoke script are included. A forwarding entry point under `services/event-service/tests/e2e` lets developers run the smoke test from the service directory without duplicating its logic. The smoke test performs terminating Account/Event readiness checks first, preventing a connection failure from cascading into misleading role errors. The phase remains in progress because separate policy history/replacement, Booking validation for capacity reduction, later lifecycle states, live component/browser/failure evidence, and production operations remain required.

#### Reason

Phase 2 requires organizers to manage event configuration and members to discover published events while preserving the boundary that Booking owns consumed capacity.

#### Compatibility and migration

This is additive. A new Event-owned database schema starts at migration version 1. No existing Event data or clients exist to migrate. REST and event schemas are introduced as v1 contracts. Future additive fields can remain v1; semantic or required-field breaks require a new major version.

#### Security, privacy, and moderation impact

Public discovery exposes only approved event catalog fields and deliberately excludes organizer IDs and exact venue names. Mutation authorization validates Account-issued ES256 tokens through cached Account JWK discovery or a static deployment key and requires the assigned organizer or administrator. The organizer UI rejects accounts without organizer/admin roles before entering the workspace, while the service remains authoritative. Lifecycle mutations create audit/outbox records without account PII. Cancellation requires a reason. No direct contact data, member profile data, preferences, booking consumption, or payment data is stored.

#### Deployment and rollback

Build the independently deployable Event image, start its PostgreSQL database, run migration 1, start Account API/JWK discovery, then start Event API and optionally its relay. Monitor readiness, HTTP error/latency logs, database health, stale-update conflicts, outbox age, publish failures, and broker confirms. Rollback stops the API/relay and reverts frontend routing; because this is a new isolated service, retain its database for forward recovery. The down migration is for disposable local/test databases only.

#### Verification

Event and Account `go test ./...` and `go vet ./...` pass on Go 1.26.7. The isolated real-PostgreSQL migration/lifecycle/audit/outbox component test passes. Web typecheck, 12 tests, and production build pass; admin typecheck, 2 tests, and production build pass. Compose configuration validates. Account/Event PostgreSQL migrations and readiness checks passed in Docker, the development organizer seed completed, and the authenticated create/publish/public-privacy smoke test passed. The live Event relay publisher-confirmed `EventCreated` and `EventPublished`, then reduced unpublished outbox count to zero. Broker outage, retry/DLQ, and replay failure evidence remains pending.

#### Documentation updated

`README.md`; Account/Event service READMEs; member/organizer app READMEs; implementation, development, data, design, security, and runbook guides; Event relay runbook; OpenAPI/AsyncAPI READMEs; `infrastructure/compose/README.md`; this change log.
