//go:build !windows

package wails

import (
	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"testing"
)

// TestUnsupportedPlatformDoesNotAdvertiseFiles keeps non-Windows claims honest.
func TestUnsupportedPlatformDoesNotAdvertiseFiles(parseT *testing.T) {
	if len(desktop.NewFileDialogHost(NewFileDialogs(), true).GetMethods()) != 0 {
		parseT.Fatal("unsupported platform advertised file dialogs")
	}
}
