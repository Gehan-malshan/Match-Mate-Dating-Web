# Administrator UI Source

The Registrations table lets administrators record cash or external card collection for eligible venue bookings and check in confirmed eligible members. The table shows persisted collection and check-in timestamps after refresh. Collection is a staff record of the booking's immutable price, not a PayHere transaction. Confirmed venue reservations have no cancellation option.

Publishing an event opens a required `event-publication-v1` review. The administrator must confirm image rights, image-content moderation, and the event safety plan. Event Service rejects bypassed or incomplete approvals and retains the approving account, UTC time, and review version. Follow `docs/runbooks/event-publication-safety.md`.

Registration rows now show current `entryEligible` from Booking through GraphQL. A confirmed booking with a safety exclusion appears as **Entry blocked**; staff must refresh before admitting a member. Historical booking and payment records remain visible to authorized administrators.

Event manager has **Delete** for draft/cancelled events (published or open events must first be cancelled). Delete requests confirmation, then archives the record; it is removed from the normal list without erasing bookings. **Show deleted** displays archived cards and **Restore** returns one to the managed list. Restoring a cancelled event does not reopen registration. Event audit history is not stored.

The draft event editor now has an optional event-photo picker and preview. It accepts JPEG/PNG/WebP, compresses to a bounded JPEG, saves the draft, then uploads the image. If upload fails, the draft remains saved and can be reopened to retry. Images cannot be replaced after publication. Choose only artwork the project has rights and consent to publish.

The event draft editor lets an administrator choose pay at venue, PayHere online, or both. The setting is visible in managed events and becomes immutable after publication under the current draft-only edit API. New drafts default to pay at venue; older events stay online-only until deliberately edited while still drafts.

In Event manager, choose **View registrations** on an event card to see the admin-only, paginated booking list with member nickname/email, booking state, payment method, collection, check-in, and amount. Pending, expired, cancelled, and payment-review records remain distinguishable.

The sidebar **Registrations** section is the main ledger across all events. Its event selector narrows the server query to one event, and previous/next load bounded pages. Event cards open this section with their event preselected. The list includes event name, booking reference, creation/confirmation times, payment method, venue collection, and check-in. It does not verify an external card settlement.

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
