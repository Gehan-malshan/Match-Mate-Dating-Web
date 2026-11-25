CREATE TABLE password_reset_token (
    token_hash BYTEA PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES account(account_id),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX password_reset_account_idx ON password_reset_token(account_id, created_at DESC);

CREATE TABLE security_rate_window (
    bucket_hash BYTEA PRIMARY KEY,
    window_start TIMESTAMPTZ NOT NULL,
    attempts INTEGER NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX security_rate_expiry_idx ON security_rate_window(expires_at);

CREATE TABLE admin_totp (
    account_id UUID PRIMARY KEY REFERENCES account(account_id),
    secret_ciphertext BYTEA NOT NULL,
    confirmed_at TIMESTAMPTZ,
    last_step BIGINT NOT NULL DEFAULT -1,
    created_at TIMESTAMPTZ NOT NULL
);
ALTER TABLE refresh_session ADD COLUMN mfa_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- Invalidate privileged sessions issued before MFA enforcement.
UPDATE account SET token_version=token_version+1,updated_at=now()
WHERE account_id IN (SELECT account_id FROM role_assignment WHERE role='admin' AND revoked_at IS NULL);
UPDATE refresh_session SET revoked_at=now()
WHERE account_id IN (SELECT account_id FROM role_assignment WHERE role='admin' AND revoked_at IS NULL) AND revoked_at IS NULL;
