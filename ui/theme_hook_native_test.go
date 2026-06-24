//go:build !(js && wasm)

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

// TestSetAndCurrentThemeRoundTrip proves the non-hook theme accessors read and
// write the shared theme state from outside a render (G: theming hooks).
func TestSetAndCurrentThemeRoundTrip(parseT *testing.T) {
	ui.SetTheme("dark")
	if parseGot := ui.CurrentTheme(); parseGot != "dark" {
		parseT.Fatalf("CurrentTheme = %q, want dark", parseGot)
	}
	ui.SetTheme("light")
	if parseGot := ui.CurrentTheme(); parseGot != "light" {
		parseT.Fatalf("CurrentTheme = %q, want light", parseGot)
	}
	// Clearing reverts to empty.
	ui.SetTheme("")
	if parseGot := ui.CurrentTheme(); parseGot != "" {
		parseT.Fatalf("CurrentTheme after clear = %q, want empty", parseGot)
	}
}

// TestSetThemeNativeDoesNotPanic proves applying the theme attribute is a safe
// no-op on native/SSR (no document).
func TestSetThemeNativeDoesNotPanic(parseT *testing.T) {
	ui.SetTheme("contrast")
	if ui.CurrentTheme() != "contrast" {
		parseT.Fatal("expected theme to persist on native")
	}
}
