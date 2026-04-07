# GWC | Devtools Library

# GoWebComponents (GWC)

## High-Level Overview

The `devtools` library provides debugging and inspection surfaces for GWC applications, including runtime snapshots and developer-focused overlays.

Devtools now resolves contributions from three layers:
- app-owned sections and actions registered directly through the `SetExtensionSections` and `SetErrorOverlayActions` helpers
- compatibility host contributions registered through `ApplyHostExtensions`
- kernel-owned contributions resolved live from the internal framework plugin kernel

`ApplyHostExtensions` remains a compatibility seam for the public companion host. It no longer overwrites app-owned devtools state.

## Public APIs

### `github.com/monstercameron/GoWebComponents/devtools` (`package devtools`)
- Functions: `ApplyHostExtensions`, `CaptureBugBundle`, `CaptureSupportDiagnosticBundle`, `CaptureTrace`, `ClearTraceReplay`, `CompareSnapshots`, `CurrentTraceReplay`, `ErrorOverlay`, `ExportBugCaptureBundleJSON`, `ExportSnapshotJSON`, `ExportSupportDiagnosticBundleJSON`, `ExportTraceCaptureJSON`, `ImportBugCaptureBundleJSON`, `ImportSupportDiagnosticBundleJSON`, `ImportTraceCaptureJSON`, `InspectBootstrapBoundaries`, `InspectCoordination`, `InspectErrorOverlayActions`, `InspectExtensionSections`, `InspectMultiClient`, `InspectSerializationBoundaries`, `Panel`, `ReplayBugCaptureBundle`, `ResetCoordinationInspection`, `ResetErrorOverlayActions`, `ResetExtensionSections`, `ResetMultiClientInspection`, `ResetSerializationBoundaryInspection`, `SanitizeBugCaptureBundleForSupport`, `SetCoordinationInspection`, `SetErrorOverlayActions`, `SetExtensionSections`, `SetMultiClientInspection`, `SetSerializationBoundaryInspection`, `SetTraceReplay`, `SnapshotNow`, `UseSnapshot`
- Types: `Boundary`, `BoundaryInspection`, `Branch`, `BugCaptureBundle`, `CacheEntry`, `Classification`, `ComponentRenderTrace`, `Coordination`, `Diagnostic`, `ErrorOverlayAction`, `ErrorOverlayActionContext`, `ErrorOverlayIssue`, `ErrorOverlayProps`, `ExtensionSection`, `FlamegraphFrame`, `Hook`, `HydrationDebug`, `Log`, `LogLevel`, `MultiClient`, `MultiClientFailure`, `MultiClientPeer`, `MultiClientTraffic`, `Node`, `PanelProps`, `Profiling`, `ProfilingEvent`, `ProfilingPhaseTotals`, `ReplayEntry`, `Route`, `RouteLoader`, `RouteMetadata`, `RouteRedirect`, `RouteStack`, `Severity`, `Snapshot`, `SnapshotComparison`, `StartupProfiling`, `Stats`, `SupportDiagnosticBundle`, `SyncEvent`, `TraceCapture`, `WorkerJob`
- Variables: _none_
- Constants: `SeverityError`, `SeverityInfo`, `SeverityWarning`

## Subfiles And Purpose

- `bug_capture.go` - Core implementation for bug_capture
- `coordination_state.go` - Core implementation for coordination_state
- `devtools_stub.go` - Core implementation for devtools_stub
- `devtools_test.go` - Tests for devtools behavior
- `devtools_wasm.go` - WebAssembly-specific implementation for devtools
- `devtools_wasm_test.go` - Tests for devtools_wasm behavior
- `doc.go` - Package-level Go documentation
- `error_overlay_actions.go` - Core implementation for error_overlay_actions
- `extension_sections.go` - Core implementation for extension_sections
- `host_extensions.go` - Core implementation for host_extensions
- `multi_client_state.go` - Core implementation for multi_client_state
- `README.md` - Folder-level documentation
- `serialization_boundaries.go` - Core implementation for serialization_boundaries
- `snapshot_export.go` - Core implementation for snapshot_export
- `support_bundle.go` - Core implementation for support_bundle
- `trace_capture.go` - Core implementation for trace_capture
- `types.go` - Type definitions

## File Map

```text
devtools/
|-- bug_capture.go
|-- coordination_state.go
|-- devtools_stub.go
|-- devtools_test.go
|-- devtools_wasm.go
|-- devtools_wasm_test.go
|-- doc.go
|-- error_overlay_actions.go
|-- extension_sections.go
|-- host_extensions.go
|-- multi_client_state.go
|-- README.md
|-- serialization_boundaries.go
|-- snapshot_export.go
|-- support_bundle.go
|-- trace_capture.go
\-- types.go
```



