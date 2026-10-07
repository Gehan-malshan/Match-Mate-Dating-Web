# Administrator UI Source

The draft event editor now has an optional event-photo picker and preview. It accepts JPEG/PNG/WebP, compresses to a bounded JPEG, saves the draft, then uploads the image. If upload fails, the draft remains saved and can be reopened to retry. Images cannot be replaced after publication. Choose only artwork the project has rights and consent to publish.

The event draft editor lets an administrator choose pay at venue, PayHere online, or both. The setting is visible in managed events and becomes immutable after publication under the current draft-only edit API. New drafts default to pay at venue; older events stay online-only until deliberately edited while still drafts.

In Event manager, choose **View registrations** on an event card to see the admin-only, paginated booking list with member nickname/email, booking state, payment method, and amount. `CONFIRMED` pay-at-venue reservations are explicitly labelled as not collected in the app. Pending, expired, cancelled, and payment-review records remain distinguishable; this is not yet a check-in screen.

This directory contains the existing administrator workspace components and styles. It is no longer a separately deployed or separately started application: `frontend/apps/web/src/routes/AdminPage.tsx` bundles this source into the unified frontend at `/admin`.

Members and administrators share `http://localhost:5173/login`. The Account-issued role directs administrators to `/admin`. The web route guard, GraphQL BFF, Event Service, and Matchmaking Service all enforce administrator access; hiding navigation is never treated as authorization.

Use the unified commands from the repository root:

```powershell
bun run dev
bun run typecheck:web
bun run test:web
bun run build:web
```

Admin API operations share the web application's GraphQL client and `VITE_GRAPHQL_API_URL`, defaulting to `http://localhost:8080/graphql`. This directory intentionally has no Vite entry point, development server, or production build of its own. Do not reintroduce direct browser calls to Account, Event, or Matchmaking service ports.
