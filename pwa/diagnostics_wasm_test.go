//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/interop"
)

func TestInspectDiagnosticsCollectsPWASnapshot(parseT *testing.T) {
	parseRestoreSW := installMockServiceWorkerEnvironment(parseT, false)
	defer parseRestoreSW()
	parseWindow := js.Global().Get("window")
	parseNavigator := parseWindow.Get("navigator")
	installEventTargetOnWindow(parseT, parseWindow)
	attachMockStorageManager(parseT, parseNavigator)
	parseRestoreCaches := installMockCacheStorage(parseT)
	defer parseRestoreCaches()
	parseRestoreQueue := installMockLocalStorageForQueue(parseT)
	defer parseRestoreQueue()

	parseInstallability, parseErr := ObserveInstallability(InstallabilityOptions{Manifest: &Manifest{Name: "Atlas", StartURL: "/"}})
	if parseErr != nil {
		parseT.Fatalf("expected installability manager, got %v", parseErr)
	}
	parseRegistration, parseErr := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: "/sw.js", Scope: "/app"})
	if parseErr != nil {
		parseT.Fatalf("expected service worker registration, got %v", parseErr)
	}
	cacheManager, parseErr := OpenCacheStorageManager()
	if parseErr != nil {
		parseT.Fatalf("expected cache storage manager, got %v", parseErr)
	}
	cachePlan := CacheStoragePlan{
		CacheName:   "atlas-release-1234",
		CachePrefix: "atlas-release-",
		Entries: []CacheStorageEntry{
			{URL: "/app.wasm", Kind: CacheStorageAssetKindWasm, Strategy: CacheStorageStrategyCacheFirst},
			{URL: "/offline.html", Kind: CacheStorageAssetKindShell, Strategy: CacheStorageStrategyNetworkFirst},
		},
	}
	if _, parseErr2 := cacheManager.Sync(context.Background(), cachePlan); parseErr2 != nil {
		parseT.Fatalf("expected cache sync, got %v", parseErr2)
	}
	parseQueue, parseErr := fetch.OpenMutationQueue(fetch.MutationQueueOptions{StorageResolver: interop.LocalStorage})
	if parseErr != nil {
		parseT.Fatalf("expected mutation queue, got %v", parseErr)
	}
	if _, parseErr3 := parseQueue.Enqueue(fetch.MutationDraft{URL: "/api/orders", Method: "POST", Kind: "order.submit"}); parseErr3 != nil {
		parseT.Fatalf("expected queue entry, got %v", parseErr3)
	}

	parseSnapshot, parseErr := InspectDiagnostics(context.Background(), DiagnosticsOptions{
		Manifest:         &Manifest{Name: "Atlas", StartURL: "/"},
		Installability:   &parseInstallability,
		ServiceWorker:    &parseRegistration,
		CacheStorage:     &cacheManager,
		CacheStoragePlan: &cachePlan,
		OfflineQueue:     BuildMutationQueueDiagnosticsSource(&parseQueue),
	})
	if parseErr != nil {
		parseT.Fatalf("expected diagnostics snapshot, got %v", parseErr)
	}
	if !parseSnapshot.Manifest.Valid {
		parseT.Fatalf("expected valid manifest diagnostics, got %+v", parseSnapshot.Manifest)
	}
	if parseSnapshot.ServiceWorker.Scope != "/app" {
		parseT.Fatalf("unexpected service worker snapshot: %+v", parseSnapshot.ServiceWorker)
	}
	if parseSnapshot.CacheStorage.EntryCount != 2 {
		parseT.Fatalf("unexpected cache storage snapshot: %+v", parseSnapshot.CacheStorage)
	}
	if parseSnapshot.OfflineQueue.TotalEntries != 1 || parseSnapshot.OfflineQueue.QueuedEntries != 1 {
		parseT.Fatalf("unexpected offline queue snapshot: %+v", parseSnapshot.OfflineQueue)
	}
	if !parseSnapshot.Storage.Available || parseSnapshot.Storage.QuotaBytes != 1000 || parseSnapshot.Storage.IndexedDBBytes != 250 {
		parseT.Fatalf("unexpected storage diagnostics: %+v", parseSnapshot.Storage)
	}
	if parseSnapshot.Storage.Pressure == "" {
		parseT.Fatalf("expected storage pressure label, got %+v", parseSnapshot.Storage)
	}
}

// TestBuildMutationQueueDiagnosticsSourceWasmReturnsNilForMissingQueue verifies the diagnostics queue wrapper stays nil-safe.
func TestBuildMutationQueueDiagnosticsSourceWasmReturnsNilForMissingQueue(parseT *testing.T) {
	if parseSource := BuildMutationQueueDiagnosticsSource(nil); parseSource != nil {
		parseT.Fatal("expected nil mutation queue diagnostics source for a missing queue")
	}
}

func installEventTargetOnWindow(parseT *testing.T, parseWindow js.Value) {
	parseT.Helper()
	parseListeners := map[string][]js.Value{}
	parseAdd := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseEventName := parseArgs[0].String()
		parseListeners[parseEventName] = append(parseListeners[parseEventName], parseArgs[1])
		return nil
	})
	parseRemove := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseEventName2 := parseArgs2[0].String()
		parseRemaining := parseListeners[parseEventName2][:0]
		for _, parseCurrent := range parseListeners[parseEventName2] {
			if !parseCurrent.Equal(parseArgs2[1]) {
				parseRemaining = append(parseRemaining, parseCurrent)
			}
		}
		parseListeners[parseEventName2] = parseRemaining
		return nil
	})
	parseDispatch := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseEventName3 := parseArgs3[0].Get("type").String()
		for _, parseListener := range parseListeners[parseEventName3] {
			parseListener.Invoke(parseArgs3[0])
		}
		return true
	})
	parseMatchMedia := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseResult := js.Global().Get("Object").New()
		parseResult.Set("matches", false)
		return parseResult
	})
	parseWindow.Set("addEventListener", parseAdd)
	parseWindow.Set("removeEventListener", parseRemove)
	parseWindow.Set("dispatchEvent", parseDispatch)
	parseWindow.Set("matchMedia", parseMatchMedia)
	parseWindow.Set("isSecureContext", true)
	parseT.Cleanup(func() {
		parseAdd.Release()
		parseRemove.Release()
		parseDispatch.Release()
		parseMatchMedia.Release()
	})
}

func attachMockStorageManager(parseT *testing.T, parseNavigator js.Value) {
	parseT.Helper()
	parseStorage := js.Global().Get("Object").New()
	parseEstimate := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseResult := js.Global().Get("Object").New()
		parseUsageDetails := js.Global().Get("Object").New()
		parseUsageDetails.Set("indexedDB", 250)
		parseUsageDetails.Set("caches", 125)
		parseResult.Set("usage", 500)
		parseResult.Set("quota", 1000)
		parseResult.Set("usageDetails", parseUsageDetails)
		return js.Global().Get("Promise").Call("resolve", parseResult)
	})
	parsePersisted := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return js.Global().Get("Promise").Call("resolve", js.ValueOf(true))
	})
	parseStorage.Set("estimate", parseEstimate)
	parseStorage.Set("persisted", parsePersisted)
	parseNavigator.Set("storage", parseStorage)
	parseNavigator.Set("standalone", false)
	parseT.Cleanup(func() {
		parseEstimate.Release()
		parsePersisted.Release()
	})
}

func installMockLocalStorageForQueue(parseT *testing.T) func() {
	parseT.Helper()
	parseStorage := js.Global().Get("Object").New()
	parseData := map[string]string{}
	getItem := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if parseValue, parseOk := parseData[parseArgs[0].String()]; parseOk {
			return parseValue
		}
		return js.Null()
	})
	setItem := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseData[parseArgs2[0].String()] = parseArgs2[1].String()
		parseStorage.Set("length", len(parseData))
		return nil
	})
	parseRemoveItem := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		delete(parseData, parseArgs3[0].String())
		parseStorage.Set("length", len(parseData))
		return nil
	})
	clearFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		for parseKey := range parseData {
			delete(parseData, parseKey)
		}
		parseStorage.Set("length", 0)
		return nil
	})
	parseKeyFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		return js.Null()
	})
	parseStorage.Set("getItem", getItem)
	parseStorage.Set("setItem", setItem)
	parseStorage.Set("removeItem", parseRemoveItem)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)
	parsePreviousIndexedDB := js.Global().Get("indexedDB")
	parsePreviousStorage := js.Global().Get("localStorage")
	js.Global().Set("indexedDB", js.Undefined())
	js.Global().Set("localStorage", parseStorage)
	return func() {
		js.Global().Set("indexedDB", parsePreviousIndexedDB)
		js.Global().Set("localStorage", parsePreviousStorage)
		getItem.Release()
		setItem.Release()
		parseRemoveItem.Release()
		clearFn.Release()
		parseKeyFn.Release()
	}
}
