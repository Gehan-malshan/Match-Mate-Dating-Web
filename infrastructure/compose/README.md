# Docker Compose

The Account/Profile development topology starts PostgreSQL and RabbitMQ, applies the versioned Account migration, and seeds shared fake logins automatically:

From the repository root, use the standard Docker Compose command:

```powershell
docker compose up --build -d
```

The root `compose.yaml` includes this Account/Profile development topology. The
explicit form remains available when working inside infrastructure:

```powershell
docker compose -f infrastructure/compose/account-profile.compose.yml up --build -d
```

Startup order is `postgres-account (healthy) -> account-migrate (exit 0) -> account-seed (exit 0)`. The migration and seed jobs are idempotent, so the command also works with an existing complete local schema. Inspect the result with:

```powershell
docker compose -f infrastructure/compose/account-profile.compose.yml ps -a
docker compose -f infrastructure/compose/account-profile.compose.yml logs account-migrate account-seed
```

`account-migrate` and `account-seed` should show `Exited (0)`; this is successful for one-shot jobs. Shared login details are documented in `services/account-service/README.md`. This automatic seed exists only in the development Compose file and must never be copied into staging or production manifests.
