# Change Management and Documentation Synchronization

Every architecture-significant or user-visible change must be understandable from repository history without external conversation context. This process applies to human and agent-authored work.

## 1. Required artifacts for a change

Depending on impact, update all applicable artifacts in the same pull request:

- Implementation and tests.
- OpenAPI and AsyncAPI contracts.
- Database migrations and migration notes.
- Canonical architecture/data/matchmaking/testing/security documentation.
- Affected service/application/infrastructure README.
- ADR for significant architecture decisions.
- Operational runbook and configuration reference.
- `CHANGELOG.md` before/after entry.
- Pull-request template.

## 2. When a change-log entry is required

Required for:

- New or changed user/organizer/support/moderator behavior.
- API, event, database, state-machine, validation, authorization, privacy, or retention changes.
- Matchmaking question, filter, weight, matrix, optimizer, explanation, response, or reveal changes.
- Payment/booking logic and provider integration changes.
- Service ownership, dependency, deployment, CI/CD, observability, or recovery changes.
- Significant defect fixes whose old behavior matters operationally.

Optional for typo-only documentation edits with no semantic effect.

## 3. Change entry lifecycle

1. Add an entry under **Unreleased** before implementation or as the first change.
2. Describe the baseline in **Before** using observable behavior and architecture.
3. Describe the intended and then actual behavior in **After**.
4. Identify all affected components/contracts/data/security/operations.
5. Record migration, compatibility, rollout, rollback, and verification.
6. Link the pull request, issue, ADR, and contract versions when available.
7. At release, move entries to a dated version section without rewriting their historical meaning.

## 4. Required entry format

```markdown
### CHG-YYYYMMDD-NNN — Concise outcome

- **Status:** Proposed | In progress | Released | Reverted
- **Date:** YYYY-MM-DD
- **Owner:** team/person
- **Issue/PR:** link or pending
- **Affected:** applications, services, APIs, events, tables, infrastructure
- **Decision/ADR:** link or Not required

#### Before

Describe exact previous behavior, ownership, data, and limitations.

#### After

Describe exact new behavior, ownership, data, and user/operational outcome.

#### Reason

Why the change is required and alternatives considered when relevant.

#### Compatibility and migration

Contract compatibility, schema expansion/backfill/contraction, active data, historical behavior.

#### Security, privacy, and moderation impact

Classification, authorization, exposure, retention, consent, audit, abuse/safety effects.

#### Deployment and rollback

Order, flags, environment/config, monitoring, rollback or forward-recovery steps.

#### Verification

Tests, reconciliation queries, metrics, manual checks, and acceptance evidence.

#### Documentation updated

List every canonical and local document updated.
```

## 5. Before/after quality rules

Bad:

```text
Before: old API
After: improved API
```

Good:

```text
Before: POST /payments/initiate accepted bookingId, amount, and currency from the browser.
The Payment Service persisted the client amount before creating the PayHere request.

After: POST /payments/initiate accepts bookingId only. Payment loads the immutable amount and
currency snapshot from the authorized booking, persists it, and rejects ineligible/expired holds.
```

The description must let a reviewer identify compatibility, migration, tests, and rollback consequences.

## 6. Architecture Decision Records

Add an ADR when a change affects:

- Service boundaries or data ownership.
- Synchronous versus asynchronous communication.
- Consistency, transaction, caching, or messaging strategy.
- Authentication, authorization, privacy, retention, or encryption strategy.
- Major technology/provider/platform choice.
- Public compatibility/versioning policy.
- Matchmaking algorithm class or fairness/safety policy.
- Operational availability, recovery, or deployment model.

An ADR includes status, context, decision, alternatives, consequences, migration, and superseded decisions. Do not use an ADR as a substitute for updating the current-state handbook.

## 7. Documentation review checklist

- [ ] Current behavior is accurate after merge.
- [ ] No future feature is described as already implemented.
- [ ] Assumptions/open questions are labeled.
- [ ] Service ownership is consistent across architecture, data, and service README files.
- [ ] Endpoint/event examples match versioned contracts.
- [ ] State models and failure paths match implementation.
- [ ] Security/privacy/moderation effects are explicit.
- [ ] Tests and operational signals prove the described outcome.
- [ ] Before/after entry can be understood without the original chat/ticket.

## 8. Reverts and partial rollouts

- A revert gets its own entry or updates the original status to `Reverted` with a linked entry explaining resulting behavior.
- Do not delete the original history.
- If rollout is partial, state which environment/cohort/version has each behavior.
- If database changes cannot be rolled back, document forward recovery before deployment.
- Update canonical docs to describe the active production target and mark temporary compatibility behavior as deprecated with removal criteria.

## 9. Agent completion rule

Before an agent declares a task complete, it must compare changed files with the impact table in `AGENTS.md`, update every required document, complete the change record, and report any unresolved open question. “Code compiles” is not completion.

