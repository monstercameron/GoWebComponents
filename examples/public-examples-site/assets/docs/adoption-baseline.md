# 1.0 Adoption Baseline

This document defines the minimum ecosystem story GoWebComponents should be able to point to before claiming 1.0-style adoption readiness.

Use it when deciding whether the repo has enough first-party or officially recommended answers for a team to start, validate, ship, and maintain a production app without reverse-engineering the codebase.

## At A Glance

The project should only present a 1.0-style adoption story when a new team can answer six practical questions without reading implementation details:

- How do we start?
- How do we test?
- How do we render and hydrate?
- How do we manage state and data?
- How do we route?
- How do we ship?

This is a release-readiness baseline, not a marketing checklist. The standard is operational clarity for adopters.

## Decision

Before the project should describe itself as having a 1.0-style ecosystem story, it needs a clear answer for all of these areas:

- starter path
- testing recipe
- SSR and hydration recipe
- state story
- routing story
- deployment guidance

Those answers do not all need to be delivered as generated scaffolds or separate packages on day one, but they do need to be first-party docs or explicitly recommended first-party examples that a new team can follow without guessing.

## How To Use This Baseline

Use this document in three situations:

- before describing the framework as production-ready for new teams
- when reviewing whether a new feature creates another required adoption surface
- when deciding whether examples and docs are sufficient to replace missing scaffolds or packages

The goal is not to prove that every workflow is equally mature. The goal is to ensure every supported workflow has one clearly documented path that can be adopted deliberately.

## Readiness Checklist

Treat each area below as a pass or fail gate:

- Starter path: a new team can choose an endorsed bootstrap path without inferring repo structure
- Testing recipe: a team can validate unit behavior, browser behavior, and SSR or hydration behavior when used
- SSR and hydration recipe: server rendering and client resume paths are documented as one coherent flow
- State story: local, shared, and async data state all have recommended first-party answers
- Routing story: supported navigation modes and route features are documented with runnable references
- Deployment guidance: browser, asset, wasm, and SSR deployment expectations are explicit

If one of these answers is weak, fragmented, or only implied by examples, the adoption story is incomplete.

## Minimum Required Pieces

### 1. Starter Path

Minimum answer:

- one officially recommended way to start a new app
- one documented browser-first path
- one documented SSR-aware path when SSR is part of the supported product story

Current repo answer:

- [ONBOARDING.md](onboarding-and-bootstrap.md) defines the official starting path
- [START_HERE.md](start-here.md) and [examples/README.md](../examples/README.md) point adopters to the closest maintained example
- until dedicated starters exist, maintained examples are the official bootstrap reference instead of repo internals

### 2. Testing Recipe

Minimum answer:

- one first-party unit-test path
- one browser or `js/wasm` validation path for runtime behavior
- one documented example of validating SSR or hydration when those features are used

Current repo answer:

- [TESTING.md](testing-surface.md) defines the intended first-party testing surface and package split
- [WORKFLOWS.md](common-workflows.md#test-a-component-or-app-flow) defines the main testing path
- [test/README.md](../test/README.md) covers broader framework and app validation
- [examples/README.md](../examples/README.md) covers example-local Playwright coverage

### 3. SSR And Hydration Recipe

Minimum answer:

- one official request-time rendering path
- one official hydration path
- one documented request pipeline or server integration story

Current repo answer:

- [WORKFLOWS.md](common-workflows.md#add-ssr-and-hydration)
- [HYDRATION.md](hydration.md)
- [SERVER_INTEGRATION.md](server-integration.md)
- [examples/server/server-side-rendering-routing](../examples/server/server-side-rendering-routing)
- [examples/server/server-side-rendering-secure-forms](../examples/server/server-side-rendering-secure-forms)

### 4. State Story

Minimum answer:

- one official answer for local component state
- one official answer for shared state
- one official answer for async data and cache-backed state when the framework claims to support that workflow

Current repo answer:

- local state and component hooks through `ui`
- shared state through `state`
- async data through `fetch`
- adoption path documented by [START_HERE.md](start-here.md), [REFERENCE_MAP.md](reference-map.md#state-and-data), and the `state` or `fetch` example catalog in [examples/README.md](../examples/README.md)

### 5. Routing Story

Minimum answer:

- one official static-hosting-safe router path
- one official browser-history router path
- one official answer for params, query state, loaders, and guarded routes if those are part of the supported product surface

Current repo answer:

- hash routing and browser routing through `router`
- route loaders, guards, metadata, and hydration-aware mount documented in [WORKFLOWS.md](common-workflows.md#add-routing), [REFERENCE_MAP.md](reference-map.md#routing), and [router/README.md](../router/README.md)
- runnable references in [examples/public/hash-router](../examples/public/hash-router), [examples/public/browser-router](../examples/public/browser-router), [examples/public/route-loaders](../examples/public/route-loaders), and [examples/public/router-guards](../examples/public/router-guards)

### 6. Deployment Guidance

Minimum answer:

- one official browser-only or static hosting path
- one official SSR deployment path when SSR is supported
- one official description of asset, wasm, browser support, and release expectations

Current repo answer:

- [WORKFLOWS.md](common-workflows.md#ship-a-production-wasm-build)
- [DEPLOYMENT_TARGETS.md](deployment-targets-and-adapter-expectations.md)
- [ASSETS.md](assets.md)
- [BROWSER_SUPPORT.md](browser-support.md)
- [WASM_RELEASES.md](wasm-build-profiles-and-release-engineering.md)

## 1.0 Adoption Rule

The project should not describe itself as having a mature 1.0-style adoption story unless each required area above has one answer that is both:

- first-party documented
- actively maintained enough that examples, policies, and release guidance stay aligned

The standard is not "every possible workflow has a scaffold." The standard is that a new team can choose a supported path without reading source to infer product policy.

An adoption-ready repo should make the supported path obvious, narrow, and repeatable. If teams need to assemble their own answer from scattered docs, the ecosystem story is not ready yet.

## Current Gaps To Watch

The current baseline is strongest in testing, SSR or hydration guidance, state, routing, and deployment docs.

The most obvious ecosystem gap before stronger 1.0-style messaging is starter shape:

- the repo has an official starting path and maintained examples
- it does not yet ship a dedicated first-party starter app package or bootstrap tool

That gap does not invalidate the rest of the adoption story, but it should be called out honestly whenever the project compares itself to ecosystems with polished app generators.

## Review Standard

When this document is used in planning or release review, prefer these questions:

- Would a new team know which doc to open first for each required area?
- Would two different maintainers recommend the same supported path?
- Would the examples still make sense if the reader never inspected framework internals?
- Are the docs and examples aligned enough that a production team would not need to guess which source is authoritative?

If the answer to any of those is no, the gap is an adoption problem even if the underlying runtime capability already exists.

## Related Docs

- [START_HERE.md](start-here.md)
- [ONBOARDING.md](onboarding-and-bootstrap.md)
- [WORKFLOWS.md](common-workflows.md)
- [WALKTHROUGHS.md](end-to-end-walkthroughs.md)
- [REFERENCE_MAP.md](reference-map.md)
- [COMPARISONS.md](framework-comparisons.md)
- [ECOSYSTEM.md](ecosystem-and-extension-model.md)
- [DEPLOYMENT_TARGETS.md](deployment-targets-and-adapter-expectations.md)