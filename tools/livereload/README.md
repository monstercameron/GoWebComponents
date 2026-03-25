# Live Reload Tooling

Location: `tools/livereload/`

This directory contains the Go implementation behind the first-class standalone hot-reload dev server.

## Status

- first-class for standalone app hot reload through `go run ./tools/gwc dev -app ...`
- separate from the launcher-owned examples flow under `go run ./tools/gwc examples`
- suitable when you want rebuild-on-save plus state-preserving reload for one app surface

## Primary Dev Server

Use `gwc dev` as the primary documented entrypoint for the path-based hot-reload server:

```powershell
go run ./tools/gwc dev -app .\test\testapp\main.go
```

That path enables hot reload by default on successful rebuilds, attempts preserve-state swaps for compatible apps and edits, and falls back to remount or full reload behavior when the new bundle cannot safely preserve local state.

If you need direct control over the underlying live-reload server, run the Go program in this directory instead of going through `gwc dev`.

## Running The Live Reload Server

From the repo root, use the launcher:

```powershell
go run ./tools/gwc dev -app .\test\testapp\main.go
```

Or run the Go program in this directory directly when you need the lower-level server.

## What It Does

- watches `.go` files under the app root
- rebuilds wasm output when watched files change
- serves a browser client with websocket-driven reload notifications
- can preserve shared atom state plus compatible component-local hook state and `UseMemo` caches across reloads when the app enables the public `hotreload` package
- requests and forwards reload snapshots on hot builds so the client can restore the previous app state after reload

## Important Caveat

If you are trying to browse the entire example catalog, prefer `go run ./tools/gwc examples`. If you are actively editing one wasm app and want state-preserving rebuilds, `gwc dev` is the intended workflow and the lower-level program in this directory remains the escape hatch.
