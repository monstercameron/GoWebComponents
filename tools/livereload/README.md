# Live Reload Tooling

Location: `tools/livereload/`

This directory contains the Go implementation behind the first-class standalone hot-reload dev server.

## Status

- first-class for standalone app hot reload through `go run ./tools/gwc dev -app ...`
- separate from the Express example catalog server under `tools/dev-server/`
- suitable when you want rebuild-on-save plus state-preserving reload for one app surface

## Primary Dev Server

Use `gwc dev` as the primary documented entrypoint for the path-based hot-reload server:

```powershell
go run ./tools/gwc dev -app .\test\testapp\main.go
```

That path enables hot reload by default on successful rebuilds, attempts preserve-state swaps for compatible apps and edits, and falls back to remount or full reload behavior when the new bundle cannot safely preserve local state.

`tools/dev.ps1` and `tools/dev.sh` remain lower-level compatibility wrappers when you want direct control over the underlying live-reload server:

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

If you are trying to browse the entire example catalog, prefer the Express server under `tools/dev-server/`. If you are actively editing one wasm app and want state-preserving rebuilds, `gwc dev` is the intended workflow and the shell wrappers are compatibility shortcuts over the same lower-level server.
