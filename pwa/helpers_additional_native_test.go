//go:build !js || !wasm

package pwa

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v4/interop"
)

func testNativeNoopHelpers(parseCtx context.Context, parseCode interop.ErrorCode) {
	_cacheStorageContext(parseCtx)
	_diagnosticsNativeInterop(parseCode)
}
