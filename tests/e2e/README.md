# End-to-End Tests

Critical journeys will cover registration, profile setup, event discovery, booking, PayHere confirmation, matching, participant responses, and moderation.

## Implemented Event browser journey

`browser/event-administration.spec.ts` uses Playwright Chromium against the real local GraphQL gateway and service stack. It signs in with a development administrator, creates a uniquely named draft with a fictional image, completes the publication review, publishes, cancels, deletes/restores, and opens the event-scoped registration ledger.

Start Docker Desktop and the backend, then run:

```powershell
docker compose up --build -d
bun run test:e2e:install
bun run test:e2e:event
```

The test creates a cancelled, restored event using a unique name. It never uses real people, venue details, or production credentials. On CI, traces, screenshots, and video are retained only when the test fails.

For a quick local run with an already installed Chromium-based browser, set `PLAYWRIGHT_CHROME_PATH` to its executable path. CI leaves this unset and installs Playwright's pinned Chromium build for reproducibility.

