package services

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// TestAPIEditMenuSingleShortcutOwner prevents native/WebView double editing.
func TestAPIEditMenuSingleShortcutOwner(parseTest *testing.T) {
	parseMenu := application.NewMenu()
	defer parseMenu.Destroy()
	buildAPIEditMenu(parseMenu)
	for _, parseLabel := range []string{"Undo", "Redo", "Cut", "Copy", "Paste", "Delete", "Select All"} {
		parseItem := parseMenu.FindByLabel(parseLabel)
		if parseItem == nil {
			parseTest.Fatalf("missing native edit role %s", parseLabel)
		}
		if parseItem.GetAccelerator() != "" {
			parseTest.Errorf("%s registers duplicate browser shortcut %s", parseLabel, parseItem.GetAccelerator())
		}
	}
}
