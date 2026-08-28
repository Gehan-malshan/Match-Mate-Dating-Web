# MatchMate Change History

This file records architecture-significant and user-visible changes with complete before/after details. Follow [`README.md`](README.md) for the required format.

## Unreleased

### CHG-20260828-010 — Automatically migrate and seed local PostgreSQL

- **Status:** In review (implementation complete; awaiting review/merge)
- **Date:** 2026-08-28
- **Affected:** Account migration runner, service image, local Compose topology, and development documentation

#### Before

Developers started PostgreSQL and then manually ran the migration and shared-account seed commands. The initial migration runner was not idempotent.

#### After

The development Compose topology waits for PostgreSQL health, runs an idempotent migration job, and then runs the development-only seed job. Existing complete pre-tracking schemas are safely baselined; partial schemas fail visibly. The runtime image contains separate API, migration, seed, and outbox executables. Automatic fake-account creation remains absent from production manifests.

#### Verification

- `go test ./...` passed.
- `go vet ./...` passed.
- Migration and seed command binaries compiled successfully.
- `docker compose ... config` and independent YAML parsing passed, including dependency conditions and resolved build context.
- A live Docker image/startup run remains for the developer terminal because this Codex process cannot access the local Docker engine.

### CHG-20260828-009 — Add hard-coded local test accounts

- **Status:** In review (implementation complete; awaiting review/merge)
- **Date:** 2026-08-28
- **Affected:** Account service development tooling, tests, Docker build, and developer documentation

#### Before

Every developer had to register and manually promote local accounts before testing member, community, moderator, and suspended journeys.

#### After

`go run ./cmd/seed-dev` idempotently creates four fake `.test` accounts with the public development password `MatchMateDev123!`. It restores known state on every run, hashes the password normally, and refuses to run outside `development` or `test`. There is no login bypass and no production account is created.

#### Verification

- `gofmt` completed.
- `go test ./...` passed, including seed environment and reserved-domain tests.
- `go vet ./...` passed.
- `go build ./cmd/seed-dev` passed.
- Live seed execution was not attempted because this Codex process did not have access to the local Docker engine; developers can run the documented command from their own terminal.

### CHG-20260828-008 — Correct the browser registration payload

- **Status:** In review (implementation complete; awaiting review/merge)
- **Date:** 2026-08-28
- **Affected:** Member registration request mapping and frontend regression tests

#### Before

The registration form spread its complete UI state into the Account API request. That included the UI-only `consent` boolean in addition to `consentVersion`; the Go API correctly rejects unknown JSON properties and returned `INVALID_JSON` / “Request body is invalid.”

#### After

The form maps API fields explicitly and sends only `email`, `password`, `nickname`, `dateOfBirth`, and `consentVersion`. The checkbox still gates submission locally, while the backend records and validates the canonical consent version. A regression test prevents `consent` from reappearing in the wire payload.

#### Verification

- `bun run typecheck` passed.
- `bun run test` passed 5 tests across 2 files, including the registration-payload regression.
- `bun run build` passed; the existing non-blocking large-chunk warning remains.

### CHG-20260828-007 — Implement the Account/Profile vertical slice

- **Status:** In review (implementation complete; production hardening evidence pending)
- **Date:** 2026-08-28
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Account/Profile Go service, PostgreSQL schema, authentication/session transport, REST and RabbitMQ contracts, member onboarding/profile frontend, local development, tests, security/data/architecture documentation
- **Decision/ADR:** [`../adr/0001-account-authentication-and-session-transport.md`](../adr/0001-account-authentication-and-session-transport.md)

#### Before

Account/Profile behavior existed only in architecture documentation. The repository had no Go service, database migration, authentication implementation, versioned Account OpenAPI/AsyncAPI contract, outbox publisher, member onboarding/profile screens, or executable Account/Profile tests.

#### After

The first executable Account/Profile vertical slice now covers adult registration and consent records, token-based email verification, Argon2id login, ES256 access tokens, rotating refresh sessions, logout/revocation, self profile and privacy preview, private preferences, safe community discovery, symmetric block exclusion, audited profile approval/hiding, deactivation, transactional outbox events, frontend registration/login/profile routes, health/log correlation, and local PostgreSQL/RabbitMQ support. Production email delivery and gateway-grade throttling remain explicit hardening work rather than hidden stubs.

#### Reason

Account/Profile is the first backend dependency in the approved implementation order and establishes identity, visibility, privacy, and eligibility foundations used by every later service.

#### Compatibility and migration

New v1 APIs, events, service database, and browser routes; no existing production behavior or data requires migration. Future changes must remain compatible with the accepted v1 contracts and versioned migrations.

#### Security, privacy, and moderation impact

High impact. Passwords and opaque tokens are hashed, access tokens contain no PII, refresh tokens rotate in HttpOnly cookies, profiles are private/pending by default, community DTOs use a strict allow-list, private preferences never appear in community responses/events/logs, contact details are rejected from profile text, blocks exclude both directions, and privileged profile decisions are audited.

#### Deployment and rollback

Target order: back up and migrate the Account database, deploy the API, deploy the outbox relay, configure gateway/origins/keys, smoke test, then deploy the frontend routes. Rollback the processes/frontend while preserving the new database and outbox history; the down migration is destructive and development-only.

#### Verification

- `go test ./...` passes account-service domain, authentication, privacy-projection, registration, and login/session unit tests.
- `go vet ./...` passes.
- `bun run typecheck`, `bun run test`, and `bun run build` pass for the member web application; the production bundle reports a non-blocking large-chunk warning.
- `docker compose ... config` validates the Account PostgreSQL/RabbitMQ topology. Runtime integration was not executed because Docker Desktop was not running in the verification environment.
- Database/RabbitMQ integration, browser E2E, load, security, backup/restore, and production verification-email tests remain required before release.

#### Documentation updated

Account service and member-web READMEs, implementation phase tracker, authentication/session ADR, OpenAPI, AsyncAPI, contract READMEs, environment examples, compose configuration, and this change history.


### CHG-20260828-006 — Recompose the member landing page

- **Status:** In progress (implementation complete; awaiting review/merge)
- **Date:** 2026-08-28
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Member landing-page structure, visual hierarchy, motion, accessibility, tests, design guidance, and member-web documentation
- **Decision/ADR:** No ADR required; this is a focused presentation-layer redesign within the existing React/Vite application and Midnight Chemistry system

#### Before

The landing page accurately described MatchMate but relied on a conventional left-aligned hero, a repeated four-card journey, a static compatibility panel, and limited interaction. The page did not yet use the repository-local `gpt-taste` guidance, and its visual rhythm did not fully express the cinematic, intentional character described by Midnight Chemistry.

#### After

The landing page uses a responsive AIDA composition with a floating glass navigation, the approved earlier full-bleed cinematic hero, a gapless `7 + 5` then `4 + 4 + 4` trust grid, a keyboard-operable horizontal journey accordion, a scrubbed explainability statement, a local-image event stage, and a truthful pre-launch action. The hero restores the two-line `Where mystery meets connection` headline, left editorial overlay, full background image, and privacy note instead of the rejected detached portrait composition. GSAP and `@gsap/react` add scoped entrance, text-reveal, and image-scale motion while reduced-motion users receive the complete static experience. The page continues to state the no-chat, no-ML, confirmed-booking, organizer-review, privacy, and consent boundaries without collecting data or claiming live availability.

#### Reason

The first public experience should feel distinct, premium, and intentional while communicating MatchMate's privacy, no-chat, deterministic no-ML, verified-booking, and consent boundaries clearly.

#### Compatibility and migration

Presentation-layer change only. No API, event, database, route, stored data, authentication, booking, payment, or deployment contract changes. `gsap@3.15.0` and `@gsap/react@2.1.2` are recorded through the existing app manifest and Bun lockfile; no second package manager or lockfile was introduced.

#### Security, privacy, and moderation impact

The page remains public and static, collects no data, exposes no community profiles, and does not imply operational registration, event inventory, or identity reveal. Motion must respect reduced-motion settings, and all interactive elements require keyboard focus and readable contrast.

#### Deployment and rollback

Rebuild and redeploy the member web application. Rollback restores the previous landing component, stylesheet, package manifest/lockfile, tests, and documentation; no backend or data rollback is required.

#### Verification

- `bun run typecheck:web` passed.
- `bun run test:web` passed four component tests covering product boundaries, navigation, accordion interaction, and approved logo use.
- `bun run build:web` produced the Vite production bundle.
- The test environment exercises the reduced-motion path so GSAP does not hide content when animation is disabled.
- Responsive desktop/tablet/mobile rules, keyboard focus, semantic navigation, local assets, overflow containment, and contrast-oriented tokens were reviewed in source.
- Git whitespace validation passed.

#### Documentation updated

Member-web README, canonical design guide, and this change history.

### CHG-20260828-004 — Adopt the supplied MatchMate logo

- **Status:** In progress (implementation complete; awaiting review/merge)
- **Date:** 2026-08-28
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Brand assets, member landing-page header/footer, favicon and browser metadata, base design documentation, frontend tests
- **Decision/ADR:** No ADR required; adopts user-supplied brand artwork within the existing Midnight Chemistry visual system

#### Before

The member landing page used a temporary circular gradient badge containing an italic `M`. The repository had no authoritative MatchMate logo source, transparent web mark, favicon, or documented logo-usage rules.

#### After

The supplied magenta-to-orange double-arch mark is MatchMate's authoritative logo. The repository preserves the supplied artwork, includes a transparent web-ready master and optimized navigation/favicon sizes, replaces the temporary header/footer marks, provides favicon and Apple touch metadata, and documents accessible logo usage and change control. The existing valid social-preview image remains in place because this change does not alter its truthful product content.

#### Reason

The project now has approved brand artwork that should be used consistently instead of the temporary generated letter badge.

#### Compatibility and migration

Visual and static-asset change only. Browser routes, APIs, events, databases, user data, and backend services are unchanged. Existing caches may temporarily retain the previous favicon until refreshed.

#### Security, privacy, and moderation impact

No user-data impact. Logo assets contain no personal information, external tracking, embedded script, or operational metadata. Accessible names remain textual rather than relying on the image alone.

#### Deployment and rollback

The frontend must be rebuilt and deployed to publish the new assets. Rollback restores the temporary mark and prior favicon metadata; no service or data rollback is required.

#### Verification

- The transparent master is 1254×1254 ARGB with transparent corner pixels.
- The optimized navigation mark is 256×256 and the favicon is 64×64.
- `bun run typecheck:web` passed.
- `bun run test:web` passed three component tests, including header/footer logo usage.
- `bun run build:web` produced the production bundle.
- The landing route, navigation logo, and favicon returned HTTP 200 in the local preview.
- Favicon/touch metadata, documentation links, and Git whitespace validation passed.

#### Documentation updated

Canonical design guide, member web README, member web metadata/component/test/style files, brand assets, and this change history.

### CHG-20260828-005 — Add project-scoped `gpt-taste` skill

- **Status:** In progress (implementation complete; awaiting review/merge)
- **Date:** 2026-08-28
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Repository-local agent skill and developer setup documentation
- **Decision/ADR:** No ADR required; the skill supplies optional agent guidance without changing application architecture or runtime behavior

#### Before

The working tree contained an uncommitted README and empty directories from an earlier multi-skill import, but no usable repository-scoped skill file.

#### After

The earlier import artifacts are removed. MatchMate contains one project-scoped skill at `.agents/skills/gpt-taste/SKILL.md`, created from the supplied Markdown instructions. It is discoverable only from this repository and remains subordinate to MatchMate's architecture, security, privacy, accessibility, and Midnight Chemistry design requirements.

#### Reason

The project needs one focused frontend design and GSAP motion skill without installing or maintaining the previous multi-skill bundle.

#### Compatibility and migration

No runtime, API, event, database, package, or user-data migration is required. The skill is a Markdown instruction artifact. Codex sessions may need to start a new turn before newly added project skills appear in discovery.

#### Security, privacy, and moderation impact

The skill receives no credentials or production data. Suggestions involving external images, dependencies, animation, or UI content remain subject to normal authorization, privacy, accessibility, content, and supply-chain review.

#### Deployment and rollback

No application deployment is required. Rollback removes `.agents/skills/gpt-taste` and the related documentation.

#### Verification

- Validate the skill name, description, frontmatter, and folder naming.
- Confirm only one project `SKILL.md` remains.
- Confirm no global Codex skill directory changed.
- Run Git whitespace validation and review the complete diff.

#### Documentation updated

Developer setup guide and this change history.

### CHG-20260828-003 — Group frontend applications and packages under `frontend`

- **Status:** In progress (migration complete; awaiting review/merge)
- **Date:** 2026-08-28
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Repository layout, Bun workspace paths and scripts, member/admin app paths, frontend package paths, developer/testing/design/implementation documentation
- **Decision/ADR:** No ADR required; organizational change preserves the existing monorepo, technology, ownership, and independent-deployment decisions

#### Before

Frontend applications were stored under root-level `apps/`, while frontend components, validation helpers, and telemetry helpers were stored under root-level `packages/`. Frontend code was therefore distributed across two top-level directories beside backend services, contracts, infrastructure, and system tests.

#### After

All frontend applications and frontend-only shared packages are grouped under `frontend/`. Deployable applications live under `frontend/apps/web` and `frontend/apps/admin`; shared frontend packages live under `frontend/packages/ui`, `frontend/packages/validation`, and `frontend/packages/telemetry`. The root Bun workspace and lockfile remain authoritative, and each frontend application remains independently buildable and deployable. `frontend/README.md` explains the boundary and supported commands.

#### Reason

A single frontend boundary makes the monorepo easier for developers and coding agents to navigate and simplifies future frontend-specific CI path filters without combining applications into one deployment.

#### Compatibility and migration

This is a repository-path migration only. Imports, workspace globs, scripts, documentation, CI path filters, Docker build contexts, and deployment definitions must use the new paths. No browser route, API, event, database, user-data, or runtime behavior changes.

#### Security, privacy, and moderation impact

No product-data or authorization impact. Existing privacy, safety, moderation, payment, and no-ML requirements remain unchanged and continue to apply through repository-level and frontend documentation.

#### Deployment and rollback

No deployment is performed. Future pipelines must build from the new paths. Rollback moves both frontend subtrees back to root-level `apps/` and `packages/`, restores Bun paths and documentation, and regenerates the lockfile without changing runtime data.

#### Verification

- `bun install --frozen-lockfile` passed with the migrated lockfile.
- `bun run typecheck:web` passed from `frontend/apps/web`.
- `bun run test:web` passed both component tests.
- `bun run build:web` produced the Vite production build under the new path.
- Active repository references were updated to the `frontend/` boundary.
- Local Markdown links and Git whitespace validation passed.

#### Documentation updated

Root README and package manifest, Bun lockfile, frontend boundary/app/package READMEs, developer guide, testing guide, design guide, implementation status, and this change history.

### CHG-20260828-002 — Establish Midnight Chemistry as the base design system

- **Status:** In progress (documentation complete; awaiting review/merge)
- **Date:** 2026-08-28
- **Owner:** Project team
- **Issue/PR:** Pending
- **Affected:** Canonical design documentation, repository reading rules, member web application guidance, project handbook index
- **Decision/ADR:** No architecture ADR required; this is a frontend visual-language and design-governance decision

#### Before

The repository had an implemented landing-page visual style but no canonical design-system document. Colors, typography, spacing, elevation, shapes, component appearance, responsive navigation, accessibility, and privacy-safe profile presentation could therefore drift between developers and coding agents.

#### After

`docs/design/README.md` defines **Midnight Chemistry** as MatchMate's base visual system. It provides canonical color, gradient, typography, spacing, radius, elevation, component, responsive, accessibility, privacy, implementation, and change-governance rules. Frontend developers and agents must read it before changing user interfaces. Where the supplied design reference contained overlapping values, the guide resolves them explicitly: `#131316` is the application background, `#0F0F12` is the deepest decorative backdrop, 4px is the atomic spacing unit with an 8px primary rhythm, and responsive gutters are defined by breakpoint. Discovery-card metadata is constrained to fields approved by MatchMate's privacy allow-list.

#### Reason

A repository-local base design system gives human developers and coding agents one consistent visual source of truth while preserving MatchMate's privacy, safety, accessibility, and no-ML product boundaries.

#### Compatibility and migration

Documentation-only change. Existing UI is not silently restyled by this change. New components must follow the guide; existing components should be aligned through focused, tested follow-up changes. No API, event, database, or stored-user-data migration is required.

#### Security, privacy, and moderation impact

The design guide is subordinate to repository security and privacy rules. Profile cards may render only explicitly approved community-profile fields and must never expose contact details or private matchmaking inputs. Error states, reports, blocks, consent, focus, contrast, reduced motion, and non-color status cues remain mandatory.

#### Deployment and rollback

No runtime deployment is required. Rollback removes the design guide and its documentation references; it does not modify the current compiled frontend. Removing the guide would restore the previous risk of inconsistent visual implementation.

#### Verification

- Confirm the canonical design guide contains every supplied token and design area.
- Confirm conflicting source values have an explicit repository interpretation.
- Confirm handbook, agent, root, and member-web references resolve.
- Confirm no runtime source, API, event, database, or dependency changed.

#### Documentation updated

Canonical design guide, `AGENTS.md`, root README, project handbook index, member web README, and this change history.

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
