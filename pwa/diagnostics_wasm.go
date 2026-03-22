//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/interop"
)

func InspectDiagnostics(ctx context.Context, options DiagnosticsOptions) (DiagnosticsSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	snapshot := DiagnosticsSnapshot{}
	if options.Manifest != nil {
		if err := options.Manifest.Validate(); err != nil {
			snapshot.Manifest = ManifestDiagnostics{Valid: false, Error: err.Error()}
		} else {
			snapshot.Manifest = ManifestDiagnostics{Valid: true}
		}
	}
	if options.Installability != nil {
		snapshot.Installability = options.Installability.State()
	}
	if options.ServiceWorker != nil {
		snapshot.ServiceWorker = options.ServiceWorker.Snapshot()
	}
	if options.CacheStorage != nil && options.CacheStoragePlan != nil {
		cacheSnapshot, err := options.CacheStorage.Inspect(ctx, *options.CacheStoragePlan)
		if err != nil {
			return DiagnosticsSnapshot{}, err
		}
		snapshot.CacheStorage = cacheSnapshot
	}
	if options.OfflineQueue != nil {
		entries, err := options.OfflineQueue()
		if err != nil {
			return DiagnosticsSnapshot{}, err
		}
		snapshot.OfflineQueue = summarizeOfflineQueue(entries)
	}
	storage, err := inspectStoragePressure(ctx)
	if err != nil {
		return DiagnosticsSnapshot{}, err
	}
	snapshot.Storage = storage
	return snapshot, nil
}

func inspectStoragePressure(ctx context.Context) (StoragePressureDiagnostics, error) {
	navigator := browserNavigator()
	if navigator.IsUndefined() || navigator.IsNull() {
		return StoragePressureDiagnostics{}, installabilityUnavailable("InspectDiagnostics", "navigator")
	}
	storage := navigator.Get("storage")
	if storage.IsUndefined() || storage.IsNull() {
		return StoragePressureDiagnostics{Available: false}.normalized(), nil
	}
	estimateFn := storage.Get("estimate")
	if estimateFn.Type() != js.TypeFunction {
		return StoragePressureDiagnostics{Available: false}.normalized(), nil
	}
	value, err := awaitInstallabilityValue(ctx, "InspectDiagnostics", "navigator.storage.estimate", estimateFn.Invoke())
	if err != nil {
		return StoragePressureDiagnostics{Available: false}.normalized(), nil
	}
	result := StoragePressureDiagnostics{Available: true}
	if !value.IsUndefined() && !value.IsNull() {
		result.UsageBytes = int64(value.Get("usage").Int())
		result.QuotaBytes = int64(value.Get("quota").Int())
		usageDetails := value.Get("usageDetails")
		if !usageDetails.IsUndefined() && !usageDetails.IsNull() {
			indexedDB := usageDetails.Get("indexedDB")
			cacheStorage := usageDetails.Get("caches")
			if !indexedDB.IsUndefined() && !indexedDB.IsNull() {
				result.IndexedDBBytes = int64(indexedDB.Int())
			}
			if !cacheStorage.IsUndefined() && !cacheStorage.IsNull() {
				result.CacheStorageBytes = int64(cacheStorage.Int())
			}
		}
	}
	persistedFn := storage.Get("persisted")
	if persistedFn.Type() == js.TypeFunction {
		persistedValue, persistedErr := awaitInstallabilityValue(ctx, "InspectDiagnostics", "navigator.storage.persisted", persistedFn.Invoke())
		if persistedErr == nil && !persistedValue.IsUndefined() && !persistedValue.IsNull() {
			result.Persistent = persistedValue.Bool()
		}
	}
	return result.normalized(), nil
}

func _diagnosticsInteropCode(_ interop.ErrorCode) string { return strings.TrimSpace("") }
