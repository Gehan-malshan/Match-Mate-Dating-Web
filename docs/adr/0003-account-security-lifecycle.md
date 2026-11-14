# ADR-0003: Account security lifecycle

Status: Accepted (2026-11-14)

## Context

The first Account slice stored hashed verification tokens and rotated refresh sessions, but did not deliver verification links or support password recovery. Privileged sign-in had no second factor. Process-local request counters were insufficient across replicas. Deactivation hid the profile but retained Account-owned identifying data.

## Decision

- Account owns verification/reset tokens, password/session state, administrator TOTP enrollment, and security rate windows in its PostgreSQL database.
- Notification owns SMTP transport. Account sends a one-time link through a separate, service-authenticated, transient internal endpoint. The raw token is not an event, notification row, or log entry. Delivery failure is operationally visible; a member may request a replacement link.
- Password reset tokens expire after 30 minutes, are stored as SHA-256 hashes, are single use, and revoke existing refresh sessions and access tokens through token-version change.
- Privileged administrator access requires a confirmed TOTP when `ADMIN_MFA_ENCRYPTION_KEY` is configured; production requires the key. Google OAuth cannot create an administrator session in that mode. An operator provisions the first TOTP secret through the restricted CLI, with a separate confirmation step. TOTP secrets are encrypted at rest with an operator-managed AES-256-GCM key.
- Account-owned login, registration, recovery, and discovery counters use PostgreSQL atomic fixed-minute windows. The existing local IP limiter remains an additional layer, not the production edge control.
- JWT signing key rotation publishes the active and configured previous public keys through JWKS during the access-token overlap window. Refresh tokens continue to rotate on use.
- Member deletion requires password reauthentication, revokes sessions, pseudonymizes Account email, and erases Account-owned profile, preferences, credentials, verification/recovery tokens, and TOTP material. The stable account ID and minimal audit/consent/booking references remain pending an approved retention schedule.

## Consequences and open questions

- Security email delivery is synchronous and has no durable retry queue. SMTP acceptance is not inbox proof. Production needs provider evidence, delivery alerts, edge rate limits, and a recovery drill.
- The first MFA enrollment is an operator step. Lost authenticator recovery requires a separately approved, audited process; no self-service bypass is provided.
- **OPEN QUESTION (product/privacy/legal owner):** Define retention and erasure behavior across Booking, Payment, Matchmaking, Notification, Moderation, logs, and backups, plus legal/financial exceptions and retention periods. Account deletion does not claim cross-service or backup erasure.
- **OPEN QUESTION (security owner):** Define MFA encryption-key rotation and privileged recovery ceremony before production.
