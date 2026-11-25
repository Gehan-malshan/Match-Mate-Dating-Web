DROP TABLE IF EXISTS admin_totp;
ALTER TABLE refresh_session DROP COLUMN IF EXISTS mfa_verified;
DROP TABLE IF EXISTS security_rate_window;
DROP TABLE IF EXISTS password_reset_token;
