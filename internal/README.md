# GWC | Internal Library

# GoWebComponents (GWC)

## High-Level Overview

The `internal` folder contains implementation packages used by core runtime systems such as scheduling, platform adapters, and diagnostics plumbing.

This folder contains internal-only GWC packages. Exported identifiers listed below are public within the module, but not part of the external import contract.

## Start Here

- `internal/platform/README.md`: browser and mock platform adapter layout
- `internal/runtime/README.md`: default single-threaded runtime file map
- `internal/runtime2/README.md`: experimental multithreaded runtime file map

## Public APIs

### `github.com/monstercameron/GoWebComponents/internal/diagnostics` (`package diagnostics`)
- Functions: `Build`, `Emit`, `Formatted`, `WriteHTTPError`
- Types: `Options`, `Report`
- Variables: _none_
- Constants: _none_

### `github.com/monstercameron/GoWebComponents/internal/platform/jsdom` (`package jsdom`)
- Functions: `AddClass`, `AddEventListener`, `AppendChild`, `BatchSetAttributes`, `BeginBatch`, `CancelIdleCallback`, `CreateElement`, `CreateEventHandler`, `CreateTextNode`, `DidTimeout`, `EndBatch`, `Equals`, `GetChildren`, `GetCurrentPath`, `GetElementById`, `GetElementsByClassName`, `GetElementsByTagName`, `GetFirstChild`, `GetHash`, `GetInnerHTML`, `GetItem`, `GetKey`, `GetKeyCode`, `GetNextSibling`, `GetParent`, `GetProperty`, `GetTarget`, `GetTextContent`, `GetValue`, `InsertBefore`, `IsChecked`, `IsNull`, `NewWASMBrowserState`, `NewWASMDOMAdapter`, `NewWASMDOMNode`, `NewWASMEventAdapter`, `NewWASMScheduler`, `OnPopState`, `PreventDefault`, `PushState`, `QuerySelector`, `QuerySelectorAll`, `Release`, `ReleaseEventHandler`, `Reload`, `RemoveAttribute`, `RemoveChild`, `RemoveClass`, `RemoveEventListener`, `RemoveItem`, `ReplaceChild`, `ReplaceState`, `RequestIdleCallback`, `ResolveNode`, `SetAttribute`, `SetHash`, `SetInnerHTML`, `SetItem`, `SetProperty`, `SetStyle`, `SetStyles`, `SetTextContent`, `SetTimeout`, `StopPropagation`, `TimeRemaining`, `ToggleClass`, `Value`, `WrapFunction`
- Types: `WASMBrowserState`, `WASMDOMAdapter`, `WASMDOMNode`, `WASMEventAdapter`, `WASMScheduler`
- Variables: _none_
- Constants: _none_

### `github.com/monstercameron/GoWebComponents/internal/platform/mockdom` (`package mockdom`)
- Functions: `AppendChild`, `AssertOperation`, `ClearOperations`, `CreateElement`, `CreateTextNode`, `DidTimeout`, `Equals`, `FlushAll`, `FlushIdleCallbacks`, `FlushTimeouts`, `GetChildren`, `GetFirstChild`, `GetNextSibling`, `GetNode`, `GetOperations`, `GetParent`, `GetPendingCount`, `GetPendingTimeoutCount`, `GetProperty`, `InsertBefore`, `IsNull`, `NewMockDOMAdapter`, `NewMockScheduler`, `RemoveAttribute`, `RemoveChild`, `ReplaceChild`, `RequestIdleCallback`, `SetAttribute`, `SetInnerHTML`, `SetProperty`, `SetStyle`, `SetStyles`, `SetTextContent`, `SetTimeout`, `TimeRemaining`, `WrapFunction`
- Types: `DOMOperation`, `MockDOMAdapter`, `MockDOMNode`, `MockDeadline`, `MockScheduler`
- Variables: _none_
- Constants: _none_

### `github.com/monstercameron/GoWebComponents/internal/runtime` (`package runtime`)
- Functions: `A`, `Abbr`, `ActionableFrameworkPanic`, `Address`, `Article`, `Aside`, `Audio`, `B`, `Bdi`, `Bdo`, `BeginStartupProfiling`, `Blockquote`, `Body`, `Br`, `Button`, `Canvas`, `Caption`, `CaptureHotReloadSnapshot`, `Circle`, `Cite`, `ClassIdProps`, `ClassProps`, `CleanupAtomSubscriptions`, `ClearDiagnostics`, `ClearLogs`, `ClearProfiling`, `Code`, `Col`, `Colgroup`, `CompatibleWith`, `ConfigureStrictDiagnostics`, `ConfigureUnhandledPanicLogging`, `CreateElement`, `CurrentFiberPath`, `CurrentStrictDiagnosticsOptions`, `CurrentUnhandledPanicLoggingOptions`, `Data`, `Datalist`, `Dd`, `Del`, `Details`, `Dfn`, `Dialog`, `DidTimeout`, `Div`, `DivWithComponents`, `Dl`, `Dt`, `Em`, `Embed`, `EmptyProps`, `EnqueueUI`, `Equals`, `Fieldset`, `Figcaption`, `Figure`, `FinalizeUnhandledPanicContext`, `Footer`, `Form`, `G`, `GetAtom`, `GetAtomCount`, `GetAtomValue`, `GetCurrentFiber`, `GetDiagnostics`, `GetGlobalRuntime`, `GetKey`, `GetKeyCode`, `GetLogs`, `GetSubscriberCount`, `GetTarget`, `GetUIQueueSize`, `GetValue`, `GoUseAtom`, `GoUseAtomGlobal`, `GoUseCallback`, `GoUseCallbackGlobal`, `GoUseContextValue`, `GoUseEffect`, `GoUseEffectGlobal`, `GoUseFetch`, `GoUseFetchGlobal`, `GoUseFunc`, `GoUseFuncGlobal`, `GoUseId`, `GoUseIdGlobal`, `GoUseMemo`, `GoUseMemoGlobal`, `GoUseMemoGlobalTyped`, `GoUseMemoTyped`, `GoUseRef`, `GoUseRefGlobal`, `GoUseState`, `GoUseStateGlobal`, `GoUseTransitionPendingGlobal`, `H1`, `H2`, `H3`, `H4`, `H5`, `H6`, `HasPendingHotReloadSnapshot`, `Head`, `Header`, `Hgroup`, `Hr`, `HrefProps`, `Html`, `Hydrate`, `HydrateInto`, `HydrateTo`, `I`, `IdProps`, `IdentityKey`, `Iframe`, `Img`, `InitAtom`, `InitGlobalRuntime`, `Input`, `InputTypeProps`, `Ins`, `Inspect`, `IsChecked`, `IsNull`, `JSValue`, `Kbd`, `Label`, `Legend`, `Li`, `Line`, `Link`, `Main`, `MainWithComponents`, `Mark`, `Menu`, `Meta`, `Meter`, `MoveSubscription`, `MoveSubscriptions`, `Nav`, `NewAtomRegistry`, `NewComponentType`, `NewContextDescriptor`, `NewContextProviderType`, `NewErrorBoundaryType`, `NewGoEvent`, `NewRuntime`, `NormalizeHotReloadValue`, `Object`, `Ol`, `Optgroup`, `Option`, `Output`, `P`, `Param`, `Path`, `Picture`, `PlaceholderProps`, `Polygon`, `Portal`, `Pre`, `PrepareForHotReload`, `PreventDefault`, `ProcessUIQueue`, `Progress`, `Q`, `RecordProfilingEvent`, `RecordStartupBootstrapRead`, `RecordStartupRouteContext`, `Rect`, `RefreshEffectsForFiber`, `RegisterDerivedAtom`, `Render`, `RenderInto`, `RenderTo`, `RenderToString`, `ReportDiagnostic`, `ReportDiagnosticWithContext`, `ReportLog`, `ReportLogWithFields`, `ReportProfilingEvent`, `ReportUnhandledPanicContext`, `ResetWASMArtifactMetadata`, `ResetWASMStackFrameMapper`, `RestoreAtomSnapshot`, `RestoreHotReloadSnapshot`, `RestoreHotReloadSnapshotWithPlan`, `RestoreSnapshot`, `Rp`, `Rt`, `Ruby`, `S`, `Samp`, `ScheduleGranularUpdateForFiber`, `ScheduleGranularUpdateForFiberWithOrigin`, `ScheduleSubscribedFiberUpdate`, `ScheduleSubscribedFiberUpdateWithOrigin`, `ScheduleTransition`, `ScheduleUpdate`, `ScheduleUpdateForFiber`, `ScheduleUpdateForFiberWithOrigin`, `Script`, `Section`, `SectionWithComponents`, `Select`, `SetAtom`, `SetAtomValue`, `SetCurrentFiber`, `SetIDSeed`, `SetImplementation`, `SetNextHydrationObserver`, `SetNextHydrationStrict`, `SetWASMArtifactMetadata`, `SetWASMStackFrameMapper`, `ShouldDeferStateUpdates`, `Slot`, `Small`, `Snapshot`, `SnapshotAtoms`, `Source`, `Span`, `SrcProps`, `StartTransition`, `StartTransitionGlobal`, `StopPropagation`, `Strong`, `Style`, `StyleProps`, `Sub`, `Subscribe`, `Summary`, `Sup`, `Svg`, `Table`, `Tbody`, `Td`, `Template`, `Text`, `Textarea`, `Tfoot`, `Th`, `Thead`, `Time`, `TimeRemaining`, `Title`, `Tr`, `Track`, `TypeProps`, `U`, `Ul`, `Unsubscribe`, `UnsubscribeFiberFromAll`, `UnsubscribeMany`, `ValueProps`, `Var`, `Video`, `Wbr`, `WithComponents`
- Types: `ActionablePanicOptions`, `AtomRegistry`, `Attrs`, `BrowserState`, `ComponentRenderTraceSnapshot`, `ComponentSignature`, `ComponentType`, `Config`, `ContextDescriptor`, `ContextProviderType`, `DOMAdapter`, `DOMNode`, `Deadline`, `Diagnostic`, `DiagnosticClassification`, `DiagnosticSeverity`, `Effect`, `Element`, `ErrorBoundaryType`, `Event`, `EventAdapter`, `EventHandler`, `FetchState`, `Fiber`, `FiberSnapshot`, `FlamegraphFrameSnapshot`, `GoEvent`, `HookSnapshot`, `Hooks`, `HotBranchSnapshot`, `HotReloadComponentSnapshot`, `HotReloadFetchSnapshot`, `HotReloadMemoSnapshot`, `HotReloadRestoreDecision`, `HotReloadRestorePlan`, `HotReloadSnapshot`, `HydrationDebugSnapshot`, `HydrationMetrics`, `InspectionSnapshot`, `InspectionStats`, `LogEntry`, `LogLevel`, `PanicLoggingOptions`, `PanicPhase`, `PanicReport`, `PortalElementType`, `ProfilingEvent`, `ProfilingPhaseTotalsSnapshot`, `ProfilingSnapshot`, `ReactiveRegionElementType`, `ReactiveTextElementType`, `RefValue`, `Runtime`, `Scheduler`, `StartupProfilingSnapshot`, `StrictDiagnosticsOptions`, `WASMArtifactMetadata`, `WASMStackFrame`, `WASMStackFrameMapper`
- Variables: `PortalNodeType`, `ReactiveRegionNodeType`, `ReactiveTextNodeType`
- Constants: `DiagnosticCorrectness`, `DiagnosticError`, `DiagnosticInfo`, `DiagnosticInformational`, `DiagnosticPerformance`, `DiagnosticRecovered`, `DiagnosticUnsupportedRecover`, `DiagnosticWarning`, `LogDebug`, `LogError`, `LogInfo`, `LogWarn`, `PanicPhaseCleanup`, `PanicPhaseDeferred`, `PanicPhaseEffect`, `PanicPhaseEvent`, `PanicPhaseHydration`, `PanicPhaseLoader`, `PanicPhaseRender`, `PanicPhaseSSR`, `PanicPhaseStartup`

## Subfiles And Purpose

- `diagnostics/` - Diagnostics models and helpers (3 files).
- `platform/` - Platform-specific abstractions (8 files).
- `runtime/` - Runtime execution internals (94 files).
- `README.md` - Folder-level documentation

## File Map

```text
internal/
|-- diagnostics/
|   |-- report.go
|   |-- report_helpers_test.go
|   \-- report_test.go
|-- platform/
|   |-- jsdom/
|   |   |-- adapters.go
|   |   |-- adapters_benchmark_test.go
|   |   |-- adapters_wasm_test.go
|   |   \-- jsdom.go
|   \-- mockdom/
|       |-- dom.go
|       |-- dom_test.go
|       |-- mockdom.go
|       \-- scheduler.go
|-- runtime/
|   |-- component_type.go
|   |-- component_type_test.go
|   |-- context.go
|   |-- context_test.go
|   |-- core_helpers_extra_test.go
|   |-- coverage_branches_test.go
|   |-- diagnostic_metadata.go
|   |-- dom_commit_test.go
|   |-- edge_cases_test.go
|   |-- error_boundary.go
|   |-- error_boundary_test.go
|   |-- event_wrap_native.go
|   |-- event_wrap_wasm.go
|   |-- events.go
|   |-- events_benchmark_test.go
|   |-- events_wasm_test.go
|   |-- fiber_structure_test.go
|   |-- fine_grained_benchmark_test.go
|   |-- fine_grained_reactivity_test.go
|   |-- hooks.go
|   |-- hooks_benchmark_test.go
|   |-- hooks_contract_test.go
|   |-- hooks_extra_test.go
|   |-- hooks_fetch.go
|   |-- hooks_fetch_benchmark_test.go
|   |-- hooks_fetch_stub.go
|   |-- hooks_fetch_stub_benchmark_test.go
|   |-- hooks_fetch_stub_test.go
|   |-- hooks_fetch_wasm_test.go
|   |-- hooks_func_validate_native.go
|   |-- hooks_func_validate_wasm.go
|   |-- hooks_test.go
|   |-- hot_reload.go
|   |-- hot_reload_helpers_extra_test.go
|   |-- hot_reload_test.go
|   |-- html.go
|   |-- html_benchmark_test.go
|   |-- html_contract_test.go
|   |-- html_exhaustive_test.go
|   |-- hydration.go
|   |-- hydration_benchmark_test.go
|   |-- hydration_metrics.go
|   |-- hydration_test.go
|   |-- inspect.go
|   |-- inspect_test.go
|   |-- interfaces.go
|   |-- interfaces_benchmark_test.go
|   |-- interfaces_contract_test.go
|   |-- layout_benchmark_test.go
|   |-- panic_artifact_metadata.go
|   |-- panic_diagnostic_test.go
|   |-- panic_matrix_test.go
|   |-- panic_report.go
|   |-- panic_report_console_stub.go
|   |-- panic_report_console_wasm.go
|   |-- panic_source_map.go
|   |-- portal_benchmark_test.go
|   |-- portal_test.go
|   |-- production_correctness_test.go
|   |-- profiling.go
|   |-- reconciler.go
|   |-- reconciler_benchmark_test.go
|   |-- reconciler_coverage_test.go
|   |-- reconciler_rerender_test.go
|   |-- reconciler_skip_test.go
|   |-- reconciler_test.go
|   |-- reconciliation_advanced_test.go
|   |-- reconciliation_edge_cases_test.go
|   |-- runtime.go
|   |-- runtime_benchmark_test.go
|   |-- runtime_contract_test.go
|   |-- runtime_scheduler_extra_test.go
|   |-- scheduler.go
|   |-- scheduler_benchmark_test.go
|   |-- scheduler_contract_test.go
|   |-- scheduler_test.go
|   |-- shim.go
|   |-- shim_benchmark_test.go
|   |-- shim_wasm_test.go
|   |-- signature.go
|   |-- ssr.go
|   |-- ssr_test.go
|   |-- state.go
|   |-- state_benchmark_test.go
|   |-- state_contract_test.go
|   |-- state_test.go
|   |-- strict_diagnostics.go
|   |-- transition.go
|   |-- transition_test.go
|   |-- types.go
|   |-- types_benchmark_test.go
|   |-- types_extra_test.go
|   |-- types_test.go
|   \-- update_regression_test.go
\-- README.md
```



