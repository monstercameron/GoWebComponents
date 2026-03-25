# GWC | Pwa Library

# GoWebComponents (GWC)

## High-Level Overview

The `pwa` library contains progressive web app helpers such as persistence, diagnostics, and client capability integrations.

## Public APIs

### `github.com/monstercameron/GoWebComponents/pwa` (`package pwa`)
- Functions: `Available`, `BackgroundSyncCapabilities`, `BuildCacheStoragePlan`, `BuildServiceWorkerAssetPlan`, `Cancel`, `Inspect`, `InspectDiagnostics`, `MarshalManifestJSON`, `MarshalManifestJSONIndented`, `MutationQueueDiagnosticsSource`, `Normalized`, `ObserveInstallability`, `OpenCacheStorageManager`, `ParseWasmReleaseManifestJSON`, `Prompt`, `RegisterServiceWorker`, `RegisterSync`, `ReloadOnControllerChange`, `Revision`, `SkipWaiting`, `Snapshot`, `State`, `Subscribe`, `SubscribeLifecycle`, `Sync`, `Unregister`, `Update`, `Valid`, `Validate`
- Types: `BackgroundSyncCapabilities`, `CacheStorageAssetKind`, `CacheStorageEntry`, `CacheStorageManager`, `CacheStoragePlan`, `CacheStoragePlanOptions`, `CacheStorageSnapshot`, `CacheStorageStrategy`, `DiagnosticsOptions`, `DiagnosticsSnapshot`, `InstallPromptResult`, `InstallabilityManager`, `InstallabilityOptions`, `InstallabilityState`, `InstallabilitySubscription`, `Manifest`, `ManifestDiagnostics`, `ManifestDisplay`, `ManifestImage`, `ManifestOrientation`, `ManifestShortcut`, `OfflineQueueDiagnostics`, `OfflineQueueEntry`, `RelatedApplication`, `ServiceWorkerAssetPlan`, `ServiceWorkerAssetPlanOptions`, `ServiceWorkerOptions`, `ServiceWorkerRegistration`, `ServiceWorkerSnapshot`, `ServiceWorkerState`, `ServiceWorkerSubscription`, `ServiceWorkerVersion`, `StoragePressureDiagnostics`, `WasmReleaseArtifact`, `WasmReleaseFlags`, `WasmReleaseManifest`
- Variables: _none_
- Constants: `CacheStorageAssetKindAsset`, `CacheStorageAssetKindMedia`, `CacheStorageAssetKindScript`, `CacheStorageAssetKindShell`, `CacheStorageAssetKindStyle`, `CacheStorageAssetKindWasm`, `CacheStorageStrategyCacheFirst`, `CacheStorageStrategyNetworkFirst`, `CacheStorageStrategyStaleWhileRevalidate`, `ManifestDisplayBrowser`, `ManifestDisplayFullscreen`, `ManifestDisplayMinimalUI`, `ManifestDisplayStandalone`, `ManifestDisplayWindowControlsOverlay`, `ManifestOrientationAny`, `ManifestOrientationLandscape`, `ManifestOrientationLandscapePrimary`, `ManifestOrientationLandscapeSecondary`, `ManifestOrientationNatural`, `ManifestOrientationPortrait`, `ManifestOrientationPortraitPrimary`, `ManifestOrientationPortraitSecondary`, `ServiceWorkerStateActivated`, `ServiceWorkerStateActivating`, `ServiceWorkerStateInstalled`, `ServiceWorkerStateInstalling`, `ServiceWorkerStateRedundant`

## Subfiles And Purpose

- `browser_globals_wasm.go` - WebAssembly-specific implementation for browser_globals
- `cache_storage.go` - Core implementation for cache_storage
- `cache_storage_native.go` - Native (non-WASM) implementation for cache_storage
- `cache_storage_native_test.go` - Tests for cache_storage_native behavior
- `cache_storage_test.go` - Tests for cache_storage behavior
- `cache_storage_wasm.go` - WebAssembly-specific implementation for cache_storage
- `cache_storage_wasm_test.go` - Tests for cache_storage_wasm behavior
- `diagnostics.go` - Core implementation for diagnostics
- `diagnostics_native.go` - Native (non-WASM) implementation for diagnostics
- `diagnostics_native_test.go` - Tests for diagnostics_native behavior
- `diagnostics_queue_wasm.go` - WebAssembly-specific implementation for diagnostics_queue
- `diagnostics_wasm.go` - WebAssembly-specific implementation for diagnostics
- `diagnostics_wasm_test.go` - Tests for diagnostics_wasm behavior
- `doc.go` - Package-level Go documentation
- `helpers_additional_native_test.go` - Tests for helpers_additional_native behavior
- `helpers_additional_test.go` - Tests for helpers_additional behavior
- `helpers_additional_wasm_test.go` - Tests for helpers_additional_wasm behavior
- `installability.go` - Core implementation for installability
- `installability_native.go` - Native (non-WASM) implementation for installability
- `installability_native_test.go` - Tests for installability_native behavior
- `installability_wasm.go` - WebAssembly-specific implementation for installability
- `installability_wasm_test.go` - Tests for installability_wasm behavior
- `manifest.go` - Core implementation for manifest
- `manifest_test.go` - Tests for manifest behavior
- `release_manifest.go` - Core implementation for release_manifest
- `release_manifest_test.go` - Tests for release_manifest behavior
- `service_worker.go` - Core implementation for service_worker
- `service_worker_native.go` - Native (non-WASM) implementation for service_worker
- `service_worker_native_test.go` - Tests for service_worker_native behavior
- `service_worker_wasm.go` - WebAssembly-specific implementation for service_worker
- `service_worker_wasm_test.go` - Tests for service_worker_wasm behavior

## File Map

```text
pwa/
|-- browser_globals_wasm.go
|-- cache_storage.go
|-- cache_storage_native.go
|-- cache_storage_native_test.go
|-- cache_storage_test.go
|-- cache_storage_wasm.go
|-- cache_storage_wasm_test.go
|-- diagnostics.go
|-- diagnostics_native.go
|-- diagnostics_native_test.go
|-- diagnostics_queue_wasm.go
|-- diagnostics_wasm.go
|-- diagnostics_wasm_test.go
|-- doc.go
|-- helpers_additional_native_test.go
|-- helpers_additional_test.go
|-- helpers_additional_wasm_test.go
|-- installability.go
|-- installability_native.go
|-- installability_native_test.go
|-- installability_wasm.go
|-- installability_wasm_test.go
|-- manifest.go
|-- manifest_test.go
|-- release_manifest.go
|-- release_manifest_test.go
|-- service_worker.go
|-- service_worker_native.go
|-- service_worker_native_test.go
|-- service_worker_wasm.go
\-- service_worker_wasm_test.go
```



