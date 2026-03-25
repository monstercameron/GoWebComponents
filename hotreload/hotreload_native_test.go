//go:build !js || !wasm
// +build !js !wasm

package hotreload

import "testing"

func TestNativeHotReloadStubs(parseT *testing.T) {
	Disable()
	if Enabled() {
		parseT.Fatalf("expected disabled hotreload by default on native builds")
	}

	Enable()
	if Enabled() {
		parseT.Fatalf("expected enable to remain disabled on native builds")
	}

	Configure(Config{
		AtomIDs:  []string{"theme", "sidebar"},
		ResetKey: "build-v1",
	})
	if Enabled() {
		parseT.Fatalf("expected configure to remain disabled on native builds")
	}

	parsePayload, parseErr := GetSnapshot()
	if parseErr != nil {
		parseT.Fatalf("expected native snapshot to return nil error, got %v", parseErr)
	}
	if parsePayload != "" {
		parseT.Fatalf("expected native snapshot payload to be empty, got %q", parsePayload)
	}

	if parseErr2 := ApplySnapshot(`{"state":{"theme":"dark"}}`); parseErr2 != nil {
		parseT.Fatalf("expected native apply snapshot to be no-op, got %v", parseErr2)
	}

	Prepare()
	Disable()
	if Enabled() {
		parseT.Fatalf("expected disable to keep native hotreload disabled")
	}
}
