# Browser Support

This page defines the intended browser-support and compatibility policy for GoWebComponents.

Use it when deciding which browsers your app can target safely, which features require capability checks, and what the framework assumes from a browser before `js/wasm` startup is expected to work.

## Support Matrix

The project targets current evergreen desktop and mobile browsers.

The intended minimum support policy is:

- latest stable Chrome and Chromium-derived desktop browsers
- latest stable Edge
- latest stable Firefox
- latest stable Safari on macOS
- latest stable mobile Safari on iOS or iPadOS
- latest stable Chrome on Android

The repo should treat these as the release-gate browser families for documented production workflows.

Older browser versions, long-outdated embedded webviews, and legacy non-evergreen engines are not part of the intended support baseline unless a future release explicitly says otherwise.

## Required Feature Baseline

Compatibility should be defined by capabilities, not by the vague label "modern browser."

The core `js/wasm` runtime and documented workflows assume:

- WebAssembly support sufficient to load and run Go's `js/wasm` output
- ES2015-plus JavaScript support needed by the matching `wasm_exec.js`
- DOM APIs for element creation, mutation, event dispatch, and selector lookup
- History and location APIs for browser-router navigation
- `fetch` and promise-based async behavior for the documented fetch flows
- `addEventListener`, timers, and microtask scheduling expected by the runtime and browser integrations
- storage APIs such as `localStorage` when apps choose features that persist browser state

Higher-level packages may additionally depend on:

- `IntersectionObserver` or `ResizeObserver` for specific interop and media patterns
- `BroadcastChannel`, storage events, or equivalent fallbacks for cross-tab workflows
- worker support for the worker helper surface
- Clipboard, media-query, or other targeted APIs only when the application enables those features

## Capability Expectations

The framework surface divides into three categories:

- core rendering, hydration, routing, forms, and typed HTML, which rely on the baseline features above
- optional advanced integrations such as workers, cross-tab sync, offline mutation replay, and richer interop wrappers, which require capability checks or graceful fallback decisions
- deployment and delivery features such as prerender, SSR, and static assets, which primarily depend on the server or build environment and only secondarily on browser capabilities

Applications should treat advanced browser APIs as opt-in features, not as guaranteed ambient availability.

## Mobile Safari Priority

Mobile Safari is part of the intended support baseline, not an afterthought.

That means memory pressure, wasm startup cost, storage quirks, viewport behavior, touch input, and media-loading constraints on iOS should be considered release-relevant for any workflow that claims broad browser support.

## Polyfills And Shims

The project does not currently ship a framework-managed polyfill bundle.

The intended stance is:

- core packages should document the browser features they assume
- applications may provide their own targeted polyfills or shims when they choose to support environments below the documented baseline
- unsupported environments are not automatically elevated into the support matrix just because an application adds its own compatibility layer

This keeps the framework contract explicit:

- GoWebComponents documents the baseline
- applications decide whether extra compatibility cost is worth carrying
- release notes should call out new browser-feature dependencies when they materially affect that decision

## Progressive Enhancement Boundaries

Not every workflow needs identical behavior when browser capabilities are limited.

The intended progressive-enhancement rules are:

- prerendered or SSR-delivered HTML should still provide meaningful static content when JavaScript or `js/wasm` startup is unavailable
- interactive hydration, router ownership, and browser-only state flows require JavaScript and wasm support
- optional features such as workers, offline replay, storage-backed cross-tab state, and richer interop helpers should degrade behind capability checks instead of crashing the route shell
- applications should separate essential content delivery from optional enhanced interaction whenever a route may be delivered as SSR or prerendered HTML

This means the minimum graceful path is usually:

- content renders on the server or at build time
- core navigation and interaction activate only when the browser meets the runtime baseline
- optional enhancements attach only when their specific APIs exist

## Workers, Offline Features, And Advanced APIs

Higher-level browser integrations are intentionally not part of the universal baseline.

The intended compatibility model is:

- workers require worker support and should expose clear unavailable handling when the API is missing
- offline mutation replay, cross-tab sync, and storage-backed flows require the relevant storage, event, or browser-channel APIs and should be treated as optional enhancements
- media observers, clipboard access, popup coordination, and similar interop features require targeted capability checks at the application boundary
- applications should choose explicit fallback behavior per feature instead of assuming every supported browser offers the same advanced APIs

Recommended application approach:

- gate advanced integrations behind capability detection
- keep the base route usable without optional APIs when practical
- surface reduced-functionality messaging when a feature is intentionally unavailable

## Unsupported Browsers And Degraded Modes

Applications should decide what unsupported means for the user, not leave it implicit.

Recommended fallback policy:

- if prerendered or SSR content exists, prefer serving readable static content over a broken interactive shell
- if the full application cannot run, show a clear unsupported-browser message instead of silent failure
- distinguish between fully unsupported browsers and partially supported browsers where only optional features are unavailable
- provide direct fallback actions when possible, such as a plain form post, a non-enhanced document view, or a link to a supported environment

The framework should help applications fail clearly, but it does not yet ship a dedicated unsupported-browser UI primitive.

## Release And Support Policy

Browser-support changes are release-policy decisions, not incidental implementation details.

The intended release policy is:

- support-matrix changes must be announced in release notes and reflected in `docs/API_POLICY.md`, `docs/MIGRATIONS.md`, and this document
- narrowing the supported browser matrix should require evidence from validation, defect history, or platform deprecation rather than casual cleanup
- a browser family or major capability should not be dropped silently in a patch release
- when a supported target is being removed, the project should provide notice through the normal deprecation and migration path before the release that makes the narrower baseline effective

This keeps browser compatibility aligned with the broader semver and migration contract instead of hiding it in issue comments or commit history.

## Current Boundary

This document defines the intended baseline and target browser families.

It does not yet claim:

- automated compatibility CI across that whole matrix
- complete validation coverage for every optional browser capability or device class

Those remain separate backlog work.
