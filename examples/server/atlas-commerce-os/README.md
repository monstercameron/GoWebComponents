# Atlas Commerce OS

This is the current production-shaped flagship example in the repo.

This example is now organized into four top-level folders:

- `client/`: js/wasm hydration entrypoint for the shared SSR Atlas page tree
- `shared/`: shared Atlas SSR contracts, render tree, repository interfaces, seed data, and tokens
- `server/`: native Go server, auth/db layers, SSR handlers, and server-owned sqlite assets
- `docs/`: example documentation, screenshots, and local helper scripts

This is the current production-shaped SSR reference server example for the repo.

## Current Status

- Atlas is the largest integrated example in the repo and the main production-shaped SSR and hydration reference.
- It combines request-time HTML, browser hydration, same-origin APIs and mutations, CSRF-protected forms, sqlite-backed server state, and a shared SSR page tree.
- The detailed project guide, route-data notes, and server contract live under `docs/` and `server/README.md`; this file is the entrypoint.

## Where To Start

- Start with `docs/README.md` for the full project guide and Atlas-specific notes.
- Open `server/README.md` when you want the concrete run flow, endpoints, and mutation rules.
- Open `docs/ATLAS_ROUTING_DATA_BASELINE.md` when you need the route, bootstrap, cache, and derived-state contract behind the rewrite.

## What Atlas Demonstrates

- shared SSR and hydration from the same Atlas page tree
- server-rendered public storefront and internal operations routes
- same-origin JSON and HTML-post mutation flows
- sqlite-backed state with mock auth and CSRF validation
- a larger multi-surface example that exercises routing, forms, fetch, state transfer, diagnostics, and deployment concerns together

## Structure

Use the folders this way:

- `client/`: hydration entrypoint for the Atlas shell after request-time SSR
- `shared/`: shared page tree, route payloads, repository contracts, tokens, and helper logic used by both server and browser paths
- `server/`: native server, data layer, SSR handlers, auth and CSRF flows, static assets, and API endpoints
- `docs/`: Atlas-specific planning notes, rewrite baselines, screenshots, and local helper scripts

