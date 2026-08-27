# Matchmaking Service

Owns candidate eligibility, versioned profile and preference snapshots, rule-based compatibility scoring, pairing optimization, organizer review and locking, participant responses, reveal consent, and match feedback.

The initial matching engine will use deterministic business rules and weighted scoring without machine learning.

The complete algorithm specification is [`../../docs/matchmaking/README.md`](../../docs/matchmaking/README.md). This README defines the service implementation boundary; the algorithm document is canonical for rules and calculations.

## Responsibilities

- Maintain immutable, approved ruleset versions.
- Build event participant projections from confirmed/cancelled booking and restriction facts.
- Snapshot approved profile/preferences for one run without owning source data.
- Evaluate reciprocal hard constraints with internal reason codes.
- Calculate component/directional scores and weighted total.
- Run deterministic event-wide maximum-weight pairing.
- Persist run/suggestion/unmatched output and explainable safe reasons.
- Support organizer review/override/lock/publish with complete audit.
- Record member responses, mutual outcomes, reveal consent, and feedback.

## Does not own

Account credentials/source profile, event catalog, capacity/booking confirmation, payment, notification delivery, or moderation evidence.

## Proposed API

```text
GET  /api/v1/events/{eventId}/matching-runs
POST /api/v1/events/{eventId}/matching-runs
GET  /api/v1/matching-runs/{runId}
POST /api/v1/matching-runs/{runId}/overrides
POST /api/v1/matching-runs/{runId}/lock
POST /api/v1/matching-runs/{runId}/publish
GET  /api/v1/matches/mine
POST /api/v1/matches/{matchId}/response
POST /api/v1/matches/{matchId}/reveal-consent
POST /api/v1/matches/{matchId}/feedback
```

Generation, override, lock, publish, response, consent, and feedback commands require appropriate idempotency/state protection.

## Proposed data

`ruleset`, `participant_snapshot`, `matching_run`, `candidate`, `compatibility_score`, `pairing_suggestion`, `pairing_override`, `locked_pairing`, `match_response`, `reveal_consent`, `match_feedback`, `inbox`, and `outbox`.

Key invariants:

- Identical snapshots/ruleset/algorithm/tie inputs produce identical result.
- Score never overrides a block, safety restriction, booking, age/preference, deal-breaker, or event constraint.
- One participant has at most one locked pairing per event/round.
- Used rulesets, snapshots, generated results, and locked history are immutable.
- Overrides preserve original suggestion and require authorized actor/reason.
- Member APIs expose only own published, policy-approved information.

## Events

Produces `MatchingRunGenerated`, `PairingsLocked`, `PairingsPublished`, `MatchResponseRecorded`, `MutualInterestEstablished`, reveal-consent facts, and safe feedback workflow facts.

Consumes account/profile eligibility, block/restriction, event lifecycle/policy, booking confirmed/cancelled/refunded/no-show, and moderation action facts. Every consumer is inbox-deduplicated.

## Workers

- Participant projection updater.
- Matching-run generator/optimizer for asynchronous large runs.
- Outbox relay.
- Deadline/response/reveal workflow processor where approved.

Workers use bounded concurrency and explicit run status; cancellation/retry must not create a second conflicting result.

## Required tests

- All filters, score components, weights, missing-data policies, and explanations.
- Known optimum matrices, greedy counterexample, ties, imbalance, disconnected graph, thresholds, repeat penalties.
- Determinism and version immutability.
- Event/organizer/member authorization.
- Projection event duplicate/reorder and restriction-before-lock race.
- Concurrent generation/override/lock and unique pairing constraints.
- No private fields/reasons in member API, events, logs, or notifications.
- Expected event-size performance plus approved safety margin.

## Completion criteria

A confirmed cohort produces reproducible explainable suggestions; authorized organizer can review/override/lock without bypassing hard rules; members see only their own published pairing information; outcome/consent/feedback state is safe, audited, and tested.

Any matchmaking change must create a new ruleset/policy version where applicable and update algorithm docs, fixtures, contracts, this README, and before/after history.

