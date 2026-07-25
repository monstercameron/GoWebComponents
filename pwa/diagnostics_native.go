//go:build !js || !wasm

package pwa

import (
	"context"
	"errors"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// InspectDiagnostics is a non-browser stub that always returns an unavailable error.
func InspectDiagnostics(parseDiagnosticsCtx context.Context, parseDiagnosticsOptions DiagnosticsOptions) (DiagnosticsSnapshot, error) {
	_ = parseDiagnosticsCtx
	_ = parseDiagnosticsOptions
	return DiagnosticsSnapshot{}, &interop.Error{
		Op:     "InspectDiagnostics",
		Target: "window",
		Code:   interop.CodeUnavailable,
		Err:    errors.New("diagnostics inspection is unavailable in this build"),
	}
}

func _diagnosticsNativeInterop(_ interop.ErrorCode) {}
