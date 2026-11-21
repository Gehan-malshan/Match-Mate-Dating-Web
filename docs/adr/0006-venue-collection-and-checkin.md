# ADR-0006: Venue collection and check-in

Status: Accepted (2026-11-22)

## Context

An `AT_VENUE` booking reserves a confirmed seat but is unpaid. Booking previously had no staff collection or check-in record. The product owner confirmed that a confirmed venue reservation has no cancellation option. The existing no-refunds policy remains in force.

## Decision

Booking owns venue collection and attendance. Only an administrator can record `CASH` or `CARD_AT_VENUE` collection; the amount and currency come from Booking's immutable event price snapshot. The record stores the acting administrator, method, and UTC time. It does not charge a card or claim PayHere settlement. Check-in requires a confirmed, currently eligible booking and, for venue bookings, recorded collection. Online bookings require confirmed state. Staff operations require an idempotency key and create Booking outbox facts. Confirmed venue bookings cannot be cancelled. Unpaid online holds retain cancellation and expiry.

## Consequences and open questions

- Booking's PostgreSQL migration 6 adds collection/check-in fields and an idempotency record. Apply it before new binaries; rollback must retain collected audit data.
- Administrator access is used until an event-scoped staff role is approved. **OPEN QUESTION (Product/Safety):** define organizer/staff scopes and correction procedure for mistaken cash or card records.
- **OPEN QUESTION (Finance):** define reconciliation of external card terminal and cash takings. MatchMate stores a declaration, not provider settlement evidence.
- Current eligibility is checked before each staff action. Strict atomicity between Moderation and Booking databases is not possible without a further coordination protocol.
- A cancelled event blocks staff actions. Automated check-in windows and no-show processing remain future policy.

## Verification

PostgreSQL component tests exercise final-seat contention, collection before check-in, repeated staff commands, no confirmed venue cancellation, hold expiry, duplicate Payment events, and capacity recovery. HTTP tests deny member access. Browser build and GraphQL tests verify the staff path.
