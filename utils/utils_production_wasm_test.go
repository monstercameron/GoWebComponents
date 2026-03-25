//go:build js && wasm
// +build js,wasm

package utils

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/hotreload"
)

func TestEnableHotReloadIsDisabledInProduction(t *testing.T) {
	if !productionBuildForTests {
		t.Skip("production-only test")
	}

	EnableHotReload(true)

	if IsHotReloadEnabled() {
		t.Fatal("expected hot reload to remain disabled in production")
	}
	if hotreload.IsEnabled() {
		t.Fatal("expected hotreload package to remain disabled in production")
	}
}
