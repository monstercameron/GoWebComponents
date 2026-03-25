# Ecosystem and Extension Model

This document defines the current project stance on plugins, directives, and companion packages.

Use it when deciding whether a new capability belongs in core, should ship as a companion package, or needs a package-specific extension hook.

## At A Glance

The default ecosystem story is intentionally simple:

- keep core small
- prefer companion packages for optional or policy-heavy reusable features
- prefer package-owned hooks when one subsystem clearly owns the concern
- keep domain and deployment policy application-owned by default
- do not introduce a framework-wide plugin lifecycle until multiple real extension categories need the same contract

This doc exists to stop ecosystem growth from becoming an accidental second framework architecture.

## Decision

- core does not define a generic plugin system today
- core does not define a directive model separate from normal Go component composition
- the default extension story is companion packages plus package-specific integration points where the framework already has a clear ownership boundary
- a shared plugin lifecycle should only be introduced after multiple extension categories need the same registration, cleanup, compatibility, and diagnostics contract

The repo now also includes an experimental `plugin` companion package for explicit application-owned plugin hosts. That package is not a privileged core runtime registry; it is a companion abstraction built on documented public APIs.

## Quick Decision Guide

When a new utility or integration is proposed, use this order:

1. Can an application compose it cleanly on current public APIs?
2. If not, does one subsystem clearly own the extension seam?
3. If the feature is reusable but optional, should it be a companion package?
4. Only after those fail, is there evidence for a shared plugin lifecycle?

If a proposal jumps straight to a generic plugin mechanism, it usually has not identified the real ownership boundary yet.

## Why There Is No Generic Plugin System Yet

The current framework already exposes explicit package boundaries for rendering, routing, state, fetch, SSR, hydration, and devtools. Most foreseeable ecosystem additions do not need privileged runtime access. They need one of these simpler shapes instead:

- a companion package layered on stable public APIs
- application-owned composition using existing `ui`, `html`, `state`, `fetch`, or `router` primitives
- a narrowly scoped package hook for one subsystem such as router metadata or devtools diagnostics

Adding a global plugin registry before those needs are proven would create a wider public contract than the repo can currently justify. It would also encourage loosely defined hooks that are difficult to stabilize under semver.

## Why There Is No Directive Model

The framework is Go-first and component-first.

- event wiring already has explicit typed APIs such as `ui.UseEvent`
- DOM structure and attributes are already expressed through `ui` and `html`
- reusable behavior can already be factored into helper functions, components, hooks, and package-level abstractions

A directive layer would only be warranted if the repo proves a repeated class of DOM-attached behavior that cannot be expressed clearly through normal typed composition. That need is not established today.

## Default Extension Path

When a new feature is proposed, use this order of preference:

1. application-owned composition on top of stable public APIs
2. a companion package built on documented package boundaries
3. a subsystem-specific extension point in the package that owns the concern
4. a shared plugin lifecycle only if multiple subsystem-specific extension points converge on the same needs

This keeps core small and makes extension pressure visible before the repo commits to a framework-wide plugin contract.

## Review Lens

Use this document during design review to answer four questions quickly:

- who owns the capability?
- is the problem mechanism or policy?
- does it need semver-backed framework guarantees?
- would a narrower package or companion boundary solve it cleanly?

The right answer is usually the smallest boundary that still keeps correctness explicit.

## Extension Boundary Rules

Every new ecosystem-facing utility should declare one owner before it is added:

- `core` when the capability is required for most apps and depends on the framework's correctness contract
- `package-owned extension point` when one subsystem needs a narrow integration seam but the rest of core does not
- `companion package` when the feature is higher-level, optional, or opinionated
- `application-owned composition` when the behavior is domain-specific or mostly policy rather than framework mechanism

Do not add cross-cutting helpers to unrelated packages just because they are convenient. If a feature spans routing, SSR, forms, and diagnostics, its owning doc or package should say so explicitly instead of scattering partial helpers across those packages independently.

## Ownership Matrix

Use this matrix before adding a new utility.

### Core

Core is the right home when all of these are true:

- most applications need the capability to use the documented framework surface correctly
- the feature participates in rendering, hydration, routing, state, fetch, or diagnostics correctness
- application code cannot reproduce the behavior safely from public APIs alone
- semver stability is worth the long-term maintenance cost

Examples:

- rendering and hook primitives
- route registration, navigation, and route metadata fields already owned by the router
- SSR and hydration primitives
- typed form state and validation lifecycle primitives

### Package-Owned Extension Points

Add a package-owned hook when one subsystem has a clear ownership boundary and extensions need narrow customization without a framework-wide lifecycle.

Examples:

- router-specific metadata, guards, or loader-related extension seams
- devtools-specific diagnostics export or snapshot integration seams
- SSR bootstrap helpers that extend server integration without exposing runtime internals

Rules:

- define the hook in the package that owns the concern
- document what inputs the hook may read or influence
- keep the hook ineffective outside that subsystem unless a broader contract is later formalized

### Companion Packages

Companion packages are the default home when a feature is useful but optional, policy-heavy, or likely to evolve faster than core.

Examples:

- richer head management
- auth-aware routing helpers
- query-cache conventions and data orchestration layers
- animation primitives and higher-level motion helpers
- testing utilities that wrap common setup patterns

Rules:

- build on documented public APIs only
- avoid importing `internal/` packages or depending on undocumented runtime behavior
- keep package-specific policy explicit instead of disguising it as framework law

### Application-Owned Composition

Keep behavior application-owned when it mostly expresses domain policy, deployment policy, or product-specific workflow.

Examples:

- translation catalog sourcing and locale-domain policy
- service-worker registration and offline cache policy
- upload storage rules and mutation conflict handling
- SEO and social metadata beyond the small router-managed slice

## Review Questions Before Adding A Utility

Before merging a new ecosystem-facing utility, answer these questions in the owning doc or PR:

1. Which package or layer owns this capability?
2. Could an application or companion package implement it cleanly on current public APIs?
3. Does it need framework correctness guarantees, or is it mostly policy?
4. If it needs hooks, which subsystem owns those hooks?
5. What tier from `docs/API_POLICY.md` will apply on release?

## Stability Tiers For Extension Authors

Extension authors should classify every dependency they take on using the same repo-wide stability language, but with an ecosystem-specific reading.

### Stable

Use stable public APIs for extension code that needs long-term compatibility across major app lifetimes.

Extension authors may safely build on:

- documented `ui`, `html`, `state`, `fetch`, and router primitives that are already classified as stable in `docs/API_POLICY.md`
- documented SSR and hydration entrypoints whose contracts are part of the stable public surface

Expectation:

- breaking changes only in major releases
- deprecation window before removal
- migration guidance when the contract changes

### Supported Companion

Use this label for ecosystem packages that are production-safe but integration-oriented.

This tier is appropriate when a package:

- layers on stable framework APIs
- exposes integration payloads, diagnostics, or configuration that may grow additively
- is meant to be reused across applications without being part of the rendering core

Expectation:

- major releases carry breaking changes
- minor releases may add optional fields, diagnostics, or helpers
- existing documented integrations must keep working across minor releases

### Experimental

Experimental extension hooks or companion packages are public, but still evolving.

Use this tier when:

- the package depends on experimental framework APIs
- the extension hook is still proving its shape
- real integrations exist, but the repo is not ready to freeze the design for long-term compatibility

Expectation:

- additive and reshaping changes may happen in minor releases
- release notes and migration guidance are still required
- consumers should avoid treating the surface as a long-lived baseline yet

### Internal

Extension authors should treat the following as unavailable for compatibility purposes:

- `internal/*`
- runtime slot, scheduler, fiber, hydration-bookkeeping, and DOM-adapter internals
- example-local glue code unless a doc explicitly promotes it
- test-only helpers under `test/` and `tools/` unless they are later published as a supported companion package

Packages that rely on internal details are experiments, not supported ecosystem integrations.

## Tiering Rules For New Ecosystem Work

When a new extension point or companion package lands, the owning doc should say which tier applies immediately.

Default rules:

- new package-owned hooks start as `Experimental` unless the subsystem contract is already well established
- first-party companion packages may start as `Supported companion` only when they depend exclusively on stable or supported companion APIs
- anything that depends on undocumented runtime behavior is `Internal` by definition
- no new extension-facing surface should ship unlabeled

## What A Companion Package May Own

Companion packages are the preferred place for higher-level ecosystem capabilities that build on stable public APIs without expanding core by default.

Good companion-package candidates include:

- auth helpers layered on router guards, SSR bootstrap, and request integration
- richer head management beyond router-owned title, description, and canonical URL
- cached query abstractions built on top of `fetch` and `state`
- animation or transition helpers that compose with normal component lifecycles
- testing helpers that package common rendering, navigation, and hydration setup

These packages should depend only on documented public APIs unless an API is explicitly marked experimental for extension authors.

## Core Versus Companion Package Split

Use these defaults when deciding where an ecosystem problem belongs.

### Keep In Core

Keep a capability in core when it defines the base framework contract rather than a higher-level integration opinion.

Core should continue to own:

- rendering, reconciliation, hook semantics, and typed element composition
- shared state primitives and snapshot transport primitives
- fetch primitives and typed resource-loading foundations
- route matching, navigation, loader execution, guards, and the small router-managed metadata slice
- SSR, hydration, bootstrap transfer, and runtime diagnostics
- typed local form state, validation lifecycle, and submit-state primitives

### Prefer A Companion Package

Prefer a companion package when the problem is real, reusable, and valuable, but not required for every app to use the framework correctly.

Companion-package candidates include:

- auth integration and session-aware router helpers
- richer head management for social tags, resource hints, JSON-LD, and sitemap helpers
- cached query or mutation orchestration above the low-level `fetch` and `state` layers
- animation and gesture helpers
- asset or media convenience helpers above raw `html` element composition
- testing utilities, harness setup, and common assertions

### Keep Application-Owned By Default

Some ecosystem problems should stay application-owned even when they are common, because they are primarily policy decisions.

Application-owned by default:

- identity-provider selection and credential UX
- locale strategy, translation catalog sourcing, and content governance
- service-worker policy, offline replay rules, and cache invalidation strategy
- upload storage policy, conflict resolution, and authorization rules
- deployment adapters, infrastructure layout, and environment-specific rollout behavior

### Escalation Rule

Do not move a capability from application code to companion package, or from companion package to core, unless one of these becomes true:

- repeated apps or packages are rebuilding the same mechanism instead of only the same policy
- correctness bugs keep appearing because the public framework surface is missing a necessary primitive
- the capability needs semver-backed guarantees across multiple applications
- the package boundary has already stabilized in practice and is costing more to keep outside core than to support directly

## What Would Justify A Shared Plugin Model Later

Revisit this decision only if several first-party or third-party integrations all need the same framework-wide contract for:

- registration during app startup
- scoped cleanup when an app or request shuts down
- compatibility checks against framework versions or capability flags
- standardized diagnostics or feature discovery
- coordinated access across multiple subsystems such as router, SSR, forms, and devtools

Until then, a generic plugin model would add more contract surface than value.

If those needs become real, the project should design a minimal plugin lifecycle around explicit subsystem hooks rather than a vague catch-all callback API.

## Extension Hook Matrix

Until a broader plugin story exists, extension hooks should stay package-owned and subsystem-specific.

### Router Extensions

Allowed influence:

- route registration helpers
- route metadata composition
- guard or policy helpers that resolve to the router's existing allow, block, or redirect decisions
- loader wrappers or revalidation helpers that compose around existing router APIs

Not allowed:

- bypassing documented route matching or navigation semantics
- mutating router internals directly
- introducing hidden navigation side effects outside documented router APIs

### Async Data Extensions

Allowed influence:

- cache-key conventions
- resource wrapper helpers
- retry, invalidation, and stale-data policy layered on `fetch` and `state`
- developer-facing diagnostics around request or cache state

Not allowed:

- changing the meaning of core fetch cancellation or ready or error semantics behind the application's back
- coupling a package to undocumented scheduler or hook internals
- silently replacing the documented low-level fetch contract

### Devtools Extensions

Allowed influence:

- snapshot presentation
- additional diagnostics derived from public devtools payloads
- opt-in developer panels and inspection helpers

Not allowed:

- relying on undocumented runtime memory layout
- shipping production-critical logic that only works when devtools are installed
- treating additive diagnostics as stable control-flow inputs unless the doc explicitly promotes them

### SSR Extensions

Allowed influence:

- document-template helpers
- bootstrap payload shaping for application-owned data
- metadata and head emission helpers layered on documented SSR APIs
- request-pipeline helpers that compose with the documented server integration model

Not allowed:

- mutating hydration bookkeeping or resume semantics through internal hooks
- changing the documented default bootstrap transport implicitly
- depending on undocumented server-render internals instead of the public SSR entrypoints

### Form Extensions

Allowed influence:

- validation helpers
- intent-aware submit wrappers
- server-error mapping helpers
- CSRF, multipart, and workflow helpers built on documented form and fetch APIs

Not allowed:

- redefining the meaning of touched, dirty, error, or submitting state behind the public form API
- introducing hidden transport behavior that bypasses explicit application ownership of endpoints
- coupling form libraries to internal hook slot behavior

## Hook Design Rule

Each extension point should answer these questions in its owning package docs:

1. What public inputs can the extension observe?
2. What outputs can it produce or influence?
3. Is the hook stable, supported companion, or experimental?
4. What cleanup or teardown responsibility belongs to the extension?
5. What part of the framework remains off-limits even when the hook is used?

## Minimal Plugin Lifecycle

The current project does not require a privileged framework-wide plugin registry, but companion packages should still describe the same minimal lifecycle shape.

### 1. Declare compatibility

Each extension should state:

- the GoWebComponents major versions it supports
- which package APIs or hook categories it depends on
- whether those dependencies are stable, supported companion, or experimental

### 2. Register through explicit application code

Registration should happen through normal application composition, not hidden global discovery.

Examples:

- wrapping router registration in a helper
- constructing an SSR document helper explicitly
- creating a devtools panel or diagnostics adapter in app startup code
- installing form or fetch helpers through explicit package calls

### 3. Receive only the documented hook inputs

An extension may read only the public inputs exposed by its owning subsystem.

That means:

- router extensions receive route-level or navigation-level inputs exposed by router APIs
- SSR extensions receive document, bootstrap, or request-shaping inputs exposed by SSR integration docs
- devtools extensions receive public snapshot or diagnostics payloads
- form and async-data extensions receive public state, validation, request, or cache inputs

### 4. Return explicit outputs or cleanup

An extension should make its effects visible through one of these shapes:

- returned helpers or components
- explicit configuration values
- documented wrapper functions
- cleanup functions or teardown steps when the integration allocates background work, subscriptions, or external resources

Cleanup should remain scoped to the package-owned feature. Extensions should not assume access to framework shutdown internals unless a future lifecycle API documents that behavior explicitly.

### 5. Fail predictably when compatibility is missing

If required framework capabilities are unavailable, the extension should:

- fail during explicit setup, not during an unrelated render path later
- report which version, tier, or hook expectation is missing
- avoid partially installing hidden behavior

## Lifecycle Design Rules

When a future first-party or third-party extension documents its lifecycle, it should answer these questions:

1. How is the extension registered explicitly?
2. Which public hook inputs does it receive?
3. What outputs, wrappers, or side effects does it contribute?
4. Does it allocate work that needs cleanup, and how is that cleanup triggered?
5. How does it declare version or capability compatibility?
6. How does it fail when those requirements are not met?

## Companion Package Compatibility And Versioning Policy

Companion packages should follow the main repo semver policy, with extra rules for extension dependencies.

### Versioning Baseline

- supported companion packages should align breaking changes to major releases
- minor releases may add new helpers, optional configuration, diagnostics, and additive payload fields
- patch releases should stay limited to bug fixes, docs, tests, and conservative behavior corrections

### Dependency Declaration Rule

Each companion package should document:

- the minimum supported GoWebComponents major version
- any required experimental hooks or package features
- whether the package is safe on stable APIs only or depends on experimental surfaces

If a package depends on experimental hooks, the package should say so prominently and should not imply the same compatibility guarantees as a stable-only companion package.

### Compatibility Expectations By Tier

When a companion package depends only on stable or supported companion APIs:

- its own public API may be treated as `Supported companion`
- breaking changes belong in major releases
- deprecations should follow the same general window documented in `docs/API_POLICY.md`

When a companion package depends on experimental hooks:

- the package should be labeled `Experimental`
- minor releases may reshape the integration when the underlying experimental framework hook changes
- release notes and migration guidance are still required for non-trivial changes

### Deprecation Rule

First-party companion packages should use the same deprecation discipline as the main repo whenever the package is classified as `Supported companion`:

1. publish a replacement or migration path first
2. mark deprecated APIs clearly in docs and Go doc comments
3. announce the change in `CHANGELOG.md` and `docs/MIGRATIONS.md` when the package is part of the main module docs story
4. remove only in a later major release

### Third-Party Package Guidance

Third-party packages are not governed by the repo's release process, but the recommended policy is the same:

- avoid depending on `internal/` or unlabeled hooks
- publish the supported framework version range clearly
- mirror the framework's tier labels so consumers can judge risk quickly
- treat experimental hook adoption as a shorter compatibility promise unless the package states otherwise explicitly

### Capability Check Rule

Where version ranges are too coarse, companion packages may document capability requirements instead of only version numbers, but those capabilities must be tied to documented public APIs or hook names rather than runtime internals.

## Companion Package Roadmap Candidates

These are the highest-probability companion packages based on current docs and repeated ecosystem pressure.

### Head Management

- likely scope: social metadata, robots, structured data, resource hints, and sitemap helpers beyond the router-managed title, description, and canonical slice
- why it stays out of core for now: the framework already ships a smaller metadata contract, while richer head policy is integration-heavy and varies by app and deployment

The repo now includes `head` as the reference companion package for this category. It stays outside core, builds only on documented public APIs, and demonstrates how a package can extend SSR head composition without privileged runtime access.

### Auth Helpers

- likely scope: route-policy helpers, SSR auth snapshot wiring, redirect-after-login helpers, and session-aware route composition
- why it stays out of core for now: identity providers, session transport, and security policy remain application-specific even when the router integration is reusable

### Query And Mutation Orchestration

- likely scope: cache-key helpers, invalidation policy, optimistic mutation helpers, and query devtools layered on `fetch` and `state`
- why it stays out of core for now: these are higher-level data conventions above the low-level fetch and state primitives the framework already owns

### Animation And Gesture Helpers

- likely scope: transition orchestration, motion primitives, gesture helpers, and route-transition affordances
- why it stays out of core for now: animation policy, browser behavior tradeoffs, and accessibility expectations vary by product and should prove out before becoming framework law

### Asset And Media Helpers

- likely scope: responsive image helpers, placeholder policy, media loading defaults, and typed asset convenience builders
- why it stays out of core for now: the `html` package already covers raw markup while higher-level asset policy remains optional and app-sensitive

### Testing Utilities

- likely scope: render harnesses, hydration test helpers, router test setup, SSR assertions, and common fixture helpers
- why it stays out of core for now: test ergonomics matter, but the stable runtime contract should stay separate from any one testing style or harness package
- current direction: one first-party companion testing module with focused packages such as `test/render`, `test/hooks`, `test/router`, and `test/ssr`; the older `testkit/...` paths remain supported as compatibility aliases; see [TESTING.md](TESTING.md)

## Roadmap Priority Rule

Companion-package candidates should be prioritized when all of these are true:

- multiple examples or app shapes are rebuilding the same mechanism
- the mechanism composes cleanly on documented public APIs
- the package would reduce repeated boilerplate without forcing a new runtime ownership model
- the scope can be documented clearly enough to classify its tier on first release

## Reference Companion Package

The current reference companion package is `head`.

It validates the ecosystem model by showing a real package that:

- depends only on documented `html`, `router`, and `ui` APIs
- registers through explicit application composition instead of hidden framework discovery
- contributes optional helpers without changing core router or hydration semantics
- fits the `Supported companion` tier because it builds on stable APIs and keeps additive semantics

The repo also includes `plugin` as an experimental companion host for explicit manifest-based registration and hook contribution. Use it when an application or companion package wants one explicit integration surface; do not confuse it with a hidden core plugin registry.

## Supported Companion Package: Query And Mutation Orchestration

The `fetch` package now provides a supported query and mutation orchestration layer built on the core `fetch`, `state`, and `interop` primitives.

### Query Orchestration

The primary API is `UseCachedResource[T]`, which provides:

- **shared cache** — concurrent subscribers share one in-flight request; new subscribers join the existing load instead of duplicating it
- **stale-while-revalidate** — `CacheOptions.MaxAge` controls freshness; stale data is served immediately while a background revalidation runs
- **cache invalidation** — `CachedResource.Invalidate()` triggers a manual invalidation; route revalidation via `router.UseRevalidator()` covers route-level cache clearing
- **SSR bootstrap restore** — `CacheBootstrap` seeds the client-side cache from SSR-rendered data so hydration avoids redundant requests
- **persistent cache** — `PersistentCacheOptions` enables durable restore from IndexedDB or localStorage for reconstructible JSON
- **cache-key decoration** — the `plugin.CapabilityAsyncData` hook allows plugins to prefix or namespace cache keys for tenancy or scope isolation
- **request observation** — the same hook allows plugins to attach request observers for logging, telemetry, or diagnostics

### Mutation Orchestration

The mutation side is handled by `OpenMutationQueue`, which provides:

- **persistent offline queue** — IndexedDB-first with localStorage fallback, survives page reload
- **deduplication** — `MutationDraft.IdempotencyKey` prevents duplicate enqueue
- **application-owned replay** — `MutationQueue.Replay()` accepts a `MutationExecutor` so transport, auth headers, and merge policy remain application-owned
- **conflict resolution** — `MutationConflictHandler` receives the `QueuedMutation` and `MutationConflict`, returns a resolution action (retry, dead-letter, remove, or replace), keeping conflict policy application-owned
- **structured diagnostics** — replay reports include succeeded, failed, retried, and dead-lettered counts plus per-mutation error details

### When To Use Each Layer

- for simple one-off requests: `UseFetch` or `Fetch`
- for typed async data with cancellation: `UseResource[T]`
- for shared cached data across components: `UseCachedResource[T]`
- for offline-resilient writes: `OpenMutationQueue`
- for route-level data loading: `router.Options.Loader`

### Boundary Rule

The query and mutation orchestration layer stays in the `fetch` package because it owns the caching, deduplication, and persistence concerns directly. Higher-level application patterns such as entity normalization, infinite scroll orchestration, or query-key conventions belong in application code or future companion packages, not in the core `fetch` package.

## Supported Companion Package: Protobuf RPC

The protobuf RPC companion strategy provides typed unary and streaming RPC clients for Go/WASM applications, layered on the framework's stable `interop`, `fetch`, and `ui` primitives rather than pushed into core.

### Reference Implementation

The `examples/100-ai-chat-wizard` demonstrates the proven pattern:

- proto definition (`chat.proto`) declares service methods with typed request and response messages
- standard `protoc` generates `chat.pb.go` and `chat_grpc.pb.go`
- the client application composes generated stubs with `interop` worker primitives for background streaming
- the server runs standard gRPC with streaming responses tunnelled through the documented server integration model

### Companion Package Scope

A supported protobuf RPC companion package should provide:

- **connection lifecycle** — managed client connection with context propagation, reconnection policy, and graceful shutdown, built on `interop` and `fetch` primitives
- **typed client wrappers** — thin adapter layer that bridges protoc-generated stubs to framework-idiomatic async resources and error handling
- **streaming integration** — typed helpers for server-streaming and bidirectional patterns that compose with `ui.UseEffect`, background workers, and component lifecycle cleanup
- **diagnostics** — structured error reporting, connection-state observation, and devtools panel contribution through the `plugin.CapabilityDevtools` hook
- **auth and metadata propagation** — explicit configuration for per-request headers, tokens, and interceptors, without hiding transport-specific behavior behind implicit framework wiring

### What The Companion Package Does Not Own

- proto schema authorship — application-owned
- identity provider and credential acquisition — application-owned
- server-side gRPC implementation — application-owned
- transport selection (WebSocket, HTTP/2, etc.) — application-owned configuration, not companion-package policy

### External Reference

When implementation work begins, the `grpc-tunnel` repository provides the transport substrate that the companion package should layer typed framework integration on top of.

### Tier

The companion package should start as `Experimental` because it depends on transport patterns that are still proving their shape in browser WASM contexts. Promotion to `Supported companion` requires at least two real applications validating the connection lifecycle, streaming integration, and diagnostics surface.

## Framework Integration Layer Above RPC Transport

The gap between raw RPC transport and framework-level data flow should be filled by a thin integration layer that connects typed RPC methods to async resources, mutation actions, pending and error state, cache invalidation, and route revalidation.

### Integration Surface

The integration layer should provide:

- **`UseRPCResource[T]`** — a typed hook that wraps a unary RPC call in the same `AsyncResource[T]` contract as `fetch.UseResource[T]`, with loading, error, and data states, automatic cancellation on unmount, and reload semantics
- **`UseRPCStream[T]`** — a typed hook for server-streaming RPCs that delivers incremental messages through a channel or callback, with lifecycle tied to component mount and cleanup
- **mutation actions** — explicit helpers that invoke unary RPC mutations and then trigger cache invalidation or route revalidation, using the same `router.UseRevalidator()` and `CachedResource.Invalidate()` surfaces
- **pending and error state** — RPC calls should surface connection errors, deadline exceeded, and status codes through the same error patterns used by `fetch` resources, so UI components do not need transport-specific error handling
- **optimistic mutation** — optional helpers that apply a predicted state update before the RPC confirms, then reconcile or roll back on error

### Composition Rule

The integration layer is a companion package that depends on the RPC companion and on stable `fetch`, `state`, `router`, and `ui` APIs. It should not introduce new runtime primitives or scheduler hooks. If a pattern cannot be expressed through existing hook composition, that is a signal to consider a new stable hook in core, not to bypass the public API from the integration layer.

### Tier

Experimental until the RPC companion itself reaches `Supported companion` status.

## Experimental Companion Package Pattern: AI Provider Applications

LLM-backed applications need one layer above the protobuf RPC transport: a companion package that turns provider catalogs, capability flags, and streaming model interactions into reusable app-facing building blocks without teaching core about model vendors or prompt policy.

### Reference Implementation

The current reference implementation is `examples/100-ai-chat-wizard`:

- `server/app/model_catalog_store.go` loads provider and model metadata from the SQL-backed `model_catalog` table
- `server/app/server.go` exposes that catalog through `ListModelOptions` plus user preference RPCs such as `GetSelectedModel` and `SetSelectedModel`
- `client/app/model_preferences.go` composes generated RPC clients with `fetch.UseCachedResource`, cached revalidation, and cross-tab sync for provider and model selection
- `client/app/stream.go` and `client/app/tts.go` show the streaming chat and speech wrappers that stay tied to component lifecycle instead of bypassing the transport layer

### Companion Package Scope

An AI provider companion package should provide:

- **provider-capability metadata retrieval** — typed helpers that load provider and model capability metadata over the existing RPC companion, including thinking, speech, modality, and pricing flags
- **SQL-backed model catalog queries** — thin query wrappers and bootstrap helpers for server-owned model catalogs so applications can ship first-paint hints while keeping the server as the source of truth
- **capability-aware selector components** — reusable hooks or components for provider and model pickers that filter against required capabilities and explain why models are unavailable
- **streaming-aware client wrappers** — app-facing chat and speech helpers that compose with the RPC companion's unary and streaming clients, `ui.UseEffect`, cancellation, and reconnect-aware lifecycle cleanup
- **preference persistence adapters** — explicit wrappers for user-selected provider or model persistence that remain transport-visible instead of hiding mutations inside framework internals

### What The Companion Package Does Not Own

- upstream provider SDKs, API keys, or server-side inference adapters
- prompt templates, agent policy, tool routing, or application-specific conversation orchestration
- auth/session acquisition; credentials still flow through the existing RPC transport configuration
- global runtime primitives, schedulers, or hidden caches outside the documented `fetch`, `interop`, and `ui` surfaces

### Composition Rule

The AI provider layer depends on the protobuf RPC companion plus stable `fetch`, `interop`, `state`, and `ui` APIs. It should stay as a thin application-integration package. If a needed behavior cannot be expressed through those public surfaces, the fix is a narrowly documented hook in the lower layer, not a new privileged LLM runtime inside core.

### Tier

This package pattern should start as `Experimental`. Promotion to `Supported companion` requires at least two production-shaped LLM applications validating the provider-catalog flow, capability-aware picker surface, and streaming wrappers against real reconnect and failure paths.

## Schema-Driven Client Codegen Strategy

Applications that consume typed API contracts should have clear guidance on which schema-first integrations the ecosystem supports and how generated clients compose with the framework.

### Supported Schema Formats

The following schema formats are candidates for officially supported codegen tooling:

- **Protocol Buffers** — primary, because the protobuf RPC companion already validates the proto-to-Go-client pipeline
- **OpenAPI / Swagger** — secondary, for REST-style APIs where teams already maintain OpenAPI specs; codegen should produce typed Go client functions that compose with `fetch.UseFetch`, `fetch.UseResource[T]`, or `fetch.UseCachedResource[T]`
- **GraphQL** — deferred unless real adoption pressure appears; the query and cache primitives in `fetch` can serve GraphQL clients, but official codegen tooling is not a priority until multiple projects demonstrate the need

### Generated Client Layout

Generated code should follow these placement rules:

- generated files live in a dedicated directory within the application, not mixed with hand-written code
- a `//go:generate` directive or build script documents the generation command, schema source, and tool version
- generated files are committed to source control so builds are reproducible without requiring the codegen tool at compile time
- generated code should not be hand-modified; customization belongs in wrapper functions or companion modules that import the generated types

### Codegen Requirements

Supported codegen tools should produce:

- typed request and response structs
- client functions or methods that accept `context.Context` for cancellation
- error types that compose with the framework's existing error patterns
- no framework-internal dependencies — generated clients should depend only on standard library, protobuf runtime, or the companion RPC package

### Boundary Rule

Codegen tooling should not become a hidden build-system requirement for ordinary applications. Applications that do not use schema-driven APIs should not need codegen tools installed. The framework's core build story remains `go build` for WASM and standard Go compilation for servers.

## Ownership And Upgrade Rules For Generated API Clients

Generated API clients should have explicit ownership, versioning, and review rules so that schema evolution does not introduce silent breakage.

### Ownership

- the proto or OpenAPI schema is application-owned; the application team decides when to evolve the schema
- generated code is derived from the schema and should be treated as a build artifact, not authored code
- wrapper functions, custom interceptors, and framework integration hooks are hand-written and follow normal code review

### Version Mapping

- each schema version should be mapped to a directory or Go package version so multiple schema versions can coexist during migration (for example, `api/v1/` and `api/v2/`)
- the generated client package should declare its minimum supported framework version in a doc comment or compatibility file
- when the schema makes a backward-incompatible change, the generated package should increment its major version or use a new versioned package path

### Review Discipline

- regenerated code should be reviewed as a diff in normal code review, not rubber-stamped as auto-generated
- the review should confirm that new fields are additive and optional, removed fields are deprecated first, and enum values follow backward-compatible evolution rules
- the `CHANGELOG.md` entry for a schema change should note which generated packages were regenerated and what the impact is on client and server compatibility

### Composition With Framework Primitives

Generated clients should compose with the framework through explicit wiring, not hidden registration:

- for query data: wrap generated client functions in `fetch.UseResource[T]` or `fetch.UseCachedResource[T]`
- for mutations: call generated client functions inside mutation actions that trigger cache invalidation or route revalidation
- for streaming: wrap generated streaming calls in the RPC companion's streaming hooks
- for auth headers: pass auth context through explicit configuration, not through ambient framework state

### Conflict With Framework Upgrades

When a framework major version changes stable APIs that generated clients depend on:

- the generated client package should publish a new version compatible with the new framework major version
- migration guidance should cover both the schema codegen step and the wrapper or integration hook updates
- the companion-package compatibility matrix should track which generated-client versions work with which framework versions

## Supported Companion Package: Animation And Gesture

The animation and gesture companion package provides motion primitives, transition orchestration, and gesture recognition helpers that integrate with the framework's component lifecycle, accessibility rules, and routing model.

### What Already Exists In Core

Core provides scheduling primitives that the companion package builds on:

- `ui.UseTransition()` / `ui.StartTransition()` — concurrent scheduling for non-urgent state updates
- `ui.UseDeferredValue()` — lagging derived values for smooth transitions during heavy renders
- CSS transition support in rendered markup
- `prefers-reduced-motion` media query support in examples and style references

These remain in core because they are scheduling primitives, not motion-specific policy.

### Companion Package Scope

The animation and gesture companion package should provide:

- **spring and tween primitives** — declarative animation functions that compute interpolated values over time, exposed as hooks that integrate with component lifecycle and unmount cleanup
- **transition orchestration** — helpers for coordinating enter, exit, and cross-fade transitions across components, including staggered-list and shared-element transitions
- **route transition helpers** — composable wrappers that animate page-level transitions during route navigation, working with the documented router navigation lifecycle
- **gesture recognition** — swipe, drag, pinch, and long-press detectors that normalize pointer and touch events, provide velocity and direction data, and integrate with reduced-motion accessiblity rules
- **reduced-motion integration** — all motion primitives should respect `prefers-reduced-motion` by default, with explicit opt-out when the motion is essential for understanding (such as a progress indicator)
- **overlay and modal transitions** — enter and exit animation helpers for overlays that compose with the overlay lifecycle model documented in `docs/OVERLAYS.md`

### What The Companion Package Does Not Own

- rendering, reconciliation, or component scheduling — core-owned
- route matching or navigation decision logic — router-owned
- overlay stacking order or focus management — application-owned or overlay-helper-owned
- CSS custom properties or design-token resolution — application-owned

### Accessibility Rule

Every motion primitive must accept an explicit `reduceMotion` override and must default to respecting the OS-level `prefers-reduced-motion` setting. The companion package should provide a `UseReducedMotion()` hook that components can use to conditionally skip animations. Animations that serve as the only indicator of state change (such as toast entrance) should provide a non-animated fallback by default.

### Tier

The companion package should start as `Experimental` because animation APIs are notoriously hard to stabilize and the right primitive set needs validation from real applications. Promotion to `Supported companion` requires stable motion primitives, proven gesture helpers, route-transition integration, and accessibility compliance across the supported browser matrix.

## Companion-Package Maturity Checks And Compatibility Matrix

Each official first-party companion package should be tracked in a compatibility matrix so ecosystem depth can be evaluated honestly from one place.

### Tracked Properties

For each companion package, the matrix should record:

| Property | Description |
|----------|-------------|
| Package | Short name (e.g., `head`, `plugin`, `fetch/cache`) |
| Tier | Stable, Supported Companion, Experimental, or Internal |
| Minimum framework version | Lowest GoWebComponents major version the package supports |
| Depends on experimental APIs | Yes or no; if yes, which experimental surfaces |
| Test coverage | Whether the package has unit, integration, or end-to-end test coverage and where those tests live |
| Ownership | Maintainer or team responsible for the package |
| Last verified | Date the package was last tested against the current framework version |
| Migration guide | Whether a migration guide exists for breaking changes |
| Known limitations | Brief list of current gaps or unsupported scenarios |

### Current Matrix

| Package | Tier | Min framework | Experimental deps | Test coverage | Ownership | Last verified | Migration guide | Known limitations |
|---------|------|---------------|-------------------|---------------|-----------|---------------|-----------------|-------------------|
| `head` | Supported Companion | v1 | No | Unit + SSR integration | Core team | Current | N/A (no breaking changes yet) | No dynamic per-request head outside SSR |
| `plugin` | Experimental | v1 | Yes (plugin host lifecycle) | Unit + example-99 integration | Core team | Current | N/A (experimental) | No automatic plugin discovery; explicit registration only |
| `fetch` (cache layer) | Supported Companion | v1 | No | Unit + SSR bootstrap + Playwright | Core team | Current | N/A (no breaking changes yet) | Entity normalization and infinite scroll orchestration are application-owned |
| `fetch` (mutation queue) | Supported Companion | v1 | No | Unit + offline replay | Core team | Current | N/A (no breaking changes yet) | Conflict resolution policy is application-owned; no built-in retry backoff |
| `ai/provider` | Experimental | v1 | Yes (`protobuf RPC` companion, transport-specific streaming patterns) | Example-100 integration + targeted server and client tests | Core team | Current | N/A (experimental) | Validated by RelayDesk only; prompt policy and server-side provider adapters remain application-owned |

### Update Rule

The compatibility matrix should be updated:

- when a companion package is added or removed
- when a companion package changes tier
- when a new framework major version is released and packages are re-verified
- when a companion package adds or removes a dependency on experimental APIs

### Verification Process

Re-verification against a new framework version should include:

1. compile the companion package against the new framework version
2. run the companion package's own test suite
3. run any integration tests that exercise the companion package in a full application context
4. update the matrix entry with the new verification date and any discovered issues

### Related Docs

Companion-package versioning rules are defined in [Companion Package Compatibility And Versioning Policy](#companion-package-compatibility-and-versioning-policy). Tier definitions are in [Stability Tiers For Extension Authors](#stability-tiers-for-extension-authors).

## Current Boundary Rule

For now, proposed ecosystem features should not add ad hoc framework-wide registries, directive syntax, or privileged runtime callbacks unless the same capability cannot be delivered through companion packages or package-owned hooks.

That rule keeps the ecosystem story consistent with the broader framework direction: explicit package surfaces, small core ownership, and semver discipline before convenience abstractions.

## Related Docs

- [FRAMEWORK_SCOPE.md](FRAMEWORK_SCOPE.md)
- [API_POLICY.md](API_POLICY.md)
- [HEAD_MANAGEMENT.md](HEAD_MANAGEMENT.md)
- [ASSETS.md](ASSETS.md)
- [CACHE.md](CACHE.md)
- [OFFLINE_MUTATIONS.md](OFFLINE_MUTATIONS.md)
- [OVERLAYS.md](OVERLAYS.md)
- [SERVER_INTEGRATION.md](SERVER_INTEGRATION.md)
- [TODO.md](TODO.md)
