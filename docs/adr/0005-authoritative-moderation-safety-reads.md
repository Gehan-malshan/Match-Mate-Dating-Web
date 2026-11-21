# ADR-0005: Authoritative moderation reads at safety decisions

Status: Accepted (2026-11-21)

## Context and baseline

Moderation already owns audited actions, appeals, expiry, and transactional enforcement facts. Account, Booking, Matchmaking, and Notification did not consume those facts, so an action could be recorded without changing admission or visibility. An asynchronous projection alone cannot guarantee an immediate restriction after the moderator receives a successful response.

## Decision

The four services make short, uncached, private REST reads from Moderation at the points where they admit a member or disclose a protected resource. Moderation remains the sole owner of action state. Its private endpoint accepts a separate service token and returns only effective booleans, derived from active actions whose effective time has arrived and expiry has not passed. No report, evidence, reason, or actor data leaves Moderation. Services fail closed for sensitive operations when the read fails. Existing RabbitMQ facts remain the audit/integration stream; they are not the authority for immediate safety checks.

`ALL` (and legacy `GLOBAL`) means every event. An event UUID scopes event exclusion, matching exclusion, or reveal prevention. Account suspension and profile restriction apply globally. Account/profile targets use the account UUID because Account's profile key is the account UUID. Moderation validates actionable target and scope combinations before recording them.

Account denies normal sessions for suspended members and hides restricted public profiles. Password sign-in for a suspended member yields a short-lived `appeal` role token only; Google cannot issue it. Account and Booking deny that role for ordinary operations. Moderation permits its owner-scoped action list and appeal submission with the restricted token. A successful reversal restores ordinary roles at the next password sign-in or refresh. Matchmaking rechecks participants at generation, override, lock, publication, member display, and reveal consent. Booking denies new reservations and payment snapshots, and computes current entry eligibility on administrative registration responses without rewriting paid booking history. Notification suppresses restricted send attempts and hides restricted feed items.

## Consequences and limits

- Moderation availability becomes a safety dependency for these paths. Do not cache a safe decision or fail open on timeouts.
- The check occurs near the owning-service write, but a moderator may commit an action in the small interval after a check and before a different service commits or sends. Cross-database atomic exclusion is not provided. Rechecks and subsequent reads close exposure for later requests. Strict zero-window enforcement would require an admission lease/lock protocol or compensating consumers and is a release decision.
- Existing paid or confirmed bookings remain historically intact. `entryEligible=false` prevents interpreting a confirmed booking as permission to enter. No automatic cancellation, refund, or capacity release is implied. **OPEN QUESTION (Product/Finance/Safety):** decide treatment of excluded paid bookings and venue payments.
- The GraphQL browser interface exposes only owner-scoped active action summaries and appeal submission. The restricted browser route `/appeal` does not load Account's ordinary profile. Moderation alone validates the authenticated owner and stores the appeal. Moderator case management remains outside the browser gateway.
- Production must configure a distinct, high-entropy `MODERATION_SERVICE_TOKEN` and private `MODERATION_API_URL`, remove direct public access to the internal safety route, and test Moderation outage behavior and load.

## Verification and rollback

Unit tests cover restricted access, appeal role, Booking admission and reversal, matching/reveal decisions, notification suppression, malformed or unavailable safety responses, and private endpoint authentication. PostgreSQL tests cover action application, appeal reversal, and expiry. Rollback requires reverting all service callers together; reverting only Moderation would cause fail-closed dependency errors. No Moderation-owned action or audit row is deleted on rollback.
