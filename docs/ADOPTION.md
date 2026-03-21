# 1.0 Adoption Baseline

This document defines the minimum ecosystem story GoWebComponents should be able to point to before claiming 1.0-style adoption readiness.

Use it when deciding whether the repo has enough first-party or officially recommended answers for a team to start, validate, ship, and maintain a production app without reverse-engineering the codebase.

## Decision

Before the project should describe itself as having a 1.0-style ecosystem story, it needs a clear answer for all of these areas:

- starter path
- testing recipe
- SSR and hydration recipe
- state story
- routing story
- deployment guidance

Those answers do not all need to be delivered as generated scaffolds or separate packages on day one, but they do need to be first-party docs or explicitly recommended first-party examples that a new team can follow without guessing.

## Minimum Required Pieces

### 1. Starter Path

Minimum answer:

- one officially recommended way to start a new app
- one documented browser-first path
- one documented SSR-aware path when SSR is part of the supported product story

Current repo answer:

- [ONBOARDING.md](ONBOARDING.md) defines the official starting path
- [START_HERE.md](START_HERE.md) and [examples/README.md](../examples/README.md) point adopters to the closest maintained example
- until dedicated starters exist, maintained examples are the official bootstrap reference instead of repo internals

### 2. Testing Recipe

Minimum answer:

- one first-party unit-test path
- one browser or `js/wasm` validation path for runtime behavior
- one documented example of validating SSR or hydration when those features are used

Current repo answer:

- [WORKFLOWS.md](WORKFLOWS.md#test-a-component-or-app-flow) defines the main testing path
- [test/README.md](../test/README.md) covers broader framework and app validation
- [examples/README.md](../examples/README.md) covers example-local Playwright coverage

### 3. SSR And Hydration Recipe

Minimum answer:

- one official request-time rendering path
- one official hydration path
- one documented request pipeline or server integration story

Current repo answer:

- [WORKFLOWS.md](WORKFLOWS.md#add-ssr-and-hydration)
- [HYDRATION.md](HYDRATION.md)
- [SERVER_INTEGRATION.md](SERVER_INTEGRATION.md)
- [examples/18-ssr-server-routing](../examples/18-ssr-server-routing)
- [examples/87-ssr-secure-forms](../examples/87-ssr-secure-forms)

### 4. State Story

Minimum answer:

- one official answer for local component state
- one official answer for shared state
- one official answer for async data and cache-backed state when the framework claims to support that workflow

Current repo answer:

- local state and component hooks through `ui`
- shared state through `state`
- async data through `fetch`
- adoption path documented by [START_HERE.md](START_HERE.md), [REFERENCE_MAP.md](REFERENCE_MAP.md#state-and-data), and the `state` or `fetch` example catalog in [examples/README.md](../examples/README.md)

### 5. Routing Story

Minimum answer:

- one official static-hosting-safe router path
- one official browser-history router path
- one official answer for params, query state, loaders, and guarded routes if those are part of the supported product surface

Current repo answer:

- hash routing and browser routing through `router`
- route loaders, guards, metadata, and hydration-aware mount documented in [WORKFLOWS.md](WORKFLOWS.md#add-routing), [REFERENCE_MAP.md](REFERENCE_MAP.md#routing), and [router/README.md](../router/README.md)
- runnable references in [examples/55-hash-router](../examples/55-hash-router), [examples/56-browser-router](../examples/56-browser-router), [examples/60-route-loaders](../examples/60-route-loaders), and [examples/65-router-guards](../examples/65-router-guards)

### 6. Deployment Guidance

Minimum answer:

- one official browser-only or static hosting path
- one official SSR deployment path when SSR is supported
- one official description of asset, wasm, browser support, and release expectations

Current repo answer:

- [WORKFLOWS.md](WORKFLOWS.md#ship-a-production-wasm-build)
- [DEPLOYMENT_TARGETS.md](DEPLOYMENT_TARGETS.md)
- [ASSETS.md](ASSETS.md)
- [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md)
- [WASM_RELEASES.md](WASM_RELEASES.md)

## 1.0 Adoption Rule

The project should not describe itself as having a mature 1.0-style adoption story unless each required area above has one answer that is both:

- first-party documented
- actively maintained enough that examples, policies, and release guidance stay aligned

The standard is not "every possible workflow has a scaffold." The standard is that a new team can choose a supported path without reading source to infer product policy.

## Current Gaps To Watch

The current baseline is strongest in testing, SSR or hydration guidance, state, routing, and deployment docs.

The most obvious ecosystem gap before stronger 1.0-style messaging is starter shape:

- the repo has an official starting path and maintained examples
- it does not yet ship a dedicated first-party starter app package or bootstrap tool

That gap does not invalidate the rest of the adoption story, but it should be called out honestly whenever the project compares itself to ecosystems with polished app generators.

## Related Docs

- [START_HERE.md](START_HERE.md)
- [ONBOARDING.md](ONBOARDING.md)
- [WORKFLOWS.md](WORKFLOWS.md)
- [WALKTHROUGHS.md](WALKTHROUGHS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md)
- [COMPARISONS.md](COMPARISONS.md)
- [ECOSYSTEM.md](ECOSYSTEM.md)
- [DEPLOYMENT_TARGETS.md](DEPLOYMENT_TARGETS.md)