ALTER TABLE event DROP CONSTRAINT IF EXISTS event_publication_review_complete;
ALTER TABLE event
  DROP COLUMN IF EXISTS publication_reviewed_at,
  DROP COLUMN IF EXISTS publication_reviewed_by,
  DROP COLUMN IF EXISTS publication_review_version;
