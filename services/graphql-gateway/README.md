# MatchMate GraphQL Gateway

The GraphQL gateway is the single browser-facing Backend for Frontend. It exposes `POST /graphql` on port `8080`, maps typed operations to the existing Go service REST APIs, forwards Account-issued bearer tokens and rotating refresh cookies, and performs an additional administrator-role check on privileged event and matchmaking operations.

It owns no database and no business state. Account, Event, Booking, Payment, Notification, Matchmaking, and Moderation remain authoritative for their domains. RabbitMQ communication is unchanged.

## Run

Normally run it with the complete stack:

```powershell
docker compose up --build -d
```

Health and development explorer:

```text
http://localhost:8080/health/live
http://localhost:8080/
http://localhost:8080/graphql
```

## Configuration

```text
GRAPHQL_HTTP_ADDRESS=:8080
GRAPHQL_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
ACCOUNT_API_URL=http://account-api:8081/api/v1
EVENT_API_URL=http://event-api:8082/api/v1
MATCHMAKING_API_URL=http://matchmaking-api:8083/api/v1
PAYMENT_API_URL=http://payment-api:8084/api/v1
BOOKING_API_URL=http://booking-api:8085/api/v1
NOTIFICATION_API_URL=http://notification-api:8086/api/v1
```

## Development

```powershell
go tool gqlgen generate
go test ./...
go vet ./...
```

Change `graph/schema.graphqls` first, regenerate, implement resolvers, update frontend operations, and add contract tests in the same change. Never log tokens, cookies, profile preferences, payment fields, or GraphQL variables containing private data.

Production must disable the explorer, use TLS, exact origins, edge rate limiting, persisted/allow-listed operations where appropriate, bounded pagination, query depth/complexity protection, upstream circuit-breaking, and multiple stateless replicas.
