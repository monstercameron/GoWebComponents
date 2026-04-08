# Deployment Targets And Adapter Expectations

This page records the intended deployment-target model for GoWebComponents apps.

Use it when deciding whether an app should run as a static-hosted client bundle, a single Go SSR server, a reverse-proxy fronted app, or a split SSR/API deployment.

## At A Glance

The deployment story is intentionally narrow:

- static hosting for client-only or prerendered output
- one Go server for unified SSR, API, and assets
- reverse-proxy or CDN fronted Go deployments
- split SSR and API processes only when the browser still sees one coherent same-origin application boundary

This is a deployment contract, not an adapter marketplace. The goal is to keep hosting expectations explicit and operationally explainable.

## Evaluation Result

The supported deployment targets should stay intentionally small and explicit:

- static hosting for client-only wasm bundles or fully prerendered output
- one Go server that serves SSR pages, APIs, and static assets together
- one Go SSR or API server behind a reverse proxy or CDN
- split SSR/API deployments when the browser still sees one same-origin front door

These are the deployment shapes the framework can describe cleanly today without inventing a second server model or a custom adapter ecosystem.

## Quick Deployment Chooser

Use this rule of thumb:

- choose static hosting when the app is client-only or prerendered and does not need request-time HTML
- choose one Go server when SSR, forms, auth context, and asset serving should stay in one process boundary
- choose reverse-proxy fronting when infrastructure needs CDN, TLS, or edge routing in front of a Go origin
- choose split SSR and API processes only when the operational team can still preserve same-origin browser behavior and a shared request contract

If a deployment shape needs the framework to pretend one host model is another, it is usually the wrong abstraction boundary.

## Adapter Expectations

Any adapter or deployment helper should preserve the core HTTP and asset contract:

- serve `*.wasm`, `wasm_exec.js`, and related static assets from stable URLs
- keep HTML, bootstrap payloads, and asset references aligned with the current base path
- preserve request identity, timeout, auth, and CSRF context for SSR handlers
- keep deep-link rewrites working for browser-router deployments
- keep cache headers, compression behavior, and MIME types consistent with the selected host
- keep same-origin browser traffic when SSR and API logic are split across processes

## Readiness Checklist

Before calling a deployment target production-ready, verify all of the following:

- `.wasm`, `wasm_exec.js`, CSS, and any other critical assets resolve from stable URLs
- base-path handling is centralized rather than hard-coded into route templates
- browser-router deep links resolve correctly on refresh and direct entry
- SSR handlers receive the request identity and security context they expect
- cache, compression, and MIME behavior match the chosen host and asset strategy
- split-process deployments still preserve a coherent same-origin browser model

If any one of those depends on undocumented adapter magic, the deployment story is not stable yet.

## What Does Not Belong In An Adapter

Adapters should not hide or rewrite the framework's runtime contract.

They should not:

- invent a new component model
- change hydration semantics
- smuggle request-scoped data into global state
- force one hosting shape to emulate a different one without making that tradeoff explicit

## Maintainer Guidance

Prefer thin integrations over clever abstractions.

- Do not add adapter layers that obscure the real HTTP contract.
- Do not let deployment helpers redefine SSR, hydration, or asset rules.
- Do not widen support claims for hosts or platforms that the repo cannot describe clearly end to end.

The framework should be portable, but that portability has to stay visible and understandable.

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
