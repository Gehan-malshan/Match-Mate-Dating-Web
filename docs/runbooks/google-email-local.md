# Google sign-in and SMTP email setup

## What is implemented

- **Google sign-in** authenticates an existing, active, email-verified MatchMate account through Google's OAuth 2.0 authorization-code flow. It does not create an incomplete MatchMate profile because MatchMate must record date of birth and privacy consent during registration.
- **SMTP email delivery** sends the existing safe Account and Booking notification templates. The Notification database stores account IDs only; the worker resolves a verified email from the Account service only while it is sending.
- Password login and the `dev-sink` notification provider remain the default local behavior.

## Google Cloud setup

1. In Google Cloud Console, create or select a project and configure the OAuth consent screen/branding.
2. Create an OAuth **Web application** client.
3. Add this exact authorized redirect URI for local Docker development:

   ```text
   http://localhost:8080/auth/google/callback
   ```

4. Copy the client ID and client secret into the root `.env` file using `.env.example` as the template.
5. Restart the stack:

   ```powershell
   docker compose up --build -d
   ```

6. Open `http://localhost:5173/login` and use **Continue with Google**. Sign in with the same email address as an existing, verified MatchMate account.

The redirect URI must match the Google Cloud Console value exactly, including scheme, hostname, port, path, case, and trailing slash behavior. Google documents the web-server OAuth flow and redirect URI requirement at <https://developers.google.com/identity/protocols/oauth2/web-server>.

For production, configure HTTPS only, use domains owned by MatchMate, and set:

```text
GOOGLE_OAUTH_REDIRECT_URL=https://api.example.com/auth/google/callback
GOOGLE_OAUTH_SUCCESS_REDIRECT_URL=https://app.example.com/login?google=success
COOKIE_SECURE=true
```

## SMTP email setup

Choose an approved transactional email provider that supports SMTP. Add its SMTP host, port, username, password, and verified sender address to root `.env`:

```text
NOTIFICATION_PROVIDER=smtp
SMTP_HOST=smtp.provider.example
SMTP_PORT=587
SMTP_USERNAME=...
SMTP_PASSWORD=...
SMTP_FROM=MatchMate <no-reply@example.com>
INTERNAL_SERVICE_TOKEN=replace-with-a-long-random-secret
```

The implementation requires STARTTLS and TLS 1.2 or newer. It retries temporary provider failures and stores sanitized result codes, never recipient addresses or email bodies in logs.

Restart the stack after changing the variables:

```powershell
docker compose up --build -d
docker compose logs --tail 100 notification-worker
```

To test safely, use a fictional local member account whose email you control only after replacing its development fixture email through a dedicated test setup. Do not send development fixture emails to a real destination or add a real email address to committed seed data.

Gmail can be used only for a limited personal test account when two-step verification and an app password are available; it is not the recommended production sender. See Google's app-password guidance: <https://support.google.com/accounts/answer/2461835>.

## Verify

1. Confirm `notification-migrate` exits with code `0`.
2. Create a supported Account or Booking event.
3. Inspect the Notification worker logs. They must contain only delivery IDs, template keys, and sanitized result codes.
4. Confirm the in-app notification appears as before.
5. Confirm the email arrives once. Repeating the same RabbitMQ event must not create a duplicate delivery.

If email is not sent, set `NOTIFICATION_PROVIDER=dev-sink`, restart Docker, and inspect the provider credentials outside Git. Never paste secrets or full email headers into tickets, chat, or source control.
