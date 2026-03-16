# GoWebComponents TODO

This backlog tracks missing, incomplete, or experimental framework capabilities compared to mature UI frameworks such as React, Svelte, Vue, Solid, Blazor, and Qwik.

Organization rules for this file:

- Open work comes first.
- Sections are numbered and grouped by problem area instead of by historical implementation phase.
- Each TODO is an action statement followed by concrete scope notes.
- Closely related ideas are merged to avoid duplicate backlog entries.
- Completed milestones are summarized at the end so shipped work remains visible without crowding active priorities.

## 1. Documentation and Public API Surface

### Documentation hygiene

- [ ] Reconcile docs with the actual supported API surface.
	Remove or clearly label aspirational examples and stale references so the docs only describe shipped, experimental, or intentionally planned features.
- [ ] Add a framework feature matrix to the docs.
	Separate stable, experimental, and planned capabilities so adopters can see what is safe to use today.
- [ ] Clarify which packages are stable public APIs versus internal-only details.
	Document which surfaces are safe for application code and which ones may change without notice.
- [ ] Add migration notes as new primitives land.
	Explain how newer APIs replace older patterns so early adopters do not accumulate legacy usage accidentally.

### Metadata and composition model

- [ ] Finalize the SSR metadata model.
	Decide whether route-level title, description, and canonical metadata are server-owned, client-owned after hydration, or jointly reconciled.
- [ ] Add a server render path for route-managed metadata.
	Ensure router metadata can be emitted during `ui.RenderToString(...)` instead of requiring post-render DOM mutation.
- [ ] Define metadata hydration reconciliation rules.
	Prevent duplicate, stale, or leaked head tags when client resume takes over a server-rendered document.
- [ ] Decide whether slots are part of the public composition model.
	Either formalize slot-style composition with clear usage guidance or explicitly mark it out of scope in favor of ordinary children and layout patterns.

## 2. Forms, Uploads, and Secure Submission Workflows

### Server-backed forms

- [ ] Define the supported form modes.
	Document the recommended split between client-only forms, progressive-enhancement posts, JSON-backed submissions, and SSR-backed form flows.
- [ ] Add form-post destination conventions for Go handlers.
	Show how `ui.UseForm` should target `net/http` handlers, JSON endpoints, and multipart upload routes so server-backed forms are not all bespoke.
- [ ] Define redirect-after-submit semantics.
	Specify how successful submissions coordinate with router navigation, flash-style success state, and history replacement.

### CSRF and secure posting

- [ ] Add CSRF-aware form helpers for server-post workflows.
	Support token injection and transport conventions for form posts targeting Go HTTP handlers without forcing every app to hand-roll hidden fields and headers.
- [ ] Define CSRF token source and refresh rules.
	Clarify whether tokens come from SSR bootstrap, cookies, headers, or explicit server endpoints and how long-lived pages refresh them safely.

### Validation and error handling

- [ ] Add server-returned field error mapping.
	Provide a normalized way to take structured validation errors from server responses and project them back onto public form state.
- [ ] Add field-level message helpers.
	Expose touched, dirty, pending, and error helpers so inline validation and summary rendering do not require repetitive app code.
- [ ] Add submit-intent helpers.
	Support workflows such as draft save vs publish, per-button pending state, and submit-intent-specific validation without ad hoc local state.
- [ ] Define optimistic vs authoritative submit behavior.
	Clarify when forms may update UI optimistically, when they must wait for the server, and how retry/reset flows behave after partial failure.

### Uploads and SSR examples

- [ ] Add multipart and file-upload support.
	Cover `multipart/form-data`, file inputs, upload progress, cancellation, and server error reporting as first-class workflows.
- [ ] Add SSR-friendly secure form examples.
	Demonstrate form defaults, server validation round-trips, CSRF-aware submission, uploads, and post-submit redirects in a request-time rendered example.

## 3. JavaScript Interop Ergonomics

### Public interop API

- [ ] Design a first-class public JS interop package or namespace.
	Move common `syscall/js` patterns behind a stable public API so applications do not depend on raw low-level browser bindings for routine work.
- [ ] Add typed wrappers for common browser APIs.
	Cover storage, history, location, clipboard, timers, custom events, media queries, and similar APIs with predictable Go-friendly shapes.
- [ ] Add module-style interop helpers.
	Support importing a JS module, calling exported functions, and disposing module handles with a lifecycle model that fits component mount and unmount behavior.

### DOM and event integration

- [ ] Add event subscription helpers around browser APIs.
	Support window and document listeners, media-query listeners, resize observers, and intersection observers with a consistent cleanup model.
- [ ] Add element-reference based interop helpers.
	Make it easy to target a rendered DOM node for measurement, imperative focus, scrolling, and third-party widget integration without leaking raw JS handles.
- [ ] Add third-party library integration examples.
	Demonstrate how to attach a JS widget, synchronize props, and tear it down cleanly on updates and unmount.

### Safety, performance, and diagnostics

- [ ] Define interop lifetime and cleanup rules.
	Document when JS references must be released, how listeners are detached, and how cleanup behaves across unmounts, route changes, and hydration fallback.
- [ ] Add SSR-safe interop guardrails.
	Ensure browser-only helpers clearly report misuse in SSR code paths instead of failing silently or returning ambiguous zero values.
- [ ] Add structured error handling for interop failures.
	Surface missing APIs, rejected promises, serialization errors, and disposed-handle usage through typed Go errors and runtime diagnostics.
- [ ] Add interop performance guidance and batching helpers.
	Reduce repetitive boundary crossings for common DOM read, storage lookup, and event payload scenarios where `syscall/js` overhead accumulates.
- [ ] Add interop-safe serialization guidance.
	Specify how primitives, structs, maps, byte slices, and opaque JS values cross the boundary so large payloads and custom types behave predictably.
- [ ] Add high-level examples that replace raw `syscall/js` usage.
	Demonstrate storage, clipboard access, resize or media observers, and custom event integration through public helpers instead of ad hoc code.

## 4. Routing, Guards, and Auth-Aware Navigation

### Async navigation guards

- [ ] Design the async guard API surface.
	Decide how async guards are declared, how they differ from synchronous guards, and how they report allow, redirect, block, and retryable failure states.
- [ ] Define async guard cancellation and race semantics.
	Cancel stale in-flight guard work when navigation changes and ensure late results cannot commit a blocked or redirected navigation.
- [ ] Define pending-navigation UX hooks.
	Allow routes to show pending indicators, disable repeated navigation attempts, or surface a loading state while an async guard resolves.
- [ ] Add test coverage for async guard edge cases.
	Cover double-click navigation, back and forward navigation, loader-plus-guard interactions, and cleanup when guarded components unmount mid-check.

### Auth-aware routing primitives

- [ ] Define a lightweight auth context model for routed apps.
	Support a framework-level auth or session signal for route gating and conditional rendering without taking on full identity-provider responsibilities.
- [ ] Add router-level unauthorized and authorizing UI states.
	Allow routes and route groups to render explicit unauthorized, forbidden, and pending-auth content instead of forcing every app into manual redirects.
- [ ] Add route-group auth inheritance rules.
	Allow a layout or route prefix to declare a shared auth requirement so protected sections do not duplicate the same gate on every leaf route.
- [ ] Add route policy hooks above simple boolean guards.
	Support reusable access rules such as authenticated-only, role-like app policies, or custom claims-style predicates without requiring a full auth framework.
- [ ] Define auth state refresh and invalidation behavior.
	Specify how route gating reacts when auth or session state changes after initial mount, including logout, token expiry, and background refresh results.
- [ ] Define server-hydrated auth hint transfer.
	Allow SSR flows to bootstrap a minimal auth or session snapshot so the first client resume can make consistent route decisions before fresh API checks complete.
- [ ] Add return-to and post-auth navigation helpers.
	Standardize preservation of the originally requested route, query string, and intended action when a user is redirected through login or re-auth flows.
- [ ] Define loader and auth-gate ordering.
	Ensure protected routes do not start expensive data loading before an auth policy rejects the navigation, while still allowing public shell data when explicitly intended.
- [ ] Add protected-route examples and security guidance.
	Demonstrate protected sections, deferred auth resolution, unauthorized fallback UI, and post-login redirect preservation, and document that client-side gating is a UX tool rather than a full security boundary.

## 5. Data Loading, Cache Reuse, and Error Boundaries

### Shared async cache and query model

- [ ] Design a shared resource cache above `UseFetch` and `UseResource`.
	Support request deduplication, stale-while-revalidate behavior, and reuse across components without splitting the async data model into incompatible layers.
- [ ] Decide where cached async state should live.
	Keep the cache coherent with atoms and route loaders instead of creating a disconnected parallel mental model.
- [ ] Define cache key normalization rules.
	Ensure URLs, methods, query params, headers, loader args, and custom keys produce deterministic identities without surprising collisions.
- [ ] Add freshness, eviction, and disposal policies.
	Support stale time, garbage collection, max-age style expiry, and explicit disposal so long-lived apps do not leak memory.
- [ ] Add request deduplication across concurrent subscribers.
	Multiple components asking for the same resource should share one in-flight request rather than stampeding the network.
- [ ] Define mutation and optimistic update APIs.
	Support local optimistic writes, rollback on failure, and targeted invalidation for list and detail refresh flows.
- [ ] Add route-loader and shared-cache interoperability.
	Allow route loaders and component-level resources to share cached payloads where keys and invalidation rules match.
- [ ] Add devtools visibility for cached resources.
	Expose cache keys, freshness, subscriber counts, and last error state so async data bugs are debuggable without ad hoc logging.
- [ ] Add SSR-aware cache bootstrap and resume.
	Allow loader and resource caches to seed from server-rendered payloads and transition cleanly into client-owned cache state after hydration.
- [ ] Define cache serialization safety.
	Clarify which cached values may be embedded in bootstrap payloads, how large payloads are handled, and when sensitive server-only data must be excluded.
- [ ] Add cache revalidation-on-resume policies.
	Support rules such as trust-once, stale-while-revalidate, and always-refetch after hydration so apps can choose consistency versus startup speed explicitly.
- [ ] Add realistic shared-cache examples.
	Cover list/detail reuse, mutation refreshes, and cache-seeded SSR flows.

### Error boundaries

- [ ] Define an error boundary component contract.
	Decide whether boundaries are function-based, wrapper-based, or another explicit component form with fallback rendering.
- [ ] Capture render-time failures at subtree boundaries.
	Prevent a child component failure from crashing the entire app tree when a boundary is present.
- [ ] Support fallback UI rendering with error details.
	Allow users to render fallback content and optionally inspect the recovered error value.
- [ ] Define reset and retry behavior.
	Specify how boundaries retry after route changes, prop changes, or explicit resets.
- [ ] Define which failure modes are caught.
	Be explicit about render, effect, event handler, and hydration failures so the boundary model is predictable.
- [ ] Decide how boundaries compose with nested routes and layouts.
	Specify whether route-level boundaries wrap only leaf routes, layout shells plus leaves, or both.
- [ ] Define boundary behavior during SSR and hydration.
	Document whether server-render failures bubble globally, render fallback HTML, or mark the subtree as client-only, and how hydration failures map onto the same model.
- [ ] Add diagnostics integration for recovered errors.
	Recovered boundary errors should appear in runtime diagnostics and devtools with component-stack context instead of failing silently.

## 6. Scheduling and Runtime Coordination

- [ ] Decide whether the runtime should expose transitions.
	Determine whether a `startTransition` or `UseTransition` equivalent fits the scheduler model and solves real UI priority problems.
- [ ] Add lower-priority update scheduling if justified.
	Distinguish urgent input updates from non-urgent tree refreshes where measurable UI jitter exists.
- [ ] Decide whether deferred values are worth exposing.
	Validate that a `UseDeferredValue`-style API solves real typeahead, filtering, or route-search problems before adding parity APIs by name alone.
- [ ] Clarify whether a layout-effect equivalent is needed.
	Define whether DOM-read-before-paint scenarios require a dedicated hook beyond `UseEffect` and how it interacts with hydration.
- [ ] Define scheduler priority classes.
	Document whether the runtime should support only urgent vs non-urgent work or a richer priority ladder.
- [ ] Prototype a pending-state API for non-urgent updates.
	Validate whether callers need both a scheduling primitive and a typed pending flag for transition-style refreshes.
- [ ] Measure interruptibility requirements under heavy updates.
	Use benchmarks and browser scenarios to determine whether long list updates, route changes, and async completions need interruptible work splitting.
- [ ] Decide how scheduling primitives interact with route loaders and async boundaries.
	Clarify whether transition-like updates suppress loading fallbacks, delay route pending indicators, or simply lower update priority.
- [ ] Add browser examples for transition-style UX.
	Cover typeahead filtering, tab switches, and route transitions so scheduler semantics are understandable in real app flows.

## 7. SSR, Hydration, State Transfer, and Streaming

### Hydration correctness

- [ ] Teach the runtime to bind fibers to existing DOM nodes.
	Hydration must reuse server-rendered DOM instead of always clearing and recreating it.
- [ ] Define the initial hydration matching rules.
	Specify how host elements, text nodes, and fragments are matched and when hydration abandons reuse.
- [ ] Defer effects and subscriptions until hydration completes.
	Prevent eager client work from racing with DOM matching.
- [ ] Add subtree fallback behavior when hydration cannot safely continue.
	Recover from mismatches at the smallest practical subtree instead of always restarting the whole render.
- [ ] Add tests for successful hydration of simple pages and post-hydration updates.
	Prove that reused trees continue to respond correctly after hydration completes.
- [ ] Define event listener attachment order during hydration.
	Specify when handlers are rebound relative to DOM matching so early user input is not lost or double-handled.
- [ ] Preserve uncontrolled form state where safe.
	Avoid clobbering server-rendered input values, selection, and focus when client hydration binds to existing DOM.
- [ ] Define hydration behavior for portals, lazy nodes, async boundaries, and event-heavy components.
	Document which subtree types hydrate in place, which fall back, and which remain explicitly out of scope for now.
- [ ] Add route-aware hydration reuse tests.
	Verify that SSR-rendered router state, layout stacks, and loader data resume without remounting the wrong subtree or duplicating route work.
- [ ] Add progressive hydration benchmarks.
	Measure cold-start latency, first interaction timing, and hydration cost on medium-size trees so future work has concrete baselines.

### Server-to-client state transfer

- [ ] Define serialization boundaries.
	Specify how IDs, route state, atoms, cache seeds, form defaults, and other initial data are transferred from server to client.
- [ ] Define how atom snapshots are exported, serialized, and restored for hydration.
	The bootstrap story should cover both route-local payloads and shared state resumption.
- [ ] Define how `UseId` stays deterministic across server and client.
	Prevent SSR and client ID generation from diverging after hydration.
- [ ] Define serialization support for non-JSON-friendly values.
	Clarify how dates, byte slices, custom structs, and opaque IDs are encoded across JSON and CBOR bootstrap paths.
- [ ] Add versioning to bootstrap payloads.
	Prevent older clients or cached sidecars from silently misreading newer payload schemas.
- [ ] Define partial bootstrap reuse rules.
	Clarify which data may be trusted on first resume and which data must be revalidated immediately on the client.
- [ ] Add typed helpers for server-to-client payload registration.
	Provide an app-facing way to register route data, form defaults, cache seeds, and session hints without manual map packing in every app.
- [ ] Define per-route and per-subtree bootstrap scoping.
	Avoid sending the entire app state when only the active route, layout chain, or a specific async resource needs to cross the boundary.
- [ ] Add payload size budgeting and diagnostics.
	Expose when inline JSON, sidecar JSON, or binary payloads become too large and recommend a transport strategy before SSR payloads silently bloat responses.
- [ ] Add server-to-client state classification guidance.
	Separate safe public bootstrap state, resumable UI state, cache seeds, and server-only secrets so apps do not over-transfer sensitive or unnecessary data.
- [ ] Define merge semantics for transferred state.
	Specify how incoming bootstrap atoms, route data, and cache entries merge with client defaults or preexisting local state when a page is resumed or revisited.
- [ ] Define state transfer ownership during hydration.
	Clarify which bootstrap values become runtime-owned state, which remain immutable hints, and when client recomputation should overwrite transferred values.

### Streaming SSR

- [ ] Treat streaming SSR as an explicit post-hydration milestone.
	Do not layer chunked transport complexity onto an unfinished hydration model.
- [ ] Design chunked HTML streaming for route loaders.
	Allow the server to flush shell HTML early, then stream slower data-backed sections once loader work completes.
- [ ] Define async-boundary behavior under streaming SSR.
	Specify whether pending boundaries flush placeholder HTML first, stream completed subtree content later, and how the client reconciles those streamed segments.
- [ ] Add transport and buffering rules for streamed responses.
	Document how reverse proxies, gzip, and chunk buffering affect incremental flush behavior outside local development.
- [ ] Add examples and benchmarks for streaming SSR.
	Use a loader-heavy page and a nested layout route to verify faster first byte, earlier shell paint, and correct hydration after incremental HTML delivery.

### Hydration mismatch diagnostics

- [ ] Add mismatch detection and reporting.
	Detect text, structure, and critical attribute mismatches and surface them through runtime diagnostics.
- [ ] Add tests for mismatch reporting and recovery behavior.
	Prove that warnings, subtree replacement, and hydration abort cases behave deterministically.
- [ ] Add component-stack context to mismatch diagnostics.
	Warnings should name the component path and DOM selector context so developers can localize failures quickly.
- [ ] Add an opt-in strict hydration mode.
	Allow tests and development runs to fail fast on mismatches instead of silently replacing the subtree.
- [ ] Define production mismatch behavior.
	Document which mismatches degrade to warnings, which trigger subtree replacement, and which should abort hydration entirely.

## 8. Server Integration, Deployment, and Production Patterns

- [ ] Define a canonical Go HTTP integration story.
	Document how request handlers, middleware, SSR rendering, asset serving, bootstrap payload emission, and API endpoints fit together in a production app.
- [ ] Add middleware guidance for SSR apps.
	Cover logging, recovery, compression, caching, CSRF or session middleware ordering, and request context propagation for server-rendered apps.
- [ ] Add a first-party SSR app reference server.
	Provide a production-shaped example that combines routes, SSR, hydration, API handlers, static assets, and secure form posts under one Go server.
- [ ] Define backend API integration patterns.
	Show how route loaders, `fetch.UseResource`, and form submissions should talk to internal Go handlers versus external APIs, including timeout and auth propagation guidance.
- [ ] Add deployment guidance for common hosting modes.
	Document static hosting, Go server hosting, reverse-proxy setups, and mixed SSR/API deployments so adopters know which patterns are officially supported.
- [ ] Add observability hooks for server-rendered apps.
	Expose request-level render timing, hydration fallback counters, and bootstrap size metrics so SSR operations are measurable in production.

## 9. Strategic Direction and Experimental Work

### Resumability and partial activation

- [ ] Decide whether resumability is a real project goal.
	Clarify whether the framework should remain hydrate-first or pursue a serialized-resume model with deferred code execution.
- [ ] Evaluate partial activation and islands-style rendering as an intermediate step.
	Determine whether route- or component-level activation can reduce startup cost without changing the whole runtime model.
- [ ] Audit which runtime assumptions block resumability.
	Identify reliance on eager hook execution, immediate event binding, global scheduler state, and non-serializable closures.
- [ ] Define success criteria for resumability experiments.
	Use measurable goals such as lower startup execution cost, preserved server HTML, and delayed activation of non-interactive subtrees.
- [ ] Record explicit non-goals if resumability is rejected.
	Avoid leaving SSR and compiler work open to incorrect long-term assumptions.

### Fine-grained reactivity direction

- [ ] Decide whether fine-grained reactivity should remain out of scope.
	Clarify whether the framework stays fiber-and-hooks first or whether signal-like primitives are worth introducing for high-frequency UI paths.
- [ ] Evaluate signal-style primitives in a companion package before core adoption.
	Prototype fine-grained subscriptions without destabilizing the existing component and hook model.
- [ ] Define the minimal primitive set for a signal experiment.
	Decide whether the experiment needs only signal, computed, and effect-style building blocks or a larger API surface.
- [ ] Benchmark fine-grained updates against current keyed reconciliation paths.
	Use realistic list filtering, spreadsheet-style updates, and dashboard panels to determine whether finer granularity is actually needed.
- [ ] Add a migration boundary between component rerenders and fine-grained subscriptions.
	Clarify when a signal update rerenders an entire component, when it updates a smaller subscribed region, and how developers reason about mixed models.
- [ ] Define interoperability rules for fine-grained primitives.
	Specify how signal-like values interact with hooks, memoization, derived atoms, and scheduling.
- [ ] Evaluate devtools implications for fine-grained updates.
	If signal-style primitives ship, inspection and profiling must expose dependency graphs and update origins rather than only component rerenders.
- [ ] Add failure-mode tests for stale reads and update loops.
	Fine-grained systems are prone to accidental cycles and subscription leaks; prove the model can fail safely before widening the experiment.

### Compiler-assisted features

- [ ] Decide whether compiler-driven ergonomics are a real product direction.
	Separate syntax sugar, dead-code elimination, reactive dependency extraction, template lowering, and SSR build optimization instead of treating “compiler” as one bucket.
- [ ] Clarify the role of the browser compiler example.
	Document whether it is educational tooling, an experiment toward production tooling, or something intentionally outside the core roadmap.
- [ ] Evaluate whether compile-time reactivity is compatible with the current hook model.
	Determine whether any Svelte- or Solid-like compile step can coexist with `UseState` and `UseEffect` semantics without splitting the framework into two mental models.
- [ ] Define source-language boundaries for compiler work.
	Clarify whether compiler experiments target Go source only, HTML-like templates, generated Go helpers, or browser-hosted tooling.
- [ ] Add a migration and fallback plan for compiler-generated output.
	Users should be able to inspect, debug, and opt out of generated code paths if compile-time ergonomics ship.

### Full-stack framework maturity

- [ ] Decide whether GoWebComponents should remain a UI framework or grow a first-party app framework layer.
	Clarify whether file-based routing, build conventions, SSR bootstrapping, and deployment adapters belong in core, a sibling package, or external starters.
- [ ] Define a recommended project structure for production apps.
	Document a canonical layout for routes, loaders, assets, WASM builds, server entrypoints, and shared UI code so larger apps stop inventing their own structure.
- [ ] Evaluate first-party code-splitting and bundle-loading conventions.
	SSR, lazy loading, and route-level boundaries need a coherent loading story if the framework is meant to scale beyond demos.
- [ ] Define deployment targets and adapter expectations.
	Clarify how static hosting, Go HTTP servers, edge-style SSR, and mixed server/client deployments should be supported.
- [ ] Add an opinionated starter or reference app once conventions stabilize.
	A real app template should exercise routing, state, SSR, hydration, forms, metadata, and async data rather than only toy examples.

### Server-interactive runtime experiments

- [ ] Decide whether server-owned interactive rendering is a real product direction.
	Clarify whether GoWebComponents should remain client-owned WASM plus SSR/hydration, or whether an additional websocket-backed interactive runtime is worth pursuing experimentally.
- [ ] Define the minimum experiment scope for server-interactive mode.
	Limit the first exploration to event transport, server-side state ownership, DOM diff or patch streaming, reconnect handling, and a small reference app instead of a full alternative platform.
- [ ] Audit which current runtime assumptions block a server-interactive mode.
	Identify where the scheduler, event system, state hooks, router, and DOM commit model assume a local browser-owned runtime and what must be abstracted.
- [ ] Evaluate transport shape for server-interactive updates.
	Compare full HTML streaming, tree-patch messages, and DOM-op style diffs over websockets so the experiment does not lock into an inefficient protocol by accident.
- [ ] Define latency and offline expectations up front.
	Specify which interaction classes must stay responsive under moderate latency, what happens on reconnect, and which UI categories are unsuitable for server-owned interactivity.
- [ ] Add a security and scalability risk review for server-interactive mode.
	Track per-session memory cost, multi-tenant isolation, auth/session propagation, backpressure, and denial-of-service concerns before treating the experiment as roadmap-grade.
- [ ] Add a narrow proof-of-concept example.
	Use a dashboard or admin-style app with modest interaction density to validate the model before attempting general-purpose parity with the client-owned runtime.

### Ecosystem and plugin story

- [ ] Decide whether the framework needs a plugin or directive model.
	Determine whether extensibility belongs in core or whether companion packages alone are sufficient.
- [ ] Define extension boundaries before adding ad hoc framework utilities.
	Avoid scattering experimental features across unrelated packages without a stable ownership model.
- [ ] Identify which ecosystem problems belong in core versus companion packages.
	Keep the base framework small while still enabling higher-level libraries.
- [ ] Publish stability tiers for extension authors.
	Mark APIs as stable, experimental, or internal so third-party packages know which surfaces are safe to depend on.
- [ ] Define extension hooks for router, async data, devtools, SSR, and forms.
	Different extension types should know whether they can influence rendering, routing, bootstrap state, validation, or diagnostics instead of all plugins sharing one vague hook surface.
- [ ] Define a minimal plugin lifecycle.
	Specify how an extension registers itself, receives framework hooks, contributes cleanup logic, and declares compatibility without needing privileged internal access.
- [ ] Define compatibility and versioning policy for companion packages.
	Third-party and first-party extensions need a documented promise around semver, experimental hooks, and deprecation timing so the ecosystem can safely grow.
- [ ] Add companion-package candidates to the roadmap.
	Track likely packages such as auth helpers, animation primitives, cached query state, head management, and testing utilities outside the core runtime.
- [ ] Add a reference plugin or companion package.
	Validate the extension model with one real integration such as head management, auth-aware routing helpers, or query-cache devtools.

### Ecosystem and adoption maturity

- [ ] Define the minimum ecosystem story for 1.0-style adoption.
	List which pieces must exist first-party or be officially recommended: starter app, testing recipe, SSR recipe, state story, routing story, and deployment guidance.
- [ ] Add comparison docs against major frameworks.
	Explain where GoWebComponents is intentionally different, where it is not yet feature-complete, and which gaps are actively being closed.
- [ ] Publish production-readiness criteria by feature area.
	Separate experimental SSR, hydration, compiler, and runtime experiments from stable component, router, and state features so adopters can judge risk quickly.
- [ ] Add a real-world case study or reference application.
	Framework maturity is hard to evaluate from isolated examples alone; a sustained medium-size app should validate routing, async data, SSR, hydration, and operational workflow together.

## Completed Milestones Summary

The list below summarizes major work that is already shipped. It is intentionally compact and is not meant to duplicate the changelog.

### Public UI and state surface

- [x] Public `ui` and `html` APIs are in place for component composition and typed DOM construction.
- [x] Context API is supported through `CreateContext`, providers, and `UseContext`.
- [x] Local state, reducer state, refs, memoization, previous-value tracking, debounced and throttled values, channel consumption, and cancellable tasks are available.
- [x] Shared state supports atoms, computed values, derived values, and snapshot persistence.
- [x] Portals are a public API with selector- and node-based targets.

### Router and route data

- [x] Hash and history routers are supported.
- [x] Route patterns, typed params, query helpers, and search-param updates are supported.
- [x] Nested layout routes and explicit `Outlet()` rendering are supported.
- [x] Route loaders, loading states, error states, and manual revalidation are supported.
- [x] Route redirects, titles, descriptions, canonical URLs, and synchronous guards are supported.

### Async UI and developer tooling

- [x] Async boundaries and lazy async subtree loading are supported.
- [x] In-browser devtools support component tree inspection, hook summaries, route inspection, diagnostics, and profiling counters.

### SSR foundations

- [x] Request-time server rendering is supported through `ui.RenderToString(...)`.
- [x] A hydration entrypoint exists through `ui.Hydrate(...)`.
- [x] Bootstrap helpers exist for inline JSON, binary payloads, and sidecar references.
- [x] Request-time SSR examples exist for both route rendering and server-integrated routing.

### Forms and examples

- [x] `ui.UseForm` provides a first-class form helper for local field state, validation, and submit lifecycle handling.
- [x] Complex form, nested routing, portals, fetch, state, goroutine, and SSR examples exist and exercise the public API.
