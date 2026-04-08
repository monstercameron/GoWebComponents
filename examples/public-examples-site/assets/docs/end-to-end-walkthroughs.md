# End-To-End Walkthroughs

This page links the current examples and docs into a few realistic app shapes.

The goal is not to claim that every workflow has a first-party project template today. The goal is to show a practical route through the current public surface without forcing readers to infer architecture from unrelated examples.

## Current Status

- This page is a shipped navigation layer over the current examples, workflow docs, and reference docs, not a future planning note.
- The walkthroughs intentionally point at real examples that already exist in the repo, including the integrated SSR path and the larger Atlas reference app.
- Use this page when you want a credible adoption route quickly, then drop into [WORKFLOWS.md](common-workflows.md), [REFERENCE_MAP.md](reference-map.md), and the linked examples for the implementation details.

## At A Glance

Use this page when you want a plausible end-to-end build path rather than a package-by-package reference tour.

Choose a walkthrough by app shape:

- Small SPA: browser-only UI with local state, forms, and optional routing
- Server-Rendered App: request-time HTML plus hydration and route-aware resume
- Static-Hosted App: client-rendered or hash-routed app deployed as static files
- Data-Heavy Dashboard: shared state, cached resources, routing, revalidation, and diagnostics

These are not scaffolds. They are curated reading and example paths through the current supported surface.

## How To Use These Walkthroughs

Each section is meant to answer three questions:

- which example to open first
- what feature layer to add next
- which docs to keep nearby while turning the example path into a real app

If a walkthrough stops short of a production-ready template, that is intentional. The purpose is to reduce guesswork, not to pretend the repo already ships a generated starter for every architecture.

## Current Boundary

- Shipped: curated reading paths through real examples and current documentation for common app shapes.
- Not shipped: generator-owned starters, one-command architecture templates, or a claim that every deployment shape already has a first-party scaffold.
- For task-by-task implementation work, use [WORKFLOWS.md](common-workflows.md). For package or capability lookups, use [REFERENCE_MAP.md](reference-map.md).

## Small SPA

Use this shape for browser-only apps that need local state, some shared state, and optional routing.

Build path:

1. Start with `ui` and `html` using [examples/public/ui-render](../examples/public/ui-render), [examples/public/create-element](../examples/public/create-element), and [examples/public/use-state](../examples/public/use-state).
2. Add form or event-heavy behavior with [examples/public/typed-events](../examples/public/typed-events), [examples/public/use-form](../examples/public/use-form), and [examples/public/html-forms](../examples/public/html-forms).
3. Add shared state with [examples/public/use-atom](../examples/public/use-atom) and [examples/public/use-derived](../examples/public/use-derived).
4. Add routing only if the app grows beyond one screen by following [examples/public/hash-router](../examples/public/hash-router) or [examples/public/browser-router](../examples/public/browser-router).

Recommended docs:

- [START_HERE.md](start-here.md)
- [WORKFLOWS.md](common-workflows.md#build-a-client-only-app)
- [REFERENCE_MAP.md](reference-map.md#rendering-and-local-state)

## Server-Rendered App

Use this shape when the first response should contain request-time HTML and the browser should resume over matching markup.

Build path:

1. Understand the SSR boundary with [examples/server/render-to-string](../examples/server/render-to-string) and [examples/public/hydration](../examples/public/hydration).
2. Add router-aware hydration with [examples/public/router-hydrate-mount](../examples/public/router-hydrate-mount) and [examples/public/server-side-rendering-route-data-reuse](../examples/public/server-side-rendering-route-data-reuse).
3. Move to a full request-time app shape with [examples/server/server-side-rendering-routing](../examples/server/server-side-rendering-routing).
4. Validate bootstrap transfer, route data reuse, and metadata after hydration.

Recommended docs:

- [MIGRATIONS.md](migration-guide.md)
- [WORKFLOWS.md](common-workflows.md#add-ssr-and-hydration)
- [TROUBLESHOOTING.md](troubleshooting.md#hydration-mismatch-warnings)

## Static-Hosted App

Use this shape when you want a browser app that can be served from static hosting or a CDN without request-time HTML.

Today this is the clearest current path for simple deployments. First-class static prerender remains backlog work, so the recommended current model is a client-rendered or hash-routed app deployed as static files.

Build path:

1. Start with the client-only path from [WORKFLOWS.md](common-workflows.md#build-a-client-only-app).
2. Prefer [examples/public/hash-router](../examples/public/hash-router) if the host cannot provide browser-router rewrites.
3. Build the wasm binary with the matching `wasm_exec.js` and validate the static asset layout locally.
4. Add browser tests before shipping to confirm route loads and fetch behavior under the final asset structure.

Recommended docs:

- [WORKFLOWS.md](common-workflows.md#ship-a-production-wasm-build)
- [TROUBLESHOOTING.md](troubleshooting.md#broken-example-serving)
- [REFERENCE_MAP.md](reference-map.md#routing)

## Data-Heavy Dashboard

Use this shape when the UI depends on shared state, derived state, async resources, manual revalidation, and richer diagnostics.

Build path:

1. Start with shared state via [examples/public/use-atom](../examples/public/use-atom), [examples/public/use-computed](../examples/public/use-computed), and [examples/public/use-derived](../examples/public/use-derived).
2. Add typed data loading through [examples/public/use-resource](../examples/public/use-resource) and shared cache behavior with [examples/public/use-cached-resource](../examples/public/use-cached-resource).
3. Add routing and manual refresh behavior with [examples/public/route-loaders](../examples/public/route-loaders) and [examples/public/use-revalidator](../examples/public/use-revalidator).
4. Add diagnostics visibility with [examples/public/devtools-panel](../examples/public/devtools-panel), [examples/public/use-snapshot](../examples/public/use-snapshot), and [examples/public/devtools-diagnostics](../examples/public/devtools-diagnostics).

Recommended docs:

- [WORKFLOWS.md](common-workflows.md#add-routing)
- [WORKFLOWS.md](common-workflows.md#test-a-component-or-app-flow)
- [REFERENCE_MAP.md](reference-map.md#state-and-data)

## Related Docs

- [START_HERE.md](start-here.md)
- [WORKFLOWS.md](common-workflows.md)
- [REFERENCE_MAP.md](reference-map.md)
- [examples/README.md](../examples/README.md)