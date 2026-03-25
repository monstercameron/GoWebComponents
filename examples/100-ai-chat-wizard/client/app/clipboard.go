//go:build js && wasm

package app

import (
	"context"

	"github.com/monstercameron/GoWebComponents/interop"
)

// copyToClipboard writes text to the browser clipboard via navigator.clipboard.writeText.
// Errors are silently ignored; the function is best-effort.
func copyToClipboard(text string) {
	cb, err := interop.NavigatorClipboard()
	if err != nil {
		return
	}
	_ = cb.WriteText(context.Background(), text)
}
