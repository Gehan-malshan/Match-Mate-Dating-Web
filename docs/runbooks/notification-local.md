# Notification Local Runbook

## Owner and purpose

**Owner:** Notification Service team

Use this runbook to start and verify the Account/Booking notification slice and authenticated in-app member feed, inspect delivery/read state, and diagnose a local API/consumer/worker failure. This procedure is for fictional development data only. It does not prove production email delivery.

## Safety warnings

- Do not add email addresses, phone numbers, rendered bodies, tokens, payment details, or private profile/safety data to RabbitMQ test events or log output.
- Do not delete/purge the shared RabbitMQ exchange or queues.
- Do not manually change inbox/delivery state to make a test pass.
- Preserve the Notification PostgreSQL volume during ordinary stop/restart operations.
- `dev-sink` must never be used in staging/production.

## Start and health verification

From the repository root:

```powershell
docker compose up --build -d
docker compose ps -a
Invoke-WebRequest -UseBasicParsing -Uri "http://localhost:8086/health/live"
Invoke-WebRequest -UseBasicParsing -Uri "http://localhost:8086/health/ready"
```

Expected:

- `notification-migrate` is `Exited (0)`.
- `postgres-notification` is healthy.
- `notification-api`, `notification-consumer`, and `notification-worker` are `Up`.
- Both health endpoints return `204`.

## Delivery verification

Create/verify a fictional Account or create/cancel/confirm a development Booking through the owning service. Then inspect sanitized logs:

```powershell
docker compose logs --tail 100 notification-consumer notification-worker
```

Expected log progression:

```text
notification_event_processed delivery_created=true
notification_dev_sink_delivered
notification_delivery_processed state=DELIVERED
```

Do not expect an email: the initial provider is intentionally a development sink.

## Member web verification

1. Start the frontend with `bun run dev`, open `http://localhost:5173/login`, and sign in as `member@example.test`.
2. Confirm the authenticated header shows the notification bell and that `/app/notifications` loads only this account's history.
3. Keep that page open, then create or cancel a development booking in another tab.
4. Within about 10 seconds, verify the bell count changes and a dismissible popup appears.
5. Open the notification, mark it read, and use **Mark all as read**. Refresh and verify read state persists.
6. Sign in as another fictional member and confirm the first member's notifications are absent.

The first successful poll initializes popup state without replaying old unread items. A popup is expected only for a new item observed by a later poll. Browser requests must use the Account bearer access token; the API accepts no recipient account query/body field.

Read-only database checks:

```powershell
docker compose exec -T postgres-notification sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -P pager=off -c "SELECT source_event_type,state,attempt_count,created_at,delivered_at FROM notification_delivery ORDER BY created_at DESC LIMIT 20;"'
docker compose exec -T postgres-notification sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -P pager=off -c "SELECT outcome,error_code,completed_at FROM notification_delivery_attempt ORDER BY completed_at DESC LIMIT 20;"'
docker compose exec -T postgres-notification sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -P pager=off -c "SELECT count(*) AS processed_events FROM notification_inbox;"'
docker compose exec -T postgres-notification sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -P pager=off -c "SELECT recipient_account_id,(read_at IS NULL) AS unread,created_at FROM notification_feed_item ORDER BY created_at DESC LIMIT 20;"'
```

The final query counts processed inbox facts; it does not inspect RabbitMQ dead letters.

## Diagnosis

### Migration failed

```powershell
docker compose logs notification-migrate postgres-notification
```

Confirm PostgreSQL is healthy and migration 1 has not been manually modified. Fix the migration/code and use forward recovery. Do not mark `schema_migration` manually.

### Consumer repeatedly restarts

```powershell
docker compose logs --tail 200 notification-consumer rabbitmq
docker compose ps -a notification-consumer rabbitmq
```

Confirm RabbitMQ is healthy and host port `5672` is available. Restart only the consumer after the dependency is healthy:

```powershell
docker compose up -d notification-consumer
```

### Delivery remains pending/retry scheduled

```powershell
docker compose logs --tail 200 notification-worker
docker compose restart notification-worker
```

The worker lease expires safely and the same delivery ID is retried. Never create a replacement delivery manually.

### Bell or history does not load

```powershell
docker compose logs --tail 200 notification-api account-api
Invoke-WebRequest -UseBasicParsing -Uri "http://localhost:8086/health/ready"
```

Confirm Account API is running so Notification can load JWKS, the frontend uses `VITE_NOTIFICATION_API_URL=http://localhost:8086/api/v1`, and the browser origin is `http://127.0.0.1:5173` or `http://localhost:5173`. A 401 requires a fresh member login; do not copy access tokens into tickets or logs.

### Dead-letter queue contains messages

Open RabbitMQ management at `http://localhost:15672` using the documented development credentials and inspect `notification.business.v1.dlq` counts/headers only. Do not copy payloads into tickets/chat. Correct the producer/contract or consumer defect before any controlled replay; replay must preserve the original `eventId` so inbox deduplication remains effective.

## Verification and rollback

After recovery, verify health, consumer/worker logs, and that each source event has at most one delivery/business key. Normal rollback stops Notification only and preserves its database:

```powershell
docker compose stop notification-api notification-consumer notification-worker
```

Account, Event, Booking, Payment, and Matchmaking transactions continue while Notification is stopped; RabbitMQ retains bound supported facts for later consumption. Do not remove the Notification volume as rollback.

Production provider outage response, recipient contact investigation, provider reconciliation, guarded DLQ replay, dashboards/alerts, and escalation contacts remain required before production deployment.

**Last locally exercised:** 2026-08-30
**Next review:** when a production email provider/contact-resolution contract is selected
