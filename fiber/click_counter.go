// ./fiber/click_counter.go

package fiber

import (
	"fmt"
	"syscall/js"
)

// ClickCounter creates a simple click counter page demonstrating state management
func ClickCounter() {
	fmt.Println("ClickCounter: Starting to render click counter page")

	// Main click counter component
	clickCounterPage := func(props Attrs) *Element {
		// State for the counter
		count, setCount := useState(0)

		// Handle click events
		handleIncrement := useFunc(func(this js.Value, args []js.Value) interface{} {
			args[0].Call("preventDefault")
			newCount := count() + 1
			setCount(newCount)
			fmt.Printf("ClickCounter: Incremented to %d\n", newCount)
			return nil
		})

		handleDecrement := useFunc(func(this js.Value, args []js.Value) interface{} {
			args[0].Call("preventDefault")
			newCount := count() - 1
			setCount(newCount)
			fmt.Printf("ClickCounter: Decremented to %d\n", newCount)
			return nil
		})

		handleReset := useFunc(func(this js.Value, args []js.Value) interface{} {
			args[0].Call("preventDefault")
			setCount(0)
			fmt.Println("ClickCounter: Reset to 0")
			return nil
		})

		// Main page structure
		return Html(Attrs{
			"lang": "en",
		},
			Head(nil,
				Meta(Attrs{"charset": "UTF-8"}),
				Meta(Attrs{
					"name":    "viewport",
					"content": "width=device-width, initial-scale=1.0",
				}),
				Title(nil, Text("Click Counter - GoWebComponents")),
				Script(Attrs{
					"src": "https://cdn.tailwindcss.com",
				}),
			),
			Body(Attrs{
				"class": "min-h-screen bg-gradient-to-br from-purple-400 via-pink-500 to-red-500",
			},
				Div(Attrs{
					"class": "min-h-screen flex items-center justify-center p-4",
				},
					Div(Attrs{
						"class": "bg-white rounded-2xl shadow-2xl p-8 max-w-md w-full text-center",
					},
						// Header
						H1(Attrs{
							"class": "text-4xl font-bold text-gray-800 mb-2",
						}, Text("🖱️ Click Counter")),

						P(Attrs{
							"class": "text-gray-600 mb-8",
						}, Text("A simple counter built with GoWebComponents")),

						// Counter display
						Div(Attrs{
							"class": "mb-8",
						},
							Div(Attrs{
								"class": "text-6xl font-bold text-purple-600 mb-2",
							}, Text(fmt.Sprintf("%d", count()))),

							P(Attrs{
								"class": "text-gray-500",
							}, Text("clicks")),
						),

						// Action buttons
						Div(Attrs{
							"class": "flex gap-4 justify-center mb-6",
						},
							Button(Attrs{
								"onclick": handleDecrement,
								"class":   "bg-red-500 hover:bg-red-600 text-white font-bold py-3 px-6 rounded-lg transition-colors duration-200 shadow-lg hover:shadow-xl transform hover:scale-105",
							}, Text("➖ Decrease")),

							Button(Attrs{
								"onclick": handleIncrement,
								"class":   "bg-green-500 hover:bg-green-600 text-white font-bold py-3 px-6 rounded-lg transition-colors duration-200 shadow-lg hover:shadow-xl transform hover:scale-105",
							}, Text("➕ Increase")),
						),

						// Reset button
						Button(Attrs{
							"onclick": handleReset,
							"class":   "bg-gray-500 hover:bg-gray-600 text-white font-bold py-2 px-4 rounded-lg transition-colors duration-200",
						}, Text("🔄 Reset")),

						// Footer info
						Hr(Attrs{
							"class": "my-6 border-gray-200",
						}),

						Div(Attrs{
							"class": "text-sm text-gray-500",
						},
							P(nil, Text("Built with ❤️ using:")),
							Ul(Attrs{
								"class": "list-none mt-2 space-y-1",
							},
								Li(nil, Text("🟢 Go + WebAssembly")),
								Li(nil, Text("⚛️ GoWebComponents (React-like)")),
								Li(nil, Text("🎨 Tailwind CSS")),
							),
						),
					),
				),
			),
		)
	}

	// Render the click counter page
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("ClickCounter: Error - No element with id 'root' found in the DOM")
		return
	}

	fmt.Println("ClickCounter: Rendering click counter page into the container")
	render(createElement(clickCounterPage, nil), container)
}
