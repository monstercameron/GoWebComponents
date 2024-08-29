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
	count, setCount := HTML.UseState(0)

	updateCount := func() {
		HTML.UpdateElement("count", count)
	}

	JsFunc("increment", func() {
		setCount(count + 1)
		updateCount()
	})

	JsFunc("decrement", func() {
		setCount(count + 1)
		updateCount()
	})

	return Html("div", map[string]string{"class": "bg-white p-8 rounded-lg shadow-md text-center"},
		Html("h1", map[string]string{"class": "text-3xl font-bold mb-4"},
			Text("Clicker"),
		),
		Html("p", map[string]string{"class": "text-xl mb-4"},
			Text("Count: "),
			Html("span", map[string]string{"id": "count", "class": "font-bold"},
				Text(fmt.Sprint(count)),
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
