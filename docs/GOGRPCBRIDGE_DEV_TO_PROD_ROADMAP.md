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
- [ ] S10.3 Verify `go get github.com/monstercameron/grpc-tunnel@latest` works from a clean consumer module. (blocked: latest published tag still declares old module path)
- [ ] S10.4 Verify `pkg.go.dev` docs are complete and canonical. (blocked: depends on S10.3 external module resolution)
- [x] S10.5 Add rollback and hotfix process.

S10.3 blocker note (2026-03-27):
- Clean-module verification still fails because the current published `@latest` (`v0.0.10`) declares module path `github.com/monstercameron/GoGRPCBridge`.
- Repository/module source now aligns to `github.com/monstercameron/grpc-tunnel`; publish a new semver tag from this aligned state to clear the blocker.

S10.4 blocker note (2026-03-27):
- `pkg.go.dev` canonical verification is coupled to the same publish state; once a new aligned tag is published from `github.com/monstercameron/grpc-tunnel`, canonical docs should resolve.

## S10A Go-Get Readiness Checks

- [x] S10A.1 Run module-path and repository-alignment check (`go.mod` path, git remote, and published module metadata).
- [x] S10A.2 Publish and maintain a canonical repository at `github.com/monstercameron/grpc-tunnel` that serves the module path declared in `go.mod`.
- [ ] S10A.3 Ensure semver tags are created on the canonical repository and resolve through module proxy as `github.com/monstercameron/grpc-tunnel@vX.Y.Z`.
- [ ] S10A.4 Pass clean-consumer smoke test: `go mod init <tmp> && go get github.com/monstercameron/grpc-tunnel@latest`.
- [ ] S10A.5 Pass clean-consumer compile test by importing `github.com/monstercameron/grpc-tunnel/pkg/grpctunnel` with no `replace`.
- [ ] S10A.6 Verify `pkg.go.dev` canonical page for `github.com/monstercameron/grpc-tunnel` renders package docs and latest version.
- [x] S10A.7 Add CI gate that fails when the clean-consumer `go get` smoke test fails.
- [x] S10A.8 Add canonical publish-identity preflight command and enforce it in release automation.

S10A analysis snapshot (2026-03-27):
- `third_party/GoGRPCBridge/go.mod` declares `module github.com/monstercameron/grpc-tunnel`.
- `git -C third_party/GoGRPCBridge remote -v` still points to `https://github.com/monstercameron/grpc-tunnel`.
- `go list -m -json github.com/monstercameron/grpc-tunnel@latest` resolves `v0.0.10`, but `go get github.com/monstercameron/grpc-tunnel@latest` fails because that tag declares path `github.com/monstercameron/GoGRPCBridge`.
- Clean consumer run:
  - `go mod init example.com/verify`
  - `go get github.com/monstercameron/grpc-tunnel@latest`
  - Failure: module path mismatch against `github.com/monstercameron/GoGRPCBridge` in `v0.0.10`.
- Conclusion: source alignment is fixed in-repo, but this project is not yet go-get ready until a new semver tag is published with the aligned module path.

S10A.7 completion note (2026-03-27):
- Added `Go Get Smoke` lane to `.github/workflows/gogrpcbridge-ci.yml` and wired it into both Fast PR Gate and Full Gate aggregation.

S10A.8 completion note (2026-03-27):
- Added `go run ./tools/runner.go canonical-publish-check` to validate:
  - local `go.mod` module path matches canonical target (`github.com/monstercameron/grpc-tunnel`)
  - git `origin` remote matches canonical repository URL
  - clean-consumer `go get github.com/monstercameron/grpc-tunnel@latest` plus import compile smoke passes
- Wired canonical publish check into `third_party/GoGRPCBridge/.github/workflows/release.yml` so release tags fail fast when publishing identity is broken.

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
- [x] S14.2 Enforce full release gate pipeline with signed approvals and artifacted quality/performance/security evidence.
- [x] S14.3 Close remaining reliability and security blockers in this roadmap (including unresolved release-publish blockers where this repo has direct control).
- [x] S14.4 Move from OTel-compatible logs to full observability implementation (runtime metrics, trace spans, dashboards, and alert wiring).
- [x] S14.5 Add secure backend transport guidance/enforcement for non-loopback deployments (TLS/mTLS boundary policy).
- [x] S14.6 Add abuse controls for public endpoints (upgrade rate limiting, connection caps, and per-client controls).
- [x] S14.7 Add governance enforcement for API lifecycle guarantees (compatibility policy checks and migration coverage in release gates).
- [x] S14.8 Expand failure-mode validation with additional resilience and chaos-style coverage for reconnect/cancellation/malformed traffic under load.

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

### Checkpoint 2026-03-26C

- completed todo:
  - S14.2 Enforce full release gate pipeline with signed approvals and artifacted quality/performance/security evidence.
- files changed:
  - `third_party/GoGRPCBridge/.github/workflows/release.yml`
  - `third_party/GoGRPCBridge/pkg/bridge/bridge.go`
  - `third_party/GoGRPCBridge/pkg/bridge/conn_test.go`
  - `third_party/GoGRPCBridge/benchmarks/comparison_test.go`
  - `third_party/GoGRPCBridge/go.sum`
  - `docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`
- validation run:
  - `go run ./tools/runner.go quality` (from `third_party/GoGRPCBridge`)
  - `go run ./tools/runner.go quality-trend` (from `third_party/GoGRPCBridge`)
  - `go run github.com/securego/gosec/v2/cmd/gosec@latest -severity high -confidence high -exclude G103 ./...` (from `third_party/GoGRPCBridge`)
  - `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` (from `third_party/GoGRPCBridge`)
- result:
  - Added release-environment approvals and release preflight gates to release workflow (`quality`, benchmark trend, and security scans) with artifact uploads for quality/trend evidence.
  - Fixed strict lint blockers in bridge pooling and websocket conn tests so release quality gates pass.
  - Stabilized benchmark gate parsing by silencing runtime logs during benchmark runs.
- residual risk:
  - Signed approval policy depends on GitHub environment protection for `gogrpcbridge-release` (manual reviewers and branch/tag restrictions must be configured in repo settings).
  - `gosec` and `govulncheck` currently run via `go run ...@latest`, which can change behavior over time; pinning tool versions is still recommended for reproducible gates.
- next suggested todo:
  - S14.3 Close remaining reliability and security blockers in this roadmap (including unresolved release-publish blockers where this repo has direct control).

### Checkpoint 2026-03-26D

- completed todo:
  - S14.3 Close remaining reliability and security blockers in this roadmap (including unresolved release-publish blockers where this repo has direct control).
- files changed:
  - `.github/workflows/gogrpcbridge-ci.yml`
  - `third_party/GoGRPCBridge/.github/workflows/release.yml`
  - `docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`
- validation run:
  - `go run github.com/securego/gosec/v2/cmd/gosec@v2.25.0 -severity high -confidence high -exclude G103 ./...` (from `third_party/GoGRPCBridge`)
  - `go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...` (from `third_party/GoGRPCBridge`)
- result:
  - Closed direct-control security/reliability blockers by pinning security scanner versions in both root CI security lane and release workflow security gate.
  - Remaining release-publish blockers are external publication/alignment dependencies tracked in S10.3, S10.4, and S10A.2-S10A.6.
- residual risk:
  - Canonical repository/module-path publication is still unresolved outside this repo and continues to block clean `go get` and `pkg.go.dev` canonical resolution.
  - GitHub branch protection and environment reviewer settings must still be configured in repository settings.
- next suggested todo:
  - S14.4 Move from OTel-compatible logs to full observability implementation (runtime metrics, trace spans, dashboards, and alert wiring).

### Checkpoint 2026-03-26E

- completed todo:
  - S14.4 Move from OTel-compatible logs to full observability implementation (runtime metrics, trace spans, dashboards, and alert wiring).
- files changed:
  - `third_party/GoGRPCBridge/pkg/grpctunnel/observability.go`
  - `third_party/GoGRPCBridge/pkg/grpctunnel/server.go`
  - `third_party/GoGRPCBridge/pkg/grpctunnel/observability_test.go`
  - `third_party/GoGRPCBridge/OBSERVABILITY_CONTRACT.md`
  - `third_party/GoGRPCBridge/observability/DASHBOARD_QUERIES.md`
  - `third_party/GoGRPCBridge/observability/PROMETHEUS_ALERT_RULES.yaml`
  - `third_party/GoGRPCBridge/README.md`
  - `docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`
- validation run:
  - `go test ./pkg/grpctunnel -count=1` (from `third_party/GoGRPCBridge`)
  - `go test ./pkg/bridge -run "TestBuildBridgeLogLine_IncludesRequestAndTraceFields|TestBuildBridgeLogLine_Defaults|TestServeHTTP_LogsUpgradeFailureStructured" -count=1` (from `third_party/GoGRPCBridge`)
- result:
  - Added OTel runtime observability in canonical bridge handler path (`pkg/grpctunnel`) with request/session spans plus upgrade and connection metrics.
  - Added focused tests that verify OTel metric emission and trace-bearing structured logs in bridge failure paths.
  - Added dashboard query templates and Prometheus alert rule templates, and linked them into observability docs.
- residual risk:
  - `bridge_rpc_errors_total`, `bridge_streams_active`, and backend dial failure counters still require service-layer and backend middleware instrumentation outside current transport-only scope.
  - Production alert routing destinations and notification policy remain environment-specific wiring steps.
- next suggested todo:
  - S14.5 Add secure backend transport guidance/enforcement for non-loopback deployments (TLS/mTLS boundary policy).

### Checkpoint 2026-03-26F

- completed todo:
  - S14.5 Add secure backend transport guidance/enforcement for non-loopback deployments (TLS/mTLS boundary policy).
- files changed:
  - `third_party/GoGRPCBridge/pkg/bridge/bridge.go`
  - `third_party/GoGRPCBridge/pkg/bridge/bridge_test.go`
  - `third_party/GoGRPCBridge/README.md`
  - `third_party/GoGRPCBridge/THREAT_MODEL.md`
  - `docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`
- validation run:
  - `go test ./pkg/bridge -run "TestNewHandler_RequireLoopbackBackendRejectsNonLoopback|TestNewHandler_RequireLoopbackBackendAllowsLoopback|TestNewHandler_InvalidTargetGuard|TestHandleBridgeEndToEnd" -count=1` (from `third_party/GoGRPCBridge`)
- result:
  - Added strict startup enforcement for plaintext backend boundary policy via `bridge.Config.ShouldRequireLoopbackBackend`.
  - Handler now rejects non-loopback plaintext backend targets with explicit policy-violation errors and logs.
  - Updated threat model and README security guidance with explicit policy usage for production deployments.
- residual risk:
  - `pkg/bridge` backend path remains plaintext h2c by design; production deployments needing encrypted backend transport require TLS termination/mTLS at infrastructure boundaries or migration to architecture that keeps this hop private.
  - mTLS policy enforcement remains deployment-level because `pkg/bridge` does not yet expose backend TLS transport configuration.
- next suggested todo:
  - S14.6 Add abuse controls for public endpoints (upgrade rate limiting, connection caps, and per-client controls).

### Checkpoint 2026-03-26G

- completed todo:
  - S14.6 Add abuse controls for public endpoints (upgrade rate limiting, connection caps, and per-client controls).
- files changed:
  - `third_party/GoGRPCBridge/pkg/grpctunnel/api.go`
  - `third_party/GoGRPCBridge/pkg/grpctunnel/server.go`
  - `third_party/GoGRPCBridge/pkg/grpctunnel/abuse_control.go`
  - `third_party/GoGRPCBridge/pkg/grpctunnel/abuse_control_test.go`
  - `third_party/GoGRPCBridge/pkg/bridge/bridge.go`
  - `third_party/GoGRPCBridge/pkg/bridge/abuse_control.go`
  - `third_party/GoGRPCBridge/pkg/bridge/bridge_test.go`
  - `third_party/GoGRPCBridge/README.md`
  - `third_party/GoGRPCBridge/THREAT_MODEL.md`
  - `docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`
- validation run:
  - `go test ./pkg/grpctunnel -run "TestGetBridgeConfigError_AbuseControlValidation|TestBridgeAbuseGuard_ConnectionCaps|TestBuildBridgeHandler_RejectsUpgradeWhenRateLimitExceeded|TestBuildBridgeHandler_LogsUpgradeFailureWithTraceIDsWhenTracerConfigured" -count=1` (from `third_party/GoGRPCBridge`)
  - `go test ./pkg/bridge -run "TestNewHandler_RejectsNegativeAbuseControlLimits|TestServeHTTP_RejectsUpgradeWhenRateLimitExceeded|TestNewHandler_RequireLoopbackBackendRejectsNonLoopback|TestHandleBridgeEndToEnd" -count=1` (from `third_party/GoGRPCBridge`)
- result:
  - Added runtime abuse controls to both canonical and compatibility bridge handlers:
    - global active connection cap
    - per-client connection cap
    - per-client upgrade-attempt rate limit (1-minute window)
  - Enforced abuse-control validation and 429 rejections with structured rejection logs.
  - Updated security docs to include abuse-control hardening guidance.
- residual risk:
  - Abuse controls are in-process memory guards; distributed deployments still need shared rate limiting and WAF/ingress-level enforcement.
  - Client keying currently uses remote address host and should be paired with trusted proxy/IP-forwarding policy where applicable.
- next suggested todo:
  - S14.7 Add governance enforcement for API lifecycle guarantees (compatibility policy checks and migration coverage in release gates).

### Checkpoint 2026-03-26H

- completed todo:
  - S14.7 Add governance enforcement for API lifecycle guarantees (compatibility policy checks and migration coverage in release gates).
- files changed:
  - `third_party/GoGRPCBridge/tools/api_compat_guard/main.go`
  - `third_party/GoGRPCBridge/api_compatibility_baseline.json`
  - `third_party/GoGRPCBridge/.github/workflows/release.yml`
  - `.github/workflows/gogrpcbridge-ci.yml`
  - `docs/GOGRPCBRIDGE_REQUIRED_CHECKS.md`
  - `third_party/GoGRPCBridge/RELEASE_CHECKLIST.md`
  - `third_party/GoGRPCBridge/README.md`
  - `docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`
- validation run:
  - `go run ./tools/api_compat_guard check` (from `third_party/GoGRPCBridge`)
  - `go test ./pkg/grpctunnel -count=1` (from `third_party/GoGRPCBridge`)
  - `go test ./pkg/bridge -count=1` (from `third_party/GoGRPCBridge`)
- result:
  - Added API compatibility guard tooling with exported-symbol baseline enforcement for `pkg/grpctunnel` and `pkg/bridge`.
  - Added migration/compatibility documentation section checks in governance guard.
  - Wired API governance checks into release and root CI gates, and updated required-check documentation.
  - Added release checklist and baseline artifacts for governance review.
- residual risk:
  - Intentional breaking API changes still require explicit baseline regeneration discipline and release governance review.
  - Symbol-level compatibility checks do not cover all behavioral compatibility changes.
- next suggested todo:
  - S14.8 Expand failure-mode validation with additional resilience and chaos-style coverage for reconnect/cancellation/malformed traffic under load.

### Checkpoint 2026-03-26I

- completed todo:
  - S14.8 Expand failure-mode validation with additional resilience and chaos-style coverage for reconnect/cancellation/malformed traffic under load.
- files changed:
  - `third_party/GoGRPCBridge/pkg/grpctunnel/resilience_load_test.go`
  - `docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`
- validation run:
  - `go test ./pkg/grpctunnel -run "TestWrap_ReconnectBurst|TestWrap_CancellationBurst|TestBuildBridgeHandler_MalformedUpgradeBurst" -count=1` (from `third_party/GoGRPCBridge`)
- result:
  - Added focused resilience coverage for reconnect bursts, cancellation bursts, and malformed-upgrade bursts under concurrent load.
  - Verified transport rejects malformed bursts without upgrade success and handles cancellation/reconnect stress without hangs.
- residual risk:
  - These tests are process-local and deterministic; full chaos validation across distributed infrastructure layers still requires staging-level traffic fault injection.
- next suggested todo:
  - No unchecked S14 todos remain.

### Checkpoint 2026-03-26J

- completed todo:
  - Resolve repository/module-path mismatch in-source by aligning module path and import surface to `github.com/monstercameron/grpc-tunnel`.
  - Advance S10A.2 (canonical repository and declared module path alignment) to complete.
- files changed:
  - `third_party/GoGRPCBridge/go.mod`
  - root `go.mod`
  - `.github/workflows/gogrpcbridge-ci.yml`
  - `third_party/GoGRPCBridge/tools/runner.go`
  - `third_party/GoGRPCBridge/tools/runner_publish_test.go`
  - `third_party/GoGRPCBridge/README.md`
  - `third_party/GoGRPCBridge/TROUBLESHOOTING.md`
  - `third_party/GoGRPCBridge/docs/pages/TROUBLESHOOTING.md`
  - `third_party/GoGRPCBridge/examples/external-consumer/go.mod`
  - `third_party/GoGRPCBridge/examples/external-consumer/README.md`
  - `third_party/GoGRPCBridge/docs/pages/examples/external-consumer/README.md`
  - import-path updates across root bridge consumers and submodule examples/tests
  - `third_party/GoGRPCBridge/examples/_shared/proto/todos.proto`
  - `third_party/GoGRPCBridge/examples/_shared/proto/todos.pb.go`
  - `third_party/GoGRPCBridge/examples/_shared/proto/todos_grpc.pb.go`
  - `docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`
- validation run:
  - `go list -m github.com/monstercameron/grpc-tunnel` (from repo root)
  - `go test ./examples/100-ai-chat-wizard/client/app -run TestDoesNotExist -count=1` (from repo root)
  - `go test ./examples/100-ai-chat-wizard/server/app -run TestTransportHelpersCoverTunnelAndWasmResolution -count=1` (from repo root)
  - `go run ./tools/api_compat_guard check` (from `third_party/GoGRPCBridge`)
  - `go run ./tools/runner.go quality` (from `third_party/GoGRPCBridge`)
  - `go run ./tools/runner.go quality-trend` (from `third_party/GoGRPCBridge`)
  - clean-consumer remote smoke: `go mod init example.com/verify && go get github.com/monstercameron/grpc-tunnel@latest`
- result:
  - Module path mismatch is fixed in source and all imports/docs/workflows now target `github.com/monstercameron/grpc-tunnel`.
  - Submodule quality and governance gates pass with the aligned module path.
  - External `@latest` smoke still fails on published tag `v0.0.10` because it was cut before this alignment.
- residual risk:
  - S10.3/S10A.3/S10A.4/S10A.5/S10A.6 remain blocked until a new semver tag is published from this aligned state.
- next suggested todo:
  - Publish the next semver tag from `github.com/monstercameron/grpc-tunnel`, then rerun clean-consumer `go get` and pkg.go.dev verification.
