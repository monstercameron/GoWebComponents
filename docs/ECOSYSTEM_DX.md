# Ecosystem and DX Surfaces

This page collects framework surfaces that support ecosystem tooling rather than
one application feature.

## Browser Devtools Extension

`devtools.BrowserExtensionManifestFor("chrome")` and
`devtools.BrowserExtensionManifestFor("firefox")` produce Manifest V3 companion
extension manifests. `devtools.BuildExtensionPanelPayload` projects a live
`devtools.Snapshot` into the stable `gwc.devtools.extension.v1` bridge payload
used by external panels for the component tree, props/state-derived tree data,
extension sections, diagnostics, logs, and commit profiling.

## Headless A11y Components

The `a11y` package wraps the core overlay, focus, composite-navigation, and
announcer primitives into headless menu, combobox, listbox, date-picker grid,
and data-table builders. The package emits semantic HTML and ARIA contracts
without visual styling, so application design systems can bring their own
classes.

## Scheduler Instrumentation

The `scheduler` package exposes `NewInstrumented(next)` for wrapping a runtime
scheduler. It records scheduled/executed idle and timeout work, inline fallback
counts, configured delays, and queue latency. `Instrumented.DevtoolsSection`
converts those metrics into a `devtools.ExtensionSection`.

## Server-Component-Style Model

The `servercomponents` package defines server-only component descriptors,
client-slot references, manifest collection, and a `ServerOnly` render helper.
Native/server builds render the supplied server node. Wasm builds emit a
placeholder template, giving build tooling a clear boundary for excluding
server-only render code from the client entry.

## RUM and OpenTelemetry Export

The `telemetry` package converts `devtools.Snapshot` profiling, diagnostics,
and logs into `RUMEvent` values, converts SSR observations through
`EventFromSSRObservation`, and exports OTLP/HTTP JSON with `BuildOTLPJSON` or
`ExportOTLPHTTP`. In browser wasm builds, Go's HTTP transport uses the browser
fetch stack, so the same exporter can send OTLP JSON from the client.
