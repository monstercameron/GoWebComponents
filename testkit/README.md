# GWC | Testkit Library

# GoWebComponents (GWC)

## High-Level Overview

The `testkit` library provides testing helpers for rendering, hooks, router behavior, and SSR-oriented verification.

## Public APIs

### `github.com/monstercameron/GoWebComponents/testkit/hooks` (`package hooks`)
- Functions: `Act`, `Cleanup`, `Current`, `Flush`, `RenderHook`, `Rerender`
- Types: `Harness`
- Variables: _none_
- Constants: _none_

### `github.com/monstercameron/GoWebComponents/testkit/render` (`package render`)
- Functions: `AllByRole`, `AllByTag`, `AttemptCount`, `Attempts`, `Attr`, `Await`, `ByID`, `ByRole`, `ByText`, `Cancel`, `Change`, `ChangeByID`, `Children`, `Cleanup`, `Click`, `ClickByID`, `Container`, `Dispatch`, `DispatchByID`, `Exists`, `Flush`, `FlushTimers`, `Input`, `InputByID`, `Loader`, `Name`, `New`, `NewResourceController`, `NodeID`, `Pending`, `Property`, `Reject`, `Render`, `Rerender`, `Resolve`, `SeedHTML`, `Stabilize`, `Started`, `Submit`, `SubmitByID`, `Tag`, `Target`, `Text`, `WithQueuedScheduler`
- Types: `Event`, `Fixture`, `Option`, `QueryNode`, `ResourceAttempt`, `ResourceController`, `SeededMarkup`
- Variables: _none_
- Constants: _none_

### `github.com/monstercameron/GoWebComponents/testkit/router` (`package routertest`)
- Functions: `AttemptCount`, `Attempts`, `ByID`, `ByText`, `Cancel`, `Cleanup`, `Inspect`, `Loader`, `Navigate`, `NewHash`, `NewHistory`, `NewLoaderController`, `Params`, `Path`, `Pending`, `Query`, `Register`, `Reject`, `Render`, `Replace`, `Resolve`, `Router`, `SetPath`, `Started`, `Text`
- Types: `Fixture`, `Inspection`, `LoaderAttempt`, `LoaderController`
- Variables: _none_
- Constants: _none_

### `github.com/monstercameron/GoWebComponents/testkit/ssr` (`package ssr`)
- Functions: `ByID`, `ByText`, `CanonicalURL`, `Cleanup`, `Contains`, `JSONLD`, `LoadStaticExport`, `MetaName`, `MetaProperty`, `Render`, `RequirePayload`, `RoundTripHydrate`, `Route`, `SmokeHydrate`, `Structured`, `Text`
- Types: `ExportedRoute`, `HydrationHarness`, `HydrationOptions`, `LinkTag`, `MetaTag`, `ScriptTag`, `Snapshot`, `StaticExport`, `StructuredSnapshot`
- Variables: _none_
- Constants: _none_

## Subfiles And Purpose

- `hooks/` - Hook-related helpers (6 files).
- `render/` - Render testing and helpers (9 files).
- `router/` - Router test helpers and fixtures (8 files).
- `ssr/` - Server-side rendering helpers (9 files).

## File Map

```text
testkit/
|-- hooks/
|   |-- consumer_examples_wasm_test.go
|   |-- doc.go
|   |-- hooks_native.go
|   |-- hooks_native_stubs_test.go
|   |-- hooks_wasm.go
|   \-- hooks_wasm_test.go
|-- render/
|   |-- async.go
|   |-- async_test.go
|   |-- consumer_examples_wasm_test.go
|   |-- doc.go
|   |-- render_native.go
|   |-- render_native_stubs_test.go
|   |-- render_wasm.go
|   |-- render_wasm_test.go
|   \-- seed_wasm.go
|-- router/
|   |-- consumer_examples_wasm_test.go
|   |-- doc.go
|   |-- loader_controller.go
|   |-- loader_controller_test.go
|   |-- router_native.go
|   |-- router_native_stubs_test.go
|   |-- router_wasm.go
|   \-- router_wasm_test.go
\-- ssr/
    |-- consumer_examples_test.go
    |-- doc.go
    |-- hydrate_native.go
    |-- hydrate_wasm.go
    |-- hydrate_wasm_test.go
    |-- ssr.go
    |-- ssr_test.go
    |-- ssr_walk_default.go
    \-- ssr_walk_wasm.go
```



