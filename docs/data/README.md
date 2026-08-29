# MatchMate Data Architecture

This guide defines service data ownership, proposed schemas, constraints, consistency, migrations, retention, and recovery. Table names are proposed until service migrations are accepted; ownership and invariants are architecture decisions.

## 1. Non-negotiable ownership rules

- Each service owns one logical PostgreSQL database, database user, migrations, backups, and restore procedure.
- No service reads or writes another service's tables, schema, replica, or migration history.
- Cross-service references are scalar IDs and are not database foreign keys.
- Cross-service joins belong in explicit read projections, APIs, or offline analytics—not application SQL.
- A data copy must have a defined source, purpose, refresh mechanism, staleness expectation, retention, and deletion behavior.
- Service events contain minimum safe facts, not database rows.
- Money uses fixed-precision decimals and explicit currency.
- Timestamps use UTC; business venue time zone is stored explicitly where required.
- Historical decisions use immutable snapshots and version references.

## 2. Database ownership map

| Owner | Core data | Authoritative invariants |
|---|---|---|
| Account/Profile | Accounts, credentials, roles, sessions, verification, profile, preferences, visibility, interests, blocks | Email uniqueness, credential/session security, source profile/preferences, account/block status |
| Event | Event, location, schedule, price, configured capacity/policy, organizer, lifecycle | Event state, configured limits, price definition, registration window |
| Booking | Booking, hold, allocation, price/policy snapshot, attendance, inbox/outbox | No oversell, one active booking, hold expiry, confirmed participation |
| Payment | Payment, provider callback, refund, reconciliation, audit, inbox/outbox | Trusted amount/currency, callback uniqueness, payment/refund state |
| Matchmaking | Ruleset, snapshots, candidates, scores, runs, suggestions, overrides, locked pairs, responses, consent, feedback | Eligibility/scoring reproducibility, unique pairing/response, immutable run history |
| Notification | Template, delivery, attempt, suppression/channel preference | Delivery idempotency, retry history, suppression |
| Moderation | Report, case, evidence reference, action, appeal, safety audit | Enforcement state, restricted evidence/audit, appeal history |

## 3. Account/Profile database

Suggested tables:

| Table | Important fields/constraints |
|---|---|
| `account` | `account_id`, normalized unique email, status, verification state, created/updated/deactivated times, version |
| `credential` | account ID unique, password hash parameters, changed time; never store plaintext |
| `role_assignment` | account, role, scope, grant/revoke audit |
| `refresh_session` | token-family ID, hash, issued/expiry/revoked/reuse times, device metadata minimized |
| `profile` | account unique, nickname, DOB private, derived/public age policy, broad location, bio, moderation/visibility state, version |
| `profile_interest` | profile, controlled interest taxonomy ID, importance where approved |
| `matching_preference` | accepted partner policy, age range, intention, location/language preferences, version |
| `questionnaire_answer` | account, question/version, approved typed answer, privacy class |
| `profile_media` | private object key, moderation state, ordering, content metadata; no public bucket URL |
| `block` | blocker, blocked, created/revoked time; unique active pair direction |
| `outbox` | event envelope and publication state |

Constraints and privacy:

- Normalize email before uniqueness comparison.
- Store date of birth privately; derive age-at-event, never expose DOB.
- Maintain an explicit public/community DTO allow-list.
- Treat preference/questionnaire answers as private by default.
- Block queries must support either-direction exclusion efficiently.
- Deactivation revokes sessions transactionally and starts retention-aware cleanup.

## 4. Event database

The first executable slice implements an Event-owned `event` aggregate table, `event_audit`, and transactional `outbox`. Separate location and policy-history tables remain planned; until they are introduced, broad/exact location fields and policy version references live on the aggregate. Booking still owns all consumed-capacity data.

Suggested tables:

| Table | Important fields/constraints |
|---|---|
| `event` | ID, organizer ID scalar, name/description, status, venue time zone, start/end, registration open/close, price decimal, currency, version |
| `event_location` | event ID unique, approved venue fields, broad public location, optional PostGIS geography |
| `event_capacity_policy` | event, total/group limits, policy version, effective time |
| `event_matching_policy` | event, matching ruleset reference, round/group configuration, version |
| `outbox` | lifecycle and policy events |

Constraints:

- End must be after start; registration close before event start.
- Published event fields change only under explicit lifecycle rules.
- Price/currency changes create new versions and do not mutate Booking snapshots.
- Event cannot hard-delete when downstream business history exists; cancel/archive instead.

## 5. Booking database

Booking migration 1 implements `booking`, `capacity_allocation`, `idempotency_record`, `inbox`, and `outbox`. Migration 2 adds the cancellation timestamp used by idempotent unpaid-hold cancellation. A partial unique index prevents more than one active account/event booking, and a conditional allocation update prevents held plus confirmed capacity from exceeding the immutable configured limit. Attendance tables remain planned.

Suggested tables:

| Table | Important fields/constraints |
|---|---|
| `booking` | ID, account/event scalar IDs, state, idempotency key, policy version, amount/currency snapshot, created/confirmed/cancelled/expired times, version |
| `seat_hold` | booking unique, event, allocation category, expiry, released time |
| `capacity_allocation` | event/category unique, configured policy version, held count, confirmed count, version |
| `attendance` | booking unique, check-in/no-show state, actor/time/audit |
| `inbox` | unique event ID per consumer/handler |
| `outbox` | booking facts |

Constraints:

- Unique active booking for account/event.
- Atomic conditional update or short row lock protects allocation.
- Held + confirmed cannot exceed approved capacity for the versioned policy.
- Price/currency and relevant policy are immutable after hold creation.
- Expiry/release is idempotent.
- Confirmation requires allowed state and verified payment fact/approved override.

## 6. Payment database

Migration 1 implements `payment`, `provider_callback`, `payment_audit`, `idempotency_record`, `inbox`, and `outbox`. Refund and reconciliation tables remain planned until finance policy is approved. The current migration is additive with no backfill; production rollback retains the isolated Payment database for financial evidence and uses forward recovery.

Suggested tables:

| Table | Important fields/constraints |
|---|---|
| `payment` | ID, booking unique active relation, order ID unique, amount/currency, state, provider, created/updated/completed time, version |
| `provider_callback` | provider event/payment ID or fingerprint unique, order, received time, verification result, sanitized metadata hash/reference |
| `refund` | payment, amount/currency, reason, state, provider reference, requested/completed times |
| `reconciliation_item` | payment/order, discrepancy type, status, owner, notes/audit, resolution |
| `payment_audit` | append-only state transition and verification decision metadata |
| `inbox` / `outbox` | idempotent integration |

Constraints:

- Store exact server-derived amount/currency before provider initiation.
- Callback uniqueness is persisted before state transition.
- Valid duplicate callbacks do not republish completion.
- Preserve immutable callback/audit evidence according to approved retention.
- Do not store card data or provider credentials.
- Refund cannot exceed captured amount minus completed refunds.

## 7. Matchmaking database

See [`../matchmaking/README.md`](../matchmaking/README.md) for algorithm behavior.

| Table | Important fields/constraints |
|---|---|
| `ruleset` | immutable version, status, weights, matrices, thresholds, missing-data/optimizer policy, approval metadata |
| `participant_snapshot` | run, participant, approved inputs, source versions, privacy classification, snapshot time; unique per run/participant |
| `matching_run` | event, run version unique, status, ruleset/algorithm version, creator, superseded link, timestamps |
| `candidate` | canonical pair key, eligibility, internal reason codes; unique per run/pair |
| `compatibility_score` | candidate unique, total, components/directions, safe reasons |
| `pairing_suggestion` | run, pair, adjusted weight, optimizer output order |
| `pairing_override` | run, original/replacement, actor, reason, before/after effect, time |
| `locked_pairing` | run/event/round, canonical pair, locked time; participant uniqueness enforced |
| `match_response` | pairing, account, question/policy version, response, time; unique active response |
| `reveal_consent` | pairing, account, policy version, decision/revocation, time |
| `reveal_consent_history` | pairing, account, increasing decision version, grant/revoke decision, policy version, idempotency key, time; append-only |
| `match_feedback` | pairing/account, structured ratings/flags, restricted free-text reference where approved |
| `inbox` / `outbox` | idempotent integration |

Rulesets and used snapshots are append-only. Never update history to make a new algorithm appear to have produced an old result.

The implemented prototype adds `event_scope` and `participant_projection` as Matchmaking-owned read models plus `pairing_selection`, `locked_participant`, `audit_log`, and transactional `outbox`. Development seed writes fixture projections only; production writers will be inbox-deduplicated consumers of minimum-safe facts. No Matchmaking query joins another service database.

## 8. Notification database

Migration 1 implements `notification_template`, `notification_delivery`, `notification_delivery_attempt`, `notification_preference`, `notification_suppression`, and `notification_inbox`. The first slice stores recipient account IDs and minimum source identifiers only; it does not replicate contact destinations, profiles, payment details, or safety evidence. Templates currently use the development channel until an approved provider/contact-resolution design is accepted.

| Table | Important fields/constraints |
|---|---|
| `template` | name, locale, channel, version, status, approved variables/content |
| `delivery` | business idempotency key unique, recipient account/channel, template version, state, scheduled time |
| `delivery_attempt` | delivery, attempt number, provider reference/status, sanitized error, timing |
| `notification_preference` | account/channel/category, allowed/suppressed, source/time |
| `suppression` | destination hash/account, reason, expiry/indefinite, audit |
| `inbox` / `outbox` | `notification_inbox` implements event consumption; a Notification outbox is deferred until safe delivery facts have an approved consumer |

Delivery state is `PENDING`, `PROCESSING`, `RETRY_SCHEDULED`, `DELIVERED`, `SUPPRESSED`, `PERMANENTLY_FAILED`, or `DEAD_LETTERED`. A unique business key prevents repeated event/template/recipient delivery, worker leases recover abandoned processing, and every completed provider attempt is append-only. Do not replicate full profiles or payment payloads. Resolve only approved recipient/channel data through a constrained mechanism and retention policy.

## 9. Moderation database

| Table | Important fields/constraints |
|---|---|
| `report` | reporter, target type/ID, category, description/reference, event, state, created time |
| `moderation_case` | case ID, risk/severity, owner, state, SLA times, restricted notes |
| `evidence_reference` | case, private object/reference, type, integrity metadata, retention/legal-hold state |
| `moderation_action` | case, target, action, scope, reason, effective/expiry time, actor; append-only |
| `appeal` | action/case, appellant, state, decision actor/reason/time |
| `moderation_audit` | append-only privileged views and changes |
| `outbox` | safe restriction/action facts only |

Reporter identity and evidence are more restricted than general organizer data.

## 10. Transaction and integration patterns

### Local ACID

Use one database transaction for an aggregate state change plus its outbox record.

### Optimistic concurrency

Use version columns for Event, Booking, Payment, matching run review, and other concurrently edited aggregates. Reject stale updates with a stable conflict error.

### Capacity allocation

Use an atomic conditional update or short pessimistic lock on one event/category allocation row. Never count bookings and then insert without a lock/constraint.

### Outbox

- Insert event record with business state.
- Relay unpublished records to RabbitMQ.
- Mark published only after broker confirmation.
- Alert on outbox age and relay failure.
- Make relay crash/restart safe.

### Inbox

- Unique key on consumer + event ID.
- Insert inbox and apply business change in one transaction.
- A duplicate is acknowledged without applying the change again.
- Business operation also enforces idempotent state transitions.

### Read projections

Projections may combine safe facts across services for event discovery/admin dashboards. They are explicitly non-authoritative. Critical commands revalidate with the owner.

## 11. Indexing guidance

Indexes follow measured queries, but baseline candidates include:

- Account: normalized email unique, status/verification, profile visibility, blocks both directions.
- Event: status + registration/start time, organizer, type, searchable fields, PostGIS geography if used.
- Booking: account/event active unique, event/state, hold expiry, allocation event/category.
- Payment: order unique, provider ID/fingerprint unique, booking, state/update time, reconciliation status.
- Matchmaking: event/run version unique, run/status, canonical pair unique, participant locked lookup, response pairing/account.
- Notification: state/scheduled time, business idempotency key, provider reference.
- Moderation: target/state, severity/SLA, assigned owner, action target/effective state.

Avoid indexing sensitive plaintext merely for convenience; evaluate encryption/search implications.

## 12. Migration policy

Every migration must include:

- Owning service and schema version.
- Purpose and affected behavior.
- Forward steps and expected lock/runtime impact.
- Backfill strategy and resumability.
- Validation queries/metrics.
- Compatibility window for old/new application versions.
- Rollback or forward-recovery plan.
- Backup/PITR impact.
- Retention/deletion effect.

For breaking changes use expand/migrate/contract:

1. Add backward-compatible schema.
2. Deploy writers/readers that support both forms.
3. Backfill and reconcile.
4. Switch authoritative reads/writes.
5. Observe and verify.
6. Remove old fields in a later deployment.

Never drop or repurpose data while a deployed version may still depend on it.

## 13. Seed and test data

- Production data is never copied into local/test environments without approved anonymization.
- Seed data contains fictional identities and no real contact/payment/safety evidence.
- Deterministic matching fixtures are versioned and expected outputs documented.
- Tests create isolated records and clean up through disposable databases/containers rather than deleting shared environments.

## 14. Retention, deletion, and audit

Exact periods require legal/product approval. Each table/event/object must be classified as operational, financial, safety, audit, analytics, or transient.

Account deletion flow:

1. Disable account and revoke sessions.
2. Publish `AccountDeactivated` with minimum identifiers.
3. Stop discovery, new booking, matching, reveal, and notifications as required.
4. Delete/anonymize service-owned data according to purpose and retention.
5. Preserve legally required financial/safety/audit records with access restrictions.
6. Track erasure through projections, object storage, analytics, and event-retention policy.
7. Document backup expiry and restoration re-deletion process.

Audit logs are tamper-resistant and access-controlled, but not an excuse to retain unlimited PII.

## 15. Backup and recovery

- Independent encrypted backups and PITR per service database.
- Backup/restore credentials separated from application credentials.
- Document dependencies and restoration order without creating cross-database transactions.
- Quarterly restore drills verify data, constraints, migrations, outbox/inbox, and application readiness.
- Reconcile booking/payment state and rebuild disposable projections after recovery.
- Planning target: RPO no more than 5 minutes, RTO no more than 30 minutes, pending approval and evidence.

## 16. Data-change checklist

Before merging a data change, verify:

- [ ] One service clearly owns it.
- [ ] No cross-service database access is introduced.
- [ ] Migration works from empty and latest production-like schema.
- [ ] Compatibility/backfill/validation/rollback are documented.
- [ ] Constraints enforce critical invariants.
- [ ] Index and query plans are appropriate.
- [ ] PII classification, encryption, logs/events, retention, and deletion are addressed.
- [ ] Component, concurrency, and recovery tests are updated.
- [ ] Service README, this guide, contracts, and change log are updated.
