# Testing Surface

This document defines the current first-party testing surface for GoWebComponents consumers.

## Current Status

Shipped today:

- preferred consumer import paths under `test/render`, `test/hooks`, `test/router`, and `test/ssr`
- compatibility aliases under `testkit/...` for existing consumers
- browser-oriented public helpers for render, hooks, router, and hydration smoke coverage
- repo-level validation through native Go tests, js/wasm tests, the `test/` Playwright workspace, and the aggregated example suites
- launcher lanes through `go run ./tools/gwc test -lane ...` for unit, wasm, hydration, browser, and release checks

Not shipped today:

- a custom framework-owned test runner
- one monolithic testing package that hides all setup behind a single import
- a first-party replacement for Playwright-style end-to-end browser automation

## Decision

Testing support should be first-party, but it should not live inside the core runtime packages.

The intended shape is one companion testing module with several focused helper packages instead of one monolithic package.

Recommended package split:

- `test/render`: component mount, DOM fixture ownership, queries, event dispatch, and deterministic flush helpers
- `test/hooks`: hook-host utilities for custom hooks and hook-driven state flows
- `test/router`: initial-route setup, navigation, params, query, guard, and loader assertions
- `test/ssr`: SSR snapshots, bootstrap assertions, and hydration smoke helpers
- `test/browser`: optional bridges for browser-runner setup when consumer tests want thin helpers over Playwright or similar tools

The root companion module may expose only shared low-level pieces that the focused packages need, such as fixture lifecycle or scheduler-settlement helpers. Consumer-facing testing APIs should stay in the focused packages so imports remain explicit and semver changes stay easier to reason about.

The preferred import paths now live under `test/...`.

The older `testkit/...` paths remain supported as compatibility aliases, so existing consumers do not need an immediate migration.

Current shipped slice:

- `test/render` now provides the public component fixture for `js/wasm` tests, including controlled mock-DOM rendering, rerendering, basic rendered-node queries, event dispatch helpers, and scheduler-settlement helpers without importing repo-local runtime test helpers.
- `test/hooks` now provides a lightweight `RenderHook(...)` harness for `js/wasm` tests, so custom hooks and hook-driven state flows can be exercised without a bespoke host component per test.
- `test/router` now provides hash and history router fixtures for `js/wasm` tests, including initial-path setup, navigation helpers, route inspection, and delegated rendered-route queries.
- `test/ssr` now provides public SSR snapshot helpers, typed bootstrap-payload assertions, and a lightweight hydration smoke harness for `js/wasm` tests.

## Why This Split

One testing package would quickly mix unrelated concerns:

- pure render and hook tests
- router and loader behavior
- SSR and hydration delivery checks
- browser-runner integration

Those concerns have different setup costs, platform needs, and failure modes. Focused packages keep each surface smaller and let applications adopt only the helpers they actually need.

This also matches the broader framework boundary already documented in [ECOSYSTEM.md](ecosystem-and-extension-model.md): testing ergonomics are important, but they are companion-package concerns built on public APIs rather than privileged runtime hooks.

## Contract

The first-party testing surface should follow these rules:

- build only on documented public packages such as `ui`, `html`, `router`, `state`, `fetch`, and SSR entrypoints
- avoid requiring consumers to import `internal/runtime` helpers
- work with ordinary Go `testing` and standard browser runners instead of replacing them
- prefer accessibility-first queries and stable semantics over CSS-selector-centric helpers
- provide deterministic flush and settlement helpers so tests do not need arbitrary sleeps
- keep browser-only behavior behind browser or `js/wasm` execution paths

## Query Semantics

The public query contract is now accessibility-first.

- prefer `ByRole(role, name)` and `AllByRole(role)` for consumer-facing assertions
- role matching is exact after whitespace normalization
- name matching is exact after whitespace normalization
- accessible names resolve in this order: `aria-label`, `aria-labelledby`, then rendered text content
- implicit roles are supported for common controls such as buttons, textboxes, checkboxes, radios, links, selects, and images
- `ByID`, `ByText`, and `AllByTag` remain available as lower-level escape hatches when a test is intentionally asserting implementation structure

This keeps the stable public path aligned with accessible UI behavior while still leaving implementation-level queries available for narrow cases.

## Recommended Patterns

Consumers should be able to copy working tests directly from the public helpers.

- component state and queued integration examples: `testkit/render/consumer_examples_wasm_test.go`
- custom hook example: `testkit/hooks/consumer_examples_wasm_test.go`
- router fixture example: `testkit/router/consumer_examples_wasm_test.go`
- SSR snapshot and bootstrap example: `testkit/ssr/consumer_examples_test.go`

Recommended usage shape:

1. unit and component tests on `js/wasm`: render with `test/render`, query by role/name, and drive events through `Click`, `Input`, or `Dispatch`
2. hook tests on `js/wasm`: use `test/hooks.RenderHook(...)` and mutate through `Act(...)`
3. router integration tests on `js/wasm`: set an initial path with `test/router`, render once, then assert params, query, and rendered output together
4. SSR delivery checks on native Go: snapshot with `test/ssr.Render(...)` and assert bootstrap payloads with `RequirePayload(...)`
5. hydration smoke checks on `js/wasm`: hydrate through `test/ssr.SmokeHydrate(...)` when a server-delivery path needs end-to-end confidence

For this repository itself, the Go-native launcher now exposes explicit lane-oriented shortcuts around the existing mix of Go, js/wasm, Playwright, and release smoke checks:

```powershell
go run ./tools/gwc test -lane unit -lane wasm
go run ./tools/gwc test -lane hydration -lane browser
go run ./tools/gwc test -lane release -app .\examples\01-counter\main.go -root .\examples\01-counter
```

Current repo lane meanings:

- `unit`: native `go test ./...` for the selected root, plus the nested `tools/livereload` module when run from the repo root
- `wasm`: discovered `*_wasm_test.go` packages under the selected root using the repo js/wasm executor helper
- `hydration`: focused js/wasm packages whose tests exercise hydration helpers or `Hydrate*` behavior
- `browser`: the Playwright workspace under `test/` when available
- `release`: a launcher-owned release smoke build into a temporary output directory

## Non-Goals

The first-party testing surface should not:

- introduce a custom test runner
- introduce privileged hidden runtime callbacks that apps cannot reach in production code
- try to replace Playwright for end-to-end browser automation
- couple framework correctness to one assertion library style

## Current Boundary

This document describes the shipped consumer testing surface.

It does claim:

- the preferred public helpers live under `test/...`
- `testkit/...` remains supported as a compatibility path
- the repo validates behavior across native Go, js/wasm, hydration-focused, and browser Playwright lanes

It does not claim:

- that all testing helpers belong inside core runtime packages
- that browser automation should be replaced by framework-specific abstractions
- that every future domain, such as workers or offline replay, already has a complete public harness

## Ownership Boundary

Core owns correctness-critical runtime behavior.

The testing companion surface owns consumer ergonomics for:

- mounting and inspecting public UI trees
- flushing scheduled work deterministically
- simulating events through public behavior contracts
- asserting router, SSR, hydration, and async flows without depending on repo-local internal test fixtures

## Implementation Order

The remaining backlog in [TODO.md](gowebcomponents-todo.md) should build this surface in this order:

1. `test/render` plus shared deterministic flush helpers
2. accessibility-first query semantics and example tests
3. router and async loader helpers
4. SSR and hydration helpers
5. browser, portal, overlay, cross-tab, worker, and offline harness extensions

## Related Docs

- [WORKFLOWS.md](common-workflows.md#test-a-component-or-app-flow)
- [ADOPTION.md](adoption-baseline.md#2-testing-recipe)
- [ECOSYSTEM.md](ecosystem-and-extension-model.md)
- [TODO.md](gowebcomponents-todo.md)
- [../test/README.md](../test/README.md)