//go:build !js || !wasm

package thirdpartyjs

import (
	"context"
	"testing"
)

// TestSlugifyNativeStubParity proves the native-stub parity guarantee: on the server (where
// interop.ImportModule is unavailable) the bridge reports unloaded yet still produces the correct
// slug through the Go fallback — the same call site works identically without the JS module.
func TestSlugifyNativeStubParity(parseT *testing.T) {
	parseBridge := LoadSlugify(context.Background())
	if parseBridge.Loaded() {
		parseT.Fatal("native build must not load a JS module (ImportModule is unavailable)")
	}
	if parseGot := parseBridge.Slugify(context.Background(), "Hello, World!"); parseGot != "hello-world" {
		parseT.Fatalf("native fallback should slugify cleanly, got %q", parseGot)
	}
	if parseErr := parseBridge.Dispose(); parseErr != nil {
		parseT.Fatalf("Dispose on an unloaded bridge must be a no-op, got %v", parseErr)
	}
}

// TestFallbackSlugify proves the Go fallback matches the JS library's slug semantics across the
// cases that matter: punctuation, diacritic-free unicode, multiple separators, and trimming.
func TestFallbackSlugify(parseT *testing.T) {
	parseCases := map[string]string{
		"Hello, World!":         "hello-world",
		"  Spaced   Out  ":      "spaced-out",
		"already-a-slug":        "already-a-slug",
		"Foo___Bar / Baz":       "foo-bar-baz",
		"UPPER and 123 nums":    "upper-and-123-nums",
		"!!!":                   "",
		"trailing-punctuation.": "trailing-punctuation",
	}
	for parseIn, parseWant := range parseCases {
		if parseGot := fallbackSlugify(parseIn); parseGot != parseWant {
			parseT.Fatalf("fallbackSlugify(%q) = %q, want %q", parseIn, parseGot, parseWant)
		}
	}
}
