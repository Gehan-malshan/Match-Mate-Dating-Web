# MatchMate API Conventions

Versioned OpenAPI files under `contracts/openapi/` are authoritative. This guide defines conventions shared by all services.

## Public edge

- Canonical base path: `/api/v1`.
- Clients call the API Gateway, not service-private addresses.
- JSON request/response bodies use consistent lower camel-case fields.
- UTC ISO-8601 timestamps; fixed-precision decimal money represented without floating-point ambiguity and with explicit ISO 4217 currency.
- Cursor pagination for mutable collections: `items`, `nextCursor`, and `limit`.
- Query filters are bounded, validated, and documented.

## Authentication and identity

- `Authorization: Bearer <access-token>` for authenticated member/operator APIs unless an accepted transport decision changes it.
- Self-service uses `/users/me` and token subject, not caller-selected user ID.
- Notification self-service similarly derives the recipient from the token subject; list/read commands never accept an account ID and conceal another member's item as not found.
- Every endpoint documents allowed roles/scopes, ownership, event scope, state restrictions, and sensitive fields.
- Provider callbacks use provider verification, request controls, and replay protection—not member JWT.

## Idempotency

Require `Idempotency-Key` for booking creation, payment initiation, cancellation, refund, matching generation/lock/publish, response/consent where duplicate side effects matter, and equivalent commands.

The owner stores key, authenticated actor/scope, request fingerprint, result/reference, and expiry. Reusing a key with different input returns a stable conflict; same input returns the original safe outcome.

## Errors

Use `application/problem+json`:

```json
{
  "type": "https://matchmate.example/problems/booking-capacity-exhausted",
  "title": "Event capacity is no longer available",
  "status": 409,
  "code": "BOOKING_CAPACITY_EXHAUSTED",
  "detail": "The requested ticket allocation cannot be reserved.",
  "instance": "/api/v1/bookings",
  "traceId": "trace-id",
  "fieldErrors": []
}
```

Do not expose stack traces, SQL/provider details, private rejection reasons, or resource existence when doing so enables enumeration.

## Status guidance

- `200/201/204` successful read/create/no-body command.
- `202` accepted asynchronous operation with status resource.
- `400` malformed/semantic request; `401` unauthenticated; `403` authenticated but forbidden.
- `404` absent or intentionally concealed resource.
- `409` state/idempotency/capacity/version conflict.
- `422` only if adopted consistently for validated domain input; otherwise use documented `400`.
- `429` rate limit with safe retry metadata.
- `5xx` unexpected/dependency failures; do not report success when outcome is unknown.

## Versioning and compatibility

- Add optional fields and endpoints compatibly within v1.
- Do not change field meaning/type, remove values, make optional fields required, or reuse error/event codes silently.
- Breaking behavior requires a new major version, migration window, deprecation date, telemetry, and change record.
- Generate frontend clients from reviewed OpenAPI and compile them in CI.

## API change checklist

- [ ] Owner, authorization, state, idempotency, errors, pagination, time/money, and sensitive fields documented.
- [ ] OpenAPI lint and compatibility pass.
- [ ] Handler and generated-client contract tests pass.
- [ ] Security/privacy/rate-limit/audit behavior tested.
- [ ] Service README, canonical docs, and before/after change history updated.
