ALTER TABLE event ADD COLUMN image_version bigint NOT NULL DEFAULT 0 CHECK (image_version >= 0);
CREATE TABLE event_image (
  event_id uuid PRIMARY KEY REFERENCES event(event_id),
  content_type text NOT NULL CHECK (content_type = 'image/jpeg'),
  image_bytes bytea NOT NULL CHECK (octet_length(image_bytes) BETWEEN 1 AND 1048576),
  updated_at timestamptz NOT NULL
);
