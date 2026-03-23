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

## Browser regression check

After building the wasm client, run:

```powershell
Set-Location .\examples
npx playwright test tests/18-ssr-server-routing.spec.ts --config=playwright.ssr-server.config.ts
```