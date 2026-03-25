# Server SSR Routing Demo

This example is the request-time SSR counterpart to the earlier static SSR shell experiments.

## Current Status

This example remains the focused server-backed SSR routing reference in the integrated example set.

Use it when you want a smaller request-time rendered app that demonstrates real HTML responses, route-aware bootstrap reuse, redirects, query-aware rendering, and hydration on top of a Go HTTP server without the larger Atlas surface area.

It demonstrates:

- a Go HTTP server that renders HTML per request with `ui.RenderToString(...)`
- a route-specific bootstrap endpoint served by the same Go process
- a browser/history router on the wasm client
- hydration that restores route bootstrap data and reuses matching DOM on startup
- a minimal streamed SSR slice for `/docs/:section?tab=loader`, where the server flushes the shell first and streams one deferred docs panel before wasm boot
- SSR head composition through `head.Render(head.Document{...})` for robots tags, social tags, JSON-LD, alternate links, and resource hints on the first response
- the ownership boundary where hydrated navigation continues to reconcile the router-managed title, description, and canonical slice while richer non-router tags stay SSR-owned
- server and client redirects for `/legacy` and `/secure`
- query-aware server rendering for `/search?q=...`

## Build the client wasm

From the repo root:

```powershell
Set-Location .\examples
.\build.ps1 -Example "18-ssr-server-routing"
```

## Run the SSR server

From the repo root:

```powershell
go run ./examples/18-ssr-server-routing
```

Then open:

- `http://127.0.0.1:8079/`
- `http://127.0.0.1:8079/docs/ssr`
- `http://127.0.0.1:8079/search?q=routing`
- `http://127.0.0.1:8079/secure`
- `http://127.0.0.1:8079/secure?auth=true&role=maintainer`

You should see real HTML requests on direct navigation and refresh, plus a bootstrap JSON request to `/_gwc/bootstrap?...` during client startup. The first client resume should reuse the server-rendered shell where it matches and only replace the affected subtree if hydration encounters a structural mismatch.

If you open `http://127.0.0.1:8079/docs/ssr?tab=loader`, the server also exercises the minimal streaming prototype: it flushes the page shell with a placeholder first, then sends the deferred docs panel before the wasm startup script runs so hydration still targets the final assembled DOM.

Two query variants exercise the first failure-oriented shapes:

- `http://127.0.0.1:8079/docs/ssr?tab=loader&stream=error`
- `http://127.0.0.1:8079/docs/ssr?tab=loader&stream=nested`

## Head ownership walkthrough

Use this example when you want one integrated head-management flow:

- direct requests and refreshes show the full SSR head bundle emitted by `head.Render(...)`
- docs routes emit alternate links and JSON-LD on first paint
- search, secure, and sign-in routes show route-specific robots behavior
- later hydrated navigation still follows the router-managed title, description, and canonical ownership rules instead of pretending the client owns every non-router head tag

Pair this example with `examples/70-render-to-string` when you want the same companion-owned head composition in a smaller prerender-style `ui.RenderToString(...)` flow.

## Observability recipe

This example is also the current end-to-end observability reference for one server-rendered request plus hydration resume. Pair it with [OBSERVABILITY.md](C:/Users/Cam/Desktop/GoWebComponents/docs/OBSERVABILITY.md) when you want to:

- create one request correlation id at HTTP request entry
- attach it to `ui.RenderToStringObserved(...)` and the observed bootstrap helpers
- transfer that same id into the first `ui.Hydrate(...)` call through `HydrationOptions{Observability: ...}`
- collect the resulting SSR and hydration events through `ui.ObserveSSR(...)`

## Browser regression check

After building the wasm client, run:

```powershell
Set-Location .\examples
npx playwright test tests/18-ssr-server-routing.spec.ts --config=playwright.ssr-server.config.ts
```
