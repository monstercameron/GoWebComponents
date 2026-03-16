# Atlas Commerce OS Project Todo

This document turns the flagship example concept into a detailed implementation backlog.

Project intent:

- Build a full-stack showcase application for GoWebComponents.
- Demonstrate public SSR and SEO, client hydration, internal operations tooling, SQLite persistence, advanced UI state, overlays, routing, forms, accessibility, theming, and localization.
- Keep the implementation coherent as one product instead of a feature collage.

Working product framing:

- Public side: brand-forward sales pages, product availability, warehouse-aware delivery messaging, comments and requests.
- Internal side: inventory control, warehouse operations, receiving, transfer planning, purchase workflows, moderation, and settings.

## 1. Product Definition

### Core vision

- [ ] Lock the product narrative.
  Finalize the one-sentence description of Atlas Commerce OS so the app reads as one believable commerce-plus-operations platform rather than a generic dashboard.
- [ ] Define the public versus internal surface boundary.
  Decide which routes are openly crawlable, which require authentication, and which features exist in both contexts with different presentation.
- [ ] Define the primary user roles.
  Document at least customer, warehouse operator, inventory manager, and buyer workflows so screen scope stays grounded in actual user jobs.
- [ ] Define the top five signature flows.
  Pick the must-demo flows that the implementation will optimize for, such as viewing a product page, submitting a restock request, resolving low stock, receiving inventory, and approving a transfer.
- [ ] Define MVP versus stretch features.
  Separate required capabilities from polish ideas so the project can ship in phases without losing direction.

### Success criteria

- [ ] Define the framework showcase goals.
  Explicitly list which GoWebComponents capabilities the example must prove in a realistic way, including SSR, hydration, routing, overlays, forms, state, i18n, and diagnostics.
- [ ] Define the product quality bar.
  Set expectations for realism, visual polish, responsiveness, accessibility, route depth, persistence, and browser behavior.
- [ ] Define the review checklist.
  Create a concise acceptance checklist covering public pages, internal workflows, persistence, a11y, hydration integrity, and visual consistency.

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

- [ ] Define the public route tree.
  Specify routes for landing page, collection pages, product detail pages, warehouse pages, warehouse-specific availability pages, and any comments or request flows that deserve deep links.
- [ ] Lock the public page inventory.
  Confirm the baseline public pages are landing, catalog, product detail, warehouses index, warehouse detail, and warehouse-specific availability pages.
- [ ] Define the sales landing page structure.
  Decide hero content, featured categories, proof points, warehouse fulfillment messaging, and call-to-action placement.
- [ ] Define the landing page modules in detail.
  Include recommended modules for hero, regional fulfillment proof, featured products, category cards, customer proof, and operator-facing product narrative.
- [ ] Define the collection or catalog route behavior.
  Decide sorting, filtering, pagination, search, query parameter behavior, canonicalization rules, and whether filter states are indexable.
- [ ] Define catalog query parameters.
  Standardize `q`, `category`, `warehouse`, `availability`, `sort`, `page`, and optional price or tag filters so server rendering and client routing stay aligned.
- [ ] Define the product sales page structure.
  Lay out media, product copy, price, inventory promise, specs, reviews or comments, related products, and lead-capture or request forms.
- [ ] Define the product page tab model.
  Decide whether detail, specs, shipping, comments, and reviews are tabbed, section-based, or route-fragment based.
- [ ] Define warehouse public pages.
  Decide what a warehouse page exposes publicly, such as service region, fulfillment speed, stocked highlights, and SKU-specific availability.
- [ ] Define warehouse-specific availability page behavior.
  Decide whether the page focuses on promise messaging, raw stock counts, nearby alternatives, or a hybrid model.

### Internal routes

- [ ] Define the authenticated app shell.
  Decide top navigation, side navigation, command palette entry, alert center, settings access, and responsive collapse behavior.
- [ ] Define the primary internal navigation groups.
  Group the internal shell into overview, inventory, warehouses, purchasing, transfers, receiving, comments, and settings.
- [ ] Define the internal route tree.
  Specify routes for dashboard, inventory list, SKU detail, warehouse list, warehouse detail, purchase orders, transfers, receiving, moderation, and settings.
- [ ] Define route-level tabs for detail pages.
  Decide whether SKU, warehouse, purchase order, and transfer detail routes use internal tab state for overview, activity, notes, comments, and analytics.
- [ ] Define nested route structure.
  Decide where nested layouts and outlets apply so related views share shell UI, filters, and context cleanly.
- [ ] Define overlay-versus-route responsibility.
  Decide which detail experiences should remain true routes and which should open as overlay-backed secondary surfaces while preserving the current route.
- [ ] Define deep-link behavior.
  Ensure major workflows can be opened directly from URLs, including filtered lists, selected tabs, and detail views.

### Route metadata and navigation

- [ ] Define metadata rules per route.
  Specify titles, descriptions, canonical URLs, structured data expectations, and which routes should emit social metadata.
- [ ] Define navigation state persistence.
  Decide which filters, tabs, and selection states live in the URL versus local state versus persisted preferences.
- [ ] Define access guard behavior.
  Decide how internal routes enforce authentication and how unsaved form workflows block navigation.
- [ ] Define page ownership matrix.
  Record for each route whether it is SSR-only, SSR+hydrate, internal-only, public SEO-sensitive, or modal-triggering.

## 3. Visual Direction And Design System

### Brand and visual language

- [ ] Define the visual identity.
  Choose the visual direction for Atlas Commerce OS, including mood, contrast level, shape language, gradients, and how public and internal surfaces differ.
- [ ] Define typography choices.
  Choose display, heading, and dense UI text styles that feel intentional and non-generic.
- [ ] Define color tokens.
  Establish semantic colors for stock health, warehouse states, alerts, forms, moderation states, and marketing accents.
- [ ] Define spacing, radius, and elevation tokens.
  Create consistent primitives for cards, tables, overlays, forms, and hero sections.

### Theme system

- [ ] Define light and dark mode behavior.
  Decide default theme, persisted preference behavior, SSR-safe initial theme resolution, and whether public and internal surfaces prefer different defaults.
- [ ] Define density modes.
  Decide whether internal lists support compact and comfortable display settings.
- [ ] Define motion principles.
  Standardize page transitions, overlay entrance and exit motion, table feedback, and reduced-motion fallbacks so the example feels deliberate rather than generic.
- [ ] Define responsive behavior.
  Specify how public pages and internal tools adapt across mobile, tablet, and desktop without collapsing into generic stacked cards.

### Component vocabulary

- [ ] Define shared UI primitives for the example.
  Identify repeated components such as stat cards, filter chips, list headers, badge systems, segmented controls, timeline rows, and detail panels.
- [ ] Define data table patterns.
  Decide list header layout, sort affordances, filter controls, sticky columns, row selection, bulk action affordances, and empty-state treatment.
- [ ] Define overlay patterns.
  Decide when to use centered modals, side sheets, anchored menus, tooltips, stacked dialogs, and command palette overlays.
- [ ] Define loading, empty, and error states.
  Standardize skeletons, no-results states, empty-first-use states, and retry surfaces for catalog pages, internal tables, detail views, and form-heavy workflows.

## 4. Data Model And SQLite Schema

### Core entities

- [ ] Define product entities.
  Specify fields for SKU, slug, title, category, pricing, summary, long description, media, SEO metadata, and active status.
- [ ] Define warehouse entities.
  Specify fields for warehouse identity, location, service region, fulfillment SLA, capacity, and public visibility.
- [ ] Define inventory level entities.
  Specify on-hand, reserved, available, inbound, damaged, safety stock, reorder point, and last-updated fields.
- [ ] Define purchase order entities.
  Specify purchase order headers, status, vendor details, ETA, line items, and approval state.
- [ ] Define transfer request entities.
  Specify source warehouse, destination warehouse, quantities, reason, priority, approval status, and shipment state.
- [ ] Define receiving session entities.
  Specify expected lines, actual lines, discrepancy notes, receiver identity, timestamps, and final reconciliation state.
- [ ] Define comment or review entities.
  Decide whether customer comments, product questions, and internal notes share a model or use separate tables.
- [ ] Define audit event entities.
  Store inventory changes, form submissions, approvals, moderation actions, and transfer lifecycle events in a timeline-friendly shape.
- [ ] Define user and preference entities.
  Store theme, locale, density, default warehouse, saved views, and role data.

### Relational design

- [ ] Define foreign-key relationships.
  Map product-to-inventory, product-to-comments, warehouse-to-inventory, order-to-lines, and transfer-to-audit relationships cleanly.
- [ ] Define indexing strategy.
  Add indexes for slug lookups, SKU lookups, warehouse-specific stock queries, moderation queues, and saved-view retrieval.
- [ ] Define seed data strategy.
  Create realistic demo data volume and content so sorting, filtering, comments, and stock scenarios feel credible.

### Persistence and migrations

- [ ] Define schema migration approach.
  Decide how the example initializes and evolves the SQLite schema reliably.
- [ ] Define development reset tooling.
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

- [ ] Define the server entrypoint.
  Decide how the Go server boots SQLite, serves static assets, performs SSR, and exposes JSON or form endpoints.
- [ ] Define the concrete page-rendering handlers.
  Map each public and internal route to a server handler that can load data, emit metadata, and serialize bootstrap payloads consistently.
- [ ] Define route handling split.
  Decide which requests return SSR HTML, which return JSON, and which process form submissions directly.
- [ ] Define the SSR route handler contract.
  Standardize what each route handler returns: route data, metadata, bootstrap payload, auth context, theme, and locale.
- [ ] Define error handling strategy.
  Standardize 404, validation, authentication, and server error behavior for public and internal surfaces.

### API and form endpoints

- [ ] Define public interaction endpoints.
  Include comment submission, quote requests, restock requests, and any contact or lead forms.
- [ ] Define the public read endpoints.
  Decide whether product pages, warehouse highlights, and comments need explicit JSON endpoints in addition to SSR loaders.
- [ ] Define inventory mutation endpoints.
  Include stock adjustments, transfer creation, purchase order updates, receiving reconciliation, and moderation actions.
- [ ] Define the internal read endpoints.
  Standardize list and detail APIs for inventory, warehouses, purchase orders, transfers, receiving sessions, and moderation queues.
- [ ] Define list query endpoints.
  Support catalog filters, internal inventory filters, moderation lists, and saved views in a consistent query format.
- [ ] Define endpoint naming conventions.
  Keep endpoints predictable by grouping them under `/api/public/...`, `/api/app/...`, or another consistent scheme.
- [ ] Define server validation rules.
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

- [ ] Define auth and session strategy for the example.
  Decide how internal routes are protected without overcomplicating the showcase.
- [ ] Define CSRF strategy for server-backed forms.
  Choose how the example will demonstrate secure posting for public and internal forms.
- [ ] Define file upload needs.
  Decide whether the example should include product media uploads or receiving attachments, and if so how they are handled.
- [ ] Define static asset strategy.
  Decide how product media, brand graphics, Open Graph images, cache headers, and fallback assets are served so the demo feels production-shaped.

## 6. SSR, Hydration, And SEO

### SSR model

- [ ] Define which routes are SSR-rendered.
  Identify the public pages and internal routes that should render meaningful HTML before wasm startup.
- [ ] Lock the SSR page set.
  Treat landing, catalog, product detail, warehouse detail, warehouse-specific availability, dashboard, inventory list, and SKU detail as the default SSR routes.
- [ ] Define hydration boundaries.
  Decide where full hydration is required and whether any page sections remain static after SSR.
- [ ] Define page-by-page hydration goals.
  Decide whether comments, filters, overlays, and internal tables hydrate immediately or progressively.
- [ ] Define bootstrap payload shape.
  Decide what route data, preferences, locale, theme, and user state transfer from server to client.
- [ ] Define a page bootstrap contract.
  Standardize route payload sections for `route`, `data`, `i18n`, `theme`, `user`, and any saved-view state.
- [ ] Define SSR-to-client consistency rules.
  Ensure theme, locale, route data, and metadata do not mismatch during hydration.

### SEO surface

- [ ] Define SEO targets for public routes.
  Decide which pages should rank or share well and what metadata or structured data they need.
- [ ] Define canonical URL policy.
  Decide how filtered catalog pages, warehouse pages, and comment pagination should normalize canonical links.
- [ ] Define structured data coverage.
  Add product and availability schema where it meaningfully improves search presentation.
- [ ] Define sitemap and social-preview coverage.
  Decide which public routes enter the sitemap, what robots behavior is expected, and how share cards or preview images are generated.
- [ ] Define content-indexing boundaries.
  Ensure internal app routes are not treated as public marketing pages.

## 7. Public Commerce Features

### Landing and discovery

- [ ] Define the marketing landing page modules.
  Decide hero content, metrics, warehouse network storytelling, featured products, and brand trust sections.
- [ ] Define the landing page CTA map.
  Decide primary and secondary calls to action, such as browse catalog, view warehouse network, request quote, or inspect availability.
- [ ] Define category and collection browsing.
  Decide sort modes, filter categories, availability-based filtering, and promotional callouts.
- [ ] Define the catalog page modules.
  Include results header, filter rail, sort control, product grid or list, saved filter state, and pagination behavior.

### Product sales pages

- [ ] Define product page content hierarchy.
  Decide how price, shipping promise, availability, specs, media, reviews, and forms are arranged.
- [ ] Define the exact sales page sections.
  Use a baseline structure of hero, purchase or inquiry block, availability by warehouse, specs, social proof, comments, and related products.
- [ ] Define inventory-aware sales messaging.
  Tie stock and ETA to warehouse data in a way that feels useful rather than fake.
- [ ] Define product route data requirements.
  Ensure the server loads product detail, warehouse availability, related products, approved comments, and form state defaults for SSR.
- [ ] Define related product recommendations.
  Decide simple recommendation logic and placement.
- [ ] Define low-stock and unavailable-product fallback UX.
  Decide what the user sees when a product is unavailable, including nearby warehouse options, substitute products, and restock or quote callouts.

### Comments, reviews, and public forms

- [ ] Define the public comment system.
  Decide whether the public surface supports comments, reviews, questions, or all three, and how they differ.
- [ ] Define the sales-page comments module.
  Decide whether comments live inline below the product, in tabbed sections, or in a secondary route-backed thread view.
- [ ] Define comment moderation rules.
  Decide approval states, visibility timing, internal review flow, and any abuse controls.
- [ ] Define restock request form behavior.
  Capture user intent when stock is low or unavailable and tie it back to internal demand views.
- [ ] Define quote request form behavior.
  Provide a bulk or business inquiry path that shows more advanced form handling than a simple contact form.
- [ ] Define public form success and error states.
  Ensure public forms feel polished, accessible, and server-backed.
- [ ] Define public endpoint payloads.
  Specify request and response shapes for comment submission, quote requests, and restock requests so form wiring stays consistent.

## 8. Warehouse Experience

### Public warehouse pages

- [ ] Define public warehouse overview pages.
  Decide how each warehouse is represented publicly, including service promise, region, and stocked highlights.
- [ ] Define the public warehouse page sections.
  Use a baseline of hero summary, service zone, featured inventory, fulfillment promise, and operational notices.
- [ ] Define SKU availability by warehouse.
  Show which locations can fulfill a SKU and what delivery promise they imply.

### Internal warehouse operations

- [ ] Define the internal warehouse list.
  Decide what summary metrics matter at the warehouse list level.
- [ ] Define the internal warehouse list columns and filters.
  Include location, service region, active SKUs, low-stock count, receiving backlog, transfer pressure, and fulfillment SLA.
- [ ] Define the internal warehouse detail page.
  Include backlog, low-stock items, receiving queue, transfer pressure, and staff notes.
- [ ] Define the warehouse detail sections.
  Split the page into overview, inventory health, inbound work, outbound pressure, notes, and activity history.
- [ ] Define warehouse comparison views.
  Decide whether the app compares stock health across locations and how that is visualized.

## 9. Inventory Management Features

### Inventory list and filters

- [ ] Define the main inventory table columns.
  Decide which fields matter most for operators and managers, such as SKU, title, warehouse, available stock, cover days, inbound, and status.
- [ ] Define sorting behavior.
  Support meaningful sorts including quantity, days of cover, revenue importance, and freshness of updates.
- [ ] Define filtering behavior.
  Support warehouse, category, supplier, stock health, availability state, moderation flags, and search.
- [ ] Define saved views.
  Let users save common filter or sort sets and restore them later.
- [ ] Define bulk actions.
  Decide whether users can bulk create transfers, adjust thresholds, or tag items for review.

### SKU detail workflows

- [ ] Define the SKU detail route layout.
  Include summary stats, warehouse breakdown, inbound shipments, recent activity, comments, and action surfaces.
- [ ] Define threshold editing.
  Allow reorder points and safety stock to be updated through a clean modal or side panel flow.
- [ ] Define stock adjustment behavior.
  Decide how manual corrections are entered, validated, and audited.
- [ ] Define internal comments or notes.
  Decide whether operators can leave notes on SKUs separately from public comments.

## 10. Purchasing, Transfers, And Receiving

### Purchase orders

- [ ] Define purchase order list and detail screens.
  Include status, vendor, ETA, line items, and approval actions.
- [ ] Define purchase order creation and approval flow.
  Show structured forms, server validation, and post-submit revalidation.

### Transfers

- [ ] Define transfer recommendation logic.
  Decide how the demo identifies stock imbalances and suggests cross-warehouse moves.
- [ ] Define transfer creation workflow.
  Support source and destination selection, quantity entry, reason capture, and confirmation.
- [ ] Define transfer lifecycle states.
  Model draft, submitted, approved, in transit, received, and cancelled behaviors.

### Receiving

- [ ] Define receiving queue and detail view.
  Show incoming purchase orders and transfers awaiting check-in.
- [ ] Define receiving form workflow.
  Capture expected versus actual units, discrepancy reasons, notes, and final confirmation.
- [ ] Define discrepancy resolution overlays.
  Use stacked modal or sheet flows for mismatch handling and audit capture.
- [ ] Define optimistic versus authoritative update behavior.
  Decide what updates appear immediately and what waits for server confirmation.

## 11. Forms And Validation

### Public forms

- [ ] Define common public form primitives.
  Standardize label, hint, error, success, pending, and confirmation behavior for comments, quote requests, and restock forms.
- [ ] Define moderation-safe comment submission UX.
  Tell the user whether comments publish immediately or await review.

### Internal forms

- [ ] Define internal form patterns.
  Standardize dense modal forms, sheet forms, inline forms, and page-level forms.
- [ ] Define validation projection rules.
  Ensure server field errors map cleanly back into the public form state shown in the UI.
- [ ] Define submit-intent behavior.
  Support actions like save draft, approve, reject, and receive with distinct button intent flows.

## 12. Overlays, Portals, And Interaction Layers

### Modal system

- [ ] Define overlay inventory.
  List every dialog, side sheet, anchored menu, tooltip, and command palette the example needs.
- [ ] Define stacking rules.
  Decide which flows can nest and how focus, escape handling, and outside-click behavior should work.
- [ ] Define side-panel strategy.
  Use sheets or drawers for SKU detail, warehouse quick views, and order summaries where preserving list context matters.
- [ ] Define destructive confirmation patterns.
  Standardize confirmation dialogs for cancel, delete, reject, and irreversible actions.
- [ ] Define notification and inline-feedback patterns.
  Standardize toast usage, inline banners, success confirmations, and long-running task feedback so mutations feel responsive without becoming noisy.

### Commanding and shortcuts

- [ ] Define the command palette.
  Support keyboard-first navigation to products, SKUs, warehouses, orders, and settings.
- [ ] Define anchored actions.
  Use menus and popovers for row-level quick actions without overloading table cells.

## 13. Theme, Preferences, And Localization

### Theme and preferences

- [ ] Define persisted preferences.
  Store theme, density, locale, default warehouse, and saved views per user.
- [ ] Define SSR-safe theme hydration.
  Ensure no visible theme flash or mismatch on first paint.

### Localization

- [ ] Define supported locales.
  Decide which locales are necessary to demonstrate the system credibly.
- [ ] Define locale-aware route strategy.
  Decide whether the public side, internal side, or both support locale-prefixed routes.
- [ ] Define translation scope.
  Decide which surfaces are translated fully and which remain single-language for scope control.
- [ ] Define RTL coverage.
  Include at least one path that proves the UI works in a right-to-left locale.

## 14. State, Async Data, And Diagnostics

### State architecture

- [ ] Define shared app state boundaries.
  Decide which state lives in atoms, route loaders, local component state, and persisted preferences.
- [ ] Define derived and computed metrics.
  Calculate low-stock indicators, days of cover, transfer pressure, and moderation counts from shared state.
- [ ] Define snapshot behavior.
  Decide whether to support saved workspace or diagnostic state snapshots.

### Async and live updates

- [ ] Define route revalidation strategy.
  Decide when loaders rerun after mutations and how stale data is presented.
- [ ] Define cached resource usage.
  Decide which datasets benefit from client-side cache reuse.
- [ ] Define live activity behavior.
  Decide whether to simulate alert feeds or activity streams through channels or tasks.

### Diagnostics and devtools

- [ ] Define the embedded diagnostics drawer.
  Decide what runtime information the example should expose without overwhelming normal users.
- [ ] Define demo-only developer affordances.
  Consider a controlled devtools panel, seed reset button, snapshot viewer, or route-state inspector.

## 15. Accessibility

### Public accessibility

- [ ] Define accessible commerce patterns.
  Ensure product pages, comments, reviews, and public forms remain navigable and readable for assistive tech.
- [ ] Define accessible filter and sort controls.
  Make data-heavy lists keyboard-operable and understandable.

### Internal accessibility

- [ ] Define keyboard-first internal workflows.
  Ensure dense tables, menus, dialogs, and command palette flows work without mouse-only assumptions.
- [ ] Define overlay accessibility behavior.
  Apply proper labelling, focus trapping, restore focus, inert background, and nested overlay handling.
- [ ] Define live announcement behavior.
  Announce route changes, validation errors, save states, and moderation outcomes where appropriate.
- [ ] Define reduced-motion and contrast requirements.
  Ensure animation-heavy flows, dense tables, and stock-status colors still work for users who need reduced motion or stronger visual separation.

## 16. Content And Copy

### Product and commerce content

- [ ] Define product catalog tone.
  Make the public side feel branded and premium rather than placeholder enterprise content.
- [ ] Define warehouse and logistics copy style.
  Ensure operations messaging is concise, legible, and realistic.

### Comment and review content

- [ ] Define seeded customer feedback content.
  Add believable approved, pending, and flagged examples to make moderation meaningful.
- [ ] Define internal audit and note content.
  Seed enough real-looking operational text to validate timelines and collaboration surfaces.

## 17. Implementation Sequencing

### Phase 1: Foundations

- [ ] Implement project skeleton.
  Set up server entrypoint, sqlite initialization, routing scaffold, shared shell, theme tokens, and seed data bootstrapping.
- [ ] Implement route tree and shell layouts.
  Establish public and internal layouts before deeper feature work.

### Phase 2: Public commerce surface

- [ ] Build landing, catalog, and product routes.
  Get the public side SSR-rendered and metadata-aware early.
- [ ] Add comments and public forms.
  Make sure form and moderation flows exist before deeper operations logic.

### Phase 3: Internal operations surface

- [ ] Build dashboard, inventory list, and SKU detail.
  Establish the heart of the operational experience.
- [ ] Add warehouses, transfers, purchase orders, and receiving.
  Expand from read-heavy views into mutation-heavy workflows.

### Phase 4: Cross-cutting systems

- [ ] Add theming, localization, overlays, saved views, and diagnostics.
  Layer in the advanced framework showcase systems after the main product flows work.
- [ ] Add SEO refinements, accessibility passes, and responsive polish.
  Finalize quality after the core product story is stable.
- [ ] Add demo-readiness documentation and fixtures.
  Finish a focused README, local run instructions, seed-reset workflow, and a reviewer-facing demo script so the example is easy to evaluate.

## 18. Testing And Validation

### User stories for testing

- [ ] Define public commerce user stories.
  Capture testable stories such as discovering a product, filtering the catalog, reading warehouse-aware availability, submitting a comment, and requesting restock or quote follow-up.
- [ ] Define internal inventory user stories.
  Capture stories such as finding a low-stock SKU, changing thresholds, creating a transfer, receiving inventory, and validating that the dashboard reflects the change.
- [ ] Define moderation user stories.
  Capture stories such as reviewing a public comment, approving it, rejecting it, and confirming public visibility changes correctly.
- [ ] Define warehouse operation user stories.
  Capture stories such as opening a warehouse page, inspecting backlog, resolving a discrepancy, and comparing warehouse stock pressure.
- [ ] Define personalization user stories.
  Capture stories such as switching theme, changing locale, restoring a saved view, and reloading into the same personalized shell state.
- [ ] Define SSR and hydration user stories.
  Capture stories such as entering a public product page from search, getting the right metadata and first paint, then hydrating into live filters, comments, and forms without mismatch.

### Recommended user story inventory

Use these as the default acceptance-story set:

- [ ] Story: Customer browses the sales catalog.
  A visitor opens `/shop`, filters by category and availability, changes sort order, and lands on a product page with the expected filtered context.
- [ ] Story: Customer inspects a product detail page.
  A visitor opens `/shop/:productSlug`, sees SSR-rendered product content, warehouse-aware promise messaging, and related products, then hydrates into a fully interactive page.
- [ ] Story: Customer submits a product comment.
  A visitor writes a comment or question, submits it through a server-backed form, receives confirmation, and sees the expected moderation message.
- [ ] Story: Customer requests a restock notification.
  A visitor on a low-stock or unavailable product submits a restock request and gets a clear success path.
- [ ] Story: Buyer requests a quote.
  A visitor submits a bulk inquiry and the request is stored and visible internally.
- [ ] Story: Operator works the inventory table.
  An internal user opens `/app/inventory`, sorts by stock health, filters to one warehouse, searches by SKU, and saves that view.
- [ ] Story: Operator edits SKU thresholds.
  An internal user opens a SKU detail route or sheet, edits reorder thresholds, saves changes, and sees audit history update.
- [ ] Story: Manager creates a transfer.
  An internal user identifies stock imbalance, opens the transfer workflow, submits a request, and sees the new transfer in queue views.
- [ ] Story: Receiver reconciles an inbound shipment.
  An internal user opens a receiving session, records mismatch quantities, resolves a discrepancy modal, and finishes reconciliation.
- [ ] Story: Moderator reviews customer feedback.
  An internal user opens `/app/comments`, filters pending items, approves or rejects one, and confirms the public-facing state updates correctly.
- [ ] Story: User resumes personalized state.
  A returning user reloads the application and sees the same theme, locale, density, default warehouse, and saved views restored.
- [ ] Story: Public page remains SEO-safe.
  A product page returns the expected title, description, canonical URL, structured data, and crawlable content before hydration.

### Server and data tests

- [ ] Add sqlite data-layer tests.
  Cover seed loading, queries, filters, comment persistence, and inventory mutations.
- [ ] Add repository-level query tests.
  Validate product lookup, warehouse lookup, availability joins, comment retrieval, saved views, and inventory filter combinations.
- [ ] Add write-path persistence tests.
  Validate comment creation, quote request creation, stock adjustment writes, threshold updates, transfer lifecycle updates, and receiving reconciliation writes.
- [ ] Add handler tests.
  Validate SSR responses, JSON endpoints, form submissions, validation failures, and moderation actions.

### Unit test ideas

- [ ] Add unit tests for query parsing.
  Test catalog and inventory query parsing for sort, filter, pagination, invalid values, and canonicalization.
- [ ] Add unit tests for metadata generation.
  Test route title, description, canonical URL, and structured-data generation for public product and warehouse routes.
- [ ] Add unit tests for theme and preference resolution.
  Test SSR-safe theme, locale, density, and default-warehouse resolution from cookies, session state, or bootstrap payload.
- [ ] Add unit tests for availability messaging.
  Test warehouse-aware delivery promise logic and stock-status labeling.
- [ ] Add unit tests for transfer recommendation logic.
  Test derived stock-pressure and cross-warehouse rebalance recommendations.
- [ ] Add unit tests for receiving reconciliation rules.
  Test expected-versus-actual quantity handling, discrepancy classification, and final state transitions.
- [ ] Add unit tests for moderation-state transitions.
  Test pending, approved, rejected, and flagged comment lifecycle rules.
- [ ] Add unit tests for saved-view serialization.
  Test encoding and decoding of filter, sort, density, and warehouse selection presets.

### UI and route tests

- [ ] Add route-level tests for public pages.
  Cover SSR render, metadata, hydration bootstrap, and query-driven filters.
- [ ] Add route-level tests for internal pages.
  Cover dashboard load, list filtering, modal flows, receiving workflows, and revalidation behavior.
- [ ] Add accessibility-focused browser coverage.
  Validate keyboard navigation, overlay focus handling, form errors, and route announcements.
- [ ] Add theming and locale coverage.
  Validate SSR-stable theme load, locale switching, formatting, and RTL support.

### Component test ideas

- [ ] Add component tests for data table primitives.
  Validate sortable headers, filter chip rendering, empty states, saved-view badges, and bulk-selection affordances.
- [ ] Add component tests for public comment and form modules.
  Validate label wiring, error rendering, success states, moderation messages, and pending states.
- [ ] Add component tests for warehouse availability cards.
  Validate stock labels, ETA messaging, fallback states, and warehouse-specific promise rendering.
- [ ] Add component tests for overlay-backed workflows.
  Validate SKU threshold modal, transfer dialog, discrepancy resolution dialog, side sheets, and command palette behavior.
- [ ] Add component tests for preference controls.
  Validate theme toggle, locale selector, density selector, and saved-view controls.
- [ ] Add component tests for activity timeline and audit rows.
  Validate event grouping, status chips, timestamps, and note rendering.

### Integration test ideas

- [ ] Add integration tests for public browsing flow.
  Exercise landing to catalog to product route transitions with SSR and hydration continuity.
- [ ] Add integration tests for public form submissions.
  Cover comment submission, restock request, and quote request server round-trips including validation errors.
- [ ] Add integration tests for inventory management flow.
  Cover loading inventory data, applying filters, saving views, editing thresholds, and revalidating list state.
- [ ] Add integration tests for transfer creation flow.
  Cover derived recommendation display, modal submission, persistence, and queue refresh.
- [ ] Add integration tests for receiving reconciliation flow.
  Cover loading a session, entering actual quantities, handling discrepancy branches, and finalizing reconciliation.
- [ ] Add integration tests for moderation flow.
  Cover pending comments entering the queue, moderation actions, and public visibility changes.
- [ ] Add integration tests for theme and locale persistence.
  Cover first paint, hydration, preference save, reload, and SSR-stable resume behavior.

### Manual Playwright and exploratory browser tests

- [ ] Add manual Playwright smoke scripts for public routes.
  Script realistic review flows for landing, catalog filtering, product detail hydration, comment submission, and warehouse availability pages.
- [ ] Add manual Playwright smoke scripts for internal routes.
  Script realistic review flows for dashboard navigation, inventory filtering, transfer creation, receiving, moderation, and settings changes.
- [ ] Add manual Playwright scripts for overlay and focus behavior.
  Verify stacked modal escape handling, focus restore, anchored menus, command palette behavior, and side-sheet layering.
- [ ] Add manual Playwright scripts for SSR and metadata checks.
  Validate titles, descriptions, canonical tags, structured data, and hydration-safe route entry on public pages.
- [ ] Add manual Playwright scripts for theme and locale checks.
  Verify no theme flash, stable localized first paint, RTL layout, and restored preferences after reload.
- [ ] Add manual Playwright scripts for table interactions.
  Validate sort order changes, debounced search, URL-driven filter updates, and saved view restore behavior.

### Performance and polish validation

- [ ] Define performance checkpoints.
  Measure route entry, hydration cost, table interaction latency, and overlay responsiveness.
- [ ] Add cross-browser smoke coverage.
  Validate the key public and internal flows in Chromium, Firefox, and WebKit so the flagship example does not look tuned to one engine only.
- [ ] Add visual review artifacts.
  Capture reference screenshots or a simple review pack for the landing page, product page, inventory table, overlays, and dark-mode states.
- [ ] Add manual demo checklist.
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

- [ ] Is the public versus internal split correct?
- [ ] Are the signature flows the right ones?
- [ ] Is the SQLite scope realistic for an example project?
- [ ] Are comments, reviews, and requests all necessary, or should one be cut?
- [ ] Is purchase-order depth necessary for v1, or should transfers and receiving lead first?
- [ ] Should locale-prefixed routes apply only to public pages?
- [ ] Should the first release include auth, or mock role state locally?
- [ ] Is the visual ambition high enough to feel like a flagship example?