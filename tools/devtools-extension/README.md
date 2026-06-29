# GoWebComponents DevTools — browser extension

A Chrome/Firefox DevTools panel showing a running GoWebComponents (wasm) app's live
**component tree**, dirty/fine-grained flags, per-node render timings, and **commit
profiling** — by consuming the `gwc.devtools.extension.v1` bridge payload that
`devtools/extension_bridge.go` emits (the app posts the latest payload as
`window.__GWC_DEVTOOLS__`).

## Design

All non-trivial logic — turning the bridge payload into the panel view model — lives in the
host-independent [`bridge.js`](./bridge.js), unit-tested under plain Node:

```sh
node --test test/bridge.test.mjs
```

`panel.js`/`devtools.js` are thin DevTools-API glue.

## Package (one command, zero-npm)

```sh
go run ./tools/devtools-extension/pack
```

Builds `dist/gwc-devtools-<version>.zip` — the directly-loadable package, with no node/web-ext
toolchain (it uses Go's stdlib `archive/zip`, keeping the repo's zero-npm posture). The packager
validates the manifest and includes exactly the runtime files (not docs/tests).

## Side-load

- **Chrome/Edge:** `chrome://extensions` → Developer mode → **Load unpacked** → pick this folder,
  or upload `dist/gwc-devtools-<version>.zip` to the Web Store / Edge Add-ons as-is.
- **Firefox:** `about:debugging` → This Firefox → **Load Temporary Add-on** → pick the zip. A
  signed `.xpi` for distribution is `web-ext sign` of that same zip (optional, not needed for
  local use).
