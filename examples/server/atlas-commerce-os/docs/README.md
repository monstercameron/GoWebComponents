# Atlas Commerce OS

This is the kickoff scaffold for the flagship Atlas Commerce OS example.

It currently establishes:

- the shared Atlas page tree used for both native-server route payloads and browser hydration
- the current product narrative and operations visual direction
- shared bootstrap shapes, repository contracts, seed data, and token hooks
- a native Go server with sqlite-backed mutations and request-time route metadata plus bootstrap payloads
- a runnable wasm hydration entrypoint that renders the Atlas shell into the server's empty app container

What it shows now:

- storefront and internal Atlas routes served with request-time head metadata (`<title>`, description, canonical) and a `__ATLAS_BOOTSTRAP__` payload carrying route, preferences, i18n, theme, page data, and CSRF token
- public catalog, product, warehouse, and warehouse-specific availability surfaces
- validated public quote, restock, and product-question forms
- internal inventory, transfers, receiving, moderation, settings, purchase-order, and warehouse routes behind mock auth
- browser hydration of the shared page tree from that same bootstrap payload; the server emits an empty `<div id="app"></div>`, so all visible markup is produced client-side and routes such as `/shop` are blank without scripting
- sqlite-backed mutations, CSRF protection, and request-time recovery flows

Current migration focus:

- rewrite the Atlas public and internal shells using the React design references under `design/`
- match the GoWebComponents HTML and CSS output to the React compositions as closely as Atlas route semantics allow
- replicate and enhance the React interaction patterns through the Atlas server bootstrap plus WASM hydration
- track execution details in `docs/ATLAS_COMMERCE_OS_TODO.md`

## Current Structure

- `client/`: wasm hydration entrypoint that renders the Atlas surface from the server bootstrap payload
- `shared/`: shared Atlas bootstrap contracts and page tree, repository interfaces, seed data, design tokens, and cross-surface helpers used by both the browser and native server paths
- `server/`: native Atlas server, auth and sqlite layers, route metadata and bootstrap handlers, mutation endpoints, and server-owned data assets under `server/data/`
- `docs/`: this README plus supporting notes and local reset workflow helpers

## Support Files

- `server/data/schema.sql`: initial sqlite schema scaffold
- `server/data/migrations/001_initial_schema.sql`: numbered startup migration used by the native server
- `shared/repository/contracts.go`: repository interfaces and query contracts
- `shared/atlas/bootstrap.go`: shared bootstrap contract now used by both the native server and the wasm client
- `shared/atlas/page.go`: shared Atlas page tree used for server route payloads and browser hydration
- `client/main.go`: js/wasm hydration entrypoint that renders the Atlas surface in the browser
- `docs/ATLAS_COMMERCE_OS_TODO.md`: active rewrite plan for React-to-GoWebComponents Atlas parity work
- `docs/scripts/reset-seed.ps1`: removes the local Atlas sqlite path so demos can return to a clean seed baseline once persistence lands
- The former standalone planning and review notes now live in the consolidated sections below.

## Validation

From `examples/`:

```powershell
go test -tags playwrightgo ../test/playwrightgo/examples -run TestAtlasSSR -v
```

Atlas now validates through the native-server suite. The retired static hash-router lane has been removed so browser coverage stays aligned with the real native-server example.

## Build

From the repo root. The wasm entrypoint is `client/main.go`, a `js/wasm` program, so the build needs `GOOS=js`/`GOARCH=wasm` or it fails with `build constraints exclude all Go files`. Output must land in `examples/static/bin/`, where the server looks for it and where it is served from as `/assets/bin/atlas-commerce-os.wasm`.

PowerShell:

```powershell
New-Item -ItemType Directory -Path .\examples\static\bin -Force | Out-Null
$env:GOOS = 'js'; $env:GOARCH = 'wasm'; go build -o .\examples\static\bin\atlas-commerce-os.wasm ./examples/server/atlas-commerce-os/client; Remove-Item Env:\GOOS, Env:\GOARCH
```

Git Bash:

```bash
mkdir -p examples/static/bin
GOOS=js GOARCH=wasm go build -o examples/static/bin/atlas-commerce-os.wasm ./examples/server/atlas-commerce-os/client
```

## Run

From the repo root, after the build above. Clear `GOOS`/`GOARCH` first if they are still set, because the server must build natively:

```powershell
go run ./examples/server/atlas-commerce-os/server
```

Default server URL:

- `http://127.0.0.1:8096/shop`

## Consolidated Notes Index

- [ACCESSIBILITY_NOTES](#accessibility-notes)
- [ACCESSIBILITY_SPOT_CHECKLIST](#accessibility-spot-checklist)
- [ARCHITECTURE](#architecture)
- [ASSET_STRATEGY](#asset-strategy)
- [AUTH_NOTES](#auth-notes)
- [COMPONENT_BOUNDARIES](#component-boundaries)
- [CONTENT_NOTES](#content-notes)
- [design](#design)
- [DIAGNOSTICS_NOTES](#diagnostics-notes)
- [FRAMEWORK_COVERAGE](#framework-coverage)
- [INTERNAL_NOTES](#internal-notes)
- [KNOWN_LIMITATIONS](#known-limitations)
- [LOCALIZATION_NOTES](#localization-notes)
- [MANUAL_TESTING](#manual-testing)
- [MIGRATIONS](#migrations)
- [MILESTONE_SUMMARY](#milestone-summary)
- [OPERATIONS_NOTES](#operations-notes)
- [OVERLAY_NOTES](#overlay-notes)
- [PERFORMANCE_CHECKPOINTS](#performance-checkpoints)
- [PUBLIC_NOTES](#public-notes)
- [RELEASE_READINESS](#release-readiness)
- [REVIEW_DECISIONS](#review-decisions)
- [REWRITE_INVARIANTS](#rewrite-invariants)
- [ROUTE_ARCHITECTURE](#route-architecture)
- [ROUTE_DATA_NOTES](#route-data-notes)
- [SCREENSHOT_CHECKLIST](#screenshot-checklist)
- [SCREENSHOT_NOTES](#screenshot-notes)
- [SEED_STRATEGY](#seed-strategy)
- [SEO_NOTES](#seo-notes)
- [SERVER_ENDPOINT_INTEGRATION_PLAN](#server-endpoint-integration-plan)
- [SSR_BOOTSTRAP_NOTES](#ssr-bootstrap-notes)
- [TESTING_OPERATIONS](#testing-operations)
- [TESTING_STORIES](#testing-stories)
- [USER_FLOW_RETHINK](#user_flow_rethink)
- [WAREHOUSE_NOTES](#warehouse-notes)

---

## Consolidated Atlas Notes

### ACCESSIBILITY_NOTES

# Atlas Commerce OS Accessibility Notes

This file records the first accessibility expectations for the Atlas example while the UI remains seed-backed and local-state-driven.

## Current Coverage Goals

- public and internal routes should expose a clear page title and first meaningful heading
- hash-route transitions should keep the shell stable and the destination content legible
- settings toggles and workflow controls should remain reachable as native buttons
- dense internal panels should preserve readable text contrast and obvious focus targets

## Accessible Commerce Patterns

- public product, warehouse, and catalog routes should present one clear heading, readable supporting copy, and native link or button controls for every primary action
- public quote, restock, and question forms should keep labels, helper text, inline validation, and success feedback visible without relying on color alone
- customer-facing inventory signals should translate raw availability into promise language so assistive technology does not need to infer business meaning from badge color

## Filter And Sort Controls

- catalog sort, warehouse filters, and pagination controls should stay keyboard reachable and should not hide the active state behind visual styling alone
- inventory saved-view, sort, density, and query controls should expose the current workspace posture through adjacent summary copy and visible badges
- dense control clusters should keep button semantics instead of clickable wrappers so keyboard review stays predictable

## Overlay And Announcement Rules

- Atlas overlays should keep clear titles, trap focus, restore the opener, and preserve a stable background shell while modal or sheet workflows are active
- route changes should continue to announce the destination heading, while validation and save feedback should remain visible through inline copy, live regions, or toast surfaces as appropriate to the workflow
- diagnostics should stay hidden from public reviewer flows and only appear when explicitly enabled for development review

## Immediate Audit Targets

- landing hero, proof panels, and CTA group on the storefront route
- inventory, SKU detail, transfer, receiving, moderation, and settings routes
- theme and locale button groups on settings
- route-not-found recovery links

## Known Gaps

- no explicit focus restoration after route changes yet
- no dedicated keyboard-order audit yet for the internal workspace shell
- no full screen-reader copy review yet for status-heavy operational panels
- live-region behavior should still be rechecked once Atlas adds server-backed writes and richer async loaders

## Contrast Audit Notes

- public light-mode cards now need to be reviewed for separation between page background, panel fill, and helper text rather than only raw text contrast
- internal dark-mode nested cards should be checked as a stack so the new shadow tiers read as hierarchy and not decorative blur
- Arabic heading tuning should be reviewed alongside badge and pill contrast because tighter rhythm changes where the eye lands first

## Reduced-Motion Requirements

- hover and keyboard-visible lift treatments should share the same hierarchy cues so motion is not required to discover focus
- `prefers-reduced-motion` should remove decorative travel while keeping focus rings, card hierarchy, and route changes readable
- overlay and route-shell motion checks should be rerun whenever Atlas adds richer async transitions or new stacked surfaces

---

### ACCESSIBILITY_SPOT_CHECKLIST

# Atlas Commerce OS Accessibility Spot Checklist

Use this checklist during manual review before calling the example demo-ready.

## Keyboard

- tab through public shell links without losing visible focus
- tab through internal shell links without trapping unexpectedly
- activate theme and locale controls from the keyboard
- switch SKU detail sections from the keyboard
- verify transfer and receiving state buttons can be reached and activated without the mouse

## Semantics

- each route exposes one clear page-level heading
- button groups remain actual buttons rather than clickable non-semantic containers
- status summaries remain readable when badges or color are ignored
- route-not-found recovery links remain clear and actionable

## Visual access

- internal dark surfaces keep enough contrast for labels, helper copy, and stats
- focus state remains obvious on shell links and workflow buttons
- reduced-motion review should be rerun after any overlay or route-shell motion change

## Known manual gaps

- no screen-reader announcement pass yet
- no full overlay focus-trap regression pass yet across every workflow
- no end-to-end screen-reader pass yet for persisted preference resume copy

---

### ARCHITECTURE

# Atlas Commerce OS Architecture Notes

This file explains the initial structure of the flagship example.

## Current Structure

- main.go mounts the example hash router and binds the current public and internal route shells.
- route_manifest.go stores the first route constants and registration inventory.
- seed/seed.go stores the current seeded route copy, stats, and first-pass inventory planning values.
- tokens/tokens.go stores shared class tokens and token-name placeholders that will later move into a richer theme layer.
- design.md stores non-technical product, content, and visual decisions.

## Why The Example Is Structured This Way

The current goal is to move the example from narrative planning into a stable skeleton without overcommitting to backend structure too early. The route manifest, seed package, and token package let the example grow in layers instead of accumulating all decisions inside one main.go file.

## Next Technical Layers

1. Add route-aware loaders and bootstrap payloads.
2. Add a shared seed-loading package for inventory, warehouses, comments, transfers, and receiving.
3. Add local component files for public shell, internal shell, inventory table, and product hero.
4. Add SSR-aware entrypoints once the data contract is stable enough to serialize.

## Current Non-Goals

- Real persistence
- Real auth
- Full SSR server
- Complete component decomposition

Those will come after the route and seed contracts settle.

---

### ASSET_STRATEGY

# Atlas Commerce OS Asset Strategy

Atlas should keep asset handling intentionally simple in the first release.

## Static Asset Direction

- seeded product imagery, brand graphics, and review artifacts should be served as static files from the example workspace
- Open Graph images should follow the route metadata map so public landing, catalog, product, warehouse, and warehouse-availability routes each have a stable preview asset
- screenshots remain reviewer artifacts and should stay in the Atlas example folder rather than mixing into public application asset paths

## Serving Rules

- public assets should be cacheable because they are versioned with the repo state rather than user-generated at runtime
- the first server pass can use straightforward static-file serving plus conservative cache headers; aggressive CDN-specific tuning is out of scope for the example
- fallback assets should exist for default metadata previews so every public route can resolve an image even before route-specific art is expanded

## Upload Scope

- Atlas v1 does not support media uploads, receiving attachments, or user-generated files
- if uploads land later, they should be isolated to clearly scoped workflows with explicit validation, storage, and retention rules instead of piggybacking on the static asset path

---

### AUTH_NOTES

# Atlas Commerce OS Auth Strategy

The first auth milestone should stay intentionally lightweight.

## Demo Strategy

- use mocked role state rather than a full auth system
- default to an inventory-manager role for internal routes
- keep public routes fully open
- allow role switching later as a review affordance rather than as security infrastructure

## Why

The example needs believable route separation without letting auth complexity consume the flagship build.

## First Internal Guard Rule

Internal routes should assume a mock session object with:

- user id
- display name
- role
- default warehouse

That is enough to drive preference and route behavior in the first implementation pass.

## Session Strategy

- the first release should keep public routes unauthenticated and internal routes behind a mocked operator session
- session state should be represented by a compact server-readable shape so future SSR can gate internal routes without inventing a second auth model later
- the initial session contract only needs identity, role, and warehouse affinity; Atlas should not add password, MFA, or account-recovery flows in the example scope

## CSRF Strategy

- the current wasm-only demo does not issue server-backed writes, so CSRF defense remains a documented contract rather than a shipped runtime concern
- once Atlas adds server-backed forms, cookie-backed sessions should pair with a synchronizer token emitted in the bootstrap payload and echoed through form posts or mutation headers
- internal write endpoints should reject requests that fail token validation or same-origin checks, and public forms should use the same token flow so the example demonstrates one clear posting model

## File Upload Direction

- Atlas v1 should not include live file uploads
- product imagery, Open Graph assets, and illustrative operational graphics should stay seeded and static so the flagship example remains easy to run locally
- future receiving attachments or editorial media uploads can be added only after server persistence lands and should then use explicit size, type, and retention rules

---

### COMPONENT_BOUNDARIES

# Atlas Commerce OS Component Boundaries

The example should split into focused components once the current route shells stop being enough.

## Public Components

- public layout shell
- landing hero section
- proof strip
- category rail
- product grid and card
- product hero
- warehouse availability summary
- quote request form
- restock request form
- comment thread

## Internal Components

- internal layout shell
- dashboard alert cards
- dashboard quick actions
- inventory table
- inventory filter bar
- saved views rail or menu
- SKU detail panel
- moderation queue
- moderation detail panel
- settings form groups

## Prop Direction

The first component props should stay plain and serializable. Avoid hiding route or preference state behind opaque globals while the example is still establishing its data contracts.

---

### CONTENT_NOTES

### USER_FLOW_RETHINK

# Atlas Commerce OS User Flow Rethink

Atlas should stop acting like one blended experience. It serves two different users with two different jobs:

- clients who are trying to evaluate, shortlist, and buy
- owners or operators who are trying to manage catalog, stock pressure, and follow-up work

The current UX mixes those intents too early, especially on product detail where quote, restock, and question capture all sit beside one another with the same weight.

## 1. Separate The Two Jobs Clearly

### Client job

The client is not trying to manage supply chain state. They are trying to answer practical buying questions:

- Is this right for my space or project?
- Can I get it in my region?
- What is the lead time?
- What should I do next if stock is tight?

The client journey should feel like guided buying, not operational exception handling.

### Owner or manager job

The manager is not browsing. They are trying to:

- monitor demand signals
- keep the catalog accurate
- decide whether to replenish, transfer, or pause a SKU
- respond to questions and quote opportunities

The manager journey should feel like queue-based decision support, not a mirrored version of the public product page.

## 2. Rethink The Product Detail Page For Clients

The product page should become a buyer decision page with one primary next step, one contextual stock message, and one secondary support path.

### Recommended client page hierarchy

1. Product story and fit: what it is, who it is for, why it is differentiated.
2. Promise block: availability, lead time, warehouse coverage, and delivery confidence in plain language.
3. Primary action: the single best next step for the current stock state.
4. Secondary actions: compare, save, ask a question, or explore alternatives.

### Replace the current generic restock form with state-aware actions

The phrase `restock request` makes sense to the business, but it is poor customer language. Customers are not requesting replenishment operations. They are expressing purchase intent.

Use these client-facing states instead:

- In stock:
	- Primary CTA: `Request pricing` or `Start a project quote`
	- Secondary CTA: `Check delivery for my region`
- Low stock:
	- Primary CTA: `Reserve upcoming availability`
	- Secondary CTA: `Talk to a specialist`
	- Support copy: explain limited stock and likely next availability window
- Out of stock or unavailable in selected hub:
	- Primary CTA: `Notify me when available`
	- Secondary CTA: `See similar in-stock options`
	- Support copy: explain whether this is a regional issue or a network-wide issue

### Keep questions, but reduce their visual weight

`Ask a product question` should remain available, but it should not compete equally with the primary buying action. It should sit under the main CTA cluster as a support path for delivery, fit, finish, or install questions.

### Improve warehouse relevance for clients

Warehouse information should answer promise questions, not expose operations structure.

Better client copy:

- `Available from New Jersey for Northeast delivery`
- `Limited Midwest stock, replenishment inbound`
- `Unavailable in your nearest hub, similar options ready to ship`

That keeps the warehouse story useful without making the customer think like an internal planner.

## 3. Build A Separate Manager Workflow

Managers need a different surface entirely. They should never work from the public product-detail logic.

### Recommended manager workflow

1. Dashboard highlights exceptions, not just summaries.
2. Demand inbox groups quote requests, availability alerts, and product questions by SKU.
3. Product manager reviews catalog record and merchandising accuracy.
4. Inventory manager reviews stock by warehouse, inbound, and threshold pressure.
5. Operations chooses action: transfer, purchase order, waitlist handling, or merchandising fallback.

### Product and inventory should be linked, not blended

Keep these roles distinct in the interface:

- Product CMS answers: what is this item, how is it presented, is it live, and what is the default managed lane?
- Inventory answers: where is stock tight, what is inbound, what threshold is broken, and which warehouses are affected?

The UI should connect these areas with clear links such as:

- from product editor: `View inventory health`
- from SKU detail: `Edit storefront content`
- from manager dashboard: `Open demand queue for this SKU`

## 4. Convert Customer Actions Into Manager Queues

Each customer action should land in a manager-friendly queue with the right operational meaning.

### Client action to manager queue mapping

- `Request pricing` -> sales or quote queue
- `Reserve upcoming availability` -> demand hold queue tied to inbound inventory
- `Notify me when available` -> back-in-stock queue tied to region and SKU
- `Ask a product question` -> product support queue tied to product detail context

This is the important shift: the customer should see intent-oriented language, while the manager sees structured operational work.

## 5. Better Product-Page UX By Stock State

### In stock

- Lead with confidence: availability, delivery region, and project quote CTA.
- De-emphasize exception workflows.

### Low stock

- Lead with urgency and the next realistic path.
- Offer `Reserve upcoming availability` instead of a raw restock form.
- Surface similar alternatives if the item may not cover the buyer timeline.

### Out of stock

- Remove the illusion of immediate fulfillment.
- Offer `Notify me when available` and `See similar options`.
- If inbound exists, show the expected restock window in customer language.

## 6. Recommended Route Mental Model

Public routes should support buyer momentum:

- `/shop` for discovery and filtering
- `/shop/:slug` for decision-making and conversion
- `/warehouses/:slug/availability/:productSlug` only when the buyer explicitly wants region-specific promise detail

Internal routes should support decision queues:

- `/app/dashboard` for exceptions and hot items
- `/app/products` for catalog management
- `/app/inventory` for stock health
- `/app/comments` or a future demand queue for buyer follow-up work
- the shipped `/app/products` route now uses a summary band, filter shell, CRUD action rail, dense catalog table, and side create rail so operators work from a reviewable product workspace instead of a merch-card list

### Products route work map

- table shell: the dense catalog table is the primary review surface, with explicit edit and warehouse-lane actions on every row
- filter bar: search, category, status, and sort stay in a dedicated route shell ahead of the table
- saved-view slice: still reserved for a later operator-preset pass rather than bolted into the first CRUD rewrite
- bulk-action slice: the current action rail stays intentionally non-destructive and routes operators into cleanup, inventory, or warehouse follow-up
- route-level empty state: the table shell already handles zero-match filters with a route-specific recovery message and a create-product fallback
- warehouse return-target continuity: product row actions still link into warehouse item profiles, and product create, update, and delete flows still preserve `return_warehouse_id` when the workflow originates from a warehouse route
- the shipped product editor now uses a preview-first layout: preview hero, preview surface, and warehouse context on the left, with grouped metadata, warehouse, and merchandising edit sections plus the existing save/delete and unsaved-change guard on the right

### Product editor work map

- metadata form: SKU identity, slug, title, category, price, status, and finish
- merchandising copy form: summary, details, SEO title, and SEO description
- warehouse context: warehouse anchor plus available and inbound volume fields
- preview surface: the left-column preview hero and merch preview card
- unsaved-changes workflow: recent draft-change summary plus the existing `before-leave` guard around the save path
- workflow lock: visual tuning must preserve the current save endpoint, delete endpoint, warehouse return-target hidden field, warehouse item handoff links, and the unsaved-change guard behavior

## 7. Practical Product-Page Rewrite

If Atlas only changes one thing next, change the product page action stack from this:

- quote request
- restock request
- ask a product question

to this:

- one primary CTA based on stock state
- one contextual availability message in plain language
- one support CTA for human help
- one alternatives path when inventory cannot support the buyer timeline

That will create a better customer experience immediately, and it will also produce cleaner signals for managers.

# Atlas Commerce OS Content Notes

Atlas copy should keep the public side premium and the internal side calm, direct, and believable.

## Product Catalog Tone

- concise
- tactile
- specific
- premium without startup cliches or fake luxury inflation

The catalog should sell complete workspace systems, not disconnected parts.

## Warehouse And Logistics Copy Style

- concise
- legible
- realistic
- operationally calm

Warehouse copy should communicate service region, promise speed, and backlog pressure without drifting into playful brand language.

## Seeded Customer Feedback Content

- approved examples should include praise, practical delivery notes, and product-specific questions
- pending examples should read like legitimate buyer questions awaiting moderation
- flagged or rejected examples should be clearly believable moderation edge cases rather than cartoon abuse strings

## Internal Audit And Note Content

- operational notes should be timestamped, practical, and secondary to the timeline
- transfer, receiving, threshold, and moderation timelines should read like concise event summaries used by working teams, not lorem ipsum scaffolding

---

### design

# Atlas Commerce OS Design Brief

This file stores non-technical decisions for the Atlas Commerce OS flagship example.

## Product Narrative

Atlas Commerce OS is a premium commerce and operations platform for a design-forward modular workspace brand. The public experience sells the products with confidence, locality, and trust. The internal experience runs the warehouses, inventory decisions, receiving work, transfer planning, and moderation queues that keep the brand reliable.

The example should feel like one believable business with two faces:

- a polished public storefront for discovery and conversion
- a dense internal console for inventory and fulfillment work

## Review Goals

The example should prove five things immediately:

1. The framework can deliver polished public pages, not just demos and utilities.
2. The framework can handle serious internal workflows with dense state and route depth.
3. SSR and hydration can coexist without visual mismatch or fragile page entry.
4. Accessibility, localization, and overlays can be first-class concerns instead of afterthoughts.
5. The final example can be shown to new users as the flagship story for the whole project.

## Brand Direction

### Brand posture

The brand should feel premium, practical, and quietly confident rather than playful or corporate-generic. Think editorial commerce on the public side and high-trust operator tooling on the internal side.

### Product vertical

The catalog should center on modular workspace and storage products:

- standing desks
- shelving systems
- drawer units
- cable-management kits
- task lighting
- modular carts
- acoustic panels
- accessories bundles

This gives the example products that photograph well, support variants and specifications, and make warehouse availability meaningful.

### Naming tone

Use concise, premium product naming instead of novelty names. Product copy should sound commercially credible and consistent with a modern design catalog.

### Brand voice

Atlas should sound measured, specific, and high-trust.

Voice rules:

- use short declarative sentences
- prefer operational clarity over marketing hype
- make premium claims through details, not adjectives alone
- avoid startup jargon, luxury parody, and fake urgency
- keep warehouse and availability language direct and useful

## Audience And Roles

### Public audience

- individual buyers comparing products
- team buyers requesting quotes
- returning customers checking local availability

### Internal roles

- inventory manager
- warehouse operator
- receiving coordinator
- buyer
- moderation or support reviewer

Each major screen should visibly belong to one of these roles.

## Signature Flows

These are the flows the example must feel built around:

1. A visitor lands on the storefront, filters the catalog, opens a product, and understands availability by warehouse.
2. A visitor submits a quote request or restock request when stock is limited.
3. An inventory manager identifies a low-stock issue and creates a transfer from a better-positioned warehouse.
4. A receiving coordinator reconciles an inbound shipment, records a discrepancy, and closes the session.
5. A reviewer moderates public comments and sees the public-facing result change.
6. A returning user reloads into the same theme, locale, density, and preferred operational context.

## MVP Versus Stretch

### MVP capabilities

- premium public landing, catalog, product, and warehouse routes with warehouse-aware availability context
- internal dashboard, inventory, warehouse, transfer, receiving, moderation, purchase-order, and settings routes
- direct route entry, query-backed browsing state, persisted preferences, and inventory resume behavior
- loader-backed internal routes, overlay-backed workflows, and focused browser smoke coverage
- diagnostics, screenshots, and reviewer documentation that make the example demo-ready

### Stretch capabilities

- server-backed data contracts, sqlite persistence, and request-time SSR bootstrap
- localized content bundles, locale-prefixed public routes, and richer SEO output
- async resource patterns, hydration reuse, and future browser-router parity
- command palette depth, export or snapshot workflows, and release-only cleanup of developer diagnostics

## Surface Split

### Public surface

The public side should be aspirational, spacious, and image-led. It should prioritize confidence:

- what the product is
- why it is credible
- where it is available
- how quickly it can ship
- what other customers are saying

### Internal surface

The internal side should be compact, fast, and information-dense. It should prioritize action:

- what is wrong
- where attention is needed
- what can be changed now
- what happened recently
- what needs approval or review

## Visual Direction

### Public look

The public interface should use a warm-light editorial palette with soft stone backgrounds, deep ink typography, muted metallic accents, and product imagery that feels tactile. The goal is to feel premium without looking like a fashion site.

### Internal look

The internal interface should use a darker, steel-and-slate visual system with strong information hierarchy, restrained accent colors, and dense but legible surfaces. It should look serious and fast, not merely dark.

Shipped internal surface system:

- hero tier: deep steel gradient for the route hero and other shell-level lead surfaces
- standard card tier: elevated slate panels for shared stats, list cards, feature cards, and grouped workspace navigation
- inset tier: denser recessed panels for summary rows, secondary nav links, and compact route context
- accent tier: brighter cyan-leaning action cards for operator workflows that need stronger visual pull than passive data panels

Shared internal surface contract before route rewrites:

- card helpers: `internalHeroSurfaceClass`, `internalSurfaceCardClass`, `internalInsetSurfaceClass`, `internalAccentSurfaceClass`, and `internalSurfacePillClass`
- table helpers: `internalTableContainerClass`, `internalTableHeaderCellClass`, and `internalTableRowClass`
- route rewrites should compose new dashboard, inventory, product, warehouse, logistics, and settings surfaces from those helpers instead of introducing one-off dark card and table wrappers

Internal surface rollout order:

- first wave: dashboard and inventory adopt the shared hero, card, and table-container helpers
- second wave: products and warehouse ops inherit the same surface contract once the first operator-heavy routes stabilize
- third wave: logistics and settings finish the rollout after the higher-traffic triage and catalog routes prove the surface system

### Typography

Use a distinctive display face for public headings and a legible operational face for dense UI. A good baseline direction is:

- display: Space Grotesk
- interface text: IBM Plex Sans
- numeric and diagnostic accents: IBM Plex Mono

### Motion

Motion should be purposeful and short. Key ideas:

- route transitions should feel stable rather than dramatic
- overlays should enter with confident but restrained movement
- table interactions should favor immediate response over decorative motion
- reduced-motion mode should remove non-essential transitions cleanly

## Experience Principles

1. Public pages should explain availability clearly, not hide it behind generic stock badges.
2. Internal pages should prefer decisive action over decorative dashboard clutter.
3. Dense views should still have breathing room and obvious focus targets.
4. Every empty state should teach the user what to do next.
5. Every success or failure state should feel intentional and calm.
6. The whole product should feel demo-ready even when disconnected from real business systems.

## Page Direction

### Landing page

The landing page should introduce the brand, show warehouse-backed reliability, and move the user quickly into products or location-specific availability. It should not read like a framework demo.

Recommended modules:

- hero with product system framing
- proof strip for fulfillment speed and warehouse footprint
- featured categories
- highlighted products
- regional fulfillment story
- customer trust or testimonial block
- quote request callout

### Catalog page

The catalog should feel commercial and highly usable. Filters and sort must feel first-class, not bolted on. The layout should support quick scanning and deep-linkable query state.

### Product detail page

The product page should make three ideas clear in the first screenful:

- why the product matters
- whether it is available near the user
- what the next best action is

When a product is constrained or unavailable, the page should pivot smoothly into alternatives, warehouse-specific availability, quote capture, or restock capture.

### Warehouse pages

Warehouse pages should make locality feel real. They should communicate service region, promise speed, and stocked highlights without becoming logistics dashboards.

### Internal dashboard

The dashboard should not become a generic KPI wall. It should behave like a triage surface with a few sharp sections:

- urgent inventory alerts
- inbound receiving work
- transfer pressure
- moderation backlog
- quick navigation into action-heavy views
- the shipped dashboard now follows that triage-first structure directly: top summary band, action cluster, high-attention panel, activity feed, and purchase-order summary, with purchase-order data loaded into the dashboard payload instead of being inferred from unrelated counts
- within that shipped structure, the summary band and action cluster are the lead surfaces, so the new hierarchy is visible before an operator scans the secondary high-attention and activity panels
- the secondary dashboard sections are now also explicit: a high-attention panel for queue risk, an activity feed for recent motion across moderation, transfers, receiving, and POs, and a denser purchase-order summary with direct route affordances

### Inventory management

The inventory view should be the strongest internal screen. It needs to feel fast, controlled, and trustworthy with visible sort state, filters, saved views, and bulk actions.

- the shipped inventory route now follows that ops-first structure directly: an inventory triage summary band, a dedicated filter model card, a route action cluster for critical or receiving pivots, a clickable triage band, and a dense SKU queue table as the main pane
- the right rail now stays subordinate to the queue and carries workspace stats, active filter context, saved-view selection, and save-view persistence so operators can confirm scope without losing the main triage surface
- the shipped SKU detail route now matches that same shell language: a hero-level lane workspace summary, a stat strip, a denser lane roster table, grouped lane editors, and a right rail that keeps SKU actions, replenishment, and threshold controls separate from the main comparison surface

### Inventory route work map

- triage summary band: lead counts and route summary keep inventory pressure readable before any operator scans the table
- filter model: query, warehouse, status, and sort controls stay grouped in one dedicated card so URL-backed filtering remains obvious
- saved-view rail: saved-view selection and persistence stay in the side rail instead of competing with the queue for primary attention
- dense queue table: the main pane stays table-first, with SKU posture, lane spread, inbound exposure, and direct route actions visible at a glance
- route action cluster: the first route-level pivots are explicit cards for critical lanes, promise-risk review, and receiving follow-up rather than loose text links

### SKU detail work map

- summary hero: the route opens with one lane-workspace hero so the operator has SKU, lane-count, and primary-warehouse context before touching a form
- lane roster: the roster stays as the first dense table, because warehouse comparison is the prerequisite for deciding whether the problem is local or network-wide
- lane edit forms: lane editors stay grouped in one dedicated section under the roster, preserving the compare-first then edit flow
- threshold-history side workflow: threshold-history remains a route-local overlay workflow so the SKU page can keep context while the historical panel opens and closes
- replenishment or transfer actions: replenishment entry and adjacent SKU handoff actions stay in the right rail so they are explicit secondary workflows rather than the route's visual center of gravity

### Warehouse operations

- the shipped warehouse list route now behaves like a facility triage board: summary band, action cluster, dense facility table, and a supporting right rail for cross-route handoffs
- the shipped warehouse detail route now follows the same ops shell language: facility hero, route summary, facility filter model, stat strip, action cluster, nested item continuity, dense warehouse-item table, and a right rail for facility context plus replenishment history

### Warehouse ops work map

- warehouse list shell: the list route now opens as a facility triage shell with summary band, route action cluster, and dense facility table
- warehouse detail workspace: the facility detail route keeps hero, route summary, filter model, and stat strip together before the operator drills into items
- warehouse item table: warehouse-managed items now live in one dense table instead of a card stack, so item pressure is scan-friendly at the facility level
- purchase-order rail: replenishment history and creation affordances stay in the right rail so inbound recovery remains an explicit secondary workflow
- nested item-detail continuity: the nested item outlet remains mounted inside the parent facility route so warehouse context survives the item drill-in instead of collapsing into a separate detached page

### Purchase orders

- the shipped purchase-order list route now behaves like a vendor recovery board with a summary band, route action cluster, dense vendor-order table, and a right rail for cross-route handoffs
- the shipped purchase-order detail route now uses a PO workspace hero and dense line-item table while preserving the existing lazy side rail for route refresh, stats, and approve or hold actions

### Purchase-order work map

- list shell: the purchase-order route now opens as a vendor recovery shell instead of a loose card list
- status summary band: order counts by submitted, approved, and on-hold posture stay visible before an operator scans the table
- detail hero: the detail route leads with one PO workspace hero so vendor, warehouse, ETA, and current status are readable at a glance
- line-item context: inbound lines stay in one dense table, making quantity and ETA review the prerequisite to any decision
- approve or hold workflow tasks: approval, hold, refresh, and supporting stats stay isolated in the lazy detail rail so the primary route body remains stable

### Logistics and support

- the shipped transfers route now behaves like a balancing workspace with a summary band, route action cluster, dense transfer table, and a supporting rail for transfer creation plus warehouse and receiving handoffs
- the shipped receiving route now behaves like a closeout workspace with a summary band, route action cluster, dense receiving table, and a supporting rail for reconciliation plus upstream or downstream handoffs
- the shipped comments route now behaves like a buyer inbox workspace with a summary band, route action cluster, dense moderation table, and a supporting rail for moderation actions and cross-route follow-up
- the shipped settings route now behaves like an operator control room with a summary band, route action cluster, and a supporting rail for preferences, saved-view exchange, and workspace snapshot flows

### Logistics and support work map

- receiving mini-rewrite: summary band, closeout action cluster, dense session table, and reconciliation rail
- transfers mini-rewrite: balancing summary band, transfer action cluster, dense movement table, and creation plus handoff rail
- comments mini-rewrite: buyer inbox summary band, moderation action cluster, dense question table, and moderation workflow rail
- settings mini-rewrite: operator control summary band, settings action cluster, and the existing preference, saved-view, and workspace snapshot cards treated as one control-room rail

## Content Direction

### Product copy

Product copy should be concise, tactile, and specific. Avoid lorem ipsum, fake startup language, or overblown luxury claims.

### Operations copy

Operational copy should be direct and calm. It should read like a system used by professionals, with short labels and clear state names.

### Comment content

Public comments should include a believable mix of praise, specific questions, shipping concerns, and moderation edge cases. Internal notes should sound practical and contextual.

## Demo Data Story

The seeded world should feel intentionally small but rich enough to support all the key flows.

Recommended baseline:

- 3 warehouses with distinct regional identities
- 24 featured products suitable for public merchandising
- a larger internal catalog for filtering and operational pressure
- several low-stock and out-of-balance scenarios
- pending, approved, rejected, and flagged comments
- inbound receiving sessions with at least one discrepancy path

## Quality Bar

The example is ready to show when:

- the public side feels polished enough to share externally
- the internal side feels like a real tool instead of a toy dashboard
- theme, locale, and route entry feel stable on first paint
- overlays and dense tables feel accessible and deliberate
- the demo can be walked from landing page to internal operations without apology

## Homepage Narrative

### Hero message hierarchy

The homepage should open with this order of emphasis:

1. Atlas makes modular workspace systems that ship with regional reliability.
2. Buyers can browse premium products with clear warehouse-aware availability.
3. Teams can request quotes or restock support without leaving the product story.

The hero should sell confidence first, not novelty. The first screen should make it obvious that the brand has both design quality and operational depth behind it.

### Supporting proof points

The first proof strip should reinforce these ideas:

- regional fulfillment from multiple warehouses
- predictable lead times for stocked products
- modular systems that scale from home office to team rollout
- support for quote-based buying on larger orders

### Homepage calls to action

Use one primary and two secondary paths:

- primary: shop workspace systems
- secondary: explore warehouse availability
- secondary: request a team quote

## Review Checklist

The example should not be presented as ready until these checkpoints are true:

- the homepage feels like a real brand landing page
- catalog filtering and sorting are obvious without explanation
- product pages show warehouse-aware availability in the first screenful
- the internal dashboard leads clearly into action-heavy views
- the inventory table feels fast, dense, and trustworthy
- theme and locale restore without visible mismatch
- overlays feel deliberate and accessible
- seeded data tells believable operational stories
- the reviewer can demo the product from public browsing into internal operations in under ten minutes

## Merchandising Direction

### Core category set

The initial public category set should be:

- desks
- shelving
- storage
- lighting
- acoustic panels
- accessories

### Featured merchandising story

The public side should merchandise complete workspace setups rather than isolated utility products. Product groupings should imply that the brand sells systems, not just parts.

### Seasonal or promotional posture

Avoid fake seasonal sales energy. Promotional language should focus on availability, bundle readiness, and fast regional fulfillment instead of discount gimmicks.

### Product naming direction

The first product families should follow a clear naming system:

- Atlas Frame Desk
- Atlas Span Shelf
- Atlas Grid Drawer Unit
- Atlas Beam Task Light
- Atlas Quiet Panel
- Atlas Utility Cart

Variant naming should stay descriptive, using dimensions, finish, or bundle language rather than novelty suffixes.

## Warehouse Identity

The seeded warehouses should feel distinct enough to support meaningful availability and transfer stories.

### Recommended warehouse set

- Nevada Hub: western distribution, strong desk and storage coverage
- Illinois Hub: central balancing location with the broadest mixed inventory
- New Jersey Hub: eastern fulfillment focus with fast accessory and lighting turns

### Personality rules

Warehouse copy should stay practical. The differences should come through in service regions, product mix, and operational pressure rather than playful branding.

## Visual System Direction

### Color system

The palette should be split by surface:

- public base: stone, parchment, warm gray, deep ink
- public accents: muted brass, forest, oxide, slate blue
- internal base: graphite, steel, midnight, smoke
- internal accents: cyan for active state, amber for attention, emerald for healthy status, coral for destructive risk

Stock health should never rely on color alone. Pair semantic color with labels and iconography.

### Spacing system

Use a generous public rhythm and a tighter internal rhythm.

- public sections should feel editorial and roomy
- internal cards and toolbars should compress cleanly without becoming cramped
- dense tables should still preserve row scanability and focus affordances

### Elevation system

Use minimal elevation on the public side and stronger separation in the internal console.

- public cards should feel layered through contrast and border treatment first
- internal overlays should use stronger elevation to preserve hierarchy during stacked workflows
- menus and sheets should feel precise rather than floaty

### Mood board keywords

Use these terms to keep visual reference gathering aligned:

- editorial commerce
- premium industrial
- warehouse clarity
- modular systems
- tactile materials
- calm operator tooling
- warm daylight storefront
- dark operational console

## Seed Direction

### Product seed set

The first 24 public-facing products should be distributed roughly like this:

- 6 desks
- 4 shelving products
- 4 storage products
- 3 lighting products
- 3 acoustic products
- 4 accessories or bundles

At least one product in each major category should support a low-stock, backorder, or reroute story.

### Warehouse seed set

Each warehouse needs a clear operational role:

- Nevada Hub carries stronger west-coast desk and storage depth
- Illinois Hub acts as the balancing location with the widest mixed assortment
- New Jersey Hub turns smaller accessories and lighting faster for eastern demand

The seed data should make transfer suggestions feel justified rather than random.

### Comment seed set

The public comment seed should include a controlled mix:

- approved praise for finish quality and packaging
- practical questions about dimensions and assembly
- quote-oriented requests from team buyers
- low-stock frustration that stays realistic rather than abusive
- a few clearly rejectable or flaggable items for moderation coverage

Customer voice should sound specific and credible. Comments should mention actual product concerns such as cable routing, tabletop depth, lighting warmth, or local delivery timing.

### Transfer seed scenarios

The initial transfer scenarios should include:

- one high-confidence west-to-central rebalance for desk inventory
- one central-to-east accessory rebalance triggered by faster sell-through
- one transfer already in transit so the lifecycle is visible in the queue
- one cancelled transfer to prove audit history and status treatment

### Receiving seed scenarios

Receiving should start with three sessions:

- a clean receiving session with no discrepancies
- a short-shipment session that requires discrepancy notes
- a damaged-goods session that changes available and damaged counts differently

These scenarios should make the receiving workflow look necessary, not artificially staged.

## Information Architecture

### Public route tree

The public route set should start with six primary destinations:

- `/` landing page
- `/shop` catalog and filter view
- `/shop/:productSlug` product detail page
- `/warehouses` warehouse index
- `/warehouses/:warehouseSlug` warehouse detail page
- `/warehouses/:warehouseSlug/availability/:productSlug` warehouse-specific availability view

Public routes should feel editorial first and utility second. Search and filter state belong primarily on the catalog route. Availability deep links belong on product and warehouse-specific pages where locality matters.

### Internal route tree

The internal route set should start with these workspaces:

- `/app/dashboard`
- `/app/inventory`
- `/app/inventory/:sku`
- `/app/warehouses`
- `/app/warehouses/:warehouseId`
- `/app/transfers`
- `/app/transfers/:transferId`
- `/app/receiving`
- `/app/receiving/:sessionId`
- `/app/comments`
- `/app/settings`

Purchase orders can remain planned for the next layer of scope, but the route tree should leave room for them without disturbing the core navigation.

### Route ownership

Ownership rules should stay explicit:

- landing, catalog, product, and warehouse routes are SSR-sensitive public routes
- availability pages are public but more utility-focused and can still be SEO-visible
- dashboard and inventory routes should SSR meaningful first paint before hydrating into denser interactions
- transfer, receiving, moderation, and settings routes can optimize more for authenticated utility than public crawl value

### Shell navigation

The public shell should keep navigation minimal:

- brand mark
- shop
- warehouses
- quote request entry

The internal shell should use a left navigation rail plus a compact top bar.

Primary internal nav groups:

- overview: dashboard and products
- stock: inventory and warehouses
- logistics: transfers, purchase orders, and receiving
- support: comments and settings

The top bar should handle command search, active warehouse context, alerts, and user preferences.

## Screen Outline

### Landing page modules

The landing page should be assembled in this order:

1. hero with headline, subhead, and primary CTA
2. warehouse-backed proof strip
3. category rail
4. featured workspace systems
5. regional fulfillment story
6. customer proof or testimonial block
7. team quote callout

### Catalog page modules

The catalog route should include:

- results header with active filter summary
- search input and sort control
- persistent filter rail on wide screens
- product grid with fast visual scanning
- empty state with guidance when filters collapse results
- soft callout for quote or restock when items are constrained

Default sort should prioritize featured merchandising first, then allow availability, newest, and price-oriented alternatives.

### Product page modules

The product page should be assembled in this order:

1. media and product hero
2. price, finish, and purchase or inquiry block
3. warehouse-aware availability summary
4. specifications and dimensions
5. related setup or bundle suggestions
6. customer comments or questions
7. quote or restock capture when relevant

### Product-detail loading policy

- first-paint critical:
  - product hero, finish and price context, and the primary buyer action rail
  - the initial warehouse-aware promise copy that tells the buyer whether to quote now, reserve later, or switch warehouse context
  - route metadata, quote or restock entry points, and the compact support framing that keeps the product page decision-ready from SSR through hydration
- secondary and allowed to load after the primary body is stable:
  - the promise-lanes comparison island, because it extends the delivery story rather than defining the first commercial decision
  - the public feedback block, because comments and questions should stay visible but secondary to the hero and main action stack
- secondary and preferred for cached repeat-open behavior:
  - related-products rails and other merchandising-adjacent side panels
  - public comments once the route has mounted, so repeat visits can reuse the last approved thread while moderation-aware refresh still happens in the background
- rule for future product-detail work:
  - do not move the primary quote, restock, or hero decision surfaces behind lazy boundaries
  - new secondary panels should default to lazy or cached loading unless they materially change the buyer's first decision on the route
- current shipped mapping:
  - the public feedback block mounts through `ui.UseLazyNode`
  - public comments and related products both reuse cached-resource loaders after hydration
  - the promise-lanes module stays behind its own async boundary so failures and refreshes remain local to that panel

### Visual parity primitive contract

Atlas now has a shared visual primitive layer in `shared/atlas/visual_primitives.go` for React-reference parity work. Route helpers should prefer those functions before adding route-local utility strings.

The first primitive set covers:

- public and internal root background treatments
- public and internal main-shell width and spacing contracts
- public header shell, inner wrapper, desktop nav item, and mobile nav item classes
- public hero, glass card, metric card, catalog card, catalog-control, form-control, and signal-pill classes
- internal hero, card, inset, accent, pill, and dense-table classes through the existing shared helpers

The current parity mapping is:

- header: `publicHeader` owns the logo block, desktop nav, mobile menu sheet, active states, and shop CTA placement through shared public header primitives
- background: `App` applies one public layered background to landing, catalog, product, warehouse detail, and availability routes, while the internal surface keeps its separate steel-and-slate root treatment
- hero: landing and product hero cards use `publicHeroSurfaceClass`; route hero copy and CTA rules stay in `renderPublicHero` and route-specific product or warehouse helpers
- cards: public feature, metric, catalog, and signal-pill helpers route through shared card primitives; internal repeated surfaces continue through `internalSurfaceCardClass`, `internalInsetSurfaceClass`, and `internalAccentSurfaceClass`
- catalog: the catalog overview, controls, product cards, URL-backed filters, deferred local filtering, and product-grid rhythm are pinned by shared helper tests
- product detail: the first-paint product hero, price block, action badges, promise lanes, support copy, lazy feedback, and cached related-products panels remain in the shipped product-detail scaffold

Rendering-efficiency rules:

- put route-level background gradients on the route root, not every nested card
- keep hero, metric, feature, and catalog card surfaces behind shared class helpers before adding route-specific details
- keep internal list routes semantic: one table shell, table children inside table nodes, and no wrapper rows inside `tbody`
- degrade glow, hover lift, blur, and drawer travel for mobile, reduced-motion, or low-power review without changing DOM shape

Validation:

- `visual_primitives_test.go` checks primitive coverage, rendered public shell/catalog/product use, and semantic internal table markup
- `render_gap_branches_test.go` continues to cover public feedback, related-products, and input helper branches that share this visual contract

### Warehouse page modules

Warehouse pages should include:

- location hero and regional service message
- promise-speed and service-zone summary
- stocked highlights relevant to that region
- availability drill-down entrypoints
- operational notices only when they help customer decision-making

### Dashboard sections

The dashboard should use five sections only:

- urgent alerts
- low-stock and imbalance queue
- inbound receiving work
- moderation backlog
- quick actions into inventory, transfers, and receiving

### Moderation queue layout

The moderation route should default to a split view:

- left side: filterable queue
- right side: selected comment or request detail

Status filters should include pending, approved, rejected, and flagged. The moderation action area should keep approve, reject, and flag actions visually distinct and easy to confirm.

### Settings page layout

The settings route should group preferences by intent rather than by storage location.

Recommended groups:

- appearance: theme and density
- language: locale and route preference
- workspace: default warehouse and saved-view defaults
- diagnostics: demo affordances and developer-facing toggles

## Reviewer Walkthrough

The first guided demo should run in this order:

1. Open the storefront landing page and show the premium brand direction.
2. Move into the catalog and explain the filterable merchandising posture.
3. Open the Frame Desk product route and highlight warehouse-aware availability.
4. Switch to the warehouse network and explain regional fulfillment differences.
5. Jump into the internal dashboard and frame it as a triage surface.
6. Open inventory to show dense list management and saved-view intent.
7. Visit moderation to show that the public surface is backed by internal review tools.
8. Visit settings to explain theme, locale, and operator preference persistence.

The walkthrough should stay under ten minutes and never require apology for placeholder logic. If a screen does not support the story cleanly, it is not ready for the flagship demo.

## Interaction Rules

### React mock interaction inventory

The two React references currently contribute these concrete interaction patterns that Atlas must account for during parity planning:

- mobile menu: both `design/homepage_store.tsx` and `design/homepage_warehouse.tsx` use a small-screen header trigger that opens an animated route list overlay. Atlas route ownership: public shell routes such as `/shop`, `/shop/:slug`, `/warehouses`, and `/warehouses/:warehouseSlug`, plus the internal shell wrapper across `/app/*`.
- animated drawers and overlays: the store mock uses an animated cart drawer with backdrop dismissal, while both mocks use animated mobile-menu overlays and hover-lift product cards. Atlas route ownership: public product overlays on `/shop/:slug`, public shell navigation overlays, and internal workflow overlays on `/app/inventory/:sku`, `/app/transfers`, `/app/receiving`, `/app/comments`, and the shared `/app/*` shell.
- toasts: both mocks mount a timed toast surface for lightweight confirmation feedback. Atlas route ownership: public follow-up actions on `/shop/:slug` and `/warehouses/:warehouseSlug`, plus internal action feedback on `/app/inventory`, `/app/transfers`, `/app/receiving`, `/app/comments`, and `/app/settings`.
- filter chips: the store mock turns active catalog filters and search terms into removable chips above the product grid. Atlas route ownership: public catalog state on `/shop`, with the same pattern already relevant to internal filter-heavy workspaces such as `/app/inventory` and `/app/warehouses/:warehouseId`.
- image gallery selection: the store product-detail view swaps the hero image from a thumbnail strip. Atlas route ownership: `/shop/:slug`.
- quantity steppers: the store product-detail quantity control and the warehouse reorder flow both use increment or decrement steppers. Atlas route ownership: public buying interactions on `/shop/:slug` and internal replenishment or receiving quantity work on `/app/inventory/:sku`, `/app/purchase-orders/:id`, and `/app/receiving/:id`.
- editable forms: the store mock includes a mocked checkout form, and the warehouse mock includes compact create or edit product forms with metadata, pricing, supplier, and status fields. Atlas route ownership: public buyer-contact and request forms on `/shop/:slug` and `/warehouses/:warehouseSlug`, plus operator CRUD and settings forms on `/app/products`, `/app/products/:slug`, `/app/comments`, and `/app/settings`.
- preview panels: the warehouse item editor keeps a live item-card preview beside the form. Atlas route ownership: `/app/products` and `/app/products/:slug`, where the merchandising editor already owns preview-oriented CRUD work.
- supplier reorder flow: the warehouse mock groups low-stock items by supplier, precomputes reorder quantities, allows row-level quantity edits, and sends a purchase order from the staged list. Atlas route ownership: stock-recovery work on `/app/inventory`, `/app/inventory/:sku`, and `/app/purchase-orders`.

The current interaction inventory is intentionally reference-level only. Route ownership, Atlas adaptation, and exclusion rationale are tracked by the following parity todos so this section stays focused on what exists in the React mocks.

### Parity planning buckets

Before implementation starts, Atlas should treat the current mock interactions in three planning buckets:

- must-port first: public mobile menu, internal mobile nav, timed toast feedback, public catalog filter chips, public product gallery selection, and quantity steppers anywhere Atlas already has a real quantity decision
- Atlas-adapt rather than copy literally: drawer or overlay behavior, editable public forms, editable internal CRUD forms, preview panels, and the warehouse reorder flow because Atlas routes already express those jobs through product requests, inventory recovery, purchase-order creation, receiving, and shared overlay primitives
- reference-only for visual or product-language inspiration: the mock cart drawer as a literal cart, the full mocked checkout route, and decorative hover-lift card motion that does not carry Atlas workflow value on its own

### Destination map

Each interaction should now be treated as one of three implementation destinations:

- direct Atlas port: mobile menu, toasts, filter chips, image gallery selection, and quantity steppers where Atlas already exposes a real quantity choice
- Atlas-adapted equivalent: animated drawers and overlays, editable forms, preview panels, and the supplier reorder flow
- intentionally excluded from literal parity: the mock cart drawer as a cart workflow, the standalone mocked checkout route, and decorative hover-lift card motion as a required product behavior

### Exclusion rationale

The currently excluded mock interactions are excluded for Atlas-native product reasons, not because they are unimplemented:

- literal cart drawer: Atlas public routes currently support request, quote, restock, and product-question workflows rather than a session cart, so a cart-specific drawer would introduce a second public purchase model that the real example does not own
- standalone checkout route: Atlas does not model direct payment capture or order placement from the public shell, so a mocked checkout page would imply backend responsibilities and fulfillment states the example intentionally does not ship
- decorative hover-lift card motion as required behavior: Atlas needs motion that reinforces route hierarchy, overlays, and operational feedback, so treating hover-lift animation as mandatory parity would create avoidable surface churn without improving task completion

### Public mobile menu parity

Atlas now gives the public shell a hydrated small-screen navigation overlay instead of a single fallback shortcut. The menu opens from the shared public header, keeps the existing storefront route set in one overlay, preserves active-route emphasis, and leaves the desktop public nav unchanged.

### Public mobile menu work map

The public mobile menu should continue as five narrowly reviewed tasks:

- open or close state: trigger, dismiss, and close-button behavior stay local to the shared public header
- focus order: first focus target, close target, and return-to-trigger behavior stay tied to the overlay container
- route selection: the overlay must expose the same public route set and active-state treatment as the desktop nav
- overlay treatment: backdrop, stacking, dismissal, and reduced-motion behavior should stay aligned with the shared Atlas overlay rules
- small-screen layout: the overlay card spacing, copy density, and CTA grouping should stay optimized for narrow storefront screens without changing the desktop header

### Motion rules

Motion should be short, directional, and functional.

- Atlas already uses reveal or panel motion where it adds hierarchy safely: public mobile nav, the public buying drawer, the internal workspace drawer, threshold-history and workflow sheets, local toast entry, and restrained card-hover lift on public merchandising tiles
- keep as structural motion: drawer or sheet entry and exit, toast entry, localized pending-state swaps, and light public merchandising hover emphasis
- simplify for Atlas stability: full-route reveal choreography, staggered section entrances that hide SSR content until hydration, and decorative motion on dense internal tables or workflow cards where immediate readability matters more than spectacle
- route transitions should use subtle opacity and vertical offset rather than large travel
- overlays should enter faster than routes and leave even faster
- row selection, filter application, and saved-view restore should favor immediate feedback over decorative animation
- motion should reinforce hierarchy, not call attention to itself

### Reduced-motion behavior

Reduced-motion mode should:

- remove non-essential route transitions
- replace spring-like overlay motion with near-instant fades
- keep focus-ring, selection, and validation feedback fully visible without motion dependency
- preserve timing clarity through state labels and visual hierarchy instead of animation

### Notification patterns

Use notifications sparingly and by severity:

- toast: short-lived confirmation for lightweight actions such as preference saves
- inline banner: non-blocking warnings or route-level issues that affect the current page
- inline field feedback: validation and submission problems tied to one form
- persistent alert card: dashboard-level operational issues that need later action

### Empty, loading, and error states

The example should use different treatments for different failure shapes:

- empty-first-use states should explain what the feature is for and what to do next
- no-results states should preserve filter context and offer a clear reset path
- loading states should use structured skeletons where layout matters and simple copy where it does not
- errors should distinguish retryable async issues from invalid user actions and route-level failures

### Overlay inventory

The first overlay set should include:

- threshold editing modal
- transfer creation modal or side sheet
- receiving discrepancy dialog
- command palette
- row action menu
- quote request confirmation surface

Atlas now also uses Atlas-native panel behavior where the React references implied separate panels instead of inline-only treatment:

- public product detail uses a small-screen buying drawer so the quote or restock action rail does not stay trapped as a static below-fold block on narrow screens
- the shared public and internal headers use sheet-based mobile navigation instead of static collapsed link rows
- threshold history and internal workflow guidance already use sheet or modal presentation rather than forcing every secondary task to live inline beside the primary route body

The first overlay proof points should stay narrow:

- public first-wave pattern: the product-detail buying drawer on `/shop/:slug`
- internal first-wave pattern: the threshold-history and workflow sheet stack on `/app/inventory/:sku`

### Modal stacking rules

Allowed stacked patterns should stay narrow:

- base modal plus one nested discrepancy or confirmation modal
- command palette should close before a full dialog opens
- anchored menus should dismiss before route-level dialogs claim focus

Focus should always return to the most recent meaningful trigger.

### Command and anchored actions

The command palette should navigate to products, SKUs, warehouses, transfers, receiving sessions, and settings. Anchored menus should be reserved for row-specific actions such as open details, create transfer, adjust thresholds, or review moderation context.

## Operational Workspace Rules

### Inventory columns

The default inventory table should include these columns:

- SKU
- title
- warehouse
- available
- cover days
- inbound
- status
- updated

### Inventory filters

The default filter set should include:

- warehouse
- category
- supplier
- stock health
- availability
- search

### Saved-view model

Saved views should preserve:

- selected filters
- sort key and direction
- density mode
- default warehouse context when relevant

The first saved views should be:

- Low stock triage
- East coast shortages
- Desk replenishment
- Accessory watchlist

### Inventory status taxonomy

Use a small status set that can be learned quickly:

- healthy
- watch
- low
- inbound
- blocked

### Inventory bulk actions

The first bulk actions should be:

- create transfer
- assign review tag
- export selection

Avoid destructive bulk actions in the first milestone.

### Dashboard alert priorities

Alerts should sort in this order:

1. blocked or failed receiving work
2. critical low-stock items tied to active products
3. transfer imbalance recommendations
4. moderation items requiring review
5. general informational warnings

### Dashboard quick actions

The dashboard quick-action group should include:

- open low-stock inventory view
- create transfer
- resume receiving session
- review pending comments

## Responsive Rules

### Overall layout

The public side should remain spacious on small screens, while the internal side should prioritize information hierarchy without forcing every dense table into unworkable cards.

### Mobile table strategy

On narrow screens, inventory should switch from full table density to prioritized row summaries that preserve status, warehouse, available units, and the primary action.

### Tablet shell behavior

Tablet layouts should keep the internal navigation collapsible but visible on demand, with filters and saved views remaining one interaction away.

### Desktop shell behavior

Desktop layouts should show the full navigation rail, persistent filter regions where useful, and enough horizontal room for operational scanning without horizontal scroll as the default experience.

## Preferences And Locale Rules

### Theme preference behavior

Theme should default to light on the public storefront and dark in the internal console, while still respecting explicit user preference once chosen.

### Locale preference behavior

Locale should persist independently of route and survive reload. Public routes may later adopt locale-prefixed paths, while the internal console can remain unprefixed for the first milestone.

### Density preference behavior

Density should default to comfortable for the public side and compact for the internal side, with user override remembered for the console workspace.

### Locale catalog scope

The first translated surfaces should be:

- landing route
- catalog route
- product route
- settings appearance and language groups

Operational microcopy can remain selectively translated in the first milestone as long as locale switching is visibly real.

### Locale fallback rules

Fallback behavior should be straightforward:

- default locale: English
- supported showcase locales: English, French, Arabic
- missing translation keys should fall back to English rather than blank output

### RTL audit targets

The first RTL validation pass should cover:

- global shell direction
- filter rail alignment
- stat-card and badge spacing
- button groups and breadcrumb-like navigation
- settings and product hero layout

## SEO And Discovery Rules

### SEO page targets

The public SEO-critical routes are:

- landing page
- catalog page for broad discovery terms
- product detail pages
- warehouse detail pages where locality adds user value

### Canonical URL policy

Canonical links should:

- normalize catalog filter combinations back to the base catalog route unless a later indexed filter strategy is chosen
- always canonicalize product pages to the product route without session or preference noise
- keep warehouse detail pages canonical to their clean slug path

### Structured data coverage

The first structured-data set should include:

- organization data on the landing page
- product data on product routes
- availability data tied to the warehouse-aware product story where appropriate

### Social preview coverage

Social preview images should exist for:

- the landing page
- category or catalog sharing fallback
- product detail routes

### Sitemap coverage

The initial sitemap should include only the public routes that are stable and worth sharing. Internal routes and temporary utility routes should stay out of it entirely.

## Workflow Rules

### Form interaction rules

Public and internal forms should share the same behavioral expectations:

- render a real `form` with server-owned `action` and `method` whenever the workflow already has a normal POST endpoint
- keep the non-JS submit path authoritative for quote, restock, question, moderation, transfer, receiving, settings, and CRUD forms unless a workflow is impossible to express progressively
- use hydration only to add local validation, pending labels, preview or staging state, focus management, and post-submit toast or announcer feedback
- validate early when fields are obviously incomplete
- keep server validation messages field-specific whenever possible
- show a pending state on the active action only
- return focus to the first actionable error when submission fails

### Comment moderation rules

Moderation should use four states only:

- pending
- approved
- rejected
- flagged

Rejected items should remain reviewable internally. Approved items should become eligible for public display. Flagged items should indicate follow-up rather than final removal.

### Quote request flow

The quote path should be triggered by quantity, stock constraints, or team-buying context. The first form should capture only the information needed for follow-up and should confirm that Atlas will respond with availability and commercial guidance rather than immediate checkout.

### Restock request flow

The restock path should appear when stock is low or unavailable. It should collect email and optional regional preference, then confirm that future contact depends on actual availability rather than a guaranteed schedule.

### Availability messaging

Availability language should stay precise and calm:

- in stock
- low stock
- inbound soon
- temporarily unavailable

Avoid fake urgency and avoid exposing raw operational noise that does not help the buyer decide.

### Transfer recommendation story

Transfer recommendations should be justified by visible imbalance, not by opaque automation claims. The system should clearly suggest when one warehouse can support another because of stronger depth or slower local demand.

### Receiving discrepancy story

Receiving discrepancies should separate short shipment, overage, and damaged goods clearly. The operator should always understand whether inventory will become available, blocked, or damaged after reconciliation.

### Internal note behavior

Internal notes should be practical, timestamped, and secondary to the event timeline. They should support operational context without turning into an unstructured chat surface.

## Non-Goals For V1

The first implementation does not need:

- full enterprise permissions depth
- complex vendor management
- exhaustive analytics or BI surfaces
- real payment checkout
- external integrations beyond believable local data

## Immediate Design Decisions Locked

- The example name is Atlas Commerce OS.
- The product vertical is modular workspace and storage systems.
- The public side is editorial and premium.
- The internal side is dense, dark, and action-oriented.
- Warehouse-aware availability is a first-class product story.
- Quote requests, restock requests, and moderation are included in scope.
- The example should optimize for flagship-demo quality over maximum feature breadth.

## Next Non-Technical Decisions To Answer Here

- final visual references and mood board keywords
- customer voice and review tone
- demo walkthrough script for reviewers

---

### DIAGNOSTICS_NOTES

# Atlas Commerce OS Debug Logging Notes

Temporary diagnostics overlays were removed before final release signoff.

## Enable Debug Logging

- set `data-atlas-debug-logs="1"` on the document root before loading Atlas, or set `window.__atlasDebugLogs = true` and reload
- debug logging is only intended for development review and should stay off in normal demos

## What Debug Logging Shows

- bootstrap read success or failure and route fetch cache or invalidation events
- overlay and focus-routing debug events for workflow-heavy screens
- sanitized route and workspace summaries without token, cookie, or secret fields
- revalidation start or completion events for loader-backed flows

## Review Use

- verify a loader-backed route mounts cleanly before checking its specific workflow
- confirm persisted theme, locale, density, and warehouse values on direct entry
- inspect query and params when testing route recovery or internal drill-in links
- use the loader ledger to spot regressions after route-loader changes

---

### FRAMEWORK_COVERAGE

# Atlas Framework Coverage

This file tracks which GoWebComponents surfaces Atlas Commerce OS actually exercises in shipped code today.

Status legend:

- `Implemented`: present in current Atlas code, not just in planning docs.
- `Not currently wired`: available in the framework or planned in Atlas, but not exercised by the current example.

## ui

- [x] `ui.CreateElement` implemented for route components and interactive public product feedback.
- [x] `ui.Hydrate` implemented in the wasm entrypoint for server-bootstrap resume.
- [x] `ui.UseState` implemented behind Atlas local-state wrappers for public feedback and client-only interaction state.
- [x] `ui.UseEffect` implemented behind Atlas wrappers for client-side state synchronization.
- [x] `ui.UseEvent` implemented for public comment-form field updates and submit behavior.
- [x] `ui.UseForm` implemented for the public product comment workflow plus internal product create or update, inventory lane edit, moderation, transfer, receiving reconcile, and settings import or export flows.
- [x] `ui.UseId` implemented in shared Atlas form helpers so CMS inputs, catalog and inventory controls, public quote or restock fields, and public feedback error text all bind through generated IDs instead of ad hoc markup.
- [x] `ui.UsePrevious` implemented for product-editor drafts, the threshold-history overlay refresh summary, and receiving reconcile drafts so Atlas can show the most recent change without keeping duplicate snapshot state by hand.
- Status: `ui.Render` is not wired in Atlas; the shipped browser entry intentionally hydrates the server bootstrap instead of using a pure client render path.
- Status: `ui.RenderToString` is exercised by shared Atlas render tests, while the shipped server response still emits an HTML shell plus bootstrap and hydrates the app into `#app`.
- [x] `ui.AsyncBoundary` now isolates the deferred public promise-lanes module and the purchase-order or receiving side-panel stat islands so panel loading and failure states stay local.
- [x] `ui.UseWorkerTask` now powers saved-view import validation on the settings route, pushing JSON parse and validation work into a dedicated worker while the operator keeps editing the import payload.
- Limitation: `ui.UseTask` remains deferred after evaluation; saved-view export, workspace snapshot export, and current diagnostics snapshot paths are still small enough to stay synchronous until Atlas gains a heavier non-worker background job.
- [x] `ui.UseChannel` now drives a shell-level Atlas toast bus: the app shell subscribes once, while distant flows such as public comment submission and internal panel refresh can broadcast completion notices without routing those events through query state.
- [x] `ui.UseReducer` now drives replenishment-order staging in the inventory purchase-order modal and receiving closeout-stage guidance in the reconcile form, so both workflows can show reducer-owned status summaries without abandoning progressive form posts.
- [x] `ui.Fragment` now groups inventory queue cells, warehouse network table cells, and repeated section-meta copy blocks without introducing extra wrapper nodes in the rendered table or card markup.
- [x] `ui.UseLazyNode` now defers the below-the-fold public feedback module and the secondary purchase-order or receiving detail rails until the primary route body is stable, while still rendering synchronously on the server for direct entry.
- [x] `ui.ErrorBoundary` now contains the public promise-lanes enhancement island plus the purchase-order and receiving side panels, so a broken enhancement panel falls back locally instead of collapsing the full Atlas route.
- [x] `ui.UseRef`, `ui.UseFocusManager`, and `ui.UseFocusTrap` now drive threshold-overlay focus restore plus the transfer, receiving, and moderation confirmation dialogs, so Atlas captures the opener and traps keyboard focus inside those flows instead of relying on manual markup behavior.
- [x] `ui.UseFocusTrap` now also wraps the replenishment-order modal, replacing the old checkbox-and-peer visibility trick with a stateful dialog that traps focus, restores the opener, and keeps the purchase-order workflow keyboard-contained.
- [x] `ui.UseCompositeNavigation` now powers the settings-route saved-view browser, giving Atlas one real listbox-style operator control with ArrowUp or ArrowDown, Home or End, and typeahead-driven inspection of saved workspace presets.
- [x] `ui.AccessibleOverlay`, `ui.Overlay`, `ui.UseOverlayStack`, `ui.Portal`, and `ui.PortalTarget` now back the shared Atlas overlay layer: the threshold-history route sheet renders through a portal host outside the shell, the transfer, receiving, and moderation confirmations reuse the same stack-aware modal wrapper, and Atlas now also uses that host for small-screen public buying drawers plus small-screen internal quick-action drawers.
- [x] `ui.UseAnnouncer` now sits at the Atlas shell boundary and announces route changes, query-string notice banners, and shell toast updates, which covers public comment submit plus threshold, receiving, and moderation success messaging without sprinkling separate live regions across each route.
- [x] `ui.UseDeferredValue` and `ui.UseDebounced` now back hydrated filter controls on `/shop`, `/app/inventory`, and `/app/warehouses/:warehouseId`, so Atlas can preview filtered lists locally while debouncing query-string replacement for deep-linkable filter state.
- [x] `ui.UseTransition` and `ui.StartTransition` now cover inventory saved-view application, settings density preview toggles, and the high-churn inventory or warehouse filter setters, so Atlas can keep dense internal rerenders non-urgent while still exposing local pending state.
- [x] `ui.UseThrottled` powered the temporary diagnostics shell panel used during rewrite review; release cleanup removed that panel before signoff, while keeping throttling available for future non-production instrumentation.
- Status: `ui.UseNavigate`, `ui.UseTask`, and `ui.Lazy` are not currently exercised by Atlas code even though some older notes listed them as implemented.
- Status: `ui.UseContext` remains planned rather than wired.

## router

- [x] History-router navigation implemented through `router.NewHistoryRouter(...)`, document-link interception, and `router.HydrateMount(...)`.
- [x] Route metadata implemented through router options for title, description, and canonical updates.
- [x] Route params and query-backed route state implemented for product, warehouse, inventory, and threshold-history paths.
- [x] Route loaders implemented across public and internal Atlas routes, including the threshold-history overlay loader.
- [x] Route redirects implemented for `/app` to `/app/dashboard`.
- [x] `BeforeEnter` auth guards implemented for internal routes.
- [x] `BeforeLeave` unsaved-change guard implemented for the product editor route.
- [x] Nested layout route usage implemented for the inventory detail and threshold-history flow.
- Status: hash routing is no longer part of the shipped Atlas path and should not be treated as current framework coverage.

## html

- [x] Semantic HTML layout and form markup implemented across public and internal routes.
- [x] Progressive HTML form posts implemented for public and internal mutation flows.
- Limitation: dense internal table and grouped-form markup still need a broader parity and accessibility pass.

## i18n

- [x] Locale bootstrap, document `lang`, and RTL direction handling implemented for SSR entry and hydration resume.
- Limitation: package-level translation resources and localized content bundles are not currently wired.

## state

- [x] `state.UseComputed` implemented for internal shell summaries and route badges derived once per route payload and reused by the header and hero.
- [x] `state.UseAtom` implemented for shell-wide presentation preferences so locale, density, and default-warehouse context have one shared ownership point during hydration.
- Status: `state.UseDerived` is not currently exercised by Atlas code; current Atlas interaction state is still mostly local hook state plus server bootstrap.
- Status: snapshot export, import, and broader shared-state ownership remain planned.

## fetch

- [x] Atlas route payload and request loaders now use shared cached-resource helpers in `shared/atlas/resource_cache*.go` via `fetch.LoadCached`, shared SSR cache bootstrap seeding, and the same mutation invalidation matrix that backs `ui.UseCachedResource`.
- [x] Public comments, related products, warehouse side data, and purchase-order or receiving detail rails now read through Atlas cached-resource hooks, so hydrated repeat-open surfaces can reuse secondary data without waiting on a full route reload.
- [x] `fetch.UseResource` now powers the deferred public product promise-lanes island and the purchase-order or receiving detail side-panel stat islands, each wrapped in `ui.AsyncBoundary` so retries and failures remain panel-local.
- [x] `fetch.Fetch` now drives the public comment POST path and the purchase-order or receiving panel refresh buttons, so imperative follow-up refresh work stays on the framework fetch path instead of raw browser client calls.
- [x] Atlas cache diagnostics now emit `bootstrap.read.ok`, `route.fetch.ok`, `route.fetch.cache.hit`, and `route.cache.invalidate` into the browser console and `window.__atlasDebugLast`, while shared Atlas tests cover route-key stability and mutation invalidation target mapping and server tests cover fresh direct-entry SSR bootstrap after mutation.
- Status: `ui.UseFetch` is not currently exercised by Atlas code; imperative refresh now uses `fetch.Fetch`, while route or panel loading stays on `fetch.UseResource` or shared cached resources.

## devtools

- [x] Temporary `devtools.Panel` and snapshot summary surfaces were removed from shipped Atlas UI during release cleanup.
- [x] Internal development inspection now uses opt-in client debug logging (`data-atlas-debug-logs` or `window.__atlasDebugLogs`) rather than query-flagged diagnostics overlays.

## bootstrap and SSR

- [x] Request-time bootstrap generation implemented for public and internal routes.
- [x] Shared bootstrap decoding implemented for client hydration.
- [x] Inline bootstrap script rendering implemented for normal SSR responses.
- [x] External bootstrap reference mode implemented as a proof of concept for the inventory threshold-history route.
- [x] SQLite-backed server data and request-time route payload generation implemented for SSR entry.
- Status: shared server-side rendering of the Atlas component tree itself is not currently wired; the shipped server response is still a bootstrap-first shell.

## framework adoption rule

- Atlas should prefer a shipped GoWebComponents surface whenever the rewrite needs state ownership, route loading, form lifecycle, overlays, scheduling, or async data reuse.
- Manual browser state, hand-rolled request caches, and ad hoc DOM wiring are acceptable only when the framework does not yet expose the needed primitive or when Atlas is intentionally documenting a gap.
- Every future Atlas rewrite step should treat a framework primitive as the first option to evaluate, not the last cleanup pass after a custom implementation has already landed.

---

### INTERNAL_NOTES

# Internal Surface Notes

The internal surface is responsible for triage, dense inventory handling, receiving, transfers, moderation, and preference persistence.

Current focus:

- dashboard triage
- inventory scanability
- moderation review workflow
- settings and operator preferences

Next implementation targets:

- inventory table component
- dashboard card system
- moderation queue and detail split
- receiving and transfer workflow surfaces

---

### KNOWN_LIMITATIONS

# Known Limitations

Current Atlas limitations after the native Go server milestone:

- mock auth is now cookie-backed, but it is still a demo session model rather than a real identity system
- locale handling remains selective and does not yet ship a full translation bundle across every internal micro-surface
- the broader accessibility pass is still incomplete even though route-entry, form, and smoke coverage exist
- several client-side resume behaviors still lean on browser storage in addition to server-backed bootstrap state
- the page tree is not rendered on the server: responses carry route head metadata and the bootstrap payload around an empty `<div id="app"></div>`, so routes are blank until wasm hydration runs

What is now shipped:

- sqlite persistence with numbered startup migrations
- request-time route metadata and bootstrap payloads from the example 86 Go server
- public and internal JSON endpoints behind one process
- branded recovery pages for missing routes and records
- mock sign-in and recovery flow for internal routes
- CSRF enforcement for public and internal write endpoints

---

### LOCALIZATION_NOTES

# Atlas Commerce OS Localization Notes

Atlas treats localization as a real showcase capability without forcing every internal micro-surface into full translation on day one.

## SSR-Safe Theme Hydration

- SSR should render the same theme and density hints that the client applies to the document element before hydration starts
- cookie or bootstrap hints may seed the first paint, but explicit user preference remains the long-lived source once the app resumes
- the goal is to avoid a visible theme flash between server HTML and client hydration

## Locale-Aware Route Strategy

- public routes may adopt locale-prefixed paths when the server-rendered variant lands
- the internal console can remain unprefixed in the first milestone to keep operational routing simpler
- locale itself still persists independently of the current route and survives reload

## Translation Scope

The first translated Atlas surfaces should be:

- landing route
- catalog route
- product route
- settings appearance and language groups

Operational microcopy may remain selectively translated in the first milestone as long as locale switching is visibly real and consistent.

## RTL Coverage

The first RTL validation pass should cover:

- global shell direction
- filter rail alignment
- stat-card and badge spacing
- button groups and breadcrumb-like navigation
- settings and product hero layout

Atlas already uses Arabic as the proof locale for right-to-left layout review.

## Cached Resource Usage

- repeated detail lookups such as product availability and warehouse promise lanes are the best future candidates for cached-resource reuse
- Atlas should only adopt client caching where it improves route re-entry or repeated drill-in without obscuring the authoritative loader refresh path

## Live Activity Behavior

- the first live-activity story remains simulated rather than socket-backed
- dashboard alerts, logistics activity timelines, and diagnostics snapshots provide the current proof of activity without requiring a real event stream
- future channel or task-based updates should be additive and should not replace the existing loader and timeline model until they can preserve deterministic demos

---

### MANUAL_TESTING

# Atlas Commerce OS Manual Testing

Open:

- `http://127.0.0.1:8096/shop`
- `http://127.0.0.1:8096/warehouses`

## Current Smoke Checklist

- Load the storefront landing or catalog route and confirm the SSR shell renders without fallback errors.
- Navigate to `Shop` and confirm the catalog route remains stable after hydration.
- Navigate to `Frame Desk` and confirm the product route shows warehouse-aware pricing and support surfaces.
- Navigate to `Warehouses` and confirm the public shell remains stable while the route content changes.
- Navigate to `Ops dashboard` and confirm the internal shell route loads.
- Move through `Inventory`, `Comments`, and `Settings` and confirm each route shows the expected surface label and stats.
- On `Inventory`, switch to `East coast shortages`, type a search query, reload the page, and confirm the saved view, warehouse filter, and query resume.
- Navigate to the inventory SKU detail route and confirm warehouse breakdown stats render for the focused SKU.
- Navigate to `Transfers` and `Receiving` and confirm the workflow guidance panels render under the internal shell.
- Open `Settings`, toggle theme and locale buttons, and confirm the local state readout updates immediately.
- On `Settings`, change density or warehouse and confirm the inline success message appears.
- On `Settings`, use `Reset workspace defaults` and confirm density and warehouse return to the defaults.
- On `Inventory`, use `Clear resume state` and confirm saved-view, warehouse, and query context return to defaults.
- Open an unmatched hash route manually and confirm the route catch-all shows the known entrypoints panel.

## Manual Playwright Script: SSR And Metadata Checks

Run this script against the live Atlas server (`go run ./examples/server/atlas-commerce-os/server`) and verify each assertion in Chromium before demo signoff.

1. Open `/` and confirm:
   - `<title>` is `Atlas Commerce OS`
   - `meta[name="description"]` is present and non-empty
   - `link[rel="canonical"]` points to `/`
   - server HTML already includes `<div id="app"></div>` plus `id="__ATLAS_BOOTSTRAP__"` before hydration
2. Open `/shop` and confirm:
   - `<title>` is `Atlas Shop`
   - canonical points to `/shop`
   - metadata remains stable after a hard reload and direct-entry navigation
3. Open `/shop/frame-desk` and confirm:
   - `<title>` starts with `Atlas`
   - canonical points to `/shop/frame-desk`
   - `meta[name="description"]` remains route-specific after reload
   - any `script[type="application/ld+json"]` payload is valid JSON when parsed in DevTools
4. Open `/warehouses` and `/warehouses/new-jersey-hub` and confirm:
   - each route exposes route-specific title, description, and canonical values
   - direct URL entry (new tab) renders stable metadata without requiring intermediate navigation
5. For each public route above, confirm hydration-safe entry:
   - no console errors during first load
   - no route-shell collapse between first paint and hydrated state
   - route metadata values remain unchanged after hydration settles

## Reviewer Checklists

### Persisted settings reviewer checklist

- Seed theme, locale, density, and default warehouse in browser storage before opening `#/app/settings`.
- Confirm the page reflects the stored values immediately on entry.
- Confirm the document element keeps `data-atlas-theme`, `data-atlas-locale`, `data-atlas-density`, and `data-atlas-default-warehouse` aligned with the visible controls.
- Confirm `Reset workspace defaults` restores density and warehouse to the default values.

### Direct-entry QA checklist

- Open `#/app/settings` directly and confirm persisted preferences render without first visiting the landing route.
- Open `#/app/inventory` directly and confirm the saved view, warehouse, and query context restore correctly.
- Open an unmatched route and confirm the recovery panel links back into a valid surface.

### RTL verification note

- Set locale to Arabic and confirm the document direction flips to `rtl`.
- Confirm labels, pills, and cards still look balanced in the settings and inventory surfaces.
- Confirm the light and dark themes both remain readable in RTL mode.

### Inventory resume QA checklist

- Persist `East coast shortages`, `new-jersey-hub`, and a query string, then open `#/app/inventory` directly.
- Confirm the saved-view button, warehouse summary, and query input all restore correctly.
- Use `Clear resume state` and confirm the view resets to `Inventory default`, the default warehouse, and an empty query.

### Nested route shell checklist

- Open `#/app/inventory/studio-console` directly and confirm the inventory route shell stays mounted above the SKU detail route.
- Open `#/app/transfers/tr-2048`, `#/app/purchase-orders/po-1042`, and `#/app/receiving/illinois-accessories-042` directly and confirm the internal shell remains stable while the detail route changes underneath it.
- Open `#/warehouses/new-jersey-hub` and `#/warehouses/new-jersey-hub/availability/frame-desk` directly and confirm the public shell keeps the Warehouses section active across both routes.

### Shared-state persistence across routes checklist

- Persist theme, locale, density, and default warehouse, then move from `#/app/settings` to `#/app/dashboard` and `#/shop/frame-desk` to confirm document attributes and shell tone remain aligned.
- Persist inventory saved view, warehouse, sort, query, and density override, then move from `#/app/inventory` to `#/app/inventory/studio-console` and back to confirm the workspace state remains intact.
- Enable `window.__atlasDebugLogs = true`, reload an internal route, and confirm debug events match the same preference and route state shown by the visible shell.

### Overlay-backed workflow checklist

- Open SKU threshold editing, save a threshold update, and confirm the activity timeline and toast both update.
- Open transfer confirmation, approve the transfer, and confirm the transfer state plus toast update without dropping the current route.
- Open the receiving discrepancy side sheet, change classification, and confirm the receiving copy updates after submission.
- Open the moderation confirmation flow and confirm closing the overlay returns focus to the action that launched it.

### Derived inventory summary reviewer guide

- Use `East coast shortages` and `new-jersey-hub` together to confirm the derived summary shifts to `Promise risk`.
- Clear the resume state and confirm the summary returns to the calmer default inventory posture.
- With debug logging enabled, open `#/app/dashboard` and compare logged inventory summary events against visible inventory route behavior.

## Reviewer Demo Steps

- Persisted settings demo: seed browser storage, open `#/app/settings`, verify the controls and document attributes match, then use `Reset workspace defaults`.
- Direct route-entry demo: open `#/app/inventory` and `#/app/settings` directly from a fresh tab and confirm state resumes without first visiting the landing route.
- Saved-view resume demo: select `East coast shortages`, enter a query, reload, verify the restored workspace, then clear the resume state.
- Overlay workflow demo: show threshold, transfer, and receiving overlays in sequence so reviewers can see each workflow return cleanly to its parent route.

## Inventory Persistence Test Notes

- Browser resume currently covers saved view, warehouse, and query state.
- Validation should confirm both visible UI copy and the actual input value after direct route entry.
- Reset coverage should verify that clearing resume state removes the persisted query as well as the saved-view context.

## Additional Reviewer Notes

### Comment-thread moderation QA note

- Confirm the public thread reads like believable buyer and specialist communication rather than scaffold bullets.
- Confirm the moderation side panel explains review timing without overpowering the product story.
- Confirm the thread still feels secondary to the product hero and request surfaces.

### Dashboard handoff QA note

- Confirm the dashboard quick-handoff links read like next actions rather than generic shortcuts.
- Confirm each handoff tile points to a route that resolves the problem described in the card copy.
- Confirm the dashboard still feels triage-first rather than analytics-first.

### Light-mode review checklist

- Confirm light-mode cards keep enough separation from the background.
- Confirm helper text, chips, and badges remain readable without relying on dark-theme contrast.
- Confirm the storefront and internal surfaces still feel distinct in light mode.

### Dark-mode review checklist

- Confirm dark-mode cards preserve hierarchy instead of flattening into one tone.
- Confirm badges, pills, and action links still stand out against the background.
- Confirm long-form copy remains readable on the richer gradient surfaces.

## What This Pass Verifies

- the new example wasm binary loads correctly
- the route registry is mounted and navigable
- the public and internal shells are distinct
- the new settings controls and operational workflow routes render without errors
- persisted settings and inventory resume state survive reload on key routes
- the example can keep growing without falling back to a blank page on unknown routes

---

### MIGRATIONS

# Atlas Commerce OS Migration Approach

The first persistence milestone should use simple numbered SQL migrations.

## Direction

- keep migrations as plain SQL files
- apply them in lexical order
- record applied versions in a small migrations table
- rebuild from seed during early development when version drift is acceptable

## Early Versioning Rule

Use a simple naming scheme:

- `001_initial_schema.sql`
- `002_saved_views.sql`
- `003_preferences.sql`

Until a request-time server exists, migration work can remain documented and manual. Once sqlite persistence lands, the example should promote this into a real startup migration step.

---

### MILESTONE_SUMMARY

# Atlas Commerce OS Milestone Summary

## Current milestone state

- Atlas now has premium public and internal shell framing instead of placeholder route panels.
- Settings resume covers theme, locale, density, and default warehouse with direct route entry support.
- Inventory resume covers saved view, warehouse, and query with a clear-resume action.
- Internal app routes now use nested layouts, redirect cleanly from `/app`, and preserve inventory context during SKU drill-in.
- Warehouse detail now acts as a second layout route showcase, keeping warehouse summary rails, filters, and local create or replenishment workflow context mounted while a nested warehouse item workspace is open.
- Shared atoms and derived inventory summaries now drive settings, inventory, and dashboard continuity across routes, the shell-level presentation state is bootstrap-seeded and mirrored to browser storage, a route-scoped workspace atom carries inventory filter labels, matched saved-view context, warehouse filter summaries, and dashboard or route badge rollups, and shared inventory or warehouse workspace snapshot helpers now feed both the shell atom and route bodies instead of duplicating those derived counts in multiple files.
- Atlas now has a real accessible threshold overlay plus a portal-mounted transfer confirmation flow wired into the internal workspace.
- Receiving now has a discrepancy side sheet, and moderation actions now confirm through a real overlay instead of inline-only state changes.
- Dashboard, inventory, SKU detail, transfers, and receiving now use real route loaders with loading and error fallbacks, query-aware inventory reloads where applicable, and a shared toast viewport for internal action feedback; transfer, purchase-order, and receiving detail routes now expose explicit `router.UseRevalidator` refresh actions instead of custom per-route reload plumbing.
- Internal warehouse list and warehouse detail routes now exist with loader-backed pressure summaries, staffing/backlog stats, and direct-entry browser coverage.
- Public warehouses now include a warehouse detail route plus a warehouse-specific availability route, and both are covered by direct-entry and recovery browser flows.
- Public catalog browsing now keeps sort and pagination in the URL, product detail now includes warehouse promise lanes plus related-product routing, and public quote, restock, and comment forms now expose real validation states.
- Atlas now includes a baseline public screenshot pack for landing, catalog, product, and warehouses routes in both light and dark desktop themes.
- Internal routes no longer ship the hidden diagnostics mode; release cleanup removed the temporary panel, snapshot cards, and devtools overlay, leaving opt-in debug logging hooks for development review.
- Route-level document attributes now distinguish public, internal, and recovery surfaces so Atlas can tune light and dark accents separately, apply nested shadow tiers, keep keyboard motion parity with hover states, and tighten Arabic heading rhythm.
- Atlas now includes narrow internal density screenshots plus a mobile-rail cleanup pass, and reviewer docs now call out contrast and motion checks for the current visual system.
- Manual QA, release-readiness, and future server-integration docs now cover nested route shells, shared-state persistence, overlay workflows, derived inventory summaries, focused browser smoke commands, and post-milestone cleanup notes.
- Milestone-four cleanup now includes the resume, release diagnostics cleanup, contrast, and reviewer-guidance alignment pass so the Atlas backlog reflects shipped surfaces instead of earlier scaffold assumptions.
- Inventory workspace now persists sort state, supports a route-local density override, renders compact row treatment, and exposes keyboard shortcut hints with browser coverage.
- SKU detail now renders a real threshold history timeline and appends fresh threshold edits directly into the route after overlay saves.
- Transfers and receiving now include detail routes, explicit approval or classification state changes, and a shared logistics activity timeline with direct-entry browser coverage.
- Purchase orders now include list and detail routes, approval affordances, and inbound shipment rows tied to real route data.
- Product hero, dashboard, comment thread, and recovery surfaces now read like product UI rather than framework scaffolding.
- Reviewer docs now cover persisted settings, direct entry, RTL review, inventory resume, screenshot naming, and release-readiness checks.
- Milestone-five cleanup now includes explicit review decisions, performance checkpoints, screenshot-backed visual artifacts, and a reviewer-facing demo checklist so the remaining backlog reflects truly unfinished work rather than closed planning questions.
- Testing operations now include a Windows command matrix, stale-wasm guard, local task grouping, and change-triggered review checklist; rewrite planning now locks route, SSR, bootstrap, cache, derived-state, framework-coverage, performance, shared HTML, accessibility, localization, and recovery invariants before the next route-family pass.

## Still open

- Async-resource, cached-resource, and full SSR bootstrap coverage remain open.
- SQLite-backed persistence and hydration reuse remain future milestones.
- Full browser-matrix and end-to-end accessibility validation still need broader coverage.

---

### OPERATIONS_NOTES

# Atlas Commerce OS Operations Notes

This file records the concrete internal workflow rules that Atlas already follows across inventory, purchasing, transfers, receiving, moderation, and interaction layers.

## Inventory Workspace

### Main table columns

- SKU
- title
- warehouse
- available
- cover days
- inbound
- status
- updated

### Sorting behavior

- the first inventory sorts should prioritize available units, stock-status grouping, and freshness of updates
- future revenue-importance or cover-days sorts can extend the same contract without replacing the operator-first defaults

### Filtering behavior

- warehouse
- category
- supplier
- stock health
- availability
- search
- moderation or review tags when Atlas expands review workflows into the inventory surface

### Saved views

Saved views preserve:

- selected filters
- sort key and direction
- density mode when the workspace needs it
- default warehouse context when the filter set is region-specific

Initial saved views:

- Low stock triage
- East coast shortages
- Desk replenishment
- Accessory watchlist

### Bulk actions

The first bulk actions should stay narrow:

- create transfer
- assign review tag
- export selection

Avoid destructive bulk mutation in the first milestone.

## SKU Detail Workflows

### Threshold editing

- reorder points and safety stock should edit through a focused modal surface so the operator keeps SKU context
- a saved threshold change should append to the threshold history timeline immediately in the current route

### Stock adjustment behavior

- manual adjustments should capture quantity delta, reason, and affected warehouse context
- invalid quantities or impossible warehouse transitions should project field-specific errors back into the adjustment surface
- adjustment summaries should feed the same audit timeline language used by transfer and receiving workflows

### Internal notes

- SKU notes stay practical, timestamped, and secondary to the event timeline
- notes should support operator context without turning the route into a chat surface

## Purchase Orders, Transfers, And Receiving

### Purchase-order list and detail

- the list route should keep status, vendor, ETA, inbound rows, and approval posture visible before drill-in
- the detail route should keep vendor context, approval state, and inbound shipment rows together in one route

### Purchase-order creation and approval

- the first Atlas pass focuses on structured approval and hold state rather than a full create-from-scratch vendor wizard
- approval flows should return explicit workflow state so route revalidation can refresh the current screen without guessing

### Transfer recommendation logic

- transfer recommendations should be justified by visible stock imbalance, not opaque automation claims
- the UI should explain when one warehouse can support another because of stronger depth, slower demand, or promise-recovery pressure

### Discrepancy resolution overlays

- receiving discrepancies belong in a side sheet so the receiving queue context stays visible while the operator records the mismatch outcome
- mismatch handling should clearly separate short shipment, overage, and damaged goods

### Optimistic versus authoritative updates

- Atlas can show local workflow state and confirmation copy immediately for demo clarity
- loader-backed routes remain the authoritative view after revalidation, especially for receiving, transfer, and purchase-order state changes

## Forms And Validation

### Common public form primitives

- visible label
- concise helper text
- field-keyed validation copy
- inline success confirmation
- pending state on the submitted action only
- moderation or follow-up message when the form does not publish directly

### Moderation-safe public submission

- public question submissions stay moderation-aware and should tell the buyer that review may delay visibility
- Atlas uses product questions rather than a full public review platform in v1

### Internal form patterns

- dense modal forms for focused edits such as thresholds or destructive confirmations
- side sheets for reconciliation workflows where the list context matters
- inline page-level controls for fast state changes like sort, filters, and simple approvals

### Validation projection rules

- server or workflow validation should map back to field-specific UI copy whenever possible
- route-level summaries should only appear when a form-wide failure cannot be expressed through one field

### Submit-intent behavior

- distinct intents such as approve, reject, hold, receive, or save draft should keep separate buttons and separate confirmation copy
- pending states should stay attached to the clicked action rather than dimming the whole route unnecessarily

## Overlays, Portals, And Shortcuts

### Overlay inventory

Atlas currently needs:

- threshold editing modal
- transfer confirmation dialog
- receiving discrepancy side sheet
- moderation confirmation dialog
- toast viewport for non-blocking internal feedback
- optional debug logging hooks via `data-atlas-debug-logs` or `window.__atlasDebugLogs`

### Stacking rules

- stacked overlays should stay narrow: one primary modal or sheet plus at most one nested confirmation
- command or anchored menus should dismiss before a modal claims focus
- focus returns to the most recent meaningful trigger

### Side-panel strategy

- sheets are the preferred pattern when Atlas needs to preserve list or queue context during resolution work
- centered dialogs remain better for short confirmations or threshold edits that do not need list context visible

### Destructive confirmation patterns

- reject, cancel, hold, and similar irreversible actions require explicit confirmation with clear record context and next-state copy

### Notification and inline feedback

- toast: short-lived confirmation for lightweight actions such as preference or workflow saves
- inline banner or copy: route-local warnings and validation outcomes
- persistent alert card: operational backlog or dashboard-level issues that need later action

### Command palette and anchored actions

- the command palette should navigate to products, SKUs, warehouses, transfers, receiving sessions, purchase orders, comments, and settings
- anchored row actions should stay reserved for row-specific open, transfer, threshold, or moderation actions instead of overloading primary table cells

---

### OVERLAY_NOTES

# Atlas Commerce OS Overlay Notes

Atlas now uses one shared overlay host for secondary workflow UI:

- `#atlas-overlay-root` in `atlas-commerce-os.html`

## Current overlay rules

- SKU threshold editing uses `ui.AccessibleOverlay` for a centered modal dialog.
- Transfer confirmation uses `ui.Portal` when a lightweight secondary confirmation surface is enough.
- Receiving discrepancy handling uses `ui.Overlay` with `OverlayKindSheet` so the queue can stay mentally present while the operator resolves a mismatch.
- Moderation actions use `ui.AccessibleOverlay` so approve, reject, and flag decisions confirm before queue state mutates.
- Internal action feedback uses a portal-backed toast viewport so success state can appear without disturbing the current route.

## Stacking guidance

- Modal-style Atlas overlays should trap focus, close on escape, restore the opener, and mark the background shell inert.
- Sheet-style Atlas overlays should still use the shared portal host and keep dismissal behavior consistent with modals.
- Secondary confirmations should prefer the shared portal host over ad hoc DOM mounts inside route content.
- Browser coverage should prove open, action, dismiss, escape, and focus-return behavior for every new overlay workflow.

---

### PERFORMANCE_CHECKPOINTS

# Atlas Commerce OS Performance Checkpoints

Use these checkpoints when Atlas changes route shells, diagnostics, overlays, or loader-backed surfaces.

## Build And Startup

- Atlas wasm should rebuild successfully using the `Build` command above into `examples/static/bin/atlas-commerce-os.wasm`.
- Public landing, catalog, and product routes should mount without blank intermediate states.
- Internal dashboard and inventory routes should mount cleanly on direct entry with persisted preferences applied.

## Loader And Interaction Checkpoints

- Revalidate each loader-backed route at least once and confirm the loader revision changes in place.
- Open at least one modal overlay and one side sheet, then confirm route context remains stable with debug logging both disabled and enabled.
- Inventory query, saved-view, sort, and density changes should remain responsive without dropping the existing workspace state.

## Review Baselines

- Run the focused Atlas smoke coverage from `examples/`.
- Recheck the mobile density screenshots after any spacing or shell change.
- Recheck debug-log output after any loader, shared-state, or route metadata change.

### Product Detail Baseline: `/shop/frame-desk`

- direct-entry baseline: use `/shop/frame-desk` so SSR and hydration both exercise the flagship public product route
- first-paint baseline: the product hero, warehouse-aware promise copy, quote or restock entry points, and the core buyer action rail should already be visible in server HTML before hydration resumes
- localized-rerender baseline: the promise-lanes async island, cached related-products rail, comment thread and submission form, and small-screen buying drawer may rerender locally, but the public shell, hero, and route metadata should stay visually stable while those panels refresh
- interaction baseline: open the buying drawer, trigger public comment validation, scroll into the feedback block, and revisit the same route to confirm repeat-open secondary panels stay responsive without forcing a full-route redraw
- comparison rule: future lazy or cached product-detail work should be compared against this baseline and should preserve SSR-readable first paint plus route-local rerender boundaries rather than regressing to whole-page pending states

### Dashboard Baseline: `/app/dashboard`

- direct-entry baseline: use `/app/dashboard` first with debug logging disabled, then optionally repeat with debug logging enabled to inspect event flow against the same route payload
- first-paint baseline: SSR should already show the internal shell header, hero badges, dashboard summary strip, admin-flow cards, alerts summary, and the buyer-inbox, transfer-watch, and receiving-exceptions sections before hydration resumes
- localized-rerender baseline: shared shell badges, the settings-side preference form, and moderation actions may update locally, but the dashboard route body should not collapse into a whole-shell pending state when those adjacent surfaces change
- interaction baseline: direct-enter the route, open inventory and comments handoff links, and stage a settings or moderation edit to confirm dashboard triage content stays readable while shell-state surfaces update around it
- comparison rule: future shared shell state or route-summary work should keep the dashboard triage-first and preserve these scoped update boundaries instead of turning the route into a generic KPI wall or a full-page rerender hotspot

### Products Baseline: `/app/products`

- direct-entry baseline: use `/app/products` for the merchandising list and then drill into `/app/products/frame-desk` so the current list-plus-editor flow is captured before a richer CRUD or preview rewrite lands
- first-paint baseline: SSR should already show the shell header, route summary strip, static filter bar, product count cards, merchandising workflow cards, the current product list, and the create-form rail before hydration resumes
- localized-rerender baseline: the product editor form, dirty-state change summary, delete action, and unsaved-change guard may update locally, but the products list route should remain stable while the editor state changes and the surrounding shell badges refresh
- interaction baseline: direct-enter the list, filter or sort once, open the `frame-desk` editor, change copy fields without saving, and return through the guarded navigation path to confirm the current editor flow stays local and does not invalidate the full merchandising shell
- comparison rule: future CRUD-table, live-preview, or richer editor work should preserve this current split between list-level route context and editor-local updates instead of turning basic edit interactions into full-route redraws

### Inventory Baseline: `/app/inventory`

- direct-entry baseline: use `/app/inventory` and then drill into `/app/inventory/frame-desk` plus the threshold-history overlay so the current inventory list, SKU detail, and overlay-backed workflow stack are all part of the same baseline
- first-paint baseline: SSR should already show the shell header, route summary strip, inventory filter bar, triage summary band, queue table, saved-view rail, and the core inventory operations rail before hydration resumes
- inventory loading policy: keep the list-route summary band, filter model, triage band, dense queue table, saved-view rail, and current-view workspace rail first-paint because they define the operator's immediate queue understanding and route scope
- inventory loading policy: treat threshold-history, transfer recommendations inside that sheet, and replenishment modal state as secondary route-local workflows; they should stay scoped to the SKU detail route or overlay and may refresh lazily or through cached request data without blocking the surrounding inventory shell
- localized-rerender baseline: saved-view application, debounced filter changes, threshold-history overlays, replenishment modal state, and route-side workspace summaries may update locally, but the inventory route should keep the broader shell and non-active panels stable while those interactions settle
- interaction baseline: apply a saved view, type into the inventory query, open a SKU detail route, open threshold history, and stage a replenishment flow to confirm the current inventory workspace stays responsive without degrading into full-route pending or losing resume context
- comparison rule: future inventory-side enhancements should preserve this route-local state model, especially the current split between list context, SKU drill-in, and overlay or modal workflows, instead of turning one dense interaction into a whole-page rerender

### Warehouse Baseline: `/app/warehouses/:warehouseId`

- direct-entry baseline: use `/app/warehouses/new-jersey-hub` and then drill into `/app/warehouses/new-jersey-hub/items/frame-desk` so the parent warehouse route and the nested item workspace are profiled as one layout-driven flow
- first-paint baseline: SSR should already show the shell header, warehouse summary strip, warehouse filter bar, facility stats, warehouse-item roster, local create or replenishment workflow entry points, and the side rail with active workspace context before hydration resumes
- localized-rerender baseline: warehouse filter changes, nested item-lane editors, replenishment panel state, and the mounted item outlet may update locally, but the parent warehouse summary rails and route-scoped context should stay mounted while the nested item workspace changes
- interaction baseline: direct-enter the warehouse route, apply one warehouse filter, open the nested `frame-desk` item route, stage a lane edit or replenishment action, and return to the parent roster to confirm warehouse-scoped context survives the drill-in without rebuilding the full route
- comparison rule: future warehouse-scoped workflow growth should preserve the current parent-and-child layout continuity rather than demoting warehouse item work back into disconnected full-page hops or full-shell rerenders

---

### PUBLIC_NOTES

# Public Surface Notes

The public surface is responsible for brand confidence, discoverability, and warehouse-aware decision-making.

Current focus:

- landing story
- catalog browsing
- product detail direction
- warehouse network explanation

Next implementation targets:

- hero and proof-strip components
- category rail and product card system
- product hero, gallery, and specs grouping
- quote and restock entry surfaces

Public comments thread structure decision:

- keep the public thread as a compact product-support surface with one visible buyer question, one staff reply, and a lightweight submission form
- route deeper moderation logic into the internal comments workspace instead of turning the public product page into a dense support forum
- keep new public questions pending by default so the public route stays premium and moderation-aware

## Catalog Browsing Contract

- category and collection browsing should stay query-backed so search, category, warehouse, sort, and page state survive reload, sharing, and direct entry
- catalog modules should include a results header, visible filter summary, sort controls, a product grid, and pagination that keeps locality context visible
- promotional callouts should favor bundle readiness, warehouse proximity, and featured systems instead of discount-heavy retail language

## Product Route Contract

- the product page hierarchy should lead with hero identity, immediate warehouse promise messaging, and the next best commercial action
- the main product sections are hero, warehouse promise lanes, related products, quote or restock capture, and a compact moderation-aware question thread
- inventory-aware messaging should turn stock posture into believable promise language instead of exposing raw fulfillment internals
- route data should cover product identity, warehouse promise lanes, related products, approved or staged comment context, and default form values for quote, restock, and question flows
- related products should come from the same merchandising family first so the recommendation rail reads like a system expansion rather than a generic upsell widget
- low-stock and unavailable states should pivot the buyer toward nearby warehouse availability, substitute products, quote capture, or restock capture rather than presenting a dead-end message

## Comments And Form Rules

- Atlas uses product questions rather than a full public review platform in the first release
- public comments live inline on the product route as a secondary support module beneath the main merchandising story
- moderation states stay limited to pending, approved, rejected, and flagged, with new public questions pending by default
- restock requests should collect email plus warehouse preference when stock is constrained so the internal side can treat them as regional demand signals
- quote requests should collect buyer identity, contact email, quantity, and a short rollout note so the path reads like a business inquiry instead of a contact stub
- success and error states should remain inline, field keyed, and accessible, with polished confirmation copy rather than route-breaking redirects

## Public Endpoint Payload Direction

- comment submissions should post product slug, buyer name, reply email, body, and optional locale hint and should return field errors plus moderation status
- restock requests should post product slug, email, warehouse slug, and optional locale hint and should return normalized warehouse identity plus confirmation state
- quote requests should post product slug, buyer name, email, requested quantity, note, and optional warehouse context and should return field errors plus a staged follow-up summary

---

### RELEASE_READINESS

# Atlas Commerce OS Release Readiness

Use this file as the release-prep and regression baseline for the current Atlas milestone.

## Release-Readiness Checklist

- Atlas wasm build completes from the repo root using the `Build` command above and writes `examples/static/bin/atlas-commerce-os.wasm`.
- Atlas SSR coverage passes using `go test -tags playwrightgo ../test/playwrightgo/examples -run TestAtlasSSR -v` from `examples/`.
- Atlas cross-browser smoke coverage passes using `go test -tags playwrightgo ../test/playwrightgo/examples -run TestAtlasCrossBrowserSmoke -v` from `examples/`.
- Public shell, internal shell, and route-recovery surfaces all render without blank states.
- Settings resume reflects theme, locale, density, and warehouse on direct route entry.
- Inventory resume reflects saved view, warehouse, and query on direct route entry.
- Light and dark themes both receive manual visual review before a demo pack is captured.

## Loader And Overlay Readiness

- Revalidate each loader-backed route at least once during review and confirm revision values change in place.
- Confirm overlay-backed workflows return focus and route context after save, approve, or dismiss actions.
- Confirm optional debug logging does not interfere with loader-backed or overlay-backed flows.

## Regression Checklist

- Rebuild Atlas wasm after any route, token, or shell-level change.
- Re-run the Atlas smoke suite after any persisted-state or route-entry change.
- Recheck the catch-all recovery route after adding new shell links.
- Recheck RTL settings entry after any locale or styling update.
- Recheck inventory resume after any saved-view, warehouse, or query behavior update.
- Recheck direct-entry plus persistence combinations for settings, inventory, SKU detail, and warehouse detail after route-state changes.

## Motion And Density Review

- Verify hover and keyboard-visible states feel equivalent on pills, link tiles, and nested cards.
- Recheck reduced-motion behavior after any route-shell or overlay motion change.
- Use the mobile density screenshots as a baseline when compact or comfortable spacing changes on internal routes.
- Recheck the mobile internal rail after any nested-layout or route-label change.

## Browser-Matrix Plan

- Chromium, Firefox, and WebKit now run through `TestAtlasCrossBrowserSmoke` for key public and internal Atlas route entry coverage.
- Chromium narrow viewport should still be used for settings and inventory responsive checks.
- If one browser fails the smoke lane, treat that result as release-blocking for the affected route family.
- Reduced-motion review should be repeated when overlays, portals, or route transitions become richer.

---

### TESTING_OPERATIONS

# Testing Operations

Use this section as the Windows command matrix and local task grouping for Atlas review. Commands assume PowerShell and the repo root unless a row says otherwise.

## Command Matrix

| Layer | Command | When to run |
| --- | --- | --- |
| Shared unit and render helpers | `go test ./examples/server/atlas-commerce-os/shared/...` | Route copy, shell markup helpers, derived state, cache helpers, tokens, seed data, and render primitives. |
| Server unit and integration | `go test ./examples/server/atlas-commerce-os/server/...` | Route handlers, API handlers, bootstrap payloads, CSRF, auth redirects, persistence rules, and integration flows. |
| Full Atlas Go package sweep | `go test ./examples/server/atlas-commerce-os/...` | Before handing off any Atlas server, shared, seed, token, or docs-linked behavior change. |
| Wasm build | `$env:GOOS = 'js'; $env:GOARCH = 'wasm'; go build -o .\examples\static\bin\atlas-commerce-os.wasm ./examples/server/atlas-commerce-os/client; Remove-Item Env:\GOOS, Env:\GOARCH` | Before any browser assertion, screenshot capture, or manual review after changing client, shared, token, route, or shell code. Clear `GOOS`/`GOARCH` before the next native `go run` or `go test`. |
| Atlas browser-flow manifest guard | `go test ./examples/tests/atlas-commerce-os` | Validates the buyer-flow, operator-flow, design-parity, recovery, E2E, helper, and screenshot planning skeletons before executable Playwright specs are promoted. |
| SSR Playwright suite | `Push-Location .\examples; go test -tags playwrightgo ../test/playwrightgo/examples -run TestAtlasSSR -v; Pop-Location` | Direct-entry public and internal route coverage on the native-server Atlas example. |
| Cross-browser smoke | `Push-Location .\examples; go test -tags playwrightgo ../test/playwrightgo/examples -run TestAtlasCrossBrowserSmoke -v; Pop-Location` | Browser-matrix release checks after shell, route, preference, or bootstrap changes. |
| Atlas startup smoke | `Push-Location .\examples; go test -tags playwrightgo ../test/playwrightgo/examples -run TestAtlasStartup -v; Pop-Location` | Fast browser startup confirmation when touching server boot or asset loading. |
| Screenshot refresh review | `go run ./examples/server/atlas-commerce-os/server` plus manual Chromium captures into `examples/server/atlas-commerce-os/docs/screenshots/` | Visual baseline updates for public desktop, internal mobile density, light and dark theme changes. |
| Manual review run | `go run ./examples/server/atlas-commerce-os/server` then start at `http://127.0.0.1:8096/shop` | Human review of route hierarchy, interaction feel, preference persistence, reduced motion, density, and localized surfaces. |

## Stale Wasm Guard

Browser runs must follow this order whenever client, shared Atlas, token, route rendering, or shell code has changed:

1. Stop any long-running Atlas server that may be serving an older binary.
2. Rebuild the example wasm with the `Wasm build` command from the command matrix above.
3. Confirm `examples\static\bin\atlas-commerce-os.wasm` has a fresh modified time after the source edit.
4. Start the server with `go run ./examples/server/atlas-commerce-os/server`.
5. Run the targeted Playwright command or manual browser review.
6. If browser behavior contradicts a recent source change, repeat the rebuild and restart before debugging route logic.

Do not capture screenshots, approve parity, or mark Playwright results as authoritative when the wasm timestamp predates the relevant source edit.

## Local Task Groups

- unit: `go test ./examples/server/atlas-commerce-os/shared/...`
- component or render: `go test ./examples/server/atlas-commerce-os/shared/atlas -run "Render|Panel|Page|Overlay|Table|Comment|Availability"`
- integration: `go test ./examples/server/atlas-commerce-os/server -run "Integration|Flow|Store|Mutation|Preference|Moderation|Receiving|Transfer"`
- browser-flow planning: `go test ./examples/tests/atlas-commerce-os`
- SSR: `Push-Location .\examples; go test -tags playwrightgo ../test/playwrightgo/examples -run TestAtlasSSR -v; Pop-Location`
- Playwright browser matrix: `Push-Location .\examples; go test -tags playwrightgo ../test/playwrightgo/examples -run "TestAtlasSSR|TestAtlasCrossBrowserSmoke|TestAtlasStartup" -v; Pop-Location`
- screenshots: rebuild wasm, restart the server, capture the named routes in `SCREENSHOT_NOTES`, and replace only intentional baseline images
- manual review prep: reset or reseed local state if needed, rebuild wasm, start the server, disable debug logging unless the review explicitly covers diagnostics, and keep `RELEASE_READINESS` open beside the browser

## Browser Flow Bucket Structure

Atlas browser-flow planning lives in `examples/tests/atlas-commerce-os/` so the future executable suites have one stable map before code lands in the Playwright runner.

| Bucket | Role | Current skeleton |
| --- | --- | --- |
| buyer-flow | Public browsing, quote, restock, comment, mobile-nav, recovery, and progressive-enhancement journeys. | `buyer-flow/*.spec.md` plus manifest stories for public navigation, mobile navigation, progressive enhancement, and public design parity. |
| operator-flow | Internal sign-in, dashboard, products, inventory, warehouses, logistics, moderation, comments, and settings journeys. | `operator-flow/*.spec.md` plus manifest stories for full internal navigation, alert-driven work, full admin session, and internal design parity. |
| design-parity | Reference checks separated from pure functionality so React-to-GWC drift is reviewed after flows work. | `design-parity/*.spec.md` keyed to `design/homepage_store.tsx` and `design/homepage_warehouse.tsx`. |
| recovery-edge-cases | Failure and async edge cases shared by public and internal flows. | `recovery-edge-cases/README.md` plus manifest entries for 404, server error, failed write, network interruption, duplicate submit, invalid input state, and async navigation races. |
| e2e | Acceptance-level tracks mirroring the buyer and operator Playwright buckets. | `e2e/README.md` plus manifest tracks for buyer-flow and operator-flow E2E stories. |
| screenshots | Capture conventions and named checkpoints used by all browser buckets. | `screenshots/README.md` plus manifest-validated surface, theme, viewport, locale, and checkpoint sets. |
| helpers | Shared helper contracts for future executable specs. | `helpers/README.md` plus manifest entries for buyer navigation, operator navigation, parity landmarks, screenshot capture, and route-shell stability. |

The manifest guard checks that required buckets, helper contracts, reference design files, screenshot dimensions, run order, failure stories, manual stories, E2E tracks, and per-bucket spec skeletons remain present. It intentionally does not claim executable browser coverage; Playwright implementation work should promote these skeletons without renaming the buckets.

## Change-Triggered Review Checklist

- shared shell primitives: rerun `/shop`, `/shop/frame-desk`, `/warehouses`, `/warehouses/new-jersey-hub`, `/app/dashboard`, `/app/inventory`, `/app/warehouses/new-jersey-hub`, `/app/products`, `/app/receiving`, and `/app/settings`; verify header hierarchy, route summaries, nested outlets, overlays, density, and mobile rail behavior.
- route loaders: rerun direct entry plus in-app navigation for every changed route family; verify loading, error, retry, revalidation, mutation invalidation, and back or forward navigation.
- bootstrap payloads: rerun direct entry for `/shop`, `/shop/frame-desk`, `/app/dashboard`, `/app/inventory?warehouse=new-jersey-hub`, `/app/products/frame-desk`, `/app/warehouses/new-jersey-hub/items/frame-desk`, and `/app/settings`; verify metadata, preferences, saved views, route data, CSRF state, and hydration resume.
- preference logic: rerun settings save, direct entry into inventory and SKU detail, light and dark modes, compact and comfortable density, default warehouse, RTL locale, and cross-tab preference sync.
- public buyer flows: rerun catalog filter or sort, product quote validation, restock capture, public question submission, warehouse detail, and warehouse-specific availability.
- internal operator flows: rerun dashboard triage, inventory threshold edit, product edit with unsaved guard, transfer approval or cancelation, receiving discrepancy classification, comment moderation, and settings import or export.

---

### REVIEW_DECISIONS

# Atlas Commerce OS Review Decisions

This file records the high-level product answers that were originally left as open review questions.

## Product And Scope Decisions

- The public versus internal split is correct: the storefront sells confidence and locality, while the internal console handles dense operational workflows.
- The signature flows are the right ones: product discovery, warehouse-aware availability, quote or restock follow-up, inventory triage, transfer planning, receiving, and moderation.
- SQLite remains a realistic example scope because Atlas needs believable persistence contracts without introducing external infrastructure.
- Atlas should keep questions, quote requests, and restock requests, but it does not need a full public review system in v1.
- Purchase orders remain in scope, but transfers and receiving should continue to lead the core operational demo story.

## Routing And Auth Decisions

- Locale-prefixed routes should apply to the public surface first; the internal console can stay unprefixed in the first milestone.
- The first release should use mocked role state for internal routes rather than a real auth system.

## Visual And Demo Direction

- The current visual ambition is high enough for a flagship example as long as reviewer screenshots, motion checks, and responsive polish stay aligned with the public or internal split.
- Atlas should optimize for flagship-demo quality over maximum feature breadth.

---

### REWRITE_INVARIANTS

# Rewrite Invariants

This section locks the planning rules for the next rewrite pass. A rewritten route may change markup shape and visual polish, but it must preserve these contracts unless the TODO and this section are updated in the same change.

## Route, Copy, Metadata, And Response Invariants

- Public route shapes stay stable: `/`, `/shop`, `/shop/:slug`, `/warehouses`, `/warehouses/:warehouseId`, and `/warehouses/:warehouseId/items/:sku` continue to support direct entry, canonical metadata, recovery states, and route-appropriate buyer copy.
- Internal route shapes stay stable: `/app`, `/app/dashboard`, `/app/products`, `/app/products/:slug`, `/app/inventory`, `/app/inventory/:sku`, `/app/warehouses`, `/app/warehouses/:warehouseId`, `/app/warehouses/:warehouseId/items/:sku`, `/app/transfers`, `/app/transfers/:id`, `/app/purchase-orders`, `/app/purchase-orders/:id`, `/app/receiving`, `/app/receiving/:id`, `/app/comments`, and `/app/settings`.
- Public copy must keep the storefront voice focused on warehouse-aware confidence, product fit, promise language, quote or restock follow-up, and moderated buyer questions.
- Internal copy must keep the operator voice focused on triage, exception handling, inventory posture, replenishment, receiving, transfer balancing, moderation, and settings continuity.
- Metadata must remain route specific: title, description, canonical path, locale, surface type, recovery intent, and direct-entry route identity must not collapse into generic shell defaults.
- Server responses must preserve explicit status codes, structured field errors, workflow summaries, CSRF handling for writes, mock-auth redirects for internal HTML entry, and JSON recovery payloads for API callers.

## SSR, Router, Bootstrap, Cache, And Derived-State Rules

- SSR is the first authoritative render for direct entry; hydration resumes that route instead of replacing it with a blank pending shell.
- Router state owns route identity, params, query strings, nested outlets, direct-entry recovery, and back or forward navigation. UI state must not shadow route state in a way that breaks reload or sharing.
- Bootstrap payloads carry only the data needed to hydrate the current route, preference snapshot, i18n snapshot, saved views, mock session, CSRF state, and diagnostics metadata.
- Bootstrap size remains a review gate. Duplicated request payload blobs, large repeated collections, or route data that can be fetched after first paint should be split before release.
- Cache defaults remain fresh-first for SSR entry and internal operator workflows. Stale-while-revalidate remains limited to secondary, read-mostly public or diagnostic surfaces named in `REVIEW_DECISIONS`.
- Mutations must declare invalidation targets before merge and must reconcile the visible route through fresh loader or server state after optimistic confirmation.
- Derived state must stay centralized in shared helpers or atoms for inventory summaries, warehouse pressure, dashboard posture, saved-view context, promise copy, and route badges.
- Browser storage may assist preference and saved-view resume, but the next SSR bootstrap response remains the source of truth.

## Framework-Coverage Sync Rule

- `FRAMEWORK_COVERAGE` must be reviewed whenever a GoWebComponents primitive lands or changes behavior in a way Atlas could use.
- A rewrite task that needs state ownership, route loading, form lifecycle, overlays, scheduling, async resources, cache reuse, focus handling, or hydration should evaluate the matching shipped GWC primitive before adding local machinery.
- If Atlas intentionally defers a relevant primitive, the route task should name the reason and the follow-up condition in the TODO.
- Before each phase starts, refresh the coverage inventory against actual imports and runtime usage so planning does not rely on aspirational framework surfaces.

## Mandatory First-Pass GWC Primitives

The first rewrite pass should treat these as required evaluation points:

- router history, route loaders, nested layouts, revalidation, and route recovery for any route-family rewrite
- SSR bootstrap and hydration resume for every public SEO route and every internal direct-entry route
- shared atoms and derived helpers for shell badges, preference state, inventory posture, warehouse pressure, and route-local workspace summaries
- form state, validation projection, and workflow result helpers for quote, restock, product edit, threshold, receiving, transfer, moderation, preferences, and saved-view flows
- overlays, portals, focus trap, focus restore, and toasts for route-preserving confirmations and side sheets
- async resource and cached resource helpers only where the stale-data policy says the surface is safe to refresh after first paint
- scheduler or transition helpers for expensive client-only filtering, deferred secondary panels, and non-blocking preference updates

## Performance Budgets And Diagnostics Checkpoints

Each rewritten route family must record these checks in the PR or handoff notes:

| Budget area | First-pass threshold |
| --- | --- |
| Direct-entry SSR | meaningful route body, primary heading, shell navigation, and recovery affordance are present in HTML before hydration |
| Hydration resume | no visible full-route blanking, duplicate shell rendering, or route metadata flicker during resume |
| Route rerender scope | local form, overlay, preference, or filter interactions do not rebuild unrelated route bands or mounted parent layouts |
| Bootstrap payload | payload growth is explained when a route adds large collections, repeated objects, or per-item diagnostics |
| Interaction latency | dense internal filters, saved-view application, overlay open or close, and route revalidation remain responsive enough for repeated operator use |
| Diagnostics | debug logging stays opt-in, removable, and separated from user-visible state |

## Performance Budget Test Stories

`server/performance_budget_test.go` is the first automated budget gate for Atlas route performance. It keeps the thresholds intentionally coarse so the test catches obvious regressions without pretending to be a lab benchmark.

### Automated bootstrap-size gate

The bootstrap budget smoke covers the highest-risk first-paint routes:

| Surface | Routes |
| --- | --- |
| Public | `/`, `/shop`, `/shop/frame-desk`, `/warehouses/new-jersey-hub`, `/warehouses/new-jersey-hub/availability/frame-desk` |
| Internal | `/app/dashboard`, `/app/inventory`, `/app/products/frame-desk`, `/app/warehouses/new-jersey-hub`, `/app/purchase-orders/po-1042`, `/app/receiving/rcv-illinois-001` |

The warning threshold is `140 KiB` for inline bootstrap payloads. The failing threshold is `220 KiB`. A route that exceeds the warning threshold may still be acceptable, but the handoff must explain why the payload grew and which panel or secondary collection owns the growth.

### Automated loader-latency smoke

The loader-latency smoke exercises the heaviest internal JSON loaders directly through the server test harness:

| Loader family | Endpoint |
| --- | --- |
| Dashboard | `/api/app/dashboard` |
| Inventory | `/api/app/inventory` |
| Product detail | `/api/app/products/frame-desk` |
| Warehouse detail | `/api/app/warehouses/new-jersey-hub` |
| Purchase order detail | `/api/app/purchase-orders/po-1042` |
| Receiving detail | `/api/app/receiving/rcv-illinois-001` |

The first-pass fail budget is `2s` per local in-memory request. This is deliberately generous; it is meant to catch accidental blocking work, unbounded query expansion, or empty payload regressions before browser checks begin.

### Browser responsiveness checks

Browser review owns the interaction checks that cannot be proven from the server harness alone:

| Story | Route | Interaction | Pass condition |
| --- | --- | --- | --- |
| Catalog search | `/shop` | Type three characters, clear, then apply one category or warehouse filter | The catalog shell stays mounted, the input remains editable, and no full-page blank state appears |
| Internal filtering | `/app/inventory` | Apply a saved view, type in the search field, then change density or sort | The table region updates locally while the shell, nav, and action rail remain stable |
| Saved-view application | `/app/products` and `/app/inventory` | Apply, rename, export, import, then reset a saved view | The route context survives each operation and validation failures preserve user-entered data |
| Dense-route sort | `/app/warehouses/new-jersey-hub` | Change item status filters and sort order, then open a SKU detail route | Only the list/detail region changes; the warehouse header and summary band remain readable |

Record these checks beside screenshot or Playwright review notes whenever shared route state, cached resources, filters, saved views, or dense table helpers change.

### Overlay and first-action latency checks

The heaviest internal routes need one representative first-action check before a release signoff:

| Route | First action | Overlay action |
| --- | --- | --- |
| `/app/inventory/frame-desk` | Open threshold history | Open and cancel replenishment or threshold editing |
| `/app/warehouses/new-jersey-hub/items/frame-desk` | Edit a lane field | Open the warehouse-scoped item action panel |
| `/app/purchase-orders/po-1042` | Approve or hold preview | Open confirmation and cancel |
| `/app/receiving/rcv-illinois-001` | Classify a discrepancy | Open reconcile confirmation and cancel |

The first visible response should be immediate enough for repeated operator use. If a reviewer can perceive delayed feedback, the route needs a local pending state, transition boundary, or smaller route-local update before the work is marked demo-ready.

### Rerender and diagnostics checks

Atlas review mode should make rerender scope visible without shipping a permanent diagnostics overlay:

- debug logs should distinguish route-shell, header, hero, table, overlay, cached-resource, and mutation-invalidation events when the opt-in debug toggle is enabled
- filter, sort, mutation, toast, and overlay-only changes should not remount the whole route shell
- catalog filters, saved views, dense table sorts, and overlay-only actions should update localized panels rather than the public or internal page root
- cache hits, invalidations, loader timings, and expensive derived-state recomputation should either appear in opt-in logs or be captured in reviewer notes for the changed route family
- before-and-after review notes for `/shop`, `/app/products`, `/app/inventory`, and `/app/warehouses/:warehouseId` should state whether the route shell, header, hero, and dense table rerendered separately or together

### Low-power review story

Run this story manually when blur layers, gradients, dense tables, mobile spacing, or motion changes:

1. Enable reduced motion in the browser or OS.
2. Use a narrow viewport and the densest available internal table.
3. Visit `/shop`, `/app/inventory`, `/app/warehouses/new-jersey-hub`, and `/app/receiving/rcv-illinois-001`.
4. Confirm hover effects have keyboard equivalents, blur or gradient layers do not obscure text, overlays remain readable, and table rows can still be scanned without relying on motion.
5. Capture a note if any visual effect should simplify on low-power hardware or reduced-motion devices.

## Shared HTML Pattern Standard

- Public shell: page-level main landmark, route-specific hero, warehouse-aware proof or promise band, merchandising or availability body, route-local action surface, and recovery copy that returns the buyer to a useful public route.
- Internal shell: persistent workspace header, route summary strip, dense content region, route-local action rail or side panel, mounted nested outlet where applicable, toast viewport, and keyboard-reachable controls.
- Lists and tables: visible filter or scope summary, sort controls, stable empty state, pagination or route drill-in affordance, and row actions that do not rely on non-semantic wrappers.
- Forms: visible labels, helper text, field-level errors, summary-level errors, preserved user input on failed writes, explicit success state, and route revalidation or invalidation after successful writes.
- Recovery surfaces: status-appropriate heading, route-specific explanation, retry or return action, and preserved shell context for internal routes.

## First Shell Requirements

- Accessibility: each rewritten shell needs one clear page heading, landmark structure, keyboardable primary actions, focus-visible controls, focus restore after overlays, readable table or list semantics, and status messages that do not rely on color alone.
- Localization: route copy must remain keyed by surface and intent, RTL must keep heading rhythm and control order legible, and locale or direction changes must not invalidate unsaved route-local work.
- Recovery state: missing records, invalid query params, failed loaders, failed writes, and auth-sensitive internal entry must produce route-appropriate recovery UI instead of blank panels.

---

### ROUTE_ARCHITECTURE

# Atlas Commerce OS Route Architecture

This file records the concrete route, shell, metadata, and persistence decisions that keep Atlas coherent as both a storefront and an operations console.

## Public Route Behavior

### Catalog route behavior

- the catalog route is `#/shop`
- catalog state stays query-backed so search, category, warehouse, sort, and page survive reload and direct entry
- canonicalization should collapse most filtered states back to the base catalog route until Atlas adopts an indexed filter strategy
- current browsing behavior prioritizes shareable operational query state over SEO indexing of every filter combination

### Catalog query parameters

- `q`: free-text search term
- `category`: public product grouping such as `desks`, `storage`, `seating`, or `accessories`
- `warehouse`: warehouse slug for locality-aware discovery
- `availability`: reserved for a future public stock-health filter
- `sort`: current supported values are `featured`, `name`, and `warehouse`
- `page`: 1-based pagination index
- optional future filters such as price or merchandising tags should remain additive and not invalidate the current contract

### Product sales page structure

The product route should be assembled in this order:

- hero with product identity, finish switching, and primary warehouse promise CTA
- warehouse promise lanes that compare regional delivery posture
- related-product rail that keeps the buyer inside the same merchandising family
- quote request, restock request, and product-question forms
- comments or moderation-aware trust content that stays secondary to the product story

### Product page tab model

- Atlas currently favors section-based product detail instead of hidden tab state
- detail, promise, related products, and request surfaces should remain scroll-visible and deep-linkable by route rather than by fragile client-only tab state
- future route fragments can be added for media galleries or long technical specifications if the page grows denser

### Warehouse public pages

- `#/warehouses` is the public network overview route
- `#/warehouses/:warehouseSlug` exposes service region, promise speed, and stocked product highlights
- warehouse pages stay public and SEO-visible because locality meaningfully changes buyer trust and fulfillment expectations

### Warehouse-specific availability behavior

- `#/warehouses/:warehouseSlug/availability/:productSlug` should focus on one product promise inside one warehouse context
- the page should combine promise messaging, stock posture, and nearby alternative routes rather than expose raw operational counts alone
- the best next actions are to return to the warehouse story, jump back to the product route, or pivot to another warehouse lane

## Internal Route Architecture

### Authenticated app shell

- all internal routes live under `#/app`
- the shell uses a persistent left rail on desktop, a compact quick-switch rail on mobile, and a shared top context area for alerts, command-entry direction, and preferences
- `#/app` itself redirects into `#/app/dashboard` while keeping the internal shell mounted

### Primary internal navigation groups

- overview: dashboard as the triage entry plus products as the catalog or merchandising workspace
- stock: inventory list and SKU detail plus warehouse list and warehouse detail
- logistics: transfers, purchase orders and purchase-order detail, and receiving list and receiving session detail
- support: comments moderation plus settings, preferences, and diagnostics entry context
- the shipped internal shell now exposes grouped workspace navigation in the header on desktop and a sheet-based mobile workspace drawer, so operators can jump routes without losing shell context even before the full left-rail rewrite lands

### Internal shell rollout order

- desktop header first: grouped workspace cards and context pills establish the shared route hierarchy on larger screens
- mobile nav and collapse behavior next: the same groups collapse into the sheet-based workspace drawer and compact quick links
- route-context badges and workspace affordances last: saved-view, summary, filter, and route-badge cues hang off the shared shell once the grouping is stable

### Internal mobile nav parity

Atlas now ships the internal mobile menu through the shared workspace header. The small-screen shell exposes a hydrated `Workspace nav` trigger, grouped route cards inside a sheet-based drawer, and compact quick links that keep the current route family visible when the full grouped header collapses.

### Internal mobile nav work map

The internal mobile drawer should keep future review and testing scoped to five slices:

- rail collapse: how the grouped desktop header contracts into compact quick links plus the drawer trigger
- route grouping: how overview, stock, logistics, and support links remain aligned between desktop cards and the drawer
- workspace context: how saved-view, summary, warehouse, and filter cues stay visible when the full desktop header is gone
- active-state behavior: how the current route family and exact route stay emphasized in both quick links and grouped drawer cards
- keyboard interaction: how the trigger, drawer close affordance, grouped links, and focus return behave under keyboard-only navigation

### Route-level tabs for detail pages

- detail routes use visible sections and focused subpanels rather than hidden tab chrome today
- SKU detail already behaves like overview plus activity workflow sections
- warehouse, transfer, purchase-order, and receiving detail routes should keep overview and activity visible together unless future data density forces explicit tabs
- any later tab state should live in the URL when it materially changes the route meaning

### Nested route structure

- the internal app shell is the first layout boundary
- inventory detail is nested beneath the inventory workspace shell so saved-view and filter context remain mounted during drill-in
- public warehouse detail and availability routes inherit the public shell and keep the Warehouses section active across nested paths
- purchase-order detail and receiving detail stay standalone first-class routes for now; Atlas already demonstrates a second deep-linkable nested workflow through warehouse item drill-in, while PO approval state, receiving discrepancy work, and related confirmations remain better expressed as route-local overlays than persistent list-and-detail shells

### Overlay-versus-route responsibility

- full operational contexts such as SKU detail, warehouse detail, transfer detail, purchase-order detail, and receiving session detail stay as first-class routes
- shorter secondary workflows such as threshold editing, transfer confirmation, moderation confirmation, and receiving discrepancy resolution stay overlay-backed so the current route context is preserved
- public forms remain inline until Atlas needs explicit confirmation or staged submission states

### Deep-link behavior

- every major list and detail route must open directly from a fresh tab
- query-backed catalog and inventory states should be restorable from the URL or persisted browser state without visiting the landing route first
- internal routes should stay deep-linkable without relying on query-flag diagnostics toggles

## Metadata And Ownership Rules

### Metadata rules per route

- every route owns a title, description, canonical path, and social preview fallback
- public landing, catalog, product, warehouse detail, and warehouse availability routes are the SEO-sensitive surfaces
- internal routes still receive titles and descriptions for testing and future SSR bootstrap, but they are not intended as crawl targets

### Navigation state persistence

- public catalog search, category, warehouse, sort, and page state live in the URL
- direct-detail routes derive their identity from params and keep related merchandising context inside the route body instead of local-only state
- theme, locale, density, default warehouse, inventory saved view, inventory warehouse, inventory sort, inventory query, and inventory density resume from browser storage
- future tab or selection state should only persist if it changes the meaning of a direct link or reviewer handoff

## State And Debug Logging Rules

### Shared app state boundaries

- shared atoms own user-level preferences and cross-route workspace state that must survive route changes or reloads
- route loaders own screen-shaped operational data such as dashboard alerts, inventory rows, SKU detail summaries, transfer recommendations, and receiving sessions
- local component state owns short-lived form validation, ephemeral workflow confirmation copy, and UI-only toggles that should not survive a route change
- browser storage is only used to resume preference and workspace context; it should not become the source of truth for route data

### Derived and computed metrics

- the inventory summary derived state calculates total visible stock, pressure warehouses, saved-view urgency, and next-action guidance from shared workspace atoms
- product detail promise lanes convert warehouse availability into customer-facing promise copy rather than exposing raw warehouse counts alone
- dashboard and warehouse surfaces should continue to prefer derived posture labels over isolated numbers so operator handoff remains legible

### Snapshot behavior

- Atlas no longer ships runtime snapshot overlays as part of the release surface
- debug logging stays developer-only and opt-in via `data-atlas-debug-logs` or `window.__atlasDebugLogs`
- Atlas should keep full reviewer or operator workspace snapshots deferred for now: saved-view export or import already covers the narrow operator handoff Atlas can justify today
- a broader workspace snapshot flow should only land once Atlas has richer reviewer-only filters or multi-route operator context that cannot be handed off cleanly through saved views and direct links
- Atlas now includes one narrow operator snapshot export on `/app/settings`: the settings route can export a copyable JSON payload containing the current shell presentation state, current route workspace atom, and saved-view metadata without exposing deeper developer tooling

### Route revalidation strategy

- loader-backed internal routes should expose explicit revalidation controls while Atlas remains seed-backed and local-state-driven
- route loaders rerun after workflow mutations or when the operator explicitly refreshes a loader-backed route so the visible screen can reconcile with the latest mock workflow state
- query-backed routes should keep visible query state stable while revalidation runs so a refresh does not silently discard the operator context

### Stale-While-Revalidate policy

- direct SSR entry should always block on fresh route data; stale-while-revalidate is only acceptable for hydrated in-session cache reuse
- internal workflow routes should block on fresh loader data whenever the route is first entered or a mutation has just completed:
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
- inventory thresholds, transfer recommendations, purchase-order status, receiving closeout, moderation state, preferences, and saved views should all prefer fresh data because they directly change operator decisions and are already wired to explicit invalidation or loader refresh
- public route shells should also block on fresh data for direct entry because SEO-sensitive HTML, canonical metadata, and warehouse-aware availability copy should not be derived from a stale client cache
- stale-while-revalidate is acceptable for read-mostly secondary data after hydration when the stale view is clearly non-authoritative and invalidation is already in place:
  - approved public comments shown beneath the primary product story
  - related-product or warehouse-promise side rails reopened during the same session
  - warehouse detail side data that supplements, rather than defines, the main route decision
  - temporary reviewer-only async panels used during development review, before release cleanup removed query-flag diagnostics surfaces
- stale-while-revalidate should not be used for mutation confirmation surfaces; Atlas should show the optimistic success message immediately, but the authoritative route state must come from the next fresh loader result
- when Atlas adds more cached resources, the default should remain `fresh-first` unless the data is secondary, read-mostly, and safe to momentarily lag behind without changing fulfillment, moderation, or operator workflow choices

### Cross-tab state policy

- preferences should sync across tabs and windows because theme, locale, density, and default warehouse are operator-level defaults that should not diverge between concurrent Atlas sessions
- saved views should sync across tabs and windows because create or import flows produce shared operator presets rather than tab-local scratch state
- debug logging should stay opt-in per tab or window and should not auto-enable in another session
- active inventory queries, unsaved forms, open overlays, and route-local work-in-progress state should stay tab-local so one tab does not stomp another tab's focused workflow
- the server-backed preference and saved-view payload remains the source of truth for fresh document entry; browser storage may assist resume behavior or broadcast change notifications, but it should not outrank the next SSR bootstrap response

### Mutation invalidation matrix

- `comment-submitted`, `comment-moderated`, and `comments-bulk-moderated` invalidate `/app/comments`, `/app/dashboard`, and `/shop` because moderation state is shared between the buyer inbox, dashboard summary cards, and public product comment threads
- `product-created`, `product-updated`, and `product-deleted` invalidate `/app/products`, `/app/inventory`, `/app/warehouses`, `/shop`, and `/warehouses` because product identity, merchandising copy, and warehouse availability all read the same catalog records
- `inventory-updated` invalidates `/app/inventory`, `/app/warehouses`, `/app/dashboard`, `/shop`, and `/warehouses` because stock, promise copy, and facility posture all derive from the edited lane
- `threshold-updated` invalidates `/app/inventory`, `/app/warehouses`, and `/app/dashboard` because reorder posture and warehouse pressure summaries depend on threshold history and lane policy
- `purchase-order-created` invalidates `/app/purchase-orders`, `/app/receiving`, `/app/dashboard`, `/app/inventory`, `/app/warehouses`, `/shop`, and `/warehouses` because the create flow changes both PO records and inbound inventory totals used by public and internal availability views
- `purchase-order-updated` invalidates `/app/purchase-orders`, `/app/receiving`, and `/app/dashboard` because approval state changes affect replenishment and receiving workflows without changing lane counts
- `receiving-reconciled` invalidates `/app/receiving`, `/app/purchase-orders`, and `/app/dashboard` because receiving closeout changes the workflow state surfaced by those routes
- `transfer-created` invalidates `/app/transfers`, `/app/inventory`, and `/app/dashboard` so newly staged balancing work clears stale transfer lists and operator summaries
- `preferences-saved`, `saved-view-created`, and `saved-views-imported` invalidate every internal route family because those values are carried in the internal SSR bootstrap payload, not only in settings or inventory page data
- `quote-request-submitted` and `restock-request-submitted` invalidate `/shop` and `/warehouses` so public action rails and warehouse-specific availability messaging do not reuse stale submission-state caches

### Access guard behavior

- public routes remain open
- internal routes assume a mock authenticated session and should later enforce role-aware entry with redirect or recovery behavior rather than blank states
- unsaved form workflows should prefer explicit overlay or route-level confirmation before navigation once Atlas adds longer-lived editing flows

### Page ownership matrix

- public SEO-sensitive: landing, catalog, product detail, warehouse detail, warehouse-specific availability
- public utility: warehouses index and future quote or restock submission confirmations
- internal-only: dashboard, inventory, SKU detail, warehouse operations, transfer flows, purchase orders, receiving, comments, settings
- modal-triggering but route-preserving: threshold edit, transfer confirmation, receiving discrepancy sheet, moderation confirmation
- future SSR plus hydrate targets: landing, catalog, product, warehouse detail, dashboard, and inventory

---

### ROUTE_DATA_NOTES

# Route Data Notes

## Route Bootstrap Contract

Every route should be able to describe itself with:

- path
- surface
- screen name
- title
- initial data payload
- preference snapshot
- locale snapshot

## Loader Payload Direction

The first loader-backed routes should return compact payloads:

- dashboard: alerts, inbound work, pending reviews, quick links
- inventory: rows, active filters, sort state, saved views
- sku detail: selected SKU, warehouse focus, threshold action track
- transfers: recommended lane, priority, and recommended transfer quantity
- receiving: active session summary, open discrepancies, and closeout state
- comments: moderation queue, selected item, action availability
- settings: theme, locale, density, default warehouse

## Query Contracts

Inventory query contract:

- warehouse
- category
- supplier
- stock-health
- availability
- search
- sort
- direction

Moderation query contract:

- status
- search
- sort
- page

## Preference Save Contract

The first preference save payload should carry:

- theme
- locale
- density
- defaultWarehouse

The response can simply echo the normalized saved values during the first milestone.

## Cookie Strategy

When SSR lands, cookie-backed restore should begin with:

- `atlas_theme`
- `atlas_locale`
- `atlas_density`

These cookies should be treated as hint-level restore inputs, not as the only source of truth once user persistence exists.

---

### SCREENSHOT_CHECKLIST

# Atlas Commerce OS Screenshot Checklist

Current public baseline pack lives in `screenshots/`:

- `atlas-public-landing-light-desktop.png`
- `atlas-public-landing-dark-desktop.png`
- `atlas-public-catalog-light-desktop.png`
- `atlas-public-catalog-dark-desktop.png`
- `atlas-public-product-light-desktop.png`
- `atlas-public-product-dark-desktop.png`
- `atlas-public-warehouses-light-desktop.png`
- `atlas-public-warehouses-dark-desktop.png`

Current internal density pack lives in `screenshots/`:

- `atlas-internal-settings-light-mobile.png`
- `atlas-internal-inventory-dark-mobile.png`

Capture these reference views once the next visual polish pass lands.

## Public Surface

- landing route with hero, proof strip, and category guidance visible
- catalog route with filter chips and result framing visible
- product route with quote and restock panels visible
- warehouses route with regional availability summary visible

## Internal Surface

- dashboard route with alert and quick-action panels visible
- inventory route with table guidance and saved-view context visible
- SKU detail route with the activity section selected
- transfers route with a submitted transfer targeting New Jersey
- receiving route with a damaged-goods discrepancy selected
- moderation route with a flagged record selected
- settings route with light theme and French locale selected

## Capture Rules

- use desktop Chromium first for the baseline review pack
- capture at least one narrow viewport for the inventory or settings route
- avoid screenshots that include devtools, browser chrome, or loading flicker
- save a matching note when a screenshot reveals spacing, copy, or hierarchy issues

## Naming Rules

- use `atlas-<surface>-<route>-<theme>-<viewport>.png`
- keep surface values to `public`, `internal`, or `recovery`
- keep theme values to `light` or `dark`
- keep viewport values to `desktop` or `mobile`
- examples: `atlas-public-product-dark-desktop.png`, `atlas-internal-settings-light-mobile.png`

## Browser-Flow Screenshot Story Map

The browser-flow skeleton in `examples/tests/atlas-commerce-os/screenshots/` expands screenshot review from route-only captures to named flow checkpoints. Future automated captures should use the manifest convention `atlas-<surface>-<route>-<state>-<theme>-<viewport>.png`; existing route-only baseline names remain valid until they are refreshed.

- buyer-flow checkpoints: landing start, catalog browsing, product-detail decision, warehouse availability, and post-submit success.
- operator-flow checkpoints: dashboard start, product editor, inventory triage, warehouse detail, purchase-order or receiving workflow, and settings persistence.
- recovery and state checkpoints: empty state, no-results state, recovery page, validation error, success state, and overlay-open state.
- locale checkpoints: representative French public flow and Arabic internal flow captures for overflow, clipping, and RTL review.
- design-parity checkpoints: public start, midpoint, and end against `design/homepage_store.tsx`; operator start, midpoint, and end against `design/homepage_warehouse.tsx`.
- screenshot refresh order: run buyer-flow and operator-flow functionality first, then capture design-parity pairs after core behavior passes.

---

### SCREENSHOT_NOTES

# Atlas Commerce OS Screenshot Notes

Use this file to record what each reference capture should prove.

## Public shots

- landing: premium positioning, warehouse proof, and CTA hierarchy should read in one frame
- catalog: browsing controls should look intentional rather than debug-heavy
- product: product story and request surfaces should feel integrated

## Internal shots

- dashboard: triage-first hierarchy should be clearer than generic KPI dashboards
- inventory: dense content should remain readable and calm
- SKU detail: section controls should read as workflow navigation, not decorative tabs
- transfers and receiving: state and discrepancy changes should be legible in the capture
- settings: theme and locale controls should look like real preferences, not test toggles
- settings mobile: comfortable density should keep the rail and preference groups calm without losing route context
- inventory mobile: compact density should preserve warehouse, status, and query context without turning the rail into a detached menu

## Contrast audit notes

- public light mode should keep hero panels, chips, and filter cards separated from the page background without washing out the brand accent
- internal dark mode should keep nested cards, badges, and route rails legible without collapsing into one flat navy mass
- Arabic and RTL states should keep heading rhythm and badge contrast balanced in both themes

## Motion review notes

- keyboard-visible controls should receive the same lift and emphasis as hover states, not a weaker fallback treatment
- nested route cards should use shadow and focus motion to clarify hierarchy instead of relying on large travel distances
- reduced-motion mode should remove the lift behavior cleanly while preserving focus visibility and route clarity

## Review prompts

- does each screenshot communicate a believable product purpose immediately
- does any panel feel too much like framework scaffolding rather than product UI
- does the visual rhythm break between public and internal surfaces
- does the light theme feel editorial and deliberate rather than washed out
- does the dark theme keep enough hierarchy without collapsing into flat navy surfaces

---

### SEED_STRATEGY

# Seed Strategy Notes

Atlas Commerce OS should use seed data that feels intentionally curated rather than randomly generated.

## Principles

- seed for demo value first
- keep every major workflow represented by data
- prefer believable operational asymmetry over fake randomness
- make public and internal stories line up around the same products and warehouses

## Initial Seed Groups

- products
- warehouses
- inventory levels
- comments
- quote requests
- restock requests
- transfers
- receiving sessions
- saved views
- preference defaults

## Reset Direction

The first reset workflow can safely rebuild from static seed definitions into a local sqlite file once persistence is introduced.

Current scaffold support:

- `scripts/reset-seed.ps1` removes the local Atlas sqlite file path so demos can return to a clean seed baseline once persistence lands

---

### SEO_NOTES

# Atlas Commerce OS SEO Notes

Atlas only treats the public storefront routes as search-facing content.

## SEO Targets

- landing, catalog, product detail, warehouses index, warehouse detail, and warehouse-specific availability are the public routes worth indexing and sharing
- internal dashboard, inventory, logistics, moderation, and settings routes should keep metadata for diagnostics and future SSR, but they are not ranking targets

## Canonical URL Policy

- landing should canonicalize to `/`
- catalog should canonicalize to `/shop` for the first release, even when local query parameters drive search, sort, warehouse, or page state
- product and warehouse detail routes should canonicalize to their concrete entity path
- warehouse-specific availability routes should canonicalize to their stable warehouse plus SKU path

## Structured Data Coverage

- product detail should be the first route to receive `Product` and availability-oriented offer schema
- warehouse-specific availability can extend the product schema with region-aware availability messaging when Atlas grows a server-rendered SEO pass
- internal routes should not emit product or organization schema intended for search discovery

## Sitemap And Social Preview Coverage

- the sitemap should include only the public landing, catalog, product detail, warehouses index, warehouse detail, and warehouse-availability routes
- internal routes should stay out of the sitemap and should later emit `noindex` behavior once a server variant exists
- public routes should keep using the per-route Open Graph image mapping already defined in Atlas metadata helpers

## Content Indexing Boundaries

- public marketing and discovery routes may be indexed
- internal routes, diagnostics mode, and workflow-heavy utility confirmations should not be treated as crawl targets
- filtered catalog states, internal query views, and moderator or operator deep links should not fragment Atlas into duplicate search entries

---

### SERVER_ENDPOINT_INTEGRATION_PLAN

# Atlas Commerce OS Server Endpoint Integration Plan

This section now records the shipped server contract plus the remaining server-side follow-up work.

## Shipped Endpoint Groups

- public catalog and product discovery endpoints for search, filtering, related products, and warehouse-aware availability
- internal inventory endpoints for saved views, SKU detail, threshold history, and transfer recommendations
- operations endpoints for warehouse pressure, purchase orders, receiving sessions, and moderation review queues
- preference and bootstrap endpoints for persisted operator defaults and route bootstrap payloads

## Remaining Endpoint Follow-Up

- diagnostics-specific APIs should stay removable before final release signoff
- richer field-level HTML error replay remains future work beyond the current JSON validation payload shape
- real auth can replace the mock session flow later without changing the `/api/public/...` and `/api/app/...` split

## Route Mapping Direction

- `/shop` hydrates from the catalog query endpoint that accepts search, category, warehouse, sort, and page
- `/shop/:productSlug` shares product, related-product, and warehouse-promise data with the public warehouse availability route
- internal SSR entry routes keep the same route shapes used by the earlier loader-backed wasm-only experience
- SSR bootstrap payloads now provide route title, surface, preference defaults, CSRF token, and the first screen of route data together

## Endpoint Design Notes

- keep public warehouse availability and product detail contracts compatible so promise language stays consistent across routes
- preserve route-entry and persistence combinations by keeping preference and workspace state serializable in a bootstrap payload
- expose loader timing metadata separately from domain payloads so diagnostics can be removed cleanly before release signoff
- design overlay-backed mutations around explicit workflow results rather than optimistic hidden side effects

## Server Entrypoint Direction

- the Go server entrypoint boots sqlite, applies migrations, loads static assets, and mounts SSR plus API handlers from one process
- public page requests render SSR HTML shells with metadata and bootstrap payloads
- internal page requests render SSR HTML plus mock auth context, CSRF state, and preference bootstrap state
- JSON and form endpoints live behind a consistent `/api/public/...` and `/api/app/...` split

## Page-Rendering Handler Contract

Each SSR handler should return or derive:

- route metadata
- route data payload
- bootstrap payload for preferences, locale, saved views, and route identity
- auth context for internal routes
- normalized theme and locale state
- status code plus structured error or recovery intent when data is missing

## Route Handling Split

- public and internal page-entry routes return SSR HTML
- read-heavy filtered or refreshable views may use JSON endpoints behind client loaders after first render
- write flows may use form-post or JSON mutation endpoints as long as they return structured field and workflow results
- comments, quote requests, and restock requests should stay compatible with both progressive form posts and future client-side mutation helpers

## Error Handling Strategy

- public 404 routes should fall back to a branded recovery page, not a raw server error
- internal auth failures now redirect browser requests into the mock sign-in flow and return structured recovery JSON for API callers
- validation errors should return field-keyed payloads plus a stable message summary
- server failures should preserve route shell context and emit retry-friendly error copy instead of generic blank screens

## Public Interaction Endpoints

- comment, quote, and restock writes should validate product identity, contact fields, and message length
- public read endpoints may remain optional for routes that SSR fully, but catalog, comments, and warehouse availability should have JSON parity for async refreshes

## Internal Read And Write Endpoints

- inventory, warehouse, purchase-order, transfer, receiving, moderation, preferences, and saved-view routes should each map to predictable list and detail APIs
- internal writes should return explicit workflow state so loader revalidation can update the current route without guessing
- endpoint naming should stay grouped under `/api/public/...` and `/api/app/...` rather than mixing route and data semantics

## Validation Rules

- public request endpoints return field-level errors for contact info, quantity, and free-text notes
- inventory and receiving mutations return domain-specific errors for invalid quantities, missing classification, or impossible warehouse transitions
- moderation and approval flows return status plus optional follow-up actions rather than only success booleans

---

### SSR_BOOTSTRAP_NOTES

# SSR Bootstrap Notes

Atlas Commerce OS now runs request-time SSR from the example 86 Go server, and the shared bootstrap contract is used by both server render and wasm hydration.

## Bootstrap Sections

- route metadata
- route data
- preference snapshot
- i18n snapshot
- saved views
- mock user session for internal routes
- CSRF token for progressive forms and imperative mutations

## Current Bootstrap Coverage

- landing
- catalog
- product
- warehouse index
- warehouse detail
- warehouse availability
- dashboard
- inventory
- SKU detail
- warehouses internal list and detail
- transfers list and detail
- purchase orders list and detail
- receiving list and detail
- moderation
- settings

## Current SSR Page Set

Atlas now server-renders these routes by default:

- landing
- catalog
- product detail
- warehouse detail
- warehouse-specific availability
- dashboard
- inventory list
- SKU detail
- warehouses internal list and detail
- transfers list and detail
- purchase orders list and detail
- receiving list and detail
- moderation
- settings

## Current SSR Goal

The current SSR milestone proves that public discovery routes and the internal operations console render meaningful HTML before the client resumes, while sharing one bootstrap contract across SSR and hydration.

## Hydration Boundaries

- public marketing copy, proof panels, and static explanatory sections may render as plain HTML first, but catalog controls, quote or restock forms, and product-detail route actions should hydrate immediately
- internal shell framing, route-local action bars, inventory controls, overlays, and moderation workflows require full hydration because they depend on shared state and immediate interaction
- debug logging stays client-only and opt-in even on SSR pages because it is a developer affordance rather than part of the public or operator product surface

## Page-By-Page Hydration Goals

- landing: hydrate CTA clusters and any locale or theme-aware controls
- catalog: hydrate search, filters, sort, and pagination immediately
- product: hydrate finish selection, quote or restock forms, comment submission, and route-aware CTA clusters immediately
- warehouse detail and availability: hydrate route-aware availability links and any warehouse filter affordances immediately
- dashboard and inventory: hydrate on first paint because loader refresh, saved views, and route-local workspace state are the point of the screen
- SKU detail: hydrate threshold history actions and overlay entry points immediately

## Bootstrap Contract

The first page bootstrap payload should be shaped as:

- `route`: path, params, query, surface, title, and canonical metadata
- `data`: route-specific loader payload
- `i18n`: locale, direction, and translated resource handles when they exist
- `theme`: active theme and density hint
- `user`: mocked internal session summary for internal pages, omitted on public routes
- `preferences`: default warehouse and any persisted preference snapshot needed before hydration
- `workspace`: saved-view or query resume state for routes that restore operator context
- `csrf`: optional token slot for future server-backed form posts

## SSR-To-Client Consistency Rules

- server-rendered theme, locale, and direction must match the values applied to the document element before hydration begins
- bootstrap route data must describe the same path, params, and canonical metadata that the client router will inspect after startup
- route reloads and direct-entry SSR must always rebuild bootstrap from server state after mutations; client memory caches may speed hydrated navigation, but they are never authoritative for a fresh document request
- SSR bootstrap should avoid duplicating route payload blobs under both `data` and `requests`; request metadata should stay serializable for hydration cache priming, but the primary payload remains the source for actual route data bytes
- pages that cannot guarantee consistent seeded data between server render and client hydration should stay client-only until that mismatch risk is removed

---

### TESTING_STORIES

# Atlas Commerce OS Testing Stories

This file turns the Atlas walkthrough and route contracts into the default user-story test inventory.

## User Story Groups

### Public commerce

- discover a product from the catalog with query-backed filters and sorting
- open a product detail route with warehouse-aware promise messaging and related products
- submit a moderation-aware product question
- request a restock notification for a constrained product
- send a business quote inquiry with quantity and follow-up context

### Internal inventory and logistics

- open inventory, switch saved views, filter by warehouse, search by SKU, and preserve the workspace state
- open SKU detail, adjust thresholds, and verify history updates in-route
- create or approve a transfer from an imbalance recommendation
- open a receiving session, classify a discrepancy, and close the session with a visible audit trail
- open purchase orders and verify vendor state plus inbound shipment rows

### Moderation and warehouse operations

- review pending public feedback, approve, reject, or flag it, and confirm the moderation message path remains coherent
- inspect a warehouse list, drill into one warehouse, and compare backlog, staffing, and transfer pressure

### Personalization and SSR

- restore theme, locale, density, warehouse, and saved-view context on reload or direct route entry
- verify public routes keep expected title, description, canonical, and crawlable content before hydration

## Recommended Acceptance Stories

- Customer browses the sales catalog.
- Customer inspects a product detail page.
- Customer submits a product comment.
- Customer requests a restock notification.
- Buyer requests a quote.
- Operator works the inventory table.
- Operator edits SKU thresholds.
- Manager creates a transfer.
- Receiver reconciles an inbound shipment.
- Moderator reviews customer feedback.
- User resumes personalized state.
- Public page remains SEO-safe.

## Coverage Direction

- browser tests should keep carrying the primary Atlas interaction proof for public browsing, direct entry, persistence, and route-aware workflows
- future persistence, handler, and repository tests should build on these stories instead of inventing separate acceptance criteria

---

### WAREHOUSE_NOTES

# Atlas Commerce OS Warehouse Notes

Atlas warehouse routes should feel like a bridge between the storefront promise and the internal operating model.

## Public Warehouse Experience

- the warehouses index should present each warehouse as a locality and service-promise surface, not as an operational dashboard
- each public warehouse page should include a hero summary, service region, stocked highlights, fulfillment promise, and a lightweight operational notice when demand is constrained
- SKU availability by warehouse should translate stock posture into buyer-facing promise copy, with direct pivots back to the product route or across to a better-fit warehouse
- the shipped warehouse directory now uses a merchandised regional-commerce overview, richer route cards with pressure and stocked-volume cues, and a warehouse-detail story band so the public warehouse routes carry the same premium editorial weight as the storefront instead of reading like detached utility screens
- the shipped warehouse-availability route now reuses the product-detail visual scaffold: dark hero, first-screen metrics, and a dedicated action rail so the regional SKU lane reads like a sibling of product detail instead of a detached utility form stack
- the shipped warehouse-availability route now layers warehouse-specific promise copy on top of that scaffold through a dedicated promise band plus regional/service support points in the action rail, so buyers keep the hub context while deciding whether to quote, reserve inbound stock, or ask for help

### Warehouse route work map

- warehouse list merchandising: regional-commerce overview plus route cards that sell each warehouse as a locality-specific choice
- warehouse detail hero: location summary, service posture, and regional promise framing in the first screenful
- capability summary: staffing, backlog, pressure, and focus translated into buyer-facing regional posture rather than internal-only jargon
- product volume story: stocked-highlight count and regional product showcase that keep the route feeling merchandised instead of purely logistical
- route-specific CTA tasks: regional browse, compare-all, and availability pivots that move the buyer forward without dropping them into a disconnected utility flow

## Internal Warehouse Experience

- the internal warehouse list should emphasize pressure comparison across locations using backlog, low-stock posture, staffing, and transfer pressure
- warehouse list columns and filters should cover location, service region, active SKUs, low-stock count, receiving backlog, transfer pressure, and fulfillment SLA posture
- the warehouse detail page should keep backlog, low-stock items, receiving queue, transfer pressure, staff notes, and next actions visible in one operational surface
- warehouse detail sections should read as overview, inventory health, inbound work, outbound pressure, notes, and activity history even when Atlas keeps them visible in one page instead of strict tabs
- warehouse comparison views can stay list-based in the first release, with the warehouse list itself acting as the comparison surface before Atlas adds richer charting

---

