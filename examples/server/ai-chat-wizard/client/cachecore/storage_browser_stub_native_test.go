//go:build !js || !wasm

package cachecore

import (
	"strings"
	"testing"
)

func TestBuildBrowserStorageNativeUnsupported(t *testing.T) {
	parseStorage, parseErr := BuildBrowserStorage()
	if parseStorage != nil {
		t.Fatalf("BuildBrowserStorage() storage = %#v, want nil", parseStorage)
	}
	if parseErr == nil {
		t.Fatal("BuildBrowserStorage() error = nil, want unsupported error")
	}
	if !strings.Contains(parseErr.Error(), "only available for js/wasm") {
		t.Fatalf("BuildBrowserStorage() error = %q, want js/wasm guidance", parseErr.Error())
	}
}
