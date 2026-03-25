//go:build !js || !wasm
// +build !js !wasm

package hotreload

import "testing"

func TestNativeHotReloadStubs(t *testing.T) {
	Disable()
	if Enabled() {
		t.Fatalf("expected disabled hotreload by default on native builds")
	}

	Enable()
	if Enabled() {
		t.Fatalf("expected enable to remain disabled on native builds")
	}

	Configure(Config{
		AtomIDs:  []string{"theme", "sidebar"},
		ResetKey: "build-v1",
	})
	if Enabled() {
		t.Fatalf("expected configure to remain disabled on native builds")
	}

	payload, err := GetSnapshot()
	if err != nil {
		t.Fatalf("expected native snapshot to return nil error, got %v", err)
	}
	if payload != "" {
		t.Fatalf("expected native snapshot payload to be empty, got %q", payload)
	}

	if err := ApplySnapshot(`{"state":{"theme":"dark"}}`); err != nil {
		t.Fatalf("expected native apply snapshot to be no-op, got %v", err)
	}

	Prepare()
	Disable()
	if Enabled() {
		t.Fatalf("expected disable to keep native hotreload disabled")
	}
}
