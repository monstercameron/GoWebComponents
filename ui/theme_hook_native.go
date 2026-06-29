//go:build !(js && wasm)

package ui

// applyThemeAttribute is a no-op on non-browser builds (no document to mark).
func applyThemeAttribute(parseName string) {}
