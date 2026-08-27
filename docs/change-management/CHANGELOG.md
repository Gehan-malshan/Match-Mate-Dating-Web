# MatchMate Change History

This file records architecture-significant and user-visible changes with complete before/after details. Follow [`README.md`](README.md) for the required format.

## Unreleased

### CHG-20260827-002 — Standardize the VS Code developer workspace

- **Status:** In progress (configuration prepared; awaiting review/merge)
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
