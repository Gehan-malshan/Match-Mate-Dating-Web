CREATE TABLE notification_template (
    id uuid PRIMARY KEY,
    template_key text NOT NULL,
    locale text NOT NULL,
    channel text NOT NULL CHECK (channel IN ('DEVELOPMENT','EMAIL','SMS','PUSH')),
    category text NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    status text NOT NULL CHECK (status IN ('ACTIVE','RETIRED')),
    subject_template text NOT NULL,
    body_template text NOT NULL,
    allowed_variables text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL,
    UNIQUE(template_key, locale, channel, version)
);

CREATE TABLE notification_delivery (
    id uuid PRIMARY KEY,
    business_key text NOT NULL UNIQUE,
    recipient_account_id uuid NOT NULL,
    source_event_id uuid NOT NULL,
    source_event_type text NOT NULL,
    source_aggregate_id text NOT NULL,
    template_id uuid NOT NULL REFERENCES notification_template(id),
    category text NOT NULL,
    channel text NOT NULL,
    variables jsonb NOT NULL DEFAULT '{}',
    state text NOT NULL CHECK (state IN ('PENDING','PROCESSING','RETRY_SCHEDULED','DELIVERED','SUPPRESSED','PERMANENTLY_FAILED','DEAD_LETTERED')),
    scheduled_at timestamptz NOT NULL,
    next_attempt_at timestamptz NOT NULL,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    max_attempts integer NOT NULL CHECK (max_attempts > 0),
    provider_reference text,
    last_error_code text,
    lease_until timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    delivered_at timestamptz
);

CREATE TABLE notification_delivery_attempt (
    id uuid PRIMARY KEY,
    delivery_id uuid NOT NULL REFERENCES notification_delivery(id),
    attempt_number integer NOT NULL CHECK (attempt_number > 0),
    outcome text NOT NULL CHECK (outcome IN ('DELIVERED','RETRYABLE_FAILURE','PERMANENT_FAILURE','DEAD_LETTERED')),
    provider_reference text,
    error_code text,
    started_at timestamptz NOT NULL,
    completed_at timestamptz NOT NULL,
    UNIQUE(delivery_id, attempt_number)
);

CREATE TABLE notification_preference (
    account_id uuid NOT NULL,
    channel text NOT NULL,
    category text NOT NULL,
    allowed boolean NOT NULL,
    source text NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY(account_id, channel, category)
);

CREATE TABLE notification_suppression (
    account_id uuid PRIMARY KEY,
    reason text NOT NULL,
    created_at timestamptz NOT NULL,
    expires_at timestamptz
);

CREATE TABLE notification_inbox (
    consumer text NOT NULL,
    event_id uuid NOT NULL,
    event_type text NOT NULL,
    processed_at timestamptz NOT NULL,
    PRIMARY KEY(consumer, event_id)
);

CREATE INDEX notification_delivery_due_idx
    ON notification_delivery(next_attempt_at, created_at)
    WHERE state IN ('PENDING','RETRY_SCHEDULED','PROCESSING');
CREATE INDEX notification_delivery_recipient_idx
    ON notification_delivery(recipient_account_id, created_at DESC);
CREATE INDEX notification_attempt_delivery_idx
    ON notification_delivery_attempt(delivery_id, attempt_number);

INSERT INTO notification_template(id,template_key,locale,channel,category,version,status,subject_template,body_template,allowed_variables,created_at) VALUES
('71000000-0000-4000-8000-000000000001','account-welcome','en-LK','DEVELOPMENT','ACCOUNT',1,'ACTIVE','Welcome to MatchMate','Your MatchMate account was created. Open MatchMate to continue the verification and profile journey.','{}',now()),
('71000000-0000-4000-8000-000000000002','account-verified','en-LK','DEVELOPMENT','ACCOUNT',1,'ACTIVE','Your account is verified','Your MatchMate account verification is complete.','{}',now()),
('71000000-0000-4000-8000-000000000003','profile-approved','en-LK','DEVELOPMENT','ACCOUNT',1,'ACTIVE','Your profile is approved','Your community profile has been approved. Open MatchMate to review its current visibility.','{}',now()),
('71000000-0000-4000-8000-000000000004','profile-hidden','en-LK','DEVELOPMENT','ACCOUNT',1,'ACTIVE','Your profile visibility changed','Your community profile is currently hidden. Open MatchMate for the current status and available next steps.','{}',now()),
('71000000-0000-4000-8000-000000000005','booking-pending','en-LK','DEVELOPMENT','BOOKING',1,'ACTIVE','Complete your event booking','Your place is held temporarily. Open MatchMate to complete payment before the hold expires.','{}',now()),
('71000000-0000-4000-8000-000000000006','booking-confirmed','en-LK','DEVELOPMENT','BOOKING',1,'ACTIVE','Your booking is confirmed','Your MatchMate event booking is confirmed. Open MatchMate to view the latest event information.','{}',now()),
('71000000-0000-4000-8000-000000000007','booking-cancelled','en-LK','DEVELOPMENT','BOOKING',1,'ACTIVE','Your booking was cancelled','Your MatchMate event booking was cancelled. Open MatchMate to review your bookings.','{}',now()),
('71000000-0000-4000-8000-000000000008','booking-hold-expired','en-LK','DEVELOPMENT','BOOKING',1,'ACTIVE','Your booking hold expired','Your temporary event booking hold expired. Open MatchMate to check current availability.','{}',now()),
('71000000-0000-4000-8000-000000000009','booking-payment-review','en-LK','DEVELOPMENT','BOOKING',1,'ACTIVE','Your payment needs review','We received a payment result that needs review. Your place is not automatically confirmed; open MatchMate for the current status.','{}',now());

