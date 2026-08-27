# MatchMate

MatchMate is a privacy-first, community-driven blind-dating platform focused on helping people move from online discovery to safe, organized, real-world dating events.

This repository is a monorepo for the MatchMate member website, organizer portal, Go microservices, API and event contracts, infrastructure definitions, and technical documentation.

## Planned technology stack

- Frontend: React, TypeScript, TanStack Router, TanStack Query, TanStack Form, and Zod
- Backend: Go microservices
- Databases: PostgreSQL with independent ownership per service
- Messaging: RabbitMQ
- Payments: PayHere
- Containers: Docker and Docker Compose
- CI/CD: GitHub Actions
- Observability: OpenTelemetry with a suitable metrics, logs, and tracing backend

## Repository structure

```text
Match-Mate-Dating-Web/
|-- apps/
|   |-- web/                       Member-facing React/TanStack application
|   `-- admin/                     Organizer and moderation portal
|-- services/
|   |-- account-service/           Authentication, profiles, preferences, and blocks
|   |-- event-service/             Event catalog, schedules, pricing, and policies
|   |-- matchmaking-service/       Rule-based compatibility and event pairing
|   |-- booking-service/           Ticket holds, capacity, bookings, and attendance
|   |-- payment-service/           PayHere payments and reconciliation
|   |-- notification-service/      Email, SMS, and other notifications
|   `-- moderation-service/        Reports, moderation, and safety actions
|-- packages/
|   |-- ui/                        Shared frontend design system
|   |-- validation/                Frontend validation and generated client helpers
|   `-- telemetry/                 Technical observability helpers
|-- contracts/
|   |-- openapi/                   REST API contracts
|   `-- asyncapi/                  RabbitMQ event contracts
|-- infrastructure/
|   |-- docker/                    Container build conventions
|   |-- compose/                   Local multi-service environment
|   |-- gateway/                   API gateway and edge configuration
|   |-- kubernetes/                Future orchestration manifests
|   `-- terraform/                 Future infrastructure-as-code
|-- docs/
|   |-- architecture/              Architecture documents and diagrams
|   |-- adr/                       Architecture decision records
|   |-- api/                       API usage and integration guidance
|   |-- security/                  Privacy, security, and threat-model documents
|   `-- runbooks/                  Operational and incident procedures
|-- tests/
|   |-- contract/                  Cross-service compatibility tests
|   |-- e2e/                       Critical user-journey tests
|   `-- performance/               Capacity and load-test plans
`-- .github/                       Repository and future GitHub Actions documentation
```

## Architecture principles

- The repository is shared, but every application and service remains independently deployable.
- Each Go service owns its domain logic, database, migrations, and runtime configuration.
- Services do not read or write another service's database.
- REST is used for immediate operations; RabbitMQ carries durable business events.
- Matchmaking uses transparent rules, weighted scoring, and deterministic pairing optimization without machine learning.
- Personally identifiable information and private matching preferences are not exposed in community profiles.
- Shared packages contain technical utilities or frontend components, not shared microservice domain models.

## Current status

The repository currently contains the initial directory structure and documentation placeholders. Application code, runtime configuration, and deployment definitions will be added incrementally.

## Suggested implementation order

1. Confirm product policies, privacy rules, matching questions, and event workflows.
2. Define OpenAPI and AsyncAPI contracts.
3. Build the frontend foundation and Account/Profile Service.
4. Add Event, Booking, Payment, and Notification services.
5. Implement the rule-based Matchmaking Service and organizer workflow.
6. Add moderation, observability, security hardening, and production delivery automation.

## Documentation

Start with the README files inside each directory. Record significant technical decisions under `docs/adr` before implementation creates dependencies on them.
