# ADR-0003: Recoverable event deletion

Status: Accepted for the current admin event-management slice.

## Context

Cancelled events remained visible in the only admin event list. Permanently deleting an event could orphan booking references.

## Decision

The UI's **Delete** action sets Event-owned `archived_at` instead of deleting rows. Only administrators can archive a draft or cancelled event, with explicit confirmation and optimistic version check. Published/open events must be cancelled first. Active managed lists exclude archived events; administrators can use **Show deleted** and Restore. Restore changes visibility only, not the lifecycle status: a cancelled event remains cancelled. Booking, image, and outbox records are retained. Event audit records are not stored.

## Consequences

- A user can recover an accidental deletion without restoring a database backup.
- Booking and reporting references remain stable.
- The UI must explain that Delete is recoverable and show a separate deleted list.
- This is not a privacy-erasure workflow; account deletion and retention policies require separate implementation.
