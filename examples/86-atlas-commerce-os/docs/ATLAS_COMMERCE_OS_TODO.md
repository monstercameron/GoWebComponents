# Atlas Commerce OS Todo

This document tracks the planned rewrite of the Atlas Commerce OS experience using the React design references in:

- `design/homepage_store.tsx`
- `design/homepage_warehouse.tsx`

The goal is not to port those mocks literally as a separate product. The goal is to translate their visual language, layout density, motion, and interaction patterns into the real Atlas Commerce OS route model and shared GoWebComponents SSR plus hydration architecture.

## Current Rewrite Goal

- make the GoWebComponents Atlas HTML structure match the React compositions as closely as possible where the concepts overlap
- make the GoWebComponents Atlas CSS match the React visual result 1:1 at the route and component level
- replicate the React interaction model in Go and WASM, then enhance it where Atlas already has stronger domain behavior
- keep Atlas route semantics, Atlas data contracts, Atlas forms, and Atlas SSR intact unless a route-level redesign requires coordinated server changes

## Route Mapping Rules

- `homepage_store.tsx` is the visual reference for the Atlas public surface, not for a new cart-first ecommerce app
- `homepage_warehouse.tsx` is the visual and interaction reference for the Atlas internal surface, not for a generic `/warehouse/*` product admin
- Atlas public routes remain:
  - `/`
  - `/shop`
  - `/shop/:slug`
  - `/warehouses`
  - `/warehouses/:slug`
  - `/warehouses/:slug/availability/:productSlug`
- Atlas internal routes remain:
  - `/app/dashboard`
  - `/app/products`
  - `/app/products/:slug`
  - `/app/inventory`
  - `/app/inventory/:sku`
  - `/app/warehouses`
  - `/app/warehouses/:warehouseId`
  - `/app/warehouses/:warehouseId/items/:sku`
  - `/app/transfers`
  - `/app/transfers/:id`
  - `/app/purchase-orders`
  - `/app/purchase-orders/:id`
  - `/app/receiving`
  - `/app/receiving/:id`
  - `/app/comments`
  - `/app/settings`
- where the React mocks introduce features that Atlas does not have, translate the pattern rather than the literal feature
  - cart drawer -> Atlas quote, restock, or action rail pattern
  - checkout -> Atlas request, follow-up, or workflow completion pattern
  - generic warehouse items CRUD -> Atlas product CMS plus inventory lane editing plus purchase-order flow

## Execution Order

### 1. Route, Data, And Server Contract Alignment

- [ ] Keep Atlas seed data and route semantics coherent with the new visuals
- [ ] Rename or revise any route copy that still reflects the older scaffold language instead of the new shell
- [ ] Ensure public merchandising copy still fits Atlas workspace systems rather than drifting toward generic gadgets
- [ ] Ensure internal route titles and descriptions still reflect real Atlas workflows after the rewrite
- [ ] Revisit route metadata in `shared/atlas/legacy_shared.go` so titles, descriptions, and canonicals stay aligned with the rewrite
- [ ] Revisit server response shapes for routes that currently overfetch or recompute similar summaries in multiple handlers
- [ ] Move shared dashboard, inventory, warehouse, and PO summary logic toward reusable server or shared-layer derivation where that reduces duplicated work
- [ ] Add one production-shaped bulk workflow, likely bulk moderation or bulk inventory threshold updates, so Atlas shows how GWC handles dense multi-record actions without becoming a toy demo
- [ ] Evaluate whether lightweight import or export flows for saved views, inventory policies, or diagnostics snapshots would improve Atlas as a real showcase app

### 2. SSR, Routing, And Bootstrap Invariants

- [ ] Preserve `ui.RenderToString` plus `ui.Hydrate` parity across every rewritten route family; do not let new interactions drift into client-only assumptions
- [ ] Keep using the history router path already in place and expand route-loader usage where rewritten routes need explicit loader and revalidation behavior
- [ ] Add more route metadata ownership through the router where it improves title, description, and canonical consistency after client-side transitions
- [ ] Audit loader granularity so route transitions only fetch what changed instead of rebuilding large shared payloads unnecessarily
- [ ] Add revalidation triggers for mutations that should refresh parent and sibling route data after save, approve, reconcile, or create actions
- [ ] Evaluate route `before-enter` and `before-leave` guards for unsaved editor flows, destructive confirmations, and auth-sensitive internal paths
- [ ] Add at least one production-grade unsaved-changes guard for a real internal editor route so Atlas demonstrates `before-leave` with a meaningful workflow
- [ ] Expand nested layout route usage if the rewritten internal shell introduces stable sub-layouts for products, inventory, or warehouses
- [ ] Evaluate whether one secondary workflow should be deep-linkable by URL, such as a SKU threshold overlay or PO detail side sheet, to show route-plus-overlay composition cleanly
- [ ] Decide which public route data should be reused from SSR bootstrap, which should revalidate on navigation, and which should be cached client-side between route transitions
- [ ] Define which internal route datasets should persist in route-local cache or shared resources so operators do not pay a full reload cost after every minor workflow action
- [ ] Audit which route data must be serialized into bootstrap and which can be fetched lazily after hydration to keep first paint fast without breaking parity
- [ ] Revisit bootstrap payload usage after the shell rewrites so route-specific UI still receives enough structured data
- [ ] Track bootstrap payload size as the rewrite progresses and split or externalize route data if first-load SSR payloads become too large
- [ ] Evaluate `ui.RenderBootstrapReferenceScript` plus external bootstrap payloads if Atlas bootstrap size grows significantly during the rewrite
- [ ] Add one proof-of-concept route or diagnostics mode that can switch to external bootstrap reference loading if payload growth makes the inline script less convincing as the long-term demo story
- [ ] Add explicit todos for any rewritten component that cannot be made SSR-safe on the first pass, and isolate it behind a clear hydration boundary instead of silently degrading the route

### 3. Server-State, Derived-State, And Cache Baseline

- [ ] Audit handler-level duplicate queries for dashboard, warehouse detail, SKU detail, and PO detail flows
- [ ] Add todos to consolidate repeated database reads where one route currently builds several related summaries separately
- [ ] Revisit whether some route summaries should be precomputed or persisted if they become expensive enough in the real app shape
- [ ] Build a shared inventory-derived-state layer so counts, badges, urgency summaries, reorder totals, and lane summaries are not recomputed independently across multiple routes
- [ ] Build a shared public-derived-state layer for product status messaging, warehouse promise summaries, and action-label decisions
- [ ] Decide which derived values belong:
  - in server responses
  - in shared atlas helpers
  - in state computed or derived primitives
  - in route-local memoized transforms
- [ ] Define the canonical derived-state layer for Atlas so expensive counts, totals, urgency bands, and filter summaries are computed once and consumed across routes
- [ ] Remove duplicated count, grouping, sorting, and summary logic once the canonical derived-state locations are defined
- [ ] Define SSR bootstrap reuse rules for each major route family
- [ ] Define client cache lifetime rules for:
  - related products
  - public comments
  - saved views
  - preferences
  - purchase-order detail
  - receiving detail
  - warehouse detail side data
- [ ] Define mutation invalidation rules for every write flow so caches stay correct after updates
- [ ] Decide where stale-while-revalidate behavior is acceptable and where Atlas must block on fresh data
- [ ] Ensure route reloads and direct-entry SSR do not regress because of assumptions made by client cache layers
- [ ] Decide whether any preferences, saved views, or diagnostics state should sync across tabs or windows so Atlas can show a real cross-tab consistency story where it adds value
- [ ] Audit bootstrap and JSON payload sizes after the rewrite and trim repeated or unnecessary data

### 4. GoWebComponents Runtime Foundation

- [ ] Audit the current Atlas implementation against actual framework surfaces and refresh `docs/README.md#framework_coverage` so the rewrite starts from shipped reality
- [ ] Prefer a real GWC feature integration when a rewrite task can reasonably use one, instead of rebuilding the behavior manually with ad hoc state

#### State and shared ownership

- [ ] Introduce `state.UseComputed` for derived stock-health totals, action counts, route badges, and shell-level summaries that should not be recomputed ad hoc in multiple components
- [ ] Expand `state.UseAtom` or `state.UseDerived` where shell-wide filters, presentation preferences, or route-scoped workspace state need consistent shared ownership
- [ ] Audit each route for duplicated derived calculations and replace them with shared computed or derived primitives before the rewrite stabilizes
- [ ] Add a real shell-state ownership layer for user, locale, theme, workspace, and diagnostics state in an SSR-safe form instead of threading everything through props
- [ ] Evaluate snapshot export or restore flows for reviewer, diagnostics, or operator workspaces if the new shell introduces richer multi-step context
- [ ] Add one exportable reviewer or operator snapshot flow if diagnostics or workspace state becomes rich enough to justify snapshot sharing across sessions

#### Async data, tasks, and resources

- [ ] Introduce `fetch.Fetch` or `ui.UseFetch` for imperative refresh surfaces where Atlas should requery without a full route reload
- [ ] Expand `ui.UseResource` and `ui.UseCachedResource` beyond diagnostics if related products, public comments, PO details, or internal side panels become lazily refreshed
- [ ] Prefer cached-resource patterns for repeat-open secondary panels, drawers, and detail views where the data shape is stable enough to avoid needless refetching
- [ ] Add `ui.AsyncBoundary` around any new route-local async enhancement islands so loading and error states stay scoped and visually coherent
- [ ] Evaluate `ui.UseTask` or `ui.UseWorkerTask` for background work that should not block the main interaction path, such as bulk formatting, diagnostics snapshots, or heavy local transforms
- [ ] Add `ui.UseChannel` if the rewritten public and internal surfaces need cross-panel event fanout, such as notifying shell-level toasts or synchronizing workflow completion state across distant components
- [ ] Add one worker-backed production-shaped flow, likely bulk inventory CSV parsing, import validation, or large client-side diagnostics transforms, so Atlas demonstrates `ui.UseWorkerTask` on a real workload
- [ ] Add one channel-backed cross-panel event flow, likely shell toast dispatch or workflow-complete fanout, so Atlas demonstrates `ui.UseChannel` with a concrete operator benefit

#### Forms and workflow state

- [ ] Expand `ui.UseForm` to every non-trivial internal editor flow so product, inventory, PO, receiving, moderation, and settings forms share one typed validation model
- [ ] Use `ui.UsePrevious` for route-level change summaries, optimistic-to-confirmed state reconciliation, and "recently changed" UI messaging in internal workflows
- [ ] Add `ui.UseReducer` where workflows become multi-stage and event-heavy, especially for purchase-order creation, receiving reconciliation, and moderation review
- [ ] Add `ui.UseId` consistently to every generated field group, modal title, and described-by relationship introduced during the rewrite

#### SSR-safe composition, overlays, and focus

- [ ] Introduce `ui.Fragment` where composite route sections and grouped table cells currently require extra wrapper nodes that hurt HTML parity
- [ ] Evaluate `ui.UseLazyNode` for below-the-fold public sections and secondary internal panels that should mount only after the primary route body is stable
- [ ] Add `ui.ErrorBoundary` around route-local enhancement islands so one broken interactive panel does not collapse the full page
- [ ] Introduce `ui.UseRef` for opener capture, focus restoration, and route-local focus targets after modal, drawer, and side-sheet dismissal
- [ ] Introduce `ui.UseFocusManager` for keyboard-first movement inside dense internal action clusters, tables, and modal toolbars where plain tab order becomes inefficient
- [ ] Introduce `ui.UseFocusTrap` anywhere Atlas needs modal-only focus containment beyond the current overlay defaults
- [ ] Introduce `ui.UseCompositeNavigation` where internal listbox-, menu-, tab-, or command-palette-like interactions emerge during the rewrite
- [ ] Introduce `ui.UseAnnouncer` for route-change, mutation-success, and overlay-state announcements that should be explicit for assistive technology
- [ ] Expand `ui.AccessibleOverlay`, `ui.Overlay`, and `ui.UseOverlayStack` usage beyond the current flows if the rewritten public and internal shells add drawers, sheets, or nested confirmations
- [ ] Add a shared portal-backed drawer pattern using `ui.Portal` and `ui.PortalTarget` if the public action rail or internal quick actions need off-canvas behavior

#### Scheduling and responsiveness

- [ ] Use `ui.UseTransition` and `ui.StartTransition` for non-urgent filter, sort, tab, and workspace updates where immediate keystroke responsiveness matters
- [ ] Add `ui.UseDeferredValue` to high-churn search and filter surfaces after the new visual shell lands, especially on catalog, products, and inventory
- [ ] Add `ui.UseDebounced` where text search should delay network or heavy local recomputation
- [ ] Add `ui.UseThrottled` for resize-aware shell state, sticky panel calculations, diagnostics sampling, or scroll-linked UI state if the new shell needs them

#### Diagnostics and devtools

- [ ] Expand `devtools.Panel` usage so rewritten routes expose their new state machines, async resources, and overlay stacks during development review
- [ ] Add diagnostics hooks for any new reusable shell primitives, especially drawers, route guards, async panels, and shared workflow forms
- [ ] Add developer-visible diagnostics for cache hits, cache invalidation, revalidation timing, and expensive derived-state recomputation during the rewrite
- [ ] Add a guided diagnostics view or checklist that deliberately surfaces the major GWC concepts used by Atlas so the example is easier to demo and explain

### 5. Performance, Observability, And Runtime Budgets

- [ ] Identify the highest-risk dense routes for hydration and rerender cost:
  - `/shop`
  - `/shop/:slug`
  - `/app/dashboard`
  - `/app/products`
  - `/app/inventory`
  - `/app/warehouses/:warehouseId`
- [ ] Add route-level performance checkpoints for first paint, first interactive action, and post-hydration responsiveness
- [ ] Ensure non-urgent UI updates use transition or deferred patterns where they can avoid keystroke lag or jank
- [ ] Keep expensive secondary panels, derived lists, and related-content regions from rerendering on unrelated state changes
- [ ] Define practical performance budgets for:
  - bootstrap payload size
  - route loader latency
  - overlay open latency
  - search interaction responsiveness
  - dense-table rerender responsiveness
- [ ] Add manual and automated checks that flag obvious regressions in those budgets during the rewrite
- [ ] Surface perf-relevant debug information in diagnostics mode so future work can see where caching or derived-state decisions need adjustment

### 6. HTML Parity Foundation

- [ ] Audit the DOM structure produced by the main public routes versus the React mock structure
- [ ] Audit the DOM structure produced by the main internal routes versus the React mock structure
- [ ] Reduce wrapper mismatches where extra GWC nodes make CSS parity harder than necessary
- [ ] Standardize section scaffolds so repeated React layout patterns map to repeated Atlas HTML patterns
- [ ] Standardize card internals so typography, spacing, and action rails can be matched without route-specific hacks
- [ ] Standardize table markup for internal list routes so density and styling stay consistent
- [ ] Standardize form markup and field groupings for internal and public forms
- [ ] Minimize unnecessary DOM depth while chasing visual parity so hydration cost and rerender cost stay reasonable on dense routes

### 7. CSS And Visual System Foundation

- [ ] Inventory every current public and internal shell class pattern in `shared/atlas/page.go`, `shared/atlas/public_sections.go`, `shared/atlas/products_cms.go`, and `shared/atlas/inventory_cms.go`
- [ ] Extract the React mock visual primitives that must be preserved:
  - glass surfaces
  - border radii
  - shadow depth
  - gradient and blur layers
  - spacing rhythm
  - badge treatments
  - button hierarchy
  - table and card density
- [ ] Create a side-by-side screenshot checklist for React mock versus GWC page parity before implementation starts
- [ ] Build a route-by-route class parity checklist for:
  - header
  - hero
  - section header
  - card
  - table
  - badge
  - button
  - input
  - modal or drawer
  - toast
- [ ] Define Atlas-specific equivalents for those primitives in Go-rendered class strings
- [ ] Decide which React utility patterns can be copied directly into GWC class strings
- [ ] Decide whether the parity layer lives entirely in utility classes or needs additions in `examples/static/css/example-shell.css`
- [ ] Move any non-trivial repeated CSS into `examples/static/css/example-shell.css` instead of repeating long class soup everywhere
- [ ] Match spacing, border radius, and typography scale to the React mocks before tuning color or motion
- [ ] Normalize public and internal color systems so the public surface and internal surface feel related but distinct
- [ ] Match backgrounds, shadows, borders, and blur layers after the structural spacing pass is complete
- [ ] Replace the remaining Atlas-specific visual fragments that still read like scaffold/demo UI instead of the React references
- [ ] Tune hover, focus, and active states to match React behavior
- [ ] Define a rendering-efficiency rule set for the rewrite so visual parity does not introduce unnecessary wrapper depth, repeated heavy gradients, or duplicated large DOM regions
- [ ] Identify which visual effects must remain always-on and which should degrade or simplify on mobile, reduced-motion, or low-power devices
- [ ] Verify mobile breakpoints against the React mock layouts, not just against current Atlas layouts
- [ ] Audit heavy CSS effects for paint cost and simplify them where they materially hurt dense internal screens or lower-end devices
- [ ] Avoid duplicating long utility chains when a shared class or shell helper would keep styles more maintainable and cheaper to evolve

### 8. Public Shell Rewrite

- [ ] Rebuild the Atlas public header to match the React storefront shell structure:
  - logo block
  - primary nav
  - mobile menu
  - utility affordances
- [ ] Rebuild the global public background treatment to match the React layered glow and gradient composition
- [ ] Redesign the public hero system so landing, catalog, product, warehouse detail, and availability routes all inherit the same shell logic as the React mock
- [ ] Replace current public cards with React-parity compositions:
  - hero cards
  - metric cards
  - category or feature tiles
  - product cards
  - section headers
- [ ] Rewrite landing page structure around the React storefront hierarchy while preserving Atlas copy and route intent
- [ ] Rewrite catalog page composition so filters, chips, and product grid align with the React store layout
- [ ] Ensure catalog filtering, chip updates, and search preview stay responsive under hydration by using deferred or derived state where appropriate
- [ ] Rewrite product detail composition to match the React item page rhythm:
  - image gallery
  - pricing block
  - availability and action badges
  - support copy
  - related products
- [ ] Make related-products, public comments, and other secondary product panels lazy or cached where that improves first-paint and route-transition stability
- [ ] Redesign warehouse list and warehouse detail routes with the same premium merchandised quality as the storefront mock
- [ ] Redesign warehouse availability route so it inherits the product-detail visual system instead of reading like a detached utility page

### 9. Internal Shell Rewrite

- [ ] Rebuild the Atlas internal header and mobile nav to match the React warehouse mock shell behavior
- [ ] Apply the internal background, surface, and card system from the warehouse mock across all `/app/*` routes
- [ ] Rewrite `/app/dashboard` using the React warehouse dashboard structure as the starting point:
  - top summary band
  - action cluster
  - low-stock or high-attention panel
  - activity feed
  - purchase-order summary
- [ ] Rewrite `/app/products` to use the stronger CRUD and table patterns from the warehouse mock while preserving Atlas product fields and workflows
- [ ] Rewrite `/app/products/:slug` to preserve Atlas product-editor responsibilities but adopt the cleaner form and preview structure from the React mock
- [ ] Rewrite `/app/inventory` to blend the current Atlas triage data with the warehouse mock's clearer filter and data-density approach
- [ ] Rewrite `/app/inventory/:sku` so lane roster, metrics, and lane editors feel consistent with the new internal shell
- [ ] Rewrite `/app/warehouses` and warehouse detail pages so they inherit the same ops language, table rhythm, and action affordances
- [ ] Rewrite `/app/purchase-orders` and purchase-order detail routes using the warehouse mock PO presentation as baseline inspiration
- [ ] Reframe `/app/receiving`, `/app/transfers`, `/app/comments`, and `/app/settings` so they no longer look like orphaned secondary pages under the new shell

### 10. Feature Translation And Interaction Parity

- [ ] Inventory all interactions present in the two React mocks:
  - mobile menu
  - animated drawers
  - toasts
  - filter chips
  - image gallery selection
  - quantity steppers
  - editable forms
  - preview panels
  - supplier reorder flow
- [ ] Map each interaction to one of three destinations:
  - direct Atlas port
  - Atlas-adapted equivalent
  - intentionally excluded because it conflicts with the real product
- [ ] Preserve non-JS form behavior where possible, then layer WASM enhancements on top instead of replacing progressive behavior
- [ ] Replicate the public mobile menu behavior in GWC hydration
- [ ] Replicate the internal mobile menu behavior in GWC hydration
- [ ] Replicate React-style section reveal and panel motion where it improves hierarchy without harming SSR stability
- [ ] Add Atlas-native drawer, modal, or sheet behavior where React uses separate panels and Atlas currently uses static forms
- [ ] Translate cart and checkout interaction ideas into Atlas-native public workflows:
  - quote request emphasis
  - restock action surfaces
  - product-question and review interactions
  - success and toast states
- [ ] Upgrade public success feedback so quote, restock, and comment submission feedback feels as polished as the React toast flow
- [ ] Upgrade internal save, create, update, and delete feedback across product, inventory, PO, receiving, and moderation flows
- [ ] Add performance-aware interaction rules so motion, drawers, and overlay stacks do not force unnecessary rerenders of the whole route shell

### 11. Production-Readiness Features

#### Accessibility

- [ ] Run a route-by-route accessibility pass on every public and internal page after the visual rewrite begins, not only at the end
- [ ] Preserve one clear page-level heading and a stable heading hierarchy on every route
- [ ] Audit keyboard navigation for header nav, mobile nav, filter controls, action rails, dense tables, drawers, modals, and side sheets
- [ ] Add explicit focus-visible styling parity for all interactive controls under the new visual system
- [ ] Verify focus restoration after modal, drawer, and toast-triggered workflows
- [ ] Add or refine live-region announcements for route changes, save success, validation errors, moderation decisions, and receiving or PO workflow completion
- [ ] Audit all public forms for accessible labels, helper text, error binding, and success feedback
- [ ] Audit all internal forms for accessible field grouping, error summaries, and field-level error ownership
- [ ] Replace any remaining non-semantic clickable containers with semantic buttons, links, form controls, table headers, or table cells where appropriate
- [ ] Revisit dense internal tables for caption, header scope, row labeling, and keyboard usability
- [ ] Verify overlay semantics for role, title, description, focus trap, inert background, escape handling, and nested confirmation behavior
- [ ] Add a reduced-motion pass for route transitions, overlays, hover states, and toast behavior
- [ ] Run a contrast audit after the visual rewrite lands, especially on glass surfaces, low-emphasis copy, badges, and table rows
- [ ] Add screen-reader-focused manual review stories for public buying flows and internal operational flows

#### Localization, language controls, and translations

- [ ] Add a visible language control strategy for the public shell and internal shell instead of treating locale as a hidden preference only
- [ ] Decide where locale switching lives on each surface:
  - public global shell
  - internal global shell
  - settings route
- [ ] Define the first production translation scope for public routes
- [ ] Define the first production translation scope for internal routes
- [ ] Create package-level translation resources instead of relying on English-only inline copy in route components
- [ ] Move repeated labels, button copy, status labels, helper text, and validation summaries into translation resources
- [ ] Localize route metadata, page titles, descriptions, and canonical-friendly text where appropriate
- [ ] Localize public success, error, and recovery messages
- [ ] Localize internal workflow copy for save, approve, reject, reconcile, transfer, and purchase-order actions
- [ ] Add locale-aware formatting for currency, counts, dates, times, and numeric summaries
- [ ] Audit the public and internal shells in RTL mode and fix any layout, icon-direction, spacing, or alignment regressions
- [ ] Add explicit fallback-language behavior when a translation key is missing
- [ ] Define the translation loading strategy for SSR and hydration so server HTML and client resume cannot disagree on locale text
- [ ] Add tests for language switching, locale persistence, and direct-entry SSR in non-default locales

#### Personalization and operator controls

- [ ] Expand theme, density, locale, and default-warehouse controls so they feel like production-grade shell features rather than simple demo fields
- [ ] Decide whether public users get independent theme and language controls separate from internal operator preferences
- [ ] Add route-stable persistence rules for theme, locale, density, and workspace defaults
- [ ] Audit whether rewritten routes respect persisted preferences on first SSR paint, not only after hydration
- [ ] Add clear reset-to-default actions for preferences where production users may need recovery from a bad state

#### SEO and discoverability

- [ ] Revisit SEO metadata for every rewritten public route after the visual and copy pass
- [ ] Ensure localized route titles and descriptions remain crawl-safe and consistent with canonical policy
- [ ] Recheck structured-data opportunities on public product and warehouse routes after the redesign
- [ ] Verify that non-indexable internal routes remain clearly separated from public discovery routes
- [ ] Re-audit metadata updates after client navigation so title, description, and canonical remain correct post-hydration

#### Security, privacy, and trust signals

- [ ] Recheck every rewritten form for CSRF token inclusion and same-origin behavior after HTML changes
- [ ] Ensure success states and validation errors do not leak sensitive internal workflow details into public surfaces
- [ ] Audit auth-sensitive internal routes after any shell or navigation rewrite so mock-auth redirects still behave predictably
- [ ] Add todos for replacing mock-auth assumptions cleanly if the rewrite exposes more shared shell behavior that a real auth layer would need to own

#### Reliability, recovery, and empty states

- [ ] Rewrite loading, empty, no-results, and recovery states so they match the new visual system instead of falling back to scaffold-style UI
- [ ] Ensure public route recovery pages remain branded and actionable under the new shell
- [ ] Ensure internal route recovery pages preserve enough context for the operator to recover without losing workflow orientation
- [ ] Add route-specific empty states for catalog, comments, purchase orders, receiving, warehouse item lists, and saved views where the rewrite introduces stronger layout structure
- [ ] Add degraded-network or partial-refresh recovery behavior for lazily loaded secondary panels so Atlas still feels intentional when async enhancement paths fail

### 12. Testing And Verification

- [ ] Add or update screenshot targets for every rewritten public route
- [ ] Add or update screenshot targets for every rewritten internal route
- [ ] Run the Atlas SSR validation suite after each major route family rewrite
- [ ] Add focused browser- or hydration-level tests for any new interaction that cannot be proven by SSR tests alone
- [ ] Verify navigation and form behavior with JS disabled where Atlas still promises progressive support
- [ ] Verify mobile menu, drawer, modal, and toast behaviors in WASM hydration
- [ ] Verify route recovery pages still look coherent under the new shell
- [ ] Add production-readiness verification stories for accessibility, locale switching, translation fallback, RTL, reduced motion, and preference persistence
- [ ] Add verification stories for cache invalidation, stale-data avoidance, derived-state consistency, and route-level responsiveness after mutations and navigation
- [ ] Add a benchmark or smoke-check plan for bootstrap size, loader latency, and dense-route interaction responsiveness
- [ ] Add one explicit demo walkthrough script for reviewers that shows SSR, hydration, overlays, route loaders, cached resources, derived state, devtools, locale switching, and recovery flows in a coherent sequence

## Near-Term Build Sequence

### Phase A

- [ ] Lock the route, copy, metadata, and server-response invariants that the rewrite must preserve
- [ ] Lock the SSR, router, bootstrap, cache, and derived-state rules before large-scale UI rewrites begin
- [ ] Refresh the framework-coverage inventory so the plan starts from shipped capabilities instead of stale assumptions
- [ ] Decide which additional GWC primitives will be mandatory in the first rewrite pass instead of deferred
- [ ] Define the first performance budgets and diagnostics checkpoints that every rewritten route family must satisfy

### Phase B

- [ ] Standardize the shared HTML patterns that both shells need before route-by-route rewrites begin
- [ ] Create the shared visual primitive layer for parity with the React mocks
- [ ] Lock the first accessibility, localization, and recovery-state requirements that the new shells must satisfy
- [ ] Rewrite public header, background, hero scaffolding, landing, and catalog
- [ ] Land the first public caching and lazy-secondary-content pass once the public shell structure stabilizes

### Phase C

- [ ] Rewrite product detail, warehouse list, warehouse detail, and warehouse availability
- [ ] Land the first public interaction parity pass using GWC overlays, transitions, async resources, and form-state helpers where appropriate
- [ ] Land the first production-quality public language control and translated-copy pass
- [ ] Finish the public accessibility, SEO, and recovery-state pass against the rewritten routes

### Phase D

- [ ] Rewrite the internal shell, then land dashboard, products, inventory, warehouse ops, purchase orders, receiving, transfers, comments, and settings in dependency order
- [ ] Land the internal workflow pass using richer form abstractions, shared state ownership, route revalidation, and async resources
- [ ] Finish the second-pass GWC feature adoption sweep on routes that still use simpler implementations
- [ ] Finish translation, RTL, reduced-motion, preference-persistence, recovery-state, and production-readiness audits
- [ ] Finish cache invalidation, payload-size, and responsiveness audits
- [ ] Finish interaction enhancements, then run full parity and regression review

## Outdated Planning Removed

These older ideas should not drive implementation anymore:

- "cart and checkout as literal Atlas features"
- "generic gadget storefront categories"
- "generic `/warehouse/*` admin route model"
- "treating the design mocks as standalone apps instead of Atlas route references"
- "older scaffold notes that list already-shipped Atlas SSR, seed, locale, or theme work as upcoming"
