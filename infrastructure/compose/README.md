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

The Event Service has an independent PostgreSQL database and migration job:

```powershell
docker compose -f infrastructure/compose/event.compose.yml --profile api up -d --build
docker compose -f infrastructure/compose/event.compose.yml ps -a
docker compose -f infrastructure/compose/event.compose.yml logs event-migrate
```

The `api` profile starts Event API on port 8082 and discovers the Account development signing key from the Account API on the host. Start Account API before using protected organizer commands. The optional `messaging` profile starts the Event outbox relay and expects RabbitMQ from the Account development topology on host port 5672. Full commands and checks are documented in `services/event-service/README.md`.

The root command also starts the Matchmaking prototype on port `8083` with its independent PostgreSQL database on development port `5435`. Startup order is `postgres-matchmaking (healthy) -> matchmaking-migrate (exit 0) -> matchmaking-seed (exit 0) -> matchmaking-api`. Its versioned development cohort is service-owned fixture data and never runs in production manifests.

The root command now includes `booking-payment.compose.yml`. Booking uses API port `8085` and PostgreSQL port `5437`; Payment uses API port `8084` and PostgreSQL port `5436`. It discovers Account ES256 keys from Account JWKS and uses Event API on port `8082` plus RabbitMQ from the Account topology. The local file supplies non-functional PayHere placeholders unless real sandbox values are provided. Both outbox relays and the Booking Payment consumer run by default so a verified payment fact can confirm a booking. Do not start the included files as separate Compose projects while the root stack is running, because their host ports are already owned by the root project. See `docs/runbooks/booking-payment-local.md`.

The root command also includes `notification.compose.yml`. Notification owns PostgreSQL port `5438`, exposes operational health on API port `8086`, consumes supported Account/Booking facts from RabbitMQ, and processes deliveries with a development-only sink. Startup order is `postgres-notification (healthy) -> notification-migrate (exit 0) -> notification-api/notification-consumer/notification-worker`. The consumer reaches the root RabbitMQ through `host.docker.internal`; when RabbitMQ is not ready it exits and Compose restarts it. No real recipient destination or provider credential is configured locally. See `docs/runbooks/notification-local.md`.
