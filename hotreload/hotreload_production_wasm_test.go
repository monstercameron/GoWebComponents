//go:build js && wasm
// +build js,wasm

package hotreload

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

const productionBridgeGlobalKey = "GoWebComponentsHotReloadApp"

func TestEnableIsDisabledInProduction(parseT *testing.T) {
	if !productionBuildForTests {
		parseT.Skip("production-only test")
	}

	parseGlobal, parseErr := interop.GetGlobalThis()
	if parseErr != nil {
		parseT.Fatalf("expected browser global, got %v", parseErr)
	}
	parsePrevBridge := parseGlobal.Get(productionBridgeGlobalKey)
	parseT.Cleanup(func() {
		if parsePrevBridge.Present() {
			_ = parseGlobal.Set(productionBridgeGlobalKey, parsePrevBridge)
		} else {
			_ = parseGlobal.Delete(productionBridgeGlobalKey)
		}
	})

	Enable()

	if Enabled() {
		parseT.Fatal("expected hot reload to remain disabled in production")
	}
	if parseGlobal.Get(productionBridgeGlobalKey).Present() {
		parseT.Fatal("expected hot reload bridge to stay unset in production")
	}
}
