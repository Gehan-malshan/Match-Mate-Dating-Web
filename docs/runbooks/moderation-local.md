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

An unavailable database makes readiness fail. RabbitMQ failure must not roll back reports or actions: facts remain in the Moderation-owned outbox and publish after relay recovery. Never delete or rewrite unpublished rows. Action expiry is idempotent through row locking and state checks.

Rollback stops long-running Moderation processes and preserves PostgreSQL evidence, audit, and outbox state:

```powershell
docker compose stop moderation-api moderation-outbox-relay moderation-expiry-worker
```

Do not remove the Moderation volume as operational rollback. Production alerts, distributed rate-limit response, evidence-access escalation, downstream-consumer lag/DLQ handling, retention/legal-hold procedures, and restore drills remain required before launch.
