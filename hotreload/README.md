# GWC | Hotreload Library

# GoWebComponents (GWC)

## High-Level Overview

The `hotreload` library provides development-time hot reload plumbing and production gating behavior for GWC runtimes.

## Public APIs

### `github.com/monstercameron/GoWebComponents/hotreload` (`package hotreload`)
- Functions: `ApplySnapshot`, `Configure`, `Disable`, `Enable`, `Enabled`, `GetSnapshot`, `Prepare`
- Types: `Config`
- Variables: _none_
- Constants: _none_

## Subfiles And Purpose

- `doc.go` - Package-level Go documentation
- `hotreload.go` - Core implementation for hotreload
- `hotreload_native.go` - Native (non-WASM) implementation for hotreload
- `hotreload_native_test.go` - Tests for hotreload_native behavior
- `hotreload_production.go` - Production-mode implementation or helpers
- `hotreload_production_mode.go` - Production-mode implementation or helpers
- `hotreload_production_wasm_test.go` - Tests for hotreload_production_wasm behavior
- `hotreload_wasm.go` - WebAssembly-specific implementation for hotreload
- `hotreload_wasm_test.go` - Tests for hotreload_wasm behavior
- `production_mode_production_wasm_test.go` - Tests for production_mode_production_wasm behavior
- `production_mode_wasm_test.go` - Tests for production_mode_wasm behavior

## File Map

```text
hotreload/
|-- doc.go
|-- hotreload.go
|-- hotreload_native.go
|-- hotreload_native_test.go
|-- hotreload_production.go
|-- hotreload_production_mode.go
|-- hotreload_production_wasm_test.go
|-- hotreload_wasm.go
|-- hotreload_wasm_test.go
|-- production_mode_production_wasm_test.go
\-- production_mode_wasm_test.go
```



