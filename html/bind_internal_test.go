//go:build !(js && wasm)

package html

import "testing"

// TestBindInputHandlerWritesThroughSetter proves the BindTo/BindFunc oninput
// callback forwards the input value to the setter (the two-way write path).
func TestBindInputHandlerWritesThroughSetter(parseT *testing.T) {
	parseGot := "UNSET"
	bindInputHandler(func(parseValue string) { parseGot = parseValue })("typed-value")
	if parseGot != "typed-value" {
		parseT.Fatalf("expected setter to receive 'typed-value', got %q", parseGot)
	}
}

// TestBindInputHandlerNilSetterIsSafe proves a nil setter makes input a no-op
// rather than panicking when the handler fires.
func TestBindInputHandlerNilSetterIsSafe(parseT *testing.T) {
	bindInputHandler(nil)("anything") // must not panic
}
