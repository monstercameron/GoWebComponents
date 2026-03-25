//go:build !js || !wasm
// +build !js !wasm

package routertest

import "testing"

func TestNewHashPanicsWithNilTestingTB(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic when NewHash receives nil testing.TB")
		}
	}()
	_ = NewHash(nil)
}

func TestNewHistoryPanicsWithNilTestingTB(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic when NewHistory receives nil testing.TB")
		}
	}()
	_ = NewHistory(nil)
}

