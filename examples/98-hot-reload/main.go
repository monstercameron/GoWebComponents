//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// App demonstrates hot reload basics
func App() ui.Node {
	count := ui.UseState(0)
	currentCount := count.Get()

	increment := ui.UseEvent(func() {
		count.Update(func(prev int) int { return prev + 1 })
	})

	return html.Div(html.Props{Class: "p-8"},
		html.H1(html.Props{Class: "text-2xl font-bold mb-4"},
			html.Text("Hot Reload Development Server test"),
		),
		html.P(html.Props{Class: "mb-4 text-gray-300"},
			html.Text("Try editing main.go while this is ruining. Increase the count, edit the text, and watch the hot reload! The state SHOULD be preserved!"),
		),
		html.Div(html.Props{Class: "bg-gray-800 p-6 rounded-lg"},
			html.Text(fmt.Sprintf("Current count: %d", currentCount)),
			html.Button(html.Props{
				Class:   "mx-4 bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded",
				OnClick: increment,
			},
				html.Text("Increment Counter111"),
			),
		),
	)
}

func main() {
	utils.EnableHotReload(true)

	app := ui.CreateElement(App)
	ui.Render(app, "#app")

	// Keep the Go program running
	select {}
}
