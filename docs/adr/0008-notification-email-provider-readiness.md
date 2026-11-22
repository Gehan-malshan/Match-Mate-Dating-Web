# ADR-0008: Notification email provider readiness, preferences, and replay

- **Status:** Accepted
- **Date:** 2026-11-23
- **Owners:** MatchMate Platform, Safety, and Member Experience

## Context

Notification already owns recipient-safe delivery records, retries, a member in-app feed, and an SMTP adapter. A production deployment still needed an explicit sender-verification gate, a member-owned preference surface, and a recovery method that did not duplicate messages or expose RabbitMQ payloads.

## Decision

- SMTP is the approved first real email transport. The provider/dashboard must verify the domain or sender address before deployment; production configuration refuses to start unless `SMTP_SENDER_VERIFIED=true` is explicitly attested.
- The owner-service `notification-provider-preflight` command proves TLS, SMTP authentication, and that the server accepts `MAIL FROM` for the configured sender. It does not send mail and does not replace provider-domain verification or a controlled canary delivery.
- Members can read and set their own `EMAIL` preferences only for deployment-configured `ACCOUNT` and `BOOKING` categories. Preferences store no destination and do not affect transient verification/password-recovery email, which is an Account security path rather than a general notification delivery.
- Terminal `PERMANENTLY_FAILED` and `DEAD_LETTERED` deliveries can be replayed only through the Notification owner command. The command requires a named operator, reason, explicit `REPLAY` confirmation, and a bounded additional-attempt budget. It preserves the same delivery ID and appends an immutable replay record. Suppressed deliveries cannot be replayed.
- Provider messages use the existing delivery ID as their idempotency identity. A post-provider crash may retry the same delivery, so provider selection/configuration must preserve this ID in its own deduplication mechanism where available.

## Consequences

- Migration 5 adds the isolated `notification_delivery_replay` audit table. It is additive; rollback is only for disposable environments. A production rollback stops workers and forward-fixes while retaining the audit evidence.
- Actual external sender verification, SMTP credentials, DNS/provider evidence, recipient consent/category policy approval, and controlled canary evidence remain deployment-owned actions. They cannot be truthfully completed from source code without the provider account and verified domain.
- No direct SQL updates or raw RabbitMQ replay are used for normal recovery. The runbook requires targeted owner-command replay after the cause is corrected.

## Verification

Configuration tests reject production SMTP without the sender-verification attestation. Notification's PostgreSQL component test verifies preference persistence and audited terminal-delivery replay. The member web exposes GraphQL-backed preference controls; CI runs service, gateway, and web checks.

See [notification delivery runbook](../runbooks/notification-local.md).
