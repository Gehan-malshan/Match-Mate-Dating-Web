# OpenAPI Contracts

Implemented specifications: `account-v1.yaml` and the first Event Service slice in `event-v1.yaml`. Later service specifications remain planned.

This directory contains canonical versioned REST contracts. `account-v1.yaml` is implemented; later service files remain planned. API implementation and generated clients must conform to these files.

## Planned organization

```text
openapi/
|-- gateway-v1.yaml
|-- account-v1.yaml
|-- event-v1.yaml
|-- booking-v1.yaml
|-- payment-v1.yaml
|-- matchmaking-v1.yaml
|-- notification-v1.yaml          only if public/admin API exists
`-- moderation-v1.yaml
```

Specifications must define authentication, authorization notes, idempotency, errors, examples, pagination, formats, enum behavior, sensitive fields, and deprecation. Shared schema fragments may be reused only for transport conventions—not service domain entities.

CI must lint, detect backward incompatibility, validate examples, and compile generated clients. An implementation change without matching OpenAPI is incomplete.

Follow `docs/api/README.md`, `AGENTS.md`, and the affected service README.
