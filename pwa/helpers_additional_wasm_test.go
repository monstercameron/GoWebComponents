//go:build js && wasm

package pwa

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v7/interop"
)

func testNativeNoopHelpers(_ context.Context, _ interop.ErrorCode) {}
