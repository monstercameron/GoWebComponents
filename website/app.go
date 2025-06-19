//go:build js && wasm
// +build js,wasm

package website

import (
	. "github.com/monstercameron/GoWebComponents/fiber"
)

var RendertoDom = RenderTo

// App initializes the main application with debug configuration and routing.
// This is the entry point that sets up the framework's debugging namespaces,
// enables hot reload for development, and returns the configured router.
func App(props Attrs) *Element {

	// Configure debug logging namespaces for development visibility
	SetDebugNamespacesExclusive(map[string]bool{
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
	EnableHotReload(true)

	// Initialize application routing
	return AppRouter(nil)
}
