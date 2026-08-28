# AsyncAPI and RabbitMQ Contracts

This directory contains canonical event schemas. `account-events-v1.yaml` documents implemented Account/Profile facts; broader exchange/consumer contracts remain planned.

## Event rules

- Names are past-tense completed business facts.
- Every message uses the envelope defined in `AGENTS.md` and architecture docs.
- Payloads contain minimum safe facts; never database entities, credentials, tokens, private preferences, full callbacks, evidence, or unnecessary PII.
- Additive optional fields may remain in a major version. Required-field/type/semantic/routing changes require a new major version and migration plan.
- Consumers ignore unknown additive fields, reject unsupported major versions visibly, and deduplicate by event ID.
- Every queue has owner, purpose, binding, retry schedule, DLQ, retention, replay, ordering assumption, concurrency, dashboard, and alert.

## Planned organization

```text
asyncapi/
|-- matchmate-events-v1.yaml
|-- schemas/
|   |-- envelope-v1.yaml
|   `-- <event-name>-v1.yaml
`-- examples/
```

CI must validate schemas/examples, detect incompatibility, and run producer fixtures against consumer expectations. Producer and all consumers must be listed before an event is accepted.
