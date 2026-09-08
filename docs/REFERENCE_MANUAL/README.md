# GoWebComponents Reference Manual

This manual is the task-first and API-first guide for building web apps with GoWebComponents.

Use it when you want one curated path through the current public surface without having to reconstruct the framework story from package READMEs, design notes, workflow docs, and examples by hand.

## At A Glance

- This manual is the primary developer-experience reading path for application authors.
- It consolidates the current public package surface, `gwc` workflows, example references, and design boundaries into one structured guide.
- It replaces the former top-level `docs/*.md` app-authoring set while package GoDoc and package READMEs remain the source material behind the manual.
- The intended application-authoring surface remains `ui`, `html`, `state`, `fetch`, `router`, `interop`, `devtools`, and related companion packages documented as public in this repo.

## Who This Manual Is For

Use this manual if you are:

- evaluating whether GoWebComponents fits a new browser app
- starting a real app and want the shortest correct path through the public APIs
- moving from small examples to a production-shaped app
- adopting routing, SSR, hydration, shared state, fetch, diagnostics, or PWA features incrementally
- scaling a GWC app into a larger multi-package codebase

This manual is not the best starting point when you are:

- changing framework internals under `internal/`
- debugging runtime implementation details before you understand the public API
- looking only for a dense symbol index without the chapter narrative; start with [16 API Browser](16-api-browser.md)

For those cases, use the relevant package README, GoDoc, or implementation-area docs directly.

## Reading Paths

Choose the shortest path that matches your job.

### New Adopter

Read in this order:

1. [01 Getting Started](01-getting-started.md)
2. [02 GWC Workflows](02-gwc-workflows.md)
3. [03 App Shapes](03-app-shapes.md)
4. [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)
5. [05 HTML Authoring](05-html-authoring.md)

### App Author Adding Features

Jump directly to the chapter that matches the feature:

- rendering, hooks, async boundaries, overlays: [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)
- HTML builders, shorthand helpers, semantic markup: [05 HTML Authoring](05-html-authoring.md)
- local state, reducers, atoms, selectors, snapshots: [06 State And Reactivity](06-state-and-reactivity.md)
- typed resources, cache, mutations, offline replay: [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- routers, params, loaders, guards, metadata: [08 Routing](08-routing.md)
- SSR, bootstrap, hydration, server integration: [09 SSR And Hydration](09-ssr-and-hydration.md)
- browser APIs, workers, channels, interop: [10 Browser Interop And Workers](10-browser-interop-and-workers.md)
- forms, accessibility, locale workflows: [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md)
- dense package and symbol lookup: [16 API Browser](16-api-browser.md)
- experimental Windows desktop apps: [Windows Desktop Integration](16-windows-desktop.md)

### Team Lead Or Architect

Read these first:

1. [03 App Shapes](03-app-shapes.md)
2. [14 Scaling Large Codebases](14-scaling-large-codebases.md)
3. [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md)
4. [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)
5. [12 Devtools Testing And Observability](12-devtools-testing-and-observability.md)

### Debugging Or Validation Work

Start here:

1. [02 GWC Workflows](02-gwc-workflows.md)
2. [09 SSR And Hydration](09-ssr-and-hydration.md)
3. [12 Devtools Testing And Observability](12-devtools-testing-and-observability.md)
4. [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)

## How To Use This Manual

Use the manual in layers instead of reading it straight through end to end.

Recommended approach:

1. pick the chapter that matches the app shape or immediate problem
2. copy the minimal path first
3. read the production-shaped example before widening your design
4. read the scale-up guidance before spreading the pattern across a large codebase
5. use the inline example and package links when you need edge cases or deeper source material

The chapters are intentionally structured to answer three different kinds of questions:

Each chapter ends with a pagination block that links to the previous topic, the topic index, and the next topic in the ordered sequence.

- how do I get this working
- what is the supported public API
- what are the design boundaries and production caveats

## Escalation Map

Use this map when a small app grows and you need the next chapter instead of more improvisation:

- once one component no longer owns the whole interaction, move from [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md) to [06 State And Reactivity](06-state-and-reactivity.md)
- once data is reused across routes, panels, or resume paths, move from local fetch flows to [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- once URLs, redirects, or layout shells become part of the product contract, move to [08 Routing](08-routing.md)
- once the server must own first paint or bootstrap payloads, move to [09 SSR And Hydration](09-ssr-and-hydration.md)
- once browser capabilities such as workers, multi-tab state, or installability matter, move to [10 Browser Interop And Workers](10-browser-interop-and-workers.md) and [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)
- once a local agent or CI lane needs to drive a real wasm app through a live session, move to [12 Devtools Testing And Observability](12-devtools-testing-and-observability.md) and follow the live agent bridge recipe
- once several route families, teams, or release lanes appear, move to [14 Scaling Large Codebases](14-scaling-large-codebases.md)
- once a design decision depends on stability tiers, non-goals, or ownership boundaries, confirm it in [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md)

## Stability Legend

This manual uses the stability language summarized in [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md).

- `Stable`: safe default for long-term production use
- `Supported companion`: production-safe companion surface whose payloads may grow additively
- `Experimental`: public and usable, but still evolving in shape, defaults, or lifecycle semantics
- `Internal`: not part of the supported application-authoring contract

Practical reading rule:

- treat `Stable` as the default choice when it fits the problem
- use `Supported companion` when the feature belongs in a companion package and the repo documents it as production-safe
- use `Experimental` only when the feature meaningfully helps your app and you accept a higher migration burden
- avoid depending on `Internal` repo structure for application code

## Chapter Map

The manual is organized from first-contact workflows into deeper application architecture.

1. [01 Getting Started](01-getting-started.md)
   shortest path from install and prerequisites to the first working app
2. [02 GWC Workflows](02-gwc-workflows.md)
   the canonical `gwc` command surface for development, builds, tests, release, and runner config
3. [03 App Shapes](03-app-shapes.md)
   supported adoption paths for client-only, routed, SSR, forms-heavy, and static or PWA-oriented apps
4. [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)
   the core component model, rendering flow, hooks, transitions, overlays, async helpers, and worker tasks
5. [05 HTML Authoring](05-html-authoring.md)
   typed builders, shorthand helpers, semantic markup, props, forms, and custom elements
6. [06 State And Reactivity](06-state-and-reactivity.md)
   local state, reducers, context, atoms, derived values, selectors, snapshots, and reactive regions
7. [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
   typed resources, shared cache, imperative fetch, optimistic UI, and offline replay
8. [08 Routing](08-routing.md)
   routers, params, query helpers, contracts, loaders, redirects, guards, and metadata
9. [09 SSR And Hydration](09-ssr-and-hydration.md)
   request-time HTML, bootstrap payloads, hydration reuse, and server-integration boundaries
10. [10 Browser Interop And Workers](10-browser-interop-and-workers.md)
    browser APIs, workers, message channels, cross-tab coordination, and multi-window patterns
11. [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md)
    form ownership, accessible interaction patterns, announcements, overlays, and locale-aware app flows
12. [12 Devtools Testing And Observability](12-devtools-testing-and-observability.md)
    devtools, testing helpers, diagnostics, logs, and troubleshooting workflows
13. [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)
    release artifacts, assets, deployment targets, browser support, and PWA boundaries
14. [14 Scaling Large Codebases](14-scaling-large-codebases.md)
    package layout, ownership rules, validation strategy, and CI-ready app structure
15. [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md)
    framework scope, policy boundaries, stability tradeoffs, scheduling notes, and extension rules
16. [16 API Browser](16-api-browser.md)
    dense package, function, handle, and parameter-object lookup across the public surface

## Shared Chapter Template

Major chapters in this manual follow one common structure so readers can move between topics without relearning the documentation style.

Unless a chapter is a small appendix, it should include:

1. purpose and audience
2. feature or workflow overview
3. minimal example
4. production-shaped example
5. scale-up example for larger apps or teams
6. grouped API reference notes or API-family tables
7. design notes and boundaries
8. common failure modes
9. validation
10. topic pagination

Practical authoring rule:

- start with the smallest working path
- explain the public API after the reader has the shape in mind
- add the design boundaries before the reader cargo-cults the pattern into a larger codebase

## Cross-Linking Rules

The manual is designed to be used as a connected reference set rather than as isolated files.

Each major chapter should cross-link in three directions:

- backward to the onboarding or workflow chapter that prepares the reader for the feature
- sideways to adjacent package chapters that commonly compose with the current feature
- outward to the strongest existing examples and source docs for the same capability

Use these link priorities:

1. link to the most relevant manual chapter first
2. link to the best example or example family second
3. link to the source doc or policy doc third when boundary detail matters

Preferred cross-link patterns:

- feature chapter to workflow chapter when the feature changes the app lifecycle
- feature chapter to app-shapes chapter when the feature changes architecture choice
- API chapter to scaling chapter when local patterns stop being sufficient
- implementation tutorial to policy doc when stability, non-goals, or ownership boundaries matter

Avoid these cross-link failures:

- linking only to source docs and skipping the relevant manual chapter
- linking to too many examples without naming which one proves the intended pattern
- linking to `internal/` docs as if they were application-authoring guidance
- linking to experimental surfaces without labeling them as experimental

## Consolidation Map

Skip this section on first read.

This manual is now the consolidated home for the former top-level app-authoring docs. The old topics land in the manual this way:

- Orientation and workflow material such as `START_HERE.md`, `ONBOARDING.md`, `WORKFLOWS.md`, `WALKTHROUGHS.md`, `GWC.md`, `RUNNER_CONFIG.md`, `STARTER_OUTPUT_RULES.md`, `IDE_INTEGRATION.md`, `docs/README.md`, `REFERENCE_MAP.md`, and `REPO_MAP.md` now live in [01 Getting Started](01-getting-started.md), [02 GWC Workflows](02-gwc-workflows.md), and [03 App Shapes](03-app-shapes.md).
- Rendering and authoring material such as `HTML_SUGAR.md`, `CUSTOM_ELEMENTS.md`, `ACCESSIBILITY.md`, `OVERLAYS.md`, `ERROR_BOUNDARIES.md`, `PARALLEL_REGION_AUTHORING.md`, `PARALLEL_REGION_TROUBLESHOOTING.md`, and `HOT_RELOAD.md` now live in [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md), [05 HTML Authoring](05-html-authoring.md), [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md), and [12 Devtools Testing And Observability](12-devtools-testing-and-observability.md).
- State and data material such as `STATE_ARCHITECTURE.md`, `SCALING_LOCAL_STATE_WITH_USE_REDUCER.md`, `FINE_GRAINED_REACTIVITY.md`, `CACHE.md`, `DATA_LOADING_AND_MUTATION_ARCHITECTURE.md`, `OFFLINE_MUTATIONS.md`, `RPC_TRANSPORT.md`, and `WORKSPACE_PERSISTENCE.md` now live in [06 State And Reactivity](06-state-and-reactivity.md), [07 Data Loading And Mutations](07-data-loading-and-mutations.md), and [14 Scaling Large Codebases](14-scaling-large-codebases.md).
- Routing, SSR, and server-boundary material such as `CLIENT_SHELL_ROUTING.md`, `ROUTE_CONTRACTS.md`, `ROUTER_AUTH.md`, `HEAD_MANAGEMENT.md`, `HYDRATION.md`, `STATE_TRANSFER.md`, `SERVER_INTEGRATION.md`, `SERVER_ACTIONS.md`, `SERVER_FUNCTIONS.md`, `STREAMING_SSR.md`, `AUTH_AND_SESSION_INTEGRATION.md`, and `SECURITY.md` now live in [08 Routing](08-routing.md), [09 SSR And Hydration](09-ssr-and-hydration.md), [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md), and [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md).
- Browser, worker, and platform material such as `INTEROP.md`, `WORKERS.md`, `WORKER_POOLS_VS_LANES.md`, `CROSS_TAB.md`, `MULTI_SURFACE.md`, `MULTI_CLIENTS.md`, `MULTITHREADED_RUNTIME.md`, `MULTITHREADED_RUNTIME_TODO.md`, `VIRTUALIZATION.md`, and `SERVER_INTERACTIVE.md` now live in [10 Browser Interop And Workers](10-browser-interop-and-workers.md) and [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md).
- Deployment and release material such as `ASSETS.md`, `DEPLOYMENT_TARGETS.md`, `PRERENDER.md`, `BROWSER_SUPPORT.md`, `PWA.md`, `WASM_RELEASES.md`, `BUILD_EXPERIMENTS.md`, `CODE_SPLITTING.md`, and `CONFIGURATION.md` now live in [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md) together with the workflow material in [02 GWC Workflows](02-gwc-workflows.md).
- Operations and governance material such as `TESTING.md`, `OBSERVABILITY.md`, `LOGGING.md`, `ACTIONABLE_ERRORS.md`, `TROUBLESHOOTING.md`, `API_POLICY.md`, `FRAMEWORK_SCOPE.md`, `ECOSYSTEM.md`, `PUBLIC_API_CONVENTIONS.md`, `PRODUCTION_CORRECTNESS.md`, `SCHEDULING.md`, `COMPARISONS.md`, `ADOPTION.md`, `ADOPTION_MATURITY.md`, `ENTERPRISE_PILOT.md`, `TEAM_CONVENTIONS.md`, and `PROJECT_STRUCTURE.md` now live in [12 Devtools Testing And Observability](12-devtools-testing-and-observability.md), [14 Scaling Large Codebases](14-scaling-large-codebases.md), and [15 Design Notes And Boundaries](15-design-notes-and-boundaries.md).
- Historical or planning-only notes such as `TODO.md`, `ATLAS_COMMERCE_OS_TODO.md`, and `REFERENCE_MANUAL_TODO.md` are intentionally not separate reference surfaces anymore. Any durable guidance from them belongs in the chapters above; the rest is obsolete process state.

## Manual Principles

This manual follows a few deliberate rules:

- public package surface first, repo internals second
- `gwc` workflows first, ad hoc shell glue second
- examples as proof points, not as hidden architecture requirements
- explicit design boundaries instead of hand-wavy feature claims
- incremental adoption instead of assuming every app needs every package on day one

## Current Boundary

This manual does claim:

- the current supported authoring path runs through documented public packages and documented `gwc` workflows
- examples, workflow docs, and package docs can be synthesized into one clearer application-authoring guide
- design notes such as stability, scope, scheduling, and ownership belong alongside tutorials, not only in policy files

This manual does not claim:

- that every public surface is equally mature
- that every app should adopt SSR, routing, PWA features, or companion packages immediately
- that `internal/` packages are valid application API just because they exist in the repo
- that the example catalog is a starter-generator substitute

## Validation

Use this quick entry-point validation before you follow the deeper chapter paths:

```powershell
go run ./tools/gwc doctor
go run ./tools/gwc examples
go run ./tools/gwc test -lane unit
```

Why these are the right manual-index checks:

- `doctor` confirms the local toolchain and browser-oriented prerequisites before you copy app code from later chapters
- `examples` confirms the repo can enumerate the supported example catalog that the manual keeps linking back to
- `test -lane unit` gives one fast repo-health signal before you widen into wasm, hydration, browser, release, or deployment-specific validation

## Topic Pagination

Use the ordered chapter sequence in this manual as the pagination path.

- Previous topic in the cycle: [16 API Browser](16-api-browser.md)
- Start with topic 1: [01 Getting Started](01-getting-started.md)
- Next topic in the cycle: [01 Getting Started](01-getting-started.md)
