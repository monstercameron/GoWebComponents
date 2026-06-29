//go:build js && wasm

package pwa

import (
	"context"
	"errors"
	"sort"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/interop"
)

// OpenCacheStorageManager opens the browser Cache Storage API manager.
func OpenCacheStorageManager() (CacheStorageManager, error) {
	parseCaches := js.Global().Get("caches")
	if parseCaches.IsUndefined() || parseCaches.IsNull() {
		return CacheStorageManager{}, cacheStorageUnavailable("OpenCacheStorageManager", "caches")
	}
	return CacheStorageManager{
		sync: func(parseCtx context.Context, parsePlan CacheStoragePlan) (CacheStorageSnapshot, error) {
			if parseCtx == nil {
				parseCtx = context.Background()
			}
			if strings.TrimSpace(parsePlan.CacheName) == "" {
				return CacheStorageSnapshot{}, &interop.Error{Op: "CacheStorageManager.Sync", Code: interop.CodeInvalid, Err: errors.New("cache name is empty")}
			}
			parseNames, parseErr := cacheStorageKeys(parseCtx, parseCaches)
			if parseErr != nil {
				return CacheStorageSnapshot{}, parseErr
			}
			for _, parseName := range parseNames {
				if parseName == parsePlan.CacheName {
					continue
				}
				if parsePlan.CachePrefix != "" && strings.HasPrefix(parseName, parsePlan.CachePrefix) {
					if _, parseErr2 := awaitCacheStorageValue(parseCtx, "CacheStorageManager.Sync", parseName, parseCaches.Call("delete", parseName)); parseErr2 != nil {
						return CacheStorageSnapshot{}, parseErr2
					}
				}
			}
			cacheValue, parseErr := awaitCacheStorageValue(parseCtx, "CacheStorageManager.Sync", parsePlan.CacheName, parseCaches.Call("open", parsePlan.CacheName))
			if parseErr != nil {
				return CacheStorageSnapshot{}, parseErr
			}
			for _, parseEntry := range parsePlan.Entries {
				if _, parseErr3 := awaitCacheStorageValue(parseCtx, "CacheStorageManager.Sync", parseEntry.URL, cacheValue.Call("add", parseEntry.URL)); parseErr3 != nil {
					return CacheStorageSnapshot{}, parseErr3
				}
			}
			return inspectCacheStorage(parseCtx, parseCaches, parsePlan)
		},
		inspect: func(parseCtx2 context.Context, parsePlan2 CacheStoragePlan) (CacheStorageSnapshot, error) {
			if parseCtx2 == nil {
				parseCtx2 = context.Background()
			}
			return inspectCacheStorage(parseCtx2, parseCaches, parsePlan2)
		},
	}, nil
}

// inspectCacheStorage reads one cache snapshot after sync or standalone inspection.
func inspectCacheStorage(parseCtx context.Context, parseCaches js.Value, parsePlan CacheStoragePlan) (CacheStorageSnapshot, error) {
	parseNames, parseErr := cacheStorageKeys(parseCtx, parseCaches)
	if parseErr != nil {
		return CacheStorageSnapshot{}, parseErr
	}
	cacheValue, parseErr := awaitCacheStorageValue(parseCtx, "CacheStorageManager.Inspect", parsePlan.CacheName, parseCaches.Call("open", parsePlan.CacheName))
	if parseErr != nil {
		return CacheStorageSnapshot{}, parseErr
	}
	parseKeysValue, parseErr := awaitCacheStorageValue(parseCtx, "CacheStorageManager.Inspect", parsePlan.CacheName, cacheValue.Call("keys"))
	if parseErr != nil {
		return CacheStorageSnapshot{}, parseErr
	}
	parseEntriesByURL := map[string]CacheStorageEntry{}
	for _, parseEntry := range parsePlan.Entries {
		parseEntriesByURL[parseEntry.URL] = parseEntry
	}
	parseEntries := make([]CacheStorageEntry, 0, parseKeysValue.Length())
	for parseI := 0; parseI < parseKeysValue.Length(); parseI++ {
		parseRequest := parseKeysValue.Index(parseI)
		parseUrl := strings.TrimSpace(parseRequest.Get("url").String())
		parseEntry2, parseOk := parseEntriesByURL[parseUrl]
		if !parseOk {
			parseEntry2 = CacheStorageEntry{URL: parseUrl, Kind: CacheStorageAssetKindAsset, Strategy: CacheStorageStrategyCacheFirst}
		}
		parseEntries = append(parseEntries, parseEntry2)
	}
	sort.Strings(parseNames)
	return CacheStorageSnapshot{CacheName: parsePlan.CacheName, CachePrefix: parsePlan.CachePrefix, EntryCount: len(parseEntries), Entries: parseEntries, CacheNames: parseNames}, nil
}

// cacheStorageKeys returns the normalized cache names currently exposed by the browser.
func cacheStorageKeys(parseCtx context.Context, parseCaches js.Value) ([]string, error) {
	parseValue, parseErr := awaitCacheStorageValue(parseCtx, "CacheStorageManager.Keys", "caches.keys", parseCaches.Call("keys"))
	if parseErr != nil {
		return nil, parseErr
	}
	parseNames := make([]string, 0, parseValue.Length())
	for parseI := 0; parseI < parseValue.Length(); parseI++ {
		parseNames = append(parseNames, strings.TrimSpace(parseValue.Index(parseI).String()))
	}
	return parseNames, nil
}

// awaitCacheStorageValue resolves one cache-storage return value that may already be settled or may be promise-like.
func awaitCacheStorageValue(parseCtx context.Context, parseOp string, parseTarget string, parseValue js.Value) (js.Value, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseValueType := parseValue.Type()
	// Primitive results are already settled values under syscall/js and do not safely expose promise methods.
	if parseValueType != js.TypeObject && parseValueType != js.TypeFunction {
		return parseValue, nil
	}
	parseThen := parseValue.Get("then")
	if parseThen.Type() != js.TypeFunction {
		return parseValue, nil
	}
	parseResolvedCh := make(chan js.Value, 1)
	parseRejectedCh := make(chan error, 1)
	var parseResolveFn js.Func
	var parseRejectFn js.Func
	parseCleanup := func() {
		parseResolveFn.Release()
		parseRejectFn.Release()
	}
	parseResolveFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseResolved := js.Undefined()
		if len(parseArgs) > 0 {
			parseResolved = parseArgs[0]
		}
		select {
		case parseResolvedCh <- parseResolved:
		default:
		}
		return nil
	})
	parseRejectFn = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseMessage := "cache storage promise rejected"
		if len(parseArgs2) > 0 {
			if parseText := strings.TrimSpace(parseArgs2[0].String()); parseText != "" {
				parseMessage = parseText
			}
		}
		select {
		case parseRejectedCh <- &interop.Error{Op: parseOp, Target: parseTarget, Code: interop.CodePromiseRejected, Err: errors.New(parseMessage)}:
		default:
		}
		return nil
	})
	parseValue.Call("then", parseResolveFn).Call("catch", parseRejectFn)
	defer parseCleanup()

	select {
	case parseResolved2 := <-parseResolvedCh:
		return parseResolved2, nil
	case parseErr := <-parseRejectedCh:
		return js.Undefined(), parseErr
	case <-parseCtx.Done():
		parseCode := interop.CodeCancelled
		if errors.Is(parseCtx.Err(), context.DeadlineExceeded) {
			parseCode = interop.CodeTimeout
		}
		return js.Undefined(), &interop.Error{Op: parseOp, Target: parseTarget, Code: parseCode, Err: parseCtx.Err()}
	}
}

// cacheStorageUnavailable builds one consistent cache storage unavailable error.
func cacheStorageUnavailable(parseOp string, parseTarget string) error {
	return &interop.Error{Op: parseOp, Target: parseTarget, Code: interop.CodeUnavailable, Err: errors.New("cache storage helpers are unavailable in this build")}
}
