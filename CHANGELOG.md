# Changelog

## 2026-03-25

### Agent guidance refresh for gwc usage

- Simplified `AGENTS.md` into a terse operator format and added a compact `gwc` command map with effective feature coverage, key flags, primary docs links, and practical command examples.

### GWC launcher expansion, browser-suite migration, and Example 100 UI refresh

- Added launcher-owned wasm experiment helpers under `gwc wasm` (`measure`, `compare`, `compare-compression`, `compare-cache`, `compare-toolchain`) plus `gwc bench` compare/capture follow-ups and updated launcher docs around the expanded benchmark and wasm workflow.
- Added `gwc env` so operator-facing environment variables can be inspected in text or JSON output with secret-shaped values redacted by default.
- Documented `gwc env` in both `docs/GWC.md` and `tools/README.md`, including JSON output and redaction behavior guidance.
- Migrated remaining browser-suite and workspace wiring from npm/TypeScript Playwright configs to launcher-owned Playwright-Go flows, removed deprecated npm test workspaces and legacy spec/config files, and updated docs/workflow references to match.
- Refreshed `examples/100-ai-chat-wizard` landing/auth/pricing surfaces, updated chat-shell bootstrap copy and related server tests, and rebuilt shared Tailwind output through the launcher-owned CSS pipeline.
- Fixed Example 100 landing-header navigation actions to use app-route-aware links for home/pricing/login/chat entry behavior.
- Removed deprecated `tools/*.ps1` and `tools/*.sh` wrapper scripts now superseded by the launcher-owned `gwc` command surface.
- Removed legacy `test/browser/index.ts` and `test/browser/index.test.ts` bridge files that were tied to the older browser-harness path.
- Removed checked-in generated browser helper/service-worker/static-script artifacts from examples static/script paths that are now treated as generated runtime assets.

### Tailwind CLI first-class launcher workflow

- Added a first-class `gwc tailwind` command in `tools/gwc` that regenerates the shared Tailwind manifest and stylesheet without npm, with machine-readable `-json` output support.
- Added standalone Tailwind CLI download and cache management under `third_party/tailwindcss/bin/<version>/`, including checksum verification against the upstream `sha256sums.txt` release artifact when available.
- Updated `examples/100-ai-chat-wizard/cmd/build-client` to run `go run ./tools/gwc tailwind` before wasm compilation so Example 100 no longer depends on npm for CSS rebuilds.

### GWC benchmark scoring and launcher decomposition follow-up

- Extended `gwc bench` with package-level parallelism, checked-in `docs/benchmarks/reference.json` baselines, and reference-normalized geometric bucket scores so benchmark runs keep their raw `ns/op`, `B/op`, and `allocs/op` data while also emitting a stable summary score for regression tracking.
- Split the oversized `tools/gwc/main.go` launcher file into command-specific modules such as `dev.go`, `doctor.go`, `examples.go`, and `start.go`, and added `tools/gwc/docs/README.md` so future launcher work has a clearer ownership map.
- Aligned the launcher browser lane with the checked-in Playwright-Go package layout so browser test workspace discovery and the focused `tools/gwc` tests continue to match the repo-owned runner contract.

### Model picker persistence hardening in Example 100

- Fixed `examples/100-ai-chat-wizard` model selection persistence so new chats no longer land in a blank provider/model state after reconnects or stale cached preferences.
- Added client-side recovery for invalid or blank persisted model selections, with deterministic fallback to the first available catalog model and persisted repair back to the server.
- Added server-side `GetSelectedModel` fallback repair and persistence so normalized model preferences stay stable across sessions.
- Added focused regression coverage for selection-repair behavior in both client wasm tests and server RPC tests.

### Example 100 speech synthesis fallback and opt-in flow

- Added an OpenAI TTS-only fallback path in `examples/100-ai-chat-wizard` so speech playback can be enabled without switching the active chat provider or selected model for normal text generation.
- Added client-side helper coverage for selecting a speech-capable OpenAI synthesis model, plus UI copy and modal-flow updates that describe the new opt-in behavior explicitly.
- Removed the older model-preferences speech-provider switch path and centralized speech-model resolution inside the TTS controller instead of mutating the user’s saved chat provider preference.

### Playwright-Go rollout and Example 100 runtime hardening

- Replaced the repo and examples browser npm entrypoints, release smoke steps, and compatibility workflow with Go-based `playwrightgo` suites, and updated the browser-support and runner-config docs to match the new browser-workspace contract.
- Hardened `examples/100-ai-chat-wizard` memory extraction by sending a strict OpenAI JSON-schema response request, adding lifecycle and save-failure logging around extraction, and tightening provider HTTP coverage for the structured payload.
- Exposed a configurable usage-premium percentage in the Example 100 chat bootstrap script so client-side surfaces can read the server-defined premium multiplier during startup.
- Added account-level cost aggregation in the Example 100 client so the composer can show premium-adjusted account totals with coverage-gap handling on top of the existing per-thread cost summary.

### Tooling, devtools, and testing expansion wave

- Expanded the `gwc` launcher with dashboard/start TUI work, broader release/build/import/start/verify coverage, extracted `tools/runnerconfig` docs and tests, and new starter golden fixtures for standalone, contributor-linked, browser-test, and SSR app templates.
- Expanded `devtools` with bug-capture, support-bundle, trace-capture, serialization-boundary, extension-section, and richer runtime snapshot plumbing, backed by substantial wasm/native test coverage.
- Broadened Example 100, Atlas Commerce OS, SSR, interop, plugin, PWA, router, UI, and runtime coverage with a large set of focused regression tests, plus new public `test/browser` helpers and additional `testkit` consumer fixtures.
- Added and documented new example coverage for static islands, staged rollout config, SSR route helpers, virtualization, and the repo examples site while refreshing ecosystem, IDE, testing, and runner-config docs.
- Removed checked-in generated coverage/build artifacts from the tree and tightened ignore rules so local coverage output no longer pollutes status or release-oriented workflows.

### API naming and launcher workflow consolidation

- Aligned public API naming across `ui`, `utils`, `fetch`, `hotreload`, `logging`, and `devtools`, including test and example call-site updates and refreshed `docs/PUBLIC_API_CONVENTIONS.md` guidance.
- Extended launcher-owned tooling with stronger `runnerconfig` coverage and a new `gwc files` surface for filtered project file inventory output with repeatable extension and directory filters.

### Bench coverage and benchmark reporting

- Added micro-benchmark coverage across core and companion packages (including `devtools`, `diagnostics`, `fetch`, `head`, `hotreload`, `html`, `i18n`, `interop`, `logging`, `plugin`, `prerender`, `pwa`, `router`, `state`, `ui`, `utils`, `virtualization`, `testkit`, and `tools/runnerconfig`) to keep benchmark regressions visible in routine development.
- Added `gwc bench` to discover benchmark-bearing packages, run both native and `js/wasm` lanes, and write machine-readable benchmark snapshots to `docs/benchmarks/latest.json` with baseline-to-current delta comparison support.
- Documented the launcher benchmark workflow in `docs/PERFORMANCE.md` and `tools/README.md`, including lane selection and JSON output behavior for automation and CI.

### Example Playwright and chat-wizard follow-ups

- Relocated example-specific Playwright configs under `test/playwright/examples`, updated `examples/package.json` scripts, and refreshed docs/manual-smoke references so example test entrypoints resolve from one centralized config location.
- Refined Example 100 chat-wizard desktop UI spacing and control styling (composer controls, panel density, sidebar row emphasis, and message-meta contrast) for cleaner day-to-day operator ergonomics.

### Optional Playwright-Go smoke lane

- Added a build-tagged `test/playwrightgo` Chromium smoke test that validates the `playwright-go` driver/browser lifecycle and a minimal page interaction path.
- Added the required Playwright-Go module dependencies so the smoke lane can be enabled explicitly without changing default repo test behavior.

## 2026-03-24

### Example 100 AI chat wizard and supporting tooling

- Added a new `examples/100-ai-chat-wizard` full-stack showcase with a Go `js/wasm` client, Go server, gRPC-over-WebSocket bridge, SQLite-backed auth and conversation persistence, per-user settings and memories, streamed assistant replies, worker-backed markdown rendering, and focused Playwright/manual test support.
- Added richer chat-product behavior across the example, including reconnect and idle-resume handling for the gRPC bridge, collapsible assistant thinking sections, per-message and per-thread token-cost display, Mermaid and KaTeX rendering in message bubbles, TTS playback wiring, smoother streaming scroll behavior, and a stronger wasm/bootstrap loading shell.
- Added a canvas-style code workspace path for previewable assistant artifacts with split chat/canvas layout, canvas session state, focus-region editing, patch history, console capture, overlay mode, and a dedicated canvas-only route.
- Moved the example build output and runtime data into `examples/100-ai-chat-wizard/bin/`, added dedicated build/run scripts, runtime SQL-file loading, local Tailwind build support for the example shell, and follow-up cleanup so legacy source-tree wasm outputs are removed instead of accumulating under `client/`.

### Browser interop worker and scroll primitives

- Expanded `interop` with safer JavaScript exception wrapping for `Value.Invoke(...)` and `Value.Call(...)`, preserving thrown browser errors as structured remote failures instead of surfacing vague panics or illegal-invocation behavior.
- Added explicit element scroll helpers through `Element.SetScrollTop(...)` and `Element.ScrollMetrics(...)` so apps can drive scroll containers directly without ad hoc `syscall/js` reads and writes.
- Added `NewGoWASMWorker(...)`, `CurrentWorkerScope()`, and `WorkerScope` helpers so Go-authored wasm workers can be bootstrapped from `wasm_exec.js` plus a wasm module URL while still using the existing worker message envelope and request/progress/result pattern.
- Added native/wasm coverage around worker bootstrapping, JS exception wrapping, receiver-bound method calls, and browser storage method invocation so the new interop helpers stay consistent across runtime and tests.

### Smaller framework and docs follow-ups

- Added `OnMouseUp(...)` support to the public `html` props surface and the shorthand/sugar helpers, with tests proving the prop is emitted and wrapped consistently with the other event helpers.
- Fixed wrapped panic unwrapping for uncomparable payloads so reported panic wrappers preserve original map-like payloads instead of losing the underlying deferred/runtime failure context during repeated wrapping.
- Added the `28-scaling-local-state-with-use-reducer` example and supporting docs so reducer-based local-state composition has a smaller focused teaching surface alongside the larger examples and workflow docs.

## 2026-03-22

### HTML authoring sugar, shorthand ergonomics, and browser validation

- Added the first additive `html` authoring sugar layer, including `Children(...)`, `Textf(...)`, `TextIf(...)`, `When(...)`, `ClassNames(...)`, `If(...)`, `IfElse(...)`, `Unless(...)`, `Map(...)`, `MapKeyed(...)`, `FlatMap(...)`, `FilterMap(...)`, `Join(...)`, `Maybe(...)`, `OrElse(...)`, `Coalesce(...)`, `Switch(...)`, `Case(...)`, `Default(...)`, `PropsOf(...)`, `WithProps(...)`, first-pass prop helpers, explicit event-wrapper helpers, and closure-scoped `Debounce(...)` and `Throttle(...)` helpers so primitive DOM composition can be lighter without replacing the stable typed builder surface.
- Added the companion `html/shorthand` package with mixed-argument host-tag wrappers such as `Div(...)`, `Button(...)`, and `Input(...)`, plus `FromProps(...)`, while keeping the existing typed `html.Div(...)` builder family compatibility-stable for existing `[]ui.Node` call sites.
- Extended SSR to render reactive text nodes correctly, broadened native, shorthand, and exact-markup test coverage for the html surface, and pushed html package coverage beyond the targeted threshold with parity checks against explicit builders.
- Updated package docs, API policy, roadmap tracking, and several examples to demonstrate both `PropsOf(...)`-style additive sugar and the companion shorthand path, including form-heavy, toggle, todo, and html-form example flows.
- Hardened the browser smoke-test harness by making the Playwright web-server port configurable, then validated the mounted html path with focused and broader Playwright suites plus a direct manual smoke pass covering input, form submit, effect mount or cleanup, todo creation, and fetch-driven rendering.

### Static HTML and JSX converter workflow

- Added a new `gwc import` launcher command that converts static `.html`, `.htm`, `.jsx`, and `.tsx` sources into a single inspectable `main.go` built from the public `html` builder API instead of generating an opaque scaffold or hidden transform output.
- Added focused parser and generation coverage in `tools/gwc/import_test.go`, together with persisted converter fixtures under `test/gwc-import-fixtures/` that exercise nested structures, custom tags, boolean and raw attributes, `data-*`, `aria-*`, style maps, and explicit rejection of dynamic JSX child expressions.
- Moved launcher-owned temporary output from the OS temp directory into `bin/tmp/` so release-test and converter-adjacent temporary artifacts stay rooted under the project tree.
- Added ignored repo-local converter preview outputs under `bin/converter/` and validated the catalog fixture visually by running both the original HTML and converted wasm app, capturing screenshots for each, and confirming the rendered output matched at the browser level.

### GWC launcher coverage and examples runtime

- Added broad launcher coverage for `tools/gwc`, including focused tests for `build`, `release`, `test`, `verify`, `examples`, project detection and config precedence, the `start` Bubble Tea scaffold flow, runner-config overrides, and launcher output helpers so the newer CLI surface is exercised as a maintained contract instead of only by ad hoc manual checks.
- Promoted the repo examples catalog toward a launcher-owned runtime by adding a dedicated `examples/gwc-examples-site` `js/wasm` application, generating `examples/static/catalog.json`, and replacing the older mostly static examples shell with a richer catalog experience that can load, filter, and persist catalog state from the served metadata.
- Updated the repo tooling docs to describe `gwc` as the canonical launcher for build, release, examples, dev, doctor, test, verify, and start workflows, including the current `gwc-runner.json` override points for enterprise-oriented path configuration.
- Hardened the browser-facing PWA and interop support layer with explicit wasm browser-global helpers plus expanded wasm and native coverage around persistence, installability, service-worker, and diagnostics paths so browser-only helpers behave consistently in test and runtime environments.

## 2026-03-21

### PWA helpers and durable offline persistence

- Added a new public `pwa` companion package with explicit web-app manifest helpers, installability observation, service-worker registration and update-lifecycle helpers, wasm-release-manifest-driven asset planning, Cache Storage coordination, and structured diagnostics so PWA behavior stays app-owned instead of hidden in the core rendering runtime.
- Added a first-class IndexedDB-first durable persistence seam through `interop.OpenPersistentStore(...)`, including typed JSON helpers, blocked-upgrade and quota-aware interop error codes, opt-in corruption recovery, and focused native and `js/wasm` coverage for IndexedDB and fallback-storage behavior.
- Extended `fetch` with opt-in durable shared-cache persistence through `CacheOptions.Persist` and `fetch.ConfigurePersistentCache(...)`, and moved the offline mutation queue onto the same IndexedDB-first persistence boundary while preserving explicit fallback storage control.
- Extended `state` with `SavePersistentSnapshot(...)`, `LoadPersistentSnapshot(...)`, and `RestorePersistentSnapshot(...)` so atom snapshots can survive reloads through the same durable browser-storage boundary used by cache and queue helpers.
- Added runnable PWA examples for installability and offline cache or replay behavior, plus focused Playwright coverage for installability state, service-worker update flow, offline shell warmup, stale cache cleanup, queued write replay, offline navigation fallback, and cross-tab coordination.
- Extended the offline PWA slice with app-owned Background Sync registration through `pwa.ServiceWorkerRegistration.RegisterSync(...)`, conflict-aware offline replay via `fetch.ReplayWithOptions(...)` plus `fetch.NewMutationConflict(...)`, and example coverage for graceful Background Sync fallback plus conflict requeue resolution.
- Expanded the platform docs around PWA scope, Cache Storage policy, offline mutation retention, cross-tab replay ownership, release-manifest reuse, security posture, manual testing, and backlog status so the new offline behavior is documented as an explicit public contract.

### Unified test runner and harness stabilization

- Added a canonical repo-root `npm test` entrypoint that runs the main native Go suite, the `js/wasm` Go lane, the nested `tools/livereload` Go tests, the primary Playwright workspace under `test/`, and the aggregated Playwright suites under `examples/` from one orchestrated runner.
- Added `scripts/run-main-tests.mjs` plus focused root scripts for each lane so contributors can run the full matrix or isolate one lane without manually hopping across workspaces.
- Hardened the Windows-oriented harness flow by scrubbing leaked `GOOS` and `GOARCH` values for native tools, invoking nested npm workflows through the active Node/npm runtime, and making Playwright worker counts configurable from the environment for stability.
- Folded more example coverage into the main path by aggregating dedicated example suites such as SSR server-routing, Atlas SSR, and startup experiments under the examples workspace `test:all` script.
- Stabilized the selective hot-reload Playwright coverage by using ephemeral ports, draining dev-server output, detecting early startup exits, clearing stale wasm artifacts before startup, and waiting for a single remounted changed subtree before asserting preservation behavior.
- Simplified one brittle browser stress assertion so the test continues to validate user-visible state correctness without depending on incidental render-count behavior.

### Wrapped panic contract and actionable runtime diagnostics

- Added a unified wrapped-panic reporting path across runtime render, event, effect, cleanup, loader, hydration, startup, deferred, and SSR failure phases, keeping the original panic payload while attaching stable framework codes, `where:` context, component or route `path:`, plain-language runtime consequences, remediation guidance, and grouped app/framework/platform stack sections.
- Added structured panic reporting hooks and browser-console emission support so wrapped fatal panics can be mirrored consistently into runtime diagnostics, buffered logs, devtools snapshots, and browser-observed Playwright assertions instead of appearing as raw low-context Go panic output.
- Expanded actionable framework misuse messages for hooks, atom access, `ui.CreateElement(...)`, server-only UI APIs, router registration, and nil-context misuse so invalid public API usage now follows the same stable diagnostic contract instead of ad hoc panic strings.
- Added broad runtime coverage for wrapped panic formatting, suppression behavior, recovery-versus-rethrow policy, SSR panic handling, hydration and deferred panic reporting, and propagation into devtools snapshots and browser-visible logging.
- Updated docs with a formal fatal panic log contract, new actionable error codes for server and tool failures, and explicit recovery-versus-fatal rules so future diagnostic changes stay aligned across runtime, docs, and tooling.

### First public testing companion surface

- Added `docs/TESTING.md` to define the intended first-party testing surface as a companion module made of focused helper packages instead of one monolithic core testing API.
- Added the first public `testkit/render` fixture for `js/wasm` tests with controlled mock-DOM rendering, rerendering, accessibility-first role and name queries, lower-level id/text/tag queries, synthetic event dispatch helpers, and deterministic scheduler settlement helpers.
- Added `testkit/hooks` with a lightweight `RenderHook(...)` harness so hook-driven state flows can be exercised without writing a bespoke host component for every wasm-side test.
- Added `testkit/router` with hash and history fixtures that support route registration, initial path setup, navigation, params and query inspection, and delegated rendered-route assertions from public APIs.
- Added `testkit/ssr` with snapshot helpers, typed bootstrap-payload assertions, and a lightweight hydration smoke harness so SSR and hydration paths can be tested from public entrypoints instead of repo-local fixtures.
- Added consumer-oriented example tests and direct coverage for each of the new testkit packages so the shipped surface is documented by working test patterns rather than only by API signatures.

### Example and tool error reporting

- Added a small public `diagnostics` package over the internal reporting helpers so example servers and tools can emit the same structured actionable reports used by the runtime.
- Updated several SSR and server-integrated examples to return structured request and startup failures with stable codes, runtime consequences, remediation guidance, and docs anchors instead of plain `panic(...)` or raw `http.Error(...)` responses.
- Updated the live-reload tool to emit structured diagnostics for watcher failures, websocket delivery problems, client-script injection errors, manifest generation issues, rebuild failures, and startup problems so hot-reload breakage is easier to interpret from the terminal and browser.
- Extended devtools snapshot structures and UI summaries to retain wrapped panic metadata such as top frame and runtime consequence, keeping in-app inspection aligned with the new fatal panic contract.

## 2026-03-20

### SSR observability hooks

- Added a focused SSR observability surface in `ui` through `ObserveSSR(...)` plus observed render and bootstrap helpers so applications can capture request-level render timing, bootstrap payload sizes, and inline bootstrap script sizes without adopting a framework-specific logger backend.
- Extended browser hydration instrumentation to report per-hydration duration, existing DOM counts, mismatch counts, fallback counts, and discarded-node counts, with optional correlation ids threaded from `ui.Hydrate(...)` options into the emitted observation.
- Added native and runtime coverage that verifies render observations, bootstrap size metrics, and hydration fallback summaries are emitted from the shipped public surface.

### State transfer contracts and update envelopes

- Extended `ui.SSRBootstrap` and `ui.SSRBootstrapReference` with explicit schema versions and decode-time validation so newer payload schemas are rejected instead of being silently misread by older clients.
- Added typed state-transfer helpers in `ui` for route data, form defaults, cache seeds, session hints, scoped payload inspection, and non-JSON-friendly payload encoding for bytes, time values, text-marshaled ids, and explicit CBOR payloads.
- Added `ui.SSRStateUpdate` plus JSON and CBOR encode/decode helpers so post-hydration server-to-client payload refreshes can use versioned text or binary update envelopes instead of replacing the full bootstrap payload.
- Added `ui.AnalyzeSSRBootstrapSize(...)` so applications can measure inline JSON, inline script, and CBOR payload sizes and choose between inline, sidecar JSON, or sidecar CBOR transport with explicit warning bands.

### Wasm build experiment tooling and release-size comparison

- Added repeatable wasm build experiment helpers for phase-attributed timing, cache-topology comparison, CI-friendly manifest comparison, and explicit Go toolchain comparison so build-speed and artifact-size decisions can be made from saved JSON results instead of ad hoc shell timings.
- Added a stable representative target set for small, routed mid-sized, and large wasm applications, together with updated build experiment docs and completed backlog tracking for the current measurement surface.
- Extended wasm compression and release tooling with a Node-based Brotli fallback plus `npx --package binaryen wasm-opt` support so gzip, brotli, and optimized-wasm variants can be measured on standard contributor machines even when the host PowerShell runtime lacks Brotli support or `wasm-opt` is not installed globally.
- Recorded the current accepted and rejected build-optimization outcomes in the docs, including the stripped release baseline, gzip release sidecars, cache-behavior guidance, and the still-not-default status of Brotli and `wasm-opt` until the saved comparison results justify promoting them.

### Fine-grained reactivity and subscribed-region updates

- Added explicit fine-grained subscribed-region support on top of the existing fiber runtime through reactive text nodes, general reactive regions, selector-backed shared projections, and transition-aware shared-state update paths.
- Narrow subscribed updates can now update stable text nodes, host properties, and small anchored host-only subtrees without rerendering the owning component, while hook-driven rerenders still take precedence when mixed ownership would otherwise be ambiguous.
- Expanded runtime inspection and devtools snapshots with fine-grained fiber counts, granular mark and commit counters, and per-node update-origin or reactive-source metadata so narrow updates are visible in diagnostics instead of appearing as generic dirty work.
- Added deeper runtime, wasm, SSR, benchmark, and Playwright coverage for subscribed-region behavior, selector stability, transition deferral, hydration reuse, ancestor-rerender costs, and browser-level non-rerender guarantees.
- Added `docs/FINE_GRAINED_REACTIVITY.md` plus related roadmap and performance updates so the current mixed-model contract, measurements, and limits are documented as part of the repo history.
- Followed up with targeted rerender-path optimization passes that removed the measured ancestor-rerender overhead for the current stable-region benchmark shape by reusing unchanged reactive-source bookkeeping and redirecting stale subscribed fibers to their live fine-grained twin instead of transferring ownership during every clean clone.
- Expanded profiling and devtools visibility further with descendant host and descendant text commit counters inside fine-grained regions, plus regression coverage that proves stale subscribed twins still resolve to the live current region after ancestor rerenders.

## 2026-03-18

### Browser interop and multipart workflows

- Added a new public `interop` package with typed wrappers for browser storage, history, location, clipboard, timers, custom and generic event listeners, document lookup, element handles, resize observers, intersection observers, media queries, and dynamic module import.
- Added structured interop error handling through `interop.Error`, `IsCode(...)`, `AsError(...)`, and `CodeOf(...)`, together with native and `js/wasm` test coverage for browser-only failure paths and wrapper behavior.
- Added multipart request support to `fetch` through `MultipartBody`, `MultipartFile`, and `Upload(...)`, including progress updates, cancellation, preserved response status and headers, and structured HTTP error reporting for non-success responses.
- Added public browser file helpers in `ui` so file inputs and upload flows can work through typed `ui.File` wrappers instead of raw `syscall/js` values.

### SSR forms and server-integrated application flows

- Added the `87-ssr-secure-forms` example to demonstrate request-time SSR form rendering, CSRF-aware form posting, multipart upload validation, server round-trips that preserve user input, and post-submit `303 See Other` redirects.
- Added stronger `ui` form support for server-driven workflows, including server-error shaping, request-targeted submit modes, CSRF naming helpers, and typed file helpers on both browser and native targets so SSR form code no longer has to hand-roll those pieces.
- Added focused tests and microbenchmarks for secure SSR form rendering and multipart upload handling, plus wasm-side fetch and interop benchmarks for multipart form-data creation and browser interop calls.
- Added the first Atlas Commerce OS server-integrated workflow pass with request-time rendering, mock session routing, CSRF protection, SQLite-backed read and mutation flows, bootstrap payloads, startup migrations, and dedicated SSR browser coverage.

### Documentation and adoption guides

- Added `docs/START_HERE.md`, `docs/FORMS.md`, `docs/WORKFLOWS.md`, `docs/WALKTHROUGHS.md`, `docs/TROUBLESHOOTING.md`, and `docs/REFERENCE_MAP.md` to give new adopters a task-oriented route through the current public package surface.
- Expanded the docs index and migration guidance so the current public package surface, SSR and form workflows, troubleshooting path, and reference-app adoption path are linked from one consistent set of entry documents.
- Added dedicated Atlas Commerce OS documentation, layout maps, screenshots, and server notes so the reference application is documented as a real server-integrated workflow instead of only as runnable code.
- Added `interop/README.md` and refreshed `fetch` and docs index guidance so multipart upload, SSR form, and browser interop workflows are documented from the public API perspective instead of requiring commit archaeology.

### Browser platform integration and runnable examples

- Expanded `interop` with first-class worker helpers, typed worker request/progress/result envelopes, cross-tab channels with `BroadcastChannel` and `storage` fallback, popup/opener window channels, multi-surface signals, and additional typed browser wrappers so common browser coordination flows no longer require ad hoc `syscall/js`.
- Added `html.CustomElement(...)` and related custom-element guidance so browser-defined web components can be consumed with explicit attribute, presence-attribute, and property mapping instead of raw prop spreading.
- Added explicit mount-target APIs through `ui.RenderInto(...)` and `ui.HydrateInto(...)`, plus the `ui.UseWorkerTask[...]` hook for binding worker-backed browser jobs into normal component state.
- Added `examples/88-web-components`, `89-exported-custom-element`, `90-browser-interop`, `91-worker-text-index`, `94-cross-tab-sync`, and `95-multi-window-console` to demonstrate third-party custom elements, export-side web-component prototypes, typed browser interop, worker-backed CPU-heavy UI, cross-tab sync, and multi-window coordination.

### Cache reuse, auth-routing, and offline workflows

- Expanded `fetch.UseCachedResource[T]` with cache freshness and disposal policies, imperative `LoadCached(...)`, cache inspection, SSR bootstrap restore, resume-policy handling, and route-loader interoperability so shared data reuse works across component, loader, and SSR resume flows.
- Added a durable browser-backed offline mutation queue through `fetch.OpenMutationQueue(...)` with replay, deduplication, retry scheduling, dead-letter state, and wasm-side tests for replay and cancellation behavior.
- Added safe post-auth redirect helpers in `router` for preserving and validating internal `return_to` targets, and added the `92-protected-routes` and `93-ssr-cache-bootstrap` examples to demonstrate protected navigation, shared cache reuse, and SSR-seeded cache hydration.
- Added cache, offline mutation, auth-routing, observability, and server-integration documentation so the current routed-app and shared-data model is described as a public contract instead of only through tests and examples.

### Hydration diagnostics, runtime hardening, and devtools

- Hardened hydration and recovery with component-stack-aware mismatch diagnostics, opt-in strict hydration mode, preserved browser-owned form control state during the initial reuse pass, and explicit docs for hydration behavior, state transfer, streaming SSR boundaries, and mismatch recovery.
- Expanded runtime diagnostics to record structured classifications, buffered framework logs, and component path/stack context for recovered boundary errors, and surfaced those logs and diagnostics through the devtools snapshot and panel.
- Added broader runtime correctness and benchmark coverage, including production-correctness scenarios, hydration reuse and fallback benchmarks, and transition scheduling benchmarks, while also keeping the `production` wasm utils surface aligned with development exports.
- Refreshed `examples/27-transition-hooks` into a more realistic transition-style UX example and aligned the backlog/docs around the current runtime, scheduling, error-boundary, and hydration behavior.

### Tooling, release engineering, and platform policy docs

- Added `tools/build-wasm-release.ps1` and a sample wasm size-budget file so release-style builds can emit stripped artifacts, gzip sidecars, hashes, manifests, and optional size-budget enforcement from one helper.
- Added docs covering cache behavior, custom elements, interop, workers, cross-tab sync, multi-surface coordination, error boundaries, production correctness, scheduling, hydration, streaming SSR, server integration, observability, logging, prerender, assets, browser support, PWA boundaries, security, configuration, wasm releases, build experiments, and onboarding.
- Updated example indexes, manual-testing guidance, and docs entrypoints so the newer example set and system-level docs are discoverable from the top-level navigation.
- Ignored the repo-local `.gotmp/` Go temp/build directory so local release-build and wasm test artifacts stop appearing as worktree noise.

## 2026-03-16

### Internationalization and localization

- Added a new public `i18n` package with `UseLocale(...)`, `Provider(...)`, `UseI18n()`, deterministic bundle registration, missing-message fallback behavior, interpolation, pluralization, select-style branching, and locale-aware number or date formatting helpers.
- Added route-oriented helpers `PrefixPath(...)` and `ResolvePath(...)` so locale prefixes can stay application-owned without pushing locale policy into `router` itself.
- Extended `ui.SSRBootstrap` with typed `I18n` payload support and bundle conversion helpers so server-rendered pages can transfer active locale, fallback locale, direction, and the initial message subset used during hydration.
- Added `docs/I18N.md` to define the current i18n scope, SSR transfer model, locale-aware routing guidance, and RTL or directionality expectations.
- Added `examples/83-locale-switcher`, `examples/84-ssr-i18n-bootstrap`, and `examples/85-locale-routing` together with focused Playwright coverage for runtime locale switching, bootstrap-driven locale hydration, and locale-prefixed loader-driven routing.
- Added native `i18n` tests and a translation microbenchmark covering fallback lookup, pluralization, formatting, SSR bootstrap round-tripping, and locale-aware path helpers, plus `ui` bootstrap tests that assert the new `I18n` payload is serialized and initialized correctly.

### Portal layering and overlay management

- Added `ui.Overlay(...)` and `ui.UseOverlayStack(...)` for shared overlay layering, stack-derived z-order, nested escape and outside-click routing, and coordinated focus ownership across portal-backed surfaces.
- Updated `ui.AccessibleOverlay(...)` to compose on top of the shared overlay stack instead of wiring instance-local modal behavior independently.
- Updated overlay side effects to support nested scroll-lock counting and nested background inert ownership by app-root selector.
- Added `docs/OVERLAYS.md` to document the current overlay layering model, dismissal routing, and anchored-position guidance.
- Added `examples/81-overlay-stack` and `examples/82-overlay-anchor` together with focused Playwright specs covering nested dialogs, dialog-plus-popover routing, tooltip-over-menu layering, and portal retargeting.
- Added stack-manager coverage in `ui` tests so topmost escape handling, outside-dismiss routing, focus-trap ownership, and derived z-index behavior are validated directly.

### Accessibility guidance baseline

- Added `ui.UseFocusManager()`, `ui.UseFocusTrap(...)`, `ui.UseCompositeNavigation(...)`, `ui.UseAnnouncer()`, and `ui.AccessibleOverlay(...)` as the first public accessibility-focused primitives.
- Expanded `docs/ACCESSIBILITY.md` to document the shipped accessibility model for focus restoration, modal overlays, composite keyboard navigation, live-region announcements, forms, route changes, and async UI.
- Added baseline and API-level tests covering accessible prop preservation, `ui.UseId()` behavior, composite navigation keyboard flow, and live-region rendering.
- Added focused accessibility examples for modal overlays, composite widgets, form validation announcements, and routed page announcements, together with Playwright browser coverage.

### Head management and SEO guidance

- Added `docs/HEAD_MANAGEMENT.md` to define the current head-management model, including the boundary between router-managed metadata and application-owned explicit SEO markup.
- Documented canonical URL policy, structured-data guidance, resource-hint guidance, social metadata examples, and sitemap or robots integration expectations for the current SSR surface.
- Added tests covering managed head-tag deduplication after hydrated startup and SSR output checks that enforce a single managed title, description, and canonical tag set.

### Metadata model and composition policy

- Added `router.MetadataNode(...)` so route-managed title, description, and canonical tags can be rendered during SSR through `ui.RenderToString(...)`.
- Updated client router metadata reconciliation to treat SSR and client navigation as one managed metadata flow, reusing and cleaning up only router-owned head tags.
- Updated the SSR server-routing example to render managed metadata tags through the shared router helper instead of hand-built escaped strings.
- Documented the current composition decision that slots are out of scope in favor of ordinary children, explicit props, context, portals, and layout routes.

### API policy and migration guidance

- Added `docs/API_POLICY.md` to define public API stability tiers, semver expectations, deprecation timing, breaking-change rules, and current latest-major-only support policy.
- Added `docs/MIGRATIONS.md` as the release-to-release migration index, starting with guidance for moving into the current `v3.x` public package layout.
- Linked the new policy and migration docs from the root README and docs index.
- Updated the documentation backlog to mark the API stability and support policy work complete.

### Example catalog and backlog alignment

- Expanded the feature example catalog so the examples index and shared catalog copy map the shipped examples more directly to the current public APIs and learning goals.
- Refreshed the root README and docs backlog structure around the current public package surface, numbered roadmap sections, and completed-work tracking instead of leaving newer example and docs work scattered across older notes.
- Added the first Atlas Commerce OS planning backlog so the larger reference-application effort has an explicit timeline in the repo instead of only appearing through implementation commits later.

## 2026-03-15

### Runtime optimization work

- Reduced `GoUseFunc(...)` validation overhead with build-specific fast paths for common function signatures.
- Reduced `GoUseAtom(...)` overhead by storing atom values directly, adding a single-subscriber notification fast path, and caching getter/setter accessors per hook slot.
- Added keyed-reconciliation fast paths, pooled keyed scratch state, and lower-overhead keyed-child consumption to reduce keyed list churn.
- Cached DOM prop metadata in `updateDomProperties(...)` and added a steady-state DOM property benchmark path.
- Cached additional jsdom document query bindings and switched DOM collection traversal to indexed access instead of `item(...)` calls.
- Added runtime layout and keyed reconciliation benchmark coverage used to validate and reject broader cache-layout experiments.
- Reverted broader experiments that regressed the benchmark suite, including the hot/cold `Fiber` split and a dedicated non-batching initial DOM-props fast path.

### Runtime correctness fixes

- Fixed function-component deletion so removing one DOM-less component subtree does not incorrectly remove sibling component DOM.

### SSR and hydration groundwork

- Added the first internal SSR render-to-string path for host elements, text nodes, fragments, and simple function components.
- Added public server-side rendering support through `ui.RenderToString(...)` on non-browser targets.
- Added a dedicated `Hydrate(...)` and runtime `HydrateTo(...)` entrypoint so client resume now has a real API path instead of reusing plain render calls.
- Upgraded hydration from whole-container fallback to real DOM reuse for matching host and text nodes, with subtree-level fallback when structure matching fails.
- Added hydration mismatch diagnostics for text, tag/structure, trailing-node, and critical attribute differences, with clear warn-versus-replace behavior.
- Added bootstrap-time atom snapshot restore and ID-seed application so hydration resumes shared state and `UseId` generation from the transferred SSR payload.
- Deferred hydration-time atom subscriptions and follow-up update notifications until the hydration commit completes, then ran effects against the committed tree.
- Added safe SSR bootstrap helpers for inline JSON payloads, including script-tag rendering and browser-side bootstrap-script reading.
- Added optional CBOR-based binary bootstrap encoding/decoding for SSR payload transport.
- Added sidecar bootstrap reference support so hydration can load external JSON or CBOR bootstrap payloads instead of only inline script JSON.
- Added SSR/bootstrap and hydration tests covering DOM reuse, subtree fallback, text mismatch recovery, trailing-node cleanup, bootstrap atom restore, and ID-seed resume behavior.
- Added SSR transport microbenchmarks showing CBOR bootstrap encode/decode is materially cheaper than JSON for larger sidecar-style payloads in the current implementation.

### Public package and wasm tests

- Added wasm-facing public package tests for `html`, `ui`, `state`, and `fetch` wrappers.
- Expanded HTML builder coverage to verify public prop preservation, wrapper tag selection, and text/fragment helper behavior.
- Added public wrapper coverage for `state.UseComputed`, `fetch.UseResource`, `ui.UsePrevious`, `ui.UseChannel`, and `ui.UseTask`.
- Added public wrapper coverage for `ui.UseReducer`, `ui.UseForm`, `state.UseDerived`, and state snapshot persistence helpers.

### Public hooks and state ergonomics

- Added `state.UseComputed[T]` as a typed derived-state helper on top of the memoization runtime.
- Added `state.UseDerived[T]` as a read-only shared derived-atom helper with explicit source-atom dependencies.
- Added state snapshot export/import and browser storage persistence helpers for same-process restore and JSON-compatible saved state.
- Added `fetch.UseResource[T]` for typed async loading with cancellation, reloads, and typed ready/error state.
- Added `fetch.UseCachedResource[T]` with shared keyed cache state, in-flight request deduplication, stale-while-revalidate refreshes, invalidation, and optimistic update helpers.
- Added `ui.UsePrevious[T]`, `ui.UseChannel[T]`, and `ui.UseTask[T]` to cover render-time comparisons, channel-driven UI streams, and explicit cancellable background jobs.
- Added `ui.UseReducer`, `ui.UseForm`, `ui.UseDebounced`, and `ui.UseThrottled` to cover reducer-style local state, structured form lifecycle handling, and delayed/rate-limited derived UI values.
- Added `ui.StartTransition`, `ui.UseTransition`, and `ui.UseDeferredValue` as the first concurrent-style scheduling primitives for non-urgent tree refreshes and lagging derived values.
- Added `ui.AsyncBoundary`, `ui.UseLazyNode`, and `ui.Lazy` as the first explicit async UI primitives for fallback rendering, deferred subtree resolution, and timeout-aware loading boundaries.
- Added `ui.ErrorBoundary` as a runtime-recognized subtree recovery wrapper with fallback rendering, explicit reset support, and server/client parity for render-time panic recovery.

### Router fixes and tests

- Normalized router default routes, registration paths, and navigation targets so hash/history routing behaves consistently across trailing-slash and hash-prefixed inputs.
- Hardened browser-router setup against missing browser globals in wasm test environments.
- Added wasm browser-environment test helpers and broader router wasm test coverage for navigation, registration, and normalized default-route behavior.
- Added route patterns, typed `UseParams`, `UseQuery`, and `UseNavigate` helpers, plus query-aware path normalization for hash and history routing.
- Added route-level loaders with route-scoped loading/error renderers, cancellation on navigation changes, and route+query keyed revalidation.
- Added `UseRouteData`, `UseRevalidator`, and manual current-route revalidation support with router tests for params, query parsing, loaders, cancellation, and explicit reruns.
- Added route option support for titles, declarative redirects, synchronous `BeforeEnter`/`BeforeLeave` guards, and route-managed description/canonical metadata with cleanup on route changes.
- Added nested layout routes through `router.Options{Layout: true}` and explicit child rendering through `router.Outlet()`.
- Added nested route-stack resolution so parent layout routes render from shallowest matching prefix to the final leaf route, with parent params scoped per level and leaf routes receiving the final merged param set.
- Extended nested routing support across hash and history routers, including layout-aware guard evaluation and per-route loader-data scoping during nested rendering.
- Expanded router edge-case coverage for decoded params, unsupported optional-segment syntax, exact-vs-pattern precedence, blocked navigation, redirected navigation, and metadata replacement/cleanup semantics.
- Added nested layout route tests covering outlet rendering, per-level params, per-level route data, leaf-metadata precedence, nested redirects, `InspectCurrentRoute()`, and history-router nested guard behavior.
- Added router microbenchmarks for nested route-stack resolution, nested `Current()` rendering, and nested navigation evaluation baselines in the js/wasm test lane.

### Devtools and inspection

- Added a new public `devtools` package with runtime snapshot capture and an embeddable in-browser inspector panel.
- Added committed component tree inspection, hook summaries, route inspection, and runtime totals to the new devtools surface.
- Added structured diagnostics for invalid hook usage, missing render targets, duplicate route registration, and invalid route component configuration.
- Embedded the devtools panel into the `14-omi` showcase example.
- Added baseline profiling counters for renders, scheduled updates, work-loop passes, processed units, commits, and last render/commit timings.
- Added structured missing-key diagnostics for mixed keyed/unkeyed sibling lists.
- Added runtime error-boundary recovery for render, effect, cleanup, and event-handler panics together with diagnostics and follow-up rerender scheduling.
- Added a smaller standalone `16-devtools` example focused on the public devtools package.

### Example upgrades

- Reworked the goroutine example around `ui.UseTask` and `ui.UseChannel`, replacing manual task bookkeeping and adding a streamed channel-driven UI panel.
- Reworked the fetch example around `fetch.UseResource[T]` with typed list/detail loading, explicit retry controls, and cancellation.
- Reworked the fetch example further around `ui.AsyncBoundary` and `ui.Lazy`, replacing manual loading-branch plumbing with explicit async subtree boundaries and a deferred auxiliary panel.
- Reworked the fetch example again around `fetch.UseCachedResource[T]`, shared query reuse, key-based invalidation, and optimistic detail/list updates.
- Updated the atoms example to demonstrate `state.UseComputed` for derived labels and summaries.
- Updated the text-input example to demonstrate `ui.UseDebounced` and `ui.UseThrottled` with visible delayed preview and throttled count feedback.
- Reworked the advanced-form example around `ui.UseForm`, async validation, retryable submission, and router-driven success flow.
- Expanded the OMI example with nested route navigation, loader-backed detail/search/protected routes, and manual route revalidation controls.
- Added the `19-nested-routes` example to demonstrate dashboard and docs layout shells, nested settings layouts, explicit outlet composition, and nested param-driven leaf routes.

### Documentation refresh

- Aligned the `fetch` package docs and examples around `UseFetch` as the low-level raw hook and `UseResource[T]` as the preferred typed async resource API.
- Extended the `fetch` docs to cover `UseCachedResource[T]`, shared cached queries, optimistic updates, and async-boundary integration.
- Rewrote the `router` package docs to match the current public API, including params, query helpers, loaders, and manual route revalidation.
- Expanded router docs to cover layout-route registration, `router.Outlet()`, nested route matching rules, and per-layout route-data behavior.
- Refreshed portfolio and showcase copy to use the current `UseState`/`UseEffect`/`UseMemo` and `UseFetch`/`UseResource` naming instead of stale legacy names.
- Expanded `state` and `ui` docs to cover typed computed state, previous-value tracking, channel subscriptions, and cancellable tasks.
- Documented the transition/deferred-value scheduling model and the current decision to avoid a dedicated layout-effect hook until concrete DOM-read-before-paint cases appear.
- Expanded `ui` docs further to cover the new `ErrorBoundary` contract, fallback callbacks, and explicit reset behavior.
- Added docs for shared derived state, snapshot persistence, route guards, route-managed metadata, reducer/form helpers, and debounced/throttled UI hooks.
- Updated the backlog and examples index to mark nested routes and layout routes complete and list the new multi-level example.
- Expanded the root README with a project-wide feature inventory, clarified the stable public package surface, and brought the shipped examples list up to the current `16` through `19` demo set.
- Refreshed the SSR demo and README copy so the examples describe the shipped hydration behavior accurately: bootstrap restore, matching DOM reuse, and subtree fallback on structural mismatch.

### Browser tests and benchmarks

- Expanded Playwright integration coverage for filtered todo flows, effect cleanup under broader app activity, and mixed local/shared-state burst interactions.
- Added Playwright coverage for debounced/throttled text input behavior, advanced-form validation/retry/success flow, and the standalone devtools example.
- Added jsdom wasm benchmarks for `GetElementById(...)` and `QuerySelectorAll(...)` and refreshed the React-vs-Go browser benchmark run.

## 2026-03-14

### Runtime fixes

- Fixed function-component reconciliation so components that return `nil` correctly delete previously rendered children.
- Fixed DOM-less subtree deletion so all descendant sibling DOM branches are removed during unmount.
- Fixed stale hook ownership on reused/skipped function components so later stateful updates target the correct fiber.
- Fixed `GoUseState` so nil-able state types can be reset with `set(nil)`.
- Fixed `GoUseAtom` so nil-able atom state types can be reset with `set(nil)`.
- Fixed `GoUseAtom` initialization on fresh component fibers so global atom hooks work on first use.
- Fixed scheduler update propagation so clean ancestors are marked even when a leaf is already dirty.
- Fixed global runtime initialization so `InitGlobalRuntime()` can upgrade a lazily created global runtime.
- Fixed HTML component helper wrappers so component refs are wrapped as component elements instead of being passed as raw children.
- Removed the artificial `50ms` delay from the wasm fetch hook.
- Removed runtime debug logging from the wasm fetch hook.
- Completed the runtime event wrapper so `GoEvent` exposes target and key-code access and safely handles missing event methods.
- Optimized `Text()` shim output to use the text-element layout directly.
- Normalized the native fetch stub error and removed per-call closure allocation from the stub implementation.

### Runtime optimization work

- Added work-in-progress fiber reuse through alternates in the reconciler and scheduler.
- Added batched atom unsubscription with `UnsubscribeMany(...)` to reduce lock churn during cleanup.
- Optimized component helper child creation in `html.go` to avoid unnecessary wrapper overhead.
- Specialized `WASMDOMAdapter.WrapFunction` so callback dispatch type selection happens once at wrap time instead of on every invocation.
- Kept only measured low-risk optimizations and reverted regressions found during benchmark runs.

### Runtime tests

- Added contract-focused reconciler tests.
- Added extra hooks contract and edge-case tests.
- Added scheduler contract and edge-case tests.
- Added atom state contract and cleanup tests.
- Added runtime global-init and render contract tests.
- Added HTML helper contract tests.
- Added wasm fetch hook tests.
- Added wasm event adapter tests.
- Added shim tests for global wrapper behavior.
- Added interfaces contract tests and compile-time adapter assertions.
- Added exhaustive native runtime coverage tests and raised native `internal/runtime` statement coverage to `100%`.

### Browser component and integration tests

- Added Playwright component tests for local state isolation, rerender stability, effect cleanup, and `UseId` behavior.
- Added Playwright integration tests for todo flows, shared atom propagation, theme updates, fetch flows, and form handling.
- Added deep state stress tests covering `5+`, `25+`, `100+`, repeated burst updates, mixed local/shared state, and rerender persistence.
- Added test app components to exercise stress and mixed-state scenarios.
- Added mock API and test server endpoints needed by the browser suites.

### Benchmarks

- Added runtime microbenchmarks for reconciler hot paths.
- Added hooks microbenchmarks.
- Added scheduler microbenchmarks.
- Added atom state microbenchmarks.
- Added runtime wiring microbenchmarks.
- Added HTML helper microbenchmarks.
- Added shim microbenchmarks.
- Added native fetch stub microbenchmarks.
- Added `types.go` microbenchmarks.
- Added wasm adapter boundary benchmarks for DOM operations and callback wrapping.

### Benchmark tooling and reporting

- Fixed the browser benchmark harness and added a runtime update regression test to catch render/update performance regressions.
- Added benchmark build and inspection scripts for the browser benchmark suite.
- Documented the current browser benchmark results in the project documentation.

### Public API and examples

- Introduced the typed `ui` and `html` public APIs and started moving examples onto the new surface.
- Added the `14-omi` showcase example.
- Added the `15-calculator` interactive calculator example with engine, UI, and presets.
- Updated the examples index and static landing page wiring for the new showcases.

### Documentation refresh

- Refreshed the root README to reflect the current runtime architecture, test status, public package guidance, and install/build instructions.
- Updated browser compiler and project docs to match the current repo layout and tooling.

### Tooling and developer workflow

- Rebuilt the development server around Node/Express and saved it in `tools/dev-server/`.
- Added dynamic `/examples` listing based on the filesystem instead of hard-coded routes.
- Updated server scripts and tooling docs to use the new development server.
- Added a wasm test execution helper in `tools/go_js_wasm_exec.bat`.
- Added new test scripts for component, integration, and state browser test layers.
- Added a GitHub Pages deployment workflow for the examples site and a generated redirect page for the published showcase.
- Upgraded CI workflows to current `checkout`, `setup-go`, and `setup-node` action majors.
- Upgraded CI to Go `1.25.4` and Node `24`.
- Stabilized flaky Playwright hook/browser tests by replacing transient timing assertions with stable result checks and stronger selectors.
- Tracked the Playwright test lockfile, added npm caching for the test package, and split CI dependency install, browser install, and test execution into separate steps.
- Fixed automated release version calculation so semantic releases continue from the latest `vX.Y.Z` tag instead of resetting to `v0.0.x`.

### Validation status

- `go test ./internal/runtime` passes.
- Native `internal/runtime` statement coverage is `100%`.
- Browser component, integration, and deep state stress suites pass.
- Wasm runtime and adapter tests pass in the separate `js/wasm` test lane.
- The examples site deploys through the GitHub Pages workflow.
- The `CI + Release` workflow passes on `master`.
- Automated semantic release tagging resumed and published `v3.0.3`.

## 2025-11-26

### Browser compiler experiment

- Moved the in-browser compiler experiment into its own more self-contained example flow under `examples/13-browser-compiler`.
- Added the supporting browser-compiler scripts, package index generation flow, and bundled `js/wasm` standard-library assets needed to run that experiment from the repo.

## 2025-11-19

### Runtime architecture and performance

- Reworked the runtime architecture and aligned the examples with the updated rendering and hook model.
- Optimized reconciliation and rendering hot paths across the runtime and jsdom adapter layers.
- Fixed the goroutines example so background-driven UI updates render correctly.

### Benchmarks and test structure

- Added a browser performance benchmark app and Playwright benchmark coverage, and restructured the broader test layout around that workflow.
- Expanded runtime and reconciliation coverage while tightening test expectations around hook/fiber behavior.
- Fixed advanced hash-router test behavior around navigation history and visibility-driven cases.

### CI and release workflow

- Added the initial CI/CD workflow and refreshed the project README with workflow badges and updated project assets.
- Iterated on CI reliability by excluding unsuitable benchmark/unit-test paths, switching browser-test dependency install behavior, and wiring E2E web-server startup on port `8081`.
- Scoped CI browser coverage away from portfolio-site cases that required a separate examples server.

### Examples, docs, and site polish

- Refined the portfolio-site presentation, including the 3D showcase card rendering issues and updated hero-image assets.
- Added license and example coverage for pkg.go.dev across the public packages.

## 2025-11-18

### WASM runtime and fetch groundwork

- Added early WASM runtime improvements centered on a new fetch hook path and lower-level function-wrapping support.
- Updated the jsdom/mockdom adapter and runtime shim layers to support that browser-focused hook flow.
- Added early browser test coverage for events, hooks, and reconciliation rerender behavior using the test app fixtures.

### Documentation cleanup

- Cleaned up the root README formatting and readability before the larger runtime and tooling changes that followed.
