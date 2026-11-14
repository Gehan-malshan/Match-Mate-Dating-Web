# Account security operations

## Local setup

1. Copy `.env.example` to ignored root `.env`. Set `NOTIFICATION_PROVIDER=smtp`, verified SMTP settings, `SECURITY_MAIL_TOKEN`, and `SECURITY_PUBLIC_URL=http://localhost:5173` to test real security email. Keep `INTERNAL_SERVICE_TOKEN` separate.
2. Run `docker compose up --build -d` and inspect `account-migrate` and `notification-migrate` for exit code 0. Start the web with `bun run dev`.
3. Register with an email you control. The verification link is sent by Notification; it is never included in the production registration response. Open the link, then sign in. The default local `dev-sink` does not deliver recovery messages; configure SMTP for this journey.
4. Use **Forgot password?** on `/login`, open the received link, set a new password, then confirm old sessions fail. Resend verification is available from `/verify-email`.

## Administrator TOTP

1. Generate a 32-byte random key and store its base64 value as `ADMIN_MFA_ENCRYPTION_KEY` in a secret manager. For local testing, generate it in PowerShell with `$bytes = New-Object byte[] 32; [System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes); [Convert]::ToBase64String($bytes)` and copy the output to ignored root `.env`. Give the same value to Account API and one-time CLI. Do not place the real value in Git or shell history.
2. Apply Account migration 2. The migration revokes existing administrator refresh sessions and increments their token versions.
3. After `docker compose up --build -d`, run `docker compose run --rm --entrypoint /account-admin-mfa account-api enroll admin@example.test`. Add the printed secret to the administrator's authenticator app. Protect the terminal output.
4. Run `docker compose run --rm --entrypoint /account-admin-mfa account-api confirm admin@example.test 123456` using the current app code. The just-used code cannot be reused; wait for the next 30-second code to sign in.
5. Enter email, password, and a fresh authenticator code on the common login page. Google sign in cannot bypass this requirement. In Compose, set `ADMIN_MFA_ENCRYPTION_KEY` in root `.env` to enable enforcement. Production refuses to start without it.

## JWT signing-key rotation

Provide a new `JWT_PRIVATE_KEY_PEM` and distinct `JWT_KEY_ID`; add the old key ID and public PEM in `JWT_PREVIOUS_PUBLIC_KEYS_JSON`. Roll out Account while both JWKs are published. Wait beyond the access-token TTL plus verifier cache time before removing the old public key. Keep private keys in a secret manager. Refresh sessions continue to rotate independently.

## Remaining production gates

PostgreSQL migration 2 and component tests passed on a disposable PostgreSQL 17 instance; repeat them in CI. Run real SMTP delivery and bounce tests, browser registration/recovery/deletion/MFA journeys, and multi-replica JWT rotation tests. Add edge/WAF request limits, expired rate-window cleanup, and delivery-failure alerts. Approve cross-service deletion retention, lost-MFA recovery, and MFA encryption-key rotation policies before production.
