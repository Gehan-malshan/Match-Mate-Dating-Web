# Organizer and Moderation Application

This React and TanStack application will provide event management, participant operations, pairing review and overrides, payment-support views, moderation queues, and audited administration tools.

## Planned feature areas

- Organizer event creation, lifecycle, pricing, policy, and capacity configuration.
- Event-scoped participant/booking/attendance operations.
- Matching-run generation, score/reason inspection, override, lock, and publication.
- Safe unmatched/eligibility summaries without exposing private preferences.
- Moderation report/case/action/appeal workflow according to role.
- Restricted support/finance payment reconciliation views.
- Role/configuration/audit views for administrators.

## Security boundaries

- UI hides unauthorized actions, but each backend service performs authoritative checks.
- Organizer access is event-scoped; organizer is not a global moderator, finance user, or administrator.
- Use least-privilege routes and APIs for moderator, support, finance, and admin roles.
- Require reason/confirmation for overrides, cancellation, moderation, refund, reveal/safety, and other sensitive actions.
- Avoid bulk export. Any approved export is minimized, audited, expiring, and documented.
- Mask PII/payment/safety fields by default and audit privileged views where required.
- Support strong reauthentication/MFA for privileged roles before production.

## Required tests

- Full role and event-scope authorization matrix.
- Sensitive field masking and absence from unauthorized responses/UI.
- Stale version/conflict behavior for event and matching edits.
- Pair override cannot bypass hard safety/eligibility constraints.
- Moderation action/reversal/appeal and audit.
- Payment review/refund permissions and confirmation.
- Keyboard/accessibility and destructive-action confirmation.

Update this README, canonical docs, contracts, tests, and change history whenever behavior changes.

