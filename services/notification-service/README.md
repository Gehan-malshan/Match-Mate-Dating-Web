# Notification Service

Owns notification templates, delivery attempts, retry policies, suppression preferences, and email, SMS, or future push-provider integrations.

It will primarily consume durable business events without blocking critical transactions.

## Responsibilities

- Version and approve templates by channel/locale/category.
- Convert safe business events into idempotent delivery requests.
- Resolve approved recipient/channel information through a constrained mechanism.
- Apply consent, suppression, category, and channel preferences.
- Deliver through approved email/SMS/push adapters.
- Classify retryable/permanent failures, schedule bounded retries, and route exhausted work to DLQ.
- Record provider references, attempts, outcomes, and operational metrics.

## Does not own

Account/profile source data, business transaction state, match decisions, booking/payment state, moderation cases, or event lifecycle.

## Proposed data

`template`, `delivery`, `delivery_attempt`, `notification_preference`, `suppression`, `inbox`, and optional `outbox`.

Key invariants:

- A business idempotency key prevents unintended duplicate delivery.
- Template variables are allow-listed and validated.
- Hidden identity/private preferences/payment details never enter ordinary templates.
- Suppression and legally required communication policy are explicit and versioned.
- Provider outage cannot block the originating business transaction.

## Consumed events

Expected categories include account verification/welcome, profile moderation, event changes/cancellation, booking hold/confirmation/expiry/cancellation, payment status/review/refund, pair publication, approved response/reveal outcomes, and moderation actions.

Do not subscribe to an event merely for convenience when its payload would require unnecessary PII.

## Delivery flow

```text
Business event -> inbox deduplication -> policy/template selection
-> delivery record -> provider adapter -> delivered | retry scheduled | permanently failed
-> DLQ/alert after bounded attempts
```

## Required tests

- Event deduplication and business delivery idempotency.
- Template variable validation and locale fallback.
- Privacy snapshots for every template.
- Suppression/preference/category behavior.
- Retryable/permanent provider errors, backoff, rate limit, timeout, DLQ/replay.
- No duplicate delivery after worker crash around provider acknowledgement.
- Restricted notification/admin views and sanitized logs.

## Completion criteria

Approved events produce the correct safe notification exactly according to policy; outages are observable/recoverable; critical transactions remain independent; no hidden member/payment/safety information leaks through templates or logs.

Update this README, template/event contracts, privacy/testing/runbook docs, and change history whenever behavior changes.

