# GoGRPCBridge Dev to Prod Roadmap

This roadmap tracks the work needed to move the GoGRPCBridge integration from development-ready to production-ready in this repository.

Execution rules:
- Complete one checklist item at a time.
- Validate after each completed item.
- Record a checkpoint after each item with: completed todo, files changed, validation run, result, residual risk, and next suggested todo.

## S01 Scope and Ownership

- [x] S01.1 Define product scope in one sentence (what GoGRPCBridge is, and what it is not).
- [x] S01.2 Define target users (framework maintainers, app teams, OSS consumers).
- [x] S01.3 Assign owners for API, docs, CI, security, performance, and release.
- [x] S01.4 Define production-ready exit criteria as measurable gates.

Target users:
- Framework maintainers: contributors responsible for transport API quality, compatibility, and release governance.
- App teams: product teams building Go/WASM clients and Go backends that need browser-safe gRPC streaming.
- OSS consumers: external developers importing GoGRPCBridge directly and expecting stable docs, examples, and semver behavior.

Ownership model:
- API owner: transport maintainers (`third_party/GoGRPCBridge/pkg/grpctunnel` and migration policy).
- Docs owner: documentation maintainers (root docs plus GoGRPCBridge README and migration docs).
- CI owner: tooling maintainers (`tools/gwc` workflows plus `third_party/GoGRPCBridge/.github/workflows`).
- Security owner: security/reliability maintainers (gosec policy, origin/TLS guidance, threat model, checklists).
- Performance owner: performance maintainers (benchmark gates, baseline snapshots, trend policy).
- Release owner: release maintainers (tagging, changelog quality, release checklist, rollback policy).

Production-ready exit gates:
- CI gate: required test, lint, security, and browser lanes are green on default branch and pull requests.
- Quality gate: `go run ./third_party/GoGRPCBridge/tools/runner.go quality` passes in CI strict mode and emits `bin/quality/summary.json`.
- Security gate: high-severity/high-confidence `gosec` findings fail CI; threat model and release security checklist are published.
- Performance gate: benchmark baseline exists, trend comparison is enforced in CI, and release budgets are documented with fail thresholds.
- API gate: canonical API (`pkg/grpctunnel`) is the documented default, legacy path migration guidance is published, and exported API docs are complete.
- Release gate: release checklist, rollback/hotfix process, and changelog/migration notes are completed for each release tag.
- Ops gate: production runbook, observability contract, and SLO/Smoke validation procedures are published and linked from core docs.

## S02 Repo and Tooling Baseline

- [x] S02.1 Add one root doc for submodule lifecycle (`init`, `update`, `pin`, `verify`).
- [x] S02.2 Add one bootstrap command that verifies required tools and submodule state.
- [x] S02.3 Ensure bootstrap reports actionable failures (missing Go/protoc/Playwright and similar dependencies).
- [x] S02.4 Add a first-10-minutes contributor path and verify it on a clean machine.

## S03 API Surface Cleanup

- [x] S03.1 Promote `pkg/grpctunnel` as the canonical API in all examples and docs.
- [x] S03.2 Mark legacy entry points as compatibility-only with migration mapping.
- [x] S03.3 Add typed helper constructors for common client and server setups.
- [x] S03.4 Ensure every exported symbol has current GoDoc.
- [x] S03.5 Publish an API compatibility and deprecation policy (semver plus support window).

## S04 Example and Integration Hygiene

- [x] S04.1 Keep one minimal hello-world bridge example.
- [x] S04.2 Keep one production-shaped bridge example (`third_party/GoGRPCBridge/examples/production-bridge`).
- [x] S04.3 Add one explicit consumer example without local `replace` directives.
- [x] S04.4 Remove or flag stale and duplicate examples.
- [x] S04.5 Add an integration matrix showing where this repo depends on GoGRPCBridge.

## S05 Documentation System

- [x] S05.1 Rewrite README flow to quickstart-first, deep-docs-second.
- [x] S05.2 Add troubleshooting guidance for top failure modes.
- [x] S05.3 Add production configuration guidance (origin checks, TLS/WSS, timeouts).
- [x] S05.4 Fix encoding and readability issues in bridge docs (README, migration, and troubleshooting pages).
- [x] S05.5 Add a docs index with clear paths: Quickstart, API, Migration, Ops, Security.

## S06 Test Coverage Hardening

- [x] S06.1 Add focused tests for reconnect behavior.
- [x] S06.2 Add focused tests for cancellation and context deadlines.
- [x] S06.3 Add focused tests for malformed frames and protocol edge cases.
- [x] S06.4 Add focused tests for browser online/offline and visibility transitions.
- [x] S06.5 Add deterministic smoke coverage for tunnel dial plus unary plus stream.

## S07 CI and Quality Gates

- [x] S07.1 Align root and submodule workflows into one recommended CI path.
- [x] S07.2 Enforce lint plus unit plus wasm plus browser lanes as required checks.
- [x] S07.3 Enforce quality summary artifact output on every pull request.
- [x] S07.4 Enforce gofmt/goimports and static checks before merge.
- [x] S07.5 Add a fast pull-request lane and a full-gate lane with explicit trigger rules.

## S08 Security Readiness

- [x] S08.1 Publish threat model and trust boundaries.
- [x] S08.2 Add a security checklist for release sign-off.
- [x] S08.3 Enforce CI failure on high-severity and high-confidence findings.
- [x] S08.4 Add security-focused coverage for origin validation and unsafe defaults.
- [x] S08.5 Add guidance for auth propagation and token-handling boundaries.

## S09 Performance Program

- [x] S09.1 Establish benchmark baseline snapshots in repo.
- [x] S09.2 Add CI trend tracking against baseline.
- [x] S09.3 Define release budgets (latency, allocations, memory) with fail thresholds.
- [x] S09.4 Land at least one measured optimization and document before-and-after evidence.
- [x] S09.5 Add performance regression notes template to release notes.

## S10 Release Engineering

- [x] S10.1 Create release checklist (quality, docs, performance, security, migration).
- [x] S10.2 Ensure tags, changelog, and release assets are produced consistently.
- [ ] S10.3 Verify `go get github.com/monstercameron/GoGRPCBridge@latest` works from a clean consumer module. (blocked: upstream repository/module-path publishing mismatch)
- [ ] S10.4 Verify `pkg.go.dev` docs are complete and canonical. (blocked: depends on S10.3 external module resolution)
- [x] S10.5 Add rollback and hotfix process.

S10.3 blocker note (2026-03-27):
- Clean-module verification still fails because `github.com/monstercameron/GoGRPCBridge` repository resolution returns `Repository not found` while published source remains under `github.com/monstercameron/grpc-tunnel` with module-path mismatch.

S10.4 blocker note (2026-03-27):
- `pkg.go.dev` canonical verification is coupled to the same module-path/repository publishing issue; direct external resolution of `github.com/monstercameron/GoGRPCBridge` fails from a clean module context.

## S10A Go-Get Readiness Checks

- [x] S10A.1 Run module-path and repository-alignment check (`go.mod` path, git remote, and published module metadata).
- [ ] S10A.2 Publish and maintain a canonical repository at `github.com/monstercameron/GoGRPCBridge` that serves the module path declared in `go.mod`.
- [ ] S10A.3 Ensure semver tags are created on the canonical repository and resolve through module proxy as `github.com/monstercameron/GoGRPCBridge@vX.Y.Z`.
- [ ] S10A.4 Pass clean-consumer smoke test: `go mod init <tmp> && go get github.com/monstercameron/GoGRPCBridge@latest`.
- [ ] S10A.5 Pass clean-consumer compile test by importing `github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel` with no `replace`.
- [ ] S10A.6 Verify `pkg.go.dev` canonical page for `github.com/monstercameron/GoGRPCBridge` renders package docs and latest version.
- [x] S10A.7 Add CI gate that fails when the clean-consumer `go get` smoke test fails.

S10A analysis snapshot (2026-03-27):
- `third_party/GoGRPCBridge/go.mod` declares `module github.com/monstercameron/GoGRPCBridge`.
- `git -C third_party/GoGRPCBridge remote -v` still points to `https://github.com/monstercameron/grpc-tunnel`.
- `go list -m -json github.com/monstercameron/grpc-tunnel@latest` resolves `v0.0.10`, but `go get github.com/monstercameron/grpc-tunnel@latest` fails because that module declares path `github.com/monstercameron/GoGRPCBridge`.
- Clean consumer run:
  - `go mod init example.com/verify`
  - `go get github.com/monstercameron/GoGRPCBridge@latest`
  - Failure: `remote: Repository not found` for `https://github.com/monstercameron/GoGRPCBridge/`.
- Conclusion: this project is not yet go-get ready; the blocking item is canonical repo/module-path publication alignment.

S10A.7 completion note (2026-03-27):
- Added `Go Get Smoke` lane to `.github/workflows/gogrpcbridge-ci.yml` and wired it into both Fast PR Gate and Full Gate aggregation.

## S11 Production Operations

- [x] S11.1 Publish runbook for deploy, rollback, and incident triage.
- [x] S11.2 Define observability contract (logs, metrics, health checks, dashboards).
- [x] S11.3 Add SLOs for tunnel availability and streaming reliability.
- [x] S11.4 Add canary and smoke verification for production deploys.
- [x] S11.5 Add operational diagnostics for client and server tunnel state transitions.

## S12 Launch and Adoption

- [x] S12.1 Publish launch-ready value proposition and architecture explainer.
- [x] S12.2 Publish comparison guidance (where bridge beats REST and where it does not).
- [x] S12.3 Publish migration guide for existing users.
- [x] S12.4 Publish getting-started demo and copy-paste quickstart.
- [x] S12.5 Publish community workflow (issues, discussions, response expectations).

## S13 Post-Launch Maintenance

- [x] S13.1 Define triage SLA and bug-severity policy.
- [x] S13.2 Schedule dependency and toolchain updates.
- [x] S13.3 Schedule docs freshness review cadence.
- [x] S13.4 Track adoption metrics and top friction points.
- [x] S13.5 Feed backlog from production incidents and user feedback.

## S14 Enterprise Readiness Hardening

- [x] S14.1 Enforce CI security-policy lanes in root GoGRPCBridge workflow (`gosec` high/high fail policy plus reachable vuln scanning) and make them required checks.
- [ ] S14.2 Enforce full release gate pipeline with signed approvals and artifacted quality/performance/security evidence.
- [ ] S14.3 Close remaining reliability and security blockers in this roadmap (including unresolved release-publish blockers where this repo has direct control).
- [ ] S14.4 Move from OTel-compatible logs to full observability implementation (runtime metrics, trace spans, dashboards, and alert wiring).
- [ ] S14.5 Add secure backend transport guidance/enforcement for non-loopback deployments (TLS/mTLS boundary policy).
- [ ] S14.6 Add abuse controls for public endpoints (upgrade rate limiting, connection caps, and per-client controls).
- [ ] S14.7 Add governance enforcement for API lifecycle guarantees (compatibility policy checks and migration coverage in release gates).
- [ ] S14.8 Expand failure-mode validation with additional resilience and chaos-style coverage for reconnect/cancellation/malformed traffic under load.

## Checkpoints

### Checkpoint 2026-03-26B

- completed todo:
  - S14.1 Enforce CI security-policy lanes in root GoGRPCBridge workflow (`gosec` high/high fail policy plus reachable vuln scanning) and make them required checks.
- files changed:
  - `.github/workflows/gogrpcbridge-ci.yml`
  - `docs/GOGRPCBRIDGE_REQUIRED_CHECKS.md`
  - `docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`
- validation run:
  - `go run github.com/securego/gosec/v2/cmd/gosec@latest -severity high -confidence high -exclude G103 ./...` (from `third_party/GoGRPCBridge`)
  - `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` (from `third_party/GoGRPCBridge`)
- result:
  - Added a required `Security Scan` lane to `GoGRPCBridge CI`, wired it into Fast PR Gate and Full Gate dependencies, and updated required-check documentation to include the new lane.
- residual risk:
  - Branch protection settings in GitHub must still be configured to require the new `Security Scan` status check.
- next suggested todo:
  - S14.2 Enforce full release gate pipeline with signed approvals and artifacted quality/performance/security evidence.
