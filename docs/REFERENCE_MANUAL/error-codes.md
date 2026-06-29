# Diagnostic Error Code Reference

> Generated from `internal/runtime/diagnostic_metadata.go` by `docs/errorcodes`.
> Do not edit by hand; run `ERRORCODES_WRITE=1 go test ./docs/errorcodes/` to regenerate.

Every code below is emitted in a structured runtime diagnostic. Look the code up here for its cause anchor and the `next:` remediation.

## GWC-HYDRATION-ATTRIBUTE-MISMATCH

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-hydration-attribute-mismatch`
- **next:** Compare server-rendered attributes with the first client render inputs and verify metadata, params, query state, and SSR bootstrap reuse.

## GWC-HYDRATION-DISCARDED-NODES

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-hydration-discarded-nodes`
- **next:** Ensure the server document does not inject extra nodes into the hydrated subtree and keep client-only placeholders outside the hydrated tree.

## GWC-HYDRATION-FALLBACK

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-hydration-fallback`
- **next:** Fix the underlying server-client mismatch instead of relying on fallback rendering. In strict hydration flows, treat the output as correctness-threatening until the mismatch is resolved.

## GWC-HYDRATION-TEXT-MISMATCH

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-hydration-text-mismatch`
- **next:** Make the first client render deterministic with the server HTML. Recheck IDs, params, query-derived branches, time-dependent strings, and bootstrap data reuse.

## GWC-ROUTER-COMPONENT-ARITY

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-router-component-arity`
- **next:** Return exactly one element tree from the route component. Wrap siblings in ui.Fragment(...) when needed.

## GWC-ROUTER-COMPONENT-NIL

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-router-component-nil`
- **next:** Register a concrete component function or static node for the route instead of leaving a nil placeholder.

## GWC-ROUTER-COMPONENT-TYPE

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-router-component-type`
- **next:** Register a component function, ui.Node, or route-compatible element producer instead of a raw config or data value.

## GWC-ROUTER-DUPLICATE-ROUTE

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-router-duplicate-route`
- **next:** Remove duplicate route registrations or confirm that last-write-wins replacement is intentional.

## GWC-ROUTER-LOADER-FAILED

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-router-loader-failed`
- **next:** Inspect the loader inputs, network failure, and route-local error UI. Supply an Options.Error fallback when the route depends on remote data.

## GWC-ROUTER-REDIRECT-LOOP

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-router-redirect-loop`
- **next:** Compare redirect source and target normalization and ensure guards or default routes do not bounce between the same paths.

## GWC-RUNTIME-ATOM-ACCESSOR-MISMATCH

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-atom-accessor-mismatch`
- **next:** Do not reuse one hook slot for different atom value types. Keep GoUseAtom call order stable and preserve a single value shape per atom hook position.

## GWC-RUNTIME-ATOM-REGISTRY-NIL

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-atom-registry-nil`
- **next:** Create or initialize the runtime before calling GoUseAtom so the shared atom registry exists for subscriptions and updates.

## GWC-RUNTIME-CONTAINER-NOT-FOUND

- **Docs:** `TROUBLESHOOTING.md#broken-example-serving`
- **next:** Verify the target selector exists before calling ui.Render(...) or ui.Hydrate(...), and confirm the page mounted the expected root container.

## GWC-RUNTIME-DOM-ADAPTER-NIL

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-dom-adapter-nil`
- **next:** Initialize the runtime with a DOM adapter before rendering components that call GoUseFunc or wire event handlers.

## GWC-RUNTIME-HOOK-FUNC-TYPE

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-hook-func-type`
- **next:** Pass a real function to GoUseFunc and move any non-callable config or data values outside the event-hook wrapper.

## GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-hook-outside-component`
- **next:** Call framework hooks only while rendering a component through ui.CreateElement(...). Move the hook call out of package init code, route factories, and ordinary helpers that are not rendering components.

## GWC-RUNTIME-HOOK-THREADING

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-hook-threading`
- **next:** Call hooks only on the component render goroutine. Move hook calls back into render, and move background work into UseEffect, UseTask, UseChannel, or event handlers that update existing state instead of creating hooks.

## GWC-RUNTIME-PANIC-ASYNC

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-async`
- **next:** inspect the async task named by where/path first; the panicking goroutine was contained and abandoned, so fix its body to return explicit errors instead of panicking.

## GWC-RUNTIME-PANIC-CLEANUP

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-cleanup`
- **next:** Inspect the cleanup function for the named component and make teardown idempotent so unmount or dependency changes do not panic mid-cleanup.

## GWC-RUNTIME-PANIC-DEFERRED

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-deferred`
- **next:** Inspect deferred callbacks, transition work, and scheduled runtime continuations so delayed work returns explicit errors instead of panicking.

## GWC-RUNTIME-PANIC-EFFECT

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-effect`
- **next:** Inspect the effect body for the named component and move failure-prone work behind validation, explicit error handling, or an error boundary-friendly fallback path.

## GWC-RUNTIME-PANIC-EVENT

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-event`
- **next:** Inspect the event handler named in the diagnostic, remove panic-based control flow, and return explicit errors or guarded updates instead of crashing the runtime callback.

## GWC-RUNTIME-PANIC-HYDRATION

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-hydration`
- **next:** Inspect the first client render inputs and server markup, then fix the mismatch or hydration-time panic before relying on strict resume paths.

## GWC-RUNTIME-PANIC-LOADER

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-loader`
- **next:** Inspect the route loader or async data callback named in the diagnostic, remove panic-based control flow, and return explicit errors so route error UI can recover without a fatal exit.

## GWC-RUNTIME-PANIC-RENDER

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-render`
- **next:** Inspect the component render path named in the diagnostic and replace panic-based control flow with guarded branches, fallback UI, or an error boundary around the failing subtree.

## GWC-RUNTIME-PANIC-SSR

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-ssr`
- **next:** Inspect the server render path and bootstrap generation code first, then surface the returned error instead of letting request-time rendering fail invisibly.

## GWC-RUNTIME-PANIC-STARTUP

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-runtime-panic-startup`
- **next:** Verify startup selectors, bootstrap inputs, and runtime initialization so mount-time failures do not terminate before the app can render.

## GWC-UI-CONTEXT-NIL

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-ui-context-nil`
- **next:** Pass the descriptor returned by ui.CreateContext(...) and avoid nil placeholder contexts.

## GWC-UI-CREATE-ELEMENT-TYPE

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-ui-create-element-type`
- **next:** Pass a component function or ui.Node to ui.CreateElement, keep component signatures to zero or one props argument, and return exactly one ui.Node tree.

## GWC-UI-UNSUPPORTED-ON-SERVER

- **Docs:** `ACTIONABLE_ERRORS.md#gwc-ui-unsupported-on-server`
- **next:** Use the SSR-safe alternative documented for this API, or call the browser-only API only from js/wasm builds after the client runtime is active.

