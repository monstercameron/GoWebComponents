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

### Example Alignment and Modernization

- [x] Replace legacy router compatibility calls in shipped examples with the primary router API.
	Shipped examples now consistently demonstrate `Register`, `Current`, and the main router surface instead of `GoRegisterRoute` and `GoGetRoute` compatibility calls.
- [x] Refresh `examples/12-portfolio-site` to stop teaching stale public APIs.
	The portfolio site documentation now points at the current `ui`, `state`, `fetch`, `router`, and `devtools` surface instead of obsolete `fiber`-era examples.
- [x] Add a review pass for example-local API reference pages whenever public APIs change.
	Embedded example docs were reviewed and refreshed so the larger showcase apps stay aligned with the current public API terminology.
- [x] Add explicit `UseDebounced` versus `UseThrottled` guidance to the text-input example or its README.
	`examples/02-text-input` now explains when to debounce versus throttle so the timing tradeoff is visible in the example itself.
- [x] Add a first-class `state.UseDerived` showcase to the examples set.
	The catalog now includes a dedicated `state.UseDerived` example alongside the other core state primitives.
- [x] Add an example inventory that maps each shipped example to the public APIs it demonstrates.
	`examples/README.md` now acts as an inventory that maps the catalog to specific public APIs and teaching goals.

### Feature-Isolated Example Expansion

- [x] Build a one-example-per-public-feature catalog instead of stopping at an arbitrary count.
	Treat the goal as coverage of the stable public surface, not a fixed number of demos, so each public primitive has at least one clean, isolated example.
- [x] Define a naming and directory convention for feature-isolated examples.
	Readers should be able to tell what an example teaches from the folder name alone instead of opening a large mixed-concern app first.

- [x] Add a dedicated `ui.Render` example.
	Show the smallest browser entrypoint using `ui.Render(ui.CreateElement(App), selector)` so the canonical mount path has its own minimal reference.
- [x] Add a dedicated `ui.RenderToString` example.
	Show the smallest server-side render flow so SSR starts from a tiny public example before readers move to the larger routing demos.
- [x] Add a dedicated `ui.Hydrate` example.
	Show client resume over pre-rendered HTML in a minimal app so hydration is not only taught through the full SSR examples.
- [x] Add a dedicated `ui.CreateElement` and typed props example.
	Show function components, props structs, and direct element creation so the core composition model is visible beneath helper wrappers.
- [x] Add a dedicated `ui.Fragment` example.
	Show multi-root child composition and conditional sibling rendering in a tiny app instead of leaving fragments as an invisible helper.
- [x] Add a dedicated `ui.UseRef` example.
	Teach persistent mutable values, DOM-node refs, and non-rendering instance state so developers know when refs are appropriate instead of `UseState`.
- [x] Add a dedicated `ui.UsePrevious` example.
	Keep it narrowly focused on render-time comparisons, first-render behavior, and previous committed values without routing or fetch noise.
- [x] Add a dedicated `ui.UseDeferredValue` example.
	Show a fast-changing input and a deferred results pane so scheduling behavior is obvious without a larger search app.
- [x] Add a dedicated `ui.StartTransition` and `ui.UseTransition` example.
	Demonstrate urgent versus non-urgent updates with visible pending state so the transition lane has a clear teaching example.
- [x] Add a dedicated `ui.UseReducer` example.
	Use a small state-machine-style component where reducer actions read more clearly than ad hoc `UseState.Update` calls.
- [x] Add a dedicated `ui.UseDebounced` example.
	Teach delayed derived values with a minimal search or validation preview so debounce stands on its own.
- [x] Add a dedicated `ui.UseThrottled` example.
	Teach rate-limited derived values with a tiny counter, scroll, or resize UI so the distinction from debounce is explicit.
- [x] Add a dedicated `ui.CreateContext` and `ui.UseContext` example.
	Show provider scoping, nested overrides, and default fallback resolution in isolation.
- [x] Add a dedicated `ui.AsyncBoundary` example.
	Use a small async subtree with explicit pending, content, and error fallbacks so the boundary contract is understandable without the full fetch example.
- [x] Add a dedicated `ui.Lazy` example.
	Show delayed node resolution, fallback rendering, and error fallback behavior with a minimal async component.
- [x] Add a dedicated `ui.ErrorBoundary` example.
	Create a deliberately failing child component and a local recovery UI so render/event/effect failure containment is visible without digging through tests.
- [x] Add a dedicated `ui.UseId` example.
	Show stable generated IDs across labeled inputs, repeated rows, and SSR-friendly markup so the purpose of the hook is concrete.
- [x] Add a dedicated typed event handling example.
	Demonstrate `ui.UseEvent` with `MouseEvent`, `InputEvent`, `KeyboardEvent`, and `FormEvent` so event typing is taught directly.
- [x] Add a dedicated `ui.RawHandler` example.
	Show how raw handler passthrough differs from `ui.UseEvent` and document when callers should avoid it.
- [x] Add a dedicated `ui.Portal` selector-target example.
	Teach the common modal or overlay case in a tiny app where only selector-based portal mounting is in play.
- [x] Add a dedicated `ui.PortalTarget` explicit-node example.
	Show rendering into an explicit DOM host node so advanced portal targeting is documented separately from the basic selector flow.
- [x] Add a dedicated `ui.UseChannel` example.
	Teach streamed values, latest-value semantics, and closed-channel state without coupling the hook to a larger goroutines showcase.
- [x] Add a dedicated `ui.UseTask` example.
	Show start, cancel, ready, cancelled, and error states in a tiny background-job UI so the task lifecycle is visible at a glance.
- [x] Add a dedicated `ui.UseForm` example.
	Focus on touched fields, dirty fields, validation, field errors, and submit lifecycle without router or multi-page concerns.

- [x] Add a dedicated `html` semantic layout example.
	Use the `html` package to build a small semantic page with headings, sections, nav, and article content so the package is taught independently from hooks.
- [x] Add a dedicated `html` form controls example.
	Show text fields, selects, checkboxes, labels, and textarea composition through typed `html.Props` instead of older attr-map patterns.
- [x] Add a dedicated `html.Tag` custom-element example.
	Show when to use the generic tag builder for custom elements or uncommon tags rather than relying only on predefined wrappers.

- [x] Add a dedicated `state.UseAtom` example.
	Keep one tiny shared-state demo focused on cross-component reads and writes before layering in computed or derived state.
- [x] Add a dedicated `state.UseComputed` example.
	Teach local typed derived values separately from shared derived atoms so developers can see when `UseComputed` is enough.
- [x] Add a dedicated `state.UseDerived` example.
	Keep it separate from `UseAtom` and `UseComputed` basics so shared derived atoms, explicit source dependencies, and read-only semantics are immediately obvious.
- [x] Add a dedicated state snapshot export/import example.
	Show `state.ExportSnapshot`, `state.ImportSnapshot`, and JSON round-tripping in one isolated example instead of leaving persistence buried in docs and tests.
- [x] Add a dedicated browser-storage snapshot restore example.
	Show `SaveSnapshot`, `LoadSnapshot`, and `RestoreSnapshot` flows in a tiny app so local/session storage persistence is taught separately from in-memory export/import.

- [x] Add a dedicated `fetch.UseFetch` example.
	Keep one example focused on the low-level raw fetch surface so developers can see when `UseFetch` is the right fit before graduating to typed resources.
- [x] Add a dedicated `fetch.UseResource[T]` example.
	Show typed loading, reload, cancellation, and typed error handling without the extra complexity of shared caching.
- [x] Add a dedicated `fetch.UseCachedResource[T]` example.
	Focus on shared cache reuse, stale-while-revalidate behavior, manual invalidation, and optimistic updates across two components reading the same key.
- [x] Add a dedicated imperative `fetch.Fetch` example.
	Show event-handler and goroutine-driven fetch flows so imperative fetching has a clear teaching reference alongside the hook-based surfaces.

- [x] Add a dedicated hash-router basics example.
	Show `router.NewHashRouter`, `Register`, and `Mount` in the smallest possible app so the default static-hosting router path is obvious.
- [x] Add a dedicated browser-router basics example.
	Show `router.NewRouter`, path-based URLs, and server-rewrite expectations without also introducing loaders or SSR.
- [x] Add a dedicated `router.UseNavigate` example.
	Teach push versus replace navigation through a tiny two-page app so imperative routing is easy to discover.
- [x] Add a dedicated router params example.
	Show `router.UseParams()` with typed accessors on a small detail page so route parameter parsing is taught cleanly before nested routes or SSR.
- [x] Add a dedicated router query example.
	Show `router.UseQuery()` and `router.UseSearchParams()` on a small filter or sort page so query reading, replacing, deleting, and serialization are demonstrated without unrelated route loaders.
- [x] Add a dedicated `router.UseRevalidator` example.
	Show route-loader reruns without path changes so manual revalidation is discoverable outside the SSR demos.
- [x] Add a dedicated route loaders example.
	Teach `router.Options{Loader: ...}` with loading and error renderers in a minimal app before readers encounter the larger OMI or SSR routing examples.
- [x] Add a dedicated nested layout routes example.
	Show `router.Options{Layout: true}` and `router.Outlet()` in the smallest possible app so nested composition is taught separately from dashboards and docs shells.
- [x] Add a dedicated router guards example.
	Use a minimal auth or unsaved-changes flow to show `BeforeEnter`, `BeforeLeave`, `AllowNavigation`, `BlockNavigation`, and `RedirectNavigation` without SSR concerns.
- [x] Add a dedicated router redirects example.
	Show declarative `router.Options{Redirect: ...}` behavior in isolation so redirect semantics are not hidden inside larger auth demos.
- [x] Add a dedicated router metadata example.
	Show route-managed title, description, and canonical URL updates in a tiny app so metadata behavior is discoverable without reading the router README.
- [x] Add a dedicated router `HydrateMount` example.
	Teach how a router attaches to already-hydrated markup without forcing an immediate rerender, independent of the larger server-routing examples.

- [x] Add a dedicated `devtools.Panel` example.
	Keep a minimal panel-embedding example that only teaches setup, refresh cadence, and placement, separate from broader diagnostics scenarios.
- [x] Add a dedicated `devtools.UseSnapshot` example.
	Show polling and rendering inspection snapshots inside app UI so the snapshot hook is documented apart from the panel itself.
- [x] Add a dedicated `devtools.SnapshotNow` example.
	Show one-off inspection reads from a button or diagnostics drawer so imperative inspection is demonstrated separately from subscription-based polling.
- [x] Add a dedicated devtools diagnostics example.
	Keep a smaller companion to the current panel showcase that intentionally triggers duplicate-route, invalid-hook, or missing-key-style diagnostics so the debugging surface can be learned feature-by-feature.

- [x] Add a dedicated SSR bootstrap example.
	Show `ui.RenderToString`, bootstrap payload emission, and first-client resume in the smallest possible app so the bootstrap story is documented outside the full SSR routing examples.
- [x] Add a dedicated SSR route-data reuse example.
	Show how first-route loader data is reused across hydration before normal client navigation takes over, without the extra surface area of the full server-routing example.
- [x] Add a dedicated example index page that groups the expanded catalog by package and feature.
	Once the one-feature-per-example set grows, the examples landing page should expose filters by `ui`, `html`, `state`, `fetch`, `router`, `devtools`, and SSR or hydration topics.

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

<<<<<<< HEAD
- [x] Design nested route composition.
	The router now supports opt-in layout routes through `router.Options{Layout: true}`, so parent routes can render persistent shells while a more specific child route renders into `router.Outlet()`.
- [x] Add an outlet-style API.
	`router.Outlet()` now exposes the matched child route during layout-route rendering, keeping nested composition explicit instead of relying on implicit child props.
- [x] Support shared layouts for dashboards, docs, and authenticated sections.
	Layout routes now wrap matching descendants without remounting shared chrome, and nested guard evaluation applies layout hooks across child-route transitions.
- [x] Define active route resolution rules.
	The router now builds layout stacks from shallowest matching `Layout: true` prefixes to the final leaf route, with parent params scoped to each layout level and child routes receiving the final merged param set.
- [x] Add examples for multi-level apps.
	`examples/19-nested-routes` now demonstrates docs navigation, a dashboard shell, and a nested settings layout stack using explicit `router.Outlet()` composition.

### Go-Native Hook Additions

#### Best Fit

- [x] Add router access hooks such as `router.UseParams`, `router.UseQuery`, and `router.UseNavigate`.
	The router now ships `UseNavigate`, `UseQuery`, and `UseParams`, so routed components no longer need to parse `location` or call navigation globals directly for common cases.
- [x] Define typed parameter access patterns.
	`router.Params` now includes typed helpers such as `Int` and `Bool`, reducing the need for manual scalar parsing while keeping the surface area small.
- [x] Ensure hook behavior is consistent across hash and history routers.
	The current hook set reads path/query information through shared router helpers, including query parsing for both `window.location.search` and hash-based query strings.
- [x] Add nested route navigation examples to round out the router hook story.
	The OMI example now covers detail-page params, query-driven filtered search, and nested child-path navigation under `/playground/:mode`.

- [x] Design a typed async resource hook as `fetch.UseResource[T]`.
	The initial implementation accepts a Go-style loader of type `func(ctx context.Context) (T, error)` and returns a typed handle instead of unstructured state.
- [x] Define the handle shape for typed async resources.
	`fetch.UseResource[T]` now exposes a typed handle with `Get`, `Reload`, and `Cancel`, while the returned state carries `Value`, `Loading`, `Error`, and `Ready`.
- [x] Include cancellation and retry semantics in the resource hook.
	The initial version uses `context.Context` cancellation and explicit `Reload()` retries rather than JS-style promise orchestration.
- [x] Align typed async resources with existing `fetch.UseFetch` behavior.
	The fetch package docs and examples now clearly position `UseFetch` as the low-level raw fetch hook and `UseResource[T]` as the preferred typed option for non-trivial loading.
- [x] Add examples for list loading, detail loading, retries, and cancellation.
	The fetch example app now uses `fetch.UseResource[T]` for list and detail loading, and includes explicit reload and cancellation controls for both resource flows.

- [x] Evaluate `ui.UseTask[T]` for cancellable background work.
	`ui.UseTask[T]` now models long-running jobs with explicit `Start`, `Cancel`, and typed state instead of open-coded goroutine bookkeeping.
- [x] Design `UseTask[T]` around `context.Context` and typed results.
	The first version uses `context.Context` cancellation and a typed `TaskState[T]`, keeping the API aligned with Go concurrency patterns.
- [x] Define task lifecycle states.
	The initial implementation supports idle, running, ready, cancelled, started, and error states with deterministic transitions.
- [x] Add examples that replace manual task state structs.
	The goroutine example now uses `ui.UseTask` for its background task flow, replacing the previous manual cancel-channel task wiring.

### Route Data Loading

- [x] Add route-level loaders.
	Routes can now define async loaders through `router.Options{Loader: ...}` and receive loader data through route props.
- [x] Define loader caching and revalidation behavior.
	Loaders are keyed by route path plus query string, rerun when that key changes, and reuse the latest resolved result for the current key.
- [x] Add loading and error states at the route layer.
	Routes can now define route-scoped `Loading` and `Error` renderers, with built-in defaults when those are omitted.
- [x] Support cancellation on rapid navigation.
	In-flight route loaders are cancelled when navigation switches to a different route key so stale results do not commit.
- [x] Add protected-route loader examples to finish out the route-data story.
	The OMI example now covers a detail-page loader (`/data/:id`), a query-driven search loader (`/search?q=...`), and a protected-route loader flow (`/secure`).

## Priority 3: Runtime and Tree Coordination Work

### Context API

- [x] Design a public context API for the `ui` package.
	`ui.CreateContext`, `context.Provider`, `ui.ContextProviderProps[T]`, and `ui.UseContext` now expose the first public subtree-context surface in the current `ui` API.
- [x] Define behavior for missing providers.
	Missing providers currently resolve to the context default value; `ui.UseContext` only panics when called with a nil context handle.
- [x] Add runtime support for scoped context propagation through the fiber tree.
	`internal/runtime` now propagates provider values through the fiber tree, supports subtree overrides, and forces dependent descendants to rerender when provider values change.
- [x] Add tests for nested providers and re-render behavior.
	Runtime tests now cover default resolution, nested provider overrides, and provider-change rerender behavior; `ui` package tests cover the public provider element surface.
- [x] Document the intended usage model.
	README and `ui` package docs now describe context usage for theme, auth/session state, configuration, and service-style dependency injection.

### Async State and Resource Caching

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
=======
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
>>>>>>> 0b39694 (docs: reorganize TODO backlog)

### Error boundaries

<<<<<<< HEAD
- [x] Define an error boundary component contract.
	Decide whether boundaries are function-based, struct-based, or a special component wrapper with fallback rendering.
- [x] Capture render-time panics at subtree boundaries.
=======
- [ ] Define an error boundary component contract.
	Decide whether boundaries are function-based, wrapper-based, or another explicit component form with fallback rendering.
- [ ] Capture render-time failures at subtree boundaries.
>>>>>>> 0b39694 (docs: reorganize TODO backlog)
	Prevent a child component failure from crashing the entire app tree when a boundary is present.
- [x] Support fallback UI rendering with error details.
	Allow users to render fallback content and optionally inspect the recovered error value.
<<<<<<< HEAD
- [x] Define reset behavior after recovery.
	Specify how boundaries retry after route changes, prop changes, or explicit resets.
- [x] Add coverage for render, effect, and event handler failure cases.
	Be explicit about which failure modes boundaries catch and which remain global errors.
=======
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
>>>>>>> 0b39694 (docs: reorganize TODO backlog)

## 6. Scheduling and Runtime Coordination

<<<<<<< HEAD
- [x] Design a Suspense-like async boundary model.
	The first async-boundary slice now ships as explicit `ui.AsyncBoundary`, where callers provide `Pending`, `Error`, fallback nodes, and optional delay/timeout behavior instead of relying on implicit promise throwing.
- [x] Add lazy component loading support.
	`ui.UseLazyNode` and `ui.Lazy` now provide a first-class way to resolve a subtree asynchronously while reusing the same boundary semantics.
- [x] Define async resource integration points.
	The initial integration model is explicit rather than magical: `fetch.UseResource` and other async state can now flow through `ui.AsyncBoundary` without open-coded branch ladders in each component.
- [x] Specify timeout and retry behavior.
	`ui.AsyncBoundary` now supports explicit delay and timeout fallback thresholds, while `ui.UseLazyNode` exposes `Reload` and `Cancel` for caller-driven retry/cancellation control.
- [x] Add examples that replace manual loading flag plumbing.
	The fetch example now routes both list and detail loading through `ui.AsyncBoundary` and includes a deferred `ui.Lazy` panel to demonstrate nested fallback behavior.

### Concurrent-style Scheduling Primitives

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
=======
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
>>>>>>> 0b39694 (docs: reorganize TODO backlog)

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

<<<<<<< HEAD
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
=======
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
>>>>>>> 0b39694 (docs: reorganize TODO backlog)

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
