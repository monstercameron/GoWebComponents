//go:build !js || !wasm
// +build !js !wasm

package hooks

import "testing"

func TestNativeHarnessStubMethods(t *testing.T) {
	h := &Harness[int]{}
	if got := h.Current(); got != 0 {
		t.Fatalf("expected zero-value current state in native stub, got %d", got)
	}

	called := false
	h.Rerender()
	h.Flush()
	h.Act(func() {
		called = true
	})
	h.Cleanup()

	if called {
		t.Fatalf("expected native stub Act to no-op without executing callback")
	}
}
