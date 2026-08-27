# MatchMate Project Handbook

This documentation is the canonical, repository-local source of truth for MatchMate. It is written for developers, reviewers, operators, and coding agents who may have no access to external business plans or architecture documents.

## Mandatory reading order

1. Read the repository-level [`AGENTS.md`](../AGENTS.md).
2. Read [`architecture/README.md`](architecture/README.md) to understand the complete system.
3. Read [`implementation/README.md`](implementation/README.md) to identify the current phase and permitted next work.
4. Read [`development/README.md`](development/README.md) before configuring tools or running local dependencies.
5. Read the relevant service or application README before changing that area.
6. Read the specialized guide for matchmaking, data, testing, security, or operations when the change touches that concern.
7. Read [`design/README.md`](design/README.md) before designing or changing any frontend interface or shared UI component.

## Canonical documentation map

| Topic | Canonical file | Must be updated when |
|---|---|---|
| Product and system architecture | [`architecture/README.md`](architecture/README.md) | Capabilities, boundaries, interactions, states, security, or deployment change |
| Developer workspace | [`development/README.md`](development/README.md) | Tool versions, VS Code, extensions, package manager, Compose, Dev Container, tasks, or local workflow change |
| Implementation sequence | [`implementation/README.md`](implementation/README.md) | Phases, dependencies, deliverables, or completion criteria change |
| Base design system | [`design/README.md`](design/README.md) | Colors, typography, spacing, shape, elevation, components, responsive behavior, imagery, motion, or UI accessibility change |
| Matchmaking | [`matchmaking/README.md`](matchmaking/README.md) | Questions, filters, weights, optimizer, reveal, responses, or fairness rules change |
| Data architecture | [`data/README.md`](data/README.md) | Tables, ownership, migrations, retention, constraints, or consistency change |
| Testing and CI | [`testing/README.md`](testing/README.md) | Test levels, required cases, tooling, or quality gates change |
| Security and privacy | [`security/README.md`](security/README.md) | Authentication, authorization, visibility, moderation, secrets, or retention change |
| REST APIs | [`api/README.md`](api/README.md) and `contracts/openapi/` | Endpoint behavior, schemas, errors, versioning, or authorization change |
| RabbitMQ events | `contracts/asyncapi/` | Producers, consumers, routing, envelopes, payloads, or versions change |
| Architecture decisions | [`adr/README.md`](adr/README.md) | A significant decision is proposed, accepted, replaced, or reversed |
| Operations | [`runbooks/README.md`](runbooks/README.md) | Deployment, alerts, recovery, reconciliation, or incident behavior changes |
| Change history | [`change-management/CHANGELOG.md`](change-management/CHANGELOG.md) | Every user-visible or architecture-significant change |

## Documentation authority

When files disagree, use this precedence and fix the conflict in the same change:

1. Accepted ADRs for explicit architecture decisions.
2. Versioned OpenAPI and AsyncAPI contracts for integration behavior.
3. The architecture, data, matchmaking, testing, and security handbooks.
4. Service and application README files.
5. Source-code comments.

Do not silently choose one conflicting description. Record the discrepancy, identify the owner, and update all affected sources of truth.

## Decision labels

Use these labels when a statement is not equally certain:

- **DECISION:** approved target behavior that implementation must follow.
- **ASSUMPTION:** temporary working input requiring validation.
- **OPEN QUESTION:** unresolved product, legal, operational, or technical decision with an owner.
- **DEPRECATED:** behavior retained temporarily with a removal condition and date.

## Documentation maintenance rule

Documentation changes are required in the same pull request as implementation changes. Every significant change must include a before/after entry in the change log, affected contracts, migrations, tests, operational impact, and rollback notes. A pull request is incomplete when its implementation and documentation disagree.
