//go:build !js || !wasm

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestRunUnsupportedOnServerPanicUsesUnifiedContract verifies ui.Run mirrors
// ui.Render on the native/SSR slice: it is browser-only and panics with the
// unified unsupported-on-server contract instead of mounting.
func TestRunUnsupportedOnServerPanicUsesUnifiedContract(parseT *testing.T) {
	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseT.Fatal("expected Run to panic on server")
		}
		parseMessage, parseOk := parseRecovered.(string)
		if !parseOk {
			parseT.Fatalf("expected string panic, got %T", parseRecovered)
		}
		if !strings.Contains(parseMessage, "GWC-UI-UNSUPPORTED-ON-SERVER") ||
			!strings.Contains(parseMessage, "ui.Run") ||
			!strings.Contains(parseMessage, "docs: ACTIONABLE_ERRORS.md#gwc-ui-unsupported-on-server") {
			parseT.Fatalf("unexpected unsupported-on-server message: %q", parseMessage)
		}
	}()

	ui.Run("#app", func() ui.Node { return ui.Text("hello") })
}
