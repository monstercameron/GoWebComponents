# Cross-Tab Sync

Mirrors state across browser tabs in real time over `BroadcastChannel` — theme, auth status, a cache
revision, and a shared draft all stay in sync between every open tab of the app.

## What it shows
- Publishing and subscribing to named channels via the `interop` BroadcastChannel binding.
- Typed signals (JSON-encoded structs) per channel, so each concern syncs independently.
- The same-origin, no-server sync substrate that the durable `kvstate` layer uses for its cross-tab
  watch.

## Run
```
gwc dev examples/public/cross-tab-sync
```
Open the page in two tabs and change the theme or draft in one — the other updates immediately.
