CREATE TABLE reveal_consent_history (
    pairing_id UUID NOT NULL REFERENCES locked_pairing(pairing_id),
    account_id UUID NOT NULL,
    decision_version BIGINT NOT NULL CHECK (decision_version > 0),
    decision TEXT NOT NULL CHECK (decision IN ('GRANTED','REVOKED')),
    policy_version TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    recorded_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (pairing_id, account_id, decision_version)
);

INSERT INTO reveal_consent_history (
    pairing_id,
    account_id,
    decision_version,
    decision,
    policy_version,
    idempotency_key,
    recorded_at
)
SELECT
    pairing_id,
    account_id,
    1,
    decision,
    policy_version,
    idempotency_key,
    recorded_at
FROM reveal_consent
ON CONFLICT DO NOTHING;
