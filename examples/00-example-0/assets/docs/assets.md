# Assets

This page defines the current framework-level asset delivery story for GoWebComponents.

Use it when you need a consistent plan for wasm binaries, `wasm_exec.js`, CSS, images, fonts, copied static files, and hashed build outputs across SSR, prerender, and static-hosted apps.

## At A Glance

The asset story should let application code refer to stable logical asset names while build and deployment layers decide the final emitted filenames.

In practice, that means:

- application code points at logical assets
- build tooling fingerprints, copies, and emits manifests
- SSR, prerender, and static HTML all resolve through the same deployment-aware lookup rules
- hosting infrastructure serves immutable assets aggressively and keeps HTML or manifest entrypoints short-lived

If those responsibilities blur together, teams usually end up hard-coding output filenames into templates, route handlers, or bootstrap payloads.

## Scope

The asset story spans three layers:

- core rendering and HTML generation, which only need stable asset references
- build or export tooling, which can fingerprint files, copy assets, and emit manifests
- deployment infrastructure, which serves immutable files, applies cache headers, and handles base paths or CDN origin layout

The framework should not require every app to invent its own conventions for these boundaries.

## What Belongs In Core

Core should stay responsible for:

- rendering stable asset URLs that come from application code or a build manifest
- keeping SSR, hydration, and prerender flows compatible with hashed or base-path-prefixed asset references
- documenting the safe way to reference wasm, JS helpers, stylesheets, images, fonts, and media from generated HTML

Core should not become a bundler or image optimizer.

## What Belongs In Tooling

Build or export tooling should own:

- copying static files into deployable output directories
- emitting asset manifests for fingerprinted outputs
- hashing or versioning filenames
- reconciling route HTML with hashed asset references
- choosing whether bootstrap payloads are inline, sidecar JSON, or both

This keeps the runtime API small while still giving applications one documented deployment shape.

## Output Expectations

The intended output contract across SSR and prerender is:

- server-rendered HTML and prerendered HTML can both resolve assets through the same manifest lookup rules
- generated HTML can honor deployment base paths and origin separation without hard-coded absolute URLs
- fingerprinted assets are safe to serve with long-lived immutable caching
- mutable entrypoints such as HTML documents or manifest files stay short-lived and easy to invalidate

## Asset Manifest Guidance

Application builds should have one logical source of truth for emitted asset names.

The intended manifest rules are:

- map stable logical names such as `app.wasm`, `wasm_exec.js`, `app.css`, or `hero.jpg` to emitted filenames
- allow SSR and prerender code to resolve the same logical asset names without hard-coded build output paths
- keep route HTML generation dependent on the manifest lookup, not on guessed hashing rules
- include public URL or base-path-aware values rather than only local filesystem paths

Manifests should be sufficient to resolve:

- wasm binaries
- JS helpers such as `wasm_exec.js` or module entrypoints
- CSS bundles
- images, fonts, and copied static assets when they are fingerprinted

Applications do not need a single mandatory manifest file format yet, but they should avoid embedding emitted filenames directly throughout templates or route handlers.

## Example Manifest Shape

The framework does not require one manifest schema, but a good manifest should be readable, deployment-aware, and broad enough to cover wasm, CSS, JS helpers, and routed split artifacts.

```json
{
	"app.wasm": "/static/app.4f3a2b1c.wasm",
	"wasm_exec.js": "/static/wasm_exec.a91c3342.js",
	"app.css": "/static/app.2d5e7f09.css",
	"route:catalog": "/static/routes/catalog.7c1ab442.js",
	"route:checkout": "/static/routes/checkout.d9e33110.js",
	"hero.jpg": "/static/media/hero.6f2210aa.jpg"
}
```

What matters here is not the exact JSON format. What matters is that HTML emitters, SSR handlers, prerender jobs, and deployment validation all resolve the same logical names the same way.

## Split Bundle Output Shape

Split bundles should follow the same manifest-backed asset rules as the rest of the deployment output.

The intended conventions are:

- use one logical namespace for split artifacts so route entry chunks, shared shell chunks, and lazy feature chunks can be named consistently
- treat route-family entry chunks as first-class build outputs alongside wasm, CSS, and other fingerprinted assets
- keep emitted chunk filenames content-hashed and resolve them through the manifest rather than embedding physical paths in HTML or bootstrap data
- let HTML, bootstrap payloads, and preload hints refer to logical chunk names only, with the manifest resolving the final URL
- keep chunk names stable enough that an unchanged logical route family keeps the same human-readable key across releases
- publish the HTML, bootstrap data, manifest, and chunk files together so clients do not mix an old document with a new chunk graph
- fail build or deploy validation visibly when a logical split artifact has no manifest entry instead of discovering that mismatch at runtime

## Cache Busting And Versioning

The intended cache-busting strategy is filename-based, not query-string-based by default.

Recommended rules:

- fingerprint immutable assets with content hashes in the filename
- keep HTML documents, manifests, and equivalent entrypoints on short-lived caching so new releases can redirect clients to the new immutable assets
- treat the manifest as the lookup boundary between stable logical names and hashed physical files
- avoid mixing mutable filenames with long-lived immutable cache headers

This applies most strongly to:

- wasm binaries
- JS helpers and route bundles
- CSS outputs
- large media derivatives that are generated by build tooling

## Operational Checklist

Before calling an asset pipeline production-ready, verify these conditions:

- the wasm binary and matching `wasm_exec.js` come from the same Go toolchain release
- HTML never depends on guessed hashed filenames
- SSR and prerender resolve assets through the same manifest or lookup layer
- base-path or CDN prefix handling is centralized rather than repeated in route templates
- immutable assets and mutable entrypoints use different cache policies
- split chunks, preload hints, and bootstrap payloads are published from the same build graph

If any one of those is handled ad hoc, the deployment story is still brittle.

## Responsive Images And Media

Responsive media remains application-authored markup today, but the contract should be explicit.

Recommended rules:

- emit `srcset` and `sizes` when the image has meaningful size variants
- provide intrinsic `width` and `height` where possible to preserve layout stability
- use `loading="lazy"` for offscreen content rather than above-the-fold hero assets
- use `decoding="async"` when it improves scroll or route-transition smoothness and synchronous decode is not required for first paint
- use poster images for video and embed placeholders when the full media should not block first paint
- treat art-direction variants as explicit application decisions rather than automatic framework behavior

For SSR and prerender, image and media markup should resolve to stable deployment URLs the same way other assets do.

## First-Class Helper Boundary

The repo should not force every app to rebuild common image and media conventions forever, but that support should live in a companion package before it expands core surface area.

The intended split is:

- core continues to render normal HTML elements and stable asset references
- a future companion package may own higher-level image or media helpers for responsive sources, placeholders, sizing defaults, and accessibility affordances

Any future helper package should justify itself by removing repeated application boilerplate around:

- responsive source selection
- intrinsic sizing and aspect-ratio defaults
- loading priority and placeholder policy
- accessible captions, labels, and fallback behavior

Until then, applications should prefer explicit `html.Img`, `html.Video`, and related markup over ad hoc hidden abstractions.

## Preload And Prefetch Guidance

Preload and prefetch hints remain application-owned explicit head markup today.

When a route or layout wants typed convenience helpers, use the `html` package's `Preload(...)`, `ModulePreload(...)`, `Prefetch(...)`, `Preconnect(...)`, `DNSPrefetch(...)`, or generic `Link(...)` builders instead of hand-writing raw link tags in many places.

Recommended rules:

- preload only assets needed for the current route's first paint or near-immediate interaction
- use `modulepreload` for JS module entrypoints when the app actually serves modules separately
- preconnect or DNS-prefetch only to origins that are consistently used on the critical path
- prefer route- or layout-level ownership of hints so the document template can dedupe them before emission
- do not emit overlapping preload and prefetch hints for the same asset unless the transport behavior is intentional and understood

Typical preload candidates are:

- the primary CSS for the current shell
- the active wasm binary or JS bootstrap helper when early fetch materially helps startup
- above-the-fold hero images or fonts that block intended first paint

Typical prefetch candidates are:

- likely next-route images or media
- non-critical route bundles or content assets
- media that should be warm in cache but not contend with current-route critical resources

## Lazy Media Patterns For Routed Apps

Routed apps should treat heavy media as route content, not as shell-critical payload by default.

Recommended patterns:

- keep route layout space stable with intrinsic dimensions, aspect-ratio wrappers, or reserved placeholder containers before the media loads
- lazy-load offscreen images, videos, embeds, and third-party widgets only after the route shell and above-the-fold content are present
- prefer lightweight posters, preview frames, or blurred placeholders for rich media that should not block route transitions
- attach media loading to route ownership so navigating away can cancel or discard no-longer-needed work
- keep captions, transcripts, and fallback links available even when the full media payload is deferred

For routed apps specifically:

- above-the-fold hero media may preload or eager-load when it is part of the route's first meaningful paint
- below-the-fold galleries, embeds, and autoplay-capable media should default toward lazy behavior
- route transitions should not stall on non-critical media discovery if text and primary interaction can render first

## SSR And Prerender Asset References

Server-rendered and prerendered HTML should both resolve asset references through the same deployment-aware rules.

The intended rules are:

- resolve hashed filenames through a manifest or equivalent lookup, not by concatenating guessed output paths
- keep base-path handling centralized so route templates do not hard-code environment-specific prefixes
- let media variants such as responsive sources or posters derive from the same logical asset naming strategy as JS, CSS, and wasm artifacts
- keep bootstrap payloads, HTML documents, and asset references consistent across request-time SSR and prerender so hydration sees the same URLs the server emitted

Applications should avoid:

- embedding local build output paths directly in route templates
- mixing absolute production URLs into development or preview HTML without one configuration boundary
- generating one asset resolution strategy for SSR and a different one for prerender when both deliver the same route family

## CDN And Static-Host Deployment Guidance

The intended deployment story assumes immutable assets and short-lived documents.

Recommended rules:

- serve fingerprinted assets with long-lived immutable caching
- serve HTML documents and manifest-like entrypoints with short-lived or revalidated caching
- keep compression enabled for JS, CSS, JSON, SVG, and other text assets, and verify wasm is served with the expected content type and compression behavior for the chosen host
- separate origin concerns intentionally when CDNs, media hosts, or API hosts differ, and pair that with explicit `preconnect` or hint policy rather than accidental cross-origin sprawl
- treat CDN invalidation as an exception path for mutable documents or manifests, not the normal strategy for immutable hashed assets

For static hosts:

- make base-path and trailing-slash behavior predictable before generating route HTML
- ensure copied public assets and fingerprinted assets can both be addressed by the emitted manifest or deployment conventions
- keep redirect and fallback behavior aligned with the router mode the app actually uses

## Deployment Shape

Applications should plan around:

- hashed assets for wasm binaries, JS helpers, CSS, fonts, and large media derivatives when practical
- one manifest or equivalent lookup table that maps logical asset names to emitted files, including split-bundle route chunks and lazy feature chunks
- copied public assets for files that should keep stable names
- explicit cache-header strategy for HTML, manifests, and immutable assets

More detailed end-to-end example apps and concrete exporter integrations remain separate backlog work.

The standard here is not sophisticated tooling for its own sake. The standard is that asset URLs remain correct across development, prerender, SSR, and deployment without application teams inventing a new naming convention for every environment.
