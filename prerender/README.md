# GWC | Prerender Library

```text
  ____ ____      __
 / ___|\\ \\ \\    / /
| |  _ \\ \\ \\\\ /\\ / /
| |_| | \\ V  V /
 \\____|  \\_/\\_/
GoWebComponents (GWC)
```

## High-Level Overview

The `prerender` library supports ahead-of-time and server-side render preparation utilities for GWC content.

## Public APIs

### `github.com/monstercameron/GoWebComponents/prerender` (`package prerender`)
- Functions: `Export`
- Types: `ExportSummary`, `Route`, `RouteOutput`, `Target`
- Variables: _none_
- Constants: _none_

## Subfiles And Purpose

- `export.go` - Core implementation for export
- `export_additional_test.go` - Tests for export_additional behavior
- `export_test.go` - Tests for export behavior

## ASCII File List

```text
prerender/
|-- export.go
|-- export_additional_test.go
\-- export_test.go
```
