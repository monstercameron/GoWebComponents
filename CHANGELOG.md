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

### Public package and wasm tests

- Added wasm-facing public package tests for `html`, `ui`, `state`, and `fetch` wrappers.
- Expanded HTML builder coverage to verify public prop preservation, wrapper tag selection, and text/fragment helper behavior.

### Router fixes and tests

- Normalized router default routes, registration paths, and navigation targets so hash/history routing behaves consistently across trailing-slash and hash-prefixed inputs.
- Hardened browser-router setup against missing browser globals in wasm test environments.
- Added wasm browser-environment test helpers and broader router wasm test coverage for navigation, registration, and normalized default-route behavior.

### Browser tests and benchmarks

- Expanded Playwright integration coverage for filtered todo flows, effect cleanup under broader app activity, and mixed local/shared-state burst interactions.
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
