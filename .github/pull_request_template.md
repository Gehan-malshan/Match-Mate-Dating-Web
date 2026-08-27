## Summary

Describe the user or operational outcome, not only the files changed.

## Before and after

- **Before:**
- **After:**
- **Reason for change:**

## Scope

- Applications/services affected:
- APIs affected:
- Events affected:
- Databases/migrations affected:
- Privacy/security/moderation impact:
- Deployment/operations impact:

## Verification

- [ ] Unit tests added or updated
- [ ] Component/integration tests added or updated
- [ ] Contract tests added or updated
- [ ] End-to-end tests added or updated where required
- [ ] Failure, performance, or security tests added where required
- [ ] Manual verification steps and results recorded

## Documentation

- [ ] Canonical handbook pages updated
- [ ] Affected service/application README updated
- [ ] OpenAPI updated when REST behavior changed
- [ ] AsyncAPI updated when event behavior changed
- [ ] ADR added or updated for a significant decision
- [ ] `docs/change-management/CHANGELOG.md` includes complete before/after details

## Data and compatibility

- [ ] Migration and backfill plan documented
- [ ] Backward compatibility verified or migration window documented
- [ ] Rollback/forward-recovery steps documented
- [ ] No unauthorized cross-service database access introduced

## Final checklist

- [ ] No secrets, tokens, unnecessary PII, or local artifacts committed
- [ ] Logging and metrics do not expose sensitive data
- [ ] Architecture invariants in `AGENTS.md` remain satisfied
- [ ] Reviewer can understand the complete change from this PR and repository documentation alone

