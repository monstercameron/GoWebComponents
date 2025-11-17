//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

// HelloWorld component demonstrates basic usage
func HelloWorld(props dom.Attrs) *dom.Element {
	count, setCount := hooks.UseState(0)

	// UseEffect to log on mount and count changes
	hooks.UseEffect(func() func() {
		fmt.Printf("UseEffect ran: count is %d\n", count())
		return func() {
			fmt.Printf("UseEffect cleanup: count was %d\n", count())
		}
	}, count())

	// Create increment handler using functional setState to avoid stale closures
	increment := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Use functional update - always gets the latest state
		setCount(func(prev int) int {
			return prev + 1
		})
		return nil
	})
	// Don't release - it needs to stay alive for the button handler
	// The runtime will handle cleanup

	return dom.Div(dom.Attrs{"class": "container mx-auto p-8"},
		dom.H1(dom.Attrs{"class": "text-4xl font-bold mb-4"},
			dom.Text("GoWebComponents Test"),
		),
		dom.P(dom.Attrs{"class": "mb-4"},
			dom.Text(fmt.Sprintf("Count: %d", count())),
		),
		dom.Button(dom.Attrs{
			"onclick": increment,
			"class":   "px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600",
		},
			dom.Text("Increment"),
		),
	)
}

func main() {
	fmt.Println("🚀 GoWebComponents WASM initialized")

	// Create root element that will call HelloWorld during render
	app := &dom.Element{
		Type:  HelloWorld,
		Props: make(map[string]interface{}),
	}
	
	fmt.Println("About to call render.To...")
	render.To(app, "#app")
	fmt.Println("render.To completed")

	// Keep the Go program running
	select {}
}
