//go:build !(js && wasm)

package css_test

import (
	"regexp"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/css"
)

// safeVar matches the only shape Var may ever produce: var(--<identifier chars>).
// Anything else (a stray ')', ';', '}', whitespace, etc.) would let an untrusted
// token name break out of the var() call or the surrounding declaration.
var safeVar = regexp.MustCompile(`^var\(--[A-Za-z0-9_-]*\)$`)

// FuzzVarInjectionSafe proves css.Var sanitizes any input to a safe custom-
// property reference, never panicking and never emitting a breakout character.
func FuzzVarInjectionSafe(parseF *testing.F) {
	for _, parseSeed := range []string{
		"", "accent", "--accent", "a)b", "x;color:red", "y}body{display:none",
		"a b", "\x00", "café", "id\U0001F4A5", "--", "a)", ")",
	} {
		parseF.Add(parseSeed)
	}
	parseF.Fuzz(func(parseT *testing.T, parseIn string) {
		parseOut := string(css.Var(parseIn))
		if !safeVar.MatchString(parseOut) {
			parseT.Fatalf("Var(%q) = %q is not a safe var() reference", parseIn, parseOut)
		}
	})
}
