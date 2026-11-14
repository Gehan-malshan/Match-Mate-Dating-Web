-- Existing databases may still contain the Event audit table created by migration 1.
-- Dropping it permanently removes those records. Outbox facts remain untouched.
DROP TABLE IF EXISTS event_audit;
