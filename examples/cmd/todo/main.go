//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	example "github.com/monstercameron/GoWebComponents/example"
	"github.com/monstercameron/GoWebComponents/render"
)

func main() {
	fmt.Println("🚀 Building TodoApp with hash router")

	// Create a component element that wraps TodoApp
	// The render engine will call TodoApp as a component function with proper fiber context
	app := &dom.Element{
		Type:  example.TodoApp,
		Props: make(map[string]interface{}),
	}

	render.To(app, "#app")

	// Keep the Go program running to handle events
	select {}
}
