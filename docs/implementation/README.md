# MatchMate Implementation Guide

This guide defines the required development sequence, dependencies, deliverables, and completion criteria. It prevents separate agents or teams from building incompatible services in isolation.

## 1. Delivery strategy

Build vertical, demonstrable slices. A phase is not complete because backend handlers exist; it is complete when domain logic, persistence, contracts, frontend/admin behavior, events, tests, security, observability, and documentation work together.

Do not implement all databases first, all APIs second, and all frontend screens last. Complete a narrow end-to-end journey, then expand.

## 2. Global implementation rules

- Read `AGENTS.md` and canonical docs before work.
- Freeze the affected OpenAPI/AsyncAPI behavior before integrating multiple components.
- Keep one owner for every invariant and table.
- Use feature branches and focused pull requests.
- Make schema changes through service-owned migrations.
- Add idempotency and audit when the command is introduced, not after launch.
- Implement authorization in the owner service even if the gateway also checks it.
- Add telemetry and operational error states as part of the feature.
- Update before/after change documentation in the same pull request.

## 3. Dependency order

```text
Product decisions + contracts + platform conventions
                 |
          Account/Profile
                 |
              Event
            /       \
        Booking   Matchmaking prototype with fixtures
            |
          Payment
            |
     Confirmed participant integration
            |
        Matchmaking production flow
            |
     Notification + event experience
            |
     Moderation hardening + production readiness
```

Notification and basic moderation concerns begin early, but their complete service extraction follows the critical transaction flow.

## 4. Phase 0 — decisions, contracts, and engineering foundation

### Objectives

Remove irreversible ambiguity and establish conventions shared by every component.

### Required decisions

- Minimum age and age-at-event calculation.
- Community profile audience and field allow-list.
- Verification method and approval/moderation requirement.
- Partner preference and event matching-group policy.
- Matching questionnaire, public versus private answers, deal-breakers, and initial weights.
- Booking hold duration, capacity categories, one-booking policy, cancellation, refund, waitlist, and no-show rules.
- Event rounds, organizer override policy, reveal consent, and post-event responses.
- PayHere merchant/currency/sandbox and production callback/reconciliation policy.
- Retention, deletion/anonymization, audit, backup, and legal/safety policy.

Unresolved items must be recorded as open questions with owners. Use reversible safe defaults only when they do not create a hidden permanent policy.

### Engineering deliverables

- Repository handbook and agent instructions.
- API conventions and problem-details error format.
- Event envelope, naming, versioning, routing, retry, and DLQ conventions.
- Go service layout convention and frontend workspace convention.
- Database migration convention.
- Correlation/trace propagation convention.
- Local Docker Compose design.
- CI repository checks and future reusable workflows.
- Initial threat model and data classification.

### Exit criteria

- Required product decisions are accepted or explicitly assigned.
- Architecture and ownership are approved.
- Initial contracts can be reviewed without source code.
- Team can run one minimal service and dependency locally once code is introduced.

## 5. Phase 1 — Account/Profile vertical slice

### Business outcome

A user can register, verify, sign in, manage a safe community profile and private preferences, and access an authenticated member shell.

### Backend deliverables

- Account aggregate, credentials, roles/scopes, verification, refresh sessions, profile, visibility, preferences, interests, questionnaire answers, blocks, and account status.
- Registration, login, refresh, logout/revocation, self read/update, preferences replace, safe profile discovery, block/unblock, and deactivation endpoints.
- Password hashing, login throttling, normalized unique email, token signing/JWK publication, refresh reuse detection.
- Profile public allow-list and contact-information validation/quarantine.
- `AccountRegistered`, `ProfileApproved`, `ProfileHidden`, `AccountDeactivated`, and block-related facts as approved.

### Frontend deliverables

- Registration, verification, login/logout.
- Profile completion and private preference/questionnaire forms.
- Safe profile preview showing exactly what others can see.
- Authenticated route handling and session-expiry behavior.
- Block/report entry points; reporting may use an initial moderation module.

### Required tests

- Registration uniqueness and normalization.
- Password and refresh-session security.
- Authorization/ownership for every self/member/admin path.
- Public profile field leakage tests.
- Block symmetry and exclusion behavior.
- Profile validation including attempted contact details.
- Token expiry, rotation, revocation, and reuse.

### Exit criteria

- A verified test user can complete the journey through the web application.
- No private fields appear in community responses, tokens, logs, or events.
- Account lifecycle and events are idempotent and observable.

## 6. Phase 2 — Event vertical slice

### Business outcome

Organizers can manage events and members can discover eligible published events.

### Backend deliverables

- Event aggregate, organizer assignment, venue/broad location, schedule, registration window, price/currency, configured limits, ruleset reference, status, and version.
- Organizer/admin create, update, publish, open/close registration, cancel, and list operations.
- Member event discovery with cursor pagination and bounded filters.
- PostGIS only if approved location discovery requires it.
- Event lifecycle validation and optimistic concurrency.
- Event lifecycle and capacity-policy events.

### Frontend/admin deliverables

- Member event list/detail.
- Organizer event create/edit/publish/lifecycle controls.
- Clear display of registration state, price, policy, and availability disclaimer.

### Required tests

- Invalid date/lifecycle transitions.
- Organizer scope and stale update rejection.
- Discovery visibility and pagination.
- Price precision/currency.
- Cancellation behavior before bookings are integrated.

### Exit criteria

- A published/open event is discoverable by an eligible member.
- Only assigned organizers/admins can mutate it.
- Event states and changes are auditable and versioned.

## 7. Phase 3 — Booking and capacity

### Business outcome

A member can reserve an available ticket for a limited time without overselling the event.

### Backend deliverables

- Booking, seat hold, capacity allocation, immutable price/policy snapshot, attendance placeholder, inbox/outbox.
- Create hold, read own booking, list own bookings, cancel, expire hold, and internal authoritative usage query.
- Unique active booking per member/event.
- Atomic conditional allocation or short lock protecting last-seat behavior.
- Hold-expiry worker and idempotent release.
- `BookingPending`, `HoldExpired`, and `BookingCancelled` events.

### Frontend deliverables

- Reserve action, countdown/expiry state, booking details, cancellation, sold-out and conflict behavior.

### Required tests

- Concurrent last-seat attempts.
- Repeated request with same/different idempotency key.
- Duplicate booking, expired registration, ineligible account/event.
- Hold expiry and allocation release.
- Cancellation authorization and state transitions.
- Event capacity-policy version race.

### Exit criteria

- Load/concurrency tests prove capacity cannot be exceeded.
- Holds expire safely and repeat processing is harmless.
- Booking price/currency snapshot is immutable and ready for Payment.

## 8. Phase 4 — Payment and confirmed booking

### Business outcome

A member can pay through PayHere and receive exactly one correctly confirmed booking.

### Backend deliverables

- Payment, callback receipt/fingerprint, provider reference, refund, reconciliation item, inbox/outbox.
- Initiate payment using booking ID only; derive amount/currency server-side.
- Generate PayHere request/hash through an isolated provider adapter.
- Validate callback signature, merchant, order, exact amount/currency, booking relation, state, and replay.
- Idempotent `PaymentCompleted`/`PaymentFailed` events.
- Booking consumer transitions valid holds to confirmed and publishes `BookingConfirmed`.
- Pending/late/mismatch review and scheduled reconciliation.

### Frontend/support deliverables

- PayHere redirect/initiation, pending/success/failure/review states.
- Booking confirmation status polling or event-driven refresh.
- Restricted support/reconciliation view when approved.

### Required tests

- Valid, malformed, invalid-signature, wrong-merchant, wrong-order, wrong-amount, wrong-currency callbacks.
- Duplicate and reordered callbacks.
- Provider timeout and reconciliation.
- Broker outage after local commit.
- Hold expiry before late success.
- Consumer duplicate delivery and booking confirmation idempotency.
- Monetary precision and audit immutability.

### Exit criteria

- PayHere sandbox end-to-end journey passes.
- No client-controlled amount can affect stored/verified payment.
- Every successful callback leads to at most one payment completion and booking confirmation.
- Late and mismatch cases are recoverable and visible to operations.

## 9. Phase 5 — Matchmaking core

### Business outcome

The system creates reproducible, explainable event pairings from confirmed eligible participants and allows organizer review.

### Backend deliverables

- Versioned ruleset and questionnaire dimensions.
- Immutable participant/profile/preference snapshots.
- Hard eligibility filter with reason codes.
- Component scoring and weighted total.
- Maximum-weight pairing strategy and deterministic tie-breaking.
- Versioned matching run, pair suggestions, unmatched reasons, override audit, lock/publish.
- Confirmed/cancelled/restricted/block events update eligibility projections.

### Admin deliverables

- Participant eligibility summary.
- Generate run with ruleset version.
- View pair score and safe generalized reasons.
- Replace/override with required reason.
- Lock and publish immutable run.
- View unmatched participants and reason codes.

### Required tests

- Every hard constraint and combination boundary.
- Exact component and total score fixtures.
- Symmetry/asymmetry rules as specified.
- Optimizer global result, tie-breaking, unmatched behavior, repeat-pair penalties.
- Same snapshots/ruleset always produce same output.
- Block/restriction/cancellation immediately excludes unlocked runs.
- Locked history cannot be mutated.
- Organizer authorization and override audit.
- Performance at expected event size and larger safety margin.

### Exit criteria

- A confirmed event cohort produces a reproducible proposed run.
- Organizer can understand reasons, override, and lock.
- Private answers do not leak in explanations/events/logs.
- All algorithm behavior matches `docs/matchmaking/README.md`.

## 10. Phase 6 — Event interaction, responses, and notifications

### Business outcome

Members receive the right event information, complete structured responses, and progress through approved mutual-interest/reveal workflows.

### Deliverables

- Notification templates and delivery workers for registration, booking, payment, event reminder/change, pairing publication, and approved response outcomes.
- Member match/event surface with policy-limited information.
- Continue/switch/interest response uniqueness and deadline rules.
- Consent recording and reveal policy implementation.
- Feedback, comfort/safety rating, report path, and no-show handling.
- Delivery retries, suppression, provider failures, and notification audit.

### Required tests

- No notification leaks hidden identity/private preference.
- Duplicate events create no duplicate delivery unless policy permits.
- Response/reveal authorization, deadlines, and state transitions.
- Mutual consent and one-sided/no-consent behavior.
- Provider outage and retry/DLQ behavior.

### Exit criteria

- Complete register-to-event-to-response journey passes end to end.
- Safety controls are present at every interaction surface.

## 11. Phase 7 — Moderation, security, and production hardening

### Business outcome

MatchMate can be operated safely, audited, recovered, and deployed predictably.

### Deliverables

- Full moderation case/action/appeal workflow and restricted access.
- Profile content/media review and contact-information prevention.
- Threat-model closure, penetration/security review, dependency and image scanning.
- Gateway/WAF/rate-limit policies and managed secrets.
- Structured logs, metrics, traces, dashboards, SLOs, and alerts.
- PostgreSQL backups/PITR, RabbitMQ policies, DLQ replay, reconciliation, and restore drills.
- CI quality gates, signed images, staging, production approval, progressive rollout, smoke tests, rollback.
- Operational runbooks and on-call ownership.

### Required tests

- Authorization matrix and admin misuse scenarios.
- PII/log/event leakage and account deletion/anonymization.
- Credential stuffing, token theft/reuse, callback replay, upload abuse, injection/SSRF as applicable.
- Database/broker/provider failure and recovery.
- Load, soak, queue recovery, backup restore, and rollback rehearsal.

### Exit criteria

- Production-readiness checklist is complete.
- Approved SLO/RPO/RTO are demonstrated.
- Security/privacy/safety owners sign off.
- Staging deployment and restore/rollback rehearsals pass.

## 12. Standard service implementation layout

Each Go service should converge on a structure similar to:

```text
service-name/
|-- cmd/api/ or cmd/worker/       process entry points
|-- internal/domain/              aggregates, values, invariants
|-- internal/application/         use cases and ports
|-- internal/adapters/http/       handlers and DTO mapping
|-- internal/adapters/postgres/   repositories and generated queries
|-- internal/adapters/rabbitmq/   producers/consumers
|-- migrations/                   service-owned schema migrations
|-- tests/ or colocated tests
|-- Dockerfile
|-- go.mod
`-- README.md
```

Exact names may change through an ADR, but domain logic must not depend directly on transport/provider frameworks.

## 13. Standard feature workflow

For every story:

```text
Confirm requirement/open questions
-> identify owner and invariants
-> update contract/design/change baseline
-> implement domain rules with unit tests
-> implement migration/repository with component tests
-> implement HTTP/events with contract tests
-> implement UI/admin path where applicable
-> add observability, audit, and failure behavior
-> run end-to-end/security/performance tests as required
-> update canonical docs and before/after record
-> review and merge
```

## 14. Pull-request sizing and dependency rules

- Prefer one deployable behavior per pull request.
- Contract-first changes may be separate, but consumers must not deploy against unsupported required fields.
- Use backward-compatible expansion before producer/consumer behavior changes.
- Database contraction occurs only after old readers/writers are removed and observed.
- Feature flags must have an owner, removal condition, safe default, and documentation.
- A temporary shortcut must be recorded as debt with scope and removal criteria; never disguise it as target architecture.

## 15. Phase status tracking

When implementation starts, maintain this table in the same pull request as phase changes:

| Phase | Status | Owner | Evidence | Blocking decisions |
|---|---|---|---|---|
| 0 Foundation | In progress | Project team | Root Bun workspace and tested `frontend/apps/web` public landing slice | Product/policy decisions and contracts |
| 1 Account/Profile | Not started | Unassigned | — | Verification/profile policy |
| 2 Event | Not started | Unassigned | — | Event lifecycle/capacity policy |
| 3 Booking | Not started | Unassigned | — | Hold/cancel/waitlist policy |
| 4 Payment | Not started | Unassigned | — | PayHere/refund/reconciliation policy |
| 5 Matchmaking | Not started | Unassigned | — | Questionnaire/groups/weights/rounds |
| 6 Interaction/Notification | Not started | Unassigned | — | Reveal/response/channel policy |
| 7 Production hardening | Not started | Unassigned | — | Hosting/SLO/compliance/operations |

Allowed status values are `Not started`, `In progress`, `Blocked`, `In review`, and `Complete`. A phase is `Complete` only when its exit criteria and evidence are present.
