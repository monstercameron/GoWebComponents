# Head Management and SEO Surface

This document defines the current head-management contract for GoWebComponents.

The short version is:

- route metadata is first-class only for `Title`, `Description`, and `CanonicalURL`
- those three fields are jointly reconciled between SSR and the client router
- broader head state such as robots tags, Open Graph, Twitter cards, JSON-LD, and resource hints is currently application-owned explicit markup
- there is no dedicated framework head manager yet

That split is intentional. The repo now has a stable, test-covered route metadata slice, but it does not yet claim a larger dynamic head-management runtime than it actually ships.

## Ownership Model

Current ownership is divided into two layers.

### 1. Router-owned managed metadata

The router owns these fields when they are supplied through `router.Options` or `router.MetadataNode(...)`:

- document title
- `meta[name="description"]`
- `link[rel="canonical"]`

This is the only framework-managed head state today.

### 2. Application-owned explicit head markup

Everything else remains application-owned until a broader head manager exists.

That includes:

- `meta[name="robots"]`
- Open Graph and Twitter or X card tags
- alternate locale links
- preload, modulepreload, preconnect, and DNS-prefetch hints
- JSON-LD and other structured-data scripts
- verification tags, app manifests, icons, and similar integration metadata

The recommended current approach is to render those tags explicitly in the server document template or outer SSR document builder, using `ui.RenderToString(...)` plus normal element construction where appropriate.

## SSR Emission For Route-Driven Apps

For route-driven SSR, render the managed route metadata through `router.MetadataNode(...)`, then append any app-owned head tags explicitly.

Example:

```go
headMarkup, err := ui.RenderToString(ui.Fragment(
    router.MetadataNode(router.Metadata{
        Title:        resolved.Title,
        Description:  resolved.Description,
        CanonicalURL: resolved.CanonicalURL,
    }),
    html.Tag("meta", html.Props{Raw: map[string]interface{}{
        "name":    "robots",
        "content": "index,follow",
    }}),
    html.Tag("meta", html.Props{Raw: map[string]interface{}{
        "property": "og:title",
        "content":  resolved.Title,
    }}),
    html.Tag("link", html.Props{Raw: map[string]interface{}{
        "rel":  "preconnect",
        "href": "https://cdn.example.com",
    }}),
))
```

Current recommendation:

- use `router.MetadataNode(...)` for route-managed title, description, and canonical tags
- render robots, social tags, and resource hints explicitly in the same head markup or outer document template
- avoid a client-only patch-up step for first paint on SSR routes whenever the data is already known on the server

## Hydration Reconciliation Rules

Hydration and later navigation follow these rules.

### Managed router metadata

- server-rendered router metadata should be emitted with `router.MetadataNode(...)`
- those tags are marked `data-gwc-router-managed="true"`
- client navigation updates and removes only managed router tags
- duplicate managed tags are deduped to a single surviving managed tag per field
- if a later route omits title, description, or canonical metadata, the router removes or restores only its own managed state instead of mutating unrelated app-owned tags

### Unmanaged application head tags

- unmanaged tags are preserved by router hydration and navigation
- the router does not reorder, merge, or clean up unmanaged tags
- if an application wants route-specific Open Graph, robots, or preload hints after client-side navigation, that logic must be implemented explicitly outside the current router-managed slice

This avoids the worst failure mode right now: a partial head manager that silently stomps on unrelated app-owned tags.

## Route-Level Composition Rules

The current metadata composition rules are intentionally simple.

- leaf routes override parent layout routes for `Title`, `Description`, and `CanonicalURL`
- layout routes provide defaults only when the leaf route omits a field
- there is no title-template API today; compose final title strings when registering the route or resolving the SSR route
- there is no field-by-field merge for social cards, robots directives, or resource hints because those remain application-owned explicit head markup

For nested routes, treat the leaf route as the final authority for canonical URL and primary page description. Layout routes are best used for reusable defaults rather than for partial composition logic.

## Canonical URL Guidance

Canonical URLs should identify the preferred public URL for the content the crawler is seeing.

Recommended rules:

- for parameterized detail pages, emit the fully resolved canonical detail URL
- for filter and sort UIs, only include query parameters in the canonical URL when they change the primary indexable content rather than only the presentation
- for pagination, choose intentionally: either each page is self-canonical if each page is meant to rank independently, or the sequence points to a consolidated listing if only the root listing is meant to rank
- for locale variants, emit one canonical per locale page and pair that with explicit alternate locale tags at the application level
- for preview, authenticated, or environment-specific routes, prefer `noindex` plus an environment-correct canonical strategy rather than exposing production canonicals from non-production pages
- for prerendered and request-time SSR routes, emit the same canonical URL for the same public route regardless of delivery mode

## Structured Data Guidance

Structured data is supported today as an explicit SSR escape hatch, not a first-class router primitive.

Current recommendation:

- emit JSON-LD from the server document template when the route is server-rendered
- treat JSON-LD as server-owned output and avoid rewriting it during hydration unless the app has a fully explicit client-side head integration
- keep structured-data generation close to the same route resolver that determines title, description, canonical URL, and sitemap inclusion

Because the current `ui.RenderToString(...)` path escapes text content normally, applications that need raw JSON-LD script bodies should generate that specific script block in the surrounding server document template until a dedicated raw-script helper exists.

## Resource Hint Guidance

Resource hints are also application-owned explicit markup today.

Recommended rules:

- emit `preload`, `modulepreload`, `preconnect`, and `dns-prefetch` links on the server when the route resolver already knows they are needed
- dedupe hints at the document-template layer rather than expecting the router to reconcile them
- reserve router-managed metadata for document identity, not for transport optimization hints
- prefer stable, high-value hints only; do not emit large per-navigation hint sets that are hard to keep correct

For the broader asset-delivery contract behind those hints, see [ASSETS.md](ASSETS.md).

## Social Metadata Examples

Current recommendation for social metadata is explicit SSR markup next to the route-managed metadata:

```go
ui.Fragment(
    router.MetadataNode(router.Metadata{
        Title:        article.Title + " | Docs",
        Description:  article.Summary,
        CanonicalURL: canonicalURL,
    }),
    html.Tag("meta", html.Props{Raw: map[string]interface{}{
        "property": "og:type",
        "content":  "article",
    }}),
    html.Tag("meta", html.Props{Raw: map[string]interface{}{
        "property": "og:title",
        "content":  article.Title,
    }}),
    html.Tag("meta", html.Props{Raw: map[string]interface{}{
        "property": "og:description",
        "content":  article.Summary,
    }}),
    html.Tag("meta", html.Props{Raw: map[string]interface{}{
        "property": "og:image",
        "content":  previewImageURL,
    }}),
    html.Tag("meta", html.Props{Raw: map[string]interface{}{
        "name":    "twitter:card",
        "content": "summary_large_image",
    }}),
)
```

This keeps the identity metadata under router ownership while leaving richer social metadata explicit and server-owned.

## Sitemap, Robots, and Crawl Control

Use one route resolver or route metadata source of truth for all crawl-sensitive outputs.

Recommended rules:

- canonical URL, robots directives, and sitemap inclusion should come from the same route-level decision logic
- routes that are gated, preview-only, search-result-like, or duplicated across environments should usually be excluded from sitemaps and often emit `noindex`
- authenticated or personalized routes should not be exposed as canonical public content
- keep sitemap generation aligned with the same public URL policy used by SSR canonical emission

## Current Scope Boundary

The current framework contract is intentionally narrower than a full head manager.

Today the project explicitly supports:

- route-managed title, description, and canonical URL
- SSR emission of those managed tags through `router.MetadataNode(...)`
- hydration-safe cleanup and replacement of router-managed tags
- explicit SSR emission of broader head tags using normal element construction or server-template markup

Today the project does not yet support as a first-class managed runtime feature:

- dynamic reconciliation of social cards
- managed resource-hint deduplication
- managed structured-data updates on client navigation
- sitemap or robots generation APIs
- locale alternate-link composition

That broader surface should not be implied in app architecture until the repo ships it explicitly.
