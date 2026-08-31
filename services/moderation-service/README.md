# Moderation and Safety Service

Owns reports, moderation cases, safety actions, evidence references, appeals, and restricted audit records as an independently deployable Go service on local port `8087` with its own PostgreSQL database on port `5439`.

## Implementation status

The first executable vertical slice includes authenticated member/organizer reporting, duplicate and bounded in-process abuse controls, owner-safe report history, moderator/admin case access and assignment, audited case views, explicit investigation/dismissal transitions, versioned append-only actions, owner-bound appeals, uphold/reverse decisions, scheduled action expiry, restricted audit records, reference-only evidence metadata, a transactional outbox, publisher-confirmed RabbitMQ relay, OpenAPI/AsyncAPI contracts, health endpoints, Compose wiring, and unit/security/component-test harnesses.

Downstream Account, Booking, Matchmaking, and Notification consumers remain required before the service meets its cross-domain completion criteria. Until those inbox-deduplicated consumers are deployed, published enforcement facts do not by themselves prove that active exclusions are enforced everywhere.

## Responsibilities

- Accept member/organizer/system reports with safe validation and rate limits.
- Triage severity/risk, assign cases, track SLA, and restrict access.
- Store private evidence references and integrity/retention metadata.
- Apply versioned actions such as content hide, profile restriction, event exclusion, account suspension, pairing invalidation, or reveal prevention.
- Support action expiry, appeal, uphold/reverse, and complete audit.
- Publish minimum safe enforcement facts to Account, Booking, Matchmaking, and Notification.
- Provide emergency operational controls under approved safety policy.

## Does not own

Account credentials/profile source, event catalog, booking/payment state, matching score, notification delivery, or the underlying reported content object.

## Proposed API

The following v1 API is implemented:

```text
POST /api/v1/reports
GET  /api/v1/reports/mine
GET  /api/v1/moderation/cases
GET  /api/v1/moderation/cases/{caseId}
POST /api/v1/moderation/cases/{caseId}/assign
POST /api/v1/moderation/cases/{caseId}/status
POST /api/v1/moderation/cases/{caseId}/actions
POST /api/v1/moderation/actions/{actionId}/appeals
POST /api/v1/moderation/appeals/{appealId}/decision
```

Moderator/admin mutation endpoints require least-privilege roles and an audit reason. Reading a restricted case creates a privileged-view audit record. Reporter identity/evidence is not included in reported-member responses.

## Proposed data

Migration 1 owns:

`report`, `moderation_case`, `evidence_reference`, `moderation_action`, `appeal`, `moderation_audit`, and `outbox`.

Key invariants:

- Evidence and reporter identity are restricted beyond general organizer access.
- Actions are append-only with actor/reason/effective/expiry/scope.
- Reversal creates a new audit decision; history is never rewritten.
- Active safety exclusions propagate quickly and prevent new matching/reveal.
- Service events reveal only target ID, action class/scope/version, and safe effective state.

## State

```text
OPEN -> TRIAGED -> INVESTIGATING -> ACTIONED | DISMISSED
ACTIONED -> APPEALED -> UPHELD | REVERSED
```

## Automated coverage

- Report validation, duplicate/abuse/rate-limit behavior.
- Moderator/support/organizer/member authorization matrix.
- Reporter/evidence/private-note leakage tests.
- Action effective/expiry/reversal and minimum-safe outbox facts.
- Audit append-only behavior and privileged-view audit.
- Strict triage/investigation/action/dismissal transitions.
- ES256 issuer/audience/expiry/algorithm/subject validation.
- PostgreSQL constraints, outbox claim/publish state, and expiry idempotency when `MODERATION_TEST_DATABASE_URL` is set.

Production completion still requires match generation/lock/reveal safety races, retention/legal-hold/deletion policy tests, RabbitMQ retry/downstream-unavailable tests, concurrency/load evidence, and browser journeys.

## Completion criteria

Reports are triaged and resolved through audited least-privilege workflows; active actions consistently restrict all affected domains; reporters/evidence remain protected; appeals preserve history; safety operations have alerts and runbooks.

Update this README, security/architecture/data/testing docs, event contracts, runbooks, and change history whenever behavior changes.

## Configuration

| Variable | Development default | Purpose |
|---|---|---|
| `DATABASE_URL` | Required | Moderation-owned PostgreSQL connection |
| `HTTP_ADDRESS` | `:8087` | API listener |
| `RABBITMQ_URL` | Local broker | Transactional-outbox destination |
| `EVENT_EXCHANGE` | `matchmate.events` | Shared topic exchange |
| `ACCOUNT_JWKS_URL` | Account JWKS | ES256 signing-key discovery |
| `JWT_ISSUER` | `matchmate-account` | Required token issuer |
| `JWT_AUDIENCE` | `matchmate-api` | Required token audience |
| `MODERATION_REPORT_RATE_LIMIT_PER_HOUR` | `5` | Per-process member report bound; gateway/distributed enforcement remains required for multi-replica production |
| `MODERATION_EXPIRY_INTERVAL` | `30s` | Due-action expiry scan |

## Safe defaults and open questions

- Evidence is metadata-only. This service does not upload, fetch, or expose evidence objects.
- `SAFETY` reports default to `HIGH`; other accepted categories default to `MEDIUM`. Automated penalties are not applied.
- Members may appeal only an active action whose target ID equals their authenticated account ID.
- Evidence retention, legal-hold authorization, moderator SLA durations, emergency suspension policy, distributed rate limiting, and privileged evidence retrieval remain **OPEN QUESTION — Safety/Product owner**.
- Rollback stops Moderation processes and preserves the isolated database and unpublished outbox. The down migration is only for disposable development databases.

