//go:build !(js && wasm)

package router_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/router"
)

// TestHrefNativePassthrough proves Href returns the path unchanged on native
// (no active router mode to resolve against) (G7).
func TestHrefNativePassthrough(parseT *testing.T) {
	if parseGot := router.Href("/x"); parseGot != "/x" {
		parseT.Fatalf("native Href = %q, want /x", parseGot)
	}
}

// TestFragmentHrefNative proves the native FragmentHref returns a bare fragment
// and tolerates a leading '#'.
func TestFragmentHrefNative(parseT *testing.T) {
	if parseGot := router.FragmentHref("main"); parseGot != "#main" {
		parseT.Fatalf("native FragmentHref = %q, want #main", parseGot)
	}
	if parseGot := router.FragmentHref("#main"); parseGot != "#main" {
		parseT.Fatalf("native FragmentHref('#main') = %q, want #main", parseGot)
	}
}
