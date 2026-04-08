# SSR Routing Demo

This example is the static-shell SSR prototype.

## Current Status

This example remains the static-document SSR and hydration prototype in the examples set.

Use it when you want to inspect pre-rendered HTML plus bootstrap-driven hydration inside a static example shell. If you need true request-time HTML responses on direct navigation, the current integrated server-backed reference is `examples/server/server-side-rendering-routing`.

If you want the request-time server-rendered version with real HTML responses on direct URL navigation, use `examples/server/server-side-rendering-routing` instead.

This example demonstrates:

- static SSR markup shipped directly in the HTML document
- a sidecar bootstrap payload loaded from `bootstrap.json`
- wasm hydration startup through `ui.Hydrate(...)`
- DOM reuse for matching server-rendered nodes, with subtree fallback if hydration cannot continue safely
- advanced routing features including params, redirects, guards, query-aware loaders, and manual revalidation

Serve examples from the repo root with:

```powershell
go run ./tools/gwc examples
```

Then open:

`/examples/public/static-server-side-rendering-routing/ssr-routing.html`

The initial document contains pre-rendered docs content for `/docs/ssr`. Once wasm starts, the client restores the bootstrap payload, reuses matching server DOM where possible, and hydrates into a routed app that supports:

- `/docs/:section`
- `/search?q=...`
- `/secure?auth=true`
- `/legacy` redirecting into the docs route
