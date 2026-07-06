//go:build js && wasm

package runtime

import "syscall/js"

// normalizeForeignPropertyValue unwraps a js.Value returned by the browser DOM
// adapter's GetProperty into the plain Go value hydration's comparison helpers
// expect. Without this every browser hydration pass read nodeType as a js.Value
// (which normalizeHydrationInt cannot see through), failed the first node
// match, and silently fell back to a full client render — discarding the
// server-rendered DOM. Native tests never caught it because mockdom returns Go
// values directly.
//
// undefined/null map to nil so absent properties read as "not comparable"
// instead of stringifying to garbage and producing false mismatch diagnostics.
func normalizeForeignPropertyValue(parseValue any) any {
	parseJS, parseOk := parseValue.(js.Value)
	if !parseOk {
		return parseValue
	}
	switch parseJS.Type() {
	case js.TypeString:
		return parseJS.String()
	case js.TypeNumber:
		return parseJS.Float()
	case js.TypeBoolean:
		return parseJS.Bool()
	case js.TypeNull, js.TypeUndefined:
		return nil
	default:
		// Objects/functions have no meaningful attribute comparison form.
		return nil
	}
}
