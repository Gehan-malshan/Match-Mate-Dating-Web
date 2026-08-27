# GitHub Configuration

Future repository templates, ownership rules, dependency automation, security configuration, and GitHub Actions workflows will be maintained under this directory.

## Current governance

- `pull_request_template.md` enforces before/after, impact, testing, documentation, migration, security, and rollback review.
- Root `AGENTS.md` defines mandatory engineering behavior.

## Planned CI/CD

```text
Pull request -> always-running CI gate
 -> documentation/contracts
 -> affected frontend/service tests
 -> migration and security checks
 -> one aggregate required result

Merge to main -> build changed image once -> scan/SBOM/sign -> registry
 -> staging deploy -> smoke/E2E -> protected production approval
 -> progressive production rollout -> verify/rollback criteria
```

Use reusable workflows for Go services, frontend, container build, and deployment. Use path-aware change detection while keeping an always-running required gate. Grant minimum workflow permissions, pin third-party actions to immutable SHAs, and prefer OIDC over long-lived cloud credentials.

Do not add deployment until a component is buildable and a target environment/runbook is approved.

