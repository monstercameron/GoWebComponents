// ./fiber/click_counter.go

package fiber

import (
	"fmt"
	"syscall/js"
)

// Page head component - Meta tags, title, and external scripts
func PageHead(props Attrs) *Element {
	title := "Click Counter - GoWebComponents"
	if props != nil && props["title"] != nil {
		title = props["title"].(string)
	}

	return Head(nil,
		Meta(Attrs{"charset": "UTF-8"}),
		Meta(Attrs{
			"name":    "viewport",
			"content": "width=device-width, initial-scale=1.0",
		}),
		Title(nil, Text(title)),
		Script(Attrs{
			"src": "https://cdn.tailwindcss.com",
		}),
	)
}

// Page layout component - Gradient background container
func PageLayout(props Attrs) *Element {
	return Body(Attrs{
		"class": "min-h-screen bg-gradient-to-br from-purple-400 via-pink-500 to-red-500",
	},
		Div(Attrs{
			"class": "min-h-screen flex items-center justify-center p-4",
		},
			props["children"],
		),
	)
}

// Counter card component - White container with rounded corners
func CounterCard(props Attrs) *Element {
	return Div(Attrs{
		"class": "bg-white rounded-2xl shadow-2xl p-8 max-w-md w-full text-center",
	},
		props["children"],
	)
}

// Counter header component - Title and description
func CounterHeader(props Attrs) *Element {
	return Div(nil,
		H1(Attrs{
			"class": "text-4xl font-bold text-gray-800 mb-2",
		}, Text("🖱️ Click Counter")),

		P(Attrs{
			"class": "text-gray-600 mb-8",
		}, Text("A simple counter built with GoWebComponents")),
	)
}

// Counter display component - Shows the current count
func CounterDisplay(props Attrs) *Element {
	count := 0
	if props != nil && props["count"] != nil {
		count = props["count"].(int)
	}

	return Div(Attrs{
		"class": "mb-8",
	},
		Div(Attrs{
			"class": "text-6xl font-bold text-purple-600 mb-2",
		}, Text(fmt.Sprintf("%d", count))),

		P(Attrs{
			"class": "text-gray-500",
		}, Text("clicks")),
	)
}

// Action button component - Reusable button with customizable styling
func ActionButton(props Attrs) *Element {
	buttonClass := "font-bold py-3 px-6 rounded-lg transition-colors duration-200 shadow-lg hover:shadow-xl transform hover:scale-105"
	if props != nil && props["class"] != nil {
		buttonClass = props["class"].(string)
	}

	var onClick interface{}
	if props != nil && props["onclick"] != nil {
		onClick = props["onclick"]
	}

	text := "Button"
	if props != nil && props["text"] != nil {
		text = props["text"].(string)
	}

	return Button(Attrs{
		"onclick": onClick,
		"class":   buttonClass,
	}, Text(text))
}

// Action buttons component - Increment and decrement buttons
func ActionButtons(props Attrs) *Element {
	var onIncrement, onDecrement interface{}
	if props != nil {
		onIncrement = props["onIncrement"]
		onDecrement = props["onDecrement"]
	}

	return Div(Attrs{
		"class": "flex gap-4 justify-center mb-6",
	},
		ActionButton(Attrs{
			"onclick": onDecrement,
			"class":   "bg-red-500 hover:bg-red-600 text-white font-bold py-3 px-6 rounded-lg transition-colors duration-200 shadow-lg hover:shadow-xl transform hover:scale-105",
			"text":    "➖ Decrease",
		}),

		ActionButton(Attrs{
			"onclick": onIncrement,
			"class":   "bg-green-500 hover:bg-green-600 text-white font-bold py-3 px-6 rounded-lg transition-colors duration-200 shadow-lg hover:shadow-xl transform hover:scale-105",
			"text":    "➕ Increase",
		}),
	)
}

// Reset button component - Standalone reset button
func ResetButton(props Attrs) *Element {
	var onReset interface{}
	if props != nil && props["onReset"] != nil {
		onReset = props["onReset"]
	}

	return ActionButton(Attrs{
		"onclick": onReset,
		"class":   "bg-gray-500 hover:bg-gray-600 text-white font-bold py-2 px-4 rounded-lg transition-colors duration-200",
		"text":    "🔄 Reset",
	})
}

// Tech stack info component - Footer with technology information
func TechStackInfo(props Attrs) *Element {
	return Div(nil,
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
	)
}

// Main click counter component - Composed of atomic components
func MainClickCounter(props Attrs) *Element {
	// State for the counter using GoUseState
	count, setCount := GoUseState(0)

	// State for demonstration input using GoUseState
	inputValue, setInputValue := GoUseState("")

	// Handle click events using GoUseFunc for cleaner syntax
	handleIncrement := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		newCount := count() + 1
		setCount(newCount)
		fmt.Printf("ClickCounter: Incremented to %d\n", newCount)
	})

	handleDecrement := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		newCount := count() - 1
		setCount(newCount)
		fmt.Printf("ClickCounter: Decremented to %d\n", newCount)
	})

	handleReset := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		setCount(0)
		fmt.Println("ClickCounter: Reset to 0")
	})

	// Demo of GoEvent.GetValue() for input handling
	handleInputChange := GoUseFunc(func(event GoEvent) {
		value := event.GetValue() // Clean, no JS interop needed!
		setInputValue(value)
		fmt.Printf("ClickCounter: Input changed to: %s\n", value)
	})

	// Demo input component to show GoEvent.GetValue()
	demoInput := Div(Attrs{
		"class": "mb-6",
	},
		Label(Attrs{
			"class": "block text-sm font-medium text-gray-700 mb-2",
		}, Text("Demo Input (shows GoEvent.GetValue()):")),
		Input(Attrs{
			"type":        "text",
			"value":       inputValue(),
			"oninput":     handleInputChange,
			"placeholder": "Type something...",
			"class":       "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent",
		}),
		P(Attrs{
			"class": "mt-2 text-sm text-gray-600",
		}, Text(fmt.Sprintf("Current value: %s", inputValue()))),
	)

	// Counter card content using component composition
	counterCardContent := Div(nil,
		CounterHeader,
		CounterDisplay(Attrs{"count": count()}),
		demoInput, // Add the demo input
		ActionButtons(Attrs{
			"onIncrement": handleIncrement,
			"onDecrement": handleDecrement,
		}),
		ResetButton(Attrs{"onReset": handleReset}),
		TechStackInfo,
	)

	// Main page structure using atomic components
	return Html(Attrs{"lang": "en"},
		PageHead(Attrs{"title": "Click Counter - GoWebComponents"}),
		PageLayout(Attrs{
			"children": CounterCard(Attrs{
				"children": counterCardContent,
			}),
		}),
	)
}

// ClickCounter creates a simple click counter page demonstrating state management
func ClickCounter() {
	fmt.Println("ClickCounter: Starting to render click counter page")

	// Main click counter component using atomic components
	clickCounterPage := func(props Attrs) *Element {
		return MainClickCounter(nil)
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
