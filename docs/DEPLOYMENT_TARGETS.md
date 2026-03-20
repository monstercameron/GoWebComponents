# Deployment Targets And Adapter Expectations

This page records the intended deployment-target model for GoWebComponents apps.

Use it when deciding whether an app should run as a static-hosted client bundle, a single Go SSR server, a reverse-proxy fronted app, or a split SSR/API deployment.

## Evaluation Result

The supported deployment targets should stay intentionally small and explicit:

- static hosting for client-only wasm bundles or fully prerendered output
- one Go server that serves SSR pages, APIs, and static assets together
- one Go SSR or API server behind a reverse proxy or CDN
- split SSR/API deployments when the browser still sees one same-origin front door

These are the deployment shapes the framework can describe cleanly today without inventing a second server model or a custom adapter ecosystem.

## Adapter Expectations

Any adapter or deployment helper should preserve the core HTTP and asset contract:

- serve `*.wasm`, `wasm_exec.js`, and related static assets from stable URLs
- keep HTML, bootstrap payloads, and asset references aligned with the current base path
- preserve request identity, timeout, auth, and CSRF context for SSR handlers
- keep deep-link rewrites working for browser-router deployments
- keep cache headers, compression behavior, and MIME types consistent with the selected host
- keep same-origin browser traffic when SSR and API logic are split across processes

## What Does Not Belong In An Adapter

Adapters should not hide or rewrite the framework's runtime contract.

They should not:

- invent a new component model
- change hydration semantics
- smuggle request-scoped data into global state
- force one hosting shape to emulate a different one without making that tradeoff explicit

## Relationship To Existing Docs

This page is the short deployment-target summary.

The detailed operational guidance lives in:

- `SERVER_INTEGRATION.md` for SSR request flow, middleware order, and host-specific deployment behavior
- `ASSETS.md` for manifests, hashed assets, and base-path handling
- `PRERENDER.md` for build-time HTML output and prerendered route contracts
- `PWA.md` for service-worker and offline deployment concerns

## Current Boundary

The repo does not yet define a separate first-party adapter SDK.

If one ever appears, it should be a thin integration layer over the deployment modes above, not a new runtime abstraction.
