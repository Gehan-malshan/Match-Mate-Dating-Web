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

The first implemented slice is the public `/` landing route. Its AIDA composition includes minimal responsive navigation, a simple left-aligned hero over original happy-couple artwork, a concise image-led About section, an image-dominant privacy/safety composition with one open editorial principles column, a fully visible five-step matchmaking path paired with event imagery, a simple preferences-to-rules-to-review explanation, a truthful pre-launch event stage, and a structured footer with product navigation, trust navigation, and honest development status. Section spacing uses one consistent rhythm, warmer plum surfaces separate content chapters, and decorative marquees, grid lines, oversized empty gaps, and repeated boxed layouts are intentionally avoided. The page shows the complete path from private profile and confirmed event booking through deterministic compatibility guidance, guided in-person conversations, and mutual-choice follow-up without hiding essential content behind controls. Distinct project-local imagery gives the hero, café conversation, trust, event check-in, and rooftop-event chapters their own visual purpose instead of repeating one scene. Original project-local imagery, visible focus, text reflow, and a complete reduced-motion path are covered by the implementation and component tests. Login, registration, and event inventory are intentionally not presented as operational until their backend journeys exist.

GSAP and `@gsap/react` provide progressive entrance and scroll motion. Motion is presentation-only: the page remains complete without JavaScript-driven animation, and `prefers-reduced-motion: reduce` bypasses GSAP timelines and disables the marquee animation. Do not use motion to create pressure around consent, privacy, payment, or safety decisions.

All new visual work must follow the canonical [Midnight Chemistry base design system](../../../docs/design/README.md). The landing page now implements its semantic dark surfaces, brand gradient, 12-column rhythm, tonal elevation, responsive gutters, approved logo, and motion rules. Future changes must preserve the varied composition and must not drift back toward repeated generic cards or unsupported product claims.

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
