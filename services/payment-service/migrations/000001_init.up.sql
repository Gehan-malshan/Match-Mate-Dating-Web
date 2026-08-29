CREATE TABLE payment (
    id uuid PRIMARY KEY,
    booking_id uuid NOT NULL,
    account_id uuid NOT NULL,
    order_id text NOT NULL UNIQUE,
    amount numeric(18,2) NOT NULL CHECK (amount > 0),
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    provider text NOT NULL CHECK (provider = 'PAYHERE'),
    provider_payment_id text UNIQUE,
    state text NOT NULL CHECK (state IN ('PENDING','COMPLETED','FAILED','REVIEW')),
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    completed_at timestamptz,
    CONSTRAINT one_payment_per_booking UNIQUE (booking_id)
);

CREATE TABLE idempotency_record (
    actor_id uuid NOT NULL,
    operation text NOT NULL,
    idempotency_key text NOT NULL,
    request_fingerprint text NOT NULL,
    payment_id uuid NOT NULL REFERENCES payment(id),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (actor_id, operation, idempotency_key)
);

CREATE TABLE provider_callback (
    id uuid PRIMARY KEY,
    fingerprint char(64) NOT NULL UNIQUE,
    payment_id uuid REFERENCES payment(id),
    provider_payment_id text,
    order_id text NOT NULL,
    verification_result text NOT NULL,
    received_at timestamptz NOT NULL
);

CREATE TABLE payment_audit (
    id uuid PRIMARY KEY,
    payment_id uuid REFERENCES payment(id),
    action text NOT NULL,
    from_state text,
    to_state text,
    actor_id text NOT NULL,
    reason_code text,
    occurred_at timestamptz NOT NULL
);

CREATE TABLE reconciliation_item (
    id uuid PRIMARY KEY,
    payment_id uuid NOT NULL REFERENCES payment(id),
    discrepancy_type text NOT NULL,
    status text NOT NULL CHECK (status IN ('OPEN','RESOLVED')),
    created_at timestamptz NOT NULL,
    resolved_at timestamptz,
    UNIQUE(payment_id, discrepancy_type, status)
);

CREATE TABLE inbox (
    consumer text NOT NULL,
    event_id uuid NOT NULL,
    processed_at timestamptz NOT NULL,
    PRIMARY KEY (consumer, event_id)
);

CREATE TABLE outbox (
    event_id uuid PRIMARY KEY,
    event_type text NOT NULL,
    schema_version integer NOT NULL,
    aggregate_id uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid,
    actor_id text,
    payload jsonb NOT NULL,
    occurred_at timestamptz NOT NULL,
    published_at timestamptz,
    claimed_at timestamptz,
    attempts integer NOT NULL DEFAULT 0
);

CREATE INDEX payment_state_updated_idx ON payment(state, updated_at);
CREATE INDEX outbox_unpublished_idx ON outbox(occurred_at) WHERE published_at IS NULL;
