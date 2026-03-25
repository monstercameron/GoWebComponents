//go:build js && wasm && production
// +build js,wasm,production

package hotreload

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestEnableIsDisabledInProduction(t *testing.T) {
	global, err := interop.GetGlobalThis()
	if err != nil {
		t.Fatalf("expected browser global, got %v", err)
	}
	prevBridge := global.Get(appBridgeGlobal)
	t.Cleanup(func() {
		if prevBridge.Present() {
			_ = global.Set(appBridgeGlobal, prevBridge)
		} else {
			_ = global.Delete(appBridgeGlobal)
		}
	})

	Enable()

	if Enabled() {
		t.Fatal("expected hot reload to remain disabled in production")
	}
	if global.Get(appBridgeGlobal).Present() {
		t.Fatal("expected hot reload bridge to stay unset in production")
	}
}
