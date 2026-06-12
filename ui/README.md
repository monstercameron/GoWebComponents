# GWC | Ui Library

# GoWebComponents (GWC)

## High-Level Overview

The `ui` library is the primary GWC UI runtime surface, including node construction, rendering orchestration, effects, and hydration helpers.

## Start Here

Use `ui` when you need:

- component composition and rendering
- hooks and local state
- hydration or SSR bootstrap helpers
- form helpers, async boundaries, overlays, and worker-task helpers

Recommended reading order:

1. [../README.md](../README.md) for the repo-level picture
2. [../docs/REFERENCE_MANUAL/04-ui-rendering-and-hooks.md](../docs/REFERENCE_MANUAL/04-ui-rendering-and-hooks.md) for the guided `ui` learning path
3. This README for the package surface
4. [../examples/README.md](../examples/README.md) for runnable API examples

If you are debugging implementation internals rather than using the public surface, jump to [../internal/runtime/README.md](../internal/runtime/README.md) instead of reading the API list below as architecture documentation.

## CSP Nonces

Use `RenderBootstrapScriptWithOptions` or `RenderBootstrapReferenceScriptWithOptions` with `SSRScriptOptions.Nonce` when serving under strict CSP. The nonce is HTML-escaped and applied to generated bootstrap script tags; streamed SSR boundary scripts accept the matching nonce through `SSRStreamOptions.ScriptNonce`.

## Public APIs

### `github.com/monstercameron/GoWebComponents/ui` (`package ui`)
- Functions: `AccessibleOverlay`, `ActiveDescendant`, `ActiveID`, `ActiveIndex`, `AddStateUpdatePayload`, `AnalyzeSSRBootstrapSize`, `Announce`, `ApplySSRStateUpdate`, `ApplyServerActionResult`, `ApplyServerErrors`, `Assertive`, `AssertiveID`, `AsyncBoundary`, `Cancel`, `Clear`, `Closed`, `Component`, `CorrelationIDFromContext`, `CreateContext`, `CreateElement`, `Default`, `Dirty`, `DirtyAny`, `Dispatch`, `Error`, `Errors`, `ExtractFiles`, `FieldMessage`, `FieldStatus`, `FileFromJSValue`, `FocusByID`, `FocusFirst`, `FocusFirstError`, `FocusSelector`, `FormError`, `FormErrors`, `FormField`, `FormMessage`, `Fragment`, `Get`, `GetKey`, `GetKeyCode`, `GetValue`, `HasErrors`, `HasFieldError`, `HasRedirect`, `HasRefresh`, `Header`, `HotReloadBoundary`, `Hydrate`, `HydrateInto`, `If`, `InspectBootstrapPayloads`, `IntentPending`, `IsActive`, `IsChecked`, `IsValid`, `JSValue`, `LastModified`, `Lazy`, `MarshalSSRBootstrap`, `MarshalSSRBootstrapBinary`, `MarshalSSRBootstrapBinaryObserved`, `MarshalSSRBootstrapObserved`, `MarshalSSRStateUpdateBinary`, `MarshalSSRStateUpdateText`, `Match`, `MoveEnd`, `MoveHome`, `MoveNext`, `MovePrevious`, `Name`, `NewCSRFToken`, `NewSSRBootstrapBudget`, `ObserveSSR`, `Ok`, `OnKeyDown`, `Overlay`, `Pending`, `Polite`, `PoliteID`, `Portal`, `PreventDefault`, `ReactiveRegion`, `ReadBootstrapPayload`, `ReadBootstrapReference`, `ReadBootstrapReferenceScript`, `ReadBootstrapScript`, `ReadCacheBootstrapSeed`, `ReadFormBootstrapDefaults`, `ReadRouteBootstrapData`, `ReadSessionBootstrapHint`, `RedirectLocation`, `Region`, `RegisterBootstrapPayload`, `RegisterCacheBootstrapSeed`, `RegisterFormBootstrapDefaults`, `RegisterRouteBootstrapData`, `RegisterSessionBootstrapHint`, `Reload`, `RememberActive`, `Render`, `RenderBootstrapReferenceScript`, `RenderBootstrapScript`, `RenderBootstrapScriptObserved`, `RenderInto`, `RenderToString`, `RenderToStringObserved`, `Reset`, `Restore`, `SSRCorrelationMiddleware`, `SSRObservabilityOptionsFromContext`, `SSRObservationAttributes`, `Set`, `SetActive`, `SetBootstrapCorrelationID`, `SetErrors`, `SetField`, `SetFormError`, `SetSubmitIntent`, `Size`, `Start`, `StartTransition`, `StopPropagation`, `Submit`, `SubmitError`, `SubmitIntent`, `SubmitWithIntent`, `Submitted`, `Submitting`, `TabIndex`, `Text`, `Touch`, `Touched`, `TouchedAny`, `TraceContextFromContext`, `Traceparent`, `Type`, `UnmarshalSSRBootstrap`, `UnmarshalSSRBootstrapBinary`, `UnmarshalSSRBootstrapReference`, `UnmarshalSSRStateUpdateBinary`, `UnmarshalSSRStateUpdateText`, `UnsupportedOnServer`, `Update`, `UseAnnouncer`, `UseCallback`, `UseChannel`, `UseCompositeNavigation`, `UseContext`, `UseDebounced`, `UseDeferredValue`, `UseEffect`, `UseEvent`, `UseFocusManager`, `UseFocusTrap`, `UseForm`, `UseId`, `UseLazyNode`, `UseMemo`, `UseOverlayStack`, `UsePrevious`, `UseReducer`, `UseRef`, `UseState`, `UseTask`, `UseThrottled`, `UseTransition`, `UseWorkerTask`, `Validate`, `ValidateAsync`, `ValidateIntent`, `Validated`, `Validating`, `Value`, `When`, `WithCorrelationID`, `WithW3CTraceContext`, `WrapHandler`
- Types: `AccessibleOverlayProps`, `AnnouncementMode`, `Announcer`, `AsyncBoundaryProps`, `CSRFToken`, `ChangeEvent`, `Channel`, `CompositeItem`, `CompositeNavigation`, `CompositeNavigationOptions`, `Context`, `ContextProvider`, `ContextProviderProps`, `Debounced`, `Element`, `ErrorBoundaryProps`, `Event`, `FieldErrors`, `FieldStatus`, `File`, `FocusEvent`, `FocusManager`, `FocusOptions`, `FocusTrapOptions`, `Form`, `FormEvent`, `Handler`, `HotReloadBoundaryProps`, `HydrationOptions`, `InputEvent`, `KeyboardEvent`, `LazyNode`, `LazyNodeState`, `LazyProps`, `MatchBuilder`, `MouseEvent`, `Node`, `NodeFactory`, `OverlayKind`, `OverlayProps`, `OverlayStack`, `OverlayStackOptions`, `PortalProps`, `PortalTarget`, `Previous`, `ReactiveSource`, `Reducer`, `Ref`, `SSRBootstrap`, `SSRBootstrapBudget`, `SSRBootstrapMetrics`, `SSRBootstrapReference`, `SSRBootstrapSizeReport`, `SSRHydrationMetrics`, `SSRI18nBootstrap`, `SSRI18nMessage`, `SSRObservabilityOptions`, `SSRObservation`, `SSRPayloadEncoding`, `SSRPayloadEnvelope`, `SSRPayloadFilter`, `SSRPayloadKind`, `SSRPayloadMetadata`, `SSRPayloadOptions`, `SSRPayloadReusePolicy`, `SSRPayloadScope`, `SSRPayloadValue`, `SSRRenderMetrics`, `SSRRouteBootstrap`, `SSRStateUpdate`, `ServerActionFlash`, `ServerActionOutcome`, `ServerActionRedirect`, `ServerActionRefresh`, `ServerActionResult`, `ServerFormErrors`, `State`, `Task`, `TaskState`, `Throttled`, `Transition`, `W3CTraceContext`, `WorkerTask`, `WorkerTaskState`
- Variables: `ErrorBoundary`
- Constants: `AnnouncementAssertive`, `AnnouncementPolite`, `CurrentSSRBootstrapVersion`, `CurrentSSRStateUpdateVersion`, `DefaultBootstrapReferenceScriptID`, `DefaultBootstrapScriptID`, `DefaultCSRFFormFieldName`, `DefaultCSRFHeaderName`, `DefaultRouteBootstrapPayloadKey`, `OverlayKindCustom`, `OverlayKindDialog`, `OverlayKindMenu`, `OverlayKindPopover`, `OverlayKindSheet`, `OverlayKindTooltip`, `SSRBootstrapFormatCBOR`, `SSRBootstrapFormatJSON`, `SSRPayloadEncodingBinary`, `SSRPayloadEncodingCBOR`, `SSRPayloadEncodingJSON`, `SSRPayloadEncodingText`, `SSRPayloadEncodingTimeRFC3339`, `SSRPayloadEncodingTimeUnixNano`, `SSRPayloadKindCacheSeed`, `SSRPayloadKindData`, `SSRPayloadKindFormDefaults`, `SSRPayloadKindRouteData`, `SSRPayloadKindSessionHint`, `SSRPayloadReuseClientOwned`, `SSRPayloadReuseRevalidateAfterResume`, `SSRPayloadReuseTrustOnFirstResume`, `SSRPayloadScopeApp`, `SSRPayloadScopeRoute`, `SSRPayloadScopeSubtree`, `ServerActionOutcomeAuthError`, `ServerActionOutcomeRedirect`, `ServerActionOutcomeRetryableError`, `ServerActionOutcomeSuccess`, `ServerActionOutcomeValidationError`

## Subfiles And Purpose

- `accessibility_native.go` - Native (non-WASM) implementation for accessibility
- `accessibility_shared.go` - Core implementation for accessibility_shared
- `accessibility_wasm.go` - WebAssembly-specific implementation for accessibility
- `bootstrap_wasm.go` - WebAssembly-specific implementation for bootstrap
- `branching.go` - Core implementation for branching
- `component_handle_shared.go` - Core implementation for component_handle_shared
- `component_handle_test.go` - Tests for component_handle behavior
- `context_shared.go` - Core implementation for context_shared
- `correlation_native.go` - Native (non-WASM) implementation for correlation
- `correlation_native_test.go` - Tests for correlation_native behavior
- `doc.go` - Package-level Go documentation
- `error_boundary_shared.go` - Core implementation for error_boundary_shared
- `file_native.go` - Native (non-WASM) implementation for file
- `file_wasm.go` - WebAssembly-specific implementation for file
- `form.go` - Core implementation for form
- `form_native.go` - Native (non-WASM) implementation for form
- `form_native_test.go` - Tests for form_native behavior
- `hot_reload_boundary.go` - Core implementation for hot_reload_boundary
- `hot_reload_boundary_shared.go` - Core implementation for hot_reload_boundary_shared
- `hot_reload_boundary_test.go` - Tests for hot_reload_boundary behavior
- `hydration.go` - Core implementation for hydration
- `overlay.go` - Core implementation for overlay
- `overlay_manager_test.go` - Tests for overlay_manager behavior
- `overlay_native.go` - Native (non-WASM) implementation for overlay
- `overlay_runtime_wasm.go` - WebAssembly-specific implementation for overlay_runtime
- `overlay_subscribe_wasm.go` - WebAssembly-specific implementation for overlay_subscribe
- `overlay_wasm.go` - WebAssembly-specific implementation for overlay
- `panic_logging.go` - Core implementation for panic_logging
- `panic_logging_native.go` - Native (non-WASM) implementation for panic_logging
- `shared_helpers_test.go` - Tests for shared_helpers behavior
- `ssr_bootstrap.go` - Core implementation for ssr_bootstrap
- `ssr_bootstrap_test.go` - Tests for ssr_bootstrap behavior
- `ssr_observability.go` - Core implementation for ssr_observability
- `ssr_transfer.go` - Core implementation for ssr_transfer
- `ssr_transfer_additional_test.go` - Tests for ssr_transfer_additional behavior
- `ssr_transfer_test.go` - Tests for ssr_transfer behavior
- `ui.go` - Core implementation for ui
- `ui_native.go` - Native (non-WASM) implementation for ui
- `ui_native_additional_test.go` - Tests for ui_native_additional behavior
- `ui_native_benchmark_test.go` - Tests for ui_native_benchmark behavior
- `ui_native_test.go` - Tests for ui_native behavior
- `ui_wasm_test.go` - Tests for ui_wasm behavior
- `worker_wasm.go` - WebAssembly-specific implementation for worker

## File Map

```text
ui/
|-- accessibility_native.go
|-- accessibility_shared.go
|-- accessibility_wasm.go
|-- bootstrap_wasm.go
|-- branching.go
|-- component_handle_shared.go
|-- component_handle_test.go
|-- context_shared.go
|-- correlation_native.go
|-- correlation_native_test.go
|-- doc.go
|-- error_boundary_shared.go
|-- file_native.go
|-- file_wasm.go
|-- form.go
|-- form_native.go
|-- form_native_test.go
|-- hot_reload_boundary.go
|-- hot_reload_boundary_shared.go
|-- hot_reload_boundary_test.go
|-- hydration.go
|-- overlay.go
|-- overlay_manager_test.go
|-- overlay_native.go
|-- overlay_runtime_wasm.go
|-- overlay_subscribe_wasm.go
|-- overlay_wasm.go
|-- panic_logging.go
|-- panic_logging_native.go
|-- shared_helpers_test.go
|-- ssr_bootstrap.go
|-- ssr_bootstrap_test.go
|-- ssr_observability.go
|-- ssr_transfer.go
|-- ssr_transfer_additional_test.go
|-- ssr_transfer_test.go
|-- ui.go
|-- ui_native.go
|-- ui_native_additional_test.go
|-- ui_native_benchmark_test.go
|-- ui_native_test.go
|-- ui_wasm_test.go
\-- worker_wasm.go
```



