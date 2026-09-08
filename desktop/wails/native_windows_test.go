//go:build windows

package wails

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/desktop"
)

// TestNativeBackendMissingCallerNeverUsesFocusedWindow verifies every adapter operation requires its caller.
func TestNativeBackendMissingCallerNeverUsesFocusedWindow(parseTest *testing.T) {
	parseBackend := NewNativeBackend()
	if _, parseErr := parseBackend.ClipboardRead(context.Background()); parseErr == nil {
		parseTest.Fatal("clipboard read accepted missing caller")
	}
	if parseErr := parseBackend.ClipboardWrite(context.Background(), desktop.ClipboardWriteRequest{Text: "x"}); parseErr == nil {
		parseTest.Fatal("clipboard write accepted missing caller")
	}
	if _, parseErr := parseBackend.ShowMessage(context.Background(), desktop.MessageRequest{Kind: "info"}); parseErr == nil {
		parseTest.Fatal("message accepted missing caller")
	}
	if _, parseErr := parseBackend.Window(context.Background(), desktop.WindowRequest{Action: "info"}); parseErr == nil {
		parseTest.Fatal("window accepted missing caller")
	}
	if _, parseErr := parseBackend.Screens(context.Background()); parseErr == nil {
		parseTest.Fatal("screens accepted missing caller")
	}
}
