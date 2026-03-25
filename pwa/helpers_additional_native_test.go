//go:build !js || !wasm
// +build !js !wasm

package pwa

import (
	"context"

	"github.com/monstercameron/GoWebComponents/interop"
)

func testNativeNoopHelpers(ctx context.Context, code interop.ErrorCode) {
	_cacheStorageContext(ctx)
	_diagnosticsNativeInterop(code)
}
