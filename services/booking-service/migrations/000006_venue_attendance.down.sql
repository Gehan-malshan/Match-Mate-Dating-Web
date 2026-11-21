DROP TABLE IF EXISTS booking_staff_action;
ALTER TABLE booking DROP COLUMN IF EXISTS checked_in_by;
ALTER TABLE booking DROP COLUMN IF EXISTS checked_in_at;
ALTER TABLE booking DROP COLUMN IF EXISTS venue_collection_method;
ALTER TABLE booking DROP COLUMN IF EXISTS venue_collected_by;
ALTER TABLE booking DROP COLUMN IF EXISTS venue_collected_at;
