//go:build !js || !wasm
// +build !js !wasm

package pwa

import (
	"context"

	"github.com/monstercameron/GoWebComponents/interop"
)

// InspectDiagnostics is a non-browser stub that always returns an unavailable error.
func InspectDiagnostics(ctx context.Context, options DiagnosticsOptions) (DiagnosticsSnapshot, error) {"InspectDiagnostics", "window")
}

func _diagnosticsNativeInterop(_ interop.ErrorCode) {}
