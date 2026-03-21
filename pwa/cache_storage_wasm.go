//go:build js && wasm
// +build js,wasm

package pwa

import (
	"context"
	"errors"
	"sort"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/interop"
)

func OpenCacheStorageManager() (CacheStorageManager, error) {
	caches := js.Global().Get("caches")
	if caches.IsUndefined() || caches.IsNull() {
		return CacheStorageManager{}, cacheStorageUnavailable("OpenCacheStorageManager", "caches")
	}
	return CacheStorageManager{
		sync: func(ctx context.Context, plan CacheStoragePlan) (CacheStorageSnapshot, error) {
			if ctx == nil {
				ctx = context.Background()
			}
			if strings.TrimSpace(plan.CacheName) == "" {
				return CacheStorageSnapshot{}, &interop.Error{Op: "CacheStorageManager.Sync", Code: interop.CodeInvalid, Err: errors.New("cache name is empty")}
			}
			names, err := cacheStorageKeys(ctx, caches)
			if err != nil {
				return CacheStorageSnapshot{}, err
			}
			for _, name := range names {
				if name == plan.CacheName {
					continue
				}
				if plan.CachePrefix != "" && strings.HasPrefix(name, plan.CachePrefix) {
					if _, err := awaitCacheStorageValue(ctx, "CacheStorageManager.Sync", name, caches.Call("delete", name)); err != nil {
						return CacheStorageSnapshot{}, err
					}
				}
			}
			cacheValue, err := awaitCacheStorageValue(ctx, "CacheStorageManager.Sync", plan.CacheName, caches.Call("open", plan.CacheName))
			if err != nil {
				return CacheStorageSnapshot{}, err
			}
			for _, entry := range plan.Entries {
				if _, err := awaitCacheStorageValue(ctx, "CacheStorageManager.Sync", entry.URL, cacheValue.Call("add", entry.URL)); err != nil {
					return CacheStorageSnapshot{}, err
				}
			}
			return inspectCacheStorage(ctx, caches, plan)
		},
		inspect: func(ctx context.Context, plan CacheStoragePlan) (CacheStorageSnapshot, error) {
			if ctx == nil {
				ctx = context.Background()
			}
			return inspectCacheStorage(ctx, caches, plan)
		},
	}, nil
}

func inspectCacheStorage(ctx context.Context, caches js.Value, plan CacheStoragePlan) (CacheStorageSnapshot, error) {
	names, err := cacheStorageKeys(ctx, caches)
	if err != nil {
		return CacheStorageSnapshot{}, err
	}
	cacheValue, err := awaitCacheStorageValue(ctx, "CacheStorageManager.Inspect", plan.CacheName, caches.Call("open", plan.CacheName))
	if err != nil {
		return CacheStorageSnapshot{}, err
	}
	keysValue, err := awaitCacheStorageValue(ctx, "CacheStorageManager.Inspect", plan.CacheName, cacheValue.Call("keys"))
	if err != nil {
		return CacheStorageSnapshot{}, err
	}
	entriesByURL := map[string]CacheStorageEntry{}
	for _, entry := range plan.Entries {
		entriesByURL[entry.URL] = entry
	}
	entries := make([]CacheStorageEntry, 0, keysValue.Length())
	for i := 0; i < keysValue.Length(); i++ {
		request := keysValue.Index(i)
		url := strings.TrimSpace(request.Get("url").String())
		entry, ok := entriesByURL[url]
		if !ok {
			entry = CacheStorageEntry{URL: url, Kind: CacheStorageAssetKindAsset, Strategy: CacheStorageStrategyCacheFirst}
		}
		entries = append(entries, entry)
	}
	sort.Strings(names)
	return CacheStorageSnapshot{CacheName: plan.CacheName, CachePrefix: plan.CachePrefix, EntryCount: len(entries), Entries: entries, CacheNames: names}, nil
}

func cacheStorageKeys(ctx context.Context, caches js.Value) ([]string, error) {
	value, err := awaitCacheStorageValue(ctx, "CacheStorageManager.Keys", "caches.keys", caches.Call("keys"))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, value.Length())
	for i := 0; i < value.Length(); i++ {
		names = append(names, strings.TrimSpace(value.Index(i).String()))
	}
	return names, nil
}

func awaitCacheStorageValue(ctx context.Context, op string, target string, value js.Value) (js.Value, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	then := value.Get("then")
	if then.Type() != js.TypeFunction {
		return value, nil
	}
	resolvedCh := make(chan js.Value, 1)
	rejectedCh := make(chan error, 1)
	var resolveFn js.Func
	var rejectFn js.Func
	cleanup := func() {
		resolveFn.Release()
		rejectFn.Release()
	}
	resolveFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolved := js.Undefined()
		if len(args) > 0 {
			resolved = args[0]
		}
		select {
		case resolvedCh <- resolved:
		default:
		}
		return nil
	})
	rejectFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		message := "cache storage promise rejected"
		if len(args) > 0 {
			if text := strings.TrimSpace(args[0].String()); text != "" {
				message = text
			}
		}
		select {
		case rejectedCh <- &interop.Error{Op: op, Target: target, Code: interop.CodePromiseRejected, Err: errors.New(message)}:
		default:
		}
		return nil
	})
	value.Call("then", resolveFn).Call("catch", rejectFn)
	defer cleanup()

	select {
	case resolved := <-resolvedCh:
		return resolved, nil
	case err := <-rejectedCh:
		return js.Undefined(), err
	case <-ctx.Done():
		code := interop.CodeCancelled
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			code = interop.CodeTimeout
		}
		return js.Undefined(), &interop.Error{Op: op, Target: target, Code: code, Err: ctx.Err()}
	}
}

func cacheStorageUnavailable(op string, target string) error {
	return &interop.Error{Op: op, Target: target, Code: interop.CodeUnavailable, Err: errors.New("cache storage helpers are unavailable in this build")}
}
