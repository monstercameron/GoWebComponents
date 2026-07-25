//go:build !js || !wasm

package pwa

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

func TestOpenCacheStorageManagerReportsUnavailableOnNativeBuilds(parseT *testing.T) {
	_, parseErr := OpenCacheStorageManager()
	if !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("expected unavailable error, got %v", parseErr)
	}
}
