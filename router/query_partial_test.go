//go:build js && wasm

package router

import "testing"

// TestParseNavigationTargetKeepsPartialQuery pins that a malformed query param
// does not discard the valid params parsed alongside it (url.ParseQuery returns
// the good pairs plus an error; the router must keep the good pairs).
func TestParseNavigationTargetKeepsPartialQuery(parseT *testing.T) {
	parsePath, parseQuery := parseNavigationTarget("/search?q=golang&tag=%zz")
	if parsePath != "/search" {
		parseT.Fatalf("path = %q, want /search", parsePath)
	}
	if parseGot := parseQuery.Get("q"); parseGot != "golang" {
		parseT.Fatalf("q = %q, want golang (valid param dropped due to a malformed sibling)", parseGot)
	}
}
