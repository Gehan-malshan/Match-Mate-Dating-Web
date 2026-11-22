# Notification Local Runbook

## Owner and purpose

**Owner:** Notification Service team

Use this runbook to start and verify the Account/Booking notification slice, authenticated in-app member feed, real SMTP sender readiness, preference behavior, and one-delivery recovery. This procedure uses fictional data only. It does not authorize sending to real members or expose recipient addresses.

## Safety warnings

- Do not add email addresses, phone numbers, rendered bodies, tokens, payment details, or private profile/safety data to RabbitMQ test events or log output.
- Do not delete/purge the shared RabbitMQ exchange or queues.
- Do not manually change inbox/delivery state to make a test pass.
- Preserve the Notification PostgreSQL volume during ordinary stop/restart operations.
- `dev-sink` must never be used in staging/production.
- Do not set `SMTP_SENDER_VERIFIED=true` until the provider dashboard has verified the sender/domain and the target-environment preflight succeeds.
- A replay can resend an external message. Correct the cause first, obtain the required operational approval, and use a specific delivery ID only.

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

## Production SMTP sender readiness

**Required access:** deployment secret manager, SMTP provider dashboard, and the target environment's restricted operator shell. Do not paste SMTP credentials into tickets, terminals shared with others, or Git.

1. Verify the sender domain/address in the email provider dashboard and preserve the provider's verification evidence in the restricted deployment record.
2. Add the following target-environment secrets/configuration: `NOTIFICATION_PROVIDER=smtp`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_SENDER_VERIFIED=true`, `INTERNAL_SERVICE_TOKEN`, `SECURITY_MAIL_TOKEN`, and HTTPS `SECURITY_PUBLIC_URL`.
3. Set `NOTIFICATION_PREFERENCE_CATEGORIES` only after Product/Safety approves which Account/Booking categories a member may control. Security verification and password recovery always bypass this preference setting.
4. Deploy the new Notification image and run the no-send sender preflight in the target environment:

```powershell
docker compose run --rm --no-deps --entrypoint /notification-provider-preflight notification-api
```

The command checks STARTTLS, credentials, and that the SMTP server accepts the configured sender envelope. It sends no recipient/message. A failure leaves normal delivery disabled for the rollout: fix the sender/provider configuration and rerun preflight.

5. Use an approved fictional/internal canary account to create one safe Account or Booking fact. Confirm one provider acceptance, one Notification `DELIVERED` attempt, and the expected member feed item. Do not record the recipient address or rendered body in evidence.
6. If the canary fails, stop the worker or revert to the previously verified configuration. Do not turn on `dev-sink` in production and do not edit delivery rows directly.

## Recipient preferences

Members manage only their own email preference categories at **Updates** → **Choose your delivery preferences**. The UI is backed by member-authenticated GraphQL and Notification's owner-scoped REST API; it never displays an email address, provider reference, or delivery failure. In-app feed entries continue independently of email preference. Verify a toggle with a fictional member and confirm `notification_preference` stores only account ID/channel/category/decision/time.

## Terminal delivery replay

Use this only after the provider/outage/root cause is fixed and a second operator has reviewed the target and reason.

1. Identify one terminal delivery without querying recipient destinations or message body:

```powershell
docker compose exec -T postgres-notification sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -P pager=off -c "SELECT id,source_event_type,state,attempt_count,last_error_code,created_at FROM notification_delivery WHERE state IN ('\''PERMANENTLY_FAILED'\'','\''DEAD_LETTERED'\'') ORDER BY updated_at DESC LIMIT 20;"'
```

2. Run the owner command with the exact UUID, named operator, specific reason, and explicit confirmation:

```powershell
docker compose run --rm --no-deps --entrypoint /notification-delivery-replay notification-api --delivery-id <uuid> --actor <operator-id> --reason "Provider sender restriction was corrected" --confirm REPLAY
```

3. Verify that the same delivery is `RETRY_SCHEDULED`, its original ID is unchanged, and one `notification_delivery_replay` audit row exists. The worker retains ordinary safety checks and provider idempotency.
4. Never replay `SUPPRESSED`, `PROCESSING`, `PENDING`, or `DELIVERED` rows. Do not use RabbitMQ payload replay for a completed Notification delivery; inbox/business-key deduplication and provider side effects make it unsafe for this purpose.

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

Confirm Account API is running so Notification can load JWKS, GraphQL gateway is healthy at `http://localhost:8080/health/live`, and the frontend uses `VITE_GRAPHQL_API_URL=http://localhost:8080/graphql`. A 401 requires a fresh member login; do not copy access tokens into tickets or logs.

### Dead-letter queue contains messages

Open RabbitMQ management at `http://localhost:15672` using the documented development credentials and inspect `notification.business.v1.dlq` counts/headers only. Do not copy payloads into tickets/chat. Correct the producer/contract or consumer defect before any controlled replay; replay must preserve the original `eventId` so inbox deduplication remains effective.

## Verification and rollback

After recovery, verify health, consumer/worker logs, and that each source event has at most one delivery/business key. Normal rollback stops Notification only and preserves its database:

```powershell
docker compose stop notification-api notification-consumer notification-worker
```

Account, Event, Booking, Payment, and Matchmaking transactions continue while Notification is stopped; RabbitMQ retains bound supported facts for later consumption. Do not remove the Notification volume as rollback.

Provider dashboard outage response, bounce/complaint webhook processing, metrics/traces/alerts, retention approval, and named escalation contacts remain required before broad production launch.

**Last locally exercised:** 2026-11-23
**Next review:** before the first real provider canary and after any provider/domain change
