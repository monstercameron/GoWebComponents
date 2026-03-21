# GoWebComponents Documentation

This directory contains the project-level documentation that is still useful after the runtime and tooling cleanup on 2026-03-14.

## Files

### `START_HERE.md`
Recommended entrypoint for new adopters, including the preferred package surface, first example picks, and the modern path through the docs.

### `WORKFLOWS.md`
Task-oriented guidance for common developer jobs such as building a client-only app, adding routing, adding SSR, testing, shipping a wasm build, and debugging hydration.

### `WALKTHROUGHS.md`
End-to-end app-shape walkthroughs that connect the current examples and package surface into realistic starting paths.

### `REFERENCE_MAP.md`
Cross-links for concepts, public APIs, examples, production caveats, and related docs.

### `TROUBLESHOOTING.md`
Common setup and runtime failure guidance for wasm builds, `wasm_exec.js`, example serving, hydration, routing, and browser interop.

### `FORMS.md`
Current supported `ui.UseForm` modes, server-post conventions, redirect-after-submit guidance, and the boundary between shipped form state helpers and application-owned server workflows.

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

### `BUILD_EXPERIMENTS.md`
Current intended experiment matrix for wasm build flags, size measurements, startup-cost tradeoffs, and compatibility-sensitive optimization work.

### `COMPILER_ASSISTED_FEATURES.md`
Current project direction for compiler-assisted features, including the decision to keep plain Go plus ordinary `go build` as the default path while limiting compiler work to opt-in experiments.

### `ONBOARDING.md`
Current intended onboarding, prerequisites, choose-your-path, starter-upgrade, and inner-loop workflow guidance for new adopters.

### `API_POLICY.md`
Stability tiers, semver rules, deprecation lifecycle, migration requirements, and latest-major support expectations.

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

## Where The Core Lives

## Start Here

If you are new to the repo, use this order:

1. [START_HERE.md](START_HERE.md)
2. [WORKFLOWS.md](WORKFLOWS.md)
3. [WALKTHROUGHS.md](WALKTHROUGHS.md)
4. [REFERENCE_MAP.md](REFERENCE_MAP.md)
5. [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

The old docs referred to a `/fiber` directory. That is stale.

The current implementation core is:

- `internal/runtime/`
- `internal/platform/jsdom/`
- `dom/`, `hooks/`, `render/`, `state/`, `fetch/`, `router/`

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

## Related Docs

- [../README.md](../README.md)
- [../CHANGELOG.md](../CHANGELOG.md)
- [START_HERE.md](START_HERE.md)
- [WORKFLOWS.md](WORKFLOWS.md)
- [WALKTHROUGHS.md](WALKTHROUGHS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md)
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
- [BUILD_EXPERIMENTS.md](BUILD_EXPERIMENTS.md)
- [COMPILER_ASSISTED_FEATURES.md](COMPILER_ASSISTED_FEATURES.md)
- [ONBOARDING.md](ONBOARDING.md)
- [API_POLICY.md](API_POLICY.md)
- [ACCESSIBILITY.md](ACCESSIBILITY.md)
- [OVERLAYS.md](OVERLAYS.md)
- [I18N.md](I18N.md)
- [HEAD_MANAGEMENT.md](HEAD_MANAGEMENT.md)
- [ERROR_BOUNDARIES.md](ERROR_BOUNDARIES.md)
- [MIGRATIONS.md](MIGRATIONS.md)
- [../examples/README.md](../examples/README.md)
- [../test/README.md](../test/README.md)
- [../tools/README.md](../tools/README.md)
