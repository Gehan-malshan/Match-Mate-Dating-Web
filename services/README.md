# Go Services

This directory contains independently deployable Go microservices. Each service will own its domain model, database, migrations, APIs, events, tests, Dockerfile, and operational documentation.

Account/Profile, Event, Matchmaking, Booking, Payment, and the Notification in-app vertical slice are executable. Moderation remains documentation-only. Notification has an authenticated member feed plus a development-only provider sink; it is not production email/SMS delivery.

Business entities and database models must not be shared between services.

## Service map

| Service | Source of truth for | Depends on facts from |
|---|---|---|
| Account/Profile | Identity, sessions, profiles, preferences, visibility, blocks | Moderation actions |
| Event | Event catalog, lifecycle, price, configured capacity/policy | Booking projection for non-authoritative availability |
| Booking | Holds, consumed capacity, bookings, attendance | Event policy; Payment completion; account restrictions |
| Payment | PayHere orders/callbacks, payment/refund/reconciliation state | Booking eligibility and immutable price snapshot |
| Matchmaking | Eligibility snapshots, scores, runs, pairings, responses, consent, feedback | Account/profile, Event, Booking, Moderation facts |
| Notification | Templates, delivery/attempts, member feed/read state, suppression | Minimum-safe Account and Booking events; later approved domain facts |
| Moderation | Reports, cases, actions, appeals, safety audit | Reports/content/account/event context |

## Standard requirements

Every service must have:

- An independent Go module and OCI image.
- Domain/application/adapters separation.
- Service-owned PostgreSQL migrations and credentials.
- OpenAPI for REST and AsyncAPI for events.
- Transactional outbox and consumer inbox where events are used.
- Idempotent commands and consumers.
- Health/readiness/liveness, structured logs, metrics, traces, dashboards, alerts, and runbook.
- Unit, component, contract, failure, security, and relevant performance tests.
- A README updated whenever ownership, behavior, APIs, events, data, configuration, or operations change.

Before implementing a service, read `AGENTS.md`, the architecture/implementation/data/testing guides, and this service's README.
