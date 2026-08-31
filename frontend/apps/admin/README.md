# Administrator UI Source

This directory contains the existing administrator workspace components and styles. It is no longer a separately deployed or separately started application: `frontend/apps/web/src/routes/AdminPage.tsx` bundles this source into the unified frontend at `/admin`.

Members and administrators share `http://localhost:5173/login`. The Account-issued role directs administrators to `/admin`. The web route guard, GraphQL BFF, Event Service, and Matchmaking Service all enforce administrator access; hiding navigation is never treated as authorization.

Use the unified commands from the repository root:

```powershell
bun run dev
bun run typecheck:web
bun run test:web
bun run build:web
```

Admin API operations share the web application's GraphQL client and `VITE_GRAPHQL_API_URL`, defaulting to `http://localhost:8080/graphql`. This directory intentionally has no Vite entry point, development server, or production build of its own. Do not reintroduce direct browser calls to Account, Event, or Matchmaking service ports.
