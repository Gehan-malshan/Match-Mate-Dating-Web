# MatchMate Engineering Instructions for Developers and Coding Agents

This file defines mandatory repository-wide rules. It applies to every directory unless a deeper `AGENTS.md` adds stricter instructions. Human developers and coding agents must follow the same workflow.

## 1. Mission and non-negotiable product boundaries

MatchMate is a privacy-first blind-dating and event platform that helps verified community members progress to safe, organized, real-world interactions.

The initial product has these boundaries:

- No member-to-member chat.
- No publication of phone numbers, email addresses, social handles, exact addresses, or other direct contact details.
- Community profiles expose only explicitly approved, non-sensitive fields.
- Private matchmaking preferences are not public profile fields.
- Event participation requires an eligible, confirmed booking.
- Matchmaking is deterministic and explainable; the initial system uses no machine-learning model.
- Organizers may review and override generated pairings, but every override is audited.
- Identity or contact reveal requires an approved policy and explicit consent.
- Safety, blocking, reporting, moderation, and auditability are core requirements.

Do not weaken these boundaries to simplify implementation.

## 2. Required reading before work

Before planning or editing, read:

1. `README.md`
2. `docs/README.md`
3. `docs/architecture/README.md`
4. `docs/implementation/README.md`
5. `docs/development/README.md` before changing tools, workspace configuration, or local dependencies
6. The README in every application, service, package, contract, or infrastructure directory you will change
7. Relevant specialized guides under `docs/`
8. Existing ADRs and change-log entries related to the change
9. `docs/design/README.md` before designing or changing a frontend interface, visual token, shared component, imagery rule, or responsive navigation behavior

Do not start implementation from a ticket title alone.

## 3. Architecture invariants

- The monorepo does not imply one deployment. Every application and service is independently buildable and deployable.
- Each service owns its domain logic, PostgreSQL database, migrations, credentials, backups, and operational signals.
- A service must never read, join, or write another service's database.
- Cross-service identifiers are scalar IDs, not cross-database foreign keys.
- The browser uses the typed GraphQL gateway for immediate commands and queries. The gateway calls service-owned REST/JSON APIs; RabbitMQ events propagate completed business facts.
- Critical dual writes use a transactional outbox; consumers use inbox deduplication and idempotent state transitions.
- Business entities and database models are not shared across services.
- Shared packages are limited to frontend UI, contract helpers, telemetry, and other technical concerns.
- Booking owns consumed capacity and seat holds.
- Payment owns PayHere integration, callback verification, reconciliation, and payment state.
- Payment amount and currency come from an immutable server-side booking price snapshot, never from the client.
- Matchmaking owns eligibility, scoring, optimization, pairing history, responses, reveal consent, and feedback.
- The API Gateway performs edge controls, but every service repeats domain authorization.

An exception requires an accepted ADR before implementation.

## 4. Required change workflow

For every change:

1. **Understand:** identify the affected user journey, service owners, APIs, events, tables, privacy classes, tests, and operational signals.
2. **Record the baseline:** describe the current behavior and problem in the pull-request before/after summary required by `docs/change-management/README.md`.
3. **Design:** update or add contracts and an ADR when the change affects architecture, ownership, security, consistency, or external behavior.
4. **Implement a vertical slice:** include domain logic, persistence, API/event integration, frontend behavior where applicable, observability, and error handling.
5. **Test:** add the required unit, component, contract, integration, end-to-end, performance, failure, or security tests defined in `docs/testing/README.md`.
6. **Update documentation:** describe the final behavior in canonical docs and the affected service/application README.
7. **Complete the change record:** include before, after, migration, compatibility, security/privacy, deployment, rollback, and verification details.
8. **Review:** confirm all definition-of-done items in the pull-request template.

Documentation must describe the implemented result, not an intended result that was not delivered.

## 5. Change impact rules

| Change type | Required updates |
|---|---|
| GraphQL or REST endpoint/DTO | GraphQL SDL or OpenAPI, API guide, service README, contract tests, change log |
| RabbitMQ event | AsyncAPI, producer and consumer READMEs, compatibility tests, replay/retention notes, change log |
| Database schema | Service migrations, data guide, rollback/forward plan, migration tests, change log |
| Service boundary or ownership | Architecture guide, data guide, ADR, affected READMEs, change log |
| Match rule or score | Matchmaking guide, ruleset version, explanation behavior, deterministic tests, change log |
| Privacy or visibility | Architecture and security guides, authorization/privacy tests, threat model, change log |
| Booking/payment state | Architecture/data guides, API/events, reconciliation behavior, failure tests, change log |
| Deployment/configuration | Infrastructure README, runbook, environment variables, rollback steps, change log |
| CI/test policy | Testing guide, GitHub documentation, branch protection notes, change log |
| Developer tool/workspace | Development guide, `.vscode`, version/config files, CI compatibility, change log |
| Visual design or shared UI | Design guide, affected app/package README, accessibility/visual tests, change log |

## 6. Contract and compatibility rules

- Use `/graphql` as the browser-facing endpoint and `/api/v1` as the canonical internal service base path.
- Use `application/problem+json` with stable error codes and a `traceId`.
- Use UTC ISO-8601 timestamps and decimal money with an explicit ISO 4217 currency.
- Require `Idempotency-Key` for booking, payment initiation, cancellation, refund, and other non-repeatable commands.
- Events use past-tense business facts and include `eventId`, `eventType`, `schemaVersion`, `occurredAt`, `aggregateId`, `correlationId`, `causationId`, and `actorId` when applicable.
- Additive contract changes may remain in the current major version; required-field or semantic breaks require a new major version and migration plan.
- Never serialize database entities directly as API or event payloads.
- Never include credentials, tokens, private preferences, full payment callbacks, or unnecessary PII in events or logs.

## 7. Data and migration rules

- Every schema change uses a versioned migration owned by one service.
- Migrations must work on an empty database and the latest production-like schema.
- Prefer expand/migrate/contract for breaking data changes.
- Do not combine irreversible data destruction with the deployment that introduces new readers.
- Define backfill, validation, rollback, retention, and backup effects before migration.
- Monetary values use fixed precision; never floating point.
- Store immutable snapshots where historical reproducibility matters, including booking price and matchmaking inputs/ruleset version.

## 8. Security, privacy, and safety rules

- Enforce least privilege at the gateway, service, database, broker, and operator levels.
- Use short-lived access tokens and rotating refresh sessions; avoid PII in token claims.
- Store secrets in an approved secret manager or local ignored environment file, never in Git.
- Public profile responses are built from an explicit allow-list.
- Authorize every object access against the authenticated subject, role, event scope, and current resource state.
- Rate-limit authentication, profile discovery, report submission, booking, payment initiation, and provider callbacks appropriately.
- Audit profile visibility changes, organizer overrides, moderation actions, booking/payment transitions, and reconciliation.
- A blocked, suspended, hidden, or safety-excluded member cannot enter new pairing runs or identity reveal.

## 9. Testing expectations

- Domain rules require deterministic unit tests.
- Persistence, migrations, locks, outbox/inbox, and broker behavior require component tests with real PostgreSQL/RabbitMQ dependencies.
- API and event changes require contract compatibility tests.
- Critical user journeys require end-to-end tests.
- Capacity, payment callbacks, matching runs, and queue recovery require concurrency/performance/failure tests before production.
- Security-sensitive changes require authorization, privacy leakage, replay, rate-limit, and audit tests.
- A test may not depend on execution order or shared mutable state.

## 10. Observability requirements

Every deployable service must provide:

- Health, readiness, and liveness behavior.
- Structured logs with service, environment, trace, span, correlation, route/operation, result code, and latency.
- Metrics for request rate/errors/latency, dependency saturation, domain failures, outbox age, queue lag, and DLQ depth as applicable.
- OpenTelemetry context propagation across HTTP, outbox publication, RabbitMQ, and consumers.
- Actionable alerts and a runbook for production-critical signals.

Never log passwords, tokens, secrets, raw identity documents, full provider callbacks, or unnecessary PII.

## 11. Git and pull-request rules

- Keep changes focused and independently reviewable.
- Do not mix unrelated refactoring with product behavior changes.
- Use conventional, descriptive commit messages where practical.
- Do not commit generated secrets, local databases, build outputs, or editor-specific state.
- Do not rewrite shared branch history.
- Every pull request must complete `.github/pull_request_template.md`.
- A change is not complete until code, contracts, migrations, tests, documentation, and change history agree.

## 12. Handling uncertainty

Do not invent product, legal, privacy, payment, or event-safety policy. Mark uncertainty as an **OPEN QUESTION**, identify an owner, and implement only a reversible safe default when work can proceed without making the policy decision irreversible.

## 13. Definition of done

A change is done only when:

- Acceptance criteria are met.
- Architecture invariants remain true or an ADR approves the exception.
- APIs/events are versioned and compatible.
- Migrations and rollback/forward behavior are verified.
- Required tests pass.
- Security, privacy, moderation, and audit impacts are addressed.
- Observability and operational behavior are documented.
- Canonical documentation and service README files match implementation.
- The before/after change-log entry is complete.
