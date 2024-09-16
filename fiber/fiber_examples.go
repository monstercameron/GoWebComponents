package fiber

import (
	"fmt"
	"syscall/js"
)

// Example1 is a function that renders a calculator component using GoWebComponents.
// It initializes the state for the calculator, handles button clicks for numbers and operators,
// evaluates expressions, and renders the calculator UI.
//
// The calculator component consists of a display area for the previous expression and the input expression,
// as well as buttons for numbers, operators, clear, and equal.
//
// The function takes no parameters and returns no values.
// It finds the container in the DOM to render the component into and renders the calculator component into the container.
// If no element with the id 'root' is found in the DOM, an error message is printed.
//
// Example1 is intended to be used as an example of how to use the GoWebComponents library to create a calculator component.
func Example1() {
	fmt.Println("Example1: Starting to render calculator")

	// Calculator component
	calculator := func(props map[string]interface{}) *Element {
		// Initialize state for the calculator
		input, setInput := useState("")
		result, setResult := useState("")
		previousExpression, setPreviousExpression := useState("")

		useEffect(func() {
			fmt.Println("Result changed:", result())
		}, []interface{}{result()})


		// Function to handle button clicks for numbers and operators
		handleButtonClick := func() js.Func {
			cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				// Get the value from the button clicked
				value := args[0].Get("target").Get("innerText").String()
				fmt.Println("Button clicked:", value)
				// Append the value to the input
				newInput := input() + value
				setInput(newInput)
				// Clear the result since we're building a new expression
				setResult("")
				return nil
			})
			// Store the callback to keep it alive
			eventCallbacks = append(eventCallbacks, cb)
			return cb
		}

		// Function to handle the equal button click
		handleEqual := func() js.Func {
			cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				expr := input()
				fmt.Println("Evaluating expression:", expr)
				// Evaluate the expression using JavaScript's eval
				res, err := jsEval(expr)
				if err != nil {
					fmt.Println("Error evaluating expression:", err)
					setResult("Error")
				} else {
					setResult(res)
					// Store the previous expression
					setPreviousExpression(expr + " = " + res)
					// Set the input to the result for the next calculation
					setInput(res)
				}
				return nil
			})
			// Store the callback to keep it alive
			eventCallbacks = append(eventCallbacks, cb)
			return cb
		}

		// Function to handle the clear button click
		handleClear := func() js.Func {
			cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				setInput("")
				setResult("")
				setPreviousExpression("")
				return nil
			})
			// Store the callback to keep it alive
			eventCallbacks = append(eventCallbacks, cb)
			return cb
		}

		// Render the calculator UI
		return createElement("div", map[string]interface{}{"class": "container mx-auto p-4 grid grid-cols-12"},
			createElement("h1", map[string]interface{}{"class": "text-2xl font-bold mb-4"}, Text("GoWebComponent Calculator")),
			createElement("div", map[string]interface{}{
				"class": "mb-4 col-start-5 col-end-9",
			},
				// Display the previous expression
				createElement("div", map[string]interface{}{"class": "h-5 text-right text-gray-500 text-sm"}, Text(previousExpression())),
				// Display the input expression
				createElement("div", map[string]interface{}{
					"class": "h-16 text-right text-green-500 text-3xl font-mono bg-gray-800 p-4 rounded",
				}, Text(input())),
			),
			// Calculator buttons
			createElement("div", map[string]interface{}{"class": "col-start-5 col-end-9 grid grid-cols-4 gap-4"},
				// Row 1: Clear (C), Divide (/)
				createElement("button", map[string]interface{}{
					"class":   "col-span-3 bg-red-600 text-white p-4 rounded hover:bg-red-700 transition duration-200",
					"onclick": handleClear(),
				}, Text("C")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-500 text-white p-4 rounded hover:bg-gray-700 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("/")),
				// Row 2: 7,8,9,*
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("7")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("8")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("9")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-500 text-white p-4 rounded hover:bg-gray-700 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("*")),
				// Row 3: 4,5,6,-
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("4")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("5")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("6")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-500 text-white p-4 rounded hover:bg-gray-700 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("-")),
				// Row 4: 1,2,3,+
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("1")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("2")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("3")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-500 text-white p-4 rounded hover:bg-gray-700 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("+")),
				// Row 5: 0, ., =
				createElement("button", map[string]interface{}{
					"class":   "col-span-2 bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("0")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text(".")),
				createElement("button", map[string]interface{}{
					"class":   "bg-blue-600 text-white p-4 rounded hover:bg-blue-700 transition duration-200",
					"onclick": handleEqual(),
				}, Text("=")),
			),
		)
	}

	// Find the container in the DOM to render the component into
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example1: Error - No element with id 'root' found in the DOM")
		return
	}

	// Render the calculator component into the container
	fmt.Println("Example1: Rendering calculator into the container")
	render(createElement(calculator, nil), container)
}

// jsEval evaluates a mathematical expression using JavaScript's eval function.
// Note: In production, using eval can be unsafe; consider using a proper parser.
func jsEval(expr string) (string, error) {
	// Use JavaScript's eval function via the Function constructor to safely evaluate the expression.
	evalFunc := js.Global().Call("Function", "expr", "try { return eval(expr).toString(); } catch (e) { return 'Error'; }")
	res := evalFunc.Invoke(expr)
	resultStr := res.String()
	if resultStr == "Error" {
		return "", fmt.Errorf("error evaluating expression")
	}
	return resultStr, nil
}


// Example2 demonstrates the usage of a simple click counter component. The click counter component keeps track of the number of times a button is clicked. It renders a div container with a heading and a button. The button displays the current count. When the button is clicked, the count is incremented and displayed. The component utilizes the useState and useEffect hooks from the GoWebComponents library. The useState hook is used to manage the count state, while the useEffect hook is used to log a message when the component is mounted. Example2 also demonstrates how to render the component into the DOM using the render function.
func Example2() {
	fmt.Println("Example2: Starting to render ClickCounter")

	// simple click counter component
	clickCounter := func(props map[string]interface{}) *Element {
		count, setCount := useState(0)

		handleClick := func() js.Func {
			cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				fmt.Printf("handleClick: Clicked, count is %d\n", count())
				setCount(count() + 1)
				return nil
			})
			eventCallbacks = append(eventCallbacks, cb) // Keep callback alive
			return cb
		}

		useEffect(func() {
			fmt.Println("useEffect: Component mounted")
		}, emptyDeps)

		return createElement("div", map[string]interface{}{"class": "container mx-auto p-4"},
			createElement("h1", map[string]interface{}{"class": "text-2xl font-bold mb-4"},
				Text("Click Counter")),
			createElement("button", map[string]interface{}{
				"onclick": handleClick(), // Pass the function reference
				"class":   "px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition duration-200",
			}, Text(fmt.Sprintf("Clicked %d times", count())))) // Pass the function reference for `count`

	}

	// Start rendering
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example6: Error - No element with id 'root' found in the DOM")
		return
	}
	fmt.Println("Example6: Rendering BlogListComponent into the container")
	render(createElement(clickCounter, nil), container)
}

// Example3 demonstrates rendering a BlogListComponent into a container element in the DOM.
func Example3() {
	// Start rendering
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example5: Error - No element with id 'root' found in the DOM")
		return
	}
	render(createElement(BlogListComponent, nil), container)
}