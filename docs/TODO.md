# GoWebComponents TODO

This backlog focuses on the top-level framework features that are still missing compared to React, Svelte, Vue, and Solid. Items are ordered from easier, lower-risk implementation work toward deeper runtime and architecture changes.

## Priority 1: Easiest Wins and API Cleanup

### Documentation and Public API Hygiene

- [x] Reconcile docs with the actual supported API surface.
	The root README, package docs, and example docs now align on the current public APIs, shipped examples, SSR/hydration behavior, and the distinction between shipped versus future-facing backlog items.
- [x] Add a framework feature matrix to the docs.
	The root README now includes a project-wide feature inventory covering the runtime, hooks, state, router, SSR/hydration, devtools, testing, benchmarks, and example coverage.
- [x] Clarify which packages are stable public APIs versus internal/runtime-only details.
	The README and package docs now call out `ui`, `html`, `state`, `fetch`, `router`, and `devtools` as the public surface, while `internal/runtime` remains implementation detail.
- [ ] Add migration notes as new primitives land.
	Keep early adopters from depending on unstable patterns.

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

### Head and Metadata Management

- [x] Add a strategy for document title and metadata updates.
	The router now manages route titles, descriptions, and canonical URLs through `router.Options`, so common page metadata no longer requires direct `syscall/js` usage.
- [x] Define ownership and cleanup behavior.
	Route-managed metadata is replaced on route changes, and omitted description/canonical values are removed so stale tags do not linger.
- [ ] Decide whether SSR metadata support is a future requirement.
	This should align with any server rendering plan.
- [x] Add examples for page title, description, and canonical URL management.
	The router docs now include route option examples for title, description, and canonical URL management.

### Portals and Slots as Public APIs

- [x] Promote runtime portal support into the public API.
	`ui.Portal(...)` now exposes a stable public helper for rendering a subtree outside the current DOM parent while keeping the logical component ownership in place.
- [x] Clarify target selection for portals.
	Portals now support both selector-based targets and explicit DOM node targets through `ui.PortalTarget`.
- [ ] Decide whether slots are part of the public composition model.
	If they are kept, document when to use them versus regular children.
- [x] Add examples for modals, tooltips, and popovers.
	`examples/20-portals` now demonstrates modal, tooltip, and popover overlays sharing a dedicated portal mount.
- [x] Add tests for event propagation and cleanup.
	Runtime tests now cover selector and explicit-node targets, retargeting, and cleanup on unmount, while the example Playwright suite validates interactive overlay behavior through the public portal API.

### Go-Native Hook Additions

#### Best Fit

- [x] Add `ui.UsePrevious[T]` as a small but practical utility hook.
	Implement it on top of refs and keep the contract minimal: expose the previous committed value without triggering extra renders.
- [x] Document the intended use cases for `UsePrevious[T]`.
	Restrict it to comparison, transition detection, and debugging-oriented UI logic so it does not become cargo-culted everywhere.

- [x] Design `state.UseComputed[T]` as a typed derived-state hook.
	Provide a first-class derived-state primitive that fits alongside `state.UseAtom` instead of forcing callers into raw `UseMemo` usage at every call site.
- [x] Define dependency tracking rules for `state.UseComputed[T]`.
	The initial version uses explicit dependency arguments, keeping recomputation rules predictable while leaving room for richer derived-atom dependency tracking later.
- [x] Decide whether computed state is read-only or optionally writable.
	The first version stays read-only through `state.Computed[T]` and `Get()`, avoiding writable derived state until a repeated real-world need emerges.
- [x] Add examples for theme, filtered collections, and derived totals.
	The state README now covers all three `UseComputed` patterns, and the atoms example uses computed theme/count display values so the hook is exercised in a real component.

- [x] Evaluate `ui.UseChannel[T]` as a Go-specific differentiator.
	Components can now subscribe to a receive-only channel through a typed handle that exposes the latest value together with availability and closed state.
- [x] Define how `UseChannel[T]` integrates with the scheduler.
	The initial version feeds channel updates through the existing state hook path, which keeps scheduling behavior consistent with other component-driven updates.
- [x] Define cleanup and closure behavior.
	The initial implementation stops its reader goroutine on unmount or channel change and records observed closure state on the public handle.
- [x] Add examples based on timers, worker-style tasks, and streamed updates.
	The goroutine example now includes a timed channel stream alongside the existing background task and timer flows, validating the hook against real concurrency-heavy UI patterns.

#### Medium Fit

- [x] Evaluate lightweight lifecycle wrappers only if they add real semantics.
	Current conclusion: no `UseMounted`/`UseLifecycle` wrapper has been added because a small `UseEffect` remains clearer than another lifecycle alias.
- [x] Reject thin wrappers that only rename `UseEffect` without reducing complexity.
	The current public hook set keeps this line tight and adds behavior-focused hooks rather than renaming lifecycle phases.

- [x] Evaluate `ui.UseDebounced[T]` and `ui.UseThrottled[T]` as optional convenience hooks.
	The current implementation uses `time.Duration`, trailing updates, and hook-managed timer cleanup, which keeps the feature aligned with the existing hook model.
- [x] Decide whether debounce and throttle helpers belong in `ui` or a companion package.
	They now live in `ui` because the behavior is small, broadly useful in input-heavy components, and simpler to adopt than a separate utility package.
- [x] Add search and input-heavy examples before stabilizing these hooks.
	The text-input example now shows immediate, debounced, and throttled views of the same input so the hooks are exercised in a real component.

- [x] Evaluate `ui.UseReducer` only if complex local state patterns become common.
	`ui.UseReducer` is now part of the public `ui` API and has wasm coverage for typed reducer-style local state transitions.
- [x] Compare `ui.UseReducer` against existing struct-state patterns.
	For simple structs, `UseState.Update` remains smaller, but reducer-style actions are clearer once several fields and submit states transition together.
- [x] Add reducer examples only if the API proves clearer than `UseState.Update`.
	Current conclusion: keep `UseReducer` available and tested, but do not force a dedicated showcase example where `UseState.Update` or `UseForm` reads more clearly.

## Priority 2: Moderate Complexity Feature Work

### State Persistence and Snapshots

- [x] Decide whether snapshot APIs should be promoted from examples into supported public exports.
	Snapshot export/import and storage persistence now live in the public `state` package, so the feature is no longer example-only.
- [x] Add a public persistence API if the feature is kept.
	The `state` package now exposes snapshot export/import plus key-based selection for partial persistence flows.
- [x] Define serialization constraints.
	The docs now distinguish exact same-process snapshot restore from JSON/browser-storage persistence, which is only stable for JSON-compatible atom values.
- [x] Add browser storage helpers.
	Local and session storage helpers now wrap JSON persistence so apps do not need to hand-roll storage glue.
- [x] Add hot reload state restoration tests.
	Runtime and wasm-level tests now cover snapshot export/import and browser-storage restore flows for atom state.

### Form State and Validation

- [x] Decide whether forms stay manual or become a first-class feature.
	The repo now exposes a minimal `ui.UseForm` helper for multi-field local forms, covering field values, touched state, dirty state, validation errors, reset, and submit lifecycle without introducing a separate heavy forms package.
- [x] Add validation helpers.
	`ui.Form` now supports synchronous validation, asynchronous validation, structured field errors, and form-level validation errors before submit.
- [x] Add structured error collection.
	The new `ui.FieldErrors` shape provides a consistent field-error map instead of ad hoc per-example structs.
- [x] Support submission lifecycle handling.
	`ui.Form` now reports pending, success, failure, and retry/reset flows through its submission helpers.
- [x] Add examples for complex forms.
	The advanced-form example now covers multi-field validation, async submit, and route-integrated submission/reset flows through the router.

### Devtools and Inspection

- [x] Define the minimum useful debugging surface.
	The first shipped `devtools` surface now covers committed component tree visibility, hook state inspection, route inspection, and runtime summary stats.
- [x] Decide whether tooling should live in-browser, in VS Code, or both.
	The first implementation is intentionally in-browser through an embeddable `devtools.Panel`, avoiding extension-only tooling while the surface is still stabilizing.
- [x] Add structured runtime diagnostics for development builds.
	The runtime and router now report structured diagnostics for invalid hook usage, missing render targets, duplicate route registration, and invalid route component configuration.
- [x] Expand profiling beyond the shipped baseline counters.
	The devtools surface now includes subtree hot-branch attribution plus effect and cleanup timing, and reports slow effect/cleanup paths through runtime diagnostics.

### Derived and Computed State

- [x] Add first-class derived atom support.
	The `state` package now exposes read-only shared derived atoms through `UseDerived`, so shared derivation no longer needs to be open-coded with component-local memo hooks.
- [x] Define dependency tracking for derived state.
	Derived atoms now use explicit source atom ID dependencies, and recompute only when those source atoms change.
- [x] Add read-only computed atoms and writable derived atoms if useful.
	The current design stays intentionally simple: `UseComputed` and `UseDerived` are both read-only, while writes continue to happen through source atoms.
- [x] Add cycle detection or safe failure modes.
	The runtime rejects simple self-referential derived registrations and reports derived-cycle failures as diagnostics instead of allowing silent infinite loops.
- [x] Document best practices for expensive shared computations.
	The state docs now distinguish local `UseComputed` from shared `UseDerived`, recommend explicit dependency lists, and discourage unnecessary long derived chains.

### Route Matching and Params

- [x] Replace exact-path-only registration with declarative route patterns.
	The router now supports parameter routes such as `/users/:id`, prefix wildcard routes, and catch-all routes through the public registration API.
- [x] Expose typed route params to components.
	Components can now use `router.UseParams()` together with typed helpers such as `Int` and `Bool` instead of parsing `location` manually.
- [x] Add query-string helpers.
	The router now exposes `UseSearchParams` for reading, updating, deleting, replacing, and serializing query params while preserving the active route path.
- [x] Define path normalization and trailing-slash rules.
	The router docs now spell out normalization rules for blank paths, leading slashes, trailing slashes, query handling, decoded params, and unsupported optional segments across hash and history routing.
- [x] Add tests for edge cases.
	Router tests now cover decoded params, invalid and empty param segments, unsupported optional-segment syntax, and exact/catch-all precedence rules.

### Navigation Guards and Redirects

- [x] Turn route options into a real feature surface.
	`router.Options` now supports route titles, declarative redirects, route loaders, and route-scoped loading/error renderers.
- [x] Add before-enter and before-leave hooks.
	The router now supports synchronous `BeforeEnter` and `BeforeLeave` guards for auth checks, permission gates, and unsaved-change blocking on router-driven navigation.
- [x] Add redirect helpers.
	The router now supports declarative redirects through `router.Options{Redirect: ...}`, while programmatic replacement continues to use `UseNavigate().Replace(...)`.
- [ ] Define async guard behavior.
	Synchronous guard hooks now exist; pending auth checks and other async route validation still need an explicit model.
- [x] Add deterministic tests for blocked and redirected navigation.
	The router test suite now covers blocked and redirected guard behavior for both hash and history routers.

### Nested Routes and Layout Routes

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

### Error Boundaries

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

### Async UI Primitives

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

## Priority 4: Deepest Architecture Work

### SSR and Hydration


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

## Priority 5: Long-Term Strategic Work

### Compiler-Assisted Features

- [ ] Decide whether compiler-driven ergonomics are a real product direction.
	This includes any Svelte-like or compile-time optimization path rather than only runtime improvements.
- [ ] Clarify the role of the browser compiler example.
	It currently demonstrates tooling ideas, not a production-ready framework compiler.
- [ ] Evaluate compile-time transforms only after the public runtime model stabilizes.
	Avoid introducing a second programming model before the first one is complete.

### Ecosystem Extension Points

- [ ] Evaluate whether the framework needs a plugin or directive model.
	This could cover animation helpers, head management, data integration, and router extensions.
- [ ] Define extension boundaries before adding framework-specific utilities ad hoc.
	Avoid scattering experimental features across unrelated packages.
- [ ] Identify which ecosystem problems belong in core versus companion packages.
	Keep the base framework small while still enabling higher-level libraries.
