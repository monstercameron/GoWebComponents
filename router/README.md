# GWC | Router Library

```text
  ____ ____      __
 / ___|\\ \\ \\    / /
| |  _ \\ \\ \\\\ /\\ / /
| |_| | \\ V  V /
 \\____|  \\_/\\_/
GoWebComponents (GWC)
```

## High-Level Overview

The `router` library provides client-side routing, route matching, navigation, and loader orchestration for GWC single-page apps.

## Public APIs

### `github.com/monstercameron/GoWebComponents/router` (`package router`)
- Functions: `AllowNavigation`, `BlockNavigation`, `Bool`, `Current`, `DefineRoute`, `Delete`, `Encode`, `Get`, `GetCurrentPath`, `GetCurrentRouterPath`, `GetOutlet`, `GetRoute`, `GetRouter`, `GoGetRoute`, `GoRegisterRoute`, `Has`, `Href`, `HrefFor`, `HydrateMount`, `HydrateMountElement`, `InspectCurrentRoute`, `Int`, `IsLoading`, `Loading`, `MetadataNode`, `Mount`, `MountElement`, `MustDefineRoute`, `MustHref`, `MustHrefFor`, `MustPath`, `MustPathFor`, `Navigate`, `NavigateReplace`, `NewHashRouter`, `NewHistoryRouter`, `ParamNames`, `Path`, `PathFor`, `Pattern`, `PreserveReturnTo`, `ReadReturnTo`, `RedirectNavigation`, `Register`, `RegisterRoute`, `Replace`, `ReplaceAll`, `Revalidate`, `RouteWithElement`, `Set`, `UseNavigate`, `UseParams`, `UseQuery`, `UseRevalidator`, `UseRouteData`, `UseSearchParams`, `Values`
- Types: `AsyncGuardFunc`, `AsyncLeaveGuardFunc`, `Attrs`, `Component`, `Element`, `GuardDecision`, `GuardFunc`, `GuardResult`, `LeaveGuardFunc`, `LoaderFunc`, `Metadata`, `Navigator`, `Options`, `Params`, `Query`, `Revalidator`, `RouteContext`, `RouteContract`, `RouteInspection`, `RouteLoaderInspection`, `RouteParamsProvider`, `RouteQueryProvider`, `RouteRedirectInspection`, `RouteStackInspection`, `Router`, `RouterOptions`, `SearchParams`
- Variables: _none_
- Constants: `ReturnToParam`

## Subfiles And Purpose

- `browser_router_test.go` - Tests for browser_router behavior
- `browser_test_helpers_wasm_test.go` - Tests for browser_test_helpers_wasm behavior
- `contracts.go` - Core implementation for contracts
- `contracts_additional_test.go` - Tests for contracts_additional behavior
- `contracts_test.go` - Tests for contracts behavior
- `doc.go` - Package-level Go documentation
- `example_test.go` - Tests for example behavior
- `metadata.go` - Core implementation for metadata
- `metadata_test.go` - Tests for metadata behavior
- `README.md` - Folder-level documentation
- `router.go` - Core implementation for router
- `router_benchmark_test.go` - Tests for router_benchmark behavior
- `router_test.go` - Tests for router behavior

## ASCII File List

```text
router/
|-- browser_router_test.go
|-- browser_test_helpers_wasm_test.go
|-- contracts.go
|-- contracts_additional_test.go
|-- contracts_test.go
|-- doc.go
|-- example_test.go
|-- metadata.go
|-- metadata_test.go
|-- README.md
|-- router.go
|-- router_benchmark_test.go
\-- router_test.go
```
