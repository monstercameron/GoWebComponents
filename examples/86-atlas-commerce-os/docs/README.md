# Atlas Commerce OS

This is the kickoff scaffold for the flagship Atlas Commerce OS example.

It currently establishes:

- the shared Atlas SSR page tree used by both the native server and browser hydration
- the current product narrative and operations visual direction
- shared bootstrap shapes, repository contracts, seed data, and token hooks
- a native Go server with sqlite-backed mutations and request-time rendering
- a runnable wasm hydration entrypoint for the server-rendered Atlas shell

What it shows now:

- server-rendered storefront and internal Atlas routes
- public catalog, product, warehouse, and warehouse-specific availability surfaces
- validated public quote, restock, and product-question forms
- internal inventory, transfers, receiving, moderation, settings, purchase-order, and warehouse routes behind mock auth
- shared hydration from the same Atlas payload used during SSR
- sqlite-backed mutations, CSRF protection, and request-time recovery flows

Current migration focus:

- rewrite the Atlas public and internal shells using the React design references under `design/`
- match the GoWebComponents HTML and CSS output to the React compositions as closely as Atlas route semantics allow
- replicate and enhance the React interaction patterns through Atlas SSR plus WASM hydration
- track execution details in `docs/ATLAS_COMMERCE_OS_TODO.md`

## Current Structure

- `client/`: wasm hydration entrypoint for the server-rendered Atlas surface
- `shared/`: shared Atlas SSR contracts and page tree, repository interfaces, seed data, design tokens, and cross-surface helpers used by both the browser and native server paths
- `server/`: native Atlas server, auth and sqlite layers, SSR handlers, mutation endpoints, and server-owned data assets under `server/data/`
- `docs/`: this README plus supporting notes and local reset workflow helpers

## Support Files

- `server/data/schema.sql`: initial sqlite schema scaffold
- `server/data/migrations/001_initial_schema.sql`: numbered startup migration used by the native server
- `shared/repository/contracts.go`: repository interfaces and query contracts
- `shared/atlas/bootstrap.go`: shared bootstrap contract now used by both the native server and the wasm client
- `shared/atlas/page.go`: shared Atlas SSR page tree used for server rendering and browser hydration
- `client/main.go`: js/wasm hydration entrypoint for the server-rendered Atlas surface
- `docs/ATLAS_COMMERCE_OS_TODO.md`: active rewrite plan for React-to-GoWebComponents Atlas parity work
- `docs/scripts/reset-seed.ps1`: removes the local Atlas sqlite path so demos can return to a clean seed baseline once persistence lands
- The former standalone planning and review notes now live in the consolidated sections below.

## Validation

From `examples/`:

```powershell
npm run test:atlas-ssr
```

Atlas now validates through the native-server SSR suite. The retired static hash-router lane has been removed so browser coverage stays aligned with the real server-rendered example.

## Build

From the repo root:

```powershell
Set-Location .\examples
.\build.ps1 -Example 86-atlas-commerce-os
```

The example build script now detects the reorganized wasm entrypoint at `client/main.go`; the equivalent direct build is:

```powershell
go build -o .\examples\static\bin\atlas-commerce-os.wasm ./examples/86-atlas-commerce-os/client
```

## Run

From the repo root:

```powershell
go run ./examples/86-atlas-commerce-os/server
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
- [ROUTE_ARCHITECTURE](#route-architecture)
- [ROUTE_DATA_NOTES](#route-data-notes)
- [SCREENSHOT_CHECKLIST](#screenshot-checklist)
- [SCREENSHOT_NOTES](#screenshot-notes)
- [SEED_STRATEGY](#seed-strategy)
- [SEO_NOTES](#seo-notes)
- [SERVER_ENDPOINT_INTEGRATION_PLAN](#server-endpoint-integration-plan)
- [SSR_BOOTSTRAP_NOTES](#ssr-bootstrap-notes)
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

### Inventory management

The inventory view should be the strongest internal screen. It needs to feel fast, controlled, and trustworthy with visible sort state, filters, saved views, and bulk actions.

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

- dashboard
- inventory
- warehouses
- transfers
- receiving
- comments
- settings

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

### Motion rules

Motion should be short, directional, and functional.

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

# Atlas Commerce OS Diagnostics Notes

Atlas diagnostics are intentionally hidden from normal reviewer flows.

## Enable Diagnostics

- open any internal Atlas route with `?diag=1` in the hash query
- example: `#/app/dashboard?diag=1`
- diagnostics are only intended for development review and should stay off in normal demos

## What Diagnostics Show

- the developer-only Atlas diagnostics panel inside the internal shell
- a manual `SnapshotNow` summary for current route, runtime counts, and first diagnostic message
- the floating `devtools.Panel` overlay for route, runtime, profiling, diagnostics, and tree inspection
- current preference state and route-state inspection values
- derived inventory summary values even when the inventory route is not the active leaf
- the latest loader timing ledger for Atlas loader-backed routes

## Review Use

- verify a loader-backed route mounts cleanly before checking its specific workflow
- confirm persisted theme, locale, density, and warehouse values on direct entry
- inspect query and params when testing route recovery or internal drill-in links
- use the loader ledger to spot regressions after route-loader changes

---

### FRAMEWORK_COVERAGE

# Atlas Framework Coverage

This file tracks whether Atlas Commerce OS exercises the major GoWebComponents framework features in real product flows.

Status legend:

- `Implemented`: present in Atlas today with a concrete screen or workflow.
- `Planned`: required for Atlas, but not wired into the example yet.

## ui

- [x] `ui.Render` and `ui.CreateElement` implemented in the Atlas app entry.
- [x] `ui.UseState` implemented across settings, inventory, SKU detail, transfer, receiving, and moderation screens.
- [x] `ui.UseEffect` implemented for browser preference and DOM attribute synchronization.
- [x] `ui.UseEvent` implemented for routed controls and workflow actions.
- [x] `ui.UsePrevious` implemented in the transfer workflow state summary.
- [x] `ui.UseDeferredValue` implemented in the inventory workspace search preview.
- [x] `ui.UseDebounced` implemented in the inventory workspace search preview.
- [x] `ui.UseReducer` implemented in the transfer workflow stage controls.
- [ ] `ui.Fragment` planned for composite route sections and grouped table cells.
- [ ] `ui.UseRef` planned for focus restoration in overlays and command surfaces.
- [ ] `ui.UseThrottled` planned for dense table telemetry and resize-driven UI state.
- [x] `ui.UseNavigate` implemented for internal app-shell redirects.
- [x] `ui.UseId` implemented for public request-form labeling and hint associations.
- [x] public request forms now exercise structured form validation flows; richer internal adjustment forms are still planned.
- [x] `ui.Overlay` and `ui.UseOverlayStack` implemented through the receiving discrepancy sheet and Atlas dialog-backed workflows.
- [x] `ui.Portal` and `ui.PortalTarget` implemented for Atlas secondary workflow UI through the shared overlay host, transfer confirmation dialog, and internal toast viewport.
- [ ] `ui.UseChannel` planned for cross-surface event fanout.
- [x] `ui.UseTask` implemented for the diagnostics reviewer-handoff background task.
- [ ] `ui.UseContext` planned for shell-level operator context.
- [x] `ui.UseTransition` implemented for non-urgent diagnostics probe switching.
- [x] `ui.AsyncBoundary` implemented for async diagnostics probe and cached reviewer-check panels.
- [x] `ui.Lazy` implemented for the deferred reviewer note in diagnostics mode.
- [ ] `ui.ErrorBoundary` planned for route-local failure containment.
- [ ] `ui.RenderToString` and `ui.Hydrate` planned for the SSR hydration path.

## router

- [x] Hash routing implemented for the Atlas browser example.
- [x] Route metadata implemented for current screens.
- [x] Direct route entry and catch-all recovery implemented and browser-tested.
- [ ] Browser router planned for the server-rendered Atlas variant.
- [x] Route params implemented for product and SKU detail routes.
- [x] Query state implemented for catalog browsing.
- [x] Route loaders and revalidation implemented for dashboard, inventory, SKU detail, transfers, and receiving screens.
- [x] Redirects implemented for app-shell entry; auth guards are still planned.
- [x] Nested layout routes implemented for the internal app shell and inventory drill-in.

## html

- [x] Semantic layout primitives implemented throughout Atlas pages and workflow panels.
- [x] Rich HTML forms implemented for public quote, restock, and question-request flows.
- [ ] Additional semantic table and description-list treatment planned for dense operations screens.

## i18n

- [x] Locale persistence and RTL direction implemented for Atlas preferences.
- [ ] Package-level translation resources planned for public and internal copy.
- [ ] Localized route metadata and content loading planned for SSR.

## state

- [x] `state.UseAtom` implemented for shared preferences and inventory workspace state.
- [ ] `state.UseComputed` planned for derived stock-health summaries.
- [x] `state.UseDerived` implemented for inventory summary state that feeds inventory and dashboard surfaces.
- [x] diagnostics now use runtime snapshots for reviewer-visible state inspection through `devtools.SnapshotNow` and `devtools.UseSnapshot`.
- [ ] Snapshot export, import, and storage planned for operator workspaces and saved reviews.

## fetch

- [ ] `fetch.Fetch` planned for data-backed dashboard and catalog refreshes.
- [ ] `ui.UseFetch` planned for imperative refresh surfaces.
- [x] `ui.UseResource` implemented for route-local async diagnostics probe loading.
- [x] `ui.UseCachedResource` implemented for shared cached reviewer checks in diagnostics mode.

## devtools

- [x] `devtools.Panel` implemented for Atlas diagnostics, route state, and runtime inspection behind hidden diagnostics mode.
- [ ] Additional diagnostics wiring planned once async data flows and SSR bootstrap land.

## bootstrap and SSR

- [x] Bootstrap default helpers implemented for route payload defaults and locale direction.
- [ ] Bootstrap script rendering and reading planned for the SSR entry path.
- [ ] SQLite-backed server bootstrap planned for Atlas SSR and hydration reuse.

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
- the broader accessibility pass is still incomplete even though route-entry, form, and SSR smoke coverage exist
- diagnostics cleanup before final release signoff is still outstanding
- several client-side resume behaviors still lean on browser storage in addition to server-backed bootstrap state

What is now shipped:

- sqlite persistence with numbered startup migrations
- request-time SSR from the example 86 Go server
- public and internal JSON endpoints behind one process
- branded recovery pages for missing SSR routes and records
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
- Toggle `?diag=1` on an internal route and confirm the diagnostics panel reflects the same preference and route state that the visible shell is using.

### Overlay-backed workflow checklist

- Open SKU threshold editing, save a threshold update, and confirm the activity timeline and toast both update.
- Open transfer confirmation, approve the transfer, and confirm the transfer state plus toast update without dropping the current route.
- Open the receiving discrepancy side sheet, change classification, and confirm the receiving copy updates after submission.
- Open the moderation confirmation flow and confirm closing the overlay returns focus to the action that launched it.

### Derived inventory summary reviewer guide

- Use `East coast shortages` and `new-jersey-hub` together to confirm the derived summary shifts to `Promise risk`.
- Clear the resume state and confirm the summary returns to the calmer default inventory posture.
- Open diagnostics mode with `#/app/dashboard?diag=1` and compare the developer-only inventory summary panel against the visible inventory route behavior.

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
- Shared atoms and derived inventory summaries now drive settings, inventory, and dashboard continuity across routes.
- Atlas now has a real accessible threshold overlay plus a portal-mounted transfer confirmation flow wired into the internal workspace.
- Receiving now has a discrepancy side sheet, and moderation actions now confirm through a real overlay instead of inline-only state changes.
- Dashboard, inventory, SKU detail, transfers, and receiving now use real route loaders with loading and error fallbacks, manual revalidation, query-aware inventory reloads where applicable, and a shared toast viewport for internal action feedback.
- Internal warehouse list and warehouse detail routes now exist with loader-backed pressure summaries, staffing/backlog stats, and direct-entry browser coverage.
- Public warehouses now include a warehouse detail route plus a warehouse-specific availability route, and both are covered by direct-entry and recovery browser flows.
- Public catalog browsing now keeps sort and pagination in the URL, product detail now includes warehouse promise lanes plus related-product routing, and public quote, restock, and comment forms now expose real validation states.
- Atlas now includes a baseline public screenshot pack for landing, catalog, product, and warehouses routes in both light and dark desktop themes.
- Internal routes now expose a hidden `?diag=1` diagnostics mode with a developer-only shell panel, manual snapshot summaries, loader timing ledger, and the embedded Atlas devtools overlay.
- Route-level document attributes now distinguish public, internal, and recovery surfaces so Atlas can tune light and dark accents separately, apply nested shadow tiers, keep keyboard motion parity with hover states, and tighten Arabic heading rhythm.
- Atlas now includes narrow internal density screenshots plus a mobile-rail cleanup pass, and reviewer docs now call out contrast and motion checks for the current visual system.
- Manual QA, release-readiness, and future server-integration docs now cover nested route shells, shared-state persistence, overlay workflows, derived inventory summaries, focused browser smoke commands, and post-milestone cleanup notes.
- Milestone-four cleanup now includes the resume, diagnostics, contrast, and reviewer-guidance alignment pass so the Atlas backlog reflects the shipped surfaces instead of earlier scaffold assumptions.
- Inventory workspace now persists sort state, supports a route-local density override, renders compact row treatment, and exposes keyboard shortcut hints with browser coverage.
- SKU detail now renders a real threshold history timeline and appends fresh threshold edits directly into the route after overlay saves.
- Transfers and receiving now include detail routes, explicit approval or classification state changes, and a shared logistics activity timeline with direct-entry browser coverage.
- Purchase orders now include list and detail routes, approval affordances, and inbound shipment rows tied to real route data.
- Product hero, dashboard, comment thread, and recovery surfaces now read like product UI rather than framework scaffolding.
- Reviewer docs now cover persisted settings, direct entry, RTL review, inventory resume, screenshot naming, and release-readiness checks.
- Milestone-five cleanup now includes explicit review decisions, performance checkpoints, screenshot-backed visual artifacts, and a reviewer-facing demo checklist so the remaining backlog reflects truly unfinished work rather than closed planning questions.

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
- diagnostics overlay via the hidden devtools surface

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

- Atlas wasm should rebuild successfully from `./examples/build.ps1 -Example 86-atlas-commerce-os`.
- Public landing, catalog, and product routes should mount without blank intermediate states.
- Internal dashboard and inventory routes should mount cleanly on direct entry with persisted preferences applied.

## Loader And Interaction Checkpoints

- Revalidate each loader-backed route at least once and confirm the loader revision changes in place.
- Open at least one modal overlay, one side sheet, and the diagnostics surface to confirm route context remains stable.
- Inventory query, saved-view, sort, and density changes should remain responsive without dropping the existing workspace state.

## Review Baselines

- Run the focused Atlas smoke coverage from `examples/`.
- Recheck the mobile density screenshots after any spacing or shell change.
- Recheck diagnostics mode after any loader, shared-state, or route metadata change.

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

- Atlas wasm build completes from the repo root using `./examples/build.ps1 -Example 86-atlas-commerce-os`.
- Atlas SSR coverage passes using `npm run test:atlas-ssr` from `examples/`.
- Public shell, internal shell, and route-recovery surfaces all render without blank states.
- Settings resume reflects theme, locale, density, and warehouse on direct route entry.
- Inventory resume reflects saved view, warehouse, and query on direct route entry.
- Light and dark themes both receive manual visual review before a demo pack is captured.

## Loader And Overlay Readiness

- Revalidate each loader-backed route at least once during review and confirm revision values change in place.
- Confirm overlay-backed workflows return focus and route context after save, approve, or dismiss actions.
- Confirm diagnostics mode does not interfere with loader-backed or overlay-backed flows when `?diag=1` is enabled.

## Regression Checklist

- Rebuild Atlas wasm after any route, token, or shell-level change.
- Re-run the Atlas smoke suite after any persisted-state or route-entry change.
- Recheck the catch-all recovery route after adding new shell links.
- Recheck RTL settings entry after any locale or styling update.
- Recheck inventory resume after any saved-view, warehouse, or query behavior update.
- Recheck direct-entry plus persistence combinations for settings, inventory, SKU detail, warehouse detail, and diagnostics mode after route-state changes.

## Motion And Density Review

- Verify hover and keyboard-visible states feel equivalent on pills, link tiles, and nested cards.
- Recheck reduced-motion behavior after any route-shell or overlay motion change.
- Use the mobile density screenshots as a baseline when compact or comfortable spacing changes on internal routes.
- Recheck the mobile internal rail after any nested-layout or route-label change.

## Browser-Matrix Plan

- Chromium desktop is the default baseline for every Atlas pass.
- Chromium narrow viewport should be used for settings and inventory responsive checks.
- Firefox should be added once the next overlay and query-state pass lands.
- WebKit should be added once Atlas moves closer to SSR and hydration validation.
- Reduced-motion review should be repeated when overlays, portals, or route transitions become richer.

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

- overview: dashboard
- inventory: inventory list and SKU detail
- warehouses: warehouse list and warehouse detail
- purchasing: purchase orders and purchase-order detail
- transfers: transfers and transfer detail
- receiving: receiving list and receiving session detail
- comments: moderation workspace
- settings: preferences and diagnostics entry context

### Route-level tabs for detail pages

- detail routes use visible sections and focused subpanels rather than hidden tab chrome today
- SKU detail already behaves like overview plus activity workflow sections
- warehouse, transfer, purchase-order, and receiving detail routes should keep overview and activity visible together unless future data density forces explicit tabs
- any later tab state should live in the URL when it materially changes the route meaning

### Nested route structure

- the internal app shell is the first layout boundary
- inventory detail is nested beneath the inventory workspace shell so saved-view and filter context remain mounted during drill-in
- public warehouse detail and availability routes inherit the public shell and keep the Warehouses section active across nested paths

### Overlay-versus-route responsibility

- full operational contexts such as SKU detail, warehouse detail, transfer detail, purchase-order detail, and receiving session detail stay as first-class routes
- shorter secondary workflows such as threshold editing, transfer confirmation, moderation confirmation, and receiving discrepancy resolution stay overlay-backed so the current route context is preserved
- public forms remain inline until Atlas needs explicit confirmation or staged submission states

### Deep-link behavior

- every major list and detail route must open directly from a fresh tab
- query-backed catalog and inventory states should be restorable from the URL or persisted browser state without visiting the landing route first
- diagnostics mode remains deep-linkable on internal routes through `?diag=1`

## Metadata And Ownership Rules

### Metadata rules per route

- every route owns a title, description, canonical path, and social preview fallback
- public landing, catalog, product, warehouse detail, and warehouse availability routes are the SEO-sensitive surfaces
- internal routes still receive titles and descriptions for diagnostics, testing, and future SSR bootstrap, but they are not intended as crawl targets

### Navigation state persistence

- public catalog search, category, warehouse, sort, and page state live in the URL
- direct-detail routes derive their identity from params and keep related merchandising context inside the route body instead of local-only state
- theme, locale, density, default warehouse, inventory saved view, inventory warehouse, inventory sort, inventory query, and inventory density resume from browser storage
- future tab or selection state should only persist if it changes the meaning of a direct link or reviewer handoff

## State And Diagnostics Rules

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

- Atlas diagnostics mode exposes manual runtime snapshots through `devtools.SnapshotNow()` plus the live `devtools.Panel`
- snapshot output is for developer review only and stays behind `?diag=1` so normal reviewer flows remain product-facing
- Atlas does not yet support persisted export or import of workspace snapshots; those remain a future expansion once server-backed review flows exist

### Route revalidation strategy

- loader-backed internal routes should expose explicit revalidation controls while Atlas remains seed-backed and local-state-driven
- route loaders rerun after workflow mutations or when the operator explicitly refreshes a loader-backed route so the visible screen can reconcile with the latest mock workflow state
- query-backed routes should keep visible query state stable while revalidation runs so a refresh does not silently discard the operator context

### Access guard behavior

- public routes remain open
- internal routes assume a mock authenticated session and should later enforce role-aware entry with redirect or recovery behavior rather than blank states
- unsaved form workflows should prefer explicit overlay or route-level confirmation before navigation once Atlas adds longer-lived editing flows

### Page ownership matrix

- public SEO-sensitive: landing, catalog, product detail, warehouse detail, warehouse-specific availability
- public utility: warehouses index and future quote or restock submission confirmations
- internal-only: dashboard, inventory, SKU detail, warehouse operations, transfer flows, purchase orders, receiving, comments, settings
- modal-triggering but route-preserving: threshold edit, transfer confirmation, receiving discrepancy sheet, moderation confirmation, diagnostics overlay
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
- settings: theme, locale, density, default warehouse, diagnostics toggles

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
- diagnostics mode stays client-only even on SSR pages because it is a developer affordance rather than part of the public or operator product surface

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

## Internal Warehouse Experience

- the internal warehouse list should emphasize pressure comparison across locations using backlog, low-stock posture, staffing, and transfer pressure
- warehouse list columns and filters should cover location, service region, active SKUs, low-stock count, receiving backlog, transfer pressure, and fulfillment SLA posture
- the warehouse detail page should keep backlog, low-stock items, receiving queue, transfer pressure, staff notes, and next actions visible in one operational surface
- warehouse detail sections should read as overview, inventory health, inbound work, outbound pressure, notes, and activity history even when Atlas keeps them visible in one page instead of strict tabs
- warehouse comparison views can stay list-based in the first release, with the warehouse list itself acting as the comparison surface before Atlas adds richer charting

---

