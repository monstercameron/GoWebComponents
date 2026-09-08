//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v6/hotreload"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

// App initializes the main application with debug configuration and routing.
// This is the entry point that sets up the framework's debugging namespaces,
// enables hot reload for development, and returns the configured router.
func App(parseProps Attrs) *Element {

	// Configure debug logging namespaces for development visibility
	utils.ConfigureDebugNamespacesExclusive(map[string]bool{
		"HOOKS":  false,
		"RENDER": false,
		"MEMORY": false,
		"DOM":    false,
		"FETCH":  false,
		"EVENTS": false,
		"COMMIT": false,
		"FIBER":  false,
	})

	// Enable hot reload for instant development feedback
	hotreload.Enable()

	// Initialize application routing
	return AppRouter(nil)
}
