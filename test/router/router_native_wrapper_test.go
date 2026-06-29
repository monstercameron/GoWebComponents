//go:build !js || !wasm

package routertest

import "testing"

func TestNewHashPanicsWithNilTestingTB(parseT *testing.T) {
	defer func() {
		if recover() == nil {
			parseT.Fatalf("expected panic when NewHash receives nil testing.TB")
		}
	}()
	_ = NewHash(nil)
}

func TestNewHistoryPanicsWithNilTestingTB(parseT *testing.T) {
	defer func() {
		if recover() == nil {
			parseT.Fatalf("expected panic when NewHistory receives nil testing.TB")
		}
	}()
	_ = NewHistory(nil)
}
