# GWC | Utils Library

```text
  ____ ____      __
 / ___|\\ \\ \\    / /
| |  _ \\ \\ \\\\ /\\ / /
| |_| | \\ V  V /
 \\____|  \\_/\\_/
GoWebComponents (GWC)
```

## High-Level Overview

The `utils` library contains shared utility helpers used across runtime and tooling code paths.

## Public APIs

### `github.com/monstercameron/GoWebComponents/utils` (`package utils`)
- Functions: `ConsoleLog`, `ConsoleStructured`, `DisableAllDebug`, `DisableGoroutineMonitoring`, `EnableAllDebug`, `EnableGoroutineMonitoring`, `EnableHotReload`, `GetDebugStatus`, `GetGoroutineStats`, `GetMemStatsSampleRate`, `InstallHotReloadBridge`, `IsHotReloadEnabled`, `ResetGoroutineBaseline`, `ResolveDocumentURL`, `SetDebug`, `SetDebugNamespace`, `SetDebugNamespaces`, `SetDebugNamespacesExclusive`, `SetGoroutineThreshold`, `SetMemStatsSampleRate`, `WaitForever`
- Types: `FastComparable`
- Variables: _none_
- Constants: _none_

## Subfiles And Purpose

- `production_mode_production_wasm_test.go` - Tests for production_mode_production_wasm behavior
- `production_mode_wasm_test.go` - Tests for production_mode_wasm behavior
- `README.md` - Folder-level documentation
- `utils.go` - Core implementation for utils
- `utils_production.go` - Production-mode implementation or helpers
- `utils_production_mode.go` - Production-mode implementation or helpers
- `utils_production_wasm_test.go` - Tests for utils_production_wasm behavior
- `utils_wasm_test.go` - Tests for utils_wasm behavior

## ASCII File List

```text
utils/
|-- production_mode_production_wasm_test.go
|-- production_mode_wasm_test.go
|-- README.md
|-- utils.go
|-- utils_production.go
|-- utils_production_mode.go
|-- utils_production_wasm_test.go
\-- utils_wasm_test.go
```
