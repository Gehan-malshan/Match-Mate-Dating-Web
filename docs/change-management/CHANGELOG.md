# MatchMate Change Log

## Unreleased

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
