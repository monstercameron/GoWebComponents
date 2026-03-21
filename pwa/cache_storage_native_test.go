package pwa

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestOpenCacheStorageManagerReportsUnavailableOnNativeBuilds(t *testing.T) {
	_, err := OpenCacheStorageManager()
	if !interop.IsCode(err, interop.CodeUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}
