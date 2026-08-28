# ADR-0001: Account authentication and browser session transport

- **Status:** Accepted
- **Date:** 2026-08-28
- **Owners:** Project team
- **Related issue/PR/change:** CHG-20260828-007
- **Supersedes:** None
- **Superseded by:** None

## Context

The Account/Profile vertical slice needs secure credentials, independently verifiable service identity, browser session continuity, revocation, reuse detection, and a migration path to gateway deployment. The frontend must not persist long-lived bearer credentials in local storage, while backend services must be able to authorize short-lived requests without synchronously calling Account.

## Decision drivers

- Minimize useful token material exposed to browser script.
- Support service-local authorization and key rotation.
- Detect stolen refresh-token reuse and revoke its family.
- Avoid PII in tokens and profile data in cookies.
- Work in local Vite development and future same-site gateway deployment.

## Considered options

### Stateful server session cookie for every request

Strong browser ergonomics and simple revocation, but every service request would require centralized session lookup or gateway impersonation. This conflicts with independently authorizing services and increases synchronous coupling.

### Access and refresh tokens in browser storage

Simple API clients, but persistent JavaScript-readable refresh tokens substantially increase the impact of XSS and make secure family rotation harder to contain.

### Short-lived ES256 bearer access token plus rotating HttpOnly refresh cookie

Services verify access tokens locally through JWKs. Browser script keeps only the current access token in memory. The opaque refresh token is `HttpOnly`, `Secure` outside local development, `SameSite=Lax`, narrowly scoped, hashed at rest, rotated on every use, and protected by Origin checks.

## Decision

Use ES256 access tokens with a planning lifetime of 10 minutes. Claims contain `sub`, roles/scopes, token version, issuer, audience, issue/not-before/expiry times, and a unique token ID—no email, profile fields, preferences, or other PII.

Use opaque 256-bit refresh tokens in an HttpOnly cookie scoped to `/api/v1/auth`. Store only SHA-256 token hashes. Rotate on every refresh, link sessions into a token family, and revoke the entire family when a revoked token is reused. Login and refresh return the short-lived access token in JSON; the frontend keeps it in memory and attempts refresh on startup. Logout revokes the presented session and expires the cookie.

Use a configured P-256 private key in deployed environments. Non-production may generate an ephemeral startup key when none is supplied, with an explicit warning. Publish the public key through `/.well-known/jwks.json` and support overlapping keys before production rotation.

## Consequences

### Positive

- Refresh credentials are unavailable to normal browser JavaScript.
- Services can validate access independently.
- Refresh reuse becomes detectable and auditable.
- Tokens remain small and contain no direct PII.

### Negative and trade-offs

- Access state is lost on reload and must be restored with a refresh call.
- Cross-origin local development requires explicit allowed origins and credentials.
- Cookie-based refresh/logout endpoints need Origin/CSRF protections.
- Key distribution and overlapping rotation require operational discipline.

### Risks and mitigations

- XSS can use an in-memory token while the page is compromised: enforce CSP, output encoding, dependency review, and short expiry.
- Cookie CSRF: use SameSite, exact Origin validation, narrow paths, and non-GET mutation methods.
- Ephemeral development keys invalidate tokens on restart: documented as development-only behavior.
- Access tokens remain valid until expiry after revocation: keep lifetime short and use token-version checks for high-risk owner operations when approved.

## Compatibility and migration

This is the first implementation. Breaking claim, issuer/audience, cookie, or refresh semantics require a new ADR and coordinated migration. Key rotation adds a new JWK before issuing tokens with it and retains the old key until all old tokens expire.

## Security, privacy, safety, and operations

Use Argon2id password hashes with per-password salts. Never log passwords, access/refresh/verification tokens, cookies, authorization headers, or private profile data. Rate-limit registration, verification, login, and refresh. Audit family reuse, logout, deactivation, role/profile decisions, and privileged changes. Production keys and database/broker credentials come from an approved secret manager.

## Verification

- Password hash/verify and malformed-hash tests.
- Access claim, issuer/audience, expiry, and JWK tests.
- Refresh rotation, duplicate/reuse, family revocation, and cookie tests.
- Origin, CORS, authorization, profile leakage, and deactivation tests.
- Operational key-rotation rehearsal before production.

## Documentation updates

Architecture, API, data, security, Account service, member web, local development, OpenAPI/AsyncAPI, and change history.
