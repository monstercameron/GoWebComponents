# Feature Backlog

Gap analysis against React/Next/Solid ecosystems (2026-06-11). Ordered by
impact; exactly three active items carry the next-work marker.

## V7 — Go 1.27, performance, flexible APIs, errors, OTel, tests (planned 2026-10-03)

Goals: Go 1.27+, adopt the new platform features, faster rendering, more flexible
APIs, better error messages, better OTel, refined tests, full bug hunt. Plan:
`C:\Users\mreca\.claude\plans\virtual-sprouting-quill.md` (copy into `docs/plans/`
if it should live in the repo). Tags: **[impl]** build it, **[research]** spike or
measure first. Every perf claim is gated on the V7-P0 baseline. One todo at a time;
`gwc verify` + clean `gwc lint` is the definition of done.

Decisions recorded: v7 is a breaking major with a generated migration guide; heavy
tool deps move to a nested tools module; core OTel is API-only (no SDK, never in the
wasm build); Go floor is `go 1.27.0`; v6.1.0 is finished and tagged before v7 branches.
The v6.1-first release decision is superseded by the maintainer decision recorded on
2026-10-21: release v7 before v6.1 manual acceptance; v6.1 acceptance remains
deferred and the historical decision above is retained for provenance.

### V7 RESUME PLAN (work paused 2026-10-03 evening; pick up Saturday 2026-10-10)

Rules that apply when resuming (from this effort):
- Do NOT run other CPU-heavy work (agent lanes, builds, tests) while a timing run is in progress. This broke the first fair v6/v7 comparison. Timing runs need an idle machine (close Chrome and other sessions' test runs too).
- Core library only; examples are migrated later. Every fix goes into `CHANGELOG.md` (v7 development log) in the same step as `todos.md`, with a test name.
- Long runs must log progress to a tailable file (the bench suite does: `<out>.log`, heartbeat every minute). Agent lanes write step logs under the scratchpad `logs\` folder.
- `go env -w GOARCH=amd64` is set on this arm64 host: pass `-goarch arm64` to the suite, set `GOOS=js GOARCH=wasm` explicitly for wasm commands. The child `go` of `gwc` gets a scrubbed environment.
- Lanes sometimes drop zero-byte `*.go` files and build outputs (`counter`) at the repo root; check `git status` for strays before committing.
- `.gitignore` has `*.txt`: new `api_baseline.txt` files need `git add -f`.

State snapshot:
- Main checkout `C:\Users\mreca\Desktop\GoWebComponents`, branch `v7/dev`: commits `e280dd23` (core wave 1), `dfa8a996` (examples mechanical), `71cea4bb` (harness fixes + complete changelog), `bb6808cd` (iterative sibling walks, empty class/style, bench -tags/-packages/-skip), plus the WIP commit made when work was paused. Nothing pushed.
- Interop experiment worktree `C:\Users\mreca\Desktop\GoWebComponents-interop`, branch `v7/interop-exp` (WIP commit; based on the v6 base commit plus a copy of the v7/dev working tree at that time, so it does NOT contain `bb6808cd`'s two core fixes except the lane's own copies; merge or rebase before comparing). Experiment-only paths: `internal/platform/jsdom/buf_*.go`, `internal/platform/jsdom/applier/`, `internal/platform/jsdom/testdata/workerapp/`, `research-interop/`, `test/interopbench/`, `internal/runtime/*gwcbuf*.go`, `internal/runtime/node_release_*.go`, `internal/runtime/reconciler_template_slab_*.go`; shared files it edited: `ui/ui.go`, `router/router_api.go`, `ui/dom_ref_wasm.go`, `internal/runtime/events.go` and `events_pointer.go` (tag line only), `internal/runtime/shim.go`.
- Local artifacts (git-ignored, under `bin\`): `bin\v6-ref` (pristine export of `1e61695c`, benchmark loops converted to `b.Loop`, `.gwc-source-sha` file), `bin\gwc-suite.exe` and `bin\gowasmexec.exe` (rebuild: `go build -o bin\gwc-suite.exe ./tools/gwc`, `go build -o bin\gowasmexec.exe ./tools/gowasmexec`), `bin\baseline\` (v6/v7 native arm64 runs with dev tags), `bin\fair\` (production-tag runs, contaminated by concurrent lanes; `run-prod.sh` has the exact commands), `bin\v6-baseline-run2-evidence\` (first full v6 run incl. wasm lane), worktree `bin\interopbench\` (`measure.log`, `measure.console.txt`).
- Reader document (private artifact): https://claude.ai/artifact/Ry9xonBJpDNmRyDPN6qaQH ("Wasm Interop Rethink").

Results so far to remember:
- Screening v6 vs v7 native, production tag (172 entries, both sides contended, NOT a verdict): geomean -5.4%, 33 improved, 17 regressed, 122 noise; v7 `devtools` and `head` targets failed from environment ("cannot allocate memory"; devtools passes alone). Candidate regressions to confirm on a clean run: `GoUseMemoSameDeps` +288%, `GoUseEffectSameDeps` +26% (is the new hook-dep diagnostic really compiled out under `-tags production`? check `internal/runtime/hook_deps.go`, `hook_order.go`, `hooks.go`), `FiberSiblingWalk256` +106% and `FiberHotFieldScan256` +74% (did the new `Fiber.slotGap` field grow the struct? compare `unsafe.Sizeof(Fiber{})` and the layout benchmark), `RenderSteadyState` +85%, `DivWithTextChildren` +81%, `TransitionListRefresh250` +76%, `ui` SSR bootstrap marshal/reference-script +28..+61%, `FineGrainedSignalStyleDashboardReactiveRegions16` +168%. Biggest improvements: html builders (`MergePropsHotPath` -86%, `DivNodeConstructionMicro` -66%, `VirtualListBuild/items-1000` -53%), `AtomRegistrySubscribeUnsubscribe` -68%, projection `ApplyUpdateBatch` -51..-67%, async-dashboard scenario -59%.
- Interop experiment, first real browser measurement (headless Edge 154 arm64, 10 reps after 2 warm-ups, medians; machine idle; run aborted at create N=10000): create 100 rows: legacy 2.29 ms, buf 3.40, tpl 3.64, worker 9.22; create 1,000: 66.1 / 70.8 / 73.2 / 77.5; create 10,000: 1,107 / ~1,146 / ~1,150 / ~925 (1-2 samples); update-one-text (200x via scheduler): 2,230 / 2,183 / 2,197 / 1,738 ms; update-one-sync (1,000 flushed updates, 100 rows): 1,410 / 1,378 / 1,376 / 8,471 ms (worker 6x slower). Go mallocs per row equal (~29-30). Conclusion so far: NOT faster than the status quo; the boundary seems to be a small share of the total. Wasm size: buffered +441 KB raw (+6.4%), +96 KB gzip, +62 KB brotli; templates +147 KB raw more.

Resume steps, in order:
- [ ] **[impl]** 0. Sanity: `git log --oneline -6`, `git worktree list`, rebuild the two binaries above, `go build ./...`, `go vet ./...`, check no stray root files, confirm the machine is idle.
- [ ] **[research]** 1. Finish the interop verdict (profile lane was stopped before reporting). Tasks: (a) what does the harness timer include (style/layout/paint? `test/interopbench/page/driver.js`, `app/main.go`); (b) phase table per variant for create N=1000/10000: Go reconcile/render, encode, flush crossing, applier apply, forced layout read (`document.body.offsetHeight`) timed separately, behind a `gwcbufprofile` build tag; (c) Chrome CPU profile (CDP) for legacy and buf at N=10000: Go compute vs crossings vs applier DOM vs style/layout/paint vs wasm GC; (d) why buf loses at N=100 (3.40 vs 2.29 ms): per-frame fixed costs (template detection below a minimum, optimizer pass, view re-creation, TextDecoder, microtask flush); (e) worker: fix `Unexpected non-whitespace character after JSON at position 42` + `commit timeout in create` at N=10000, and explain the 6x slowdown on `update-one-sync`; (f) fix the integration lane's own test bug (`TestBufWasmWideSiblingListsSurvive10kAnd20k` counts `<li>` in a shared container) and run the `gwctemplates` and `gwcworker` test combinations that never ran; (g) re-run `go run -tags playwrightgo ./test/interopbench -worker -isolated -Measure` (about 15 minutes, must finish without error) and write the verdict: ship parts, keep as an option, or drop. If the boundary share is small, drop the heavy parts and keep only cheap adapter fixes (one JS helper for `GetValue`/`GetKey`, avoid object `Get` in `SetTextContent`).
- [ ] **[impl]** 2. Clean v6 vs v7 comparison (about 65 minutes, nothing else running): `bash bin/fair/run-prod.sh` (16 core packages, `-tags production`, `-goarch arm64`, count 5, warm-up 1, seed 6001, `-skip '^BenchmarkHydrateMediumTreeFirstUpdate$'`; v6 from `bin\v6-ref`, v7 from the main tree; compare is written to `bin/fair/compare-prod.txt`). Then investigate each confirmed regression from the list above and fix before optimizing anything. Optional dev-tag pair (same command without `-tags production`).
- [ ] **[impl]** 3. Wasm lanes for both trees with the fixed harness (v6 `internal/runtime` and `internal/platform/jsdom` wasm targets failed or timed out in the first run), merged with `gwc bench suite merge`; freeze the accepted baseline under `docs/benchmarks/` and refresh the `bench-drift` workflow.
- [ ] **[impl]** 4. Performance work packages (see V7-P3 profile findings): WP1 SSR goroutine-id removal first (native SSR spends 69-95% of its CPU in `runtime.Stack`), then css fold key, fiber/props allocation, html presence mask, typed component calls, dev-guard cost, wasm boundary batching, commit slicing; size items (`GOEXPERIMENT=nojsonv2` in `buildWasmGoEnv`, CBOR opt-in, gate panic/diagnostics code out of release builds, `serverfn` off `net/http` on wasm).
- [ ] **[impl]** 5. Remaining API, diagnostics, OTel, quality, examples and release items are listed in this file's V7 sections (P4 design findings, P5/P6 steps, Wave logs, bug-hunt candidates, V7 open questions) and in the CHANGELOG "Known issues and decisions pending" list; pending decisions: `goValueToJS` precision beyond 2^53, `EncryptedStore` key binding, `sqlite.sanitizeName` collisions, `kvstate` table-rename migration, `css.Inject` replace semantics, the `GOARCH=amd64` setting, and whether the interop experiment ships. The module split and `/v7` rename have landed, pending full validation.

2026-10-20 checkpoint: `v7/dev` is still on `/v6`, and no `v6.1.0` tag exists locally or on origin. Sanity gate 0 is partially verified: `go build ./...` and `go vet ./...` passed with go1.27.1, `bin/gwc-suite.exe` and `bin/gowasmexec.exe` were rebuilt, and there are no stray root binaries. Focused tests passed for `graphcanvas`, `tools/uicodegen`, `tools/livereload` `TestClassifyUpdateBranches`, and `tools/gwc` `TestRunDevExecutesForwardedCommandAndSurfacesFailure`. The host was not idle (roughly 33-61% CPU during samples, including another `server.test` run and antivirus), so no timing comparison was started. In `v7/interop-exp`, the wasm wide-list test originally counted prior tests' rows in shared `#app` (5,002 instead of 5,000). Commit `c203ddcc` isolates its container and keeps the effect-child function identity stable across renders. `TestBufWasmWideSiblingListsSurvive10kAnd20k` and the focused `TestBufWasm` wasm suite now pass: 20,000 keyed updates emit 20,000 SET_TEXT ops, no structural ops, and exactly twice the words of 10,000. The `gwctemplates` focused wasm variant also passes. The interop verdict and full browser measurements remain open.

2026-10-20 size checkpoint: `buildWasmGoEnv` now forces `nojsonv2` while retaining unrelated `GOEXPERIMENT` values. `TestBuildWasmGoEnvDisablesJSONV2` failed before the change and passes after it. `go run ./tools/gwc build -app .\\examples\\public\\counter\\main.go -root .\\examples\\public\\counter -profile ci` passed and produced a 5,961,386-byte wasm file. This addresses the measured Go 1.27 size regression for `gwc` builds; direct Go invocations and any other wasm launchers still need an audit. Next suggested todo: finish the interop verdict once the host is quiet.

2026-10-20 interop worker checkpoint: commit `3536572e` clears alignment padding in reused SharedArrayBuffer JSON records before publication; otherwise stale bytes caused `JSON.parse` to reject a valid control message and the ring reader could strand the worker. Focused `gwcbuf gwcworker` native tests pass, and a focused browser run at N=10,000 completed every scenario, including the formerly hanging update-all, with matching DOM semantics. Commit `98c9d716` marks worker RPC execution as discrete, so synchronous action frames do not wait for requestAnimationFrame; a 20-action focused run fell from about 164 ms to 31 ms (buffered main-thread about 23 ms). `TestWorkerRPCCommitsOnDiscreteLane` in commit `899d8356` pins the lane and its restoration. The full measurement remains open; these short runs are diagnostic, not the release benchmark.

2026-10-20 interop phase-instrumentation checkpoint: commit `b3d3bfe1` adds an opt-in `gwcbufprofile` lane, recording raw action-to-commit duration (including scheduler delay), encode time, flush time, applier time, their remainder, and a separate forced layout read. The phase table uses medians for legacy/buf/tpl; legacy has no buffered encode/applier phases. `TestActionCommitRemainder`, native interopbench tests, legacy/buf/tpl wasm builds, and a focused N=1,000 browser run passed with matching DOM semantics. At N=10,000 in an earlier focused run, buf encode was about 12 ms and applier about 47 ms while action-to-commit remainder was about 826 ms; tpl encode about 33 ms, applier about 53 ms, remainder about 836 ms. These are diagnostic figures on a busy host, not the controlled verdict or CPU profile.

2026-10-20 v6.1 prerequisite recheck (isolated clean `v6` worktree): focused desktop and kvstate native tests, desktop wasm tests, six JS tests, Wails counter tests, Wails adapter tests and vet, desktop doctor, and `TestDesktopLabWebE2E` passed. A fresh Windows EXE built, but the actual native WebView smoke timed out before a complete report (first run reached the tester with `api-window-info-result` missing; later attempts also timed out with features disabled). The host was contended by lint/build/browser work, so this is not a proven regression; rerun on an idle host and inspect smoke logs before the v6.1 tag. Historical manual UI gates remain open.

2026-10-20 v6.1 Go 1.26.6 rerun: with isolated `GOCACHE`/`GOTMPDIR`, a rebuilt EXE passed its normal native WebView smoke with all 26 checks and exit 0. The `--features none --file-dialogs=false` variant passed all 20 applicable checks. The missing-binding and missing-wasm fault variants produced the documented visible-boot-error/native-controls-unavailable reports and expected exit 1. The streaming fault variant unexpectedly timed out at `remount native event` after 13 checks; rerun it alone after host contention clears. Logs: `bin/test-results/v6.1-*-go1266*.log` in the clean v6 worktree. Manual native UI gates remain open, so no v6.1 tag yet.

2026-10-20 v7 module-split integration: isolated branch `v7/module-split` was fast-forwarded into `v7/dev` at `6eebe6c4`; the pre-existing working changes were stashed and reapplied without conflicts, and the safety stash remains. `/v7` imports plus nested tools, Playwright, SDK-backed OTel tests, AI wizard and example modules are in the main checkout. Root `go.sum` fell from 213 to 112 lines. A fresh external consumer importing `/v7/ui` builds natively and for js/wasm; its `go.sum` has six lines and no Playwright, Bubble Tea, AI SDK, Chroma or golangci entries. Integrated `go build ./...`, `go vet ./...`, `go test ./graphcanvas`, `go test ./uicodegen` in the tools module, full livereload module tests, and compile-only tools tests pass. Full release lanes and the remaining module commands are open.

Verification addendum: `gwc verify` for the counter app passed its CI wasm build (there are no app-local tests). Full `gwc lint` ran with a locally built Go 1.27-compatible golangci-lint v2.15.0; it reported 642 issues across the current tree (287 a11y, 187 hook-rule, 50 errcheck, 50 staticcheck, 44 unused, 15 deprecation-rule, 9 ineffassign). This gate is not clean and must be triaged before release. The previously installed v1.64.8 linter could not load Go 1.27 code; the replacement binary is ignored under `bin/` and did not change the global installation.

2026-10-20 root rerun: after fixing the `go -C tools` doclint parser and generated-path handling, Atlas compound CSS selectors, and the Atlas server test's missing wasm fixture, `go test -p 2 ./... -count=1` passed across the root module (log: `bin/test-results/v7-root-tests-rerun-20261020.log`). The release workflow's counter `gwc verify -audit` command now uses `..\examples` paths and passed locally. Active README/reference-manual commands were corrected to the tools-module working directory. The isolated v6.1 `--smoke-fault streaming` rerun also passed all 27 checks, so that earlier timeout did not reproduce; its native manual gates remain open. Nested-module, browser, lint, performance, and release validation remain open.

2026-10-20 nested-module checkpoint: `telemetry/otel`, `examples/tests`, `tools/agenthub`, `desktop/wails`, and `examples/desktop/wails-counter` tests pass; `examples/site` builds for js/wasm. The `examples/v5-two-artifact` full suite now passes after its direct wasm builds use the same `nojsonv2` profile as `gwc`; its measured app artifact is 1,617,303 B gzipped, below the 1,800,000 B ratchet. The AI wizard `server/app` full package suite passes after rejecting traversal segments in asset paths. Tools module failures revealed Windows CRLF root detection and stale module-split fixtures; after fixes, `go -C tools test -p 2 ./... -count=1` passes (log: `bin/test-results/v7-tools-tests-rerun-20261020.log`). The release workflow now covers nested modules and accepts RC tags with prerelease publication. The v7 versioning policy records the landed module boundary. Lint, browser lanes, controlled benchmarks, migration notes, and v6.1 manual acceptance remain open.

2026-10-21 validation checkpoint: after scoped hook, errcheck, staticcheck, accessibility and deprecation fixes, `go -C tools run ./gwc lint` reports zero findings (log: `bin/test-results/v7-lint-final4.log`). Full uncached root tests pass after the final Atlas edits (`bin/test-results/v7-root-tests-final-atlas-20261021.log`); the full tools module passed after the lint batch (`bin/test-results/v7-tools-tests-after-lint-20261020.log`). The Playwright main and benchmark smoke suites passed after correcting test-server launches to the nested tools module; `TestExamplesAll` passed in Chromium, Firefox and WebKit (`bin/test-results/v7-browser-*.log`). Release CI and PR browser CI now include that examples suite. Residual gates: complete the v6.1 native manual acceptance and tag, run the other wasm/JS/browser and race lanes, obtain controlled v6/v7 and interop performance verdicts, and prepare the versioned v7 changelog/RC release. The branch has no upstream PR or tag; pre-existing user changes and the safety stash remain.

2026-10-21 wasm checkpoint: counter `gwc verify` passed its CI wasm build. The first full `gwc test -lane wasm -root ..` run found a v7 API migration break in `test/router/router_wasm.go` (`[]RouterOptions` forwarded to `...RouterOption`); the focused wasm router test passed after changing the shim, and the full lane then passed all 28 discovered packages (`bin/test-results/v7-wasm-lane-fixed-20261021.json`). Remaining release validation includes wasm vet, nested module tests not yet rerun after the browser fixes, broader browser/JS coverage, race on a supported host, and the performance and v6.1 gates above.

2026-10-21 draft PR checkpoint: `v7/dev` is pushed to origin and draft PR #61 targets `main`. Example100 browser failures exposed an auth-role ordering bug, missing first-paint pricing/FAQ strings, a billing fixture without a paid plan, and repeated hard navigation before Wasm boot completion. Focused Admin Journey (twice with rebuilt runtime), Chats, Ops, Billing Summary and Pricing First Paint tests pass. A deterministic scheduler regression first failed because a state write between `currentRoot` publication and clearing `updateScheduled` was coalesced without a follow-up; it passes after marking that commit window as owed work. The grouped Example100 sweep with the old runtime artifact still failed intermittently on the 0% auth shell, so the full suite must be rerun with the rebuilt artifact before this browser todo is checked. PR CI also found separate API-guard/agent cwd errors, a Linux/Wasm desktop fixture mismatch, incompatible lint/security analyzers, a reachable `x/text` advisory, and a real fetch cache race. No merge, tag or deployment is ready.

2026-10-21 PR follow-up checkpoint: PR CI fixes now cover the API/agent working-directory errors, desktop platform fixture, fetch test cleanup race, bridge analyzer toolchain, and Windows Wasm runner environment limit; the dependency graph uses fixed `x/net` and `x/text` versions. Runtime regressions cover both commit scheduling windows and resolve late component-owned updates onto the in-flight tree. After rebuilding Example100 Wasm with that fix, a stable focused Admin Ops browser run passed (`bin/test-results/v7-example100-ops-stable-wip-20261021.log`). The broader Example100 browser sweep and fresh PR checks remain open. A Windows test run also hit temporary executable cleanup locks; those failures are separate from browser assertions. No merge, tag or deployment is ready.

2026-10-21 grouped browser/CI checkpoint: all eight previously affected Example100 browser cases passed together with the rebuilt Wasm client (`bin/test-results/v7-example100-affected-after-wip-20261021.log`). Fresh PR checks then exposed nested-module metadata drift after the dependency update, which has been tidied for desktop, tools/livereload, and agenthub. The GoGRPCBridge standalone security lane still needs fixed dependencies and a patched Go toolchain; compatibility-smoke still reports incomplete child-process I/O after browser assertions pass. Full browser and release gates remain open.

2026-10-21 bridge/cache checkpoint: `go clean -cache` reclaimed about 158 GB; one locked cache entry denied removal. Draft bridge PR #8 rebases cleanly onto bridge `main`, updates reachable dependencies and Go 1.26 CI, and is pinned by parent draft PR #61. The parent CI fixes nested Example 100 prebuild commands and Unix Playwright child-process cleanup. Against the rebased bridge, seven affected Example 100 browser cases passed in a grouped run; the chats case initially failed because a seed-command failure silently copied an incompatible historical DB fixture, then passed on a focused rerun after the seed helper was corrected. The agent bridge dogfood browser case passed. The full Example 100 native suite passed after its snapshot cache test began waiting for the asynchronous persistence that it asserts (`bin/test-results/v7-example100-native-bridge-rebased-fixed.log`; focused test passed 100 repeats). Both PRs' refreshed checks remain pending; full browser and release gates, v6.1 manual acceptance/tag, benchmarks, and v7 RC work remain open. No merge, tag, or deployment is ready.

2026-10-21 CI follow-up: bridge PR #8's full test job and parent PR #61's WASM lane both found the nested `examples/wasm-client` module's stale indirect requirements. The bridge module metadata was updated and read-only WASM builds passed under Go 1.26.3 and 1.27.3. Parent Unit Lane exposed a keepalive test that waited on a WebSocket read deadline that HTTP/2 could replace; an independent pong watchdog and a deadline-reset regression passed 20 focused repeats and the bridge short package suite. Parent compatibility-smoke reached `TestExamplesAll` without generated public example and Atlas WASM assets, and Ubuntu lacked WebKit system libraries. Its workflow now builds those assets and installs the Playwright dependencies; the release workflow builds public example assets before browser tests. Fresh CI validation is pending. Bridge dependency PR must land before the parent PR; v6.1 manual acceptance/tag, final release gates, benchmarks, CashFlux acceptance, and v7 RC remain open.

2026-10-21 root gate follow-up: parent PR #61's refreshed bridge Unit and WASM lanes passed. The separate root WASM test job found that `TestProductionCorrectness_ComposedAppFlowWithBoundaryPortalAndHydration` drained only one scheduled task under the default 5 ms wall-clock slice budget; `FrameBudgetMs: -1` made the composed hydration test pass 20 WASM repeats and the full WASM runtime package. Root Integration Smoke queried the bridge module from the root module, where it is not required; it now checks from the nested Example 100 consumer module. Both changes need refreshed CI validation.

2026-10-21 PR checkpoint: bridge PR #8 passes all checks (full test, WASM, lint, security). Parent PR #61 passes all current checks except Windows WebView: its three positive desktop smokes and two expected startup failures behaved correctly, but PowerShell returned the final expected exit code 1 at the step boundary. The workflow now exits 0 after asserting both expected failures; refreshed CI is pending. Merge still waits on release prerequisites and final validation.

2026-10-21 agent dogfood follow-up: parent PR #61's Windows WebView check passed after the exit-code fix. Its only failing check was `TestExample100AgentBridgeDogfood`: the browser showed a sent prompt and assistant reply, but the test searched for prompt text inside a budgeted `bridge.snapshot` tree. The test now verifies the exact user message bubble, checks prompt text through semantic `bridge.query`, and still asserts non-null `bridge.snapshot` roots after send and reload. The focused browser test passed locally; CI rerun is pending.

2026-10-21 release-order decision and WASM follow-up: the maintainer explicitly authorized v7 release before v6.1 while deferring the remaining v6.1 Windows manual acceptance. The earlier v6.1-first plan is superseded for release ordering; v6.1 acceptance and publication remain open on their own track. Parent PR #61's next WASM run found the two remaining strict hydration mismatch tests using one scheduled callback with a 5 ms wall-clock budget. Both now use `FrameBudgetMs: -1`; focused WASM tests passed 20 repeats and the full WASM runtime package passed. Fresh CI is pending. Other v7 gates (fair benchmarks, CashFlux acceptance, RC validation) still apply.

2026-10-21 interop scope decision: the buffered/template/worker experiment remains solely on `v7/interop-exp`; `v7/dev` contains none of its 67 experimental files and the release workflow does not execute its harness. Partial browser measurements showed parity but no reliable end-to-end advantage, with larger WASM artifacts, so the experiment is deferred from v7 and its incomplete 15-minute measurement is future research rather than a release gate. The core v7 fair v6/v7 production benchmark remains required. Existing fair JSONs are invalid because the prior run was contended and partial. The maintainer chose a dedicated CI runner for the rerun; `.github/workflows/fair-v6-v7.yml` now prepares clean sources and runs both versions sequentially on one amd64 runner with count 5, warm-up 1, seed 6001, and the production tag. Its v6 `b.N` normalization emits an audit report and preserves each loop index. The manual run and comparison remain pending.

2026-10-21 PR gate checkpoint: parent PR #61's current `da7ba87e` check suite is green, including root integration, WASM tests, security, lint, compatibility smoke, browser dogfood, Windows WebView, race, and bench drift. Bridge PR #8's test, WASM, lint, and security checks are also green. These PR results do not replace the manual fair benchmark, CashFlux acceptance, or RC release workflow.

2026-10-21 release changelog guard: `changelogcheck` previously accepted a `v7.0.0-rc.1` heading as the `v7.0.0` stable entry because `-` did not extend a version token. The parser now treats prerelease and build metadata suffixes as part of the token; focused tests pass. A dedicated v7 RC heading and docs-site release-note mirror are drafted and pushed; their pending-gate language must be replaced with verified results before the RC tag. A separate stable heading remains required before `v7.0.0`.

2026-10-21 router WASM fixture checkpoint: the docs-only PR rerun exposed `TestRouteLoaderRevalidatesOnQueryChange` timing out after a deferred first-commit panic. CI logged `syscall/js: Value.Call: property dispatchEvent is not a function` before the timeout. The fake document now exposes its existing `parseDispatchEvent` function and releases it during cleanup; the focused test and full router WASM package pass locally. The new PR check run is pending. The fair v6/v7 production benchmark is running on a separate GitHub runner; CashFlux's v5 baseline passes its all-routes smoke while the v7 migration currently times out on a sequential route sweep. Repeated route timings and profiling remain open, so neither the RC nor stable tag is ready.

2026-10-21 CashFlux and dogfood follow-up: the isolated CashFlux `/v7` migration passed its Wasm build, native compile, vet, all-routes smoke, and light-theme smoke. Three repeated v5/v7 route timings did not show a consistent v7 slowdown; the earlier sequential v7 timeouts occurred under local CPU contention. The PR rerun passed 31 checks but failed `TestExample100AgentBridgeDogfood`: a reply can complete before the refreshed conversation list resolves its public route ID, temporarily replacing the visible thread with the starter shell. The harness now waits for the exact prompt and assistant bubble on the thread route before each semantic bridge query, allowing 15 seconds for route reconciliation. The focused browser test passed; refreshed CI remains pending. The dedicated fair v6/v7 benchmark remains in progress, so neither v7 tag is ready.

2026-10-21 post-bridge verification: `TestExamplesAll` passed in Chromium, Firefox, and WebKit (`bin/test-results/v7-examples-all-post-bridge.log`). Counter `gwc verify` passed its CI Wasm build (`bin/test-results/v7-verify-counter-post-bridge.log`), and `gwc lint` with the repo-local Go 1.27-compatible golangci-lint v2.15.0 reported zero issues (`bin/test-results/v7-lint-post-bridge-v2.log`). The globally installed linter is built with Go 1.26 and cannot load the Go 1.27 module; use `-tool ../bin/golangci-lint.exe` from the tools module for local reruns. Bridge PR #8's first security check found only `GO-2026-6809` in the runner's Go 1.26.9 standard library (fixed in 1.26.10 and 1.27.3); both blocking vulnerability workflows now set up Go 1.27.3. Its pinned govulncheck v1.1.4 then panicked under Go 1.27.3; upgrading to v1.9.0 produced `No vulnerabilities found` locally (`bin/test-results/bridge-govuln-go1273-v19.log`). Its quality job failed a large-dataset "memory savings" gate on cumulative `B/op` even on bridge `main` under the same toolchain; that metric remains reported, but the invalid peak-memory pass/fail check was removed while the other performance thresholds remain enforced. Remote PR checks and release acceptance gates remain open.

2026-10-21 dedicated benchmark checkpoint: the fair native CI run matched all 179 entries and measured a -2.13% v7 geomean; diagnostic/report redaction costs and two leaf-commit microbenchmarks are documented in `docs/benchmarks/V7-RELEASE-EVIDENCE.md`. The corrected fair Wasm run matched all 148 entries and reduced the >15% failures to one reproducible `TransitionListRefresh250` regression (+25.4%). A focused iterative-commit helper optimization preserves wide-sibling stack safety and passes native and Wasm commit/portal tests; the dedicated Wasm comparison must be rerun before RC. The PR drift run showed a cluster of late native runtime slowdowns that did not reproduce in the full fair run. Its per-entry threshold now applies only when base and head share a module path; major upgrades retain suite/coverage checks and use the full fair release evidence for acceptance. No RC or stable tag exists yet.

### V7 progress log (2026-10-03, branch v7/dev, nothing committed)

Integrated check after wave 2: `go build ./...`, `go vet ./...`, wasm `go vet` over the core packages, and `go test -p 2` on 44 core packages all green.

- [x] P5 step 6/9 and part of step 10: 11 new codes (`GWC-UI-FORM-FIELD-INVALID`, `GWC-HYDRATION-ISLAND-INVALID`, `GWC-UI-AWAIT-FAILED`, `GWC-RUNTIME-SSR-GOROUTINE-IDENTITY`, `GWC-ROUTER-ROUTE-PATTERN-INVALID`, `GWC-ROUTER-ROUTE-PARAMS-INVALID`, `GWC-RUNTIME-DERIVED-ATOM-INVALID`, `GWC-UI-SSR-PAYLOAD-INVALID`, `GWC-ROUTER-QUERY-PARAM-INVALID`, `GWC-STATE-SNAPSHOT-INVALID`, `GWC-RUNTIME-ASYNC-FALLBACK-PANIC`); all 12 uncoded panics coded; worst `Errorf` sites converted (state.go atom registry/derived atom, `ui/ssr_transfer.go` 17 sites, `router/searchparams_typed.go` now names param and field, `state/state.go` snapshot versions, `async_boundary.go`); panic reports read a leading `[GWC-...]` token; guard test `internal/errcode/guard_test.go` (empty allowlist, stale-entry check, registry text check). Still uncoded: `router/contracts.go` Define/Path returns, `ui/ssr_transfer.go:330,340,350,444`, remaining runtime/fetch/pwa/interop `Errorf` sites (about 350).
- [x] P5 step 7 and 8: component `(file:line)` in error-boundary and hook-order stacks (`internal/runtime/callable_source.go`, diagnostics only, empty in production wasm); dev-only hook-order mismatch detection (`internal/runtime/hook_order.go`, new code `GWC-RUNTIME-HOOK-ORDER`, zero steady-state allocations, one nil pointer field in production). Limits: only `renderFunctionComponent` renders; a component with no hooks on first render has no baseline.
- [x] Go 1.27 closure identity: go1.27.1 does NOT break component identity today (ui and internal/runtime tests identical on both toolchains; probe literals keep distinct names and pointers; `sameFunctionIdentity` compares code and closure data pointers). Added `ui/component_closure_merge_test.go`. Remaining name dependency: `internal/runtime/panic_report.go:465` filters frames by the literal `SetTimeout.func1`; make it robust before relying on it under 1.27.
- [x] P6 steps 6-8: `telemetry/otel` (`NewBridge`, `Install`, `Middleware` with `gwc.ssr.render` server span, `ExportRecorderSpans`, `ExportRUMEvents`, one redaction conversion function), `ui.SetTracePropagator` (native only) and `serverfn.SetTracePropagator`/`SetServerSpanHook` hooks, `serverfn` client injects and handler spans `gwc.serverfn.<name>`; core stays OTel-free (guard green); no go.mod change (tests use an in-memory provider because `otel/sdk` is not a dependency). Not done: native `fetch` client injection (`fetch/fetch_native.go` is a stub), metrics (`otel/metric` promotion), `tracetest`-based tests (needs `otel/sdk` in go.mod). `serverfn/api_baseline.txt` regenerated.

- [x] Wave 3 (2026-10-03, up to 10 concurrent lanes, each with a failing test first unless noted):
  - Islands/leftovers: multi-root hydration (`internal/runtime/hydration_multiroot.go`, `ResolveHydrationRuntime`; first container keeps the global runtime, others get an isolated runtime sharing adapters; `getHookRuntime` fixes state hooks binding to the global runtime in second islands and `RenderDetached`); `UsePersistedState` re-reads on key change; `MustSetField`/`SetField` deferred-update bug; `Reset` vs `validateSeq` collision (process-wide counter); realtime `MinStableDuration` (default 1 s; negative restores old reset-on-open). Limits: islands have separate atom registries, isolated runtimes are never torn down (no unmount API), island event dispatch verified by reading only. Flaky under wasm (also with the new lookup disabled): `TestHydrationRoundTripFuzz`, `TestHydrateEffectStateUpdatesScheduleAfterCommit` and one more hydration test; investigate.
  - Render packages (20 defects): sanitize kept interior spaces and now checks `poster/cite/background/ping/data/longdesc` and per-candidate `srcset` (19 XSS-vector table); markdown email autolinks get `mailto:`; typed-nil nodes/Stringers no longer kept or panicking in `html`; `PropsOf(Value(""), Value("x"))` and `MergeProps` empty-value ordering; `RawHTMLUnsafe` style split on `;` inside `url(data:...)`; virtual list NaN/Inf/overscan overflow and scrolling past the end (`html/virtual.go`, `virtualization/`), fractional row heights (`pxExact`); css `-0` vs `0` minted two classes, `CubicBezier`/`Repeat`/`GridSpan` clamps, `DynamicVar`/`El` name injection; `css/u` `MtN` negative index; head layer merge duplicates and missing twitter tags in client `UseHead`; anim NaN dt/easing, stiff-spring divergence (substepping only when dt*rate > 1.9), zero-value Spring, exact `Interpolate(…,1,…)`, nil easing, zero-duration transitions. Still open (assigned to the SSR/css hardening lane): SSR `serializeStyleMap` writes keys verbatim (virtualization emits `overflowY:` in SSR) and does not sanitize style values; `css.Inject` re-inject false collision; `u.Sm..Xl2` always-on when the theme lacks the breakpoint; `UseHead` never blanks tags; `GridLineAt(0)`.
  - Server/storage packages (15 bugs): `db/offthread/server` hang on exec without tx id while a tx is open and `Begin` using a cancellable request context; `serverfn` `statusForError` accepting 200/out-of-range codes and an unlimited remote error body (now 1 MiB); `agentbridge` concurrent socket writes and backlog ordering (write mutex), `javascript:` filter bypasses (tab, control chars, `formaction/action/xlink:href/poster/srcset/background`), mount id leaked on factory panic, `timeoutMs` Duration overflow, audit summary length cap; `kvstate.BindAtom` pre-hydration `Set` loss; `projection` duplicate insert and backoff overflow/retry after ctx expiry; `escalate.ClassifySQL` misclassification (`WITH ... DELETE`, tab after `INSERT`, `PRAGMA x=y`, multi-statement); `validate` string rules skipping `*string`, `oneof` on pointers, NaN in min/max; `desktop.Execute` panic containment; `db/sqlite/persist_wasm.go` returned an empty DB on IndexedDB error and then overwrote the stored image (compile-checked only). Deleted the stray ignored 3.8 MB wasm file `backgroundworker` at the repo root (dated 2026-09-08, untracked and gitignored).
  - Error codes: `internal/errcode` now 72 entries (interop 14 `GWC-INTEROP-*`, devtools, css, fetch, router, state, runtime, ui families); `interop.Error.GWCCode()`; about 55 more `Errorf` sites converted; ratchet `TestNoNewUncodedErrors` (allowlist: fetch 36, interop 158, router 21, runtime 38, serverfn 5, state 0, ui 4; only lowers); `llms.txt`/`llms-full.txt` regenerated with `go run ./tools/gwc llms`; `ErrCircuitOpen` is now an `*errcode.Error` (`errors.Is` still works).
  - OTel metrics: `telemetry/otel` metrics (`Options.MeterProvider`, SSR/serverfn histograms, panic and hydration-mismatch counters, goroutine/latency/heap gauges, `BoundedSet` cap 200), `internal/diagnostics.SetPanicObserver/NotifyPanic` wired from `internal/runtime/panic_report.go` (`emitWrappedPanicReport`), `BatchExporter` with retry/backoff/`Retry-After`, bounded `EventQueue`, `pagehide` flush, `gwcattrs.ParseErrorCode/NormalizeErrorAttrs`; `go.mod` gained direct requires `otel/metric`, `otel/sdk`, `otel/sdk/metric` v1.43.0 (go.sum unchanged; moves with the module split). Caveats: callers must use `Bridge.ObserveSSR` for the hydration counter; the serverfn histogram is bounded by the cap, not by registered function names.
  - Go 1.27: `go.mod` is now `go 1.27.0` / `toolchain go1.27.1` and `go version` in the repo is go1.27.1 (lane l7 still verifying; see its report).
  - Design (API lane l8): see "V7-P4 design findings".
  - Tooling: progress logging, `-target-timeout`, `-resume`, partial reports in `gwc bench suite`; stray `.exe` and `backgroundworker` removed.

- [x] Wave 4 (2026-10-03):
  - Go 1.27 bump: root and `tools/livereload`, `tools/agenthub`, `desktop/wails` `go.mod` are `go 1.27.0` (+ `toolchain go1.27.1`); all workflows `1.27.x` (`desktop-windows.yml` 1.27.1); README/CONTRIBUTING/VERSIONING updated; `strings.CutLast` applied at 9 sites; `go mod tidy` run (root go.sum gained `go.uber.org/goleak`, nested modules goldmark 1.7.17 and x/net 0.55.0); no 1.26-vs-1.27 test differences (flate, unicode, gofmt alignment, closure names). Not done: `CutLast` at `internal/runtime/signature.go:148,151`, `panic_report.go:338,383,402`, `ui/component_handle_shared.go:253,256`; `google/uuid` still required (only examples, third_party use it; no core code does). NOTE: the `gofmt` on PATH can be older than the toolchain and rejects generic methods; use the go1.27 `gofmt`. Left alone on purpose: `third_party/GoGRPCBridge` (go 1.25.0), examples' go.mod files, `tools/gwc/testdata` go.mod files.
  - Wasm tests executed for the first time (via `bin/gowasmexec.exe`): PASS for jsdom, state, internal/runtime, interop, utils, testkit/*, devtools, pwa, hotreload, db/sqlite, css, desktop, logging, validate, virtualization, test/{browser,hooks,render,ssr}. Fixed: `BenchmarkHydrateMediumTreeFirstUpdate` (benchmark assumed one scheduler slice; wasm yields after the 5 ms budget, now drains continuations), jsdom benchmark mock nodes lacked `setAttribute/removeAttribute/listeners`, `state` wasm tests lacked `Reset: true` (atom leakage between runs), fetch upload test now ignores the injected `traceparent`, router wasm tests leaked subscriptions. `internal/apidump.Check` now skips under `js` (`os.ReadDir` fails with `O_DIRECTORY is not supported` on Windows under node). Open: `ui.TestComponentIdentitySameFunctionValueReusesOneCachedHandle` fails on the second repeat (`-count=2`) under wasm; fetch wasm tests print "call to released function" noise (a `js.Func` released while still in use).
  - API: typed `OnInput`/`OnChange` (`InputFn = ~func() | ~func(string) | ~func(ui.Event) | ui.Handler`, wrong signatures are compile errors; `OnInputHandler`/`OnChangeHandler`; `OnClick/OnSubmit/OnKeyDown` stay `any`), 46-48 generated attribute builders (`html/internal/attrtable` + `attrgen`, `go generate`; `Slot` is `SlotAttr` because `shorthand.Slot` exists; `Transform` omitted because examples declare it), `CheckedIf/RequiredIf/IfProp/UnlessProp`, `Markdown(src, ...MarkdownOption)` + `WithURLSchemes` (an explicit empty list now allows NO schemes; default http, https, mailto), `defer runtime.BeginListRender()()` in `Map`/`MapKeyed`. Breaking: untyped/nil/wrong-signature `OnInput`/`OnChange`; spreading `[]MarkdownRenderOptions` into `Markdown`; new dot-import names (`D`, `R`, `Fill`, `Stroke`, `Points`, `Part`, `Scope`...) can collide with package-level declarations in dot-importing code.
  - Router: `UseParams/UseQuery/UseSearchParams/UseRouteData` subscribe like `UseRoute` (not inside route factories or outside render); `WithBasePath` (history routers), native `SPAFallbackHandler`; `RouterOption`/`RouteOption` interfaces satisfied by the old structs plus `With*` options; `Params.As[V]`, `Query.Decode[T]` generic methods. No native `UseParams` etc. exist.
  - Hooks diagnostics: `GWC-RUNTIME-HOOK-DEP-UNSTABLE` (dev only; fires only when a dep changed identity: new closure, or map/slice DeepEqual but different identity; `fastEqual` compares func/map/slice by identity), `ui.Always` sentinel (sole dep only), `GWC-RUNTIME-HOOK-IN-LIST` (dev only, `runtime.BeginListRender`), `gwc lint` / `hookcheck` flag inline `On*(func...)` in loops and `Map` callbacks.
  - Agentbridge/kvstate/projection/agentui follow-ups: redaction now applied to console/window-error capture and `set-atom` dry-run (`runtime.AgentRedactionActive`); wasm bridge uses `wss` on https pages, override only via `__GWC_AGENT_BRIDGE.url`; `kvstate.sanitizeIdent` is injective (hash suffix when a character was replaced). MIGRATION NOTE: names that previously contained replaced characters (for example `a-b`) now map to a different table; needs a changelog note and possibly a one-time rename. `123abc` vs `_123abc` still collide. `UsePersistedState` uses the park-then-persist pattern (wiring wasm-only, compile-checked); `projection` failed `Post` is now a transport failure (optimistic state rolls back; replaced the test that pinned the old behavior); `agentui` class tokens scoped to an `agentui-` prefix.
  - SSR/css hardening: `serializeStyleMap` kebab-cases keys and drops declarations whose value could break out (top-level `;`, braces, `/*`, `</style`, unbalanced quotes/parens); `css.Inject` first-wins on a repeated id (existing test pins it; a different body is ignored silently); `u.Sm..Xl2` no-op when the theme lacks the breakpoint; `UseHead` removes stale tags (removal is browser-only, no mock-DOM test); `GridLineAt(0)` clamps to 1. Open: a raw string `style=` prop is only attribute-escaped; client DOM style path not audited.
  - Platform bug hunt (25 defects): prerender Windows path traversal, flags 2^32 divide-by-zero and stale-refresh re-enabling killed flags, events cancelled subscriber still delivered, localfirst duplicate-op resurrection and clock saturation, delta `Apply` rollback, logging typed-nil and depth-limit secret leak, telemetryredaction dotted keys and JSON-string secrets, plugin Setup panic rollback and duplicate IDs, pluginruntime teardown, i18n (CLDR plurals for fractions, open-redirect via `//evil.com`, LazyBundle once, zh-Hant-TW, direction by script, relative time, format 1e19), pwa canary bucket bias (50% route reached 58.7% of clients; cohorts reshuffle once on deploy: release note) and update-prompt retry and statechange listener leak. Open (assigned to the platform follow-up lane): locale-aware `FormatDate`/`FormatList`, bootstrap key with dotted namespace, `ResolvePath` without supported locales, plugin `Register` inside `Setup` deadlock, `awaitServiceWorkerValue` released `js.Func`, `Text.Delete` negative index.
  - Deprecations: added `deprecation.Warn` to `state.UseComputed`; every other listed deprecated symbol already had one (`css/u.BgToken/TextToken` and agentbridge `Build*Envelope` unverified); `//go:fix inline` NOT added because the inliner skips wrappers whose body has a `deprecation.Warn` call before the forward, and `gwc-deprecations` lint requires that Warn (conflict: decide whether the lint should exempt `//go:fix inline` wrappers); added `api_baseline.txt` + test for `ui` (485 lines), `html`, `router`, `fetch` (NOTE: `.gitignore` has `*.txt`, so the new `api_baseline.txt` files must be added with `git add -f`), wired `api_compat_guard check` into `api-baseline.yml`, regenerated `tools/api_compat_guard/api_compatibility_baseline.json` after the intentional router signature change. Deprecation table (dates are bulk-import commit dates, 2026-06-11 / 06-20 / 06-28; most replacements are NOT named in CHANGELOG, so write those entries before removing anything): `state.Select/ExportSnapshot/ImportSnapshot/UseComputed`, `hotreload.IsEnabled`, `utils.{EnableHotReload,IsHotReloadEnabled,InstallHotReloadBridge}`, `css.{Property,Opacity,WhenDisabled}`, `css/u.{BgToken,TextToken}`, `fetch.{ReturnChannel,UseFetch}`, `agentbridge.Build*Envelope` (5), `router.{GoRegisterRoute,GoGetRoute,GetCurrentRouterPath,GetOutlet}`, `WorkerPool.{GetSize,GetQueueLimit}`.
  - Housekeeping: removed three zero-byte untracked files left at the repo root by a lane (`metrics.go`, `middleware.go`, `otel.go`, all created 10:56 and breaking `go build ./...`).

- [x] Wave 5 (2026-10-03):
  - Streamed SSR bug (security/functional, fixed by hand with a failing test first): `writeSSRStreamBoundaryPatchScript` (`internal/runtime/ssr_stream.go`) never wrote the closing `"` of `data-gwc-stream-boundary="..."`, so the streamed replacement `<script>` body was read as attribute text and no streamed `AsyncBoundary` replacement ever executed (old tests only checked for the substring `<script`). Test: `ssr_stream_script_quote_test.go`.
  - Runtime review round 2 (6 fixes): SSR `ErrorBoundary` swallowed suspensions (`ssr.go`, `ssr_stream.go`), hostile `BoundaryIDPrefix` filtered to `[A-Za-z0-9_.-]`, Fragment siblings keep nil slots (`flattenFragments` + `slotGap`), refused derived-atom re-registration no longer unlinks the old atom, refs published on hydrated nodes, ref sink swap on surviving elements (`retargetDOMRef`, hot path: one nil check on the typed lane, two map reads on the map lane, no allocations). Root cause of both flaky wasm hydration tests: the 5 ms wall-clock slice budget (`NewRuntime` default `frameBudgetMs` 5; on slow wasm the first pass yields before commit); tests now pass `FrameBudgetMs: -1` (300 green runs). Not fixed: nested AsyncBoundary inside streamed resolved content bakes its fallback (no nested streaming; design limitation); unmounted still-suspended boundary leaves its watcher goroutine parked (documented). Not reviewed: scheduler lanes/timers, hydration mismatch recovery, `state/` selectors and migrations, keyed-diff key-type changes.
  - `ui` review round 2 (11 fixes): native `UseState[any]` `Update`/`Set(nil)`, native `UseId` format (`gwc-native-N`; ids still differ across repeated SSR renders, documented in `docs/ATLAS_PERF_BASELINE.md` 2.6, needs a per-render counter), `UsePointerEvents`/`UseWheel`/timer triggers/`UseIdle` stale closures and deps, `UseTaskCtx` parent cancel, typeahead modifier chords, focus-trap initial focus, `UseSpring` exact settle, panicking `UseAsyncMutation`/`Submit`/`ValidateAsync`/`UseTask`/`UseLazyNode`/`UseAsyncDefer` bodies no longer wedge the hook. Open: `writeSSRStreamBoundaryStart/End` write `BoundaryIDPrefix` raw into an HTML comment (fixed for the prefix in `nextSSRStreamBoundaryID`; re-check the end/start writers).
  - Interop/devtools/a11y/platform review (about 20 fixes): cookie attribute injection (`Path`/`Domain`/`SameSite` CRLF and `;`), `WorkerPool.Drain` hang, `Decrypt` mis-reporting cancelled contexts as auth-tag mismatch, IndexedDB `SetItem` success before transaction complete, `Worker.Request` hang after `Terminate`, devtools support-bundle redaction gaps (JSON-quoted keys, `Authorization: Basic`, multiple Cookie headers, compound names, URL userinfo and `#access_token=`, more sensitive key names, unredacted route/stack/diagnostic/peer fields), extension payload skipped telemetry redaction, host-extension cleanup double release, dispatcher heartbeat after death, ownership restore at the epoch floor, mockdom `InsertBefore(n, n)` and `ReplaceChild`, gcpacing negative limit, sqlite pragma case, `testkit/render` cancel marking and `Await(nil)`, `testkit/ssr` sidecar choice, `hotreload.SchemaFingerprint` collisions, a11y empty `aria-activedescendant` and extra tab stop. Found, not changed (decisions needed): `goValueToJS` loses int64/uint64 precision beyond 2^53 (making it an error could break nanosecond-timestamp callers); IndexedDB `blocked` open clears `onupgradeneeded`; `Subscription.Cancel` not idempotent; `awaitValue` maps context errors to `CodePromiseRejected` (wasm tests pin it); `EncryptedStore` does not bind the storage key as GCM additional data (fixing breaks existing data); `db/sqlite.sanitizeName` collisions (`a b` vs `a_b`); `Dispatcher` fallback sticks after a later `Start` and `Cancel` drops a queued Start; devtools `UseSnapshot` snapshots on every render and polls every 750 ms while closed; `AlertDialog.OnDismiss` never wired; mockdom `GetChildren` returns text nodes but jsdom only elements, `SetStyle("")` keeps the key in mockdom. Reverted: `WindowChannel` name filter (test/browser uses different per-side names).
  - Size lane results applied: `validate` regexps replaced by hand-written matchers (differential test over a large corpus, 0 allocations; `regexp` gone from the wasm graph); markdown engine is now opt-in (`html.RegisterMarkdownEngine`, new `html/markdown` package, `markdown.Enable()`; without an engine: escaped text in `<div data-gwc-markdown-unavailable>` + dev warning `GWC-UI-MARKDOWN-ENGINE-MISSING`; goldmark gone from the default wasm graph). EXAMPLES that lose markdown until migrated (not edited): `examples/site/pages.go:205,280`, `examples/public-examples-site/markdown.go:70` (+ `main_test.go:409`). Counter wasm measured at 6,920,866 bytes with `-trimpath -ldflags="-s -w"` after these two (no before number taken by that lane).
  - `gwc` production cleanup tolerance: `tools/gwc/go_cleanup_tolerance.go` (`isBenignGoCleanupFailure`, `runLauncherGoCommand`), `Warnings []string` on verify/test summaries, used by `test_lanes.go`, `verify_seed.go`, `examples_managed.go`, `warm.go`, `wasm.go`, `agentic_check.go`, `agentic_toolchain.go`, `lint.go`, `main.go`; `gwc verify` on the counter exits 0 (the counter has no Go tests, so the warning path was covered only by unit tests). `bench` keeps its own classifier.
  - Interop research: published as the reader document "Wasm Interop Rethink" (artifact https://claude.ai/artifact/Ry9xonBJpDNmRyDPN6qaQH). Key numbers (ratios, busy machine, Node + stub DOM): raw `wasmimport` 17 ns vs `syscall/js` string/object `Value` 5.5-6.2 µs (~300x); whole-frame batch 30-140x faster than per-op `syscall/js`; events `js.Func` ~36 µs vs `wasmexport` 9 ns (770 ns with JS writing the payload). Go 1.27.1 accepts `//go:wasmimport` and `//go:wasmexport` in user packages on `GOOS=js`; reactor mode is wasip1 only; WasmGC for Go is an open unplanned issue.
- [ ] **[impl]** Interop rethink, staged (from the document): 0) cheap fixes inside the current adapter (one JS helper for `GetValue`/`GetKey`, avoid object `Get` in `SetTextContent`, coarse helper per element: 3-4x in the prototype); 1) buffered adapter behind a build tag (`gwcbuf`) implementing `DOMAdapter` (create, set attr, set text, append, remove first; fallback to `syscall/js`); 2) integer node ids replace `js.Value` in `WASMDOMNode`; 3) delegated events with one exported Go entry; 4) hydration ids from walk order; 5) retire the per-node `js.Func` path after Example 201 parity. Also reconcile `template.innerHTML` use in `CreatePreparedElement`/subtree paths with the no-raw-HTML-sink guard (prefer `template.content.cloneNode`).

- [x] Wave 6 (2026-10-03, wasm interop experiment in worktree `C:\Users\mreca\Desktop\GoWebComponents-interop`, branch `v7/interop-exp`, based on the v6 base commit plus a copy of the v7/dev working tree at that time; nothing committed there):
  - Built and tested (node + compiled wasm, no timings yet): buffered adapter `-tags gwcbuf` (`internal/platform/jsdom/buf_ops.go` op contract, `buf_core.go`, `buf_adapter_wasm.go`, `buf_flush_wasm.go`, `applier/gwc_applier.js` about 210 lines; one flush per commit via `BeginAttrUpdateBatch/EndAttrUpdateBatch`; ops: DEFINE_TAG/NAME, CREATE_EL/TEXT, SET_ATTR/PROP/STYLE, REMOVE_ATTR, SET_TEXT, APPEND/INSERT_BEFORE/REPLACE/REMOVE, LISTEN/UNLISTEN, FREE_IDS, ADOPT_ROOT; UTF-16 string blob; Go-allocated node ids). Applier security: no HTML sinks, `on*` and `srcdoc` rejected, URL attributes through the same scheme policy as the Go sanitizer (shared vector table), hostile frames and unknown opcodes ignored. Differential proof: real reconciler -> mockdom vs op buffer -> JS applier, identical serialized DOM after every frame (keyed reorder, 1000 rows + edit, id reuse). Real traces: update one text node = 1 `SET_TEXT`, 48 bytes; 1000 rows = about 5,000 ops, 112 KB. Gaps: hydration, `ReplaceChildren`/`CreatePreparedElement`/subtree paths (fall back to per-node ops), adopted-node id release.
  - Delegated events (`buf_events*.go`, `applier/gwc_events.js`, runtime twins `events_*gwcbuf.go`): one root listener per kind, one `gwc_dispatch` export, payload in a shared buffer; click through a list: 22 `syscall/js` calls -> 1. 56 allowlisted kinds; other handler signatures keep the old `js.Func` path. Semantic differences (11, see the lane report): handlers run at the document (after non-GWC inner listeners), `preventDefault` applied after the handler returns, payload value captured at dispatch, one plain + one passive slot per node/kind. Open requests: `NewDefaultDOMAdapter` should return `WrapBufEvents(NewBufDOMAdapter())`; applier `idOf(el)` O(1) reverse lookup; `func(GoEvent)` handlers still use `reflect.MakeFunc` in `internal/runtime/event_wrap_cell.go` (untagged file).
  - Real-browser harness `test/interopbench` (Edge 154 native arm64, headless, own profile): smoke shows identical DOM and semantics in every scenario; buffered build is 1.4% larger raw (7,124,149 vs 7,021,214 bytes). Real run (idle machine only): `.\test\interopbench\run.ps1 -Measure` (N=100/1000/10000, 10 reps, 2 warmups).
  - Ideas lane (`research-interop/`, deterministic counts from real traces; speed needs a browser): template cloning keeps 11-40% of the bytes (rows1000 111,708 -> 41,814; table1000 597,918 -> 68,466; ops 27,014 -> 26) and cuts Go-side node allocations 8,002 -> 1,010 if the reconciler stops holding a node per inner element; `APPEND_MANY`/`CREATE_EL_FULL` coalescing rows1000 111,708 -> 77,730 bytes; per-frame blob string dedupe is protocol-compatible (table1000 597,918 -> 456,034); redundant-op elimination finds nothing (the reconciler already diffs; one leftover: `class=""` written on fresh elements, a reconciler fix); a cross-frame value dictionary is worse in 7 of 10 cases (DROP); `SET_CLASS` saves little (DROP); varint/UTF-8 codec is 32-39% of raw size, deflate 11.7 KB vs 108.6 KB, useful only for worker/recording/SSR transports. Off-main-thread Go: the real `gwcbuf` wasm runs in a `worker_threads` Worker feeding the shipped applier, three transports (transfer 1.0x bytes, clone 2.0x, SAB ring 1.0x with 0 data postMessages) DOM-identical to inline over 22 frames; a naive `postMessage(view)` would clone about 15 MB per frame (copies all of wasm memory); `preventDefault` must be decided on the main thread from declarative LISTEN flags; layout reads need an Atomics mailbox or async variants. Recorder/replayer with op-table-hash handshake replays a 3-frame session identically (compare serialized DOM, attribute order varies by map iteration). Frame scheduler policy (lanes, read barrier, hidden-tab fallback): 260 commits -> 70 crossings. Memory: `memory.grow` invalidates views (5,000 rows grew memory 6 times in 7 flushes, peak 38 MiB), finalizer-driven id release leaves 700-2,700 removed nodes held, explicit release at unmount leaves 0; `ReleaseID` is not idempotent (double release returns the same id twice). Claim hydration: creation order equals document order, claim stream is 15-24% of mount words, precise mismatch errors. Debug validator: per-frame CRC32 word catches the 41% of corruptions that structural checks miss (1,079 of 1,079).
  - Ranked merge list from the ideas lane: 1) blob dedupe + `APPEND_MANY` + `CREATE_EL_FULL`; 2) template cloning with a reconciler fast path; 3) explicit unmount release with an idempotent `ReleaseID`; 4) worker transport (`transfer` mode, behind a flag); 5) recorder with handshake; 6) lane scheduler; 7) claim hydration; 8) debug validator + CRC32; 9) varint for transports only. Drop: value dictionary, `SET_CLASS`.
- [ ] **[impl]** BUG (core runtime, found by the ideas lane, not fixed): lists with more than about 5,000 siblings overflow the JS stack in `commitWork` recursion (5k works, 10k crashes) in the wasm build. Fix in `internal/runtime/reconciler_commit.go` by iterating siblings instead of recursing per sibling; add a regression test with 10k siblings under wasm.
- [ ] **[impl]** BUG (core runtime reconciler): `class=""` is written on freshly created elements (1000 in the table1000 trace); skip empty-string attribute values on create.

- [x] Core fixes ported to main (2026-10-03): iterative sibling commit/deletion/effect walks (10k and 20k siblings no longer overflow the stack; test crashed at 10k before the fix) and empty `class`/`style` skipped on create; changelog updated (200 fixes).
- [ ] **[impl]** Wide-list scaling: the test DOM adapter (`testDOMAdapter` in `internal/runtime/reconciler_test.go`) scans the parent child slice in `InsertBefore`, `RemoveChild` and `GetNextSibling` (O(n) per op) so wide-list tests are quadratic (2k..16k keyed list update: 49, 144, 714, 1921 ms); index it. Possible production hotspot to check against a real DOM: the greedy repair branch of `applyCommittedChildOrder` does one O(n) memmove per displaced node via `parseFindObservedChildNodeIndex`/`parseRemoveObservedChildNode`/`parseInsertObservedChildNode`; derive moves from fiber order instead of reading the DOM back (see the `buildCommittedChildOrderMatch` comment).

### V7 bug-hunt candidates (adversarial read-only review, 2026-10-03; each needs a failing test first, lanes running)

- [x] **[impl]** Fixed: failed Ready-resource revalidation now stops retrying while the stale error remains (`fetch/cache.go:894-907`); `fetch/cache_effect_loop_native_test.go` covers bounded retries.
- [x] **[impl]** Fixed: `UseRevalidateOnFocus` invalidates and invokes mounted query revalidators (`ui/revalidate_focus_hook.go:39-60`); `ui/revalidate_focus_hook_wasm_test.go:14` covers refetch.
- [x] **[impl]** Fixed: `LoadNext` preserves appended pages and rejects stale results after reload (`fetch/query_cache.go:200-240`); `fetch/cache_effect_loop_native_test.go:87` covers mounted pagination.
- [x] **[impl]** Fixed: effect cleanup panics are contained and sibling effects continue (`internal/runtime/p7_review_regressions_test.go:13`, `panic_diagnostic_test.go`).
- [ ] **[impl]** `ui/hydration_island_wasm.go:62`: multiple hydration islands share the single runtime root and clobber each other (may need runtime multi-root support).
- [ ] **[impl]** `fetch/resilience.go:67,68,186,214`: zero-value traps (`Multiplier`, `MaxDelay`, `HalfOpenMaxCalls`, `FailureThreshold`) plus a half-open breaker stuck forever after a panicking op or ctx cancel, and retries that keep hitting a downstream after the breaker opens. Decision: zero means documented default.
- [ ] **[impl]** `fetch/realtime.go:700-705,514-525,520,587`: false heartbeat timeout right after reconnect (stale `LastMessage`), `ReconnectAttempts` reset on every open so it never gives up, state outlives controller.
- [ ] **[impl]** `fetch/mutation_queue.go:398-400,329`: conflict-handler error leaves already-succeeded mutations persisted (they replay again); `Clear` takes no lock.
- [x] **[impl]** Fixed: nil children retain slot alignment through `Fiber.slotGap` (`internal/runtime/reconciler.go:845+`, `reconciler_elements.go:939+`); coverage tests pin nil-slot preservation.
- [ ] **[impl]** `query/query.go:310,351,369,455` and `fetch/query_cache.go:330-374`: overlapping optimistic updates roll back to the wrong state; `Invalidate` does not bump the generation so an in-flight fetch commits stale data as fresh.
- [x] **[impl]** Fixed: `RestoreSnapshot` returns subscribed fibers and `RestoreAtomSnapshot` schedules them (`internal/runtime/state.go:579-593,925-933`); `internal/runtime/p7_review_regressions_test.go:167` covers derived dependents.
- [x] **[impl]** Fixed: a second panic in one boundary commit is handled (`internal/runtime/p7_review_regressions_test.go:71`, `TestP7SecondPanicInSameBoundaryIsHandled`).
- [x] **[impl]** Fixed: `GoUseEffectOf` leaves cleanup to commit when a render is discarded (`internal/runtime/p7_review_regressions_test.go:98`, `TestP7TypedEffectCleanupNotRunDuringDiscardedRender`).
- [ ] **[impl]** `internal/runtime/hooks_fetch.go:84-209`: deprecated `GoUseFetch` has no sequence guard (older slow response overwrites a newer one).
- [x] **[impl]** Fixed: `Render`/`Hydrate` flush deferred passive effects before replacing pass bookkeeping (`internal/runtime/p7_review_regressions_test.go:137`, `TestP7RenderFlushesDeferredPassiveEffects`).
- [x] **[impl]** Fixed: a disposed key's late loader cannot clear `Loading` on a recreated entry (`fetch/cache_effect_loop_native_test.go:136`, `TestFetchNativeDisposedKeyLateLoaderDoesNotTouchNewEntry`).
- [x] **[impl]** Fixed: empty text during hydration does not consume the next real node (`internal/runtime/p7_review_regressions_test.go:230`, `TestP7HydrationEmptyTextChildDoesNotConsumeNextNode`).
- [ ] **[impl]** Lower confidence: `ui/form.go:229-256,648` deferred `SetField` misreports and `Reset` reuses `validateSeq`; `ui/persisted_state.go:84-86` value not re-read on key change; `state/autotrack.go:42-56` tracking state leaks on panic; `ui/timers.go` and `ui/global_events.go` keep the first render's closure; `fetch/realtime_wasm.go:122` EventSource reconnect drops `Last-Event-ID`; `fetch/cache_persistent.go:253` restore can overwrite a newer `Set`.

### V7-P0 — Baseline and branch

- [ ] **[impl]** Finish and tag v6.1.0 (the two open items in the v6.1 section below); v7 branches from that tag.
- [ ] **[impl]** Uncached full run (root, nested modules, wasm, JS, browser) on the tag; record every failure as the v7 bug-hunt seed list (v6.0.0 known limitations: chat-example admin/sign-in, settings routing, billing summary, boot/isolation; Atlas and Example 100 failures; relocated benchmark harness).
- [x] **[research]** DONE 2026-10-03: go1.27.1 installed at `C:\Users\mreca\sdk\go1.27.1` (wrapper `go1.27.1`; system go stays 1.26.3). On 27 core packages, 1.27.1 vs 1.26.3: vet, build, compile-only tests and unit tests (ui, internal/runtime, css, state, router, fetch) all identical, zero new findings (no stdversion, printf `%w`, GODEBUG, closure-name, flate or unicode issues); wasm builds fine; the only wasm failure was the `telemetry` bug fixed above. NOTE: `%APPDATA%\go\env` sets `GOARCH=amd64`, so `go install` of a wrapper builds amd64 on this arm64 host; override with `GOARCH=arm64`. Original item: Install `go1.27.1` (`golang.org/dl`), confirm `GOOS=js GOARCH=wasm` builds and the wasm test lane runs on windows/arm64; the 1.27 notes list no js/wasm changes, so record any difference.
- [x] **[research]** Benchmark inventory and gap audit of what v6 has today: 493 `Benchmark*` funcs in 129 files; `gwc bench` plus `docs/benchmarks/{latest,reference,v5-baseline,v5-measured,run-arm64,build-times}.json`; `docs/benchmarks/MICROBENCH_REPORT_arm64.md`; `docs/ATLAS_PERF_BASELINE.md`; browser macro in `examples/testing/render-benchmark` (Example 201 vs React) and `test/benchmark`; `examples/testing/{atlas-perf,use-state-rerender-trace,v5-load-harness}`. Write down which layers have no benchmark (candidates: hydration end to end, router navigation, `fetch` cache/query, list virtualization, devtools overhead, memory/heap over time, long-frame count, battery/idle). Done 2026-10-03 (audit): 493 `Benchmark*` in 143 files, about 175 third-party; `gwc bench` skips `test/render`, `atlas-perf`, `render-benchmark/shared` (now reachable with `-include`); bench-drift compares an ubuntu run against a windows/amd64 March baseline at count 1; missing layers: streaming-SSR TTFB, heap/goroutine series, devtools on/off, machine facts.
- [ ] **[impl]** Build one comprehensive **v6 baseline suite**, run against the v6.1.0 tag (the v7 branch point), covering every layer, each with a stated metric and noise budget:
  1. Native micro: all 493 benchmarks via `gwc bench` with `-count` for `benchstat` (ns/op, B/op, allocs/op); `b.Loop` conversion first.
  2. Wasm micro: the same set under `GOOS=js GOARCH=wasm` through `go_js_wasm_exec` (wasm runs about 4.5-6.5x slower than native, so wasm-specific regressions need their own numbers).
  3. Browser macro: Example 201 (19 scenarios) vs React 19.2.4, DOM-ready and interaction timings, geomean plus per-scenario ratios; same-window phase probe protocol; at least N runs with median and spread.
  4. SSR/hydration: Atlas native SSR (time, bytes, allocs per element, CPU profile split), streaming SSR time-to-first-byte, `HydrateMediumTree`, browser hydration time.
  5. Boot and size: raw/gzip/brotli size for the counter, a mid app and Atlas; cold boot phase breakdown (compile, instantiate, init, first render); warm boot; `gwc wasm measure`.
  6. Runtime health: long-frame count (budget 0), GC pause (budget 3 ms), peak and steady heap, goroutine count over a long-running session, frame budget behavior under load.
  7. App-level flows: router navigation, `fetch` query/cache/invalidate, virtualized list scroll, form input latency, devtools on vs off.
  8. Build-time: `gwc build`/`dev` cold and warm (`build-times.json`).
- [ ] **[impl]** Make the suite reproducible: pin machine facts (CPU, OS, Go version, browser version, power plan/thermals; this machine is fanless windows/arm64 with ±25-30% noise), warm-up and cooldown rules, run order randomization, and a single command (`gwc bench suite` or equivalent) that emits one versioned JSON plus a human report (`docs/benchmarks/v6-baseline.json`, `V6_BASELINE.md`); record the Go toolchain used so toolchain and code effects can be separated later.
- [x] **[impl]** Done 2026-10-03 (scaffold): `gwc bench suite` and `gwc bench suite compare` (`tools/gwc/bench_suite*.go`, schema `gwc.bench.suite/v1`; lanes native, wasm, size, build; browser/health/refs written as `not-run`/empty placeholders; MAD-based significance, refuses machine mismatch). Caveats: the `build` lane's `coldMs` is not a true cold build (Go cache not cleared); wasm lane warm-up only when `-warmup` is passed; the real size/build/wasm lanes are covered by stubs only; top-level `gwc -h` still says `[capture|compare]`. Still to do: browser lanes (Example 201, Atlas, v5 load harness), heap/goroutine series, TTFB, devtools on/off.
- [ ] **[impl]** Add a compare mode: `gwc bench compare v6-baseline.json <candidate>` produces per-benchmark deltas with significance (`benchstat`) and a pass/fail against the V7 exit gates (V7-P3 gate, size budgets, GC/long-frame budgets); wire it into the `bench-drift` workflow so every V7 PR reports against the v6 baseline. Also run it on a second machine class (the old baseline was amd64) so numbers are not tied to one device.
- [x] **[research]** Run 1 (pilot, 2026-10-03, native arm64, 45 min, count 5, seed 6001, 38 package targets, `bin/v6-baseline-arm64-run1.json`): NOT a valid baseline. The tree was edited by parallel lanes during the run, so native `./internal/runtime` and `./ui` and wasm `./router` failed to compile mid-run (`undefined: errcode`), and the machine was also running other work (a separate SchemaFlux test session, lane test runs). Noise: median MAD/median 8.0%, p90 25.5% across 101 entries with spread. Size lane works (counter: raw 6,536,691, gzip 1,758,678, brotli 1,279,800 B; build first 2.5 s / rebuild 1.2 s). Pre-existing failures to track: wasm `./internal/platform/jsdom` benchmarks panic (`syscall/js: Value.Call: property setAttribute is not a function` under plain node, no DOM; also failed in `run-arm64.json`); wasm `./internal/runtime` `BenchmarkHydrateMediumTreeFirstUpdate` fails with "expected hydrated component setter" (passes natively; whether it is a regression from this session or pre-existing is unknown, check once the runtime bug lane is merged). Run 2 must be taken on a frozen tree with no other work running. To cut time, a quiet run can drop the slowest targets after checking run 1's per-target durations.
- [ ] **[impl]** Freeze the v6 baseline: commit the JSON, tag the commit that produced it, and keep the harness building against v6 so any later "is v7 faster" question can be rerun on the same code.
- [ ] **[impl]** Re-baseline under Go 1.26 first (Green Tea GC and smaller wasm heap increments are already in force), then re-run the same suite after the 1.27 bump (V7-P1) to isolate toolchain effects.
- [x] **[impl]** Convert the 2 remaining `b.N` benchmarks to `b.Loop` (1.26 stops it blocking inlining), then re-baseline. Done 2026-10-03: `css/fold_benchmark_test.go` (3) and `sanitize/sanitize_test.go` (1) now use `b.Loop`; many other `b.N` loops remain in `internal/runtime` and the `micro_benchmark_test.go` files (the audit counts about 300 loosely), so convert those before the baseline run.
- [x] **[research]** v6 failure seed list (native, 2026-10-03; 97 root-module packages): 92 pass, 3 fail (`tools/changelogcheck` real, fixed below; `hookcheck` and `tools/gwc` Windows cleanup). Wasm compile-only on 65 core packages: 64 OK, `telemetry` broke (fixed below). Not covered: examples, browser lanes, wasm execution, `-race`, nested modules.
- [x] **[impl]** Fixed 2026-10-03: `telemetry` did not build for js/wasm (`ui.GetSSRObservationAttributes` was native-only; moved to tag-free `ui/correlation_attrs.go`); `wasm-tests.yml` gained a `wasm-vet` job over the core packages.
- [x] **[impl]** Fixed 2026-10-03: `tools/changelogcheck` (docs-site `latest-release-notes.md` mirror re-synced to the CHANGELOG top entry); `go build ./...` now clean on the host (added `main_native.go` stubs to 10 wasm-only example packages whose untagged siblings broke the host build).
- [x] **[impl]** Delete the 14 stray `.exe` files in the repo root (`gwc.exe`, `server.exe`, `runtime.test.exe`, ...) and add a CI check that fails on tracked binaries; ad hoc binaries go to `./bin`. Done 2026-10-03: 14 untracked root .exe files (about 139 MB) removed; guard added as `tools/binarycheck` + `.github/workflows/no-tracked-binaries.yml` (empty allowlist, unit-tested).
- [ ] **[impl]** Remove any `GOWASM=signext/satconv` flags from scripts (ignored since 1.26).
- [x] **[impl]** Done 2026-10-03: `gwc bench` now records a `machine` block (arch, hostArch, cpuModel, logicalCpus, ramBytes, goVersion, gomaxprocs, goflags, buildTags, gitSha, gitDirty), per-benchmark `n`/`median`/`mad`, `-stat mean|median` (default mean), and `-include` (tools/gwc/bench_machine*.go). Committed baselines unchanged.
- [x] **[impl]** Fixed 2026-10-03 (tooling): `go_js_wasm_exec.bat` mangled `-test.run=^$` (cmd.exe eats the caret), so the wasm bench lane ran the WHOLE test suite; replaced by the shell-free `tools/gowasmexec` (built to `bin/gowasmexec.exe` by `resolveWasmTestExec`, `.bat` kept as fallback; still referenced by `desktop-windows.yml`, docs, `desktop/README.md`). `gwc bench`/`bench suite` now treat a PASS with only the benign Windows `unlinkat ... .exe: Access is denied` cleanup error as ok with a `warnings` entry; `bench capture` still treats it as failure. Observation: `./state` wasm tests pass alone but fail when benchmarks run in the same process (assertions like "expected initial derived value 4, got 10" suggest shared global atom state between tests/benchmarks); consider isolating that state.
- [x] **[research]** DECIDED 2026-10-03: the v6 baseline is taken natively as arm64 (`GOARCH=arm64` set for the run only, user config untouched; recorded in the machine block). MACHINE FACT: `go env GOARCH` is `amd64` (set with `go env -w`) on this arm64 host, so native tests and benchmarks run as emulated amd64. Decide whether the v6 baseline is taken natively (unset GOARCH for the run) or as amd64; either way the machine block now records it.
- [ ] **[impl]** Windows environment: Go cannot delete freshly built `*.test.exe` ("Access is denied", probably antivirus/indexer) and 5 `tools/gwc` scaffold tests plus `tools/hookcheck/cmd/hookcheck` `TestCmdEndToEnd` fail because of it (inner tests pass; about 6 minutes per `tools/gwc` run). Exclude the temp dir from the scanner or retry cleanup in the harness; `TestRunVerifyJSONRunsTestsAndCIBuild` is flaky for the same reason.

### V7-P1 — Go 1.27 toolchain upgrade

- [ ] **[impl]** Bump `go.mod`, `tools/livereload`, nested modules, all ~20 workflows (`1.26.x` -> `1.27.x`, `desktop-windows.yml` pin), `.vscode`, docs, Windows desktop pin to 1.27; update `VERSIONING.md` ("current module is /v6").
- [ ] **[impl]** Bump `third_party/GoGRPCBridge` (`go 1.25.0`, `toolchain go1.25.8`) inside its own lifecycle first, then update the root pin (`third_party/README.md`); until then `stdversion` vet may report in that module.
- [ ] **[impl]** Run the new `go fix` modernizers (`atomictypes`, `embedlit`, `slicesbackward`, `unsafefuncs`; `waitgroup` is now `waitgroupgo`) one package at a time; whole-tree runs silently drop conflicting packages (1.26 lesson).
- [ ] **[impl]** One isolated `gofmt` commit (1.27 alignment fix; output no longer varies by CPU architecture).
- [ ] **[impl]** Audit removed GODEBUG settings: search `go.mod` `godebug` lines and `//go:debug` for `asynctimerchan` and other removed names (the go command now fails on removed settings set to old values); confirm timer code (`UseDebounced`, `UseThrottled`, scheduler timers) does not rely on buffered timer channels.
- [x] **[research]** Audit done 2026-10-03 (read-only). HIGH risk for Go 1.27 closure renaming/merging (identity derived from `FuncForPC(...).Name()` or code pointers): `ui/component_handle_shared.go:223-256` (component identity from symbol name, comments at 44-76 assume distinct `.funcN`), `internal/runtime/signature.go:134-151`, `internal/runtime/panic_report.go:461` (hardcoded `SetTimeout.func1`), tests `internal/runtime/panic_diagnostic_test.go:120` and `ui/component_identity_test.go:32`. MEDIUM (function values compared by `Pointer()`): `internal/runtime/hooks.go:31,1118,1128,1162`, `ui/typed.go:118-120`. Action: give components an explicit identity (registered key or type-based) instead of symbol names before the 1.27 bump, and run the component-identity tests under go1.27.1. GOWASM flags and removed-GODEBUG settings: none found in core. `ReverseProxy.Director`: none in core (only in `third_party/GoGRPCBridge`). `errors.AsType`: 21 sites in 14 core files converted. Go 1.27-only candidate sites (apply after the bump): `CutLast` at `html/markdown.go:356`, `internal/diagnostics/report.go:195`, `internal/runtime/signature.go:148,151`, `internal/runtime/panic_report.go:338,383,402`, `internal/telemetryredaction/redaction.go:259`, `ui/component_handle_shared.go:253,256`, `tools/gwc/{agentic_read_model.go:536,680,792; agentic_toolchain.go:546; lint_hooks.go:293; start.go:533}`, `docs/doclint/goblocks.go:263`; no hand-rolled `url.URL` copies found. Original item: Audit function-pointer comparisons and tools that parse closure names (1.27 simplifies and may merge function-literal names): `gwc wasm measure` attribution, flamegraph names.
- [ ] **[impl]** Re-baseline byte-exact tests and size budgets after `compress/flate` changes (gzip sidecars, `prerender`, `gwc release` manifests) and `unicode` 15 -> 17 (`i18n`, `sanitize`).
- [ ] **[impl]** `go mod tidy` under 1.27 (merges `require` blocks) in every module.
- [ ] **[impl]** Rename the module to `/v7` (scripted import rewrite); `api-baseline.yml` and `api_compat_guard` confirm only intended surface changes.
- [ ] **[research]** Measure wasm size and speed on 1.27 alone (before any code change) so toolchain effects are not confused with code effects; include the ~60 KB size-specialized-malloc cost and the `GOEXPERIMENT=nosizespecializedmalloc` A/B.

### V7-P2 — Platform feature adoption

- [x] **[research]** FINDING 2026-10-03 (wasm size lane, measured): Go 1.27 turns `GOEXPERIMENT=jsonv2` on by default, so `encoding/json` is now the v2 engine and the v7 counter wasm (go1.27.1) is 7,424,798 raw / 1,964,808 gzip / 1,434,742 brotli against 6,448,987 / 1,716,268 / 1,262,382 for v6 on go1.26.3: +976 KB raw, +248 KB gzip. Code from `encoding/json/v2` + `jsontext` + `jsonwire` is about 697 KB against 226 KB for the old `encoding/json`; hello-world importing only `encoding/json` is 4.38 MB with the default and 3.11 MB with `nojsonv2`. `GOEXPERIMENT=nojsonv2` on the counter: 6,514,048 / 1,725,225 / 1,270,189 (saves 910,750 raw, 239,583 gzip, 164,553 brotli). DECISION: do NOT adopt `encoding/json/v2` in wasm paths; set `GOEXPERIMENT=nojsonv2` in `buildWasmGoEnv()` (`tools/gwc/test_lanes.go:862`), release templates and docs (the experiment's opt-out is documented as temporary, so re-check on every Go release; 1.28 may remove it). Native server paths may still use v2 where measured faster. Original item: `encoding/json/v2`: decide per payload which paths move (SSR bootstrap sidecars, hydration data, OTLP JSON, agent-bridge snapshots, fetch bodies). Measure unmarshal speed and wasm size; list golden tests that match JSON error text (message text changes in 1.27); check strictness changes (invalid UTF-8 and duplicate keys are rejected) against untrusted-input paths; map tag changes (`inline` -> `embed`, `format`/`unknown` removed) and recheck the recorded `omitempty`/`omitzero` decisions in `docs/CONVENTIONS.md`. 290 files import `encoding/json`.
- [ ] **[impl]** Move the chosen hot payload paths to `json/v2` with explicit options (and `jsontext` for streaming in the telemetry exporter and SSE) after the research item.
- [ ] **[impl]** Replace `github.com/google/uuid` (1 file) with the new std `uuid`; verify v4/v7, parse and format parity.
- [ ] **[impl]** Add `bytes/strings.CutLast` where `LastIndex` is used (12 files).
- [ ] **[impl]** `io.ReadAll`/`fmt.Errorf` allocation wins are automatic (1.26); no code change, note in the changelog only.
- [ ] **[impl]** `net/url`: use `URL.Clone`/`Values.Clone` in `router` state and `fetch` request builders.
- [ ] **[impl]** Hardening: set `net/http` `Server.MaxHeaderValueCount` on the SSR, `serverfn`, examples and `gwc serve` servers; check HTTP/1 response-body auto-drain does not change idle-connection behavior where `MaxIdleConns` is customized.
- [ ] **[impl]** Move `ReverseProxy.Director` users (dev proxies in `gwc dev`, livereload, examples) to `Rewrite` (deprecated in 1.26, header-stripping hole); the first grep was too noisy, so list real call sites first.
- [ ] **[impl]** `os/signal.NotifyContext` cause reporting in `gwc dev`, `gwc serve` and example servers.
- [ ] **[research]** `crypto/mldsa` (post-quantum signatures) for release-manifest signing and `gwc deploy`; not needed on the wasm client. Defer unless release signing is in scope.
- [ ] **[research]** Tools-only wins: `go/types.Hasher`, `go/scanner.Scanner.End`, `go/token.File.String`, `go/constant.StringLen` in `tools/gwc` analyzers (`gwc-hooks`, `inspect`, `mutate`, `model`) and `tools/uicodegen`.
- [ ] **[impl]** `go doc pkg@version` / `go doc -ex`: add executable examples to `ui` and `html` and reference them from error-code docs and `gwc explain`.
- [ ] **[impl]** Check the vet `printf` rule (`%w` operand of type `*E` where `E` implements error) across `internal/runtime`, `fetch`, `tools/gwc`; fix every latent `errors.Is/As` bug it reports.

### V7-P3 — Rendering performance

All items gated on the V7-P0 baseline; record before/after numbers in each checkpoint.

### V7-P3 profile findings (read-only lane p3-render, 2026-10-03, pristine v6 in bin/v6-ref, `-cpu 1`; shares and allocs/op are evidence, timings are not)

HEADLINE: native SSR spends 69-95% of CPU in `runtime.Stack` goroutine-id lookups. Chain: `html.Tag` -> `acquireRenderHostElement` (`internal/runtime/reconciler_elements.go:130`) and `AcquireRenderHostAttrs` (`:166`) -> `GetCurrentFiber` (`:197`) -> `getSSRHookFiber` -> `getSSRHookOwner` (`ssr_hook_fiber_native.go:18`) -> `runtime.Stack`. Those two slab helpers cause 98.9% of the `Stack` calls, and under SSR the slab path is useless (the SSR fiber never equals `rt.activeRenderFiber`, so both fall back to `new(Element)`/`make` after paying a full traceback; cost grows with recursion depth). `ui.BenchmarkRenderToStringPublicSSRSurface`: 93 µs for only 61 allocs/op. With `-ignore=getSSRHookOwner` only 5.4% of samples remain.

Claims refuted or corrected by the profile (the earlier survey was wrong on these): `css.New` does NOT canonicalize before the memo lookup (`css/css.go:47-50` computes `computeFoldKey` and checks `foldCache` first; the comment at `atlas_perf_render_test.go:190` is stale; `css.New` is 0.9% of the Atlas profile; warm `design.Class` is 22 µs, `computeFoldKey` 44% and `growslice` 41% of it, the `growslice` being example-code bundle concatenation); `reflect.DeepEqual`/`TypeOf` prop comparison is NOT hot (`fastEqual` `hooks.go:995` switches on about 25 types first; `propsEqual` 2.2% cum; reflect < 0.5%); `wrapEventHandler` in `event_wrap_wasm.go` is dead code (live path is `wrapEventHandlerCell`, which types only `func()` and `func(string)`, everything else including `func(GoEvent)` goes through `reflect.MakeFunc` + `Call` per event); GOGC=300 is only `ProfileThroughput` for domain workers, the render-thread profile is `Responsive` (40) and no library code applies a profile; `unique`/`maphash` are not used and not needed. Confirmed: `reflect.Value.Call` in `ui.buildComponentRenderer` (`component_handle_shared.go:196`) is 6% of the Atlas profile (about 25% of what remains once `Stack` is gone; typed fast paths exist for `func() Node`, `func(map[string]any) Node`, `func(runtime.Attrs) Node`); fiber cloning (`Fiber` is 432 B; `buildUpdatedFiber` `reconciler.go:397` 22% cum in the production keyed-update profile, the `*new = Fiber{...}` literal at :406 is 210 ms of 530 ms); the map props lane still costs 24% of the keyed-update profile (`cloneElementProps` `reconciler_elements.go:589` `maps.Copy` 19%, `mapassign_faststr` 19%; compact lane is 241 ns/1 alloc vs 777 ns/5 allocs for `CreateElement(map)`; `html.toRuntimeCompactProps` runs about 60 sequential `if field != ""` tests per element); commit has no deadline check at all (`commitRoot` `reconciler_commit.go:~340-470`, only render has `newFrameBudgetDeadline`). Extra: dev-only `renderPassOwnerID` -> `computeHookGoroutineID` (`runtime.Stack`) is 27.7% of non-production keyed component update (`reconciler_elements.go:233`; compiled out under `-tags production`); `css.computeFoldKey` re-hashes every rule's scope strings per call (`fastfold.go:96`, `mixString` 34% of warm fold); `syscall/js` costs 24-31 allocs/op per CreateElement/AppendChild and 16 per AddRemoveListener; `attr_batch.go` batches only attr set/remove/text ('a','r','t').

Work packages (separate file ownership, ranked; final factors need a quiet machine):
- [ ] **[impl]** WP1 SSR goroutine-id removal (files `internal/runtime/reconciler_elements.go` acquire helpers, `ssr_hook_fiber_native.go`, `ssr.go`): add `activeSlabFiber()` returning nil when `storeSSRHookCount.Load() != 0` else `currentFiber`, no `sync.Map` or `Stack`; use it in `acquireRenderHostElement`, `acquireRenderComponentProps`, `AcquireRenderHostAttrs`; seed the owner id in `ssrHookOwnerContext` regardless of `hookThreadingGuardEnabled` and make `setSSRHookFiber` use its `parseOwner` argument (today `_`). Hook calls under SSR still cost one `Stack` each (about 1%). Expected: Atlas `AllRoutes` from about 36 ms to 2-3 ms/op (>=10x, confirm quietly); allocs/op unchanged. Risk low. Acceptance: `BenchmarkAtlasPerfAllRoutes`, `ui.BenchmarkRenderToStringPublicSSRSurface`, `Stack` < 2% in the profile, >= 8x lower ns/op, allocs within +-2%.

2026-10-20 WP1 implementation checkpoint (commit `8d3b4bfd`): the three slab helpers now call `activeSlabFiber()`, which skips the goroutine lookup while any native SSR component is active; each SSR entry seeds a native owner id once and `setSSRHookFiber` reuses it. `TestSSRSlabLookupSkipsHookOwner` protects the distinction. Focused SSR tests pass in default and `production` builds; full `go test ./internal/runtime ./ui`, `go vet` for both packages, and wasm test-binary compilation pass. The production `internal/runtime` binary's SSR suite also passes when run directly; a package-level `go test -tags production` run reported both packages passing but exited nonzero because Windows denied deletion of `runtime.test.exe`. Benchmark and CPU-profile acceptance are still open because the host is busy. A detached `cc72b7b1` baseline worktree exists at `C:\Users\mreca\Desktop\GoWebComponents-ssr-base` for the controlled A/B.
- [ ] **[impl]** WP2 css fold key (css/ only): precompute `scope.hash` at scope construction; add a `css.Bundle` type or `NewBundles(bundles ...[]Rule)` caching a per-bundle digest (removes the flatten allocation in `design.Class`); keep `canonicalize` unchanged. Risk medium-low (a stale hash is a silent wrong-class bug; fuzz/determinism tests must stay green). Acceptance: `FoldWarmFourBundles`, `AtlasPerfDesignClassWarm`; >= 3x fewer ns, <= 1 alloc.
- [ ] **[impl]** WP3 client fiber and props (`internal/runtime/types.go`, `reconciler.go`, `reconciler_elements.go`): split `Fiber` into a hot header of about 150 B plus a `*fiberExt` for cold fields shared across alternates until written; route `CreateElement(map)` through the compact lane when all values are strings; skip `cloneElementProps` for owned maps. Expected 15-20% off component-update CPU, allocs 110 -> about 75. Risk high (hot reload, hydration, layout tests). Acceptance: `FineGrainedKeyedDashboardComponentUpdate16`, `ComponentUpdateWithProps`, `RuntimeLayoutBaselines`; fiber bytes <= 300, allocs -20%.
- [ ] **[impl]** WP4 html builder presence mask (`html/html.go`): bitmask set by `With*` options so `toRuntimeCompactProps` tests mask words instead of about 60 fields. Expected 20-25% of its flat time; risk medium (a missed bit drops an attribute). Acceptance: `TagDivWithCommonProps`, `VirtualListBuild`; >= 15% fewer ns at the same allocs.
- [ ] **[impl]** WP5 ui reflect fallback (`ui/component_handle_shared.go`, `ui/ui.go`): migrate hot call sites (Atlas first) to the typed `ui.Typed` path; do not try to speed up `reflect.Call` itself. About 25% of post-WP1 Atlas time; acceptance `reflect.Value.Call` < 1%.
- [ ] **[impl]** WP6 dev-guard cost (dev builds only; `reconciler_elements.go`, `runtime.go`): compute the render owner id once per Runtime on the frame-loop goroutine (`frame_loop_owner.go`) and re-verify only on foreign entry; about 27% of dev-build update CPU; risk medium (weakens a diagnostic).
- [ ] **[impl]** WP7 wasm boundary (`internal/platform/jsdom/*`, `event_wrap_cell.go`): add 'p' (property set) and 'e' (listener add/remove) ops to the batch; typed fast paths for `func(GoEvent)` and `func(js.Value)` handlers; atomic `firstInteractionCaptured` check before the `schedulerMu` lock in `recordFirstInteraction`; delegated root listeners later. See the interop research document for the larger rethink (command buffer + applier).
- [ ] **[impl]** WP8 commit slicing (`reconciler_commit.go`): resumable commit cursor; risky, only matters for very large commits; needs a browser long-frame harness.
- Caveat: `go version` inside `bin\v6-ref` printed go1.26.3 (its own go.mod, as intended); the jsdom benchmark set crashes under plain node after the first benchmark (fake DOM lacks `setAttribute`; fixed in the v7 tree by the wasm-tests lane).

### V7-P3 size findings (read-only lane p3-size, measured; gzip level 9, brotli q11)

v6 counter (go1.26.3, release profile) 6,448,987 raw / 1,716,268 gzip / 1,262,382 brotli; v7/dev counter (go1.27.1, same flags) 7,424,798 / 1,964,808 / 1,434,742 (+976 KB raw) almost entirely because Go 1.27 enables `GOEXPERIMENT=jsonv2` by default (see the JSON finding in V7-P2). Hello world go1.27.1: 1,855,042 / 546,468 / 422,947. Atlas client (release) 21.5 MB / 4.94 MB / 3.50 MB (with `nojsonv2`: 20.58 / 4.72 / 3.36 MB). Counter wasm code section 4.84 MB, data 2.57 MB (pclntab/type metadata/strings, not attributable). Top code by package: runtime 1,186 KB, first-party `internal/runtime` 670 KB, `encoding/json/v2` 447 KB, `reflect` 303 KB, `jsontext` 194 KB, `regexp/syntax` 177 KB, `math/big` 153 KB, `time` 148 KB, `slices` 139 KB, `fmt` 119 KB, `html` 106 KB, jsdom 87 KB. The release profile's `-trimpath -buildvcs=false` saves 43 KB raw / 10 KB gzip; `-s -w` about 157 KB raw. `perf_budgets.json` allows 8,000,000 raw bytes for the counter, so 7.42 MB still passes (the 1.6 MB gzip target is missed).
- [ ] **[impl]** Size #1 (one line, biggest win): `GOEXPERIMENT=nojsonv2` in `buildWasmGoEnv()` (`tools/gwc/test_lanes.go:862`), release templates and docs: saves 910,750 raw / 239,583 gzip / 164,553 brotli (measured). Risk medium (unsupported long term; error texts differ; json-compare tests may break; re-check each Go release).
- [ ] **[impl]** Size #2: `validate` regexps (`validate/validate.go:87-91`, 5 patterns pulled into every wasm app via `ui`) replaced by hand-written matchers (lane running) and goldmark moved off the default import graph (`html/markdown.go` imports goldmark which brings 17 package-level regexp compiles at boot; plan: pluggable engine + `html/markdown` package that registers itself; lane running). Combined 565,058 raw / 141,146 gzip / 99,496 brotli (measured, prototype on top of #1).
- [ ] **[impl]** Size #3: CBOR opt-in for SSR resume (`ui/ssr_bootstrap.go`, `ui/ssr_transfer.go` have no build tag and pull `cbor` and `math/big`; `kvstate/codec.go` also imports cbor): optional codec with JSON fallback; 284,772 raw / 74,890 gzip / 53,964 brotli (measured). Medium risk.
- Combined #1+#2+#3: 1,760,785 raw / 454,433 gzip / 319,311 brotli saved (measured with overlays): counter 5,664,013 / 1,510,375 / 1,115,431; wasm code section 4.84 -> 3.62 MB (-25%).
- [ ] **[impl]** Size #4: gate panic report/diagnostics/hydrate builders in `internal/runtime` under `!production` (about 100-140 KB code est; panic 108 KB, hydrate 38 KB, diagnostics 34 KB, serialize 33 KB, hot-reload 23 KB of the 670 KB first-party runtime); release panics must still report. Medium risk.
- [ ] **[impl]** Size #5: `serverfn` browser `Call` via fetch instead of `net/http` (Atlas only; `crypto/tls`+`net`+`net/http`+`x509` about 2 MB of Atlas code, estimate); breaks `SetClient(*http.Client)` on wasm; the Atlas example itself still imports `net/http` (example change out of scope).
- Rejected: `wasm-opt -Oz` (v116 installed: -392 KB raw but only -3.4 KB gzip and +10.7 KB WORSE brotli; not browser-tested); hand-rolled JSON replacement (about 250 KB code est, high risk); `nogreenteagc` (44 KB raw/12 KB gzip but changes GC behavior). Regexp inits: 5 first-party `MustCompile` in the wasm graph (`validate/validate.go:87-91`), goldmark 17, unicode `Categories`/`Scripts` map init 50 KB; the repo has 13 `MustCompile` sites (not 11), 19 in v6-ref.

- [ ] **[impl]** `css.New`: hash or intern the raw input before `canonicalize` (`css/rule.go:180`, `css/registry.go:159,169`); 46% of Atlas native render CPU and 63% of allocs, warm four-bundle fold 23.9µs/26.7 KB.
- [ ] **[impl]** Replace the `reflect.Value.Call` in `ui.buildComponentRenderer` (about 10% of CPU, 10.8% of allocs) with typed generic lanes; enabled by generic methods (1.27).
- [ ] **[impl]** Replace `reflect.DeepEqual`/`reflect.TypeOf` prop comparison (`internal/runtime/reconciler.go:1339-1445`) with typed equality; keep `reflect` iterators (1.26 `Type.Fields`, `Value.Fields`) for the remaining reflective paths.
- [ ] **[impl]** Reduce allocations in `buildUpdatedFiber` and `cloneChildFibers` (17.5% and 3.5% of reconcile CPU); move host props off `map[string]any` toward a slice/struct table with interned names.
- [ ] **[impl]** Extend the `internal/platform/jsdom` batching (`attr_batch.go`) to property sets, event add/remove and structural ops (about 65 direct `Call/Set` sites in `adapters.go`).
- [ ] **[impl]** Remove the per-event closure plus `defer` in `internal/runtime/event_wrap_wasm.go` where a shared dispatcher works.
- [ ] **[impl]** Slice commit work inside the frame budget (`commitRoot`, `reconciler_commit.go:355`); target the 8 long frames (budget 0) and 6.4 ms GC pause (budget 3 ms) in `docs/PRODUCTION_READINESS.md` (M2, M7).
- [ ] **[research]** Re-tune `gcpacing/pacing.go` profiles and the `GOGC=300` release policy under Green Tea GC (default since 1.26); measure pauses and long frames.
- [ ] **[research]** Variable-size `make` stack allocation (1.26): inspect `-gcflags=-m` for the hot slice builders in reconcile and child lists; use the `bisect` tool flag `-compile=variablemake` if it misbehaves.
- [ ] **[research]** Size-specialized small-object allocation (1.27; about 1% overall expected, up to 30% per small alloc): A/B on Atlas SSR (about 12 allocs per element) and the browser benchmark, against its +60 KB binary cost.
- [ ] **[research]** `unique.Handle[string]` (and `weak`) for CSS class and attr-name interning and fiber/DOM caches.
- [ ] **[research]** `hash/maphash.Hasher`/`ComparableHasher` for memo, fetch cache and `css.New` keys; adopt only if it beats the current hash.
- [ ] **[research]** `//go:wasmexport` and reactor mode to cut JS-to-Go bridge calls (zero uses today); confirm against the 1.24/1.25 notes first.
- [ ] **[impl]** Boot and size: lazy `regexp` compile (11 files with package-level `MustCompile`; `regexp` init is 4.4% of cold boot), trim package init (2.6%), `encoding/json` cost (8%), `wasm-opt`, route splitting, streaming instantiate. Target the Atlas client (4.55 MB gzip against 1.6 MB budget).
- [ ] **[impl]** Make sure devtools `Inspect`/`collectFlamegraph` are compiled out of production builds (more than 50% of CPU and 90% of allocs when on).
- [x] **[research]** SIMD spike DONE 2026-10-03, verdict: do NOT add to v7 core. Real wasm SIMD128 works (`simd.Emulated()=false`, 128-bit, v128 opcodes in the compiler listing; the portable API has no any-true/all-true reduction, so no early exit). Median ns/op, scalar vs simd: byte-scan contains 16 B 71 vs 77 (wasm, a tie), 64 B 207 vs 78, 256 B 823 vs 135, 4096 B 13645 vs 848 (native 4096: 4007 vs 904); but it loses to `bytes.Equal` on native and costs +204 KB raw / +49 KB gzip wasm (+8.2%) plus a build-wide `GOEXPERIMENT`; measurements were noisy (about 2x spread, throttled machine). `strings.ContainsAny` is far slower than either (118 µs at 4096 on wasm). Revisit when `simd` graduates and profiling shows large wasm string scans. Original item: SIMD spike. 1.27 `simd/archsimd` adds wasm 128-bit vectors and the portable `simd` package exists on all architectures (both need `GOEXPERIMENT=simd`; API unstable; `simd.Emulated()` reports real vs emulated). Steps: build with `GOEXPERIMENT=simd GOOS=js GOARCH=wasm`; confirm `Emulated()` is false in the browser; micro-benchmark (a) the SSR inline-script escape scan in `ui/ssr_transfer.go:157` and (b) byte/string equality in props and keyed-list compare in `internal/runtime/reconciler.go`, on wasm and native; adopt only with an end-to-end win on Example 201 or the Atlas SSR profile and a small size cost, behind a build tag with a pure-Go fallback (a library cannot set `GOEXPERIMENT` for consumers; `gwc build` could). Do not use it for FNV hashing (byte-serial, no gain).
- [ ] **[impl]** SIMD follow-up replacement (pure Go): `escapeJSONForInlineScript` (`ui/ssr_transfer.go:157`) builds a new `strings.NewReplacer` on every call; hoist it to a package-level value and add a fast path for input with no special bytes (`strings.IndexByte`-style pre-check); benchmark before and after on native and wasm.
- [ ] **[impl]** Exit gate: no Example 201 scenario regresses beyond noise; geomean better than the V7-P0 baseline by a stated margin (stretch: back to the v4-era 0.8 band).

### V7-P4 — API flexibility and breaking changes

### V7-P4 design findings (read-only lane l8, 2026-10-03; code claims verified by grep unless marked)

Most CashFlux gaps are ALREADY shipped; todos.md still lists them as open. Verified present: variadic `UseEffect(fn, deps ...any)` (ui/ui.go:586, G12); `DisabledIf/ReadOnlyIf/SelectedIf/AttrIf/ClassIf/StyleIf` (html/sugar.go:652-1276, U2); `ClassNames/ClassMap/css.Class` (U3); `UseDOMRef`, `UseAutoFocus`, `Portal`, `UseTimeout/UseInterval`, `UseMediaQuery`, `MapKeyedComponent`, `router.Href/FragmentHref`, reactive `UseRoute` on wasm. Real v7 work is the list below. NOT verified: 1.27 generic-method syntax (toolchain was 1.26.3), `fastEqual` behavior for func/slice/map deps, whether `hookcheck` runs in CI, whether CHANGELOG names a replacement for every deprecated symbol.

- [ ] **[impl]** G17 typed `OnInput`: `func OnInput[F InputFn](fn F) PropOption` with `InputFn interface{ ~func() | ~func(string) | ~func(ui.Event) }`, plus `OnInputHandler(ui.Handler)`; today `OnInput(any)` (html/sugar.go:729) goes through `toHandler` (calls the `ui.UseEvent` hook) and `invokeEventCallback` (sugar.go:1070) silently ignores a wrong signature. Breaking for `any`/`ui.Handler` arguments (compile error is the migration); no `go:fix inline` possible.
- [ ] **[impl]** G11 `css.Inline(rules ...Rule) html.PropOption` (must live in `css`, which imports `html`); values containing `;`/`}` rejected; keep `Style(map[string]string)`.
- [ ] **[impl]** U1 attribute table: generate from one table shared by `html` and `html/shorthand`: `Draggable, Scope, Download, ContentEditable, InputMode, Slot, Part, Popover`, form attributes, `AriaLabel/AriaHidden/AriaExpanded/AriaLabelledBy/AriaDescribedBy`, SVG presentation attributes (`ViewBox, Stroke, Fill, D, Cx`...); golden test against the WHATWG list. U2: add `CheckedIf`, `RequiredIf`, `IfProp(cond, opts ...PropOption)`. U3: document one canonical class path, add no third.
- [ ] **[impl]** G12 follow-ups: dev diagnostic when an effect dep is a func/map/slice; named `ui.Always` sentinel for every-render effects.
- [ ] **[impl]** G1 hooks in loops: (b) first (non-breaking): dev-mode `MapKeyed` sets a list-render depth counter and `GoUseFunc` (internal/runtime/hooks.go:877, positional hook slot) warns inside it, naming `MapKeyedComponent`; `hookcheck` only flags `Use*` calls, not `OnClick(func)`. (a) later and behind the benchmark gate: bind handlers by host fiber + prop name instead of hook slot (`toHandler` returns an unwrapped `ui.Handler`; commit binds one cell per host node, released on unmount). Also: `a11y/dialog.go:46` comments `html.OnClick` is NOT a hook but it is (via `toHandler`); `GoUseFunc` panics outside render so `OnClick(func)` cannot build nodes in global handlers.
- [ ] **[impl]** G6/G7 router: make `UseParams`, `UseQuery`, `UseRouteData` subscribe to the same atom as `UseRoute` (they read package globals, router/router_api.go:84-98; `UseRoute` native is a zero-value stub); add `router.WithBasePath(string)` applied in `Href`/`Navigate`/matching and a documented SPA-fallback server helper; convert `RouterOptions`/route `Options` to `RouterOption`/`RouteOption`.
- [ ] **[impl]** Missing primitives (the rest exist): `ui.UseConfirm() func(ConfirmOptions) <-chan bool` on `Portal`; `UseFocusTrap(ref DOMRef, active bool, opts ...TrapOption)` (breaking; keep old `FocusTrapOptions` form as a deprecated wrapper); `PickFile(accept, cb, opts ...PickOption)` (`PickMultiple`, `PickCapture`, `PickOnError`) and `Download(...) error` (breaking, wrapper); merge the two `UsePersistedState` (ui/persisted_state.go:83 vs kvstate/hook.go:36) into the kvstate one with a thin `ui.UsePersistedState(key, init, ui.PersistIn(ui.Session))` wrapper.
- [ ] **[impl]** Options convention for v7: functional options (`<Target>Option`, `With<Thing>`) for tuning knobs; plain structs only for props/wire config; use `*T` with Go 1.26 `new(expr)` where zero is meaningful and the default is not zero; ban variadic-first-struct parameters (`parseOptions[0]` silently drops the rest: `router.Register` router.go:550, `RenderMarkdown` html/markdown.go:63, `SmokeHydrate`). Model: `query.New(...Option)` with `WithStaleTime`/`WithMaxEntries`. Rewrites: `fetch.RetryPolicy` -> `fetch.Retry(...RetryOption)`; `fetch.BreakerConfig` -> `NewCircuitBreaker(...BreakerOption)` (note `OpenDuration` has no default); `ui.HydrationOptions` -> `Hydrate(root, sel, ...HydrateOption)` (overlapping `Bootstrap`/`BootstrapRef`); `ui.CompositeNavigationOptions.Orientation` free string -> typed; `html.MarkdownRenderOptions` -> `Markdown(src, ...MarkdownOption)` with `WithURLSchemes` (nil = default allowlist is security-relevant); `fetch.Options`/`Fetch` -> `fetch.Do(ctx, url, ...RequestOption)`; `fetch.QueryOptions/CacheOptions` -> `WithStaleAfter/WithMaxAge/WithTags/WithPersist`; `livereload.LiveReloadOptions.Port` string -> int. Also reconcile the three different `UseQuery` (ui, fetch, router) and the two parallel query stacks (`query` and `fetch`).
- [ ] **[impl]** Generic methods (concrete types only; they cannot satisfy interfaces or be method values; keep bodies thin for wasm size): `Atom[T].Select[U]` (also Derived/Signal/ComputedSignal/GlobalAtom), `*query.Cache` `Fetch[T]/Mutate/SWR/Snapshot` with the free functions kept as `//go:fix inline` wrappers, `TaskState[T].Map[U]`, `fetch.Query[T].Select[U]`, `Params.As[V]`, `router.Query.Decode[T]`, `Form[T].Field[V](sel func(*T) *V)`, `State[T].Lens[V]`, `Snapshot.Get[T]`, `Debounced[T].Map[U]`. Must NOT change: `selectorSource[T]` (state/state.go:39), `PropOption`, `reactiveSource`, `persistedStateHandle`, `interop` interfaces. Hook-creating methods stay free functions.
- [ ] **[impl]** Removals (deprecated >= 96 days ago, v4.1/v4.2/v5/v6 released since; confirm CHANGELOG names each replacement first): `css.Property` (`css.Raw`), `fetch.ReturnChannel`, `hotreload.IsEnabled` x3 build variants (`Enabled`), `Router.GoRegisterRoute` (`Register`), `Router.GoGetRoute` (`Current`), `state.Select` (`UseSelector`), `state.ExportSnapshot` (`GetSnapshot`), `state.ImportSnapshot` (`ApplySnapshot`), `utils.EnableHotReload/IsHotReloadEnabled/InstallHotReloadBridge` (6 Warn sites, two files); deprecated without `Warn`: `agentbridge.Build*Envelope` (`New*Envelope`), `css.Opacity` (`OpacityNum`), `css/u.BgToken/TextToken` (`BgC/TextC`), `css.WhenDisabled` (`Disabled`), `fetch.UseFetch`, `WorkerPool.GetSize/GetQueueLimit`, `Router.GetCurrentRouterPath`, `router.GetOutlet` (`UseOutlet`), `state.UseComputed` (`ui.UseMemo`). Deprecate now (add `Warn`): `ui.WrapHandler`, `fetch.Fetch`, `router.UseQuery`, `router.UseLocation`, old `FocusTrapOptions` form, one of the two `UsePersistedState`. The repo has no `//go:fix inline` yet; add them on replacement wrappers.
- [ ] **[impl]** Before changing signatures: add `api_baseline.txt` for `ui`, `html`, `router`, `fetch` (only 9 packages have one, so the major/minor gate does not cover the most affected packages).
- Sequence (lowest risk first, lanes by package ownership): A `deprecation/` + gwc lint + CHANGELOG (Warn calls, `go:fix` wrappers, baselines); E `html/`, `html/shorthand`, `css/` (attribute table, `CheckedIf`, `IfProp`, `css.Inline`, typed `OnInput`); B `state/` + `query/` (generic methods 1, 2, 9) before C `fetch/` (options rewrite, generic method 4); D `router/`; F `ui/`, `internal/runtime`, `a11y/`, `kvstate/` (focus trap, `PickFile/Download`, `UseConfirm`, persisted state, G1 (b) then (a) behind the benchmark gate); G last and serial, one commit per package: the removals above.

- [ ] **[impl]** Use generic methods (1.27) for builders and hook helpers (`State[T]`, `Task[T]`, `UseMemoOf`/`UseEffectOf`-style, `ExecuteWithPolicy[T]`-style free functions, `html` typed children) while keeping interface-based public contracts as free functions (interface methods cannot take type parameters).
- [ ] **[impl]** Adopt `new(expr)` for optional fields and F-bounded constraints (`type Adder[A Adder[A]]`, 1.26) in typed builder/fluent APIs; use the 1.27 struct-literal field selectors for composed option structs (`HydrationOptions`, `FocusTrapOptions`, `RetryPolicy`).
- [ ] **[impl]** Standardize constructors and hooks on functional options; fix zero-value traps (`RetryPolicy.Multiplier:0` disables backoff, `HalfOpenMaxCalls:0` allows unlimited probes).
- [ ] **[impl]** CashFlux API-shape gaps (`todos.md` section "CashFlux-driven framework gaps"): G12 `UseEffect` multi-dep, G17 `OnInput` accepting plain funcs, G11 typed `Style`, U1 (about 200 attributes fall back to `Attr(k,v)`), U2 conditional helpers (`DisabledIf`, `CheckedIf`, `AttrIf`), U3 class builder, G1 handlers in variable-length loops.
- [ ] **[impl]** Missing primitives P-A to P-D: DOM ref and autofocus, portal, dialog, focus trap, timer hook, media-query hook, persisted atom, file picker/download helper.
- [ ] **[impl]** Router: G6 reactive `router.InspectCurrentRoute`, G7 deep-link refresh 404 and `<base href>` breaking in-page anchors.
- [ ] **[impl]** G15: split UI-layer logic out of `js && wasm` files so it is natively unit-testable.
- [ ] **[impl]** Remove v6-deprecated symbols that meet the 2-minor/90-day rule: `css.Property`, `fetch.ReturnChannel`, `hotreload.IsEnabled`, `router.Router.GoRegisterRoute`/`GoGetRoute`, `state.Select`/`ExportSnapshot`/`ImportSnapshot`; shim the rest. Regenerate `api_baseline.txt` and `api_compatibility_baseline.json`; the diff becomes the migration guide.
- [ ] **[impl]** Put `//go:fix inline` on each renamed or deprecated v6 wrapper so `go fix` migrates users automatically.
- [ ] **[impl]** Module split: move CLI/test tooling (`tools/gwc`, playwright-go, bubbletea, AI SDKs, agent-bridge server side) to a nested tools module; record consumer `go.sum` before/after. (Open: in scope for v7.0 or a v7.1 follow-up; assumed in.)
- [ ] **[impl]** Make `api_compat_guard` run in `api-baseline.yml` (today only `gogrpcbridge-ci.yml:109` invokes it).
- [ ] **[impl]** Docs-only: update `docs/STARTERS.md`, reference manual and examples for the new option and method shapes.

### V7-P5 — Better error messages

- [ ] **[impl]** Typed error-code registry package (`ErrCode`: code, headline, remediation, docs URL, recoverability). Attach codes at the panic/error site; keep the substring table in `internal/runtime/diagnostic_metadata.go` as a legacy fallback, then delete it (codes are currently assigned by `strings.Contains` on message text, so rewording drops them).
- [ ] **[impl]** Generate `docs/errorcodes` and `gwc explain` from the registry; delete the regex parser in `docs/errorcodes/errorcodes.go` and the drift test's parser dependency.
- [ ] **[impl]** Runtime hook-signature check (hook count/order changing between renders) in `internal/runtime/hooks.go`, dev builds only; today only hot reload diffs hook order (`hot_reload.go:653`).
- [ ] **[impl]** Component source location in reports: extend `describeFiber`/`diagnosticComponentStack` (`internal/runtime/inspect.go:682`) to include file and line, and attach the component stack to panic reports, not only error-boundary ones (`error_boundary.go:~60`).
- [ ] **[impl]** Give the bare errors a code and actionable wording: `internal/runtime/async_boundary.go:224`, `ui/ssr_stream.go:161,173,188` (new `GWC-SSR-*` family), CLI errors such as `tools/gwc/enterprise_runtime.go:170`.
- [ ] **[impl]** Make docs links absolute and configurable (today relative `ACTIONABLE_ERRORS.md#...`).
- [ ] **[impl]** Convert `errors.As` to `errors.AsType` (15 files, including `interop.IsCode` and error-code lookup paths).
- [ ] **[impl]** Label scheduler, render, effect and bridge goroutines with `runtime/pprof` labels (`gwc.component`, `gwc.lane`, `trace_id`); go 1.27 tracebacks now include them. Redact sensitive values the way `internal/telemetryredaction` does.
- [ ] **[impl]** Guard test: a new `panic`/`fmt.Errorf` in framework code without a registry code fails CI; every code resolves in `gwc explain`; golden-file tests for rendered reports.

STATUS 2026-10-03: P5 steps 1-5 done and verified: internal/errcode (32 codes, Error/New/Is/CodeOf), diagnosticMetadata and phase functions read the registry with golden parity tests (35 messages x severities x classifications + 12 phases), ActionablePanicOptions.Code wins when registered, docs/errorcodes renders from the registry, all Docs links now resolve to docs/REFERENCE_MANUAL/error-codes.md#code, gwc explain reads the registry. Follow-ups: move interop (5 gwc-interop-* codes), devtools, css, tools/gwc, livereload codes into the registry; stale ACTIONABLE_ERRORS.md mentions remain in docs/REFERENCE_MANUAL/README.md, llms-full.txt and diagnostics test fixtures; tests in router/ui had their Docs assertions updated by line-scoped sed (review). Steps 6-10 remain.

Research findings (2026-10-03, read-only lane): 51 `panic(` sites in framework packages (12 have no code, 5 are plain re-throws, 3 control flow), 411 `fmt.Errorf`/`errors.New` sites with 0 GWC codes (interop 151, runtime 57, fetch 42, pwa 41, ui 37, router 29, agentbridge 23); `interop`/`pwa` already use their own lowercase `*interop.Error` codes. Codes are matched by `strings.Contains` in `diagnosticMetadata` (`internal/runtime/diagnostic_metadata.go:18-155`, 31 codes); an unmatched message falls back to `GWC-FRAMEWORK-USAGE`. Codes also live outside it: `devtools/devtools_wasm.go:408-432`, `css/registry.go:44,179`, `tools/gwc/agentic_*.go`. BUG: every `Docs` anchor points at `ACTIONABLE_ERRORS.md`, which does not exist in the repo (dead links); `gwc explain` reads `diagnostic_metadata.go` from disk, so it breaks on an installed binary. 29 test files assert error text (top: runtime `hook_threading_test.go`, `panic_diagnostic_test.go`, `panic_guard_test.go`, `panic_matrix_test.go`, `inspect_test.go`).

- [x] **[impl]** Step 1: create tag-free `internal/errcode` (`Code`, `Family`, `Severity`, `Entry{Code,Family,Headline,Remediation,Docs,Recoverable,Severity}`, `Lookup`, `All`); it imports no framework package, so no cycle with runtime/ui/router/interop.
- [x] **[impl]** Step 2: move the 31 existing codes plus the devtools, css and tool codes into the registry, with a parity test against the current switch.
- [x] **[impl]** Step 3: render `docs/REFERENCE_MANUAL/error-codes.md` from `errcode.All()` (drop the regex parser in `docs/errorcodes/errorcodes.go`, keep the drift test) and repair the dead `ACTIONABLE_ERRORS.md` anchors.
- [x] **[impl]** Step 4: `gwc explain` uses `errcode.Lookup` (`tools/gwc/agentic_read_commands.go:463,503`), no source-tree read.
- [x] **[impl]** Step 5: `errcode.Error` (`New`, `Unwrap`, `Is` by code, `CodeOf` via `errors.AsType`); `ActionableFrameworkPanic` takes the code directly, the substring table stays only as fallback.
- [ ] **[impl]** Step 6: code the 12 uncoded panics: `ui/form.go:275`, `ui/form_native.go:277`, `ui/hydration_island.go:56`, `ui/suspense_shared.go:22`, `internal/runtime/ssr_hook_fiber_native.go:23,27`, `router/contracts.go:101,152,174,188,202`.
- [ ] **[impl]** Step 7: component `File`/`Line` via `FuncForPC(...).FileLine` in `describeCallableIdentity` (`internal/runtime/signature.go:114-144`), cached per `ComponentType`, added to `diagnosticComponentStack` (`inspect.go:682`) and `Where`; near-zero wasm size cost (pclntab already ships).
- [ ] **[impl]** Step 8: hook-order detection: `Hooks.signature` already records kinds per render (`signature.go:70`, gated by `hookSignatureRecordingEnabled`, off in production wasm) but is only compared on hot reload; add `prevSignature` reuse and report `GWC-RUNTIME-HOOK-ORDER` ("slot 2: was state, now memo") once per fiber. A same-kind reorder needs a call-site PC hash (one `runtime.Caller` per hook); decide separately.
- [ ] **[impl]** Step 9: AST guard test over framework packages: `panic(X)` / `fmt.Errorf` / `errors.New` must use `errcode.New` unless waived with `//errcode:ok`; start with an allowlist (5 re-throws, the suspension panic) that only shrinks; second test: every entry has headline, remediation and docs anchor.
- [ ] **[impl]** Step 10: convert runtime, ui, router, state, fetch `Errorf` sites in batches, retire `diagnosticMetadata`; add `(*interop.Error).GWCCode()` and `GWC-INTEROP-*` entries; update the 29 text-asserting tests as wording improves.

### V7-P6 — Better OTel

- [ ] **[impl]** New native-only `telemetry/otel` package (`!js`): bridge `trace.Recorder` spans and `telemetry.RUMEvent` to a user-supplied `TracerProvider`/`MeterProvider`; API-only deps (`otel`, `otel/trace`, `otel/metric`), no SDK.
- [ ] **[impl]** SSR: start render spans in `WrapSSRCorrelation` (`ui/correlation_native.go`) using the global text-map propagator instead of the bespoke `traceparent` parser.
- [ ] **[impl]** Propagation: inject/extract W3C `traceparent` in `fetch` (wasm and native) and `serverfn`, so browser fetch -> server function -> server component forms one trace (no propagation exists today).
- [ ] **[impl]** Metrics: render and commit duration histograms, hydration mismatch counter, panic counter by `gwc.error.code`; add the 1.26 `runtime/metrics` scheduler gauges (`/sched/goroutines:*`, threads, goroutines-created) on native.
- [ ] **[impl]** Browser exporter (`telemetry/telemetry.go`): replace padded-index IDs with real trace/span IDs; add resource, scope, span kind and status to the OTLP JSON; use `jsontext` if JSON v2 is adopted; keep the wasm build free of OTel imports (assert in a test).
- [ ] **[impl]** Emit diagnostics as span events with `exception.*` semantic-convention attributes plus `gwc.error.code` and `gwc.component_stack` (shared schema with V7-P5).
- [ ] **[impl]** `logging`: fan out with `slog.NewMultiHandler` (1.26) to console, file sink and the OTel log bridge.
- [ ] **[impl]** Keep `internal/telemetryredaction` in every export path; add a test that nothing bypasses it.
- [ ] **[impl]** Move `examples/server/ai-chat-wizard` onto the framework bridge instead of its own OTel wiring (it and `third_party/GoGRPCBridge` are the only current importers).
- [ ] **[research]** Check `agentbridge` and `devtools` for existing OTel hooks (not surveyed).

STATUS 2026-10-03: P6 steps 1-5 done and verified (vet native+wasm, tests): internal/gwcattrs constants; telemetry OTLP now validates/derives deterministic IDs, sets kind/status/parentSpanId/resource+scope attrs, redacts in one function, golden JSON test; interop NewTraceID/NewSpanID/Format/ParseTraceparent; fetch injects traceparent (wasm fetch+XHR) when caller did not set one and the origin is same-origin or in the new Options.TracePropagationOrigins (NOTE: behavior change, every same-origin wasm fetch now carries a header and generates a fresh trace; reconsider default or make opt-in); test/wasmdeps guards no go.opentelemetry.io in wasm or core. Steps 6-10 (telemetry/otel bridge, SSR span, native propagation, metrics, cross-boundary test) remain.

Research findings (2026-10-03, read-only lane): `fetch` (wasm `fetch_wasm.go:65-73`, XHR `:219-224`, native `fetch_native.go:11-20`) and `serverfn` (`serverfn.go:349`, handler `:278-310`) send and read no trace headers. `telemetry/telemetry.go:223-241` fabricates IDs by hex-padding the index and only length-checks them; `telemetry` has no redaction call (`internal/telemetryredaction` is wired only through `logging/redaction.go`). `trace/timeline.go` has its own Span/Correlation system with per-thread clocks, so its spans can be replayed into OTel after they finish, not started live. `resolveTraceContext` defaults trace flags to "00" (unsampled). Dependencies: `otel`, `otel/trace`, `otel/metric` v1.43.0 already resolve; `otel/propagation`, `attribute`, `codes`, `baggage` and `semconv/v1.40.0` live in the `otel` module (no go.sum growth); promote `otel/metric` to direct. The root `otel` and `otel/propagation` packages import `net/http` and `crypto/tls`, so the wasm build must import NO `go.opentelemetry.io/*`; core packages (`ui`, `fetch`, `serverfn`, `logging`) stay OTel-free and gain a tiny tag-free hook (`ui.SetTracePropagator`), and only the new native-only `telemetry/otel` imports OTel.

- [x] **[impl]** Step 1: tag-free `gwc/attrs` constants shared with the error registry: `gwc.error.code`, `gwc.component_stack`, `exception.*`, existing `gwc.ssr.*`.
- [x] **[impl]** Step 2: fix `BuildOTLPJSON` IDs (validate hex, reject all-zero) with golden JSON tests (fixed clock and IDs, negative cases).
- [x] **[impl]** Step 3: browser exporter: resource attributes (`telemetry.sdk.language=go`, `service.version`, `deployment.environment`), scope version, per-type span `kind` (today hard-coded `3`), `status` (code 2 on error), `parentSpanId`.
- [x] **[impl]** Step 4: interop random-ID helper (`crypto.getRandomValues`, 16/8 bytes, never all zero) and wasm `fetch` `traceparent` injection for same-origin or allowlisted origins only (cross-origin triggers CORS preflight); honor the incoming sampled flag.
- [x] **[impl]** Step 5: wasm import guard test: `GOOS=js GOARCH=wasm go list -deps` over wasm-built packages fails on any `go.opentelemetry.io`; native guard that `ui`, `fetch`, `serverfn` import no OTel.
- [ ] **[impl]** Step 6: `telemetry/otel` (`!js || !wasm`): `Install(Options)`, tracer/meter scope `.../v7`, bridge `trace.Recorder` spans and `RUMEvent`s to a user `TracerProvider` (replay finished spans with `WithTimestamp`; non-hex correlation IDs normalized by hashing).
- [ ] **[impl]** Step 7: SSR: `WrapSSRCorrelation` extracts with the global propagator via the hook, starts a `gwc.ssr.render` server span, derives `X-Correlation-ID` from the span trace ID, injects response headers; keep the legacy header fallback; default to ParentBased sampling.
- [ ] **[impl]** Step 8: native `fetch` and `serverfn` client inject; the `serverfn` handler wraps the call in a `gwc.serverfn.<name>` server span.
- [ ] **[impl]** Step 9: metrics: `gwc.render.duration`, `gwc.commit.duration`, `gwc.hydration.mismatch`, `gwc.hydration.duration`, `gwc.panic` (by `gwc.error.code`), `gwc.ssr.render.duration`, `gwc.serverfn.duration`, `gwc.fetch.duration`; observable gauges from `runtime/metrics` (`/sched/goroutines:goroutines`, `/sched/latencies:seconds`, `/gc/heap/live:bytes`). Bound cardinality: `gwc.component` capped near 200 with an `_other` bucket, no URLs or IDs as attributes.
- [ ] **[impl]** Step 10: cross-boundary test: `httptest` + `tracetest.InMemoryExporter` asserting one trace ID and correct parentage across SSR, serverfn and a `fetch_native` call; redaction applied inside the single attribute-conversion function (fail closed on redactor error; redact `exception.message` and `gwc.component_stack`).

### V7-P7 — Test refinement and bug hunt

- [x] **[impl]** FIRST PASS DONE 2026-10-03: 26 sleep-based tests converted to `testing/synctest` bubbles (internal/runtime 11, fetch 10, query 1, ui 5), `-count=20` stable in all four packages; `ssr_stream_outoforder` tests went from 0.70 s of real delay to about 0.01 s with exact timing assertions. Only about 20 core native test files actually qualified; the rest are real I/O, wasm-only (synctest is host-only) or timeout guards. Production blockers for deterministic time (findings, not changed): `fetch/realtime.go:561,610,667` direct `time.AfterFunc`; `fetch/cache.go` (11 sites) and `fetch/cache_persistent.go` (3) direct `time.Now()`; `internal/runtime/scheduler.go:50,54`, `hooks.go:646-648`, and the real-sleep poll loop at `internal/runtime/agent_write.go:162-173` (untested). Introduce an injectable clock in `fetch`, then convert the rest. Original item: Convert sleep-based tests to `testing/synctest` (67 files use `time.Sleep`, 0 use `synctest`): scheduler, debounce/throttle, retry/circuit breaker, fetch, SSE/WebSocket, async boundary; use `httptest.NewTestServer` (1.27) for `fetch`/`serverfn`. Host-only, not wasm.
- [ ] **[impl]** Run the `goroutineleak` profile (GA in 1.27) at the end of every test lane and expose it via `diagnostics/diagnostics_http.go`; target agent bridge, `fetch` WebSocket/SSE hooks, `UseTask`/`UseChannel`, `backgroundworker`, scheduler, `servercomponents`. Confirm it works under wasm or host-only.
- [ ] **[impl]** Use `T.ArtifactDir` (1.26) for Playwright screenshots, traces and console captures in CI failures; make hanging browser tests fail with the captured console/page error (the M10 lesson).
- [ ] **[impl]** Use `go test -json` `OutputType` (1.27) in `gwc test --json` and `verify --agent` NDJSON to classify errors instead of scraping.
- [ ] **[impl]** CI: run the `playwrightgo` browser lanes (only 6 of 20 workflows use the tag), root `TestMainSuite`/`TestExamplesAll`, and the race lane on every PR on a supported host; document that windows/arm64 local runs skip race.
- [ ] **[impl]** Re-establish the `bench-drift` gate on the new baseline (the committed one is stale: `GoUseEffectSameDeps` +60%, `FineGrainedSelectorDashboardReactiveTextUpdate16` +123% in the old drift note); fix the bimodal churn benchmark instead of only reporting it.
- [ ] **[impl]** Raise `ui` test coverage (largest exported surface, 363 symbols, lowest test ratio) and natively test the logic freed by G15.
- [ ] **[impl]** Close React parity gaps marked partial in `docs/REACT_TEST_PARITY.md`: ReactHooksWithNoopRenderer, ReactIncrementalSideEffects, ReactMemo, Fizz out-of-order Suspense streaming.
- [ ] **[impl]** Fuzz/property tests: reconciler keyed diffing, SSR/hydration round-trip, attribute escaping (extend `internal/runtime/attr_escape_fuzz_test.go`), `localfirst` CRDTs.
- [ ] **[impl]** Bug hunt round 1: fix the V7-P0 seed list; every bug gets a failing test first.
- [ ] **[impl]** Bug hunt round 2: static pass (`go vet` incl. `stdversion` and `printf`, `gwc lint`, race, `goroutineleak`).
- [ ] **[impl]** Bug hunt round 3: adversarial review per package, starting with `ui` and `internal/runtime`; `tools/gwc` and the chat-wizard example were excluded from the earlier test review, include them now.
- [ ] **[impl]** Remaining open `todos.md` items worth folding in: WAILS-005, 007, 015, 018 through 024; agent-bridge wasm-client reload signal (`todos.md` ~3288); typed CSS layers (~3356-3368); OPFS VFS (~3457).
- [ ] **[impl]** Real-app acceptance: migrate CashFlux (still on v4.2) to v7; run the GoGRPCBridge stress run that was never done.

### V7-P8 — Docs and release

- [ ] **[impl]** CHANGELOG v7.0.0 with a migration guide generated from the API baseline diff; update README, `VERSIONING.md`, `ROADMAP.md`, reference manual, `docs/errorcodes`.
- [ ] **[impl]** Support matrix: drop macOS 12 (Go 1.27 needs macOS 13+); note macOS/Linux desktop hosts stay out of scope for v7 unless decided otherwise.
- [ ] **[impl]** Release candidates `v7.0.0-rc.N`, full `release` lane, then tag; v6 gets 12 months of fixes per `VERSIONING.md`.

### V7 open questions

- Module split: in scope for v7.0 or a v7.1 follow-up (assumed in).
- Desktop: expand Windows-only certification to macOS/Linux in v7 (assumed out).
- Go 1.25 notes were not fetched; confirm `wasmexport`, `unique`/`weak`, `synctest` GA details before relying on them.

## V6 Wails desktop integration

### v6.1 Windows API expansion — end-of-day checkpoint

Paused at the maintainer's request on 2026-09-08. Detailed acceptance and
evidence: [v6.1 tracker](docs/plans/v6.1-windows-api-parity.md) and
[manual verification](docs/plans/v6.1-windows-manual-verification.md).

- [x] Implement typed, gated in-scope Windows API families and GWC tester controls.
- [x] Resolve Astra's adapter/fork ownership, lifecycle and threading findings.
- [x] Create/push the patched Wails fork and update the local exact revision pin.
- [x] Run native/Wasm contracts, adapter tests and the 26-check rebuilt EXE smoke.
- [x] Verify native title/menu/context/radio and basic picker interactions.
- [x] Test autostart/global shortcut registration and restore prior state.
- [x] Fix save-cancel wording, child-template mismatch and misleading error outcomes with regression tests.
- [x] Update development log, unreleased changelog and completed todos.
- [ ] Rebuild/recheck final tester fixes and finish outstanding native manual acceptance.
- [ ] Complete final release regression verification, commit GWC, tag v6.1.0 and verify production publication.

The test tray remains in the running lab awaiting manual interaction; a Windows
Security prompt interrupted UI automation. No v6.1 release is claimed. Older
multi-platform/signing and incomplete manual todos below remain open because
their full acceptance criteria have not been met.

### Full regression repair — 2026-09-08

Priority correction: framework verification comes first. Example-app repairs are
paused with their changes preserved. The immediate gate includes all root
framework, internal, docs, test/testkit and developer-tool packages, native and
Wasm modes, framework/CLI browser tests, race tests and the isolated desktop
adapter. Application packages under examples/ and research/ are reported
separately, never used to label the framework suite failing or passing.

Framework checkpoint (2026-09-08): native state updates, realtime lifecycle,
SSR request isolation, docs invocation scanning and Wasm API/CSS checks have
focused regressions passing. The complete native framework matrix passed.
Both Windows symlink safety tests executed elevated and passed without skips;
no Windows security settings were changed. GC pause measurement now handles
zero-resolution clocks using a conservative runtime-histogram upper bound;
three focused runs passed, including both measurement paths. Scaffold server
cleanup now terminates its owned listener before cancelling the launcher;
two focused smoke runs passed. Final race/runner and consolidated matrix
verification remain open. Example work remains paused.

- [x] Repair native/Wasm/JavaScript contract and lifecycle failures, with focused regressions.
- [x] Complete the framework/system native, Wasm, browser, race, nested-module and conditional-test acceptance matrix before resuming example repairs.
- [ ] Repair Atlas and Example 100 browser failures against freshly built fixtures.
- [ ] Repair relocated example and benchmark harness failures; execute previously unfinished browser cases.
- [ ] Run uncached root, nested modules, Wasm, JavaScript, browser and available race suites; Astra review final evidence.

Checkpoint: failure inventory is in `docs/plans/v6-regression-suite-20260908.md`.
The current request authorizes repairing these failures, not weakening checks or
claiming unexecuted tests passed. Existing dirty files and the Atlas database are preserved.

Framework acceptance checkpoint: `docs/plans/v6-framework-system-tests-20260908.md`
records the final passing per-package matrix and exact conditional-test
counterparts. Native and race each cover 93 tested packages plus one no-test
documentation package; Wasm covers 28 and browser covers two. Nested tools,
adapter and JavaScript checks pass. Failed historical attempts remain recorded;
the two race packages affected by Windows child-executable cleanup have explicit
passing instrumented reruns. No framework test remains unverified solely because
of a prerequisite skip. Example repairs and manual release acceptance remain
separate, unchanged work items.

Design and inspected integration points: [V6 Wails desktop plan](docs/plans/v6-wails-desktop.md).
This section is the authoritative completion tracker for the integration.
Execution started on 2026-09-08. The user authorized several Luna subagents for
bounded work and an Astra verification pass. Independent technical tasks may be
checked after verification; milestone/release gates remain open until their
  manual acceptance tasks pass. Existing next-work markers elsewhere are unchanged. Current milestone:
**Windows integration implemented and reviewed; release acceptance remains open**.
The user's continuation request authorizes code integration while manual checks
are pending; this does not waive those acceptance gates or imply platform support.
The initial release is **Windows first**, explicitly selected by the user on
2026-09-08. macOS/Linux work in the original plan is deferred, not a hidden skip.
After each item, append a checkpoint with changed files, validation command/result,
evidence location, residual risk, and next item. Do not mark skipped validation as passing.
Task descriptions retain their original acceptance criteria; current API and
command behavior is documented in `desktop/README.md` and the Windows reference.

### D0 — Version, environment, and baseline

- [x] **WAILS-001 — Verify and pin the Wails integration contract.** Depends on: none.
  Inspect the plan's candidate `v3.0.0-beta.17` source/templates; record the CLI,
  module/runtime versions, generated JS exports, cancellation API, asset routes,
  and supported binding-generation order. Acceptance: reproducible version/API
  notes explain a clean build with no pre-existing embedded assets and whether
  direct ES modules work without npm/Vite. Reassess the pin if evidence requires it.
- [x] **WAILS-002 — Validate Windows prerequisites and module isolation.** Depends on: WAILS-001.
  Check Go, WebView2, and native build requirements; verify dependency resolution
  for the proposed isolated `examples/desktop/wails-counter/` module, including
  any needed local replacements. Acceptance: documented commands resolve the
  actual dependencies without adding Wails to the root module or changing global
  Go environment settings; imports match the currently declared GWC module version.
- [x] **WAILS-003 — Record the browser counter baseline.** Depends on: WAILS-002.
  Build/run the existing browser counter and record tool versions, artifact size,
  startup measurement method/results, and interaction behavior. Acceptance:
  repeatable baseline evidence exists for comparing the desktop prototype.

### D1 — One working Windows application

- [x] **WAILS-004 — Create the isolated Wails counter shell.** Depends on: WAILS-003.
  Add separate native/frontend entrypoints, shared DTOs, embedded local assets,
  and a reproducible task file in `examples/desktop/wails-counter/`. Reuse the
  Wasm renderer, typed CSS, and hash router. Acceptance: a clean build launches
  one native Windows window and the counter responds; native code never calls ui.Render.
- [ ] **WAILS-005 — Prove native services, dialogs, and progress events.** Depends on: WAILS-004.
  Add a typed Go service round trip, an open-file dialog, and native progress
  events with cleanup. Acceptance: actual-WebView evidence covers successful
  calls, service errors, dialog cancellation, and event delivery without blocking input.
  Progress: service/error/event native smoke passes; OS picker selection/cancellation remains manual.
- [x] **WAILS-006 — Integrate generated modules and ordered startup.** Depends on: WAILS-005.
  Use interop.ImportModule/Module.Call for the generated service bindings and an
  external __gwcImportModule helper. Acceptance: bootstrap verifies runtime/adapter
  readiness before native controls activate; missing bindings surface visibly;
  selected CSP works without the Function-constructor import fallback.
  Technical acceptance verified by Astra. The wider D1 gate remains open for WAILS-005's manual picker check.
- [ ] **WAILS-007 — Validate packaged startup and shutdown.** Depends on: WAILS-006.
  Test local-only cold startup, matching wasm_exec.js, Wasm MIME/non-streaming
  fallback, routing, resize, keyboard input, boot failures, and window close.
  Acceptance: executable runs without the dev server/network, closes cleanly,
  and the existing browser counter still passes its baseline interactions.
  Progress: automated native startup/fallback/failure/routing and browser regression pass;
  visible keyboard/resize/title-bar close and disconnected launch remain manual.

### D2 — Frontend adapter and lifecycle contracts

- [x] **WAILS-008 — Extract the optional desktop frontend package.** Depends on: WAILS-007.
  Add the smallest `desktop/` adapter with js/wasm implementation, native
  unavailable stubs, and an injectable transport. Acceptance: browser/SSR use
  reports unavailable predictably; frontend dependencies exclude native Wails;
  focused native and Wasm contract tests pass.
  Verified: `desktop/` adapter and native stubs adopted by the example; injected,
  actual JS/Wasm and native-WebView contracts pass. D1 manual prerequisite remains open.
- [x] **WAILS-009 — Implement typed calls, errors, capabilities, and cancellation.** Depends on: WAILS-008.
  Define DTO/wire handling for nil, bytes, time, and wide IDs; distinguish remote,
  encode/decode, missing-binding, cancellation, timeout, and closed-window errors.
  Add a version/capability handshake and forward cancellation through the verified
  Wails request API. Acceptance: typed round trips pass and a cooperative native
  task demonstrably stops; cancellation of completed writes is not promised.
  Verified: native smoke observes backend cancellation after UI Cancel and route
  unmount; errors/wire contracts pass. Native protocol/platform/version/method-map
  handshake now runs before Wasm startup. It is compatibility detection, not authorization.
- [x] **WAILS-010 — Prove bounded request cleanup.** Depends on: WAILS-009.
  Cover cancellation before/during a call, deadlines, rejection, late completion,
  unmount/remount, window close, and never-settling promises. Reuse UseTask/UseTaskCtx.
  Acceptance: callback/request resources are bounded, stale UI results are ignored,
  and no late callback invokes a released JS function; backend and UI cancellation
  evidence is recorded separately.
  Verified: bounded adapter-owned JS registries and synchronous polling avoid Go
  callbacks retained by native promises. Cancel-during-poll, late/never settlement,
  reentrant close and deadline tests pass. Arbitrary host-retained JS heap is not bounded.
- [x] **WAILS-011 — Harden desktop event subscriptions.** Depends on: WAILS-010.
  Add typed event mapping, explicit namespaces, idempotent unsubscribe, and bounded
  burst/coalescing behavior. Acceptance: lifecycle/burst tests leave no subscriptions
  after unmount/reload/close; local events.Subscribe/Publish remain local unless
  explicitly bridged. Use a cancellation function or adapter-owned subscription type.

### D3 — Scaffolding and CLI workflow

- [x] **WAILS-012 — Add desktop target metadata.** Depends on: WAILS-011.
  Extend scaffold metadata in tools/gwc/start.go and its loading/normalization in
  dev.go; define native/frontend/assets/output fields and precedence. Acceptance:
  malformed targets fail clearly, schema migration/additive-field policy is explicit,
  and existing schema-v1 web fixtures retain their defaults. Keep app metadata out
  of machine-level runner path overrides.
- [x] **WAILS-013 — Add a reproducible desktop template.** Depends on: WAILS-012.
  Generate portable DTOs, separate entrypoints, pinned dependencies, local assets,
  and the proven clean binding/build order. Acceptance: an app generated into a
  fresh directory resolves, builds, and launches without stale dist assets;
  web templates still work. Leave unrelated tools/uicodegen work untouched.
  Verified contributor-linked fresh scaffold and native package; independent
  published templates and arbitrary layouts are explicitly not supported yet.
- [x] **WAILS-014 — Add desktop init, doctor, and build commands.** Depends on: WAILS-013.
  Reuse the existing Wasm build stage and invoke native compilation with separately
  scoped environments. Update dispatch, help, registry metadata, JSON envelopes,
  and applicable MCP schema/allowlists. Acceptance: success/failure outputs agree
  across interfaces, desktop checks are opt-in, mutation flags are accurate, and
  web build/release semantics and relevant tests remain intact.
- [ ] **WAILS-015 — Add supervised desktop development.** Depends on: WAILS-014.
  Give one supervisor ownership of frontend rebuild/reload, native restart, and
  DTO-triggered binding regeneration. Exclude generated outputs, verify runtime
  routes/readiness, and define port-conflict/process ownership behavior. Acceptance:
  each edit class triggers the correct work, rebuilds do not loop, and exit/failure
  leaves no orphan host/server. Document state behavior on native restart.
  Progress: supervisor rebuilds/restarts the full app; owned-process cleanup,
  rebuild failure and cancellation have focused tests. Frontend-only reload is
  deferred, and a complete interactive edit-class acceptance pass remains open.

### D4 — Durable storage and multiple windows

- [x] **WAILS-016 — Add native application-data persistence.** Depends on: WAILS-015.
  Replace the desktop use of db/sqlite's temp-directory approximation with an
  explicit app-owned data path, injectable in tests, and host-owned database lifecycle.
  Acceptance: data survives restart at the documented path; tests remain isolated;
  path/open/close failures surface as actionable errors.
- [x] **WAILS-017 — Implement a service-backed kvstate backend.** Depends on: WAILS-016.
  Implement kvstate.PersistenceBackend through typed native services, with bounded
  payloads and defined version/conflict behavior. Acceptance: Load/Save/Delete/Keys,
  missing keys, service failures, concurrent writes, and restart durability pass
  focused tests; browser storage remains selectable.
- [ ] **WAILS-018 — Connect native commits to state invalidation.** Depends on: WAILS-017.
  Add a configurable invalidation source or narrow equivalent around kvstate/watch.go
  to consume Wails commit events. Acceptance: other windows reload committed changes,
  duplicate/self events do not loop, cleanup is verified, and browser BroadcastChannel
  behavior remains covered.
  Progress: actual two-window native invalidation passes. Shared hub/version
  regression tests do not certify real browser BroadcastChannel persistence.
- [ ] **WAILS-019 — Validate independent windows with shared durable state.** Depends on: WAILS-018.
  Exercise two windows, conflicting edits, reload/reopen, and close during in-flight
  work. Acceptance: UI-only state remains per-window, backend state resynchronizes,
  conflicts follow the documented rule, and subscriptions/database resources close cleanly.
  Progress: native two-window/local-state smoke and storage conflict/reopen tests
  pass separately; the complete interactive conflict/reopen/close matrix is open.

### D5 — Compatibility, packaging, and release evidence

- [ ] **WAILS-020 — Add the desktop test lane and CI workflow.** Depends on: WAILS-019.
  Extend test dispatch/normalization and add explicit nested-module tests plus actual
  Windows/macOS/Linux WebView smoke jobs. Acceptance: desktop failures are reported
  distinctly, missing native environments are explicit skips/limitations, and root
  tests are not used as a substitute for testing the nested example module.
  Scope update: Windows native CI only for the user-selected initial release;
  macOS/Linux native jobs are deferred. Portable Linux race testing is separate.
  Progress: local desktop lane passes normal 17-check smoke, buffered fallback
  and both expected startup failures. Workflow added; hosted CI has not run.
- [ ] **WAILS-021 — Record desktop feature compatibility.** Depends on: WAILS-020.
  Verify focus, IME, accessibility, native dialogs, fonts, DPI, CSS, routing,
  clipboard, and external links on each claimed platform; probe workers/storage
  separately. Acceptance: a versioned matrix links evidence and documents limitations;
  PWA installation and service workers are not mandatory for desktop startup.
  Progress: [Windows compatibility matrix](docs/plans/v6-wails-windows-compatibility.md)
  records verified capabilities and manual gaps; untested features remain unclaimed.
- [ ] **WAILS-022 — Validate the privileged WebView boundary.** Depends on: WAILS-021.
  Test trusted local assets, actual Wasm/script CSP needs, external navigation,
  explicit service registration/input validation, and production removal of dev/agent
  endpoints. Acceptance: untrusted navigation cannot inherit native privileges,
  valid service calls still work, and each supported engine passes its content policy.
  Progress: strict local host/origin/referrer middleware, input bounds and CSP
  tests pass. Full hostile-navigation acceptance remains open.
- [ ] **WAILS-023 — Add desktop packaging and signing configuration.** Depends on: WAILS-022.
  Implement the proposed desktop package command, artifact manifest/hashes/tool
  versions, installer/runtime prerequisites, and configurable signing/notarization.
  Acceptance: packaged artifacts pass native smoke tests; missing signing credentials
  produce an explicit unsigned/blocked result, never a signed-release claim.
  Automatic updates, mobile, and deep links remain outside this initial release.
  Progress: fresh generated project packages successfully; ZIP/executable hashes
  match its explicitly unsigned manifest. Signing/installer configuration remains open.
- [ ] **WAILS-024 — Measure release performance and publish support documentation.** Depends on: WAILS-023.
  Compare cold start and interaction with WAILS-003; measure package size, memory
  per window, and bridge latency with representative payloads. Acceptance: measured
  budgets and repeatable commands are recorded, the desktop workflow/reference docs
  are updated, and every claimed platform has packaged-app evidence and known limits.

### Wails execution checkpoints

- **2026-09-08 / final integration drive:** Extended typed clipboard/message/window/
  screen SDKs with versioned envelopes and isolated Windows adapter; completed
  feature policies and dual-target tooling. Three Luna workers implemented scoped
  work and Astra independently reviewed/tested and fixed boundary/CLI regressions.
  Unit, real Wasm, real Chromium, native all/none/selected and compiled-ceiling
  checks pass. Latest unsigned artifact is `bin/v6-release-20260908`; exact hashes,
  commands, failures and reruns are in `docs/plans/v6-completion-verification.md`
  and `docs/plans/v6-astra-final-verification.md`. Visual Windows acceptance is
  genuinely blocked by black screenshots and failed activation, not waived.
  Signing/publishing, hosted CI and unperformed manual certification remain
  explicitly unclaimed; no release tag, commit or push was made.

- [x] **DESKTOP-SDK-001 — Prove the portable API/replaceable adapter boundary.**
  Implemented Wails-free file-dialog contracts, four typed workflows, Supports/
  Require, immutable host opt-in, separate desktop/wails module and shared legacy/
  lab picker routing. Native/Wasm/fake-backend and actual WebView enabled/disabled
  tests pass; fresh scaffold builds. See docs/plans/v6-desktop-public-api.md.
  Scope is file dialogs, not a complete Wails facade or published adapter release.
- [ ] **DESKTOP-SDK-002 — Extend only proven common desktop workflows.**
  Apply the tested boundary to clipboard read/write, narrow window operations and
  lifecycle as concrete contracts; preserve native escape hatches. Add per-feature
  host checks and adapter upgrade tests, then establish separate adapter releases.

- [x] **DESKTOP-GATE-001 — Typed capabilities and host-enforced policy.**
  Implement the reviewed proposal in `docs/plans/v6-desktop-build-gates.md`:
  minimal Client.Supports/Require, explicit optional feature groups, fail-closed
  host checks covering API actions, legacy picker, menus and report export.
  UI flags may restrict UX but never grant native authority.
  Verified native all/none/selected feature matrices and direct-call guards.
  Policy now covers every listed group, with backend intersection and compiled
  ceiling; disabled persistence is not opened. See v6-completion-verification.md.
- [x] **DESKTOP-GATE-002 — Seamless web/desktop build selection.**
  After GATE-001, add gwc_desktop source selection, standalone web bootstrap,
  distinct outputs, target/features metadata and build/dev/package command
  plumbing. Preserve web defaults and existing desktop aliases.
  Implemented typed mode query, stock Go tag selection, isolated assets/web output,
  canonical feature flags/manifest metadata and build/dev/test/watch/package paths.
  Fresh scaffold builds and real Chromium/native WebView checks pass; inherited
  GOFLAGS cannot smuggle desktop-tagged files into an explicit web build.
  Final fresh-scaffold testing reproduced and fixed missing metadata-aware web
  build/dev dispatch. First-class CLI Chromium E2E passes (15.193s); actual dev
  HTTP serving and Go-source watcher rebuild pass (8.657s), including inherited
  desktop-tag rejection. Canonical overrides fail explicitly; native assets stay
  separate. Final focused CLI safety tests pass (2.602s).
- [ ] **DESKTOP-GATE-003 — Gate matrix and developer documentation.**
  After GATE-002, test web/SSR, desktop-none/selected/full, missing host, rejected
  features and direct binding bypass attempts; verify fresh scaffold/dev restarts
  and document tagged-file versus runtime-requirement examples.

- [x] **WINAPI-001 — Build and exercise the Windows API Lab (2026-09-08).**
  User requested comprehensive native testing, including context menus and pickers.
  Add real native menu/dialog/window/clipboard/screen controls, bounded exportable
  evidence and explicit visual verdicts. Preserve the counter regression route.
  Validate unit contracts, actual WebView smoke and real Windows inputs against
  synthetic fixtures. Record cancelled versus completed versus observed-pass
  separately; do not claim full Windows API coverage from a finite tester.
  Checkpoint: native API service, menu definitions, Wasm tester UI, CLI template,
  20-check native smoke and 21-check fallback pass; fresh scaffold builds and runs.
  Real manual context menus, pickers, messages, window states, shortcuts and export
  exercised. See `docs/plans/v6-wails-windows-api-manual.md` for build identities,
  fixes, partial coverage and explicit remaining risks.

- [ ] **WINAPI-002 — Isolate bulk-entry/paste duplication and finish manual gaps.**
  Bulk computer-use text entry doubled in WebView inputs; single keypresses did
  not. Investigate pinned accelerator handled-state reset versus native Edit paste
  before choosing a minimal fix. Direct clipboard tests need explicit operator
  opt-in. Continue IME/screen-reader/tab-order, all picker Cancel variants,
  physical multi-DPI, resize and export-overwrite checks; never infer these from
  unit-test or smoke success.

- **2026-09-08 / WAILS-001:** Added `.gitmodules` entry and exact gitlink for
  `third_party/wails` at `5bce785eb1efbd121ef2e1cb2588eb50b9c59068` (beta.17).
  Luna inspected generated ES-module/runtime, service lifecycle, cancellation,
  and embedding contracts. Validation: submodule HEAD matches release commit;
  upstream checkout clean. Evidence: [contract](docs/plans/v6-wails-contract.md)
  and [submodule setup](docs/plans/v6-wails-submodule.md). Risk: beta APIs need
  actual native smoke coverage. Next: WAILS-002.
- **2026-09-08 / WAILS-002:** Isolated example module resolves local GWC and
  Wails source; root go.mod remains unchanged. Validation: `go list -m all`,
  isolated `go test ./...`, and native host build passed. Installed Go and
  WebView2 versions recorded in [environment evidence](docs/plans/v6-wails-environment.md).
  Risk: only this Windows environment is verified. Next: WAILS-003.
- **2026-09-08 / WAILS-003:** Built the existing browser counter (6,647,102 bytes)
  and recorded hash/toolchain/timing. Counter render/increment passed in Chromium,
  Firefox, and WebKit; standard browser lane passed. Full cross-browser matrix
  had an unrelated WebKit storage-server failure and Windows temp-unlink failure,
  retained in [baseline evidence](docs/plans/v6-wails-environment.md). Risk:
  browser startup timing is not native WebView startup. Next: WAILS-004.

- **2026-09-08 / WAILS-004 through WAILS-007:** Three Luna agents implemented
  the isolated host/frontend, build/bootstrap, source contract and service tests;
  Astra reviewed, corrected and verified the actual packaged Windows application.
  Changed files: `examples/desktop/wails-counter/`, example index and evidence docs.
  Validation: clean-assets `go run ./tools/build`, native tests/vet, scoped gwc
  verify, native and js/wasm lint/vet passed. Hidden WebView2 smoke covers rendered
  local/native increments, typed errors, progress, three route cycles, MIME/CSP and
  forced buffered loading; missing bindings/Wasm correctly exit 1 with visible errors.
  Fresh counter tests passed Chromium, Firefox and WebKit; browser artifact hash
  is unchanged. Full commands/results: [Astra verification](docs/plans/v6-wails-verification.md).
  Windows recursive-checkout workflows now set core.longpaths only for checkout;
  all three modified YAML files parse, but hosted CI has not run.
  Residuals: native shutdown logs WebView2 warning 1412; physical OS interactions
  and disconnected launch are not certified. WAILS-004 is complete; WAILS-006's
  technical checks pass but its prerequisite remains open. Next: finish WAILS-005
  and WAILS-007 manual gates before D2 adapter extraction. D2–D5 remain pending.

- **2026-09-08 / WAILS-008 through WAILS-011:** The subsequent continuation
  authorizes D2 onward while retaining D1 manual gates. Added optional `desktop/`
  adapter, typed wire contracts, cancellable polling transport, bounded event
  queues, native unavailable stubs and the example capability handshake. Astra
  fixed cancellation-during-poll, unsafe wire integers, hostile error getters and
  reentrant close handling. Validation: native, real JS/Wasm and Node contract
  tests plus actual WebView cancellation/unmount smoke pass; see
  [D2 evidence](docs/plans/v6-wails-adapter-verification.md). Root additionally
  verified the native handshake and local-origin guard in the 17-check smoke.
  Residuals: adapter-owned registries are bounded, not arbitrary host JS memory;
  latest-value topics are not an audit log. Next: complete fresh CLI/storage
  verification and retain the manual Windows release gates.

- **2026-09-08 / WAILS-012 through WAILS-014:** Added optional schema-v1 desktop
  metadata and `gwc desktop init/doctor/build` dispatch, help and registry entries.
  Fresh source-only scaffolds build and package with exact Wails pin checks;
  malformed paths, unsupported layouts and unsafe overwrite targets are rejected.
  Validation: targeted CLI tests and fresh generated-project doctor/build/package
  passed. Root independently compared both executable and ZIP SHA256 values with
  `bin/desktop-final-verify-20260908/bin/wails-counter.manifest.json`; both match
  and `signed` is false. A later fresh attempt exposed a smoke timer-boundary
  failure under background throttling; Astra corrected predicate/deadline order,
  added deterministic tests, and a new fresh package passed all 17 native checks.
  Residuals: template is contributor-linked, not a published standalone starter;
  unsigned packaging is not production release acceptance. Evidence:
  [final integration verification](docs/plans/v6-wails-integration-verification.md).
  Next: retain the documented manual, browser-persistence and release gates.

- **2026-09-08 / WAILS-016 and WAILS-017:** Added the example native storage
  service and `desktop.NewStorageBackend`, with explicit app-owned data, bounded
  file/record sizes, atomic replacement, kernel lock ownership and typed errors.
  Astra fixed encoded quota/reopen mismatch, bounded reads/base64 allocation,
  cancelled queued commits and stale/tombstone binding reconciliation. Validation:
  native and real Wasm desktop/kvstate tests and vet pass; nested service tests,
  vet and lint pass; subprocess crash/reopen proves persistence and lock release.
  Actual two-window smoke additionally demonstrates shared saved state while
  local counters stay independent. Evidence:
  [final integration verification](docs/plans/v6-wails-integration-verification.md).
  Residuals: default browser BroadcastChannel end-to-end coverage, complete native
  conflicting-edit/reopen scenarios and Windows manual acceptance remain open.
  Next: WAILS-018 acceptance and the outstanding Windows release gates.


---

## Archived Historical Campaigns

Completed and historical June 2026 audit/campaign checklists (3,500 lines) were archived to [docs/archive/todos-2026-06.md](docs/archive/todos-2026-06.md) to keep this file focused on active tracks.
