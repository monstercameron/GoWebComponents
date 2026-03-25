# GWC | State Library

```text
  ____ ____      __
 / ___|\\ \\ \\    / /
| |  _ \\ \\ \\\\ /\\ / /
| |_| | \\ V  V /
 \\____|  \\_/\\_/
GoWebComponents (GWC)
```

## High-Level Overview

The `state` library provides shared state primitives and hook-based integration for component-level and app-level state management.

## Public APIs

### `github.com/monstercameron/GoWebComponents/state` (`package state`)
- Functions: `ApplySnapshot`, `ExportSnapshot`, `Get`, `GetSnapshot`, `ImportSnapshot`, `LoadPersistentSnapshot`, `LoadSnapshot`, `MarshalSnapshotJSON`, `ReactiveRegionSourceIDs`, `RestorePersistentSnapshot`, `RestoreSnapshot`, `SavePersistentSnapshot`, `SaveSnapshot`, `Select`, `Set`, `Text`, `UnmarshalSnapshotJSON`, `Update`, `UseAtom`, `UseComputed`, `UseDerived`
- Types: `Atom`, `Computed`, `Derived`, `Element`, `PersistentSnapshotOptions`, `Snapshot`, `StorageArea`
- Variables: _none_
- Constants: `LocalStorage`, `SessionStorage`

## Subfiles And Purpose

- `doc.go` - Package-level Go documentation
- `example_test.go` - Tests for example behavior
- `README.md` - Folder-level documentation
- `state.go` - Core implementation for state
- `state_wasm_test.go` - Tests for state_wasm behavior

## ASCII File List

```text
state/
|-- doc.go
|-- example_test.go
|-- README.md
|-- state.go
\-- state_wasm_test.go
```
