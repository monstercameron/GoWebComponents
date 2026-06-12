//go:build !js || !wasm

package pwa

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestObserveInstallabilityReportsUnavailableOnNativeBuilds(parseT *testing.T) {
	_, parseErr := ObserveInstallability(InstallabilityOptions{})
	if !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseT.Fatalf("expected unavailable error, got %v", parseErr)
	}
}
