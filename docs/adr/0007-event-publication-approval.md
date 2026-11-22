# ADR-0007: Event publication rights, moderation, and safety approval

- **Status:** Accepted
- **Date:** 2026-11-22
- **Owners:** MatchMate Product, Safety, and Engineering

## Context

Event images and public event details affect privacy, intellectual-property rights, attendee safety, and venue security. Administrator-only upload and image re-encoding reduce technical risk but do not prove publishing rights, acceptable content, or operational readiness. Publication previously had no explicit approval record.

## Decision

Event publication uses procedure `event-publication-v1`. An administrator must confirm all three controls in the unified admin application, and Event Service repeats enforcement:

1. **Image rights:** MatchMate owns the image, has a licence permitting this use, or has documented consent from the rights holder and every identifiable adult. Images of minors are prohibited. If no event image is attached, the administrator confirms that approved MatchMate stock artwork will be used.
2. **Image moderation:** the image contains no nudity or sexual content, harassment, hate, illegal content, contact details, identity documents, payment information, exact private venue details, or unsafe depiction. Browser and server processing still enforce type, size, dimensions, and metadata removal.
3. **Event safety:** the venue contact, responsible incident lead, trained check-in coverage, emergency escalation, accessibility arrangements, attendee privacy, arrival disclosure, and cancellation communication plan have been checked using the publication runbook.

The Event database records the review version, approving account ID, and UTC approval time on the event. The `EventPublished` outbox fact includes only the review version; it does not copy the checklist, venue contacts, or incident details. Replacing an image remains possible only in `DRAFT`, so every published image is covered by the approval made at publication.

Publication approval requires the `admin` role even when an assigned organizer can perform other lifecycle operations. The GraphQL gateway checks the role and Event Service enforces it again.

## Consequences

- Migration 6 adds nullable publication-review fields with an all-or-none constraint. Existing published events have no retrospective record and must be reviewed before production data is promoted.
- Publishing without all confirmations or with an unsupported procedure version returns `EVENT_PUBLICATION_REVIEW_REQUIRED`.
- The record proves who completed the configured checklist; it does not establish legal ownership by itself. Rights evidence is retained in the approved restricted project record, outside public event data and logs.
- A later procedure change uses a new version and compatibility plan. It must not silently reinterpret historical approvals.
- Event Service still does not maintain a general event audit timeline. This narrow publication record is part of current event state and the publication business fact.

## Verification

Unit tests reject missing, partial, outdated, and organizer-issued approvals. PostgreSQL component tests verify migration and retained approval fields. The Playwright journey proves the browser requires all three confirmations before publishing and then covers cancel, archive, restore, and registrations navigation.

See [event publication and safety runbook](../runbooks/event-publication-safety.md).
