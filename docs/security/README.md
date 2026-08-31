# MatchMate Security, Privacy, and Safety Baseline

This baseline applies to every application, service, worker, contract, database, event, log, deployment, and operator workflow. Security and privacy controls are acceptance criteria, not post-launch enhancements.

## 1. Primary assets

- Account PII, credentials, sessions, verification evidence, and profile media.
- Private matchmaking preferences/questionnaire answers.
- Blocks, reports, moderation evidence/actions, and reveal consent.
- Booking, attendance, payment, refund, reconciliation, and audit data.
- PayHere credentials, JWT signing keys, database/broker/provider credentials.
- Pairing results before publication and member identity boundaries.
- Source code, CI/CD identity, container images, backups, and telemetry.

## 2. Main threats

- Credential stuffing, account enumeration, token theft/reuse, session fixation.
- Horizontal/vertical authorization bypass and organizer/admin misuse.
- Scraping, stalking, contact-information leakage, exact-location or identity exposure.
- Unsafe content/media, impersonation, harassment, report abuse, retaliation.
- Booking oversell, idempotency abuse, payment amount tampering, callback replay/forgery.
- Pairing manipulation, private-preference disclosure, safety restriction race.
- Injection, XSS, SSRF, unsafe upload, dependency/container/supply-chain compromise.
- Secrets in Git/logs/images, excessive telemetry labels, unprotected backups.
- Broker replay/schema abuse, event PII replication, missing audit/reconciliation.

Maintain a structured threat model with trust boundaries and mitigations as implementation appears.

## 3. Authentication baseline

- Hash passwords with approved Argon2id or bcrypt parameters and unique salts.
- Short-lived RS256/ES256 access tokens; publish JWKs and support overlapping key rotation.
- Rotating refresh sessions stored as non-reusable hashes; detect reuse and revoke family.
- Each frontend deduplicates concurrent refresh attempts into one in-flight request; one browser context must never submit the same rotating token concurrently.
- Validate issuer, audience, signature, expiry, issued/not-before times, and token version.
- Revoke sessions on password reset, deactivation, serious moderation action, and suspected compromise.
- Rate-limit registration, login, refresh, verification, password reset, and email-availability behavior.
- Reduce account enumeration through consistent responses/timing where practical.
- Multi-factor authentication is strongly recommended for organizer, moderator, finance, support, and admin roles before production; it is mandatory policy work for event-creation and matchmaking administrators.

## 4. Authorization baseline

- Gateway performs coarse route policy; owner service repeats domain authorization.
- Check authenticated subject, role/scope, resource ownership, organizer event assignment, state, and safety restrictions.
- Default deny; avoid role-only broad admin paths.
- Service identities have narrow audience/scopes and cannot impersonate users without explicit audited design.
- Privileged read access to PII/payment/safety data is audited.
- Test every endpoint against unrelated member/organizer and inappropriate support/moderator roles.

## 5. Privacy and visibility

- Unauthenticated internet cannot browse community profiles.
- Eligible authenticated community members receive an explicit safe-field DTO.
- Email, phone, DOB, exact address/location, credentials, verification evidence, payment data, private preferences, blocks/reports, and moderation data are never public.
- Use broad city/region, not precise coordinates, in member-facing data.
- Prevent contact details/social handles/URLs in profile content where feasible; quarantine for review.
- Private preferences may produce generalized explanation reasons but not reveal exact answers.
- Blocks apply to discovery, eligibility, pairing, reveal, and relevant notifications.
- Consent purpose/version/time is stored for reveal and other sensitive processing.
- Event discovery exposes broad location and approved catalog fields only; assigned organizer identifiers and exact venue names remain operational fields. Event mutations validate the Account-issued ES256 token again inside Event Service. Creation requires `admin`; existing-event changes enforce assigned-organizer/admin scope.
- Matching-run listing, generation, review, override, lock, and publication require `admin` inside Matchmaking Service. Organizer accounts are denied even if the frontend is bypassed. Member match actions remain pairing-participant scoped.

## 6. Payment controls

Booking now derives price/currency from Event, stores the immutable snapshot, and exposes it only after Account-token verification and subject ownership checks. Capacity updates, Payment inbox insertion, confirmation, and Booking outbox facts use local transactions. Late payment never recreates released capacity.

The first executable Payment slice validates Account-issued ES256 tokens for member operations, forwards the token to Booking's constrained snapshot check, compares ownership locally, verifies PayHere callbacks without member authentication, fingerprints callback receipts, and omits account/provider identifiers from member DTOs. Gateway/WAF callback rate limits and production secret-manager wiring remain deployment requirements.

- Payment Service derives amount/currency from immutable Booking snapshot.
- Verify PayHere signature plus merchant, order, amount, currency, booking relation, and state.
- Unique callback/provider fingerprint prevents replay.
- Do not store card data; minimize/sanitize provider callback storage/logging.
- Use TLS, WAF/request-size/rate controls, and provider IP allow-list only when officially supported.
- Reconcile pending, duplicate, mismatch, late, refund, and unknown outcomes.
- Separate finance/support scopes and audit all manual decisions.

### Notification controls

- Notification consumes only explicitly bound minimum-safe facts and stores recipient account IDs, not email/phone destinations from events.
- Template variables are allow-listed; unknown variables and unsafe multiline subjects fail before provider delivery.
- Account deactivation creates suppression and stops pending/retry deliveries.
- Inbox and business-key uniqueness make duplicate broker delivery safe.
- Member feed endpoints validate Account-issued ES256 issuer/audience/expiry, derive ownership from `sub`, accept no caller-selected recipient, and return not found for absent or other-member item IDs.
- Member DTOs expose only safe rendered template text, category, time, read state, and allow-listed application paths; provider references/errors and source payloads remain private.
- The development sink logs no recipient destination or rendered message body and is rejected outside development/test.
- Exact-origin CORS permits the member application in local development. Production edge rate limits and TLS remain deployment requirements.
- Production email delivery requires an approved provider, credentials from a secret manager, authenticated constrained Account contact resolution, provider idempotency, timeout/rate controls, preference/legal-category policy, and sanitized provider errors.

## 7. Application and upload security

- Validate at trust boundaries with size, type, range, enum, and semantic checks.
- Parameterized SQL; no string-built queries from untrusted input.
- Escape/sanitize displayed text; define approved rich-content policy or use plain text.
- Protect against SSRF by blocking arbitrary server-side URL fetches and validating provider destinations.
- Upload through private storage with randomized object keys, content/type verification, size/dimension limits, malware scan, moderation state, and controlled delivery URLs.
- Configure CORS narrowly. Apply CSRF controls if browser credentials are cookie-based.
- Set appropriate security headers and avoid exposing stack traces/internal IDs.

## 8. Secrets and cryptography

- Secrets live in an approved secret manager or ignored local environment; never Git, images, Compose defaults, logs, or docs.
- Separate secrets per environment/service and rotate with runbooks.
- Encrypt transit with TLS and storage/backups using provider controls; evaluate field encryption for highly sensitive data.
- Use approved randomness and constant-time comparison where applicable.
- Treat provider-required legacy hashes only as provider compatibility, not general integrity primitives.
- CI cloud access should use short-lived OIDC identity rather than long-lived keys when supported.

## 9. Logging, telemetry, and audit

Never log passwords, tokens, secrets, raw identity evidence, full callback payloads, private questionnaire answers, exact location, report evidence, or unnecessary PII.

Use sanitized IDs/hashes and structured fields. Metrics labels must remain low-cardinality and contain no personal data.

Business audit records include actor, target, action, prior/new state, reason, time, correlation, and source. Audit access is restricted and itself audited where sensitive.

## 10. Moderation and safety

The initial Moderation service repeats Account JWT validation, requires UUID account subjects, permits reporting to member/organizer/moderator/admin roles, restricts case/assignment/status/action/decision APIs to moderator/admin roles, binds appeals to the affected account ID, and never returns reporter identity or evidence through owner-report responses. Evidence is bounded reference metadata and hashes. Privileged case reads, assignment, investigation/dismissal, action, appeal, reversal, and expiry are audited; enforcement events omit descriptions, reporter identity, evidence, and private notes.

The current per-process report limiter is defense-in-depth for local/single-replica use. Production requires gateway plus distributed rate enforcement. Downstream services must consume enforcement facts idempotently before cross-domain exclusion is complete.

- Block/report available from relevant profile/match surfaces.
- Risk triage and temporary action available for urgent safety cases.
- Moderation actions can hide profiles, suspend accounts, exclude events/matching, invalidate unpublished pairings, and prevent reveal.
- Reporter identity is protected from the target.
- Organizers cannot access matching-run controls; administrators still cannot override block, safety, account, booking, reciprocal preference, age, deal-breaker, repeat-pair, or hard consent controls.
- Event-day check-in, emergency escalation, venue conduct, evidence, law-enforcement/legal request, and support SLA require approved operational policies before launch.

## 11. Data lifecycle

- Define purpose/classification/retention for every table, event, object, log, trace, backup, and export.
- Deactivate/revoke immediately, then delete/anonymize according to approved retention.
- Preserve only required financial/safety/audit records with restricted access.
- Handle deletion across projections, object storage, analytics, events, and restored backups.
- Do not use production data in development/test without approved irreversible anonymization.

## 12. CI/CD and supply chain

- Minimal workflow permissions and protected environments.
- Pin third-party actions to immutable commit SHAs.
- Secret, dependency, license, Go vulnerability, frontend audit, image, and IaC scans.
- Generate SBOM; sign/attest release images; deploy immutable SHA-tagged artifacts.
- Review dependency provenance and avoid unnecessary packages.
- Production requires approval, smoke tests, health verification, and rollback criteria.

## 13. Security verification checklist

- [ ] Threat model and data-flow/trust boundaries updated.
- [ ] Authentication/session and authorization matrix tests pass.
- [ ] APIs/events/logs/notifications contain no restricted fields.
- [ ] Booking/payment idempotency and callback replay tests pass.
- [ ] Match/block/restriction/reveal races pass.
- [ ] Upload/input/XSS/injection/SSRF/rate-limit tests apply where relevant.
- [ ] Secrets/dependencies/images/IaC scans pass with accepted exceptions documented.
- [ ] Retention/deletion/backup effects are defined.
- [ ] Audit, alerts, incident/runbook, and rollback are ready.
- [ ] Security/privacy/safety changes have complete before/after documentation.
