# GoWebComponents Documentation

This directory contains the project-level documentation that is still useful after the runtime and tooling cleanup on 2026-03-14.

## At A Glance

- This directory is the prose index for the current GoWebComponents runtime, platform guidance, and adoption surface.
- New adopters should start with the guided entry docs rather than reading every file alphabetically.
- Core package and runtime behavior now live primarily under `ui/`, `html/`, `state/`, `fetch/`, `router/`, and `internal/runtime/`.
- This index is most useful when you need to choose the next document quickly or confirm where a topic belongs.

## Quick Reading Guide

Read these first when you are new to the repo:

1. `START_HERE.md`
2. `WORKFLOWS.md`
3. `WALKTHROUGHS.md`
4. `REFERENCE_MAP.md`
5. `TROUBLESHOOTING.md`

Jump directly to these when you already know the question category:

- runtime and rendering behavior: `HYDRATION.md`, `SCHEDULING.md`, `PRODUCTION_CORRECTNESS.md`
- app integration: `SERVER_INTEGRATION.md`, `DEPLOYMENT_TARGETS.md`, `ASSETS.md`, `CONFIGURATION.md`
- product-policy and scope: `API_POLICY.md`, `FRAMEWORK_SCOPE.md`, `ECOSYSTEM.md`, `COMPARISONS.md`
- user-facing browser concerns: `FORMS.md`, `ACCESSIBILITY.md`, `OVERLAYS.md`, `I18N.md`, `PWA.md`
- local-state scaling guidance: `SCALING_LOCAL_STATE_WITH_USE_REDUCER.md`
- shared-state architecture guidance: `STATE_ARCHITECTURE.md`
- async read and mutation architecture guidance: `DATA_LOADING_AND_MUTATION_ARCHITECTURE.md`
- auth and session integration guidance: `AUTH_AND_SESSION_INTEGRATION.md`
- business-form workflow guidance: `BUSINESS_APP_FORM_RECIPES.md`
- server-owned mutation contract guidance: `SERVER_ACTIONS.md`
- broader typed server-call guidance: `SERVER_FUNCTIONS.md`
- typed route-definition scope and tooling direction: `ROUTE_CONTRACTS.md`
- long-lived workspace restore and retention guidance: `WORKSPACE_PERSISTENCE.md`
- virtualization direction and ownership guidance: `VIRTUALIZATION.md`
- troubleshooting and diagnostics: `ACTIONABLE_ERRORS.md`, `LOGGING.md`, `OBSERVABILITY.md`, `TROUBLESHOOTING.md`

## Files

### `START_HERE.md`
Recommended entrypoint for new adopters, including the preferred package surface, first example picks, and the modern path through the docs.

### `WORKFLOWS.md`
Task-oriented guidance for common developer jobs such as building a client-only app, adding routing, adding SSR, testing, shipping a wasm build, and debugging hydration.

### `TESTING.md`
Defines the intended first-party testing surface for consumers, including the decision to keep testing support in a companion module with focused helper packages instead of one monolithic core package.

### `WALKTHROUGHS.md`
End-to-end app-shape walkthroughs that connect the current examples and package surface into realistic starting paths.

### `REFERENCE_MAP.md`
Cross-links for concepts, public APIs, examples, production caveats, and related docs.

### `MULTI_CLIENTS.md`
Short proposal for coordinating multiple sovereign browser `js/wasm` clients by layering one typed message contract over the existing cross-tab and multi-window transports.

### `RPC_TRANSPORT.md`
Current product-boundary decision for typed browser RPC: keep protobuf-backed unary and streaming transport out of core `interop` and in a dedicated companion package until the transport, codegen, auth, diagnostics, and example story are proven.

### `TROUBLESHOOTING.md`
Common setup and runtime failure guidance for wasm builds, `wasm_exec.js`, example serving, hydration, routing, and browser interop.

### `ACTIONABLE_ERRORS.md`
Audit of the highest-friction framework failures, the current diagnostic gaps, and the stable identifiers or remediation anchors runtime errors should point to.

### `FORMS.md`
Current supported `ui.UseForm` modes, server-post conventions, redirect-after-submit guidance, and the boundary between shipped form state helpers and application-owned server workflows.

### `SCALING_LOCAL_STATE_WITH_USE_REDUCER.md`
Guidance for the point where feature-local state grows beyond a few `ui.UseState` calls and should be wrapped behind an app-specific `ui.UseReducer` workflow hook.

### `STATE_ARCHITECTURE.md`
Recommended split between local hooks, reducer-owned workflows, subtree context, shared atoms, derived selectors, fetch-owned async data, and snapshot persistence for non-trivial applications.

### `DATA_LOADING_AND_MUTATION_ARCHITECTURE.md`
Recommended split between route loaders, typed component resources, shared cache, form-owned mutations, optimistic updates, offline replay, and revalidation paths for non-trivial applications.

### `AUTH_AND_SESSION_INTEGRATION.md`
Recommended split between server-owned sessions, client-safe auth hints, guarded navigation, same-origin API calls, logout invalidation, and the explicit bearer-token escape hatch.

### `BUSINESS_APP_FORM_RECIPES.md`
Task-oriented form workflow guidance for validation, field-error projection, submit pending UX, redirects, uploads, and server-authoritative mutation handling in internal applications.

### `SERVER_ACTIONS.md`
First-class server-owned action contract for form submissions, including the shared outcome model for progressive HTML posts and hydrated enhanced submits.

### `SERVER_FUNCTIONS.md`
First-class application-owned server-function model for typed non-form mutations and queries, kept above `fetch` and below transport-specific RPC.

### `ROUTE_CONTRACTS.md`
Current route-contract scope decision: keep typed route definitions and reverse routing in the runtime, while leaving manifest generation, prerender enumeration, and metadata tooling in companion or application-owned layers until a stronger shared need is proven.

### `WORKSPACE_PERSISTENCE.md`
Current persistence contract for long-lived workspaces, including per-user browser namespaces, restore order, bounded history retention, and the durable state that must be purged on sign-out or user switch.

### `VIRTUALIZATION.md`
Current ownership and rollout direction for list and table virtualization, including the decision to start with a supported companion-package surface rather than silently expanding core `ui`.

### `README.md`
High-level documentation index and pointers to the current runtime, examples, tests, and tools.

### `PERFORMANCE.md`
Current performance notes, benchmark entrypoints, and the results that were actually measured in the current codebase.

### `PRODUCTION_CORRECTNESS.md`
Current minimum production-correctness bar for the core runtime, plus the runtime-level composed-flow, churn, and overlapping-update coverage that now backs it.

### `SCHEDULING.md`
Current urgent-versus-transition scheduler contract, `ui.UseTransition` pending semantics, current non-interruptible limits, and how scheduling interacts with route loaders and async boundaries.

### `HYDRATION.md`
Current shipped hydration contract: DOM reuse rules, matching boundaries, deferred effects and subscriptions, subtree fallback behavior, and the runtime coverage that backs those claims.

### `STATE_TRANSFER.md`
Current bootstrap-state classification, merge semantics, hydration ownership rules, and transport guidance for `ui.SSRBootstrap`.

### `STREAMING_SSR.md`
Current project direction for streaming SSR as a post-hydration milestone, plus the first planned route-loader streaming model.

### `SERVER_INTEGRATION.md`
Current canonical Go HTTP integration story for SSR apps, including request pipeline shape, middleware ordering, bootstrap emission, and backend API integration patterns.

### `DEPLOYMENT_TARGETS.md`
Current deployment-target summary and adapter expectations for static hosting, single-server SSR, reverse-proxy fronted deployments, and split SSR/API topologies.

### `OBSERVABILITY.md`
Current intended observability contract for SSR requests, hydration, navigation, correlation ids, and structured runtime event naming.

### `LOGGING.md`
Current intended structured logging contract, including stable log domains and development-versus-production output expectations.

### `PRERENDER.md`
Current intended static prerender contract, including how prerender differs from request-time SSR and where export orchestration belongs.

### `ASSETS.md`
Current intended asset-delivery contract, including the boundary between core rendering, build tooling, manifests, and deployment conventions for SSR and prerendered apps.

### `BROWSER_SUPPORT.md`
Current intended browser-support matrix and the capability baseline expected by the `js/wasm` runtime and documented workflows.

### `PWA.md`
Current intended PWA and offline-app integration contract, including service-worker boundaries, offline cache strategy, update semantics, and deployment guidance.

### `SECURITY.md`
Current intended security, data-boundary, redaction, and supply-chain review contract for SSR, hydration, browser integration, and operational tooling.

### `CONFIGURATION.md`
Current intended runtime-configuration and public flag-transfer contract for server, browser, SSR bootstrap, and environment layering.

### `WASM_RELEASES.md`
Current intended wasm build-profile and production-flag baseline for development, CI verification, benchmarking, and release builds.

### `IDE_INTEGRATION.md`
Current intended editor-integration boundary, including the VS Code-first task story, `gopls`-compatible baseline, and the limits of first-party IDE promises.

### `BUILD_EXPERIMENTS.md`
Current intended experiment matrix for wasm build flags, size measurements, startup-cost tradeoffs, and compatibility-sensitive optimization work.

### `COMPILER_ASSISTED_FEATURES.md`
Current project direction for compiler-assisted features, including the decision to keep plain Go plus ordinary `go build` as the default path while limiting compiler work to opt-in experiments.

### `ONBOARDING.md`
Current intended onboarding, prerequisites, choose-your-path, starter-upgrade, and inner-loop workflow guidance for new adopters.

### `ADOPTION.md`
Minimum 1.0-style ecosystem baseline, including the required answers for starter path, testing, SSR, state, routing, and deployment guidance.

### `COMPARISONS.md`
Framework comparison guidance describing where GoWebComponents is intentionally different from React, Vue, Svelte, Solid, Blazor, and Qwik, plus current maturity gaps.

### `API_POLICY.md`
Stability tiers, semver rules, deprecation lifecycle, migration requirements, and latest-major support expectations.

### `ECOSYSTEM.md`
Current project stance on plugins, directives, companion packages, and when a shared extension lifecycle would be justified.

### `MIGRATIONS.md`
Release-to-release upgrade guidance, starting with the transition into the current `v3.x` public package layout.

### `HEAD_MANAGEMENT.md`
Current head-management, SEO, canonical URL, structured-data, and resource-hint guidance, including the boundary between router-managed metadata and app-owned explicit head markup.

### `HOT_RELOAD.md`
First-class hot reload guide covering the public `hotreload` package, the `tools/dev` wrappers, snapshot semantics, persistence boundaries, and example commands.

### `ERROR_BOUNDARIES.md`
Current `ui.ErrorBoundary` composition rules for nested routes and layouts, plus the shipped SSR and hydration behavior.

### `ACCESSIBILITY.md`
Current accessibility support baseline, including typed semantic markup, `ui.UseId()`, application-owned accessibility responsibilities, and current non-goals.

### `OVERLAYS.md`
Current overlay layering model, stack coordination rules, anchored-position guidance, and the boundary between framework-owned overlay behavior and application-owned placement logic.

### `I18N.md`
Current internationalization scope, locale context model, message catalog helpers, SSR bootstrap transfer, locale-aware routing guidance, and RTL directionality guidance.

### `TODO.md`
Current backlog and near-term work. This is a live backlog, not a historical archive of every idea the project has ever had.

## Start Here

If you are new to the repo, use this order:

1. [START_HERE.md](START_HERE.md)
2. [WORKFLOWS.md](WORKFLOWS.md)
3. [WALKTHROUGHS.md](WALKTHROUGHS.md)
4. [REFERENCE_MAP.md](REFERENCE_MAP.md)
5. [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

The old docs referred to a `/fiber` directory. That is stale.

## Where The Core Lives

The current implementation core is:

- `internal/runtime/`
- `internal/platform/jsdom/`
- `ui/`, `html/`, `state/`, `fetch/`, `router/`
- companion and platform packages such as `interop/`, `hotreload/`, `head/`, `pwa/`, `logging/`, and `devtools/`

If you are debugging the framework itself, start here:

1. `internal/runtime/types.go`
2. `internal/runtime/reconciler.go`
3. `internal/runtime/scheduler.go`
4. `internal/runtime/hooks.go`
5. `internal/runtime/runtime.go`

## Current Validation Status

As of 2026-03-14:

- `go test ./internal/runtime` passes
- Native `internal/runtime` statement coverage is `100%`
- Playwright component, integration, and deep state stress suites pass
- Separate `js/wasm` tests and benchmarks exist for wasm-only runtime and adapter code

## Documentation Maintenance Rules

- keep this index focused on orientation, not full topic duplication
- prefer linking to the authoritative topic page instead of expanding summaries here indefinitely
- update this file when the package surface, preferred reading order, or documentation map changes materially
- remove stale package references promptly when directory layouts or public entrypoints move

## Related Docs

- [../README.md](../README.md)
- [../CHANGELOG.md](../CHANGELOG.md)
- [START_HERE.md](START_HERE.md)
- [WORKFLOWS.md](WORKFLOWS.md)
- [TESTING.md](TESTING.md)
- [WALKTHROUGHS.md](WALKTHROUGHS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md)
- [MULTI_CLIENTS.md](MULTI_CLIENTS.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [FORMS.md](FORMS.md)
- [PRODUCTION_CORRECTNESS.md](PRODUCTION_CORRECTNESS.md)
- [SCHEDULING.md](SCHEDULING.md)
- [HYDRATION.md](HYDRATION.md)
- [STATE_TRANSFER.md](STATE_TRANSFER.md)
- [STREAMING_SSR.md](STREAMING_SSR.md)
- [SERVER_INTEGRATION.md](SERVER_INTEGRATION.md)
- [OBSERVABILITY.md](OBSERVABILITY.md)
- [LOGGING.md](LOGGING.md)
- [PRERENDER.md](PRERENDER.md)
- [ASSETS.md](ASSETS.md)
- [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md)
- [PWA.md](PWA.md)
- [SECURITY.md](SECURITY.md)
- [CONFIGURATION.md](CONFIGURATION.md)
- [WASM_RELEASES.md](WASM_RELEASES.md)
- [IDE_INTEGRATION.md](IDE_INTEGRATION.md)
- [BUILD_EXPERIMENTS.md](BUILD_EXPERIMENTS.md)
- [COMPILER_ASSISTED_FEATURES.md](COMPILER_ASSISTED_FEATURES.md)
- [ONBOARDING.md](ONBOARDING.md)
- [ACTIONABLE_ERRORS.md](ACTIONABLE_ERRORS.md)
- [ADOPTION.md](ADOPTION.md)
- [COMPARISONS.md](COMPARISONS.md)
- [API_POLICY.md](API_POLICY.md)
- [ECOSYSTEM.md](ECOSYSTEM.md)
- [SERVER_ACTIONS.md](SERVER_ACTIONS.md)
- [SERVER_FUNCTIONS.md](SERVER_FUNCTIONS.md)
- [ACCESSIBILITY.md](ACCESSIBILITY.md)
- [OVERLAYS.md](OVERLAYS.md)
- [I18N.md](I18N.md)
- [HEAD_MANAGEMENT.md](HEAD_MANAGEMENT.md)
- [ERROR_BOUNDARIES.md](ERROR_BOUNDARIES.md)
- [MIGRATIONS.md](MIGRATIONS.md)
- [../examples/README.md](../examples/README.md)
- [../test/README.md](../test/README.md)
- [../tools/README.md](../tools/README.md)
