# Member Web Application

The member-facing React and TanStack application will support registration, profile and preference management, safe community discovery, event discovery, ticket purchasing, event matches, responses, and feedback.

## Planned stack and boundaries

- React + TypeScript.
- TanStack Router for typed route/access structure.
- TanStack Query for server state; backend services remain authoritative.
- TanStack Form and Zod for accessible client validation; server validation is mandatory.
- Shared presentation components from `packages/ui` and generated OpenAPI client helpers.

The web app never connects directly to databases, RabbitMQ, PayHere secrets, or private service addresses. It calls the API Gateway over HTTPS.

## Planned feature areas

```text
public marketing/events
auth and verification
profile + public-preview + private preferences
safe community discovery + block/report
event detail + booking hold + PayHere transition
booking/payment status
own published event pairing
structured response + consent + feedback
account privacy, sessions, deactivation
```

## Frontend rules

- Route guards improve UX but are not authorization; services enforce access.
- Never persist access/refresh tokens or private profile/payment data in unsafe browser storage without an approved security decision.
- Query caches are cleared on logout, account switch, restriction, and sensitive state changes.
- Do not render private fields received accidentally; report contract leakage and fix the API.
- Accessible keyboard, focus, labels, errors, contrast, loading, empty, pending, and failure states are required.
- Payment success is confirmed from server state, never only from browser redirect parameters.
- Do not add chat or direct contact-sharing UI without an approved product/architecture change.

## Required tests

- Route/session and role behavior.
- Form validation and server error mapping.
- Public-profile leakage snapshots.
- Booking expiry/payment pending/failure/review states.
- Member can access only own match/response/consent.
- Block/report and accessibility journeys.
- Generated-client contract compile and critical browser E2E.

Update this README, relevant handbook pages, API contracts, tests, and change history whenever behavior changes.

