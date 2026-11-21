ALTER TABLE notification_delivery ADD COLUMN event_id uuid;
ALTER TABLE notification_delivery_attempt DROP CONSTRAINT notification_delivery_attempt_outcome_check;
ALTER TABLE notification_delivery_attempt ADD CONSTRAINT notification_delivery_attempt_outcome_check CHECK (outcome IN ('DELIVERED','RETRYABLE_FAILURE','PERMANENT_FAILURE','DEAD_LETTERED','SUPPRESSED'));
