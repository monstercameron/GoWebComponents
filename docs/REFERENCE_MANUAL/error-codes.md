# Diagnostic Error Code Reference

> Generated from `internal/errcode` by `docs/errorcodes`.
> Do not edit by hand; run `ERRORCODES_WRITE=1 go test ./docs/errorcodes/` to regenerate.

Every code below is emitted in a structured runtime diagnostic. Look the code up here for its cause and the `next:` remediation.

## GWC-CSS-CLASS-CHURN

CSS class registry grew past the churn threshold

- **Family:** css
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-css-class-churn`
- **next:** Route live values through css.DynamicLength or css.DynamicVar (a stable var(--name) class plus an inline value) instead of building a new class per value.

## GWC-CSS-CLASS-COLLISION

Two distinct rule-sets hashed to one CSS class name

- **Family:** css
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-css-class-collision`
- **next:** This is an astronomically rare hash collision: vary one of the colliding rule-sets slightly (for example add a no-op declaration) to force a different class name.

## GWC-FETCH-CIRCUIT-OPEN

Circuit breaker is open

- **Family:** fetch
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-fetch-circuit-open`
- **next:** The breaker tripped after repeated failures. Wait for the reset timeout, show cached or fallback data meanwhile, and fix the failing upstream; match with errors.Is(err, fetch.ErrCircuitOpen).

## GWC-FETCH-LOADER-NIL

Fetch query loader is nil

- **Family:** fetch
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-fetch-loader-nil`
- **next:** Pass a non-nil loader function to the query or infinite query; the loader is what produces the cached value.

## GWC-FETCH-MUTATION-INVALID

Invalid fetch mutation queue usage

- **Family:** fetch
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-fetch-mutation-invalid`
- **next:** Enqueue mutations with a non-empty URL, give the queue an executor before replaying, and resolve conflicts with a supported resolution action (the message names the rejected value).

## GWC-FRAMEWORK-USAGE

Invalid framework API usage

- **Family:** framework
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-framework-usage`
- **next:** Inspect the where/path fields first, then correct the invalid framework API usage before retrying.

## GWC-HYDRATION-ATTRIBUTE-MISMATCH

Hydration attribute mismatch

- **Family:** hydration
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-hydration-attribute-mismatch`
- **next:** Compare server-rendered attributes with the first client render inputs and verify metadata, params, query state, and SSR bootstrap reuse.

## GWC-HYDRATION-DISCARDED-NODES

Hydration discarded unexpected DOM nodes

- **Family:** hydration
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-hydration-discarded-nodes`
- **next:** Ensure the server document does not inject extra nodes into the hydrated subtree and keep client-only placeholders outside the hydrated tree.

## GWC-HYDRATION-FAILED

Hydration failed during client resume

- **Family:** hydration
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-hydration-failed`
- **next:** Open the hydration snapshot in devtools for the recorded failure, compare the server HTML with the first client render, and fix the non-deterministic input (time, random, browser-only state) that diverged.

## GWC-HYDRATION-FALLBACK

Hydration fell back to client rendering

- **Family:** hydration
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-hydration-fallback`
- **next:** Fix the underlying server-client mismatch instead of relying on fallback rendering. In strict hydration flows, treat the output as correctness-threatening until the mismatch is resolved.

## GWC-HYDRATION-ISLAND-INVALID

Invalid hydration island options

- **Family:** hydration
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-hydration-island-invalid`
- **next:** Give ui.HydrationIsland a non-empty ID or Selector and a Strategy of HydrateImmediately, HydrateOnVisible, HydrateOnInteraction, or HydrateOnIdle; call ui.NormalizeHydrationIslandOptions first to get the error without a panic.

## GWC-HYDRATION-TEXT-MISMATCH

Hydration text mismatch

- **Family:** hydration
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-hydration-text-mismatch`
- **next:** Make the first client render deterministic with the server HTML. Recheck IDs, params, query-derived branches, time-dependent strings, and bootstrap data reuse.

## GWC-INTEROP-CANCELLED

Interop operation was cancelled

- **Family:** interop
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-cancelled`
- **next:** The context or caller cancelled the operation. Treat this as expected control flow, or extend the context deadline if it was cancelled too early.

## GWC-INTEROP-DECODE

JavaScript value could not be decoded into the Go target

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-decode`
- **next:** Make the destination type match the JavaScript value shape (field names, number versus string), or decode into any and inspect the value before converting.

## GWC-INTEROP-DISPOSED

Interop handle used after it was disposed

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-disposed`
- **next:** Do not call a released callback, subscription, or binding. Keep the handle alive for as long as the browser can invoke it, and create a new one after Release or Cancel.

## GWC-INTEROP-ENCODE

Go value could not be encoded for JavaScript

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-encode`
- **next:** Pass JSON-encodable values (no channels, funcs, or cyclic structures) across the interop boundary, or convert the value to a plain struct, map, or slice first.

## GWC-INTEROP-INVALID

Invalid interop request

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-invalid`
- **next:** Validate required names, URLs, and callbacks before creating the interop binding.

## GWC-INTEROP-MISSING-EXPORT

JavaScript export or property is missing

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-missing-export`
- **next:** Verify the host page defines the named global or property before the Go code reads it, and that the script defining it ran first.

## GWC-INTEROP-MULTI-CLIENT-UNAUTHORIZED

Multi-client traffic was not authorized

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-multi-client-unauthorized`
- **next:** Verify topic ownership, peer role, and origin policy before accepting or publishing privileged multi-client traffic.

## GWC-INTEROP-NOT-FUNCTION

JavaScript target is not callable

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-not-function`
- **next:** Verify the target export or property exists and is callable before invoking it.

## GWC-INTEROP-PERSISTENT-STORE-BLOCKED

Persistent store upgrade is blocked by another connection

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-persistent-store-blocked`
- **next:** Close older tabs, workers, or windows that still hold the same IndexedDB database open, then retry the upgrade.

## GWC-INTEROP-PERSISTENT-STORE-QUOTA

Persistent store quota exceeded

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-persistent-store-quota`
- **next:** Purge stale durable data or reduce the payload size before retrying the write.

## GWC-INTEROP-PROMISE-REJECTED

A JavaScript promise rejected

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-promise-rejected`
- **next:** Read the wrapped rejection reason in the error message, then fix the JavaScript function or the arguments passed to it; handle the rejection on the Go side with interop.IsCode(err, interop.CodePromiseRejected).

## GWC-INTEROP-REMOTE-ERROR

Remote peer reported an error

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-remote-error`
- **next:** Read the remote error text in the message; the failure originated in the other worker, window, or tab, not in this one.

## GWC-INTEROP-TIMEOUT

Interop operation timed out

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-timeout`
- **next:** Raise the timeout, or check that the browser API or remote peer responds at all; retry with backoff for transient failures.

## GWC-INTEROP-UNAVAILABLE

Browser interop is unavailable in this build

- **Family:** interop
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-interop-unavailable`
- **next:** This call needs a js/wasm build running in a browser. Guard the call with a build-tagged file pair (_wasm.go plus a _native.go stub), or skip the browser feature when interop.IsCode(err, interop.CodeUnavailable) on the host.

## GWC-ROUTER-CHUNK-LOAD-FAILED

Route chunk failed to load

- **Family:** router
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-chunk-load-failed`
- **next:** Check the chunk script URL is served with a JavaScript content type, is reachable from the page origin, and matches the deployed build; retry navigation after fixing the asset.

## GWC-ROUTER-COMPONENT-ARITY

Route component returned the wrong element count

- **Family:** router
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-component-arity`
- **next:** Return exactly one element tree from the route component. Wrap siblings in ui.Fragment(...) when needed.

## GWC-ROUTER-COMPONENT-NIL

Route component is nil

- **Family:** router
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-component-nil`
- **next:** Register a concrete component function or static node for the route instead of leaving a nil placeholder.

## GWC-ROUTER-COMPONENT-TYPE

Unsupported route component type

- **Family:** router
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-component-type`
- **next:** Register a component function, ui.Node, or route-compatible element producer instead of a raw config or data value.

## GWC-ROUTER-DUPLICATE-ROUTE

Route registered twice

- **Family:** router
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-duplicate-route`
- **next:** Remove duplicate route registrations or confirm that last-write-wins replacement is intentional.

## GWC-ROUTER-LOADER-FAILED

Route loader failed

- **Family:** router
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-loader-failed`
- **next:** Inspect the loader inputs, network failure, and route-local error UI. Supply an Options.Error fallback when the route depends on remote data.

## GWC-ROUTER-LOADER-KEY-INVALID

Router loader key is empty or not active

- **Family:** router
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-loader-key-invalid`
- **next:** Pass the key of a loader registered on the currently matched route; keys are only active while their route is mounted.

## GWC-ROUTER-QUERY-PARAM-INVALID

Query parameter cannot be decoded

- **Family:** router
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-query-param-invalid`
- **next:** Fix the query value named in the message so it parses as the struct field type (bool, integer, number, or repeated string), or change the field type or its `query:"name"` tag in the struct passed to router.DecodeQuery.

## GWC-ROUTER-REDIRECT-LOOP

Route redirect loop

- **Family:** router
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-redirect-loop`
- **next:** Compare redirect source and target normalization and ensure guards or default routes do not bounce between the same paths.

## GWC-ROUTER-ROUTE-PARAMS-INVALID

Route params do not satisfy the route contract

- **Family:** router
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-route-params-invalid`
- **next:** Pass a non-empty value for every :param named in the contract pattern and no extra params; use RouteContract.ParamNames() to list the required names, or call the non-Must variant (Path, Href) and handle the returned error.

## GWC-ROUTER-ROUTE-PATTERN-INVALID

Invalid route contract pattern

- **Family:** router
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-router-route-pattern-invalid`
- **next:** Rewrite the pattern passed to router.DefineRoute or MustDefineRoute using only literal segments and unique :name params; catch-all (*), optional (?), wildcard, empty, and unnamed segments are not allowed in route contracts.

## GWC-RUNTIME-ASYNC-FALLBACK-PANIC

Async boundary fallback callback panicked

- **Family:** runtime
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-async-fallback-panic`
- **next:** Fix the errorFallback function passed to the async boundary so it renders without panicking for any error value; the panic value that escaped is included after the headline.

## GWC-RUNTIME-ATOM-ACCESSOR-MISMATCH

Atom hook slot changed value type

- **Family:** hook
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-atom-accessor-mismatch`
- **next:** Do not reuse one hook slot for different atom value types. Keep GoUseAtom call order stable and preserve a single value shape per atom hook position.

## GWC-RUNTIME-ATOM-REGISTRY-NIL

Atom registry is not initialized

- **Family:** hook
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-atom-registry-nil`
- **next:** Create or initialize the runtime before calling GoUseAtom so the shared atom registry exists for subscriptions and updates.

## GWC-RUNTIME-CONTAINER-NOT-FOUND

Render target container was not found

- **Family:** runtime
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-container-not-found`
- **next:** Verify the target selector exists before calling ui.Render(...) or ui.Hydrate(...), and confirm the page mounted the expected root container.

## GWC-RUNTIME-DERIVED-ATOM-INVALID

Invalid derived atom registration

- **Family:** runtime
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-derived-atom-invalid`
- **next:** Pass a non-nil compute function and a dependency list that does not include the derived atom id itself when calling RegisterDerivedAtom; break dependency cycles by deriving from a base atom.

## GWC-RUNTIME-DOM-ADAPTER-NIL

GoUseFunc has no DOM adapter

- **Family:** hook
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-dom-adapter-nil`
- **next:** Initialize the runtime with a DOM adapter before rendering components that call GoUseFunc or wire event handlers.

## GWC-RUNTIME-HOOK-DEP-UNSTABLE

Hook dependency changes identity every render

- **Family:** hook
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-hook-dep-unstable`
- **next:** A func, map or slice dependency is compared by identity, and this one is a new value on every render, so the hook re-runs every time. Hoist the value out of the component, wrap it in UseMemo/UseCallback, keep it in UseState/UseRef, or depend on the primitive fields it is built from. If re-running every render is intended, pass ui.Always instead.

## GWC-RUNTIME-HOOK-FUNC-TYPE

GoUseFunc received a non-function

- **Family:** hook
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-hook-func-type`
- **next:** Pass a real function to GoUseFunc and move any non-callable config or data values outside the event-hook wrapper.

## GWC-RUNTIME-HOOK-IN-LIST

Handler hook allocated inside a list render

- **Family:** hook
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-hook-in-list`
- **next:** A handler hook (UseEvent/UseFunc) was allocated while a list was being mapped, so it takes a positional hook slot that shifts when items are added, removed or reordered. Render each row through MapKeyedComponent (one component per item, with its own hooks) or pass a plain func to the row instead of calling a hook per item.

## GWC-RUNTIME-HOOK-ORDER

Hook call order changed between renders

- **Family:** hook
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-hook-order`
- **next:** Hooks must be called in the same order every render: do not call hooks in conditions or loops, and do not return early before a hook. Move the condition inside the hook (or into the hook's dependencies) instead of around the call.

## GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT

Hook called outside a component render

- **Family:** hook
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-hook-outside-component`
- **next:** Call framework hooks only while rendering a component through ui.CreateElement(...). Move the hook call out of package init code, route factories, and ordinary helpers that are not rendering components.

## GWC-RUNTIME-HOOK-THREADING

Hook called off the render goroutine

- **Family:** hook
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-hook-threading`
- **next:** Call hooks only on the component render goroutine. Move hook calls back into render, and move background work into UseEffect, UseTask, UseChannel, or event handlers that update existing state instead of creating hooks.

## GWC-RUNTIME-PANIC

Panic in an unclassified runtime phase

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic`
- **next:** inspect the where/path fields first, then replace panic-based control flow with guarded branches, explicit errors, fallback UI, or a boundary when recovery is expected.

## GWC-RUNTIME-PANIC-ASYNC

Panic in a contained async task

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-async`
- **next:** inspect the async task named by where/path first; the panicking goroutine was contained and abandoned, so fix its body to return explicit errors instead of panicking.

## GWC-RUNTIME-PANIC-CLEANUP

Panic in an effect cleanup

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-cleanup`
- **next:** Inspect the cleanup function for the named component and make teardown idempotent so unmount or dependency changes do not panic mid-cleanup.

## GWC-RUNTIME-PANIC-DEFERRED

Panic in deferred runtime work

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-deferred`
- **next:** Inspect deferred callbacks, transition work, and scheduled runtime continuations so delayed work returns explicit errors instead of panicking.

## GWC-RUNTIME-PANIC-EFFECT

Panic in an effect body

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-effect`
- **next:** Inspect the effect body for the named component and move failure-prone work behind validation, explicit error handling, or an error boundary-friendly fallback path.

## GWC-RUNTIME-PANIC-EVENT

Panic in an event handler

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-event`
- **next:** Inspect the event handler named in the diagnostic, remove panic-based control flow, and return explicit errors or guarded updates instead of crashing the runtime callback.

## GWC-RUNTIME-PANIC-HYDRATION

Panic during hydration

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-hydration`
- **next:** Inspect the first client render inputs and server markup, then fix the mismatch or hydration-time panic before relying on strict resume paths.

## GWC-RUNTIME-PANIC-LOADER

Panic in a route loader or async data callback

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-loader`
- **next:** Inspect the route loader or async data callback named in the diagnostic, remove panic-based control flow, and return explicit errors so route error UI can recover without a fatal exit.

## GWC-RUNTIME-PANIC-RENDER

Panic during component render

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-render`
- **next:** Inspect the component render path named in the diagnostic and replace panic-based control flow with guarded branches, fallback UI, or an error boundary around the failing subtree.

## GWC-RUNTIME-PANIC-SSR

Panic during server rendering

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-ssr`
- **next:** Inspect the server render path and bootstrap generation code first, then surface the returned error instead of letting request-time rendering fail invisibly.

## GWC-RUNTIME-PANIC-STARTUP

Panic during runtime startup

- **Family:** panic
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-panic-startup`
- **next:** Verify startup selectors, bootstrap inputs, and runtime initialization so mount-time failures do not terminate before the app can render.

## GWC-RUNTIME-SSR-GOROUTINE-IDENTITY

Cannot identify the SSR rendering goroutine

- **Family:** runtime
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-ssr-goroutine-identity`
- **next:** Render server-side hooks only on the goroutine that started the SSR render; do not call hooks from goroutines spawned inside a component, and report this error if it appears on a plain render path.

## GWC-RUNTIME-SSR-STREAM-FAILED

Streaming SSR failed

- **Family:** runtime
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-ssr-stream-failed`
- **next:** Pass a writer to the stream renderer, and make async boundaries suspend exactly once with a completion channel; the message names the boundary that failed.

## GWC-RUNTIME-SSR-UNSUPPORTED-COMPONENT

Component or element cannot be rendered on the server

- **Family:** runtime
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-runtime-ssr-unsupported-component`
- **next:** Server rendering supports element tags and components of the form func(props) *Element or func() *Element; change the component signature or render it client-side only.

## GWC-STATE-SNAPSHOT-INVALID

Invalid state snapshot version

- **Family:** runtime
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-state-snapshot-invalid`
- **next:** Use a snapshot version between 0 and the current version; register migrations with state.RegisterSnapshotMigration(fromVersion, fn) for every older version and clear stored snapshots written by a newer build.

## GWC-STATE-SNAPSHOT-MIGRATION-FAILED

Snapshot migration is missing or failed

- **Family:** state
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-state-snapshot-migration-failed`
- **next:** Register a migration for every version step from the stored version up to the current one, and make each migration tolerate older shapes; clear the stored snapshot if it cannot be migrated.

## GWC-STATE-STORAGE-AREA-INVALID

Unknown persistent storage area

- **Family:** state
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-state-storage-area-invalid`
- **next:** Use state.LocalStorage or state.SessionStorage as the storage area, and make sure the browser exposes it (it can be blocked in private or sandboxed contexts).

## GWC-UI-AWAIT-FAILED

ui.Await received a failed SuspenseValue

- **Family:** ui
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-ui-await-failed`
- **next:** Handle SuspenseValue.Error before rendering (show an error state) or wrap the component in an error boundary; the original error is wrapped and available through errors.Unwrap.

## GWC-UI-BOOTSTRAP-UNAVAILABLE

SSR bootstrap could not be read

- **Family:** ui
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-ui-bootstrap-unavailable`
- **next:** Make sure the server emitted the bootstrap script element with the expected id (or a reachable reference URL in a supported format) and that it loads before the client reads it.

## GWC-UI-CONTEXT-NIL

Nil context descriptor

- **Family:** ui
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-ui-context-nil`
- **next:** Pass the descriptor returned by ui.CreateContext(...) and avoid nil placeholder contexts.

## GWC-UI-CREATE-ELEMENT-TYPE

Invalid ui.CreateElement component

- **Family:** ui
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-ui-create-element-type`
- **next:** Pass a component function or ui.Node to ui.CreateElement, keep component signatures to zero or one props argument, and return exactly one ui.Node tree.

## GWC-UI-FORM-FIELD-INVALID

Form field unknown or value type mismatched

- **Family:** ui
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-ui-form-field-invalid`
- **next:** Pass the exact exported struct field name (check spelling and case) and a value assignable to that field type to Form.MustSetField, or use SetField and check its bool result.

## GWC-UI-MARKDOWN-ENGINE-MISSING

html.Markdown was called but no markdown engine is registered

- **Family:** ui
- **Severity:** warning
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-ui-markdown-engine-missing`
- **next:** Import the engine once in your app: import _ "github.com/monstercameron/GoWebComponents/v6/html/markdown" (or call markdown.Enable()). Until then the source renders as escaped plain text inside a div with data-gwc-markdown-unavailable.

## GWC-UI-SSR-BOOTSTRAP-VERSION

Unsupported SSR bootstrap version

- **Family:** ui
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-ui-ssr-bootstrap-version`
- **next:** The server and client builds disagree on the bootstrap version; redeploy matching server and client builds.

## GWC-UI-SSR-PAYLOAD-INVALID

Invalid SSR payload or bootstrap usage

- **Family:** ui
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-ui-ssr-payload-invalid`
- **next:** Pass a non-nil SSRBootstrap, a non-empty payload key, and a supported payload kind, scope, reuse policy, encoding, and state-update version (the message names the rejected value); decode into a destination type matching the encoded payload.

## GWC-UI-UNSUPPORTED-ON-SERVER

Browser-only API used on a server build

- **Family:** ui
- **Severity:** error
- **Docs:** `docs/REFERENCE_MANUAL/error-codes.md#gwc-ui-unsupported-on-server`
- **next:** Use the SSR-safe alternative documented for this API, or call the browser-only API only from js/wasm builds after the client runtime is active.

