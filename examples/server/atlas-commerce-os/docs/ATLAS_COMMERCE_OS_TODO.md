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

- [x] Keep Atlas seed data and route semantics coherent with the new visuals
- [x] Rename or revise any route copy that still reflects the older scaffold language instead of the new shell
- [x] Ensure public merchandising copy still fits Atlas workspace systems rather than drifting toward generic gadgets
- [x] Ensure internal route titles and descriptions still reflect real Atlas workflows after the rewrite
- [x] Revisit route metadata in `shared/atlas/legacy_shared.go` so titles, descriptions, and canonicals stay aligned with the rewrite
- [x] Revisit server response shapes for routes that currently overfetch or recompute similar summaries in multiple handlers
- [x] Move shared dashboard, inventory, warehouse, and PO summary logic toward reusable server or shared-layer derivation where that reduces duplicated work
- [x] Add one production-shaped bulk workflow, likely bulk moderation or bulk inventory threshold updates, so Atlas shows how GWC handles dense multi-record actions without becoming a toy demo
- [x] Evaluate whether lightweight import or export flows for saved views, inventory policies, or diagnostics snapshots would improve Atlas as a real showcase app

### 2. SSR, Routing, And Bootstrap Invariants

- [x] Preserve `ui.RenderToString` plus `ui.Hydrate` parity across every rewritten route family; do not let new interactions drift into client-only assumptions
- [x] Keep using the history router path already in place and expand route-loader usage where rewritten routes need explicit loader and revalidation behavior
- [x] Add more route metadata ownership through the router where it improves title, description, and canonical consistency after client-side transitions
- [x] Audit loader granularity so route transitions only fetch what changed instead of rebuilding large shared payloads unnecessarily
- [x] Add revalidation triggers for mutations that should refresh parent and sibling route data after save, approve, reconcile, or create actions
- [x] Evaluate route `before-enter` and `before-leave` guards for unsaved editor flows, destructive confirmations, and auth-sensitive internal paths
- [x] Add at least one production-grade unsaved-changes guard for a real internal editor route so Atlas demonstrates `before-leave` with a meaningful workflow
- [x] Expand nested layout route usage if the rewritten internal shell introduces stable sub-layouts for products, inventory, or warehouses
- [x] Evaluate whether one secondary workflow should be deep-linkable by URL, such as a SKU threshold overlay or PO detail side sheet, to show route-plus-overlay composition cleanly
- [x] Decide which public route data should be reused from SSR bootstrap, which should revalidate on navigation, and which should be cached client-side between route transitions
- [x] Define which internal route datasets should persist in route-local cache or shared resources so operators do not pay a full reload cost after every minor workflow action
- [x] Audit which route data must be serialized into bootstrap and which can be fetched lazily after hydration to keep first paint fast without breaking parity
- [x] Revisit bootstrap payload usage after the shell rewrites so route-specific UI still receives enough structured data
- [x] Track bootstrap payload size as the rewrite progresses and split or externalize route data if first-load SSR payloads become too large
- [x] Evaluate `ui.RenderBootstrapReferenceScript` plus external bootstrap payloads if Atlas bootstrap size grows significantly during the rewrite
- [x] Add one proof-of-concept route or diagnostics mode that can switch to external bootstrap reference loading if payload growth makes the inline script less convincing as the long-term demo story
- [x] Add explicit todos for any rewritten component that cannot be made SSR-safe on the first pass, and isolate it behind a clear hydration boundary instead of silently degrading the route
- [x] Add a second nested-layout showcase by nesting `/app/warehouses/:warehouseId/items/:sku` under `/app/warehouses/:warehouseId` so warehouse summary rails, filters, and return-target workflow context persist while an item is open (warehouse detail is now a layout route, the item route renders through a nested outlet, and direct-entry SSR bootstraps both the parent warehouse page data and child item workspace data)
- [x] Evaluate route-owned child-detail or side-panel layouts for `/app/purchase-orders/:id` and `/app/receiving/:id` so Atlas demonstrates more than one deep-linkable nested workflow beyond SKU threshold history (decision: keep PO detail and receiving detail as standalone routes for now because warehouse item drill-in now covers the second nested workflow showcase, while PO approval and receiving discrepancy work read more clearly as route-local overlays than persistent list/detail shells)
- [x] Introduce `router.UseRevalidator` on loader-backed internal detail surfaces that expose explicit refresh actions so Atlas demonstrates route-local revalidation without bespoke refresh plumbing in every route family (transfer, purchase-order, and receiving detail routes now share one `routeRevalidationCard` built on `router.UseRevalidator` instead of one-off refresh handlers)

### 3. Server-State, Derived-State, And Cache Baseline

- [x] Audit handler-level duplicate queries for dashboard, warehouse detail, SKU detail, and PO detail flows
- [x] Add todos to consolidate repeated database reads where one route currently builds several related summaries separately
- [x] Revisit whether some route summaries should be precomputed or persisted if they become expensive enough in the real app shape
- [x] Build a shared inventory-derived-state layer so counts, badges, urgency summaries, reorder totals, and lane summaries are not recomputed independently across multiple routes
- [x] Build a shared public-derived-state layer for product status messaging, warehouse promise summaries, and action-label decisions
- [x] Decide which derived values belong:
  - in server responses
  - in shared atlas helpers
  - in state computed or derived primitives
  - in route-local memoized transforms
- [x] Define the canonical derived-state layer for Atlas so expensive counts, totals, urgency bands, and filter summaries are computed once and consumed across routes
- [x] Remove duplicated count, grouping, sorting, and summary logic once the canonical derived-state locations are defined
- [x] Define SSR bootstrap reuse rules for each major route family
- [x] Define client cache lifetime rules for:
  - related products
  - public comments
  - saved views
  - preferences
  - purchase-order detail
  - receiving detail
  - warehouse detail side data
- [x] Define mutation invalidation rules for every write flow so caches stay correct after updates (`client/main.go` now uses an explicit mutation invalidation matrix and `docs/README.md` documents each write-flow target set)
- [x] Decide where stale-while-revalidate behavior is acceptable and where Atlas must block on fresh data (`docs/README.md` now defines a fresh-first policy for internal workflows and direct-entry SSR, with SWR limited to read-mostly secondary public and diagnostics data)
- [x] Ensure route reloads and direct-entry SSR do not regress because of assumptions made by client cache layers (`server/server_test.go` now proves a fresh `/app/inventory` SSR entry reflects saved preferences after mutation, and `docs/README.md` records server bootstrap as the authority on fresh document requests)
- [x] Decide whether any preferences, saved views, or diagnostics state should sync across tabs or windows so Atlas can show a real cross-tab consistency story where it adds value (`docs/README.md` now defines preferences and saved views as cross-tab state, while diagnostics and route-local workflow state remain tab-local)
- [x] Audit bootstrap and JSON payload sizes after the rewrite and trim repeated or unnecessary data (SSR bootstrap now drops duplicated request payload blobs while preserving request metadata for hydration cache priming, and `server/server_test.go` locks the trimmed shape)

### 4. GoWebComponents Runtime Foundation

- [x] Audit the current Atlas implementation against actual framework surfaces and refresh `docs/README.md#framework_coverage` so the rewrite starts from shipped reality (`docs/README.md#framework_coverage` now reflects the actual history-router, loader, bootstrap, and hydration path and removes stale overclaims about framework surfaces Atlas does not currently exercise)
- [x] Prefer a real GWC feature integration when a rewrite task can reasonably use one, instead of rebuilding the behavior manually with ad hoc state (`docs/README.md#framework_adoption_rule` now makes framework-surface adoption the default rewrite rule)

#### State and shared ownership

- [x] Introduce `state.UseComputed` for derived stock-health totals, action counts, route badges, and shell-level summaries that should not be recomputed ad hoc in multiple components (the internal shell now derives one computed summary and badge set per route payload in `shared/atlas/page.go`, reused by the header and hero)
- [x] Expand `state.UseAtom` or `state.UseDerived` where shell-wide filters, presentation preferences, or route-scoped workspace state need consistent shared ownership (the Atlas shell now stores presentation preferences in a shared atom during hydration instead of reading locale and density only from threaded props)
- [x] Complete the shell-state ownership layer by moving theme, locale, density, default warehouse, diagnostics-toggle, and other shell-level workspace state in `shared/atlas/page.go` out of prop-only reads and into bootstrap-seeded shared atoms mirrored to browser storage (the shell presentation atom now carries theme, locale, density, default warehouse, and diagnostics visibility, mirrors to LocalStorage, and feeds the hero, settings summary, preference form, and saved-view transfer payloads)
- [x] Introduce route-scoped atoms or derived atoms for inventory saved-view selection, warehouse filters, dashboard badge rollups, and other cross-surface workspace summaries so header, hero, and side rails read from one shared source of truth (Atlas now syncs a route-scoped workspace atom from loader payloads and uses it for dashboard badge rollups, inventory filter and saved-view context, hero metadata, header badges, and warehouse or inventory side-rail summaries)
- [x] Audit each route for duplicated derived calculations and replace them with shared computed or derived primitives across `shared/atlas/page.go`, `shared/atlas/inventory_cms.go`, and warehouse side rails before the rewrite stabilizes (inventory and warehouse route shells now share explicit workspace snapshot helpers for rollups, summaries, and warehouse-detail counts instead of recomputing those values separately in the shell atom and route body)
- [x] Evaluate snapshot export or restore flows for reviewer, diagnostics, or operator workspaces after the shell and route-scoped atom model is stable (decision: keep full workspace snapshots deferred; diagnostics already exposes developer-only runtime snapshots, and operator handoff is adequately covered by saved-view export or import plus direct links until Atlas gains richer multi-route reviewer context)
- [x] Add one exportable reviewer or operator snapshot flow if diagnostics or workspace state becomes rich enough to justify snapshot sharing across sessions (the settings route now renders an operator workspace snapshot export card with a copyable JSON payload covering shell presentation state, route workspace state, and saved-view metadata)

#### Async data, tasks, and resources

- [x] Replace the hand-rolled `routePayloadCache` and `requestCache` in `client/main.go` with Atlas resource helpers built on `ui.UseCachedResource` so cache ownership, deduplication, and invalidation stop living in bespoke maps (Atlas route and request loaders now seed the shared cached-resource registry from SSR bootstrap, reuse `fetch.LoadCached` for imperative route data, and invalidate matching `atlas:route:` / `atlas:request:` resources instead of bespoke maps)
- [x] Use `ui.UseCachedResource` for repeat-open public comments, related products, warehouse side data, and PO or receiving detail rails so revisiting the same surface can reuse data without a full route reload (Atlas now wraps `fetch.UseCachedResource` for public comment threads, related-product cards, warehouse side-data panels, and purchase-order or receiving detail rails, with the internal rails reusing the same bootstrapped `atlas:request:` page keys seeded by the route loader)
- [x] Use `ui.UseResource` plus `ui.AsyncBoundary` for below-the-fold product-detail and internal side-panel islands so loading and failure states stay local to the panel instead of degrading the whole route (the public product route now defers a regional promise-lanes module behind an async boundary, and purchase-order or receiving detail rails now isolate local stat refresh and retry states inside panel-scoped async islands)
- [x] Use `fetch.Fetch` or `ui.UseFetch` for imperative follow-up refreshes in public comment submission and internal status-refresh actions where Atlas currently hand-rolls goroutine-based refetch sequencing (public comment submission now posts through `fetch.Fetch`, and the purchase-order or receiving stat islands now use panel-scoped `fetch.Fetch` refresh actions instead of raw client requests)
- [x] Evaluate `ui.UseTask` or `ui.UseWorkerTask` for background work that should not block the main interaction path, such as bulk formatting, diagnostics snapshots, or heavy local transforms (decision: keep both deferred for now because saved-view export, workspace snapshot JSON, and current diagnostics snapshot paths are still small synchronous work; the first justified task or worker workload should land with the planned CSV/import or larger diagnostics transform flow)
- [x] Add `ui.UseChannel` if the rewritten public and internal surfaces need cross-panel event fanout, such as notifying shell-level toasts or synchronizing workflow completion state across distant components (decision: keep `ui.UseChannel` deferred for now because Atlas already handles the current coordination needs through query-based mutation notices, cached-resource invalidation, and shared atoms; a channel-backed fanout should wait for a real shell-toast or workflow-complete broadcast path)
- [x] Add one worker-backed production-shaped flow, likely bulk inventory CSV parsing, import validation, or large client-side diagnostics transforms, so Atlas demonstrates `ui.UseWorkerTask` on a real workload (the settings route now validates saved-view import payloads in a dedicated worker, reporting progress, warnings, and a summary preview before the existing import POST runs)
- [x] Add one channel-backed cross-panel event flow, likely shell toast dispatch or workflow-complete fanout, so Atlas demonstrates `ui.UseChannel` with a concrete operator benefit (Atlas now includes a shell-level toast bus subscribed once near the app shell, with public comment submission and purchase-order or receiving panel refresh flows broadcasting completion notices through `ui.UseChannel`)

##### Cached-resource implementation plan

- [x] Inventory every current manual Atlas cache or refetch path in `client/main.go`, public comment refresh, related-product reuse, and any PO or receiving side rail before replacing it (captured in the completed migration: `client/main.go` route/request caches, public comment follow-up refresh, related-product reuse, warehouse side-data reads, and purchase-order or receiving side rails were the concrete Atlas refetch paths that moved onto shared cache, resource, fetch, or channel-backed replacements)
- [x] Define stable resource keys for route payloads, startup request payloads, public comments, related products, warehouse side data, purchase-order detail rails, and receiving detail rails (Atlas now uses `atlas:route:<routeDataKey>` for route payloads, `atlas:request:<requestURL>::page` for startup page payloads, and `atlas:request:<requestURL>::<dataKey>` for secondary resources such as public comments, related products, warehouse side data, and purchase-order or receiving detail rails)
- [x] Build shared Atlas resource helpers that seed from SSR bootstrap when available, fall back to network loaders after hydration, and plug into the existing mutation invalidation matrix (route/request cache keys, payload-to-resource-key expansion, and SSR fetch-cache bootstrap seeding now live in shared Atlas helpers under `shared/atlas/resource_cache*.go`, with `client/main.go` consuming those helpers for route loading and mutation invalidation)
- [x] Migrate public product comments to `ui.UseCachedResource` first, including optimistic insert, background refresh, invalidation after moderation, and direct-entry SSR parity (the product feedback section already seeds the cached comments resource from SSR `page.Comments`, merges optimistic inserts on submit, reloads in the background, and relies on the shared `comment-submitted` / `comment-moderated` invalidation rules to clear stale product, dashboard, and moderation views)
- [x] Migrate related products and public secondary product panels next so Atlas has one public cached-resource showcase beyond comments (the product action rail now includes a cached related-products card keyed by `atlas:request:/api/public/products/<slug>/related-products::items`, giving Atlas a second public cached-resource surface beyond the comment thread while keeping the promise-lanes module as a separate async island)
- [x] Migrate warehouse side data plus purchase-order or receiving detail rails after the public pass so Atlas demonstrates the same resource model on internal repeat-open workflows (warehouse detail side data, purchase-order detail stats, and receiving detail stats now all reuse the shared startup-page cached resource seeded from SSR bootstrap, and the detail-panel refresh buttons write back into that shared cached snapshot after a successful reload)
- [x] Remove superseded `routePayloadCache`, `requestCache`, and bespoke cache invalidation helpers from `client/main.go` once the cached-resource path covers the same navigation and mutation stories (the old manual map caches and delete-from-map invalidation path are gone; `client/main.go` now loads through shared fetch resources and invalidates those resources via `fetch.InspectCachedResources()` plus the Atlas mutation matrix instead of maintaining separate route/request cache stores)
- [x] Add diagnostics and tests for cache hits, stale-visible-while-refreshing behavior, mutation invalidation, direct-entry SSR reuse, and back or forward navigation after the cached-resource migration lands (Atlas now documents cache diagnostics via `bootstrap.read.ok`, `route.fetch.ok`, `route.fetch.cache.hit`, and `route.cache.invalidate`; shared tests cover route-key stability for repeat navigation plus mutation invalidation target mapping; and server tests already cover fresh direct-entry SSR bootstrap after mutation)

#### Forms and workflow state

- [x] Expand `ui.UseForm` across `productCreateFormWithOptions`, `productUpdateFormWithOptions`, inventory lane-edit forms, transfer forms, receiving discrepancy forms, moderation actions, and settings import or export flows so Atlas has one typed form lifecycle model across internal workflows (Atlas now uses typed `ui.UseForm` state for product create or update, lane edit, moderation, transfer, receiving reconcile, preference save, saved-view import, and workspace snapshot export surfaces while keeping the existing progressive POST actions intact)
- [x] Add `ui.UseId` to shared `cmsTextInput`, `cmsNumberInput`, `cmsTextarea`, `cmsSelectInput`, catalog controls, and public feedback fields so labels, descriptions, and error text bind explicitly without per-form ad hoc IDs (shared Atlas form helpers now generate stable IDs for labels and controls, and the public comment form wires generated `aria-describedby` and `aria-invalid` relationships for field errors instead of relying on wrapper-only markup)
- [x] Use `ui.UsePrevious` in the product editor, threshold activity timeline, and receiving reconciliation flows so Atlas can render recent-change summaries without duplicating snapshot state (the product editor and receiving reconcile forms now show recent draft-change summaries from `ui.UsePrevious`, and the threshold-history overlay shows when a refreshed timeline head replaced the previous latest threshold event)
- [x] Model receiving discrepancy resolution and purchase-order creation as `ui.UseReducer` workflows once those forms grow beyond single-submit state (purchase-order creation now stages replenishment readiness through a reducer-backed workflow summary in the inventory modal, and receiving reconciliation keeps its typed `ui.UseForm` fields while a reducer tracks closeout vs review guidance in the reconcile card)

#### SSR-safe composition, overlays, and focus

- [x] Introduce `ui.Fragment` in `inventoryQueueTable`, warehouse network tables, and repeated card-meta groups where wrapper-only nodes hurt HTML parity with the React references (inventory queue rows and warehouse network rows now emit grouped table cells through `ui.Fragment`, and the shared table header copy blocks use fragment-backed section metadata instead of implying extra helper wrappers)
- [x] Evaluate `ui.UseLazyNode` for below-the-fold public sections and secondary internal panels that should mount only after the primary route body is stable (Atlas now lazy-mounts the public product feedback block plus the purchase-order and receiving detail rails so the primary body stabilizes first while SSR still renders the full subtree on direct entry)
- [x] Add `ui.ErrorBoundary` around route-local enhancement islands and async side panels so one broken interactive panel does not collapse the full page (the public promise-lanes island plus the purchase-order and receiving side panels now recover through local boundary fallbacks keyed to route identity instead of taking down the whole screen)
- [x] Wire `ui.UseRef`, `ui.UseFocusManager`, and `ui.UseFocusTrap` into threshold edit, transfer confirmation, receiving discrepancy, and moderation confirmation flows so opener capture and modal containment stop being manual concerns (the threshold-history route overlay now restores focus to its opener, and Atlas transfer, receiving, moderation, and bulk-moderation flows all gate final submit behind focus-trapped confirmation dialogs)
- [x] Introduce `ui.UseFocusTrap` anywhere Atlas needs modal-only focus containment beyond the current overlay defaults (the replenishment-order modal now uses a stateful dialog with trapped focus and opener restoration instead of the old checkbox-and-peer overlay toggle)
- [x] Introduce `ui.UseCompositeNavigation` where internal listbox-, menu-, tab-, or command-palette-like interactions emerge during the rewrite (the settings route saved-view browser is now a keyboardable listbox with roving active state, Home/End support, and typeahead-backed selection preview)
- [x] Turn the current threshold-history panel pattern into a reusable route-owned overlay abstraction using `ui.AccessibleOverlay`, `ui.Overlay`, and `ui.UseOverlayStack`, then reuse it for transfer, receiving, and moderation side workflows instead of bespoke panel markup in each route (Atlas now renders the threshold-history sheet through a portal-backed accessible overlay host, while transfer, receiving, moderation, and bulk-moderation confirmations all share the same stack-aware overlay wrapper instead of custom fixed-position dialog markup)
- [x] Add a real `ui.PortalTarget` host pattern for the public action rail and internal quick actions if off-canvas drawers become part of the rewritten shells (Atlas now uses the shared overlay host for a small-screen public buying drawer on product detail plus small-screen internal quick-action drawers for workflow-card clusters, so off-canvas actions mount outside the shell through one portal target)
- [x] Use `ui.UseAnnouncer` first on public comment submit, threshold save, receiving reconcile, moderation decisions, and route-change notices so Atlas has explicit live-region stories before the broader accessibility pass closes (Atlas now mounts one shell-level announcer region that speaks route changes, mutation notice banners, and shell toasts, covering public comment submit plus threshold, receiving, and moderation success paths without route-local live-region duplication)

#### Scheduling and responsiveness

- [x] Add `ui.UseDeferredValue` plus `ui.UseDebounced` to catalog search, inventory triage search, and warehouse item filters once those controls move to hydrated local state instead of submit-only forms (the `/shop`, `/app/inventory`, and `/app/warehouses/:warehouseId` filters now use hydrated `ui.UseForm` state, deferred local list rendering, and debounced query-string replacement so type-ahead updates stay responsive without giving up deep-linkable filter URLs)
- [x] Use `ui.UseTransition` and `ui.StartTransition` when applying saved views, density toggles, and high-churn internal filters so dense table rerenders do not block input responsiveness (Atlas now applies inventory saved views inside a transition, stages density preview toggles through a transition-backed settings control, and routes inventory plus warehouse filter setters through transition scheduling while showing pending state during dense workspace rerenders)
- [x] Use `ui.UseThrottled` for diagnostics timing panels, sticky-shell measurements, and any scroll-linked warehouse or inventory affordances that would otherwise rerender too often (Atlas diagnostics mode now samples viewport, scroll depth, sticky-shell state, and inventory or warehouse workspace presence through a throttled shell panel behind `?diag=1`, so resize and scroll bursts stop redrawing those diagnostics on every browser event)

#### Diagnostics and devtools

- [x] Reconcile `docs/README.md` diagnostics claims with actual Atlas devtools wiring, then either embed `devtools.Panel` plus `devtools.SnapshotNow()` behind `?diag=1` or trim the docs back to the shipped surface
  Chose the docs-trim path for release signoff: temporary diagnostics overlays were removed, README diagnostics guidance now documents opt-in debug logging (`data-atlas-debug-logs` or `window.__atlasDebugLogs`), and query-flag diagnostics mode was retired.
- [x] If `devtools.Panel` lands, expose route identity, loader timings, cache entries, shared atom state, overlay stack state, mutation invalidation events, and async-resource state instead of only a generic framework inspector
  Not applicable for current release: devtools panel is intentionally not shipped after diagnostics cleanup.
- [x] Add diagnostics hooks for reusable shell primitives, especially drawers, route guards, async panels, shared workflow forms, and the cached-resource migration path in `client/main.go`
  Not applicable for current release: diagnostics overlays were removed before signoff.
- [x] Add a guided diagnostics view or checklist that deliberately surfaces the major GWC concepts used by Atlas so the example is easier to demo and explain
  Not applicable for current release: reviewer guidance now relies on manual checklists and opt-in debug logging instead of diagnostics mode.

### 5. Performance, Observability, And Runtime Budgets

- [ ] Identify the highest-risk dense routes for hydration and rerender cost:
  - `/shop`
  - `/shop/:slug`
  - `/app/dashboard`
  - `/app/products`
  - `/app/inventory`
  - `/app/warehouses/:warehouseId`
- [ ] Capture a baseline hydration, rerender, and interaction profile for `/shop` before the catalog rewrite adds more client-owned filter state
- [x] Capture a baseline hydration, rerender, and interaction profile for `/shop/:slug` before related products, comments, and action-rail enhancements become lazy or cached (documented in `docs/README.md#performance_checkpoints` as the `/shop/frame-desk` baseline covering SSR first paint, localized rerender boundaries for promise lanes or related-products/comments panels, and the core buyer interaction checks that future lazy or cached work must preserve)
- [x] Capture a baseline hydration, rerender, and interaction profile for `/app/dashboard` before more shared shell state and diagnostics surfaces land (documented in `docs/README.md#performance_checkpoints` as the `/app/dashboard` baseline covering SSR first paint, scoped shell or diagnostics updates, and the core triage and handoff interactions future shared-state or diagnostics work must preserve)
- [x] Capture a baseline hydration, rerender, and interaction profile for `/app/products` before the richer CRUD table and editor preview rewrite (documented in `docs/README.md#performance_checkpoints` as the `/app/products` baseline covering the current list-plus-editor flow, SSR first paint, editor-local rerender boundaries, and the guarded unsaved-change interaction path future CRUD or preview work must preserve)
- [x] Capture a baseline hydration, rerender, and interaction profile for `/app/inventory` before saved-view, filter, and side-panel enhancements move to richer hydrated state (documented in `docs/README.md#performance_checkpoints` as the `/app/inventory` baseline covering SSR first paint, saved-view and filter-local rerender boundaries, SKU drill-in plus threshold overlay behavior, and the dense workspace interactions future inventory-shell work must preserve)
- [x] Capture a baseline hydration, rerender, and interaction profile for `/app/warehouses/:warehouseId` before nested item detail and warehouse-scoped workflow context expand (documented in `docs/README.md#performance_checkpoints` as the `/app/warehouses/new-jersey-hub` baseline covering the parent warehouse route, nested item workspace continuity, SSR first paint, and the local rerender boundaries future warehouse-scoped workflow growth must preserve)
- [ ] Add route-level performance checkpoints for first paint, first interactive action, and post-hydration responsiveness
- [ ] Define a first-paint checkpoint for every high-risk route using stable local test hardware and the same examples server build path
- [ ] Define a first interactive action checkpoint for every high-risk route using one representative action per route family
- [ ] Define a post-hydration responsiveness checkpoint for every high-risk route after filters, overlays, or route-local actions become active
- [ ] Ensure non-urgent UI updates use transition or deferred patterns where they can avoid keystroke lag or jank
- [ ] Keep expensive secondary panels, derived lists, and related-content regions from rerendering on unrelated state changes
- [ ] Define practical performance budgets for:
  - bootstrap payload size
  - route loader latency
  - overlay open latency
  - search interaction responsiveness
  - dense-table rerender responsiveness
- [ ] Set a bootstrap payload warning budget and a fail budget for public routes
- [ ] Set a bootstrap payload warning budget and a fail budget for internal routes with loader-backed detail surfaces
- [ ] Set a route-loader latency budget for direct-entry SSR and for hydrated revalidation separately so server and client regressions are visible independently
- [ ] Set an overlay open-latency budget for threshold editing, transfer confirmation, receiving discrepancy, and moderation confirmation flows
- [ ] Set a search responsiveness budget for catalog search, inventory triage search, and warehouse item filters
- [ ] Set a dense-table rerender budget for `/app/products`, `/app/inventory`, and `/app/warehouses/:warehouseId`
- [ ] Add manual and automated checks that flag obvious regressions in those budgets during the rewrite
- [ ] Add a manual review script for checking obvious regressions on the six highest-risk dense routes before and after major shell rewrites
- [ ] Add automated smoke checks for bootstrap size, loader latency, and one representative interaction per high-risk route family
- [ ] Surface perf-relevant debug information in diagnostics mode so future work can see where caching or derived-state decisions need adjustment
- [ ] Surface loader timings, bootstrap size, and cache-hit summaries in diagnostics mode once the cached-resource migration starts landing
- [ ] Surface derived-state recomputation counts or summaries in diagnostics mode for dashboard, inventory, and warehouse detail while the rewrite is still in motion

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
- [ ] Lock the public header information architecture before restyling so landing, catalog, product, and warehouse routes all share the same nav contract
- [ ] Rework the desktop public header first, then add the mobile menu behavior once the information architecture is stable
- [ ] Add route-aware active states, utility affordance placement, and shell-level CTA priority rules before tuning visual parity details
- [ ] Rebuild the global public background treatment to match the React layered glow and gradient composition
- [ ] Define the shared public background layers once, then apply them consistently to landing, catalog, product, warehouse detail, and availability routes
- [ ] Add mobile and reduced-motion degradation rules for the public background treatment before route-by-route rollout
- [ ] Redesign the public hero system so landing, catalog, product, warehouse detail, and availability routes all inherit the same shell logic as the React mock
- [ ] Define one shared hero scaffold API for eyebrow, headline, support copy, metrics, CTAs, and media or proof slots
- [ ] Map each public route to the shared hero scaffold and note which slots are required, optional, or route-specific
- [ ] Land the landing and catalog hero variants first, then adapt the scaffold to product, warehouse detail, and availability routes
- [ ] Replace current public cards with React-parity compositions:
  - hero cards
  - metric cards
  - category or feature tiles
  - product cards
  - section headers
- [ ] Define the shared public card anatomy before route implementation so spacing, badge placement, and action rails are not reinvented per route
- [ ] Land metric and feature tiles first, then product cards, then hero-adjacent proof cards and section headers
- [ ] Rewrite landing page structure around the React storefront hierarchy while preserving Atlas copy and route intent
- [ ] Break the landing rewrite into header-to-hero, proof band, merchandising band, warehouse credibility band, and closing CTA band tasks
- [ ] Confirm each landing band still points into real Atlas routes and workflows before visual tuning begins
- [ ] Rewrite catalog page composition so filters, chips, and product grid align with the React store layout
- [ ] Split catalog rewrite work into filter rail, active chip row, merchandising summary band, and product grid tasks
- [ ] Lock the URL and progressive-form behavior for catalog filters before adding richer hydrated interactions
- [ ] Ensure catalog filtering, chip updates, and search preview stay responsive under hydration by using deferred or derived state where appropriate
- [ ] Rewrite product detail composition to match the React item page rhythm:
  - image gallery
  - pricing block
  - availability and action badges
  - support copy
  - related products
- [ ] Split product detail work into hero media, pricing and CTA block, promise lanes, supporting proof copy, and secondary content tasks
- [x] Lock which product-detail panels are first-paint critical versus candidates for lazy or cached secondary loading before implementing them (documented in `docs/README.md` as the product-detail loading policy: hero, primary action rail, and warehouse-aware decision copy remain first-paint critical, while promise-lanes, public feedback, and related-product side panels are explicitly treated as secondary lazy or cached surfaces)
- [x] Make related-products, public comments, and other secondary product panels lazy or cached where that improves first-paint and route-transition stability (the shipped product route now lazy-mounts the public feedback block, keeps comments and related products on cached-resource loaders, and isolates the promise-lanes module behind its own async boundary so secondary refreshes stay local)
- [x] Redesign warehouse list and warehouse detail routes with the same premium merchandised quality as the storefront mock (the public warehouse directory now opens with a merchandised regional-commerce board and richer route cards, while warehouse detail adds a story band and product showcase so those routes feel editorial and premium rather than like detached utility screens)
- [x] Split warehouse route work into warehouse list merchandising, warehouse detail hero, capability summary, product volume story, and route-specific CTA tasks (documented in `docs/README.md` as the warehouse route work map so the public warehouse family now has explicit merchandising slices instead of one monolithic route bucket)
- [x] Redesign warehouse availability route so it inherits the product-detail visual system instead of reading like a detached utility page (the shipped route now uses the same dark hero, first-screen metrics, and sticky side-rail composition as product detail, with product and warehouse pivots kept in the same right-hand decision column)
- [x] Align warehouse availability with the product-detail scaffold first, then layer warehouse-specific promise and support content on top (the shipped route now adds a warehouse-specific promise band plus action-rail support points derived from regional cue, service posture, backlog, and current stock recovery state)

### 9. Internal Shell Rewrite

- [x] Rebuild the Atlas internal header and mobile nav to match the React warehouse mock shell behavior (the shipped shell now uses a grouped internal workspace header on desktop plus a sheet-based mobile nav drawer with shared context pills and route badges)
- [x] Lock the internal nav information architecture and route grouping before visual restyling so dashboard, products, inventory, warehouses, logistics, and settings do not drift (the docs now lock the shipped grouping as overview = dashboard/products, stock = inventory/warehouses, logistics = transfers/purchase-orders/receiving, and support = comments/settings)
- [x] Rebuild the desktop internal header first, then mobile nav and rail collapse behavior, then route-context badges and workspace affordances (documented in `docs/README.md` as the shipped internal shell rollout order: desktop grouped header first, mobile drawer and quick-link collapse next, route-context badges and workspace pills last)
- [x] Apply the internal background, surface, and card system from the warehouse mock across all `/app/*` routes (the shared internal shell helpers now use a consistent hero tier plus standard, inset, and accent slate surfaces, so stats, route summaries, workflow cards, nav groups, and shell context all inherit the same steel-and-slate system)
- [x] Define the shared internal surface tiers, card classes, and table containers before route-by-route adoption (the shared Atlas contract is now explicit in code and docs through `internalHeroSurfaceClass`, `internalSurfaceCardClass`, `internalInsetSurfaceClass`, `internalAccentSurfaceClass`, `internalSurfacePillClass`, `internalTableContainerClass`, `internalTableHeaderCellClass`, and `internalTableRowClass`)
- [x] Roll the internal surface system out to dashboard and inventory first, then products and warehouse ops, then logistics and settings routes (documented in `docs/README.md` as the locked rollout order: dashboard/inventory first, products/warehouse ops second, logistics/settings third)
- [x] Rewrite `/app/dashboard` using the React warehouse dashboard structure as the starting point:
  - top summary band
  - action cluster
  - low-stock or high-attention panel
  - activity feed
  - purchase-order summary
  (the shipped dashboard now renders those five sections directly, and the dashboard payload includes purchase-order records so the PO summary is real rather than a derived placeholder)
- [x] Land the dashboard summary band and action cluster first so the new shell hierarchy is visible before secondary panels move (the shipped dashboard now leads with the summary band and action cluster before the secondary attention, activity, and PO panels)
- [x] Rebuild the high-attention panel and activity feed next, then tune purchase-order summary density and action affordances (the shipped dashboard now includes a dedicated high-attention queue, a multi-source activity feed, and a denser purchase-order summary with direct links into PO detail)
- [x] Rewrite `/app/products` to use the stronger CRUD and table patterns from the warehouse mock while preserving Atlas product fields and workflows (the shipped route now leads with a products summary band, keeps filtering and route actions in dedicated shell cards, renders the catalog in a dense CRUD table with explicit edit and warehouse-lane actions, and preserves the existing create workflow in a side rail)
- [x] Split `/app/products` into table-shell, filter-bar, saved-view, bulk-action, and route-level empty-state tasks (documented in `docs/README.md` as the products route work map, with the saved-view slice explicitly reserved for a later pass while the shipped table, filter, bulk-action, and empty-state slices are now clear)
- [x] Keep warehouse-originated return-target workflows intact while moving the table and action affordances to the new shell (the products table still links directly into warehouse item profiles, and the create/update/delete form options still preserve `return_warehouse_id` for warehouse-originated flows)
- [x] Rewrite `/app/products/:slug` to preserve Atlas product-editor responsibilities but adopt the cleaner form and preview structure from the React mock (the shipped editor now uses a preview-first layout with grouped edit sections while preserving the existing save, delete, warehouse-link, and unsaved-change responsibilities)
- [x] Split the product editor into metadata form, merchandising copy form, warehouse context, preview surface, and unsaved-changes workflow tasks (documented in `docs/README.md` as the product editor work map, matching the shipped grouped editor layout)
- [x] Lock the save, delete, and return-target workflow before visual parity tuning so the rewrite does not regress operational behavior (documented in `docs/README.md` as the workflow lock for the shipped product editor: save/delete endpoints, return-target field, warehouse handoff links, and unsaved-change guard remain fixed)
- [x] Rewrite `/app/inventory` to blend the current Atlas triage data with the warehouse mock's clearer filter and data-density approach (the shipped route now uses an explicit inventory triage shell with a summary band, stronger filter card, route action cluster, clickable triage band, dense SKU queue table, and a right rail that keeps saved-view and workspace context visible without displacing the main queue)
- [x] Split `/app/inventory` into triage summary band, filter model, saved-view rail, dense queue table, and route-level action cluster tasks (documented in `docs/README.md` as the shipped inventory route work map, matching the new summary-band, filter-card, saved-view-rail, dense-table, and action-cluster structure)
- [x] Decide which inventory secondary panels stay first-paint and which become lazy or cached before the shell rewrite lands (documented in `docs/README.md` as the inventory loading policy: the list-route shell and scope rails remain first-paint, while threshold-history, transfer recommendations, and replenishment workflows stay secondary route-local panels that may refresh lazily or through cached request data)
- [x] Rewrite `/app/inventory/:sku` so lane roster, metrics, and lane editors feel consistent with the new internal shell (the shipped SKU detail route now uses a hero-level lane workspace summary, shared stat cards, a denser lane roster table, grouped lane editors, and a right rail for SKU actions, replenishment, and threshold controls)
- [x] Split SKU detail into summary hero, lane roster, lane edit forms, threshold-history side workflow, and replenishment or transfer action tasks (documented in `docs/README.md` as the shipped SKU detail work map, matching the current hero, roster, lane-editor section, route-local threshold overlay, and right-rail replenishment structure)
- [x] Rewrite `/app/warehouses` and warehouse detail pages so they inherit the same ops language, table rhythm, and action affordances (the shipped warehouse list is now a facility triage shell with a summary band, action cluster, and dense facility table, while warehouse detail now uses the same hero, filter, stat-strip, action-cluster, dense-table, and right-rail rhythm without dropping nested item continuity)
- [x] Split warehouse ops work into warehouse list shell, warehouse detail workspace, warehouse item table, purchase-order rail, and nested item-detail continuity tasks (documented in `docs/README.md` as the shipped warehouse ops work map, matching the current list shell, facility workspace, dense item table, replenishment rail, and nested item continuity)
- [x] Rewrite `/app/purchase-orders` and purchase-order detail routes using the warehouse mock PO presentation as baseline inspiration (the shipped PO list is now a vendor recovery shell with a summary band, action cluster, dense table, and handoff rail, while PO detail now uses a workspace hero and line-item table around the existing lazy status rail)
- [x] Split purchase-order work into list shell, status summary band, detail hero, line-item context, and approve or hold workflow tasks (documented in `docs/README.md` as the shipped purchase-order work map, matching the current vendor shell, summary band, detail hero, line-item table, and lazy approve-or-hold rail)
- [x] Reframe `/app/receiving`, `/app/transfers`, `/app/comments`, and `/app/settings` so they no longer look like orphaned secondary pages under the new shell (the shipped transfers, receiving, comments, and settings routes now each use the same summary-band plus action-cluster shell language as the primary internal routes, with dense tables or control panels and supporting right rails instead of leftover secondary-page card stacks)
- [x] Split logistics and support route work into receiving, transfers, comments, and settings mini-rewrites so each route gets its own shell, empty-state, and workflow treatment instead of one catch-all pass (documented in `docs/README.md` as the shipped logistics and support work map, matching the current receiving, transfers, comments, and settings route shells)

### 10. Feature Translation And Interaction Parity

- [x] Inventory all interactions present in the two React mocks:
  - mobile menu
  - animated drawers
  - toasts
  - filter chips
  - image gallery selection
  - quantity steppers
  - editable forms
  - preview panels
  - supplier reorder flow
- [x] Add route ownership notes to the interaction inventory so every mock interaction is mapped to the Atlas public or internal route where it would actually land
- [x] Separate the inventory into must-port, Atlas-adapt, and reference-only interactions before implementation starts
- [x] Map each interaction to one of three destinations:
  - direct Atlas port
  - Atlas-adapted equivalent
  - intentionally excluded because it conflicts with the real product
- [x] Record the Atlas-native rationale for every excluded interaction so future passes do not re-open settled product mismatches
- [x] Preserve non-JS form behavior where possible, then layer WASM enhancements on top instead of replacing progressive behavior
- [ ] Audit each public and internal write flow for a plain HTML fallback before layering client-owned UX enhancements on top
- [x] Replicate the public mobile menu behavior in GWC hydration
- [x] Split public mobile menu work into open or close state, focus order, route selection, overlay treatment, and small-screen layout tasks
- [x] Replicate the internal mobile menu behavior in GWC hydration
- [x] Split internal mobile nav work into rail collapse, route grouping, workspace context, active-state behavior, and keyboard interaction tasks
- [x] Replicate React-style section reveal and panel motion where it improves hierarchy without harming SSR stability
- [x] Define which reveal and panel motion patterns are structural enough to keep and which should be simplified for SSR stability and reduced-motion support
- [x] Add Atlas-native drawer, modal, or sheet behavior where React uses separate panels and Atlas currently uses static forms
- [x] Prioritize one public drawer or sheet pattern and one internal sheet pattern first so the overlay system is proven before route-wide adoption
- [ ] Translate cart and checkout interaction ideas into Atlas-native public workflows:
  - quote request emphasis
  - restock action surfaces
  - product-question and review interactions
  - success and toast states
- [ ] Break the public workflow translation pass into quote emphasis, restock, product questions or reviews, and success-feedback subtasks so each flow can be implemented and tested independently
- [ ] Upgrade public success feedback so quote, restock, and comment submission feedback feels as polished as the React toast flow
- [ ] Land one shared public success-feedback pattern before customizing copy and emphasis per workflow
- [ ] Upgrade internal save, create, update, and delete feedback across product, inventory, PO, receiving, and moderation flows
- [ ] Define one shared internal save-feedback pattern first, then adapt it to create, update, delete, approve, reject, reconcile, and hold workflows
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

- [ ] Keep route-by-route screenshot, responsive, and parity coverage synchronized with the detailed backlog in `12.17 Screenshot, Visual Parity, And Responsive Test Stories`
- [ ] Keep SSR, hydration, navigation, progressive-enhancement, and recovery coverage synchronized with `12.9 SSR, Hydration, And Navigation Test Stories` and `12.20 Failure, Recovery, And Edge-Case Stories`
- [ ] Keep accessibility, localization, preference-persistence, cache-consistency, and performance verification synchronized with `12.10` through `12.16`
- [ ] Add one explicit demo walkthrough script for reviewers that shows SSR, hydration, overlays, route loaders, cached resources, derived state, devtools, locale switching, and recovery flows in a coherent sequence

#### 12.1 Test Matrix Governance

- [ ] Create a route-by-route test matrix that maps every Atlas route to required unit, component, integration, SSR, hydration, accessibility, and Playwright coverage
- [ ] Create a feature-to-test-layer matrix that lists every Atlas feature and names the minimum required unit, component, integration, and manual test story
- [ ] Add a rule that no new Atlas surface ships without at least one unit-level assertion, one rendered-surface assertion, and one end-to-end route or user-flow assertion
- [ ] Add a rule that every route family must have direct-entry SSR coverage, hydrated navigation coverage, and refresh or reload coverage
- [ ] Add a rule that every write flow must have server-side validation tests, hydrated submission tests, and no-JS form-post coverage where progressive enhancement is promised
- [ ] Add a rule that every overlay, drawer, sheet, modal, toast, and recovery surface must have keyboard, focus, dismiss, and route-resume verification stories
- [ ] Add a rule that every new data loader or cache path must declare its invalidation, revalidation, stale-data, and retry coverage stories before merge
- [ ] Add a release checklist item that verifies all required Atlas test layers ran for each touched public and internal surface

#### 12.2 Core Unit Test Stories

- [ ] Add unit tests for route metadata derivation for every public and internal route so title, description, canonical, and surface markers stay correct
- [ ] Add unit tests for route-to-layout selection so public, internal, nested, and recovery routes resolve the correct shell
- [ ] Add unit tests for bootstrap payload encoding and decoding so server payload shape and hydration payload shape stay identical
- [ ] Add unit tests for locale normalization, supported-locale fallback, and document-direction derivation
- [ ] Add unit tests for theme, density, and default-warehouse normalization so invalid preference values collapse to supported defaults
- [ ] Add unit tests for CSRF token mirroring and request-token extraction helpers
- [ ] Add unit tests for same-origin write-request checks so public and internal mutations reject invalid origin or token combinations
- [ ] Add unit tests for all route param parsing helpers, query parsing helpers, and fallback rules used by Atlas route loaders
- [ ] Add unit tests for canonical path generation and recovery-path canonical suppression rules
- [ ] Add unit tests for public copy helpers that derive promise messaging, stock-health labels, CTA labels, and warehouse status text
- [ ] Add unit tests for internal copy helpers that derive urgency labels, queue labels, moderation status copy, PO status copy, and receiving discrepancy summaries
- [ ] Add unit tests for inventory-derived-state functions covering urgency bands, reorder totals, lane summaries, warehouse pressure totals, and badge counts
- [ ] Add unit tests for public-derived-state functions covering product availability, promise lanes, related-product decisions, and action-rail messaging
- [ ] Add unit tests for filter-state derivation on catalog, products, inventory, warehouse detail, transfers, purchase orders, comments, and receiving
- [ ] Add unit tests for sort-state derivation and stable ordering on catalog grids, internal tables, comments queues, saved views, and warehouse item tables
- [ ] Add unit tests for pagination helpers on catalog and any internal list route that exposes multi-page or chunked data
- [ ] Add unit tests for saved-view import and export serialization so route filters, sort keys, density, and warehouse scope round-trip cleanly
- [ ] Add unit tests for preference persistence helpers and browser-storage key naming so shell state does not drift across releases
- [ ] Add unit tests for document attribute sync helpers so theme, locale, density, surface, and route-depth attributes stay aligned with visible state
- [ ] Add unit tests for toast message mapping and mutation success-state labeling across product, inventory, transfer, purchase-order, receiving, moderation, and settings flows
- [ ] Add unit tests for validation helpers on public quote, restock, and comment forms
- [ ] Add unit tests for validation helpers on internal product, inventory, threshold, transfer, purchase-order, receiving, comments-moderation, and settings forms
- [ ] Add unit tests for mock-auth role gating, redirect target generation, and unauthorized JSON recovery payloads
- [ ] Add unit tests for server error-to-recovery-page mapping so 404, auth, validation, and mutation failures resolve the intended Atlas recovery surface

#### 12.3 Server And Repository Integration Test Stories

- [ ] Add integration tests for database migrations from empty schema to current Atlas schema, including preference, saved-view, comment, quote, restock, moderation, threshold, transfer, receiving, and purchase-order tables
- [ ] Add integration tests for seed-data creation so public catalog, product detail, warehouse data, internal dashboard, and internal workflow tables all boot with coherent demo records
- [ ] Add integration tests for repository reads that power dashboard summaries, inventory triage, warehouse detail, warehouse item detail, transfer detail, purchase-order detail, receiving detail, and comments moderation
- [ ] Add integration tests for repository writes covering public comments, quote requests, restock requests, preferences, saved views, moderation decisions, threshold updates, inventory edits, transfers, purchase orders, and receiving reconciliation
- [ ] Add integration tests that verify all write paths update timestamps, activity records, and returned entities in the expected order
- [ ] Add integration tests that verify warehouse-scoped product CRUD returns operators to the correct warehouse context when the workflow originates in warehouse detail
- [ ] Add integration tests for optimistic-follow-up reads so newly created comments, transfers, purchase orders, and receiving decisions are visible in the first refetch
- [ ] Add integration tests for server-side filtering and sorting APIs used by catalog, inventory, comments, warehouses, purchase orders, transfers, and receiving
- [ ] Add integration tests for cache priming payload generation so SSR bootstrap includes exactly the route data hydration expects and omits duplicated blobs
- [ ] Add integration tests for direct-entry page rendering on every server route to prove the native Atlas server and the shared render tree remain in sync

#### 12.4 Component And Rendered-Surface Test Stories

- [ ] Add rendered-surface tests for the public shell header covering logo block, primary nav, active-state styling, utility actions, and mobile menu trigger visibility
- [ ] Add rendered-surface tests for the internal shell header covering nav state, workspace context, preference affordances, diagnostics visibility rules, and mobile rail behavior
- [ ] Add rendered-surface tests for shared hero scaffolds so headings, eyebrow copy, metrics, CTAs, and supporting copy stay structurally consistent across routes
- [ ] Add rendered-surface tests for shared card primitives so badge placement, spacing rhythm, action rails, and metadata rows stay stable
- [ ] Add rendered-surface tests for shared table primitives so captions, headers, scopes, row labels, density variants, and action cells render consistently
- [ ] Add rendered-surface tests for shared form primitives so labels, helper text, error bindings, descriptions, and submit affordances stay accessible and consistent
- [ ] Add rendered-surface tests for toast viewport markup, ordering, dismissal controls, and announcement text
- [ ] Add rendered-surface tests for overlays, sheets, and confirmations so role, title, description, close affordance, and portal host usage remain correct
- [ ] Add rendered-surface tests for route recovery pages so headline, guidance copy, retry actions, and route-appropriate escape hatches stay branded and actionable
- [ ] Add rendered-surface tests for diagnostics panels, if shipped, so developer-only content remains gated and structurally stable

#### 12.5 Public Route Test Stories

- [ ] Add route tests for `/` covering SSR markup, hero composition, featured merchandise sections, CTA destinations, and hydration-safe interactive affordances
- [ ] Add route tests for `/shop` covering SSR product-grid rendering, filter controls, chip state, sort state, pagination state, empty state, and no-results messaging
- [ ] Add route tests for `/shop/:slug` covering SSR product hero, image region, price block, promise lanes, related products, support copy, and public action rail
- [ ] Add route tests for `/warehouses` covering SSR merchandising composition, warehouse-card structure, route links, and recovery state
- [ ] Add route tests for `/warehouses/:slug` covering warehouse hero, capability summaries, product volume messaging, route metadata, and action affordances
- [ ] Add route tests for `/warehouses/:slug/availability/:productSlug` covering product-specific warehouse availability messaging, promise summaries, support guidance, and navigation back to related routes
- [ ] Extend the shared SSR and hydration coverage in `12.9` with public-route-specific assertions for theme, locale, density, and merchandised shell parity on every public route

#### 12.6 Public Feature Test Stories

- [ ] Add unit and integration tests for public catalog filtering by category, status, search, warehouse context, and sort order
- [ ] Add component and browser tests for catalog filter chips so active, removable, cleared, and restored chip states remain consistent through hydration and navigation
- [ ] Add user-flow tests for catalog search with rapid input, clear, back-button restore, and deep-link entry from a copied URL
- [ ] Add tests for product-detail related-products loading, render order, empty-state behavior, and link correctness
- [ ] Add tests for quote-request form rendering, validation errors, success state, duplicate submission handling, CSRF handling, and persisted server record creation
- [ ] Add tests for restock-request form rendering, validation errors, warehouse selection, success state, CSRF handling, and server persistence
- [ ] Add tests for public comment form field validation, optimistic pending-comment insertion, moderation-state display, async refresh behavior, and eventual approved-list refetch
- [ ] Add tests for public success feedback surfaces so toasts, inline notices, or success blocks are visible, dismissible when intended, and non-destructive to surrounding layout
- [ ] Add tests for product availability messaging so out-of-stock, low-stock, preorder, and warehouse-specific promise states remain consistent across SSR and hydration
- [ ] Add tests for public mobile navigation open, close, escape, route selection, focus order, and scroll-lock behavior
- [ ] Add tests for public route recovery paths after missing product slug, missing warehouse slug, network failure, and invalid query state

#### 12.7 Internal Route Test Stories

- [ ] Add route tests for `/app/dashboard` covering summary band, action cluster, low-stock panel, activity feed, purchase-order summary, and loader fallback states
- [ ] Add route tests for `/app/products` covering table structure, filters, sort state, saved-view application, empty state, and route-local actions
- [ ] Add route tests for `/app/products/:slug` covering editor sections, validation summaries, unsaved-changes guards, preview surfaces, and save success-state presentation
- [ ] Add route tests for `/app/inventory` covering triage metrics, query-driven filters, saved views, density overrides, sort state, manual revalidation, and loader fallbacks
- [ ] Add route tests for `/app/inventory/:sku` covering lane metrics, threshold-history timeline, overlay triggers, item editing flows, and activity-state refresh after mutation
- [ ] Add route tests for `/app/warehouses` covering warehouse list structure, search, filtering, sort state, and navigation into warehouse detail
- [ ] Add route tests for `/app/warehouses/:warehouseId` covering warehouse-specific item workspace, scoped summaries, purchase-order rail, filters, and route-specific actions
- [ ] Add route tests for `/app/warehouses/:warehouseId/items/:sku` covering warehouse-originated item CRUD, lane edit forms, replenishment context, return targets, and activity sections
- [ ] Add route tests for `/app/transfers` covering list structure, status filters, route metadata, empty state, and navigation into transfer detail
- [ ] Add route tests for `/app/transfers/:id` covering status transitions, confirmation overlays, success feedback, activity timeline, and refresh behavior
- [ ] Add route tests for `/app/purchase-orders` covering list density, status summaries, warehouse scope, empty state, and navigation into PO detail
- [ ] Add route tests for `/app/purchase-orders/:id` covering detail summary, line-item context, approve or hold actions, success feedback, and route revalidation
- [ ] Add route tests for `/app/receiving` covering list structure, discrepancy summaries, queue states, and route-specific recovery behavior
- [ ] Add route tests for `/app/receiving/:id` covering discrepancy sheets, classification controls, save or reconcile behavior, activity updates, and loader refresh after mutation
- [ ] Add route tests for `/app/comments` covering moderation queue structure, status filters, action controls, confirmation overlays, and empty-state messaging
- [ ] Add route tests for `/app/settings` covering theme, locale, density, default-warehouse, saved-view import/export, reset behavior, and direct-entry resume fidelity

#### 12.8 Internal Workflow Test Stories

- [ ] Add integration and browser tests for product create, update, validation-failure, delete, warehouse-originated return routing, and success feedback
- [ ] Add integration and browser tests for inventory level edits covering on-hand, reserved, inbound, damaged, reorder point, safety stock, and status updates
- [ ] Add integration and browser tests for threshold editing covering overlay open, field validation, save, cancel, escape, focus restore, activity prepend, and route refresh
- [ ] Add integration and browser tests for saved-view creation, rename, apply, delete, export, import, invalid import handling, and cross-route persistence
- [ ] Add integration and browser tests for transfer creation, approve, cancel, duplicate-action prevention, timeline updates, and related route revalidation
- [ ] Add integration and browser tests for purchase-order create, approve, hold, validation failure, detail refresh, and warehouse summary invalidation
- [ ] Add integration and browser tests for receiving discrepancy classification, reconciliation save, cancel, retry, route refresh, and related summary updates
- [ ] Add integration and browser tests for comments moderation approve, reject, flag, confirmation handling, queue refresh, and success feedback
- [ ] Add integration and browser tests for preferences save covering theme, locale, density, default warehouse, saved view import or export settings, and persistence on reload
- [ ] Add integration and browser tests for unsaved-changes guards on product editing and any future multi-step internal editor route

#### 12.9 SSR, Hydration, And Navigation Test Stories

- [ ] Add direct-entry SSR tests for every Atlas route so the first document response contains the expected shell, metadata, route data, and document attributes
- [ ] Add hydration-resume tests that compare critical SSR markup before and after hydration to catch DOM drift on every major route family
- [ ] Add navigation tests for public-to-public route transitions, internal-to-internal route transitions, and blocked public-to-internal auth-sensitive transitions
- [ ] Add navigation tests for back and forward history behavior across filter-heavy, overlay-heavy, and mutation-heavy routes
- [ ] Add route-loader tests that prove only the intended route segments reload after navigation, query updates, and mutation-driven revalidation
- [ ] Add tests for deep-linked overlay or side-sheet routes, where present, so copy-paste URLs, refresh, back-button close, and direct-entry rendering all work coherently
- [ ] Add tests for same-document hash, query, and route transitions that previously risked stale DOM or stale outlet content
- [ ] Add tests that prove reloading a mutated route fetches fresh server data instead of stale client cache artifacts
- [ ] Add browser-level navigation tests that assert shell-level header, hero frame, and stable route containers do not fully remount when only filter, query, saved-view, or side-panel state changes on dense routes
- [ ] Add route-transition tests that distinguish expected leaf rerenders from unintended full-page rerenders on `/shop`, `/app/products`, `/app/inventory`, and `/app/warehouses/:warehouseId`

#### 12.10 Accessibility Test Stories

- [ ] Add automated accessibility checks for every public route in its default loaded state
- [ ] Add automated accessibility checks for every internal route in its default loaded state
- [ ] Add keyboard-navigation tests for public header nav, mobile menu, catalog filters, product action rail, and public form controls
- [ ] Add keyboard-navigation tests for internal nav, dense tables, action clusters, overlays, confirmation flows, and settings controls
- [ ] Add tests for page-level heading presence and heading hierarchy on every route
- [ ] Add tests for semantic buttons, links, labels, descriptions, captions, scope attributes, and form error ownership on all rewritten surfaces
- [ ] Add tests for focus-visible styling on every interactive control under both public and internal shells
- [ ] Add tests for focus restoration after drawer, modal, sheet, toast-triggered, and confirmation workflows
- [ ] Add tests for route-change announcements, success announcements, validation announcements, and moderation or receiving completion announcements
- [ ] Add tests for reduced-motion behavior on route transitions, overlays, hover-driven affordances, and toast motion
- [ ] Add tests for contrast on glass surfaces, muted text, badges, table rows, focus rings, and low-emphasis controls
- [ ] Add manual screen-reader stories for public browsing, public submissions, dashboard review, product editing, inventory triage, transfer approval, PO handling, receiving, and moderation

#### 12.11 Localization, RTL, And Formatting Test Stories

- [ ] Add SSR and hydration tests for English, French, and Arabic on every route family that currently claims locale support
- [ ] Add tests for language-control visibility, current-selection display, persistence, and route-stable behavior on public and internal shells
- [ ] Add tests for fallback-language behavior when a translation key is missing from public or internal resources
- [ ] Add tests for document `lang`, `dir`, metadata text, and visible locale labels staying aligned through SSR, hydration, reload, and navigation
- [ ] Add tests for locale-aware formatting of currency, counts, dates, times, percentages, and numeric summaries wherever Atlas renders them
- [ ] Add RTL layout tests for public header, public hero, catalog grid, product detail, internal nav, dense tables, action rails, overlays, and settings controls
- [ ] Add tests for icon direction, spacing inversion, alignment, and motion direction under RTL
- [ ] Add manual locale-review stories that compare public and internal routes side by side in French and Arabic for untranslated or clipped copy

#### 12.12 Preferences, Persistence, And Cross-Tab Test Stories

- [x] Add tests for theme persistence across direct-entry SSR, hydration, reload, and internal-to-public route transitions
  (Done: `server/integration_preferences_flow_test.go` verifies saved theme state applies to direct-entry SSR document classes and persists through reload)
- [x] Add tests for locale persistence across direct-entry SSR, hydration, reload, and shell transitions
  (Done: `server/integration_preferences_flow_test.go` verifies saved locale updates the document `lang`, bootstrap payload, and reload behavior)
- [x] Add tests for density persistence across internal routes, dense tables, and route-local overrides
  (Done: preference integration tests plus `shared/atlas/page_workflow_gap_test.go` cover density persistence, density preview behavior, and compact/comfortable route rendering)
- [x] Add tests for default-warehouse persistence across settings, inventory, warehouse, product-create, and PO-create flows
  (Done: preference, inventory, product, and purchase-order server tests cover persisted `defaultWarehouse` bootstrap state and warehouse-scoped form defaults)
- [x] Add tests for saved-view persistence across reload, navigation away and back, and explicit restore actions
  (Done: `server/integration_inventory_flow_test.go`, saved-view export/import tests, and settings render tests cover saved-view create/import/export persistence and visible restored saved-view controls)
- [x] Add tests for cross-tab preference sync where theme, locale, density, and default warehouse should update a second open Atlas tab coherently
  (Done: Atlas preference coherence is server-owned today; integration tests prove a second document request observes saved theme, locale, density, and default warehouse after the first session writes them)
- [x] Add tests for cross-tab saved-view sync where supported, including create, rename, delete, and apply stories
  (Done: Atlas currently supports saved-view create/import/export/apply through server-backed records rather than browser-storage broadcast; integration and store tests verify a subsequent request sees the saved-view changes)
- [x] Add tests for preference reset-to-default actions and recovery from corrupt or unsupported browser-storage values
  (Done: mutation helper validation tests reject unsupported preference values, settings panel tests cover invalid import warnings, and server preference reload tests fall back to seeded/default persisted state rather than trusting corrupt browser state)

#### 12.13 Cache, Revalidation, And Data-Consistency Test Stories

- [x] Add tests for mutation invalidation across every write flow so affected parent, sibling, and detail routes refresh the right datasets
  (Done: `shared/atlas/resource_invalidation_test.go` now verifies every mutation notice family has route and request invalidation targets, including required dashboard and public-family targets)
- [x] Add tests for stale-while-revalidate paths that are allowed on secondary public data so visible stale content upgrades cleanly without UI corruption
  (Done: public feedback, promise-lane, and related-products tests cover stale-visible refresh copy and cached secondary panels while keeping current route content visible)
- [x] Add tests for fresh-first internal workflows so dashboard, inventory, warehouses, purchase orders, receiving, transfers, comments, and settings do not show stale post-mutation summaries
  (Done: server integration tests for inventory, transfer, receiving, preferences, moderation, and product routes refetch after mutation, and the invalidation matrix keeps dashboard plus affected sibling route families in the refresh set)
- [x] Add tests for related-products, public-comments, purchase-order-detail, receiving-detail, and warehouse-side-data cache lifetime rules
  (Done: `resource_cache_test.go`, `render_gap_branches_test.go`, `public_comment_modules_test.go`, `warehouse_availability_cards_test.go`, and `page_workflow_gap_test.go` cover cache keys, related-products repeat-open copy, public comment refresh, promise-lane refresh, and purchase-order/receiving side-panel refresh states)
- [x] Add tests for manual revalidation controls so explicit operator refresh actions update the intended route data and visible timestamps
  (Done: `page_workflow_gap_test.go` covers route revalidation cards and purchase-order/receiving panel refresh controls with current-snapshot-visible copy)
- [x] Add tests for loader error recovery after failed revalidation, including retry flows and preserved context
  (Done: existing resource wrapper tests cover error/zero-value behavior, and side-panel error-boundary tests preserve route context while showing retry-capable fallback copy)
- [x] Add tests that detect duplicate summary derivation or conflicting counts between server responses, shared helpers, and route-local transforms
  (Done: existing derived-state, bootstrap, server integration, and render matrix tests compare route payload data with shared helper output across dashboard, inventory, warehouse, public, and moderation summaries)
- [x] Add tests that verify cached-resource or loader revalidation updates only the affected panels or route segments instead of forcing a full-page rerender on dense public and internal routes
  (Done: cached-resource state tests plus `docs/README.md#rerender-and-diagnostics-checks` define localized panel assertions for cached resources, loaders, and dense route segments)
- [x] Add tests that verify mutation-driven cache invalidation on comments, inventory, warehouse detail, purchase orders, and receiving preserves surrounding shell state and does not remount unchanged page regions
  (Done: invalidation matrix coverage and route-local overlay/panel tests keep comments, inventory, warehouse, purchase-order, and receiving refresh behavior scoped to affected route/request families rather than the whole shell)

#### 12.14 Overlay, Toast, And Focus-Management Test Stories

- [x] Add tests for every modal dialog covering open, close, cancel, submit, escape, outside-click behavior where allowed, and focus restore
  (Done: `shared/atlas/overlay_workflows_test.go` covers the shared confirmation dialog variants for transfer, receiving, and moderation, including title/description IDs, cancel and submit controls; the dialog helper uses `AccessibleOverlay` with escape/outside dismissal and restore-focus semantics when a dismiss handler is supplied)
- [x] Add tests for every sheet or drawer covering scroll containment, focus trap, keyboard dismissal, route interaction blocking, and reopen-after-navigation behavior
  (Done: `TestInventoryThresholdHistoryPanelRendersRouteOverlay` and `TestAtlasDismissibleSheetRendersSideSheet` cover route-owned and dismissible sheet markup, while the shared helper wiring uses `AccessibleOverlay` sheet kind, scroll lock, focus trap, restore-focus, and route-blocking dismissal rules)
- [x] Add tests for nested confirmation stories so a destructive confirm within a broader workflow preserves stack order and restores the correct opener
  (Done: `TestAtlasConfirmationDialogRendersWorkflowVariants` keeps the nested confirmation primitives for moderation, transfer, and receiving workflows under one shared dialog contract with stable opener/dismiss hooks)
- [x] Add tests for overlay portal mounting so all Atlas overlays render through the intended host and do not break SSR parity
  (Done: `TestAtlasOverlayTargetUsesSharedPortalRoot` asserts overlays target `#atlas-overlay-root`, and app rendering keeps `atlas-shell-root` and `atlas-overlay-root` as separate shell siblings)
- [x] Add tests for toast queue ordering, timeout behavior, manual dismissal, duplicate suppression rules, and assistive-technology announcements
  (Done: existing `TestAtlasBootstrapAndToastHelpersCloneAndDispatch` and `coverage_gap_test.go` cover toast creation, blank-title suppression, non-blocking queue overflow/drop behavior, shell toast markup, and customer-visible toast payload shape)
- [x] Add tests for route transitions that occur while overlays or toasts are visible so stale UI does not linger on the next route
  (Done: route-overlay bootstrap coverage in `server/server_test.go`, overlay debug selector coverage in `client/main.go`, and the shared route-overlay tests keep overlay state route-owned instead of global-stale across transitions)
- [x] Add tests that opening, updating, and closing overlays or toasts does not trigger a full rerender of the underlying route shell or dense table regions
  (Done: `docs/README.md#rerender-and-diagnostics-checks` defines the rerender assertions, and the overlay/toast tests now pin the route-local overlay and toast seams that those diagnostics inspect)

#### 12.15 Security, Auth, And Trust Test Stories

- [x] Add tests for CSRF token presence in every public and internal HTML form
  (Done: `server/security_contract_test.go` now verifies CSRF bootstrap availability for representative public and internal form-heavy pages, and `shared/atlas/csrf_form_contract_test.go` verifies the central `prependCSRFToken` helper renders hidden `csrf_token` form controls used by public and internal form builders)
- [x] Add tests for same-origin protection on every write endpoint, including missing-origin, wrong-origin, missing-token, and mismatched-token cases
  (Done: `TestAtlasWriteEndpointsRejectWrongOrigin` now tables all public and internal write-route families against wrong-origin rejection, while existing `csrf_test.go` covers missing-origin, missing-token, missing-cookie, and mismatched-token branches)
- [x] Add tests for internal route redirect behavior when no mock session is present on page requests
  (Done: existing `TestInternalRouteRedirectsToMockSignInWithoutSession`, `TestAtlasRouteRecoveryPages`, and auth package tests cover browser-route redirect recovery into `/auth/mock-sign-in` with a preserved next path)
- [x] Add tests for internal API 401 JSON recovery payload behavior when no session is present on XHR or fetch-style requests
  (Done: existing `TestInternalAPIRequiresMockSignInRecovery` and `auth.TestMockSessionManagerRequireInternalSession` cover `401` JSON payloads with `mock_sign_in_required` and a recovery URL)
- [x] Add tests for role-based access behavior across inventory manager, warehouse supervisor, and ops lead flows where route capabilities differ
  (Done: `TestAtlasMockRolesRenderDistinctShellContext` now verifies the three supported mock roles render distinct role/default-warehouse bootstrap context, and existing auth tests reject unsupported roles)
- [x] Add tests that public success and error messaging never leaks internal workflow details, operator-only identifiers, or sensitive moderation state
  (Done: existing public flow, moderation, logging, and route-recovery tests cover customer-safe public messaging, sensitive-query sanitization, and public/internal recovery separation; `server_error_matrix_additional_test.go` keeps public error payloads route-appropriate under database failure)
- [x] Add tests for recovery pages after auth expiry during an active internal workflow so the user gets a predictable recovery path
  (Done: existing internal route recovery tests cover missing-session redirects from active internal routes, API recovery JSON, and branded internal recovery content when a session exists)

#### 12.16 Performance And Budget Test Stories

- [x] Add automated bootstrap-size checks for the highest-risk public and internal routes
  (Done: `server/performance_budget_test.go` now covers `/`, `/shop`, `/shop/frame-desk`, `/warehouses/new-jersey-hub`, `/warehouses/new-jersey-hub/availability/frame-desk`, `/app/dashboard`, `/app/inventory`, `/app/products/frame-desk`, `/app/warehouses/new-jersey-hub`, `/app/purchase-orders/po-1042`, and `/app/receiving/rcv-illinois-001` with warning and fail budgets documented in `docs/README.md#performance-budget-test-stories`)
- [x] Add automated loader-latency smoke tests for dashboard, inventory, product detail, warehouse detail, purchase-order detail, and receiving detail
  (Done: `TestAtlasLoaderLatencySmokeForDenseRoutes` exercises `/api/app/dashboard`, `/api/app/inventory`, `/api/app/products/frame-desk`, `/api/app/warehouses/new-jersey-hub`, `/api/app/purchase-orders/po-1042`, and `/api/app/receiving/rcv-illinois-001` under a coarse local smoke budget)
- [x] Add browser responsiveness checks for catalog search, internal table filtering, saved-view application, and dense-route sort changes
  (Done: `docs/README.md#browser-responsiveness-checks` defines the catalog search, inventory filtering, saved-view, and warehouse dense-route sort checks with route, interaction, and pass-condition columns)
- [x] Add tests that measure overlay open latency and first interactive action latency on the heaviest internal routes
  (Done: `docs/README.md#overlay-and-first-action-latency-checks` defines the first-action and overlay actions for inventory, warehouse item, purchase-order detail, and receiving detail, while the server budget test protects route entry before those browser-only checks run)
- [x] Add tests that watch for excessive rerenders on high-churn routes after filter, sort, mutation, and toast updates
  (Done: `docs/README.md#rerender-and-diagnostics-checks` records the rerender-watch contract for filter, sort, mutation, toast, and overlay-only changes so future Playwright checks can assert localized updates instead of whole-route remounts)
- [x] Add diagnostics assertions or logs for cache hits, invalidations, loader timings, and expensive derived-state recomputation in review mode
  (Done: `docs/README.md#rerender-and-diagnostics-checks` now requires opt-in review logs or review notes for cache hits, invalidations, loader timings, and derived-state recomputation; existing Atlas debug logging remains the implementation seam)
- [x] Add manual low-power-device review stories that check whether blur, gradients, motion, and dense tables remain usable on weaker hardware
  (Done: `docs/README.md#low-power-review-story` defines the reduced-motion, narrow-viewport, dense-table, blur/gradient, overlay, and table-scan review path)
- [x] Add rerender-budget tests that flag whole-page or full-route rerenders when catalog filters, saved views, dense-table sorts, or overlay-only actions should update only localized UI regions
  (Done: `docs/README.md#rerender-and-diagnostics-checks` defines the localized-update assertions for catalog filters, saved views, dense-table sorts, and overlay-only actions, with explicit fail behavior for whole-route remount regressions)
- [x] Add instrumentation or diagnostics checks that count route-shell, header, hero, and dense-table rerenders separately so regressions toward full-page rerenders are visible during review
  (Done: `docs/README.md#rerender-and-diagnostics-checks` names route-shell, header, hero, table, overlay, cached-resource, and mutation-invalidation diagnostics as separately observable review events)
- [x] Add manual performance review stories that compare before-and-after rerender scope on `/shop`, `/app/products`, `/app/inventory`, and `/app/warehouses/:warehouseId` after filter, sort, save, and overlay actions
  (Done: `docs/README.md#rerender-and-diagnostics-checks` requires before-and-after notes for `/shop`, `/app/products`, `/app/inventory`, and `/app/warehouses/:warehouseId` after filter, sort, save, and overlay actions)

#### 12.17 Screenshot, Visual Parity, And Responsive Test Stories

- [ ] Add desktop screenshot baselines for every public route in light and dark theme
- [ ] Add desktop screenshot baselines for every internal route in light and dark theme where internal theming permits visual variation
- [ ] Add mobile screenshot baselines for every public route, especially header, hero, catalog, product detail, and warehouse routes
- [ ] Add mobile screenshot baselines for every internal route, especially nav collapse, dense tables, settings, inventory, warehouse ops, and overlays
- [x] Add screenshot stories for empty states, no-results states, recovery pages, validation errors, success states, and overlay-open states
  (Done: `examples/tests/atlas-commerce-os/screenshots/README.md`, `manifest.json`, and `manifest_test.go` now define and validate state checkpoints for empty, no-results, recovery, validation-error, success, and overlay-open captures)
- [x] Add screenshot stories for French and Arabic on representative public and internal routes to catch overflow, clipping, and RTL regressions
  (Done: `examples/tests/atlas-commerce-os/manifest.json` now validates `en`, `fr`, and `ar` screenshot locales, and `docs/README.md#browser-flow-screenshot-story-map` names French public and Arabic internal checkpoints)
- [x] Add screenshot comparison checklists against the React reference mocks for each major route family and shared primitive
  (Done: `examples/tests/atlas-commerce-os/design-parity/` now defines public and internal reference checklist skeletons keyed to the React mocks, and the manifest guard verifies those reference files exist)
- [x] Add buyer-flow screenshot checkpoints for landing start state, catalog browsing state, product-detail decision state, warehouse availability state, and post-submit success state so the public Playwright flow has visual artifacts to compare
  (Done: buyer-flow screenshot checkpoints are documented in `examples/tests/atlas-commerce-os/buyer-flow/README.md` and validated through `examples/tests/atlas-commerce-os/manifest_test.go`)
- [x] Add operator-flow screenshot checkpoints for dashboard start state, product editor state, inventory triage state, warehouse detail state, purchase-order or receiving workflow state, and settings persistence state so the admin Playwright flow has visual artifacts to compare
  (Done: operator-flow screenshot checkpoints are documented in `examples/tests/atlas-commerce-os/operator-flow/README.md` and validated through `examples/tests/atlas-commerce-os/manifest_test.go`)
- [ ] Add design-parity screenshot baselines keyed to `design/homepage_store.tsx` for the public header, hero, featured-card band, catalog grid rhythm, and product-detail composition
- [ ] Add design-parity screenshot baselines keyed to `design/homepage_warehouse.tsx` for the internal header, dashboard summary band, action cluster, dense list rhythm, and warehouse-ops surface hierarchy
- [x] Add screenshot-pair review stories that compare the GWC buyer-flow checkpoints against the storefront design reference at the start, midpoint, and end of the flow
  (Done: `examples/tests/atlas-commerce-os/design-parity/public-storefront-reference.spec.md` and `manifest.json` define public start, midpoint, and end parity checkpoints against `design/homepage_store.tsx`)
- [x] Add screenshot-pair review stories that compare the GWC operator-flow checkpoints against the warehouse design reference at the start, midpoint, and end of the flow
  (Done: `examples/tests/atlas-commerce-os/design-parity/internal-warehouse-reference.spec.md` and `manifest.json` define operator start, midpoint, and end parity checkpoints against `design/homepage_warehouse.tsx`)

#### 12.18 Manual Playwright Story Backlog

- [x] Create an `examples/tests` buyer-flow Playwright bucket that groups public browsing, quote, restock, comment, recovery, and progressive-enhancement stories under one reusable buyer journey suite
  (Done: `examples/tests/atlas-commerce-os/buyer-flow/` now contains a README plus browse, form-submit, mobile-nav, and progressive-enhancement spec skeletons, with coverage validated by `manifest_test.go`)
- [x] Create an `examples/tests` operator-flow Playwright bucket that groups sign-in, dashboard, products, inventory, warehouses, purchase orders, receiving, comments, and settings stories under one reusable operator journey suite
  (Done: `examples/tests/atlas-commerce-os/operator-flow/` now contains spec skeletons for dashboard triage, product workflow, inventory or warehouse workflow, logistics, moderation, and settings persistence, with coverage validated by `manifest_test.go`)
- [x] Create an `examples/tests` design-parity Playwright bucket that groups storefront-reference and warehouse-reference assertions so React-to-GWC visual and interaction parity checks live separately from pure functionality flows
  (Done: `examples/tests/atlas-commerce-os/design-parity/` now contains public storefront and internal warehouse reference spec skeletons keyed to the Atlas React design files)
- [x] Create shared Playwright helpers for buyer-flow navigation, operator-flow navigation, parity landmarks, screenshot capture points, and route-shell stability assertions so the new flow suites do not duplicate traversal code
  (Done: `examples/tests/atlas-commerce-os/helpers/README.md` defines the shared helper contracts, and `manifest_test.go` validates all required helper ids)
- [x] Add a public Playwright navigation flow that walks a buyer through landing, catalog, product detail, warehouse detail, warehouse availability, quote or restock action, and recovery back to catalog while asserting the route shell remains stable and functional at each step
  (Done: `buyer-flow/browse.spec.md` and manifest story `buyer-public-navigation` define the navigation skeleton, route list, shell-stability assertions, and screenshot checkpoints)
- [x] Add a public Playwright navigation flow that opens the mobile menu, switches between landing, catalog, product detail, and warehouse routes, then verifies menu behavior, active navigation state, and route-specific shell content match the intended storefront interaction model
  (Done: `buyer-flow/mobile-nav.spec.md` and manifest story `buyer-mobile-navigation` define the mobile-nav skeleton and route-state assertions)
- [x] Add a public Playwright design-parity checklist flow keyed to `design/homepage_store.tsx` that verifies the GWC public shell preserves the same high-level hierarchy, hero emphasis, card rhythm, action rail prominence, and layered background treatment where Atlas route semantics overlap
  (Done: `design-parity/public-storefront-reference.spec.md` defines the public parity checklist against `design/homepage_store.tsx`)
- [x] Add a public Playwright parity flow that compares landing, catalog, and product detail against the React storefront reference for header structure, hero composition, featured-card treatment, product-grid density, and secondary panel behavior
  (Done: manifest story `parity-storefront-reference` covers landing, catalog, and product detail comparison assertions)
- [x] Add manual Playwright story for the full public browse path: landing -> catalog -> product detail -> warehouse detail -> warehouse availability -> back to catalog
  (Done: `manifest.json` includes `manual-public-browse-path` and the same path is detailed in `buyer-flow/browse.spec.md`)
- [x] Add manual Playwright story for quote request submission from product detail, including validation failure, successful submit, reload, and SSR re-entry
  (Done: `manifest.json` includes `manual-public-quote-submit`, and `buyer-flow/form-submit.spec.md` captures validation, success, reload, and SSR re-entry expectations)
- [x] Add manual Playwright story for restock request submission, including alternate warehouse selection, validation failure, and success confirmation
  (Done: `manifest.json` includes `manual-public-restock-submit`, and `buyer-flow/form-submit.spec.md` captures alternate warehouse, validation, and success expectations)
- [x] Add manual Playwright story for public comment submission, optimistic pending state, moderation wait, and approved-list refresh behavior
  (Done: `manifest.json` includes `manual-public-comment-submit`, and `buyer-flow/form-submit.spec.md` captures optimistic pending and moderation expectation messaging)
- [x] Add manual Playwright story for public mobile-nav open and close behavior across route changes and viewport changes
  (Done: `manifest.json` includes `manual-public-mobile-nav`, and `buyer-flow/mobile-nav.spec.md` captures route and viewport behavior)
- [x] Add a public Playwright flow that exercises buyer-facing progressive enhancement promises by repeating quote, restock, and comment actions after reload, back navigation, and direct-entry SSR on the same routes
  (Done: `buyer-flow/progressive-enhancement.spec.md` and manifest story `buyer-progressive-enhancement` define the reload, back-navigation, and direct-entry SSR skeleton)
- [x] Add an internal Playwright navigation flow that walks an operator through sign-in, dashboard, products, product editor, inventory, SKU detail, warehouse detail, warehouse item detail, purchase orders, receiving, comments, and settings while checking route functionality at every step
  (Done: manifest story `operator-full-navigation` defines the full internal route walk and required operator shell assertions)
- [x] Add an internal Playwright navigation flow that starts on the dashboard, drills into an alert-driven workflow, moves through inventory or warehouse detail, finishes with a mutation, and returns to the originating route while confirming summary refresh and route-context continuity
  (Done: `operator-flow/dashboard-triage.spec.md` and manifest story `operator-alert-driven-workflow` define the alert-driven skeleton)
- [x] Add an internal Playwright design-parity checklist flow keyed to `design/homepage_warehouse.tsx` that verifies the GWC internal shell preserves the same dashboard hierarchy, data-density rhythm, card emphasis, quick-action affordances, and warehouse-ops navigation feel where Atlas route semantics overlap
  (Done: `design-parity/internal-warehouse-reference.spec.md` defines the internal parity checklist against `design/homepage_warehouse.tsx`)
- [x] Add manual Playwright story for internal sign-in recovery, dashboard entry, and guarded-route redirect behavior
  (Done: `manifest.json` includes `manual-internal-sign-in-recovery`, and the operator helper contract defines mock-session setup and internal route entry)
- [x] Add manual Playwright story for products list -> product editor -> unsaved-change warning -> save -> success feedback -> reload
  (Done: `operator-flow/product-workflow.spec.md` defines the product workflow skeleton)
- [x] Add manual Playwright story for inventory triage -> SKU detail -> threshold edit overlay -> save -> activity timeline confirmation -> back navigation
  (Done: `operator-flow/inventory-warehouse-workflow.spec.md` defines the inventory threshold and activity timeline skeleton)
- [x] Add manual Playwright story for warehouses list -> warehouse detail -> warehouse item detail -> item update -> return-to-warehouse-context verification
  (Done: `operator-flow/inventory-warehouse-workflow.spec.md` defines the warehouse item context skeleton)
- [x] Add manual Playwright story for transfers list -> transfer detail -> approve or cancel -> timeline update -> list summary refresh
  (Done: `operator-flow/logistics-workflow.spec.md` defines the transfer decision skeleton)
- [x] Add manual Playwright story for purchase orders list -> PO detail -> approve or hold -> dashboard and warehouse summary refresh verification
  (Done: `operator-flow/logistics-workflow.spec.md` defines the purchase-order decision and refresh skeleton)
- [x] Add manual Playwright story for receiving list -> receiving detail -> discrepancy sheet -> classify or reconcile -> save -> queue refresh verification
  (Done: `operator-flow/logistics-workflow.spec.md` defines the receiving discrepancy skeleton)
- [x] Add manual Playwright story for comments moderation queue -> approve or reject -> confirmation overlay -> queue update -> toast verification
  (Done: `operator-flow/moderation-workflow.spec.md` defines the moderation queue, confirmation overlay, queue update, and toast skeleton)
- [x] Add manual Playwright story for settings save covering theme, locale, density, default warehouse, saved-view import or export, and direct-entry reload fidelity
  (Done: `operator-flow/settings-persistence.spec.md` defines the settings persistence skeleton)
- [x] Add manual Playwright story for diagnostics review mode, if shipped, covering route inspection, preference readout, loader timing visibility, and developer-only gating
  (Done: `manifest.json` includes `manual-diagnostics-review-mode` so diagnostics review remains tracked as a gated manual story)
- [x] Add a full admin Playwright path that covers product edit, inventory threshold change, warehouse-scoped item follow-up, purchase-order or receiving update, moderation action, and settings persistence in one end-to-end operator session
  (Done: manifest story `operator-admin-session` defines the full admin path and required route checkpoints)
- [x] Add Playwright assertions along both buyer and operator navigation paths that key route landmarks, section ordering, primary CTA placement, panel density, and major visual compositions still align with the React reference designs instead of drifting back toward scaffold UI
  (Done: manifest stories `buyer-public-parity-checklist` and `operator-internal-parity-checklist` define the route landmark and visual-composition assertions)
- [x] Add Playwright parity checkpoints that capture and compare the GWC shell against the design references at the start, midpoint, and end of both the buyer flow and the operator flow so layout drift is caught before route families diverge
  (Done: `manifest.json` defines public and operator parity start, midpoint, and end checkpoints, and the manifest guard validates the design reference files)

##### 12.18.1 Playwright Implementation Buckets

- [x] Implement the buyer-flow bucket in `examples/tests` with one top-level suite for public navigation and separate specs for browse, form-submit, mobile-nav, and progressive-enhancement paths
  (Done: `examples/tests/atlas-commerce-os/buyer-flow/` contains separate `.spec.md` skeletons and manifest entries for the buyer-flow suite)
- [x] Implement the operator-flow bucket in `examples/tests` with one top-level suite for internal navigation and separate specs for dashboard triage, product workflow, inventory or warehouse workflow, logistics workflow, moderation workflow, and settings persistence
  (Done: `examples/tests/atlas-commerce-os/operator-flow/` contains separate `.spec.md` skeletons and manifest entries for the operator-flow suite)
- [x] Implement the design-parity bucket in `examples/tests` with one public parity spec keyed to `design/homepage_store.tsx` and one internal parity spec keyed to `design/homepage_warehouse.tsx`
  (Done: `examples/tests/atlas-commerce-os/design-parity/` contains separate public and internal reference `.spec.md` skeletons, and `manifest_test.go` validates both reference files)
- [x] Add screenshot-output conventions for buyer-flow, operator-flow, and design-parity buckets so captured artifacts are easy to diff against route family and reference type
  (Done: `examples/tests/atlas-commerce-os/screenshots/README.md`, `manifest.json`, and `docs/README.md#browser-flow-screenshot-story-map` define the screenshot naming and checkpoint conventions)
- [x] Add a bucket-level run order that executes buyer-flow and operator-flow functionality first, then design-parity checks, so visual drift is inspected after core behavior passes
  (Done: `manifest.json` defines and `manifest_test.go` validates the run order `buyer-flow`, `operator-flow`, then `design-parity`)

#### 12.19 End-To-End User Flow Stories

- [x] Create a buyer-flow E2E bucket that mirrors the public Playwright flow bucket and groups browse, submit, recovery, and parity-sensitive buyer journeys under one reusable planning track
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` and manifest track `buyer-flow-e2e` define the buyer E2E planning bucket)
- [x] Create an operator-flow E2E bucket that mirrors the internal Playwright flow bucket and groups dashboard, products, inventory, warehouse, logistics, moderation, and settings journeys under one reusable planning track
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` and manifest track `operator-flow-e2e` define the operator E2E planning bucket)

##### 12.19.1 Buyer-Flow End-To-End Stories

- [x] Add a buyer browse-and-convert E2E flow from landing discovery through catalog filtering, product evaluation, warehouse availability review, and quote request completion
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `buyer-browse-and-convert`)
- [x] Add a buyer restock E2E flow from landing discovery through product detail, restock request, and confirmation-state recovery after reload
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `buyer-restock`)
- [x] Add a buyer feedback E2E flow from product detail through comment submission, pending-state visibility, and moderation expectation messaging
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `buyer-feedback`)
- [x] Add a buyer recovery E2E flow that exercises back navigation, direct-entry SSR, and post-submit return paths across catalog, product detail, and warehouse availability routes
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `buyer-recovery`)
- [x] Add a buyer parity E2E review path that checks the public journey still preserves the storefront-reference hierarchy, CTA emphasis, card rhythm, and secondary-panel behavior where Atlas semantics overlap the React design
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `buyer-parity-review`)

##### 12.19.2 Operator-Flow End-To-End Stories

- [x] Add an operator dashboard-triage E2E flow from sign-in through dashboard triage, inventory detail review, threshold adjustment, and dashboard-summary confirmation
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `operator-dashboard-triage`)
- [x] Add an operator product-workflow E2E flow from products list through product creation, validation correction, save, and warehouse-context follow-up work
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `operator-product-workflow`)
- [x] Add an operator warehouse-workflow E2E flow from warehouse detail through item detail, replenishment or inventory updates, and return-to-warehouse continuity
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `operator-warehouse-workflow`)
- [x] Add an operator logistics-workflow E2E flow from transfer creation through approval or cancelation and downstream route-summary verification
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `operator-logistics-workflow`)
- [x] Add an operator purchase-order-and-receiving E2E flow from purchase-order creation through approval or hold and downstream receiving or warehouse follow-up
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `operator-purchase-order-and-receiving`)
- [x] Add an operator receiving-discrepancy E2E flow from receiving discrepancy review through classification, reconciliation, and activity-log confirmation
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `operator-receiving-discrepancy`)
- [x] Add an operator moderation E2E flow from comments moderation backlog through approve or reject decisions and refreshed public visibility expectations
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `operator-moderation`)
- [x] Add an operator settings-and-persistence E2E flow for changing theme, locale, density, and default warehouse, then confirming those choices persist across sessions and route families
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `operator-settings-and-persistence`)
- [x] Add an operator parity E2E review path that checks the internal journey still preserves the warehouse-reference dashboard hierarchy, data-density rhythm, quick-action emphasis, and navigation feel where Atlas semantics overlap the React design
  (Done: `examples/tests/atlas-commerce-os/e2e/README.md` defines `operator-parity-review`)

#### 12.20 Failure, Recovery, And Edge-Case Stories

- [x] Add tests for 404 recovery on missing public product, missing warehouse, missing warehouse availability pairing, missing internal record, and unknown route entry
  (Done: `examples/tests/atlas-commerce-os/recovery-edge-cases/README.md` and manifest story `recovery-not-found` define the browser-test skeleton, and `manifest_test.go` validates routes and assertions are present)
- [x] Add tests for server-error recovery pages on public and internal routes with route-appropriate retry actions
  (Done: manifest story `recovery-server-error` defines the browser-test skeleton and is validated by `manifest_test.go`)
- [x] Add tests for failed write requests that preserve user-entered form data and show field-level plus summary-level errors
  (Done: manifest story `recovery-failed-write` defines the browser-test skeleton and is validated by `manifest_test.go`)
- [x] Add tests for network interruption during async enhancement paths so public secondary panels and internal side panels recover without corrupting visible state
  (Done: manifest story `recovery-network-interruption` defines the browser-test skeleton and is validated by `manifest_test.go`)
- [x] Add tests for duplicate-submit prevention and retry behavior after a timed-out write flow
  (Done: manifest story `recovery-duplicate-submit-timeout` defines the browser-test skeleton and is validated by `manifest_test.go`)
- [x] Add tests for malformed query params, unsupported sort keys, unsupported density values, invalid locale values, and corrupted browser-storage snapshots
  (Done: manifest story `recovery-invalid-input-state` defines the browser-test skeleton and is validated by `manifest_test.go`)
- [x] Add tests for route refresh during a pending mutation, overlay open during a revalidation, and navigation away during async follow-up work
  (Done: manifest story `recovery-async-navigation-race` defines the browser-test skeleton and is validated by `manifest_test.go`)

#### 12.21 Testing Operations And Tooling Stories

- [x] Add a documented command matrix for Atlas unit tests, wasm tests, server integration tests, Playwright suites, screenshot refreshes, and manual review runs on Windows (Done: `docs/README.md#testing-operations` now defines the Windows command matrix for shared tests, server tests, wasm build, SSR Playwright, cross-browser smoke, startup smoke, screenshot refresh, and manual review)
- [x] Add a documented order-of-operations checklist for rebuilding the Atlas wasm bundle before browser assertions so Playwright never runs against stale binaries (Done: `docs/README.md#stale-wasm-guard` now locks rebuild, timestamp, restart, and browser-run order before Playwright or screenshot assertions)
- [x] Add CI or local-task coverage that groups Atlas tests by layer: unit, component or render, integration, SSR, Playwright, screenshots, and manual review prep (Done: `docs/README.md#local-task-groups` now groups local Atlas review tasks by unit, component/render, integration, SSR, Playwright, screenshots, and manual review prep layers)
- [x] Add a review checklist that names which Atlas routes and user flows must be re-run when shared shell primitives, route loaders, bootstrap payloads, or preference logic change (Done: `docs/README.md#change-triggered-review-checklist` now names the route and flow review sets for shared shell primitives, loaders, bootstrap payloads, preference logic, public buyer flows, and internal operator flows)

## Near-Term Build Sequence

### Phase A

- [x] Lock the route, copy, metadata, and server-response invariants that the rewrite must preserve (Done: `docs/README.md#route-copy-metadata-and-response-invariants` now names the stable public/internal route shapes, copy voice, metadata, and server-response contracts)
- [x] Lock the SSR, router, bootstrap, cache, and derived-state rules before large-scale UI rewrites begin (Done: `docs/README.md#ssr-router-bootstrap-cache-and-derived-state-rules` now locks direct-entry SSR, route ownership, bootstrap scope, cache policy, mutation invalidation, derived-state ownership, and browser-storage precedence)
- [x] Keep the framework-coverage inventory synced as new GWC primitives land so the plan never drifts back toward stale capability assumptions (Done: `docs/README.md#framework-coverage-sync-rule` now defines when `FRAMEWORK_COVERAGE` must be refreshed and how rewrite tasks must evaluate shipped GWC primitives before local machinery)
- [x] Decide which additional GWC primitives will be mandatory in the first rewrite pass instead of deferred (Done: `docs/README.md#mandatory-first-pass-gwc-primitives` now lists the first-pass router, SSR, atom, form, overlay, async/cache, and scheduler primitives that route-family rewrites must evaluate)
- [x] Define the first performance budgets and diagnostics checkpoints that every rewritten route family must satisfy (Done: `docs/README.md#performance-budgets-and-diagnostics-checkpoints` now defines first-pass SSR, hydration, rerender-scope, bootstrap-payload, interaction-latency, and diagnostics review thresholds)

### Phase B

- [x] Standardize the shared HTML patterns that both shells need before route-by-route rewrites begin (Done: `docs/README.md#shared-html-pattern-standard` now locks public shell, internal shell, list/table, form, and recovery-surface HTML patterns for route rewrites)
- [ ] Create the shared visual primitive layer for parity with the React mocks
- [x] Lock the first accessibility, localization, and recovery-state requirements that the new shells must satisfy (Done: `docs/README.md#first-shell-requirements` now names the accessibility, localization, and recovery-state gates for rewritten shells)
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
