//go:build js && wasm

package pwa

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v8/interop"
)

func testNativeNoopHelpers(_ context.Context, _ interop.ErrorCode) {}
