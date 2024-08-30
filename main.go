package main

import (
	"fmt"
	HTML "goHTML/html"
	"syscall/js"
)


type Node = HTML.Node

func GenerateClicker() Node {
	// Initialize state with an initial count of 0.
	count, setCount, countId := HTML.UseState(0)

	// Define the JavaScript functions that will be exposed to the global scope.
	HTML.WasmFunc("increment", func() {
		// Increment the count value and update the display.
		setCount(*count + 1)
	})

	HTML.WasmFunc("decrement", func() {
		// Decrement the count value and update the display.
		setCount(*count - 1)
	})

	// Construct and return the HTML structure for the clicker component.
	return HTML.HTML("div", map[string]string{"class": "bg-white p-8 rounded-lg shadow-md text-center"}, []string{},
		HTML.HTML("h1", map[string]string{"class": "text-3xl font-bold mb-4"}, []string{}, HTML.Text("Clicker", []string{})),
		HTML.HTML("p", map[string]string{"class": "text-xl mb-4"}, []string{},
			HTML.Text("Count: ", []string{}),
			HTML.HTML("span", map[string]string{
				"id":         "count",
				"class":      "font-bold",
			}, []string{},
				HTML.Text(fmt.Sprintf("%d", *count), []string{countId}),
			),
		),
		HTML.HTML("button", map[string]string{
			"onclick": "increment()",
			"class":   "bg-green-500 hover:bg-green-600 text-white font-bold py-2 px-4 rounded mr-2 focus:outline-none focus:shadow-outline",
		}, []string{},
			HTML.Text("+", []string{}),
		),
		HTML.HTML("button", map[string]string{
			"onclick": "decrement()",
			"class":   "bg-red-500 hover:bg-red-600 text-white font-bold py-2 px-4 rounded focus:outline-none focus:shadow-outline",
		}, []string{},
			HTML.Text("-", []string{}),
		),
	)
}

func GeneratePage(content Node) Node {
	return HTML.HTML("html", nil, []string{},
		HTML.HTML("head", nil, []string{},
			HTML.HTML("title", nil, []string{}, HTML.Text("Clicker Demo", []string{})),
			HTML.HTML("script", map[string]string{"src": "https://cdn.tailwindcss.com"}, []string{}, HTML.Text("", []string{})),
		),
		HTML.HTML("body", map[string]string{"class": "bg-gray-100 h-screen flex items-center justify-center"}, []string{},
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
