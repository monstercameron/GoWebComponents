# Actionable Errors And Diagnostics

This page audits the current framework error and warning surface with one goal: identify which failures already carry enough context to fix quickly, and which ones still need structured remediation, stable identifiers, or better developer guidance.

Use it when improving runtime diagnostics, reviewing developer-experience regressions, or deciding which failures deserve first-class error codes and troubleshooting anchors.

## Audit Outcome

The highest-friction failures today cluster into five buckets:

1. hook and context misuse that still reaches developers as raw panics
2. route registration and route-component misuse where the failure is detected, but remediation is still thin
3. hydration mismatch and fallback warnings that already contain good runtime context, but need stable identifiers and one canonical remediation page
4. async loader, form, and boundary failures that are observable, but inconsistent in how strongly they distinguish app errors from framework misuse
5. browser interop failures that already have structured codes, but are not yet tied back into the main troubleshooting flow

That means the repo does not have one uniformly poor diagnostics story. It has a mixed one:

- hydration and interop are the strongest current surfaces
- raw hook and component-construction panics are the weakest
- router and async-data failures sit in the middle: useful enough to debug with effort, but not yet concise enough to be called fully actionable

## Current Surface Map

### 1. Raw Panics And Assertions

Examples in the current codebase include:

- `GoUseState called outside component context`
- `GoUseEffect called outside component context`
- `GoUseContextValue called with nil context descriptor`
- `ui.UseContext called with nil context`
- `ui.CreateElement requires a component function or ui.Node`
- `router: component cannot be nil`
- `router: unsupported component type`

These failures are valuable because they fail fast, but they are still the highest-friction surface when they do not also provide:

- a stable diagnostic identifier
- a direct remediation step
- a pointer to the relevant doc or workflow

### 2. Runtime Diagnostics With Context

The runtime already has a stronger diagnostic path through `internal/runtime.ReportDiagnosticWithContext(...)`.

That path can carry:

- severity
- classification
- component stack
- inspected fiber path
- structured log entries derived from the diagnostic

Hydration mismatch reporting is the best current example because it already records path and component ancestry for warnings such as:

- hydration text mismatch
- hydration attribute mismatch
- hydration discarded unexpected DOM nodes
- hydration fell back to client rendering

### 3. Structured Interop Errors

The `interop` package is already ahead of most other framework surfaces for stable error identification.

It exposes structured `interop.Error` values with stable `ErrorCode` values such as:

- `invalid`
- `decode`
- `not_function`
- `disposed`
- `timeout`
- `remote_error`

That means interop already proves the repo can support durable diagnostic identifiers without depending on brittle full-message matching.

## Highest-Friction Findings

## A. Hook Misuse Outside Render Is Still Too Vague

Current surface:

- raw panic from hook entrypoints in `internal/runtime/hooks.go`
- matching runtime diagnostic message with no stable identifier

Why it is high friction:

- the failure often appears after a refactor moved hook usage into a helper or route factory
- the current message names the immediate problem, but not the common mistake pattern
- it does not tell the user what a valid replacement shape looks like

Required improvement direction:

- stable hook-misuse identifier
- explicit remediation text: hooks belong inside component render functions reached through `ui.CreateElement(...)`
- link to one troubleshooting anchor instead of forcing repo search

## B. Nil Context And Invalid Component Construction Fail Fast, But Not Helpfully Enough

Current surface:

- `ui.UseContext called with nil context`
- `ui.CreateElement requires a component function or ui.Node`
- `ui.CreateElement components may accept at most one props argument`
- `ui.CreateElement components must return ui.Node`

Why it is high friction:

- these are almost always high-confidence framework misuse cases
- the framework is correct to stop immediately
- the message should therefore be direct and prescriptive, not merely descriptive

Required improvement direction:

- keep the fast failure behavior in development
- add a stable identifier and one-sentence next step
- point to the exact docs page that explains the valid shape

## C. Route Component Misuse Needs Better Guidance

Current surface:

- `route component cannot be nil`
- `unsupported route component type`
- `route component must return exactly one element`
- duplicate-route and redirect-loop warnings already emitted through runtime diagnostics

Why it is high friction:

- route registration mistakes often happen far away from where the runtime finally panics
- the current messages do not tell developers whether the router expects a component function, a static node, or a route factory result
- redirect-loop warnings are useful, but they still need stable identifiers to support docs, CI logs, and issue reports

Required improvement direction:

- make route misuse messages say what to register instead
- assign stable identifiers to duplicate-route and redirect-loop diagnostics
- connect route diagnostics to [TROUBLESHOOTING.md](TROUBLESHOOTING.md#route-misconfiguration)

## D. Hydration Diagnostics Are Strong, But They Need Stable Codes And One Canonical Remediation Page

Current surface:

- hydration warnings already include component path and stack context
- strict hydration can escalate warnings into correctness errors
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md#hydration-mismatch-warnings) already documents the main debugging workflow

Why it is still high friction:

- teams cannot yet refer to a stable identifier such as "the text mismatch diagnostic" in CI or support notes
- the messages are good, but they are still free-form strings
- developers still need to jump between hydration docs, workflows, and troubleshooting pages manually

Required improvement direction:

- assign stable identifiers to text mismatch, attribute mismatch, discarded-node, and fallback diagnostics
- expose whether the runtime recovered or whether output should be treated as untrusted
- point warnings directly to the troubleshooting guidance

## E. Interop Already Has Codes, But It Is Not Integrated Into The Main Diagnostics Story

Current surface:

- structured `interop.Error` values with stable `ErrorCode`
- operation and target metadata in the rendered error string

Why it is still incomplete:

- these failures are not yet tied into the main runtime troubleshooting narrative
- developers have to know that interop is the exception rather than the rule

Required improvement direction:

- treat interop as the model for stable diagnostic identifiers elsewhere
- link common interop codes back into this page and [TROUBLESHOOTING.md](TROUBLESHOOTING.md#interop-mistakes)

## Priority Order

Use this order for diagnostics work that materially improves developer experience:

1. invalid hook and context usage
2. route component registration and route loop diagnostics
3. hydration mismatch and fallback identifiers plus remediation links
4. development-time assertions for component-construction misuse
5. integration of interop codes into the shared troubleshooting model

This ordering is intentional. Hook misuse and invalid construction errors fail on the shortest path and currently produce the least guided fixes, so they should improve before already-contextual hydration warnings.

## Stable Diagnostic Anchors

The anchors below are reserved for stable framework diagnostics and remediation links.

### GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT

Use for hook entrypoints called outside component render.

Expected remediation:

- move the hook call into a component function rendered through `ui.CreateElement(...)`
- do not call framework hooks in package init code, route registration helpers, or other ordinary helpers that are not rendering components

### GWC-RUNTIME-HOOK-FUNC-TYPE

Use for `GoUseFunc(...)` calls that receive a non-function value.

Expected remediation:

- pass a real function into `GoUseFunc(...)`
- keep raw data, options, and config objects outside the event-hook wrapper

### GWC-RUNTIME-DOM-ADAPTER-NIL

Use when runtime event-hook setup runs before a DOM adapter exists.

Expected remediation:

- initialize the runtime with a DOM adapter before rendering interactive components
- avoid calling event-hook setup against a partially constructed runtime

### GWC-RUNTIME-ATOM-REGISTRY-NIL

Use when `GoUseAtom(...)` runs against a runtime that was never fully initialized.

Expected remediation:

- create or initialize the runtime before calling `GoUseAtom(...)`
- ensure the shared atom registry exists before rendering components with atom subscriptions

### GWC-RUNTIME-ATOM-ACCESSOR-MISMATCH

Use when one atom hook slot is reused with an incompatible cached accessor shape.

Expected remediation:

- keep `GoUseAtom(...)` call order stable across renders
- do not reuse one hook position for different atom value types

### GWC-UI-CONTEXT-NIL

Use for `ui.UseContext(...)` or runtime context access with a nil context descriptor.

Expected remediation:

- pass the result of `ui.CreateContext(...)`
- avoid nil placeholder contexts

### GWC-UI-CREATE-ELEMENT-TYPE

Use for invalid `ui.CreateElement(...)` inputs.

Expected remediation:

- pass either a component function or a `ui.Node`
- keep component signatures to zero or one props argument and return `ui.Node`

### GWC-UI-UNSUPPORTED-ON-SERVER

Use for browser-only UI APIs called from the native SSR slice.

Expected remediation:

- switch to the SSR-safe alternative for the current API
- call browser-only UI APIs only from `js/wasm` builds after the client runtime is active

### GWC-EXAMPLE-SERVER-REQUEST

Use for example-server request failures such as SSR render, bootstrap serialization, or request parsing errors.

Expected remediation:

- inspect the `where:` and `path:` lines for the failing request handler first
- fix the server render, bootstrap, CSRF, or request-decoding path named in `next:` before retrying the request

### GWC-EXAMPLE-SERVER-STARTUP

Use for example-server startup failures such as working-directory discovery, repo-root resolution, or `ListenAndServe` bind errors.

Expected remediation:

- inspect the startup path named in `where:` first
- fix the port, repo-root, working-directory, or asset-discovery problem before restarting the example server

### GWC-TOOL-LIVERELOAD

Use for live-reload tool failures such as watcher attachment errors, websocket delivery failures, client-script injection errors, or failed rebuilds.

Expected remediation:

- inspect the tool stage named in `where:` first
- fix the watcher, websocket, manifest, client-script, or rebuild failure named in `next:` before trusting hot reload again

### GWC-RUNTIME-CONTAINER-NOT-FOUND

Use when `ui.Render(...)` or `ui.Hydrate(...)` targets a selector that does not exist.

Expected remediation:

- verify the page mounted the expected root container before rendering
- ensure the selector passed to render or hydrate matches the actual document

### GWC-RUNTIME-PANIC-RENDER

Use for uncaught panics thrown while rendering a component outside any recovering error boundary.

Expected remediation:

- inspect the named component render path first
- replace panic-based control flow with guarded branches, fallback UI, or an error boundary around the failing subtree

### GWC-RUNTIME-PANIC-EVENT

Use for uncaught panics thrown from event handlers outside any recovering error boundary.

Expected remediation:

- inspect the named handler first
- return explicit errors or guarded updates instead of panicking inside browser event callbacks

### GWC-RUNTIME-PANIC-EFFECT

Use for uncaught panics thrown from effects outside any recovering error boundary.

Expected remediation:

- inspect the effect body for failure-prone work
- move risky logic behind validation, explicit error handling, or a boundary-friendly fallback flow

### GWC-RUNTIME-PANIC-CLEANUP

Use for uncaught panics thrown from effect cleanup logic outside any recovering error boundary.

Expected remediation:

- inspect the cleanup function for teardown assumptions
- make cleanup idempotent so unmount or dependency changes do not panic mid-teardown

### GWC-RUNTIME-PANIC-LOADER

Use for uncaught panics thrown from route loaders or route-owned async data callbacks.

Expected remediation:

- inspect the route loader and its data dependencies first
- return explicit errors so route error UI can recover without a fatal exit

### GWC-RUNTIME-PANIC-HYDRATION

Use for strict hydration failures or hydration-time panics that must abort resume work.

Expected remediation:

- compare server markup with the first client render inputs
- fix the mismatch or hydration-time panic before relying on strict hydration

### GWC-RUNTIME-PANIC-STARTUP

Use for fatal startup failures before the app can mount safely.

Expected remediation:

- verify the target root selector exists before mounting
- validate bootstrap and initialization inputs before calling `ui.Render(...)` or `ui.Hydrate(...)`

### GWC-RUNTIME-PANIC-DEFERRED

Use for uncaught panics thrown from deferred runtime work such as transitions or scheduled callbacks.

Expected remediation:

- inspect the delayed callback or transition work first
- replace panic-based control flow with explicit errors or guarded branches

### GWC-RUNTIME-PANIC-SSR

Use for uncaught panics thrown while server-rendering HTML or integrating SSR output.

Expected remediation:

- inspect the server render path and bootstrap generation code first
- surface the returned error to the request handler instead of treating the panic as a transport-level failure

## Fatal Panic Log Contract

Wrapped fatal panic output now follows this line order:

- original panic payload as the first line
- stable framework code and panic phase
- `where:` with the first app-owned frame when available
- `path:` with the resolved component ancestry or route/startup target
- `error:` with the compact summary
- `runtime:` describing whether work stopped and why the runtime is no longer trustworthy
- `next:` with remediation guidance
- `docs:` with the stable anchor
- grouped `stack:` sections for `app:`, `framework:`, and `platform:` frames when available

Use this contract when refining future panic output so browser console logs, runtime diagnostics, and in-app debugging surfaces stay aligned.

## Recovery Versus Rethrow

The runtime currently allows an `ErrorBoundary` to recover only framework-owned panics that happen while executing user component work in these phases:

- render
- event handlers
- effects
- cleanup

These phases first try the nearest error boundary. If a boundary handles the failure, the runtime degrades into fallback UI or scheduled recovery instead of terminating immediately.

The following phases are still treated as fatal even after logging:

- route loaders and route-owned async data callbacks
- strict hydration escalation and hydration-time fatal mismatches
- startup and mount-target failures
- deferred or scheduled runtime work outside boundary-owned component execution
- SSR render failures

Those paths rethrow after logging because the runtime cannot currently prove that state, DOM reuse, route data, or server response generation remain trustworthy after the panic.

If future work adds safe route-level or server-level recovery semantics, update both this section and the central phase policy in `internal/runtime/panic_report.go` together.

### GWC-ROUTER-COMPONENT-NIL

Use for nil route component registration.

Expected remediation:

- register a concrete component function or static node
- avoid leaving placeholder route entries in the router table

### GWC-ROUTER-COMPONENT-TYPE

Use for unsupported route component types.

Expected remediation:

- register a component function, `ui.Node`, or route-compatible element producer
- avoid passing unrelated values such as raw data or config structs as route components

### GWC-ROUTER-COMPONENT-ARITY

Use when a route component returns the wrong shape.

Expected remediation:

- make the route component return exactly one element tree
- wrap siblings in `ui.Fragment(...)` when needed

### GWC-ROUTER-DUPLICATE-ROUTE

Use for route registration replacement warnings.

Expected remediation:

- remove duplicate registrations
- confirm whether the replacement is intentional before relying on last-write-wins behavior

### GWC-ROUTER-REDIRECT-LOOP

Use for redirect loops in route options or navigation guards.

Expected remediation:

- compare redirect source and target normalization
- ensure auth and default-route guards do not bounce between the same two paths

### GWC-ROUTER-LOADER-FAILED

Use for route-loader failures surfaced through runtime logs.

Expected remediation:

- inspect loader inputs, network failures, and error-boundary coverage
- supply route-local loading and error UI when the route depends on remote data

### GWC-HYDRATION-TEXT-MISMATCH

Use for server-versus-client text differences during hydration.

Expected remediation:

- make first-render text deterministic across server and client
- verify route data reuse, IDs, and time-dependent strings

### GWC-HYDRATION-ATTRIBUTE-MISMATCH

Use for mismatched comparable attributes during hydration.

Expected remediation:

- compare server-rendered attributes with first client render inputs
- check metadata, params, query-derived branches, and SSR bootstrap reuse

### GWC-HYDRATION-DISCARDED-NODES

Use when hydration removes unexpected DOM under a matched subtree.

Expected remediation:

- ensure the server document does not inject extra nodes into the hydrated subtree
- keep client-only placeholders outside the hydrated tree or render them consistently on the server

### GWC-HYDRATION-FALLBACK

Use when hydration abandons reuse and rerenders a subtree on the client.

Expected remediation:

- fix the underlying mismatch rather than treating fallback as harmless
- in strict hydration flows, treat the output as correctness-threatening until the mismatch is resolved

### GWC-INTEROP-INVALID

Use for invalid interop setup such as nil callbacks, empty names, or missing required arguments.

Expected remediation:

- validate browser-facing options before wiring subscriptions, workers, modules, or channels

### GWC-INTEROP-NOT-FUNCTION

Use for interop calls against non-callable values.

Expected remediation:

- verify the target export or property exists and is callable before invoking it

## Related Docs

- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [WORKFLOWS.md](WORKFLOWS.md)
- [ERROR_BOUNDARIES.md](ERROR_BOUNDARIES.md)
- [HYDRATION.md](HYDRATION.md)
- [router/README.md](../router/README.md)
- [FORMS.md](FORMS.md)
- [API_POLICY.md](API_POLICY.md)