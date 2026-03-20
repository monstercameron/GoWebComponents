# Live Reload Tooling

Location: `tools/livereload/`

This directory contains the Go implementation behind the first-class standalone hot-reload dev server.

## Status

- first-class for standalone app hot reload through `tools/dev.ps1` and `tools/dev.sh`
- separate from the Express example catalog server under `tools/dev-server/`
- suitable when you want rebuild-on-save plus state-preserving reload for one app surface

## Primary Dev Server

Use `tools/dev.ps1` or `tools/dev.sh` for the path-based hot-reload server:

```powershell
.\tools\dev.ps1 -App .\test\testapp\main.go
```

That builds the app from the directory that contains `main.go`, serves the HTML shell, and keeps the browser state bridge alive across rebuilds.

## Running The Live Reload Server

From the repo root on Windows:

```powershell
.\tools\dev.ps1 -App .\test\testapp\main.go
```

On Unix-like systems:

```bash
./tools/dev.sh ./test/testapp/main.go
```

Or run the Go program directly from this directory.

## What It Does

- watches `.go` files under the app root
- rebuilds wasm output when watched files change
- serves a browser client with websocket-driven reload notifications
- can preserve shared atom state plus compatible component-local hook state and `UseMemo` caches across reloads when the app enables the public `hotreload` package
- requests and forwards reload snapshots on hot builds so the client can restore the previous app state after reload

## Important Caveat

If you are trying to browse the entire example catalog, prefer the Express server under `tools/dev-server/`. If you are actively editing one wasm app and want state-preserving rebuilds, this is the intended workflow.
