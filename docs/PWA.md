# PWA and Offline App Support

This page defines the intended Progressive Web App and offline-application support boundary for GoWebComponents.

Use it when deciding how web app manifests, service workers, offline caches, mutation replay, installability, and deployment concerns should fit around the current runtime and browser-support model.

## Support Boundary

The intended PWA scope is integration-oriented, not a hidden runtime feature.

The current project direction is:

- document clear integration points for manifests, service workers, offline caching, and installability
- keep the core rendering and hydration runtime independent from service-worker lifecycle logic
- treat offline mutation replay, cache invalidation, and static asset delivery as related but distinct concerns

The framework does not yet claim:

- a built-in service-worker runtime
- automatic app-manifest generation from app code
- one-click offline enablement for every app

Applications should choose PWA features deliberately instead of assuming every GoWebComponents app is installable or offline-ready by default.

## Web App Manifest Boundary

Installability metadata belongs in a manifest pipeline, not in the rendering runtime itself.

The intended model is:

- applications provide a web app manifest with app name, icons, start URL, display mode, theme color, and related metadata
- manifest generation or templating may live in companion tooling rather than in core rendering APIs
- SSR, prerender, or static-hosted HTML should reference the manifest explicitly through ordinary head markup

The project may grow first-party manifest helpers later, but the current contract is the integration boundary, not a shipped manifest generator.

## Service-Worker Integration Story

Service workers should be treated as an application-owned deployment concern with documented framework touchpoints.

The intended integration story is:

- register the service worker from explicit client bootstrap code, not from hidden framework side effects
- precache shell assets, wasm artifacts, and other immutable build outputs through service-worker tooling or application-owned registration logic
- keep cache naming and versioning aligned with the same asset manifest and hashing rules documented in [ASSETS.md](ASSETS.md)
- keep service-worker scope, update flow, and fallback behavior explicit in the application instead of coupling them to route rendering

This keeps the runtime small while still making PWA integration a first-class documented path.

## Offline Caching Strategies

Offline behavior should separate shell delivery, static assets, data reads, and mutations.

The intended cache split is:

- app shell and route HTML: cache for repeat visits and offline fallback only when the deployment model supports serving those documents safely
- immutable static assets: cache aggressively by hashed filename
- route or API data: cache deliberately with app-owned freshness rules instead of assuming offline persistence is always correct
- queued writes: use the offline mutation queue where durable replay is needed, rather than conflating write replay with read caching

Recommended model:

- use immutable asset caching for wasm, JS helpers, CSS, and fingerprinted media
- use explicit offline fallback documents or cached shells for routes that should still open without network
- prefer revalidation-aware read caching over silent indefinite API caching
- use `fetch.OpenMutationQueue(...)` for durable browser-side write replay when offline writes matter

## Update And Invalidation Semantics

PWA update behavior must keep old clients from running half-old asset graphs.

The intended rules are:

- version shell assets, wasm artifacts, JS helpers, and CSS through the same hashed-asset or manifest strategy used outside the service worker
- treat HTML documents and manifest-like entrypoints as short-lived so they can point clients at the newest immutable assets
- when a new service worker activates, make stale caches disposable as one versioned unit instead of mixing assets across releases
- if an update changes the active wasm or JS bootstrap set, the application should prefer prompting for refresh or safe reload rather than silently continuing on a mismatched graph

Offline caches should be invalidated by release version or manifest revision, not by ad hoc per-file guessing.

## Production Deployment Guidance

PWA deployments add operational requirements beyond ordinary static hosting.

Recommended rules:

- serve over HTTPS for service-worker registration and installability
- ensure service-worker scope matches the intended route base path
- keep cache headers aligned with the asset strategy: immutable for hashed assets, short-lived for HTML and manifest-like entrypoints
- validate CDN or reverse-proxy behavior so service-worker scripts, manifests, and offline fallbacks are not cached incorrectly
- treat failed service-worker rollout, stale caches, and offline fallback loops as release risks, not as edge cases

PWA deployments should be validated together with:

- [ASSETS.md](ASSETS.md)
- [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md)
- [OFFLINE_MUTATIONS.md](OFFLINE_MUTATIONS.md)

## Current Boundary

This document defines the intended integration contract only.

It does not yet claim:

- shipped manifest generation helpers
- a first-party service-worker implementation
- installability examples or production validation suites

Those remain separate backlog work.
