# Moderation Local Operations

**Owner:** Safety/Moderation service team  
**Scope:** fictional local-development data only

Start the backend with `docker compose up --build -d`. Expected processes are `postgres-moderation`, successful `moderation-migrate`, `moderation-api`, `moderation-outbox-relay`, and `moderation-expiry-worker`. Readiness is available at `http://localhost:8087/health/ready`.

Run Compose commands from the repository root. `infrastructure/compose/moderation.compose.yml` is an include fragment that references the shared Account and RabbitMQ services; it is not a standalone Compose project.

Use Account-issued development tokens and the canonical OpenAPI contract. Do not copy access tokens, reporter identity, descriptions, evidence references, private notes, or raw database rows into logs or tickets. Structured logs contain operation metadata only.

```powershell
docker compose ps -a
docker compose logs --tail 100 moderation-api moderation-outbox-relay moderation-expiry-worker
Invoke-WebRequest -UseBasicParsing http://localhost:8087/health/ready
```

Run fast tests from the Go module directory:

```powershell
cd services\moderation-service
go test -count=1 ./...
go vet ./...
```

Run the PostgreSQL component test from the repository root. It creates and removes its own schema and does not reuse application tables:

```powershell
docker compose up -d postgres-moderation
$env:MODERATION_TEST_DATABASE_URL = "postgres://matchmate:matchmate@127.0.0.1:5439/matchmate_moderation?sslmode=disable"
go -C services/moderation-service test -count=1 ./internal/store/postgres
Remove-Item Env:MODERATION_TEST_DATABASE_URL
```

For a manual workflow, obtain real Account-issued member and moderator tokens. Submit a report, assign its case, move it to `INVESTIGATING`, then either apply an action or move it to `DISMISSED`. Use the exact URLs from `contracts/openapi/moderation-v1.yaml`; do not paste Markdown link syntax into PowerShell.

After an action is applied, query the private current-state endpoint from the local workstation with the development-only `MODERATION_SERVICE_TOKEN` value in the ignored root `.env` (or the Compose default). For example, set `$safetyToken` in the current PowerShell session and call `Invoke-RestMethod -Uri "http://localhost:8087/api/v1/internal/safety/ACCOUNT_UUID?eventId=EVENT_UUID" -Headers @{"X-Moderation-Service-Token"=$safetyToken}`. The response contains booleans only. Never paste a real production token in a command transcript. The private route must not be exposed publicly in production.

Apply `ACCOUNT_SUSPENSION` or `PROFILE_RESTRICTION` to an Account UUID with scope `ALL`; use an event UUID or `ALL` for `EVENT_EXCLUSION`, `MATCHMAKING_EXCLUSION`, and `REVEAL_PREVENTION`. A suspended member can password-sign-in to obtain an `appeal`-only token. With that token, `GET /api/v1/moderation/actions/mine` returns the member's active action IDs; `POST /api/v1/moderation/actions/{actionId}/appeals` submits a reason. A moderator can uphold or reverse it. Re-query the current-state endpoint and retry Booking/Matchmaking/Notification actions after reversal; a reversed action stops applying without deleting audit history. The member must refresh or sign in again to regain ordinary roles.

Booking administrative registration rows now include `entryEligible`. Treat `false` as an entry prohibition even when the historical booking state remains `CONFIRMED`. Do not cancel a paid booking or release capacity automatically until the approved policy covers exclusions. A Moderation outage makes safety-sensitive operations return unavailable or retry; investigate the service rather than bypassing it. See [ADR-0005](../adr/0005-authoritative-moderation-safety-reads.md).

An unavailable database makes readiness fail. RabbitMQ failure must not roll back reports or actions: facts remain in the Moderation-owned outbox and publish after relay recovery. Never delete or rewrite unpublished rows. Action expiry is idempotent through row locking and state checks.

Rollback stops long-running Moderation processes and preserves PostgreSQL evidence, audit, and outbox state:

```powershell
docker compose stop moderation-api moderation-outbox-relay moderation-expiry-worker
```

Do not remove the Moderation volume as operational rollback. Production alerts, distributed rate-limit response, evidence-access escalation, downstream-consumer lag/DLQ handling, retention/legal-hold procedures, and restore drills remain required before launch.
