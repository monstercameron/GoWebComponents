package errcode

// This file is the single source of truth for framework diagnostic codes.
// Every Docs anchor is derived by buildRegistry from the code itself.

// Typed constants for every registered code.
const (
	FrameworkUsage              Code = "GWC-FRAMEWORK-USAGE"
	HydrationAttributeMismatch  Code = "GWC-HYDRATION-ATTRIBUTE-MISMATCH"
	HydrationDiscardedNodes     Code = "GWC-HYDRATION-DISCARDED-NODES"
	HydrationFallback           Code = "GWC-HYDRATION-FALLBACK"
	HydrationTextMismatch       Code = "GWC-HYDRATION-TEXT-MISMATCH"
	RouterComponentArity        Code = "GWC-ROUTER-COMPONENT-ARITY"
	RouterComponentNil          Code = "GWC-ROUTER-COMPONENT-NIL"
	RouterComponentType         Code = "GWC-ROUTER-COMPONENT-TYPE"
	RouterDuplicateRoute        Code = "GWC-ROUTER-DUPLICATE-ROUTE"
	RouterLoaderFailed          Code = "GWC-ROUTER-LOADER-FAILED"
	RouterRedirectLoop          Code = "GWC-ROUTER-REDIRECT-LOOP"
	RuntimeAtomAccessorMismatch Code = "GWC-RUNTIME-ATOM-ACCESSOR-MISMATCH"
	RuntimeAtomRegistryNil      Code = "GWC-RUNTIME-ATOM-REGISTRY-NIL"
	RuntimeContainerNotFound    Code = "GWC-RUNTIME-CONTAINER-NOT-FOUND"
	RuntimeDOMAdapterNil        Code = "GWC-RUNTIME-DOM-ADAPTER-NIL"
	RuntimeHookFuncType         Code = "GWC-RUNTIME-HOOK-FUNC-TYPE"
	RuntimeHookOutsideComponent Code = "GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT"
	RuntimeHookThreading        Code = "GWC-RUNTIME-HOOK-THREADING"
	RuntimePanic                Code = "GWC-RUNTIME-PANIC"
	RuntimePanicAsync           Code = "GWC-RUNTIME-PANIC-ASYNC"
	RuntimePanicCleanup         Code = "GWC-RUNTIME-PANIC-CLEANUP"
	RuntimePanicDeferred        Code = "GWC-RUNTIME-PANIC-DEFERRED"
	RuntimePanicEffect          Code = "GWC-RUNTIME-PANIC-EFFECT"
	RuntimePanicEvent           Code = "GWC-RUNTIME-PANIC-EVENT"
	RuntimePanicHydration       Code = "GWC-RUNTIME-PANIC-HYDRATION"
	RuntimePanicLoader          Code = "GWC-RUNTIME-PANIC-LOADER"
	RuntimePanicRender          Code = "GWC-RUNTIME-PANIC-RENDER"
	RuntimePanicSSR             Code = "GWC-RUNTIME-PANIC-SSR"
	RuntimePanicStartup         Code = "GWC-RUNTIME-PANIC-STARTUP"
	UIContextNil                Code = "GWC-UI-CONTEXT-NIL"
	UICreateElementType         Code = "GWC-UI-CREATE-ELEMENT-TYPE"
	UIUnsupportedOnServer       Code = "GWC-UI-UNSUPPORTED-ON-SERVER"
	HydrationIslandInvalid      Code = "GWC-HYDRATION-ISLAND-INVALID"
	RouterQueryParamInvalid     Code = "GWC-ROUTER-QUERY-PARAM-INVALID"
	RouterRouteParamsInvalid    Code = "GWC-ROUTER-ROUTE-PARAMS-INVALID"
	RouterRoutePatternInvalid   Code = "GWC-ROUTER-ROUTE-PATTERN-INVALID"
	RuntimeAsyncFallbackPanic   Code = "GWC-RUNTIME-ASYNC-FALLBACK-PANIC"
	RuntimeDerivedAtomInvalid   Code = "GWC-RUNTIME-DERIVED-ATOM-INVALID"
	RuntimeSSRGoroutineIdentity Code = "GWC-RUNTIME-SSR-GOROUTINE-IDENTITY"
	StateSnapshotInvalid        Code = "GWC-STATE-SNAPSHOT-INVALID"
	UIAwaitFailed               Code = "GWC-UI-AWAIT-FAILED"
	UIFormFieldInvalid          Code = "GWC-UI-FORM-FIELD-INVALID"
	UIMarkdownEngineMissing     Code = "GWC-UI-MARKDOWN-ENGINE-MISSING"
	UISSRPayloadInvalid         Code = "GWC-UI-SSR-PAYLOAD-INVALID"
	RuntimeHookOrder            Code = "GWC-RUNTIME-HOOK-ORDER"
	RuntimeHookDepUnstable      Code = "GWC-RUNTIME-HOOK-DEP-UNSTABLE"
	RuntimeHookInList           Code = "GWC-RUNTIME-HOOK-IN-LIST"
)

// Codes for interop, css, fetch, router, state, runtime, and ui diagnostics.
const (
	InteropUnavailable             Code = "GWC-INTEROP-UNAVAILABLE"
	InteropPromiseRejected         Code = "GWC-INTEROP-PROMISE-REJECTED"
	InteropDisposed                Code = "GWC-INTEROP-DISPOSED"
	InteropEncode                  Code = "GWC-INTEROP-ENCODE"
	InteropDecode                  Code = "GWC-INTEROP-DECODE"
	InteropMissingExport           Code = "GWC-INTEROP-MISSING-EXPORT"
	InteropNotFunction             Code = "GWC-INTEROP-NOT-FUNCTION"
	InteropInvalid                 Code = "GWC-INTEROP-INVALID"
	InteropPersistentStoreBlocked  Code = "GWC-INTEROP-PERSISTENT-STORE-BLOCKED"
	InteropPersistentStoreQuota    Code = "GWC-INTEROP-PERSISTENT-STORE-QUOTA"
	InteropMultiClientUnauthorized Code = "GWC-INTEROP-MULTI-CLIENT-UNAUTHORIZED"
	InteropCancelled               Code = "GWC-INTEROP-CANCELLED"
	InteropTimeout                 Code = "GWC-INTEROP-TIMEOUT"
	InteropRemote                  Code = "GWC-INTEROP-REMOTE-ERROR"
	HydrationFailed                Code = "GWC-HYDRATION-FAILED"
	CSSClassChurn                  Code = "GWC-CSS-CLASS-CHURN"
	CSSClassCollision              Code = "GWC-CSS-CLASS-COLLISION"
	FetchLoaderNil                 Code = "GWC-FETCH-LOADER-NIL"
	FetchMutationInvalid           Code = "GWC-FETCH-MUTATION-INVALID"
	FetchCircuitOpen               Code = "GWC-FETCH-CIRCUIT-OPEN"
	RouterLoaderKeyInvalid         Code = "GWC-ROUTER-LOADER-KEY-INVALID"
	RouterChunkLoadFailed          Code = "GWC-ROUTER-CHUNK-LOAD-FAILED"
	StateStorageAreaInvalid        Code = "GWC-STATE-STORAGE-AREA-INVALID"
	StateSnapshotMigrationFailed   Code = "GWC-STATE-SNAPSHOT-MIGRATION-FAILED"
	RuntimeSSRUnsupportedComponent Code = "GWC-RUNTIME-SSR-UNSUPPORTED-COMPONENT"
	RuntimeSSRStreamFailed         Code = "GWC-RUNTIME-SSR-STREAM-FAILED"
	UIBootstrapUnavailable         Code = "GWC-UI-BOOTSTRAP-UNAVAILABLE"
	UISSRBootstrapVersion          Code = "GWC-UI-SSR-BOOTSTRAP-VERSION"
)

// registryEntries lists every code without its derived Docs reference.
var registryEntries = []Entry{
	{
		Code:        FrameworkUsage,
		Family:      FamilyFramework,
		Headline:    "Invalid framework API usage",
		Remediation: "Inspect the where/path fields first, then correct the invalid framework API usage before retrying.",
		Severity:    SeverityError,
	},
	{
		Code:                   HydrationAttributeMismatch,
		Family:                 FamilyHydration,
		Headline:               "Hydration attribute mismatch",
		Remediation:            "Compare server-rendered attributes with the first client render inputs and verify metadata, params, query state, and SSR bootstrap reuse.",
		RecoverableUnlessError: true,
		Severity:               SeverityWarning,
	},
	{
		Code:                   HydrationDiscardedNodes,
		Family:                 FamilyHydration,
		Headline:               "Hydration discarded unexpected DOM nodes",
		Remediation:            "Ensure the server document does not inject extra nodes into the hydrated subtree and keep client-only placeholders outside the hydrated tree.",
		RecoverableUnlessError: true,
		Severity:               SeverityWarning,
	},
	{
		Code:                   HydrationFallback,
		Family:                 FamilyHydration,
		Headline:               "Hydration fell back to client rendering",
		Remediation:            "Fix the underlying server-client mismatch instead of relying on fallback rendering. In strict hydration flows, treat the output as correctness-threatening until the mismatch is resolved.",
		RecoverableUnlessError: true,
		Severity:               SeverityWarning,
	},
	{
		Code:                   HydrationTextMismatch,
		Family:                 FamilyHydration,
		Headline:               "Hydration text mismatch",
		Remediation:            "Make the first client render deterministic with the server HTML. Recheck IDs, params, query-derived branches, time-dependent strings, and bootstrap data reuse.",
		RecoverableUnlessError: true,
		Severity:               SeverityWarning,
	},
	{
		Code:        RouterComponentArity,
		Family:      FamilyRouter,
		Headline:    "Route component returned the wrong element count",
		Remediation: "Return exactly one element tree from the route component. Wrap siblings in ui.Fragment(...) when needed.",
		Severity:    SeverityError,
	},
	{
		Code:        RouterComponentNil,
		Family:      FamilyRouter,
		Headline:    "Route component is nil",
		Remediation: "Register a concrete component function or static node for the route instead of leaving a nil placeholder.",
		Severity:    SeverityError,
	},
	{
		Code:        RouterComponentType,
		Family:      FamilyRouter,
		Headline:    "Unsupported route component type",
		Remediation: "Register a component function, ui.Node, or route-compatible element producer instead of a raw config or data value.",
		Severity:    SeverityError,
	},
	{
		Code:        RouterDuplicateRoute,
		Family:      FamilyRouter,
		Headline:    "Route registered twice",
		Remediation: "Remove duplicate route registrations or confirm that last-write-wins replacement is intentional.",
		Recoverable: true,
		Severity:    SeverityWarning,
	},
	{
		Code:        RouterLoaderFailed,
		Family:      FamilyRouter,
		Headline:    "Route loader failed",
		Remediation: "Inspect the loader inputs, network failure, and route-local error UI. Supply an Options.Error fallback when the route depends on remote data.",
		Severity:    SeverityError,
	},
	{
		Code:                   RouterRedirectLoop,
		Family:                 FamilyRouter,
		Headline:               "Route redirect loop",
		Remediation:            "Compare redirect source and target normalization and ensure guards or default routes do not bounce between the same paths.",
		RecoverableUnlessError: true,
		Severity:               SeverityWarning,
	},
	{
		Code:        RuntimeAtomAccessorMismatch,
		Family:      FamilyHook,
		Headline:    "Atom hook slot changed value type",
		Remediation: "Do not reuse one hook slot for different atom value types. Keep GoUseAtom call order stable and preserve a single value shape per atom hook position.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeAtomRegistryNil,
		Family:      FamilyHook,
		Headline:    "Atom registry is not initialized",
		Remediation: "Create or initialize the runtime before calling GoUseAtom so the shared atom registry exists for subscriptions and updates.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeContainerNotFound,
		Family:      FamilyRuntime,
		Headline:    "Render target container was not found",
		Remediation: "Verify the target selector exists before calling ui.Render(...) or ui.Hydrate(...), and confirm the page mounted the expected root container.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeDOMAdapterNil,
		Family:      FamilyHook,
		Headline:    "GoUseFunc has no DOM adapter",
		Remediation: "Initialize the runtime with a DOM adapter before rendering components that call GoUseFunc or wire event handlers.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeHookFuncType,
		Family:      FamilyHook,
		Headline:    "GoUseFunc received a non-function",
		Remediation: "Pass a real function to GoUseFunc and move any non-callable config or data values outside the event-hook wrapper.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeHookOutsideComponent,
		Family:      FamilyHook,
		Headline:    "Hook called outside a component render",
		Remediation: "Call framework hooks only while rendering a component through ui.CreateElement(...). Move the hook call out of package init code, route factories, and ordinary helpers that are not rendering components.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeHookThreading,
		Family:      FamilyHook,
		Headline:    "Hook called off the render goroutine",
		Remediation: "Call hooks only on the component render goroutine. Move hook calls back into render, and move background work into UseEffect, UseTask, UseChannel, or event handlers that update existing state instead of creating hooks.",
		Severity:    SeverityError,
	},
	{
		Code:             RuntimePanic,
		Family:           FamilyPanic,
		Headline:         "Panic in an unclassified runtime phase",
		Remediation:      "inspect the where/path fields first, then replace panic-based control flow with guarded branches, explicit errors, fallback UI, or a boundary when recovery is expected.",
		PhaseRemediation: "inspect the where/path fields first, then replace panic-based control flow with guarded branches, explicit errors, fallback UI, or a boundary when recovery is expected.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicAsync,
		Family:           FamilyPanic,
		Headline:         "Panic in a contained async task",
		Remediation:      "inspect the async task named by where/path first; the panicking goroutine was contained and abandoned, so fix its body to return explicit errors instead of panicking.",
		PhaseRemediation: "inspect the async task named by where/path first; the panicking goroutine was contained and abandoned, so fix its body to return explicit errors instead of panicking.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicCleanup,
		Family:           FamilyPanic,
		Headline:         "Panic in an effect cleanup",
		Remediation:      "Inspect the cleanup function for the named component and make teardown idempotent so unmount or dependency changes do not panic mid-cleanup.",
		PhaseRemediation: "inspect the cleanup path named by where/path first; make teardown idempotent and avoid panicking during unmount or dependency changes.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicDeferred,
		Family:           FamilyPanic,
		Headline:         "Panic in deferred runtime work",
		Remediation:      "Inspect deferred callbacks, transition work, and scheduled runtime continuations so delayed work returns explicit errors instead of panicking.",
		PhaseRemediation: "inspect the deferred callback or transition work named by where/path first; replace panic-based control flow with explicit errors or guarded branches.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicEffect,
		Family:           FamilyPanic,
		Headline:         "Panic in an effect body",
		Remediation:      "Inspect the effect body for the named component and move failure-prone work behind validation, explicit error handling, or an error boundary-friendly fallback path.",
		PhaseRemediation: "inspect the effect body named by where/path first; move failure-prone work behind validation, explicit errors, or a recoverable fallback path.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicEvent,
		Family:           FamilyPanic,
		Headline:         "Panic in an event handler",
		Remediation:      "Inspect the event handler named in the diagnostic, remove panic-based control flow, and return explicit errors or guarded updates instead of crashing the runtime callback.",
		PhaseRemediation: "inspect the event handler named by where/path first; return explicit errors or guarded updates instead of panicking from runtime callbacks.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicHydration,
		Family:           FamilyPanic,
		Headline:         "Panic during hydration",
		Remediation:      "Inspect the first client render inputs and server markup, then fix the mismatch or hydration-time panic before relying on strict resume paths.",
		PhaseRemediation: "compare server markup with the first client render at where/path first; fix the mismatch or hydration panic before relying on strict resume.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicLoader,
		Family:           FamilyPanic,
		Headline:         "Panic in a route loader or async data callback",
		Remediation:      "Inspect the route loader or async data callback named in the diagnostic, remove panic-based control flow, and return explicit errors so route error UI can recover without a fatal exit.",
		PhaseRemediation: "inspect the loader or async data path named by where/path first; return explicit errors so route error UI can recover instead of crashing route resolution.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicRender,
		Family:           FamilyPanic,
		Headline:         "Panic during component render",
		Remediation:      "Inspect the component render path named in the diagnostic and replace panic-based control flow with guarded branches, fallback UI, or an error boundary around the failing subtree.",
		PhaseRemediation: "inspect the component render path named by where/path first; replace panic-based control flow with guarded branches, fallback UI, or an ErrorBoundary when local recovery is expected.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicSSR,
		Family:           FamilyPanic,
		Headline:         "Panic during server rendering",
		Remediation:      "Inspect the server render path and bootstrap generation code first, then surface the returned error instead of letting request-time rendering fail invisibly.",
		PhaseRemediation: "inspect the server render path named by where/path first; surface the returned error to the request handler instead of treating it as a transport-level panic.",
		Severity:         SeverityError,
	},
	{
		Code:             RuntimePanicStartup,
		Family:           FamilyPanic,
		Headline:         "Panic during runtime startup",
		Remediation:      "Verify startup selectors, bootstrap inputs, and runtime initialization so mount-time failures do not terminate before the app can render.",
		PhaseRemediation: "verify the startup target and bootstrap inputs named by where/path first; ensure mount selectors and initialization state exist before rendering.",
		Severity:         SeverityError,
	},
	{
		Code:        UIContextNil,
		Family:      FamilyUI,
		Headline:    "Nil context descriptor",
		Remediation: "Pass the descriptor returned by ui.CreateContext(...) and avoid nil placeholder contexts.",
		Severity:    SeverityError,
	},
	{
		Code:        UICreateElementType,
		Family:      FamilyUI,
		Headline:    "Invalid ui.CreateElement component",
		Remediation: "Pass a component function or ui.Node to ui.CreateElement, keep component signatures to zero or one props argument, and return exactly one ui.Node tree.",
		Severity:    SeverityError,
	},
	{
		Code:        UIUnsupportedOnServer,
		Family:      FamilyUI,
		Headline:    "Browser-only API used on a server build",
		Remediation: "Use the SSR-safe alternative documented for this API, or call the browser-only API only from js/wasm builds after the client runtime is active.",
		Severity:    SeverityError,
	},
	{
		Code:        HydrationIslandInvalid,
		Family:      FamilyHydration,
		Headline:    "Invalid hydration island options",
		Remediation: "Give ui.HydrationIsland a non-empty ID or Selector and a Strategy of HydrateImmediately, HydrateOnVisible, HydrateOnInteraction, or HydrateOnIdle; call ui.NormalizeHydrationIslandOptions first to get the error without a panic.",
		Severity:    SeverityError,
	},
	{
		Code:        RouterQueryParamInvalid,
		Family:      FamilyRouter,
		Headline:    "Query parameter cannot be decoded",
		Remediation: "Fix the query value named in the message so it parses as the struct field type (bool, integer, number, or repeated string), or change the field type or its `query:\"name\"` tag in the struct passed to router.DecodeQuery.",
		Severity:    SeverityError,
	},
	{
		Code:        RouterRouteParamsInvalid,
		Family:      FamilyRouter,
		Headline:    "Route params do not satisfy the route contract",
		Remediation: "Pass a non-empty value for every :param named in the contract pattern and no extra params; use RouteContract.ParamNames() to list the required names, or call the non-Must variant (Path, Href) and handle the returned error.",
		Severity:    SeverityError,
	},
	{
		Code:        RouterRoutePatternInvalid,
		Family:      FamilyRouter,
		Headline:    "Invalid route contract pattern",
		Remediation: "Rewrite the pattern passed to router.DefineRoute or MustDefineRoute using only literal segments and unique :name params; catch-all (*), optional (?), wildcard, empty, and unnamed segments are not allowed in route contracts.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeAsyncFallbackPanic,
		Family:      FamilyRuntime,
		Headline:    "Async boundary fallback callback panicked",
		Remediation: "Fix the errorFallback function passed to the async boundary so it renders without panicking for any error value; the panic value that escaped is included after the headline.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeDerivedAtomInvalid,
		Family:      FamilyRuntime,
		Headline:    "Invalid derived atom registration",
		Remediation: "Pass a non-nil compute function and a dependency list that does not include the derived atom id itself when calling RegisterDerivedAtom; break dependency cycles by deriving from a base atom.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeSSRGoroutineIdentity,
		Family:      FamilyRuntime,
		Headline:    "Cannot identify the SSR rendering goroutine",
		Remediation: "Render server-side hooks only on the goroutine that started the SSR render; do not call hooks from goroutines spawned inside a component, and report this error if it appears on a plain render path.",
		Severity:    SeverityError,
	},
	{
		Code:        StateSnapshotInvalid,
		Family:      FamilyRuntime,
		Headline:    "Invalid state snapshot version",
		Remediation: "Use a snapshot version between 0 and the current version; register migrations with state.RegisterSnapshotMigration(fromVersion, fn) for every older version and clear stored snapshots written by a newer build.",
		Severity:    SeverityError,
	},
	{
		Code:        UIAwaitFailed,
		Family:      FamilyUI,
		Headline:    "ui.Await received a failed SuspenseValue",
		Remediation: "Handle SuspenseValue.Error before rendering (show an error state) or wrap the component in an error boundary; the original error is wrapped and available through errors.Unwrap.",
		Severity:    SeverityError,
	},
	{
		Code:        UIFormFieldInvalid,
		Family:      FamilyUI,
		Headline:    "Form field unknown or value type mismatched",
		Remediation: "Pass the exact exported struct field name (check spelling and case) and a value assignable to that field type to Form.MustSetField, or use SetField and check its bool result.",
		Severity:    SeverityError,
	},
	{
		Code:        UIMarkdownEngineMissing,
		Family:      FamilyUI,
		Headline:    "html.Markdown was called but no markdown engine is registered",
		Remediation: "Import the engine once in your app: import _ \"github.com/monstercameron/GoWebComponents/v6/html/markdown\" (or call markdown.Enable()). Until then the source renders as escaped plain text inside a div with data-gwc-markdown-unavailable.",
		Severity:    SeverityWarning,
	},
	{
		Code:        UISSRPayloadInvalid,
		Family:      FamilyUI,
		Headline:    "Invalid SSR payload or bootstrap usage",
		Remediation: "Pass a non-nil SSRBootstrap, a non-empty payload key, and a supported payload kind, scope, reuse policy, encoding, and state-update version (the message names the rejected value); decode into a destination type matching the encoded payload.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeHookOrder,
		Family:      FamilyHook,
		Headline:    "Hook call order changed between renders",
		Remediation: "Hooks must be called in the same order every render: do not call hooks in conditions or loops, and do not return early before a hook. Move the condition inside the hook (or into the hook's dependencies) instead of around the call.",
		Severity:    SeverityWarning,
	},
	{
		Code:        RuntimeHookDepUnstable,
		Family:      FamilyHook,
		Headline:    "Hook dependency changes identity every render",
		Remediation: "A func, map or slice dependency is compared by identity, and this one is a new value on every render, so the hook re-runs every time. Hoist the value out of the component, wrap it in UseMemo/UseCallback, keep it in UseState/UseRef, or depend on the primitive fields it is built from. If re-running every render is intended, pass ui.Always instead.",
		Severity:    SeverityWarning,
	},
	{
		Code:        RuntimeHookInList,
		Family:      FamilyHook,
		Headline:    "Handler hook allocated inside a list render",
		Remediation: "A handler hook (UseEvent/UseFunc) was allocated while a list was being mapped, so it takes a positional hook slot that shifts when items are added, removed or reordered. Render each row through MapKeyedComponent (one component per item, with its own hooks) or pass a plain func to the row instead of calling a hook per item.",
		Severity:    SeverityWarning,
	},
	{
		Code:        InteropUnavailable,
		Family:      FamilyInterop,
		Headline:    "Browser interop is unavailable in this build",
		Remediation: "This call needs a js/wasm build running in a browser. Guard the call with a build-tagged file pair (_wasm.go plus a _native.go stub), or skip the browser feature when interop.IsCode(err, interop.CodeUnavailable) on the host.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropPromiseRejected,
		Family:      FamilyInterop,
		Headline:    "A JavaScript promise rejected",
		Remediation: "Read the wrapped rejection reason in the error message, then fix the JavaScript function or the arguments passed to it; handle the rejection on the Go side with interop.IsCode(err, interop.CodePromiseRejected).",
		Severity:    SeverityError,
	},
	{
		Code:        InteropDisposed,
		Family:      FamilyInterop,
		Headline:    "Interop handle used after it was disposed",
		Remediation: "Do not call a released callback, subscription, or binding. Keep the handle alive for as long as the browser can invoke it, and create a new one after Release or Cancel.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropEncode,
		Family:      FamilyInterop,
		Headline:    "Go value could not be encoded for JavaScript",
		Remediation: "Pass JSON-encodable values (no channels, funcs, or cyclic structures) across the interop boundary, or convert the value to a plain struct, map, or slice first.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropDecode,
		Family:      FamilyInterop,
		Headline:    "JavaScript value could not be decoded into the Go target",
		Remediation: "Make the destination type match the JavaScript value shape (field names, number versus string), or decode into any and inspect the value before converting.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropMissingExport,
		Family:      FamilyInterop,
		Headline:    "JavaScript export or property is missing",
		Remediation: "Verify the host page defines the named global or property before the Go code reads it, and that the script defining it ran first.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropNotFunction,
		Family:      FamilyInterop,
		Headline:    "JavaScript target is not callable",
		Remediation: "Verify the target export or property exists and is callable before invoking it.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropInvalid,
		Family:      FamilyInterop,
		Headline:    "Invalid interop request",
		Remediation: "Validate required names, URLs, and callbacks before creating the interop binding.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropPersistentStoreBlocked,
		Family:      FamilyInterop,
		Headline:    "Persistent store upgrade is blocked by another connection",
		Remediation: "Close older tabs, workers, or windows that still hold the same IndexedDB database open, then retry the upgrade.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropPersistentStoreQuota,
		Family:      FamilyInterop,
		Headline:    "Persistent store quota exceeded",
		Remediation: "Purge stale durable data or reduce the payload size before retrying the write.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropMultiClientUnauthorized,
		Family:      FamilyInterop,
		Headline:    "Multi-client traffic was not authorized",
		Remediation: "Verify topic ownership, peer role, and origin policy before accepting or publishing privileged multi-client traffic.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropCancelled,
		Family:      FamilyInterop,
		Headline:    "Interop operation was cancelled",
		Remediation: "The context or caller cancelled the operation. Treat this as expected control flow, or extend the context deadline if it was cancelled too early.",
		Severity:    SeverityWarning,
	},
	{
		Code:        InteropTimeout,
		Family:      FamilyInterop,
		Headline:    "Interop operation timed out",
		Remediation: "Raise the timeout, or check that the browser API or remote peer responds at all; retry with backoff for transient failures.",
		Severity:    SeverityError,
	},
	{
		Code:        InteropRemote,
		Family:      FamilyInterop,
		Headline:    "Remote peer reported an error",
		Remediation: "Read the remote error text in the message; the failure originated in the other worker, window, or tab, not in this one.",
		Severity:    SeverityError,
	},
	{
		Code:        HydrationFailed,
		Family:      FamilyHydration,
		Headline:    "Hydration failed during client resume",
		Remediation: "Open the hydration snapshot in devtools for the recorded failure, compare the server HTML with the first client render, and fix the non-deterministic input (time, random, browser-only state) that diverged.",
		Severity:    SeverityError,
	},
	{
		Code:        CSSClassChurn,
		Family:      FamilyCSS,
		Headline:    "CSS class registry grew past the churn threshold",
		Remediation: "Route live values through css.DynamicLength or css.DynamicVar (a stable var(--name) class plus an inline value) instead of building a new class per value.",
		Severity:    SeverityWarning,
	},
	{
		Code:        CSSClassCollision,
		Family:      FamilyCSS,
		Headline:    "Two distinct rule-sets hashed to one CSS class name",
		Remediation: "This is an astronomically rare hash collision: vary one of the colliding rule-sets slightly (for example add a no-op declaration) to force a different class name.",
		Severity:    SeverityWarning,
	},
	{
		Code:        FetchLoaderNil,
		Family:      FamilyFetch,
		Headline:    "Fetch query loader is nil",
		Remediation: "Pass a non-nil loader function to the query or infinite query; the loader is what produces the cached value.",
		Severity:    SeverityError,
	},
	{
		Code:        FetchMutationInvalid,
		Family:      FamilyFetch,
		Headline:    "Invalid fetch mutation queue usage",
		Remediation: "Enqueue mutations with a non-empty URL, give the queue an executor before replaying, and resolve conflicts with a supported resolution action (the message names the rejected value).",
		Severity:    SeverityError,
	},
	{
		Code:        FetchCircuitOpen,
		Family:      FamilyFetch,
		Headline:    "Circuit breaker is open",
		Remediation: "The breaker tripped after repeated failures. Wait for the reset timeout, show cached or fallback data meanwhile, and fix the failing upstream; match with errors.Is(err, fetch.ErrCircuitOpen).",
		Severity:    SeverityWarning,
	},
	{
		Code:        RouterLoaderKeyInvalid,
		Family:      FamilyRouter,
		Headline:    "Router loader key is empty or not active",
		Remediation: "Pass the key of a loader registered on the currently matched route; keys are only active while their route is mounted.",
		Severity:    SeverityError,
	},
	{
		Code:        RouterChunkLoadFailed,
		Family:      FamilyRouter,
		Headline:    "Route chunk failed to load",
		Remediation: "Check the chunk script URL is served with a JavaScript content type, is reachable from the page origin, and matches the deployed build; retry navigation after fixing the asset.",
		Severity:    SeverityError,
	},
	{
		Code:        StateStorageAreaInvalid,
		Family:      FamilyState,
		Headline:    "Unknown persistent storage area",
		Remediation: "Use state.LocalStorage or state.SessionStorage as the storage area, and make sure the browser exposes it (it can be blocked in private or sandboxed contexts).",
		Severity:    SeverityError,
	},
	{
		Code:        StateSnapshotMigrationFailed,
		Family:      FamilyState,
		Headline:    "Snapshot migration is missing or failed",
		Remediation: "Register a migration for every version step from the stored version up to the current one, and make each migration tolerate older shapes; clear the stored snapshot if it cannot be migrated.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeSSRUnsupportedComponent,
		Family:      FamilyRuntime,
		Headline:    "Component or element cannot be rendered on the server",
		Remediation: "Server rendering supports element tags and components of the form func(props) *Element or func() *Element; change the component signature or render it client-side only.",
		Severity:    SeverityError,
	},
	{
		Code:        RuntimeSSRStreamFailed,
		Family:      FamilyRuntime,
		Headline:    "Streaming SSR failed",
		Remediation: "Pass a writer to the stream renderer, and make async boundaries suspend exactly once with a completion channel; the message names the boundary that failed.",
		Severity:    SeverityError,
	},
	{
		Code:        UIBootstrapUnavailable,
		Family:      FamilyUI,
		Headline:    "SSR bootstrap could not be read",
		Remediation: "Make sure the server emitted the bootstrap script element with the expected id (or a reachable reference URL in a supported format) and that it loads before the client reads it.",
		Severity:    SeverityError,
	},
	{
		Code:        UISSRBootstrapVersion,
		Family:      FamilyUI,
		Headline:    "Unsupported SSR bootstrap version",
		Remediation: "The server and client builds disagree on the bootstrap version; redeploy matching server and client builds.",
		Severity:    SeverityError,
	},
}
