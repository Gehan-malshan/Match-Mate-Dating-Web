# ADR-0003: Administrator-only event creation and matchmaking operations

- Status: Accepted
- Date: 2026-08-29

Frontend topology note: the separate admin runtime described here was superseded by [ADR 0004](0004-unified-role-aware-frontend-and-graphql-bff.md). The administrator-only authorization decision remains active.

## Context

The initial Event and Matchmaking prototypes treated the assigned organizer and administrator as equivalent privileged operators. The admin frontend therefore accepted either role, Event Service allowed either role to create a draft, and Matchmaking Service allowed an assigned organizer to list, generate, review, override, lock, and publish runs.

The approved product direction now requires a separate administrator dashboard. Only administrators may create events or view and operate the matchmaking management workflow. Member-facing match actions remain participant-scoped, and assigned organizers may still perform approved later operations on existing events where Event policy permits.

## Decision

- Event Service requires the `admin` role for `POST /api/v1/events`.
- Matchmaking Service requires the `admin` role for matching-run list, generate, detail, review, override, lock, and publish operations.
- Matchmaking authorization is enforced in the service before reading run details. Frontend visibility is never the security boundary.
- Member match list, response, reveal-consent, and feedback routes remain restricted to the authenticated pairing participant.
- `frontend/apps/admin` is the protected event and matchmaking application. It accepts only an administrator session and keeps its access token in memory.
- The member login flow detects the `admin` role and redirects to the separate admin application. The admin application restores the rotating `HttpOnly` refresh session instead of transferring an access token through a URL or browser storage.
- Administrator views expose participant codes, scores, safe generalized explanations, unmatched reason codes, and aggregate rejection diagnostics. They do not expose raw matching answers or community/member PII.

## Before and after

- Before: organizer or administrator could create events; assigned organizers could access matching-run operations; the admin app was an organizer event-management scaffold.
- After: event creation and all matching-run management are administrator-only; the admin app provides Overview, Event Manager, and deterministic Matchmaking workspaces connected to real APIs.
- Reason: provide the explicitly approved operational separation and reduce access to sensitive matching diagnostics and irreversible lock/publication controls.

## Consequences

- Existing organizer clients calling event creation or matching-run endpoints receive stable `403` problem codes.
- Existing-event update and lifecycle authorization remains assigned-organizer/admin unless a later policy decision narrows it.
- No database migration is required. Development seeds add `admin@example.test`; shared Event/Matchmaking fixtures are assigned to its stable test account ID.
- Account Service CORS allow-lists the exact local admin origins on port `5174`; wildcard origins are not introduced.
- Production deployment must add MFA or step-up authentication, least-privilege administration, audit monitoring, and gateway routing before release.
- Reversing this decision requires coordinated backend, OpenAPI, security, UI, tests, and documentation changes; changing only the frontend is not an acceptable rollback.
