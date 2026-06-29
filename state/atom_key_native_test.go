//go:build !(js && wasm)

package state

import "testing"

// TestAtomKeyIDAndDefault proves the key centralizes the id and default in one declaration.
func TestAtomKeyIDAndDefault(parseT *testing.T) {
	parseKey := NewAtomKey("test:key:meta", "light")
	if parseKey.ID() != "test:key:meta" {
		parseT.Fatalf("ID() = %q, want test:key:meta", parseKey.ID())
	}
	if parseKey.Default() != "light" {
		parseT.Fatalf("Default() = %q, want light", parseKey.Default())
	}
}

// TestAtomKeyGlobalRoundTripsAndSharesID proves key.Global() seeds the default, round-trips a
// write, and addresses the SAME atom as a NewGlobalAtom with the same id (the key is just a typed,
// declared-once front for that id).
func TestAtomKeyGlobalRoundTripsAndSharesID(parseT *testing.T) {
	parseKey := NewAtomKey("test:key:global", "idle")
	parseHandle := parseKey.Global()
	if parseGot := parseHandle.Get(); parseGot != "idle" {
		parseT.Fatalf("Global().Get() = %q, want seeded default idle", parseGot)
	}
	parseHandle.Set("busy")
	if parseGot := parseHandle.Get(); parseGot != "busy" {
		parseT.Fatalf("after Set, Global().Get() = %q, want busy", parseGot)
	}

	// A plain GlobalAtom for the same id observes the key's write — same underlying atom.
	parseOther := NewGlobalAtom("test:key:global", "ignored-default")
	if parseGot := parseOther.Get(); parseGot != "busy" {
		parseT.Fatalf("key.Global() and NewGlobalAtom must address the same atom; got %q", parseGot)
	}
}

// TestUseAtomKeySharesAtomWithKeyGlobal proves the hook path: UseAtomKey subscribes to the key's
// atom, and the key's Global() handle observes the same value (one logical atom, one key).
func TestUseAtomKeySharesAtomWithKeyGlobal(parseT *testing.T) {
	installStateNativeHookContext(parseT)

	parseKey := NewAtomKey("test:key:use", 2)
	parseAtom := UseAtomKey(parseKey)
	parseAtom.Set(3)
	parseAtom.Update(func(parsePrev int) int { return parsePrev + 1 })
	if parseGot := parseAtom.Get(); parseGot != 4 {
		parseT.Fatalf("UseAtomKey atom should update to 4, got %d", parseGot)
	}
	if parseGot := parseKey.Global().Get(); parseGot != 4 {
		parseT.Fatalf("key.Global() should observe the UseAtomKey write (4) via the shared id, got %d", parseGot)
	}
}

// TestUseAtomKeyObservesPreRenderWrite proves the G39 guarantee through the key wrapper: a value
// written via key.Global() BEFORE the component's UseAtomKey runs survives — UseAtomKey reads the
// pre-written value, not the key's default (the seed never clobbers an existing write).
func TestUseAtomKeyObservesPreRenderWrite(parseT *testing.T) {
	installStateNativeHookContext(parseT)

	parseKey := NewAtomKey("test:key:prewrite", "default")
	parseKey.Global().Set("written-before-render") // pre-render write

	parseAtom := UseAtomKey(parseKey)
	if parseGot := parseAtom.Get(); parseGot != "written-before-render" {
		parseT.Fatalf("UseAtomKey must observe a pre-render Global write (G39), got %q", parseGot)
	}
}

// TestAtomKeySameIDSharesState documents that AtomKey centralizes a contract, it does not namespace
// ids: two keys built with the same id address the same atom (so share one key var per atom).
func TestAtomKeySameIDSharesState(parseT *testing.T) {
	parseA := NewAtomKey("test:key:samedid", 0)
	parseB := NewAtomKey("test:key:samedid", 0)
	parseA.Global().Set(42)
	if parseGot := parseB.Global().Get(); parseGot != 42 {
		parseT.Fatalf("same-id keys must share state, got %d", parseGot)
	}
}
