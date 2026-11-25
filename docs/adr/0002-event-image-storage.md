# ADR-0002: Event-owned draft image and public media route

- **Status:** Accepted
- **Date:** 2026-10-07
- **Owners:** MatchMate project team

## Context

Event cards previously used rotating stock artwork, and administrators could not attach an image. A URL-only field would depend on untrusted external hosts and would not give Event control over visibility or retention. Production object storage is not yet configured; publication rights and moderation are governed by [ADR-0007](0007-event-publication-approval.md).

## Decision

The Event Service owns one optional image per event in its PostgreSQL `event_image` table. Admins can replace an image only while the event is a draft. Browser-side resizing produces a JPEG of at most 650 KiB and 1600×1600 pixels. Event revalidates and re-encodes it, stripping embedded metadata, then stores at most 1 MiB. `image_version` on Event allows the public catalogue to select the image without including bytes in GraphQL/event lists. A draft image is never served publicly; published/registration-open/closed images are public.

GraphQL carries the authenticated, bounded upload mutation. The gateway exposes one narrow, read-only `/media/events/{eventId}` route for JPEG bytes. This media GET is an explicit exception to GraphQL-only browser data queries; it is not a general proxy or service URL. Events with no image retain existing local artwork. The image is not shared across service databases.

## Consequences and open work

The schema migration is additive; existing events have `image_version=0`. Saving a new draft followed by upload is a two-step UI workflow. A failed image upload leaves the draft intact and visible for retry. Image replacement increments `image_version` while the event configuration version remains unchanged. Rollback should retain the Event-owned bytes; do not drop `event_image` while uploaded images exist.

PostgreSQL byte storage is deliberately bounded for the current small catalogue. Moving to object storage/CDN later requires a new ADR, migration, backup/retention plan, and URL compatibility strategy. Publication now requires the versioned rights, moderation, and event-safety approval in ADR-0007. Object-storage scaling, edge upload rate limits, backup-size evidence, and a richer accessibility text policy remain infrastructure/product work. Upload is admin-only, but that is not a substitute for those controls.

## Verification

Unit tests cover image validation/metadata stripping. Disposable-schema PostgreSQL tests cover draft-only storage, public visibility after publication, image version, and retained publication review. Gateway tests cover narrow proxy behavior. The Playwright Event journey covers real browser image upload and approval-gated publication.
