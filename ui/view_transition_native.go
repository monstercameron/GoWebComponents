//go:build !(js && wasm)

package ui

// ViewTransition calls apply directly on non-browser builds (no View Transitions
// API). See the wasm implementation for the animated path.
func ViewTransition(parseApply func()) {
	if parseApply != nil {
		parseApply()
	}
}
