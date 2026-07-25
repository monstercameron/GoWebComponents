//go:build js && wasm

package app

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// copyToClipboard writes text to the browser clipboard via navigator.clipboard.writeText.
// Errors are silently ignored; the function is best-effort.
func parseCopyToClipboard(parseText string) {
	parseCb, parseErr := interop.GetClipboard()
	if parseErr != nil {
		return
	}
	_ = parseCb.WriteText(context.Background(), parseText)
}
