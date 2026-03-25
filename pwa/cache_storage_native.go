//go:build !js || !wasm
// +build !js !wasm

package pwa

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/interop"
)

// OpenCacheStorageManager is a non-browser stub that always returns an unavailable error.
func OpenCacheStorageManager() (CacheStorageManager, error) { "caches")
}

func cacheStorageUnavailable(op string, target string) error {
	return &interop.Error{Op: op, Target: target, Code: interop.CodeUnavailable, Err: errors.New("cache storage helpers are unavailable in this build")}
}

func _cacheStorageContext(_ context.Context) {}
