# GoGRPCBridge Required Checks

This document defines the required check lanes for GoGRPCBridge in this repository.

## Workflow

- Workflow file: `.github/workflows/gogrpcbridge-ci.yml`
- Workflow name: `GoGRPCBridge CI`

## Required Lanes

### Fast PR Gate (default on every pull request)

Configure branch protection to require these checks:

1. `GoGRPCBridge CI / Lint Lane`
2. `GoGRPCBridge CI / Security Scan`
3. `GoGRPCBridge CI / API Governance`
4. `GoGRPCBridge CI / Unit Lane`
5. `GoGRPCBridge CI / WASM Lane`
6. `GoGRPCBridge CI / Root Integration Smoke`
7. `GoGRPCBridge CI / Go Get Smoke`
8. `GoGRPCBridge CI / Fast PR Gate`

### Full Gate (push/manual and labeled PRs)

The full gate adds browser coverage:

1. `GoGRPCBridge CI / Browser Lane`
2. `GoGRPCBridge CI / Security Scan`
3. `GoGRPCBridge CI / API Governance`
4. `GoGRPCBridge CI / Go Get Smoke`
5. `GoGRPCBridge CI / Benchmark Trend Lane`
6. `GoGRPCBridge CI / Full Gate`

Trigger rules:

- Runs automatically on `push` to `main`
- Runs on `workflow_dispatch`
- Runs on pull requests only when label `gogrpcbridge-full-gate` is present

## Branch Protection Setup

In GitHub repository settings:

1. Open `Settings` -> `Branches`.
2. Edit protection for `main` (or create a ruleset targeting protected branches).
3. Enable `Require status checks to pass before merging`.
4. Add the required checks listed above.
5. Enable `Require branches to be up to date before merging`.
6. Enable `Require review from Code Owners` and keep `.github/CODEOWNERS` current for workflow and `third_party/GoGRPCBridge` paths.

## Notes

- Required-check enforcement is a repository settings concern and cannot be fully enforced by code alone.
- Keep check names stable; renaming a workflow job can silently remove enforcement until settings are updated.
