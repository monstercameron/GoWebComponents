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
- troubleshooting and diagnostics: `ACTIONABLE_ERRORS.md`, `LOGGING.md`, `OBSERVABILITY.md`, `TROUBLESHOOTING.md`

## Current Status

- This example-0 bundle mirrors the current project docs set and should stay aligned with the source docs under `docs/`.
- The quickest path through the bundle is still `start-here.md`, then `common-workflows.md`, `end-to-end-walkthroughs.md`, and `reference-map.md`.
- The file map below now covers the current mirrored docs instead of a partial subset.

## Files

### Orientation And Adoption

- `start-here.md`: recommended entrypoint for new adopters, including the preferred package surface, first example picks, and the modern path through the docs.
- `common-workflows.md`: task-oriented guidance for common developer jobs such as building a client-only app, adding routing, adding SSR, testing, shipping a wasm build, and debugging hydration.
- `end-to-end-walkthroughs.md`: end-to-end app-shape walkthroughs that connect the current examples and package surface into realistic starting paths.
- `reference-map.md`: cross-links for concepts, public APIs, examples, production caveats, and related docs.
- `onboarding-and-bootstrap.md`: onboarding, prerequisites, choose-your-path guidance, and inner-loop setup for new adopters.
- `adoption-baseline.md`: minimum ecosystem readiness bar for starter path, testing, SSR, state, routing, and deployment guidance.
- `framework-comparisons.md`: where GoWebComponents is intentionally different from React, Vue, Svelte, Solid, Blazor, and Qwik.
- `api-stability-and-support-policy.md`: stability tiers, semver rules, deprecation lifecycle, and support expectations.
- `ecosystem-and-extension-model.md`: current stance on plugins, directives, companion packages, and extension lifecycle boundaries.
- `migration-guide.md`: release-to-release upgrade guidance for the current package layout.
- `gowebcomponents-todo.md`: current project backlog and near-term work.
- `documentation-push-todo.md`: documentation-program backlog for repo-wide coverage, API mapping, and cross-linking.

### Runtime, Rendering, And State

- `hydration.md`: shipped hydration contract, DOM reuse rules, matching boundaries, and fallback behavior.
- `scheduling.md`: urgent-versus-transition scheduler contract and route-loader interaction.
- `production-correctness.md`: current production-correctness bar and the runtime coverage that backs it.
- `state-transfer.md`: bootstrap-state classification, merge semantics, hydration ownership rules, and transport guidance.
- `streaming-ssr.md`: current streaming SSR direction and the planned route-loader streaming model.
- `fine-grained-reactivity.md`: design notes for reactive-region-style updates and the boundary with the core hooks model.
- `error-boundaries.md`: `ui.ErrorBoundary` composition rules for nested routes, SSR, and hydration.
- `forms.md`: supported `ui.UseForm` modes, server-post conventions, and redirect-after-submit guidance.
- `html-sugar.md`: current HTML helper surface and the boundary between ergonomic sugar and core rendering primitives.
- `shared-cache-design.md`: current cache contract, persistence boundaries, and bootstrap restore behavior.

### Platform, Integration, And Deployment

- `server-integration.md`: canonical Go HTTP integration story for SSR apps and request pipeline shape.
- `deployment-targets-and-adapter-expectations.md`: supported deployment shapes and adapter expectations.
- `assets.md`: current asset-delivery contract across rendering, manifests, tooling, and deployment.
- `runtime-configuration-and-feature-flags.md`: runtime configuration and public flag-transfer guidance.
- `wasm-build-profiles-and-release-engineering.md`: current wasm build profiles, release defaults, and manifest handoff.
- `wasm-build-experiments.md`: experiment matrix for wasm flags, size measurements, and startup-cost tradeoffs.
- `browser-support.md`: browser-support matrix and capability baseline for the `js/wasm` runtime.
- `prerender.md`: current static prerender contract and the boundary with request-time SSR.
- `pwa-and-offline-app-support.md`: PWA integration, service-worker boundaries, offline cache strategy, and update semantics.
- `offline-mutation-queueing.md`: current offline mutation queue design and ownership boundaries.
- `worker-guidance.md`: current dedicated-worker contract, typed worker tasks, and service-worker boundary guidance.
- `cross-tab-synchronization.md`: current cross-tab coordination model and its transport assumptions.
- `multi-client-coordination.md`: typed coordination model for multiple sovereign browser clients.
- `multi-surface-coordination.md`: current direction for coordinating apps that span more than one browser surface.
- `recommended-project-structure.md`: recommended package and application layout for real projects.
- `interop-guidance.md`: browser interop, JS bridge boundaries, and host integration patterns.
- `custom-elements.md`: custom-element interop guidance and ownership boundaries.
- `server-interactive-experiments.md`: experimental server-interactive direction and current limits.
- `router-async-guards-and-auth-design.md`: router auth, async guard, and protected-navigation design guidance.

### Diagnostics, Observability, And Operations

- `troubleshooting.md`: setup and runtime failure guidance for wasm builds, hydration, routing, and browser interop.
- `actionable-errors-and-diagnostics.md`: high-friction failure audit and the diagnostic anchors runtime errors should expose.
- `observability.md`: current observability contract for SSR requests, hydration, navigation, correlation ids, and runtime event naming.
- `logging.md`: current structured logging contract, log domains, and development-versus-production expectations.
- `performance-notes.md`: current performance notes, benchmark entrypoints, and measured results.
- `security-compliance-and-governance.md`: security, data-boundary, redaction, and supply-chain review guidance.

### UX, Accessibility, And Product Surface

- `accessibility-guidance.md`: current accessibility baseline, `ui.UseId()`, and application-owned responsibilities.
- `overlay-layering-and-portal-coordination.md`: overlay layering rules, stack coordination, and placement ownership.
- `head-management-and-seo-surface.md`: head-management, SEO, canonical URL, and metadata guidance.
- `internationalization-and-localization.md`: locale context, catalog helpers, SSR bootstrap transfer, and RTL guidance.
- `framework-scope.md`: product boundary for what the framework owns versus what applications own.

### Architecture And Experimental Direction

- `compiler-assisted-features.md`: project direction for compiler-assisted features while keeping ordinary `go build` as the default path.
- `code-splitting-and-bundle-loading.md`: current code-splitting model and bundle-loading constraints.
- `atlas-commerce-os-project-todo.md`: Atlas-specific backlog and integration planning notes.

## Start Here

If you are new to the repo, use this order:

1. [START_HERE.md](start-here.md)
2. [WORKFLOWS.md](common-workflows.md)
3. [WALKTHROUGHS.md](end-to-end-walkthroughs.md)
4. [REFERENCE_MAP.md](reference-map.md)
5. [TROUBLESHOOTING.md](troubleshooting.md)

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
- [START_HERE.md](start-here.md)
- [WORKFLOWS.md](common-workflows.md)
- [TESTING.md](testing-surface.md)
- [WALKTHROUGHS.md](end-to-end-walkthroughs.md)
- [REFERENCE_MAP.md](reference-map.md)
- [MULTI_CLIENTS.md](multi-client-coordination.md)
- [MULTI_SURFACE.md](multi-surface-coordination.md)
- [TROUBLESHOOTING.md](troubleshooting.md)
- [FORMS.md](forms.md)
- [PRODUCTION_CORRECTNESS.md](production-correctness.md)
- [CACHE.md](shared-cache-design.md)
- [SCHEDULING.md](scheduling.md)
- [HYDRATION.md](hydration.md)
- [STATE_TRANSFER.md](state-transfer.md)
- [STREAMING_SSR.md](streaming-ssr.md)
- [SERVER_INTEGRATION.md](server-integration.md)
- [DEPLOYMENT_TARGETS.md](deployment-targets-and-adapter-expectations.md)
- [OBSERVABILITY.md](observability.md)
- [LOGGING.md](logging.md)
- [PRERENDER.md](prerender.md)
- [ASSETS.md](assets.md)
- [BROWSER_SUPPORT.md](browser-support.md)
- [CODE_SPLITTING.md](code-splitting-and-bundle-loading.md)
- [CROSS_TAB.md](cross-tab-synchronization.md)
- [CUSTOM_ELEMENTS.md](custom-elements.md)
- [PWA.md](pwa-and-offline-app-support.md)
- [WORKERS.md](worker-guidance.md)
- [SECURITY.md](security-compliance-and-governance.md)
- [CONFIGURATION.md](runtime-configuration-and-feature-flags.md)
- [WASM_RELEASES.md](wasm-build-profiles-and-release-engineering.md)
- [BUILD_EXPERIMENTS.md](wasm-build-experiments.md)
- [COMPILER_ASSISTED_FEATURES.md](compiler-assisted-features.md)
- [ONBOARDING.md](onboarding-and-bootstrap.md)
- [ACTIONABLE_ERRORS.md](actionable-errors-and-diagnostics.md)
- [ADOPTION.md](adoption-baseline.md)
- [COMPARISONS.md](framework-comparisons.md)
- [API_POLICY.md](api-stability-and-support-policy.md)
- [ECOSYSTEM.md](ecosystem-and-extension-model.md)
- [FRAMEWORK_SCOPE.md](framework-scope.md)
- [HTML_SUGAR.md](html-sugar.md)
- [INTEROP.md](interop-guidance.md)
- [OFFLINE_MUTATIONS.md](offline-mutation-queueing.md)
- [PROJECT_STRUCTURE.md](recommended-project-structure.md)
- [ROUTER_AUTH.md](router-async-guards-and-auth-design.md)
- [SERVER_INTERACTIVE.md](server-interactive-experiments.md)
- [ACCESSIBILITY.md](accessibility-guidance.md)
- [OVERLAYS.md](overlay-layering-and-portal-coordination.md)
- [I18N.md](internationalization-and-localization.md)
- [HEAD_MANAGEMENT.md](head-management-and-seo-surface.md)
- [ERROR_BOUNDARIES.md](error-boundaries.md)
- [FINE_GRAINED_REACTIVITY.md](fine-grained-reactivity.md)
- [HOT_RELOAD.md](hot-reload.md)
- [MIGRATIONS.md](migration-guide.md)
- [TODO.md](gowebcomponents-todo.md)
- [DOCUMENTATION_PUSH_TODO.md](documentation-push-todo.md)
- [../examples/README.md](../examples/README.md)
- [../test/README.md](../test/README.md)
- [../tools/README.md](../tools/README.md)
