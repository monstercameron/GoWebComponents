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

## Current Boundary Rule

For now, proposed ecosystem features should not add ad hoc framework-wide registries, directive syntax, or privileged runtime callbacks unless the same capability cannot be delivered through companion packages or package-owned hooks.

That rule keeps the ecosystem story consistent with the broader framework direction: explicit package surfaces, small core ownership, and semver discipline before convenience abstractions.

## Related Docs

- [FRAMEWORK_SCOPE.md](FRAMEWORK_SCOPE.md)
- [API_POLICY.md](API_POLICY.md)
- [HEAD_MANAGEMENT.md](HEAD_MANAGEMENT.md)
- [ASSETS.md](ASSETS.md)
- [TODO.md](TODO.md)