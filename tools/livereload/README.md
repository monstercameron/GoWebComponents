# Live Reload Tooling

Location: `tools/livereload/`

This directory contains the older Go-based live reload server. It is still available, but it is no longer the primary documented dev path for the repo.

## Status

- Experimental / legacy relative to `tools/dev-server/`
- Useful if you specifically want file watching and hot-reload behavior
- Not the main example-serving workflow documented in the root README

## Primary Dev Server

Use this for normal example serving:

```powershell
npm run dev:examples
```

That starts the Express server on `http://127.0.0.1:8090`.

## Running The Live Reload Server

From the repo root on Windows:

```powershell
.\tools\livereload.ps1
```

On Unix-like systems:

```bash
./tools/livereload.sh
```

Or run the Go program directly from this directory.

## What It Does

- watches `.go` files
- rebuilds wasm output when watched files change
- serves a browser client with websocket-driven reload notifications
- can preserve limited app state if the application exports the expected hooks

## Important Caveat

This tooling still reflects an older workflow. If you are trying to manually open and test repo examples, prefer the Express server under `tools/dev-server/` unless you explicitly need live reload behavior.
