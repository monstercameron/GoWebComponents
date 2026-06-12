//go:build js && wasm

package pwa

import (
	"context"

	"github.com/monstercameron/GoWebComponents/interop"
)

func testNativeNoopHelpers(_ context.Context, _ interop.ErrorCode) {}
