# Frontend Workspace

This directory contains every MatchMate frontend application and frontend-only shared package.

```text
frontend/
|-- apps/
|   |-- web/                 Member-facing React/TanStack application
|   `-- admin/               Organizer and moderation application
`-- packages/
    |-- ui/                  Shared design tokens and React components
    |-- validation/          Zod schemas and generated client helpers
    `-- telemetry/           Browser/frontend observability helpers
```

The root `package.json` and `bun.lock` manage this workspace. Do not introduce a second lockfile inside `frontend` or an application/package.

Grouping code beneath `frontend/` does not create one deployment. Each application remains independently buildable, testable, containerizable, and deployable. Shared packages must not contain backend service domain models, database entities, authoritative business validation, credentials, or provider secrets.

Before changing frontend code, read:

1. [`../AGENTS.md`](../AGENTS.md).
2. [`../docs/design/README.md`](../docs/design/README.md).
3. The affected application/package README.
4. Relevant architecture, security, testing, and change-management guides.

Run supported commands from the repository root. For the member web application:

```powershell
bun install --frozen-lockfile
bun run dev:web
bun run typecheck:web
bun run test:web
bun run build:web
```

