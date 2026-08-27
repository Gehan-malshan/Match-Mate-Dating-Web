# Moderation and Safety Service

Owns reports, moderation cases, safety actions, evidence references, appeals, and restricted audit records when extracted as a standalone service.

This capability may begin as a well-isolated module if a separate deployment would add unnecessary early operational cost.

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

```text
POST /api/v1/reports
GET  /api/v1/reports/mine
GET  /api/v1/moderation/cases
GET  /api/v1/moderation/cases/{caseId}
POST /api/v1/moderation/cases/{caseId}/assign
POST /api/v1/moderation/cases/{caseId}/actions
POST /api/v1/moderation/actions/{actionId}/appeals
POST /api/v1/moderation/appeals/{appealId}/decision
```

Moderator/admin endpoints require least-privilege scopes, reason, and audit. Reporter identity/evidence is not included in reported-member responses.

## Proposed data

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

## Required tests

- Report validation, duplicate/abuse/rate-limit behavior.
- Moderator/support/organizer/member authorization matrix.
- Reporter/evidence/private-note leakage tests.
- Action effective/expiry/reversal and propagation.
- Match generation/lock/reveal safety races.
- Audit append-only behavior and privileged-view audit.
- Retention/legal hold/deletion interactions.
- Event duplicate/retry and downstream unavailable behavior.

## Completion criteria

Reports are triaged and resolved through audited least-privilege workflows; active actions consistently restrict all affected domains; reporters/evidence remain protected; appeals preserve history; safety operations have alerts and runbooks.

Update this README, security/architecture/data/testing docs, event contracts, runbooks, and change history whenever behavior changes.

