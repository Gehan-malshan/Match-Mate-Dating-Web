# MatchMate Deterministic Matchmaking Specification

This is the canonical specification for matchmaking. Implementations, tests, admin explanations, API contracts, and stored ruleset versions must agree with it.

## 1. Scope and principles

The MVP uses no machine-learning model. Matching is based on explicit user inputs, approved business rules, transparent weighted scoring, and deterministic event-wide optimization.

Principles:

- Hard user preferences and safety constraints are never overridden by score.
- Do not infer gender, sexuality, ethnicity, religion, health, or other sensitive traits.
- Use only approved attributes supplied for the stated purpose.
- Persist the exact snapshots and ruleset used so results are reproducible.
- Show generalized reasons without revealing private answers.
- Optimize the complete event, not each participant independently.
- Allow audited organizer intervention without erasing algorithm history.
- Store outcome data for product analysis, but do not silently train or deploy ML.

## 2. Matching inputs

The product owner must approve the exact questionnaire and visibility of every field.

| Category | Example inputs | Public? | Matching use |
|---|---|---|---|
| Identity/eligibility | Account status, age-at-event, permitted matching group | Limited/No | Hard filter |
| Partner preference | Accepted group/gender policy, preferred age range | No | Reciprocal hard filter |
| Relationship intention | Long-term, intentional dating, other approved values | Optional generalized | Hard or weighted per ruleset |
| Interests | Travel, music, films, fitness, books | Approved selections may be public | Weighted similarity |
| Personality | Introversion/extraversion and approved questionnaire dimensions | Optional/generalized | Compatibility matrix/distance |
| Lifestyle | Smoking, alcohol, activity, schedule, social style | Usually private/optional | Hard deal-breaker or weighted compatibility |
| Values/preferences | Explicit approved questions | Usually private | Weighted compatibility |
| Language | Sinhala, Tamil, English, others | Optional | Compatibility |
| Location | Broad city/region only | Broad value optional | Compatibility; never reveal exact location |
| Deal-breakers | Explicit user-selected constraints | No | Hard filter |
| Safety/relationship history | Blocks, restrictions, prior pairings | No | Hard filter/optimizer penalty |

Missing optional answers must not be treated as negative answers. A ruleset defines whether a dimension is omitted, scored neutrally, or makes the participant ineligible because a required question is missing.

## 3. Eligibility pipeline

Candidate pair `(A, B)` is eligible only when all active hard rules pass.

Required baseline filters:

1. Both accounts are active, verified as required, and profile-approved.
2. Both have confirmed bookings for the same event and meet event participation policy.
3. Neither booking is cancelled, expired, refunded in a disqualifying state, or excluded.
4. A satisfies B's partner preference and B satisfies A's partner preference under the approved reciprocity policy.
5. Each participant's age-at-event is inside the other's accepted range.
6. Neither participant has blocked the other.
7. No moderation/safety exclusion applies.
8. All approved hard deal-breakers pass in both directions.
9. They are not already locked to another partner in the same round.
10. Event-specific repeat-pair or organizer exclusion rules pass.

Every rejection stores a machine-readable reason code for diagnostics and aggregate organizer summaries. Do not reveal one participant's private rejection reason to another participant.

Suggested reason codes:

```text
ACCOUNT_INELIGIBLE
PROFILE_INCOMPLETE
BOOKING_NOT_CONFIRMED
PARTNER_PREFERENCE_MISMATCH
AGE_RANGE_MISMATCH
BLOCKED_RELATIONSHIP
SAFETY_EXCLUSION
DEAL_BREAKER_MISMATCH
ALREADY_ASSIGNED
REPEAT_PAIR_NOT_ALLOWED
EVENT_POLICY_MISMATCH
```

## 4. Compatibility scoring

Only eligible pairs are scored. Every component is normalized to `[0, 100]`.

Initial proposed weights requiring product validation:

| Component | Initial weight | Method |
|---|---:|---|
| Relationship intention | 25% | Exact/approved compatibility matrix |
| Personality compatibility | 20% | Approved matrix or normalized dimension distance |
| Shared interests | 20% | Weighted Jaccard similarity |
| Lifestyle compatibility | 15% | Per-question compatibility matrix |
| Values/preferences | 10% | Approved per-question compatibility |
| Language and broad location | 10% | Shared language plus coarse region compatibility |

Formula:

```text
total = 0.25 * relationship
      + 0.20 * personality
      + 0.20 * interests
      + 0.15 * lifestyle
      + 0.10 * values_preferences
      + 0.10 * language_location
```

Weights are configuration stored in a versioned ruleset, not scattered constants. The sum of active weights must equal 1.0 after handling omitted dimensions.

### Missing data

The ruleset defines one policy per dimension:

- `REQUIRED`: participant cannot enter the run without an answer.
- `IGNORE_AND_RENORMALIZE`: omit the dimension and renormalize remaining weights.
- `NEUTRAL`: use an explicitly approved neutral score.

Never use zero for missing data unless product policy explicitly defines missing as incompatibility.

### Interest similarity

Weighted Jaccard example:

```text
similarity = sum(weight of shared interests)
           / sum(weight of interests in the union)

component score = similarity * 100
```

Cap user-selected interests and validate taxonomy to prevent keyword stuffing. Important interests may have approved weights.

### Personality and lifestyle

Do not assume that identical personality always means compatible. Use a product-approved matrix for each trait/dimension. Store matrix version in the ruleset. Lifestyle questions may be hard deal-breakers or weighted dimensions, never both accidentally.

### Directional preferences

Some component scores may be directional. Compute `score(A prefers B)` and `score(B prefers A)` separately, then combine using an approved policy such as arithmetic mean or minimum. The minimum is more conservative when mutual satisfaction matters. Record both directional values where used.

## 5. Pairing optimization

For a two-group event, build a weighted bipartite graph:

- Vertices are eligible confirmed participants.
- An edge exists only for an eligible pair.
- Edge weight is the compatibility score adjusted by approved penalties/bonuses.

Use maximum-weight bipartite matching, such as the Hungarian algorithm, to maximize event-wide compatibility while assigning each participant at most once per round.

Do not choose each person's top candidate independently; that duplicates candidates and may produce a worse overall result.

### Optimization objectives

Use this priority order unless an ADR/ruleset changes it:

1. Satisfy all hard constraints.
2. Maximize number of valid pairings when appropriate.
3. Maximize total adjusted compatibility.
4. Prefer the solution with the higher minimum pair score.
5. Prefer fewer repeated prior pairings.
6. Apply deterministic ID-based tie-breaking.

If event policy supports non-bipartite groups, do not force those participants into a bipartite model. Introduce and document an approved general matching strategy.

### Penalties and thresholds

Possible approved controls:

- Minimum publishable compatibility threshold.
- Penalty for prior pairing within a configured event history window.
- Exclusion for a prior negative/safety response where policy permits.
- Organizer-declared event constraint.

Every penalty/threshold is versioned, explainable, and tested. It cannot override a hard exclusion.

### Unmatched participants

An unmatched participant is a valid result, not an algorithm failure. Store a safe reason category:

```text
NO_ELIGIBLE_CANDIDATE
GROUP_CAPACITY_IMBALANCE
BELOW_MINIMUM_SCORE
EVENT_CONSTRAINT
REMOVED_DURING_REVIEW
```

Organizer UI may show aggregate/internal operational reasons but must not reveal another member's private preference.

## 6. Matching run lifecycle

```text
DRAFT -> GENERATED -> UNDER_REVIEW -> LOCKED -> PUBLISHED
```

- A run references one event, participant snapshot set, ruleset version, optimizer version, seed/tie-break policy, creator, and timestamps.
- Generation creates immutable candidate/score output for that version.
- Organizer changes create override records; they do not overwrite original suggestions.
- Lock validates eligibility again and creates the immutable selected set.
- Publication controls member-visible output and notifications.
- A rerun creates a new version linked to the superseded run.
- A safety action may invalidate an unpublished run or flag a locked run for controlled incident handling; never silently rewrite history.

## 7. Organizer review and override

Organizer view may include:

- Participant codes, not unnecessary PII.
- Eligibility counts and safe exclusion summaries.
- Proposed pairs with total and component scores.
- Generalized reasons such as shared intention, interests, compatible lifestyle, shared language, and age-range pass.
- Unmatched participants and safe reason categories.

Override requirements:

- Organizer has scope for the event.
- Replacement pair passes all current hard constraints.
- Reason code and optional note are required.
- Original pair, replacement, actor, time, correlation ID, and ruleset/run version are audited.
- System displays the effect on total/minimum score and unmatched participants.
- Override cannot bypass block, safety, account, booking, or consent constraints.

## 8. Explainability

Store:

- Total score.
- Component scores and active weights.
- Directional scores where applicable.
- Safe explanation reason codes.
- Ruleset and algorithm version.
- Input snapshot IDs.

Example safe explanation:

```text
Compatibility: 88/100
- Same approved relationship intention
- Four shared interests
- Compatible lifestyle preferences
- Shared language
- Both satisfy preferred age range
```

Never expose: exact private deal-breakers, hidden lifestyle answers, other participant's preferred age limits, moderation data, or inferred traits.

## 9. Data model

Suggested Matchmaking-owned tables/aggregates:

| Table | Purpose |
|---|---|
| `ruleset` | Versioned weights, matrices, thresholds, missing-data and optimizer policy |
| `participant_snapshot` | Immutable event/run-safe copy of approved profile/preference inputs |
| `matching_run` | Event, version, status, ruleset/algorithm version, creator, timestamps |
| `candidate` | Pair eligibility result and internal reason codes |
| `compatibility_score` | Total, component/directional scores, explanations |
| `pairing_suggestion` | Optimizer-generated pair and rank/adjusted weight |
| `pairing_override` | Original/replacement, actor, reason, effect, audit |
| `locked_pairing` | Immutable selected pairing per run/round |
| `match_response` | Member continue/switch/interest response and deadline |
| `reveal_consent` | Versioned consent decision and policy context |
| `match_feedback` | Structured quality, comfort, safety, and product feedback |
| `inbox` / `outbox` | Idempotent integration |

Important constraints:

- Unique run version per event.
- Unique participant snapshot per run/participant.
- Canonical pair key orders participant IDs to prevent duplicates.
- At most one locked pairing per participant/event/round.
- One active response per participant/pairing/question version.
- Locked run history is append-only.

## 10. API outline

| Method | Endpoint | Authorization |
|---|---|---|
| GET | `/api/v1/events/{eventId}/matching-runs` | Event organizer/admin |
| POST | `/api/v1/events/{eventId}/matching-runs` | Event organizer/admin; event in matching state |
| GET | `/api/v1/matching-runs/{runId}` | Scoped organizer/admin |
| POST | `/api/v1/matching-runs/{runId}/overrides` | Scoped organizer/admin |
| POST | `/api/v1/matching-runs/{runId}/lock` | Scoped organizer/admin; idempotent command |
| POST | `/api/v1/matching-runs/{runId}/publish` | Scoped organizer/admin; approved state |
| GET | `/api/v1/matches/mine` | Authenticated eligible member |
| POST | `/api/v1/matches/{matchId}/response` | Pair participant; deadline/state checks |
| POST | `/api/v1/matches/{matchId}/reveal-consent` | Pair participant; policy/state checks |
| POST | `/api/v1/matches/{matchId}/feedback` | Pair participant; event/state checks |

Generation, lock, publish, response, and consent commands require idempotency protection.

## 11. Events

| Event | Minimum safe payload intent |
|---|---|
| `MatchingRunGenerated` | runId, eventId, runVersion, rulesetVersion, counts, generatedAt |
| `PairingsLocked` | runId, eventId, runVersion, pairing IDs, lockedAt; no private score inputs |
| `PairingsPublished` | runId, eventId, publish version/time |
| `MatchResponseRecorded` | matchId, responding account ID, response type/version, recordedAt |
| `MutualInterestEstablished` | matchId, eventId, policy version, establishedAt |
| `RevealConsentGranted` / `Revoked` | matchId, account ID, consent policy version, occurredAt |
| `MatchFeedbackSubmitted` | feedback ID, matchId, safe workflow flags; no free-text content in general event |

## 12. Deterministic algorithm outline

```text
function generate(eventId, rulesetVersion):
    participants = loadConfirmedEligibleParticipants(eventId)
    snapshots = createImmutableSnapshots(participants, rulesetVersion)

    candidates = []
    for each allowed unordered/directional pair (a, b):
        eligibility = evaluateHardRules(a, b, event, ruleset)
        persist eligibility and internal reason codes
        if eligibility passes:
            components = calculateComponents(a, b, ruleset)
            total = calculateWeightedTotal(components, ruleset)
            adjusted = applyApprovedOptimizationPenalties(total, history, ruleset)
            candidates.add(edge(a, b, adjusted, components))

    result = optimizer.solve(candidates, ruleset.objectives, deterministicTieBreak)
    persist suggestions and unmatched reason categories
    return immutable generated run
```

Generation must be idempotent for the same command key and must not produce a different result for identical snapshots, ruleset, algorithm version, and tie-break inputs.

## 13. Testing requirements

### Unit/property tests

- Every hard rule passes/fails at boundaries.
- Score components stay in `[0,100]`; total stays in range.
- Weight sum and missing-data renormalization.
- Pair key canonicalization and no self-pairs.
- Directional score combination.
- Determinism and stable tie-breaking.
- Hard constraints can never be overridden by score/penalty.

### Optimizer fixtures

- Known matrices with known optimum.
- Greedy-choice counterexample.
- Equal-score ties.
- Unequal group sizes.
- Disconnected candidate graph.
- Threshold-created unmatched participants.
- Repeat-pair penalties.
- Organizer replacement impact.

### Component/integration tests

- Snapshot and run transaction behavior.
- Confirmed/cancelled/restricted event consumption and inbox deduplication.
- Lock race and unique participant constraints.
- Outbox publication and replay.
- Ruleset immutability after use.

### Security/privacy tests

- Organizer event scope.
- Member can access only own published pairing.
- Private inputs/rejection reasons absent from member APIs/events/logs.
- Block/restriction prevents generation, lock, publication, response, and reveal.
- Override cannot bypass hard safety rules.

### Performance tests

Measure candidate generation, score calculation, optimizer time, memory, database writes, and run-lock behavior at expected event size and an approved growth margin. Store benchmark inputs and expected limits.

## 14. Rule changes and governance

Changing questions, eligibility, weights, matrices, thresholds, optimizer objectives, explanations, responses, or reveal behavior requires:

1. Product and privacy review.
2. A new immutable ruleset/policy version.
3. Before/after examples and impact analysis.
4. Deterministic regression fixtures.
5. Documentation and change-log update.
6. Compatibility/migration decision for active events and historical runs.
7. Monitoring for unmatched rate, score distribution, override rate, response outcomes, and safety signals.

Never modify a ruleset already used by a generated run.

