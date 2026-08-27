# MatchMate Operational Runbooks

Every production service and critical cross-service workflow must have a runnable procedure that an operator unfamiliar with its implementation can follow safely.

## Required runbooks

- Service health/degraded/unavailable response.
- Deployment, smoke verification, progressive rollout, and rollback/forward recovery.
- PostgreSQL failover, backup, point-in-time restore, and quarterly restore drill.
- RabbitMQ outage, queue lag, retry/DLQ inspection, controlled replay, and poison-message handling.
- Outbox relay stall and inbox/consumer failure.
- PayHere callback failure, pending/mismatch/late payment, reconciliation, and refund.
- Booking capacity conflict and event cancellation.
- Matching-run failure, invalidation, rerun, lock conflict, and safety exclusion after lock.
- Notification provider outage and suppression/delivery investigation.
- Account compromise, PII exposure, secret rotation, moderation emergency, and event safety incident.

## Runbook template

Each runbook must include:

```text
Title and owner
Purpose and symptoms/alerts
Severity and business/user impact
Required access and safety warnings
Dashboards/log queries/metrics (without sensitive data)
Diagnosis decision tree
Step-by-step mitigation
Verification of business invariants
Communication/escalation contacts
Rollback or forward-recovery
Data reconciliation queries/procedures
Evidence to preserve and privacy restrictions
When to stop and escalate
Post-incident tasks and documentation/change updates
Last tested date and next review date
```

Never instruct an operator to directly edit another service's database or delete queues/data broadly. Use owner-service commands and verified, scoped recovery procedures. Material manual actions require audit and a second reviewer where risk justifies it.

