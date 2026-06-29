# Streaming SSR

This page records the current project direction for streaming SSR work.

Use it to understand what is intentionally deferred until after hydration correctness, and what the first streaming design should optimize for when the project starts that work.

## Current Status

Shipped today:

- request-time SSR through `ui.RenderToString(...)`
- hydration through `ui.Hydrate(...)` and `router.HydrateMount(...)`
- bootstrap transfer, hydration diagnostics, and SSR observability for the non-streaming request-response model

Not shipped today:

- chunked HTML delivery as part of the stable SSR contract
- streamed segment reconciliation in the browser
- partial hydration or event replay tied to a streaming transport

The current shipped SSR model is full-response HTML plus hydration. Streaming remains a design boundary that must fit around that model rather than replace it prematurely.

## Current Position

Streaming SSR is a post-hydration milestone.

That means:

- the current supported SSR path is request-time HTML generation through `ui.RenderToString(...)`
- the current supported resume path is `ui.Hydrate(...)` plus the documented bootstrap helpers
- chunked HTML delivery should not be treated as part of the stable shipped SSR contract yet

The project should not layer streamed shell or segment protocols on top of an ambiguous hydration model. The hydration, bootstrap, and mismatch rules need to stay authoritative first.

## Why Streaming Waits

Streaming introduces several coupled problems at once:

- shell-versus-segment HTML boundaries
- loader completion ordering
- async fallback flushing rules
- streamed-segment client reconciliation
- proxy and gzip buffering behavior
- failure handling after partial bytes are already committed

Until hydration rules are already explicit and verified, streaming would multiply uncertainty instead of adding a well-scoped capability.

That remains the key boundary: streaming is blocked by correctness work, not by a lack of ideas for chunk transport.

## First Route-Loader Streaming Design

When this work starts, the first design should target route loaders rather than a fully general arbitrary streaming renderer.

Recommended first milestone:

1. render and flush a shell that includes stable head markup, the initial route frame, and explicit placeholders for slow loader-backed regions
2. continue loader work on the server
3. stream completed HTML for those deferred regions as later chunks
4. emit enough segment identity for the client to map streamed content to the correct server placeholder region
5. hydrate against the final assembled DOM, not against a transport-specific parallel tree

Constraints for the first design:

- shell HTML must be structurally valid on its own
- placeholders must be explicit in the HTML, not implied by transport timing
- the client should not need a websocket-like patch protocol just to consume streamed route segments
- route loader identity should stay deterministic so later hydration and diagnostics can point to the same region names

## Ownership And Scope

Recommended ownership model:

- server owns chunk scheduling and flush timing
- route loaders own the data that determines whether a region can complete
- the renderer owns shell placeholders and segment identifiers
- hydration still owns the final client resume rules once the browser has the completed DOM

Out of scope for the first streaming slice:

- partial hydration
- client-side event replay for not-yet-streamed regions
- arbitrary DOM patch streams
- a new data transport replacing route loaders

## Relationship To Async Boundaries

The first streaming design should treat async boundaries as server-visible placeholders, not as a second unrelated streaming mechanism.

Recommended rule:

- if a region is pending on the server, flush placeholder HTML first
- when the region resolves, stream its completed HTML later using the same region identity
- the client hydrates the final assembled DOM and should not need a separate "async boundary stream protocol" beyond those stable segment markers

This keeps server streaming aligned with the existing fallback/content mental model instead of inventing a second set of semantics.

More explicit boundary behavior for the first streaming slice:

- if a boundary is already resolved during the initial server pass, emit the resolved subtree in the shell HTML
- if a boundary is still pending, emit placeholder or fallback HTML in the shell
- when the server later resolves that region, stream the completed subtree with the same stable region identifier
- the browser should not hydrate the fallback as a permanent second truth; it should hydrate the final assembled DOM once the streamed HTML for that region is in place
- if a deferred region fails after shell flush, the server should stream an explicit error-region replacement rather than silently dropping the segment

## Transport And Buffering Expectations

Any future implementation should document and test:

- reverse proxy buffering
- gzip or brotli buffering and flush thresholds
- CDN behavior that defeats small chunk flushes
- application server behavior when a loader fails after shell bytes already left the process
- minimum chunk sizes that actually escape buffering in the intended production stack
- whether HTML comments, script sentinels, or explicit region wrappers are needed to preserve segment boundaries through buffering layers

The first implementation should assume local-development chunk visibility is not proof of production incremental delivery.

Recommended transport rules for the first implementation:

- flush one meaningful shell chunk first, not a stream of tiny fragments
- use explicit region wrappers or sentinels so later streamed content can be mapped to the right placeholder deterministically
- measure behavior with compression on and off, because local uncompressed behavior is often misleading
- treat reverse-proxy and CDN configuration as part of the feature contract, not as an afterthought outside the framework
- define what happens when buffering defeats incremental flushes: the page should still complete correctly as a normal full response

## Validation Goals For The First Real Streaming Slice

When the project begins implementation, success should be measured with:

- earlier first byte than the equivalent fully-blocking request
- earlier shell paint for loader-heavy pages
- correct final DOM assembly before hydration
- no duplicate loader work during hydration
- deterministic fallback behavior when a deferred region errors

## Not Shipped Yet

This page is a design contract, not a statement that streaming SSR already exists.

The current shipped SSR surface remains:

- `ui.RenderToString(...)`
- bootstrap serialization helpers
- `ui.Hydrate(...)`
- `router.HydrateMount(...)`

## Current Boundary

This document describes a future design boundary, not a shipped transport.

It does claim:

- the first streaming slice should be layered on top of the existing hydration contract
- route-loader-driven shell and segment streaming is the preferred first scope over a fully general arbitrary stream renderer
- buffering, proxy behavior, and failure-after-flush handling are part of the feature contract, not deployment afterthoughts

It does not claim:

- that streaming SSR already exists in core
- that websocket-style patch protocols are part of the first streaming design
- that hydration, bootstrap, or async-boundary semantics should be replaced by a second unrelated streaming model
