//go:build js && wasm

package interop

import (
	"context"
	"syscall/js"
	"testing"
)

// TestAwaitRejectNonErrorValues probes rejection with non-Error payloads (string,
// number, bare reject) — common in real JS — to ensure Await returns an error
// (not a panic, not a nil error/zero value).
func TestAwaitRejectNonErrorValues(parseT *testing.T) {
	parseCases := []struct {
		parseName string
		parseMake func() js.Value
	}{
		{"string", func() js.Value { return js.Global().Get("Promise").Call("reject", "boom") }},
		{"number", func() js.Value { return js.Global().Get("Promise").Call("reject", 42) }},
		{"bare-reject", func() js.Value {
			parseExec := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
				if len(parseArgs) > 1 {
					parseArgs[1].Invoke() // reject() with no argument
				}
				return nil
			})
			return js.Global().Get("Promise").New(parseExec)
		}},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.parseName, func(parseT *testing.T) {
			parseValue := Value{raw: parseCase.parseMake()}
			_, parseErr := parseValue.Await(context.Background())
			if parseErr == nil {
				parseT.Fatalf("%s rejection: expected error, got nil", parseCase.parseName)
			}
		})
	}
}
