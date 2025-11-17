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
		return nil // No cleanup needed for this simple example
	}, count())

	// UseMemo to compute expensive value (for testing)
	doubledCount := hooks.UseMemo(func() interface{} {
		fmt.Printf("UseMemo computing: count=%d\n", count())
		return count() * 2
	}, count()).(int)

	// Create increment handler using functional setState
	increment := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		setCount(func(prev int) int {
			return prev + 1
		})
		return nil
	})

	return dom.Div(dom.Attrs{"class": "container mx-auto p-8"},
		dom.H1(dom.Attrs{"class": "text-4xl font-bold mb-4"},
			dom.Text("GoWebComponents Test"),
		),
		dom.P(dom.Attrs{"class": "mb-4"},
			dom.Text(fmt.Sprintf("Count: %d", count())),
		),
		dom.P(dom.Attrs{"class": "mb-4", "id": "doubled"},
			dom.Text(fmt.Sprintf("Doubled: %d", doubledCount)),
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
