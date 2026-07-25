//go:build js && wasm

package hotreload

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/interop"
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

// TestProductionCompatibilityAPIsRemainNoOps verifies the remaining production compatibility APIs stay disabled and side-effect free.
func TestProductionCompatibilityAPIsRemainNoOps(parseT *testing.T) {
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
			return
		}
		_ = parseGlobal.Delete(productionBridgeGlobalKey)
	})

	Configure(Config{AtomIDs: []string{"theme", "sidebar"}, ResetKey: "build-v1"})
	if Enabled() || IsEnabled() {
		parseT.Fatal("expected production Configure to keep hot reload disabled")
	}
	if parseGlobal.Get(productionBridgeGlobalKey).Present() {
		parseT.Fatal("expected production Configure to avoid installing a bridge")
	}

	parsePayload, parseErr := GetSnapshot()
	if parseErr != nil {
		parseT.Fatalf("expected production GetSnapshot to stay side-effect free, got %v", parseErr)
	}
	if parsePayload != "" {
		parseT.Fatalf("expected production GetSnapshot to return an empty payload, got %q", parsePayload)
	}

	if parseErr = ApplySnapshot(`{"theme":"dark"}`); parseErr != nil {
		parseT.Fatalf("expected production ApplySnapshot to ignore payloads, got %v", parseErr)
	}

	Prepare()
	Disable()
	if Enabled() || IsEnabled() {
		parseT.Fatal("expected production Disable to keep hot reload disabled")
	}
}
