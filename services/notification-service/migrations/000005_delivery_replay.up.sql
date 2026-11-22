CREATE TABLE notification_delivery_replay (
    id uuid PRIMARY KEY,
    delivery_id uuid NOT NULL REFERENCES notification_delivery(id),
    actor_id text NOT NULL,
    reason text NOT NULL CHECK (char_length(reason) BETWEEN 8 AND 500),
    previous_state text NOT NULL CHECK (previous_state IN ('PERMANENTLY_FAILED','DEAD_LETTERED')),
    previous_attempt_count integer NOT NULL CHECK (previous_attempt_count >= 1),
    additional_attempts integer NOT NULL CHECK (additional_attempts BETWEEN 1 AND 20),
    requested_at timestamptz NOT NULL
);

CREATE INDEX notification_delivery_replay_delivery_idx
    ON notification_delivery_replay(delivery_id, requested_at DESC);
