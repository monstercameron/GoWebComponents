//go:build !js || !wasm
// +build !js !wasm

package pwa

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestInspectDiagnosticsReportsUnavailableOnNativeBuilds(parseT *testing.T) {
	_, parseErr := InspectDiagnostics(context.Background(), DiagnosticsOptions{})
	if !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("expected unavailable error, got %v", parseErr)
	}
}
