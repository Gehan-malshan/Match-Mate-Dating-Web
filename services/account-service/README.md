# Account and Profile Service

Owns registration, authentication, roles, sessions, account lifecycle, community-safe profiles, private matchmaking preferences, profile visibility, and member blocks.

Implemented owned data includes accounts, credentials, consent records, roles, sessions, profiles, preferences, visibility settings, blocks, moderation audit records, and an event outbox.

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

## Implemented API

```text
POST   /api/v1/auth/register
POST   /api/v1/auth/verify-email
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
GET    /api/v1/users/me
PATCH  /api/v1/users/me/profile
PUT    /api/v1/users/me/matching-preferences
GET    /api/v1/community/profiles
GET    /api/v1/community/profiles/{profileId}
POST   /api/v1/users/me/blocks
DELETE /api/v1/users/me/blocks/{accountId}
POST   /api/v1/admin/profiles/{accountId}/decision
DELETE /api/v1/users/me
GET    /.well-known/jwks.json
```

Self-service identity comes from the access-token subject. Admin paths require explicit scopes and audit.

## Implemented data

`account`, `credential`, `consent_record`, `role_assignment`, `email_verification_token`, `refresh_session`, `profile`, `profile_interest`, `matching_preference`, `member_block`, `audit_log`, and `outbox`.

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

## Current implementation and boundaries

The executable vertical slice is in this directory. `cmd/api` serves REST, `cmd/migrate` applies the first service-owned migration, and `cmd/outbox-relay` publishes durable facts to the `matchmate.events` RabbitMQ topic exchange. Domain rules are transport-independent; PostgreSQL is accessed only by the repository adapter.

Registration creates an adult-only, `PRIVATE`, `PENDING` profile and records the accepted consent policy version. Login requires a verified, active account. Passwords use Argon2id. ES256 access tokens expire after 10 minutes by default and contain subject, roles, and token version—not PII. A 256-bit opaque refresh token rotates in an `HttpOnly`, `SameSite=Lax` cookie; only its SHA-256 hash is stored. Reuse revokes the entire session family. Deactivation increments token version, revokes sessions, hides the profile, and emits a minimum-data event.

The community projection is an explicit allow-list: profile ID, nickname, five-year age band, broad location, bio, and interests. It requires active + verified + approved + community-visible state and excludes blocks in either direction. Exact DOB, email, preferences, deal-breakers, credentials, and moderation data cannot be serialized by this DTO.

Profile approval/hiding requires `moderator` or `admin` and creates an audit record. Full moderation case/evidence/appeal ownership remains in the future Moderation Service. Profile media and questionnaire answers are intentionally deferred until their policy and schema decisions are approved. Production verification-email delivery also remains an integration task: development may return a token only when `DEV_EXPOSE_VERIFICATION_TOKEN=true`; production must keep it false and connect an approved private delivery path without putting email addresses in public facts.

## Run locally on Windows / PowerShell

Prerequisites: Go 1.26+, Docker Desktop, and ports 5433, 5672, 15672, and 8081 available.

From the repository root, the preferred GNU Make workflow is:

```text
make setup
make up
make status
```

This starts the Account API and outbox relay in Docker in addition to PostgreSQL/RabbitMQ and the automatic migration/seed jobs. Run `make web` in a second terminal.

```powershell
docker compose -f infrastructure/compose/account-profile.compose.yml up -d --build
Copy-Item services/account-service/.env.example services/account-service/.env
```

Load the local variables in the terminal, then run from `services/account-service`:

```powershell
$env:DATABASE_URL='postgres://matchmate:matchmate@localhost:5433/matchmate_account?sslmode=disable'
$env:DEV_EXPOSE_VERIFICATION_TOKEN='true'
go run ./cmd/api
```

Run the relay in a second terminal with `go run ./cmd/outbox-relay`. Run the frontend from `frontend/apps/web` with `bun run dev`; it uses `http://localhost:8081/api/v1` unless `VITE_ACCOUNT_API_URL` overrides it.

### Shared development logins

The development Compose command automatically runs the idempotent migration and `seed-dev` jobs after PostgreSQL becomes healthy. You can also run `go run ./cmd/seed-dev` manually after setting `APP_ENV=development` and `DATABASE_URL`. The seed command refuses to run unless `APP_ENV` is `development` or `test`. It creates verified fake accounts using reserved `.test` addresses and the hard-coded public test password `MatchMateDev123!`:

| Email | Purpose |
|---|---|
| `member@example.test` | Normal private member |
| `community@example.test` | Approved community-visible member |
| `moderator@example.test` | Moderator API testing |
| `suspended@example.test` | Suspended-account denial testing |
| `organizer@example.test` | Event organizer UI/API testing |

These are public development fixtures, not secrets. Never copy them into staging or production, never add an authentication bypass for them, and never use real personal information in seed data. Re-running the command restores the known password/state and invalidates existing access tokens by incrementing token versions.

The same seed creates six fictional, approved Community profiles with reserved `.test` identities. They have no credential rows and cannot log in; they exist only so every developer sees a useful Community directory after starting Docker.

## Configuration

Copy `.env.example` as a reference. The binaries read process environment variables; they do not automatically parse `.env`. `DATABASE_URL` is required. Production also requires a PKCS#8 P-256 `JWT_PRIVATE_KEY_PEM`, `COOKIE_SECURE=true`, approved exact `ALLOWED_ORIGINS`, secret-manager injection, TLS at the gateway, and `DEV_EXPOSE_VERIFICATION_TOKEN=false`.

## Verification

```powershell
go test ./...
go vet ./...
```

Unit tests cover password hashing, ES256/JWK claims, opaque-token hashing, adult age boundaries, contact-information rejection, and community DTO leakage. Database integration, migration rollback, RabbitMQ retry/DLQ, browser E2E, load, and security tests remain required before production. The canonical REST and event contracts are `contracts/openapi/account-v1.yaml` and `contracts/asyncapi/account-events-v1.yaml`.

## Change rule

Any behavior, field, endpoint, event, data, privacy, or security change must update its implementation and tests plus this README, the canonical OpenAPI/AsyncAPI file, affected architecture/data/security docs, and the pull-request before/after summary described in `docs/change-management/README.md`.
