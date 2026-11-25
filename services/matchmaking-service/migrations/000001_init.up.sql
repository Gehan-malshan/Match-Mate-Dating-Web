CREATE TABLE ruleset (
    version TEXT PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('APPROVED','RETIRED')),
    configuration JSONB NOT NULL,
    approved_by TEXT NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL,
    CHECK (jsonb_typeof(configuration) = 'object')
);

CREATE TABLE event_scope (
    event_id UUID PRIMARY KEY,
    organizer_id TEXT NOT NULL,
    event_status TEXT NOT NULL,
    ruleset_version TEXT NOT NULL REFERENCES ruleset(version),
    source_version BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE participant_projection (
    event_id UUID NOT NULL REFERENCES event_scope(event_id),
    account_id UUID NOT NULL,
    participant_code TEXT NOT NULL,
    group_code TEXT NOT NULL,
    input JSONB NOT NULL,
    source_version BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (event_id, account_id),
    UNIQUE (event_id, participant_code),
    CHECK (jsonb_typeof(input) = 'object')
);

CREATE TABLE matching_run (
    run_id UUID PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES event_scope(event_id),
    run_version INTEGER NOT NULL,
    aggregate_version BIGINT NOT NULL DEFAULT 1,
    status TEXT NOT NULL CHECK (status IN ('GENERATED','UNDER_REVIEW','LOCKED','PUBLISHED','INVALIDATED')),
    ruleset_version TEXT NOT NULL REFERENCES ruleset(version),
    optimizer_version TEXT NOT NULL,
    tie_break_policy TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    created_by TEXT NOT NULL,
    participant_count INTEGER NOT NULL,
    eligible_pair_count INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    locked_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    UNIQUE (event_id, run_version),
    UNIQUE (event_id, idempotency_key)
);

CREATE TABLE participant_snapshot (
    snapshot_id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES matching_run(run_id),
    account_id UUID NOT NULL,
    participant_code TEXT NOT NULL,
    group_code TEXT NOT NULL,
    source_version BIGINT NOT NULL,
    input JSONB NOT NULL,
    snapshot_at TIMESTAMPTZ NOT NULL,
    UNIQUE (run_id, account_id)
);

CREATE TABLE candidate (
    candidate_id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES matching_run(run_id),
    participant_a UUID NOT NULL,
    participant_b UUID NOT NULL,
    canonical_pair_key TEXT NOT NULL,
    eligible BOOLEAN NOT NULL,
    rejection_codes JSONB NOT NULL DEFAULT '[]',
    components JSONB NOT NULL DEFAULT '{}',
    total_score INTEGER,
    safe_reasons JSONB NOT NULL DEFAULT '[]',
    UNIQUE (run_id, canonical_pair_key),
    CHECK (participant_a <> participant_b),
    CHECK (total_score IS NULL OR total_score BETWEEN 0 AND 100)
);

CREATE TABLE pairing_suggestion (
    suggestion_id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES matching_run(run_id),
    participant_a UUID NOT NULL,
    participant_b UUID NOT NULL,
    score INTEGER NOT NULL CHECK (score BETWEEN 0 AND 100),
    safe_reasons JSONB NOT NULL,
    optimizer_order INTEGER NOT NULL,
    UNIQUE (run_id, participant_a, participant_b)
);

CREATE TABLE pairing_selection (
    selection_id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES matching_run(run_id),
    participant_a UUID NOT NULL,
    participant_b UUID NOT NULL,
    score INTEGER NOT NULL CHECK (score BETWEEN 0 AND 100),
    safe_reasons JSONB NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('ALGORITHM','OVERRIDE')),
    UNIQUE (run_id, participant_a),
    UNIQUE (run_id, participant_b),
    CHECK (participant_a <> participant_b)
);

CREATE TABLE unmatched_participant (
    run_id UUID NOT NULL REFERENCES matching_run(run_id),
    account_id UUID NOT NULL,
    reason_code TEXT NOT NULL,
    PRIMARY KEY (run_id, account_id)
);

CREATE TABLE pairing_override (
    override_id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES matching_run(run_id),
    removed_selection_id UUID,
    replacement_selection_id UUID NOT NULL,
    actor_id TEXT NOT NULL,
    reason TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (run_id, idempotency_key)
);

CREATE TABLE locked_pairing (
    pairing_id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES matching_run(run_id),
    event_id UUID NOT NULL,
    round_no INTEGER NOT NULL DEFAULT 1,
    participant_a UUID NOT NULL,
    participant_b UUID NOT NULL,
    score INTEGER NOT NULL CHECK (score BETWEEN 0 AND 100),
    safe_reasons JSONB NOT NULL,
    locked_at TIMESTAMPTZ NOT NULL,
    UNIQUE (run_id, participant_a, participant_b)
);

CREATE TABLE locked_participant (
    pairing_id UUID NOT NULL REFERENCES locked_pairing(pairing_id),
    event_id UUID NOT NULL,
    round_no INTEGER NOT NULL,
    account_id UUID NOT NULL,
    PRIMARY KEY (pairing_id, account_id),
    UNIQUE (event_id, round_no, account_id)
);

CREATE TABLE match_response (
    pairing_id UUID NOT NULL REFERENCES locked_pairing(pairing_id),
    account_id UUID NOT NULL,
    response TEXT NOT NULL CHECK (response IN ('INTERESTED','PASS')),
    question_version TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    recorded_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (pairing_id, account_id)
);

CREATE TABLE reveal_consent (
    pairing_id UUID NOT NULL REFERENCES locked_pairing(pairing_id),
    account_id UUID NOT NULL,
    decision TEXT NOT NULL CHECK (decision IN ('GRANTED','REVOKED')),
    policy_version TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    recorded_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (pairing_id, account_id)
);

CREATE TABLE match_feedback (
    feedback_id UUID PRIMARY KEY,
    pairing_id UUID NOT NULL REFERENCES locked_pairing(pairing_id),
    account_id UUID NOT NULL,
    comfort_rating INTEGER NOT NULL CHECK (comfort_rating BETWEEN 1 AND 5),
    quality_rating INTEGER NOT NULL CHECK (quality_rating BETWEEN 1 AND 5),
    safety_concern BOOLEAN NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    submitted_at TIMESTAMPTZ NOT NULL,
    UNIQUE (pairing_id, account_id)
);

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
    causation_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    routing_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    published_at TIMESTAMPTZ
);
CREATE INDEX outbox_unpublished_idx ON outbox(occurred_at) WHERE published_at IS NULL;
