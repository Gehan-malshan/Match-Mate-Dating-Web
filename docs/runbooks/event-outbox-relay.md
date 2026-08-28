# Event Outbox Relay Runbook

- **Owner:** Event Service team
- **Last tested:** 2026-08-28 using local Docker PostgreSQL/RabbitMQ; two lifecycle facts were publisher-confirmed and the unpublished backlog reached zero
- **Next review:** Before Phase 2 is marked complete

## Purpose and symptoms

Use this runbook when Event lifecycle facts remain unpublished, `outbox` age grows, relay logs report claim/publish/confirm errors, or RabbitMQ is unavailable. Member discovery and committed Event state may continue, but Booking, Matchmaking, and Notification projections can become stale.

## Access and safety

Use Event Service database and broker read/operate permissions only. Do not edit another service database, delete outbox rows, purge queues, or mark events published manually. Payloads contain identifiers and safe Event state only; still avoid copying them into public tickets or chat.

## Diagnosis

1. Check Event API readiness and PostgreSQL health.
2. Check the `event-outbox-relay` process/container and sanitized logs for `outbox_claim_failed`, `outbox_publish_failed`, `outbox_publish_not_acknowledged`, `outbox_publish_confirm_timeout`, or `outbox_mark_failed`.
3. Check RabbitMQ node/exchange availability, connection saturation, and the `matchmate.events` topic exchange without inspecting unrelated message payloads.
4. Measure unpublished count and oldest age using a read-only Event database query:

```sql
SELECT count(*) AS unpublished, min(occurred_at) AS oldest
FROM outbox
WHERE published_at IS NULL;
```

5. If rows are claimed, confirm whether `claimed_at` is older than one minute. The relay intentionally permits safe reclaim after that lease.

## Mitigation

1. Restore PostgreSQL or RabbitMQ connectivity first.
2. Restart only the Event relay. It reclaims abandoned rows after one minute and publishes persistent messages with broker confirms.
3. Observe unpublished count and oldest age decreasing.
4. Confirm downstream consumers deduplicate by `eventId`; duplicate delivery after a relay crash is expected and must be harmless.
5. If a specific message repeatedly fails, stop automated restart, preserve its event ID and sanitized error, and escalate for schema/consumer investigation. Do not delete or rewrite the record.

## Verification and recovery

- Event API state remains authoritative and unchanged.
- Unpublished outbox count returns to zero or an understood current baseline.
- RabbitMQ confirms succeed and no new publish/confirm errors appear.
- Queue lag and DLQ depth return to baseline after consumer recovery.
- Reconcile affected Event IDs in owner-service projections without cross-database writes.

Rollback the relay binary/container if failures began after deployment; keep the Event database schema and committed outbox rows for forward recovery. Preserve correlation IDs, event IDs, deployment version, timings, and sanitized logs for review. Escalate when broker confirms remain unavailable, backlog age breaches the approved alert threshold, unsupported schema versions appear, or recovery would require data mutation.
