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

1. Start with `ui` and `html` using [examples/21-ui-render](../examples/21-ui-render), [examples/22-create-element](../examples/22-create-element), and [examples/75-use-state](../examples/75-use-state).
2. Add form or event-heavy behavior with [examples/36-typed-events](../examples/36-typed-events), [examples/51-use-form](../examples/51-use-form), and [examples/53-html-forms](../examples/53-html-forms).
3. Add shared state with [examples/37-use-atom](../examples/37-use-atom) and [examples/39-use-derived](../examples/39-use-derived).
4. Add routing only if the app grows beyond one screen by following [examples/55-hash-router](../examples/55-hash-router) or [examples/56-browser-router](../examples/56-browser-router).

Recommended docs:

- [START_HERE.md](start-here.md)
- [WORKFLOWS.md](common-workflows.md#build-a-client-only-app)
- [REFERENCE_MAP.md](reference-map.md#rendering-and-local-state)

## Server-Rendered App

Use this shape when the first response should contain request-time HTML and the browser should resume over matching markup.

Build path:

1. Understand the SSR boundary with [examples/70-render-to-string](../examples/70-render-to-string) and [examples/71-hydrate](../examples/71-hydrate).
2. Add router-aware hydration with [examples/72-router-hydrate-mount](../examples/72-router-hydrate-mount) and [examples/74-ssr-route-data-reuse](../examples/74-ssr-route-data-reuse).
3. Move to a full request-time app shape with [examples/18-ssr-server-routing](../examples/18-ssr-server-routing).
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
2. Prefer [examples/55-hash-router](../examples/55-hash-router) if the host cannot provide browser-router rewrites.
3. Build the wasm binary with the matching `wasm_exec.js` and validate the static asset layout locally.
4. Add browser tests before shipping to confirm route loads and fetch behavior under the final asset structure.

Recommended docs:

- [WORKFLOWS.md](common-workflows.md#ship-a-production-wasm-build)
- [TROUBLESHOOTING.md](troubleshooting.md#broken-example-serving)
- [REFERENCE_MAP.md](reference-map.md#routing)

## Data-Heavy Dashboard

Use this shape when the UI depends on shared state, derived state, async resources, manual revalidation, and richer diagnostics.

Build path:

1. Start with shared state via [examples/37-use-atom](../examples/37-use-atom), [examples/38-use-computed](../examples/38-use-computed), and [examples/39-use-derived](../examples/39-use-derived).
2. Add typed data loading through [examples/43-use-resource](../examples/43-use-resource) and shared cache behavior with [examples/44-use-cached-resource](../examples/44-use-cached-resource).
3. Add routing and manual refresh behavior with [examples/60-route-loaders](../examples/60-route-loaders) and [examples/61-use-revalidator](../examples/61-use-revalidator).
4. Add diagnostics visibility with [examples/66-devtools-panel](../examples/66-devtools-panel), [examples/67-use-snapshot](../examples/67-use-snapshot), and [examples/69-devtools-diagnostics](../examples/69-devtools-diagnostics).

Recommended docs:

- [WORKFLOWS.md](common-workflows.md#add-routing)
- [WORKFLOWS.md](common-workflows.md#test-a-component-or-app-flow)
- [REFERENCE_MAP.md](reference-map.md#state-and-data)

## Related Docs

- [START_HERE.md](start-here.md)
- [WORKFLOWS.md](common-workflows.md)
- [REFERENCE_MAP.md](reference-map.md)
- [examples/README.md](../examples/README.md)