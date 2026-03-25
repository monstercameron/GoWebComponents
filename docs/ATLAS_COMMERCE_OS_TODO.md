# Atlas Commerce OS Project Todo

This document turns the flagship example concept into a detailed implementation backlog.

## At A Glance

Atlas Commerce OS is the flagship end-to-end product example for GoWebComponents. This backlog exists to keep that example credible as both:

- a realistic commerce-plus-operations product
- a framework showcase that proves SSR, hydration, routing, forms, overlays, state, diagnostics, accessibility, theming, and localization in one coherent system

Treat this as the planning source of truth for scope, sequencing, and demo-readiness.

Project intent:

- Build a full-stack showcase application for GoWebComponents.
- Demonstrate public SSR and SEO, client hydration, internal operations tooling, SQLite persistence, advanced UI state, overlays, routing, forms, accessibility, theming, and localization.
- Keep the implementation coherent as one product instead of a feature collage.

Working product framing:

- Public side: brand-forward sales pages, product availability, warehouse-aware delivery messaging, comments and requests.
- Internal side: inventory control, warehouse operations, receiving, transfer planning, purchase workflows, moderation, and settings.

## Status Snapshot

The backlog is no longer in early discovery. Most product definition, route design, UI system, data model, SSR contract, and core workflow planning is already locked.

The remaining work is concentrated in a smaller set of finish-line areas:

- closing Atlas-specific framework coverage for fetch, SSR bootstrap, and developer diagnostics
- deepening automated test coverage across data, handlers, route flows, components, and integration paths
- finishing observability and structured logging polish
- deciding which stretch goals are worth shipping versus keeping as future-facing backlog

That means this document now functions less like open ideation and more like a release and validation backlog.

## How To Use This Backlog

Use the document in this order:

- sections 1 through 17 define the intended product and implementation contract
- section 18 and later sections define validation, hardening, and release-readiness work
- unchecked items should be treated as explicit remaining scope, not informal ideas

When work lands in the example codebase, update this backlog and the supporting Atlas notes together so the planning doc does not drift from the shipped example.

## 1. Product Definition

### Core vision

- [x] Lock the product narrative.
  Finalize the one-sentence description of Atlas Commerce OS so the app reads as one believable commerce-plus-operations platform rather than a generic dashboard.
- [x] Define the public versus internal surface boundary.
  Decide which routes are openly crawlable, which require authentication, and which features exist in both contexts with different presentation.
- [x] Define the primary user roles.
  Document at least customer, warehouse operator, inventory manager, and buyer workflows so screen scope stays grounded in actual user jobs.
- [x] Define the top five signature flows.
  Pick the must-demo flows that the implementation will optimize for, such as viewing a product page, submitting a restock request, resolving low stock, receiving inventory, and approving a transfer.
- [x] Define MVP versus stretch features.
  Separate required capabilities from polish ideas so the project can ship in phases without losing direction.

### Success criteria

- [x] Define the framework showcase goals.
  Explicitly list which GoWebComponents capabilities the example must prove in a realistic way, including SSR, hydration, routing, overlays, forms, state, i18n, and diagnostics.
- [x] Define the product quality bar.
  Set expectations for realism, visual polish, responsiveness, accessibility, route depth, persistence, and browser behavior.
- [x] Define the review checklist.
  Create a concise acceptance checklist covering public pages, internal workflows, persistence, a11y, hydration integrity, and visual consistency.

Framework coverage reference:

- [x] Create an explicit framework coverage matrix.
  Track Atlas package and primitive coverage in `examples/86-atlas-commerce-os/FRAMEWORK_COVERAGE.md` so missing features stay visible.
- [x] Add real Atlas coverage for additional `ui` primitives.
  Wire `ui.UsePrevious`, `ui.UseDeferredValue`, `ui.UseDebounced`, and `ui.UseReducer` into active Atlas workflows instead of leaving them as isolated examples.
- [x] Add real Atlas coverage for routing primitives.
  Use params, query state, redirects, guards, nested layouts, loaders, and revalidation in real Atlas routes.
- [x] Add real Atlas coverage for async and overlay primitives.
  Use overlays, portals, async boundaries, lazy loading, tasks, and transition lanes in product and operations workflows.
- [x] Add real Atlas coverage for shared state and snapshot primitives.
  Use atoms, computed values, derived slices, and snapshot persistence for operator workspaces and reviewer handoff.
- [x] Add real Atlas coverage for fetch, SSR bootstrap, and devtools.
  Use data resources, cached resources, bootstrap scripts, hydration reuse, and a live diagnostics panel.
  Atlas now uses cached fetch resources and bootstrap reuse in `client/main.go`, and diagnostics mode mounts `devtools.Panel` plus `devtools.SnapshotNow()` summary from `shared/atlas/page.go`.

## 2. Information Architecture And Route Map

### Recommended route map

Use this route map as the default planning baseline unless a later review explicitly changes it.

Public pages:

- `/`
  Brand-forward landing page with hero, featured categories, fulfillment proof, and top products.
- `/shop`
  Product collection page with query-driven search, sort, and filter state.
- `/shop/:productSlug`
  Primary sales page with product detail, warehouse-aware availability, comments, and public forms.
- `/warehouses`
  Warehouse index page explaining regional fulfillment footprint.
- `/warehouses/:warehouseSlug`
  Public warehouse page with local inventory highlights, service promise, and featured products.
- `/warehouses/:warehouseSlug/availability/:productSlug`
  Warehouse-specific product availability page that ties stock and delivery promise to one product.

Internal pages:

- `/app`
  Authenticated shell route that redirects to `/app/dashboard`.
- `/app/dashboard`
  Operations overview showing alerts, low-stock items, inbound shipments, and moderation counts.
- `/app/inventory`
  Main sortable and filterable inventory management table.
- `/app/inventory/:sku`
  SKU detail route with stock by warehouse, audit timeline, internal notes, and actions.
- `/app/warehouses`
  Internal warehouse list with throughput and pressure summaries.
- `/app/warehouses/:warehouseId`
  Internal warehouse detail route for staffing, backlog, and location-specific inventory issues.
- `/app/purchase-orders`
  Purchase order list view.
- `/app/purchase-orders/:purchaseOrderId`
  Purchase order detail and approval route.
- `/app/transfers`
  Transfer planning, queue, and recommendation view.
- `/app/transfers/:transferId`
  Transfer detail route with timeline and receipt state.
- `/app/receiving`
  Receiving queue view for inbound purchase orders and transfers.
- `/app/receiving/:sessionId`
  Receiving session route with discrepancy workflow.
- `/app/comments`
  Public comment, review, and request moderation queue.
- `/app/settings`
  Theme, locale, density, default warehouse, and saved-view preferences.

### Public routes

- [x] Define the public route tree.
  Specify routes for landing page, collection pages, product detail pages, warehouse pages, warehouse-specific availability pages, and any comments or request flows that deserve deep links.
- [x] Lock the public page inventory.
  Confirm the baseline public pages are landing, catalog, product detail, warehouses index, warehouse detail, and warehouse-specific availability pages.
- [x] Define the sales landing page structure.
  Decide hero content, featured categories, proof points, warehouse fulfillment messaging, and call-to-action placement.
- [x] Define the landing page modules in detail.
  Include recommended modules for hero, regional fulfillment proof, featured products, category cards, customer proof, and operator-facing product narrative.
- [x] Define the collection or catalog route behavior.
  Decide sorting, filtering, pagination, search, query parameter behavior, canonicalization rules, and whether filter states are indexable.
- [x] Define catalog query parameters.
  Standardize `q`, `category`, `warehouse`, `availability`, `sort`, `page`, and optional price or tag filters so server rendering and client routing stay aligned.
- [x] Define the product sales page structure.
  Lay out media, product copy, price, inventory promise, specs, reviews or comments, related products, and lead-capture or request forms.
- [x] Define the product page tab model.
  Decide whether detail, specs, shipping, comments, and reviews are tabbed, section-based, or route-fragment based.
- [x] Define warehouse public pages.
  Decide what a warehouse page exposes publicly, such as service region, fulfillment speed, stocked highlights, and SKU-specific availability.
- [x] Define warehouse-specific availability page behavior.
  Decide whether the page focuses on promise messaging, raw stock counts, nearby alternatives, or a hybrid model.

### Internal routes

- [x] Define the authenticated app shell.
  Decide top navigation, side navigation, command palette entry, alert center, settings access, and responsive collapse behavior.
- [x] Define the primary internal navigation groups.
  Group the internal shell into overview, inventory, warehouses, purchasing, transfers, receiving, comments, and settings.
- [x] Define the internal route tree.
  Specify routes for dashboard, inventory list, SKU detail, warehouse list, warehouse detail, purchase orders, transfers, receiving, moderation, and settings.
- [x] Define route-level tabs for detail pages.
  Decide whether SKU, warehouse, purchase order, and transfer detail routes use internal tab state for overview, activity, notes, comments, and analytics.
- [x] Define nested route structure.
  Decide where nested layouts and outlets apply so related views share shell UI, filters, and context cleanly.
- [x] Define overlay-versus-route responsibility.
  Decide which detail experiences should remain true routes and which should open as overlay-backed secondary surfaces while preserving the current route.
- [x] Define deep-link behavior.
  Ensure major workflows can be opened directly from URLs, including filtered lists, selected tabs, and detail views.

### Route metadata and navigation

- [x] Define metadata rules per route.
  Specify titles, descriptions, canonical URLs, structured data expectations, and which routes should emit social metadata.
- [x] Define navigation state persistence.
  Decide which filters, tabs, and selection states live in the URL versus local state versus persisted preferences.
- [x] Define access guard behavior.
  Decide how internal routes enforce authentication and how unsaved form workflows block navigation.
- [x] Define page ownership matrix.
  Record for each route whether it is SSR-only, SSR+hydrate, internal-only, public SEO-sensitive, or modal-triggering.

## 3. Visual Direction And Design System

### Brand and visual language

- [x] Define the visual identity.
  Choose the visual direction for Atlas Commerce OS, including mood, contrast level, shape language, gradients, and how public and internal surfaces differ.
- [x] Define typography choices.
  Choose display, heading, and dense UI text styles that feel intentional and non-generic.
- [x] Define color tokens.
  Establish semantic colors for stock health, warehouse states, alerts, forms, moderation states, and marketing accents.
- [x] Define spacing, radius, and elevation tokens.
  Create consistent primitives for cards, tables, overlays, forms, and hero sections.

### Theme system

- [x] Define light and dark mode behavior.
  Decide default theme, persisted preference behavior, SSR-safe initial theme resolution, and whether public and internal surfaces prefer different defaults.
- [x] Define density modes.
  Decide whether internal lists support compact and comfortable display settings.
- [x] Define motion principles.
  Standardize page transitions, overlay entrance and exit motion, table feedback, and reduced-motion fallbacks so the example feels deliberate rather than generic.
- [x] Define responsive behavior.
  Specify how public pages and internal tools adapt across mobile, tablet, and desktop without collapsing into generic stacked cards.

### Component vocabulary

- [x] Define shared UI primitives for the example.
  Identify repeated components such as stat cards, filter chips, list headers, badge systems, segmented controls, timeline rows, and detail panels.
- [x] Define data table patterns.
  Decide list header layout, sort affordances, filter controls, sticky columns, row selection, bulk action affordances, and empty-state treatment.
- [x] Define overlay patterns.
  Decide when to use centered modals, side sheets, anchored menus, tooltips, stacked dialogs, and command palette overlays.
- [x] Define loading, empty, and error states.
  Standardize skeletons, no-results states, empty-first-use states, and retry surfaces for catalog pages, internal tables, detail views, and form-heavy workflows.

## 4. Data Model And SQLite Schema

### Core entities

- [x] Define product entities.
  Specify fields for SKU, slug, title, category, pricing, summary, long description, media, SEO metadata, and active status.
- [x] Define warehouse entities.
  Specify fields for warehouse identity, location, service region, fulfillment SLA, capacity, and public visibility.
- [x] Define inventory level entities.
  Specify on-hand, reserved, available, inbound, damaged, safety stock, reorder point, and last-updated fields.
- [x] Define purchase order entities.
  Specify purchase order headers, status, vendor details, ETA, line items, and approval state.
- [x] Define transfer request entities.
  Specify source warehouse, destination warehouse, quantities, reason, priority, approval status, and shipment state.
- [x] Define receiving session entities.
  Specify expected lines, actual lines, discrepancy notes, receiver identity, timestamps, and final reconciliation state.
- [x] Define comment or review entities.
  Decide whether customer comments, product questions, and internal notes share a model or use separate tables.
- [x] Define audit event entities.
  Store inventory changes, form submissions, approvals, moderation actions, and transfer lifecycle events in a timeline-friendly shape.
- [x] Define user and preference entities.
  Store theme, locale, density, default warehouse, saved views, and role data.

### Relational design

- [x] Define foreign-key relationships.
  Map product-to-inventory, product-to-comments, warehouse-to-inventory, order-to-lines, and transfer-to-audit relationships cleanly.
- [x] Define indexing strategy.
  Add indexes for slug lookups, SKU lookups, warehouse-specific stock queries, moderation queues, and saved-view retrieval.
- [x] Define seed data strategy.
  Create realistic demo data volume and content so sorting, filtering, comments, and stock scenarios feel credible.

### Persistence and migrations

- [x] Define schema migration approach.
  Decide how the example initializes and evolves the SQLite schema reliably.
- [x] Define development reset tooling.
  Provide a way to reseed or reset the database so testing and demos stay deterministic.

## 5. Server Architecture

### Recommended endpoint style

Use a split model:

- SSR HTML routes for page entry
- JSON endpoints for async client refresh and table data
- form-post or JSON mutation endpoints for writes

Suggested conventions:

- Public page routes return SSR HTML.
- Internal app entry routes return SSR HTML plus bootstrap payload.
- Read-heavy filtered tables can use route loaders backed by internal JSON endpoints where needed.
- Mutations should return structured results that support revalidation and field-level errors.

### HTTP server design

- [x] Define the server entrypoint.
  Decide how the Go server boots SQLite, serves static assets, performs SSR, and exposes JSON or form endpoints.
- [x] Define the concrete page-rendering handlers.
  Map each public and internal route to a server handler that can load data, emit metadata, and serialize bootstrap payloads consistently.
- [x] Define route handling split.
  Decide which requests return SSR HTML, which return JSON, and which process form submissions directly.
- [x] Define the SSR route handler contract.
  Standardize what each route handler returns: route data, metadata, bootstrap payload, auth context, theme, and locale.
- [x] Define error handling strategy.
  Standardize 404, validation, authentication, and server error behavior for public and internal surfaces.

### API and form endpoints

- [x] Define public interaction endpoints.
  Include comment submission, quote requests, restock requests, and any contact or lead forms.
- [x] Define the public read endpoints.
  Decide whether product pages, warehouse highlights, and comments need explicit JSON endpoints in addition to SSR loaders.
- [x] Define inventory mutation endpoints.
  Include stock adjustments, transfer creation, purchase order updates, receiving reconciliation, and moderation actions.
- [x] Define the internal read endpoints.
  Standardize list and detail APIs for inventory, warehouses, purchase orders, transfers, receiving sessions, and moderation queues.
- [x] Define list query endpoints.
  Support catalog filters, internal inventory filters, moderation lists, and saved views in a consistent query format.
- [x] Define endpoint naming conventions.
  Keep endpoints predictable by grouping them under `/api/public/...`, `/api/app/...`, or another consistent scheme.
- [x] Define server validation rules.
  Ensure product comments, internal mutations, and warehouse workflows all return structured validation errors.

### Proposed endpoint inventory

Use this as the default API surface unless scope is intentionally reduced.

Public read endpoints:

- `GET /api/public/catalog`
  Return catalog results for search, filtering, sort, and pagination.
- `GET /api/public/products/:productSlug`
  Return product detail payload for hydrated enhancements beyond SSR.
- `GET /api/public/products/:productSlug/comments`
  Return approved comments, reviews, or questions with pagination.
- `GET /api/public/warehouses`
  Return public warehouse summaries.
- `GET /api/public/warehouses/:warehouseSlug`
  Return public warehouse detail and stocked highlights.
- `GET /api/public/warehouses/:warehouseSlug/availability/:productSlug`
  Return warehouse-specific product availability detail.

Public write endpoints:

- `POST /api/public/products/:productSlug/comments`
  Submit a comment, review, or question.
- `POST /api/public/products/:productSlug/restock-requests`
  Submit a restock notification request.
- `POST /api/public/products/:productSlug/quote-requests`
  Submit a bulk quote or business inquiry.

Internal read endpoints:

- `GET /api/app/dashboard`
  Return alerts, summary metrics, inbound shipments, and low-stock lists.
- `GET /api/app/inventory`
  Return filtered inventory table results.
- `GET /api/app/inventory/:sku`
  Return SKU detail, warehouse breakdown, activity, and comments.
- `GET /api/app/warehouses`
  Return internal warehouse summaries.
- `GET /api/app/warehouses/:warehouseId`
  Return internal warehouse detail.
- `GET /api/app/purchase-orders`
  Return purchase order list results.
- `GET /api/app/purchase-orders/:purchaseOrderId`
  Return purchase order detail.
- `GET /api/app/transfers`
  Return transfers and recommendation data.
- `GET /api/app/transfers/:transferId`
  Return transfer detail and history.
- `GET /api/app/receiving`
  Return receiving queue data.
- `GET /api/app/receiving/:sessionId`
  Return receiving session detail.
- `GET /api/app/comments`
  Return moderation queue items.
- `GET /api/app/saved-views`
  Return saved filter and sort presets.

Internal write endpoints:

- `POST /api/app/inventory/:sku/adjustments`
  Create a manual stock adjustment.
- `POST /api/app/inventory/:sku/thresholds`
  Update reorder and safety stock thresholds.
- `POST /api/app/transfers`
  Create a transfer request.
- `POST /api/app/transfers/:transferId/approve`
  Approve a transfer.
- `POST /api/app/transfers/:transferId/cancel`
  Cancel a transfer.
- `POST /api/app/purchase-orders`
  Create a purchase order.
- `POST /api/app/purchase-orders/:purchaseOrderId/approve`
  Approve a purchase order.
- `POST /api/app/receiving/:sessionId/reconcile`
  Submit expected versus actual receiving results.
- `POST /api/app/comments/:commentId/approve`
  Approve a public comment or review.
- `POST /api/app/comments/:commentId/reject`
  Reject or hide a public comment or review.
- `POST /api/app/comments/:commentId/flag`
  Mark a moderation item for follow-up.
- `POST /api/app/saved-views`
  Create a saved view.
- `POST /api/app/preferences`
  Save theme, locale, density, and default warehouse preferences.

### Security and operational concerns

- [x] Define auth and session strategy for the example.
  Decide how internal routes are protected without overcomplicating the showcase.
- [x] Define CSRF strategy for server-backed forms.
  Choose how the example will demonstrate secure posting for public and internal forms.
- [x] Define file upload needs.
  Decide whether the example should include product media uploads or receiving attachments, and if so how they are handled.
- [x] Define static asset strategy.
  Decide how product media, brand graphics, Open Graph images, cache headers, and fallback assets are served so the demo feels production-shaped.

## 6. SSR, Hydration, And SEO

### SSR model

- [x] Define which routes are SSR-rendered.
  Identify the public pages and internal routes that should render meaningful HTML before wasm startup.
- [x] Lock the SSR page set.
  Treat landing, catalog, product detail, warehouse detail, warehouse-specific availability, dashboard, inventory list, and SKU detail as the default SSR routes.
- [x] Define hydration boundaries.
  Decide where full hydration is required and whether any page sections remain static after SSR.
- [x] Define page-by-page hydration goals.
  Decide whether comments, filters, overlays, and internal tables hydrate immediately or progressively.
- [x] Define bootstrap payload shape.
  Decide what route data, preferences, locale, theme, and user state transfer from server to client.
- [x] Define a page bootstrap contract.
  Standardize route payload sections for `route`, `data`, `i18n`, `theme`, `user`, and any saved-view state.
- [x] Define SSR-to-client consistency rules.
  Ensure theme, locale, route data, and metadata do not mismatch during hydration.

### SEO surface

- [x] Define SEO targets for public routes.
  Decide which pages should rank or share well and what metadata or structured data they need.
- [x] Define canonical URL policy.
  Decide how filtered catalog pages, warehouse pages, and comment pagination should normalize canonical links.
- [x] Define structured data coverage.
  Add product and availability schema where it meaningfully improves search presentation.
- [x] Define sitemap and social-preview coverage.
  Decide which public routes enter the sitemap, what robots behavior is expected, and how share cards or preview images are generated.
- [x] Define content-indexing boundaries.
  Ensure internal app routes are not treated as public marketing pages.

## 7. Public Commerce Features

### Landing and discovery

- [x] Define the marketing landing page modules.
  Decide hero content, metrics, warehouse network storytelling, featured products, and brand trust sections.
- [x] Define the landing page CTA map.
  Decide primary and secondary calls to action, such as browse catalog, view warehouse network, request quote, or inspect availability.
- [x] Define category and collection browsing.
  Decide sort modes, filter categories, availability-based filtering, and promotional callouts.
- [x] Define the catalog page modules.
  Include results header, filter rail, sort control, product grid or list, saved filter state, and pagination behavior.

### Product sales pages

- [x] Define product page content hierarchy.
  Decide how price, shipping promise, availability, specs, media, reviews, and forms are arranged.
- [x] Define the exact sales page sections.
  Use a baseline structure of hero, purchase or inquiry block, availability by warehouse, specs, social proof, comments, and related products.
- [x] Define inventory-aware sales messaging.
  Tie stock and ETA to warehouse data in a way that feels useful rather than fake.
- [x] Define product route data requirements.
  Ensure the server loads product detail, warehouse availability, related products, approved comments, and form state defaults for SSR.
- [x] Define related product recommendations.
  Decide simple recommendation logic and placement.
- [x] Define low-stock and unavailable-product fallback UX.
  Decide what the user sees when a product is unavailable, including nearby warehouse options, substitute products, and restock or quote callouts.

### Comments, reviews, and public forms

- [x] Define the public comment system.
  Decide whether the public surface supports comments, reviews, questions, or all three, and how they differ.
- [x] Define the sales-page comments module.
  Decide whether comments live inline below the product, in tabbed sections, or in a secondary route-backed thread view.
- [x] Define comment moderation rules.
  Decide approval states, visibility timing, internal review flow, and any abuse controls.
- [x] Define restock request form behavior.
  Capture user intent when stock is low or unavailable and tie it back to internal demand views.
- [x] Define quote request form behavior.
  Provide a bulk or business inquiry path that shows more advanced form handling than a simple contact form.
- [x] Define public form success and error states.
  Ensure public forms feel polished, accessible, and server-backed.
- [x] Define public endpoint payloads.
  Specify request and response shapes for comment submission, quote requests, and restock requests so form wiring stays consistent.

## 8. Warehouse Experience

### Public warehouse pages

- [x] Define public warehouse overview pages.
  Decide how each warehouse is represented publicly, including service promise, region, and stocked highlights.
- [x] Define the public warehouse page sections.
  Use a baseline of hero summary, service zone, featured inventory, fulfillment promise, and operational notices.
- [x] Define SKU availability by warehouse.
  Show which locations can fulfill a SKU and what delivery promise they imply.

### Internal warehouse operations

- [x] Define the internal warehouse list.
  Decide what summary metrics matter at the warehouse list level.
- [x] Define the internal warehouse list columns and filters.
  Include location, service region, active SKUs, low-stock count, receiving backlog, transfer pressure, and fulfillment SLA.
- [x] Define the internal warehouse detail page.
  Include backlog, low-stock items, receiving queue, transfer pressure, and staff notes.
- [x] Define the warehouse detail sections.
  Split the page into overview, inventory health, inbound work, outbound pressure, notes, and activity history.
- [x] Define warehouse comparison views.
  Decide whether the app compares stock health across locations and how that is visualized.

## 9. Inventory Management Features

### Inventory list and filters

- [x] Define the main inventory table columns.
  Decide which fields matter most for operators and managers, such as SKU, title, warehouse, available stock, cover days, inbound, and status.
- [x] Define sorting behavior.
  Support meaningful sorts including quantity, days of cover, revenue importance, and freshness of updates.
- [x] Define filtering behavior.
  Support warehouse, category, supplier, stock health, availability state, moderation flags, and search.
- [x] Define saved views.
  Let users save common filter or sort sets and restore them later.
- [x] Define bulk actions.
  Decide whether users can bulk create transfers, adjust thresholds, or tag items for review.

### SKU detail workflows

- [x] Define the SKU detail route layout.
  Include summary stats, warehouse breakdown, inbound shipments, recent activity, comments, and action surfaces.
- [x] Define threshold editing.
  Allow reorder points and safety stock to be updated through a clean modal or side panel flow.
- [x] Define stock adjustment behavior.
  Decide how manual corrections are entered, validated, and audited.
- [x] Define internal comments or notes.
  Decide whether operators can leave notes on SKUs separately from public comments.

## 10. Purchasing, Transfers, And Receiving

### Purchase orders

- [x] Define purchase order list and detail screens.
  Include status, vendor, ETA, line items, and approval actions.
- [x] Define purchase order creation and approval flow.
  Show structured forms, server validation, and post-submit revalidation.

### Transfers

- [x] Define transfer recommendation logic.
  Decide how the demo identifies stock imbalances and suggests cross-warehouse moves.
- [x] Define transfer creation workflow.
  Support source and destination selection, quantity entry, reason capture, and confirmation.
- [x] Define transfer lifecycle states.
  Model draft, submitted, approved, in transit, received, and cancelled behaviors.

### Receiving

- [x] Define receiving queue and detail view.
  Show incoming purchase orders and transfers awaiting check-in.
- [x] Define receiving form workflow.
  Capture expected versus actual units, discrepancy reasons, notes, and final confirmation.
- [x] Define discrepancy resolution overlays.
  Use stacked modal or sheet flows for mismatch handling and audit capture.
- [x] Define optimistic versus authoritative update behavior.
  Decide what updates appear immediately and what waits for server confirmation.

## 11. Forms And Validation

### Public forms

- [x] Define common public form primitives.
  Standardize label, hint, error, success, pending, and confirmation behavior for comments, quote requests, and restock forms.
- [x] Define moderation-safe comment submission UX.
  Tell the user whether comments publish immediately or await review.

### Internal forms

- [x] Define internal form patterns.
  Standardize dense modal forms, sheet forms, inline forms, and page-level forms.
- [x] Define validation projection rules.
  Ensure server field errors map cleanly back into the public form state shown in the UI.
- [x] Define submit-intent behavior.
  Support actions like save draft, approve, reject, and receive with distinct button intent flows.

## 12. Overlays, Portals, And Interaction Layers

### Modal system

- [x] Define overlay inventory.
  List every dialog, side sheet, anchored menu, tooltip, and command palette the example needs.
- [x] Define stacking rules.
  Decide which flows can nest and how focus, escape handling, and outside-click behavior should work.
- [x] Define side-panel strategy.
  Use sheets or drawers for SKU detail, warehouse quick views, and order summaries where preserving list context matters.
- [x] Define destructive confirmation patterns.
  Standardize confirmation dialogs for cancel, delete, reject, and irreversible actions.
- [x] Define notification and inline-feedback patterns.
  Standardize toast usage, inline banners, success confirmations, and long-running task feedback so mutations feel responsive without becoming noisy.

### Commanding and shortcuts

- [x] Define the command palette.
  Support keyboard-first navigation to products, SKUs, warehouses, orders, and settings.
- [x] Define anchored actions.
  Use menus and popovers for row-level quick actions without overloading table cells.

## 13. Theme, Preferences, And Localization

### Theme and preferences

- [x] Define persisted preferences.
  Store theme, density, locale, default warehouse, and saved views per user.
- [x] Define SSR-safe theme hydration.
  Ensure no visible theme flash or mismatch on first paint.

### Localization

- [x] Define supported locales.
  Decide which locales are necessary to demonstrate the system credibly.
- [x] Define locale-aware route strategy.
  Decide whether the public side, internal side, or both support locale-prefixed routes.
- [x] Define translation scope.
  Decide which surfaces are translated fully and which remain single-language for scope control.
- [x] Define RTL coverage.
  Include at least one path that proves the UI works in a right-to-left locale.

## 14. State, Async Data, And Diagnostics

### State architecture

- [x] Define shared app state boundaries.
  Decide which state lives in atoms, route loaders, local component state, and persisted preferences.
- [x] Define derived and computed metrics.
  Calculate low-stock indicators, days of cover, transfer pressure, and moderation counts from shared state.
- [x] Define snapshot behavior.
  Decide whether to support saved workspace or diagnostic state snapshots.

### Async and live updates

- [x] Define route revalidation strategy.
  Decide when loaders rerun after mutations and how stale data is presented.
- [x] Define cached resource usage.
  Decide which datasets benefit from client-side cache reuse.
- [x] Define live activity behavior.
  Decide whether to simulate alert feeds or activity streams through channels or tasks.

### Diagnostics and devtools

- [x] Define the embedded diagnostics drawer.
  Decide what runtime information the example should expose without overwhelming normal users.
- [x] Define demo-only developer affordances.
  Consider a controlled devtools panel, seed reset button, snapshot viewer, or route-state inspector.

## 15. Accessibility

### Public accessibility

- [x] Define accessible commerce patterns.
  Ensure product pages, comments, reviews, and public forms remain navigable and readable for assistive tech.
- [x] Define accessible filter and sort controls.
  Make data-heavy lists keyboard-operable and understandable.

### Internal accessibility

- [x] Define keyboard-first internal workflows.
  Ensure dense tables, menus, dialogs, and command palette flows work without mouse-only assumptions.
- [x] Define overlay accessibility behavior.
  Apply proper labelling, focus trapping, restore focus, inert background, and nested overlay handling.
- [x] Define live announcement behavior.
  Announce route changes, validation errors, save states, and moderation outcomes where appropriate.
- [x] Define reduced-motion and contrast requirements.
  Ensure animation-heavy flows, dense tables, and stock-status colors still work for users who need reduced motion or stronger visual separation.

## 16. Content And Copy

### Product and commerce content

- [x] Define product catalog tone.
  Make the public side feel branded and premium rather than placeholder enterprise content.
- [x] Define warehouse and logistics copy style.
  Ensure operations messaging is concise, legible, and realistic.

### Comment and review content

- [x] Define seeded customer feedback content.
  Add believable approved, pending, and flagged examples to make moderation meaningful.
- [x] Define internal audit and note content.
  Seed enough real-looking operational text to validate timelines and collaboration surfaces.

## 17. Implementation Sequencing

### Phase 1: Foundations

- [x] Implement project skeleton.
  Set up server entrypoint, sqlite initialization, routing scaffold, shared shell, theme tokens, and seed data bootstrapping.
- [x] Implement route tree and shell layouts.
  Establish public and internal layouts before deeper feature work.

### Phase 2: Public commerce surface

- [x] Build landing, catalog, and product routes.
  Get the public side SSR-rendered and metadata-aware early.
- [x] Add comments and public forms.
  Make sure form and moderation flows exist before deeper operations logic.

### Phase 3: Internal operations surface

- [x] Build dashboard, inventory list, and SKU detail.
  Establish the heart of the operational experience.
- [x] Add warehouses, transfers, purchase orders, and receiving.
  Expand from read-heavy views into mutation-heavy workflows.

### Phase 4: Cross-cutting systems

- [x] Add theming, localization, overlays, saved views, and diagnostics.
  Layer in the advanced framework showcase systems after the main product flows work.
- [x] Add SEO refinements, accessibility passes, and responsive polish.
  Finalize quality after the core product story is stable.
- [x] Add demo-readiness documentation and fixtures.
  Finish a focused README, local run instructions, seed-reset workflow, and a reviewer-facing demo script so the example is easy to evaluate.

## 18. Testing And Validation

### User stories for testing

- [x] Define public commerce user stories.
  Capture testable stories such as discovering a product, filtering the catalog, reading warehouse-aware availability, submitting a comment, and requesting restock or quote follow-up.
- [x] Define internal inventory user stories.
  Capture stories such as finding a low-stock SKU, changing thresholds, creating a transfer, receiving inventory, and validating that the dashboard reflects the change.
- [x] Define moderation user stories.
  Capture stories such as reviewing a public comment, approving it, rejecting it, and confirming public visibility changes correctly.
- [x] Define warehouse operation user stories.
  Capture stories such as opening a warehouse page, inspecting backlog, resolving a discrepancy, and comparing warehouse stock pressure.
- [x] Define personalization user stories.
  Capture stories such as switching theme, changing locale, restoring a saved view, and reloading into the same personalized shell state.
- [x] Define SSR and hydration user stories.
  Capture stories such as entering a public product page from search, getting the right metadata and first paint, then hydrating into live filters, comments, and forms without mismatch.

### Recommended user story inventory

Use these as the default acceptance-story set:

- [x] Story: Customer browses the sales catalog.
  A visitor opens `/shop`, filters by category and availability, changes sort order, and lands on a product page with the expected filtered context.
- [x] Story: Customer inspects a product detail page.
  A visitor opens `/shop/:productSlug`, sees SSR-rendered product content, warehouse-aware promise messaging, and related products, then hydrates into a fully interactive page.
- [x] Story: Customer submits a product comment.
  A visitor writes a comment or question, submits it through a server-backed form, receives confirmation, and sees the expected moderation message.
- [x] Story: Customer requests a restock notification.
  A visitor on a low-stock or unavailable product submits a restock request and gets a clear success path.
- [x] Story: Buyer requests a quote.
  A visitor submits a bulk inquiry and the request is stored and visible internally.
- [x] Story: Operator works the inventory table.
  An internal user opens `/app/inventory`, sorts by stock health, filters to one warehouse, searches by SKU, and saves that view.
- [x] Story: Operator edits SKU thresholds.
  An internal user opens a SKU detail route or sheet, edits reorder thresholds, saves changes, and sees audit history update.
- [x] Story: Manager creates a transfer.
  An internal user identifies stock imbalance, opens the transfer workflow, submits a request, and sees the new transfer in queue views.
- [x] Story: Receiver reconciles an inbound shipment.
  An internal user opens a receiving session, records mismatch quantities, resolves a discrepancy modal, and finishes reconciliation.
- [x] Story: Moderator reviews customer feedback.
  An internal user opens `/app/comments`, filters pending items, approves or rejects one, and confirms the public-facing state updates correctly.
- [x] Story: User resumes personalized state.
  A returning user reloads the application and sees the same theme, locale, density, default warehouse, and saved views restored.
- [x] Story: Public page remains SEO-safe.
  A product page returns the expected title, description, canonical URL, structured data, and crawlable content before hydration.

### Server and data tests

- [x] Add sqlite data-layer tests.
  Cover seed loading, queries, filters, comment persistence, and inventory mutations.
  Covered by `examples/86-atlas-commerce-os/server/db/store_integration_test.go` (`TestLoadMigrationsAndMigrateFallback`, `TestStoreReadFlows`, and `TestStoreAdminAndMutationFlows`).
- [x] Add repository-level query tests.
  Validate product lookup, warehouse lookup, availability joins, comment retrieval, saved views, and inventory filter combinations.
  Query coverage lives in `examples/86-atlas-commerce-os/server/db/store_integration_test.go` via `TestStoreReadFlows` and includes catalog filters, product and warehouse lookups, availability joins, comments, and saved-view reads.
- [x] Add write-path persistence tests.
  Validate comment creation, quote request creation, stock adjustment writes, threshold updates, transfer lifecycle updates, and receiving reconciliation writes.
  Persistence paths are covered in `examples/86-atlas-commerce-os/server/db/store_integration_test.go` under `TestStoreAdminAndMutationFlows`, including comments, quote/restock requests, inventory and threshold updates, transfers, purchase orders, and receiving reconciliation.
- [x] Add handler tests.
  Validate SSR responses, JSON endpoints, form submissions, validation failures, and moderation actions.
  Covered by `examples/86-atlas-commerce-os/server/server_test.go` plus `products_routes_error_test.go`, including SSR route assertions, JSON endpoint checks, form-post success and validation-failure paths, and comment moderation flows.

### Unit test ideas

- [x] Add unit tests for query parsing.
  Test catalog and inventory query parsing for sort, filter, pagination, invalid values, and canonicalization.
  Added focused parsing coverage in `examples/86-atlas-commerce-os/server/query_parsing_test.go` for pagination fallback, public warehouse query defaults, Atlas bootstrap-query filtering, and inventory sort/filter normalization.
- [x] Add unit tests for metadata generation.
  Test route title, description, canonical URL, and structured-data generation for public product and warehouse routes.
- [x] Add unit tests for theme and preference resolution.
  Test SSR-safe theme, locale, density, and default-warehouse resolution from cookies, session state, or bootstrap payload.
- [x] Add unit tests for availability messaging.
  Test warehouse-aware delivery promise logic and stock-status labeling.
- [x] Add unit tests for transfer recommendation logic.
  Test derived stock-pressure and cross-warehouse rebalance recommendations.
- [x] Add unit tests for receiving reconciliation rules.
  Test expected-versus-actual quantity handling, discrepancy classification, and final state transitions.
  Added `examples/86-atlas-commerce-os/server/db/receiving_rules_test.go` for expected-vs-actual receiving-line checks plus reconcile transition coverage, and expanded `validateReceivingRequest` status coverage in `examples/86-atlas-commerce-os/server/mutations_helpers_test.go`.
- [x] Add unit tests for moderation-state transitions.
  Test pending, approved, rejected, and flagged comment lifecycle rules.
- [x] Add unit tests for saved-view serialization.
  Test encoding and decoding of filter, sort, density, and warehouse selection presets.

### UI and route tests

- [x] Add route-level tests for public pages.
  Cover SSR render, metadata, hydration bootstrap, and query-driven filters.
  Public-route coverage is in `examples/86-atlas-commerce-os/server/server_test.go` (`TestPublicSSRRoutes` and public cases in `TestPublicAndInternalJSONAPIsReturnData`) and validates SSR output, metadata tags, bootstrap payloads, and filter-driven route requests.
- [x] Add route-level tests for internal pages.
  Cover dashboard load, list filtering, modal flows, receiving workflows, and revalidation behavior.
  Internal-route coverage is exercised in `examples/86-atlas-commerce-os/server/server_test.go` through `TestInternalSSRRoutes`, nested-route bootstrap tests, threshold-history external-bootstrap checks, and internal API route assertions.
- [x] Add accessibility-focused browser coverage.
  Validate keyboard navigation, overlay focus handling, form errors, and route announcements.
- [x] Add theming and locale coverage.
  Validate SSR-stable theme load, locale switching, formatting, and RTL support.
  Covered by SSR and bootstrap tests in `examples/86-atlas-commerce-os/server/server_test.go` (direct-entry preference save and SSR class/locale assertions) plus locale-direction helper coverage in `examples/86-atlas-commerce-os/shared/atlas/derived_state_bootstrap_test.go`.

### Component test ideas

- [x] Add component tests for data table primitives.
  Validate sortable headers, filter chip rendering, empty states, saved-view badges, and bulk-selection affordances.
  Added table-primitive coverage in `examples/86-atlas-commerce-os/shared/atlas/table_primitives_test.go`, including empty-state rows, sortable headers, filter and saved-view rail chips, and bulk moderation affordances on the comments table route.
- [x] Add component tests for public comment and form modules.
  Validate label wiring, error rendering, success states, moderation messages, and pending states.
  Added `examples/86-atlas-commerce-os/shared/atlas/public_comment_modules_test.go` covering comment-form validation, ARIA label and error wiring, moderation badge rendering, pending refresh/submit states, and pending-comment merge behavior.
- [x] Add component tests for warehouse availability cards.
  Validate stock labels, ETA messaging, fallback states, and warehouse-specific promise rendering.
  Added coverage in `examples/86-atlas-commerce-os/shared/atlas/warehouse_availability_cards_test.go` for promise-lane cards, refresh and empty fallbacks, warehouse-specific availability promise rendering, and status-driven support-point messaging.
- [x] Add component tests for overlay-backed workflows.
  Validate SKU threshold modal, transfer dialog, discrepancy resolution dialog, side sheets, and command palette behavior.
  Added overlay component coverage in `examples/86-atlas-commerce-os/shared/atlas/overlay_workflows_test.go` for threshold route-sheet rendering, transfer/receiving/moderation confirmation dialogs, and shared dismissible side-sheet behavior.
- [x] Add component tests for preference controls.
  Validate theme toggle, locale selector, density selector, and saved-view controls.
  Added settings-route preference control coverage in `examples/86-atlas-commerce-os/shared/atlas/preference_controls_test.go`, including theme/locale/density/default-warehouse controls and the saved-view listbox workflow.
- [x] Add component tests for activity timeline and audit rows.
  Validate event grouping, status chips, timestamps, and note rendering.
  Added timeline and audit-row coverage in `examples/86-atlas-commerce-os/shared/atlas/activity_timeline_test.go`, including dashboard activity grouping/empty state and threshold-history timestamp plus note rendering.

### Integration test ideas

- [x] Add integration tests for public browsing flow.
  Exercise landing to catalog to product route transitions with SSR and hydration continuity.
  Added sequential public-flow coverage in `examples/86-atlas-commerce-os/server/integration_public_flow_test.go` to assert landing, catalog-query, and product SSR responses keep bootstrap continuity for hydration.
- [x] Add integration tests for public form submissions.
  Cover comment submission, restock request, and quote request server round-trips including validation errors.
  Existing integration coverage in `examples/86-atlas-commerce-os/server/server_test.go` (`TestPublicMutationValidationReturnsFieldErrors` and `TestAdditionalMutationSuccessPaths`) now maps directly to this requirement.
- [x] Add integration tests for inventory management flow.
  Cover loading inventory data, applying filters, saving views, editing thresholds, and revalidating list state.
  Added end-to-end inventory workspace coverage in `examples/86-atlas-commerce-os/server/integration_inventory_flow_test.go`, including filtered inventory loads, saved-view creation, threshold updates, and post-mutation list and history checks.
- [x] Add integration tests for transfer creation flow.
  Cover derived recommendation display, modal submission, persistence, and queue refresh.
  Added transfer-flow integration coverage in `examples/86-atlas-commerce-os/server/integration_transfer_flow_test.go`, including recommendation endpoint checks, transfer submission, persisted queue growth, and refreshed transfer list reads.
- [ ] Add integration tests for receiving reconciliation flow.
  Cover loading a session, entering actual quantities, handling discrepancy branches, and finalizing reconciliation.
- [ ] Add integration tests for moderation flow.
  Cover pending comments entering the queue, moderation actions, and public visibility changes.
- [ ] Add integration tests for theme and locale persistence.
  Cover first paint, hydration, preference save, reload, and SSR-stable resume behavior.

### Manual Playwright and exploratory browser tests

- [x] Add manual Playwright smoke scripts for public routes.
  Script realistic review flows for landing, catalog filtering, product detail hydration, comment submission, and warehouse availability pages.
- [x] Add manual Playwright smoke scripts for internal routes.
  Script realistic review flows for dashboard navigation, inventory filtering, transfer creation, receiving, moderation, and settings changes.
- [x] Add manual Playwright scripts for overlay and focus behavior.
  Verify stacked modal escape handling, focus restore, anchored menus, command palette behavior, and side-sheet layering.
- [ ] Add manual Playwright scripts for SSR and metadata checks.
  Validate titles, descriptions, canonical tags, structured data, and hydration-safe route entry on public pages.
- [x] Add manual Playwright scripts for theme and locale checks.
  Verify no theme flash, stable localized first paint, RTL layout, and restored preferences after reload.
- [x] Add manual Playwright scripts for table interactions.
  Validate sort order changes, debounced search, URL-driven filter updates, and saved view restore behavior.

### Performance and polish validation

- [x] Define performance checkpoints.
  Measure route entry, hydration cost, table interaction latency, and overlay responsiveness.
- [ ] Add cross-browser smoke coverage.
  Validate the key public and internal flows in Chromium, Firefox, and WebKit so the flagship example does not look tuned to one engine only.
- [x] Add visual review artifacts.
  Capture reference screenshots or a simple review pack for the landing page, product page, inventory table, overlays, and dark-mode states.
- [x] Add manual demo checklist.
  Create a concise walkthrough of the top signature flows for review and regression checks.

## 19. Code Comments And Debug Logging Guidance

### Useful code comments

- [ ] Add comments at architecture boundaries.
  Add brief comments where SSR handlers, bootstrap serialization, route-data loading, and hydration assumptions are non-obvious.
- [ ] Add comments around tricky query parsing.
  Explain any canonicalization, default sort rules, or URL-to-filter translation that would otherwise be hard to infer.
- [ ] Add comments around derived business rules.
  Explain transfer recommendation logic, availability messaging, discrepancy classification, and moderation visibility rules where the logic is subtle.
- [ ] Add comments around overlay coordination.
  Explain any stacked modal, sheet, or command palette interactions that depend on shared focus or dismissal rules.
- [ ] Add comments around theme and locale resume behavior.
  Explain how SSR and hydration avoid visible mismatches for theme and locale.
- [ ] Add comments around seed-data assumptions.
  Clarify why particular demo data patterns exist when they support testability or believable workflows.

### Useful console and server logs

- [ ] Add structured server logs for page entry.
  Log SSR route resolution, handler timing, and major route-data loading outcomes in a concise structured form.
- [ ] Add structured server logs for mutations.
  Log comment submission, quote request creation, stock adjustments, transfers, receiving reconciliation, and moderation actions with identifiers and result states.
- [ ] Add debug-only client logs for hydration issues.
  Log route bootstrap read failures, theme or locale mismatch detection, and hydration fallback conditions only in debug-friendly builds.
- [ ] Add debug logs for list query state.
  Surface parsed filters, sorts, and saved-view application when debugging table behavior.
- [ ] Add debug logs for overlay-heavy workflows.
  Log modal open, dismiss, nested discrepancy dialog activation, and focus-routing issues in development mode when needed.
- [ ] Add debug logs for async revalidation.
  Log loader start, loader completion, mutation-triggered revalidation, and stale-data refresh behavior to support integration debugging.
- [ ] Add logging guardrails.
  Ensure logs avoid noisy repetition, do not leak sensitive data, and can be disabled cleanly for polished demo use.

## 20. Stretch Goals

- [ ] Add a visual warehouse map.
  Show transfers and regional pressure graphically if scope allows.
- [ ] Add richer analytics panels.
  Include trends for sell-through, stockouts, and fulfillment speed.
- [ ] Add media upload or attachment workflows.
  Include supporting documents or receiving evidence if file handling becomes desirable.
- [ ] Add role-switching demo modes.
  Let reviewers inspect how the UI changes for customer, buyer, and warehouse staff.
- [ ] Add a guided demo mode.
  Provide seeded walkthrough prompts for the most important product flows.

## 21. Review Questions

Use these questions before implementation starts:

- [x] Is the public versus internal split correct?
- [x] Are the signature flows the right ones?
- [x] Is the SQLite scope realistic for an example project?
- [x] Are comments, reviews, and requests all necessary, or should one be cut?
- [x] Is purchase-order depth necessary for v1, or should transfers and receiving lead first?
- [x] Should locale-prefixed routes apply only to public pages?
- [x] Should the first release include auth, or mock role state locally?
- [x] Is the visual ambition high enough to feel like a flagship example?

## 22. Additional Backlog Expansion

### Preferences and resume state

- [x] 01. Persist theme in browser storage.
- [x] 02. Persist locale in browser storage.
- [x] 03. Persist density in browser storage.
- [x] 04. Persist default warehouse in browser storage.
- [x] 05. Reflect persisted theme on the document element.
- [x] 06. Reflect persisted locale and direction on the document element.
- [x] 07. Add a reset-to-default preferences action.
- [x] 08. Add a persisted preference summary banner.
- [x] 09. Add browser-storage fallback messaging for unavailable storage.
- [x] 10. Add a debug panel note for preference resume state.

### Inventory resume and saved views

- [x] 11. Persist inventory saved-view selection in browser storage.
- [x] 12. Persist inventory warehouse filter in browser storage.
- [x] 13. Persist inventory sort selection.
- [x] 14. Persist inventory search query.
- [x] 15. Persist inventory density override separately from global settings.
- [x] 16. Add a clear inventory resume action.
- [x] 17. Add a direct inventory route-entry resume test.
- [x] 18. Add saved-view resume copy in the inventory panel.
- [x] 19. Add saved-view resume notes to manual testing guidance.
- [x] 20. Add a dedicated inventory resume QA checklist.

### Direct route entry and recovery

- [x] 21. Add a direct settings route-entry browser test.
- [x] 22. Add a direct inventory route-entry browser test.
- [x] 23. Add a catch-all recovery browser test.
- [x] 24. Verify direct route entry keeps persisted theme.
- [x] 25. Verify direct route entry keeps persisted locale.
- [x] 26. Verify direct route entry keeps persisted density.
- [x] 27. Verify direct route entry keeps persisted default warehouse.
- [x] 28. Verify direct route entry keeps persisted inventory saved view.
- [x] 29. Add a cold-entry test for the SKU detail route.
- [x] 30. Add a cold-entry test for the transfers route.

### Internal workflow next steps

- [x] 31. Add a threshold edit stub control on SKU detail.
- [x] 32. Add a transfer confirmation summary state.
- [x] 33. Add a receiving discrepancy detail state.
- [x] 34. Add a moderation decision confirmation state.
- [x] 35. Add saved-view status messaging in the internal shell.
- [x] 36. Add inventory warehouse badge treatment.
- [x] 37. Add a dashboard quick-link to transfers.
- [x] 38. Add a dashboard quick-link to receiving.
- [x] 39. Add comments-route record detail variants.
- [x] 40. Add settings-surface success feedback.

### Accessibility and keyboard

- [x] 41. Add a settings keyboard smoke spec.
- [x] 42. Add RTL keyboard coverage for the locale switcher.
- [x] 43. Add focus-order audit notes for settings.
- [x] 44. Add keyboard smoke for inventory saved-view controls.
- [x] 45. Add keyboard smoke for transfer state buttons.
- [x] 46. Add keyboard smoke for receiving state buttons.
- [x] 47. Add keyboard smoke for moderation filter buttons.
- [x] 48. Add route-heading focus behavior.
- [x] 49. Add live-region route announcements.
- [x] 50. Add reduced-motion browser coverage.

### Browser QA and route coverage

- [x] 51. Add a consolidated Atlas Playwright command to the README.
- [x] 52. Run the three-spec Atlas browser suite together.
- [x] 53. Add a browser assertion for the HTML theme class.
- [x] 54. Add a browser assertion for the HTML locale attribute.
- [x] 55. Add a browser assertion for the HTML density class or attribute.
- [x] 56. Add a browser assertion for the HTML warehouse attribute.
- [x] 57. Add a browser assertion for the HTML direction attribute.
- [x] 58. Add a browser assertion for reload-persisted locale.
- [x] 59. Add a browser assertion for reload-persisted density.
- [x] 60. Add a browser assertion for reload-persisted saved view.

### Bootstrap and test helpers

- [x] 61. Add an Atlas supported-locales helper.
- [x] 62. Add an Atlas locale-direction helper.
- [x] 63. Add a default preferences bootstrap helper.
- [x] 64. Add a default theme bootstrap helper.
- [x] 65. Add a default i18n bootstrap helper.
- [x] 66. Add a route bootstrap lookup helper.
- [x] 67. Add bootstrap-default unit tests.
- [x] 68. Add route-bootstrap lookup unit tests.
- [x] 69. Add metadata coverage tests for every manifest route.
- [x] 70. Add manifest uniqueness and surface consistency tests.

### Documentation and reviewer guidance

- [x] 71. Document the broader Atlas Playwright command.
- [x] 72. Document client-side preference resume in the README.
- [x] 73. Document the accessibility spot checklist.
- [x] 74. Document screenshot notes for Atlas review.
- [x] 75. Document inventory saved-view resume behavior.
- [x] 76. Add a persisted-settings reviewer checklist.
- [x] 77. Add a direct-entry QA checklist.
- [x] 78. Add an RTL verification note for reviewers.
- [x] 79. Add a known-limitations note for non-SSR preference resume.
- [x] 80. Add a milestone summary note for the current Atlas pass.

### Visual and copy polish

- [x] 81. Add compact versus comfortable density styling differences.
- [x] 82. Format persisted warehouse labels for friendlier UI copy.
- [x] 83. Polish saved-view active-state styling.
- [x] 84. Polish settings control grouping hierarchy.
- [x] 85. Polish inventory summary copy.
- [x] 86. Polish settings helper copy around persistence.
- [x] 87. Polish the route-not-found recovery panel.
- [x] 88. Polish dashboard-to-inventory handoff copy.
- [x] 89. Run an internal panel spacing audit.
- [x] 90. Run a public-versus-internal contrast audit.

### Release and milestone prep

- [x] 91. Add a launcher-owned command for the Atlas smoke suite.
- [x] 92. Add an Atlas release-readiness checklist.
- [x] 93. Add an Atlas regression checklist.
- [x] 94. Add a browser-matrix plan for Atlas.
- [x] 95. Add screenshot capture naming rules.
- [x] 96. Add reviewer demo steps for persisted settings.
- [x] 97. Add reviewer demo steps for direct route entry.
- [x] 98. Add reviewer demo steps for saved-view resume.
- [x] 99. Add inventory persistence test notes.
- [x] 100. Add a milestone-four cleanup sweep.

## 23. Additional Backlog Expansion II

### Premium storefront polish

- [x] 01. Rebuild the product hero as a premium split-layout surface.
- [x] 02. Add finish-selection controls to the product hero.
- [x] 03. Add a warehouse-promise card to the product hero.
- [x] 04. Add launch-copy guidance chips to the product hero.
- [x] 05. Add an editorial media gallery stub.
- [x] 06. Add a finish-specific specification summary.
- [x] 07. Add a route-aware CTA cluster for quote, request, and availability.
- [x] 08. Add a comparison strip for adjacent product variants.
- [x] 09. Add trust-copy around returns and installation support.
- [x] 10. Add a responsive stacked-mobile hero variant.

### Dashboard triage and handoff

- [x] 11. Replace the minimal dashboard stats with triage cards.
- [x] 12. Add a quick-handoff rail from dashboard to action routes.
- [x] 13. Add dashboard copy that pushes operators into concrete workspaces.
- [x] 14. Add severity ordering across dashboard cards.
- [x] 15. Add warehouse-specific alert segmentation.
- [x] 16. Add a next-shift staffing note.
- [x] 17. Add an inbound shipment ETA strip.
- [x] 18. Add dashboard keyboard shortcuts copy.
- [x] 19. Add route-aware alert totals.
- [x] 20. Add a compact dashboard mobile arrangement.

### Comment and moderation polish

- [x] 21. Replace the comment thread bullet list with believable threaded cards.
- [x] 22. Add staff-reply styling in the comment thread.
- [x] 23. Add a moderation-expectations side panel to the comment thread.
- [x] 24. Add product-context labels per comment.
- [x] 25. Add timestamps and freshness hints.
- [x] 26. Add a pending-comment placeholder treatment.
- [x] 27. Add a comment escalation badge pattern.
- [x] 28. Add a route to the internal moderation queue from the public thread.
- [x] 29. Add a seller-response draft preview.
- [x] 30. Add an empty-thread state for new launches.

### Settings and resume UX

- [x] 31. Add a workspace-defaults summary banner in settings.
- [x] 32. Add inline success feedback after workspace preference changes.
- [x] 33. Add friendlier current-warehouse copy to settings.
- [x] 34. Add a reset-to-default workspace action.
- [x] 35. Add a saved-view reset control.
- [x] 36. Add storage-unavailable fallback copy.
- [x] 37. Add a reviewer-facing persistence explainer block.
- [x] 38. Add per-control helper copy for locale and theme parity.
- [x] 39. Add a settings route success icon treatment.
- [x] 40. Add a settings density preview comparison strip.

### Inventory workspace polish

- [x] 41. Add saved-view and warehouse badges above inventory controls.
- [x] 42. Add friendlier warehouse labels in inventory summaries.
- [x] 43. Add richer persistence copy to the inventory surface.
- [x] 44. Add inventory sort persistence.
- [x] 45. Add inventory query persistence.
- [x] 46. Add a clear-resume action for inventory workspace state.
- [x] 47. Add badge tones tied to saved-view urgency.
- [x] 48. Add query-empty messaging in the search preview.
- [x] 49. Add keyboard shortcut hints for switching saved views.
- [x] 50. Add a compact-density inventory presentation mode.

### Shell and route framing

- [x] 51. Add a saved-workspace note to the internal shell.
- [x] 52. Add richer public-shell brand framing.
- [x] 53. Add richer internal-shell operator framing.
- [x] 54. Rebuild the route catch-all as a recovery panel.
- [x] 55. Add route-local breadcrumb stubs.
- [x] 56. Add shell-level heading subtitles by route.
- [x] 57. Add surface-specific empty-state messaging.
- [x] 58. Add a shell-level command-bar placeholder.
- [x] 59. Add a shell-level alert summary pill.
- [x] 60. Add a mobile shell rail collapse treatment.

### Visual system and theming

- [x] 61. Add true visual density differences for compact and comfortable modes.
- [x] 62. Add reduced-motion visual variants for Atlas cards.
- [x] 63. Add a public-versus-internal contrast audit note.
- [x] 64. Add storefront accent color balancing for light mode.
- [x] 65. Add internal accent color balancing for dark mode.
- [x] 66. Add card hover motion rules for keyboard parity.
- [x] 67. Add tokenized shadows for primary versus secondary surfaces.
- [x] 68. Add typography tuning for Arabic headings.
- [x] 69. Add form focus-ring consistency rules.
- [x] 70. Add visual test screenshots for both themes.

### Framework coverage follow-through

- [x] 71. Add query-state routing to the catalog screen.
- [x] 72. Add route-params rendering to product or warehouse detail routes.
- [x] 73. Add a redirect or guard flow for the internal shell.
- [x] 74. Add nested layouts for the internal route family.
- [x] 75. Add an overlay-backed action surface.
- [x] 76. Add portal-driven secondary UI.
- [x] 77. Add shared atoms for preferences or workspace state.
- [x] 78. Add computed or derived state for stock summaries.
- [x] 79. Add async resource loading to a real Atlas screen.
- [x] 80. Add a devtools diagnostics surface.

### Reviewer and QA follow-through

- [x] 81. Add a persisted-settings reviewer checklist.
- [x] 82. Add a direct-entry QA checklist.
- [x] 83. Add an RTL verification note for reviewers.
- [x] 84. Add an inventory resume QA checklist.
- [x] 85. Add screenshot naming guidance for Atlas capture.
- [x] 86. Add route-entry demo steps for reviewers.
- [x] 87. Add a comment-thread moderation QA note.
- [x] 88. Add a dashboard handoff QA note.
- [x] 89. Add a light-mode review checklist.
- [x] 90. Add a dark-mode review checklist.

### Packaging and release prep

- [x] 91. Add a launcher-owned command dedicated to Atlas smoke coverage.
- [x] 92. Add an Atlas release-readiness checklist.
- [x] 93. Add an Atlas regression checklist.
- [x] 94. Add a browser-matrix plan for Atlas.
- [x] 95. Add screenshot capture naming rules.
- [x] 96. Add persisted-settings demo steps.
- [x] 97. Add direct-route-entry demo steps.
- [x] 98. Add saved-view resume demo steps.
- [x] 99. Add inventory persistence test notes.
- [x] 100. Add a milestone-five cleanup sweep.

## 24. Additional Backlog Expansion III

### Nested routing and shell follow-through

- [x] 101. Add a nested receiving layout for session drill-in routes.
- [x] 102. Add a nested transfers layout for transfer-detail drill-in routes.
- [ ] 103. Add a nested comments layout for moderation filters and record detail.
- [x] 104. Add route-local breadcrumbs inside nested internal layouts.
- [x] 105. Add layout-level revalidation affordances for internal workspaces.
- [x] 106. Add route-aware internal section metadata beyond the page title.
- [ ] 107. Add warehouse-detail nested layouts under the internal app shell.
- [ ] 108. Add purchase-order nested layouts under the internal app shell.
- [ ] 109. Add settings sub-routes for appearance, locale, and workspace defaults.
- [x] 110. Add route tests that prove nested shells stay mounted across child navigation.

### Overlay and portal implementation

- [x] 111. Replace the SKU threshold stub with a real accessible overlay.
- [x] 112. Add a transfer confirmation dialog mounted through ui.Portal.
- [x] 113. Add a receiving discrepancy side sheet.
- [x] 114. Add a moderation decision confirmation dialog.
- [x] 115. Add a lightweight toast or inline notice portal root for internal actions.
- [x] 116. Add focus-return coverage for the first Atlas overlay flow.
- [x] 117. Add escape-key dismissal coverage for the first Atlas overlay flow.
- [x] 118. Add overlay stacking rules documentation for Atlas.
- [x] 119. Add a portal host contract note to the Atlas HTML shell.
- [x] 120. Add browser tests for overlay-open, action, and dismiss flows.

### Shared state and persistence follow-through

- [x] 121. Move theme, locale, density, and warehouse persistence into snapshot-friendly state helpers.
- [x] 122. Add shared atom state for transfer workflow filters.
- [x] 123. Add shared atom state for moderation queue filters.
- [x] 124. Add shared atom state for receiving workflow status.
- [x] 125. Add derived stock pressure labels keyed by warehouse and saved view.
- [x] 126. Add derived recommendation text for transfer planning.
- [x] 127. Add snapshot export of Atlas preference atoms for diagnostics.
- [x] 128. Add snapshot restore of Atlas preference atoms for reviewer resets.
- [x] 129. Add a debug-facing summary of current shared Atlas atoms.
- [x] 130. Add tests that prove shared state survives route transitions inside the internal shell.

### Async data and revalidation

- [x] 131. Add a route loader for the dashboard surface.
- [x] 132. Add a route loader for inventory workspace data.
- [x] 133. Add a route loader for SKU detail data.
- [x] 134. Add a route loader for transfer planning data.
- [x] 135. Add a route loader for receiving queue data.
- [x] 136. Add a loading fallback for the first Atlas route loader.
- [x] 137. Add an error fallback for the first Atlas route loader.
- [x] 138. Add manual revalidation control to at least one internal route.
- [x] 139. Add query-aware revalidation for inventory search changes.
- [x] 140. Add browser coverage for loader success, loading, and retry states.

### Warehouse and inventory surface depth

- [x] 141. Add the internal warehouse list route.
- [x] 142. Add the internal warehouse detail route.
- [x] 143. Add warehouse pressure comparison cards.
- [x] 144. Add warehouse staffing and backlog summary rows.
- [x] 145. Add inventory sort persistence.
- [x] 146. Add inventory density override separate from global density.
- [x] 147. Add compact-density inventory row treatment.
- [x] 148. Add keyboard shortcut hints for saved-view switching.
- [x] 149. Add a real threshold history timeline.
- [x] 150. Add inventory browser coverage for sort and density persistence.

### Transfers, receiving, and purchasing

- [x] 151. Add a transfer detail route.
- [x] 152. Add transfer approval and cancel action states.
- [x] 153. Add a purchase-order list route.
- [x] 154. Add a purchase-order detail route.
- [x] 155. Add a receiving-session detail route.
- [x] 156. Add discrepancy classification controls beyond static copy.
- [x] 157. Add purchase-order approval copy and action affordances.
- [x] 158. Add inbound shipment rows tied to real route data.
- [x] 159. Add an activity timeline shared between transfers and receiving.
- [x] 160. Add browser coverage for transfer-detail and receiving-session entry.

### Public commerce and warehouse expansion

- [x] 161. Add a warehouse detail public route.
- [x] 162. Add a warehouse-specific availability route.
- [x] 163. Add public catalog sort state.
- [x] 164. Add catalog pagination state.
- [x] 165. Add related products on the product route.
- [x] 166. Add richer warehouse-aware promise messaging on product detail.
- [x] 167. Add a public comments thread structure decision note to the implementation backlog.
- [x] 168. Add form validation states for quote, comment, and restock requests.
- [x] 169. Add browser coverage for warehouse route entry and recovery.
- [x] 170. Add screenshots for public storefront and warehouse surfaces in both themes.

### Devtools and diagnostics

- [x] 171. Add a devtools panel surface to Atlas.
- [x] 172. Add snapshot-now diagnostics for Atlas shared state.
- [x] 173. Add route inspection diagnostics to the internal shell.
- [x] 174. Add loader timing diagnostics once Atlas loaders exist.
- [x] 175. Add a developer-only panel for current preference and route state.
- [x] 176. Add a developer-only panel for current inventory derived summaries.
- [x] 177. Add a diagnostics toggle that is hidden from normal reviewer flows.
- [x] 178. Add tests that prove diagnostics can mount without breaking the shell.
- [x] 179. Add documentation for Atlas diagnostics usage.
- [ ] 180. Add a cleanup pass to remove temporary diagnostics before final release signoff.

### Visual system and interaction polish

- [x] 181. Add tokenized shadow tiers for nested route surfaces.
- [x] 182. Add public light-mode accent balancing.
- [x] 183. Add internal dark-mode accent balancing.
- [x] 184. Add hover motion parity for keyboard-visible controls.
- [x] 185. Add Arabic heading spacing and weight tuning.
- [x] 186. Add contrast audit notes for public versus internal surfaces.
- [x] 187. Add motion guidance for nested-route and overlay transitions.
- [x] 188. Add screenshot coverage for compact and comfortable density modes.
- [x] 189. Add a polish sweep for mobile internal rails after nested layouts settle.
- [x] 190. Add a visual cleanup sweep for milestone six.

### Documentation, QA, and release follow-through

- [x] 191. Add a manual test checklist for nested route shells.
- [x] 192. Add a manual test checklist for shared-state persistence across routes.
- [x] 193. Add a manual test checklist for overlay-backed workflows.
- [x] 194. Add a reviewer guide for derived inventory summaries.
- [x] 195. Add browser smoke commands for the next Atlas milestone.
- [x] 196. Add an integration-plan note for future Atlas server endpoints.
- [x] 197. Add a release-readiness section for loaders and overlays.
- [x] 198. Add a regression note for route-entry plus persistence combinations.
- [x] 199. Add a repo-level memory note summarizing Atlas route and state conventions.
- [x] 200. Add a milestone-six cleanup sweep.
