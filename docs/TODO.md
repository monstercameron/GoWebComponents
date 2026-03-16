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

- [x] Reconcile docs with the actual supported API surface.
	The root README, package docs, and example docs now align on the current public APIs, shipped examples, SSR/hydration behavior, and the distinction between shipped versus future-facing backlog items.
- [x] Add a framework feature matrix to the docs.
	The root README now includes a project-wide feature inventory covering the runtime, hooks, state, router, SSR/hydration, devtools, testing, benchmarks, and example coverage.
- [x] Clarify which packages are stable public APIs versus internal/runtime-only details.
	The README and package docs now call out `ui`, `html`, `state`, `fetch`, `router`, and `devtools` as the public surface, while `internal/runtime` remains implementation detail.
- [ ] Add migration notes as new primitives land.
	Explain how newer APIs replace older patterns so early adopters do not accumulate legacy usage accidentally.

### Documentation discoverability and task-oriented guidance

- [ ] Reorganize docs around common developer tasks.
	Provide clear entry points for workflows such as building a client-only app, adding routing, adding SSR, testing a component, shipping a production wasm build, and debugging hydration issues instead of relying mostly on package-by-package reading order.
- [ ] Add end-to-end walkthroughs for common app shapes.
	Create guided docs for a small SPA, a server-rendered app, a static-export app, and a data-heavy dashboard so developers can follow a realistic path rather than stitching together isolated examples.
- [ ] Add cross-linked concept, API, and example references.
	Ensure each major feature page links directly to the public API, a runnable example, production caveats, and related debugging or testing guidance so discoverability improves for new users.
- [ ] Add troubleshooting guides for common setup and runtime failures.
	Document likely causes and fixes for wasm build failures, missing `wasm_exec.js`, broken example serving, hydration mismatch warnings, route misconfiguration, and interop mistakes.
- [ ] Add a clearly documented recommended path for new adopters.
	State which packages, examples, commands, and architecture patterns are the preferred modern route so developers are not left choosing between stale and current approaches.

### API stability and support policy

- [ ] Define stability tiers for all major framework surfaces.
	Explicitly classify public packages, experimental features, internal runtime details, and companion-package APIs so adopters know which layers are safe for long-term production use.
- [ ] Publish a semver and compatibility policy.
	Document what kinds of changes are allowed in patch, minor, and major releases, and which behavior changes count as breaking for consumers.
- [ ] Define a deprecation lifecycle and migration window.
	Specify how long deprecated APIs remain supported, how removals are announced, and what migration guidance must exist before a breaking cleanup is allowed.
- [ ] Add upgrade and migration guides for major framework changes.
	Provide release-to-release guidance for runtime, router, SSR, forms, and state changes so enterprise teams can upgrade without reverse-engineering diffs.
- [ ] Define long-term support expectations for enterprise adopters.
	Clarify whether the project intends to support LTS-style release lines, security backports, or only latest-version support before large organizations bet on it operationally.

### Metadata and composition model

- [ ] Finalize the SSR metadata model.
	Decide whether route-level title, description, and canonical metadata are server-owned, client-owned after hydration, or jointly reconciled.
- [ ] Add a server render path for route-managed metadata.
	Ensure router metadata can be emitted during `ui.RenderToString(...)` instead of requiring post-render DOM mutation.
- [ ] Define metadata hydration reconciliation rules.
	Prevent duplicate, stale, or leaked head tags when client resume takes over a server-rendered document.
- [ ] Decide whether slots are part of the public composition model.
	Either formalize slot-style composition with clear usage guidance or explicitly mark it out of scope in favor of ordinary children and layout patterns.

### Head management and SEO surface

- [ ] Define the first-class head management model.
	Decide whether document title, meta tags, canonical URLs, link tags, social metadata, and structured data are owned by route metadata, explicit components, or a dedicated head manager so applications do not mix several incompatible patterns.
- [ ] Add server-rendered head emission for route-driven apps.
	Ensure titles, descriptions, canonicals, robots directives, social cards, and preload hints can be produced during `ui.RenderToString(...)` without requiring a client-only patch-up step after the first paint.
- [ ] Define hydration reconciliation rules for head state.
	Prevent duplicated tags, stale canonicals, leaked route metadata, and incorrect tag ordering when client navigation takes ownership of a document that was initially rendered on the server.
- [ ] Add route-level title and metadata composition rules.
	Clarify how layouts, nested routes, and leaf pages merge or override title templates, meta descriptions, robots directives, social tags, and canonical URLs instead of leaving every router integration to invent its own precedence rules.
- [ ] Add canonical URL and duplicate-content guidance.
	Document how apps should generate canonical links for parameterized routes, locale variants, paginated content, and prerendered versus request-time rendered pages so SEO-sensitive apps avoid accidental duplicate indexing.
- [ ] Add structured-data support guidance.
	Define whether JSON-LD and other machine-readable metadata are first-class primitives, helper components, or documented escape hatches, and how they participate in SSR and hydration safely.
- [ ] Add preload, preconnect, and resource-hint management.
	Support or document how routes and layouts contribute preload, modulepreload, preconnect, DNS-prefetch, and other performance-relevant head hints without duplicated or stale hints accumulating over time.
- [ ] Add social-sharing metadata examples.
	Demonstrate Open Graph, Twitter/X card, article metadata, and route-specific preview images for both static export and server-rendered routes so the recommended head model is validated with realistic content pages.
- [ ] Add sitemap, robots, and crawl-control integration guidance.
	Connect route metadata, prerender output, and deployment guidance so sitemap generation, robots directives, noindex routes, and preview environments behave consistently.
- [ ] Add tests for head correctness across SSR, hydration, and navigation.
	Verify that title, canonical, meta, structured data, and resource hints remain correct through initial server render, client hydration, and subsequent route transitions.

### Accessibility primitives and guidance

- [ ] Define the accessibility support baseline for the public UI surface.
	Document which accessibility responsibilities are already handled by typed HTML props and IDs, and which behaviors still require first-class framework primitives.
- [ ] Add a first-class focus-management toolkit.
	Support common needs such as returning focus after dialog close, focusing first invalid form fields, trapping focus inside overlays, and restoring focus after route-driven UI changes.
- [ ] Add keyboard-navigation primitives for composite widgets.
	Provide reusable patterns for roving tabindex, arrow-key navigation, typeahead navigation, and active-descendant style controls so menus, tabs, listboxes, and command palettes do not require bespoke logic in every app.
- [ ] Add live-region and announcement helpers.
	Support polite and assertive announcements for async loading, validation results, toasts, and route transitions without forcing every app to hand-roll `aria-live` containers.
- [ ] Define accessible overlay primitives.
	Document and eventually support the semantics required for dialogs, popovers, dropdown menus, and sheet-style overlays, including focus trapping, escape handling, inert-background behavior, and aria wiring.
- [ ] Add accessibility guidance for forms, routed apps, and async UI.
	Document label/input pairing, field error announcements, pending-state semantics, route-change announcements, and loading-state patterns that work with screen readers.
- [ ] Add accessibility-focused examples and tests.
	Create examples and browser tests for accessible modal, tabs, listbox or combobox, form validation feedback, and routed page-announcement behavior so the guidance is enforced by real usage.

### Portal layering and overlay management

- [ ] Define a first-class overlay and portal layering model.
	Document how modals, popovers, tooltips, dropdowns, sheets, and nested portals participate in shared stacking order instead of leaving z-index policy to ad hoc application code.
- [ ] Add a centralized overlay manager primitive.
	Provide a framework-level way to register active overlays, assign stack order, and coordinate mount or unmount behavior for nested and sibling portal trees.
- [ ] Define escape-key and dismissal routing for nested overlays.
	Specify which overlay handles escape first, how outside-click dismissal behaves across stacked layers, and how parent overlays remain stable when child overlays close.
- [ ] Add scroll-lock and background-inert behavior.
	Support consistent body scroll locking, nested overlay lock counting, and background interaction suppression so portal-heavy apps do not reimplement these details for every dialog flow.
- [ ] Define focus and accessibility coordination for layered overlays.
	Ensure the overlay manager composes correctly with focus trapping, restoration, announcement semantics, and aria relationships when several portal-driven surfaces are open at once.
- [ ] Add positioning and anchor coordination guidance for floating overlays.
	Document how tooltips, anchored popovers, and context menus should manage viewport collision, resize or scroll repositioning, and nested stacking without conflicting portal ownership.
- [ ] Add examples and tests for complex overlay stacks.
	Demonstrate nested dialogs, dialog-plus-popover, tooltip-over-menu, and portal retargeting scenarios so layering behavior is enforced by real browser coverage.

### Internationalization and localization

- [ ] Define the first-class i18n scope for the framework.
	Decide whether the framework should own only message lookup and locale context, or also pluralization, formatting helpers, locale-aware routing, and SSR locale transfer.
- [ ] Add a locale context and switching model.
	Provide a stable way to expose the active locale to component trees, update it at runtime, and coordinate locale changes with rerendering, route changes, and persisted user preference.
- [ ] Add message catalog loading and lookup primitives.
	Support organizing translated messages by locale and namespace, loading them deterministically, and resolving missing-message fallback behavior without every app inventing its own structure.
- [ ] Add message formatting and pluralization helpers.
	Support interpolated messages, plural rules, select-style branching, and locale-aware number or date formatting so application text does not rely on ad hoc string concatenation.
- [ ] Define SSR and hydration behavior for locale data.
	Clarify how active locale, selected messages, and formatting configuration are transferred from server to client so SSR output and hydrated UI stay consistent.
- [ ] Add locale-aware routing and content-loading guidance.
	Document whether locale prefixes, locale domains, or route metadata should be handled by the router, application code, or a companion package, and how loaders select locale-specific content.
- [ ] Add RTL and directionality support guidance.
	Specify how locale changes affect document direction, component-level `dir` overrides, layout assumptions, and mixed-direction content in real applications.
- [ ] Add i18n-focused examples and tests.
	Create examples for locale switching, pluralized UI, date or number formatting, SSR locale bootstrapping, and locale-prefixed routing so the public i18n story is validated end to end.

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

### Custom element and web component interop

- [ ] Define the supported custom-element interop model.
	Decide whether the public goal is only consuming browser custom elements from GoWebComponents trees, exporting GoWebComponents trees as custom elements, or supporting both with separate APIs.
- [ ] Add first-class custom-element consumption helpers.
	Support property assignment, attribute reflection, custom event subscription, slot content passing, and element-handle access for browser-defined custom elements without ad hoc `syscall/js` code.
- [ ] Define prop-versus-attribute mapping for custom elements.
	Specify how strings, booleans, numbers, objects, callbacks, and opaque JS values are passed into custom elements so wrappers behave predictably.
- [ ] Add custom-event bridging for web components.
	Allow typed subscriptions to `CustomEvent` payloads and document how event detail values cross the JS boundary into Go handlers.
- [ ] Add SSR and hydration rules for custom-element hosts.
	Clarify whether custom elements are rendered as inert tags on the server, when properties are applied on the client, and how hydration avoids clobbering upgraded element state.
- [ ] Explore exporting GoWebComponents components as standards-based custom elements.
	Prototype a wrapper that mounts a component into a shadow root or light DOM host, maps observed attributes to props, and cleans up runtime resources on disconnect.
- [ ] Define style and shadow-DOM expectations for exported custom elements.
	Clarify whether exported elements assume shadow DOM isolation, light DOM composition, external CSS ownership, or multiple supported modes.
- [ ] Add interoperability examples against third-party web components.
	Demonstrate consuming a real custom-element library and, separately, exposing a GoWebComponents widget for use from plain HTML or another framework.

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

### Worker and background-thread integration

- [ ] Define the public worker integration model.
	Decide whether worker support is limited to browser Web Workers, also covers Shared Workers, or extends to other background execution helpers, and how it fits the existing Go concurrency story.
- [ ] Add first-class worker lifecycle helpers.
	Support worker creation, startup handshake, termination, restart, and cleanup through a public API that composes cleanly with component mount and unmount behavior.
- [ ] Add typed message-channel helpers for workers.
	Provide a structured way to post requests, receive responses, stream progress updates, and surface typed errors without every app hand-rolling raw message event parsing.
- [ ] Define cancellation, timeout, and backpressure behavior for worker jobs.
	Specify how long-running worker tasks are cancelled, how late responses are ignored, and how large message bursts or busy workers avoid overwhelming the main UI thread.
- [ ] Add helpers for connecting workers to components and async state.
	Make it straightforward to bind worker-backed jobs into hooks, task handles, or resource state so heavy CPU work can report pending, ready, progress, and failure states through familiar primitives.
- [ ] Define serialization boundaries for worker payloads.
	Document how structs, byte slices, transferable objects, large arrays, and non-cloneable values cross the worker boundary so background execution remains predictable.
- [ ] Add examples for CPU-heavy background work.
	Demonstrate a worker-backed parser, formatter, search index, or computation-heavy UI flow that would otherwise block rendering on the main thread.

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

- [x] Evaluate a resource/cache abstraction above `UseFetch`.
	Support request deduplication, stale-while-revalidate behavior, and reuse across components.
- [x] Decide where cached async state should live.
	Keep the API coherent with atoms instead of creating a separate parallel mental model.
- [x] Add invalidation primitives.
	Support manual refresh, key-based invalidation, and optimistic mutation flows.
- [x] Integrate resource state with async boundaries.
	Loading and error handling should compose cleanly with suspense-style rendering.
- [x] Add realistic examples.
	Cover list/detail fetches, mutation refreshes, and shared cached queries.
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
- [ ] Add realistic shared-cache examples for SSR-seeded and route-loader reuse.
	Extend the shipped shared-cache examples to cover cache-seeded SSR flows and route-loader interoperability.

### Offline mutation queueing and background sync

- [ ] Define the scope of first-class offline mutation support.
	Decide whether the framework should support only queued form or mutation retries, or a broader model that includes background sync registration, optimistic UI, and conflict resolution after reconnect.
- [ ] Add a persistent mutation queue abstraction.
	Provide a way to enqueue writes while offline, persist them across reloads, and replay them in order once connectivity returns without forcing every app to hand-roll storage and retry logic.
- [ ] Define optimistic-update and rollback semantics for queued writes.
	Specify how local UI updates behave before queued mutations reach the server, how failures roll back or surface conflicts, and how related caches or atoms are invalidated after replay.
- [ ] Add retry, deduplication, and backoff policies for queued mutations.
	Support transient failure retry, duplicate submission suppression, exponential backoff, and user-visible terminal failure states so reconnect behavior is not arbitrary.
- [ ] Define queue serialization and security boundaries.
	Clarify which request payloads may be stored locally, how sensitive data should be excluded or encrypted, and how queued operations are versioned across app updates.
- [ ] Integrate offline mutation replay with service workers and background sync where available.
	Document or provide a first-party pattern for using Background Sync or equivalent service-worker coordination when the platform supports it, with graceful fallback when it does not.
- [ ] Add examples for offline write replay.
	Demonstrate a draft save, queued form submission, or cache-backed mutation flow that survives offline periods and reconciles cleanly after connectivity returns.

### Cross-tab state and cache synchronization

- [ ] Define the scope of first-party cross-tab synchronization.
	Decide whether the framework should sync only atom state, also support auth or session hints and shared caches, and whether synchronization is opt-in per key or global by default.
- [ ] Add BroadcastChannel-based synchronization helpers.
	Provide a first-class way to publish and receive atom, cache, or session updates across tabs without forcing every app to wire browser channels manually.
- [ ] Add storage-event fallback support where appropriate.
	Document and optionally support `localStorage`-driven synchronization for environments where BroadcastChannel is unavailable or intentionally avoided.
- [ ] Define conflict resolution and merge semantics for cross-tab updates.
	Specify how simultaneous updates, stale tabs, logout events, optimistic cache mutations, and conflicting writes resolve so shared state does not oscillate unpredictably.
- [ ] Add opt-in scoping and filtering for synchronized values.
	Allow apps to choose which atoms, cache keys, auth hints, or persisted form drafts synchronize across tabs instead of forcing all client state into one global channel.
- [ ] Define bootstrap and resume interaction for synchronized state.
	Clarify how cross-tab synchronization interacts with SSR bootstrap, cache rehydration, and persisted snapshots so initial page load does not immediately overwrite fresher tab state.
- [ ] Add diagnostics and examples for cross-tab behavior.
	Demonstrate theme sync, logout propagation, cache invalidation broadcasting, and draft-state sharing across tabs, together with tooling to inspect synchronization events during development.

### Multi-window and multi-surface coordination

- [ ] Define the multi-surface coordination model.
	Decide whether first-class support covers browser tabs only, popup windows, embedded iframes, side panels, and external control surfaces, and how that model differs from simple cross-tab state sync.
- [ ] Add coordination channels for popup and secondary-window workflows.
	Support opening a secondary window or control panel and exchanging typed events, shared state hints, and lifecycle signals without every app hand-rolling `window.postMessage` or related browser plumbing.
- [ ] Define ownership and synchronization rules across multiple active surfaces.
	Specify which surface owns authoritative state, how mirrored views subscribe to updates, and how conflicting edits or stale secondary windows are resolved.
- [ ] Add shared session and route coordination helpers.
	Support propagating logout, session expiry, active-document selection, route focus, and window-intent actions across multiple app surfaces.
- [ ] Define teardown and orphan-surface behavior.
	Clarify how the runtime responds when a popup closes unexpectedly, a parent window disappears, or an embedded surface loses connectivity to its coordinator.
- [ ] Add examples for multi-window applications.
	Demonstrate patterns such as a popup inspector, detached dashboard panel, or operator console coordinating with the main app through first-class framework helpers.

### Error boundaries

- [x] Define an error boundary component contract.
	Decide whether boundaries are function-based, struct-based, or a special component wrapper with fallback rendering.
- [x] Capture render-time panics at subtree boundaries.
	Prevent a child component failure from crashing the entire app tree when a boundary is present.
- [x] Support fallback UI rendering with error details.
	Allow users to render fallback content and optionally inspect the recovered error value.
- [x] Define reset behavior after recovery.
	Specify how boundaries retry after route changes, prop changes, or explicit resets.
- [x] Add coverage for render, effect, and event handler failure cases.
	Be explicit about which failure modes boundaries catch and which remain global errors.
- [ ] Decide how boundaries compose with nested routes and layouts.
	Specify whether route-level boundaries wrap only leaf routes, layout shells plus leaves, or both.
- [ ] Define boundary behavior during SSR and hydration.
	Document whether server-render failures bubble globally, render fallback HTML, or mark the subtree as client-only, and how hydration failures map onto the same model.
- [ ] Add diagnostics integration for recovered errors.
	Recovered boundary errors should appear in runtime diagnostics and devtools with component-stack context instead of failing silently.

### Production correctness hardening

- [ ] Define the minimum production-correctness bar for core runtime flows.
	List which rendering, routing, async, SSR, hydration, and state behaviors must be deterministic and fully specified before the framework can claim serious production readiness.
- [ ] Add scenario-level correctness suites for complex app flows.
	Test nested routes, shared atoms, async loaders, portals, forms, hydration, offline replay, and interop-heavy components together so correctness is proven under composition instead of only isolated unit behavior.
- [ ] Add long-running stability and leak detection coverage.
	Exercise repeated navigation, mount/unmount churn, worker activity, overlay churn, and cache turnover to catch memory leaks, stale subscriptions, and degraded behavior in long-lived sessions.
- [ ] Add concurrency and race-condition stress tests.
	Stress overlapping route transitions, async cancellations, repeated submissions, cross-tab updates, worker messages, and hydration fallback to prove stale results and conflicting updates do not corrupt runtime state.
- [ ] Add production-mode behavior parity checks.
	Validate that stripped or optimized builds preserve the same correctness guarantees as debug-oriented builds so release flags do not silently change behavior.

## 6. Scheduling and Runtime Coordination

- [x] Evaluate whether the runtime should expose transitions.
	The runtime now exposes `ui.StartTransition` and `ui.UseTransition`, using a small non-urgent scheduling lane for deferred `UseState` and `UseAtom` updates.
- [x] Add lower-priority update scheduling support if feasible.
	Transition-scoped state work now commits through a delayed low-priority lane so urgent input updates can land first.
- [x] Decide whether deferred values are worth exposing.
	`ui.UseDeferredValue` now keeps rendering the last committed value until a transition updates the deferred copy.
- [x] Evaluate a reducer-style state primitive.
	Completed earlier under Go-native hook additions: `UseReducer` shipped, and the remaining open work in this section is about scheduling primitives rather than reducer API design.
- [x] Clarify whether a layout-effect equivalent is needed.
	Current decision: keep `UseEffect` as the only effect hook until concrete DOM-read-before-paint scenarios justify a dedicated layout-effect API.
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

- [x] Decide whether server-side rendering is a project goal.
	SSR is now an explicit project goal. The current direction is request-time HTML generation on native Go targets through `ui.RenderToString(...)`, paired with browser hydration/resume through `ui.Hydrate(...)` using bootstrap restore, DOM reuse, mismatch diagnostics, and subtree fallback.
	Phase 1 scope decisions:
	- [x] Decide whether the first SSR pass targets static HTML generation only, with hydration deferred.
		The current SSR pass supports both request-time HTML generation and a real hydration path with DOM matching and mismatch recovery for the supported host/text subtree cases.
	- [x] Decide whether the first public API lives in `ui`, `html`, or a dedicated SSR package.
		The first public SSR API lives in `ui`.
	- [x] Define which component forms are supported in the first pass: host elements only, simple function components, or full `ui.CreateElement(...)` trees.
		The current pass supports host elements, fragments, and simple function components that produce SSR-safe `ui.Node` / `runtime.Element` trees; hook-heavy browser-only component behavior remains constrained.
	- [x] Decide whether router integration is explicit bootstrap data or automatic coupling to the current router globals.
		The current SSR direction uses explicit bootstrap payloads rather than implicit router-global coupling.
	- [x] Define which existing features are explicitly out of scope for v1: portals, async boundaries, error boundaries, browser-only effects, devtools overlay.
		Those remain out of scope for the current SSR v1 surface while the current render-to-string, bootstrap transfer, and hydration feature set matures.
- [x] Add a server render entrypoint.
	`ui.RenderToString(...)` now provides the public non-browser HTML render entrypoint, and `examples/18-ssr-server-routing` exercises request-time SSR over a real Go HTTP server.
	Phase 1 implementation tasks:
	- [x] Add an internal render-to-string prototype for `runtime.Element` trees.
	- [x] Add a public server render entrypoint.
		`ui.RenderToString(...)` now exposes the first public SSR render surface for non-js/wasm builds.
	- [x] Support deterministic text and attribute escaping.
		The current SSR renderer escapes text and attribute values before emitting HTML.
	- [x] Support fragments and void elements.
		The initial renderer now handles fragments and common void tags such as `input`, `meta`, and `link`.
	- [x] Decide how props such as `className`, `htmlFor`, `style`, and boolean attributes serialize.
		The first pass normalizes `className -> class`, `htmlFor -> for`, serializes style maps deterministically, and emits true boolean attributes without values.
	- [x] Decide how unsupported browser-only props and handlers are filtered.
		The current SSR path skips `children`, `key`, and `on*` handler props when emitting HTML.
	- [x] Add tests for simple host trees, text escaping, attribute escaping, and void-element output.
	- [x] Add tests for supported function-component rendering rules.
		Current coverage includes simple one-argument function components returning `ui.Node` / `*runtime.Element` trees.
	- [x] Add a request-time server-rendered reference example.
		`examples/18-ssr-server-routing` now serves HTML per request from a Go HTTP server, emits a route-specific bootstrap sidecar, and hydrates a browser-router client over real URLs.
- [x] Add a hydration path for browser startup.
	Reuse server-rendered markup instead of always doing a fresh client render.
	Phase 2 hydration tasks:
	- [x] Add a hydration entrypoint distinct from fresh client render.
		`ui.Hydrate(...)` now restores the bootstrap payload, attempts DOM reuse, and falls back per subtree when hydration cannot safely continue.
	- [x] Teach the runtime to bind fibers to existing DOM nodes instead of always creating new ones.
		Hydration now reuses matching host and text DOM nodes instead of eagerly recreating the entire subtree.
	- [x] Define the initial hydration matching rules for host elements, text nodes, and fragments.
		The current hydration matcher reuses same-tag host elements, text nodes, and fragment/function-provider descendants that can continue through a shared parent boundary.
	- [x] Defer effects and subscriptions until hydration completes.
		Hydration now queues atom subscriptions and hydration-time update notifications until the commit finishes, then runs effects against the committed tree.
	- [x] Add subtree fallback behavior when hydration cannot safely continue.
		Hydration now warns and replaces only the affected subtree when structure matching fails or expected nodes are missing.
	- [x] Add tests for successful hydration of simple pages and component updates after hydration.
		Coverage now includes simple DOM reuse, text mismatch recovery, trailing-node cleanup, subtree fallback, and state updates after hydration.
- [x] Define serialization boundaries.
	Specify how IDs, props, route state, and initial data are transferred from server to client.
	Bootstrap/transfer tasks:
	- [x] Define a bootstrap payload shape for route path, query, params, and initial data.
		`ui.SSRBootstrap` now provides a first public payload shape for route, atoms, arbitrary data, and ID seed transfer.
	- [x] Define how atom snapshots are exported, serialized, and restored for hydration.
		Hydration now applies bootstrap atom snapshots before resuming the client tree, building on the existing snapshot export/import helpers.
	- [x] Define how `UseId` stays deterministic across server and client.
		Hydration now applies the bootstrap ID seed before client render so `UseId` generation resumes from the transferred seed.
	- [x] Decide whether loader data is embedded inline, fetched again, or optionally reused.
		The current direction is explicit bootstrap transfer with optional reuse on the first client resume. The request-time server SSR example reuses route-specific bootstrap data on the initial browser route before subsequent route resolution falls back to normal client behavior.
	- [x] Add helpers for embedding and reading bootstrap JSON safely.
		`MarshalSSRBootstrap`, `UnmarshalSSRBootstrap`, and `RenderBootstrapScript` now cover safe JSON/script embedding for initial SSR payload transfer.
	- [x] Add an optional binary bootstrap encoding path.
		`MarshalSSRBootstrapBinary` and `UnmarshalSSRBootstrapBinary` now provide a CBOR-based binary transport option for sidecar/bootstrap payload delivery without replacing the default inline JSON path.
	- [x] Add a sidecar bootstrap reference path.
		`SSRBootstrapReference`, `RenderBootstrapReferenceScript`, and wasm-side `ReadBootstrapReference(...)` now support pointing hydration at external JSON or CBOR bootstrap payloads instead of only inline script JSON.
- [x] Add mismatch detection and error reporting.
	Surface hydration mismatches clearly during development.
	Developer-experience tasks:
	- [x] Detect text mismatches during hydration.
	- [x] Detect tag/structure mismatches during hydration.
	- [x] Detect critical attribute mismatches during hydration.
	- [x] Report hydration warnings through the runtime diagnostics surface.
	- [x] Decide when to warn versus when to replace the subtree.
		Text and attribute mismatches now warn and continue; structural mismatches fall back to client rendering for the affected subtree.
	- [x] Add tests for mismatch reporting and recovery behavior.
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

### Observability and runtime instrumentation

- [ ] Define a structured runtime event model.
	Standardize how render phases, route transitions, loader activity, hydration phases, async boundary resolution, worker events, and cache invalidations are emitted so tools can consume one coherent event stream.
- [ ] Add correlation IDs across SSR, bootstrap, and hydration.
	Allow one page load or interaction to be traced from server render through bootstrap serialization and client hydration instead of fragmenting diagnostics across unrelated logs.
- [ ] Add request-scoped tracing for server-rendered apps.
	Expose route match timing, loader timing, render timing, bootstrap serialization timing, and response write timing as a coherent per-request trace.
- [ ] Add client-side lifecycle instrumentation hooks.
	Expose navigation timing, async boundary wait time, hydration duration, cache hit or miss rates, and render or commit cost through a stable instrumentation surface.
- [ ] Define export formats for traces and runtime metrics.
	Support devtools viewing, local file export, and future integration with tracing or metrics backends without locking the project into one transport early.
- [ ] Add sampling and noise-control rules for instrumentation.
	Prevent high-volume runtime events from overwhelming local debugging or production telemetry while still preserving enough detail to diagnose correctness and performance issues.
- [ ] Add examples and docs for end-to-end observability.
	Demonstrate how an application traces one routed page load, one async resource flow, and one server-rendered request through the public instrumentation hooks.

### Logging and diagnostics surface

- [ ] Define a first-class logger interface and domain model.
	Standardize log levels, structured fields, and named domains such as runtime, router, fetch, state, hydration, SSR, interop, worker, and forms instead of relying on scattered `fmt.Println(...)` output.
- [ ] Add structured development and production log outputs.
	Support human-readable local logs and machine-readable structured logs for production servers without forcing applications to fork framework logging behavior.
- [ ] Add redaction and secret-boundary rules for logs.
	Ensure bootstrap payloads, auth/session hints, request headers, form submissions, and other sensitive fields are never logged accidentally through convenience diagnostics.
- [ ] Add log integration for routes, loaders, and mutations.
	Emit meaningful lifecycle logs for navigation, redirect decisions, loader failures, form submissions, cache invalidations, and offline replay events so developers can debug app flows without ad hoc instrumentation.
- [ ] Add in-memory devtools log buffering.
	Allow recent framework logs and diagnostics to appear inside the devtools surface instead of forcing developers to correlate browser console and server terminal output manually.
- [ ] Add warning classification and escalation rules.
	Differentiate correctness failures, performance warnings, unsupported-but-recovered cases, and informational notices so logs remain actionable instead of noisy.

### Static prerender and export workflows

- [ ] Define whether static prerender is a first-class output mode.
	Clarify how prerender differs from request-time SSR, which app types it targets, and whether it belongs in core tooling or a companion build package.
- [ ] Add a prerender-to-files pipeline.
	Support rendering one or more routes to HTML files plus associated bootstrap payloads so docs sites, marketing pages, and hybrid static apps do not need a live Go server for initial delivery.
- [ ] Define route enumeration for prerendered apps.
	Support explicit route lists, parameter expansion hooks, sitemap-style generation, and fallback handling for routes that cannot be known at build time.
- [ ] Define asset and bootstrap output conventions for prerender.
	Specify where HTML, sidecar payloads, manifest files, and copied static assets land so deployment to static hosts is predictable.
- [ ] Add partial-hydration or client-resume expectations for prerendered output.
	Clarify whether prerendered pages always hydrate, may remain static, or can opt into selective activation depending on page needs.
- [ ] Add invalidation and rebuild guidance for prerendered content.
	Document how content changes, route-data changes, and shared-layout changes trigger rebuilds in local development and CI pipelines.
- [ ] Add a first-party static export example.
	Ship a docs or marketing-style example that prerenders several routes to files and then serves them from a static host without a custom server runtime.

### Image, media, and asset delivery ergonomics

- [ ] Define the framework-level asset delivery story.
	Clarify which concerns belong in core, build tooling, or companion packages for static assets, media optimization, manifests, cache busting, and deployment-friendly asset references so apps are not forced into ad hoc conventions.
- [ ] Add asset-manifest guidance for application builds.
	Define how wasm binaries, JS helpers, CSS files, images, fonts, and copied static assets are named, versioned, and referenced from SSR, prerender, and static-hosted outputs.
- [ ] Add cache-busting and static asset versioning conventions.
	Specify how fingerprinted filenames, content hashes, manifest lookups, and cache headers should work so deployments can serve aggressively cached assets without stale-client breakage.
- [ ] Add responsive image and media guidance.
	Document the recommended way to emit `srcset`, `sizes`, lazy-loading attributes, decoding hints, poster images, and art-direction variants for image-heavy or content-heavy apps.
- [ ] Evaluate first-class helpers for image and media components.
	Decide whether common concerns such as responsive sources, intrinsic sizing, loading priority, placeholder behavior, and accessibility should live in a companion package rather than every app hand-rolling them.
- [ ] Add preload and prefetch guidance for static assets.
	Document how routes, layouts, and head management should declare preload or prefetch hints for images, fonts, CSS, wasm, and other critical assets without over-fetching or producing duplicate hints.
- [ ] Add lazy media loading patterns for routed apps.
	Provide guidance for deferring offscreen images, videos, embeds, and heavy media widgets while preserving layout stability and accessible fallback behavior.
- [ ] Add SSR and prerender rules for asset references.
	Ensure server-rendered HTML and prerendered files can resolve hashed assets, media variants, and deployment base paths correctly without requiring brittle hard-coded URLs.
- [ ] Add CDN and static-host deployment guidance for assets.
	Document cache headers, immutable asset serving, origin separation, compression expectations, and invalidation behavior for assets served behind CDNs or static hosts.
- [ ] Add example apps that stress real asset delivery concerns.
	Ship a content-rich example with responsive images, route-scoped preload hints, hashed asset output, and lazy media loading so the public story is validated end to end.

### Browser support and compatibility policy

- [ ] Publish an explicit browser support matrix.
	List the minimum supported desktop and mobile browsers, including Safari and mobile Safari expectations, so adopters know which environments the runtime and examples are expected to work in.
- [ ] Define the required browser feature baseline for the wasm runtime.
	Document which web platform features are assumed by core packages, router behavior, fetch helpers, workers, devtools, and SSR hydration so compatibility is based on concrete capabilities rather than vague “modern browser” language.
- [ ] Define the project stance on polyfills and shims.
	Clarify whether the framework ships any compatibility helpers, expects application-level polyfills, or treats unsupported environments as out of scope.
- [ ] Add progressive-enhancement boundaries for partial support cases.
	Specify what should still work when JavaScript, wasm features, service workers, advanced navigation APIs, or storage features are limited, especially for prerendered and SSR-delivered content.
- [ ] Add mobile Safari and constrained-device validation coverage.
	Test the framework against the browser family most likely to expose wasm, caching, input, and memory edge cases before calling any workflow production-ready.
- [ ] Define compatibility expectations for workers, offline features, and advanced APIs.
	Document which higher-level features degrade gracefully, require capability checks, or are unavailable on some browser families so applications can choose safe fallbacks intentionally.
- [ ] Add browser-compatibility CI coverage or release checks.
	Run a small compatibility matrix against representative examples so regressions in supported browsers are detected before release instead of after user reports.
- [ ] Add fallback and degradation guidance for unsupported browsers.
	Document how apps should communicate unsupported environments, offer reduced functionality, or fall back to static SSR content when full interactive behavior is not available.
- [ ] Add a browser-support policy to release documentation.
	State how support changes are announced, how long deprecated browser targets remain supported, and what evidence is required before narrowing the compatibility matrix.

### PWA and offline app support

- [ ] Define the framework’s PWA support boundary.
	Clarify whether the goal is first-party service-worker tooling, manifest generation, offline cache guidance, or only documented integration points for external tools.
- [ ] Add web app manifest generation or templating helpers.
	Support app name, icons, theme colors, display mode, start URL, and installability metadata without forcing every app to hand-roll the same manifest pipeline.
- [ ] Add a service-worker integration story.
	Document or provide a first-party pattern for precaching static assets, versioning build artifacts, and wiring service-worker registration into GoWebComponents apps.
- [ ] Define offline caching strategies for app shells and route data.
	Separate shell caching, static asset caching, API response caching, and mutation replay so offline behavior is explicit rather than accidental.
- [ ] Add update and invalidation semantics for offline assets.
	Specify how new WASM binaries, JS helpers, HTML shells, and cached route payloads invalidate old caches and how users are prompted to refresh.
- [ ] Add offline and installability examples.
	Create a small app-shell example that supports install prompt behavior, offline fallback UI, and cache-aware reload behavior on repeat visits.
- [ ] Add production guidance for PWA deployments.
	Document HTTPS requirements, cache headers, service-worker scope, and failure modes when deploying behind CDNs or reverse proxies.

### Security, compliance, and governance

- [ ] Produce a framework threat model for browser, SSR, and bootstrap flows.
	Document the primary trust boundaries around hydration payloads, route data, auth/session hints, interop calls, offline storage, and worker communication so security-sensitive behavior is designed rather than assumed.
- [ ] Define strict server-only versus client-safe data boundaries.
	Specify which configuration, loader data, auth/session fields, and runtime metadata may cross into bootstrap payloads and which must never be serialized to the browser.
- [ ] Add secure-by-default guidance for SSR and hydration.
	Document escaping rules, payload embedding rules, script injection boundaries, CSP considerations, and safe defaults for rendering user-controlled content.
- [ ] Add redaction and secret-handling policy for diagnostics and logs.
	Ensure framework diagnostics, structured logs, and exported traces avoid leaking tokens, headers, personally sensitive values, or internal bootstrap details in development and production.
- [ ] Define dependency and supply-chain review practices.
	Document expectations for third-party toolchain dependencies, npm-based dev tooling, wasm post-processors, and release-time artifacts so enterprise adopters can assess supply-chain risk.
- [ ] Add security regression tests for critical surfaces.
	Cover bootstrap serialization, SSR escaping, interop boundaries, config injection, and logging redaction so security-sensitive invariants are enforced in CI.
- [ ] Publish incident response and vulnerability reporting guidance.
	Document how security issues are reported, triaged, patched, and disclosed so enterprise consumers have a credible escalation path.
- [ ] Define compliance-oriented deployment guidance.
	Provide recommendations for audit logging, retention, reproducible builds, environment segregation, and operational controls that matter in regulated enterprise environments.

### Feature flags and environment configuration

- [ ] Define the framework-level runtime configuration model.
	Decide how applications inject environment-specific values, public runtime config, and deployment metadata into client, server, and hydrated code paths without relying on scattered global variables.
- [ ] Add a first-class feature-flag context and evaluation surface.
	Support exposing flag values to component trees, route loaders, and form or mutation flows so staged rollouts do not require bespoke context wiring in every app.
- [ ] Define server-to-client transfer rules for public config and flags.
	Clarify how runtime config and evaluated flags move through SSR bootstrap, hydration, and static export flows without leaking server-only secrets into public payloads.
- [ ] Add environment layering and override rules.
	Document how defaults, build-time values, request-time overrides, user-targeted flags, and local development overrides compose when several config sources are present.
- [ ] Add gating semantics for routes, components, and experiments.
	Support feature-flagged route availability, component branching, experimental UI exposure, and progressive rollout behavior without forcing every app to scatter flag checks manually.
- [ ] Add diagnostics and safety guidance for flag usage.
	Expose active config and flag state during development, document safe secret boundaries, and discourage stale or permanently dead rollout branches from lingering unnoticed.
- [ ] Add examples for staged rollout and environment-aware apps.
	Demonstrate a feature-gated route, an environment-configured API endpoint, and an SSR-aware flag evaluation flow that stays consistent between server and client.

### Wasm build optimization, size, and release engineering

- [ ] Define first-class development and production wasm build profiles.
	Document the canonical differences between local development builds, CI verification builds, benchmark builds, and production release builds so the repo stops relying on one generic `go build` path.
- [ ] Add recommended production build flags for wasm targets.
	Evaluate and document a baseline production command using options such as `-trimpath`, `-ldflags="-s -w"`, and other safe release-oriented flags, together with the tradeoffs for debugging and reproducibility.
- [ ] Define build metadata and debug-info policy for release wasm artifacts.
	Clarify when symbol tables, DWARF data, build IDs, and VCS metadata should be stripped, preserved, or emitted separately so production size and debugging needs are balanced intentionally.
- [ ] Add wasm size budgets and regression tracking.
	Track artifact sizes for key examples or release targets, define acceptable thresholds, and fail CI when size regressions exceed agreed budgets.
- [ ] Add artifact size reporting and comparison tooling.
	Produce per-build size summaries for raw `.wasm`, gzip-compressed size, brotli-compressed size, and any sidecar assets so regressions are attributable instead of anecdotal.
- [ ] Add release-time compression support for wasm artifacts.
	Generate gzip and brotli sidecars for production builds and document the required server headers and cache behavior for serving compressed wasm safely.
- [ ] Evaluate post-link wasm optimization tooling.
	Test tools such as `wasm-opt` or equivalent post-processing pipelines and document whether they improve size, startup time, or runtime behavior enough to justify adding them to the release workflow.
- [ ] Define asset-manifest and build-output conventions for optimized wasm releases.
	Specify where optimized wasm files, compressed variants, manifest files, and helper scripts land so static and server-backed deployments can reference artifacts consistently.
- [ ] Add reproducible release-build guidance.
	Document the toolchain versions, flags, manifest outputs, and hash expectations required to produce stable release artifacts across CI runs and local release workflows.
- [ ] Add startup-cost measurements for release artifacts.
	Track download size, decompression cost, compile or instantiate time, and first-interaction timing for representative builds so size work is tied to real user-facing outcomes.

### Build speed and optimization experiments

- [ ] Establish an experiment matrix for wasm build optimization.
	Define a repeatable comparison matrix covering plain builds, stripped builds, `-trimpath`, `-buildvcs=false`, profile-guided builds if applicable, `wasm-opt` post-processing, and compressed delivery variants so optimization decisions are evidence-driven.
- [ ] Add scripted measurement of cold and warm build times.
	Benchmark first build, cached rebuild, and small-edit rebuild times for representative examples so build-speed optimizations target real developer pain instead of assumptions.
- [ ] Add experiment harnesses for wasm size and startup tradeoffs.
	Automate capture of raw artifact size, compressed size, browser download cost, instantiate time, and initial render or hydration timing across build variants.
- [ ] Evaluate Go build flags for size versus speed tradeoffs.
	Measure the effect of release-oriented flags on output size, compile time, startup time, and runtime benchmarks instead of assuming smaller binaries are always better overall.
- [ ] Evaluate `GOWASM` feature toggles and compatibility tradeoffs where relevant.
	Document whether available wasm feature switches materially affect compatibility, code size, or performance for supported target browsers before standardizing on a default.
- [ ] Evaluate post-processing and compression combinations.
	Compare plain wasm, stripped wasm, optimized wasm, gzip delivery, and brotli delivery to identify which combination provides the best real-world startup characteristics for this project.
- [ ] Add CI-friendly benchmark comparison for build experiments.
	Reuse or extend the existing benchmark tooling so candidate build settings can be compared against a saved baseline before they are adopted into the documented release path.
- [ ] Record accepted and rejected build optimizations in docs.
	Keep a short history of build-flag and tooling experiments, including what was tried, what regressed build speed or startup time, and what combinations are currently recommended.

### Developer workflow and project bootstrap

- [ ] Define the official way to start a new GoWebComponents app.
	Decide whether the project should provide a starter template, a small bootstrap CLI, a repo template, or a documented manual flow so new adopters are not forced to assemble build, serve, routing, and wasm wiring from scattered examples.
- [ ] Add at least one maintained starter application for the common path.
	Ship a polished baseline app that includes routing, async data, state, forms, testing, and production-minded build setup so teams can begin from a realistic foundation instead of a toy counter example.
- [ ] Add starter variants for the major adoption modes.
	Provide clearly scoped starting points for client-only apps, SSR-enabled apps, and content or dashboard-style apps so teams can choose an architecture without reverse-engineering the examples directory.
- [ ] Define upgrade and template-sync guidance for starter-based apps.
	Document how apps created from official starters absorb framework improvements, build changes, and recommended conventions without blindly diffing the repo by hand.
- [ ] Add a one-command local bootstrap workflow.
	Make it easy to install prerequisites, build wasm, start the dev server, and open a working example or starter app without several undocumented manual steps.
- [ ] Document environment prerequisites and platform expectations clearly.
	State the required Go version, Node or npm usage, browser expectations, optional tooling, and Windows/macOS/Linux workflow notes in one place so setup failures happen less often.
- [ ] Add a “choose your path” onboarding flow for new adopters.
	Help developers select the right starting point based on whether they want SSR, routing, forms, data fetching, static export, or a mostly client-rendered app.

### Build feedback loop ergonomics

- [ ] Define the recommended inner-loop workflow for application authors.
	Document the fastest supported edit-build-refresh loop for typical app work, including which commands to run, when rebuilds occur, and how developers should reason about browser refresh versus state-preserving updates.
- [ ] Add a simplified watch-mode entry point for apps.
	Reduce the need to coordinate several scripts manually by providing one supported development command that rebuilds wasm, serves assets, and reports failures coherently.
- [ ] Improve incremental rebuild behavior for small source edits.
	Measure and reduce avoidable rebuild work so common component, route, and style changes do not feel disproportionately expensive during local development.
- [ ] Add clearer surfacing for build progress and failure states.
	Show whether the system is compiling, serving stale output, waiting on a reload, or blocked on an error so developers are not left guessing which part of the toolchain failed.
- [ ] Add recovery guidance for broken local development loops.
	Document what to check when watch mode stalls, browsers run stale wasm, assets are not reloaded, or build output and served output drift apart.
- [ ] Add example workflows for repo contributors versus framework consumers.
	Differentiate the commands and expectations for working on the framework itself versus building an app on top of it so the tooling story scales beyond this repository.

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
- [ ] Evaluate whether compile-time reactivity is compatible with the current hook model.
	Determine whether any Svelte- or Solid-like compile step can coexist with `UseState` and `UseEffect` semantics without splitting the framework into two mental models.
- [ ] Define source-language boundaries for compiler work.
	Clarify whether compiler experiments target Go source only, HTML-like templates, generated Go helpers, or browser-hosted tooling.
- [ ] Add a migration and fallback plan for compiler-generated output.
	Users should be able to inspect, debug, and opt out of generated code paths if compile-time ergonomics ship.

### State-preserving hot reload

- [ ] Decide whether state-preserving hot reload is a first-class development goal.
	Clarify whether the framework aims for true module-level replacement during development or a narrower snapshot-and-remount workflow that preserves common state shapes.
- [ ] Audit which runtime assumptions block hot replacement.
	Identify where component identity, hook ordering, global registries, route registration, and effect lifecycle assumptions prevent swapping updated code without a full reload.
- [ ] Define the preservation boundary for local and shared state.
	Specify which state categories may survive a development reload, such as local hook state, atom state, form state, route state, and in-flight async resource state, and which categories reset intentionally.
- [ ] Add a development-time component identity and signature model.
	Track enough metadata to determine when a component edit is shape-compatible for state preservation and when the runtime must fall back to full remount.
- [ ] Add effect cleanup and re-run semantics for hot updates.
	Ensure preserved components still dispose stale effects, listeners, timers, and JS handles when their implementation changes during development.
- [ ] Define error recovery behavior during failed hot updates.
	Specify how syntax errors, build errors, and runtime failures roll back or invalidate pending hot updates without leaving the dev session in a corrupted state.
- [ ] Integrate the live reload server with state snapshot transport.
	Extend the current dev tooling so it can coordinate update payloads, runtime invalidation, and optional state snapshot restore instead of only forcing a full page reload.
- [ ] Add examples and benchmarks for preserved-state development flows.
	Validate editing a counter, form, routed screen, and atom-backed shared state without losing state unnecessarily while still preserving correctness.

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

### Code splitting and lazy bundle delivery

- [ ] Define the code-splitting model for the framework.
	Decide whether splitting is route-driven, component-driven, build-tool-driven, or some combination, and how it should relate to the existing `ui.Lazy` and async-boundary surface.
- [ ] Add a first-class lazy asset and module loading pipeline.
	Support loading deferred WASM-adjacent assets, JS helpers, route bundles, or generated code chunks through a public mechanism instead of leaving lazy delivery entirely to ad hoc app tooling.
- [ ] Define route-level code-splitting conventions.
	Specify how large routed applications split feature areas, preload likely next routes, and avoid loading the full app codepath before first paint.
- [ ] Define component-level chunk boundaries and loading semantics.
	Clarify how lazily loaded components declare their loading boundary, error fallback, retry behavior, and compatibility with hydration or resumed state.
- [ ] Add preload and prefetch hooks for deferred bundles.
	Allow apps and routers to warm likely-next chunks on hover, idle time, viewport visibility, or route intent rather than waiting for the final navigation click.
- [ ] Define SSR interactions for split bundles.
	Specify how prerendered or server-rendered pages declare which chunks are needed on the client, how those assets are discovered, and how hydration avoids bundle-order races.
- [ ] Add build-output and asset-manifest conventions for split bundles.
	Document how split artifacts are named, versioned, referenced from HTML, and invalidated across deployments so lazy loading remains production-safe.
- [ ] Add end-to-end examples for route and component splitting.
	Create examples showing a heavy route and a heavy nested panel loading on demand with meaningful fallbacks, preload hints, and measured startup improvements.

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

### Team-scale conventions and developer ergonomics

- [ ] Publish recommended project structure for non-trivial apps.
	Document how teams should organize routes, reusable components, state modules, async resources, forms, tests, and deployment-specific code once an app grows beyond small examples.
- [ ] Define conventions for shared UI and domain abstractions.
	Show how teams should factor design-system components, feature modules, route-local logic, and shared utility layers so large apps do not become a flat collection of unrelated files.
- [ ] Add guidance for framework usage consistency across teams.
	Document preferred patterns for hooks, atoms, route loaders, async resources, and form handling so teams converge on one idiomatic style instead of inventing incompatible local conventions.
- [ ] Define code-review and migration checklists for framework-heavy changes.
	Provide practical review criteria for hydration-sensitive code, route changes, async data flows, and interop boundaries so quality does not depend entirely on tribal knowledge.
- [ ] Add recommended linting, formatting, and repository hygiene guidance.
	Even if enforcement lives outside core, document what a healthy app repository should standardize for imports, generated artifacts, tests, examples, and build outputs.
- [ ] Add guidance for multi-person ownership of app architecture.
	Show how teams can divide route areas, shared state, testing responsibility, and deployment concerns without producing conflicting local patterns or duplicate framework wrappers.
- [ ] Add examples of medium-sized app structure and conventions.
	Use a realistic reference app to demonstrate folder layout, dependency boundaries, naming patterns, and testing placement for a team-maintained codebase.

### Enterprise readiness and deployment proof

- [ ] Define the minimum bar for an enterprise pilot.
	List the required correctness, testing, observability, security, deployment, and support capabilities that must be complete before the project should be recommended for a serious internal pilot.
- [ ] Add a production-shaped reference application.
	Ship a medium-size app that exercises SSR, hydration, routing, auth-aware flows, forms, async data, offline behavior, observability hooks, and operational deployment patterns under one coherent codebase.
- [ ] Add operational runbooks for production incidents.
	Document how to debug hydration failures, loader failures, offline replay issues, cache corruption, multi-window sync problems, and degraded route performance in deployed environments.
- [ ] Add upgrade rehearsal guidance for real applications.
	Show how a non-trivial app verifies framework upgrades through contract tests, benchmark comparisons, and regression checks before rolling into production.
- [ ] Add deployment validation checklists.
	Provide pre-release checks for artifact integrity, config correctness, observability wiring, cache headers, compression, CSP, and SSR/bootstrap behavior so teams can standardize release readiness.
- [ ] Add sustained-load and long-session validation for the reference app.
	Run medium-duration browser and server scenarios that mimic real enterprise usage patterns instead of relying only on short-lived example interactions.

### Profiling and flamegraph-style analysis

- [ ] Define the next-level profiling surface beyond summary counters.
	Decide which runtime events should be profiled in detail, such as component rerenders, reconciliation work, DOM commit cost, effect timing, async boundary resolution, and route transitions.
- [ ] Add per-component render tracing.
	Record which components rerendered, how often they rerendered, and what high-level trigger caused the update so developers can find wasteful tree churn.
- [ ] Add timing attribution for render, diff, commit, and effect phases.
	Break runtime work into meaningful buckets so profiling output can distinguish expensive rendering from expensive DOM mutation or effect cleanup.
- [ ] Add flamegraph-style capture and visualization support.
	Provide a structured profiling format and devtools visualization that can show nested render cost over time instead of only flat counters or aggregate summaries.
- [ ] Add async and route-lifecycle profiling.
	Capture loader timing, hydration timing, async-boundary waits, transition delays, and route navigation phases so full app interactions can be profiled end to end.
- [ ] Add devtools export and snapshot comparison support.
	Allow developers to save profiling sessions, compare before/after traces, and inspect regressions across optimization attempts instead of relying on one-off local observation.
- [ ] Add profiling examples and performance regression tests.
	Use representative apps such as large lists, nested routes, async dashboards, and portal-heavy overlays to ensure the profiling surface remains useful for real bottlenecks.

### Debugging and devtools workflows

- [ ] Add richer component-stack and failure context for runtime errors.
	Include component ancestry, route context, active async resource state, and hydration phase details when render, effect, loader, or interop failures are reported.
- [ ] Add a “why did this rerender?” inspection surface.
	Show whether a rerender was triggered by props, local state, context, atoms, route changes, loader updates, or parent rerenders so wasted work is easier to diagnose.
- [ ] Add hook-slot and state-transition inspection.
	Expose current hook values, dependency snapshots, recent transitions, and effect lifecycle state for a selected component during development.
- [ ] Add route and loader debugging panels.
	Show current route stack, params, query, active guards, loader state, redirect causes, and route metadata ownership so route bugs can be inspected live.
- [ ] Add hydration debugging tools.
	Highlight reused nodes, replaced subtrees, mismatch locations, and fallback boundaries so hydration failures are easier to localize than raw console warnings.
- [ ] Add cache, worker, and synchronization inspectors.
	Expose active cache keys, worker jobs, cross-tab or multi-window events, and offline replay state in devtools so coordination bugs can be debugged without custom logging.
- [ ] Add strict development-mode toggles.
	Allow tests and local development to escalate specific recovered warnings into hard failures so incorrect-but-recovered behavior does not linger unnoticed.
- [ ] Add trace capture and replay support for debugging sessions.
	Allow developers to save one interaction trace, compare before/after behavior, and replay difficult timing-sensitive bugs without manually reconstructing state.

### Actionable errors and developer guidance

- [ ] Audit the highest-friction framework errors and warnings.
	Identify which failures currently surface as vague panics, generic console noise, or low-context runtime errors so error-improvement work targets the worst developer experience first.
- [ ] Add structured, actionable error messages for common mistakes.
	Improve messages for invalid hook usage, hydration mismatches, missing router context, misconfigured async boundaries, broken form wiring, and interop misuse so developers get a concrete next step instead of a dead end.
- [ ] Add error codes or stable diagnostic identifiers where appropriate.
	Allow documentation, troubleshooting guides, issue reports, and CI logs to refer to consistent framework diagnostics without depending on fragile message text.
- [ ] Link runtime diagnostics to docs and remediation guidance.
	Make warnings and errors point to the relevant troubleshooting or API guidance so users can move from failure to fix without searching the repo manually.
- [ ] Distinguish between recoverable warnings and correctness-threatening failures clearly.
	Ensure developers know when the framework recovered with degraded behavior versus when the application state or rendered output should not be trusted.
- [ ] Add development-time assertions for high-confidence misuse cases.
	Fail fast on incorrect usage patterns that should never be silently tolerated in development, while keeping production behavior intentional and documented.
- [ ] Add tests that lock in diagnostic quality.
	Verify not only that failures occur, but that key errors include route context, component context, or remediation hints when those details are expected.

### Public testing utilities for app authors

- [ ] Define the first-party testing surface for consumers.
	Decide whether testing support lives in one package or several focused helpers for component rendering, hook testing, router testing, and SSR assertions.
- [ ] Add a component render and query harness for tests.
	Provide a public way to mount a component into a controlled DOM fixture, trigger updates, and inspect rendered output without depending on internal runtime helpers.
- [ ] Add event and async-flush helpers for browser-facing tests.
	Support deterministic event dispatch, timer flushing, microtask draining, and render stabilization so consumer tests do not need fragile sleeps.
- [ ] Add hook-level testing utilities where feasible.
	Support isolated testing of custom hooks or hook-driven state flows without requiring a full hand-written host component for every test.
- [ ] Add router testing helpers.
	Provide utilities for setting initial routes, exercising navigation, asserting params and query behavior, and validating guard or loader outcomes in component tests.
- [ ] Add SSR and hydration assertion helpers.
	Support snapshotting `RenderToString(...)` output, bootstrap payload assertions, and hydration smoke tests so server-rendered apps can test their delivery path directly.
- [ ] Define stable public query semantics for test helpers.
	Decide whether first-party test queries are DOM-role and accessibility-first, selector-based, or a thin integration layer over existing browser-test tools.
- [ ] Add example tests and documentation for consumers.
	Document recommended unit, integration, router, and SSR testing patterns using the public helpers so adopters can copy real working examples instead of reverse-engineering repo internals.
- [ ] Add deterministic scheduler and flush helpers for tests.
	Support explicit render flushing, timer draining, microtask advancement, and effect settlement so tests do not depend on sleeps or incidental event-loop timing.
- [ ] Add async resource and loader test utilities.
	Provide helpers for resolving, rejecting, cancelling, retrying, and stalling resource or route-loader work so async UI flows can be tested precisely.
- [ ] Add portal and overlay testing helpers.
	Support asserting active overlay stacks, focus restoration, escape dismissal, outside-click behavior, and scroll-lock coordination in portal-heavy tests.
- [ ] Add accessibility-first assertions for public test utilities.
	Prefer role, label, description, and live-region queries so framework tests and consumer tests encourage accessible UI structure rather than brittle CSS selectors.
- [ ] Add mismatch and failure-injection helpers.
	Allow tests to intentionally trigger hydration mismatches, loader failures, route guard failures, cache conflicts, and offline replay errors so recovery behavior can be asserted directly.
- [ ] Add cross-tab, worker, and offline test harnesses.
	Provide controlled test environments for synchronization channels, worker messaging, background retry flows, and reconnect behavior so these coordination features are not tested only through ad hoc browser scripts.
- [ ] Add render-count and warning assertions.
	Allow tests to fail when scenarios emit unexpected warnings or rerender more often than expected so correctness and performance regressions can be caught earlier.

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

### Experimental tooling boundaries

- [x] The browser compiler example is explicitly documented as experimental and outside the core runtime path.

### Forms and examples

- [x] `ui.UseForm` provides a first-class form helper for local field state, validation, and submit lifecycle handling.
- [x] Complex form, nested routing, portals, fetch, state, goroutine, and SSR examples exist and exercise the public API.
