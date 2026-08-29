# Notification Local Runbook

## Owner and purpose

**Owner:** Notification Service team

Use this runbook to start and verify the first Account/Booking notification slice, inspect delivery state, and diagnose a local consumer/worker failure. This procedure is for fictional development data only. It does not prove production email/SMS delivery.

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

Read-only database checks:

```powershell
docker compose exec -T postgres-notification sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -P pager=off -c "SELECT source_event_type,state,attempt_count,created_at,delivered_at FROM notification_delivery ORDER BY created_at DESC LIMIT 20;"'
docker compose exec -T postgres-notification sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -P pager=off -c "SELECT outcome,error_code,completed_at FROM notification_delivery_attempt ORDER BY completed_at DESC LIMIT 20;"'
docker compose exec -T postgres-notification sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -P pager=off -c "SELECT count(*) AS processed_events FROM notification_inbox;"'
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

### Dead-letter queue contains messages

Open RabbitMQ management at `http://localhost:15672` using the documented development credentials and inspect `notification.business.v1.dlq` counts/headers only. Do not copy payloads into tickets/chat. Correct the producer/contract or consumer defect before any controlled replay; replay must preserve the original `eventId` so inbox deduplication remains effective.

## Verification and rollback

After recovery, verify health, consumer/worker logs, and that each source event has at most one delivery/business key. Normal rollback stops Notification only and preserves its database:

```powershell
docker compose stop notification-api notification-consumer notification-worker
```

Account, Event, Booking, Payment, and Matchmaking transactions continue while Notification is stopped; RabbitMQ retains bound supported facts for later consumption. Do not remove the Notification volume as rollback.

Production provider outage response, recipient contact investigation, provider reconciliation, guarded DLQ replay, dashboards/alerts, and escalation contacts remain required before production deployment.

**Last locally exercised:** 2026-08-29  
**Next review:** when a production notification provider/contact-resolution contract is selected
