CREATE TABLE notification_feed_item (
    id uuid PRIMARY KEY,
    delivery_id uuid NOT NULL UNIQUE REFERENCES notification_delivery(id),
    recipient_account_id uuid NOT NULL,
    read_at timestamptz,
    created_at timestamptz NOT NULL
);

CREATE INDEX notification_feed_recipient_created_idx
    ON notification_feed_item(recipient_account_id, created_at DESC, id DESC);

CREATE INDEX notification_feed_unread_idx
    ON notification_feed_item(recipient_account_id, created_at DESC)
    WHERE read_at IS NULL;

-- Existing development deliveries are safe, versioned notification facts. Exclude
-- suppressed rows so an account/channel suppression cannot become member-visible.
INSERT INTO notification_feed_item(id, delivery_id, recipient_account_id, created_at)
SELECT d.id, d.id, d.recipient_account_id, d.created_at
FROM notification_delivery d
WHERE d.state <> 'SUPPRESSED'
ON CONFLICT (delivery_id) DO NOTHING;
