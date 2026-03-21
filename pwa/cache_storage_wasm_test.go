//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"sort"
	"syscall/js"
	"testing"
)

func TestCacheStorageManagerSyncAndInspect(t *testing.T) {
	restore := installMockCacheStorage(t)
	defer restore()

	manager, err := OpenCacheStorageManager()
	if err != nil {
		t.Fatalf("expected cache storage manager, got %v", err)
	}
	plan := CacheStoragePlan{
		CacheName:   "atlas-release-1234",
		CachePrefix: "atlas-release-",
		Entries: []CacheStorageEntry{
			{URL: "/app.wasm", Kind: CacheStorageAssetKindWasm, Strategy: CacheStorageStrategyCacheFirst},
			{URL: "/index.html", Kind: CacheStorageAssetKindShell, Strategy: CacheStorageStrategyNetworkFirst},
		},
	}
	snapshot, err := manager.Sync(context.Background(), plan)
	if err != nil {
		t.Fatalf("expected cache storage sync, got %v", err)
	}
	if snapshot.CacheName != plan.CacheName || snapshot.EntryCount != 2 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	inspected, err := manager.Inspect(context.Background(), plan)
	if err != nil {
		t.Fatalf("expected cache storage inspect, got %v", err)
	}
	if len(inspected.CacheNames) != 1 || inspected.CacheNames[0] != plan.CacheName {
		t.Fatalf("unexpected cache names: %#v", inspected.CacheNames)
	}
	if len(inspected.Entries) != 2 {
		t.Fatalf("unexpected cache entries: %#v", inspected.Entries)
	}
}

func installMockCacheStorage(t *testing.T) func() {
	t.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	makePromise := func(value js.Value) js.Value {
		return global.Get("Promise").Call("resolve", value)
	}
	type cacheRecord struct {
		urls []string
	}
	records := map[string]*cacheRecord{}
	makeCache := func(name string) js.Value {
		cache := objectCtor.New()
		add := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			url := args[0].String()
			record := records[name]
			found := false
			for _, current := range record.urls {
				if current == url {
					found = true
					break
				}
			}
			if !found {
				record.urls = append(record.urls, url)
				sort.Strings(record.urls)
			}
			return makePromise(js.Undefined())
		})
		keys := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			array := js.Global().Get("Array").New()
			for _, url := range records[name].urls {
				request := objectCtor.New()
				request.Set("url", url)
				array.Call("push", request)
			}
			return makePromise(array)
		})
		cache.Set("add", add)
		cache.Set("keys", keys)
		t.Cleanup(func() {
			add.Release()
			keys.Release()
		})
		return cache
	}
	caches := objectCtor.New()
	open := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		name := args[0].String()
		if records[name] == nil {
			records[name] = &cacheRecord{}
		}
		return makePromise(makeCache(name))
	})
	keys := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		array := js.Global().Get("Array").New()
		names := make([]string, 0, len(records))
		for name := range records {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			array.Call("push", name)
		}
		return makePromise(array)
	})
	deleteFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		delete(records, args[0].String())
		return makePromise(js.ValueOf(true))
	})
	caches.Set("open", open)
	caches.Set("keys", keys)
	caches.Set("delete", deleteFn)
	previous := global.Get("caches")
	global.Set("caches", caches)
	return func() {
		global.Set("caches", previous)
		open.Release()
		keys.Release()
		deleteFn.Release()
	}
}
