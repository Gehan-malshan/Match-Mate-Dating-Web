# MatchMate Developer Workstation Setup

This is the canonical setup guide for developers and coding agents working in Visual Studio Code. It describes the current documentation-only repository and the toolchain that will be used as Go services, React applications, contracts, and Docker infrastructure are introduced.

## 1. Supported development model

The initial supported setup is:

```text
Windows 10/11
  -> Visual Studio Code
  -> Git for Windows
  -> local Go and Node.js toolchains
  -> Docker Desktop with WSL 2 engine
  -> PostgreSQL, RabbitMQ, Redis, and services through Docker Compose
```

Do not install PostgreSQL or RabbitMQ directly on Windows for this project unless an accepted decision changes the local-development model.

A Dev Container may be introduced after the first Go module, frontend workspace, and Docker Compose environment exist. Until then, a container configuration would guess tool versions and startup behavior that have not been implemented.

## 2. Required software

Install:

- Visual Studio Code stable.
- Git for Windows with Git Credential Manager.
- The stable Go version required by the repository's future `go.work`/`go.mod` files.
- Current Node.js LTS; use the version pinned by the future repository version file when present.
- Corepack for the repository-pinned pnpm version.
- Docker Desktop with the WSL 2 backend and Docker Compose.

Optional operating tools:

- A browser used for member/admin testing.
- Windows Terminal.
- An approved password manager for developer credentials.

Never install PayHere, database, broker, cloud, or application secrets as global shell variables shared across unrelated projects.

## 3. Verify the installation

Open PowerShell and run:

```powershell
git --version
go version
node --version
corepack --version
docker --version
docker compose version
code --version
```

When the root `package.json` includes a `packageManager` field, enable Corepack and let it use that pinned pnpm version:

```powershell
corepack enable
pnpm --version
```

Do not independently upgrade Go, Node, pnpm, linters, or generators beyond repository/CI versions. Propose a version change through the normal documented change process.

## 4. Clone and open the repository

Example:

```powershell
git clone https://github.com/Gehan-malshan/Match-Mate-Dating-Web.git
cd "Match-Mate-Dating-Web"
code .
```

The current local path may contain spaces. Git and VS Code support this, but always quote absolute paths in scripts and terminal commands.

After opening, trust the workspace only after confirming that it is the expected repository and reviewing checked-in tasks/configuration.

## 5. Create a VS Code profile

Recommended:

1. Open the VS Code gear menu.
2. Select **Profiles** and **Create Profile**.
3. Create an empty profile named `MatchMate Development`.
4. Open this repository with that profile.
5. Install the workspace-recommended extensions when VS Code prompts.

The profile isolates project tooling from unrelated work. Personal themes, fonts, keybindings, and UI layout stay in the user profile and are not committed.

## 6. Recommended extensions

The canonical list is `.vscode/extensions.json`.

| Extension | ID | Purpose |
|---|---|---|
| Go | `golang.go` | Go language server, navigation, formatting, tests, debugging |
| ESLint | `dbaeumer.vscode-eslint` | React/TypeScript lint diagnostics and explicit fixes |
| Prettier | `esbenp.prettier-vscode` | Frontend, JSON, Markdown formatting |
| Container Tools | `ms-azuretools.vscode-containers` | Dockerfile/Compose editing and container inspection |
| Dev Containers | `ms-vscode-remote.remote-containers` | Future reproducible container workspace |
| YAML | `redhat.vscode-yaml` | GitHub Actions, Compose, AsyncAPI/OpenAPI YAML validation |
| PostgreSQL | `ms-ossdata.vscode-pgsql` | Local database inspection when authorized |
| markdownlint | `davidanson.vscode-markdownlint` | Markdown consistency |
| Code Spell Checker | `streetsidesoftware.code-spell-checker` | Documentation and identifier spelling |
| GitHub Actions | `github.vscode-github-actions` | Workflow validation and run visibility |

To install from PowerShell:

```powershell
code --install-extension golang.go
code --install-extension dbaeumer.vscode-eslint
code --install-extension esbenp.prettier-vscode
code --install-extension ms-azuretools.vscode-containers
code --install-extension ms-vscode-remote.remote-containers
code --install-extension redhat.vscode-yaml
code --install-extension ms-ossdata.vscode-pgsql
code --install-extension davidanson.vscode-markdownlint
code --install-extension streetsidesoftware.code-spell-checker
code --install-extension github.vscode-github-actions
```

Optional personal extensions such as GitLens, REST Client, or Error Lens may be used, but the project must not require them to build, test, or operate.

## 7. Workspace behavior

Committed `.vscode/settings.json` provides:

- Format on save.
- Explicit ESLint fix action.
- `gofmt` and Go import organization.
- Prettier for frontend, JSON, and Markdown.
- YAML language-server formatting.
- LF line endings, final newline, and trailing-whitespace removal.
- Exclusion of generated dependency/build/coverage folders from file watching and search.

`.editorconfig` applies compatible formatting conventions outside VS Code. `.gitattributes` normalizes repository text to LF while preserving CRLF for Windows batch/command files and marks common media/document formats as binary.

Editor formatting improves feedback, but repository commands and CI remain authoritative.

## 8. Repository reading order

Before implementation work, read:

1. `AGENTS.md`.
2. Root `README.md`.
3. `docs/README.md`.
4. `docs/architecture/README.md`.
5. `docs/implementation/README.md`.
6. The affected app/service/package/contract/infrastructure README.
7. Relevant matchmaking, data, testing, security, ADR, runbook, and change-history documents.

Opening the repository successfully is not authorization to ignore architecture or documentation rules.

## 9. Go workspace setup when implementation begins

Each service will have its own `go.mod`. A root `go.work` will reference local modules:

```text
services/account-service
services/event-service
services/booking-service
services/payment-service
services/matchmaking-service
services/notification-service
services/moderation-service
```

After those files exist:

```powershell
go work sync
go env GOWORK
go test ./...
```

The exact test/lint commands will be documented and wrapped in repository tasks/scripts before they become required. Do not invent different local commands that bypass CI options.

## 10. Frontend workspace setup when implementation begins

The root pnpm workspace will eventually include:

```text
apps/web
apps/admin
packages/ui
packages/validation
packages/telemetry
```

After the root package and lock files exist:

```powershell
corepack enable
pnpm install --frozen-lockfile
pnpm lint
pnpm typecheck
pnpm test
pnpm build
```

Only run commands actually defined in the checked-in root `package.json`. ESLint, Prettier, TypeScript, and related tools must be local pinned development dependencies; do not rely on global installations.

## 11. Docker and local dependencies

Docker Desktop must be running before Compose commands. After checked-in Compose files exist:

```powershell
docker compose config
docker compose up -d
docker compose ps
docker compose logs --tail 100
```

Planned local containers include PostgreSQL databases, RabbitMQ, optional Redis, gateway, Go services, and web applications.

Use named development volumes and fictional seed data. Never mount or import production data. Do not delete shared volumes or databases unless the target and recovery effect are understood.

## 12. PostgreSQL extension use

Connect only to local/development databases using non-production credentials. Name connections clearly by environment and service, for example:

```text
matchmate-local-account
matchmate-local-booking
matchmate-local-payment
```

Do not connect one service with another service's credential. Do not edit schema manually; migrations are authoritative. Do not save passwords in repository files or screenshots.

## 13. Secrets and environment configuration

- Commit only `.env.example` templates containing safe variable names and non-secret examples.
- Keep real `.env` files ignored.
- Use separate credentials per service/environment.
- Never place JWT keys, PayHere secrets, database passwords, broker credentials, tokens, or PII in VS Code settings, tasks, launch files, Git history, logs, or documentation.
- Use an approved secret manager in deployed environments.

If a secret is committed, treat it as compromised: rotate/revoke it, remove exposure safely, and follow the incident/change process. Deleting the latest line is not sufficient.

## 14. Daily developer workflow

```text
git switch main
-> git pull --ff-only
-> create focused feature branch
-> read ticket and canonical docs
-> add draft before/after change entry
-> implement vertical slice and tests
-> run formatting/lint/test/contracts locally
-> review Git diff for secrets and unintended files
-> update docs/change record
-> push and open pull request
-> complete PR checklist and address CI/review
```

Do not work directly on `main` for feature changes.

## 15. First-time setup checklist

- [ ] Correct repository opened and Git remote verified.
- [ ] `MatchMate Development` VS Code profile created.
- [ ] Recommended extensions installed.
- [ ] Git, Go, Node, Corepack, Docker, Compose, and VS Code commands work.
- [ ] Docker Desktop uses WSL 2 and starts successfully.
- [ ] Workspace files use LF and save without unexpected formatting conflicts.
- [ ] `AGENTS.md` and project handbook read.
- [ ] No real secrets or production connections configured.
- [ ] Local build/test/Compose commands verified once they exist.

## 16. Troubleshooting

### Go features are unavailable

- Confirm the Go extension is enabled in the current profile/workspace.
- Run `go version` in the VS Code integrated terminal.
- Use **Go: Install/Update Tools** only with repository/CI compatibility in mind.
- Reopen the repository after `go.work` exists.

### ESLint or Prettier does not run

- Confirm the extension is enabled.
- Install repository dependencies after `package.json`/lockfile exist.
- Use the workspace version, not a global package.
- Check the VS Code Output panel for ESLint/Prettier diagnostics.

### Docker is unavailable

- Start Docker Desktop and wait for the engine.
- Verify WSL 2 integration.
- Run `docker version` and `docker compose version`.
- Ensure the `D:` project path is available to Docker Desktop if file-sharing configuration requires it.

### Git shows widespread line-ending changes

- Do not commit blindly.
- Confirm `.gitattributes`, `.editorconfig`, and VS Code `files.eol` are active.
- Separate intentional normalization from feature changes and review with the team.

### Commands fail because the path contains spaces

Quote the path: `"D:\My projects\Match-Mate-Dating-Web"`. Repository scripts must handle quoted paths. Do not hard-code one developer's absolute path.

## 17. Updating this setup

Changing required tool versions, extensions, workspace settings, package manager, Go workspace, Compose environment, Dev Container, tasks, or debug configuration requires:

1. Before/after entry in the change history.
2. Compatibility impact for all developers and CI.
3. Updates to this guide, `.vscode`, version files, CI, and relevant READMEs.
4. Verification on a clean setup or documented equivalent.
5. Migration/rollback guidance when the change can interrupt existing work.

