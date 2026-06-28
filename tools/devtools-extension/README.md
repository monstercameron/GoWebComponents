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

## Side-load (one command)

- **Chrome/Edge:** `chrome://extensions` → enable Developer mode → **Load unpacked** → pick
  this folder. (Or `npx web-ext run` for Firefox from this directory.)
- The `.crx`/`.xpi` package is just this folder zipped (and signed for store distribution);
  the source here is the extension.
