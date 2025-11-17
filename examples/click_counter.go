// ./examples/click_counter.go
// This file demonstrates a complete click counter application using the GoWebComponents framework.
// It showcases modern React-like patterns in Go including:
// - Component composition and reusability
// - State management with GoUseState hooks
// - Event handling with GoUseFunc and GoEvent
// - Tailwind CSS styling integration
// - Real-time UI updates and user interaction

//go:build js && wasm
// +build js,wasm

package example

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

// ClickPageHead creates the HTML head section with meta tags and title
// This component demonstrates:
// - Conditional prop handling (dynamic title)
// - HTML meta tag creation for responsive design
// - Component reusability with configurable props
func ClickPageHead(props Attrs) *Element {
	// Default title that can be overridden via props
	title := "Click Counter - GoWebComponents"
	if props != nil && props["title"] != nil {
		// Type assertion to safely extract string value from interface{}
		title = props["title"].(string)
	}

	// Head() creates an HTML <head> element
	// Meta() creates <meta> tags for charset and viewport
	// Title() creates a <title> tag with Text() for the content
	return dom.Head(nil,
		dom.Meta(Attrs{"charset": "UTF-8"}), // Ensures proper character encoding
		dom.Meta(Attrs{
			"name":    "viewport",
			"content": "width=device-width, initial-scale=1.0", // Makes the page responsive
		}),
		dom.Title(nil, dom.Text(title)), // Text() creates a text node
	)
}

// ClickPageLayout creates the main page container with gradient background
// This component demonstrates:
// - Tailwind CSS class usage for styling
// - Children prop pattern for component composition
// - Flexbox layout for centering content
func ClickPageLayout(props Attrs) *Element {
	// Body() creates an HTML <body> element with Tailwind classes
	// - min-h-screen: minimum height of 100vh (full viewport height)
	// - bg-gradient-to-br: diagonal gradient background
	// - from-purple-400 via-pink-500 to-red-500: gradient color stops
	return dom.Body(Attrs{
		"class": "min-h-screen bg-gradient-to-br from-purple-400 via-pink-500 to-red-500",
	},
		// Inner container for centering and padding
		dom.Div(Attrs{
			"class": "min-h-screen flex items-center justify-center p-4", // Flexbox centering
		},
			// props["children"] allows parent components to pass child elements
			// This is the React-like children pattern
			props["children"],
		),
	)
}

// ClickCounterCard creates a white card container for the counter content
// This component demonstrates:
// - Card-like UI design with shadows and rounded corners
// - Responsive design with max-width constraints
// - Children prop pattern for flexible content
func ClickCounterCard(props Attrs) *Element {
	// Div() creates a <div> element with Tailwind styling classes
	// - bg-white: white background
	// - rounded-2xl: large border radius for rounded corners
	// - shadow-2xl: large drop shadow for depth
	// - p-8: padding of 2rem on all sides
	// - max-w-md: maximum width constraint
	// - w-full: full width within constraints
	// - text-center: center-align text content
	return dom.Div(Attrs{
		"class": "bg-white rounded-2xl shadow-2xl p-8 max-w-md w-full text-center",
	},
		props["children"], // Flexible content area
	)
}

// ClickCounterHeader creates the title and description section
// This component demonstrates:
// - Typography hierarchy with H1 and P elements
// - Emoji usage for visual appeal
// - Tailwind typography classes
func ClickCounterHeader(props Attrs) *Element {
	return dom.Div(nil, // Container div with no special attributes
		// H1() creates an <h1> heading element
		dom.H1(Attrs{
			"class": "text-4xl font-bold text-gray-800 mb-2", // Large, bold, dark text with margin
		}, dom.Text("🖱️ Click Counter")), // Text() creates the actual text content

		// P() creates a <p> paragraph element for description
		dom.P(Attrs{
			"class": "text-gray-600 mb-8", // Gray text with bottom margin
		}, dom.Text("A simple counter built with GoWebComponents")),
	)
}

// ClickCounterDisplay shows the current counter value
// This component demonstrates:
// - Props-based data passing (count value)
// - Conditional rendering based on props
// - Large number display with visual hierarchy
func ClickCounterDisplay(props Attrs) *Element {
	// Default count value
	count := 0
	// Extract count from props if provided
	if props != nil && props["count"] != nil {
		// Type assertion to convert interface{} to int
		count = props["count"].(int)
	}

	return dom.Div(Attrs{
		"class": "mb-8", // Bottom margin for spacing
	},
		// Large number display
		dom.Div(Attrs{
			"class": "text-6xl font-bold text-purple-600 mb-2", // Very large, bold, purple text
		}, dom.Text(fmt.Sprintf("%d", count))), // Format integer as string

		// Label for the number
		dom.P(Attrs{
			"class": "text-gray-500", // Muted gray text
		}, dom.Text("clicks")),
	)
}

// ClickActionButton creates a reusable button component
// This component demonstrates:
// - Reusable component design with configurable props
// - Event handler prop passing
// - Dynamic CSS class handling
// - Default values with prop overrides
func ClickActionButton(props Attrs) *Element {
	// Default button styling - can be overridden via props
	buttonClass := "font-bold py-3 px-6 rounded-lg transition-colors duration-200 shadow-lg hover:shadow-xl transform hover:scale-105"
	if props != nil && props["class"] != nil {
		buttonClass = props["class"].(string)
	}

	// Extract click handler from props
	var onClick interface{}
	if props != nil && props["onclick"] != nil {
		onClick = props["onclick"]
	}

	// Extract button text from props
	text := "Button"
	if props != nil && props["text"] != nil {
		text = props["text"].(string)
	}

	// Button() creates an HTML <button> element
	// The onclick attribute connects to our Go event handler
	return dom.Button(Attrs{
		"onclick": onClick,     // Event handler function
		"class":   buttonClass, // CSS classes for styling
	}, dom.Text(text)) // Button text content
}

// ClickActionButtons creates the increment and decrement button group
// This component demonstrates:
// - Component composition (using ClickActionButton)
// - Event handler prop passing to child components
// - Flexbox layout for button arrangement
// - Color-coded buttons (red for decrease, green for increase)
func ClickActionButtons(props Attrs) *Element {
	// Extract event handlers from props
	var onIncrement, onDecrement interface{}
	if props != nil {
		onIncrement = props["onIncrement"]
		onDecrement = props["onDecrement"]
	}

	return dom.Div(Attrs{
		"class": "flex gap-4 justify-center mb-6", // Flexbox with gap and centering
	},
		// Decrement button (red styling)
		ClickActionButton(Attrs{
			"onclick": onDecrement,
			"class":   "bg-red-500 hover:bg-red-600 text-white font-bold py-3 px-6 rounded-lg transition-colors duration-200 shadow-lg hover:shadow-xl transform hover:scale-105",
			"text":    "➖ Decrease",
		}),

		// Increment button (green styling)
		ClickActionButton(Attrs{
			"onclick": onIncrement,
			"class":   "bg-green-500 hover:bg-green-600 text-white font-bold py-3 px-6 rounded-lg transition-colors duration-200 shadow-lg hover:shadow-xl transform hover:scale-105",
			"text":    "➕ Increase",
		}),
	)
}

// ClickResetButton creates a standalone reset button
// This component demonstrates:
// - Single-purpose component design
// - Reusing ClickActionButton for consistency
// - Gray styling to indicate secondary action
func ClickResetButton(props Attrs) *Element {
	// Extract reset handler from props
	var onReset interface{}
	if props != nil && props["onReset"] != nil {
		onReset = props["onReset"]
	}

	// Reuse ClickActionButton with reset-specific styling
	return ClickActionButton(Attrs{
		"onclick": onReset,
		"class":   "bg-gray-500 hover:bg-gray-600 text-white font-bold py-2 px-4 rounded-lg transition-colors duration-200",
		"text":    "🔄 Reset",
	})
}

// ClickTechStackInfo creates a footer section showing the technology stack
// This component demonstrates:
// - Informational content display
// - List creation with Ul() and Li() elements
// - Visual separation with Hr() (horizontal rule)
// - Typography hierarchy for information display
func ClickTechStackInfo(props Attrs) *Element {
	return dom.Div(nil,
		// Hr() creates a horizontal line for visual separation
		dom.Hr(Attrs{
			"class": "my-6 border-gray-200", // Vertical margin and light gray border
		}),

		// Information section
		dom.Div(Attrs{
			"class": "text-sm text-gray-500", // Small, muted text
		},
			dom.P(nil, dom.Text("Built with ❤️ using:")),
			// Ul() creates an unordered list
			dom.Ul(Attrs{
				"class": "list-none mt-2 space-y-1", // No bullets, margin top, vertical spacing
			},
				// Li() creates list items
				dom.Li(nil, dom.Text("🟢 Go + WebAssembly")),
				dom.Li(nil, dom.Text("⚛️ GoWebComponents (React-like)")),
				dom.Li(nil, dom.Text("🎨 Tailwind CSS")),
			),
		),
	)
}

// ClickMainClickCounter is the main component that orchestrates the entire counter application
// This component demonstrates:
// - State management with GoUseState hooks
// - Event handling with GoUseFunc and GoEvent
// - Component composition and data flow
// - Real-time UI updates based on state changes
func ClickMainClickCounter(props Attrs) *Element {
	fmt.Println("🔧 ClickMainClickCounter: Component initializing...")

	// STATE MANAGEMENT SECTION
	// GoUseState is a React-like hook that provides state management
	// It returns two functions: a getter and a setter
	// The state persists across component re-renders
	count, setCount := hooks.UseState(0) // Initialize counter to 0
	fmt.Printf("📊 ClickMainClickCounter: Counter state initialized with value: %d\n", count())

	// Second state for the demo input field
	inputValue, setInputValue := hooks.UseState("") // Initialize input to empty string
	fmt.Printf("📝 ClickMainClickCounter: Input state initialized with value: '%s'\n", inputValue())

	// EVENT HANDLER SECTION
	// GoUseFunc creates optimized event handlers that work with GoEvent objects
	// These handlers are memoized to prevent unnecessary re-creation on each render
	fmt.Println("🎯 ClickMainClickCounter: Setting up event handlers...")

	// Increment button handler
	// GoUseFunc wraps our handler function and provides a GoEvent object
	handleIncrement := hooks.GoUseFunc(func(event dom.GoEvent) {
		fmt.Println("➕ handleIncrement: Button clicked!")
		// event.PreventDefault() stops the default browser behavior
		event.PreventDefault()
		// Get current count value using the getter function
		oldCount := count()
		newCount := oldCount + 1
		// Update state using the setter function - this triggers a re-render
		setCount(newCount)
		fmt.Printf("📈 handleIncrement: Counter updated from %d to %d\n", oldCount, newCount)
	})

	// Decrement button handler
	handleDecrement := hooks.GoUseFunc(func(event dom.GoEvent) {
		fmt.Println("➖ handleDecrement: Button clicked!")
		event.PreventDefault()
		oldCount := count()
		newCount := oldCount - 1
		setCount(newCount) // State update triggers re-render
		fmt.Printf("📉 handleDecrement: Counter updated from %d to %d\n", oldCount, newCount)
	})

	// Reset button handler
	handleReset := hooks.GoUseFunc(func(event dom.GoEvent) {
		fmt.Println("🔄 handleReset: Reset button clicked!")
		event.PreventDefault()
		oldCount := count()
		setCount(0) // Reset to initial value
		fmt.Printf("🔄 handleReset: Counter reset from %d to 0\n", oldCount)
	})

	// Input change handler - demonstrates GoEvent.GetValue() usage
	// This shows how to handle form inputs without direct JavaScript interop
	handleInputChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		fmt.Println("⌨️ handleInputChange: Input event triggered")
		// GoEvent.GetValue() extracts the current input value cleanly
		// No need for event.target.value JavaScript interop
		value := event.GetValue()
		oldValue := inputValue()
		setInputValue(value) // Update input state
		fmt.Printf("📝 handleInputChange: Input value changed from '%s' to '%s'\n", oldValue, value)
	})

	fmt.Println("✅ ClickMainClickCounter: All event handlers configured")

	// DEMO INPUT COMPONENT SECTION
	// This demonstrates form input handling with GoWebComponents
	fmt.Println("🎨 ClickMainClickCounter: Creating demo input component...")
	demoInput := dom.Div(Attrs{
		"class": "mb-6", // Bottom margin for spacing
	},
		// Label for accessibility and user guidance
		dom.Label(Attrs{
			"class": "block text-sm font-medium text-gray-700 mb-2",
		}, dom.Text("Demo Input (shows GoEvent.GetValue()):")),

		// Input element with event handling
		dom.Input(Attrs{
			"type":        "text",
			"value":       inputValue(),      // Controlled input - value comes from state
			"oninput":     handleInputChange, // Event handler for input changes
			"placeholder": "Type something...",
			// Comprehensive styling with focus states
			"class": "w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent text-black",
		}),

		// Real-time display of current input value
		dom.P(Attrs{
			"class": "mt-2 text-sm text-gray-600",
		}, dom.Text(fmt.Sprintf("Current value: %s", inputValue()))), // Shows live updates
	)

	// COMPONENT COMPOSITION SECTION
	// This demonstrates how to compose larger components from smaller ones
	fmt.Println("🏗️ ClickMainClickCounter: Assembling counter card content...")
	counterCardContent := dom.Div(nil,
		// Header component (title and description)
		ClickCounterHeader,
		// Display component with current count passed as prop
		ClickCounterDisplay(Attrs{"count": count()}), // Pass current state as prop
		// Demo input component
		demoInput,
		// Action buttons with event handlers passed as props
		ClickActionButtons(Attrs{
			"onIncrement": handleIncrement, // Pass event handlers down
			"onDecrement": handleDecrement,
		}),
		// Reset button with its handler
		ClickResetButton(Attrs{"onReset": handleReset}),
		// Tech stack information footer
		ClickTechStackInfo,
	)
	fmt.Println("✅ ClickMainClickCounter: Counter card content assembled")

	// MAIN PAGE STRUCTURE SECTION
	// This creates the complete HTML document structure
	fmt.Println("🏛️ ClickMainClickCounter: Building main page structure...")
	pageStructure := dom.Html(Attrs{"lang": "en"}, // HTML root element with language
		// Page head with meta tags and title
		ClickPageHead(Attrs{"title": "Click Counter - GoWebComponents"}),
		// Page layout with the counter card as children
		ClickPageLayout(Attrs{
			"children": ClickCounterCard(Attrs{
				"children": counterCardContent, // Nested children pattern
			}),
		}),
	)
	fmt.Println("🎉 ClickMainClickCounter: Component render complete!")
	return pageStructure
}

// ClickCounterExample is the main entry point for the click counter application
// This function demonstrates:
// - Application initialization and setup
// - DOM container discovery and validation
// - Component rendering using the GoWebComponents framework
// - Error handling for missing DOM elements
func ClickCounterExample() {
	fmt.Println("🚀 ClickCounterExample: Starting click counter application...")

	// COMPONENT CREATION SECTION
	// Create a wrapper component function that returns our main component
	// This follows the React pattern of having a root component
	fmt.Println("🔧 ClickCounterExample: Creating click counter page component...")
	clickCounterPage := func(props Attrs) *Element {
		fmt.Println("📄 clickCounterPage: Component function called")
		// Return our main counter component
		return ClickMainClickCounter(nil)
	}

	// DOM CONTAINER DISCOVERY SECTION
	// Find the HTML element where we'll render our Go component
	// This uses JavaScript interop to access the browser's DOM
	fmt.Println("🔍 ClickCounterExample: Searching for DOM container with id 'root'...")
	container := js.Global().Get("document").Call("getElementById", "root")

	// Error handling for missing container
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("❌ ClickCounterExample: ERROR - No element with id 'root' found in the DOM!")
		fmt.Println("💡 ClickCounterExample: Make sure your HTML has a <div id='root'></div> element")
		return // Exit early if no container found
	}
	fmt.Println("✅ ClickCounterExample: DOM container found successfully")

	// RENDERING SECTION
	// Convert our Go component into a virtual DOM element and render it
	fmt.Println("🎨 ClickCounterExample: Creating element and rendering to container...")

	// CreateElement converts our component function into a virtual DOM element
	// This is similar to React.createElement()
	element := dom.CreateElement(clickCounterPage, nil)
	fmt.Printf("🧩 ClickCounterExample: Element created: %+v\n", element.Type)

	// Render takes our virtual DOM element and converts it to real DOM
	// It then inserts it into the specified container
	// This is where the Go code becomes actual HTML in the browser
	render.ToElement(element, container)

	fmt.Println("🎉 ClickCounterExample: Click counter application rendered successfully!")
	fmt.Println("👆 ClickCounterExample: Ready for user interaction - try clicking the buttons!")
}
