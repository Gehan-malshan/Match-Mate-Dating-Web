# Notification Service

Owns versioned notification templates, idempotent delivery records, delivery attempts, retry/dead-letter state, suppression, channel/category preferences, and provider adapters. Business services remain independent of notification delivery success.

## Implementation status

The first executable vertical slice provides:

- An independently buildable Go module and OCI image.
- An independent PostgreSQL database and versioned migration.
- RabbitMQ consumption from the shared `matchmate.events` topic exchange.
- Inbox deduplication and a unique business delivery key.
- Active, versioned `en-LK` Account and Booking templates.
- Account-deactivation suppression.
- Database-backed delivery claims, leases, attempts, exponential retry scheduling, permanent-failure state, and exhausted/dead-letter state.
- A RabbitMQ dead-letter queue for malformed or oversized business events.
- A privacy-safe development sink that records successful attempts without resolving or logging email/phone destinations.
- Liveness/readiness endpoints and sanitized structured logs.
- An authenticated member-owned in-app feed with cursor pagination, unread counts, idempotent mark-read operations, and historical template-version rendering.
- A Midnight Chemistry member notification bell, recent-items popover, full history page, and polling-based popup toasts.

This is an executable in-app development slice, not production email/SMS completion. Real email remains intentionally deferred until provider selection, credentials, constrained Account contact resolution, and channel/legal policy are approved. Consent/preference source integration, retry replay operations, metrics/traces, and broader Event/Payment/Matchmaking/Moderation recipients also remain required.

## Safe initial boundary

Notification consumes only facts containing a safe recipient account ID. It does not copy email, phone, profile, payment, preference, or moderation evidence into its database or event payloads.

Supported facts:

| Routing key | Result |
|---|---|
| `account.AccountRegistered` | `account-welcome` delivery |
| `account.AccountVerified` | `account-verified` delivery |
| `account.ProfileApproved` | `profile-approved` delivery |
| `account.ProfileHidden` | `profile-hidden` delivery |
| `account.AccountDeactivated` | Suppress pending/future deliveries |
| `booking.BookingPending` | `booking-pending` delivery |
| `booking.BookingConfirmed` | `booking-confirmed` delivery |
| `booking.BookingCancelled` | `booking-cancelled` delivery |
| `booking.HoldExpired` | `booking-hold-expired` delivery |
| `booking.BookingPaymentReviewRequired` | `booking-payment-review` delivery |

Payment facts currently omit the owning account ID. Event cancellation and Matchmaking publication can involve multiple recipients. Those facts are not subscribed to until minimum-safe recipient contracts/projections are approved.

## Processes

```text
notification-api       liveness/readiness on port 8086
notification-migrate   applies ordered service-owned migrations
notification-consumer  Account/Booking RabbitMQ inbox consumer
notification-worker    claims and delivers due records
postgres-notification  service-owned PostgreSQL on local port 5438
```

The API exposes public operational health plus a JWT-protected member feed:

```text
GET /health/live
GET /health/ready
GET /api/v1/notifications?limit=20&cursor=...
GET /api/v1/notifications/unread-count
PATCH /api/v1/notifications/{notificationId}/read
POST /api/v1/notifications/read-all
```

Member endpoints validate Account-issued ES256 tokens and derive ownership only from the token subject. They never accept a caller-selected account ID, never expose delivery/provider diagnostics, and conceal another member's notification as not found. The canonical contract is `contracts/openapi/notification-v1.yaml`. There is no admin delivery-list API.

## Delivery flow

```text
Business outbox -> matchmate.events -> notification.business.v1
-> validate envelope/schema/recipient -> notification_inbox
-> select active template -> suppression/preference check
-> notification_delivery PENDING | SUPPRESSED
-> worker lease -> render allow-listed variables -> provider adapter
-> DELIVERED | RETRY_SCHEDULED | PERMANENTLY_FAILED | DEAD_LETTERED
-> append notification_delivery_attempt
```

In the same event-consumer transaction, every non-suppressed delivery receives one `notification_feed_item`. The member API joins that item to the immutable delivery/template version, renders only allow-listed variables, and returns a safe title, message, category, timestamp, and allow-listed application path. Email/provider delivery state and in-app read state are intentionally independent.

The inbox row and delivery/suppression change commit in one PostgreSQL transaction. Duplicate RabbitMQ delivery is acknowledged without creating a second delivery. External providers must accept `delivery.id` as their idempotency key because a worker can crash after provider acceptance but before local completion.

## Data ownership

Migration 1 owns:

- `notification_template`
- `notification_delivery`
- `notification_delivery_attempt`
- `notification_preference`
- `notification_suppression`
- `notification_inbox`

Migration 2 adds `notification_feed_item` and backfills existing non-suppressed development deliveries. It stores only delivery/recipient identifiers, read time, and creation time; member-visible text remains versioned in the existing template/delivery snapshot.

Key invariants:

- `business_key` is unique for one event/template/recipient outcome.
- Inbox uniqueness is `(consumer,event_id)`.
- Template version/key/locale/channel is unique.
- Only allow-listed variables may render.
- Provider errors are stored as stable sanitized codes, not raw response bodies.
- Deactivated accounts cannot receive a newly consumed delivery.
- A provider outage never rolls back Account or Booking state.

The migration is additive and has no production backfill. Rollback stops Notification processes and retains the isolated database for delivery/audit diagnosis; the down migration is for disposable development databases only.

## Development provider

`NOTIFICATION_PROVIDER=dev-sink` performs no external network call. It renders the selected template, stores a deterministic `dev-sink:<delivery-id>` provider reference, and logs only delivery/template/event metadata. It never logs message content or recipient contact information.

Configuration rejects `dev-sink` when `APP_ENV` is not `development` or `test`. This prevents an accidental production deployment that silently discards real messages.

## Configuration

| Variable | Development default | Purpose |
|---|---|---|
| `APP_ENV` | `development` | Environment safety policy |
| `NOTIFICATION_HTTP_ADDRESS` | `:8086` | Operational HTTP listener |
| `NOTIFICATION_DATABASE_URL` | Required | Notification PostgreSQL connection |
| `NOTIFICATION_RABBITMQ_URL` | Local RabbitMQ | Broker connection |
| `NOTIFICATION_EVENT_EXCHANGE` | `matchmate.events` | Source topic exchange |
| `NOTIFICATION_QUEUE` | `notification.business.v1` | Durable consumer queue |
| `NOTIFICATION_DEAD_LETTER_EXCHANGE` | `matchmate.notification.dlx` | Rejected-message exchange |
| `NOTIFICATION_DEAD_LETTER_QUEUE` | `notification.business.v1.dlq` | Operations dead-letter queue |
| `NOTIFICATION_PROVIDER` | `dev-sink` | Provider adapter |
| `NOTIFICATION_DEFAULT_LOCALE` | `en-LK` | Template locale |
| `NOTIFICATION_MAX_ATTEMPTS` | `5` | Maximum provider attempts |
| `NOTIFICATION_POLL_INTERVAL` | `1s` | Worker polling interval |
| `NOTIFICATION_LEASE_DURATION` | `30s` | Crash-recovery delivery lease |
| `NOTIFICATION_RETRY_BASE` | `1m` | Exponential retry base |
| `NOTIFICATION_JWT_PUBLIC_KEY_PEM` | Empty | Optional static Account ES256 public key |
| `ACCOUNT_JWKS_URL` | Account JWKS URL | Account signing-key discovery |
| `JWT_ISSUER` | `matchmate-account` | Required access-token issuer |
| `JWT_AUDIENCE` | `matchmate-api` | Required access-token audience |
| `ALLOWED_ORIGINS` | Member Vite origins | Exact browser CORS allow-list |

Real provider credentials must use the deployment secret manager and must never be committed or placed in Compose defaults.

## Local development

From the repository root:

```powershell
docker compose up --build -d
docker compose ps -a
Invoke-WebRequest -UseBasicParsing -Uri "http://localhost:8086/health/ready"
docker compose logs --tail 100 notification-consumer notification-worker
```

`notification-migrate` should show `Exited (0)`. The API, consumer, worker, and PostgreSQL container should show `Up`.

Run service checks:

```powershell
go -C services/notification-service test ./...
go -C services/notification-service vet ./...
```

Run the disposable PostgreSQL component test against the local Notification database:

```powershell
$env:NOTIFICATION_TEST_DATABASE_URL = "postgres://matchmate:matchmate@localhost:5438/matchmate_notification?sslmode=disable"
go -C services/notification-service test ./internal/store/postgres -count=1
Remove-Item Env:NOTIFICATION_TEST_DATABASE_URL
```

The test creates and removes only its own randomly named schema.

Browser verification:

1. Run `bun run dev` and sign in at `http://localhost:5173/login`.
2. Use the bell in the member header or open `/app/notifications`.
3. Create/cancel a booking in another tab while the member page remains open.
4. Within about 10 seconds, the unread badge and privacy-safe popup toast should appear.
5. Opening the item or using the read controls must update only that signed-in member's feed.

The frontend reads notifications through `VITE_GRAPHQL_API_URL`, which defaults to `http://localhost:8080/graphql`. The GraphQL gateway calls this service internally.

## Required production follow-up

- Approve email/SMS/push and legally required versus suppressible categories.
- Add a narrow authenticated Account contact-resolution API; do not place destinations in RabbitMQ facts.
- Implement an approved provider adapter with timeouts, provider idempotency, rate limits, and sandbox evidence.
- Add preference-update ingestion and unsubscribe/bounce suppression policy.
- Add Event/Payment/Matchmaking/Moderation recipient projections/contracts.
- Add broker retry topology, guarded DLQ inspection/replay tooling, dashboards, metrics, traces, alerts, and provider outage runbook evidence.
- Add real PostgreSQL/RabbitMQ component and failure tests to required CI.
- Define retention/deletion periods and backup/restore evidence.
- Add production real-time delivery only if polling no longer meets the approved experience/SLO; WebSocket/SSE is not required by the current slice.

Update this README, AsyncAPI, architecture/data/testing/security guides, Compose/runbooks, and pull-request before/after summary whenever behavior changes.
