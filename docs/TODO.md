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
- [x] Publish an API stability and support policy.
	The docs now define stability tiers for public packages, experimental features, companion APIs, internal details, semver expectations, the deprecation lifecycle, and latest-major support policy in `docs/API_POLICY.md`.
- [x] Add migration notes as new primitives land.
	The docs now include `docs/MIGRATIONS.md` as the release-to-release upgrade index, starting with the transition into the current `v3.x` public package layout.
- [x] Keep release-to-release migration guides current for future major changes.
	Each future major release still needs subsystem-specific upgrade guidance for runtime, router, SSR, forms, state, and deployment changes before the release is considered ready for enterprise adoption.

### Documentation discoverability and task-oriented guidance

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
- [ ] Add test coverage for async guard edge cases.
	Cover double-click navigation, back and forward navigation, loader-plus-guard interactions, and cleanup when guarded components unmount mid-check.

### Auth-aware routing primitives

- [x] Define a lightweight auth context model for routed apps.
	`docs/ROUTER_AUTH.md` now defines the planned `AuthState` / `AuthStatus` model, the intended provider and hook access pattern, and the boundary that the router consumes auth hints without becoming an identity-provider framework.
- [ ] Add router-level unauthorized and authorizing UI states.
	Allow routes and route groups to render explicit unauthorized, forbidden, and pending-auth content instead of forcing every app into manual redirects.
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
	`docs/OFFLINE_MUTATIONS.md` now defines the shipped scope as durable browser-side queued writes with explicit replay, while leaving service-worker coordination, conflict resolution, and encrypted storage out of the current first-class boundary.
- [x] Add a persistent mutation queue abstraction.
	`fetch.OpenMutationQueue(...)` now provides enqueue, list, remove, clear, and replay helpers backed by browser storage, with wasm coverage for persistence across reopen and replay-order removal on success.
- [x] Define optimistic-update and rollback semantics for queued writes.
	`docs/OFFLINE_MUTATIONS.md` now defines the application-owned model: optimistic atom or cache updates happen separately from the queue, successful replay should invalidate authoritative reads, and rollback or conflict handling stays under app control.
- [x] Add retry, deduplication, and backoff policies for queued mutations.
	Queued mutations now support `DedupKey` suppression, exponential backoff through `BaseDelay` and `MaxDelay`, deferred replay until `NextAttemptAt`, and terminal `dead` state after `MaxAttempts`, all covered by wasm tests.
- [x] Define queue serialization and security boundaries.
	`docs/OFFLINE_MUTATIONS.md` now defines the JSON-shaped storage contract, warns against persisting secrets or browser-native handles, and recommends metadata version hints for queued payloads that must survive app upgrades.
- [ ] Integrate offline mutation replay with service workers and background sync where available.
	Document or provide a first-party pattern for using Background Sync or equivalent service-worker coordination when the platform supports it, with graceful fallback when it does not.
- [ ] Add examples for offline write replay.
	Demonstrate a draft save, queued form submission, or cache-backed mutation flow that survives offline periods and reconciles cleanly after connectivity returns.

### Cross-tab state and cache synchronization

- [x] Define the scope of first-party cross-tab synchronization.
	`docs/CROSS_TAB.md` now defines the first shipped scope as opt-in cross-tab state hints, auth or logout signals, cache invalidation, and draft updates, while explicitly leaving global auto-sync, multi-window orchestration, and service-worker coordination out of the current boundary.
- [x] Add BroadcastChannel-based synchronization helpers.
	`interop.OpenCrossTabChannel(...)` now provides named cross-tab publish and subscribe helpers backed by `BroadcastChannel` when available, with typed decode through `interop.DecodeCrossTabEnvelope[T](...)` and `interop.SubscribeDecodedCrossTab[T](...)`.
- [x] Add storage-event fallback support where appropriate.
	The cross-tab channel helper now falls back to `localStorage` plus `storage` events when `BroadcastChannel` is unavailable, and the wasm tests cover both transports.
- [x] Define conflict resolution and merge semantics for cross-tab updates.
	`docs/CROSS_TAB.md` now defines transport-level ordering versus application-owned merge rules, including immediate application of narrowing auth signals, revision-aware handling for optimistic entity updates, and the preference for invalidation over blind overwrite when conflict risk is high.
- [x] Add opt-in scoping and filtering for synchronized values.
	The current model is explicit per named channel and optional custom storage key, so apps choose which topics synchronize instead of forcing every atom or cache key onto one global bus.
- [x] Define bootstrap and resume interaction for synchronized state.
	`docs/CROSS_TAB.md` now defines the current bootstrap rule set: restore SSR bootstrap or persisted snapshots first, open channels after local state is initialized, and ignore incoming messages that are older than the locally restored revision or timestamp.
- [x] Add diagnostics and examples for cross-tab behavior.
	`examples/94-cross-tab-sync` now demonstrates theme sync, logout propagation, cache invalidation broadcasting, and draft-state sharing across tabs, while its diagnostics panel reports the resolved transport and recent received messages.

### Multi-window and multi-surface coordination

- [x] Define the multi-surface coordination model.
	`docs/MULTI_SURFACE.md` now defines the current first-party model as same-origin popup or secondary-window coordination plus the already-shipped cross-tab channel surface, while explicitly leaving iframe, side-panel, and external control-surface orchestration for later work.
- [x] Add coordination channels for popup and secondary-window workflows.
	`interop.OpenSecondaryWindowChannel(...)`, `interop.WindowOpenerChannel(...)`, `interop.DecodeWindowEnvelope[T](...)`, and `interop.SubscribeDecodedWindow[T](...)` now provide first-class `postMessage` coordination for opener and popup workflows, with native and wasm coverage in `interop`.
- [x] Define ownership and synchronization rules across multiple active surfaces.
	`docs/MULTI_SURFACE.md` now defines the current ownership model: the opener remains authoritative for popup lifecycle, secondary windows act as specialized or mirrored collaborators, and conflicting edits should resolve through one canonical state owner rather than blind cross-surface overwrite.
- [x] Add shared session and route coordination helpers.
	`interop.SurfaceSignal`, `interop.SubscribeSurfaceSignals(...)`, `interop.PublishLogout(...)`, `interop.PublishSessionExpired(...)`, `interop.PublishRouteFocus(...)`, `interop.PublishSelection(...)`, and `interop.PublishIntent(...)` now provide typed helpers for the common opener or popup workflows of logout, expiry, route focus, active-document selection, and intent propagation.
- [x] Define teardown and orphan-surface behavior.
	`docs/MULTI_SURFACE.md` now documents popup loss, opener loss, `WindowChannel.Closed()`, and the expected degraded-state behavior for orphaned surfaces instead of implying automatic recovery or ownership transfer.
- [x] Add examples for multi-window applications.
	`examples/95-multi-window-console` now demonstrates an opener plus popup operator-console workflow with shared session, route, selection, and intent signals, including unexpected close or orphan handling.

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
- [ ] Add observability hooks for server-rendered apps.
	Expose request-level render timing, hydration fallback counters, and bootstrap size metrics so SSR operations are measurable in production.

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
- [ ] Add web app manifest generation or templating helpers.
	Support app name, icons, theme colors, display mode, start URL, and installability metadata without forcing every app to hand-roll the same manifest pipeline.
- [x] Add a service-worker integration story.
	`docs/PWA.md` now defines the intended application-owned service-worker registration, precache, scope, and asset-versioning story without overclaiming a shipped service-worker runtime.
- [x] Define offline caching strategies for app shells and route data.
	`docs/PWA.md` now defines the intended separation between shell caching, immutable asset caching, route-data caching, and queued mutation replay.
- [x] Add update and invalidation semantics for offline assets.
	`docs/PWA.md` now defines versioned cache invalidation, manifest-aligned asset updates, and safe refresh expectations for new wasm or JS asset graphs.
- [ ] Add offline and installability examples.
	Create a small app-shell example that supports install prompt behavior, offline fallback UI, and cache-aware reload behavior on repeat visits.
- [x] Add production guidance for PWA deployments.
	`docs/PWA.md` now defines HTTPS, scope, cache-header, CDN, reverse-proxy, and rollout guidance for PWA deployments.

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
- [x] Add recommended production build flags for wasm targets.
	`docs/WASM_RELEASES.md` now defines the intended production baseline around `-trimpath` and `-ldflags="-s -w"` together with the reproducibility and debugging tradeoffs.
- [x] Define build metadata and debug-info policy for release wasm artifacts.
	`docs/WASM_RELEASES.md` now defines the intended split between stripped production artifacts, explicit debug-friendly builds, and external provenance records for release metadata.
- [x] Add wasm size budgets and regression tracking.
	`tools/build-wasm-release.ps1` now supports optional raw/gzip/brotli budget enforcement, and `docs/WASM_RELEASES.md` defines the intended budget contract for release-oriented builds.
- [x] Add artifact size reporting and comparison tooling.
	`tools/build-wasm-release.ps1` now emits `wasm-release-manifest.json` with relative paths, sizes, and sha256 hashes for release artifacts and compressed sidecars.
- [ ] Add release-time compression support for wasm artifacts.
	Generate gzip and brotli sidecars for production builds and document the required server headers and cache behavior for serving compressed wasm safely.
- [ ] Evaluate post-link wasm optimization tooling.
	Test tools such as `wasm-opt` or equivalent post-processing pipelines and document whether they improve size, startup time, or runtime behavior enough to justify adding them to the release workflow.
- [x] Define asset-manifest and build-output conventions for optimized wasm releases.
	`docs/WASM_RELEASES.md` now defines the intended output directory shape around the raw wasm artifact, compressed sidecars, and `wasm-release-manifest.json`, and `tools/build-wasm-release.ps1` emits that convention directly.
- [x] Add reproducible release-build guidance.
	`docs/WASM_RELEASES.md` now defines the intended toolchain, flag, manifest, and hash-record expectations for reproducible release builds.
- [ ] Add startup-cost measurements for release artifacts.
	Track download size, decompression cost, compile or instantiate time, and first-interaction timing for representative builds so size work is tied to real user-facing outcomes.

### Build speed and optimization experiments

- [x] Establish an experiment matrix for wasm build optimization.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended comparison matrix across plain, release-style, stripped, compressed, and post-processed wasm build variants.
- [x] Add scripted measurement of cold and warm build times.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended cold-build, warm-build, and small-edit rebuild timing set and the measurement rules for repeatable build-speed experiments.
- [x] Add experiment harnesses for wasm size and startup tradeoffs.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended harness inputs for raw size, compressed size, download cost, instantiate time, and first-interaction timing.
- [x] Evaluate Go build flags for size versus speed tradeoffs.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended tradeoff table for evaluating build flags across size, build speed, startup cost, runtime impact, and debugging cost.
- [x] Evaluate `GOWASM` feature toggles and compatibility tradeoffs where relevant.
	`docs/BUILD_EXPERIMENTS.md` now defines the intended policy for recording and evaluating `GOWASM` toggles before standardizing on any non-default release setting.
- [ ] Evaluate post-processing and compression combinations.
	Compare plain wasm, stripped wasm, optimized wasm, gzip delivery, and brotli delivery to identify which combination provides the best real-world startup characteristics for this project.
- [ ] Add CI-friendly benchmark comparison for build experiments.
	Reuse or extend the existing benchmark tooling so candidate build settings can be compared against a saved baseline before they are adopted into the documented release path.
- [ ] Record accepted and rejected build optimizations in docs.
	Keep a short history of build-flag and tooling experiments, including what was tried, what regressed build speed or startup time, and what combinations are currently recommended.

### Developer workflow and project bootstrap

- [x] Define the official way to start a new GoWebComponents app.
	`docs/ONBOARDING.md` now defines the current official starting path as a documented manual flow based on maintained examples and the public package surface until first-party starters exist.
- [ ] Add at least one maintained starter application for the common path.
	Ship a polished baseline app that includes routing, async data, state, forms, testing, and production-minded build setup so teams can begin from a realistic foundation instead of a toy counter example.
- [ ] Add starter variants for the major adoption modes.
	Provide clearly scoped starting points for client-only apps, SSR-enabled apps, and content or dashboard-style apps so teams can choose an architecture without reverse-engineering the examples directory.
- [x] Define upgrade and template-sync guidance for starter-based apps.
	`docs/ONBOARDING.md` now defines the intended starter-upgrade model around starter-specific release notes, core migration docs, and explicit support windows instead of blind monorepo diffing.
- [ ] Add a one-command local bootstrap workflow.
	Make it easy to install prerequisites, build wasm, start the dev server, and open a working example or starter app without several undocumented manual steps.
- [x] Document environment prerequisites and platform expectations clearly.
	`docs/ONBOARDING.md` now defines the intended Go, Node, browser, and Windows/macOS/Linux baseline in one place and points to the browser support contract where relevant.
- [x] Add a â€œchoose your pathâ€ onboarding flow for new adopters.
	`docs/ONBOARDING.md` now defines the intended path chooser for client-rendered, routed, SSR, forms-heavy, and static/prerender-oriented adoption modes.

### Build feedback loop ergonomics

- [x] Define the recommended inner-loop workflow for application authors.
	`docs/ONBOARDING.md` now defines the intended edit-build-refresh loop, the current recommended commands, and the browser-refresh versus stale-wasm reasoning model for local application work.
- [ ] Add a simplified watch-mode entry point for apps.
	Reduce the need to coordinate several scripts manually by providing one supported development command that rebuilds wasm, serves assets, and reports failures coherently.
- [ ] Improve incremental rebuild behavior for small source edits.
	Measure and reduce avoidable rebuild work so common component, route, and style changes do not feel disproportionately expensive during local development.
- [ ] Add clearer surfacing for build progress and failure states.
	Show whether the system is compiling, serving stale output, waiting on a reload, or blocked on an error so developers are not left guessing which part of the toolchain failed.
- [x] Add recovery guidance for broken local development loops.
	`docs/TROUBLESHOOTING.md`, `docs/ONBOARDING.md`, and `tools/README.md` now document the current recovery path for stale wasm output, broken example serving, missing runtime bootstrap files, and local build-versus-served-output drift in the repo's supported manual dev loop.
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
	Separate syntax sugar, dead-code elimination, reactive dependency extraction, template lowering, and SSR build optimization instead of treating â€œcompilerâ€ as one bucket.
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
- [ ] Add a â€œwhy did this rerender?â€ inspection surface.
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
