ALTER TABLE event
  ADD COLUMN publication_review_version TEXT,
  ADD COLUMN publication_reviewed_by TEXT,
  ADD COLUMN publication_reviewed_at TIMESTAMPTZ;

ALTER TABLE event ADD CONSTRAINT event_publication_review_complete CHECK (
  (publication_review_version IS NULL AND publication_reviewed_by IS NULL AND publication_reviewed_at IS NULL)
  OR
  (publication_review_version IS NOT NULL AND publication_reviewed_by IS NOT NULL AND publication_reviewed_at IS NOT NULL)
);
