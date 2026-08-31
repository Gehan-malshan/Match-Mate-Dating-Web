# ADR 0004: Unified role-aware frontend and GraphQL BFF

- Status: Accepted
- Date: 2026-08-31

## Context

MatchMate previously ran member and administrator Vite applications separately. The browser called each service REST API directly, which multiplied origins, environment variables, authentication clients, and deployment configuration. The product direction now requires one login and one frontend with separately protected member and administrator capabilities, plus GraphQL as the browser API.

The expected registration count of 10,000+ does not by itself require GraphQL. GraphQL is selected for a single typed client contract and controlled aggregation; capacity remains an independent operational concern.

## Decision

- `frontend/apps/web` is the only frontend runtime and deployment. It bundles the existing admin workspace at `/admin`.
- One login mutation obtains an Account-issued access token. The frontend reads the returned role and routes administrators to `/admin`; member and admin route guards restore the rotating `HttpOnly` session and re-check authority.
- `services/graphql-gateway` exposes schema-first GraphQL at `/graphql` and owns no database or domain state.
- The gateway forwards identity and cookies to existing service-owned `/api/v1` REST endpoints. Services remain authoritative and repeat authorization.
- RabbitMQ, outbox/inbox behavior, PostgreSQL ownership, and PayHere callback routing are unchanged.
- Admin GraphQL resolvers require the `admin` role server-side before forwarding privileged commands.
- Query complexity, request size, upstream timeout, exact-origin CORS, cursor pagination, and idempotency keys are mandatory controls.

## Before and after

Before: two Vite processes on ports 5173 and 5174; direct browser-to-service REST calls on ports 8081–8086; role login redirected between applications.

After: one Vite process on port 5173; `/admin` and member routes are separately guarded; all browser data calls use GraphQL on port 8080; the BFF calls the existing REST services internally.

## Consequences

- Frontend deployment and CORS configuration are simpler.
- The GraphQL schema becomes a new compatibility contract and requires resolver/operation tests.
- The gateway is another stateless runtime that must be monitored and horizontally scalable.
- A GraphQL outage affects all browser data access, so health checks, timeouts, replica scaling, and rollback are required.
- REST remains supported internally; this is not a big-bang rewrite of domain services.

## Rollback

Revert the unified frontend API modules and Compose include, then redeploy the previous direct-REST frontend. No domain database or event migration is involved.

