//go:build js && wasm
// +build js,wasm

package utils

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/hotreload"
)

func TestEnableHotReloadIsDisabledInProduction(parseT *testing.T) {
	if !productionBuildForTests {
		parseT.Skip("production-only test")
	}

	EnableHotReload(true)

	if IsHotReloadEnabled() {
		parseT.Fatal("expected hot reload to remain disabled in production")
	}
	if hotreload.IsEnabled() {
		parseT.Fatal("expected hotreload package to remain disabled in production")
	}
}
