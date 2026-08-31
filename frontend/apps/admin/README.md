# MatchMate Administration Application

The administration application is a separate React/TanStack Query frontend at `http://localhost:5174`. It is not a member interface. Account Service authentication, Event Service authorization, and Matchmaking Service authorization all require the `admin` role for the privileged actions implemented here.

## Implemented workspaces

### Overview

- Event totals, registration-open count, draft count, and upcoming schedule.
- Direct navigation to protected event creation and deterministic matchmaking.
- Clear administrator-session and least-privilege indicators.

### Event Manager

- List all managed events for an administrator.
- Create an administrator-owned private draft.
- Edit draft configuration with optimistic concurrency.
- Publish, open/close registration, and cancel with an audited reason.
- Keep exact venue data inside the protected interface; member event DTOs remain restricted.
- Label configured capacity as Event configuration, not Booking-owned availability.

Only administrators can create events. Event Service enforces this rule and returns `EVENT_ADMIN_REQUIRED` for organizer/member callers; hiding the UI is not the security boundary.

### Matchmaking

- Select an event and list immutable matching-run history.
- Generate a new deterministic, non-ML run with an idempotency key.
- Inspect participant-code-only pairings, compatibility scores, safe generalized reasons, unmatched outcomes, and aggregate hard-rule exclusions.
- Move a run through `GENERATED -> UNDER_REVIEW -> LOCKED -> PUBLISHED`.
- Replace a selected pairing only with an eligible candidate and a required audit reason.
- Preserve server-side hard eligibility, minimum score, optimistic version, participant uniqueness, and immutable lock rules.

Matching-run management is administrator-only in Matchmaking Service. Member response, mutual-interest, reveal-consent, and feedback endpoints remain accessible only to the relevant pairing participant.

## Authentication behavior

The access token stays in memory; it is not written to local storage. Account Service provides the rotating refresh session in an `HttpOnly` cookie. Concurrent client refresh attempts share one in-flight request so React development checks and simultaneous API retries cannot reuse a rotating token. The member website is the only login entry point: it detects the `admin` role after sign-in and redirects to this application, which restores the session through `/auth/refresh`. Opening this app without an active administrator session redirects to the member login page.

Development-only fixture:

```text
Email: admin@example.test
Password: MatchMateDev123!
```

The organizer fixture is intentionally denied access to this application and cannot create an event or manage a matching run.

## Run locally

Start backend services from the repository root:

```powershell
docker compose up --build -d
```

Start the admin frontend in another terminal:

```powershell
bun run dev:admin
```

Sign in as the administrator at `http://localhost:5173/login`; the role-based redirect opens `http://localhost:5174`. Opening `http://localhost:5174` directly without a session also redirects to the member login page.

Verification:

```powershell
bun run typecheck:admin
bun run test:admin
bun run build:admin
```

## Prototype boundary

Only the fixed event `11111111-1111-4111-8111-000000000001` currently has Matchmaking-owned participant projections. Newly created events can be configured and published, but matching generation remains unavailable until confirmed Booking, Account, Event, and Moderation facts populate the Matchmaking projection. The UI explains this state instead of inventing participant data.

Before production, add MFA/step-up authentication, gateway deployment, production event consumers, end-to-end browser automation, load/concurrency evidence, metrics/traces, retention controls, and approved operational runbooks.

Any future behavior change must update backend authorization tests, API contracts, this README, canonical architecture/implementation guidance, and the pull-request before/after record.
