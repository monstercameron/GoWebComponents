//go:build js && wasm

package runtime

import (
	"syscall/js"
	"testing"
)

// Pins the wasm-side unwrapping of js.Value property reads for hydration: the
// browser adapter's GetProperty returns raw js.Value, which the type switches
// in normalizeHydrationInt/stringifyHydrationValue cannot see through.
func TestHydrationNormalizesJSValueProperties(parseT *testing.T) {
	if parseValue, parseOk := normalizeHydrationInt(normalizeForeignPropertyValue(js.ValueOf(1))); !parseOk || parseValue != 1 {
		parseT.Fatalf("nodeType js.Value(1) normalized to %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationString(normalizeForeignPropertyValue(js.ValueOf("DIV"))); !parseOk || parseValue != "DIV" {
		parseT.Fatalf("tagName js.Value(\"DIV\") normalized to %q, %t", parseValue, parseOk)
	}
	if parseNormalized := normalizeForeignPropertyValue(js.Undefined()); parseNormalized != nil {
		parseT.Fatalf("undefined should normalize to nil, got %#v", parseNormalized)
	}
	if parseNormalized := normalizeForeignPropertyValue(js.Null()); parseNormalized != nil {
		parseT.Fatalf("null should normalize to nil, got %#v", parseNormalized)
	}
	if parseNormalized := normalizeForeignPropertyValue(js.ValueOf(true)); parseNormalized != true {
		parseT.Fatalf("boolean should normalize to Go bool, got %#v", parseNormalized)
	}
	if parseText := stringifyHydrationValue(normalizeForeignPropertyValue(js.ValueOf(2))); parseText != "2" {
		parseT.Fatalf("numeric property should stringify as \"2\", got %q", parseText)
	}
}
