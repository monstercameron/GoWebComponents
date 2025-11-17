# GoWebComponents Refactor & Testing Plan

## Objectives
- Decouple core runtime logic from `syscall/js` so it can compile and be unit-tested on any platform.
- Contain browser/WASM-specific behavior inside small adapter packages.
- Establish a reliable automated testing strategy combining fast unit tests and Playwright-driven integration tests.
- Fix correctness issues discovered during the review (router hash handling, wildcard matching, state snapshot paths, etc.).

## Workstream A – Core Runtime Extraction
1. Create an `internal/runtime` (or `pkg/runtime`) module that owns:
   - Fiber structs, reconciliation, scheduling, diffing.
   - Hooks (`GoUseState`, `GoUseEffect`, memoization, atom store).
   - State snapshot serialization/deserialization.
   - Generic interfaces: `DOMAdapter`, `Scheduler`, `EventAdapter`, `BrowserState`.
2. Move current logic from `fiber/fiber.go`, `hooks.go`, `state_management.go`, etc., into this package, replacing direct DOM/event calls with adapter injections.
3. Add host-side unit tests for runtime behavior (hook order, atom propagation, state restore).

## Workstream B – Platform Adapters
1. Add `platform/jsdom` (or `fiber/jsadapter`) that implements the runtime interfaces using `syscall/js` (`createElement`, `updateDom`, event listeners, idle callbacks, storage).
2. Derive a `platform/mockdom` for tests: records DOM operations, simulates events without a browser.
3. Ensure the public `fiber` package simply wires runtime + adapters for the WASM build; keep build tags so only adapters import `syscall/js`.

## Workstream C – Testing Strategy
1. **Unit tests:** `go test ./internal/runtime/...` runs on any OS/arch.
2. **Mock-adapter tests:** use `platform/mockdom` to verify DOM diffs, router behavior, and state snapshots in host builds.
3. **Playwright-Go integration:**
   - Build `main.wasm`, serve via existing scripts.
   - Use Playwright-Go to launch Chromium/Firefox, drive UI interactions, and assert via exported helper functions (`ExportAppState`, DOM queries).
   - Integrate into CI (GitHub Actions or similar) with caching for Go modules and Playwright browsers.

## Workstream D – Identified Bug Fixes
1. Router hash handling (`fiber/router.go:205-213`) should derive hash from the requested path, not hard-code `/` or `/docs`.
2. Router wildcard pointer bug (range variable capture) – copy the loop variable before taking its address.
3. `GoRegisterRoute` default branch must invoke component functions so routes always return `*Element`.
4. State snapshot sibling paths should trim the last segment, not just the last character.
5. `SetMemoryLimits` should actually apply the provided limits or be removed.
6. Ensure default styles use `map[string]string` or make `createDom` accept `map[string]interface{}` for `style`.

## Workstream E – Tooling & DX
1. Update documentation (README/docs) to explain the new package layout and testing commands.
2. Provide Makefile (or `mage`) targets for `test-unit`, `test-wasm`, `test-playwright`.
3. Consider enabling debug namespace configuration via env vars to simplify integration tests.

## Deliverables
- Refactored package structure with clear separation of runtime vs platform adapters.
- Comprehensive host-side test suite and Playwright-Go integration tests.
- Fixed router/state/memory issues validated by tests.
- Updated docs and scripts guiding contributors through the new workflow.
