# Change Management and Documentation Synchronization

Every architecture-significant or user-visible change must be understandable from repository history without external conversation context. Update implementation, tests, contracts, migrations, architecture/security documentation, affected READMEs, ADRs, and runbooks in the same pull request when applicable.

The repository no longer maintains a shared `CHANGELOG.md`. Record the following before/after information in the pull-request description and commit history instead:

- Previous observable behavior.
- New behavior and reason for the change.
- API, event, data, compatibility, and migration effects.
- Security, privacy, moderation, deployment, and rollback effects.
- Tests, builds, manual checks, and operational evidence.
- Documentation updated by the change.

Add an ADR for service boundaries, data ownership, communication/consistency strategy, authentication/authorization/privacy, major technology choices, compatibility policy, matchmaking algorithm class, or production availability/recovery decisions.

Before completion, verify current documentation matches the implementation, future work is labeled, service ownership is consistent, examples match contracts, security effects are explicit, and the pull request can be understood without the original conversation.
