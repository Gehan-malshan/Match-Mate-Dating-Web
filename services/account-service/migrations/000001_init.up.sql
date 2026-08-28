CREATE TABLE account (
    account_id UUID PRIMARY KEY,
    normalized_email TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE','SUSPENDED','DEACTIVATED')),
    verification_state TEXT NOT NULL CHECK (verification_state IN ('PENDING','VERIFIED')),
    token_version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deactivated_at TIMESTAMPTZ
);

CREATE TABLE credential (
    account_id UUID PRIMARY KEY REFERENCES account(account_id),
    password_hash TEXT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE consent_record (
    consent_id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES account(account_id),
    consent_type TEXT NOT NULL,
    policy_version TEXT NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);
CREATE INDEX consent_record_account_idx ON consent_record(account_id, consent_type);

CREATE TABLE role_assignment (
    account_id UUID NOT NULL REFERENCES account(account_id),
    role TEXT NOT NULL CHECK (role IN ('member','organizer','moderator','support','finance','admin')),
    scope TEXT NOT NULL DEFAULT '',
    granted_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    PRIMARY KEY (account_id, role, scope)
);

CREATE TABLE email_verification_token (
    token_hash BYTEA PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES account(account_id),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE refresh_session (
    session_id UUID PRIMARY KEY,
    family_id UUID NOT NULL,
    account_id UUID NOT NULL REFERENCES account(account_id),
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    rotated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    reuse_detected_at TIMESTAMPTZ,
    replaced_by UUID
);
CREATE INDEX refresh_session_family_idx ON refresh_session(family_id);
CREATE INDEX refresh_session_account_idx ON refresh_session(account_id);

CREATE TABLE profile (
    account_id UUID PRIMARY KEY REFERENCES account(account_id),
    nickname TEXT NOT NULL,
    date_of_birth DATE NOT NULL,
    broad_location TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    visibility TEXT NOT NULL CHECK (visibility IN ('PRIVATE','COMMUNITY','HIDDEN')),
    approval_state TEXT NOT NULL CHECK (approval_state IN ('PENDING','APPROVED','HIDDEN')),
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE profile_interest (
    account_id UUID NOT NULL REFERENCES account(account_id) ON DELETE CASCADE,
    interest TEXT NOT NULL,
    PRIMARY KEY (account_id, interest)
);

CREATE TABLE matching_preference (
    account_id UUID PRIMARY KEY REFERENCES account(account_id),
    min_age INTEGER NOT NULL CHECK (min_age >= 18),
    max_age INTEGER NOT NULL CHECK (max_age >= min_age AND max_age <= 120),
    intentions JSONB NOT NULL DEFAULT '[]',
    interested_in JSONB NOT NULL DEFAULT '[]',
    languages JSONB NOT NULL DEFAULT '[]',
    deal_breakers JSONB NOT NULL DEFAULT '[]',
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE member_block (
    blocker_account_id UUID NOT NULL REFERENCES account(account_id),
    blocked_account_id UUID NOT NULL REFERENCES account(account_id),
    created_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    CHECK (blocker_account_id <> blocked_account_id)
);
CREATE UNIQUE INDEX member_block_active_unique ON member_block(blocker_account_id, blocked_account_id) WHERE revoked_at IS NULL;
CREATE INDEX member_block_reverse_idx ON member_block(blocked_account_id, blocker_account_id) WHERE revoked_at IS NULL;

CREATE TABLE audit_log (
    audit_id UUID PRIMARY KEY,
    actor_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    action TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    prior_state JSONB NOT NULL DEFAULT '{}',
    new_state JSONB NOT NULL DEFAULT '{}',
    correlation_id TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE outbox (
    event_id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    schema_version INTEGER NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    aggregate_id TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    causation_id TEXT NOT NULL DEFAULT '',
    actor_id TEXT NOT NULL DEFAULT '',
    routing_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    claimed_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX outbox_unpublished_idx ON outbox(occurred_at) WHERE published_at IS NULL;
