//go:build js && wasm
// +build js,wasm

package website

import (
	. "github.com/monstercameron/GoWebComponents/fiber"
)

var RendertoDom = RenderTo

// AppSection creates the main app section
func App(props Attrs) *Element {

	// Configure debug logging
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

	// Enable hot reload for development
	EnableHotReload(true)

	// can pass props to AppRouter for child components
	return AppRouter(nil)
}
