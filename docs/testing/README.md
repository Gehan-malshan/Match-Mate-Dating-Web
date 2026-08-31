# MatchMate Testing and Quality Strategy

This guide defines mandatory testing levels, ownership, critical scenarios, environments, and CI gates. Tests are part of the architecture: they prove privacy, money, capacity, matching, moderation, and compatibility invariants.

## 1. Principles

- Test behavior and invariants, not framework implementation details.
- Use deterministic tests with controlled time, IDs, randomness, and provider responses.
- Keep service unit/component tests with the service; keep cross-system tests under `/tests`.
- Use real PostgreSQL/RabbitMQ dependencies for behavior that depends on transactions, locks, SQL, migrations, redelivery, or acknowledgements.
- Mock external providers at the adapter boundary; also run approved sandbox/recorded-contract tests.
- Every defect receives a regression test at the lowest useful level.
- Flaky tests are failures to fix, not checks to rerun until green.
- Tests may not use real customer PII, credentials, payment data, or safety evidence.

## 2. Test levels

| Level | Purpose | Typical scope | Runs |
|---|---|---|---|
| Unit | Domain rules, values, scoring, state transitions | Pure Go/TypeScript, no network/database | Every change |
| Component | One service with real database/broker/adapters | Migrations, repositories, HTTP, outbox/inbox, locks | Every affected service PR |
| Contract | Producer/consumer and frontend/API compatibility | OpenAPI, AsyncAPI, generated clients, schema compatibility | Every contract/consumer change |
| Integration | Multiple real services/dependencies | Payment-to-booking, account restriction propagation | Relevant PR and main |
| End-to-end | Critical user journeys through public edge/UI | Register-to-event-to-match | Main, staging, release |
| Performance | Latency, throughput, contention, saturation | Discovery, last seat, callbacks, matching, queues | Scheduled/release and risk changes |
| Failure/recovery | Controlled dependency failure and restoration | Broker/DB/provider outage, redelivery, restore | Release and architecture changes |
| Security/privacy | Abuse, authorization, leakage, replay, secrets | API/UI/infrastructure and audit | Every sensitive change and release |

Moderation changes additionally require member/organizer/moderator/admin authorization tests, owner-report leakage checks, append-only action/appeal/audit tests, action expiry/reversal tests, duplicate report/event tests, and matching/reveal race evidence once downstream consumers exist.

## 3. Test ownership and location

```text
frontend/apps/web/                member frontend unit/component tests
frontend/apps/admin/              admin frontend unit/component tests
services/<service>/              service unit/component tests
contracts/                        schemas and compatibility fixtures
tests/contract/                   cross-component contract checks
tests/e2e/                        critical complete journeys
tests/performance/                load/contention/soak plans and scenarios
```

Every test suite has an owner, execution command, dependencies, expected duration, and failure-triage guidance in its local README once code exists.

## 4. Unit test requirements by domain

### Account/Profile

- Email normalization/uniqueness.
- Password/session/token lifecycle and reuse detection.
- Profile visibility allow-list.
- Age-at-event calculation boundaries.
- Preference/questionnaire validation.
- Block/unblock and account-state rules.
- Contact-information/content validation.

### Event

- Date, registration, and lifecycle transitions.
- Price/currency validation.
- Administrator-only event creation, organizer ownership for later event operations, and event policy versions.
- Publication/cancellation constraints.

### Booking

The first executable slice has deterministic tests for Event registration validation and server-derived price/expiry behavior. Real PostgreSQL last-seat contention, migration, inbox/outbox, expiry crash/retry, and RabbitMQ failure tests remain required before release.

- State transitions and cancellation policy.
- Hold expiry and idempotent release.
- One active booking per account/event.
- Capacity category/policy rules.
- Immutable price snapshot.

### Payment

The first executable slice has deterministic unit coverage for snapshot money/state validation, PayHere request hashing, callback signature verification, tampering rejection, and provider status mapping. The Booking consumer is implemented, but real PostgreSQL migration/repository concurrency, HTTP/auth contract, broker outage/redelivery, sandbox E2E, and recovery evidence remain required before release.

- Request/hash adapter inputs.
- Exact decimal amount/currency verification.
- Callback signature and all mismatch outcomes.
- Idempotent state transitions, refund limits, reconciliation classification.

### Matchmaking

- Every hard filter and reason code.
- Component calculations, missing-data policy, weight normalization.
- Directional score combination.
- Optimizer fixtures, determinism, tie-breaking, unmatched reasons.
- Override/lock/response/reveal rules.

### Notification

The executable slice has deterministic coverage for event routing/schema validation, privacy-safe ignored payloads, template variable allow-lists, subject safety, bounded retry timing, configuration safety, health behavior, ES256 member authentication, owner-scoped feed/read behavior, cursor/popup selection, frontend API behavior, and a disposable-schema PostgreSQL component harness for inbox/delivery/feed/suppression/attempt behavior. Required RabbitMQ redelivery/DLQ, provider-failure, crash-window, load, browser E2E, and production email-provider evidence remain open.

- Template variable validation and privacy policy.
- Business idempotency key.
- Suppression/preference and retry classification.
- Notification-feed ownership, concealed cross-account reads, pagination, unread/read idempotency, and no provider/private-data leakage.
- Popup polling must not replay old unread items on initial page load and must honor reduced motion.

### Moderation

- Case/action/appeal states.
- Effective/expiry behavior.
- Target restrictions and role/scope rules.
- Reporter/evidence visibility.

The executable Moderation suite covers report validation, malformed/oversized-shape JSON, per-subject rate limiting, role isolation, invalid JWT issuer/audience/expiry/algorithm/subject, invalid resource IDs, privileged-view auditing, strict `OPEN -> TRIAGED -> INVESTIGATING -> ACTIONED | DISMISSED` transitions, duplicate reports/actions, owner-only appeals, single appeal decisions, expiry idempotency, owner-history description suppression, minimum-safe outbox payloads, outbox claim/publish behavior, and append-only audit counts. PostgreSQL coverage is enabled with `MODERATION_TEST_DATABASE_URL`; without it, that disposable-schema component test is intentionally skipped.

Still required before production are RabbitMQ outage/redelivery confirmation, multi-worker concurrency/load evidence, downstream Account/Booking/Matchmaking enforcement and race tests, gateway/distributed rate-limit tests, browser moderator/member journeys, and approved retention/legal-hold tests.

## 5. Component test requirements

Every Go service component suite must cover:

- Migrations from empty database.
- Migrations from the latest supported previous schema fixture.
- Repository constraints and query behavior.
- Transaction rollback.
- Optimistic/pessimistic concurrency where used.
- HTTP serialization, validation, authorization, errors, pagination, idempotency.
- Outbox creation and relay confirmation behavior.
- Inbox deduplication and repeated event delivery.
- Retryable versus permanent consumer/provider errors.
- Sanitized logging and expected metrics/audit records.

Use disposable dependency containers or isolated ephemeral databases. Do not share mutable databases between parallel tests.

## 6. Contract testing

### REST/OpenAPI

- Lint every specification.
- Validate requests/responses/errors against schemas.
- Compile generated TypeScript clients and any Go interfaces.
- Detect breaking removal, required-field, type, enum, status, and semantic changes.
- Verify authentication/authorization documentation and examples.
- Verify `application/problem+json`, pagination, time, money, and idempotency conventions.

### RabbitMQ/AsyncAPI

- Validate event envelope and payload at producer and consumer boundaries.
- Detect incompatible required fields, type/enum changes, routing/version changes.
- Run producer fixtures against all known consumer schemas.
- Verify unknown additive fields are ignored.
- Verify unsupported major versions fail visibly and enter the defined recovery path.
- Verify duplicate, delayed, reordered, and replayed delivery where ordering is not guaranteed.

No database entity may be used as a contract fixture.

## 7. Mandatory end-to-end journeys

### Journey A — member onboarding

```text
Register -> verify -> login -> complete profile/preferences
-> profile approved -> safe community profile visible
```

Verify no private field is exposed.

### Journey B — event purchase

```text
Discover event -> create hold -> PayHere sandbox success
-> callback verification -> confirmed booking -> matching eligibility
```

### Journey C — payment failure and recovery

```text
Hold -> payment pending/timeout/failure/duplicate/late callback
-> correct final state -> reconciliation/refund where required
```

### Journey D — matching

```text
Confirmed cohort -> generate run -> inspect reasons -> override if needed
-> lock -> publish -> each member sees only own permitted information
```

### Journey E — post-event

```text
Attend -> structured response -> mutual/non-mutual outcome
-> consent-controlled reveal -> feedback/report
```

### Journey F — safety restriction

```text
Report/block/restrict member -> removed from discovery and unlocked matching
-> reveal prevented -> audit preserved -> authorized appeal/reversal
```

## 8. Concurrency and performance tests

Baseline scenarios:

- Hundreds of concurrent attempts for the final available seat; capacity never exceeded.
- Repeated booking/payment idempotency keys.
- PayHere callback burst including duplicates and reordered outcomes.
- RabbitMQ outage and backlog recovery without duplicate business effects.
- Event discovery at expected launch query load with representative filters/data.
- Matchmaking generation at expected event size and approved growth multiplier.
- Concurrent administrator review/lock attempts; one valid lock/version result.
- Notification provider slowdown without starving critical workers.

For each scenario record workload, data volume, duration, target, measured result, bottleneck, and environment. A performance test without a target is diagnostic, not a release gate.

## 9. Security and privacy tests

### Authentication/session

- Brute-force/rate-limit behavior.
- Invalid/expired/wrong-issuer/audience/signature tokens.
- Refresh rotation, reuse, revocation, deactivation.
- Email enumeration resistance.

### Authorization

- Every object endpoint tested as owner, unrelated member, scoped organizer, unrelated organizer, moderator, support, admin, and service account as applicable.
- Horizontal and vertical privilege escalation.
- State-dependent authorization after cancellation, restriction, or event completion.

### Privacy

- Community profile allow-list snapshot tests.
- No private fields in APIs, events, logs, traces, metrics labels, notifications, exports, or errors.
- Block symmetry and reporter identity protection.
- Exact location, DOB, preferences, moderation evidence, and payment metadata protection.

### Payment/security abuse

- Callback replay/tampering, wrong amount/currency/order/merchant.
- Request-size, malformed payload, signature comparison, rate limit.
- Provider timeout and forged client success.

### Application security

- SQL/command/template injection as applicable.
- XSS and unsafe rich content.
- SSRF in URL/media/provider adapters.
- File type/content mismatch, malware scanning, oversized/decompression abuse.
- CORS/CSRF/cookie configuration according to chosen token transport.
- Dependency, container, secret, and IaC scanning.

## 10. Failure and recovery tests

- PostgreSQL connection loss and failover.
- RabbitMQ unavailable during local commit and during consumption.
- Outbox relay crash before/after publish acknowledgement.
- Consumer crash before/after inbox/business commit.
- PayHere timeout/unavailable/malformed response.
- Notification provider outage and DLQ replay.
- Redis outage causes safe fallback, never incorrect authority.
- Backup restore to an isolated environment followed by migration and reconciliation.
- Deployment rollback with backward-compatible schema.

Recovery tests verify final business invariants, not merely process restart.

## 11. Test data strategy

- Use builders/factories with fictional, obvious test identities.
- Centralize controlled clocks and ID generation.
- Maintain versioned deterministic matching matrices and expected outputs.
- Maintain sanitized provider callback fixtures for every verification path.
- Generate large datasets deterministically for performance tests.
- Never use production exports unless explicitly approved and irreversibly anonymized.

## 12. CI workflow evolution

### Repository-only stage

- Markdown lint/link checks.
- Required folder/document validation.
- Secret and filename checks.

### Code stage

An always-running CI gate determines changed components and calls reusable workflows:

```text
PR -> repository/docs/contracts checks
   -> affected frontend lint/type/test/build
   -> affected Go format/vet/lint/unit/component
   -> migration and contract compatibility
   -> security/dependency scan
   -> aggregate required CI result
```

### Main/release stage

- Build each changed image once.
- Tag with immutable commit SHA.
- Scan, generate SBOM, sign/attest, and publish.
- Deploy same artifact to staging.
- Run smoke/critical E2E.
- Require protected production approval.
- Progressive production rollout, verify health, rollback on defined criteria.

## 13. Quality gates

A pull request cannot merge when:

- Required test suite fails or is skipped without approved reason.
- Contract compatibility fails.
- Migration validation fails.
- Critical/high unaccepted security findings exist.
- Privacy/authorization behavior is untested.
- Documentation and change record are missing for behavior changes.
- A flaky test is hidden by retry-only behavior.

Coverage percentages are secondary. Critical invariants require explicit tests even when aggregate coverage appears high.

## 14. Test completion checklist

- [ ] Acceptance criteria map to tests.
- [ ] Happy, boundary, invalid, unauthorized, duplicate, concurrent, and dependency-failure paths are considered.
- [ ] Migrations and contracts are tested.
- [ ] Logs/events/notifications are checked for sensitive data.
- [ ] Test data is fictional and isolated.
- [ ] Performance/security/recovery tests are included when risk requires them.
- [ ] CI commands and local reproduction are documented.
- [ ] Change log records verification evidence.
