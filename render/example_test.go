//go:build js && wasm
// +build js,wasm

package render_test

import (
	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/render"
)

func ExampleTo() {
	// Create an element
	app := dom.Div(nil, dom.Text("Hello World"))

	// Render it to the element with id "app"
	// This starts the application
	render.To(app, "#app")
}
