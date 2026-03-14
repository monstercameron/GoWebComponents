# TODO

Last updated: 2026-03-14

This is the current backlog. Older versions of this file mixed historical work, completed items, abandoned ideas, and stale path names. Those details were removed.

## Current Priorities

### Runtime and reconciliation

- [ ] Add repeated benchmark comparison tooling for native runtime benchmarks
- [ ] Build better steady-state reconciliation/fiber reuse benchmarks
- [ ] Investigate keyed reconciliation support for larger dynamic lists
- [ ] Continue tightening event/property cleanup on DOM updates and unmount paths

### Hooks and state

- [ ] Decide whether `UseEffect` should keep the current variadic dependency semantics or be reshaped for clearer omitted-vs-empty behavior
- [ ] Add more browser-level tests for complex remount/reset semantics
- [ ] Evaluate whether the public `UseState` setter shape should stay `func(interface{})` or move toward a stricter typed API in a future breaking revision

### Router and platform features

- [ ] Review router ergonomics and docs for current browser/hash routing coverage
- [ ] Add more wasm-side integration tests around fetch + router + browser state interactions
- [ ] Profile more of the `internal/platform/jsdom` adapter boundary under realistic UI workloads

### Developer workflow

- [ ] Add CI steps for native runtime tests, wasm runtime tests, and Playwright suites
- [ ] Add Linux or WSL coverage reporting for wasm-only packages, since Windows toolchains are awkward for wasm coverage export
- [ ] Decide whether the older live reload tooling should stay experimental, be modernized, or be removed

### Documentation

- [ ] Keep package-level READMEs aligned with the current runtime and test layout
- [ ] Add a short contributor guide for native vs `js/wasm` test lanes
- [ ] Add a benchmark workflow doc once repeated comparison tooling is in place

## Completed Recently

- [x] Rebuilt the example dev server around Node/Express
- [x] Added dynamic `/examples` listing based on filesystem structure
- [x] Fixed major reconciler deletion and nil-render bugs
- [x] Fixed nil reset behavior for local and atom state
- [x] Added native runtime contract tests and microbenchmarks across the runtime core
- [x] Added wasm tests and benchmarks for fetch/runtime adapter paths
- [x] Added Playwright component, integration, and deep state stress suites
- [x] Reached native `internal/runtime` statement coverage of `100%`
- [x] Removed tracked generated package archives and other committed build artifacts
- [x] Refreshed stale top-level docs and tooling docs

## Out Of Scope For Now

These are not current working priorities and should not be treated as active roadmap commitments:

- Full React feature parity
- SSR/hydration
- Suspense/transition-style concurrency APIs
- Browser-hosted full Go compilation as a near-term product feature
- A large built-in component library
