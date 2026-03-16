# SSR Routing Demo

This example is the static-shell SSR prototype.

If you want the request-time server-rendered version with real HTML responses on direct URL navigation, use `examples/18-ssr-server-routing` instead.

This example demonstrates:

- static SSR markup shipped directly in the HTML document
- a sidecar bootstrap payload loaded from `bootstrap.json`
- wasm hydration startup through `ui.Hydrate(...)`
- advanced routing features including params, redirects, guards, query-aware loaders, and manual revalidation

Open:

`/examples/17-ssr-routing/ssr-routing.html`

The initial document contains pre-rendered docs content for `/docs/ssr`. Once wasm starts, the client hydrates into a routed app that supports:

- `/docs/:section`
- `/search?q=...`
- `/secure?auth=true`
- `/legacy` redirecting into the docs route