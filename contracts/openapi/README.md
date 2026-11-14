# OpenAPI Contracts

Account v1 now describes verification resend, password recovery/reset, optional administrator TOTP login, and password-confirmed member deletion. Notification v1 describes a private service-authenticated transient security email endpoint. These are additive v1 operations except the existing Account DELETE now requires password reauthentication; clients must update together. See ADR-0003 and the Account security runbook.

Event v1 now includes the optional event image workflow: admin-only draft JPEG replacement and public-after-publication binary read, plus `imageVersion` metadata. GraphQL remains the browser command surface; the gateway owns the narrow public media GET.

Implemented specifications: `account-v1.yaml`, the Event slice in `event-v1.yaml`, the first Booking slice in `booking-v1.yaml`, the deterministic prototype in `matchmaking-v1.yaml`, the first Payment slice in `payment-v1.yaml`, and the authenticated member feed in `notification-v1.yaml`.

Payment v1 additionally documents an administrator-only, audited, read-only cursor-paged reconciliation queue and strict PayHere callback rejection. No refund endpoint is implemented under the current no-refunds direction.

This directory contains canonical versioned REST contracts. Account, Event, first Booking, Matchmaking prototype, first Payment, Notification, and the first Moderation reporting/case-status/action/appeal API are implemented. API implementation and generated clients must conform to these files.

## Planned organization

```text
openapi/
|-- gateway-v1.yaml
|-- account-v1.yaml
|-- event-v1.yaml
|-- booking-v1.yaml
|-- payment-v1.yaml
|-- matchmaking-v1.yaml
|-- notification-v1.yaml          authenticated member-owned feed
`-- moderation-v1.yaml
```

Specifications must define authentication, authorization notes, idempotency, errors, examples, pagination, formats, enum behavior, sensitive fields, and deprecation. Shared schema fragments may be reused only for transport conventions—not service domain entities.

CI must lint, detect backward incompatibility, validate examples, and compile generated clients. An implementation change without matching OpenAPI is incomplete.

Follow `docs/api/README.md`, `AGENTS.md`, and the affected service README.
