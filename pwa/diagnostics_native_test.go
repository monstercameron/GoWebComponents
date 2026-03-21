package pwa

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestInspectDiagnosticsReportsUnavailableOnNativeBuilds(t *testing.T) {
	_, err := InspectDiagnostics(context.Background(), DiagnosticsOptions{})
	if !interop.IsCode(err, interop.CodeUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}
