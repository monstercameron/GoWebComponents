package main

import (
	html "goHTML/components"
	"sync"
	"syscall/js"
	"fmt"
)

var (
	CreateComponent = html.CreateComponent
	Tag             = html.Tag
	Text            = html.Text
	Render          = html.Render
	Function        = html.Function
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	// Composed Document Component with inlined markup
	document := CreateComponent(func(c *html.Component, _ html.Props, _ ...*html.Component) *html.Component {
		counter, setCounter := html.AddState(c, "counter", 0)

		handleClick := Function(c, "handleClick", func(js.Value) {
			fmt.Println("Button clicked!")
			setCounter(*counter + 1)
		})

		Render(c, Tag("div", map[string]string{"class": "min-h-screen flex flex-col"},
			// Header
			Tag("header", map[string]string{"class": "bg-gradient-to-r from-blue-500 to-purple-500 text-white p-6 shadow-md text-center"},
				Tag("h1", map[string]string{"class": "text-3xl font-bold"}, Text("Modern Clicker App")),
				Tag("p", map[string]string{"class": "text-lg mt-2"}, Text("Sleek and modern design with Tailwind CSS")),
			),

			// Main content with Clicker
			Tag("main", map[string]string{"class": "flex-grow flex items-center justify-center"},
				Tag("div", map[string]string{"class": "flex flex-col items-center justify-center mt-8"},
					Tag("button", map[string]string{
						"class":   "px-6 py-3 bg-blue-600 text-white font-semibold rounded-md shadow-lg hover:bg-blue-700 transition duration-300",
						"onclick": handleClick,
					}, Text(fmt.Sprintf("Click Me! Count: %d", *counter))),
				),
			),

			// Footer
			Tag("footer", map[string]string{"class": "bg-gray-800 text-white p-4 text-center mt-8"},
				Text("© 2024 My Clicker App"),
			),
		))

		return c
	})

	// Use the RenderToBody function to handle rendering and updating the DOM
	html.RenderToBody(document(html.Props{}))

	wg.Wait()
}
