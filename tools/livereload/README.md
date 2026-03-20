# Live Reload Tooling

Location: `tools/livereload/`

This directory contains the older Go-based live reload server. It is still available, but it is no longer the primary documented dev path for the repo.

## Status

- Experimental / legacy relative to `tools/dev-server/`
- Useful if you specifically want file watching and hot-reload behavior
- Not the main example-serving workflow documented in the root README

## Primary Dev Server

Use `tools/dev.ps1` or `tools/dev.sh` for the path-based hot-reload server:

```powershell
.\tools\dev.ps1 -Main .\test\testapp\main.go
```

That builds the app from the directory that contains `main.go`, serves the HTML shell, and keeps the browser state bridge alive across rebuilds.

## Running The Live Reload Server

From the repo root on Windows:

```powershell
.\tools\dev.ps1 -Main .\test\testapp\main.go
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
- can preserve shared atom state plus compatible component-local hook state and `UseMemo` caches across reloads when the app installs the hot-reload snapshot bridge via `utils.EnableHotReload(true)`
- requests and forwards reload snapshots on hot builds so the client can restore the previous app state after reload

## Important Caveat

This tooling still reflects an older workflow. If you are trying to manually open and test repo examples, prefer the Express server under `tools/dev-server/` unless you explicitly need live reload behavior.
