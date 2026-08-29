# Matchmaking Service

The executable prototype owns deterministic candidate eligibility, immutable participant snapshots, explainable weighted scoring, event-wide optimization, administrator review/override/lock/publish, member responses, reveal consent, structured feedback, audit, and transactional outbox records. It uses no machine-learning model.

The canonical rule specification remains [`../../docs/matchmaking/README.md`](../../docs/matchmaking/README.md).

## Implemented prototype boundary

- Go API on port `8083` with Account-issued ES256 JWT validation.
- Independent PostgreSQL database on development port `5435`.
- Immutable `prototype-v1` ruleset and ten fixture participant projections for event `11111111-1111-4111-8111-000000000001`.
- Reciprocal account/profile/booking, group, age, block, safety, deal-breaker, and repeat-pair hard filters.
- Relationship, personality, interests, lifestyle, values, and language/broad-location component scores.
- `IGNORE_AND_RENORMALIZE` missing-data policy; absent optional answers are not scored as zero.
- Hungarian maximum-weight bipartite optimization with cardinality priority and lexicographic deterministic tie-breaking.
- Original suggestions preserved separately from review selections and audited overrides.
- Hard-eligibility revalidation before immutable locking.
- Member-safe published-match responses using participant codes and generalized reasons only.
- Idempotency keys for generation, override, lock, publish, response, consent, and feedback commands.
- Reversible reveal consent with an append-only, policy-versioned decision history; current member state is a separate projection.
- Transactional outbox rows for lifecycle facts. RabbitMQ relay delivery is explicitly deferred from this prototype.

The fixture projection is a Matchmaking-owned simulation of future Booking, Account, Event, and Moderation facts. The service never reads another service database. Production ingestion through inbox-deduplicated events remains a later integration phase.

## Ruleset

`prototype-v1` uses the repository's proposed initial weights:

| Component | Weight |
|---|---:|
| Relationship intention | 25% |
| Personality compatibility | 20% |
| Shared interests | 20% |
| Lifestyle compatibility | 15% |
| Values | 10% |
| Shared language and broad location | 10% |

The minimum selectable score is 45. Used rulesets are never updated; any rule, weight, matrix, threshold, missing-data, or optimizer semantic change requires a new version.

## API

```text
GET  /health/live
GET  /health/ready
GET  /api/v1/events/{eventId}/matching-runs
POST /api/v1/events/{eventId}/matching-runs
GET  /api/v1/matching-runs/{runId}
POST /api/v1/matching-runs/{runId}/review
POST /api/v1/matching-runs/{runId}/overrides
POST /api/v1/matching-runs/{runId}/lock
POST /api/v1/matching-runs/{runId}/publish
GET  /api/v1/matches/mine
POST /api/v1/matches/{matchId}/response
POST /api/v1/matches/{matchId}/reveal-consent
POST /api/v1/matches/{matchId}/feedback
```

The canonical request/response contract is [`../../contracts/openapi/matchmaking-v1.yaml`](../../contracts/openapi/matchmaking-v1.yaml). Protected commands require `Authorization: Bearer <access-token>`; non-repeatable commands also require `Idempotency-Key`.

Administrator fixture login:

```text
admin@example.test
MatchMateDev123!
```

Matching-run listing, generation, review, override, lock, and publication require the `admin` role. Member match/response/consent/feedback routes remain participant-scoped. No development authentication bypass exists.

## Run lifecycle

```text
GENERATED -> UNDER_REVIEW -> LOCKED -> PUBLISHED
```

Generation snapshots all inputs and persists every candidate eligibility decision. Review creates a mutable selected-set projection while preserving original optimizer suggestions. Overrides must select an already eligible, above-threshold candidate and cannot select a participant twice. Lock re-evaluates current hard eligibility, writes immutable pairings, and enforces one pairing per event participant/round. Publication is the only state visible through `/matches/mine`.

## Local startup

From the repository root:

```powershell
docker compose up --build -d
docker compose ps -a
Invoke-RestMethod http://localhost:8083/health/ready
```

Successful one-shot jobs show `matchmaking-migrate` and `matchmaking-seed` as `Exited (0)`. The API and Matchmaking PostgreSQL remain running.
The migrator applies ordered, additive schema versions; version 2 introduces reveal-consent history and safely backfills existing decisions.

Fast verification:

```powershell
go -C services/matchmaking-service test ./...
go -C services/matchmaking-service vet ./...
```

## Privacy and safety

- Participant snapshots contain private inputs and never appear in member APIs or events.
- Candidate rejection codes are administrator/internal diagnostics, never member explanations.
- Member results include only their match ID, event ID, partner code, score, generalized reasons, and their own response/consent state.
- Exact locations, private deal-breakers, blocks, safety state, preferences, emails, and dates of birth are not emitted.
- Override cannot bypass any hard constraint.
- Reveal consent requires mutual `INTERESTED` responses; grant/revoke changes record policy and decision versions without erasing history.
- Feedback is structured; free text is excluded from this prototype to reduce PII and moderation risk.

## Current limitations and next integration

This is a complete deterministic prototype vertical slice, not production completion. Before production, replace fixtures with inbox-deduplicated Account/Event/Booking/Moderation facts, add the RabbitMQ relay/retry/DLQ runbook, event-state integration, component/concurrency/performance suites, rate limiting, metrics/traces, retention/backup evidence, and approved questionnaire/group/reveal policies. The administrator run-management UI is implemented in `frontend/apps/admin`.

Every future matching behavior change must update the ruleset version, deterministic fixtures/tests, OpenAPI/AsyncAPI contracts, canonical matchmaking guide, this README, and the pull-request before/after record.
