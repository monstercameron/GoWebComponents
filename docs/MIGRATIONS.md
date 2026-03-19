# Migration Guide

This file collects release-to-release upgrade guidance for GoWebComponents adopters.

The current project-level migration window starts with the `v3.x` public package layout. Future major releases should extend this file rather than forcing adopters to reconstruct changes from commit history and changelog bullets.

No major release should be considered documentation-complete until this file includes subsystem-specific upgrade guidance for runtime and component authoring, router behavior, SSR and hydration, forms and local state, shared state and data loading, and testing or deployment expectations.

For compatibility guarantees and deprecation timing, see [API_POLICY.md](API_POLICY.md).

## Pre-v3 To v3.x

`v3.x` is the first release line where the repo consistently documents the current public package layout as the supported authoring surface.

### Runtime And Component Authoring

If older code or docs refer to a split such as `dom`, `hooks`, or `render` as the primary authoring model, move application code to the `ui` and `html` packages.

Use these public packages as the default entry points:

- `ui` for components, hooks, render, hydrate, context, portals, async helpers, and form helpers
- `html` for typed DOM builders
- `state` for shared atoms and snapshot helpers
- `fetch` for resource loading
- `router` for routing
- `devtools` for optional inspection and diagnostics

Migration guidance:

- stop importing runtime internals from `internal/runtime`
- stop treating repo structure as public contract
- update examples and app code to use `ui.CreateElement(...)`, `ui.Render(...)`, and `html.*` helpers as the normal composition path
- treat `internal/*` as unsupported implementation detail even when the repo contains useful test coverage around it

### Router

The router surface in `v3.x` is centered on `NewHashRouter`, `NewRouter`, `Register`, `Mount`, `UseNavigate`, `UseParams`, `UseQuery`, `UseSearchParams`, `UseRevalidator`, and `Outlet()`.

If older code still uses compatibility helpers such as `GoRegisterRoute` or `GoGetRoute`, treat those as migration shims rather than preferred APIs.

Migration guidance:

- register routes with canonical leading-slash paths
- use `Register(...)` instead of compatibility registration helpers in new code
- prefer route params, query helpers, redirects, guards, and metadata through documented `router.Options` flows rather than custom URL parsing
- for nested UIs, register parent layouts with `router.Options{Layout: true}` and render child routes with `router.Outlet()`

Review manually if your app depended on older path-normalization quirks, especially around trailing slashes, hash-prefixed inputs, or fallback-route assumptions.

### SSR And Hydration

`v3.x` establishes explicit public entrypoints for server rendering and browser resume:

- `ui.RenderToString(...)` for native-side HTML generation
- `ui.Hydrate(...)` for browser resume over existing markup
- `router.HydrateMount(...)` for router-aware hydration flows
- SSR bootstrap helpers for route data, atom snapshots, and ID seed reuse
- streaming SSR remains intentionally out of the shipped stable surface for now

Migration guidance:

- replace ad hoc render reuse with explicit `RenderToString` and `Hydrate` entrypoints
- move transferred initial state into the documented bootstrap helpers instead of custom inline globals when possible
- treat hydration mismatch recovery as subtree fallback behavior, not as a guarantee that every mismatch will be patched in place
- verify any SSR app that depends on route data reuse or seeded IDs after upgrading

### Forms And Local State

`v3.x` documents `ui.UseState`, `ui.UseReducer`, `ui.UsePrevious`, `ui.UseForm`, and related hooks as the public local-state surface.

Migration guidance:

- prefer `ui.UseForm` over hand-rolled touched, dirty, error, and submit bookkeeping when the form lifecycle is non-trivial
- use `ui.UseReducer` when the local state behaves like a state machine with named actions
- use `ui.UsePrevious` for render-time comparisons instead of storing duplicate state solely to remember the last committed value

### Shared State And Data Loading

`v3.x` makes `state.UseAtom`, `state.UseComputed`, `state.UseDerived`, snapshot helpers, `fetch.UseFetch`, `fetch.UseResource`, and `fetch.Fetch` first-class public APIs.

Migration guidance:

- move shared state into `state` instead of rebuilding ad hoc global registries
- use `state.UseComputed` for component-local derived values and `state.UseDerived` for shared read-only derivation
- use `fetch.UseResource[T]` as the preferred typed async resource API
- keep `fetch.UseFetch` for lower-level request control
- review snapshot persistence if you store non-JSON-compatible atom values in browser storage

### Testing And Deployment

The current repo validates the framework through native Go tests, js or wasm tests, and Playwright browser suites.

When upgrading older applications:

- add at least one js or wasm smoke test if your app uses browser-only APIs
- add a hydration check for SSR apps
- add a router smoke test if the app depends on guards, nested routes, params, or route loaders
- validate generated wasm and the correct `wasm_exec.js` flow in your deployment pipeline
- if you adopt prerender or hashed static assets, centralize route enumeration and asset-manifest generation instead of scattering custom file layout rules across app code

## Future Release Template

Each future major release should add a new section here using this structure:

1. Runtime and component authoring
2. Router
3. SSR and hydration
4. Forms and local state
5. Shared state and fetch
6. Testing and deployment
7. Removed APIs and deadline summary

Minor releases should add notes here too when a public behavior change is allowed under the experimental policy and requires user action.

## Migration Release Gate

Before a future major release is treated as ready for broad adoption, confirm that this file includes:

- a new release section named for the outgoing and incoming major versions
- explicit upgrade guidance for runtime and component authoring changes
- router migration notes, including changed defaults or matching behavior
- SSR and hydration notes, including bootstrap or mismatch-handling changes
- forms, state, and fetch notes where user code or payload shape changes
- testing and deployment notes for wasm output, browser tests, and server expectations
- browser-support changes, capability-baseline shifts, or dropped browser families when the release changes them
- runtime-configuration or public-flag transfer changes when the release changes those boundaries
- removed or deprecated API deadlines that match the changelog and API policy
