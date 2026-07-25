//go:build !js || !wasm

package pwa

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// OpenCacheStorageManager is a non-browser stub that always returns an unavailable error.
func OpenCacheStorageManager() (CacheStorageManager, error) {
	return CacheStorageManager{}, cacheStorageUnavailable("OpenCacheStorageManager", "caches")
}

func cacheStorageUnavailable(parseCacheOp string, parseCacheTarget string) error {
	return &interop.Error{Op: parseCacheOp, Target: parseCacheTarget, Code: interop.CodeUnavailable, Err: errors.New("cache storage helpers are unavailable in this build")}
}

func _cacheStorageContext(_ context.Context) {}
