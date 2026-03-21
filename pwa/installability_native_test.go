package pwa

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestObserveInstallabilityReportsUnavailableOnNativeBuilds(t *testing.T) {
	_, err := ObserveInstallability(InstallabilityOptions{})
	if !interop.IsCode(err, interop.CodeUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}
