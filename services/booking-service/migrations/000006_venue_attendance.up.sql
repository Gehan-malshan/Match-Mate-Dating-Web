ALTER TABLE booking ADD COLUMN venue_collected_at timestamptz;
ALTER TABLE booking ADD COLUMN venue_collected_by uuid;
ALTER TABLE booking ADD COLUMN venue_collection_method text CHECK (venue_collection_method IN ('CASH','CARD_AT_VENUE'));
ALTER TABLE booking ADD COLUMN checked_in_at timestamptz;
ALTER TABLE booking ADD COLUMN checked_in_by uuid;
CREATE TABLE booking_staff_action (
  actor_id uuid NOT NULL,
  operation text NOT NULL CHECK (operation IN ('venue.collect','booking.check_in')),
  idempotency_key text NOT NULL,
  booking_id uuid NOT NULL REFERENCES booking(id),
  request_fingerprint text NOT NULL,
  created_at timestamptz NOT NULL,
  PRIMARY KEY (actor_id,operation,idempotency_key)
);
