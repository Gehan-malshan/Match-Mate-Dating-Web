# Account and Profile Service

Owns registration, authentication, roles, sessions, account lifecycle, community-safe profiles, private matchmaking preferences, profile visibility, and member blocks.

Planned owned data includes accounts, credentials, roles, sessions, profiles, preferences, visibility settings, and blocks.

## Responsibilities

- Register, verify, authenticate, refresh, log out, revoke, deactivate, and anonymize accounts according to retention policy.
- Hash passwords and detect refresh-token reuse.
- Manage member/organizer/moderator/support/admin roles and scopes.
- Store private PII separately from community-safe profile fields.
- Store source matching preferences, interests, and questionnaire answers with privacy classification/version.
- Produce safe profile discovery responses from an explicit allow-list.
- Manage profile visibility, approval/hidden state, media references, and blocks.
- Propagate lifecycle/safety eligibility facts without leaking private data.

## Does not own

Bookings, event capacity, event catalog, payments, matching scores/pairings, notification delivery, or moderation evidence/case decisions.

## Proposed API

```text
POST   /api/v1/auth/register
POST   /api/v1/auth/verify
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
GET    /api/v1/users/me
PATCH  /api/v1/users/me
PUT    /api/v1/users/me/preferences
GET    /api/v1/community/profiles
GET    /api/v1/community/profiles/{profileId}
POST   /api/v1/blocks
DELETE /api/v1/blocks/{accountId}
DELETE /api/v1/users/me
GET    /.well-known/jwks.json
```

Self-service identity comes from the access-token subject. Admin paths require explicit scopes and audit.

## Proposed data

`account`, `credential`, `role_assignment`, `refresh_session`, `profile`, `profile_interest`, `matching_preference`, `questionnaire_answer`, `profile_media`, `block`, and `outbox`.

Key invariants:

- Normalized email is unique.
- Passwords are one-way hashed; tokens are not stored as usable plaintext.
- Community DTOs never include email, phone, DOB, exact address, credentials, private preferences, verification evidence, or moderation notes.
- Date of birth remains private; age is derived using approved policy.
- A block affects discovery and is propagated for matching/reveal exclusion.
- Deactivation revokes sessions before downstream propagation.

## Events

Produces approved versions of `AccountRegistered`, `AccountVerified`, `ProfileApproved`, `ProfileHidden`, `ProfileUpdated`, `AccountDeactivated`, and block/account-restriction facts. Payloads contain minimum IDs and safe state/version fields.

Consumes restricted moderation-action facts that change profile/account eligibility.

## Security and privacy

- Short-lived RS256/ES256 access tokens and rotating refresh sessions.
- Login/email-check throttling and enumeration resistance.
- Field-level classification and public allow-list tests.
- Contact-information validation/quarantine for profile content.
- Media stored privately and served through approved controlled URLs.
- Audit role, visibility, verification, deactivation, and privileged profile changes.

## Required tests

- Email normalization/uniqueness and concurrent registration.
- Password/session/token expiry, rotation, revocation, and reuse.
- Ownership and complete role authorization matrix.
- Public profile leakage snapshots.
- Preference validation and age boundaries.
- Block symmetry across profile discovery and emitted facts.
- Deactivation/anonymization and retention behavior.
- Outbox retry and duplicate-safe downstream facts.

## Completion criteria

A verified member can register/login/manage safe profile and private preferences through the web app; unauthorized users cannot access private data; community responses/events/logs contain no restricted fields; lifecycle events and observability are verified.

Update this README, architecture/data/security docs, OpenAPI/AsyncAPI, tests, and change history whenever behavior changes.

