# Frontend Workspace

This directory contains every MatchMate frontend application and frontend-only shared package.

```text
frontend/
|-- apps/
|   |-- web/                 Unified role-aware React/TanStack runtime
|   `-- admin/               Admin UI source bundled by the web runtime
`-- packages/
    |-- graphql-client/      Shared GraphQL HTTP/session client
    |-- ui/                  Shared design tokens and React components
    |-- validation/          Zod schemas and generated client helpers
    `-- telemetry/           Browser/frontend observability helpers
```

The root `package.json` and `bun.lock` manage this workspace. Do not introduce a second lockfile inside `frontend` or an application/package.

The current decision is one frontend deployment. Member and administrator route groups remain separately protected, and backend authorization remains authoritative. Shared packages must not contain backend service domain models, database entities, authoritative business validation, credentials, or provider secrets.

Before changing frontend code, read:

1. [`../AGENTS.md`](../AGENTS.md).
2. [`../docs/design/README.md`](../docs/design/README.md).
3. The affected application/package README.
4. Relevant architecture, security, testing, and change-management guides.

Run the unified frontend from the repository root:

```powershell
bun run dev
```

This starts one application at `http://localhost:5173`. Administrators use `/admin`; members use separately protected member routes. Stop it with `Ctrl+C`.

Build and test the unified application:

```powershell
bun install --frozen-lockfile
bun run dev:web
bun run typecheck:web
bun run test:web
bun run build:web
```

The browser sends all data operations to `VITE_GRAPHQL_API_URL`. The GraphQL gateway forwards them to domain-owned REST services; the browser never calls service ports directly.
