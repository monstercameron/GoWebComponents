//go:build !js || !wasm
// +build !js !wasm

package pwa

import (
	"context"

	"github.com/monstercameron/GoWebComponents/interop"
)

func InspectDiagnostics(ctx context.Context, options DiagnosticsOptions) (DiagnosticsSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	_ = ctx
	_ = options
	return DiagnosticsSnapshot{}, installabilityUnavailable("InspectDiagnostics", "window")
}

func _diagnosticsNativeInterop(_ interop.ErrorCode) {}
