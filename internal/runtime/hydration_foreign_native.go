//go:build !(js && wasm)

package runtime

// normalizeForeignPropertyValue is the native no-op twin of the wasm version:
// native adapters (mockdom, test doubles) already return plain Go values.
func normalizeForeignPropertyValue(parseValue any) any {
	return parseValue
}
