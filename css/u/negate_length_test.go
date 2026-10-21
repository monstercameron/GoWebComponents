package u

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/css"
)

// TestNegateLengthProducesValidCSS pins that negating a spacing value never emits
// a double minus and wraps non-numeric values in calc().
func TestNegateLengthProducesValidCSS(parseT *testing.T) {
	parseCases := map[css.Length]string{
		"1rem":          "-1rem",
		"-1rem":         "1rem",
		"0":             "0",
		"":              "0",
		".5rem":         "-.5rem",
		"var(--gap)":    "calc(-1 * var(--gap))",
		"calc(1px+2px)": "calc(-1 * calc(1px+2px))",
	}
	for parseIn, parseWant := range parseCases {
		if parseGot := negateLength(parseIn); parseGot != parseWant {
			parseT.Errorf("negateLength(%q) = %q, want %q", parseIn, parseGot, parseWant)
		}
	}
}
