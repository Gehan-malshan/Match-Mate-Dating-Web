# Member Web Application

The member-facing React and TanStack application will support registration, profile and preference management, safe community discovery, event discovery, ticket purchasing, event matches, responses, and feedback.

## Implemented foundation and boundaries

- React + TypeScript.
- Bun for package management, workspace dependency installation, and script execution.
- Vite for the development server and production frontend build.
- TanStack Router for typed route/access structure.
- TanStack Query for server state; backend services remain authoritative.
- TanStack Form and Zod for accessible client validation; server validation is mandatory.
- Shared presentation components from `frontend/packages/ui` and generated OpenAPI client helpers.

The web app never connects directly to databases, RabbitMQ, PayHere secrets, or private service addresses. It calls the API Gateway over HTTPS.

The repository uses one frontend lockfile: `bun.lock`. Do not add pnpm, npm, or Yarn lockfiles. Bun runs the checked-in Vite, ESLint, TypeScript, test, and build scripts; Bun's bundler is not the default frontend bundler unless a later ADR changes the Vite decision.

Implemented routes include the redesigned public `/` landing page, `/register`, `/login`, the authenticated `/app/profile` workspace, `/app/bookings`, `/app/notifications`, authenticated `/community` and `/community/$profileId` discovery/detail routes, public `/events`, and `/events/$eventId` discovery/detail routes. Registration uses TanStack Form and Zod before server-authoritative validation. The API client keeps access tokens in memory, sends refresh cookies only with credentialed requests, retries once after refresh, and never writes tokens or private profile data to browser storage. The Midnight Chemistry profile workspace provides a community-safe preview, profile visibility and moderation state, editable introduction/interests, and a visually separate private matching blueprint. It intentionally omits unsupported avatar generation, credential changes, and unapproved lifestyle fields. Community discovery renders only the Account Service allow-list, excludes blocked relationships, provides a safe detail view, and exposes no chat or contact-sharing action. Event discovery shows only the Event Service public allow-list and clearly labels configured capacity as non-authoritative.

The landing page uses minimal responsive navigation, a happy-couple hero, an image-led About section, a privacy/safety composition, a fully visible five-step journey, explainable preferences-to-rules-to-review matching, a truthful pre-launch event stage, and structured product/trust footer navigation. Distinct project-local imagery gives the hero, café conversation, event check-in, and rooftop-event chapters separate visual purposes. The working registration call-to-action links to `/register`; event actions continue to link only to confirmed announcement information.

GSAP and `@gsap/react` provide progressive entrance and scroll motion. Motion is presentation-only: the page remains complete without JavaScript-driven animation, and `prefers-reduced-motion: reduce` bypasses GSAP timelines and disables the marquee animation. Do not use motion to create pressure around consent, privacy, payment, or safety decisions.

All new visual work must follow the canonical [Midnight Chemistry base design system](../../../docs/design/README.md). The landing page now implements its semantic dark surfaces, brand gradient, 12-column rhythm, tonal elevation, responsive gutters, approved logo, and motion rules. Future changes must preserve the varied composition and must not drift back toward repeated generic cards or unsupported product claims.

## Booking and payment journey

The member app now provides atomic seat reservation and PayHere checkout initiation on open event detail pages. Event detail loads the authenticated member's bookings and automatically reopens checkout for an existing `PENDING_PAYMENT` hold, so navigating away never forces a duplicate reservation. `/app/bookings` labels that route **Complete payment**, polls authoritative Booking and Payment state, shows pending, confirmed, failed, expired, cancelled, and review outcomes, and allows unpaid holds to be cancelled safely. Browser return parameters are never treated as payment confirmation.

Booking defaults to `http://localhost:8085/api/v1` and Payment to `http://localhost:8084/api/v1`. Configure `VITE_BOOKING_API_URL` and `VITE_PAYMENT_API_URL` alongside the existing Account and Event variables.

## In-app notifications

The member header includes an authenticated notification bell, unread badge, recent-items popover, and link to the paginated `/app/notifications` history. TanStack Query polls the owner-scoped Notification API every 10 seconds. Notifications that arrive after the initial successful load appear as dismissible, reduced-motion-safe popup toasts; existing unread history does not repeatedly pop up on reload. Opening an item or using mark-read/mark-all updates server-owned read state.

The frontend never sends a recipient account ID: Notification derives ownership from the Account access-token subject. It defaults to `http://localhost:8086/api/v1`; configure `VITE_NOTIFICATION_API_URL` for another environment. Real email is deliberately not represented as complete.

## Local commands

Run these commands from the repository root:

```powershell
bun install --frozen-lockfile
bun run dev:web
bun run typecheck:web
bun run test:web
bun run build:web
```

The development URL is `http://127.0.0.1:5173`; the production build is written to `frontend/apps/web/dist` and is ignored by Git.

The Account API defaults to `http://localhost:8081/api/v1` and Event API to `http://localhost:8082/api/v1`. Override them with `VITE_ACCOUNT_API_URL` and `VITE_EVENT_API_URL` (see `.env.example`). APIs must allow the exact Vite origin and use TLS/secure cookies in production.

## Brand assets

Approved logo files live under `public/brand`. The supplied artwork is preserved as `matchmate-logo-source.png`; runtime navigation uses the transparent, optimized `matchmate-logo-nav.png`, and browser metadata uses `matchmate-favicon.png`. Do not replace, redraw, recolor, or repurpose these files without updating the canonical design guide and change history.

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
