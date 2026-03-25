# GWC | Logging Library

# GoWebComponents (GWC)

## High-Level Overview

The `logging` library provides structured logging surfaces and adapters used by the GWC runtime and tools.

## Public APIs

### `github.com/monstercameron/GoWebComponents/logging` (`package logging`)
- Functions: `AttachBrowserConsole`, `Debug`, `Error`, `Info`, `Log`, `New`, `Scope`, `Warn`
- Types: `BrowserConsoleOptions`, `Fields`, `Logger`
- Variables: _none_
- Constants: _none_

## Subfiles And Purpose

- `backend_native.go` - Native (non-WASM) implementation for backend
- `backend_wasm.go` - WebAssembly-specific implementation for backend
- `browser_console_stub.go` - Core implementation for browser_console_stub
- `browser_console_wasm.go` - WebAssembly-specific implementation for browser_console
- `doc.go` - Package-level Go documentation
- `logging.go` - Core implementation for logging
- `logging_additional_test.go` - Tests for logging_additional behavior
- `logging_test.go` - Tests for logging behavior
- `redaction.go` - Core implementation for redaction
- `redaction_test.go` - Tests for redaction behavior

## File Map

```text
logging/
|-- backend_native.go
|-- backend_wasm.go
|-- browser_console_stub.go
|-- browser_console_wasm.go
|-- doc.go
|-- logging.go
|-- logging_additional_test.go
|-- logging_test.go
|-- redaction.go
\-- redaction_test.go
```



