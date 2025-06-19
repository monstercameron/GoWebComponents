//go:build js && wasm
// +build js,wasm

package website

import (
	. "github.com/monstercameron/GoWebComponents/fiber"
)

// AppRouter sets up and manages the application routing
func AppRouter(props Attrs) *Element {
	// Register routes - library handles all state and navigation
	GoRegisterRoute("/", DocsWebsite)
	GoRegisterRoute("/docs", DocsPage)

	// Return the current route component
	return GoGetRoute().(*Element)
}
