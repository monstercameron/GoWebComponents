# v6 regression verification

Status: example sweep paused by the maintainer; /v6 release checks in progress.
This is a verification record, not a claim that every
example migration is complete. Canonical source migrations remain tracked in
`v6-examples-migration.md`.

## Requested acceptance

- [x] Build every inventoried canonical example with its intended native, Wasm or Windows
  desktop target; identify unsupported target combinations explicitly.
- [x] Pass root native tests and first-party nested-module tests.
- [x] Pass Wasm and hydration tests, accounting for skips.
- [ ] Pass the complete Playwright-Go suite, not only TestMainSuite.
- [x] Pass core race checks and Windows desktop smoke/fault probes.
- [x] Pass repository JavaScript tests.
- [x] Compile marked documentation samples and check generated server functions.
- [ ] Review results after fixes and report exact remaining limitations.

## Evidence

2026-09-08: seven JavaScript test files passed via `node --test`, including
desktop lifecycle, VS Code diagnostics, devtools bridge, and harness stats,
budgets, dashboard and integration checks. Node reported 22 test entries,
zero failures and zero skips; several entries contain additional internal
assertion suites. Log: `bin/test-results/v6-full-js.log`.

All 15 `gwc:build` documentation samples compiled under the `doccompile` gate.
Log: `bin/test-results/v6-doccompile.log`.

`gwc server check` confirms Atlas's generated server function is current.
Log: `bin/test-results/v6-server-functions-check.log`.

Other lanes are running under Luna/Sol agents, with core race checks coordinated
by the primary agent. No full-suite passing claim is made until results are
collected and failures resolved. No test assertion or performance budget may be
relaxed merely to obtain a pass.

Core race run passed on Windows/amd64 with the configured LLVM-MinGW compiler:
19 packages, 2,147 passing test/subtest events, zero failures and zero skips.
This covers the CI core race list plus desktop, interop and the server-interactive
POC; it is not a claim that every repository package was race-tested.
Log: `bin/test-results/v6-full-race.jsonl`.

Sol's uncached Node Wasm replay passed 29 packages with 2,390 passing test/subtest
events, zero failures and seven skips. The distinct hydration replay passed five
packages with 1,195 passing events and two overlapping skips. Three Wasm skips
are production-only tests: the complementary production-tagged run passed both
packages and all ten events without skips. Four tests deliberately invoke APIs
outside legal Wasm hook context and remain native-only coverage exclusions,
not Wasm passes. Chat markup and clipboard-unavailable regressions were included.
Logs: `wasm-full-node.jsonl`, `hydration-node.jsonl`, and
`wasm-production-node.jsonl` under `bin/test-results`.

The server-interactive timeout was caused by client mounting alongside server
markup, producing duplicate status IDs. Sol changed the client to public GWC
hydration and added an exact-one-status assertion. Focused browser, native and
Wasm build checks pass. Full browser verification remains in progress.

Luna's build matrix passed all 135 public Wasm packages (131 main.go entry
directories), 17 additional browser targets, 11 native server/tool targets, and
the separate desktop frontend/native module. Native-only static export and the
test-only Atlas performance package are not browser/native executable targets
respectively; desktop must be built inside its nested module.

An initial desktop probe used an old executable and failed a shared-counter
check. After the canonical `go run ./tools/build` refreshed frontend, bindings
and embedded executable, all five smoke/fault probes passed (including expected
exit 1 for missing binding/Wasm faults). Logs: `desktop-canonical-build.log` and
`desktop-smoke-fresh.log` under `bin/test-results`. Build outputs accidentally
written to the repository root were moved into `bin/v6-build-matrix-artifacts`;
no user source was removed.

The complete browser run has exposed a failing chat admin billing-events
assertion. Investigation is active; earlier focused passes do not override this
failure. Full project acceptance is still open.

The opt-in Atlas size diagnostic executed against a freshly built, stripped,
isolated client: raw 20,674,058 bytes; gzip 4,778,300; brotli 3,312,366. The test
passes as a measurement but reports the historical 1,600,000-byte M5 gzip target
MISSED by 3,178,300 bytes. Do not describe this as a passing size budget. Log:
`bin/test-results/atlas-perf-wasm-size.log`. Fixture recapture was not enabled:
it is an opt-in generator that overwrites checked-in fixtures, not an ordinary
regression assertion.

Uncached root native replay passed: 151 packages and 6,467 passing test/subtest
events, zero failures, four test skips plus five no-test packages. Log:
`bin/test-results/native-root-uncached-final.jsonl`. Root skips were fixture
recapture, size measurement (subsequently executed separately), and two symlink
privilege checks. Earlier elevated evidence at 14:52 executed both exact symlink
checks with zero skips; their tracked test/implementation files remain unchanged.
That prior evidence is not described as a fresh elevated replay.

Nested agenthub passed 25 test events and livereload passed 74. Livereload's
WebSocket session test now waits for asynchronous registration before asserting
status; the focused regression passed 50 repeats. Other desktop adapter checks
are being collected separately. Chat browser fixture fixes are still active.

The isolated desktop/wails adapter passed all three native tests, with zero
skips, and `go vet ./...` passed without diagnostics. Root, agenthub, livereload,
desktop/wails and wails-counter account for the maintained first-party modules.
Upstream dependencies, research comparison apps and generated golden fixture
modules are excluded from the independent-module sweep.

## Maintainer-directed release cutoff

The full browser sweep was stopped at the maintainer's request. Eight top-level
failures were observed: AdminBusinessWorkflow, AdminCustomersWorkflow,
AdminJourney, DashboardSettingsMutations, AdminRoleGuardsAndDeepLinks,
BillingSummary, BootCatalogFirstPaint and CrossAccountIsolation (all Example100).
The billing-events fixture fix passed its focused test afterward. A billing-plan
fixture fix passed the mutation stage but still failed the later settings route.
The role-guard correction requires a rerun after the module migration. The
partial log is retained in `bin/test-results/playwrightgo-full.log`; the sweep
did not complete and cannot be called green.

At cutoff, the maintainer approved migrating imports/modules to /v6 before
publishing v6.0.0 to main and the configured GitHub Pages site. Results recorded
above predate that import migration; post-migration release checks are recorded
separately rather than silently reusing old passes.

## Post-migration release checks

- `/v6` Wasm replay: 29 packages, 2,396 passing test/subtest events, zero
  failures, seven accounted skips; hydration: five packages, 1,195 passing
  events, zero failures, two overlapping skips. Production supplement passes
  ten events with no skips. Logs use the `v6-release-` prefix.
- The root Playwright package passes uncached in 100.97s. The focused chat role
  guard passes all three role scenarios; its Wasm unit tests pass. This does
  not reclassify the paused full example sweep as passing.
- An external consumer with a local candidate replacement compiles natively
  and for Wasm using only public `/v6/html` and `/v6/ui` imports.
- `govulncheck` with Go 1.26.3 failed on ten reachable standard-library
  advisories. A fresh scan with `GOTOOLCHAIN=go1.26.6` exits zero and reports no
  reachable vulnerabilities; it still lists two imported-package and eighteen
  required-module findings without reachable calls. Both logs are retained.
- Astra's bounded release review verified no Wails/native dependency in the
  Wasm UI/HTML/desktop graph and no further concrete role-guard blocker. It
  identified a release consumer gate that used `@latest`; that gate now checks
  the exact pushed tag, or the candidate checkout before manual tagging, and
  rejects a pre-existing manual tag pointing at a different commit.

Patched Go 1.26.6 desktop rebuild and all five smoke/fault probes pass. Production
Pages builds refreshed the site, 130 served example binaries, previews, source
mirrors and catalog; sitegen produced its output successfully.

One post-migration native attempt used GOTMPDIR inside repository `bin`, causing
four negative repository-discovery tests to see the parent module and causing
the dev watcher to ignore a fixture under its intentionally ignored bin path.
These five failures were retained, not hidden. The dev-loop regression passed
five repeats with an external temp directory without source changes. A clean
full rerun is in progress. No production deployment is claimed here.
