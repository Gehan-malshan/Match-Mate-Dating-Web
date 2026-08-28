# ADR-0002: Deterministic bipartite matchmaking prototype

- **Status:** Accepted for prototype
- **Date:** 2026-08-28
- **Owners:** MatchMate project team
- **Related change:** Matchmaking Phase 5 prototype vertical slice
- **Supersedes:** None
- **Amended by:** ADR-0003 for operator authorization; the algorithm decision remains active

## Context

MatchMate requires transparent event-wide pairing without machine learning. The canonical specification requires reciprocal hard constraints, weighted component scores, deterministic optimization, immutable snapshots, privileged review, and privacy-safe member results. Booking and production event-consumer integrations are not yet implemented, so a fixture-backed projection is required to validate the engine and workflow independently.

## Decision drivers

- Hard safety and preference rules must never be overridden by a score.
- Identical approved inputs and ruleset must produce identical pair participants and scores.
- Event-wide quality must be optimized instead of greedily selecting each participant's top candidate.
- Private questionnaire and rejection details must remain inside Matchmaking.
- The prototype must evolve into event-driven production integration without changing service ownership.

## Considered options

### Greedy top-candidate selection

Simple, but can duplicate candidates and produce a lower total event result. Rejected.

### Machine-learning ranking

Would be difficult to explain and lacks approved training data, governance, and fairness evidence. Rejected for the MVP.

### Deterministic weighted bipartite optimization

Creates edges only after hard rules pass, calculates versioned component scores, and uses maximum-weight assignment with deterministic tie-breaking. Selected.

## Decision

The prototype uses exactly two approved event groups, six normalized weighted components from immutable `prototype-v1`, and an O(n³) Hungarian assignment. Edge weight first maximizes valid pairing count, then total compatibility score, then deterministic lexicographic order. Optional missing dimensions are ignored and active weights are renormalized. A minimum score of 45 controls selectable edges.

Development participant projections are service-owned fixtures representing future consumed facts. They are not Account, Event, Booking, or Moderation source records and Matchmaking never queries those databases.

## Consequences

### Positive

- Reproducible, explainable pairings and optimizer regression fixtures.
- Global event optimum rather than locally greedy choices.
- Clear migration path from fixtures to inbox-deduplicated facts.
- Private inputs stay within Matchmaking-owned storage.

### Negative and trade-offs

- Prototype supports two groups only; non-bipartite policies require a new approved strategy.
- Fixture eligibility is not live Booking/Moderation authority.
- Hungarian optimization handles the expected event scale but still requires measured production limits.
- Product approval of questionnaire semantics and compatibility matrices remains open.

## Compatibility and migration

The v1 REST and event contracts are additive new surfaces. Production integration replaces fixture projection writes with idempotent consumers while retaining snapshot, ruleset, run, candidate, selection, lock, response, consent, audit, and outbox ownership. A new algorithm semantic requires a new optimizer/ruleset version; historical runs are never rewritten.

Rollback stops Matchmaking API and preserves its independent PostgreSQL volume. No other service database is changed.

## Security, privacy, safety, and operations

Account ES256 tokens are validated inside Matchmaking. As amended by ADR-0003, listing, generation, review, override, lock, and publication are administrator-only; organizer accounts cannot access these operations. Member queries are participant-scoped and return only published partner codes plus generalized reasons. Blocks, safety exclusions, deal-breakers, questionnaire inputs, and exact locations are excluded from member DTOs and outbox payloads. Lock revalidates hard rules; administrator overrides cannot bypass them.

Production requires rate limiting, inbox and outbox relay operations, telemetry, backup/restore, retention, and incident runbooks.

## Verification

Unit fixtures cover hard exclusions, missing-data renormalization, deterministic global optimization, unmatched behavior, and invalid rulesets. PostgreSQL migration/seed, authenticated lifecycle API, member-safe publication, Compose configuration, Go tests, and Go vet are required prototype evidence.

## Documentation updates

`docs/matchmaking`, `docs/data`, `docs/implementation`, service README, OpenAPI, AsyncAPI, Compose documentation, root README, and pull-request before/after record.
