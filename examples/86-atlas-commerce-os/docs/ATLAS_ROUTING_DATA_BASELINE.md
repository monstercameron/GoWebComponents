# Atlas Routing And Data Baseline

This document locks the current Atlas routing, bootstrap, cache, and derived-state rules for the rewrite. It exists so the shell rewrite can build on explicit route-data contracts instead of informal assumptions spread across handlers and components.

## 1. Nested layout and deep-link baseline

- Atlas now uses a nested layout route for `/app/inventory/:sku`.
- The parent SKU route owns the full Atlas shell and primary SKU detail payload.
- `/app/inventory/:sku/threshold-history` is the first deep-linkable secondary workflow.
- The threshold-history route renders as a route-owned side panel, not a separate full-page shell.
- Direct-entry SSR for `/app/inventory/:sku/threshold-history` renders the same SKU route plus threshold panel structure that hydration reuses on the client.

## 2. Public route reuse rules

Public route data should follow these rules:

| Route family | Reuse from SSR bootstrap | Revalidate on navigation | Client cache between transitions |
| --- | --- | --- | --- |
| `/` | shell only | no | no |
| `/shop` | current result page and query | yes when query changes | yes for identical query strings |
| `/shop/:slug` | primary product detail | yes on slug change | yes per product slug |
| `/warehouses` | warehouse directory list | no unless query support is added later | yes for identical route |
| `/warehouses/:slug` | warehouse detail page | yes on slug change | yes per warehouse slug |
| `/warehouses/:slug/availability/:productSlug` | availability detail | yes on slug or product change | yes per warehouse and product pair |

Public lazy or secondary payload rules:

- Product detail keeps the main product payload in bootstrap.
- Related products and public comments remain eligible for lazy or cached follow-up fetches after hydration.
- Warehouse availability keeps the core promise payload in bootstrap because the route story depends on it for first paint parity.

## 3. Internal cache and reuse rules

Internal route datasets should persist locally when operators are likely to bounce between adjacent routes:

| Route family | Bootstrap payload | Shared route/request cache | Notes |
| --- | --- | --- | --- |
| `/app/dashboard` | summary, queue slices | yes | refresh after dashboard-adjacent mutations |
| `/app/products` and detail | page payload | yes | product editor keeps unsaved-change guard local |
| `/app/inventory` | page payload and saved views | yes | filters and queue payload stay cacheable |
| `/app/inventory/:sku` | page payload | yes | parent layout payload is cacheable |
| `/app/inventory/:sku/threshold-history` | page plus overlay payload | yes | proof-of-concept nested route and overlay cache |
| `/app/warehouses` and detail | page payload | yes | warehouse detail remains route-local rather than shell-global |
| `/app/purchase-orders`, `/app/receiving`, `/app/transfers` | page payload | yes | detail routes can reuse list-family cache prefixes |
| `/app/comments`, `/app/settings` | page payload | yes | settings also keeps saved-view data cacheable |

Internal data that should not be shell-global yet:

- purchase-order detail rows
- receiving detail rows
- warehouse item detail
- threshold-history overlay data

Those remain route-local caches until the rewrite introduces repeated cross-route reopen behavior that justifies promotion into a shared cached resource primitive.

## 4. Bootstrap serialization rules

Serialize into bootstrap:

- route metadata
- preferences, locale, theme, saved views, and user session
- the primary route page payload for every SSR route
- the threshold-history overlay payload for `/app/inventory/:sku/threshold-history`

Fetch lazily after hydration:

- product-related content that is secondary to the main buying path
- public comments
- any future heavy secondary drawer, side sheet, or diagnostic panel

Bootstrap payload rule of thumb:

- first paint and hydration parity wins over early laziness
- secondary or repeat-open panels should move out of bootstrap before primary route data does

## 5. Bootstrap size and externalization budget

Atlas now tracks bootstrap size on SSR responses with:

- `X-Atlas-Bootstrap-Bytes`
- `X-Atlas-Bootstrap-Mode`

Budget rules:

- stay inline by default while the serialized bootstrap remains comfortably small and route-local
- consider externalization first for routes that carry primary page data plus secondary panel data
- treat 25 KB to 35 KB of serialized JSON as the warning band for Atlas rewrite review, not as a hard fail threshold

External bootstrap proof of concept:

- `?atlas_bootstrap=external` is now wired for `/app/inventory/:sku/threshold-history`
- that route emits `ui.RenderBootstrapReferenceScript` and serves the full bootstrap payload from `/__atlas/bootstrap.json`
- unsupported routes ignore the flag and stay in inline mode so the example does not overclaim coverage

## 6. SSR-safe boundary register

Current SSR boundary register:

- none

Rule for follow-up work:

- if a rewritten Atlas component cannot stay SSR-safe on first pass, it must be listed here and isolated behind a clear hydration-only boundary
- no component should silently degrade from SSR into client-only behavior without an explicit todo entry

## 7. Duplicate-query audit and consolidation follow-ups

Current duplicate-query findings:

- `/app/inventory/:sku/threshold-history` fetches the parent SKU page payload and the overlay payload separately on client navigation
- warehouse item detail still loads purchase-order detail rows per candidate order when filtering warehouse-specific orders
- dashboard summary is assembled from separate comments, transfers, and receiving queries
- threshold-history direct-entry SSR loads SKU detail plus threshold-panel data separately

Consolidation follow-ups to keep in the backlog while the shell rewrite proceeds:

1. collapse the threshold-history child-route bootstrap and client loader path into one reusable combined route payload helper
2. replace per-order warehouse item detail reads with a warehouse-and-SKU scoped purchase-order read
3. revisit dashboard summary composition if the page grows beyond the current queue slices

Precompute or persisted-summary rule:

- do not precompute Atlas route summaries yet
- revisit precompute or persisted summary tables only if route-family summaries become expensive enough to show up in payload-size or responsiveness review

## 8. Canonical derived-state layer

The canonical derived-state split is now:

- server responses: route summaries, warehouse pressure records, and other data that should already be stable before HTML is emitted
- shared Atlas helpers: public merchandising copy, warehouse promise copy, and inventory rollups reused by both server summaries and UI composition
- state computed or derived primitives: reserved for future shell-wide filters, preferences, and cross-route workspace state once the rewrite introduces them
- route-local memoized transforms: lightweight presentation-only sorting or grouping that is not worth promoting into shared helpers

The current canonical shared derived-state layer now owns:

- inventory rollups for visible lanes, risk lanes, inbound totals, reorder totals, and SKU-with-inbound counts
- inventory summary-card grouping for the inventory queue
- public status messaging for catalog cards, product support cues, warehouse service tone, and warehouse availability guidance

Rule:

- expensive counts, totals, urgency bands, and filter summaries should be computed once in the shared derived-state layer or on the server, not rederived ad hoc inside multiple route components

## 9. SSR bootstrap reuse rules by route family

- Landing: no route data request; bootstrap holds only shell state
- Public list/detail routes: bootstrap holds the primary route payload, then the same request URL becomes the cache key for transition reuse
- Internal list/detail routes: bootstrap holds the primary page payload, and route-local request caches reuse that exact API URL on transition
- Nested inventory threshold-history route: bootstrap holds the parent page payload plus overlay payload so direct-entry SSR and hydrated nested routing stay aligned

## 10. Client cache lifetime rules

| Dataset | Cache lifetime rule |
| --- | --- |
| related products | keep until product slug changes or a product mutation invalidates `/shop` |
| public comments | keep per product slug until comment mutation notice invalidates `/shop` |
| saved views | keep for the session until saved-view create or import notices invalidate settings, inventory, and dashboard caches |
| preferences | keep for the session until preference-save notice invalidates settings-adjacent routes |
| purchase-order detail | keep per detail route until purchase-order create or status-change notice invalidates the PO family |
| receiving detail | keep per detail route until receiving reconcile invalidates receiving and PO families |
| warehouse detail side data | keep per warehouse route until inventory, transfer, or purchase-order notices invalidate the warehouse family |

