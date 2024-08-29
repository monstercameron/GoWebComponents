package main

import (
	"fmt"
	HTML "goHTML/html"
	"syscall/js"
)

var (
	Html   = HTML.HTML
	Text   = HTML.Text
	JsFunc = HTML.WasmFunc
)

type Node = HTML.Node

func GenerateClicker() Node {
	// Initialize state with an initial count of 0.
	count, setCount, countId := HTML.UseState(0)

	// Define the JavaScript functions that will be exposed to the global scope.
	JsFunc("increment", func() {
		// Increment the count value and update the display.
		setCount(*count + 1)
	})

	JsFunc("decrement", func() {
		// Decrement the count value and update the display.
		setCount(*count - 1)
	})

	// Construct and return the HTML structure for the clicker component.
	return Html("div", map[string]string{"class": "bg-white p-8 rounded-lg shadow-md text-center"},
		Html("h1", map[string]string{"class": "text-3xl font-bold mb-4"},
			Text("Clicker"),
		),
		Html("p", map[string]string{"class": "text-xl mb-4"},
			Text("Count: "),
			Html("span", map[string]string{"id": "count",
				"data-state": countId, "class": "font-bold"},
				Text(fmt.Sprintf("%d clicks", *count)), // Render the initial count value.
			),
		),
		Html("button", map[string]string{
			"onclick": "increment()",
			"class":   "bg-green-500 hover:bg-green-600 text-white font-bold py-2 px-4 rounded mr-2 focus:outline-none focus:shadow-outline",
		},
			Text("+"),
		),
		Html("button", map[string]string{
			"onclick": "decrement()",
			"class":   "bg-red-500 hover:bg-red-600 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline",
		},
			Text("-"),
		),
	)
}

func GeneratePage(content Node) Node {
	return Html("html", nil,
		Html("head", nil,
			Html("title", nil, Text("Clicker Demo")),
			Html("script", map[string]string{"src": "https://cdn.tailwindcss.com"}, Text("")),
		),
		Html("body", map[string]string{"class": "bg-gray-100 h-screen flex items-center justify-center"},
			content,
		),
	)
}

func main() {
	c := make(chan struct{}, 0)

	fmt.Println("WebAssembly Go Initialized")

	page := GeneratePage(GenerateClicker())
	renderedHTML, err := page.Render()
	if err != nil {
		panic(err)
	}

	// Write the rendered HTML to the document
	js.Global().Get("document").Call("write", renderedHTML)

	<-c
}
