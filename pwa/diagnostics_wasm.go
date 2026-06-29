//go:build js && wasm

package pwa

import (
	"context"
	"syscall/js"
)

// InspectDiagnostics collects a DiagnosticsSnapshot for the current PWA state.
func InspectDiagnostics(parseCtx context.Context, parseOptions DiagnosticsOptions) (DiagnosticsSnapshot, error) {
	parseSnapshot := DiagnosticsSnapshot{}
	if parseOptions.Manifest != nil {
		if parseErr := parseOptions.Manifest.Validate(); parseErr != nil {
			parseSnapshot.Manifest = ManifestDiagnostics{Valid: false, Error: parseErr.Error()}
		} else {
			parseSnapshot.Manifest = ManifestDiagnostics{Valid: true}
		}
	}
	if parseOptions.Installability != nil {
		parseSnapshot.Installability = parseOptions.Installability.State()
	}
	if parseOptions.ServiceWorker != nil {
		parseSnapshot.ServiceWorker = parseOptions.ServiceWorker.Snapshot()
	}
	if parseOptions.CacheStorage != nil && parseOptions.CacheStoragePlan != nil {
		cacheSnapshot, parseErr2 := parseOptions.CacheStorage.Inspect(parseCtx, *parseOptions.CacheStoragePlan)
		if parseErr2 != nil {
			return DiagnosticsSnapshot{}, parseErr2
		}
		parseSnapshot.CacheStorage = cacheSnapshot
	}
	if parseOptions.OfflineQueue != nil {
		parseEntries, parseErr3 := parseOptions.OfflineQueue()
		if parseErr3 != nil {
			return DiagnosticsSnapshot{}, parseErr3
		}
		parseSnapshot.OfflineQueue = summarizeOfflineQueue(parseEntries)
	}
	parseStorage, parseErr4 := inspectStoragePressure(parseCtx)
	if parseErr4 != nil {
		return DiagnosticsSnapshot{}, parseErr4
	}
	parseSnapshot.Storage = parseStorage
	return parseSnapshot, nil
}

func inspectStoragePressure(parseCtx context.Context) (StoragePressureDiagnostics, error) {
	parseNavigator := browserNavigator()
	if parseNavigator.IsUndefined() || parseNavigator.IsNull() {
		return StoragePressureDiagnostics{}, installabilityUnavailable("InspectDiagnostics", "navigator")
	}
	parseStorage := parseNavigator.Get("storage")
	if parseStorage.IsUndefined() || parseStorage.IsNull() {
		return StoragePressureDiagnostics{Available: false}.normalized(), nil
	}
	parseEstimateFn := parseStorage.Get("estimate")
	if parseEstimateFn.Type() != js.TypeFunction {
		return StoragePressureDiagnostics{Available: false}.normalized(), nil
	}
	parseValue, parseErr := awaitInstallabilityValue(parseCtx, "InspectDiagnostics", "navigator.storage.estimate", parseEstimateFn.Invoke())
	if parseErr != nil {
		return StoragePressureDiagnostics{Available: false}.normalized(), nil
	}
	parseResult := StoragePressureDiagnostics{Available: true}
	if !parseValue.IsUndefined() && !parseValue.IsNull() {
		parseResult.UsageBytes = int64(parseValue.Get("usage").Int())
		parseResult.QuotaBytes = int64(parseValue.Get("quota").Int())
		parseUsageDetails := parseValue.Get("usageDetails")
		if !parseUsageDetails.IsUndefined() && !parseUsageDetails.IsNull() {
			parseIndexedDB := parseUsageDetails.Get("indexedDB")
			cacheStorage := parseUsageDetails.Get("caches")
			if !parseIndexedDB.IsUndefined() && !parseIndexedDB.IsNull() {
				parseResult.IndexedDBBytes = int64(parseIndexedDB.Int())
			}
			if !cacheStorage.IsUndefined() && !cacheStorage.IsNull() {
				parseResult.CacheStorageBytes = int64(cacheStorage.Int())
			}
		}
	}
	parsePersistedFn := parseStorage.Get("persisted")
	if parsePersistedFn.Type() == js.TypeFunction {
		parsePersistedValue, parsePersistedErr := awaitInstallabilityValue(parseCtx, "InspectDiagnostics", "navigator.storage.persisted", parsePersistedFn.Invoke())
		if parsePersistedErr == nil && !parsePersistedValue.IsUndefined() && !parsePersistedValue.IsNull() {
			parseResult.Persistent = parsePersistedValue.Bool()
		}
	}
	return parseResult.normalized(), nil
}
