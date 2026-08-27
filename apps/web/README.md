# Member Web Application

The member-facing React and TanStack application will support registration, profile and preference management, safe community discovery, event discovery, ticket purchasing, event matches, responses, and feedback.

## Implemented foundation and boundaries

- React + TypeScript.
- Bun for package management, workspace dependency installation, and script execution.
- Vite for the development server and production frontend build.
- TanStack Router for typed route/access structure.
- TanStack Query for server state; backend services remain authoritative.
- TanStack Form and Zod for accessible client validation; server validation is mandatory.
- Shared presentation components from `packages/ui` and generated OpenAPI client helpers.

The web app never connects directly to databases, RabbitMQ, PayHere secrets, or private service addresses. It calls the API Gateway over HTTPS.

The repository uses one frontend lockfile: `bun.lock`. Do not add pnpm, npm, or Yarn lockfiles. Bun runs the checked-in Vite, ESLint, TypeScript, test, and build scripts; Bun's bundler is not the default frontend bundler unless a later ADR changes the Vite decision.

The first implemented slice is the public `/` landing route. It includes responsive navigation, a privacy/safety overview, an accurate four-step member journey, an explicit no-ML matchmaking explanation, a pre-launch event state, original project-local imagery, reduced-motion support, visible keyboard focus, and component tests. Login, registration, and event inventory are intentionally not presented as operational until their backend journeys exist.

All new visual work must follow the canonical [Midnight Chemistry base design system](../../docs/design/README.md). Existing landing-page styling predates the canonical token guide and should be aligned through focused, tested follow-up changes rather than undocumented one-off edits.

## Local commands

Run these commands from the repository root:

```powershell
bun install --frozen-lockfile
bun run dev:web
bun run typecheck:web
bun run test:web
bun run build:web
```

The development URL is `http://127.0.0.1:5173`; the production build is written to `apps/web/dist` and is ignored by Git.

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
