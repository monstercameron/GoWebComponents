# Runtime Configuration and Feature Flags

This page defines the intended runtime-configuration model for GoWebComponents applications.

Use it when deciding how public environment values, deployment metadata, and evaluated feature flags should move through server code, browser code, SSR bootstrap, and hydrated application state without relying on ad hoc globals.

## Runtime Configuration Model

The intended configuration model is layered and explicit.

Applications should distinguish between:

- build-time constants used only by toolchains or server startup
- server-only runtime configuration such as secrets, upstream credentials, signing keys, and internal service endpoints
- public runtime configuration that the browser is allowed to observe, such as public API origins, CDN base paths, environment labels, or feature exposure hints
- evaluated feature flags or experiment assignments that the server may compute per request and then pass to the client as public state

The framework direction is:

- keep configuration values in typed application-owned structures
- keep server-only values in server-owned initialization or request dependencies
- expose public runtime config to client code through one documented bootstrap or provider path instead of scattered globals and string lookups

## Server-To-Client Transfer Rules

Runtime config and flag values may cross into the browser only when they are already safe to expose.

The intended rules are:

- server-only configuration never belongs in SSR bootstrap, prerender payloads, client bundles, logs, or devtools snapshots
- public runtime config may move through SSR bootstrap or explicit client initialization when the browser needs it to resume correctly
- feature flags transferred to the browser should be the evaluated public decision, not the secret targeting inputs that produced that decision
- prerendered and static-exported outputs should treat config payloads as public artifacts and therefore keep them within the same browser-safe boundary as bootstrap data

Examples of browser-safe values:

- public API base URLs
- asset base paths
- environment labels such as `staging` or `production`
- already-evaluated experiment membership for a public UI branch

Examples of server-only values:

- raw targeting rules keyed by private user traits
- signing secrets or auth-provider credentials
- internal-only service discovery data
- moderation or policy inputs that should not be revealed to the browser

## Environment Layering And Override Rules

Configuration should compose by source, not by whichever value is easiest to reach.

The intended order is:

1. framework or application defaults checked into source
2. build-time values that shape emitted assets or deployment metadata
3. server runtime environment values available at process start
4. request-time evaluated public config or flag decisions
5. local development overrides used only in explicitly non-production workflows

Recommended rules:

- later layers may narrow or refine earlier public values, but they should not silently convert server-only values into client-visible ones
- request-time values should be used for per-user or per-request public behavior such as experiment assignment, region label, or locale hint
- local development overrides should be visibly labeled and should never be treated as production defaults
- one configuration owner should be responsible for final normalization before values are injected into handlers or bootstrap payloads

This avoids the common failure mode where build-time constants, server environment variables, and request-scoped flags are all read from unrelated global lookups with no precedence contract.

## Feature-Flag Context And Evaluation Surface

The intended feature-flag model is first-class, but still application-owned in source of truth.

The framework direction is:

- one typed flag snapshot should be exposable to component trees through a provider or context-style surface
- the same evaluated flag snapshot should be available to route loaders, forms, and mutation flows without each app wiring a different ad hoc context
- the browser should receive only the already-evaluated public decision it needs, not the private targeting logic that produced it

Recommended evaluation model:

- server-rendered apps evaluate public flags at request time and pass the public snapshot through bootstrap
- client-only apps initialize from public config and then use one shared provider for the active flag set
- later refresh or revalidation of flags is application-owned unless a future first-party helper makes that lifecycle explicit

This keeps the runtime surface focused on consumption and consistency rather than hard-coding one flag vendor or targeting engine.

## Gating Semantics For Routes, Components, And Experiments

Flag checks should have clear ownership boundaries.

The intended gating rules are:

- route gating should happen at route registration or route-resolution boundaries so unavailable routes do not half-render and then disappear
- component gating should be explicit at the branch where the experimental UI or alternate implementation is chosen
- form and mutation flows may read the same evaluated flags when submit paths or validation rules differ by rollout state
- SSR and hydration should use the same evaluated flag snapshot for the first render so the server and browser do not branch differently on first paint

Recommended pattern:

- gate route availability with server or router-owned decisions
- gate component experiments with explicit branch components or wrappers
- keep long-lived experiments named and centralized instead of sprinkling raw string checks across the tree

## Diagnostics And Safety Guidance

Flag systems create operational debt when they are invisible.

The intended safety rules are:

- expose the active public config and evaluated flag snapshot in development-oriented diagnostics or debug views when safe to do so
- keep secret targeting inputs and server-only config out of browser devtools snapshots and logs
- name flags by behavior, not by vague ticket or temporary wording
- retire stale rollout branches once a flag is fully on or fully off instead of leaving dead checks in place indefinitely
- document default values and ownership for each long-lived flag so release behavior is predictable

The diagnostics goal is visibility into active public decisions, not leakage of private targeting logic.

## Current Boundary

This document defines the intended runtime-configuration contract only.

It does not yet claim:

- a shipped feature-flag provider or context API
- built-in route or component gating helpers
- a first-party staged-rollout example

Those remain separate backlog work.
