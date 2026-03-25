# GWC | Virtualization Library

# GoWebComponents (GWC)

## High-Level Overview

The `virtualization` library provides list/window virtualization primitives for rendering large collections efficiently.

## Public APIs

### `github.com/monstercameron/GoWebComponents/virtualization` (`package virtualization`)
- Functions: `Cancel`, `ComputeViewportState`, `Diagnostics`, `Len`, `List`, `ObserveOwnedViewport`, `WithRowLifecycle`
- Types: `ListProps`, `Range`, `RowRenderProps`, `Subscription`, `ViewportConfig`, `ViewportDiagnostics`, `ViewportState`
- Variables: _none_
- Constants: _none_

## Subfiles And Purpose

- `list.go` - Core implementation for list
- `list_test.go` - Tests for list behavior
- `observe_native.go` - Native (non-WASM) implementation for observe
- `observe_wasm.go` - WebAssembly-specific implementation for observe
- `viewport.go` - Core implementation for viewport
- `viewport_test.go` - Tests for viewport behavior

## File Map

```text
virtualization/
|-- list.go
|-- list_test.go
|-- observe_native.go
|-- observe_wasm.go
|-- viewport.go
\-- viewport_test.go
```



