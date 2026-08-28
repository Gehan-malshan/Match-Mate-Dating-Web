# MatchMate

MatchMate is a privacy-first, community-driven blind-dating platform focused on helping people move from online discovery to safe, organized, real-world dating events.

This repository is a monorepo for the MatchMate member website, organizer portal, Go microservices, API and event contracts, infrastructure definitions, and technical documentation.

## Planned technology stack

- Frontend: React, TypeScript, TanStack Router, TanStack Query, TanStack Form, Zod, and Vite
- Frontend tooling: Bun for package management and script execution; Vite for development and production builds
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
|-- frontend/
|   |-- apps/
|   |   |-- web/                   Member-facing React/TanStack application
|   |   `-- admin/                 Organizer and moderation portal
|   `-- packages/
|       |-- ui/                    Shared frontend design system
|       |-- validation/            Frontend validation and generated client helpers
|       `-- telemetry/             Frontend observability helpers
|-- services/
|   |-- account-service/           Authentication, profiles, preferences, and blocks
|   |-- event-service/             Event catalog, schedules, pricing, and policies
|   |-- matchmaking-service/       Rule-based compatibility and event pairing
|   |-- booking-service/           Ticket holds, capacity, bookings, and attendance
|   |-- payment-service/           PayHere payments and reconciliation
|   |-- notification-service/      Email, SMS, and other notifications
|   `-- moderation-service/        Reports, moderation, and safety actions
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
|   |-- development/               VS Code and developer workstation setup
|   |-- design/                    Canonical Midnight Chemistry design system
|   |-- implementation/            Phased development plan and completion criteria
|   |-- matchmaking/               Deterministic matching specification
|   |-- data/                      Database ownership, schemas, and migrations
|   |-- testing/                   Test strategy and CI quality gates
|   |-- change-management/         Required before/after project history
|   |-- adr/                       Architecture decision records
|   |-- api/                       API usage and integration guidance
|   |-- security/                  Privacy, security, and threat-model documents
|   `-- runbooks/                  Operational and incident procedures
|-- tests/
|   |-- contract/                  Cross-service compatibility tests
|   |-- e2e/                       Critical user-journey tests
|   `-- performance/               Capacity and load-test plans
|-- .vscode/                       Shared extension recommendations and workspace settings
|-- .editorconfig                  Cross-editor formatting conventions
|-- .gitattributes                 Repository line-ending and binary-file rules
|-- Makefile                       Common setup, run, test, build, and cleanup commands
`-- .github/                       Repository governance and future GitHub Actions
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

The repository contains the member landing page plus the first Account/Profile vertical slice: Go REST API, PostgreSQL migration, development users, ES256/refresh-session authentication, private profiles/preferences, community-safe projections, moderation decisions, RabbitMQ outbox relay, and React registration/login/profile routes. Event, Booking, Payment, Matchmaking, Notification, and full Moderation services remain planned.

## Quick start with Make

Prerequisites are Go 1.26+, Bun 1.3+, Docker Desktop with Docker Compose, and GNU Make. On Windows, run these commands from a terminal where `make --version` succeeds (for example Git Bash or a GNU Make installation). Keep Docker Desktop running.

First-time setup:

```bash
make doctor
make setup
make up
```

`make up` builds and starts PostgreSQL, RabbitMQ, the Account API, and outbox relay. It automatically runs the idempotent database migration and creates the shared development users. Verify startup:

```bash
make status
```

The `account-migrate` and `account-seed` jobs should show `Exited (0)`, which means success. `postgres-account`, `rabbitmq`, `account-api`, and `account-outbox` should be running.

In a second terminal, start the web application:

```bash
make web
```

Open `http://127.0.0.1:5173`. The API is available at `http://localhost:8081`, and RabbitMQ management is at `http://localhost:15672` using `matchmate` / `matchmate`.

Shared web login:

```text
Email: member@example.test
Password: MatchMateDev123!
```

Additional development users are `community@example.test`, `moderator@example.test`, and `suspended@example.test`; all use the same public development-only password. These accounts never belong in staging or production.

## Make command reference

```text
make help                         Show all commands
make doctor                       Check Go, Bun, Docker, and Compose
make setup                        Install Bun and Go dependencies
make up                           Start/rebuild the local backend
make web                          Start the frontend in the current terminal
make status                       Show running and completed containers
make logs                         Follow backend/infrastructure logs
make migrate                      Re-run the safe migration job
make seed                         Restore shared development accounts
make test                         Run backend and frontend tests
make vet                          Run Go vet and TypeScript checking
make build                        Validate/build backend and frontend
make down                         Stop containers; preserve database data
make reset-local CONFIRM=YES      Delete and recreate local data on next start
```

`make reset-local CONFIRM=YES` permanently deletes the local Account PostgreSQL volume. Do not use it when local data must be retained, and never adapt it for production.

If GNU Make is unavailable, the underlying Bun, Go, and Docker commands remain documented in the Account service and Compose READMEs.

## Suggested implementation order

1. Confirm product policies, privacy rules, matching questions, and event workflows.
2. Define OpenAPI and AsyncAPI contracts.
3. Build the frontend foundation and Account/Profile Service.
4. Add Event Service.
5. Add Booking capacity and hold flow.
6. Add Payment Service and confirmed-booking integration.
7. Implement Matchmaking Service and organizer review/lock workflow.
8. Add event interaction, Notification, moderation hardening, observability, security, and production delivery.

## Documentation

This repository is designed to be understandable without external project documents. Developers and coding agents must read the following files before making changes:

1. [`AGENTS.md`](AGENTS.md) — mandatory engineering and documentation rules.
2. [`docs/README.md`](docs/README.md) — canonical project handbook index.
3. [`docs/architecture/README.md`](docs/architecture/README.md) — complete system architecture and workflows.
4. [`docs/implementation/README.md`](docs/implementation/README.md) — phased implementation plan and definition of done.
5. The README belonging to the application, service, contract, or infrastructure area being changed.

Specialized references:

- [`docs/development/README.md`](docs/development/README.md) — VS Code, toolchain, Docker, secrets, and daily workflow setup.
- [`docs/design/README.md`](docs/design/README.md) — canonical Midnight Chemistry colors, typography, spacing, components, responsive behavior, imagery, and accessibility rules.
- [`docs/matchmaking/README.md`](docs/matchmaking/README.md) — deterministic matchmaking rules and pairing algorithm.
- [`docs/data/README.md`](docs/data/README.md) — database ownership, schemas, migrations, retention, and consistency.
- [`docs/testing/README.md`](docs/testing/README.md) — mandatory test strategy and CI quality gates.
- [`docs/change-management/README.md`](docs/change-management/README.md) — documentation synchronization and pull-request change summaries.

Documentation is part of the product. Any change to behavior, APIs, events, data, security, deployment, or workflows must update the relevant documentation in the same pull request.
