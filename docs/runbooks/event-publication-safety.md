# Event publication and safety approval

**Owner:** MatchMate Safety and Event Operations  
**Procedure version:** `event-publication-v1`  
**Approved:** 2026-11-22  
**Next review:** 2027-02-22 or after any event safety incident

## Purpose

Use this procedure before an administrator publishes an event. It protects image rights, member privacy, venue confidentiality, and safe on-site operation. Do not publish when evidence is missing or a concern remains unresolved.

## Required access and evidence

- Use an individual administrator account with MFA in production. Never share the development fixture account.
- Keep rights evidence, venue contacts, staff assignments, accessibility notes, and emergency contacts in the approved restricted operations system. Do not put these details in public descriptions, RabbitMQ events, tickets, screenshots, or general logs.
- Use fictional information in development and browser tests.

## Image-rights review

1. Identify the image source and rights holder.
2. Confirm MatchMate owns it, has a licence covering web publication, or has written permission for this use.
3. Confirm every identifiable person is an adult and has consented to this publication. Reject images containing minors.
4. Confirm the image does not expose an identity document, contact detail, payment information, attendee list, home address, or exact private venue information.
5. If evidence is unclear, remove the upload and use approved MatchMate stock artwork.

## Image-content moderation

Reject the image if it contains nudity or sexual content, exploitation, harassment, hate, threats, illegal activity, weapons presented unsafely, intoxication presented as an event promise, misleading branding, embedded advertising, QR codes, social handles, or other contact routes. Escalate uncertain content to the Safety owner and do not publish while review is pending.

The upload pipeline accepts JPEG, PNG, or WebP in the browser, produces a bounded JPEG, verifies and re-encodes it in Event Service, strips metadata, and never serves a draft image publicly. These technical controls do not replace human review.

## Event safety review

Before approval, confirm:

- the venue and its responsible contact have been verified;
- the public event displays only a broad location and the exact arrival details use the controlled attendee journey;
- an incident lead and trained check-in staff are assigned for the whole event;
- check-in uses Booking eligibility and does not reveal the public attendee list;
- emergency exits, local emergency contact procedure, first-aid availability, accessibility, transport/closing arrangements, and venue capacity are understood;
- staff know how to separate participants, preserve evidence privately, contact emergency services when immediate danger exists, and create a Moderation report without copying sensitive evidence into logs;
- the cancellation communication plan is ready and no-refund/payment handling follows the approved Booking and Payment policy;
- no activity requires members to reveal contact details or bypass consent, block, suspension, event exclusion, or reveal prevention.

## Publication

1. Sign in to the unified web application and open **Event manager**.
2. Review the draft fields and selected image.
3. Select **Publish**.
4. Complete all three confirmations. The Event Service rejects incomplete or stale procedure versions.
5. Verify the card displays **PUBLISHED**, the public event omits the exact venue, and the image route serves the sanitized JPEG.

The event stores the approving account, UTC time, and `event-publication-v1`. Preserve the restricted rights and safety evidence according to the organization retention policy.

## Incident, cancellation, and escalation

- For immediate danger, contact local emergency services and venue security first.
- Stop admission when eligibility checks are unavailable, capacity is uncertain, or safety conditions are not acceptable.
- Cancel the event through the administrator interface when the event cannot proceed safely. Confirm member notification and Booking/Payment reconciliation; do not hard-delete business history.
- Report unsafe media or event conduct through Moderation. Hide or cancel affected public content while investigation proceeds.
- Escalate to Product/Safety before correcting a mistaken publication record. Never edit Event PostgreSQL directly.

## Verification and evidence

Run `bun run test:e2e:event` against the local Compose stack. The test must pass create, image preparation/upload, approval-gated publish, cancel, delete/restore, and event-scoped registration navigation. CI retains traces, screenshots, and video only on failure for 14 days; artifacts use fictional data.

After an incident or quarterly review, record the drill/result in the restricted operations system and update this procedure, ADR, tests, and review version when behavior changes.
