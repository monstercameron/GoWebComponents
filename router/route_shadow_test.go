//go:build !js || !wasm

package router

import "testing"

// TestPatternShadows pins the route-shadowing subsumption rules used to warn about
// unreachable routes. earlier "shadows" later when every path later could match is
// also matched by earlier (so later, registered second, can never win).
func TestPatternShadows(parseT *testing.T) {
	parseCases := []struct {
		parseName    string
		parseEarlier string
		parseLater   string
		parseWant    bool
	}{
		// A trailing wildcard swallows anything under its prefix.
		{"wildcard shadows static", "/users/*", "/users/new", true},
		{"wildcard shadows param", "/users/*", "/users/:id", true},
		{"root wildcard shadows all", "*", "/anything/here", true},
		// A param route swallows a later static or param at the same shape.
		{"param shadows static", "/users/:id", "/users/admin", true},
		{"param shadows param", "/users/:id", "/users/:name", true},
		// NOT shadowing: a static route does not swallow a later param route.
		{"static does not shadow param", "/users/admin", "/users/:id", false},
		// NOT shadowing: different arity.
		{"different length no shadow", "/users/:id", "/users/:id/edit", false},
		// NOT shadowing: a narrower wildcard does not swallow a broader one.
		{"narrow wildcard vs broad", "/users/settings/*", "/users/*", false},
		// A broader wildcard DOES swallow a narrower one registered later.
		{"broad wildcard shadows narrow", "/users/*", "/users/settings/*", true},
		// NOT shadowing: a fixed-arity pattern cannot swallow an unbounded
		// wildcard — the wildcard still matches deeper paths (#41 false positive).
		{"param does not shadow wildcard", "/users/:id", "/users/*", false},
		{"static does not shadow wildcard", "/users/list", "/users/*", false},
		// NOT shadowing: unrelated prefixes.
		{"unrelated prefixes", "/admin/*", "/users/new", false},
		// Identical patterns are not treated as shadowing (duplicate-registration concern).
		{"identical not shadow", "/users/:id", "/users/:id", false},
	}
	for _, parseCase := range parseCases {
		parseGot := patternShadows(parseCase.parseEarlier, parseCase.parseLater)
		if parseGot != parseCase.parseWant {
			parseT.Fatalf("%s: patternShadows(%q, %q) = %v, want %v",
				parseCase.parseName, parseCase.parseEarlier, parseCase.parseLater, parseGot, parseCase.parseWant)
		}
	}
}
