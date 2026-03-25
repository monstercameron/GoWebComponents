//go:build js && wasm
// +build js,wasm

package hotreload

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

const productionBridgeGlobalKey = "GoWebComponentsHotReloadApp"

func TestEnableIsDisabledInProduction(t *testing.T) {
	if !productionBuildForTests {
		t.Skip("production-only test")
	}

	global, err := interop.GetGlobalThis()
	if err != nil {
		t.Fatalf("expected browser global, got %v", err)
	}
	prevBridge := global.Get(productionBridgeGlobalKey)
	t.Cleanup(func() {
		if prevBridge.Present() {
			_ = global.Set(productionBridgeGlobalKey, prevBridge)
		} else {
			_ = global.Delete(productionBridgeGlobalKey)
		}
	})

	Enable()

	if Enabled() {
		t.Fatal("expected hot reload to remain disabled in production")
	}
	if global.Get(productionBridgeGlobalKey).Present() {
		t.Fatal("expected hot reload bridge to stay unset in production")
	}
}
