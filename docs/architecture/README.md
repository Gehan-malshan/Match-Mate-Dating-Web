# MatchMate System Architecture

This is the canonical architecture description for MatchMate. It must remain synchronized with implementation, OpenAPI, AsyncAPI, service READMEs, ADRs, migrations, tests, and runbooks.

## 1. Product purpose

MatchMate is a privacy-first, community-driven blind-dating platform for Sri Lanka, initially focused on Colombo. Its purpose is to move relationship-minded adults from safe online discovery to curated real-world interaction instead of prolonged chatting or swiping.

The platform supports verified member registration, limited-identity community profiles, private matching preferences, curated ticketed events, explainable rule-based pairings, protected administrator controls, structured responses, consent, feedback, blocking, reporting, moderation, and audit. The MVP provides no direct member chat and uses no machine learning.

## 2. Requirement and decision labels

- **DECISION:** approved architecture target that implementation must follow.
- **ASSUMPTION:** temporary input requiring validation.
- **OPEN QUESTION:** unresolved decision requiring an owner.

Important open questions include exact age policy, verification method, gender/group inclusion policy, cancellation/refund windows, reveal behavior, event round format, data retention, production infrastructure, and notification channels. Agents must not silently convert an open question into a permanent decision.

## 3. User roles

| Role | Capabilities |
|---|---|
| Visitor | Public marketing and permitted event summaries; no community profiles |
| Member | Account/profile/preferences, safe discovery, bookings, own matches/responses, feedback, block/report |
| Organizer | Assigned existing-event lifecycle and attendance operations; no event creation or matchmaking-run access |
| Moderator | Reports/content, safety actions, profile hiding, restrictions, suspension, appeals |
| Support/Finance | Constrained support and reconciliation without broad profile access |
| Administrator | Event creation, matching runs/overrides/locks/publication, roles, configuration, rulesets, audit access, emergency controls |
| Service account | Machine identity with only required integration scopes |

Possession of a role never bypasses ownership, event scope, resource state, privacy policy, or audit requirements.

## 4. Data visibility model

| Classification | Examples | Member visibility | Owner/control |
|---|---|---|---|
| Private account PII | Legal name, email, phone, DOB, credentials, verification evidence | Never | Account; encrypted/restricted |
| Community profile | Nickname, approved age/age band, permitted gender, broad city, bio, interests, approved photos | Eligible authenticated members | Account allow-list and visibility policy |
| Private matching inputs | Partner preference, age range, intention, deal-breakers, lifestyle answers | Not public | Account source; Matchmaking snapshots |
| Event operations | Booking, attendance, participant code, operational notes | Restricted | Booking/Event and event-scoped organizer access |
| Payment metadata | Order, amount, currency, callback fingerprint, status, refund/reconciliation | Never | Payment and finance/support scopes |
| Safety data | Reports, evidence, moderator notes, enforcement, appeals | Never | Moderation and restricted audit access |

Public/community responses must use explicit field allow-lists. Never serialize internal account objects directly.

## 5. Complete member journey

```text
Discover -> register/consent -> verify -> complete safe profile
-> supply private preferences/questionnaire -> moderation/approval
-> community discovery -> discover event -> time-limited ticket hold
-> PayHere payment -> confirmed booking -> eligible matching pool
-> deterministic matching run -> administrator review/override/lock
-> policy-limited pairing information -> attend event
-> structured continue/switch/interest response
-> consent-controlled reveal -> feedback/report/safety follow-up
```

At each sensitive step, revalidate account status, blocks, reports, event eligibility, and current state.

## 6. Logical architecture

```text
Member Web                 Protected Admin Web
     \                           /
      +-- WAF / Load Balancer / API Gateway --+
                           |
  +----------+---------+----------+---------+---------+----------+
  |          |         |          |         |         |          |
Account    Event    Matchmaking  Booking   Payment  Moderation  Notification
/Profile     |         |          |         |         |          |
  +----------+---------+----------+---------+---------+----------+
                           |
                       RabbitMQ

Each service -> independently owned PostgreSQL database
Optional safe read cache -> Redis
Approved profile media -> private object storage
Payment provider -> PayHere over TLS
Telemetry -> centralized logs, metrics, and traces
```

## 7. Technology decisions

| Area | Target |
|---|---|
| Frontend | React, TypeScript, TanStack Router/Query/Form/Table as needed, Zod, accessible design system |
| Frontend tooling | Bun package manager and script runner; Vite development server and production bundler |
| Backend | Go; prefer `net/http` or `chi`, `pgx`, and `sqlc` unless an ADR changes it |
| Persistence | PostgreSQL database/user per service; PostGIS for approved geographic discovery |
| Messaging | RabbitMQ, durable queues, retry/DLQ, transactional outbox/inbox, AsyncAPI |
| API | REST/JSON through canonical `/api/v1`; OpenAPI is authoritative |
| Payment | PayHere adapter isolated in Payment Service |
| Containers | OCI/Docker; Docker Compose for local integration |
| Delivery | GitHub Actions, independent images, path-aware CI, staged deployment |
| Observability | OpenTelemetry plus selected logs, metrics, traces, dashboards, alerts |

## 8. Service boundaries

### API Gateway

Owns edge routing, TLS/WAF integration, request IDs, coarse authentication, throttling, CORS, request-size limits, and temporary aliases. It owns no business data and cannot replace service authorization.

### Account/Profile Service

Owns identity, credentials, verification, roles, sessions, lifecycle, community profile, visibility, private source preferences, blocks, and profile moderation status. It does not own bookings, payments, capacity, pairings, or safety-case evidence.

### Event Service

Owns event catalog/lifecycle, organizer assignment, venue, schedule, price, registration window, configured capacity/policy, and discovery. It owns configured limits but not consumed capacity.

### Booking Service

Owns ticket holds, bookings, consumed capacity, allocation categories where approved, expiry, cancellation, attendance, and check-in. It is authoritative for confirmed event participation and stores the immutable price/currency snapshot.

The first executable slice implements Event-derived immutable pricing, atomic capacity holds, owner-authorized Payment snapshots, expiry, Payment completion inbox consumption, late-payment review, and outbox publication. Cancellation, waitlist, attendance, and full policy integration remain incomplete.

### Payment Service

Owns PayHere initiation, provider orders, state, callback fingerprints, verification, reconciliation, refunds, and payment audit. It never trusts client amount/currency and never allocates capacity.

The first executable slice implements initiation, callback verification/replay evidence, member status, audit, transactional outbox persistence, and publisher-confirmed RabbitMQ relay. Booking integration/consumption, scheduled reconciliation, refunds, and production delivery remain incomplete, so this slice is not yet an end-to-end payment capability.

### Matchmaking Service

Owns participant snapshots, eligibility, component scores, weighted total, optimizer, runs, suggestions, administrator overrides, locks, responses, reveal consent, and feedback. It does not own source profiles or booking/payment state.

The current executable prototype uses Matchmaking-owned deterministic fixtures to simulate future consumed Account/Event/Booking/Moderation facts. It validates Account-issued ES256 tokens, stores data in its independent PostgreSQL database, and exposes an administrator-only run lifecycle plus participant-scoped published results. Production integration will replace fixture projection writes with inbox-deduplicated events without changing ownership.

### Notification Service

Owns templates, delivery requests/attempts, provider results, retries, suppression, and channel preferences. It normally consumes facts outside the critical transaction path.

The executable slice consumes minimum-safe Account and Booking facts containing a recipient account ID, deduplicates each fact in a service-owned PostgreSQL inbox, selects versioned `en-LK` templates, applies account suppression/preferences, and records leased delivery attempts with retry/dead-letter state. Each non-suppressed delivery also creates an independently readable member-feed item. The JWT-protected API derives ownership from the Account token subject and exposes only rendered safe template text, category, time, read state, and an allow-listed application path. Local development uses a no-contact provider sink. Production email/SMS/push delivery is disabled until channel policy, an approved provider, credentials, and a constrained authenticated Account contact-resolution contract are accepted; destinations are not added to RabbitMQ facts.

### Moderation/Safety Service

Owns reports, cases, evidence references, risk classification, enforcement, appeals, and restricted audit. It may start as an isolated module, but the boundary must permit later extraction.

## 9. Communication model

Use synchronous REST when an immediate result is required or a narrow validation protects an invariant. Avoid long synchronous chains. Use RabbitMQ for completed facts, projections, notifications, matching-pool updates, deactivation propagation, and payment-to-booking confirmation.

Canonical event envelope:

```json
{
  "eventId": "01J...",
  "eventType": "PaymentCompleted",
  "schemaVersion": 1,
  "occurredAt": "2026-08-27T10:30:00Z",
  "aggregateId": "payment-id",
  "correlationId": "request-id",
  "causationId": "provider-callback-id",
  "actorId": "system:payhere",
  "payload": {}
}
```

Producers persist business state and outbox in one transaction. Consumers persist inbox deduplication and their business update in one transaction. Duplicate delivery is normal and safe.

## 10. Core API map

OpenAPI becomes authoritative when specifications exist.

| Method | Endpoint | Owner | Purpose |
|---|---|---|---|
| POST | `/api/v1/auth/register` | Account | Register |
| POST | `/api/v1/auth/login` | Account | Access and rotating refresh session |
| POST | `/api/v1/auth/refresh` | Account | Rotate tokens/session |
| GET/PATCH | `/api/v1/users/me` | Account | Self account/profile |
| PUT | `/api/v1/users/me/preferences` | Account | Private preferences |
| GET | `/api/v1/community/profiles` | Account | Safe discovery |
| POST | `/api/v1/blocks` | Account | Block member |
| GET/POST | `/api/v1/events` | Event | Discover/create events |
| GET/PATCH | `/api/v1/events/{eventId}` | Event | Read/update event |
| POST | `/api/v1/bookings` | Booking | Create hold |
| GET | `/api/v1/bookings/{bookingId}` | Booking | Read authorized booking |
| POST | `/api/v1/bookings/{bookingId}/cancel` | Booking | Cancel/request cancellation |
| POST | `/api/v1/payments/initiate` | Payment | Start PayHere from booking ID |
| POST | `/api/v1/payments/payhere/callback` | Payment | Provider callback |
| GET/POST | `/api/v1/events/{eventId}/matching-runs` | Matchmaking | List/generate runs |
| POST | `/api/v1/matching-runs/{runId}/lock` | Matchmaking | Lock reviewed run |
| POST | `/api/v1/matches/{matchId}/response` | Matchmaking | Structured response |
| POST | `/api/v1/matches/{matchId}/reveal-consent` | Matchmaking | Record consent |
| GET | `/api/v1/notifications` | Notification | List own in-app notifications |
| PATCH | `/api/v1/notifications/{notificationId}/read` | Notification | Mark an owned notification read |
| POST | `/api/v1/notifications/read-all` | Notification | Mark all own notifications read |
| POST | `/api/v1/reports` | Moderation | Report behavior/content |

Self-service uses authenticated subject identity. Caller-provided account IDs are not trusted.

## 11. Event catalog

| Event | Producer | Principal consumers |
|---|---|---|
| `AccountRegistered` | Account | Notification |
| `ProfileApproved` / `ProfileHidden` | Account/Moderation | Matchmaking, Notification |
| `AccountDeactivated` / `AccountRestricted` | Account/Moderation | Booking, Matchmaking, Notification |
| `EventCreated` / `EventUpdated` / `EventCancelled` | Event | Booking, Matchmaking, Notification |
| `EventCapacityChanged` | Event | Booking |
| `BookingPending` | Booking | Payment, Notification |
| `BookingConfirmed` | Booking | Matchmaking, Event projection, Notification |
| `BookingCancelled` / `HoldExpired` | Booking | Payment, Matchmaking, Event projection, Notification |
| `PaymentInitiated` / `PaymentCompleted` / `PaymentFailed` | Payment | Booking, Notification |
| `PairingsGenerated` / `PairingsLocked` | Matchmaking | Notification, audit/projection |
| `MatchResponseRecorded` / `RevealConsentGranted` | Matchmaking | Notification, analytics, moderation as needed |
| `ReportCreated` / `ModerationActionApplied` | Moderation | Account, Matchmaking, Notification |

Events contain only minimum identifiers and safe fields. Notification currently binds only supported Account/Booking routing keys with a safe recipient account ID; Event/Payment/Matchmaking/Moderation recipient expansion requires an explicit minimum-safe contract or projection.

## 12. State models

```text
Event: DRAFT -> PUBLISHED -> REGISTRATION_OPEN -> REGISTRATION_CLOSED
       -> MATCHING -> ONGOING -> COMPLETED; approved states -> CANCELLED

Booking: PENDING_PAYMENT -> CONFIRMED | EXPIRED | CANCELLED
         CONFIRMED -> CANCELLED; mismatch/late success -> PAYMENT_REVIEW

Payment: PENDING -> COMPLETED | FAILED
         COMPLETED -> REFUND_PENDING -> REFUNDED; mismatch -> REVIEW

Matching run: DRAFT -> GENERATED -> UNDER_REVIEW -> LOCKED -> PUBLISHED
              rerun creates a new immutable version

Moderation: OPEN -> TRIAGED -> INVESTIGATING -> ACTIONED | DISMISSED
            ACTIONED -> APPEALED -> UPHELD | REVERSED
```

## 13. Booking and PayHere flow

```text
Member -> Booking: create hold with Idempotency-Key
Booking: validate policy and atomically reserve capacity
Booking: persist PENDING_PAYMENT + immutable price snapshot + outbox
Booking -> RabbitMQ: BookingPending
Member -> Payment: initiate with bookingId only
Payment: verify eligibility; create unique PayHere order/hash
PayHere -> Payment: callback
Payment: verify signature, merchant, order, amount, currency, state, replay
Payment: persist callback + COMPLETED + outbox atomically
Payment -> RabbitMQ: PaymentCompleted
Booking: deduplicate and transition to CONFIRMED
Booking -> RabbitMQ: BookingConfirmed
Matchmaking: update eligible participant projection
```

Failure rules:

- Duplicate callback is acknowledged without repeated transition/event.
- Broker outage leaves state and outbox committed for later relay.
- Provider timeout remains pending and is reconciled; never assume failure.
- Late success after expiry enters payment review/refund policy.
- Last-seat contention is resolved by one atomic Booking allocation decision.

## 14. Matchmaking summary

The complete specification is [`../matchmaking/README.md`](../matchmaking/README.md).

```text
Confirmed participants -> hard filters -> component scores -> weighted total
-> event-wide optimizer -> administrator review/override -> immutable lock
-> responses -> consent-controlled reveal -> feedback
```

The engine is deterministic, versioned, reproducible, explainable, and contains no ML or inferred sensitive traits.

## 15. Data and consistency

See [`../data/README.md`](../data/README.md). Strong consistency is required inside the owner for credentials/sessions/restrictions, capacity/booking, payment verification/state, pairing locks/responses, and moderation actions. Eventual consistency is acceptable for discovery projections, notifications, dashboards, analytics, and non-authoritative availability. Critical commands revalidate against the owner.

## 16. Authentication and authorization

- Account signs short-lived RS256/ES256 access tokens and publishes JWKs.
- Refresh sessions rotate, are revocable, and detect reuse.
- Claims include `sub`, roles/scopes, token version, issuer/audience, issue/expiry times, and no unnecessary PII.
- Gateway applies coarse policy; services enforce ownership, role, event scope, state, and domain constraints.
- Admin/service access uses explicit scopes and audit.
- Deactivation revokes sessions first, publishes the fact, then follows retention-aware anonymization/deletion.

## 17. Privacy, moderation, and safety

- Validate/scan profile text and media; quarantine contact information or unsafe content.
- Provide block/report from profile and match surfaces.
- A block applies to discovery, eligibility, pairing, reveal, and relevant notifications.
- Notification APIs derive the recipient from the authenticated subject, conceal other-member item identifiers, and never expose provider destinations, delivery errors, private preferences, or moderation evidence.
- Moderators can hide profiles, suspend accounts, exclude participation, invalidate unpublished pairings, and prevent reveal.
- Reporter identity is not shown to the reported member.
- Admin/organizer actions store actor, target, reason, prior/new state, time, and correlation ID.
- Event emergency, venue, check-in, and escalation policy must be approved before launch.

## 18. Observability and planning SLOs

Every service emits sanitized structured logs, rate/error/latency/saturation metrics, database and broker signals, domain metrics, OpenTelemetry traces, and separate business audit records.

Planning targets requiring approval:

- Public API availability 99.9% monthly.
- Discovery p95 below 300 ms.
- Booking command p95 below 500 ms excluding payment UI.
- Callback validation/persistence p95 below 2 seconds.
- Payment-completed to booking-confirmed p95 below 10 seconds.
- RPO at most 5 minutes and RTO at most 30 minutes.

## 19. Deployment topology

```text
Internet -> DNS/CDN(optional) -> WAF/load balancer -> API Gateway x2+
-> independently scaled stateless apps/services
-> isolated managed PostgreSQL databases/users
-> managed RabbitMQ, optional Redis, private object storage
-> secret manager and centralized telemetry -> PayHere over TLS
```

Local development uses Docker Compose. Production may begin on a managed container service or well-operated VMs. Kubernetes is optional and requires an ADR based on real operational need.

## 20. Scalability, recovery, and architecture evolution

- Scale discovery, booking, callbacks, matching workers, and notifications independently.
- Use stateless APIs, bounded database pools/workers, timeouts, transient retries with jitter, circuit breakers, and bulkheads.
- Never use Redis as authority for capacity, payment, restriction, or pairing lock.
- Every queue has retry, DLQ, replay, retention, dashboard, alert, and owner definitions.
- Back up every database independently, support point-in-time recovery, and perform quarterly restore drills.
- Before changing ownership, consistency, security, communication, deployment, or major technology: add an ADR, define compatibility/migration, update canonical docs/contracts/tests, record before/after, then implement.
