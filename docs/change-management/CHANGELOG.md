# MatchMate Change History

This file records architecture-significant and user-visible changes with complete before/after details. Follow [`README.md`](README.md) for the required format.

## Unreleased

### CHG-20260828-001 — Introduce the MatchMate member landing page

- **Status:** In progress (implementation complete; awaiting review/merge)
- **Date:** 2026-08-28
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Root Bun workspace, `apps/web`, public marketing experience, generated visual assets, frontend tests and documentation
- **Decision/ADR:** No ADR required; implements the approved Bun/Vite/React/TanStack frontend direction without changing service boundaries

#### Before

The repository documented a planned member web application but contained no frontend package manifest, runtime entry point, routes, visual assets, tests, or production build. Visitors could not view a MatchMate marketing experience from this repository.

#### After

The repository contains an independently buildable Bun workspace for a responsive MatchMate landing page using Vite, React, TypeScript, TanStack Router, and TanStack Query. The page communicates privacy-first curated events and explainable rule-based matching without presenting unverified business metrics or implying machine learning. It includes original project-local hero, event, and social-preview imagery; truthful pre-launch states; responsive layouts; keyboard focus; reduced-motion handling; and component tests for core boundaries and navigation.

#### Reason

The project needs its first user-visible web vertical slice, based on the supplied Google Stitch visual reference while following the repository's established technology, privacy, safety, and no-ML boundaries.

#### Compatibility and migration

This is a new public frontend with no existing browser contract or stored data to migrate. The root will gain a Bun workspace and lockfile. Backend API contracts are unchanged because the first landing page uses static, truthful product content and no live service integration.

#### Security, privacy, and moderation impact

The page collects no personal data, stores no authentication token, and exposes no community profiles. Login and registration remain clearly identified as future routes until those journeys are implemented. Generated imagery contains fictional adults and no contact or identity information.

#### Deployment and rollback

The member web app is independently buildable and can be deployed separately from all Go services. Production hosting is not selected by this change. Rollback removes the new workspace/app files and restores the documentation-only repository; no database, event, payment, or service rollback is required.

#### Verification

- `bun install --frozen-lockfile` completed without lockfile changes.
- `bun run typecheck:web` passed.
- `bun run test:web` passed two component tests.
- `bun run build:web` produced the Vite production bundle.
- The local Vite preview returned HTTP 200 for the landing route with the expected title.
- The generated project-local images and social-preview metadata were verified.

#### Documentation updated

Root README, member web README, implementation phase status, and this change history.

### CHG-20260827-003 — Adopt Bun for frontend package management and scripts

- **Status:** In progress (documentation updated; awaiting review/merge)
- **Date:** 2026-08-27
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Root technology stack, member web application guidance, architecture tooling decision, developer workstation setup, future frontend workspace and CI commands
- **Decision/ADR:** Bun selected before frontend implementation; Vite remains the frontend development server and production bundler

#### Before

The repository development guide selected Node.js, Corepack, and pnpm for the future frontend workspace. Planned commands used `pnpm install`, `pnpm lint`, `pnpm typecheck`, `pnpm test`, and `pnpm build`, with a future pnpm workspace and lockfile.

#### After

Bun is the single selected frontend package manager and script runner. The future monorepo uses npm-compatible workspaces in the root `package.json`, an approved pinned Bun version, and one `bun.lock`. Developers and CI use `bun install --frozen-lockfile` and `bun run <script>`. Vite remains the frontend development server and production bundler, and TypeScript type checking remains an explicit script. pnpm, npm, and Yarn lockfiles are prohibited.

#### Reason

The team chose Bun before frontend code or lockfiles existed, making this the lowest-risk point to standardize faster dependency installation and script execution without a package-manager migration.

#### Compatibility and migration

No frontend package files or dependencies exist yet, so no lockfile conversion or dependency migration is required. Future root/app/package manifests, CI, Dockerfiles, and Dev Container configuration must use the pinned Bun version. Node.js may be added only for a documented tool compatibility requirement.

#### Security, privacy, and moderation impact

No user data impact. Bun versions and dependencies must be pinned and scanned in CI under the same supply-chain controls as other tools.

#### Deployment and rollback

No runtime deployment. Before frontend implementation, rollback requires documentation changes only. After `bun.lock` and CI exist, changing package managers requires a separate migration plan and reproducibility verification.

#### Verification

- Confirm no current-state documentation instructs developers to use pnpm or Corepack.
- Confirm Bun/Vite responsibilities are unambiguous.
- Confirm all local Markdown links and JSON files remain valid.
- Confirm the unrelated local implementation-guide modification is preserved.

#### Documentation updated

Root README, architecture guide, web application README, developer workstation guide, and change history.

### CHG-20260827-002 — Standardize the VS Code developer workspace

- **Status:** Released; package-manager guidance superseded by CHG-20260827-003
- **Date:** 2026-08-27
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** VS Code recommendations/settings, editor/line-ending conventions, developer setup documentation, handbook reading order
- **Decision/ADR:** No architecture ADR required; implements the planned VS Code/Go/React/Docker development model

#### Before

Developers had an informal list of suggested VS Code extensions and tools, but the repository did not recommend extensions automatically, apply shared workspace settings, define cross-editor/line-ending conventions, or contain a reproducible first-time workstation guide. Each developer or agent could configure formatting, package tooling, databases, and Docker differently.

#### After

The repository recommends the agreed Go, React/TypeScript, container, PostgreSQL, YAML, GitHub Actions, and documentation extensions through `.vscode/extensions.json`. Shared settings define formatting, ESLint behavior, Go formatting/imports, LF line endings, and monorepo exclusions. `.editorconfig` and `.gitattributes` align editors and Git. The development guide documents prerequisites, installation verification, VS Code profile setup, future Go/pnpm workspace behavior, Docker dependencies, secrets, daily workflow, troubleshooting, and setup-change governance.

#### Reason

A consistent setup reduces environment-specific failures and gives new developers and coding agents one repository-local onboarding process.

#### Compatibility and migration

No runtime compatibility impact. Existing developers should install recommended extensions and review line-ending changes before committing. Go/Node/pnpm versions remain governed by future repository version files rather than hard-coded machine-global versions.

#### Security, privacy, and moderation impact

The guide prohibits secrets and production data/connections in VS Code configuration and local test environments. No user data is processed by this change.

#### Deployment and rollback

No deployment. Reverting removes shared editor guidance but does not change runtime behavior. Do not normalize unrelated files in a feature commit without review.

#### Verification

- Parse `extensions.json` and `settings.json` as JSON.
- Verify all local Markdown links.
- Verify EditorConfig/Git attributes express LF and binary conventions.
- Confirm workspace configuration contains no absolute developer path or secret.
- Confirm Git diff contains only intended setup/documentation files.

#### Documentation updated

Root README, project handbook index, `AGENTS.md`, `.vscode/README.md`, developer setup guide, and change history.

### CHG-20260827-001 — Establish repository-local architecture and agent workflow

- **Status:** In progress (documentation prepared; awaiting review/merge)
- **Date:** 2026-08-27
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Root documentation, agent instructions, architecture, implementation, matchmaking, data, testing, service READMEs, pull-request process
- **Decision/ADR:** Documentation governance established by `AGENTS.md`; no product ADR required

#### Before

The repository contained a monorepo folder skeleton and short README placeholders. The full architecture existed outside the repository, so a new developer or coding agent could not independently determine service dependencies, detailed workflows, matchmaking rules, proposed data ownership, testing expectations, implementation phases, or documentation maintenance requirements.

#### After

The repository contains a canonical project handbook, mandatory `AGENTS.md`, detailed architecture, implementation sequence, deterministic matchmaking specification, database ownership/schema guidance, testing strategy, service-level implementation references, pull-request checklist, and this before/after change process. Agents must update implementation and documentation together.

#### Reason

The team uses developer agents that may not have external documents or previous conversation context. Repository-local documentation is required to prevent inconsistent architecture and undocumented behavior changes.

#### Compatibility and migration

Documentation-only change. No runtime API, event, database, or deployment migration is required.

#### Security, privacy, and moderation impact

The new handbook makes existing privacy, moderation, payment, identity, and matchmaking constraints explicit. No user data is processed by this change.

#### Deployment and rollback

No runtime deployment. Reverting would remove the repository-local source of truth and is not recommended.

#### Verification

- Verify all Markdown links resolve within the repository.
- Verify architecture ownership is consistent across handbook and service READMEs.
- Verify the pull-request template references the canonical change process.
- Verify Git contains only intended Markdown documentation changes.

#### Documentation updated

All handbook, agent, service, and pull-request documentation introduced or expanded by this change.

---

## Entry template

Copy the following block under **Unreleased** for each new change.

```markdown
### CHG-YYYYMMDD-NNN — Concise outcome

- **Status:** Proposed
- **Date:** YYYY-MM-DD
- **Owner:** Unassigned
- **Issue/PR:** Pending
- **Affected:**
- **Decision/ADR:**

#### Before

#### After

#### Reason

#### Compatibility and migration

#### Security, privacy, and moderation impact

#### Deployment and rollback

#### Verification

#### Documentation updated
```
