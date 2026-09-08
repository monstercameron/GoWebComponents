package services

import "github.com/wailsapp/wails/v3/pkg/application"

// buildAPIEditMenu leaves editing shortcuts owned by WebView2, not two handlers.
func buildAPIEditMenu(parseMenu *application.Menu) {
	parseMenu.AddRole(application.EditMenu)
	// beta.17 clears the accelerator's handled flag after invoking its callback.
	// Registering Ctrl+V here therefore permits both native paste and WebView paste.
	// Keep clickable native roles, but let WebView2 handle editing keys exactly once.
	for _, parseLabel := range []string{"Undo", "Redo", "Cut", "Copy", "Paste", "Delete", "Select All"} {
		if parseItem := parseMenu.FindByLabel(parseLabel); parseItem != nil {
			parseItem.RemoveAccelerator()
		}
	}
}
