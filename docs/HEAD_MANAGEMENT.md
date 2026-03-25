# Head Management and SEO Surface

This document defines the current head-management contract for GoWebComponents.

## At A Glance

- Use `router.MetadataNode(...)` for the router-managed slice: title, description, and canonical URL.
- Use `head.Render(head.Document{...})` when you want one supported companion-owned SSR bundle for robots tags, social tags, alternate locale links, JSON-LD, and resource hints on top of router metadata.
- Treat richer head output as companion-owned SSR composition, not as a new core router-managed runtime.
- Design SSR routes so the first response already contains the correct identity metadata; do not rely on client-side patch-up for the initial paint.

## Quick Decision Guide

Use this rule of thumb:

- if the value identifies the current route itself, prefer `router.MetadataNode(...)`
- if the value is extra SEO or integration metadata, prefer `head.Render(head.Document{...})` or `head.Compose(...)`
- if the value needs custom ordering, deduplication, or raw script handling, keep it in the server document template instead of pretending the router owns it
- if a feature would require the framework to diff arbitrary head state across navigations, treat it as outside the current managed contract

The short version is:

- route metadata is first-class only for `Title`, `Description`, and `CanonicalURL`
- those three fields are jointly reconciled between SSR and the client router
- broader head state such as robots tags, Open Graph, Twitter cards, JSON-LD, alternate locale links, and resource hints remains outside core router ownership, but now has a supported SSR composition surface through the optional `head` companion package
- there is still no broader core-managed head diffing runtime beyond the router-owned slice

That split is intentional. The repo now has a stable, test-covered route metadata slice and a companion-owned SSR composition layer, but it still does not claim a larger dynamic client head-management runtime than it actually ships.

## Ownership Model

Current ownership is divided into two layers.

### 1. Router-owned managed metadata

The router owns these fields when they are supplied through `router.Options` or `router.MetadataNode(...)`:

- document title
- `meta[name="description"]`
- `link[rel="canonical"]`

This is the only framework-managed head state today.

### 2. Application-owned or companion-owned explicit head markup

Everything else remains outside core router ownership even though the companion package now provides a higher-level SSR composition surface.

That includes:

- `meta[name="robots"]`
- Open Graph and Twitter or X card tags
- alternate locale links
- preload, modulepreload, preconnect, and DNS-prefetch hints
- JSON-LD and other structured-data scripts
- verification tags, app manifests, icons, and similar integration metadata

The recommended current approach is to render those tags through the optional `head` companion package when the app wants one reusable SSR head bundle, or directly in the server document template when the app needs fully custom ordering or raw markup.

## Example Shape

The current companion package now has a higher-level SSR bundle surface. Use it to reduce repetitive explicit markup without implying that the framework owns all head reconciliation.

```go
import (
    headpkg "github.com/monstercameron/GoWebComponents/head"
    "github.com/monstercameron/GoWebComponents/router"
    "github.com/monstercameron/GoWebComponents/ui"
)

func renderHead(resolved RouteSEO) (string, error) {
    return headpkg.Render(headpkg.Document{
        Metadata: router.Metadata{
            Title:        resolved.Title,
            Description:  resolved.Description,
            CanonicalURL: resolved.CanonicalURL,
        },
        Robots: resolved.Robots,
        Social: headpkg.SocialMetadata{
            Type:        "article",
            Title:       resolved.Title,
            Description: resolved.Description,
            ImageURL:    resolved.PreviewImageURL,
            URL:         resolved.CanonicalURL,
        },
        Alternates: []headpkg.AlternateLink{
            {HrefLang: "en", Href: resolved.CanonicalURL},
        },
        ResourceHints: []headpkg.ResourceHint{
            {Rel: "preconnect", Href: "https://cdn.example.com"},
        },
        JSONLD: []headpkg.JSONLDBlock{
            {ID: "route-jsonld", Value: resolved.StructuredData},
        },
        Extras: []ui.Node{
            headpkg.LinkRel("alternate", resolved.FeedURL),
        },
    })
}
```

This keeps route identity under router ownership while letting the companion package own the repetitive SSR composition for the richer metadata slice.

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
- prefer `head.Render(head.Document{...})` when the route also needs robots, social tags, alternate locale links, JSON-LD, or resource hints
- keep fully custom tags in the same document template through `head.Compose(...)` or handwritten markup when the app needs precise manual ordering
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

The current composition rules now have one supported companion path for larger apps: `head.Merge(...)` and `head.Resolve(...)` over `head.Document` layers.

- leaf routes override parent layout routes for `Title`, `Description`, and `CanonicalURL`
- layout routes provide defaults only when the leaf route omits a field
- `head.Merge(...)` applies string-field overrides only when the later layer provides a non-empty value
- `head.Resolve(...)` folds multiple `head.RouteLayer` values from parent to leaf so larger apps can centralize head composition rules instead of wiring them ad hoc in every route
- there is no title-template API today; compose final title strings when registering the route or resolving the SSR route
- `Robots` keeps the parent value by default, but `MergeOptions{ClearRobots: true}` clears an inherited value explicitly
- social metadata merges field by field by default, but `MergeOptions{ClearSocial: true}` clears the inherited social bundle explicitly
- alternate links, resource hints, JSON-LD blocks, and extra nodes append by default, but each bundle also has an explicit replace mode through `MergeOptions`

For nested routes, treat the leaf route as the final authority for canonical URL and primary page description. Layout routes are best used for reusable defaults rather than for partial composition logic.

Example:

```go
resolved := head.Resolve(
    head.RouteLayer{
        Document: head.Document{
            Metadata: router.Metadata{
                Title:       "Docs",
                Description: "Shared docs shell",
            },
            Robots: "index,follow",
            Social: head.SocialMetadata{
                Type:  "website",
                Title: "Docs",
            },
            ResourceHints: []head.ResourceHint{
                {Rel: "preconnect", Href: "https://cdn.example.com"},
            },
        },
    },
    head.RouteLayer{
        Document: head.Document{
            Metadata: router.Metadata{
                Title:        "Guide",
                CanonicalURL: "https://example.com/docs/guide",
            },
            Social: head.SocialMetadata{
                Title:    "Guide",
                ImageURL: "https://example.com/guide.png",
            },
            JSONLD: []head.JSONLDBlock{
                {ID: "guide-jsonld", Value: guideStructuredData},
            },
        },
        Merge: head.MergeOptions{
            ReplaceJSONLD: true,
        },
    },
)

headMarkup, err := head.Render(resolved)
```

That gives layout defaults, leaf overrides, and explicit replacement rules in one place while keeping only title, description, and canonical URL under router-owned hydration reconciliation.

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

Structured data is supported today through the `head` companion package as an SSR-only composition concern, not as a first-class router primitive.

Current recommendation:

- emit JSON-LD through `head.RenderJSONLD(...)` or `head.Render(head.Document{JSONLD: ...})` when the route is server-rendered
- treat JSON-LD as server-owned output and avoid rewriting it during hydration unless the app has a fully explicit client-side head integration
- keep structured-data generation close to the same route resolver that determines title, description, canonical URL, and sitemap inclusion

The companion package handles safe inline JSON serialization for JSON-LD blocks so SSR apps no longer need to hand-build that script tag in the surrounding template.

## Resource Hint Guidance

Resource hints are also companion-owned or application-owned explicit markup today.

Recommended rules:

- emit `preload`, `modulepreload`, `preconnect`, and `dns-prefetch` links on the server when the route resolver already knows they are needed, using `head.ResourceHints(...)` for the supported bundle path or the typed `html.Preload(...)`, `html.ModulePreload(...)`, `html.Preconnect(...)`, `html.DNSPrefetch(...)`, or `html.Link(...)` helpers when they keep the route template clearer
- dedupe hints at the document-template layer rather than expecting the router to reconcile them
- reserve router-managed metadata for document identity, not for transport optimization hints
- prefer stable, high-value hints only; do not emit large per-navigation hint sets that are hard to keep correct

For the broader asset-delivery contract behind those hints, see [ASSETS.md](ASSETS.md).

## Social Metadata Examples

Current recommendation for social metadata is explicit SSR markup next to the route-managed metadata, either handwritten or produced with `head.SocialTags(...)`:

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

## Review Checklist

- does the server emit title, description, and canonical metadata before hydration starts
- are router-managed tags limited to the three supported fields instead of mixing in broader head concerns
- are robots, social tags, icons, manifests, and resource hints rendered explicitly and intentionally
- does any JSON-LD or raw script content stay in the server document template until a dedicated helper exists
- do canonical, robots, and sitemap decisions come from the same route-level source of truth

## Runnable Examples

Use these together when you want the end-to-end ownership story instead of isolated policy notes:

- `examples/18-ssr-server-routing`: request-time SSR plus hydration, with `head.Render(...)` composing robots tags, social tags, JSON-LD, alternate links, and resource hints on the first response while later browser navigation still follows the router-owned metadata contract
- `examples/63-router-metadata`: the smallest client-side route-navigation example for the router-managed title, description, and canonical slice
- `examples/70-render-to-string`: the smaller prerender-style `ui.RenderToString(...)` path using the same companion-owned `head.Render(...)` helpers for static or export-oriented HTML generation

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
- companion-owned SSR composition of robots tags, social tags, alternate locale links, JSON-LD, and resource hints through `head.Render(...)`
- explicit SSR emission of broader custom head tags using normal element construction, server-template markup, or lower-level `head` helpers

Today the project does not yet support as a first-class managed runtime feature:

- dynamic reconciliation of social cards
- managed resource-hint deduplication
- managed structured-data updates on client navigation
- sitemap or robots generation APIs
- locale alternate-link composition

That broader surface should not be implied in app architecture until the repo ships it explicitly.
