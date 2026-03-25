# GoWebComponents TODO

This backlog tracks missing, incomplete, or experimental framework capabilities compared to mature UI frameworks such as React, Svelte, Vue, Solid, Blazor, and Qwik.

## At A Glance

- This file is the active framework backlog, not a changelog and not end-user documentation.
- Open work stays near the top so current priorities are visible without digging through completed history.
- The backlog mixes product-surface gaps, documentation gaps, runtime follow-up, and ecosystem maturity work because all of them affect framework readiness.
- Completed items should remain concise proof of shipped direction, not become the dominant content of the file.

## How To Read This Backlog

Use this file when you need to answer one of these questions quickly:

- what important framework gaps are still open
- which maturity areas are active versus merely historical
- whether a topic already has backlog coverage before adding a duplicate item
- how current docs, examples, runtime work, and ecosystem work connect to the same roadmap

Read the top unchecked items in a section as the real current pressure. Treat completed items as evidence of direction, not as a substitute for the source docs that describe shipped behavior.

## Backlog Rules

- prefer one action-oriented item per real gap instead of many near-duplicates
- keep scope notes concrete enough that a maintainer can tell what done means
- move stale or superseded items into clearer merged entries instead of letting the file fragment
- keep this file focused on framework work; repo-local chores that do not affect framework readiness belong elsewhere

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
- [x] Publish an API stability and support policy.
	The docs now define stability tiers for public packages, experimental features, companion APIs, internal details, semver expectations, the deprecation lifecycle, and latest-major support policy in `docs/API_POLICY.md`.
- [x] Add migration notes as new primitives land.
	The docs now include `docs/MIGRATIONS.md` as the release-to-release upgrade index, starting with the transition into the current `v3.x` public package layout.
- [x] Keep release-to-release migration guides current for future major changes.
	Each future major release still needs subsystem-specific upgrade guidance for runtime, router, SSR, forms, state, and deployment changes before the release is considered ready for enterprise adoption.

### Documentation discoverability and task-oriented guidance

- [ ] Publish an official state-management architecture guide.
	Show the recommended split between local hook state, shared atoms, derived state, context, snapshots, persistence, and fine-grained selectors so non-trivial apps do not invent incompatible state layers from scratch.
- [ ] Publish an official data-loading and mutation architecture guide.
	Document when to use route loaders, `fetch.UseResource[T](...)`, shared cached resources, server-backed forms, optimistic updates, offline replay, and revalidation so the async story has one golden path for serious applications.
- [ ] Publish a first-class auth and session integration guide.
	Explain the supported patterns for cookies, bearer tokens, server-hydrated auth hints, role and claim checks, logout handling, and same-origin API calls across client-rendered and SSR-enabled apps.
- [ ] Publish business-app form workflow recipes.
	Turn the existing form primitives into task-oriented guidance for validation, pending UX, field-error projection, optimistic submit flows, redirects, uploads, and server-owned mutation handling in real internal apps.

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
- [ ] Keep the explicit shipped-example checklist aligned with the real catalog.
	Update the backlog checklist and surrounding summary whenever new numbered examples or integrated reference apps land so the claim that shipped examples are tracked explicitly remains true through examples such as `77` through `102`, the secure-forms reference, and later catalog additions.
- [x] Track example page shell parity against the landing page theme.
	The example entry pages are now covered as a single page-shell pass across the integrated demos (`01-counter` through `20-portals`), the feature-isolated catalog pages (`21-ui-render` through `76-use-effect`), the server-routing entry page (`18-ssr-server-routing/index.html`), and the static companion pages under `examples/static/`.
- [x] Track all shipped examples in the backlog explicitly.
	Coverage is now tracked across every shipped example so catalog parity work, docs review, and logging passes can be checked against the full set instead of an arbitrary subset.
- [x] Rewrite shared example copy into Overview / Functional / Implementation sections.
	The feature-isolated catalog pages now use one consistent structure: overview for the high-level purpose, functional guidance for how and why to use the tool, and implementation guidance for the technical shape, hazards, and optimal usage patterns.
- [x] Add Playwright interaction coverage for every shipped example entrypoint.
	The examples suite now includes a dev-server interaction sweep that visits every numbered example under `/examples/...`, performs safe browser interactions, and pairs with the dedicated SSR server-routing spec for the real HTTP example.

#### Example Coverage Checklist

- [x] `01-counter`: Counter
- [x] `02-text-input`: Text Input
- [x] `03-toggle`: Toggle
- [x] `04-form`: Form
- [x] `05-todo-basic`: Todo Basic
- [x] `06-todo-advanced`: Todo Advanced
- [x] `07-goroutines`: Goroutines
- [x] `08-fetch`: Fetch
- [x] `09-atoms`: Atoms
- [x] `10-advanced-form`: Advanced Form
- [x] `11-blog`: Blog Landing
- [x] `12-portfolio-site`: Portfolio Site
- [x] `13-browser-compiler`: Browser Compiler
- [x] `14-omi`: OMI Demo
- [x] `15-calculator`: Calculator
- [x] `16-devtools`: Devtools Showcase
- [x] `17-ssr-routing`: SSR Routing
- [x] `18-ssr-server-routing`: SSR Server Routing
- [x] `19-nested-routes`: Nested Routes
- [x] `20-portals`: Portals
- [x] `21-ui-render`: `ui.Render`
- [x] `22-create-element`: `ui.CreateElement`
- [x] `23-fragment`: `ui.Fragment`
- [x] `24-use-ref`: `ui.UseRef`
- [x] `25-use-previous`: `ui.UsePrevious`
- [x] `26-use-deferred-value`: `ui.UseDeferredValue`
- [x] `27-transition-hooks`: `ui.StartTransition`, `ui.UseTransition`
- [x] `28-use-reducer`: `ui.UseReducer`
- [x] `29-use-debounced`: `ui.UseDebounced`
- [x] `30-use-throttled`: `ui.UseThrottled`
- [x] `31-context-api`: `ui.CreateContext`, `ui.UseContext`
- [x] `32-async-boundary`: `ui.AsyncBoundary`
- [x] `33-lazy`: `ui.Lazy`
- [x] `34-error-boundary`: `ui.ErrorBoundary`
- [x] `35-use-id`: `ui.UseId`
- [x] `36-typed-events`: typed events, `ui.UseEvent`
- [x] `37-use-atom`: `state.UseAtom`
- [x] `38-use-computed`: `state.UseComputed`
- [x] `39-use-derived`: `state.UseDerived`
- [x] `40-snapshot-export-import`: snapshot export and import
- [x] `41-snapshot-storage`: snapshot storage
- [x] `42-use-fetch`: `fetch.UseFetch`
- [x] `43-use-resource`: `fetch.UseResource`
- [x] `44-use-cached-resource`: `fetch.UseCachedResource`
- [x] `45-fetch-imperative`: imperative `fetch.Fetch`
- [x] `46-raw-handler`: `ui.RawHandler`
- [x] `47-portal-selector`: selector `ui.Portal`
- [x] `48-portal-target`: `ui.PortalTarget`
- [x] `49-use-channel`: `ui.UseChannel`
- [x] `50-use-task`: `ui.UseTask`
- [x] `51-use-form`: `ui.UseForm`
- [x] `52-semantic-html`: semantic `html` helpers
- [x] `53-html-forms`: typed form controls
- [x] `54-html-tag`: `html.Tag`
- [x] `55-hash-router`: hash router basics
- [x] `56-browser-router`: browser router basics
- [x] `57-use-navigate`: `router.UseNavigate`
- [x] `58-route-params`: route params
- [x] `59-route-query`: route query helpers
- [x] `60-route-loaders`: route loaders
- [x] `61-use-revalidator`: `router.UseRevalidator`
- [x] `62-router-redirects`: router redirects
- [x] `63-router-metadata`: router metadata
- [x] `64-nested-layout-routes`: nested layout routes
- [x] `65-router-guards`: router guards
- [x] `66-devtools-panel`: `devtools.Panel`
- [x] `67-use-snapshot`: `devtools.UseSnapshot`
- [x] `68-snapshot-now`: `devtools.SnapshotNow`
- [x] `69-devtools-diagnostics`: devtools diagnostics
- [x] `70-render-to-string`: `ui.RenderToString`
- [x] `71-hydrate`: `ui.Hydrate`
- [x] `72-router-hydrate-mount`: `router.HydrateMount`
- [x] `73-ssr-bootstrap`: SSR bootstrap helpers
- [x] `74-ssr-route-data-reuse`: SSR route-data reuse
- [x] `75-use-state`: `ui.UseState`
- [x] `76-use-effect`: `ui.UseEffect`

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
- [ ] Add a dedicated `ui.UseCallback` example.
	Teach stable callback identity, dependency-driven callback replacement, and the intended split between `UseCallback(...)`, `UseEvent(...)`, and ordinary inline handlers so callback memoization is documented as a first-class public hook instead of only appearing incidentally in larger examples.
- [ ] Add a dedicated `ui.UseLazyNode` example.
	Show deferred node resolution, fallback timing, and the intended boundary between `UseLazyNode(...)`, `ui.Lazy`, and route- or worker-driven loading so the hook has one minimal teaching surface instead of only app-shaped references.
- [ ] Make the example inventory and checklist reflect `ui.UseWorkerTask` as a first-class public hook.
	Keep `examples/91-worker-text-index` in the catalog, but also ensure the explicit inventory and backlog coverage treat `ui.UseWorkerTask[...]` as public `ui` surface rather than leaving it discoverable only through the broader `interop` section.

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

- [x] Reorganize docs around common developer tasks.
	Provide clear entry points for workflows such as building a client-only app, adding routing, adding SSR, testing a component, shipping a production wasm build, and debugging hydration issues instead of relying mostly on package-by-package reading order.
- [x] Add end-to-end walkthroughs for common app shapes.
	Create guided docs for a small SPA, a server-rendered app, a static-export app, and a data-heavy dashboard so developers can follow a realistic path rather than stitching together isolated examples.
- [x] Add cross-linked concept, API, and example references.
	Ensure each major feature page links directly to the public API, a runnable example, production caveats, and related debugging or testing guidance so discoverability improves for new users.
- [x] Add troubleshooting guides for common setup and runtime failures.
	Document likely causes and fixes for wasm build failures, missing `wasm_exec.js`, broken example serving, hydration mismatch warnings, route misconfiguration, and interop mistakes.
- [x] Add a clearly documented recommended path for new adopters.
	State which packages, examples, commands, and architecture patterns are the preferred modern route so developers are not left choosing between stale and current approaches.

### API stability and support policy

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
- [x] Track example page shell parity against the landing page theme.
	The example entry pages are now covered as a single page-shell pass across the integrated demos (`01-counter` through `20-portals`), the feature-isolated catalog pages (`21-ui-render` through `76-use-effect`), the server-routing entry page (`18-ssr-server-routing/index.html`), and the static companion pages under `examples/static/`.
- [x] Track all shipped examples in the backlog explicitly.
	Coverage is now tracked across every shipped example so catalog parity work, docs review, and logging passes can be checked against the full set instead of an arbitrary subset.
- [x] Rewrite shared example copy into Overview / Functional / Implementation sections.
	The feature-isolated catalog pages now use one consistent structure: overview for the high-level purpose, functional guidance for how and why to use the tool, and implementation guidance for the technical shape, hazards, and optimal usage patterns.
- [x] Add Playwright interaction coverage for every shipped example entrypoint.
	The examples suite now includes a dev-server interaction sweep that visits every numbered example under `/examples/...`, performs safe browser interactions, and pairs with the dedicated SSR server-routing spec for the real HTTP example.

#### Example Coverage Checklist

- [x] `01-counter`: Counter
- [x] `02-text-input`: Text Input
- [x] `03-toggle`: Toggle
- [x] `04-form`: Form
- [x] `05-todo-basic`: Todo Basic
- [x] `06-todo-advanced`: Todo Advanced
- [x] `07-goroutines`: Goroutines
- [x] `08-fetch`: Fetch
- [x] `09-atoms`: Atoms
- [x] `10-advanced-form`: Advanced Form
- [x] `11-blog`: Blog Landing
- [x] `12-portfolio-site`: Portfolio Site
- [x] `13-browser-compiler`: Browser Compiler
- [x] `14-omi`: OMI Demo
- [x] `15-calculator`: Calculator
- [x] `16-devtools`: Devtools Showcase
- [x] `17-ssr-routing`: SSR Routing
- [x] `18-ssr-server-routing`: SSR Server Routing
- [x] `19-nested-routes`: Nested Routes
- [x] `20-portals`: Portals
- [x] `21-ui-render`: `ui.Render`
- [x] `22-create-element`: `ui.CreateElement`
- [x] `23-fragment`: `ui.Fragment`
- [x] `24-use-ref`: `ui.UseRef`
- [x] `25-use-previous`: `ui.UsePrevious`
- [x] `26-use-deferred-value`: `ui.UseDeferredValue`
- [x] `27-transition-hooks`: `ui.StartTransition`, `ui.UseTransition`
- [x] `28-use-reducer`: `ui.UseReducer`
- [x] `29-use-debounced`: `ui.UseDebounced`
- [x] `30-use-throttled`: `ui.UseThrottled`
- [x] `31-context-api`: `ui.CreateContext`, `ui.UseContext`
- [x] `32-async-boundary`: `ui.AsyncBoundary`
- [x] `33-lazy`: `ui.Lazy`
- [x] `34-error-boundary`: `ui.ErrorBoundary`
- [x] `35-use-id`: `ui.UseId`
- [x] `36-typed-events`: typed events, `ui.UseEvent`
- [x] `37-use-atom`: `state.UseAtom`
- [x] `38-use-computed`: `state.UseComputed`
- [x] `39-use-derived`: `state.UseDerived`
- [x] `40-snapshot-export-import`: snapshot export and import
- [x] `41-snapshot-storage`: snapshot storage
- [x] `42-use-fetch`: `fetch.UseFetch`
- [x] `43-use-resource`: `fetch.UseResource`
- [x] `44-use-cached-resource`: `fetch.UseCachedResource`
- [x] `45-fetch-imperative`: imperative `fetch.Fetch`
- [x] `46-raw-handler`: `ui.RawHandler`
- [x] `47-portal-selector`: selector `ui.Portal`
- [x] `48-portal-target`: `ui.PortalTarget`
- [x] `49-use-channel`: `ui.UseChannel`
- [x] `50-use-task`: `ui.UseTask`
- [x] `51-use-form`: `ui.UseForm`
- [x] `52-semantic-html`: semantic `html` helpers
- [x] `53-html-forms`: typed form controls
- [x] `54-html-tag`: `html.Tag`
- [x] `55-hash-router`: hash router basics
- [x] `56-browser-router`: browser router basics
- [x] `57-use-navigate`: `router.UseNavigate`
- [x] `58-route-params`: route params
- [x] `59-route-query`: route query helpers
- [x] `60-route-loaders`: route loaders
- [x] `61-use-revalidator`: `router.UseRevalidator`
- [x] `62-router-redirects`: router redirects
- [x] `63-router-metadata`: router metadata
- [x] `64-nested-layout-routes`: nested layout routes
- [x] `65-router-guards`: router guards
- [x] `66-devtools-panel`: `devtools.Panel`
- [x] `67-use-snapshot`: `devtools.UseSnapshot`
- [x] `68-snapshot-now`: `devtools.SnapshotNow`
- [x] `69-devtools-diagnostics`: devtools diagnostics
- [x] `70-render-to-string`: `ui.RenderToString`
- [x] `71-hydrate`: `ui.Hydrate`
- [x] `72-router-hydrate-mount`: `router.HydrateMount`
- [x] `73-ssr-bootstrap`: SSR bootstrap helpers
- [x] `74-ssr-route-data-reuse`: SSR route-data reuse
- [x] `75-use-state`: `ui.UseState`
- [x] `76-use-effect`: `ui.UseEffect`

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

- [x] Reorganize docs around common developer tasks.
	Provide clear entry points for workflows such as building a client-only app, adding routing, adding SSR, testing a component, shipping a production wasm build, and debugging hydration issues instead of relying mostly on package-by-package reading order.
- [x] Add end-to-end walkthroughs for common app shapes.
	Create guided docs for a small SPA, a server-rendered app, a static-export app, and a data-heavy dashboard so developers can follow a realistic path rather than stitching together isolated examples.
- [x] Add cross-linked concept, API, and example references.
	Ensure each major feature page links directly to the public API, a runnable example, production caveats, and related debugging or testing guidance so discoverability improves for new users.
- [x] Add troubleshooting guides for common setup and runtime failures.
	Document likely causes and fixes for wasm build failures, missing `wasm_exec.js`, broken example serving, hydration mismatch warnings, route misconfiguration, and interop mistakes.
- [x] Add a clearly documented recommended path for new adopters.
	State which packages, examples, commands, and architecture patterns are the preferred modern route so developers are not left choosing between stale and current approaches.

### API stability and support policy

- [x] Define stability tiers for all major framework surfaces.
	`docs/API_POLICY.md` now classifies stable public packages, supported companion APIs, experimental surfaces, deprecated compatibility surface expectations, and unsupported internal implementation details.
- [x] Publish a semver and compatibility policy.
	`docs/API_POLICY.md` now defines what is allowed in patch, minor, and major releases and documents what this project counts as breaking behavior for consumers.
- [x] Define a deprecation lifecycle and migration window.
	`docs/API_POLICY.md` now requires a replacement first, changelog and migration-guide notice, and a minimum support window of two minor releases and 90 days before stable API removal.
- [x] Add upgrade and migration guides for major framework changes.
	`docs/MIGRATIONS.md` now provides the migration index and first project-level guide, covering runtime, router, SSR or hydration, forms, state, fetch, testing, and deployment changes for the current public package layout.
- [x] Define long-term support expectations for enterprise adopters.
	`docs/API_POLICY.md` now states the current support posture explicitly: latest-major support only, no LTS line promised yet, and no documented backport policy for older majors.
- [ ] Decide whether to offer an LTS or backport support line for enterprise adopters.
	Make the long-term support posture explicit by defining whether older majors ever receive security fixes, whether starter templates track only latest-major, and what support window procurement reviewers should assume.
- [ ] Define graduation criteria from experimental to supported or stable surfaces.
	Specify the documentation, testing, compatibility, and migration requirements needed before features such as hot reload, multi-client coordination, companion packages, or compiler-assisted experiments stop being labeled experimental.

### Metadata and composition model

- [x] Finalize the SSR metadata model.
	Route title, description, and canonical metadata are now documented as jointly reconciled: the server emits the initial router-managed tags, hydration preserves them, and subsequent client navigation updates the same managed surface.
- [x] Add a server render path for route-managed metadata.
	`router.MetadataNode(...)` now renders SSR-safe head nodes that can be emitted through `ui.RenderToString(...)`, and the server-routing example now uses that path instead of hand-built metadata strings.
- [x] Define metadata hydration reconciliation rules.
	Client route metadata reconciliation now updates and removes only `data-gwc-router-managed="true"` tags, dedupes managed tags, and clears stale SSR-managed title state when a later route omits metadata.
- [x] Decide whether slots are part of the public composition model.
	Slots are now explicitly out of scope for the current public API. The documented composition model is ordinary children, explicit props, context, portals, and layout routes.

### Head management and SEO surface

- [x] Define the first-class head management model.
	`docs/HEAD_MANAGEMENT.md` now defines the shipped ownership split: router-managed title, description, and canonical metadata are first-class, while broader SEO tags remain application-owned explicit head markup until a dedicated head manager exists.
- [x] Add server-rendered head emission for route-driven apps.
	`docs/HEAD_MANAGEMENT.md` and the router metadata tests now show the SSR path: compose `router.MetadataNode(...)` with explicit head tags under `ui.RenderToString(...)` so route-driven apps can emit managed metadata plus robots, social tags, and resource hints on first paint.
- [x] Define hydration reconciliation rules for head state.
	`docs/HEAD_MANAGEMENT.md` now defines managed-versus-unmanaged head ownership during hydration, and the router tests verify managed tag deduplication and cleanup across hydrated documents and later navigation.
- [x] Add route-level title and metadata composition rules.
	`docs/HEAD_MANAGEMENT.md` now documents the current precedence contract: leaf routes override layouts for managed metadata, layouts provide defaults, and no title-template or deep merge API is implied yet.
- [x] Add canonical URL and duplicate-content guidance.
	`docs/HEAD_MANAGEMENT.md` now covers parameterized routes, filters, locale variants, pagination, previews, and matching canonical behavior across prerendered and request-time SSR delivery.
- [x] Add structured-data support guidance.
	`docs/HEAD_MANAGEMENT.md` now documents JSON-LD as an explicit SSR escape hatch that should be emitted from the server document template until a dedicated raw-script helper exists.
- [x] Add preload, preconnect, and resource-hint management.
	`docs/HEAD_MANAGEMENT.md` now documents resource hints as application-owned SSR markup and explains how to emit and dedupe them intentionally without implying router-managed reconciliation.
- [x] Add social-sharing metadata examples.
	`docs/HEAD_MANAGEMENT.md` now includes a concrete Open Graph and Twitter/X SSR example that composes with `router.MetadataNode(...)`.
- [x] Add sitemap, robots, and crawl-control integration guidance.
	`docs/HEAD_MANAGEMENT.md` now ties canonical URLs, robots directives, sitemap inclusion, preview environments, and authenticated routes back to one route-level source of truth.
- [x] Add tests for head correctness across SSR, hydration, and navigation.
	The router metadata tests now cover explicit SSR head composition plus managed-tag deduplication after hydrated startup, and the SSR server-routing tests verify the initial HTML emits exactly one managed title, description, and canonical tag set.
- [ ] Add a higher-level head manager surface for non-router metadata.
	Provide a supported first-party or companion-owned way to manage Open Graph, Twitter/X, JSON-LD, robots, hreflang, and resource hints without forcing every SSR app to hand-compose head nodes in its document template.
- [ ] Add route-to-head composition helpers for larger apps.
	Support layout defaults, leaf overrides, and predictable merge or removal rules for metadata that sits beyond title, description, and canonical URL so multi-route apps do not accumulate ad hoc head wiring.
- [ ] Add end-to-end head ownership examples across SSR, prerender, and hydrated navigation.
	Demonstrate one metadata-heavy app flow that covers initial SSR, later client navigation, social tags, structured data, and resource hints so the head story reaches H3-level integrated maturity rather than staying mostly policy and low-level helpers.

### Accessibility primitives and guidance

- [x] Define the accessibility support baseline for the public UI surface.
	`docs/ACCESSIBILITY.md` now defines the shipped baseline around typed semantic HTML, ARIA and role props, `ui.UseId()`, and the new accessibility primitives layered on top of that markup surface.
- [x] Add a first-class focus-management toolkit.
	`ui.UseFocusManager()` and `ui.UseFocusTrap(...)` now cover focus restoration, focus-to-error helpers, and modal-style focus trapping, with example coverage in `examples/77-accessible-overlay` and `examples/79-form-accessibility`.
- [x] Add keyboard-navigation primitives for composite widgets.
	`ui.UseCompositeNavigation(...)` now provides roving tabindex, arrow-key movement, Home/End handling, typeahead, and active-descendant support, with coverage in `examples/78-composite-navigation` and `ui/ui_wasm_test.go`.
- [x] Add live-region and announcement helpers.
	`ui.UseAnnouncer()` now provides polite and assertive live-region output for validation, async status, and route updates, with coverage in `examples/79-form-accessibility`, `examples/80-routed-accessibility`, and `ui/ui_wasm_test.go`.
- [x] Define accessible overlay primitives.
	`ui.AccessibleOverlay(...)` now defines the modal overlay contract for portal-backed dialogs, including focus trapping, escape handling, background inerting, scroll locking, and aria wiring.
- [x] Add accessibility guidance for forms, routed apps, and async UI.
	`docs/ACCESSIBILITY.md` now documents label/input pairing, `aria-*` usage, route announcements, async pending-state semantics, and the shipped accessibility helper APIs used to implement those flows.
- [x] Add accessibility-focused examples and tests.
	Added `examples/77-accessible-overlay`, `examples/78-composite-navigation`, `examples/79-form-accessibility`, and `examples/80-routed-accessibility` together with focused Playwright specs and package-level `ui` hook tests.
- [ ] Add accessibility audit workflow guidance for teams and CI.
	Document a repeatable accessibility review loop covering keyboard flows, live-region announcements, route changes, forms, overlays, and automated checks so enterprise teams can keep a11y quality from drifting after the first pass.
- [ ] Add accessibility regression recipes for routed and form-heavy apps.
	Show how application tests should assert focus movement, announcement timing, validation feedback, modal trapping, and route-update semantics across wasm tests and Playwright runs.

### Portal layering and overlay management

- [x] Define a first-class overlay and portal layering model.
	Document how modals, popovers, tooltips, dropdowns, sheets, and nested portals participate in shared stacking order instead of leaving z-index policy to ad hoc application code.
- [x] Add a centralized overlay manager primitive.
	Provide a framework-level way to register active overlays, assign stack order, and coordinate mount or unmount behavior for nested and sibling portal trees.
- [x] Define escape-key and dismissal routing for nested overlays.
	Specify which overlay handles escape first, how outside-click dismissal behaves across stacked layers, and how parent overlays remain stable when child overlays close.
- [x] Add scroll-lock and background-inert behavior.
	Support consistent body scroll locking, nested overlay lock counting, and background interaction suppression so portal-heavy apps do not reimplement these details for every dialog flow.
- [x] Define focus and accessibility coordination for layered overlays.
	Ensure the overlay manager composes correctly with focus trapping, restoration, announcement semantics, and aria relationships when several portal-driven surfaces are open at once.
- [x] Add positioning and anchor coordination guidance for floating overlays.
	Document how tooltips, anchored popovers, and context menus should manage viewport collision, resize or scroll repositioning, and nested stacking without conflicting portal ownership.
- [x] Add examples and tests for complex overlay stacks.
	Demonstrate nested dialogs, dialog-plus-popover, tooltip-over-menu, and portal retargeting scenarios so layering behavior is enforced by real browser coverage.

### Internationalization and localization

- [x] Define the first-class i18n scope for the framework.
	Decide whether the framework should own only message lookup and locale context, or also pluralization, formatting helpers, locale-aware routing, and SSR locale transfer.
- [x] Add a locale context and switching model.
	Provide a stable way to expose the active locale to component trees, update it at runtime, and coordinate locale changes with rerendering, route changes, and persisted user preference.
- [x] Add message catalog loading and lookup primitives.
	Support organizing translated messages by locale and namespace, loading them deterministically, and resolving missing-message fallback behavior without every app inventing its own structure.
- [x] Add message formatting and pluralization helpers.
	Support interpolated messages, plural rules, select-style branching, and locale-aware number or date formatting so application text does not rely on ad hoc string concatenation.
- [x] Define SSR and hydration behavior for locale data.
	Clarify how active locale, selected messages, and formatting configuration are transferred from server to client so SSR output and hydrated UI stay consistent.
- [x] Add locale-aware routing and content-loading guidance.
	Document whether locale prefixes, locale domains, or route metadata should be handled by the router, application code, or a companion package, and how loaders select locale-specific content.
- [x] Add RTL and directionality support guidance.
	Specify how locale changes affect document direction, component-level `dir` overrides, layout assumptions, and mixed-direction content in real applications.
- [x] Add i18n-focused examples and tests.
	Create examples for locale switching, pluralized UI, date or number formatting, SSR locale bootstrapping, and locale-prefixed routing so the public i18n story is validated end to end.

## 2. Forms, Uploads, and Secure Submission Workflows

### Server-backed forms

- [x] Define the supported form modes.
	Document the recommended split between client-only forms, progressive-enhancement posts, JSON-backed submissions, and SSR-backed form flows.
- [x] Add form-post destination conventions for Go handlers.
	Show how `ui.UseForm` should target `net/http` handlers, JSON endpoints, and multipart upload routes so server-backed forms are not all bespoke.
- [x] Define redirect-after-submit semantics.
	Specify how successful submissions coordinate with router navigation, flash-style success state, and history replacement.
- [ ] Add a first-class server action contract for form submissions.
	Define one typed mutation model that lets hydrated UI and progressive forms target the same server-owned action semantics for success, redirect, validation failure, auth failure, and recoverable retry without hand-rolled per-endpoint conventions.
- [ ] Add typed server-action result envelopes and mapping helpers.
	Standardize how server-owned form handlers return field errors, form errors, redirects, flash-style success metadata, and post-submit refresh instructions so applications do not keep reinventing transport-specific mutation response contracts.
- [ ] Add integrated examples for progressive plus hydrated server actions.
	Ship at least one example where the same form works as a normal HTML post without JavaScript, upgrades into richer hydrated pending/error UX, and still preserves one authoritative server mutation contract.
- [ ] Define a first-class server function model beyond forms.
	Specify whether GWC should support typed server-owned actions for non-form mutations and queries, how those calls are declared from Go code, and how they differ from plain fetch helpers or transport-specific RPC clients.
- [ ] Define how server functions compose with loaders, revalidation, auth, and SSR.
	Clarify how a server-owned action can invalidate route data, refresh shared caches, honor auth and CSRF boundaries, participate in SSR-aware request context, and degrade when JavaScript or hydration is unavailable.

### CSRF and secure posting

- [x] Add CSRF-aware form helpers for server-post workflows.
	Support token injection and transport conventions for form posts targeting Go HTTP handlers without forcing every app to hand-roll hidden fields and headers.
- [x] Define CSRF token source and refresh rules.
	Clarify whether tokens come from SSR bootstrap, cookies, headers, or explicit server endpoints and how long-lived pages refresh them safely.

### Validation and error handling

- [x] Add server-returned field error mapping.
	Provide a normalized way to take structured validation errors from server responses and project them back onto public form state.
- [x] Add field-level message helpers.
	Expose touched, dirty, pending, and error helpers so inline validation and summary rendering do not require repetitive app code.
- [x] Add submit-intent helpers.
	Support workflows such as draft save vs publish, per-button pending state, and submit-intent-specific validation without ad hoc local state.
- [x] Define optimistic vs authoritative submit behavior.
	Clarify when forms may update UI optimistically, when they must wait for the server, and how retry/reset flows behave after partial failure.

### Uploads and SSR examples

- [x] Add multipart and file-upload support.
	Cover `multipart/form-data`, file inputs, upload progress, cancellation, and server error reporting as first-class workflows.
- [x] Add SSR-friendly secure form examples.
	Demonstrate form defaults, server validation round-trips, CSRF-aware submission, uploads, and post-submit redirects in a request-time rendered example.

## 3. JavaScript Interop Ergonomics

### Public interop API

- [x] Design a first-class public JS interop package or namespace.
	Move common `syscall/js` patterns behind a stable public API so applications do not depend on raw low-level browser bindings for routine work.
- [x] Add typed wrappers for common browser APIs.
	Cover storage, history, location, clipboard, timers, custom events, media queries, and similar APIs with predictable Go-friendly shapes.
- [x] Add module-style interop helpers.
	Support importing a JS module, calling exported functions, and disposing module handles with a lifecycle model that fits component mount and unmount behavior.

### DOM and event integration

- [x] Add event subscription helpers around browser APIs.
	The `interop` package now exposes generic `Listen(...)` support for window, document, and element event targets, keeps media-query listeners on the same cleanup model, and adds resize plus intersection observer helpers that cancel through `Subscription`.
- [x] Add element-reference based interop helpers.
	The public `interop.Element` and `CurrentDocument()` surface now covers DOM lookup, focus/blur/click, scroll-into-view, bounding-rect measurement, and element-scoped listeners/observers without forcing app code back into raw `syscall/js`.
- [x] Add third-party library integration examples.
	`examples/88-web-components` now demonstrates consuming a browser-defined custom element from GoWebComponents, synchronizing reflected attributes and property-only config from Go state, and cleaning up the typed custom-event subscription through effect cleanup.

### Typed RPC and streaming transport

- [ ] Define whether typed browser RPC belongs in core interop or a supported companion package.
	Make an explicit product decision that protobuf-backed RPC for browser clients is transport substrate rather than base rendering contract unless the implementation proves broad enough to justify first-class framework status.
	External implementation reference when this work starts: `grpc-tunnel` repo as the candidate unary plus streaming transport substrate for Go/WASM browser clients.
- [ ] Define the browser transport contract for gRPC-style RPC over WebSocket.
	Specify connection ownership, handshake behavior, unary versus server-streaming versus client-streaming versus bidirectional-streaming semantics, browser reconnect expectations, and how transport closure surfaces to application code.
	External implementation reference when this work starts: `grpc-tunnel` repo.
- [ ] Define protobuf code-generation and client-binding workflow for Go/WASM apps.
	Document how service definitions, generated Go types, browser-safe client stubs, and versioned protobuf contracts enter a normal GWC application build without forcing ad hoc shell scripts or hidden codegen steps.
	External implementation reference when this work starts: `grpc-tunnel` repo.
- [ ] Define auth, session, and metadata propagation for tunneled RPC calls.
	Clarify how cookies, bearer tokens, CSRF-adjacent mutation protection, request metadata, tenant context, and per-stream identity are attached, refreshed, and redacted in browser-to-server RPC flows.
	External implementation reference when this work starts: `grpc-tunnel` repo.
- [ ] Define cancellation, timeout, and backpressure semantics for tunneled unary and streaming RPC.
	Support browser-side cancellation, deadline propagation, bounded outbound buffering, stream-level flow control expectations, and clear handling for late messages after a consumer unsubscribes or a route unmounts.
	External implementation reference when this work starts: `grpc-tunnel` repo.
- [ ] Define SSR and hydration boundaries for browser-only RPC clients.
	Document that live tunneled RPC is a hydrated-browser concern, not an SSR transport, and specify how route loaders, bootstrap payloads, and later RPC attachment compose without duplicate authority or inconsistent initial data.
	External implementation reference when this work starts: `grpc-tunnel` repo.
- [ ] Add observability, diagnostics, and actionable errors for typed RPC transport.
	Surface connection state, handshake failure, auth rejection, stream restart, schema mismatch, backpressure, and decode errors through the same structured logging, diagnostics, and devtools model used by router, fetch, and multi-client coordination.
	External implementation reference when this work starts: `grpc-tunnel` repo.
- [ ] Add one production-shaped RPC example that proves unary plus streaming value.
	Use a live dashboard, presence surface, collaborative queue, or operator console to demonstrate protobuf contracts, browser connection lifecycle, bidi or server streaming, and UI integration without inventing a second bespoke protocol for the example.
	External implementation reference when this work starts: `grpc-tunnel` repo.

### Custom element and web component interop

- [x] Define the supported custom-element interop model.
	`docs/CUSTOM_ELEMENTS.md` now defines the supported model as consuming browser-defined custom elements from GoWebComponents trees today, while leaving export-as-custom-element work explicitly out of scope for the current public API.
- [x] Add first-class custom-element consumption helpers.
	`html.CustomElement(...)`, `html.CustomElementProps`, and `html.Props{Slot: ...}` now provide explicit host rendering for browser-defined custom elements, while `interop.CurrentDocument()` and `interop.Element` remain the supported imperative handle path after mount.
- [x] Define prop-versus-attribute mapping for custom elements.
	`docs/CUSTOM_ELEMENTS.md` now defines `Attributes`, `Presence`, and `Properties` as separate channels so reflected strings, presence booleans, and client-only property payloads behave predictably across SSR and client render paths.
- [x] Add custom-event bridging for web components.
	`interop.DecodeCustomEvent[T](...)` and `interop.SubscribeDecoded[T](...)` now support typed `CustomEvent` detail projection, and the custom-element guidance documents the expected JSON-shaped detail boundary.
- [x] Add SSR and hydration rules for custom-element hosts.
	`docs/CUSTOM_ELEMENTS.md` now documents that SSR renders inert host tags with reflected attributes only, while client-only properties are applied after mount or hydration and should not be treated as serialized server state.
- [x] Explore exporting GoWebComponents components as standards-based custom elements.
	`examples/89-exported-custom-element` now prototypes the current export-side pattern: a browser custom-element wrapper owns the lifecycle and shadow-root target, while Go mounts and unmounts the subtree through `ui.RenderInto(...)`.
- [x] Define style and shadow-DOM expectations for exported custom elements.
	`docs/CUSTOM_ELEMENTS.md` now defines the current expectation: exported prototypes default to wrapper-owned shadow DOM for style isolation, while light-DOM export remains possible but explicitly caller-owned in terms of CSS and collision risk.
- [x] Add interoperability examples against third-party web components.
	`examples/88-web-components` now covers consumption of a browser-defined custom-element surface from Go code, and `examples/89-exported-custom-element` covers the separate export-side prototype where plain HTML hosts consume a Go-rendered shadow-root widget.

### Safety, performance, and diagnostics

- [x] Define interop lifetime and cleanup rules.
	`docs/INTEROP.md` now defines ownership, cleanup, listener cancellation, module disposal, and DOM-handle reacquisition rules so apps have one documented lifecycle model for browser interop.
- [x] Add SSR-safe interop guardrails.
	The `interop` constructors and zero-value wrappers now have documented `unavailable` behavior for non-`js/wasm` builds, and the package guidance calls out the expected server-versus-client branching model explicitly.
- [x] Add structured error handling for interop failures.
	The public `interop.Error` surface now includes stable inspection helpers through `AsError(...)`, `CodeOf(...)`, and `IsCode(...)` so apps can branch on unavailable APIs, rejected promises, serialization failures, or disposed handles without parsing strings.
- [x] Add interop performance guidance and batching helpers.
	`interop.Storage.GetMany(...)`, `interop.Document.ElementsByID(...)`, `interop.SubscribeDecoded[T](...)`, and the updated `docs/INTEROP.md` guidance now give public grouped-read and typed-event paths for common repeated boundary-crossing scenarios.
- [x] Add interop-safe serialization guidance.
	`docs/INTEROP.md` now documents the supported JSON-shaped boundary, when to use `Decode(...)`, and which values should remain in dedicated browser wrapper types instead of trying to serialize host objects.
- [x] Add high-level examples that replace raw `syscall/js` usage.
	`examples/90-browser-interop` now demonstrates storage, clipboard access, resize and media observation, grouped DOM lookup, and typed custom-event handling entirely through the public `interop` package.

### Worker and background-thread integration

- [x] Define the public worker integration model.
	`docs/WORKERS.md` now defines the supported first-party scope as dedicated browser Web Workers under `interop`, explicitly leaving `SharedWorker`, service-worker workflows, and broader background coordination for later slices.
- [x] Add first-class worker lifecycle helpers.
	`interop.NewWorker(...)`, `worker.Terminate()`, and `worker.Restart(...)` now provide browser-worker creation, optional ready-handshake waiting, termination, and restart through the public interop layer.
- [x] Add typed message-channel helpers for workers.
	`worker.Request(...)`, `interop.RequestWorkerDecoded[...]`, `worker.Subscribe(...)`, `interop.SubscribeDecodedWorker[T](...)`, and the documented `id`/`phase`/`name`/`payload` envelope now cover typed request, progress, result, and remote-error flows.
- [x] Define cancellation, timeout, and backpressure behavior for worker jobs.
	`docs/WORKERS.md` now defines context-driven cancellation and timeouts, the rule that late responses are ignored after listener cleanup, and the current expectation that apps enforce request throttling or queueing until first-class backpressure helpers land.
- [x] Add helpers for connecting workers to components and async state.
	`ui.UseWorkerTask[...]` now binds typed worker requests into a hook-shaped task handle with progress, cancellation, worker reuse, and automatic cleanup so browser CPU work composes with normal component state.
- [x] Define serialization boundaries for worker payloads.
	`docs/WORKERS.md` now defines the current clone-friendly, JSON-shaped worker payload boundary, explicitly calls out unsupported host objects and functions, and notes that transferable-object ergonomics are not yet first-class in the Go API.
- [x] Add examples for CPU-heavy background work.
	`examples/91-worker-text-index` now demonstrates worker-backed token counting and term indexing with progress updates and cancellation while the main UI remains responsive.

## 4. Routing, Guards, and Auth-Aware Navigation

### Async navigation guards

- [x] Design the async guard API surface.
	`docs/ROUTER_AUTH.md` now defines the planned `BeforeEnterAsync` / `BeforeLeaveAsync` route options, the shared `GuardDecision` result shape, and how async decisions differ from the existing synchronous guard contract.
- [x] Define async guard cancellation and race semantics.
	The router auth design now specifies attempt-scoped contexts, cancellation of stale guard work, redirect-as-new-attempt behavior, and the rule that late results from cancelled attempts must be ignored.
- [x] Define pending-navigation UX hooks.
	The same design doc now defines a planned `UseGuardNavigation()` state surface plus `GuardPending` semantics so routes and layouts can expose pending-auth or pending-guard UI intentionally.
- [x] Add test coverage for async guard edge cases.
		Covered double-click navigation, back and forward navigation, loader-plus-guard interactions, and cleanup when guarded components unmount mid-check with wasm browser tests in `router/browser_router_test.go`.

### Auth-aware routing primitives

- [x] Define a lightweight auth context model for routed apps.
	`docs/ROUTER_AUTH.md` now defines the planned `AuthState` / `AuthStatus` model, the intended provider and hook access pattern, and the boundary that the router consumes auth hints without becoming an identity-provider framework.
- [ ] Add router-level unauthorized and authorizing UI states.
	`router.Options` already exposes `Unauthorized`, `Authorizing`, and `GuardPending`, but the runtime and docs still effectively require manual in-page unauthorized or pending-auth UI; finish the route-level behavior and update `router/README.md` so the documented pattern matches the shipped surface.
- [x] Add route-group auth inheritance rules.
	`docs/ROUTER_AUTH.md` now defines layout-driven policy inheritance, strengthening-only child policy, and the rule that protected layout trees should remain authoritative for descendant access requirements.
- [x] Add route policy hooks above simple boolean guards.
	The router auth design now defines a planned policy-helper layer such as `RequireAuthenticated`, `RequireClaim`, and named custom policy hooks that compile down to the same guard-decision model.
- [x] Define auth state refresh and invalidation behavior.
	The router auth design now specifies versioned auth invalidation, logout and expiry handling, background refresh behavior, and when auth changes should rerun active route policy without unnecessarily rerunning public-route loaders.
- [x] Define server-hydrated auth hint transfer.
	The design now defines a minimal SSR-to-client auth bootstrap payload so first client guard evaluation can use a compact auth hint before fresh auth checks complete.
- [x] Add return-to and post-auth navigation helpers.
	`router.ReturnToParam`, `router.PreserveReturnTo(...)`, and `router.ReadReturnTo(...)` now standardize safe internal return-target preservation and post-auth redirect recovery without allowing external URL injection.
- [x] Define loader and auth-gate ordering.
	`docs/ROUTER_AUTH.md` now defines the intended order of leave guards, enter guards, auth checks, and route loaders, including the rule that protected-route loaders must not start before the target route is allowed.
- [x] Add protected-route examples and security guidance.
	`examples/92-protected-routes` now demonstrates guarded entry, deferred auth resolution, manual unauthorized fallback UI, and safe post-login redirect preservation, while `docs/ROUTER_AUTH.md` and `router/README.md` now document that client-side gating is a UX tool rather than the real security boundary.
- [ ] Add typed route-definition and reverse-routing contracts.
	Provide one supported way to define routes so links, redirects, params, and query values can be generated and validated from typed Go APIs instead of manually repeating string paths across loaders, navigation code, and tests.
- [ ] Add route-contract tooling for navigation, metadata, and generated examples.
	Decide whether typed route contracts should stay runtime-only helpers or also emit code-generated route manifests for links, metadata ownership, prerender enumeration, and starter examples so larger apps can avoid stringly-typed route drift.

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
- [x] Define cache key normalization rules.
	`docs/CACHE.md` now defines deterministic string-key rules for shared cached resources, including stable inclusion of path, query, locale, auth scope, and other payload-shaping inputs.
- [x] Add freshness, eviction, and disposal policies.
	`fetch.CacheOptions` now supports `StaleAfter`, `MaxAge`, and `DisposeAfter`, cached resources can be explicitly cleared through `Dispose()` or `DisposeResource(...)`, and `fetch.SweepCachedResources()` gives long-lived apps deterministic cleanup.
- [x] Add request deduplication across concurrent subscribers.
	`fetch.UseCachedResource` already shares one in-flight load per key, the wasm tests lock that behavior in, and `docs/CACHE.md` now documents deduplication as part of the shipped cache contract.
- [x] Define mutation and optimistic update APIs.
	`docs/CACHE.md` now documents the current optimistic-write surface through `Set`, `Update`, and `Invalidate`, plus the application-owned rollback and authoritative revalidation model around those helpers.
- [x] Add route-loader and shared-cache interoperability.
	`fetch.LoadCached[T](...)` now lets route loaders reuse the same shared cache registry as `fetch.UseCachedResource[T](...)`, with the contract documented in `docs/CACHE.md` and exercised in `examples/92-protected-routes`.
- [x] Add devtools visibility for cached resources.
	`fetch.InspectCachedResources()` now exposes key, ready or stale state, subscriber count, resume policy, and last error, and the `devtools` panel now renders that cache inspection data.
- [x] Add SSR-aware cache bootstrap and resume.
	`fetch.RestoreCacheBootstrap(ui.SSRBootstrap)` now seeds shared cached resources from `ui.SSRBootstrap.Data["fetchCache"]` before hydration so first client reads can start warm.
- [x] Define cache serialization safety.
	`docs/CACHE.md` now defines the JSON-shaped bootstrap contract, payload-size expectations, and the rule that secrets or server-only authorization context must stay out of cache bootstrap entries.
- [x] Add cache revalidation-on-resume policies.
	Bootstrap entries now support explicit `trust-once`, `stale-while-revalidate`, and `always-refetch` resume policies, with the contract documented in `docs/CACHE.md` and implemented in `fetch.RestoreCacheBootstrap(...)`.
- [x] Add realistic shared-cache examples for SSR-seeded and route-loader reuse.
	`examples/93-ssr-cache-bootstrap` now demonstrates SSR-seeded shared cache restore during hydration, and `examples/92-protected-routes` already covers route-loader reuse through `fetch.LoadCached(...)`.

### Offline mutation queueing and background sync
- [x] Define the scope of first-class offline mutation support.
- [x] Add a persistent mutation queue abstraction.
- [x] Define optimistic-update and rollback semantics for queued writes.
- [x] Add retry, deduplication, and backoff policies for queued mutations.
	Queued mutations now support `DedupKey` suppression, exponential backoff through `BaseDelay` and `MaxDelay`, deferred replay until `NextAttemptAt`, and terminal `dead` state after `MaxAttempts`, all covered by wasm tests.
- [x] Define queue serialization and security boundaries.
	`docs/OFFLINE_MUTATIONS.md` now defines the JSON-shaped storage contract, warns against persisting secrets or browser-native handles, and recommends metadata version hints for queued payloads that must survive app upgrades.
	`pwa.ServiceWorkerRegistration.RegisterSync(...)` now exposes app-owned Background Sync registration, and `97-pwa-offline-cache` demonstrates explicit scheduling with manual replay fallback when the browser does not expose one-shot Background Sync.
	The examples catalog now includes `97-pwa-offline-cache`, which demonstrates a durable queued write, explicit replay, offline fallback behavior, and focused browser coverage for the replay path.
### Cross-tab state and cache synchronization
- [x] Define the scope of first-party cross-tab synchronization.
	`docs/CROSS_TAB.md` now defines the first shipped scope as opt-in cross-tab state hints, auth or logout signals, cache invalidation, and draft updates, while explicitly leaving global auto-sync, multi-window orchestration, and service-worker coordination out of the current boundary.
- [x] Add BroadcastChannel-based synchronization helpers.
	`interop.OpenCrossTabChannel(...)` now provides named cross-tab publish and subscribe helpers backed by `BroadcastChannel` when available, with typed decode through `interop.DecodeCrossTabEnvelope[T](...)` and `interop.SubscribeDecodedCrossTab[T](...)`.
	The cross-tab channel helper now falls back to `localStorage` plus `storage` events when `BroadcastChannel` is unavailable, and the wasm tests cover both transports.
	`docs/CROSS_TAB.md` now defines transport-level ordering versus application-owned merge rules, including immediate application of narrowing auth signals, revision-aware handling for optimistic entity updates, and the preference for invalidation over blind overwrite when conflict risk is high.
	The current model is explicit per named channel and optional custom storage key, so apps choose which topics synchronize instead of forcing every atom or cache key onto one global bus.
- [x] Define bootstrap and resume interaction for synchronized state.
	`docs/CROSS_TAB.md` now defines the current bootstrap rule set: restore SSR bootstrap or persisted snapshots first, open channels after local state is initialized, and ignore incoming messages that are older than the locally restored revision or timestamp.
- [x] Add diagnostics and examples for cross-tab behavior.


	`docs/MULTI_SURFACE.md` now defines the current first-party model as same-origin popup or secondary-window coordination plus the already-shipped cross-tab channel surface, while explicitly leaving iframe, side-panel, and external control-surface orchestration for later work.
- [x] Add coordination channels for popup and secondary-window workflows.
	`interop.OpenSecondaryWindowChannel(...)`, `interop.WindowOpenerChannel(...)`, `interop.DecodeWindowEnvelope[T](...)`, and `interop.SubscribeDecodedWindow[T](...)` now provide first-class `postMessage` coordination for opener and popup workflows, with native and wasm coverage in `interop`.
- [x] Define ownership and synchronization rules across multiple active surfaces.
- [x] Add shared session and route coordination helpers.
- [x] Define teardown and orphan-surface behavior.
- [x] Add examples for multi-window applications.

### Multi-client coordination
- [x] Decide whether the `docs/MULTI_CLIENTS.md` proposal should become a first-class public `interop` surface.
	The project now treats multi-client coordination as an experimental public `interop` slice: the JSON helper layer is implemented in `interop`, its protocol contract is documented in `docs/MULTI_CLIENTS.md`, and its stability tier is called out through `docs/API_POLICY.md`.
- [x] Define the stable public API shape for multi-client coordination.
- [x] Define protocol versioning and compatibility rules for multi-client messages.
- [x] Define capability negotiation for mixed client populations.
- [x] Define the authority model per multi-client topic.
- [x] Define delivery guarantees for client-mesh traffic.
	`docs/MULTI_CLIENTS.md` now states the current guarantees explicitly: best-effort browser-local delivery, at-most-once sender expectations, duplicate tolerance, no global ordering promise, and no built-in replay contract.
- [x] Define request correlation, timeout, and cancellation semantics for `query` and `result` flows.
	The multi-client note now defines `ID` as the correlation key, requester-owned timeout policy, late-result discard, duplicate-result handling, and local-only cancellation semantics for the current first-class contract.
	`docs/MULTI_CLIENTS.md` now records the intended lease and heartbeat model, including refresh on boot, resume, and reconnect, conservative expiry handling, and the expectation that background-tab throttling is normal rather than exceptional.
- [x] Define the full local lifecycle event model for multi-client coordination.
- [x] Define the wire-level handshake and discovery flow in detail.
	`docs/MULTI_CLIENTS.md` now defines subscribe-before-publish startup, `hello` and `goodbye`, late-join `query(topic="clients")`, targeted `result` replies, and the recommendation to derive discovery from handshake and lease state rather than unsupported browser-wide scanning.
	`docs/MULTI_CLIENTS.md` now defines the client mesh as message-oriented and logically full duplex, clearly separating that from socket-style byte streams while leaving payload-size and backpressure policy as follow-up work.
	The multi-client note now defines the required JSON message fields, and `interop` now validates and publishes the JSON control-plane message shape through `ClientMessage` and the new helper functions.
	`docs/MULTI_CLIENTS.md` now defines binary as a planned optional data-plane encoding through `ClientBinaryPayload`, while keeping JSON as the current implemented control plane and leaving transport-specific binary publish helpers for later implementation.
- [x] Define binary capability boundaries across existing transports.
- [x] Define payload-size limits and backpressure behavior for multi-client topics.
- [x] Define topic namespace and ownership conventions.
- [x] Define schema ownership and validation policy per topic.
- [x] Define structured error codes for multi-client failures.
	The multi-client contract now defines the stable failure categories the `interop` error model should cover for unsupported transport, unsupported binary, version mismatch, unauthorized publish, peer unavailability, timeout, payload limits, decode failure, expiry, and orphaned windows.
	`docs/MULTI_CLIENTS.md` now defines same-origin defaults, advisory-versus-enforced role expectations, topic-level authorization expectations, and the rule that privileged intents must still flow through application-owned authorization.
	The multi-client contract now documents same-origin defaults, target-origin validation, trust invalidation on navigation or origin change, and the rule that cross-origin coordination is not the default supported path.
	`docs/MULTI_CLIENTS.md` now defines expired, orphaned, and disconnected peer behavior, including degraded-mode expectations and the rule that degraded presence is operational state rather than a fatal runtime error.
	`docs/MULTI_CLIENTS.md` now defines the intended multi-client event family, and `docs/OBSERVABILITY.md` now references peer discovery, transport resolution, timeout, reconnect, and encoding failures as part of the broader observability model.
- [x] Add structured logging guidance for multi-client coordination.
- [x] Add devtools inspection for multi-client state.
- [x] Add actionable-error guidance for multi-client failures.
- [x] Add a first-party example for multi-client presence and discovery.
- [x] Add a first-party example for mixed JSON and binary multi-client payloads.
- [x] Add a realistic enterprise-shaped reference topology for multi-client coordination.
	`docs/MULTI_CLIENTS.md` now includes a concrete storefront-tab, operator-tab, and popup-inspector topology that exercises authority, topic ownership, transport choice, discovery, degraded mode, and binary-capability boundaries in one production-shaped scenario.
- [x] Add conformance tests for mixed-version and mixed-capability client populations.
	`interop/interop_native_test.go` now covers incompatible major versions, missing capability flags, JSON-only peers, binary-capable peers, topic mismatch, staged rollouts, and capability decoding, while `interop/interop_wasm_test.go` verifies that cross-tab and window hello traffic advertises transport-appropriate default capabilities.
- [x] Add transport-conformance tests for cross-tab and window multi-client flows.
	`interop/interop_wasm_test.go` now covers BroadcastChannel late join, reconnect, duplicate `hello`, and `goodbye`; storage-fallback lifecycle delivery with deterministic lease-expiry simulation from message timestamps; popup orphaning and disposal; and opener-path `hello` plus `goodbye` lifecycle traffic with stable js/wasm assertions.
- [x] Add reliability tests for query, result, timeout, and duplicate delivery handling.
	`interop/interop_native_test.go` now uses a shared mock cross-tab bus to cover request ID correlation, timeout handling, late-result discard, duplicate-result tolerance, concurrent query handling, and simultaneous bidirectional request or reply traffic without adding speculative production machinery.
- [x] Add binary-transport tests for supported and unsupported surfaces.
	`interop/interop_wasm_test.go` now covers binary publish on BroadcastChannel and window paths, verifies content type plus byte identity on round-trip, and asserts that storage-event fallback rejects binary publish with a structured interop error.
- [x] Add security and authorization tests for multi-client topic handling.
	`interop/interop.go` now exposes a dedicated unauthorized error code and enforces privileged topic-family checks for non-privileged roles, while `interop/interop_wasm.go` surfaces same-peer origin mismatch as a structured unauthorized error and still fails closed on stale popup or opener handles; native and js/wasm tests cover unauthorized topic publish, privileged intent rejection, target-origin mismatch, stale opener handles, and malformed peer identities.
- [x] Define the public stability tier for multi-client coordination.
	The multi-client surface is now explicitly classified as experimental public API in `docs/MULTI_CLIENTS.md`, and `docs/API_POLICY.md` now lists the multi-client coordination layer under experimental public surfaces.
- [x] Define explicit non-goals for the multi-client feature set.
	`docs/MULTI_CLIENTS.md` now documents the key non-goals: no distributed database semantics, no shared-memory runtime state, no generic byte-stream transport, no offline replay by default, and no same-document co-ownership without isolation boundaries.
- [ ] Evaluate whether typed RPC streams should become an optional transport for multi-client coordination.
	Decide when presence, shared-session, live dashboard, and collaboration flows should stay on the current JSON control-plane helpers versus moving onto a protobuf-defined RPC stream layer with the existing multi-client authority and observability rules preserved.
	External implementation reference when this work starts: `grpc-tunnel` repo.
- [ ] Define interoperability rules between multi-client topics and RPC-backed live streams.
	Specify whether invalidation, presence, targeted query/result flows, and binary payload cases remain native multi-client topics, become RPC method families, or are allowed to coexist with one explicit ownership rule per concern.
	External implementation reference when this work starts: `grpc-tunnel` repo.

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
- [x] Decide how boundaries compose with nested routes and layouts.
	`docs/ERROR_BOUNDARIES.md` now defines the current rule that `ui.ErrorBoundary` follows ordinary subtree placement: boundaries around `router.Outlet()` isolate leaves, while boundaries around layout shells plus outlets recover both together.
- [x] Define boundary behavior during SSR and hydration.
	`docs/ERROR_BOUNDARIES.md` now documents the shipped behavior: server-side boundaries can render fallback HTML during `ui.RenderToString(...)`, while hydration mismatches still fall back per subtree to client rendering and only later client panics flow through boundary fallback UI.
- [x] Add diagnostics integration for recovered errors.
	Recovered boundary failures now report through runtime diagnostics with subtree path and component-stack context, and the `devtools` diagnostics panel now surfaces that extra context instead of only showing a flat message.

### Production correctness hardening

- [x] Define the minimum production-correctness bar for core runtime flows.
	`docs/PRODUCTION_CORRECTNESS.md` now defines the current minimum bar for render determinism, state convergence, portal cleanup, boundary recovery, hydration fallback, and churn safety before the runtime should claim serious production readiness.
- [x] Add scenario-level correctness suites for complex app flows.
	`internal/runtime/production_correctness_test.go` now adds composed-flow coverage that exercises hydration reuse, shared atoms, portals, and boundary recovery together instead of only in isolated unit tests.
- [x] Add long-running stability and leak detection coverage.
	The same runtime correctness file now includes repeated mount/unmount churn coverage that checks cleanup execution, atom-subscriber release, and scheduled-work settlement across long-lived toggling.
- [x] Add concurrency and race-condition stress tests.
	The runtime correctness pass now includes overlapping urgent plus transition update bursts and verifies that the final DOM and shared atom state converge consistently without leftover pending work.
- [x] Add production-mode behavior parity checks.
	`utils` now provides an export-complete `production` build variant, and the release workflow now validates production-tag wasm compiles so stripped builds keep the same public helper and scheduling surface instead of silently dropping symbols.

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
- [x] Define scheduler priority classes.
	`docs/SCHEDULING.md` now defines the current scheduler as a deliberate two-lane model only: urgent work and transition work, with no richer public priority ladder implied yet.
- [x] Prototype a pending-state API for non-urgent updates.
	The same scheduling contract now records the shipped answer: `ui.UseTransition()` already provides both the non-urgent starter and `Pending()` flag, so no second generic scheduler-pending hook is needed today.
- [x] Measure interruptibility requirements under heavy updates.
	`internal/runtime/scheduler_benchmark_test.go` now includes transition-heavy list refresh benchmarking, and `docs/SCHEDULING.md` records the current conclusion that work is deferred but not yet time-sliced or split mid-render.
- [x] Decide how scheduling primitives interact with route loaders and async boundaries.
	`docs/SCHEDULING.md` now defines that transitions lower the priority of local state work only; they do not suppress router loader pending UI or `ui.AsyncBoundary` fallback behavior.
- [x] Add browser examples for transition-style UX.
	`examples/27-transition-hooks` now covers typeahead filtering, dashboard tab swaps, and route-style section transitions so `ui.StartTransition(...)` and `ui.UseTransition()` are demonstrated in real app-shaped flows instead of only a toy counter.

## 7. SSR, Hydration, State Transfer, and Streaming

### Hydration correctness

- [x] Teach the runtime to bind fibers to existing DOM nodes.
	The shipped hydration path in `internal/runtime/hydration.go` now binds hydrated fibers to matching DOM candidates instead of clearing the container, and `docs/HYDRATION.md` documents that contract.
- [x] Define the initial hydration matching rules.
	`docs/HYDRATION.md` now records the shipped matching rules for host elements, text nodes, ignored whitespace, fragments, extra trailing nodes, and mismatch-triggered boundary fallback.
- [x] Defer effects and subscriptions until hydration completes.
	Hydration queues atom subscriptions and hydration-time updates until commit finishes, then runs effects against the committed tree; `docs/HYDRATION.md` now states that public contract.
- [x] Add subtree fallback behavior when hydration cannot safely continue.
	The runtime already falls back at the current hydration boundary instead of restarting the whole app, and `docs/HYDRATION.md` now documents that subtree-scoped behavior explicitly.
- [x] Add tests for successful hydration of simple pages and post-hydration updates.
	`internal/runtime/hydration_test.go` already covers simple reuse, mismatch recovery, trailing-node cleanup, and post-hydration updates, so the backlog now reflects the shipped coverage.
- [x] Define event listener attachment order during hydration.
	`docs/HYDRATION.md` now records the shipped rule: handlers are rebound during the hydrated commit on reused nodes after DOM matching, before effects run, and the runtime does not replay pre-hydration browser events.
- [x] Preserve uncontrolled form state where safe.
	Hydration now preserves live `value`, `checked`, `selected`, and `autofocus` state on reused nodes during the initial resume pass, and `internal/runtime/hydration_test.go` covers preserving a live input draft until a later explicit update.
- [x] Define hydration behavior for portals, lazy nodes, async boundaries, and event-heavy components.
	`docs/HYDRATION.md` now defines the current shipped behavior for portal children, async-boundary branches, lazy loaders starting after commit, and handler rebinding on event-heavy subtrees.
- [x] Add route-aware hydration reuse tests.
	`router/router_test.go` now covers nested loader-cache reuse for `HydrateMount(...)`, verifying the first hydrated route read can reuse preseeded layout and leaf loader state without duplicating route work.
- [x] Add progressive hydration benchmarks.
	`internal/runtime/hydration_benchmark_test.go` now measures simple reuse, mismatch fallback, medium-tree hydration cost, and a first-update-after-hydration path so the runtime has concrete baseline numbers beyond the tiny host-tree case.

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
- [x] Define serialization support for non-JSON-friendly values.
	`docs/STATE_TRANSFER.md` now defines the current typed payload encoding rules for `time.Time`, byte slices, custom structs, text-marshaled ids, and explicit CBOR payloads, and `ui` exposes typed bootstrap payload helpers instead of relying on ad hoc `map[string]interface{}` packing alone.
- [x] Add versioning to bootstrap payloads.
	`ui.SSRBootstrap`, `ui.SSRBootstrapReference`, and the post-bootstrap update envelope now carry explicit schema versions, default missing versions to the current shipped schema for backward compatibility, and reject unsupported newer versions during decode instead of silently misreading them.
- [x] Define partial bootstrap reuse rules.
	`docs/STATE_TRANSFER.md` now records framework-level typed payload reuse policies (`trust-once`, `revalidate-after-resume`, and `client-owned`) so first-resume trust and immediate revalidation decisions are explicit instead of living only in package-specific or example-specific notes.
- [x] Add typed helpers for server-to-client payload registration.
	`ui.RegisterBootstrapPayload(...)`, `ui.RegisterRouteBootstrapData(...)`, `ui.RegisterFormBootstrapDefaults(...)`, `ui.RegisterCacheBootstrapSeed(...)`, `ui.RegisterSessionBootstrapHint(...)`, and the corresponding read helpers now provide typed registration and retrieval for app-facing bootstrap payloads.
- [x] Define per-route and per-subtree bootstrap scoping.
	Typed payload envelopes now carry explicit `Scope` and `Target` metadata, and `ui.InspectBootstrapPayloads(...)` exposes that scoping information for diagnostics and app-level routing or subtree ownership decisions.
- [x] Add incremental text and binary state update transports.
	`ui.SSRStateUpdate` plus JSON and CBOR encode/decode helpers now define a versioned post-bootstrap text or binary update envelope for application-owned payload upserts and removals after hydration.
- [x] Add payload size budgeting and diagnostics.
	`ui.AnalyzeSSRBootstrapSize(...)` now measures JSON, inline-script, and CBOR payload sizes, reports warning and error bands, and recommends an inline JSON, sidecar JSON, or sidecar CBOR delivery mode.
- [ ] Add a serialization boundary verifier across app transport edges.
	Provide one analysis surface that checks what crosses server-to-client bootstrap, SSR-to-hydrate resume, worker-to-main-thread envelopes, RPC request or response payloads, and shared-session or multi-client synchronization boundaries so hybrid app correctness is not left to ad hoc JSON errors.
- [ ] Add static serialization analysis for non-deterministic and non-serializable values.
	Inspect app code and typed payload registration paths for functions or closures, browser-native handles, hidden mutable references, nondeterministic values such as time- or random-dependent fields, and other values that cannot safely survive transfer or replay across serialization boundaries.
- [ ] Add runtime payload sampling and bootstrap snapshot verification.
	Capture representative payload shapes during SSR bootstrap emission, hydration resume, worker messaging, RPC calls, and client-sync transport so the framework can flag giant payloads, accidental secret leakage, and boundary-shape drift that static analysis alone cannot prove.
- [ ] Add serializable-safe type tagging and policy modes.
	Define how apps can mark payload types as explicitly serializable-safe, allowlisted, redacted, or boundary-owned, and support strict, warn, and allowlist policy modes so teams can ratchet enforcement from advisory diagnostics to CI-blocking verification.
- [ ] Add HTML and bootstrap boundary analyzers for SSR output.
	Inspect emitted bootstrap scripts, sidecar payload references, and HTML snapshots for secret-bearing fields, oversized inline payloads, unstable ordering, and resume-time mismatches so SSR and hydration bugs are caught before they ship.
- [x] Add server-to-client state classification guidance.
	`docs/STATE_TRANSFER.md` now separates public bootstrap state, runtime-owned resumable state, cache or feature seeds, and server-only secrets so SSR payload design has an explicit trust model.
- [x] Define merge semantics for transferred state.
	`docs/STATE_TRANSFER.md` now defines bucket-specific merge rules for `Route`, `Atoms`, `Data`, `I18n`, and `IDSeed` instead of relying on one generic deep-merge story.
- [x] Define state transfer ownership during hydration.
	`docs/STATE_TRANSFER.md` now records which bootstrap values become runtime-owned immediately during hydration, which remain application-owned hints, and when later client state should take over.

### Streaming SSR

- [x] Treat streaming SSR as an explicit post-hydration milestone.
	`docs/STREAMING_SSR.md` now makes streamed SSR an explicit post-hydration milestone instead of part of the current supported SSR contract.
- [x] Design chunked HTML streaming for route loaders.
	`docs/STREAMING_SSR.md` now defines the first intended model: flush a stable shell first, stream loader-backed region HTML later by explicit segment identity, then hydrate against the fully assembled DOM.
- [x] Define async-boundary behavior under streaming SSR.
	`docs/STREAMING_SSR.md` now defines the first intended boundary behavior: resolved regions flush as normal HTML, pending boundaries flush placeholder HTML first, later streamed segments keep the same region identity, and final hydration is against the assembled DOM.
- [x] Add transport and buffering rules for streamed responses.
	`docs/STREAMING_SSR.md` now records the first transport expectations around reverse proxies, gzip or brotli buffering, CDN buffering, meaningful shell chunk sizing, and fallback behavior when incremental flushes are defeated.
- [ ] Add examples and benchmarks for streaming SSR.
	Use a loader-heavy page and a nested layout route to verify faster first byte, earlier shell paint, and correct hydration after incremental HTML delivery.
- [ ] Add a minimal real streaming implementation for one routed app shape.
	Move beyond design notes by shipping one loader-heavy route family that actually flushes shell HTML early, streams a deferred region later, and proves that the first implementation works under the current SSR and hydration contract.
- [ ] Add failure-mode tests for late segment errors and nested streamed layouts.
	Cover streamed error replacement, cancellation of abandoned regions, nested layout shells, and hydration against the final assembled DOM so the first streaming slice is correct under more than the happy path.
- [ ] Add proxy-aware streaming verification fixtures.
	Measure behavior with and without compression, through realistic buffering layers, and under meaningful chunk sizes so the streaming story can be scored as H3 only when it survives production-shaped delivery paths rather than local-dev flush behavior.

### Hydration mismatch diagnostics

- [x] Add mismatch detection and reporting.
	`internal/runtime/hydration.go` already detects text, structure, trailing-node, and critical attribute mismatches and reports them through the runtime diagnostics surface; `docs/HYDRATION.md` now reflects that shipped behavior instead of leaving the duplicate backlog item open.
- [x] Add tests for mismatch reporting and recovery behavior.
	`internal/runtime/hydration_test.go` already covers deterministic warning and recovery behavior for text mismatches, structural subtree fallback, and trailing-node cleanup, and `docs/HYDRATION.md` now points at that coverage.
- [x] Add component-stack context to mismatch diagnostics.
	`internal/runtime/hydration.go` now reports hydration mismatches through `ReportDiagnosticWithContext(...)`, so the runtime diagnostics surface includes fiber path and component-stack context for text, attribute, fallback, and trailing-node failures; `internal/runtime/hydration_test.go` covers the added context.
- [x] Add an opt-in strict hydration mode.
	`ui.HydrationOptions{Strict: true}` now turns hydration mismatches into fail-fast diagnostics and panics instead of warning and continuing, and `internal/runtime/hydration_test.go` covers both text-mismatch and structural-mismatch strict-mode failures.
- [x] Define production mismatch behavior.
	`docs/HYDRATION.md` now records the current production-oriented rule set: text and attribute mismatches warn and continue, structural mismatches trigger subtree replacement, and strict fail-fast hydration is available only as an opt-in development and test mode.

## 8. Server Integration, Deployment, and Production Patterns

- [x] Define a canonical Go HTTP integration story.
	`docs/SERVER_INTEGRATION.md` now defines the current production-shaped `net/http` model for SSR apps, covering request handlers, request-scoped rendering, static asset serving, bootstrap emission, and same-origin APIs and form posts.
- [x] Add middleware guidance for SSR apps.
	`docs/SERVER_INTEGRATION.md` now records the recommended middleware order for SSR apps, including logging, recovery, compression, cache headers, auth or session context, CSRF handling, and request-context propagation.
- [x] Add a first-party SSR app reference server.
	`examples/86-atlas-commerce-os/server` is now documented in `docs/SERVER_INTEGRATION.md` and `examples/README.md` as the current production-shaped SSR reference server, combining SSR, hydration, same-origin APIs and mutations, static asset serving, bootstrap payloads, and server-owned session-aware routes in one Go application.
- [x] Define backend API integration patterns.
	`docs/SERVER_INTEGRATION.md` now defines the current internal-versus-external API guidance for route loaders, `fetch.UseResource[T](...)`, and form handlers, including timeout and auth propagation expectations.
- [x] Add deployment guidance for common hosting modes.
	`docs/SERVER_INTEGRATION.md` now documents the supported hosting modes for static bundles, single-process Go SSR, reverse-proxy fronted Go servers, and split SSR/API deployments, including route rewrite, cache, and `.wasm` serving guidance.
- [x] Add observability hooks for server-rendered apps.
	`ui` now exposes SSR observability callbacks for server render timing, bootstrap payload and inline-script size metrics, and hydration summaries including fallback and mismatch counters with optional correlation ids.

### Observability and runtime instrumentation

- [x] Define a structured runtime event model.
	`docs/OBSERVABILITY.md` now defines the intended event envelope for framework instrumentation, including stable event names, domains, phases, timestamps, correlation ids, span ids, and safe structured attributes across SSR, runtime, router, fetch, state, interop, and worker flows.
- [x] Add correlation IDs across SSR, bootstrap, and hydration.
	`docs/OBSERVABILITY.md` now defines the intended correlation-id flow from HTTP request entry through SSR, bootstrap transfer, first hydration, and later client-side navigations.
- [x] Add request-scoped tracing for server-rendered apps.
	`docs/OBSERVABILITY.md` now defines the intended SSR trace shape, including route match timing, auth/session resolution, loader timing, render timing, bootstrap serialization timing, and response write timing under one request-scoped trace.
- [x] Add client-side lifecycle instrumentation hooks.
	`docs/OBSERVABILITY.md` now defines the intended client lifecycle hook surface for navigation, loader, hydration, async-boundary, cache, worker, and render or commit instrumentation without claiming that every emitter already exists.
- [x] Define export formats for traces and runtime metrics.
	`docs/OBSERVABILITY.md` now defines the intended export targets for devtools, local file capture, trace-shaped exports, and metrics snapshots while keeping the internal event model transport-agnostic.
- [x] Add sampling and noise-control rules for instrumentation.
	`docs/OBSERVABILITY.md` now defines the intended sampling defaults, aggregation boundaries, ring-buffer expectations, and error-first retention rules for high-volume instrumentation streams.
- [ ] Add examples and docs for end-to-end observability.
	Demonstrate how an application traces one routed page load, one async resource flow, and one server-rendered request through the public instrumentation hooks.
- [ ] Add first-party integration recipes for traces, metrics, perf marks, and external error reporting.
	Show how the framework event model connects to OpenTelemetry-style tracing, browser performance marks, structured app metrics, and hosted error-reporting sinks without forcing every team to rediscover the same wiring boundaries.

### Logging and diagnostics surface

- [x] Define a first-class logger interface and domain model.
	`docs/LOGGING.md` now defines the intended structured logger contract, including stable levels, domains, correlation ids, and safe structured fields across runtime, router, fetch, state, hydration, SSR, interop, worker, and forms.
- [x] Add structured development and production log outputs.
	`docs/LOGGING.md` now defines the intended split between human-readable development logs and machine-readable production records while keeping one shared event model across both outputs.
- [x] Add redaction and secret-boundary rules for logs.
	`docs/LOGGING.md` now defines the framework-owned redaction rules for bootstrap payloads, auth/session material, request headers, form payloads, and correlation identifiers so convenience logging does not leak sensitive data by default.
- [x] Add log integration for routes, loaders, and mutations.
	The runtime now buffers structured framework logs, `router` emits navigation/redirect/loader lifecycle logs, `fetch` emits cache invalidation and offline mutation replay logs, and the coverage in `router/router_test.go` plus `fetch/mutation_queue_wasm_test.go` exercises those lifecycle entries.
- [x] Add in-memory devtools log buffering.
	The runtime inspection surface now includes a bounded in-memory log buffer, `devtools.SnapshotNow()` maps it into `Snapshot.Logs`, `devtools.Panel` renders the recent framework logs, and `devtools/devtools_wasm_test.go` covers the snapshot side of that buffer.
- [x] Add warning classification and escalation rules.
	Runtime diagnostics and buffered logs now carry classification metadata for correctness, performance, recovered, unsupported-but-recovered, and informational cases, and `docs/LOGGING.md` records the intended escalation rules for each class.

### Static prerender and export workflows

- [x] Define whether static prerender is a first-class output mode.
	`docs/PRERENDER.md` now defines static prerender as an intended first-class output mode distinct from request-time SSR, and records the current tooling boundary between core rendering primitives and companion export orchestration.
- [ ] Add a prerender-to-files pipeline.
	Support rendering one or more routes to HTML files plus associated bootstrap payloads so docs sites, marketing pages, and hybrid static apps do not need a live Go server for initial delivery.
- [x] Define route enumeration for prerendered apps.
	`docs/PRERENDER.md` now defines explicit route lists, parameter expansion hooks, application-owned dynamic enumeration, and fallback treatment for routes that cannot be known safely at build time.
- [x] Define asset and bootstrap output conventions for prerender.
	`docs/PRERENDER.md` now defines the intended HTML, sidecar bootstrap, manifest, and copied-asset output shape for prerendered sites without overclaiming a shipped exporter implementation.
- [x] Add partial-hydration or client-resume expectations for prerendered output.
	`docs/PRERENDER.md` now records the intended split between fully static pages, fully hydrated pages, and future selective activation, while keeping prerendered resume on the normal hydration contract.
- [ ] Define a first-class islands or selective-hydration model.
	Move beyond future-looking notes by specifying the ownership model for static regions versus selectively activated islands, how island boundaries compose with routing and async boundaries, and what guarantees remain for SSR and hydration correctness.
- [ ] Add one islands-style reference example and budget-driven validation.
	Ship a content-heavy page or marketing-style route that hydrates only selected interactive regions, then measure startup, hydration, and interaction costs so selective activation is evaluated as a real product feature instead of a design note.
- [x] Add invalidation and rebuild guidance for prerendered content.
	`docs/PRERENDER.md` now defines the intended rebuild triggers for route content, shared layouts, asset manifests, and expanded route data in local development and CI.
- [ ] Add a first-party static export example.
	Ship a docs or marketing-style example that prerenders several routes to files and then serves them from a static host without a custom server runtime.

### Image, media, and asset delivery ergonomics

- [x] Define the framework-level asset delivery story.
	`docs/ASSETS.md` now defines the boundary between core rendering, export tooling, manifests, hashing, copied static files, and deployment responsibilities for SSR and prerendered outputs.
- [x] Add asset-manifest guidance for application builds.
	`docs/ASSETS.md` now defines the intended manifest role, the logical-to-emitted asset mapping model, and how SSR and prerendered HTML should resolve wasm, JS helpers, CSS, images, fonts, and copied static files.
- [x] Add cache-busting and static asset versioning conventions.
	`docs/ASSETS.md` now defines filename-based fingerprinting, manifest lookup boundaries, and cache-header expectations for mutable entrypoints versus immutable assets.
- [x] Add responsive image and media guidance.
	`docs/ASSETS.md` now records the intended `srcset`, `sizes`, intrinsic sizing, lazy-loading, decoding, poster, and art-direction guidance for application-authored media markup.
- [x] Evaluate first-class helpers for image and media components.
	`docs/ASSETS.md` now defines the intended split between core HTML rendering and a future companion image or media helper package, without overclaiming a shipped helper surface.
- [x] Add preload and prefetch guidance for static assets.
	`docs/ASSETS.md` now defines preload, modulepreload, preconnect, DNS-prefetch, and prefetch guidance, and `docs/HEAD_MANAGEMENT.md` links the resource-hint ownership rules back to the broader asset-delivery contract.
- [x] Add lazy media loading patterns for routed apps.
	`docs/ASSETS.md` now defines route-oriented lazy media patterns, including layout stability, placeholder strategy, cancellation ownership, and above-the-fold exceptions for route-critical media.
- [x] Add SSR and prerender rules for asset references.
	`docs/ASSETS.md` now defines the manifest-driven asset resolution, base-path handling, and shared URL rules that SSR and prerendered HTML should follow.
- [x] Add CDN and static-host deployment guidance for assets.
	`docs/ASSETS.md` now defines immutable-asset caching, HTML and manifest freshness, origin separation, compression expectations, and static-host deployment rules for emitted asset trees.
- [ ] Add example apps that stress real asset delivery concerns.
	Ship a content-rich example with responsive images, route-scoped preload hints, hashed asset output, and lazy media loading so the public story is validated end to end.

### Browser support and compatibility policy

- [x] Publish an explicit browser support matrix.
	List the minimum supported desktop and mobile browsers, including Safari and mobile Safari expectations, so adopters know which environments the runtime and examples are expected to work in.
- [x] Define the required browser feature baseline for the wasm runtime.
	Document which web platform features are assumed by core packages, router behavior, fetch helpers, workers, devtools, and SSR hydration so compatibility is based on concrete capabilities rather than vague â€œmodern browserâ€ language.
- [x] Define the project stance on polyfills and shims.
	`docs/BROWSER_SUPPORT.md` now defines the no-framework-polyfill stance, the application-owned compatibility boundary, and how new feature dependencies should be communicated.
- [x] Add progressive-enhancement boundaries for partial support cases.
	`docs/BROWSER_SUPPORT.md` now defines what remains available for prerendered and SSR-delivered content when JavaScript, wasm startup, or optional browser APIs are unavailable.
- [ ] Add mobile Safari and constrained-device validation coverage.
	Test the framework against the browser family most likely to expose wasm, caching, input, and memory edge cases before calling any workflow production-ready.
- [x] Define compatibility expectations for workers, offline features, and advanced APIs.
	`docs/BROWSER_SUPPORT.md` now defines the capability-check and reduced-functionality expectations for workers, offline flows, cross-tab features, and other advanced interop APIs.
- [ ] Add browser-compatibility CI coverage or release checks.
	Run a small compatibility matrix against representative examples so regressions in supported browsers are detected before release instead of after user reports.
- [x] Add fallback and degradation guidance for unsupported browsers.
	`docs/BROWSER_SUPPORT.md` now defines unsupported-browser and degraded-mode guidance, including when to prefer readable SSR content, reduced functionality, or explicit unsupported-browser messaging.
- [x] Add a browser-support policy to release documentation.
	`docs/BROWSER_SUPPORT.md`, `docs/API_POLICY.md`, and `docs/MIGRATIONS.md` now define how browser-support changes are announced, documented, and gated before a narrower compatibility matrix takes effect.

### PWA and offline app support

- [x] Define the framework's PWA support boundary.
	`docs/PWA.md` now defines the intended PWA boundary as documented integration points for manifests, service workers, offline caching, and mutation replay rather than a hidden core-runtime feature.
- [x] Add web app manifest generation or templating helpers.
	The new `pwa` package now provides `pwa.Manifest`, related icon and shortcut structs, validation, and JSON marshalling helpers so apps can generate installability metadata without hand-rolling manifest payloads.
- [x] Add a service-worker integration story.
	`docs/PWA.md` now defines the intended application-owned service-worker registration, precache, scope, and asset-versioning story without overclaiming a shipped service-worker runtime.
- [x] Define offline caching strategies for app shells and route data.
	`docs/PWA.md` now defines the intended separation between shell caching, immutable asset caching, route-data caching, and queued mutation replay.
- [x] Add update and invalidation semantics for offline assets.
	`docs/PWA.md` now defines versioned cache invalidation, manifest-aligned asset updates, and safe refresh expectations for new wasm or JS asset graphs.
- [x] Add offline and installability examples.
	The examples catalog now includes `97-pwa-installability` for manifest wiring, installability state, and scoped service-worker ownership, plus `97-pwa-offline-cache` for versioned cache warmup, offline fallback documents, queue inspection, and structured PWA diagnostics.
- [x] Add production guidance for PWA deployments.
	`docs/PWA.md` now defines HTTPS, scope, cache-header, CDN, reverse-proxy, and rollout guidance for PWA deployments.
- [x] Add a first-class browser persistence abstraction for durable offline data.
	`interop.OpenPersistentStore(...)` now provides an IndexedDB-first durable key/value surface with explicit fallback storage, typed JSON helpers, capability-aware errors, and focused native/wasm tests so later cache, queue, and state work can reuse one maintained persistence boundary.
- [x] Add a first-class IndexedDB story for cache, queue, and state persistence.
	`interop.OpenPersistentStore(...)` now defines a stable IndexedDB store layout, versioned object-store creation, blocked-upgrade diagnostics, quota-aware request failures, and opt-in corruption recovery by database reset. `fetch.ConfigurePersistentCache(...)`, `fetch.OpenMutationQueue(...)`, and `state.SavePersistentSnapshot(...)` all ride that same persistence seam so cache, queue, and state work share one maintained durable-storage contract instead of separate ad hoc IndexedDB implementations.
- [x] Add Cache Storage helpers for immutable assets and offline shells.
	`pwa.BuildCacheStoragePlan(...)` now builds release-scoped cache namespaces plus explicit strategies for shell HTML, wasm, scripts, styles, media, and other immutable assets, and `pwa.OpenCacheStorageManager()` now applies and inspects that plan against browser `caches` with versioned namespace cleanup.
- [x] Add durable shared-read cache persistence.
	`fetch.CacheOptions` now supports `Persist`, `fetch.ConfigurePersistentCache(...)` configures the durable backing store, and shared cached resources now restore prior ready values from IndexedDB-first browser storage before cold loads while still respecting `StaleAfter`, `MaxAge`, and explicit disposal semantics.
- [x] Add IndexedDB-backed persistence options for mutation queues.
	`fetch.OpenMutationQueue(...)` now uses the same IndexedDB-first durable persistence seam as the shared fetch cache while still supporting explicit store overrides and `localStorage` fallback when IndexedDB is unavailable.
- [x] Add IndexedDB-backed persistence options for state snapshots.
	`state.SavePersistentSnapshot(...)`, `state.LoadPersistentSnapshot(...)`, and `state.RestorePersistentSnapshot(...)` now provide IndexedDB-first durable snapshot persistence with `localStorage` fallback so atom snapshots are not limited to the smaller local/session storage helpers.
- [x] Add installability helpers and diagnostics.
	`pwa.ObserveInstallability(...)` now surfaces manifest validation, `beforeinstallprompt`, `appinstalled`, explicit prompting, installed state, and user-visible reasons install is unavailable so applications can own install flows deliberately instead of reverse-engineering browser behavior.
- [x] Add explicit service-worker registration and update-lifecycle helpers.
	`pwa.RegisterServiceWorker(...)` now provides app-owned registration, typed lifecycle snapshots, waiting-worker inspection, `SkipWaiting(...)`, `ReloadOnControllerChange()`, `Update(...)`, `Unregister(...)`, and native unavailable stubs so service-worker ownership stays explicit without pushing this logic into the core rendering runtime.
- [x] Add service-worker asset-manifest integration for wasm releases.
	`pwa.ParseWasmReleaseManifestJSON(...)` now parses the existing `wasm-release-manifest.json` format and `pwa.BuildServiceWorkerAssetPlan(...)` turns that release record into a stable revision, cache namespace, wasm URL, and deduped precache inputs so service workers can reuse the same release manifest for cache invalidation and safe reload decisions.
- [x] Add offline route-opening and fallback behavior.
	`docs/PWA.md` now defines route-family offline policy for deep links, app-shell fallbacks, partial offline routes, and online-required routes so service-worker fallbacks do not quietly invent application behavior when the network is unavailable.
- [x] Add replay and cache coordination across tabs.
	`docs/PWA.md` and `docs/CROSS_TAB.md` now define single-owner replay, authoritative logout propagation, and cache-invalidation fanout on top of `interop.OpenCrossTabChannel(...)` so multi-tab offline work has one documented ownership model instead of ad hoc duplicate replay.
- [x] Add first-class PWA diagnostics and devtools visibility.
	`pwa.InspectDiagnostics(...)` now returns one structured snapshot for manifest validation, installability state, service-worker lifecycle, Cache Storage inspection, offline queue summary, and browser storage-pressure signals, with `pwa.MutationQueueDiagnosticsSource(...)` bridging `fetch.MutationQueue` into that view without relying on raw console output.
- [x] Add security and retention rules for durable offline data.
	`docs/PWA.md`, `docs/CACHE.md`, `docs/OFFLINE_MUTATIONS.md`, and `docs/SECURITY.md` now define what may be persisted, which data classes are reconstructible versus sensitive, how logout or user-switch flows should purge durable state, and why durable offline stores must carry explicit retention windows instead of relying on browser eviction.
- [x] Add production-grade PWA validation coverage.
	The examples Playwright suite now covers installability signals and update requests, offline shell warmup plus stale-cache cleanup, Background Sync fallback messaging, conflict-aware replay resolution, offline navigation fallback, queued write replay, and multi-tab coordination through focused browser automation across `97-pwa-installability`, `97-pwa-offline-cache`, `97-pwa-multi-client`, and `94-cross-tab-sync`, with shared PWA or cross-tab helpers under `examples/tests/support/pwa.ts`.
- [ ] Add a full offline-first reference app that combines the current pieces.
	Ship one coherent example that exercises installability, shell caching, offline navigation, durable queued writes, replay, multi-tab ownership, and logout-safe data purging together instead of validating each slice only in isolation.
- [ ] Define a framework-level offline-first sync model beyond queue replay.
	Move from low-level cache plus queue primitives to one supported application model for offline mutations, queued writes, reconnect reconciliation, merge policy selection, and conflict ownership so product teams do not have to invent their own sync architecture for field, warehouse, inspection, or healthcare-style apps.
- [ ] Add stronger conflict-resolution and operator-recovery patterns for durable replay.
	Define how queued writes surface idempotency, server-side conflicts, superseded local intents, and replay review UIs so the offline mutation story matures from low-level queueing into a production-grade workflow pattern.
- [ ] Add first-class merge policies and reconnect reconciliation semantics.
	Define supported strategies such as server-wins, client-wins, revision-aware reject, field-level merge, and operator-reviewed reconciliation, plus the point in the reconnect flow where each policy runs so offline replay does not silently overwrite authoritative state.
- [ ] Add per-entity sync health and conflict state inspection.
	Expose whether each tracked entity is clean, pending, replaying, conflicted, blocked, or stale so applications can render row-level or form-level sync health instead of reducing the entire offline state to one generic queue counter.
- [ ] Add conflict-oriented UI helpers and reference workflows.
	Provide supported patterns for conflict banners, per-record repair screens, replay review queues, and reconnect summaries so offline-capable apps can present merge failures and operator decisions without rebuilding the same UX primitives in every product.
- [ ] Add operational guidance for long-lived offline data management.
	Document and test storage pressure recovery, stale queue expiry, replay backlogs after long disconnects, and purge-on-user-switch or logout behavior so the offline story reaches H3-level deployment readiness rather than stopping at core primitives.

### Security, compliance, and governance

- [x] Produce a framework threat model for browser, SSR, and bootstrap flows.
	`docs/SECURITY.md` now defines the intended trust boundaries around bootstrap payloads, route data, interop calls, offline storage, workers, and operational sinks.
- [x] Define strict server-only versus client-safe data boundaries.
	`docs/SECURITY.md` now defines the intended classification between client-safe data, rendered results, and server-only secrets that must never cross into bootstrap or browser-visible logs.
- [x] Add secure-by-default guidance for SSR and hydration.
	`docs/SECURITY.md` now defines the intended escaping, serialization, payload-embedding, and CSP-aware defaults for SSR and hydration flows.
- [x] Add redaction and secret-handling policy for diagnostics and logs.
	`docs/SECURITY.md`, `docs/LOGGING.md`, and the linked SSR/bootstrap docs now define the intended redaction-first policy for framework-owned logs, diagnostics, and traces.
- [x] Define dependency and supply-chain review practices.
	`docs/SECURITY.md` now defines the intended review posture for Go modules, npm-based tooling, generated assets, post-processing steps, and release-time artifacts.
- [ ] Add security regression tests for critical surfaces.
	Cover bootstrap serialization, SSR escaping, interop boundaries, config injection, and logging redaction so security-sensitive invariants are enforced in CI.
- [x] Publish incident response and vulnerability reporting guidance.
	`docs/SECURITY.md` now defines the intended reporting, triage, patch, and disclosure path for security-sensitive issues.
- [x] Define compliance-oriented deployment guidance.
	`docs/SECURITY.md` now defines the intended operational posture for reproducible builds, environment segregation, retention, audit logging, and regulated deployments.

### Feature flags and environment configuration

- [x] Define the framework-level runtime configuration model.
	`docs/CONFIGURATION.md` now defines the intended split between build-time values, server-only runtime config, public runtime config, and evaluated feature state.
- [x] Add a first-class feature-flag context and evaluation surface.
	`docs/CONFIGURATION.md` now defines the intended first-class flag snapshot model for component trees, route loaders, and form or mutation flows without overclaiming a shipped provider implementation.
- [x] Define server-to-client transfer rules for public config and flags.
	`docs/CONFIGURATION.md` now defines the browser-safe transfer boundary for public runtime config and evaluated flags across SSR, hydration, and static export flows.
- [x] Add environment layering and override rules.
	`docs/CONFIGURATION.md` now defines the intended precedence between defaults, build-time values, server runtime config, request-time evaluation, and local development overrides.
- [x] Add gating semantics for routes, components, and experiments.
	`docs/CONFIGURATION.md` now defines the intended gating boundaries for route availability, component branching, and experiment ownership across SSR and hydration.
- [x] Add diagnostics and safety guidance for flag usage.
	`docs/CONFIGURATION.md` now defines the intended diagnostics visibility, secret-boundary rules, and stale-flag cleanup guidance for runtime config and feature flags.
- [ ] Add examples for staged rollout and environment-aware apps.
	Demonstrate a feature-gated route, an environment-configured API endpoint, and an SSR-aware flag evaluation flow that stays consistent between server and client.

### Wasm build optimization, size, and release engineering

- [x] Define first-class development and production wasm build profiles.
	`docs/WASM_RELEASES.md` now defines the intended development, CI verification, benchmark, and production wasm build profiles and their differing goals.
- [x] Add a Go-native profile runner for app builds.
	`go run ./tools/gwc build --profile ...` now resolves a js/wasm build through one Go launcher entrypoint with `development`, `ci`, `benchmark`, and `release` profiles, explicit output-path handling, and machine-readable JSON summaries for build automation.
- [x] Add a Go-native release packaging command.
	`go run ./tools/gwc release` now resolves the target app through the launcher, writes a release-profile wasm artifact into a release output directory, emits `wasm-release-manifest.json`, generates a gzip sidecar by default, supports optional budget enforcement, and can emit a structured JSON summary for automation.
- [x] Define scaffold metadata fields for build and release profiles.
	`gwc-start.json` now records the default wasm entrypoint, HTML shell, dev host/port, build profile, release output directory, release binary name, release compression policy, and optional release budget path so `gwc dev`, `gwc build`, and `gwc release` can resolve launcher defaults from metadata first instead of relying on heuristics and ad hoc command flags.
- [x] Add profile-aware machine-readable launcher output.
	`gwc build -json` and `gwc release -json` now emit the resolved app path, project root, package directory, selected profile, output/manifest paths, emitted artifact sizes, and sha256 hashes so automation can consume launcher output without scraping human-readable logs.
- [x] Add recommended production build flags for wasm targets.
	`docs/WASM_RELEASES.md` now defines the intended production baseline around `-trimpath` and `-ldflags="-s -w"` together with the reproducibility and debugging tradeoffs.
- [x] Define build metadata and debug-info policy for release wasm artifacts.
	`docs/WASM_RELEASES.md` now defines the intended split between stripped production artifacts, explicit debug-friendly builds, and external provenance records for release metadata.
- [x] Add wasm size budgets and regression tracking.
	`tools/build-wasm-release.ps1` now supports optional raw/gzip/brotli budget enforcement, and `docs/WASM_RELEASES.md` defines the intended budget contract for release-oriented builds.
- [x] Add artifact size reporting and comparison tooling.
	`tools/build-wasm-release.ps1` now emits `wasm-release-manifest.json` with relative paths, sizes, and sha256 hashes for release artifacts and compressed sidecars.
- [x] Add release-time compression support for wasm artifacts.
	`gwc release` now supports `none`, `gzip`, `brotli`, or `gzip+brotli` compression policies, defaults to gzip plus Brotli sidecars, and records the selected policy in the release manifest so the Go-native release path now matches the documented compressed-artifact output shape.
- [x] Decide whether launcher-owned Brotli must be pure Go.
	`gwc release` now uses a pure-Go Brotli encoder for the default launcher-owned `.br` sidecar path, so deployable release artifacts no longer depend on PowerShell-only or Node-only compression helpers.
- [ ] Evaluate post-link wasm optimization tooling.
	Test tools such as `wasm-opt` or equivalent post-processing pipelines and document whether they improve size, startup time, or runtime behavior enough to justify adding them to the release workflow.
- [ ] Add per-package or symbol-level wasm size attribution.
	Help developers answer which packages, generated assets, or feature slices are responsible for release-artifact growth instead of only reporting final raw and compressed totals.
- [ ] Add release-to-release wasm size diff reports with likely culprit summaries.
	Compare current and previous release manifests so teams can see what changed size, which profiles regressed, and which emitted artifacts or dependencies most likely drove the increase.
- [x] Define asset-manifest and build-output conventions for optimized wasm releases.
	`docs/WASM_RELEASES.md` now defines the intended output directory shape around the raw wasm artifact, compressed sidecars, and `wasm-release-manifest.json`, and `tools/build-wasm-release.ps1` emits that convention directly.
- [x] Add reproducible release-build guidance.
	`docs/WASM_RELEASES.md` now defines the intended toolchain, flag, manifest, and hash-record expectations for reproducible release builds.
- [ ] Add startup-cost measurements for release artifacts.
	Track download size, decompression cost, compile or instantiate time, and first-interaction timing for representative builds so size work is tied to real user-facing outcomes.
- [ ] Add release smoke validation to the launcher.
	Let `gwc release` optionally run a minimal post-build verification pass that checks artifact presence, manifest consistency, wasm MIME assumptions, and boot-time smoke probes before declaring a release artifact ready.

### Build speed and optimization experiments

- [x] Establish an experiment matrix for wasm build optimization.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended comparison matrix across plain, release-style, stripped, compressed, and post-processed wasm build variants.
- [x] Add scripted measurement of cold and warm build times.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended cold-build, warm-build, and small-edit rebuild timing set and the measurement rules for repeatable build-speed experiments.
- [x] Add representative multi-target coverage for build experiments.
	`docs/BUILD_EXPERIMENTS.md` now defines the canonical small (`./examples/21-ui-render`), routed mid-sized (`./examples/56-browser-router`), and large showcase (`./examples/86-atlas-commerce-os/client`) wasm targets, and `tools/wasm-build-experiment-targets.json` provides the same list for tooling.
- [x] Add experiment harnesses for wasm size and startup tradeoffs.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended harness inputs for raw size, compressed size, download cost, instantiate time, and first-interaction timing.
- [x] Evaluate Go build flags for size versus speed tradeoffs.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended tradeoff table for evaluating build flags across size, build speed, startup cost, runtime impact, and debugging cost.
- [x] Evaluate `GOWASM` feature toggles and compatibility tradeoffs where relevant.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended policy for recording and evaluating `GOWASM` toggles before standardizing on any non-default release setting.
- [x] Attribute build-loop timing by phase.
	`tools/measure-wasm-build.ps1` now emits phase-attributed JSON manifests with `go_build_ms`, compression timings, optional `serve_reload_ms`, total wall-clock timing, and artifact metadata, and `docs/BUILD_EXPERIMENTS.md` defines that measurement shape.
- [x] Evaluate post-processing and compression combinations.
	`tools/compare-wasm-compression.ps1` now compares plain, stripped, optimized, gzip, and brotli variants, using a Node-based Brotli fallback and `npx --package binaryen wasm-opt` when a direct optimizer install is absent, and `docs/BUILD_EXPERIMENTS.md` records that those variants are now measured before any release-default decision is made.
- [x] Track Go toolchain upgrade regressions for wasm builds explicitly.
	`tools/compare-wasm-go-toolchain.ps1` now runs the same wasm target through explicit baseline and candidate Go executables, reuses `tools/measure-wasm-build.ps1` for per-toolchain manifests, compares those manifests with `tools/compare-wasm-experiment.ps1`, and records the compared versions in `wasm-toolchain-comparison.json`.
- [x] Measure cache-strategy effects on build experiments.
	`tools/compare-wasm-build-cache.ps1` now records shared-cache cold, warm, and small-edit rebuilds together with isolated build-cache and CI-style cold, warm, and small-edit runs, and `docs/BUILD_EXPERIMENTS.md` defines that cache-topology comparison shape.
- [x] Add CI-friendly benchmark comparison for build experiments.
	`tools/compare-wasm-experiment.ps1` now compares saved wasm experiment manifests, applies configurable timing and size regression thresholds, emits a machine-readable comparison summary, and exits non-zero when a candidate exceeds the configured budget.
- [x] Record accepted and rejected build optimizations in docs.
	`docs/BUILD_EXPERIMENTS.md` now records the current accepted release baseline, accepted gzip delivery sidecar, rejected plain release default, rejected use of CI-cold timings as inner-loop guidance, and the not-yet-accepted `wasm-opt` and Brotli paths.

### Developer workflow and project bootstrap

- [x] Define the official way to start a new GoWebComponents app.
	`docs/ONBOARDING.md` now defines the current official starting path as a documented manual flow based on maintained examples and the public package surface until first-party starters exist.
- [ ] Add `tools/gwc` as the canonical launcher entrypoint.
	Make `go run ./tools/gwc <command>` the one documented CLI surface for scaffolding, development, testing, build, release, diagnostics, and example browsing.
- [x] Add a preset-first `gwc start` TUI.
	`go run ./tools/gwc start` now launches an interactive Bubble Tea scaffold flow that begins with a preset picker, captures project metadata, and offers an immediate post-generation `gwc dev` path for the generated app.
- [ ] Add feature-matrix scaffold generation.
	Generate scaffold files from selected capabilities such as router, SSR, forms, fetch, state, devtools, hot reload, and browser tests so starters stay composable instead of being hard-coded starter copies.
- [x] Add scaffold preset definitions for the main adoption modes.
	The `gwc start` preset set now includes `minimal-client`, `routed-spa`, `ssr-app`, and `reference-app`, giving the launcher built-in adoption-mode choices instead of one generic starter.
- [ ] Add starter output rules that keep generated apps disposable.
	Ensure scaffolded code stays small, readable, conventionally organized, and easy to delete or rewrite instead of generating repo-internal scaffolding that is hard to own afterward.
- [x] Add launcher-owned project metadata for generated apps.
	Generated starters now write `gwc-start.json` with preset metadata plus launcher-owned tooling defaults for app path, HTML shell, wasm output, dev host or port, and release settings so later commands can resolve project intent consistently.
- [ ] Add at least one maintained starter application for the common path.
	Ship a polished baseline app that includes routing, async data, state, forms, testing, and production-minded build setup so teams can begin from a realistic foundation instead of a toy counter example.
- [ ] Add starter variants for the major adoption modes.
	Provide clearly scoped starting points for client-only apps, SSR-enabled apps, and content or dashboard-style apps so teams can choose an architecture without reverse-engineering the examples directory.
- [x] Define upgrade and template-sync guidance for starter-based apps.
	`docs/ONBOARDING.md` now defines the intended starter-upgrade model around starter-specific release notes, core migration docs, and explicit support windows instead of blind monorepo diffing.
- [ ] Add a one-command local bootstrap workflow.
	Make it easy to install prerequisites, build wasm, start the dev server, and open a working example or starter app without several undocumented manual steps.
- [ ] Add scaffold-time prerequisite checks and optional setup steps.
	Let `gwc start` and `gwc doctor` verify Go, browser-test dependencies, and runtime assets early, then optionally run project initialization steps such as `go mod tidy` and first-build asset copying after scaffold generation.
- [x] Document environment prerequisites and platform expectations clearly.
	`docs/ONBOARDING.md` now defines the intended Go, Node, browser, and Windows/macOS/Linux baseline in one place and points to the browser support contract where relevant.
- [x] Add a â€œchoose your pathâ€ onboarding flow for new adopters.
	`docs/ONBOARDING.md` now defines the intended path chooser for client-rendered, routed, SSR, forms-heavy, and static/prerender-oriented adoption modes.

### Build feedback loop ergonomics

- [x] Define the recommended inner-loop workflow for application authors.
	`docs/ONBOARDING.md` now defines the intended edit-build-refresh loop, the current recommended commands, and the browser-refresh versus stale-wasm reasoning model for local application work.
- [x] Add `gwc dev` as the canonical app development command.
	`go run ./tools/gwc dev` now resolves the current app or scaffold metadata, prints the dev plan, and runs the maintained livereload-based serve, watch, rebuild, and hot-reload loop through one launcher command.
- [x] Add project detection for app mode and entrypoints.
	The dev resolver now prefers `gwc-start.json` metadata when present, then falls back to detected app entrypoints such as `main.go` or `cmd/web/main.go`, inferred roots, and detected HTML shells for hand-built projects.
- [x] Add dev-server mode selection for client-only versus SSR apps.
	The launcher now classifies dev plans as `client-only-wasm` or `server-app` and reports the corresponding server mode as `livereload-wasm` or `server-entrypoint` before execution.
- [x] Port the examples catalog server to Go.
	Replace the current Node example browser server with `gwc examples`, preserving catalog browsing, generated example listing, no-cache behavior, wasm MIME handling, and health checks.
- [x] Add example filtering and search to `gwc examples`.
	Support tag- or keyword-based browsing for examples so the Go examples server becomes a real discovery tool rather than only a static redirector.
- [x] Add resolved-plan output before launcher execution.
	Print the detected project root, app mode, wasm entry, HTML shell, output path, server mode, hot-reload state, and listening URL before the dev server starts so launcher behavior is auditable instead of implicit.
- [x] Add a simplified watch-mode entry point for apps.
	`gwc dev` now acts as the supported watch-mode entrypoint for generated and hand-built apps, forwarding one resolved plan into the maintained rebuild-and-serve loop instead of requiring several separate scripts.
- [ ] Improve incremental rebuild behavior for small source edits.
	Measure and reduce avoidable rebuild work so common component, route, and style changes do not feel disproportionately expensive during local development.
- [ ] Add clearer surfacing for build progress and failure states.
	Show whether the system is compiling, serving stale output, waiting on a reload, or blocked on an error so developers are not left guessing which part of the toolchain failed.
- [ ] Add launcher-visible dev status endpoints and summaries.
	Expose current mode, last successful build, current error state, hot-reload eligibility, and listening URLs through both human-readable output and a machine-readable status endpoint for tooling and editor integration.
- [ ] Add an interactive `gwc dev` status TUI.
	Provide one optional live surface for current resolved plan, compile state, stale-output state, last successful build time, latest failure, hot-reload eligibility, listening URLs, and recovery hints so `gwc dev` feels like a product command rather than a thin wrapper around terminal logs.
- [x] Add recovery guidance for broken local development loops.
	`docs/TROUBLESHOOTING.md`, `docs/ONBOARDING.md`, and `tools/README.md` now document the current recovery path for stale wasm output, broken example serving, missing runtime bootstrap files, and local build-versus-served-output drift in the repo's supported manual dev loop.
- [ ] Add example workflows for repo contributors versus framework consumers.
	Differentiate the commands and expectations for working on the framework itself versus building an app on top of it so the tooling story scales beyond this repository.

### IDE and editor integration

- [ ] Define the first-party IDE support boundary.
	Decide whether editor support lives as VS Code-focused tooling first, gopls-compatible conventions plus snippets, or a broader extension strategy so the public DevX story does not imply richer IDE support than actually exists.
- [ ] Add launcher-owned task and problem-matcher definitions for the supported editor workflow.
	Provide first-party task definitions for `gwc dev`, `gwc test`, `gwc verify`, `gwc doctor`, and future audit commands so teams can adopt the documented workflow in editors without hand-writing their own local shell wrappers.
- [ ] Add snippets, boilerplates, and hover-friendly discoverability for the public surface.
	Make hooks, typed HTML helpers, router APIs, SSR helpers, and testing entrypoints easier to discover through editor snippets, concise inline docs, and example-backed symbol descriptions.
- [ ] Add focused snippets for routed apps, SSR bootstrap, forms, and async resources.
	Ship snippets for route registration, loader and revalidation wiring, SSR bootstrap registration, typed form handlers, and cache-backed resource patterns so discoverability reinforces the recommended app architecture instead of generic boilerplate.
- [ ] Add hover docs that explain recommended usage, not only signatures.
	Make symbol help describe when to use hooks, atoms, loaders, form helpers, cache helpers, hydration entrypoints, and devtools surfaces so editor discovery teaches the framework's intended workflow.
- [ ] Add editor-visible project diagnostics and task integration.
	Surface missing `wasm_exec.js`, broken scaffold metadata, unresolved dev-server entrypoints, missing browser-test prerequisites, and launcher status through IDE-friendly diagnostics instead of only terminal output.
- [ ] Project scaffold, launcher, and audit diagnostics into editor problems.
	Map `gwc doctor`, future golden-path auditor results, scaffold metadata failures, and release-budget or startup-budget findings into editor diagnostics with severity, remediation text, and stable codes so issues are fixable from the editing surface.
- [ ] Evaluate navigation and refactor support for routes, typed HTML builders, and scaffolded apps.
	Decide how far the project should go on go-to-definition, rename safety, route-path references, template or starter upgrades, and other editor-assisted refactors for Go-first frontend codebases.
- [ ] Add route-symbol indexing and reference discovery.
	Support go-to-definition and find-references for route paths, layout chains, loader ownership, and scaffolded entrypoints so larger apps are navigable without grep-driven archaeology.
- [ ] Add lightweight code actions for common framework fixes.
	Offer quick fixes for missing scaffold metadata, missing editor tasks, baseline test generation, and known auditor suppressions so editor integration has corrective value rather than only passive warnings.

### HTML authoring sugar and shorthand ergonomics

- [x] Define the product boundary for additive HTML authoring sugar.
	Decide and document that the shorthand layer stays plain Go, composes on top of `html` and `ui`, does not replace the explicit builder surface, and does not introduce JSX, hidden reactivity, or a second runtime ownership model.
- [x] Decide where the shorthand surface lives.
	Choose whether the first pass lives in `html`, in a dedicated additive package, or as a narrowly documented dot-import style over `html`, and record the support tier before any new public API is added.
- [x] Define the accepted mixed-child contract for the shorthand layer.
	Specify exactly which child inputs are supported, such as `ui.Node`, `string`, `fmt.Stringer`, nested `[]ui.Node`, nested `[]string`, and mixed `[]interface{}`, so variadic sugar behavior is fixed before helpers are implemented.
- [x] Define nil and scalar child handling rules.
	Decide whether `nil` children are silently skipped, whether non-string scalar values are rejected or stringified, and whether the shorthand layer is allowed to use `fmt.Sprint(...)` as a last-resort fallback.
- [x] Add one shared child-normalization helper for the sugar path.
	Implement a single internal normalization pass that flattens nested children, auto-wraps accepted text-like inputs into text nodes, preserves order, and avoids each shorthand builder reimplementing its own child parsing.
- [x] Add tests for child normalization behavior.
	Cover mixed strings and nodes, nested child slices, nil entries, empty input, and deterministic output ordering so sugar builders do not drift subtly over time.
- [x] Defer automatic string-to-text lifting inside existing shorthand builders until a compatibility-safe migration exists.
	Record that changing the existing typed builder signatures to mixed variadic inputs would break common `[]ui.Node` expansion callsites, so the non-breaking first pass keeps explicit builders plus `Text(...)` and `Children(...)` instead.
- [x] Ship compatibility-safe mixed-input host tags in a companion `html/shorthand` package.
	Keep the stable typed `html.Div(...)` builder family unchanged, and add the reopened second-pass mixed-argument call shape through `html/shorthand.Div(...)`, `Button(...)`, and the rest of the selected primitive tag set.
- [x] Add SSR parity tests for auto-lifted text children in the companion shorthand package.
	Validate that `html/shorthand` mixed string children serialize identically to equivalent explicit `html` trees once auto-lifted host tags actually exist.
- [x] Define the first-pass shorthand tag set.
	Choose the initial supported subset of common tags such as `Div`, `Span`, `Button`, `Input`, `Label`, `Form`, `P`, `Pre`, `Code`, `Section`, `Ul`, `Li`, `H1`, `H2`, `H3`, `Img`, `Select`, and `Option` instead of mirroring every HTML builder immediately.
- [x] Keep existing typed builders as the first-pass host-tag entrypoints instead of adding parallel shorthand wrapper names.
	Use the already-exported package-level builders such as `Div`, `Button`, `Input`, and `Option` together with `PropsOf(...)`, `Children(...)`, and text helpers in the first pass, while the reopened second pass keeps mixed-input wrappers isolated in `html/shorthand` instead of replacing the typed `html` API.
- [x] Define the shorthand call-shape for tags.
	Decide whether shorthand tag functions accept only mixed variadic arguments, whether options and children are parsed from one shared argument list, and how empty calls or zero-child host nodes are represented.
- [x] Define the option model for shorthand DOM props.
	Choose the representation for additive props such as a marker interface, typed option structs, or another explicit value shape so shorthand tags can stay ergonomic without becoming reflection-heavy or ambiguous.
- [x] Add shorthand helpers for the highest-value common props.
	Implement the first set of option helpers for `Class`, `ID`, `Title`, `Value`, `Placeholder`, `Type`, `Href`, `Src`, `Disabled`, `Checked`, `Selected`, `Required`, `ReadOnly`, `AutoFocus`, `Style`, `Data`, and `Aria` so primitive DOM authoring can move away from repetitive explicit prop structs where appropriate.
- [x] Define duplicate-option precedence rules.
	Document whether repeated shorthand options use last-write-wins semantics or another explicit rule so option parsing stays deterministic and reviewable.
- [x] Add tests for shorthand option parsing and precedence.
	Verify that shorthand options map cleanly onto `html.Props`, that duplicate options resolve consistently, and that omitted zero values do not accidentally emit extra props.
- [x] Add shorthand event option helpers that reuse `ui.UseEvent(...)`.
	Implement helpers such as `OnClick`, `OnInput`, `OnChange`, `OnSubmit`, `OnKeyDown`, `OnKeyUp`, `OnFocus`, and `OnBlur` that stay on the existing handler path instead of creating a second event abstraction.
- [x] Support both zero-argument and typed-event callback signatures through shorthand event helpers.
	Ensure shorthand event helpers can accept the already-supported `func()` and typed event callback forms cleanly so authoring is lighter without changing runtime behavior.
- [x] Add tests for shorthand event helper behavior.
	Cover zero-argument handlers, typed event handlers, stable prop emission, and browser event execution so event sugar remains a pure surface-level convenience.
- [x] Add a `Textf(...)` helper for formatted text nodes.
	Provide a small formatting helper that reduces repetitive `fmt.Sprintf(...)` plus `html.Text(...)` pairs while keeping the underlying output equal to an ordinary text node.
- [x] Add tests for `Textf(...)`.
	Verify formatting behavior across empty strings, multiple arguments, numeric formatting, and equivalent explicit output so the helper is easy to trust.
- [x] Evaluate whether `TextIf(...)` belongs in the first shorthand pass.
	Decide whether a small conditional text helper materially improves compact status and label rendering over `If(...)` plus string auto-lifting or whether it only duplicates existing composition patterns.
- [x] Add tests for `TextIf(...)` if that helper ships.
	Cover true and false conditions, empty strings, SSR parity, and interaction with auto-text normalization so conditional text sugar remains straightforward.
- [x] Add a `When(...)` helper for conditional class fragments.
	Provide a simple conditional-string helper that lets class composition stay readable without open-coded string concatenation for common boolean cases.
- [x] Add a `ClassNames(...)` helper.
	Join class fragments, drop empty values, normalize whitespace boundaries, and support a mix of raw strings plus conditional fragments so class composition feels modern without hiding actual class output.
- [x] Define whether `ClassNames(...)` flattens nested class-part slices.
	Specify whether nested `[]string`, grouped helper results, or other class-part collections are recursively flattened so callers do not need a second normalization step when composing reusable class fragments.
- [x] Evaluate whether `ClassIf(...)` adds enough value beyond `When(...)`.
	Decide whether a dedicated boolean-to-class helper materially improves readability or merely duplicates `When(...)` under another name.
- [x] Add tests for `When(...)` and `ClassNames(...)`.
	Cover empty fragments, repeated spaces, mixed truthy and falsey fragments, and deterministic join behavior so styling helpers remain mechanical and unsurprising.
- [x] Add `If(...)` for inline conditional node emission.
	Support the common case where a subtree should render only when a condition is true, without forcing authors to allocate temporary slices or wrapper nodes for small inline decisions.
- [x] Add `IfElse(...)` for inline branch selection.
	Support the common case where a component chooses between two sibling node shapes inline, while keeping the output on the normal `ui.Node` path rather than inventing an expression-specific runtime contract.
- [x] Add `Unless(...)` as the inverse conditional helper.
	Support the common case where a subtree should render only when a predicate is false so small fallback or empty-state branches do not require manual boolean negation at every callsite.
- [x] Evaluate whether switch-style expression helpers belong in the sugar layer.
	Decide whether `Switch(...)`, `Case(...)`, and `Default(...)` materially improve authored tree readability over ordinary Go branching or whether they push the shorthand API too far toward a custom DSL.
- [x] Add switch-style helpers for second-pass inline branch selection.
	Ship `Switch(...)`, `Case(...)`, and `Default(...)` with first-match-wins semantics and explicit fallback behavior.
- [x] Define false-branch semantics for conditional helpers.
	Decide whether a false condition returns `nil`, an empty fragment, or another explicit zero-node contract so conditional helpers integrate predictably with existing child flattening rules.
- [x] Add tests for `If(...)` and `IfElse(...)`.
	Verify nil handling, empty-branch behavior, SSR output parity, and composition with mixed shorthand children so conditional helpers do not accidentally emit wrapper markup.
- [x] Evaluate whether optional-value helpers belong in the first shorthand pass.
	Decide whether helpers such as `Maybe(...)`, `OrElse(...)`, and `Coalesce(...)` are important enough to be part of the initial authoring story or whether they should remain a later convenience layer to avoid turning `ui` into a general utility bag.
- [x] Add pointer-based optional-value helpers for second-pass sugar.
	Ship `Maybe(...)`, `OrElse(...)`, and `Coalesce(...)` with pointer-only absence semantics so missing values stay explicit and zero values are not overloaded.
- [x] Add a generic `Map(...)` helper for slice-to-node expansion.
	Provide a small helper that expands typed slices into `[]ui.Node` while preserving order and avoiding repetitive `make([]ui.Node, 0, len(items))` boilerplate in DOM-heavy list rendering.
- [x] Decide whether first-pass `Map(...)` also supports index-aware callbacks.
	Choose whether the initial API is `func(T) ui.Node` only or whether `func(T, int) ui.Node` belongs in the first pass so list helpers stay minimal and stable.
- [x] Add tests for `Map(...)`.
	Cover empty slices, single-item slices, order preservation, and direct variadic composition into parent shorthand tags so list sugar remains predictable.
- [x] Add a truthful `MapKeyed(...)` helper with explicit key propagation.
	Ship keyed list sugar only by explicitly writing computed keys onto each realized node so reconciliation behavior remains reviewable instead of cosmetic.
- [x] Evaluate whether higher-order collection helpers belong in the shorthand surface.
	Decide whether `FlatMap(...)`, `FilterMap(...)`, and `Join(...)` materially reduce real list-composition boilerplate in examples or whether they expand the sugar layer too quickly for the first release.
- [x] Add higher-order collection helpers for second-pass shorthand composition.
	Ship `FlatMap(...)`, `FilterMap(...)`, and `Join(...)` with explicit flattening, filtering, and separator behavior so denser tree construction stays mechanical.
- [x] Decide whether the shorthand layer should expose `Fragment` and a short alias.
	Evaluate whether to re-expose `Fragment` only, add a short alias such as `F`, or deliberately avoid the alias to keep the API from drifting into DSL-style shorthand.
- [x] Define first-class void-tag shorthand expectations.
	Ensure helpers for childless tags such as `Br()`, `Hr()`, `Img(...)`, and `Input(...)` remain cheap to call, require no placeholder children, and compose cleanly with the same shorthand option model as non-void host nodes.
- [x] Add tests for void-tag shorthand behavior.
	Cover empty child lists, attr emission, SSR serialization, and equivalence with explicit `html` builders so void tags stay simple and unsurprising.
- [x] Evaluate whether buttons and links need dedicated content-normalization sugar.
	Decide whether common cases such as `Button("Save")`, `Button(icon, "Save")`, and string-heavy link content are already solved by the general normalization model or whether these heavily-used primitives need additional shorthand conveniences.
- [x] Evaluate whether anchor shorthand should support positional destination sugar.
	Decide whether forms like `A("/settings", "Settings")` are worth supporting or whether explicit `Href(...)` plus normalized children keeps the argument contract clearer and more Go-like.
- [x] Do not add button or link normalization shortcut tests for the first pass.
	The first pass relies on the general child normalization path and explicit `Href(...)` usage rather than shipping primitive-specific shortcut forms that need separate coverage.
- [x] Decide whether the shorthand layer needs explicit attribute-builder helpers beyond typed prop options.
	Evaluate whether `Attr(...)`, `Attrs(...)`, `Data(...)`, and `Aria(...)` materially improve readability over typed options alone, especially for uncommon or mixed attribute-heavy host nodes.
- [x] If attribute-builder helpers ship, define how they merge with typed options.
	Specify precedence between `Attr(...)` and dedicated helpers like `Class(...)` or `ID(...)`, how repeated `Attrs(...)` groups merge, and how raw attributes interact with typed boolean props so the shorthand layer does not hide conflicts.
- [x] Add tests for attribute-builder helper behavior if that API ships.
	Cover raw attribute emission, merge precedence, duplicate keys, mixed `Data(...)` and `Aria(...)` usage, and parity with explicit `html.Props` output.
- [x] Add shorthand helpers for `Role(...)` and `TabIndex(...)` if the first attr set expands.
	Treat landmark, dialog, listbox, and keyboard-navigation attributes as common enough to justify first-class helpers once the baseline attr sugar surface is stable.
- [x] Evaluate conditional boolean attr helpers for common host props.
	Decide whether helpers such as `DisabledIf(...)`, `ReadOnlyIf(...)`, and `SelectedIf(...)` improve tree readability enough to justify dedicated exports beyond the base boolean prop helpers.
- [x] Add tests for conditional boolean attr helpers if they ship.
	Cover true and false emission, merge precedence with direct boolean options, and parity with explicit boolean props so conditional attr sugar remains predictable.
- [x] Evaluate whether `StyleMap(...)` should be a dedicated shorthand helper.
	Decide whether a named `StyleMap(...)` helper improves readability enough over `Style(...)` with a literal map to justify a separate public API.
- [x] Do not add `StyleMap(...)` as a first-pass helper.
	`Style(...)` already covers the intended map-based usage, so no separate helper or test track is needed.
- [x] Add simple event-wrapper helpers for common browser-control flows.
	Evaluate and, if appropriate, implement wrappers such as `Prevent(...)` and `Stop(...)` that compose with existing event helpers and reduce repetitive `PreventDefault()` and `StopPropagation()` boilerplate without hiding handler ownership.
- [x] Define how simple event wrappers interact with zero-argument and typed-event callbacks.
	Specify whether wrappers synthesize an event-aware adapter around `func()` callbacks, whether typed-event callbacks remain unchanged, and how wrapper composition order behaves so event helpers are stackable but still explicit.
- [x] Add tests for simple event-wrapper helpers if they ship.
	Cover prevent-default behavior, stop-propagation behavior, zero-argument callbacks, typed-event callbacks, and nested wrapper composition so the helpers remain transparent.
- [x] Evaluate whether temporal event helpers belong in the first-party sugar surface.
	Decide whether `Debounce(...)` and `Throttle(...)` should ship as first-party event sugar, remain separate scheduling helpers, or stay deferred until the cleanup, timing, and value-extraction contract is more mature.
- [x] Add temporal event helpers for second-pass event sugar.
	Ship `Debounce(...)` and `Throttle(...)` with explicit closure-scoped scheduling semantics rather than leaving them as permanently deferred placeholders.
- [x] Define the shorthand split between primitive option-style components and business components.
	Document that primitive DOM-like components may accept option-style sugar such as `Variant(...)` or `Size(...)`, while app-level or business components should continue to use typed props structs for clarity and long-term maintainability.
- [x] Evaluate whether primitive visual options such as `Variant(...)` and `Size(...)` belong in core shorthand.
	Decide whether these option helpers should be limited to a small first-party primitive set or left to application and design-system packages so the framework does not overclaim a built-in visual component model.
- [x] Leave primitive visual option helpers out of the first-pass sugar surface.
	Do not add framework-level `Variant(...)` or `Size(...)` options to generic html sugar, so no helper-specific tests are needed.
- [x] Decide whether the shorthand layer should expose a `Memo(...)` alias distinct from the existing hook names.
	Evaluate whether a standalone `Memo(...)` concept genuinely improves ergonomics or whether it would duplicate `UseMemo(...)` semantics and increase naming surface without enough value.
- [x] Add a documented happy-path import recommendation for the shorthand surface.
	Choose whether the preferred authored form is `ui.*`, dot-imported shorthand helpers, or an additive package alias, and document the tradeoffs honestly instead of implying a no-prefix style that Go does not naturally guarantee.
- [x] Evaluate whether small app-level convenience helpers belong in core shorthand or companion packages.
	Classify candidates such as `Plural(...)`, `Ellipsis(...)`, and `TestID(...)` as core sugar, companion utilities, or app-level helpers so the framework does not quietly accumulate unrelated generic utilities under the same authoring layer.
- [x] Leave small app-level convenience helpers out of the first-pass sugar surface.
	Keep app-specific conveniences such as pluralization, truncation, and test-id helpers out of core html sugar rather than defining contracts and tests for them here.
- [x] Separate router-aware sugar candidates from generic DOM sugar.
	Treat helpers such as `LinkTo(...)`, `IsActiveRoute(...)`, and route-matching conveniences as router-surface decisions rather than silently folding them into generic HTML shorthand work.
- [x] Separate resource-state sugar candidates from generic DOM sugar.
	Treat helpers such as `Match(resource, ...)` as fetch or async-boundary ergonomics that should be evaluated with the broader resource API instead of being bundled uncritically into the base HTML authoring layer.
- [x] Separate binding-style sugar candidates from first-pass shorthand work.
	Treat helpers such as `BindValue(...)` and `BindChecked(...)` as higher-magic form abstractions that need a dedicated decision on ownership, conversion rules, and event semantics before they are allowed into the public API.
- [x] Pilot the shorthand layer on a small set of examples.
	The shorthand surface now has real authored validation beyond package tests: `examples/03-toggle` exercises the reopened second-pass `html/shorthand` path directly, and `examples/02-text-input` covers the form-heavy input path. A dedicated list-heavy shorthand port remains optional follow-up rather than a blocker for the shipped HTML sugar surface.
- [x] Add parity tests that compare shorthand output to explicit `html` output.
	Use representative host trees to confirm that shorthand-built nodes serialize and render identically to equivalent explicit `html` builders so the sugar layer can stay a mechanical facade.
- [x] Add authoring docs for the shorthand layer.
	Document the intended relationship between `ui`, `html`, and the shorthand helpers, show side-by-side examples, explain when to prefer the explicit builder layer, and state clearly which ergonomics are intentionally deferred.
- [x] Add adoption and migration guidance for shorthand usage.
	Show how teams can mix shorthand helpers with explicit `html` builders safely, when dot-import is acceptable, which identifiers are likely to collide in larger packages, and how to keep business components on typed props structs while using shorthand primarily for primitive DOM composition.
- [x] Classify the shorthand layer in the API policy before broad rollout.
	Mark the first shipped shorthand surface as stable, supported companion, or experimental and update the package docs and migration guidance so additive authoring sugar does not arrive as unlabeled permanent public API.

### Launcher testing, verification, and diagnostics

- [x] Add `gwc test` with explicit test lanes.
	`go run ./tools/gwc test` now supports explicit `unit`, `wasm`, `hydration`, `browser`, and `release` lanes, defaults to `unit` plus `wasm`, exposes repeatable `-lane` flags with JSON summaries, and reuses the repo js/wasm executor, Playwright workspace, and release smoke-build path so launcher-driven validation no longer depends on ad hoc command memorization.
- [x] Add `gwc verify` as a CI-oriented aggregate command.
	`go run ./tools/gwc verify` now resolves the target app through the launcher, runs app-local `go test ./...` when `_test.go` files exist under the resolved project root, and then performs a `ci`-profile js/wasm build with both human-readable and JSON output so CI can use one documented baseline entrypoint.
- [x] Add `gwc doctor` for environment and project diagnostics.
	`go run ./tools/gwc doctor` now checks Go, Node.js, npm, `wasm_exec.js`, Playwright install state, scaffold metadata, current-directory project-detection signals, and requested port availability, with both human-readable and JSON output.
- [ ] Extend `gwc doctor` into a golden-path app auditor.
	Move beyond environment checks by letting the launcher inspect a target app for framework-level architecture guidance such as duplicated state, server-only code leaking into client paths, route or page shapes that should be prerendered, interactions that should defer behind `ui.Lazy` or route-level splitting, and mutation flows that are missing retry or error-handling strategy.
- [ ] Add static golden-path rules for state and ownership boundaries.
	Flag duplicated local-versus-shared state, server-only data leaking into browser-visible payloads, component trees with conflicting ownership across loaders, caches, forms, and atoms, and browser-only handles crossing SSR or worker boundaries.
- [ ] Add route-shape and delivery audit rules.
	Detect routes that look prerender-friendly, routes that should remain request-time SSR only, screens that over-hydrate above the fold, and route trees that should split shell versus feature work instead of paying one startup cost.
- [ ] Add mutation and resilience audit rules.
	Identify writes that lack retry strategy, idempotency hints, rollback semantics, offline replay posture, or conflict handling so serious product workflows are not shipped with only happy-path mutation behavior.
- [ ] Add suppressions, baselines, and policy levels for audit adoption.
	Support per-rule suppression, checked-in baselines, and warn-versus-error policy modes so teams can adopt the auditor incrementally instead of treating all historical architectural debt as an immediate hard failure.
- [ ] Add evidence-backed audit rules for startup-cost and ownership mistakes.
	Flag dependencies, imports, or startup wiring that unnecessarily bloat initial wasm startup, identify client bundles that should stay server-owned, and explain which rule fired with concrete file or symbol evidence instead of vague style warnings.
- [ ] Add runtime evidence collection for golden-path audits.
	Combine static rules with runtime observations such as payload sizes, hydration mismatch hotspots, lazy-boundary hit rates, route transfer sizes, and repeated replay failures so architectural warnings are backed by real app behavior instead of only source heuristics.
- [ ] Add machine-readable golden-path audit output and severity levels.
	Emit stable rule ids, severity, affected file locations, and remediation guidance so editor tooling, CI, and adoption reviews can consume the same architectural audit surface without scraping human-readable terminal output.
- [ ] Add CI and generated-workflow entrypoints for the golden-path auditor.
	Let `gwc verify`, scaffolded GitHub Actions, and editor task integrations run the same audit surface with stable exit semantics and severity filtering so architecture checks become part of ordinary delivery rather than an optional side command.
- [ ] Add scaffold-generated baseline tests keyed to selected features.
	Emit the smallest believable test set for chosen capabilities such as routing, SSR, forms, fetch, or hot reload so generated apps start with real verification instead of an empty test folder.
- [ ] Add starter-generated GitHub Actions workflows for the default CI path.
	Have generated standalone apps include functional `.github/workflows` definitions that install the toolchain, run the starter's documented test command, and perform the baseline build or verify flow so new teams get a working CI path on day one instead of reverse-engineering repo workflows.
- [ ] Verify scaffolded GitHub Actions against generated starters.
	Cover the emitted workflow files in scaffold golden tests and at least one end-to-end generated-app smoke path so starter CI does not silently drift from the commands and files that `gwc start` actually produces.
- [x] Make `gwc start` generate standalone apps in user-owned workspaces by default.
	The scaffold flow now targets a user-owned generated-project root outside the framework checkout by default, and the TUI explicitly describes the result as a standalone project.
- [ ] Split `gwc start` into standalone-app mode versus contributor-linked mode.
	Keep a deliberate framework-contributor path for local source linking, but make the default generated app behave like its own project with its own lifecycle.
- [ ] Remove the default scaffold dependency on the current framework checkout.
	Replace the current local `replace`-to-repo behavior with a real standalone dependency strategy so generated apps remain portable after the original GWC clone is moved or deleted.
- [x] Add explicit generated-project location policy for Windows, macOS, and Linux.
	The launcher now implements OS-aware generated-project defaults, using `Documents` on Windows and macOS, `Documents` or `Projects` fallback behavior on Linux, and override support through launcher config.
- [ ] Add scaffold metadata that records project ownership and framework source mode.
	Record whether the generated app is standalone, locally linked, vendored, or otherwise framework-coupled so later launcher commands can resolve dependencies and diagnostics coherently.
- [ ] Add metadata-first resolution tracing and fallback diagnostics for launcher commands.
	Show for `gwc dev`, `gwc build`, `gwc release`, `gwc test`, `gwc verify`, and `gwc doctor` whether the resolved app path, entrypoint, output path, profile, and port came from explicit flags, `gwc-start.json`, launcher config, or convention fallback so remaining heuristics stay visible and debuggable.
- [ ] Add metadata schema-versioning and migration validation for generated apps.
	Define how older or partially populated `gwc-start.json` files are upgraded, defaulted, warned on, or rejected so launcher evolution does not silently reinterpret generated-app intent across versions.
- [ ] Add optional bootstrap for app-local git initialization.
	Offer to initialize a fresh git repository for generated standalone apps so they start with their own version-control boundary instead of inheriting the framework repo context.
- [x] Add launcher integration tests for config precedence and project detection.
	`tools/gwc` now has focused tests for metadata-driven resolution, configured-app metadata discovery, convention fallback to `main.go` or `cmd/web/main.go`, and explicit flag overrides across `dev`, `build`, and `release`.
- [ ] Add golden tests for scaffold output combinations.
	Snapshot the files generated for key preset and feature combinations so starter evolution stays reviewable and does not drift silently.
- [ ] Add end-to-end launcher tests for dev, build, and release commands.
	Exercise representative generated apps through `gwc dev`, `gwc build`, `gwc test`, `gwc verify`, and `gwc release` so the launcher contract is validated as a product surface rather than only by unit tests.
- [ ] Add machine-readable diagnostics contracts for launcher failures.
	Standardize structured failure output for build, serve, test, verify, and release commands so editors, CI jobs, and enterprise wrappers can distinguish configuration problems from code failures.

#### Runner Config Centralization And Path Policy

- [ ] Define one canonical runner-config schema for Go launcher, JS test runner, and nested livereload tooling.
	Keep `gwc-runner.json` as the single external contract so artifact paths, helper binaries, browser workspaces, and livereload settings do not drift across implementations.
- [ ] Split launcher config code into schema, defaults, and resolver layers.
	Move away from a flat "magic strings in command handlers" model by keeping literal defaults in one place and computed repo-root-aware resolution logic in a separate shared layer.
- [ ] Route every launcher-owned path decision through shared resolvers.
	Ensure `dev`, `build`, `release`, `test`, `verify`, `doctor`, `import`, `examples`, and `start` all consume the same path-policy helpers instead of reconstructing fallbacks inline.
- [ ] Remove remaining repo-owned `tmp/` and `dist/` assumptions from launcher-adjacent tooling.
	Finish the `bin/` normalization pass so PowerShell helpers, Playwright outputs, manifests, and generated import artifacts all obey the same artifact-root policy.
- [ ] Make `artifactRoot` govern every launcher-owned output class consistently.
	Cover wasm builds, release directories, temporary import workdirs, experiment manifests, browser-result files, and any future generated diagnostics so enterprise overrides are complete rather than partial.
- [ ] Make browser-workspace, livereload-workspace, client-script, and wasm-exec resolution behavior identical across Go and Node entrypoints.
	Prevent cases where `gwc test`, `scripts/run-main-tests.mjs`, and standalone livereload each accept different overrides or silently fall back to different directories.
- [ ] Add explicit resolution tracing for runner-config-derived values.
	Show whether each resolved path came from CLI flags, project metadata, runner config, environment variables, or convention fallback so operators can debug why a command picked a specific workspace or artifact directory.
- [ ] Add stable, structured diagnostics for invalid runner overrides.
	Return actionable failures for missing configured paths, wrong workspace shapes, unreadable config files, and invalid relative-path bases without raw panic-style output or silent skip behavior.
- [ ] Add config-discovery tests for local, environment, and home-directory runner config files.
	Lock down precedence between repo-local `gwc-runner.json`, `GWC_RUNNER_CONFIG`, and home-level defaults so enterprise setups remain predictable across shells and CI agents.
- [ ] Add end-to-end parity tests for runner-config path overrides.
	Exercise one representative override file through `gwc dev`, `gwc test`, `gwc doctor`, `gwc build`, `gwc release`, the Node runner, and nested `tools/livereload` so the shared contract is validated across process boundaries.
- [x] Publish a documented example runner-config file and field reference.
	`docs/examples/gwc-runner.example.json` now provides the copyable baseline, and `tools/README.md` now documents the current path fields and relative-path resolution semantics.
- [ ] Define organization-policy versus project-config ownership for runner settings.
	Clarify which overrides belong in enterprise-managed shared config, which belong in checked-in project config, and which should remain explicit per-command flags.

## 9. Strategic Direction and Experimental Work

### Fine-grained reactivity direction

- [x] Adopt fine-grained reactivity as an explicit performance direction.
	`docs/FRAMEWORK_SCOPE.md` now records the intended direction: reduce unnecessary full-component and page-level rerenders in high-frequency UI paths while keeping the framework hooks-compatible and fiber-based overall.
- [x] Define the first shipped boundary for narrower updates.
	`docs/FINE_GRAINED_REACTIVITY.md` now defines the first pass as explicit subscribed render regions inside the current component tree: stable anchored host regions may update narrowly, while hooks, props, context, routing, async boundaries, hydration recovery, and other structural changes still fall back to normal component reconciliation.
- [x] Choose the initial primitive set for in-core adoption.
	`docs/FINE_GRAINED_REACTIVITY.md` now records the recommended first set: reuse `state.Atom[T]` and `state.Derived[T]`, add explicit subscribed-region support in `ui`, and prefer selector helpers over introducing a full standalone signal or effect runtime in the first pass.
- [x] Prototype fine-grained state reads on top of the existing `state` package.
	The first prototype now ships as a text-only narrow-update path: `state.Atom[T]` and `state.Derived[T]` can render reactive text nodes that subscribe by atom ID, mark only the text fiber dirty on updates, and avoid rerendering the owning component when the change is isolated to that text node.
- [x] Benchmark narrow updates against current keyed reconciliation paths.
	`internal/runtime/fine_grained_benchmark_test.go` now covers both the keyed dashboard hot-value case and the 64-region ancestor-rerender case: the keyed reactive text path remains materially cheaper than full component rerenders, and the latest ancestor-rerender pass brought stable reactive regions to practical parity with static leaves on the current Windows amd64 benchmark shape.
- [x] Define the mixed-model boundary between hooks and fine-grained subscriptions.
	`docs/FINE_GRAINED_REACTIVITY.md` now defines the ownership and fallback rules: hooks still own lifecycle, structure, props, context, effects, and local state semantics, while fine-grained subscriptions may bypass component rerenders only inside explicit subscribed regions with explicit reactive sources and a stable DOM anchor.
- [x] Define scheduling and batching semantics for fine-grained updates.
	`docs/FINE_GRAINED_REACTIVITY.md` now defines the first scheduling contract: fine-grained updates reuse the existing root scheduler and commit boundary, urgent writes coalesce by pass, transition updates defer before granularity is applied, hook-driven rerenders take precedence over narrow updates in the same subtree, and text-only narrow updates do not trigger a separate effect phase.
- [x] Add devtools support for fine-grained updates.
	Runtime inspection and the devtools snapshot now expose fine-grained fiber counts, granular dirty-mark and commit counters, descendant host and text commit counters for subscribed regions, per-node update origin, and the reactive source ID currently attached to a subscribed region so narrow updates are visible without guessing from generic dirty flags.
- [x] Add failure-mode tests for stale reads, update loops, and leaked subscriptions.
	`internal/runtime/fine_grained_reactivity_test.go` now covers source swaps without stale old-source updates, cyclic derived dependencies that report diagnostics instead of notifying fine-grained subscribers, and deleted subscribed subtrees that release atom subscriptions on unmount.
- [x] Add a first selector helper for projected fine-grained reads.
	`state.Select(...)` now projects read-only shared values from atoms or derived sources, and the runtime skips derived subscriber notifications when the projected value does not actually change so selector-backed hot paths can reduce churn instead of only renaming it.
- [x] Add a ui-level subscribed-region primitive and broaden narrow updates beyond text.
	`ui.ReactiveRegion(...)` now creates an explicit subscribed region fiber that rerenders only its own child subtree, enabling narrow host property and small anchored host subtree updates without rerendering the owning component.
- [x] Scope selector identity per hook instance instead of relying on globally shared caller strings.
	`state.Select(...)` now derives its internal shared-state identity from a stable hook-scoped ID, so reused selector labels no longer collide across component instances or separate subscribed regions.
- [x] Make transition deferral apply consistently across public shared-state update paths.
	Direct runtime atom writes and snapshot restores now honor `StartTransition(...)` by deferring their notifications through the same transition lane used by hook-backed state and atom setters.
- [ ] Add an explicit long-term policy for opt-in versus default fine-grained reactivity.
	Decide whether fine-grained updates remain permanently opt-in through selectors and reactive regions, or whether any future preset, package, or subsystem is allowed to make them the default authoring expectation for high-frequency UI surfaces.
- [ ] Add authoring guidance for choosing hooks versus selectors versus reactive regions.
	Publish concrete workload-driven rules for when normal component rerenders remain preferred, when `state.Select(...)` is enough, and when a subscribed region is justified so the non-default model is easy to adopt consistently.
- [ ] Add benchmark comparisons that defend the non-default stance explicitly.
	Measure hook-only, opt-in fine-grained, and signal-first-style hotspot workloads side by side so the project can explain why fine-grained support is H3 as an option but still L1 as the default programming model.
- [ ] Add first-class virtualization primitives for large lists and tables.
	Provide one supported answer for windowed rendering, stable row identity, measurement or overscan policy, and scroll restoration so data-heavy apps do not have to hand-roll large-collection performance patterns on top of the base reconciler.
- [ ] Decide whether virtualization belongs in core `ui`, a supported companion package, or an examples-first proving ground.
	Choose the ownership boundary before designing public APIs so large-list support does not arrive as an unlabeled permanent surface without a clear maintenance story.
- [ ] Define the first supported virtualization workload shapes.
	Decide whether the initial supported scope covers uniform-height vertical lists only, variable-height lists, tables and grids, nested grouped lists, or some smaller subset so the first release has honest constraints.
- [ ] Define the virtualization API shape.
	Choose whether the public surface is a component, a hook, a render-helper pair, or a lower-level state plus measurement primitive so consumers can reason about list ownership and rendering boundaries clearly.
- [ ] Define row identity and key requirements for virtualized items.
	Specify how callers provide stable item identity, how virtualization interacts with reconciliation keys, and which misuse patterns should be documented or rejected so row reuse does not corrupt local row state.
- [ ] Define the viewport and scroll-container ownership model.
	Decide whether the primitive owns its own scroll container, can bind to an external scrolling parent, or supports both modes so larger app shells do not have to fight the virtualization contract.
- [ ] Define overscan policy and defaults.
	Choose how many extra rows render before and after the visible window, whether overscan is item-count-based or pixel-based, and how callers override it so the first implementation has a defensible jank-versus-work tradeoff.
- [ ] Define the measurement model for row size.
	Decide whether the first pass assumes fixed row height, supports measured variable row heights, or supports an estimated-plus-correction model so the public API does not overpromise variable-height behavior.
- [ ] If variable-height rows are supported, define measurement invalidation rules.
	Specify how row remeasurement is triggered after content changes, width changes, font loading, or async image load so scroll position and visible windows stay coherent over time.
- [ ] Define scroll restoration and anchor behavior.
	Specify how virtualization restores scroll after rerender, route transitions, hydration, and data refresh, and whether restoration is item-anchor-based, pixel-offset-based, or both.
- [ ] Define SSR and hydration behavior for virtualized lists.
	Decide whether SSR renders only the initial window, a fixed above-the-fold budget, or a non-virtualized fallback so hydration and initial HTML stay predictable instead of silently changing list semantics between server and browser.
- [ ] Define accessibility expectations for virtualized content.
	Document focus retention, roving navigation interactions, screen-reader row or count metadata, and how offscreen items should be represented so virtualization does not undermine the accessibility work already present elsewhere in the framework.
- [ ] Define table-specific behavior if tables are in scope for the first pass.
	Specify whether virtualization supports semantic `<table>` structures directly, requires div-based grids, or needs a dedicated table virtualization primitive so row and header semantics remain honest.
- [ ] Define interaction behavior for sticky headers, sticky columns, and grouped sections.
	Decide whether these patterns are explicitly unsupported in the first release, supported only for certain layout shapes, or part of the first-class contract so feature scope stays reviewable.
- [ ] Define integration guidance for local state, forms, and focus inside virtualized rows.
	Explain how row-local hook state behaves when rows unmount outside the visible window, when callers should externalize row state, and how form inputs or expanded rows preserve meaningful user work across virtualization boundaries.
- [ ] Define integration guidance for fine-grained reactivity versus virtualization.
	Clarify whether virtualization is expected to compose with `state.Select(...)` and `ui.ReactiveRegion(...)`, and which mechanism should own high-frequency row updates versus window-range updates in large data surfaces.
- [ ] Add the first low-level viewport state primitive if needed.
	Provide a minimal maintained way to track scroll offset, viewport size, and visible range so virtualization logic does not depend on every consumer hand-rolling browser measurement and resize bookkeeping.
- [ ] Add the first public virtualized list primitive for uniform-height rows.
	Ship one narrow, well-tested answer for large vertical lists before expanding into more complex variable-height or grid-style workloads.
- [ ] Add variable-height virtualization only after fixed-height behavior is correct and measured.
	Treat measured or estimated variable-height rows as a second phase so the first release does not inherit avoidable complexity in scroll correction and measurement churn.
- [ ] Add a virtualization-specific row renderer contract.
	Define exactly what row index, item value, item key, style or offset data, and visibility metadata the render callback receives so the primitive is expressive without leaking internal implementation details.
- [ ] Add virtualization-specific diagnostics for rendered range and overscan behavior.
	Expose current start and end indexes, rendered row count, overscan count, total item count, and visible viewport metrics so large-list behavior can be inspected without guessing.
- [ ] Add diagnostics for measurement churn and scroll correction.
	Record how often row sizes are measured or invalidated, how often scroll offsets are corrected after estimate mismatch, and whether measurement work clusters around specific interactions so variable-height support can be profiled honestly.
- [ ] Add diagnostics for row mount or unmount churn in virtualized surfaces.
	Track how many row fibers mount, unmount, or reuse across scroll movements so performance work can distinguish reconciler overhead from raw DOM work.
- [ ] Add virtualization-aware performance budgets.
	Define target ceilings for rendered-row count, scroll handler cost, measurement churn, dropped-frame signals, and row mount churn so future changes can be judged against repeatable budgets rather than subjective smoothness.
- [ ] Add a fixed-height feed example that proves the basic contract.
	Ship one realistic long feed or event log example that exercises scrolling, selection, and row reuse so virtualization is taught through an app-shaped workload rather than only through an abstract API demo.
- [ ] Add a data-table example if tables remain in scope.
	Use a wide, data-dense table or operator grid example to validate header behavior, keyboard flow, stable row identity, and scroll performance under enterprise-style list pressure.
- [ ] Add examples that show row-local state pitfalls and recommended ownership patterns.
	Demonstrate what happens when row-local hook state is lost on unmount and show the recommended pattern for preserving selection, expansion, edits, or draft form values outside the virtualized row lifecycle.
- [ ] Add browser benchmarks for large virtualized scroll surfaces.
	Measure fixed-height and, if supported, variable-height list scrolling with representative item counts so the project can compare virtualized rendering to full rendering under realistic row counts.
- [ ] Add regression tests for visible-range calculation.
	Cover empty lists, tiny lists, large lists, start-of-list, end-of-list, overscan boundaries, and resize-driven recalculation so window math does not drift silently.
- [ ] Add regression tests for stable row identity across scroll reuse.
	Verify that keyed rows preserve intended identity when items enter and leave the rendered window so row-local state does not bleed between unrelated records.
- [ ] Add regression tests for scroll restoration and anchor retention.
	Verify that route transitions, data refreshes, resorting, and container resizes preserve the documented restoration behavior instead of snapping users to the wrong offset.
- [ ] Add regression tests for hydration behavior on virtualized surfaces.
	Verify the documented SSR-to-browser contract for initial rendered range, placeholder height strategy, and post-hydration scroll correctness so virtualization does not destabilize hydration.
- [ ] Add regression tests for keyboard and focus behavior in virtualized lists.
	Verify focus retention, tabbability, active-row movement, and screen-reader-relevant metadata under row recycling and scroll movement so accessibility remains intact.
- [ ] Add virtualization examples, diagnostics, and performance budgets.
	Ship at least one realistic table or feed example plus profiling hooks that show rendered-row counts, measurement churn, and scroll-jank signals so virtualization can be validated as a production feature rather than an isolated helper.

### Compiler-assisted features

- [x] Decide whether compiler-driven ergonomics are a real product direction.
	`docs/COMPILER_ASSISTED_FEATURES.md` now records the project stance: compiler-assisted features stay opt-in and capability-specific, while plain Go plus ordinary `go build` remain the default product path rather than a required compiler-first workflow.
- [x] Evaluate whether compile-time reactivity is compatible with the current hook model.
	`docs/COMPILER_ASSISTED_FEATURES.md` now records the compatibility rule: compile-time reactivity is acceptable only as an opt-in lowering into explicit runtime primitives that preserve normal hook semantics, not as a second default authoring model with different lifecycle or scheduling rules.
- [x] Define source-language boundaries for compiler work.
	`docs/COMPILER_ASSISTED_FEATURES.md` now defines the boundary: the supported product path stays plain Go plus the ordinary toolchain, while generated Go helpers, template-lowering experiments, and browser-hosted tooling may exist only as opt-in experiments with inspectable output.
- [x] Add a migration and fallback plan for compiler-generated output.
	`docs/COMPILER_ASSISTED_FEATURES.md` now records the rollout rule: generated output must stay opt-in, inspectable, attributable, and reversible, with explicit opt-out paths and a documented non-generated fallback whenever compiler-assisted workflows touch supported product features.
- [ ] Add a bounded template- or JSX-authored experiment that lowers into inspectable Go.
	Prototype one clearly optional source-authoring experiment that compiles down to readable generated Go so the project can evaluate template-first ergonomics without silently changing the default runtime model or toolchain contract.
- [ ] Compare template-lowered authoring against plain Go on debugging and mixed-mode adoption.
	Measure code review readability, stack traces, generated diff noise, mixed-mode component composition, and migration cost before deciding whether template- or JSX-style authoring should remain experimental, graduate into a companion tool, or stay a non-goal.
- [ ] Publish a stable policy for template-first and compiler-first authoring.
	Make the comparisons chart defensible by documenting whether template- or JSX-first authoring and compiler-first optimization are permanent non-goals, long-term companion-package experiments, or gaps that may eventually move into the supported product story.

### State-preserving hot reload

- [x] Decide whether true in-place HMR is a goal beyond reload-and-restore.
	`docs/HOT_RELOAD.md` now records the implemented hot reload model: rebuild the bundle, replace the WASM module in page, clean up the old runtime resources, and restore shared plus migratable local state without forcing a full page reload.
- [x] Define state migration behavior for shape-changing component edits.
	`docs/HOT_RELOAD.md` now records the guarded migration policy: serializable hook slots can survive effect or callback shape changes and tail growth or shrink, while ambiguous serializable layout changes still remount safely.
- [x] Define preserve-versus-restart semantics for async and router-owned work.
	`docs/HOT_RELOAD.md` now records a restart-first policy for pending async work, router registrations, listeners, guards, loaders, and other closure-backed router internals, while still allowing atom snapshots and compatible serializable hook state to restore.
- [x] Improve hot reload diagnostics for dropped local state.
	The runtime now reports a hot-reload fallback warning with component path, component stack, and the specific remount reason when saved local state is discarded because identity, key, or hook shape no longer matches.
- [x] Decide whether callback or effect-owned resources should gain a first-class restore model.
	`docs/HOT_RELOAD.md` now records the implemented cleanup-plus-rebind model: callback closures are recreated from the new bundle, and effect-owned listeners, timers, observers, and wrapped handlers are cleaned up before reload and rebound afterward.

- [x] Decide whether state-preserving hot reload is a first-class development goal.
	State-preserving reload is now treated as an opt-in development aid: `hotreload.Enable()` installs the snapshot bridge, the live-reload client persists exported state through `sessionStorage`, runs a pre-reload cleanup hook, and restores shared state plus migratable serializable hook state while keeping DOM identity intact when the reload succeeds.
- [x] Audit which runtime assumptions block hot replacement.
	See `docs/HOT_RELOAD.md` for the current blocker list: component identity, ordered hook slots, global registries, route registration, effect cleanup, and JS handle ownership still require a reload-and-remount model rather than true module-level replacement.
- [x] Define the preservation boundary for local and shared state.
	See `docs/HOT_RELOAD.md` for the current boundary: shared atom values and compatible local hook state survive reloads through the snapshot bridge, while effect-owned side effects, router internals, async work, and incompatible DOM subtrees still remount or reset.
- [x] Add a development-time component identity and signature model.
	Runtime inspection now records ordered hook kinds plus component identity metadata, and devtools surfaces a compatibility signature that can tell shape-compatible edits from remount-required changes.
- [x] Add effect cleanup and re-run semantics for hot updates.
	`Runtime.RefreshEffectsForFiber(...)` now cleans stale effect cleanups and bumps an effect-generation counter so the next render reruns effects even when dependencies are unchanged.
- [x] Define error recovery behavior during failed hot updates.
	`GoLiveReload.triggerHotReload()` now records the current snapshot, falls back to a full reload on hot-update failure, and preserves the rollback snapshot so the next load can restore state cleanly.
- [x] Integrate the live reload server with state snapshot transport.
	Hot builds now request a snapshot from the client, carry the exported payload through `build_complete`, and let the browser reuse that snapshot on reload instead of relying only on a local-only fallback.
- [x] Add examples and benchmarks for preserved-state development flows.
	`test/specs/hot_reload_preserved_flow.spec.js` demonstrates counter, form, atom, and effect cleanup behavior across a hot reload, and `test/specs/hot_reload_preserved_flow_benchmark.spec.js` guards the snapshot round-trip timing.
- [x] Add explicit opt-in reset controls for intentional state invalidation.
	`hotreload.Configure(hotreload.Config{ResetKey: ...})` now stamps exported snapshots with a reset token and discards older snapshots when that token changes, giving developers a predictable way to force a clean restart after edits that should not preserve prior local or shared state.
- [x] Add better in-browser diagnostics for preserve-versus-reset decisions.
	The hot reload bridge now exposes the last restore outcome plus filtered hot-reload diagnostics, and the live-reload GWC panel shows whether a reload restored state, intentionally reset it because `ResetKey` changed, or remounted part of the tree with the concrete fallback reason.
- [x] Add a post-rebuild reload summary for the standalone dev server.
	The live-reload client now records a per-build summary with build duration, reload mode, restore result, and dropped-state reason or classification reason, and surfaces it in the popup, last-build panel section, and recent-build history.
- [x] Add first-class reset boundaries for subtree-scoped reload resets.
	`ui.HotReloadBoundary(...)` now provides an explicit keyed subtree wrapper so apps can intentionally remount one section by changing `ResetKeys`, without forcing a full app-wide `hotreload.Config.ResetKey` change.
- [x] Improve route and async visibility during hot reload.
	The hot reload bridge now exposes recent router and async activity, the runtime records explicit pending-fetch restart notices during hot reload prepare, and the live-reload panel shows the latest loader, guard, navigation, and fetch restart events after reloads.
- [ ] Make state-preserving hot reload the default supported path inside `gwc dev` for compatible apps.
	Integrate preserve-state reload into the canonical launcher workflow with clear automatic fallback to plain live reload when an app shape, browser target, or edit type is not compatible.
- [ ] Add pre-reload compatibility reporting for preserve versus remount decisions.
	Show developers before or during rebuild whether the incoming edit will preserve local state, remount a subtree, restart async work, or force a full reload so hot-reload behavior stops feeling heuristic.
- [ ] Add broader hot-reload coverage for routed, cached, and SSR-seeded apps.
	Extend preserved-state validation beyond isolated browser examples to routed apps with loader state, cached resources, async boundaries, and bootstrap-seeded data so the hot-reload score can move from M2 toward H3.

### HMR v2 and subtree patching

- [x] Introduce stable runtime-recognized component handles.
	`ui.CreateElement(...)` now emits cached `runtime.ComponentType` handles keyed by logical component identity, and runtime identity/signature comparisons understand those handles instead of treating the shared `ui.renderComponent` wrapper as the component itself.
- [x] Move fibers from raw implementation identity to patchable component definitions.
	Stable component handles now own the current implementation binding, `ui.CreateElement(...)` no longer stores the live component function in hidden props, and fibers render through persistent component definitions whose implementations can be rebound independently of fiber identity.
- [x] Add a changed-component manifest to the dev build loop.
	The dev server now emits a `*.hotreload-manifest.json` artifact next to the rebuilt WASM output and includes the same manifest in successful build payloads, listing changed source files and top-level component identities discovered from those files.
- [x] Use changed-component identities to drive selective preserve/remount decisions with compatibility fallback.
	After each module replacement, the hot-reload restore path now uses the changed-component manifest to preserve unchanged compatible component snapshots, remount changed subtrees intentionally, and fall back to the legacy full compatible restore path when selective matching is unsafe.

### Full-stack framework maturity

- [x] Decide whether GoWebComponents should remain a UI framework or grow a first-party app framework layer.
	`docs/FRAMEWORK_SCOPE.md` records the decision to keep core focused on the UI/runtime surface and leave app-framework conventions to docs, starters, or sibling packages.
- [x] Define a recommended project structure for production apps.
	`docs/PROJECT_STRUCTURE.md` captures the recommended `cmd/` + `internal/` + `web/` split for app entrypoints, SSR integration, assets, and test scaffolding.
- [x] Evaluate first-party code-splitting and bundle-loading conventions.
	`docs/CODE_SPLITTING.md` records the current recommendation: keep splitting convention-driven and app-owned, with `ui.Lazy`, boundaries, manifests, and preload/prefetch hints handled by tooling instead of a new core bundler API.
- [x] Define deployment targets and adapter expectations.
	`docs/DEPLOYMENT_TARGETS.md` records the supported deployment shapes and adapter expectations: static hosting, one Go SSR server, reverse-proxy or CDN fronted servers, and same-origin split SSR/API deployments.
- [x] Add an opinionated starter or reference app once conventions stabilize.
	`examples/86-atlas-commerce-os` is the current production-shaped reference app, and `docs/ONBOARDING.md` now names it as the integrated full-stack reference for larger apps.

### Code splitting and lazy bundle delivery

- [x] Define the code-splitting model for the framework.
	`docs/CODE_SPLITTING.md` now defines the intended hybrid model: route-driven splits are primary, component-level `ui.Lazy` boundaries are the exception, and build tooling owns physical chunk emission plus manifest-backed delivery.
- [x] Add a first-class lazy asset and module loading pipeline.
	`examples/90-browser-interop` now demonstrates the public module-loading path through `interop.ImportModule(...)`, a browser bridge at `browser-interop.html`, and a module-backed lazy-loading panel that resolves a deferred helper and renders its exports.
- [x] Define route-level code-splitting conventions.
	`docs/CODE_SPLITTING.md` now defines route-level splits as user-journey boundaries: shell code stays in the base bundle, major route families get their own chunks, and only heavy sibling or detail screens defer behind route-specific imports.
- [x] Define component-level chunk boundaries and loading semantics.
	`docs/CODE_SPLITTING.md` now defines component-level `ui.Lazy` boundaries as optional subtrees with explicit pending, error, retry, and hydration-aware behavior rather than hidden auto-splitting.
- [x] Add preload and prefetch hooks for deferred bundles.
	`html.Preload(...)`, `html.ModulePreload(...)`, `html.Prefetch(...)`, `html.Preconnect(...)`, and `html.DNSPrefetch(...)` now provide typed resource-hint helpers, and `docs/HEAD_MANAGEMENT.md` plus `docs/ASSETS.md` document when to use them for deferred bundles.
- [x] Define SSR interactions for split bundles.
	`docs/CODE_SPLITTING.md` now defines the SSR and prerender chunk contract: route output emits manifest-backed chunk declarations, critical chunks are preloaded with the shell, and hydration does not start from a missing shell dependency or template order assumption.
- [x] Add build-output and asset-manifest conventions for split bundles.
	`docs/ASSETS.md` now defines the split-bundle output shape: route entry chunks, shared shell chunks, and lazy feature chunks use logical manifest names, hashed filenames, and deploy together so HTML never relies on guessed output paths.
- [x] Add end-to-end examples for route and component splitting.
	`examples/96-code-splitting` now demonstrates shell-preserving route-family switches with a reloadable lazy panel, and `examples/tests/96-code-splitting.spec.ts` verifies the catalog and operations flows in Playwright.

### Server-interactive runtime experiments

- [x] Decide whether server-owned interactive rendering is a real product direction.
	`docs/FRAMEWORK_SCOPE.md` now states that core stays browser-owned and that any server-owned interactive rendering work would need to live as a separate experiment or sibling package rather than a default product direction.
- [x] Define the minimum experiment scope for server-interactive mode.
	`docs/SERVER_INTERACTIVE.md` now limits the first experiment to event transport, server-owned state, DOM patch streaming, reconnect handling, and a small reference app while explicitly excluding parity with the browser-owned runtime.
- [x] Audit which current runtime assumptions block a server-interactive mode.
	`docs/SERVER_INTERACTIVE.md` now records the browser-owned assumptions in the scheduler, hook/context model, event wrapping, atom registry, hydration, commit path, and browser-state helpers that would need abstraction before a server-owned runtime could work.
- [x] Evaluate transport shape for server-interactive updates.
	`docs/SERVER_INTERACTIVE.md` now recommends WebSocket transport with JSON-shaped DOM-op patches for the first experiment, treats tree-patch messages as the longer-term abstraction, and reserves full HTML streaming for bootstrap or reconnect fallback instead of the normal interaction path.
- [x] Define latency and offline expectations up front.
	`docs/SERVER_INTERACTIVE.md` now says the first experiment should keep bounded actions, submits, and route-like changes responsive under moderate latency; avoid drag/continuous-input surfaces; keep stale UI visible on reconnect; and treat full offline operation as out of scope unless the reference app explicitly needs replay.
- [x] Add a security and scalability risk review for server-interactive mode.
	`docs/SERVER_INTERACTIVE.md` now covers per-session memory cost, tenant isolation, auth and session propagation, backpressure, DoS limits, and the recommendation to keep the first experiment small, authenticated, and bounded.
- [ ] Add a narrow proof-of-concept example.
	Use a dashboard or admin-style app with modest interaction density to validate the model before attempting general-purpose parity with the client-owned runtime.

### Ecosystem and plugin story

- [x] Decide whether the framework needs a plugin or directive model.
	`docs/ECOSYSTEM.md` now makes the current stance explicit: no generic plugin registry or directive layer in core today; prefer companion packages and package-specific extension points until multiple subsystems prove the need for a shared lifecycle.
- [x] Define extension boundaries before adding ad hoc framework utilities.
	`docs/ECOSYSTEM.md` now defines ownership rules and a core versus package-owned versus companion-package versus application-owned matrix so new utilities must declare their home before they spread across unrelated packages.
- [x] Identify which ecosystem problems belong in core versus companion packages.
	`docs/ECOSYSTEM.md` now lists the default split: core keeps correctness-critical rendering, state, routing, SSR, hydration, diagnostics, and typed form primitives, while higher-level auth, head management, query orchestration, animation, asset helpers, and testing utilities default to companion packages.
- [x] Publish stability tiers for extension authors.
	`docs/ECOSYSTEM.md` now translates the repo-wide stability labels into extension-author guidance, including when ecosystem surfaces count as stable, supported companion, experimental, or internal.
- [x] Define extension hooks for router, async data, devtools, SSR, and forms.
	`docs/ECOSYSTEM.md` now defines a subsystem hook matrix that says what router, async-data, devtools, SSR, and form extensions may observe or influence, plus what remains off-limits without internal access.
- [x] Define a minimal plugin lifecycle.
	`docs/ECOSYSTEM.md` now defines the minimum lifecycle for companion integrations: declare compatibility, register through explicit application code, receive only documented hook inputs, return explicit outputs or cleanup, and fail predictably when requirements are missing.
- [x] Define compatibility and versioning policy for companion packages.
	`docs/ECOSYSTEM.md` now defines the semver baseline, dependency declaration rules, experimental-hook caveats, deprecation expectations, and third-party guidance for companion packages.
- [x] Add companion-package candidates to the roadmap.
	`docs/ECOSYSTEM.md` now tracks the likely first companion packages, including head management, auth helpers, query or mutation orchestration, animation and gesture helpers, asset or media helpers, and testing utilities.
- [x] Add a reference plugin or companion package.
	The repo now includes `head/` as a supported companion package for SSR head composition and `plugin/` plus `examples/99-plugin-host` as the experimental explicit plugin-host reference, both built on documented public APIs rather than privileged runtime internals.
- [x] Add a supported query and mutation orchestration companion package.
	`docs/ECOSYSTEM.md` now documents the `fetch` package's `UseCachedResource[T]` with shared cache, stale-while-revalidate, SSR bootstrap restore, persistent cache, and plugin-contributed cache-key decoration as the supported query layer, and `OpenMutationQueue` with persistent offline queue, deduplication, application-owned replay, and conflict resolution as the supported mutation layer, with boundary rules keeping entity normalization and infinite scroll orchestration application-owned.
- [x] Add a supported protobuf RPC companion package for Go/WASM clients.
	`docs/ECOSYSTEM.md` now defines the companion package scope for protobuf RPC: connection lifecycle, typed client wrappers, streaming integration with `ui.UseEffect` and background workers, diagnostics via `plugin.CapabilityDevtools`, and auth propagation, with `examples/100-ai-chat-wizard` as the reference implementation and `grpc-tunnel` as the transport substrate. Tier starts as Experimental until two real applications validate the surface.
- [x] Add a framework integration layer above RPC transport for actions, queries, and revalidation.
	`docs/ECOSYSTEM.md` now defines the integration surface: `UseRPCResource[T]` for typed unary calls with `AsyncResource[T]` semantics, `UseRPCStream[T]` for server-streaming with lifecycle-tied cleanup, mutation actions that trigger cache invalidation and route revalidation through existing `router.UseRevalidator()` and `CachedResource.Invalidate()`, and pending/error state that reuses `fetch` error patterns. The layer is a companion package that must not introduce new runtime primitives.
- [x] Add a schema-driven client codegen strategy for OpenAPI, protobuf, and similar API contracts.
	`docs/ECOSYSTEM.md` now defines the codegen strategy: Protocol Buffers as primary, OpenAPI as secondary, GraphQL deferred. Generated code lives in dedicated directories, is committed to source control, documented via `//go:generate`, and must not be hand-modified. Codegen tools produce typed structs, context-aware client functions, and composable error types without framework-internal dependencies. Codegen is never a hidden build-system requirement.
- [x] Define ownership and upgrade rules for generated API clients.
	`docs/ECOSYSTEM.md` now defines ownership and upgrade rules: schemas are application-owned, generated code is a reviewed build artifact, versioned package paths enable coexistence during migration, diffs are reviewed for backward compatibility, and generated clients compose with framework primitives through explicit wiring (`UseResource[T]`, `UseCachedResource[T]`, mutation actions, streaming hooks) rather than hidden registration.
- [x] Add a supported animation and gesture companion package.
	`docs/ECOSYSTEM.md` now defines the animation and gesture companion package scope: spring and tween primitives, transition orchestration, route transition helpers, gesture recognition (swipe, drag, pinch, long-press), reduced-motion integration with `UseReducedMotion()` hook, and overlay transition helpers. All motion primitives respect `prefers-reduced-motion` by default. Tier starts as Experimental, requiring stable primitives and proven accessibility compliance for promotion.
- [x] Add companion-package maturity checks and compatibility matrices.
	`docs/ECOSYSTEM.md` now includes a companion-package compatibility matrix tracking tier, minimum framework version, experimental API dependencies, test coverage, ownership, last-verified date, migration guides, and known limitations for each official companion package (`head`, `plugin`, `fetch` cache layer, `fetch` mutation queue). Update rules and a re-verification process are defined for framework major version releases.

### Launcher extensibility and enterprise policy

- [ ] Define launcher config layering for framework defaults, organization policy, project config, and CLI overrides.
	Make `gwc` resolve ports, output paths, release budgets, test lanes, compression policy, and org-required checks through one explicit precedence model instead of hidden enterprise patching.
- [ ] Add simple pre and post command hooks to the launcher.
	Support auditable hook points such as pre-dev, post-build, pre-release, post-release, and pre-verify before introducing a broader plugin runtime.
- [ ] Define a capability-based launcher plugin contract.
	Allow plugins to contribute scaffold features, test lanes, verify checks, release validators, or deployment packagers without letting them silently rewrite core launcher semantics.
- [ ] Prefer external executable plugins with structured JSON I/O.
	Use an explicit executable-plugin protocol for enterprise extensions so launcher integrations stay cross-platform, auditable, and isolated from the core process.
- [ ] Add plugin-discovery and trust reporting to command output.
	Show which plugins, hooks, and policy packs were loaded for each launcher invocation so enterprise teams can audit why a command behaved differently.
- [ ] Add policy-pack support for organization-specific enterprise requirements.
	Allow reusable configuration or plugin bundles to enforce approved toolchains, required verify lanes, artifact naming rules, manifest requirements, and deployment validations across many projects.
- [ ] Add launcher security boundaries for hooks and plugins.
	Define what launcher extensions may read, modify, and emit, how secrets are passed, and how failures are isolated so enterprise customization does not become an opaque security risk.
- [ ] Add plugin-contributed feature sections to the scaffold TUI.
	Let organization features appear as an explicit optional section in `gwc start`, clearly separated from built-in presets and features so starter output remains understandable.
- [ ] Add plugin and policy integration tests.
	Cover discovery, ordering, config precedence, failure attribution, and machine-readable plugin diagnostics so enterprise customization stays supportable.

### Ecosystem and adoption maturity

- [x] Define the minimum ecosystem story for 1.0-style adoption.
	`docs/ADOPTION.md` now defines the baseline explicitly: a 1.0-style story needs one first-party or officially recommended answer for starter path, testing, SSR and hydration, state, routing, and deployment, with the current repo mappings and the remaining starter-gap called out directly.
- [x] Add comparison docs against major frameworks.
	`docs/COMPARISONS.md` now compares GoWebComponents against React, Vue, Svelte, Solid, Blazor, and Qwik, separating intentional design choices from current maturity gaps and active areas of closure.
- [ ] Publish production-readiness criteria by feature area.
	Separate experimental SSR, hydration, compiler, and runtime experiments from stable component, router, and state features so adopters can judge risk quickly.
- [ ] Add a real-world case study or reference application.
	Framework maturity is hard to evaluate from isolated examples alone; a sustained medium-size app should validate routing, async data, SSR, hydration, and operational workflow together.
- [ ] Publish a maintained starter matrix with support tiers and update cadence.
	Track which starters are minimal, routed, SSR, or enterprise-shaped; which are fully supported versus experimental; and how often they are verified against current releases so starter maturity becomes measurable instead of implied.
- [ ] Add an ecosystem inventory with maintenance and support signals.
	Catalog first-party packages, supported companions, example-only integrations, and notable external integrations with ownership, test status, and compatibility notes so ecosystem depth can be judged honestly from one place.
- [ ] Publish enterprise evaluation packets for architects and procurement reviewers.
	Provide one concise package that covers support posture, browser support, security model, upgrade discipline, operational ownership boundaries, and current maturity risks so enterprise comfort does not depend on a repo-wide scavenger hunt.
- [ ] Add guided learning tracks and workshop-style training material.
	Expand beyond isolated docs pages into sequential learning paths for client-only apps, routed SPAs, SSR apps, forms-heavy systems, and offline-capable apps so learning depth can move toward H3.
- [ ] Add public community-growth paths beyond core-runtime contribution.
	Create clear contribution lanes for guides, examples, companion packages, integration recipes, and case studies so community reach can grow through more than low-level runtime code changes.

### Team-scale conventions and developer ergonomics

- [x] Publish recommended project structure for non-trivial apps.
	`docs/PROJECT_STRUCTURE.md` now documents a recommended larger-app layout covering entrypoints, route and app integration code, reusable UI, domain or service layers, assets, templates, and test placement for production-style apps.
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
- [ ] Add team-onboarding playbooks for adopting the framework at scale.
	Show how a new engineer learns the route structure, state layers, form patterns, async resource ownership, and deployment boundaries in a medium-size app so team-scale adoption stops depending on local tribal knowledge.
- [ ] Publish design-system and component-library workflow guidance.
	Show how teams should build reusable presentational components, tokens, theming, accessibility guarantees, and SSR-safe shared UI layers without coupling everything directly to one application tree.
- [ ] Add packaging and release guidance for internal component libraries.
	Document how shared UI packages should version themselves, test consumer compatibility, expose styles or assets, and integrate with starter apps and framework upgrades in multi-repo or monorepo environments.

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
- [ ] Add startup, hydration, and first-interaction profiling workflows.
	Make it easy to diagnose download, instantiate, bootstrap, hydration, and time-to-first-meaningful-interaction costs so Go plus WASM performance work is tied to the moments adopters actually feel.
- [ ] Add route-by-route startup budget reporting.
	Attribute download, bootstrap payload size, hydration cost, and first-interaction timing per route family so teams can see which screens are actually causing user-visible startup regressions.
- [ ] Add wasm artifact and bootstrap cost attribution.
	Break startup cost into binary size, decoded bootstrap size, cache warmup, service-worker overhead, and initial route data so optimization work can target the real dominant factor instead of treating startup as one opaque number.
- [x] Add devtools export and snapshot comparison support.
	`devtools.ExportSnapshotJSON(...)` and `devtools.CompareSnapshots(...)` now let developers save inspection snapshots, compare before/after traces, and inspect which top-level sections changed across an optimization attempt instead of relying on one-off local observation.
- [ ] Add profiling examples and performance regression tests.
	Use representative apps such as large lists, nested routes, async dashboards, and portal-heavy overlays to ensure the profiling surface remains useful for real bottlenecks.
- [ ] Add regression fixtures for startup, hydration, and rerender budgets.
	Keep representative performance budgets for a small app, a routed mid-sized app, and a production-shaped app so perf tooling is validated against user-visible flows instead of only microbenchmarks.

### Debugging and devtools workflows

- [ ] Add richer component-stack and failure context for runtime errors.
	Include component ancestry, route context, active async resource state, and hydration phase details when render, effect, loader, or interop failures are reported.
- [ ] Add a â€œwhy did this rerender?â€ inspection surface.
	The current devtools baseline already shows the committed tree, hook summaries, route inspection, cache inspection, diagnostics, and profiling hotspots; add causal rerender attribution on top of that baseline so developers can see whether a rerender was triggered by props, local state, context, atoms, route changes, loader updates, or parent rerenders.
- [ ] Add hook-slot and state-transition inspection.
	Deepen the existing hook-summary view into selected-component inspection that exposes current hook values, dependency snapshots, recent transitions, and effect lifecycle state during development.
- [ ] Add route and loader debugging panels.
	Expand the current route inspection into a deeper route-debugging panel that shows the route stack, params, query, active guards, redirect causes, loader state, and route metadata ownership live.
- [ ] Add offline replay and sync debugging panels.
	Expose queue entries, replay ownership, reconnect status, conflict state, per-entity sync health, and last replay error so offline-capable apps can debug the hardest field failures without custom logging.
- [ ] Add hydration debugging tools.
	Highlight reused nodes, replaced subtrees, mismatch locations, and fallback boundaries so hydration failures are easier to localize than raw console warnings.
- [ ] Add bootstrap and serialization-boundary inspectors.
	Show which payloads crossed SSR, worker, RPC, and multi-client boundaries, how large they were, and which values were redacted, downgraded, or rejected so boundary bugs become inspectable instead of inferred from logs.
- [ ] Add cache, worker, and synchronization inspectors.
	Build on the current shared-cache inspection by exposing worker jobs, cross-tab or multi-window events, and offline replay state in devtools so coordination bugs can be debugged without custom logging.
- [ ] Add a browser error overlay for development failures.
	Show routed runtime errors, hydration failures, loader crashes, and startup faults in one structured in-browser overlay with stable codes, top user frames, and docs links instead of relying on terminal or raw console output alone.
- [ ] Add recovery actions to the development error overlay.
	Let the overlay surface quick actions such as retry loader, clear bootstrap payload, reveal related launcher diagnostics, or open docs anchors so it shortens the path from failure to recovery instead of only restating the error.
- [ ] Add source-mapped stack translation for wasm runtime failures.
	Resolve browser-observed frames back to application-owned Go files and symbols where possible so debugging render, event, loader, and startup failures does not stop at low-level wasm offsets.
- [ ] Correlate source-mapped runtime failures with launcher artifact metadata.
	Connect browser-observed wasm offsets, mapped Go frames, launcher build metadata, and release manifests so developers can tie a runtime failure back to the exact emitted artifact and symbol set they shipped.
- [ ] Add strict development-mode toggles.
	Allow tests and local development to escalate specific recovered warnings into hard failures so incorrect-but-recovered behavior does not linger unnoticed.
- [ ] Add trace capture and replay support for debugging sessions.
	Allow developers to save one interaction trace, compare before/after behavior, and replay difficult timing-sensitive bugs without manually reconstructing state.
- [ ] Add reproducible local bug-capture bundles.
	Package the current route, diagnostics, relevant bootstrap payloads, cache state, replay queue summary, component snapshot, and recent logs into one local artifact that developers can reopen or attach to an issue without exposing secrets by default.
- [ ] Add a first-class component-tree inspector with route and cache context.
	The current devtools panel already renders the committed component tree with hook summaries; extend it into a selected-node inspector with props or state summaries, active route segment, cache subscriptions, and async-resource ownership so the in-app tooling goes beyond the current overview surface.
- [ ] Add devtools extension points for companion packages and app-owned inspectors.
	Let supported companions and applications register redacted custom panels or snapshot sections so the shared devtools surface can grow into an H3-level integrated debugging workflow instead of remaining framework-only.
- [ ] Add support-safe diagnostic bundle export.
	Produce one redacted bug-report artifact that can include logs, diagnostics, route state, cache state, hydration mismatches, and devtools snapshots without leaking secrets so in-app tooling becomes usable in real support and enterprise debugging loops.

### Actionable errors and developer guidance

- [x] Audit the highest-friction framework errors and warnings.
	`docs/ACTIONABLE_ERRORS.md` now audits the current panic, diagnostic, hydration, router, and interop surfaces, identifies the highest-friction failures, and defines the first stable diagnostic anchors the runtime should target next.
- [x] Add structured, actionable error messages for common mistakes.
	Hook misuse, nil-context access, invalid `ui.CreateElement(...)` shapes, route component misuse, and structured interop failures now all surface concrete next steps and point back to `docs/ACTIONABLE_ERRORS.md` instead of stopping at bare panic or error text.
- [x] Add error codes or stable diagnostic identifiers where appropriate.
	Runtime diagnostics and framework logs now carry stable codes such as `GWC-HYDRATION-TEXT-MISMATCH`, `GWC-ROUTER-DUPLICATE-ROUTE`, and `GWC-ROUTER-LOADER-FAILED`, while interop keeps its stable `ErrorCode` model.
- [x] Link runtime diagnostics to docs and remediation guidance.
	Known diagnostics and interop failures now include docs anchors and remediation text so devtools snapshots, logs, and errors can send developers directly to the relevant fix guidance.
- [x] Distinguish between recoverable warnings and correctness-threatening failures clearly.
	Runtime diagnostics and logs now expose an explicit `Recoverable` flag alongside severity and classification so degraded-but-recovered behavior is distinguishable from correctness failures.
- [x] Add development-time assertions for high-confidence misuse cases.
	Existing fast-fail panics for invalid hook usage, nil context descriptors, invalid `ui.CreateElement(...)` shapes, and invalid route components now include stable identifiers and concrete remediation instead of opaque assertion text.
- [x] Add tests that lock in diagnostic quality.
	Runtime, hydration, router, and UI tests now assert stable codes, docs anchors, remediation hints, and actionable panic text so future changes do not silently degrade diagnostic quality.

### Fatal panic wrapping and console-log design

- [x] Audit every framework-owned uncaught panic boundary.
	List which render, event, effect, cleanup, loader, hydration, bootstrap, and SSR paths can still escape as raw Go panic output so panic wrapping work targets the last fatal surfaces instead of only the easiest ones.
- [x] Intercept uncaught component render panics before fatal exit.
	Wrap function-component render entrypoints and other app-owned render callbacks so plain component panics no longer fall straight through as classic raw Go panic output.
- [x] Intercept uncaught event-handler panics before fatal exit.
	Wrap DOM event handlers, synthetic event bridges, and callback adapters so button clicks, form submissions, and other user events emit wrapped fatal diagnostics instead of bare runtime panics.
- [x] Intercept uncaught effect-body panics before fatal exit.
	Wrap effect execution paths so panics during mount or dependency-driven reruns produce the improved fatal log shape with app frame and component path context.
- [x] Intercept uncaught cleanup panics before fatal exit.
	Wrap effect cleanup and teardown paths so unmount-time failures, dependency cleanup failures, and disposal-time crashes do not surface as classic raw panic dumps.
- [x] Intercept uncaught async loader and route-data panics.
	Wrap route loaders, async resource callbacks, route revalidation hooks, and data-driven retry paths so route-owned failures use the same fatal log contract as component panics.
- [x] Intercept uncaught hydration-phase panics.
	Wrap hydration mismatch escalation, subtree reuse checks, and hydration-time component work so strict or fatal hydration failures preserve app-first context before any unavoidable exit.
- [x] Intercept uncaught bootstrap and startup panics.
	Wrap root mount, bootstrap payload decode, wasm startup, and runtime initialization entrypoints so first-load failures still emit the wrapped panic design instead of only low-level startup noise.
- [x] Intercept uncaught scheduler and deferred work panics.
	Wrap scheduler callbacks, queued work-loop continuations, timer-driven rerenders, and deferred work entrypoints so delayed crashes retain the same structured fatal log layout.
- [x] Intercept uncaught server-render and SSR integration panics where applicable.
	Wrap request-time render, bootstrap generation, and SSR integration boundaries so server-owned failures can use the same app-frame-first contract even when the transport differs from browser console output.
- [x] Preserve the original panic payload as the first visible log line.
	Keep the exact panic text developers would otherwise see from Go so improved formatting adds clarity without hiding the original failure signal or making issue reports harder to correlate.
- [x] Surface the first app-owned frame before framework internals.
	Extract the top user-code frame such as `main.HelloWorld at test/testapp/main.go:351` and show it as `where:` so authors can jump to the actionable location before reading reconciler or scheduler frames.
- [x] Add component-path context to fatal panic logs.
	Show the resolved component ancestry such as `App > HelloWorld` as a `path:` line so panics inside nested layouts, portals, or async subtrees can be localized without reconstructing the tree manually.
- [x] Add a short labeled error summary line.
	Repeat the panic payload in a compact `error:` field so noisy browser consoles still show the core failure clearly even when the first panic line scrolls out of view.
- [x] Explain fatal runtime consequence in plain language.
	Replace confusing Go-specific wording such as `[recovered, repanicked]` with a framework-owned `runtime:` line that says whether no boundary handled the panic, whether render work stopped, and whether the app can still be trusted.
- [x] Group stack output into app, framework, and platform sections.
	Trim or reorder raw stack output so user frames appear first, framework frames remain available for debugging, and low-level wasm or `syscall/js` frames are still preserved but visually demoted.
- [x] Avoid duplicate fatal panic noise.
	Ensure the runtime does not emit both a wrapped panic block and a second competing framework log entry that repeats the same information with different wording.
- [x] Mirror wrapped fatal panics into devtools and diagnostics buffers.
	Send the same stable code, path, top frame, remediation, and recoverability metadata into runtime diagnostics and devtools logs so browser console output and in-app debugging views stay consistent.
- [x] Decide which fatal panic paths should recover versus rethrow.
	Document and encode the boundary between panics that may safely degrade into boundary fallback UI and panics that must still terminate after logging because runtime correctness can no longer be trusted.
- [x] Add positive-path unit tests for wrapped fatal panic formatting.
	Assert that render, event, effect, cleanup, loader, hydration, and bootstrap failures each emit the expected code, path, error summary, consequence line, grouped stacks, and docs anchor when interception succeeds.
- [x] Add negative-path tests to ensure non-panic flows are unchanged.
	Assert that successful renders, handled error-boundary fallbacks, ordinary warnings, and non-fatal diagnostics do not emit wrapped fatal panic logs or duplicate error records.
- [x] Add edge-case tests for unusual panic payloads.
	Cover empty panic messages, `error` values, non-string panic payloads, repeated panics, missing component paths, anonymous component functions, and nested boundary interactions so the formatter stays readable in odd cases.
- [x] Add integration tests for runtime-to-devtools panic propagation.
	`devtools/devtools_wasm_test.go` now proves that a wrapped fatal panic snapshot mirrors the runtime diagnostic and log metadata exactly, including stable code, message, docs, remediation, recoverability, top frame, consequence, and mirrored path or runtime fields.
- [x] Add Playwright console tests for wrapped fatal panic output.
	`test/specs/panic_contract.spec.js` now drives an intentionally crashing wasm render path and asserts the browser-observed wrapped panic contract, including the panic payload, stable code, `where:`, `path:`, `runtime:`, and grouped stack sections.
- [x] Add Playwright regression tests for non-fatal boundaries.
	The same focused Playwright coverage now exercises a boundary-owned render crash in `test/testapp/main.go`, verifies fallback UI renders, and asserts that no fatal wrapped panic block is printed when the runtime recovers locally.
- [x] Add docs for the fatal panic log contract.
	Document the intended line order, stable fields, and recovery-versus-fatal rules in `docs/ACTIONABLE_ERRORS.md` so future message refinements do not drift or regress.

### Public testing utilities for app authors

- [x] Define the first-party testing surface for consumers.
	`docs/TESTING.md` now defines the intended shape as one first-party companion testing module with several focused helper packages, keeping consumer ergonomics first-party while leaving runtime correctness inside core packages.
- [x] Add a component render and query harness for tests.
	`testkit/render` now provides a first public `js/wasm` fixture over `mockdom`, with controlled mounting, rerendering, and basic rendered-node queries for component tests without importing repo-local runtime test helpers.
- [x] Add event and async-flush helpers for browser-facing tests.
	`testkit/render` now exposes synthetic dispatch helpers such as click and input plus `Flush`, `FlushTimers`, and `Stabilize`, so `js/wasm` consumer tests can drive handlers and settle queued work without fragile sleeps or raw handler-property access.
- [x] Add hook-level testing utilities where feasible.
	`testkit/hooks` now provides a first public `RenderHook(...)` harness for `js/wasm` tests, letting hook-driven state flows be exercised without a bespoke host component in every test.
- [x] Add router testing helpers.
	`testkit/router` now provides first public hash and history router fixtures for `js/wasm` tests, covering initial-path setup, navigation, route inspection, params and query assertions, and delegated rendered-route queries.
- [x] Add SSR and hydration assertion helpers.
	`testkit/ssr` now provides public HTML snapshot helpers, typed bootstrap-payload assertions, and a lightweight `js/wasm` hydration smoke harness so SSR delivery paths can be tested without repo-local fixtures.
- [x] Define stable public query semantics for test helpers.
	`testkit/render` now treats accessibility-first role/name queries as the preferred public contract, with exact normalized matching and lower-level id/text/tag helpers retained as explicit escape hatches.
- [x] Add example tests and documentation for consumers.
	Consumer-copyable examples now live alongside `testkit/render`, `testkit/hooks`, `testkit/router`, and `testkit/ssr`, and `docs/TESTING.md` now points to the recommended unit, integration, router, SSR, and hydration patterns.
- [x] Add deterministic scheduler and flush helpers for tests.
	`testkit/render` now exposes `Flush`, `FlushTimers`, and `Stabilize`, and the mockdom scheduler now supports full queued-work draining so `js/wasm` tests can settle render, timeout, and follow-up effect work without sleeps or incidental event-loop timing.
- [ ] Add consumer-copyable examples under the preferred `test/...` import paths.
	The public docs now present `test/render`, `test/hooks`, `test/router`, and `test/ssr` as the preferred import paths, but the copyable consumer examples still live only under `testkit/...`; add real preferred-path examples or duplicate coverage so the documented golden path does not immediately fall back to the compatibility aliases.
- [ ] Add parity coverage for `test/...` wrappers and `testkit/...` compatibility aliases.
	The preferred public packages are thin wrappers over `testkit/...` today; add tests that lock API and behavior parity so wrapper drift, missing exports, or accidental alias-only improvements do not split the testing surface in practice.
- [ ] Add a first-class `test/browser` helper package.
	`docs/TESTING.md` already names `test/browser` as the intended browser-runner companion slice, but no public package exists yet; provide thin helpers for Playwright-style setup, stable app-ready waits, and safe console or diagnostic capture without trying to replace browser automation frameworks.
- [ ] Add explicit browser-environment simulation helpers for public tests.
	The current router fixture still installs private ad hoc `window`, `location`, and `history` shims, while newer public surfaces also depend on storage, media, worker, and window-channel behavior; provide supported test helpers for controlled browser-environment state so consumer tests do not keep rebuilding those globals by hand.
- [ ] Add async resource and loader test utilities.
	Provide helpers for resolving, rejecting, cancelling, retrying, and stalling resource or route-loader work so async UI flows can be tested precisely.
- [ ] Add round-trip hydration test helpers that start from real server markup.
	`test/ssr.SmokeHydrate(...)` currently hydrates into an empty controlled fixture and proves only a lightweight no-crash resume path; add helpers that seed SSR-emitted HTML or DOM state first so public tests can assert node reuse, managed metadata preservation, and mismatch diagnostics against actual server-shaped markup instead of only an empty-container smoke run.
- [ ] Add structured SSR and head assertions for snapshot tests.
	Current `test/ssr` helpers and companion head tests still rely heavily on raw substring checks for titles, descriptions, canonical tags, social metadata, JSON-LD, and bootstrap scripts; add typed assertion helpers so metadata-heavy SSR tests can verify meaning without brittle string-order coupling.
- [ ] Add prerender and static-export test helpers.
	`prerender.Export(...)` currently falls back to package-local file assertions, but the public testing surface has no companion helpers for exported route paths, emitted HTML files, bootstrap sidecars, or metadata checks across static output; add a supported path so prerendered apps are not tested only through ad hoc disk reads.
- [ ] Add portal and overlay testing helpers.
	Support asserting active overlay stacks, focus restoration, escape dismissal, outside-click behavior, and scroll-lock coordination in portal-heavy tests.
- [ ] Add accessibility-first assertions for public test utilities.
	Prefer role, label, description, and live-region queries so framework tests and consumer tests encourage accessible UI structure rather than brittle CSS selectors.
- [ ] Add accessibility-first query and interaction delegation to router fixtures.
	`test/router` currently delegates only `ByID`, `ByText`, and raw text from its underlying render fixture; expose `ByRole`, `AllByRole`, and basic event helpers there too so routed-shell tests can stay on the same public query and interaction contract as `test/render` instead of reaching around the router harness.
- [ ] Add mismatch and failure-injection helpers.
	Allow tests to intentionally trigger hydration mismatches, loader failures, route guard failures, cache conflicts, and offline replay errors so recovery behavior can be asserted directly.
- [ ] Add cross-tab, worker, and offline test harnesses.
	Provide controlled test environments for synchronization channels, worker messaging, background retry flows, and reconnect behavior so these coordination features are not tested only through ad hoc browser scripts.
- [ ] Add diagnostics and buffered-log assertions to the public test utilities.
	Expose structured access to runtime diagnostics and recent framework logs from the public harnesses so consumer tests can assert recoverable warnings, hydration mismatches, router diagnostics, and other actionable framework signals without reaching into internal runtime state.
- [ ] Add an explicit parallel-safety contract for `js/wasm` test fixtures.
	The current `testkit/render` fixture serializes ownership because the global runtime and hook state are process-wide on `js/wasm`; either add supported isolation helpers for safe concurrent fixture use or document and enforce the single-fixture contract so consumer tests do not assume `t.Parallel()` is safe when it is not.
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
