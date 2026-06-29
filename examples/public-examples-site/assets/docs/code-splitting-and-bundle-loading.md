# Code Splitting And Bundle Loading

This page records the intended bundle-delivery model for GoWebComponents apps.

Use it when deciding how route modules, heavy components, and deferred assets should load without turning the core runtime into a bundler.

## At A Glance

The intended model is simple:

- routes define the primary split boundaries
- `ui.Lazy` defines selective component-level deferral inside those routes
- build tooling emits hashed chunks and a manifest
- SSR, prerender, and the browser all resolve those chunks through the same manifest-backed contract

This keeps code splitting explicit and debuggable. The framework owns loading boundaries and recovery behavior; tooling owns the physical bundle graph.

## Evaluation Result

The current recommendation is:

- keep first-party code splitting convention-driven rather than adding a new core bundler API
- treat route-level splitting as an application or router concern
- treat component-level lazy loading as a `ui.Lazy` and boundary concern
- keep bundle discovery, naming, and invalidation in build tooling and manifests

This keeps the framework focused on rendering, state, routing, SSR, hydration, and runtime hooks while still allowing production apps to load code on demand.

## Framework Model

The intended framework model is a hybrid with a clear default:

- route-driven splitting is the primary organizing principle for large app surfaces
- component-driven splitting is available for heavy nested panels, dialogs, or optional subtrees
- build-tool-driven chunk emission is responsible for the actual output files and hashes
- runtime code should only react to chunk availability, loading state, and failure recovery
- manifests are the shared contract that lets SSR, prerender, and the browser agree on what needs to load

In practice, this means:

- use routes to separate major user journeys and shell areas first
- use `ui.Lazy` only where the deferred subtree has a meaningful fallback and clear loading boundary
- let tooling decide the physical chunk graph, while the app decides the logical split points
- keep the delivery contract manifest-backed so deployment, cache invalidation, and hydration stay predictable

## Mental Model

Treat bundle loading as a three-layer contract:

- application code chooses logical split points
- tooling turns those split points into emitted chunk files
- runtime and SSR layers consume the resulting manifest and loading states

If one layer starts guessing what another layer emitted, the delivery model becomes brittle. The manifest is the boundary that keeps those responsibilities aligned.

## What Belongs In Core

Core should continue to provide the pieces needed to *use* split bundles:

- `ui.Lazy` for deferred component loading
- `ui.AsyncBoundary` and `ui.ErrorBoundary` for loading and retry fallback behavior
- SSR and hydration primitives that can resume once the needed shell code is available
- asset-reference behavior that can consume manifest-backed URLs

Core should not become a bundler, chunk planner, or filename emitter.

## What Belongs In Tooling

Build and release tooling should own:

- deciding which modules or routes become separate chunks
- naming and hashing emitted bundle files
- emitting or consuming a manifest that maps logical bundle names to physical files
- generating `modulepreload`, `preload`, and `prefetch` hints when the app wants to warm a chunk
- ensuring SSR and prerender can discover which chunks the current route family needs
- keeping deployment invalidation aligned with immutable bundle names

## Recommended Conventions

The intended loading model is:

- keep the initial shell lean
- split on route boundaries when a route contains substantial independent feature areas
- split on component boundaries only when the deferred subtree has a clear loading state and an obvious fallback
- prefer explicit `ui.Lazy` usage over hidden auto-splitting heuristics
- keep chunk names and manifest keys stable enough for debugging and cache invalidation
- preload or prefetch chunks from user intent, not from every possible route change

## Rollout Checklist

Before calling a split-bundle strategy production-ready, verify all of the following:

- the initial shell still covers first paint and first interaction for the starting route
- route splits map to meaningful user journeys rather than arbitrary file boundaries
- each lazy subtree has an intentional pending fallback and local recovery path
- emitted chunks are resolved only through a manifest or equivalent lookup layer
- SSR and prerender know which critical chunks belong to the resolved route family
- preload or prefetch behavior is selective and measurable rather than blanket eager loading

If splitting only moves startup cost into a longer chain of runtime fetches, the app is not actually winning.

## Route-Level Conventions

Route-level splitting should organize the app by user journey, not by every individual component.

The intended route rules are:

- keep the app shell, global navigation, and route-independent providers in the base bundle
- split major route groups when a screen family is large enough to justify an obvious loading boundary
- prefer one chunk per meaningful route family over many tiny route-specific fragments
- keep detail pages, admin areas, and heavy secondary workspaces behind their own route boundaries when they are not needed for first paint
- reuse shared layout code in the parent route or shell chunk so child routes stay focused on their own content
- avoid splitting a route so aggressively that the first meaningful paint depends on a long chain of tiny imports

Practical guidance:

- prefetch only the routes that are likely next, such as sibling detail pages, the immediate next step in a wizard, or the next tab in a high-intent flow
- keep route bundles named after the user-facing area they represent so profiling and cache invalidation stay understandable
- if a route is required for the initial landing experience, treat it as part of the shell rather than as a deferred route chunk
- align browser-router, SSR, and prerender routes on the same logical split boundaries so the server and client agree on which chunk family owns each screen

## Component-Level Conventions

Component-level splitting should be reserved for self-contained subtrees that are optional, heavy, or only needed after the shell becomes interactive.

The intended component rules are:

- use `ui.Lazy` for nested panels, dialogs, tabs, widgets, or feature islands that can mount after the main route shell
- pair every lazy subtree with an obvious pending fallback so the user sees that work is happening instead of a blank gap
- pair every lazy subtree with an error fallback that explains the failure and exposes a retry or recovery path
- use `ui.AsyncBoundary` when the lazy subtree needs a distinct pending state, delay, or timeout treatment around the component
- use `ui.ErrorBoundary` when the lazy subtree should recover locally without taking down the route shell
- keep lazy component boundaries stable across rerenders so the loading surface does not move around under the user
- avoid deferring components that are required for the first meaningful interaction on the current route

Practical guidance:

- preserve layout with fixed sizing, skeletons, or placeholder containers so the deferred subtree does not cause layout shift
- if a component owns state that must survive hydration or resumed state, ensure the state is available before the lazy boundary activates or keep that component in the critical shell
- keep retry controls explicit so users can ask for the deferred subtree again without reloading the whole page
- if a component has a large visual or data cost but is still visible above the fold, prefer a shell-friendly split that keeps the immediate interactive surface intact
- treat lazy child components as load-on-demand leaves, not as a way to hide the route's essential structure

## SSR And Hydration

Server-rendered and prerendered pages should know which chunks are required before the client starts interactive work.

The intended rule is:

- the server or prerender output should emit a manifest-backed chunk declaration for the resolved route or page shell
- the document should include the shell code plus any chunks needed for first paint or first interaction, either through direct preload hints or an equivalent bootstrap reference
- chunk discovery should use the same logical names and manifest lookup rules that the build tooling emitted, not guessed filenames
- lazy chunks may load later, but hydration should not race a missing shell dependency
- if a page depends on a deferred chunk during immediate resume, that chunk should be treated as part of the critical delivery path
- bundle order should be resolved by the manifest and the runtime loader, not by template ordering alone

## Maintainer Guidance

Prefer fewer, clearer split points over aggressive fragmentation.

- Do not create tiny chunks that only shift latency into hydration.
- Do not hide critical route structure behind lazy boundaries.
- Do not let server-rendered output and browser runtime use different chunk-discovery rules.

The goal is not maximum chunk count. The goal is predictable delivery, stable hydration, and understandable debugging.

## Current Boundary

This doc intentionally stops short of defining a first-party bundler implementation.

That work remains in application tooling, companion build steps, or future experiments until the repo has stronger evidence that a stable bundler abstraction is worth adding to core.
