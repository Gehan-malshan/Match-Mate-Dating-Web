ALTER TABLE event ADD COLUMN archived_at timestamptz;
CREATE INDEX event_managed_archive_idx ON event(archived_at, updated_at DESC, event_id DESC);
