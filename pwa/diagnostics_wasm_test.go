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

func TestInspectDiagnosticsCollectsPWASnapshot(t *testing.T) {
	restoreSW := installMockServiceWorkerEnvironment(t, false)
	defer restoreSW()
	window := js.Global().Get("window")
	navigator := window.Get("navigator")
	installEventTargetOnWindow(t, window)
	attachMockStorageManager(t, navigator)
	restoreCaches := installMockCacheStorage(t)
	defer restoreCaches()
	restoreQueue := installMockLocalStorageForQueue(t)
	defer restoreQueue()

	installability, err := ObserveInstallability(InstallabilityOptions{Manifest: &Manifest{Name: "Atlas", StartURL: "/"}})
	if err != nil {
		t.Fatalf("expected installability manager, got %v", err)
	}
	registration, err := RegisterServiceWorker(context.Background(), ServiceWorkerOptions{URL: "/sw.js", Scope: "/app"})
	if err != nil {
		t.Fatalf("expected service worker registration, got %v", err)
	}
	cacheManager, err := OpenCacheStorageManager()
	if err != nil {
		t.Fatalf("expected cache storage manager, got %v", err)
	}
	cachePlan := CacheStoragePlan{
		CacheName:   "atlas-release-1234",
		CachePrefix: "atlas-release-",
		Entries: []CacheStorageEntry{
			{URL: "/app.wasm", Kind: CacheStorageAssetKindWasm, Strategy: CacheStorageStrategyCacheFirst},
			{URL: "/offline.html", Kind: CacheStorageAssetKindShell, Strategy: CacheStorageStrategyNetworkFirst},
		},
	}
	if _, err := cacheManager.Sync(context.Background(), cachePlan); err != nil {
		t.Fatalf("expected cache sync, got %v", err)
	}
	queue, err := fetch.OpenMutationQueue(fetch.MutationQueueOptions{StorageResolver: interop.LocalStorage})
	if err != nil {
		t.Fatalf("expected mutation queue, got %v", err)
	}
	if _, err := queue.Enqueue(fetch.MutationDraft{URL: "/api/orders", Method: "POST", Kind: "order.submit"}); err != nil {
		t.Fatalf("expected queue entry, got %v", err)
	}

	snapshot, err := InspectDiagnostics(context.Background(), DiagnosticsOptions{
		Manifest:         &Manifest{Name: "Atlas", StartURL: "/"},
		Installability:   &installability,
		ServiceWorker:    &registration,
		CacheStorage:     &cacheManager,
		CacheStoragePlan: &cachePlan,
		OfflineQueue:     BuildMutationQueueDiagnosticsSource(&queue),
	})
	if err != nil {
		t.Fatalf("expected diagnostics snapshot, got %v", err)
	}
	if !snapshot.Manifest.Valid {
		t.Fatalf("expected valid manifest diagnostics, got %+v", snapshot.Manifest)
	}
	if snapshot.ServiceWorker.Scope != "/app" {
		t.Fatalf("unexpected service worker snapshot: %+v", snapshot.ServiceWorker)
	}
	if snapshot.CacheStorage.EntryCount != 2 {
		t.Fatalf("unexpected cache storage snapshot: %+v", snapshot.CacheStorage)
	}
	if snapshot.OfflineQueue.TotalEntries != 1 || snapshot.OfflineQueue.QueuedEntries != 1 {
		t.Fatalf("unexpected offline queue snapshot: %+v", snapshot.OfflineQueue)
	}
	if !snapshot.Storage.Available || snapshot.Storage.QuotaBytes != 1000 || snapshot.Storage.IndexedDBBytes != 250 {
		t.Fatalf("unexpected storage diagnostics: %+v", snapshot.Storage)
	}
	if snapshot.Storage.Pressure == "" {
		t.Fatalf("expected storage pressure label, got %+v", snapshot.Storage)
	}
}

func installEventTargetOnWindow(t *testing.T, window js.Value) {
	t.Helper()
	listeners := map[string][]js.Value{}
	add := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		eventName := args[0].String()
		listeners[eventName] = append(listeners[eventName], args[1])
		return nil
	})
	remove := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		eventName := args[0].String()
		remaining := listeners[eventName][:0]
		for _, current := range listeners[eventName] {
			if !current.Equal(args[1]) {
				remaining = append(remaining, current)
			}
		}
		listeners[eventName] = remaining
		return nil
	})
	dispatch := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		eventName := args[0].Get("type").String()
		for _, listener := range listeners[eventName] {
			listener.Invoke(args[0])
		}
		return true
	})
	matchMedia := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result := js.Global().Get("Object").New()
		result.Set("matches", false)
		return result
	})
	window.Set("addEventListener", add)
	window.Set("removeEventListener", remove)
	window.Set("dispatchEvent", dispatch)
	window.Set("matchMedia", matchMedia)
	window.Set("isSecureContext", true)
	t.Cleanup(func() {
		add.Release()
		remove.Release()
		dispatch.Release()
		matchMedia.Release()
	})
}

func attachMockStorageManager(t *testing.T, navigator js.Value) {
	t.Helper()
	storage := js.Global().Get("Object").New()
	estimate := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result := js.Global().Get("Object").New()
		usageDetails := js.Global().Get("Object").New()
		usageDetails.Set("indexedDB", 250)
		usageDetails.Set("caches", 125)
		result.Set("usage", 500)
		result.Set("quota", 1000)
		result.Set("usageDetails", usageDetails)
		return js.Global().Get("Promise").Call("resolve", result)
	})
	persisted := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.Global().Get("Promise").Call("resolve", js.ValueOf(true))
	})
	storage.Set("estimate", estimate)
	storage.Set("persisted", persisted)
	navigator.Set("storage", storage)
	navigator.Set("standalone", false)
	t.Cleanup(func() {
		estimate.Release()
		persisted.Release()
	})
}

func installMockLocalStorageForQueue(t *testing.T) func() {
	t.Helper()
	storage := js.Global().Get("Object").New()
	data := map[string]string{}
	getItem := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if value, ok := data[args[0].String()]; ok {
			return value
		}
		return js.Null()
	})
	setItem := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		data[args[0].String()] = args[1].String()
		storage.Set("length", len(data))
		return nil
	})
	removeItem := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		delete(data, args[0].String())
		storage.Set("length", len(data))
		return nil
	})
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		for key := range data {
			delete(data, key)
		}
		storage.Set("length", 0)
		return nil
	})
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.Null()
	})
	storage.Set("getItem", getItem)
	storage.Set("setItem", setItem)
	storage.Set("removeItem", removeItem)
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	storage.Set("length", 0)
	previousIndexedDB := js.Global().Get("indexedDB")
	previousStorage := js.Global().Get("localStorage")
	js.Global().Set("indexedDB", js.Undefined())
	js.Global().Set("localStorage", storage)
	return func() {
		js.Global().Set("indexedDB", previousIndexedDB)
		js.Global().Set("localStorage", previousStorage)
		getItem.Release()
		setItem.Release()
		removeItem.Release()
		clearFn.Release()
		keyFn.Release()
	}
}
