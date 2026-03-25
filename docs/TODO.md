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

- [x] Publish an official state-management architecture guide.
	`docs/STATE_ARCHITECTURE.md` now defines the practical split between `ui.UseState`, `ui.UseReducer`, context, atoms, selectors, fetch-owned async data, and snapshot persistence, and the main docs indexes now link to it as the state architecture entrypoint.
- [x] Publish an official data-loading and mutation architecture guide.
	`docs/DATA_LOADING_AND_MUTATION_ARCHITECTURE.md` now defines the practical split between route loaders, `fetch.UseResource[T](...)`, shared cache, explicit form mutations, optimistic cache updates, offline replay, and revalidation, and the main docs indexes now link to it as the async architecture entrypoint.
- [x] Publish a first-class auth and session integration guide.
	`docs/AUTH_AND_SESSION_INTEGRATION.md` now defines the practical split between cookie-backed sessions, bearer-token escape hatches, server-hydrated auth hints, route guards, logout invalidation, and same-origin API calls, and the main docs indexes now link to it as the auth entrypoint.
- [x] Publish business-app form workflow recipes.
	`docs/BUSINESS_APP_FORM_RECIPES.md` now turns the shipped `ui.UseForm[T]` surface into practical guidance for validation, pending UX, field-error projection, redirects, uploads, and server-authoritative mutation handling, and the main docs indexes now link to it as the business-form entrypoint.

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
- [x] Keep the explicit shipped-example checklist aligned with the real catalog.
	Update the backlog checklist and surrounding summary whenever new numbered examples or integrated reference apps land so the claim that shipped examples are tracked explicitly remains true through examples such as `77` through `102`, the secure-forms reference, and later catalog additions.
	The checklist below now tracks the runnable catalog entries through `106-single-shell-auth`, including the `77` through `106` additions and the `97-*` example family, while intentionally excluding the current empty placeholder directory `examples/97-server-interactive-poc` until it becomes a real shipped example.
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
- [x] `77-accessible-overlay`: `ui.AccessibleOverlay`, `ui.UseFocusManager`, `ui.UseFocusTrap`
- [x] `78-composite-navigation`: `ui.UseCompositeNavigation`
- [x] `79-form-accessibility`: `ui.UseAnnouncer`, validation announcements, focus-to-error behavior
- [x] `80-routed-accessibility`: routed accessibility announcements and focus management
- [x] `81-overlay-stack`: overlay stack coordination
- [x] `82-overlay-anchor`: anchored overlay positioning
- [x] `83-locale-switcher`: locale switching and formatting
- [x] `84-ssr-i18n-bootstrap`: SSR i18n bootstrap restore
- [x] `85-locale-routing`: locale-aware routing
- [x] `86-atlas-commerce-os`: production-shaped SSR reference app
- [x] `87-ssr-secure-forms`: secure SSR forms and server-action reference
- [x] `88-web-components`: `html.CustomElement` and decoded custom events
- [x] `89-exported-custom-element`: export-side custom-element wrapper
- [x] `90-browser-interop`: browser interop helpers
- [x] `91-worker-text-index`: `ui.UseWorkerTask`
- [x] `92-protected-routes`: guarded routes and bounded `return_to`
- [x] `93-ssr-cache-bootstrap`: SSR cache bootstrap restore
- [x] `94-cross-tab-sync`: cross-tab synchronization
- [x] `95-multi-window-console`: multi-window coordination
- [x] `96-code-splitting`: route-family code splitting
- [x] `97-pwa-installability`: manifest and installability diagnostics
- [x] `97-pwa-offline-cache`: offline cache and mutation replay
- [x] `97-pwa-multi-client`: offline-first multi-tab coordination
- [x] `97-multi-client-presence`: cross-tab presence and targeted replies
- [x] `97-multi-client-binary`: binary-capable multi-client messaging
- [x] `98-hot-reload`: state-preserving hot reload
- [x] `99-plugin-host`: explicit plugin-host reference
- [x] `101-static-islands`: selective multi-root hydration
- [x] `102-static-export-site`: prerender export
- [x] `103-virtualized-feed`: virtualization list diagnostics
- [x] `104-use-callback`: `ui.UseCallback`
- [x] `105-use-lazy-node`: `ui.UseLazyNode`
- [x] `106-single-shell-auth`: one-shell auth routing

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
- [x] Add a dedicated `ui.UseCallback` example.
	`examples/104-use-callback` now shows stable callback identity driven by `step`, contrasts that with unrelated rerenders from draft text, and explains when `ui.UseCallback(...)`, `ui.UseEvent(...)`, or an ordinary inline handler is the right fit.
- [x] Add a dedicated `ui.UseLazyNode` example.
	`examples/105-use-lazy-node` now shows hook-level lazy state inspection, explicit `ui.AsyncBoundary(...)` fallback timing, and when `ui.UseLazyNode(...)` is the right tool instead of `ui.Lazy(...)`, route loaders, or worker-backed tasks.
- [x] Make the example inventory and checklist reflect `ui.UseWorkerTask` as a first-class public hook.
	`examples/README.md` and the explicit example checklist now call out `examples/91-worker-text-index` under the `ui` surface so `ui.UseWorkerTask[...]` is treated as public hook coverage instead of being discoverable only through the broader `interop` grouping.

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
- [x] Decide whether to offer an LTS or backport support line for enterprise adopters.
	`docs/API_POLICY.md` now makes the decision explicit: no LTS or older-major backport line is offered today, first-party starters track latest-major only, and procurement or security review should assume one active supported major at a time until a future policy says otherwise.
- [x] Define graduation criteria from experimental to supported or stable surfaces.
	`docs/API_POLICY.md` now defines one repo-wide promotion bar: canonical docs, first-party example coverage, lifecycle and failure-path tests, explicit compatibility notes, actionable diagnostics, real validation beyond a sketch, and migration guidance before experimental surfaces such as hot reload, multi-client coordination, companion packages, or compiler-assisted workflows can graduate.

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
- [x] Add a higher-level head manager surface for non-router metadata.
	The supported `head` companion package now exposes `head.Render(head.Document{...})`, `RenderJSONLD(...)`, alternate-link helpers, and resource-hint helpers so SSR apps can compose Open Graph, Twitter/X, JSON-LD, robots, hreflang, and route-adjacent hints without hand-building the whole head block in each document template.
- [x] Add route-to-head composition helpers for larger apps.
	The `head` companion package now exposes `Merge(...)`, `Resolve(...)`, `RouteLayer`, and explicit `MergeOptions` so larger route trees can apply layout defaults, leaf overrides, and predictable replace or clear rules for robots, social metadata, alternate links, resource hints, JSON-LD, and other companion-owned head bundles.
- [x] Add end-to-end head ownership examples across SSR, prerender, and hydrated navigation.
	`examples/18-ssr-server-routing` now emits a metadata-heavy `head.Render(...)` bundle on first paint and documents the ownership boundary during hydrated navigation, while `examples/70-render-to-string` demonstrates the same companion-owned head composition in a smaller prerender-style server render; `examples/README.md` and `docs/HEAD_MANAGEMENT.md` now tie those flows back to the minimal client-managed route metadata example.

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
- [x] Add accessibility audit workflow guidance for teams and CI.
	`docs/ACCESSIBILITY.md` now defines a repeatable audit loop for semantic review, keyboard flow, announcements, forms, overlays, automated browser coverage, and manual spot checks, and `docs/WORKFLOWS.md` now points testing guidance back to that accessibility review gate for app and CI validation.
- [x] Add accessibility regression recipes for routed and form-heavy apps.
	`docs/ACCESSIBILITY.md` and `docs/TESTING.md` now define concrete two-layer regression recipes for routed shells, forms, overlays, and composite widgets, splitting deterministic `js/wasm` checks from browser-only Playwright assertions and pointing directly at the focused accessibility example specs that already exercise focus movement, announcements, modal trapping, and route-update behavior.

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
- [x] Add a first-class server action contract for form submissions.
	`docs/SERVER_ACTIONS.md` now defines one server-owned mutation contract for progressive HTML posts and hydrated enhanced submits, covering success, redirect, validation failure, auth failure, and retryable failure without per-endpoint convention drift.
- [x] Add typed server-action result envelopes and mapping helpers.
	`ui.ServerActionResult`, `ui.ServerActionRedirect`, `ui.ServerActionFlash`, `ui.ServerActionRefresh`, `result.FormErrors()`, and `form.ApplyServerActionResult(...)` now standardize field errors, form errors, redirects, flash-style metadata, and revalidation hints on top of the existing `ui.UseForm` server-error projection seam.
- [x] Add integrated examples for progressive plus hydrated server actions.
	`docs/SERVER_ACTIONS.md`, `examples/87-ssr-secure-forms/README.md`, and `examples/README.md` now treat `87-ssr-secure-forms` as the progressive server-action reference and `86-atlas-commerce-os` as the hydrated `ui.UseForm` projection reference, making the current integrated server-action example slice explicit.
- [x] Define a first-class server function model beyond forms.
	`docs/SERVER_FUNCTIONS.md` now defines server functions as an explicit application-owned typed server-call pattern for non-form mutations and queries, declared through ordinary Go request or result structs plus same-origin handlers, kept above `fetch` and distinct from transport-specific RPC companions.
- [x] Define how server functions compose with loaders, revalidation, auth, and SSR.
	`docs/SERVER_FUNCTIONS.md` now makes the composition rules explicit: route loaders still own first paint, successful server functions trigger targeted cache invalidation or route revalidation, same-origin auth and CSRF stay server-owned, and non-hydrated paths must have an intentional fallback instead of assuming JavaScript.

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

- [x] Define whether typed browser RPC belongs in core interop or a supported companion package.
	`docs/RPC_TRANSPORT.md`, `docs/INTEROP.md`, `interop/README.md`, and the docs index now make the product decision explicit: typed browser RPC stays out of core `interop` and should prove itself first as a dedicated companion package layered on documented public APIs before any part of it is treated as normal framework surface.
	The current direction keeps protobuf-backed unary and streaming RPC in the transport-substrate category, with `grpc-tunnel`-style transport still only a candidate implementation reference rather than a justification for moving the feature into core.
- [x] Define the browser transport contract for gRPC-style RPC over WebSocket.
	`docs/RPC_TRANSPORT.md` now defines the first browser transport contract as an explicit WebSocket-owned companion transport with app-owned connection lifetime, mandatory handshake before RPC traffic, four logical RPC shapes, conservative reconnect behavior, and typed closure surfacing at both the connection and per-call levels.
	The current direction keeps replay and retry policy application-owned, requires a fresh handshake after reconnect, and treats all in-flight writes or bidirectional streams as failed on transport drop rather than guessing resumability from the framework layer.
- [x] Define protobuf code-generation and client-binding workflow for Go/WASM apps.
	`docs/RPC_TRANSPORT.md` now defines the intended workflow: keep `.proto` sources in an app-owned `proto/` tree, generate ordinary Go files with source-relative paths, commit the generated output for reproducible review, and run the normal `gwc` build and test flow against those generated files as ordinary source input rather than hiding regeneration inside launcher commands.
	The current reference shape points at `examples/100-ai-chat-wizard/proto/`, while transport-specific browser binding remains a separate companion-owned layer over the generated schema types instead of leaking transport rules into the base message package.
- [x] Define auth, session, and metadata propagation for tunneled RPC calls.
	`docs/RPC_TRANSPORT.md` now defines the current direction: prefer same-origin cookie or session-backed auth resolved at handshake time, treat bearer-token attachment as an explicit application-owned escape hatch, propagate only allowlisted connection or per-call metadata, and force reconnect or re-handshake on logout, tenant switch, or session-expiry changes.
	The RPC transport guidance now also aligns mutation-capable RPC calls with the repo's existing server-authoritative CSRF and redaction posture by requiring explicit origin-aware handshake policy, metadata-carried mutation proof when the server needs it, and secret-safe diagnostics that never expose raw cookies, tokens, or full auth metadata.
- [x] Define cancellation, timeout, and backpressure semantics for tunneled unary and streaming RPC.
	`docs/RPC_TRANSPORT.md` now defines the intended transport contract as context-driven: caller-owned cancellation and deadlines control unary calls and streams, deadline metadata propagates explicitly, late messages are ignored after local completion or unsubscribe, and send paths reject further writes after cancellation or terminal completion.
	The current direction also requires bounded connection and stream buffers, typed backpressure failures instead of silent drops or unbounded memory growth, and explicit pressure surfacing for client-streaming or bidirectional sends rather than hiding overload behind optimistic enqueue behavior.
- [x] Define SSR and hydration boundaries for browser-only RPC clients.
	`docs/RPC_TRANSPORT.md` now makes the ownership split explicit: SSR route handlers and loaders own first paint, `ui.SSRBootstrap` owns public resume data, and live tunneled RPC attaches only after hydration commit or on intentionally client-only routes that never had SSR in the first place.
	The current direction also forbids duplicate initial-data authority by treating bootstrap and loader output as the initial snapshot, then layering RPC revalidation or streaming updates on top after resume instead of auto-replaying the same data request during hydration.
- [x] Add observability, diagnostics, and actionable errors for typed RPC transport.
	`docs/RPC_TRANSPORT.md` now defines the intended instrumentation model: stable `rpc`-domain logs and events, opaque correlation ids, safe structured fields, stable diagnostic identifiers for handshake/auth/schema/backpressure/decode failures, and devtools snapshots that expose bounded connection and stream state without leaking secrets.
	The current direction also makes the error shape actionable by requiring recoverability, remediation text, and docs linkage for typed RPC failures instead of leaving socket errors, decode failures, and auth rejection as opaque transport noise.
- [x] Add one production-shaped RPC example that proves unary plus streaming value.
	`docs/RPC_TRANSPORT.md` now points at `examples/100-ai-chat-wizard` as the current reference example: it uses checked-in protobuf contracts, unary RPC flows for session and domain operations, server-streaming chat and speech over a WebSocket-backed gRPC tunnel, reconnect-aware browser behavior, and a production-shaped Go server instead of a toy echo transport.
	That example now serves as the proof point that one typed RPC contract can cover both unary and streaming browser flows without introducing a second bespoke protocol for the UI.
- [x] Define the multiplexed tunnel pattern for apps that route auth, streaming, and API calls through a single gRPC-web connection.
	`docs/RPC_TRANSPORT.md` now defines the shell-owned multiplexed tunnel pattern explicitly: establish one tunnel on WASM init, finish the auth handshake before traffic, reuse the connection for unary calls plus long-lived streams, survive silent token refresh at the shared-client layer, and cancel every active stream cleanly on sign-out.
- [x] Add guidance for per-connection versus per-call auth metadata propagation over gRPC tunnels.
	Distinguish identity that must travel at handshake time (session-scoped user identity) from metadata that legitimately differs per call (request-scoped role hints, tenant context, or idempotency keys) so tunnel implementations do not accidentally carry stale connection-level credentials into calls made after a token refresh.
	`docs/RPC_TRANSPORT.md` now makes that split explicit: handshake metadata owns authenticated session identity plus default tunnel context, per-call metadata is limited to request-varying values such as idempotency or tracing ids, and any true identity change must tear down the tunnel and re-handshake instead of mutating stale connection-scoped credentials in place.

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
- [x] Add router-level unauthorized and authorizing UI states.
	`router/router.go` now renders `router.Options{Unauthorized, Authorizing, GuardPending}` through the same route-level fallback path used by loader states, `router/router_test.go` covers denied and retryable guard decisions, and `router/README.md` now documents route-level fallback props instead of claiming those options are only future surface.
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
- [x] Add typed route-definition and reverse-routing contracts.
	Provide one supported way to define routes so links, redirects, params, and query values can be generated and validated from typed Go APIs instead of manually repeating string paths across loaders, navigation code, and tests.
	`router/contracts.go` now adds `router.RouteContract` with `DefineRoute` and `MustDefineRoute`, plus validated `Path` and `Href` builders for raw maps or typed provider structs, `router/contracts_test.go` covers reverse-routing validation and escaping, and `router/README.md` shows the supported pattern for shared route definitions.
- [x] Add route-contract tooling for navigation, metadata, and generated examples.
	Decide whether typed route contracts should stay runtime-only helpers or also emit code-generated route manifests for links, metadata ownership, prerender enumeration, and starter examples so larger apps can avoid stringly-typed route drift.
	`docs/ROUTE_CONTRACTS.md` now records the current decision to keep `router.RouteContract` runtime-first in core, leave manifest generation and prerender enumeration in companion or app-owned layers for now, and require any future tooling to build from the same runtime contract definitions instead of inventing a second route schema.
- [x] Document runtime-owned auth patterns for single-shell GWC applications.
	Define WASM-owned sign-in, session refresh, and sign-out flows that complete inside the running client without a server-rendered detour page or full-page reload; specify where session tokens live (memory versus secure browser storage), how background refresh is scheduled before expiry, and what happens to in-flight route loaders and RPC streams on refresh failure.
	`docs/AUTH_AND_SESSION_INTEGRATION.md` now defines the single-shell runtime pattern explicitly, covering one-router-tree sign-in and workspace flow, memory-first token handling, shell-owned background refresh scheduling, and the requirement to cancel protected loaders and re-handshake or close RPC streams when refresh fails.
- [x] Add auth-route state consistency guidance for silent token refresh.
	Specify the expected behavior when a token renews mid-route: active route guards must not re-fire unnecessarily, protected-route loaders must not reload already-fetched data, and auth-derived atoms must update without causing a momentary flash of unauthorized UI or a spurious rerender cascade.
	`docs/ROUTER_AUTH.md` now defines that mid-route refresh contract explicitly: same-identity refresh should not re-fire active guards or cold-reload already-allowed protected loaders, auth-derived atoms should update in one coordinated transition without an unauthorized flash, and only a real principal or tenant change should trigger the full guard-and-loader reset path.
- [x] Define sign-out as an atomic state-reset operation.
	Specify that sign-out must clear all auth-derived atoms, cancel in-flight RPC streams, invalidate user-scoped cache entries, remove session storage, and transition the router to the sign-in or landing route in one coordinated step so no component is left holding stale identity state after logout.
	`docs/AUTH_AND_SESSION_INTEGRATION.md` now defines sign-out as one ordered reset boundary: freeze new auth work, clear auth-derived state, cancel authenticated transports, purge user-scoped cache and persisted session data, remove browser-held credentials, and only then replace the route with sign-in or public landing UI.
- [x] Add a single-shell auth example where marketing, sign-in, and workspace routes coexist.
	Demonstrate a GWC router tree where public landing pages, a sign-in/sign-up flow, and a protected workspace all live in one WASM client without any server-HTML hand-off between auth state transitions; show the router moving cleanly from a public marketing route to a protected workspace route after sign-in without a full page reload.
	`examples/106-single-shell-auth` now demonstrates one running hash-router shell with public marketing routes, a bounded return-to sign-in flow, route-contract-backed navigation targets, and a guarded workspace route that opens after sign-in without a full-page reload. `examples/README.md` now lists it as both an integrated auth example and a router reference.

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
- [x] Add a per-user preference persistence pattern.
	Define how user-scoped settings (active AI model, theme, layout density, notification preferences, and similar) are stored and restored across WASM sessions using the authenticated identity as the partition key; specify that the preference store must be fully loaded before the first workspace render so the initial paint does not flash with wrong or default settings.
	`docs/STATE_ARCHITECTURE.md` now defines the preferred pattern explicitly: keep preferences in one small atom or snapshot slice, persist them under an authenticated-identity key such as `preferences:<userID>`, and block the first protected workspace render until that preference slice is restored so the shell does not flash the wrong defaults.
- [x] Define preference isolation across users sharing a browser profile.
	Specify that preference stores keyed by user identity must be cleared or re-partitioned on sign-out so one user's settings cannot be read or silently applied when a different account signs in on the same device.
	`docs/STATE_ARCHITECTURE.md` now makes that isolation rule explicit: sign-out or user-switch must clear the active preference slice, stop reading from the previous identity's storage key before the next workspace render, and never fall back to a generic last-used preference snapshot for a different authenticated user.
- [x] Add preference synchronization across browser tabs for the active session.
	Define how user preference mutations (such as switching the active AI model or changing a display setting) propagate from the originating tab to all other open tabs using the existing cross-tab channel surface, so all windows reflect the current preference state without requiring a manual refresh.
	`docs/CROSS_TAB.md` now defines the active-session preference-sync pattern explicitly: after auth and preference restore, open one identity-scoped channel, apply the mutation locally first, publish a small revisioned preference message, fan it out only to tabs with the same active identity, and ignore older or different-user updates.
- [x] Define a conversation history and session persistence contract for long-lived workspaces.
	Specify how conversation threads, draft messages, active canvas state, and scroll position are persisted per user (IndexedDB or server-backed) so session metadata survives WASM restart and tab restore; define the maximum retained history, expiry policy, and what must be wiped on sign-out.
	`docs/WORKSPACE_PERSISTENCE.md` now defines the current contract explicitly: keep authoritative conversation history server-backed when needed, persist per-user browser mirrors and resume state under stable user-plus-thread keys, restore only after auth resolves, bound browser-resident history to the 50 most recent entries with a 30-day inactivity expiry by default, and purge drafts, canvas state, scroll markers, and browser conversation mirrors on sign-out or user switch.

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
- [x] Evaluate whether typed RPC streams should become an optional transport for multi-client coordination.
	`docs/MULTI_CLIENTS.md` now defines the decision boundary explicitly: presence, discovery, invalidation, and light browser-local fanout stay on native multi-client topics, while server-authoritative, schema-heavy, or long-lived live-update flows may justify an optional RPC companion transport without replacing the existing control plane.
	External implementation reference when this work starts: `grpc-tunnel` repo.
- [x] Define interoperability rules between multi-client topics and RPC-backed live streams.
	`docs/MULTI_CLIENTS.md` now defines the ownership split explicitly: presence, discovery, invalidation, and browser-local popup or opener query/result flows stay native, while server-authoritative live data and shared-session streams move to RPC, with coexistence allowed only when one wire contract remains authoritative per concern.
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
- [x] Add a serialization boundary verifier across app transport edges.
	`docs/STATE_TRANSFER.md` now defines one planned serialization-boundary verifier contract that spans SSR bootstrap, hydration resume, worker envelopes, RPC payloads, and multi-client or shared-session transport edges through one shared report vocabulary, while still noting that the analyzer is not implemented yet.
- [x] Add static serialization analysis for non-deterministic and non-serializable values.
	`docs/STATE_TRANSFER.md` now defines the intended static-analysis pass for typed payload registration and transport framing sites, covering functions, closures, browser-native handles, hidden mutable references, and time- or random-dependent values while distinguishing hard failures from nondeterministic-value warnings.
- [x] Add runtime payload sampling and bootstrap snapshot verification.
	`docs/STATE_TRANSFER.md` now defines the intended runtime sampling points, redacted sample contents, and verification goals for SSR bootstrap, hydration resume, worker messaging, RPC framing, and multi-client or shared-session transport so boundary-size, secret-leakage, and shape-drift checks are explicit even though the sampler is not implemented yet.
- [x] Add serializable-safe type tagging and policy modes.
	`docs/STATE_TRANSFER.md` now defines the intended payload tags (`serializable-safe`, `allowlisted`, `redacted`, and `boundary-owned`) plus `strict`, `warn`, and `allowlist` verifier modes so serialization-boundary enforcement can ratchet from advisory diagnostics to CI-blocking checks once the analyzer exists.
- [x] Add HTML and bootstrap boundary analyzers for SSR output.
	`docs/STATE_TRANSFER.md` now defines the intended SSR-output analyzer for rendered HTML, inline bootstrap scripts, bootstrap-reference scripts, and sidecar payloads, covering secret leakage, oversized inline payloads, unstable ordering, contradictory bootstrap metadata, and resume-time mismatch risks even though the analyzer is not implemented yet.
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
- [x] Add examples and benchmarks for streaming SSR.
	`docs/STREAMING_SSR.md` now points at `examples/17-ssr-routing` as the current reference example and at `examples/17-ssr-routing/ssr_routing_benchmark_test.go` as the non-streaming SSR baseline. That route family already exercises loader-backed routed views plus nested async UI, so future streaming work has a concrete example and benchmark surface to extend instead of starting from a toy shell.
- [x] Add a minimal real streaming implementation for one routed app shape.
	`examples/18-ssr-server-routing` now streams the `/docs/:section?tab=loader` route shape by flushing the SSR shell with an explicit placeholder first, then sending a deferred docs panel later before the wasm boot script runs. `examples/18-ssr-server-routing/server_test.go` covers the streamed placeholder, flush behavior, and deferred replacement script, so the repo now has one concrete routed streaming slice beyond the design notes.
- [x] Add failure-mode tests for late segment errors and nested streamed layouts.
	`examples/18-ssr-server-routing` now supports `?stream=error` and `?stream=nested` on the streamed docs route, and `examples/18-ssr-server-routing/server_test.go` covers explicit late error replacement plus nested streamed layout output on top of the shell-first streamed route. That gives the first routed streaming slice concrete non-happy-path coverage before any broader core API exists.
- [x] Add proxy-aware streaming verification fixtures.
	`examples/18-ssr-server-routing/streaming_proxy_test.go` now wraps the streamed docs route with a buffering proxy-like writer and a gzip-buffered writer. Those fixtures verify the production-safety fallback rule for the first streaming slice: even when flushes are effectively defeated by buffering or compression, the response still completes as a correct full HTML document with the placeholder replacement and wasm boot path intact.

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
- [x] Add examples and docs for end-to-end observability.
	`docs/OBSERVABILITY.md` now includes an end-to-end walkthrough that maps the shipped `ui.ObserveSSR(...)`, observed bootstrap helpers, and hydration observability options onto one server-rendered request, one routed page load, and the current async-resource story. `examples/18-ssr-server-routing/README.md` now points at that recipe as the concrete request-time SSR and hydration reference example.
- [x] Add first-party integration recipes for traces, metrics, perf marks, and external error reporting.
	`docs/OBSERVABILITY.md` now includes first-party adapter recipes that map the shipped `ui.SSRObservation` event shape into OpenTelemetry-style traces, structured metrics, browser performance marks, and hosted error-reporting sinks while keeping the framework transport-agnostic and the app-owned instrumentation boundary explicit.

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
- [x] Add a prerender-to-files pipeline.
	`prerender.Export(...)` now provides a first-party companion file writer that maps routes to stable HTML targets, emits optional JSON or CBOR bootstrap sidecars, and writes the generated files to disk; `docs/PRERENDER.md` now documents that pipeline and its remaining app-owned responsibilities.
- [x] Define route enumeration for prerendered apps.
	`docs/PRERENDER.md` now defines explicit route lists, parameter expansion hooks, application-owned dynamic enumeration, and fallback treatment for routes that cannot be known safely at build time.
- [x] Define asset and bootstrap output conventions for prerender.
	`docs/PRERENDER.md` now defines the intended HTML, sidecar bootstrap, manifest, and copied-asset output shape for prerendered sites without overclaiming a shipped exporter implementation.
- [x] Add partial-hydration or client-resume expectations for prerendered output.
	`docs/PRERENDER.md` now records the intended split between fully static pages, fully hydrated pages, and future selective activation, while keeping prerendered resume on the normal hydration contract.
- [x] Define a first-class islands or selective-hydration model.
	`docs/PRERENDER.md` now defines selective activation as explicit multi-root ownership across static regions, island roots, and route shells, including how those boundaries compose with routing, `ui.Lazy(...)`, `ui.AsyncBoundary(...)`, and hydration mismatch guarantees; `docs/HYDRATION.md` now points remaining backlog work at automatic orchestration beyond those explicit island roots.
- [x] Add one islands-style reference example and budget-driven validation.
	`examples/101-static-islands` now ships a content-heavy prerendered marketing page with two explicitly hydrated island roots, plus an in-page budget panel that reports startup, per-island hydration, and latest interaction timings; `examples/README.md` and `examples/MANUAL_TESTING.md` now point at that example as the selective-activation reference surface.
- [x] Add invalidation and rebuild guidance for prerendered content.
	`docs/PRERENDER.md` now defines the intended rebuild triggers for route content, shared layouts, asset manifests, and expanded route data in local development and CI.
- [x] Add a first-party static export example.
	`examples/102-static-export-site` now ships a multi-route marketing/docs export command built on `prerender.Export(...)`, writes `/`, `/pricing`, and `/docs/getting-started` to a `dist/` tree, and documents serving that output from a plain static host with no custom Go request runtime.

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
- [x] Add example apps that stress real asset delivery concerns.
	Ship a content-rich example with responsive images, route-scoped preload hints, hashed asset output, and lazy media loading so the public story is validated end to end.
	`examples/102-static-export-site` now acts as the asset-delivery reference: it exports content-rich routes, copies a tiny hashed asset set into `dist/static/`, emits route-scoped preload and prefetch hints through `head.Render(...)`, renders responsive hero media with `srcset` and `sizes`, and keeps secondary media lazy in the exported HTML. `docs/ASSETS.md` and `examples/README.md` now point at that example explicitly.

### Browser support and compatibility policy

- [x] Publish an explicit browser support matrix.
	List the minimum supported desktop and mobile browsers, including Safari and mobile Safari expectations, so adopters know which environments the runtime and examples are expected to work in.
- [x] Define the required browser feature baseline for the wasm runtime.
	Document which web platform features are assumed by core packages, router behavior, fetch helpers, workers, devtools, and SSR hydration so compatibility is based on concrete capabilities rather than vague â€œmodern browserâ€ language.
- [x] Define the project stance on polyfills and shims.
	`docs/BROWSER_SUPPORT.md` now defines the no-framework-polyfill stance, the application-owned compatibility boundary, and how new feature dependencies should be communicated.
- [x] Add progressive-enhancement boundaries for partial support cases.
	`docs/BROWSER_SUPPORT.md` now defines what remains available for prerendered and SSR-delivered content when JavaScript, wasm startup, or optional browser APIs are unavailable.
- [x] Add mobile Safari and constrained-device validation coverage.
	`docs/BROWSER_SUPPORT.md` now defines a concrete Mobile Safari and constrained-device release-check matrix, including the representative example set for hydration, forms, selective activation, offline flows, and product-shaped wasm shells; `examples/MANUAL_TESTING.md` now mirrors that pass as the minimum browser-family gate before calling those workflows broadly compatible.
- [x] Define compatibility expectations for workers, offline features, and advanced APIs.
	`docs/BROWSER_SUPPORT.md` now defines the capability-check and reduced-functionality expectations for workers, offline flows, cross-tab features, and other advanced interop APIs.
- [x] Add browser-compatibility CI coverage or release checks.
	`.github/workflows/browser-compatibility.yml` now runs a focused Playwright compatibility matrix on Chromium, Firefox, and WebKit against `71-hydrate`, `73-ssr-bootstrap`, and `101-static-islands` through `examples/playwright.browser-compat.config.ts`, and `docs/BROWSER_SUPPORT.md` now records that automated gate alongside the separate Mobile Safari and constrained-device manual pass.
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
- [x] Add a full offline-first reference app that combines the current pieces.
	`docs/PWA.md` and `examples/README.md` now define the current offline-first reference slice explicitly: `97-pwa-installability`, `97-pwa-offline-cache`, and `97-pwa-multi-client` are treated as one coherent reference topology for installability, shell caching, offline navigation, durable replay, multi-tab ownership, and logout-safe purge behavior instead of leaving that combined model implicit.
- [x] Define a framework-level offline-first sync model beyond queue replay.
	`docs/PWA.md` now defines the supported offline-first sync model explicitly: optimistic local state, durable mutation intent in the queue, reconstructible read caches, cross-tab replay ownership, and one diagnostics surface, together with the intended reconnect lifecycle from local intent through replay, invalidation, and repair handling.
- [x] Add stronger conflict-resolution and operator-recovery patterns for durable replay.
	`docs/PWA.md` now defines the intended conflict and operator-recovery branches for durable replay, covering idempotency and dedup keys, stale revisions, duplicate acceptance, superseded local intents, permission loss, validation failure, and app-owned review or repair workflows instead of one generic retry loop.
- [x] Add first-class merge policies and reconnect reconciliation semantics.
	`docs/PWA.md` now defines the reconnect reconciliation order explicitly and names the supported merge-policy families: server-wins, client-wins, revision-aware reject, field-level merge, and operator-reviewed handling, with policy selection scoped per mutation family or entity type instead of one silent global default.
- [x] Add per-entity sync health and conflict state inspection.
	`docs/PWA.md` now defines the intended per-entity sync-health vocabulary (`clean`, `pending`, `replaying`, `conflicted`, `blocked`, and `stale`) plus the inspection fields apps should surface for rows, records, and forms when offline work is richer than one queue summary.
- [x] Add conflict-oriented UI helpers and reference workflows.
	`docs/PWA.md` now defines the intended conflict-oriented UI patterns for offline-capable apps: conflict banners, per-record repair screens, replay review queues, reconnect summaries, and the interaction rules that connect those screens back to explicit queue outcomes instead of raw queue dumps.
- [x] Add operational guidance for long-lived offline data management.
	`docs/PWA.md` now defines the intended operating model for storage pressure, stale queue expiry, long-disconnect replay recovery, corruption handling, and purge-on-logout or user-switch behavior so long-lived offline data is treated as an operational concern instead of a browser-storage afterthought.

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
- [x] Add security regression tests for critical surfaces.
	Regression coverage now explicitly locks bootstrap serialization and script-id escaping in `ui/ssr_bootstrap_test.go`, SSR escaping in `internal/runtime/ssr_test.go`, interop boundary decode failures in `interop/interop_native_test.go`, and password-field logging redaction in `logging/redaction_test.go`.
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
- [x] Add examples for staged rollout and environment-aware apps.
	`examples/107-staged-rollout-config` now demonstrates a feature-gated hash route, a browser-safe environment-configured API endpoint, and one SSR bootstrap snapshot that keeps the server-rendered route decision aligned with hydrated client routing.

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
- [x] Evaluate post-link wasm optimization tooling.
	`gwc release` now supports the explicit opt-in flag `-post-link-opt wasm-opt`, resolves `wasm-opt` from `PATH` or via `npx --yes --package binaryen wasm-opt`, rewrites the emitted wasm artifact before compression sidecars are generated, and records the optimizer metadata in the release summary and manifest without changing the default release path.
- [x] Add per-package or symbol-level wasm size attribution.
	`gwc release` now supports `-size-attribution packages`, which writes `wasm-package-size-attribution.json` beside the release manifest using `go list -deps -json -export` under `GOOS=js GOARCH=wasm`, records per-package compiled-archive and source-byte counts, and exposes the attribution artifact in both the manifest and JSON release summary.
- [x] Add release-to-release wasm size diff reports with likely culprit summaries.
	`gwc release` now supports `-compare-manifest <baseline>`, writes `wasm-release-size-diff.json` beside the new release manifest, reports per-artifact size deltas against the prior manifest, and promotes the largest positive per-package archive deltas from paired `-size-attribution packages` artifacts as likely culprits in both the diff report and JSON release summary.
- [x] Define asset-manifest and build-output conventions for optimized wasm releases.
	`docs/WASM_RELEASES.md` now defines the intended output directory shape around the raw wasm artifact, compressed sidecars, and `wasm-release-manifest.json`, and `tools/build-wasm-release.ps1` emits that convention directly.
- [x] Add reproducible release-build guidance.
	`docs/WASM_RELEASES.md` now defines the intended toolchain, flag, manifest, and hash-record expectations for reproducible release builds.
- [x] Add startup-cost measurements for release artifacts.
	`gwc release` now supports `-startup-measure browser`, which emits `wasm-startup-report.json` by loading the release artifact through a generated Playwright probe page, records browser navigation and wasm resource timings, tracks transport encoding plus gzip decompression timing when available, measures instantiate duration, and captures a synthetic first-interaction latency so release size work is tied to observed startup behavior instead of artifact bytes alone.
- [x] Add release smoke validation to the launcher.
	`gwc release` now supports `-validate-smoke`, which reuses the launcher-owned startup probe path to perform a boot smoke check, validates that the release manifest still parses as a valid js/wasm release record, confirms on-disk artifacts still match manifest bytes and hashes, verifies launcher-served wasm MIME behavior, and emits `wasm-release-validation.json` beside the release manifest before the release is reported as ready.

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
- [x] Add `tools/gwc` as the canonical launcher entrypoint.
	`tools/gwc/main.go` now prints and dispatches a single `go run ./tools/gwc <command>` command surface, and `docs/ONBOARDING.md` plus `tools/README.md` now anchor the repo workflow around that launcher for scaffold, dev, test, build, release, diagnostics, and examples flows.
- [x] Add a preset-first `gwc start` TUI.
	`go run ./tools/gwc start` now launches an interactive Bubble Tea scaffold flow that begins with a preset picker, captures project metadata, and offers an immediate post-generation `gwc dev` path for the generated app.
- [x] Add feature-matrix scaffold generation.
	`gwc start` scaffold output is now generated from the selected capability set, with capability cards rendered into `main.go`, a generated `FEATURE_MATRIX.md` capability contract, and capability-owned extra files (including browser test stubs when `browser-tests` is selected) so starter output is composable instead of one hard-coded template.
- [x] Add scaffold preset definitions for the main adoption modes.
	The `gwc start` preset set now includes `minimal-client`, `routed-spa`, `ssr-app`, and `reference-app`, giving the launcher built-in adoption-mode choices instead of one generic starter.
- [x] Add starter output rules that keep generated apps disposable.
	`docs/STARTER_OUTPUT_RULES.md` now defines the starter-output contract explicitly, generated scaffold `README.md` now embeds disposable ownership rules and links to `FEATURE_MATRIX.md`, and scaffold generation keeps capability-owned extras narrow so generated apps stay readable and easy to replace.
- [x] Add launcher-owned project metadata for generated apps.
	Generated starters now write `gwc-start.json` with preset metadata plus launcher-owned tooling defaults for app path, HTML shell, wasm output, dev host or port, and release settings so later commands can resolve project intent consistently.
- [x] Add at least one maintained starter application for the common path.
	The `reference-app` scaffold preset now generates a maintained baseline starter with route-state controls, async-data status flow, shared-state actions, form submit/reset workflow, generated browser smoke-test stubs, and launcher-owned release defaults so teams can start from a realistic app shell instead of a toy counter.
- [x] Add starter variants for the major adoption modes.
	`gwc start` now carries and tests distinct starter variants for client-only (`minimal-client`), dashboard/content routed apps (`routed-spa`), SSR plus hydration (`ssr-app`), and a broader integrated baseline (`reference-app`), with onboarding docs now mapping those presets directly to adoption choices.
- [x] Define upgrade and template-sync guidance for starter-based apps.
	`docs/ONBOARDING.md` now defines the intended starter-upgrade model around starter-specific release notes, core migration docs, and explicit support windows instead of blind monorepo diffing.
- [x] Add a one-command local bootstrap workflow.
	`go run ./tools/gwc bootstrap` now runs launcher prerequisite checks and then enters the starter scaffold flow, while `go run ./tools/gwc bootstrap -examples` provides a one-command examples bootstrap path, so users can start from a working starter or example without assembling several manual steps.
- [x] Add scaffold-time prerequisite checks and optional setup steps.
	`gwc start` now runs early prerequisite checks for Go and runtime assets (plus Node/npm/Playwright when the selected preset includes browser tests), and scaffold generation now supports optional post-init setup skips through `-skip-tidy` and `-skip-runtime-assets` while `gwc doctor` remains the full diagnostic surface.
- [x] Document environment prerequisites and platform expectations clearly.
	`docs/ONBOARDING.md` now defines the intended Go, Node, browser, and Windows/macOS/Linux baseline in one place and points to the browser support contract where relevant.
- [x] Add a â€œchoose your pathâ€ onboarding flow for new adopters.
	`docs/ONBOARDING.md` now defines the intended path chooser for client-rendered, routed, SSR, forms-heavy, and static/prerender-oriented adoption modes.

- [x] Add a `gwc seed` command for local dev identity and fixture provisioning.
	`go run ./tools/gwc seed` now discovers and runs repo or app-local seed packages, defaults to the chat-wizard fixture seeder, supports `-command` and `-db-path` overrides, and emits known local credentials in the JSON/plain-text summary so contributors can provision deterministic login state without manually wiring `CHAT_DB_PATH` or invoking a separate seed script directly.
- [x] Add provider-switching and model-catalog test scenarios to the local dev workflow.
	The chat-wizard server now accepts `CHAT_PROVIDER_STUBS=all` (or a comma-separated provider list) to expose OpenAI, Anthropic, and Cerebras model catalogs with deterministic stub replies when real credentials are missing, the example README now documents that local-stub workflow, and focused provider/app tests now cover stub-backed model catalog exposure plus cross-provider `SetSelectedModel` switching without service restarts or manual config edits.

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
- [x] Improve incremental rebuild behavior for small source edits.
	`tools/livereload/livereload.go` now lets the active compile finish and queues exactly one follow-up rebuild for changes that land mid-build, so bursty local edits no longer waste work by repeatedly restarting the build loop or blocking change capture until `cmd.Wait()` returns.
- [x] Add clearer surfacing for build progress and failure states.
	The live-reload status model now exposes explicit phases such as checking, compiling, waiting for reload, serving fresh output, and blocked-on-error, and the in-browser GWC status icon and panel now show whether the dev loop is compiling, serving stale output, waiting on reload, or blocked on an error.
- [x] Add launcher-visible dev status endpoints and summaries.
	The livereload dev server now exposes `/__gwc/status` with current mode, last classification, last build, current error state, hot-reload eligibility, and listening URLs, and `gwc dev` now prints the status URL in the resolved plan for livereload-backed app runs.
- [x] Add an interactive `gwc dev` status TUI.
	`gwc dev -tui` now starts the livereload-backed dev loop under a small Bubble Tea status surface that polls `/__gwc/status` and shows the resolved plan, current compile phase, stale-versus-fresh output state, latest failure, hot-reload eligibility, listening URLs, and basic recovery hints.
- [x] Add recovery guidance for broken local development loops.
	`docs/TROUBLESHOOTING.md`, `docs/ONBOARDING.md`, and `tools/README.md` now document the current recovery path for stale wasm output, broken example serving, missing runtime bootstrap files, and local build-versus-served-output drift in the repo's supported manual dev loop.
- [x] Add example workflows for repo contributors versus framework consumers.
	`docs/ONBOARDING.md` and `tools/README.md` now split the practical commands and expectations between monorepo contributor work and separate app-consumer work, so the documented `gwc` loop scales beyond editing examples inside this repository.

### IDE and editor integration

- [x] Define the first-party IDE support boundary.
	`docs/IDE_INTEGRATION.md` now defines the intended support boundary explicitly: first-party editor work is VS Code-focused first, built on ordinary `gwc` commands and `gopls`-compatible Go conventions, while broader multi-editor integration remains follow-up work rather than an implied current promise.
- [x] Add launcher-owned task and problem-matcher definitions for the supported editor workflow.
	`docs/IDE_INTEGRATION.md` now points at `docs/examples/gwc-vscode.tasks.json` as the baseline VS Code task bundle for `gwc dev`, `gwc test`, `gwc verify`, and `gwc doctor`, with initial Go-file problem matching on test and verify output while richer editor diagnostics remain follow-up work.
- [x] Add snippets, boilerplates, and hover-friendly discoverability for the public surface.
	`docs/IDE_INTEGRATION.md` now points at `docs/examples/gwc-vscode.code-snippets.json` as the baseline snippet bundle for stateful components, browser mounts, routing, and testing entrypoints so the public surface is easier to discover from the editor even before richer hover and symbol-help work lands.
- [x] Add focused snippets for routed apps, SSR bootstrap, forms, and async resources.
	`docs/examples/gwc-vscode.code-snippets.json` now includes focused workflow snippets for loader-backed routes, `router.UseRevalidator(...)`, SSR bootstrap registration and hydration, typed `ui.UseForm(...)` wiring, and `fetch.UseCachedResource(...)`, and `docs/IDE_INTEGRATION.md` now calls out those patterns explicitly as the recommended architecture-oriented editor shortcuts.
- [x] Add hover docs that explain recommended usage, not only signatures.
	`docs/IDE_INTEGRATION.md` now defines the intended hover shape explicitly: each editor-visible symbol help entry should explain what the API solves, when it is preferred over nearby alternatives, the common misuse to avoid, and the workflow doc or example to consult next for hooks, state, router, async, SSR, hydration, and devtools surfaces.
- [x] Add editor-visible project diagnostics and task integration.
	`docs/IDE_INTEGRATION.md` now defines the intended editor-visible project-diagnostics sources and categories explicitly, using `gwc doctor`, `gwc verify`, scaffold metadata checks, and dev-entrypoint resolution to surface missing `wasm_exec.js`, broken `gwc-start.json`, unresolved dev entrypoints, missing browser-test prerequisites, and other launcher-blocking issues from the supported editor workflow.
- [x] Project scaffold, launcher, and audit diagnostics into editor problems.
	`docs/IDE_INTEGRATION.md` now defines the intended editor-problem contract explicitly: launcher and scaffold findings should carry stable `GWC-*` codes, deliberate severity, short summaries, remediation text, and source attribution, with `gwc doctor`, scaffold validation, future auditor findings, and budget failures identified as the first projected sources.
- [x] Evaluate navigation and refactor support for routes, typed HTML builders, and scaffolded apps.
	`docs/IDE_INTEGRATION.md` now defines the intended navigation and refactor boundary explicitly: keep normal `gopls` symbol navigation as the baseline, add framework-specific help only for route paths, loader ownership, and scaffold entrypoints, and avoid overpromising global starter or route rewrites before those contracts are stronger.
- [x] Add route-symbol indexing and reference discovery.
	`docs/IDE_INTEGRATION.md` now defines the intended route-indexing scope explicitly: index registered route paths, layout chains, loader ownership, and scaffolded route entrypoints, then expose that through go-to-definition, find-references, and route-detail navigation instead of relying on grep alone.
- [x] Add lightweight code actions for common framework fixes.
	`docs/IDE_INTEGRATION.md` now defines the intended lightweight code-action scope explicitly: repair missing launcher tasks, point at or add missing `gwc-start.json` fields, generate baseline test or component stubs, and handle documented auditor suppressions through narrow, reviewable fixes instead of broad opaque rewrites.

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
- [x] Extend `gwc doctor` into a golden-path app auditor.
	`go run ./tools/gwc doctor -audit` now emits a dedicated golden-path audit section in text and JSON output, fails the command when audit findings fail, and bootstraps the future rule surface with first-pass app-shape checks for a detectable app entrypoint, HTML shell presence, and scaffold metadata anchoring instead of limiting `doctor` to environment checks alone.
- [x] Add static golden-path rules for state and ownership boundaries.
	The golden-path audit now statically fails on obvious client/server ownership leaks such as js/wasm files importing server packages or server-only packages and non-js files importing `syscall/js`, and it warns when client files mix `fetch.UseCachedResource` with direct browser-storage access as a first-pass duplicated local-versus-shared state ownership heuristic. `tools/gwc/examples_test.go` now covers both the passing baseline audit path and these ownership-rule detections directly.
- [x] Add route-shape and delivery audit rules.
	`gwc doctor -audit` now ships a first-pass route and delivery heuristic: it warns when one app root carries multiple HTML shells, when a route tree grows to multiple registrations without any `ui.Lazy` split signal, and when marketing-style routes appear without static/prerender delivery hints. `tools/gwc/examples_test.go` already covers both the passing audit path and the warning path for these route-shape heuristics.
- [x] Add mutation and resilience audit rules.
	The golden-path audit now warns on files that expose obvious mutation-shaped code but carry no retry, idempotency, rollback, conflict, offline, replay, timeout, or deadline signal, giving `gwc doctor -audit` a first-pass mutation-resilience review instead of silently treating every write path as production-ready.
- [x] Add suppressions, baselines, and policy levels for audit adoption.
	`gwc doctor -audit` now supports `-audit-policy strict|advisory`, repeated `-audit-suppress` rule-name suppressions, `-audit-write-baseline <path>` for writing a checked-in JSON audit baseline, and `-audit-baseline <path>` for replaying that baseline as accepted findings. `tools/gwc/examples_test.go` covers advisory-mode output, explicit suppression, and baseline write/read flows so teams can adopt the auditor incrementally instead of failing every historical finding on day one.
- [x] Add evidence-backed audit rules for startup-cost and ownership mistakes.
	The golden-path audit now emits concrete file-backed evidence for ownership mistakes through explicit file/import summaries, and it adds a startup-cost evidence rule that warns on oversized client-owned source files and dependency-heavy client files with exact file paths and byte/import counts instead of vague style-only warnings.
- [x] Add runtime evidence collection for golden-path audits.
	The golden-path audit now collects local runtime artifacts such as `.wasm` payloads and `wasm-startup-report.json`, surfaces concrete runtime evidence for payload size plus observed `readyMs` and `interactionMs`, and folds that into `gwc doctor -audit` so architectural warnings can cite local runtime artifacts instead of staying purely source-heuristic.
- [x] Add machine-readable golden-path audit output and severity levels.
	The golden-path audit JSON now emits stable `ruleId`, normalized `severity`, explicit `locations`, and dedicated `remediation` guidance on audit checks, and the audit builders stamp those fields directly so `gwc doctor -json` stays stable for editor and CI consumers without scraping the terminal report.
- [x] Add CI and generated-workflow entrypoints for the golden-path auditor.
	`gwc verify` now accepts `-audit-min-severity off|error|warning|info` and carries the golden-path audit in both text and JSON summaries, `docs/examples/gwc-vscode.tasks.json` now wires the supported verify and doctor tasks through that audit surface, and `.github/workflows/release.yml` now runs `gwc verify -audit -audit-min-severity error` against a real example so architecture checks join ordinary delivery instead of staying a side command.
- [x] Add scaffold-generated baseline tests keyed to selected features.
	`gwc start` now emits a baseline `starter_test.go` file for generated apps, always validates the recorded feature selection through `gwc-start.json` and `FEATURE_MATRIX.md`, and adds feature-specific assertions for routed, forms, fetch, and browser-test scaffolds so new starters begin with a real `go test ./...` path instead of an empty test folder.
- [x] Add starter-generated GitHub Actions workflows for the default CI path.
	`gwc start` now emits `.github/workflows/ci.yml` for generated starters, using `actions/setup-go`, `go test ./...`, and a js/wasm `go build -o main.wasm .` baseline so new standalone apps begin with a functional GitHub Actions CI path instead of reverse-engineering repo workflows.
- [x] Verify scaffolded GitHub Actions against generated starters.
	`tools/gwc/start_test.go` now covers the emitted CI workflow across multiple generated starter shapes, checking that `.github/workflows/ci.yml` stays aligned with `starter_test.go`, `main.go`, and optional browser-smoke output, while the end-to-end generated-starter smoke test still builds wasm and runs the emitted `go test ./...` baseline so scaffold CI cannot silently drift from what `gwc start` actually produces.
- [x] Make `gwc start` generate standalone apps in user-owned workspaces by default.
	The scaffold flow now targets a user-owned generated-project root outside the framework checkout by default, and the TUI explicitly describes the result as a standalone project.
- [x] Split `gwc start` into standalone-app mode versus contributor-linked mode.
	`gwc start` now accepts `-mode standalone|contributor-linked`, keeps `standalone` as the default generated-app path, surfaces the selected mode in the scaffold confirmation UI, and writes a local `replace` back to the current framework checkout when contributor-linked mode is chosen so framework contributors have an explicit source-linked path without changing the default standalone lifecycle.
- [x] Remove the default scaffold dependency on the current framework checkout.
	`gwc start` now emits a standalone `go.mod` without local `replace` wiring to the current framework checkout, so generated apps stay portable even after the original repo clone is moved or deleted.
- [x] Add explicit generated-project location policy for Windows, macOS, and Linux.
	The launcher now implements OS-aware generated-project defaults, using `Documents` on Windows and macOS, `Documents` or `Projects` fallback behavior on Linux, and override support through launcher config.
- [x] Add scaffold metadata that records project ownership and framework source mode.
	Generated `gwc-start.json` files now carry an `ownership` section with `projectOwnership` and `frameworkSourceMode`, distinguishing the default standalone `module-proxy` path from contributor-linked `local-replace` scaffolds so later launcher commands can reason about how the starter is coupled to the framework checkout.
- [x] Add metadata-first resolution tracing and fallback diagnostics for launcher commands.
	Launcher resolutions now carry machine-readable `resolution` source maps through `gwc dev`, `gwc build`, `gwc release`, `gwc test`, `gwc verify`, and `gwc doctor`, attributing resolved app, root, html, wasm/output, profile, host, and port fields to explicit flags, `gwc-start.json`, launcher config, or convention fallback so remaining heuristics stay visible instead of hidden in the final resolved values.
- [x] Add metadata schema-versioning and migration validation for generated apps.
	Generated `gwc-start.json` files now declare `schemaVersion: 1`, the launcher upgrades legacy version-`0` metadata in memory by defaulting the ownership block, and `loadScaffoldMetadata(...)` now rejects unknown future schema versions instead of silently reinterpreting newer starter metadata.
- [x] Add optional bootstrap for app-local git initialization.
	`gwc start` now accepts `-init-git`, threads that choice through scaffold generation, and runs `git init` in the generated app root after files are written so standalone starters can begin with their own repository boundary without making git setup an automatic side effect.
- [x] Add launcher integration tests for config precedence and project detection.
	`tools/gwc` now has focused tests for metadata-driven resolution, configured-app metadata discovery, convention fallback to `main.go` or `cmd/web/main.go`, and explicit flag overrides across `dev`, `build`, and `release`.
- [x] Add golden tests for scaffold output combinations.
	`tools/gwc/start_test.go` now snapshots the pure rendered starter outputs under `tools/gwc/testdata/start_golden/...` for standalone minimal-client, SSR, reference-app with browser tests, and contributor-linked starter combinations so scaffold diffs stay reviewable instead of hiding behind broad integration assertions.
- [x] Add end-to-end launcher tests for dev, build, and release commands.
	`tools/gwc/start_test.go` now drives a generated starter through real `go run ./tools/gwc dev`, `build`, `test`, `verify`, and `release` command paths, checking served HTML and wasm responses plus emitted build and release artifacts so launcher behavior is covered as an end-to-end product surface instead of only through internal helper tests.
- [x] Add machine-readable diagnostics contracts for launcher failures.
	Launcher failures requested through `-json` now emit a stable JSON diagnostic envelope with `command`, `phase`, `category`, `code`, and `message` fields, and json-mode runs suppress the enterprise extension preamble so build, dev, test, verify, and release failures stay machine-readable for editors, CI jobs, and wrapper tooling.

#### Runner Config Centralization And Path Policy

- [x] Define one canonical runner-config schema for Go launcher, JS test runner, and nested livereload tooling.
	`scripts/run-main-tests.mjs` now consumes the shared `scripts/runner-paths.mjs` contract instead of carrying its own runner-config parser, `scripts/runner-paths.test.mjs` validates the canonical example from the JS side, and `docs/RUNNER_CONFIG.md` names `gwc-runner.json` as the single path-schema contract shared by the Go launcher, JS runner, and nested livereload tooling.
- [x] Split launcher config code into schema, defaults, and resolver layers.
	`tools/runnerconfig` is now split into explicit `schema.go`, `load.go`, `defaults.go`, and `resolver.go` files so the shared runner-config package separates the external schema from config discovery and path resolution without changing the existing launcher-facing API.
- [x] Route every launcher-owned path decision through shared resolvers.
	Shared resolver helpers in `tools/gwc/runner_config.go` now own the default examples wasm directory plus the default build and release output paths, and `main.go` routes those launcher-owned fallbacks through the shared resolver layer instead of rebuilding them inline at each command site.
- [x] Remove remaining repo-owned `tmp/` and `dist/` assumptions from launcher-adjacent tooling.
	Generated import staging now resolves its temp root through the shared launcher path-policy helper instead of rebuilding `bin/tmp` inline, so launcher-adjacent generated import artifacts follow the same artifact-root policy path as the other normalized launcher outputs.
- [x] Make `artifactRoot` govern every launcher-owned output class consistently.
	Shared resolver helpers now cover default build outputs, release directories, generated import temp roots, and Playwright browser-result directories, and the JS-side runner-path helper exposes artifact-root-aware output resolution so launcher-owned generated artifacts all follow the same override policy instead of mixing ad hoc `bin/...` paths with artifact-root-aware ones.
- [x] Make browser-workspace, livereload-workspace, client-script, and wasm-exec resolution behavior identical across Go and Node entrypoints.
	`scripts/runner-paths.mjs` now exposes launcher-parity helpers for `goWasmExec`, `browserWorkspace`, `livereloadWorkspace`, and `livereloadClientScript`, and `scripts/run-main-tests.mjs` consumes those helpers instead of carrying custom fallback logic, so the Go launcher and Node runner no longer drift on override validation or default workspace resolution.
- [x] Add explicit resolution tracing for runner-config-derived values.
	Build and release resolution traces now record artifact-root-derived output paths as `gwc-runner.json paths.artifactRoot` instead of the vague `launcher config` label, so machine-readable summaries identify the specific runner-config field that drove those generated output locations.
- [x] Add stable, structured diagnostics for invalid runner overrides.
	The launcher’s json failure envelope now classifies bad runner-config paths as `invalid_runner_override` and carries the offending override field name, so missing configured paths and malformed runner overrides are distinguishable from generic configuration failures in editor and CI consumers.
- [x] Add config-discovery tests for local, environment, and home-directory runner config files.
	Lock down precedence between repo-local `gwc-runner.json`, `GWC_RUNNER_CONFIG`, and home-level defaults so enterprise setups remain predictable across shells and CI agents.
	`runner_config_test.go` has `TestLoadLauncherOverridesFindsParentConfig` (local parent-directory walk), `TestLoadLauncherOverridesPrefersExplicitEnvPath` (`GWC_RUNNER_CONFIG` env var), and `TestLoadLauncherOverridesForCurrentContextFallsBackToHomeConfig` (`~/.gwc/runner.json` home fallback) — all three discovery paths are locked down.
- [x] Add end-to-end parity tests for runner-config path overrides.
	`tools/gwc/runner_config_test.go` now drives one representative `gwc-runner.json` through real `build`, `release`, and browser `test` command paths, proving that the same override contract governs artifact-root outputs and browser-workspace selection across multiple launcher surfaces instead of only through helper-level tests.
- [x] Publish a documented example runner-config file and field reference.
	`docs/examples/gwc-runner.example.json` now provides the copyable baseline, and `tools/README.md` now documents the current path fields and relative-path resolution semantics.
- [x] Define organization-policy versus project-config ownership for runner settings.
	`docs/RUNNER_CONFIG.md` and `tools/README.md` now spell out that org-wide security and path policy belong in shared `GWC_RUNNER_CONFIG` or home-level config, repository-wide layout choices belong in checked-in `gwc-runner.json`, and short-lived local needs should stay on explicit command flags.

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
- [x] Add an explicit long-term policy for opt-in versus default fine-grained reactivity.
	`docs/FINE_GRAINED_REACTIVITY.md` and `docs/FRAMEWORK_SCOPE.md` now make the long-term policy explicit: fine-grained updates stay opt-in through selectors and subscribed regions, and no future preset, package, or subsystem should silently turn them into the default authoring model.
- [x] Add authoring guidance for choosing hooks versus selectors versus reactive regions.
	`docs/FINE_GRAINED_REACTIVITY.md` now includes a workload-driven authoring guide that defines when to stay on normal hooks, when `state.Select(...)` is enough to narrow a source model, and when `ui.ReactiveRegion(...)` is justified for a genuinely hot anchored region.
- [x] Add benchmark comparisons that defend the non-default stance explicitly.
	`internal/runtime/fine_grained_benchmark_test.go` now includes a signal-first-style proxy benchmark with per-panel subscribed regions, and `docs/FINE_GRAINED_REACTIVITY.md` now records side-by-side hook-only, opt-in fine-grained, and broader subscribed-region hotspot measurements that justify keeping fine-grained support opt-in rather than default.
- [x] Add first-class virtualization primitives for large lists and tables.
	The `virtualization` companion package now ships a first-class `List(...)` primitive plus `ComputeViewportState(...)`, `ObserveOwnedViewport(...)`, and `ViewportDiagnostics`, giving large fixed-height list surfaces one supported answer for windowing, stable row identity, overscan, and scroll restoration; semantic tables remain explicitly deferred to a later dedicated virtualization surface in `docs/VIRTUALIZATION.md`.
- [x] Decide whether virtualization belongs in core `ui`, a supported companion package, or an examples-first proving ground.
	`docs/VIRTUALIZATION.md` now records the ownership boundary explicitly: virtualization should start as a supported companion-package surface, proved with examples and benchmarks, rather than arriving as an unlabeled first-pass expansion of core `ui`.
- [x] Define the first supported virtualization workload shapes.
	`docs/VIRTUALIZATION.md` now makes the first-release scope explicit: start with uniform-height vertical lists for feeds, logs, queues, and result sets, while deferring variable-height rows, semantic tables, grids, and nested grouped virtualization to later follow-up work.
- [x] Define the virtualization API shape.
	`docs/VIRTUALIZATION.md` now records the first-release API direction explicitly: start with one virtualized-list component plus a row-render callback and explicit item, key, and row-height inputs, rather than a hook-only or low-level state bag API.
- [x] Define row identity and key requirements for virtualized items.
	`docs/VIRTUALIZATION.md` now makes stable caller-supplied item identity mandatory for virtualized rows, explicitly forbids visible-index-as-identity patterns, and ties row reuse to durable item keys so scroll-window reuse does not corrupt row-local state.
- [x] Define the viewport and scroll-container ownership model.
	`docs/VIRTUALIZATION.md` now makes the first-release ownership model explicit: the virtualized list should own its own scroll container and visible-range math, while external scrolling-parent support is deferred to a later explicit mode instead of being implied in the first contract.
- [x] Define overscan policy and defaults.
	`docs/VIRTUALIZATION.md` now makes the first overscan contract explicit: use a simple symmetrical item-count-based overscan with one caller override, and defer pixel-budget, velocity-aware, or direction-specific overscan heuristics until later evidence justifies them.
- [x] Define the measurement model for row size.
	`docs/VIRTUALIZATION.md` now fixes the first-release measurement contract to one explicit uniform row height per list and defers measured or estimated variable-height behavior to a later phase instead of overpromising correction logic on day one.
- [x] If variable-height rows are supported, define measurement invalidation rules.
	`docs/VIRTUALIZATION.md` now makes the boundary explicit: the first release does not support variable-height rows at all, so no remeasurement lifecycle is promised yet; any later variable-height mode must define invalidation triggers, scroll correction, diagnostics, and regression coverage before it is considered supported.
- [x] Define scroll restoration and anchor behavior.
	`docs/VIRTUALIZATION.md` now defines a simple hybrid first-release rule: preserve pixel offset for ordinary rerenders, record the first visible stable item key as an anchor, restore from the anchor when possible, and fall back to clamped pixel offset when the anchor item disappears.
- [x] Define SSR and hydration behavior for virtualized lists.
	`docs/VIRTUALIZATION.md` now defines a predictable first-pass SSR contract: render a fixed initial row budget on the server, hydrate that same window first, preserve full spacer geometry from `item_count * row_height`, and only switch to real viewport-driven window math after mount.
- [x] Define accessibility expectations for virtualized content.
	`docs/VIRTUALIZATION.md` now defines the first accessibility boundary explicitly: only rendered rows exist in the DOM and accessibility tree, focus must be restorable by stable item identity, callers own collection semantics and keyboard behavior, and container-level labels or counts should carry the full-collection context when needed.
- [x] Define table-specific behavior if tables are in scope for the first pass.
	`docs/VIRTUALIZATION.md` now makes the boundary explicit: semantic tables are out of scope for the first virtualization primitive, and any supported table virtualization path must arrive later as a separate primitive or clearly separate mode with explicit row, header, and cell semantics.
- [x] Define interaction behavior for sticky headers, sticky columns, and grouped sections.
	`docs/VIRTUALIZATION.md` now makes the first-pass boundary explicit: the initial virtualized-list primitive is one plain scrolling list and does not promise sticky headers, sticky columns, or grouped pinned sections until a later follow-up proves those behaviors with explicit diagnostics and examples.
- [x] Define integration guidance for local state, forms, and focus inside virtualized rows.
	`docs/VIRTUALIZATION.md` now makes the integration rule explicit: offscreen rows may unmount, so row-local hook state is disposable by default, while meaningful form drafts, expanded state, and focus restoration should be externalized by stable item key.
- [x] Define integration guidance for fine-grained reactivity versus virtualization.
	`docs/VIRTUALIZATION.md` now defines the ownership split directly: virtualization owns window-range and scroll-driven mount behavior, while `state.Select(...)` and `ui.ReactiveRegion(...)` are optional tools for hotspots inside already-rendered rows rather than substitutes for list windowing.
- [x] Add the first low-level viewport state primitive if needed.
	The new `virtualization/` companion package now provides `ComputeViewportState(...)` for fixed-height visible/rendered range math and `ObserveOwnedViewport(...)` for owned-element scroll and resize observation, so the upcoming list primitive can reuse one maintained viewport-tracking surface instead of reimplementing browser bookkeeping.
- [x] Add the first public virtualized list primitive for uniform-height rows.
	The `virtualization` companion package now includes `List(...)`, a narrow owned-scroll component for fixed-height vertical lists with explicit `ID`, `Height`, `RowHeight`, `ItemKey`, and `RenderRow` inputs, built directly on the shared viewport primitive instead of repeating window math in each consumer.
- [x] Add variable-height virtualization only after fixed-height behavior is correct and measured.
	`docs/VIRTUALIZATION.md` now includes an explicit variable-height admission gate: measured-row support stays deferred until the fixed-height primitive keeps passing viewport, restoration, and hydration regressions, remains benchmarked against a full-render baseline, and continues to report zero measurement, invalidation, and scroll-correction churn.
- [x] Add a virtualization-specific row renderer contract.
	The `virtualization` package now exposes a typed `RowRenderProps[T]` contract through `List(...)`, so the row callback receives the row index, item value, stable item key, visible range, and rendered range explicitly while internal spacer and offset mechanics stay package-owned.
- [x] Add virtualization-specific diagnostics for rendered range and overscan behavior.
	The `virtualization` package now exposes `ViewportDiagnostics` and `ViewportState.Diagnostics()`, and `List(...)` can publish that snapshot through `OnViewportChange`, making the visible range, rendered range, overscan counts, total item count, and viewport metrics inspectable without recomputing them in each consumer.
- [x] Add diagnostics for measurement churn and scroll correction.
	`ViewportDiagnostics` now exposes `MeasurementCount`, `InvalidationCount`, and `ScrollCorrectionCount`; the fixed-height first release reports all three as zero explicitly so current behavior is honest, inspectable, and ready for a later variable-height phase to populate with real churn data.
- [x] Add diagnostics for row mount or unmount churn in virtualized surfaces.
	`ViewportDiagnostics` now includes `RowMountCount` and `RowUnmountCount`, and `List(...)` updates those counters as virtualized rows enter and leave the rendered window so scroll-driven churn can be inspected without guessing.
- [x] Add virtualization-aware performance budgets.
	`docs/VIRTUALIZATION.md` now defines first-pass fixed-height budgets tied to the shipped diagnostics: rendered rows should stay within the visible-plus-overscan window, measurement and scroll-correction churn should remain zero, and row mount or unmount churn should roughly track rows entering and leaving the window rather than exploding into full-list rerenders.
- [x] Add a fixed-height feed example that proves the basic contract.
	`examples/103-virtualized-feed` now demonstrates the first virtualization contract with a long event feed, fixed-height rows, stable item keys, selection that survives row unmounts, and a live diagnostics panel driven by `ViewportDiagnostics`.
- [x] Add a data-table example if tables remain in scope.
	No first-pass table example is required because `docs/VIRTUALIZATION.md` now keeps semantic tables out of scope for the initial virtualization primitive; a table-focused example only becomes necessary if a later dedicated table virtualization surface is added.
- [x] Add examples that show row-local state pitfalls and recommended ownership patterns.
	`examples/103-virtualized-feed` now includes a side-by-side row-state panel that contrasts a row-local checkbox lost on virtualization unmount with the recommended parent-owned, item-keyed state pattern that survives scrolling.
- [x] Add browser benchmarks for large virtualized scroll surfaces.
	`examples/tests/virtualized-feed-benchmark.spec.ts` plus `playwright.virtualization.config.ts` now record browser scroll timings for the fixed-height virtualized feed and a full-render comparison surface built from the same dataset, and the results are written to `bin/playwright-virtualization-benchmark.json`.
- [x] Add regression tests for visible-range calculation.
	`virtualization/viewport_test.go` now includes a broader visible-range regression table covering empty lists, tiny lists, large-list start behavior, middle overscan boundaries, negative-scroll clamping, and resize-driven recalculation so fixed-height window math does not drift silently.
- [x] Add regression tests for stable row identity across scroll reuse.
	`examples/tests/virtualized-feed-regression.spec.ts` now scrolls the pitfall list in `examples/103-virtualized-feed`, verifies a row-local checkbox does not appear checked on unrelated items after scroll-window reuse, and confirms the original row resets when it remounts. `virtualization/list.go` now threads the row renderer through explicit row-shell props so sibling virtualized lists do not accidentally reuse the wrong renderer, and `playwright.virtualization.config.ts` now always boots a fresh local server so the regression runs against the current wasm build.
- [x] Add regression tests for scroll restoration and anchor retention.
	`examples/tests/virtualized-feed-restoration.spec.ts` now scrolls the restoration lane in `examples/103-virtualized-feed`, verifies prepending rows ahead of the window keeps the same anchored item visible, verifies container-height changes preserve the same anchor, and verifies a page reload restores the prior anchor from persisted session state. `virtualization/list.go` now persists per-list restoration snapshots by stable item key and falls back to pixel offset when the anchor disappears.
- [x] Add regression tests for hydration behavior on virtualized surfaces.
	`examples/tests/virtualized-feed-hydration.spec.ts` now verifies the virtualized hydration lane against the documented SSR-to-browser contract by checking the prerendered initial rendered range with JavaScript disabled, then hydrating the same page and confirming the rendered window, diagnostics, and post-hydration scrolling stay correct under `examples/playwright.virtualization.config.ts`.
- [x] Add regression tests for keyboard and focus behavior in virtualized lists.
	`virtualization.ListProps` now accepts `OuterProps html.Props` on the owned scroll container so callers can attach `role`, `aria-*`, `data-*`, raw `tabindex`, and keyboard handlers directly to the core viewport element, and `virtualization/list_test.go` now locks down that contract with package-level markup and ID-resolution coverage instead of relying on example-only wiring.
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
- [x] Add a bounded template- or JSX-authored experiment that lowers into inspectable Go.
	`examples/13-browser-compiler/template_lowering` now ships one constrained HTML-like template experiment with a tiny lowering command, a checked-in generated Go file, and a verifier test so template-first authoring can be evaluated without changing the default Go-first runtime model or toolchain contract; `docs/COMPILER_ASSISTED_FEATURES.md` now names that experiment explicitly.
- [x] Compare template-lowered authoring against plain Go on debugging and mixed-mode adoption.
	`docs/COMPILER_ASSISTED_FEATURES.md` now compares the bounded lowering experiment against handwritten Go on code review readability, stack-trace clarity, generated diff noise, mixed-mode composition, and migration cost, and records the current conclusion: safe to evaluate as an optional companion-tool workflow, but not a better default than plain Go.
- [x] Publish a stable policy for template-first and compiler-first authoring.
	`docs/COMPILER_ASSISTED_FEATURES.md` now publishes the stable policy explicitly: plain Go and ordinary `go build` remain the supported default, template-first authoring stays outside the default product story, and compiler-assisted workflows may continue only as opt-in example-level or companion-tool experiments until they satisfy the documented graduation rule.

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
- [x] Make state-preserving hot reload the default supported path inside `gwc dev` for compatible apps.
	`gwc dev` is now the documented canonical launcher path for hot reload, with `-hot=true` enabled by default and clear guidance that compatible rebuilds preserve state while incompatible edits fall back to remount or full reload behavior automatically.
- [x] Add pre-reload compatibility reporting for preserve versus remount decisions.
	The live-reload classifier now publishes an explicit pre-reload plan, and the browser client surfaces that plan before bundle swap so developers can see whether the incoming edit will preserve compatible local state, remount changed subtrees, restart async or router work, or force a full reload.
- [x] Add broader hot-reload coverage for routed, cached, and SSR-seeded apps.
	`examples/92-protected-routes` and `examples/93-ssr-cache-bootstrap` now opt into the public `hotreload` package, `docs/HOT_RELOAD.md` names the routed, cache-backed, and SSR-seeded reference surfaces explicitly, and `examples/MANUAL_TESTING.md` includes dedicated hot-reload passes for the routed shell, guarded loader/cache flow, and SSR-seeded cache bootstrap flow.

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
- [ ] Document the single-shell runtime pattern for full-product GWC applications.
	Define how one GWC WASM client owns the complete product surface — public marketing pages, sign-in and sign-up flows, and the private authenticated workspace — all rendered client-side without server-HTML redirect pages between auth states. Specify the recommended router tree shape, layout inheritance, code-splitting strategy, and SEO/prerender considerations when adopting this pattern.
- [ ] Define routing conventions for mixed marketing and workspace routes in a single GWC router.
	Document how public routes (Home, Features, Pricing, Contact) coexist with protected workspace routes under one router without separate HTML shells, duplicated server endpoints, or split layout trees; define which layout layers are shared across public and private route families and where the auth boundary is expressed in the route tree.
- [ ] Add a canvas workspace routing pattern.
	Define how routes that own a canvas editing surface (rich text editor, diagramming canvas, media editor) share the same shell layout and router as threaded data flows; specify what canvas state persists across route changes, what resets on leave, how undo/redo history interacts with route params so the browser back button behaves intuitively, and how canvas autosave integrates with the offline mutation queue.
- [ ] Define the progressive route expansion pattern for growing client shells.
	Document how to grow a client shell over time — minimal authenticated workspace first, marketing routes next, canvas views and admin panels later — without the base workspace bundle growing proportionally; use code-splitting to keep shell-entry cost stable as the route tree expands.

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
- [x] Define an AI provider companion package pattern for LLM-backed GWC applications.
	`docs/ECOSYSTEM.md` now defines the AI-provider companion layer directly above the protobuf RPC transport: RelayDesk (`examples/100-ai-chat-wizard`) is the reference implementation, the recommended scope covers provider-capability metadata retrieval, SQL-backed model catalog queries, capability-aware selector components, streaming-aware chat and speech wrappers, and explicit preference-persistence adapters, and the boundary explicitly leaves provider SDKs, prompt policy, and server-side inference orchestration in application code. The package starts as `Experimental`, and the compatibility matrix now tracks `ai/provider` with RelayDesk as the first validating application.
- [x] Define the SQL-backed model catalog pattern for runtime provider and model discovery.
	`examples/100-ai-chat-wizard/README.md` now documents RelayDesk's SQL-backed catalog path end to end: `sql/store/schema.sql` owns provider/model metadata in `model_catalog`, `server/app/model_catalog_store.go` materializes runtime provider catalogs and default-model picks from SQL rows, `server/app/server.go` exposes the live query contract through `ListModelOptions`, and the recommended bootstrap shape mirrors that RPC in `ui.SSRBootstrap.Data`. The example also defines the cache policy explicitly: bootstrap or local-storage catalog data is only a last-known-good shell hint, and the authenticated client revalidates against the server-owned catalog when the gRPC session becomes ready.
- [x] Define capability-aware model selection components for AI provider switching.
	RelayDesk now derives capability-aware picker state directly from the runtime model catalog: when reasoning mode is active, the current provider's model dropdown filters down to reasoning-capable entries only, the toolbar explains why models disappeared, and the browser regression locks the filtered Cerebras list to the thinking-capable catalog entries instead of leaving users to guess which models still satisfy the active workflow.
- [x] Define multi-provider active model persistence, propagation, and tab synchronization.
	`examples/100-ai-chat-wizard` now treats the selected model as the authoritative per-user preference: the client refreshes the runtime model catalog as soon as the authenticated gRPC shell comes up, persists model changes through `SetSelectedModel`, and mirrors those changes across open tabs through an `interop.OpenCrossTabChannel(...)` RelayDesk channel so provider and model selectors stay aligned without a reload.
- [x] Promote `examples/100-ai-chat-wizard` (RelayDesk) as the reference implementation for runtime AI provider switching.
	`examples/100-ai-chat-wizard/README.md` now calls RelayDesk out as the repo's reference implementation for runtime provider switching, links the shipped behaviors that matter in practice (startup catalog load, live provider/model switching, per-user persistence, cross-tab sync, and local stub providers), and points maintainers to the focused Playwright regression that validates the reference flow.

### Launcher extensibility and enterprise policy

- [x] Define launcher config layering for framework defaults, organization policy, project config, and CLI overrides.
	`tools/gwc/enterprise_config.go` now resolves enterprise launcher settings through an explicit layering model (`framework defaults < organization policy pack < project config < CLI disables`), with source attribution and merge semantics covered by `tools/gwc/enterprise_config_test.go`.
- [x] Add simple pre and post command hooks to the launcher.
	`gwc` now loads enterprise hook definitions from layered config and executes auditable `pre-*` and `post-*` hook points (including `pre-all` and `post-all`) around command dispatch with explicit failure attribution and `allowFailure` handling, covered by `tools/gwc/enterprise_runtime_test.go`.
- [x] Define a capability-based launcher plugin contract.
	The launcher now defines a structured plugin invocation contract (`tools/gwc/enterprise_plugins.go`) with bounded contribution surfaces (scaffold features, test lanes, verify checks, release validators, deployment packagers), and enterprise config normalization now enforces a strict capability allowlist so plugins cannot claim unsupported rewrite-style powers.
- [x] Prefer external executable plugins with structured JSON I/O.
	Launcher plugin execution now uses an explicit external executable JSON protocol (`gwc-plugin-v1`) with typed request and response contracts, capability-gated invocation, and failure attribution, including wired verify and release plugin checks through structured JSON responses.
- [x] Add plugin-discovery and trust reporting to command output.
	Each launcher invocation now emits an extension report to stderr with discovered organization policy pack and project config sources, loaded hook points, and loaded plugins including trust markers and capability lists, so enterprise teams can audit extension-driven behavior per command.
- [x] Add policy-pack support for organization-specific enterprise requirements.
	Policy packs now enforce approved Go toolchains, required test-lane coverage, release naming patterns, mandatory budgets and compression policy in launcher runtime paths (`run test`, `run verify`, `run release`), with validation tests and runnable policy examples in `docs/examples/gwc-policy-pack.example.json`.
- [x] Add launcher security boundaries for hooks and plugins.
	Enterprise launcher config now supports explicit extension boundaries (`enterprise.security`) for trusted/untrusted execution, allowed executable roots, inherited environment allow or deny lists, and secret env references (`${ENV:KEY}`), with enforcement and focused runtime tests in `tools/gwc/enterprise_runtime_test.go`.
- [x] Add plugin-contributed feature sections to the scaffold TUI.
	`gwc start` now loads `scaffold_feature` plugin contributions into a dedicated optional enterprise step, keeps built-in preset features separate in the TUI, and carries selected enterprise capabilities into generated scaffold capability outputs and metadata.
- [x] Add plugin and policy integration tests.
	Integration-focused launcher tests now cover org policy and project config discovery, hook ordering across layers, plugin precedence overrides, failure attribution, and diagnostic-code propagation for plugin-enforced policy failures in `tools/gwc/enterprise_runtime_test.go`.

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

- [x] Define the next-level profiling surface beyond summary counters.
	`internal/runtime` profiling snapshots now include structured phase totals plus a rolling typed event timeline, and `devtools` maps and renders those surfaces so profiling moves beyond flat counter-only snapshots.
- [x] Add per-component render tracing.
	`internal/runtime.renderFunctionComponent(...)` now records per-component render traces (render count, rerender count, trigger attribution, and timing totals), and `devtools` surfaces the hottest rerendering components directly in the profiling panel.
- [x] Add timing attribution for render, diff, commit, and effect phases.
	Fiber and branch profiling now attribute `render`, `diff`, `commit`, `effect`, and `cleanup` time separately, and devtools now surfaces those buckets so expensive component execution is distinguishable from expensive DOM or effect work.
- [x] Add flamegraph-style capture and visualization support.
	Profiling snapshots now include nested `FlamegraphFrame` captures (depth, start offset, duration, and phase breakdown), and devtools now renders a flamegraph-style timeline view instead of only flat counters and hot-branch lists.
- [x] Add async and route-lifecycle profiling.
	Router lifecycle and loader phases now emit structured profiling timeline events (`navigation`, `route.lifecycle`, `loader`, and guard outcomes), and transition scheduling now records async wait and run timing so route-plus-async interactions can be profiled end to end.
- [x] Add startup, hydration, and first-interaction profiling workflows.
	Startup profiling now records bootstrap read duration, hydration duration, first commit timing, and time-to-first-interaction in runtime snapshots and devtools, with timeline events that connect startup phases to real user interaction readiness.
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

- [x] Add richer component-stack and failure context for runtime errors.
	`internal/runtime.Diagnostic` now preserves structured `Fields` alongside the existing path, component stack, top frame, and consequence metadata, so runtime error diagnostics keep route and phase context such as `path`, `component_stack`, `top_frame`, `runtime`, `phase`, and `where` instead of dropping that detail into logs only; the runtime tests and js/wasm devtools snapshot compile now cover the richer shape.
- [x] Add a â€œwhy did this rerender?â€ inspection surface.
	The existing component-render profiling surface now carries more useful rerender causes instead of collapsing everything into generic hook churn: scheduler-owned updates can record `local-state`, `atom`, `async-resource`, `context`, `hydration`, `error-boundary`, `fine-grained`, `props`, and `parent`, and the inspected `ComponentRenderTrace` plus devtools rerender summary continue to surface the latest trigger and accumulated trigger counts for each component path.
- [x] Add hook-slot and state-transition inspection.
	The inspected hook payload now carries stable per-kind slot numbers plus dependency and lifecycle metadata instead of only flattened strings: memo, callback, and effect hooks expose dependency snapshots, effect hooks expose cleanup and epoch status, fetch hooks expose a status summary, and the richer `HookSnapshot` shape is mirrored through the devtools node mapping so the existing inspector can show more than bare hook labels and values.
- [x] Add route and loader debugging panels.
	`router.InspectCurrentRoute()` now exposes a deeper route snapshot instead of only path and params: the inspection includes the resolved route stack, per-route guard and loader presence, active loader entry state, the last redirect cause, and leaf metadata ownership, and the devtools route summary maps that richer router-owned state directly instead of reconstructing it from logs.
- [ ] Add offline replay and sync debugging panels.
	Expose queue entries, replay ownership, reconnect status, conflict state, per-entity sync health, and last replay error so offline-capable apps can debug the hardest field failures without custom logging.
- [x] Add hydration debugging tools.
	The runtime inspection snapshot now preserves the last hydration pass with correlation id, timing, existing DOM counts, fallback and mismatch counters, discarded-node counts, strict/failure state, and recent hydration diagnostics, and the devtools snapshot, panel, and snapshot diffing now expose that hydration section directly instead of leaving hydration failures as raw console-only warnings.
- [x] Add bootstrap and serialization-boundary inspectors.
	`devtools` now exposes a first-class serialization-boundary inspection model with snapshot export and diff support, a panel section for boundary entries, a bootstrap-to-boundary helper built on `ui.AnalyzeSSRBootstrapSize(...)` and `ui.InspectBootstrapPayloads(...)`, and app-owned registration hooks for worker, RPC, and multi-client transport entries so boundary sizes, encodings, downgraded legacy payloads, and rejected transports become inspectable instead of living only in scattered logs.
- [x] Add cache, worker, and synchronization inspectors.
	The devtools snapshot and panel now add a first-class coordination inspection section alongside the existing shared-cache view, and `devtools.SetCoordinationInspection(...)` lets apps surface worker job state, cross-tab or multi-window synchronization events, and offline replay queue entries in one exported and diffable payload instead of relying on ad hoc logging for coordination bugs.
- [x] Add a browser error overlay for development failures.
	`devtools.ErrorOverlay(...)` now renders a focused in-browser failure surface for actionable framework diagnostics, including runtime errors, hydration failures, loader crashes, and startup faults, and it reuses the existing stable diagnostic codes, top-frame attribution, path context, and docs anchors instead of forcing developers back to raw console output.
- [x] Add recovery actions to the development error overlay.
	The development overlay now supports app-owned recovery actions through `devtools.SetErrorOverlayActions(...)`, matching buttons against stable diagnostic codes or sources and invoking typed handlers with the current snapshot and issue context so flows such as retry loader, clear bootstrap state, or reveal launcher diagnostics can be attached directly to the structured failure surface.
- [x] Add source-mapped stack translation for wasm runtime failures.
	`internal/runtime` now exposes a wasm stack-frame mapper hook that can rewrite parsed panic frames before top-frame selection, formatted panic output, diagnostics, logs, and browser panic payload emission, so build tooling can translate browser-observed wasm frames back to application-owned Go files and symbols instead of stopping at low-level offsets.
- [x] Correlate source-mapped runtime failures with launcher artifact metadata.
	`internal/runtime` now accepts launcher-owned wasm artifact metadata alongside the source-mapped frame hook, and wrapped panic reports, diagnostics, logs, and browser panic payloads now carry that build id, emitted artifact path, manifest path, hash, symbol-set hint, and version data so a browser-observed failure can be traced back to the exact shipped artifact record.
- [x] Add strict development-mode toggles.
	`internal/runtime.ConfigureStrictDiagnostics(...)` now lets tests and local development escalate selected recoverable diagnostics by stable code, source, or classification into hard failures, so hydration fallback and other warning-only recovered behavior can be turned into deterministic test failures instead of lingering unnoticed.
- [x] Add trace capture and replay support for debugging sessions.
	`devtools` now supports labeled trace capture export/import plus replay-state injection through `CaptureTrace(...)`, `ExportTraceCaptureJSON(...)`, `ImportTraceCaptureJSON(...)`, and `SetTraceReplay(...)`, so one debugging snapshot can be saved, diffed with the existing snapshot comparison support, and replayed back through the panel without manually reconstructing the original interaction context.
- [x] Add reproducible local bug-capture bundles.
	`devtools` now packages one local bug-capture bundle around the current trace snapshot through `CaptureBugBundle(...)`, `ExportBugCaptureBundleJSON(...)`, `ImportBugCaptureBundleJSON(...)`, and `ReplayBugCaptureBundle(...)`, so the current route, component tree, cache state, diagnostics, logs, hydration summary, coordination state, and boundary metadata can be saved into one local JSON artifact and reopened through the existing replay path.
- [x] Add a first-class component-tree inspector with route and cache context.
	The devtools panel now keeps a selected component-tree node and renders a dedicated inspector pane with stable node path, hook or state summaries, active route context, reactive metadata, async-resource hook ownership, and shared-cache owner matches built from tracked cache subscriber paths, so the tree view is no longer only a read-only overview dump.
- [x] Add devtools extension points for companion packages and app-owned inspectors.
	`devtools.SetExtensionSections(...)` now lets companion packages and applications register exportable custom summary sections with structured key-value metadata and freeform lines, and the panel renders those sections alongside framework-owned inspectors so the shared devtools surface can grow without bespoke framework changes for every companion-owned debug view.
- [x] Add support-safe diagnostic bundle export.
	`devtools.CaptureSupportDiagnosticBundle(...)`, `devtools.SanitizeBugCaptureBundleForSupport(...)`, `devtools.ExportSupportDiagnosticBundleJSON(...)`, and `devtools.ImportSupportDiagnosticBundleJSON(...)` now produce a separate redacted support artifact from the existing local replay bundle, scrubbing sensitive query params, auth or session fields, inline bearer or cookie values, hook state values, hydration messages, cache keys, and extension metadata while keeping the snapshot structure usable for support triage.

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
- [x] Add consumer-copyable examples under the preferred `test/...` import paths.
	Consumer-style example tests now live directly under `test/render`, `test/hooks`, `test/router`, and `test/ssr`, so the documented preferred import paths each have a real copyable example instead of sending readers immediately back through the `testkit/...` compatibility layer.
- [x] Add parity coverage for `test/...` wrappers and `testkit/...` compatibility aliases.
	Parity tests now exercise `test/render`, `test/hooks`, `test/router`, and `test/ssr` directly against the matching `testkit/...` compatibility aliases so wrapper drift, missing exports, or alias-only behavior changes have package-level regression coverage instead of being implicit.
- [x] Add a first-class `test/browser` helper package.
	`test/browser` now ships a small ESM helper surface with `waitForAppReady(...)`, `captureConsole(...)`, and `readDiagnostics(...)`, covering Playwright-style app boot waits, bounded console or page-error capture, and explicit app-owned diagnostics reads without pretending to replace the browser runner itself.
- [x] Add explicit browser-environment simulation helpers for public tests.
	`test/browser` now also exposes a public js/wasm `Install(...)` environment helper that installs controlled `window`, `location`, `history`, storage, `matchMedia`, `BroadcastChannel`, `Worker`, and popup or opener shims, and `testkit/router` now consumes that helper instead of keeping a private ad hoc browser-global fixture.
- [x] Add async resource and loader test utilities.
	`test/render.NewResourceController(...)` now provides a deterministic async resource loader control with attempt tracking plus resolve, reject, cancel, retry, and stall behavior, and `test/router.NewLoaderController()` layers the same pattern over `router.LoaderFunc` while recording route path, params, and query state for each attempt.
- [x] Add round-trip hydration test helpers that start from real server markup.
	`test/ssr.RoundTripHydrate(...)` now renders or accepts real server markup, seeds it into the mock DOM before hydration, and preserves seeded node ids through `render.SeededMarkup` plus `QueryNode.NodeID()` so public hydration tests can assert node reuse against actual SSR-shaped DOM instead of only hydrating an empty container.
- [ ] Add structured SSR and head assertions for snapshot tests.
	Current `test/ssr` helpers and companion head tests still rely heavily on raw substring checks for titles, descriptions, canonical tags, social metadata, JSON-LD, and bootstrap scripts; add typed assertion helpers so metadata-heavy SSR tests can verify meaning without brittle string-order coupling.
- [x] Add prerender and static-export test helpers.
	`test/ssr.LoadStaticExport(...)` now reads one prerendered output directory into structured HTML snapshots and bootstrap sidecars, and `StaticExport.Route(...)` resolves exported route paths back to emitted files so prerender tests can assert HTML, metadata, and bootstrap artifacts without package-local disk-walk helpers.
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
