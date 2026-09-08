//go:build !js || !wasm

package css

// resetFastFoldTest resets the native CSS registry and buffer sink.
func resetFastFoldTest() { Reset() }

// getFastFoldStyleBlock reads the native test sink output.
func getFastFoldStyleBlock() string { return StyleBlock() }
