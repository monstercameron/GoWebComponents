# Changelog

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
- Added hydration preflight/fallback behavior that inspects existing container DOM, reports diagnostics, clears server markup on fallback, and then schedules a fresh client render while true DOM matching remains under development.
- Added safe SSR bootstrap helpers for inline JSON payloads, including script-tag rendering and browser-side bootstrap-script reading.
- Added optional CBOR-based binary bootstrap encoding/decoding for SSR payload transport.
- Added sidecar bootstrap reference support so hydration can load external JSON or CBOR bootstrap payloads instead of only inline script JSON.
- Added SSR bootstrap tests covering escaping, JSON decode defaults, binary round-trips, reference scripts, wasm-side sidecar loading, and hydration fallback behavior.
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
- Added `ui.UsePrevious[T]`, `ui.UseChannel[T]`, and `ui.UseTask[T]` to cover render-time comparisons, channel-driven UI streams, and explicit cancellable background jobs.
- Added `ui.UseReducer`, `ui.UseForm`, `ui.UseDebounced`, and `ui.UseThrottled` to cover reducer-style local state, structured form lifecycle handling, and delayed/rate-limited derived UI values.
- Added `ui.AsyncBoundary`, `ui.UseLazyNode`, and `ui.Lazy` as the first explicit async UI primitives for fallback rendering, deferred subtree resolution, and timeout-aware loading boundaries.

### Router fixes and tests

- Normalized router default routes, registration paths, and navigation targets so hash/history routing behaves consistently across trailing-slash and hash-prefixed inputs.
- Hardened browser-router setup against missing browser globals in wasm test environments.
- Added wasm browser-environment test helpers and broader router wasm test coverage for navigation, registration, and normalized default-route behavior.
- Added route patterns, typed `UseParams`, `UseQuery`, and `UseNavigate` helpers, plus query-aware path normalization for hash and history routing.
- Added route-level loaders with route-scoped loading/error renderers, cancellation on navigation changes, and route+query keyed revalidation.
- Added `UseRouteData`, `UseRevalidator`, and manual current-route revalidation support with router tests for params, query parsing, loaders, cancellation, and explicit reruns.
- Added route option support for titles, declarative redirects, synchronous `BeforeEnter`/`BeforeLeave` guards, and route-managed description/canonical metadata with cleanup on route changes.
- Expanded router edge-case coverage for decoded params, unsupported optional-segment syntax, exact-vs-pattern precedence, blocked navigation, redirected navigation, and metadata replacement/cleanup semantics.

### Devtools and inspection

- Added a new public `devtools` package with runtime snapshot capture and an embeddable in-browser inspector panel.
- Added committed component tree inspection, hook summaries, route inspection, and runtime totals to the new devtools surface.
- Added structured diagnostics for invalid hook usage, missing render targets, duplicate route registration, and invalid route component configuration.
- Embedded the devtools panel into the `14-omi` showcase example.
- Added baseline profiling counters for renders, scheduled updates, work-loop passes, processed units, commits, and last render/commit timings.
- Added structured missing-key diagnostics for mixed keyed/unkeyed sibling lists.
- Added a smaller standalone `16-devtools` example focused on the public devtools package.

### Example upgrades

- Reworked the goroutine example around `ui.UseTask` and `ui.UseChannel`, replacing manual task bookkeeping and adding a streamed channel-driven UI panel.
- Reworked the fetch example around `fetch.UseResource[T]` with typed list/detail loading, explicit retry controls, and cancellation.
- Reworked the fetch example further around `ui.AsyncBoundary` and `ui.Lazy`, replacing manual loading-branch plumbing with explicit async subtree boundaries and a deferred auxiliary panel.
- Updated the atoms example to demonstrate `state.UseComputed` for derived labels and summaries.
- Updated the text-input example to demonstrate `ui.UseDebounced` and `ui.UseThrottled` with visible delayed preview and throttled count feedback.
- Reworked the advanced-form example around `ui.UseForm`, async validation, retryable submission, and router-driven success flow.
- Expanded the OMI example with nested route navigation, loader-backed detail/search/protected routes, and manual route revalidation controls.

### Documentation refresh

- Aligned the `fetch` package docs and examples around `UseFetch` as the low-level raw hook and `UseResource[T]` as the preferred typed async resource API.
- Rewrote the `router` package docs to match the current public API, including params, query helpers, loaders, and manual route revalidation.
- Refreshed portfolio and showcase copy to use the current `UseState`/`UseEffect`/`UseMemo` and `UseFetch`/`UseResource` naming instead of stale legacy names.
- Expanded `state` and `ui` docs to cover typed computed state, previous-value tracking, channel subscriptions, and cancellable tasks.
- Added docs for shared derived state, snapshot persistence, route guards, route-managed metadata, reducer/form helpers, and debounced/throttled UI hooks.

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
