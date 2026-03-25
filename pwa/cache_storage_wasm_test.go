//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"sort"
	"syscall/js"
	"testing"
)

func TestCacheStorageManagerSyncAndInspect(parseT *testing.T) {
	parseRestore := installMockCacheStorage(parseT)
	defer parseRestore()

	parseManager, parseErr := OpenCacheStorageManager()
	if parseErr != nil {
		parseT.Fatalf("expected cache storage manager, got %v", parseErr)
	}
	parsePlan := CacheStoragePlan{
		CacheName:   "atlas-release-1234",
		CachePrefix: "atlas-release-",
		Entries: []CacheStorageEntry{
			{URL: "/app.wasm", Kind: CacheStorageAssetKindWasm, Strategy: CacheStorageStrategyCacheFirst},
			{URL: "/index.html", Kind: CacheStorageAssetKindShell, Strategy: CacheStorageStrategyNetworkFirst},
		},
	}
	parseSnapshot, parseErr := parseManager.Sync(context.Background(), parsePlan)
	if parseErr != nil {
		parseT.Fatalf("expected cache storage sync, got %v", parseErr)
	}
	if parseSnapshot.CacheName != parsePlan.CacheName || parseSnapshot.EntryCount != 2 {
		parseT.Fatalf("unexpected snapshot: %+v", parseSnapshot)
	}
	parseInspected, parseErr := parseManager.Inspect(context.Background(), parsePlan)
	if parseErr != nil {
		parseT.Fatalf("expected cache storage inspect, got %v", parseErr)
	}
	if len(parseInspected.CacheNames) != 1 || parseInspected.CacheNames[0] != parsePlan.CacheName {
		parseT.Fatalf("unexpected cache names: %#v", parseInspected.CacheNames)
	}
	if len(parseInspected.Entries) != 2 {
		parseT.Fatalf("unexpected cache entries: %#v", parseInspected.Entries)
	}
}

func installMockCacheStorage(parseT *testing.T) func() {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseMakePromise := func(parseValue js.Value) js.Value {
		return parseGlobal.Get("Promise").Call("resolve", parseValue)
	}
	type cacheRecord struct {
		urls []string
	}
	parseRecords := map[string]*cacheRecord{}
	parseMakeCache := func(parseName4 string) js.Value {
		cache := parseObjectCtor.New()
		parseAdd := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseUrl := parseArgs[0].String()
			parseRecord := parseRecords[parseName4]
			isParseFound := false
			for _, parseCurrent := range parseRecord.urls {
				if parseCurrent == parseUrl {
					isParseFound = true
					break
				}
			}
			if !isParseFound {
				parseRecord.urls = append(parseRecord.urls, parseUrl)
				sort.Strings(parseRecord.urls)
			}
			return parseMakePromise(js.Undefined())
		})
		parseKeys := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseArray := js.Global().Get("Array").New()
			for _, parseUrl2 := range parseRecords[parseName4].urls {
				parseRequest := parseObjectCtor.New()
				parseRequest.Set("url", parseUrl2)
				parseArray.Call("push", parseRequest)
			}
			return parseMakePromise(parseArray)
		})
		cache.Set("add", parseAdd)
		cache.Set("keys", parseKeys)
		parseT.Cleanup(func() {
			parseAdd.Release()
			parseKeys.Release()
		})
		return cache
	}
	parseCaches := parseObjectCtor.New()
	parseOpen := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseName := parseArgs3[0].String()
		if parseRecords[parseName] == nil {
			parseRecords[parseName] = &cacheRecord{}
		}
		return parseMakePromise(parseMakeCache(parseName))
	})
	parseKeys2 := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseArray2 := js.Global().Get("Array").New()
		parseNames := make([]string, 0, len(parseRecords))
		for parseName2 := range parseRecords {
			parseNames = append(parseNames, parseName2)
		}
		sort.Strings(parseNames)
		for _, parseName3 := range parseNames {
			parseArray2.Call("push", parseName3)
		}
		return parseMakePromise(parseArray2)
	})
	parseDeleteFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		delete(parseRecords, parseArgs5[0].String())
		return parseMakePromise(js.ValueOf(true))
	})
	parseCaches.Set("open", parseOpen)
	parseCaches.Set("keys", parseKeys2)
	parseCaches.Set("delete", parseDeleteFn)
	parsePrevious := parseGlobal.Get("caches")
	parseGlobal.Set("caches", parseCaches)
	return func() {
		parseGlobal.Set("caches", parsePrevious)
		parseOpen.Release()
		parseKeys2.Release()
		parseDeleteFn.Release()
	}
}
