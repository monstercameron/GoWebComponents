//go:build !(js && wasm)

package ui

// defaultReadOnlineStatus assumes connectivity on non-browser builds.
func defaultReadOnlineStatus() bool { return true }
