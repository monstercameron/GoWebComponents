package components

import (
	"fmt"
	"strconv"
	"syscall/js"
)

type Todo struct {
	ID        int
	Text      string
	Completed bool
}

func Example1() {
	fmt.Println("Initializing Todo App Component...")

	todoApp := CreateComponent(func(c *Component, _ Props, _ ...*Component) *Component {
		todos, setTodos := AddState(c, "todos", []Todo{})
		nextID, setNextID := AddState(c, "nextID", 1)
		inputValue, setInputValue := AddState(c, "inputValue", "")

		addTodo := Function(c, "addTodo", func(_ js.Value) {
			if *inputValue != "" {
				newTodo := Todo{ID: *nextID, Text: *inputValue, Completed: false}
				fmt.Printf("Adding Todo: %+v\n", newTodo)
				setTodos(append(*todos, newTodo))
				setNextID(*nextID + 1)
				setInputValue("")
				fmt.Println("Todo Added and Input Cleared.")
			} else {
				fmt.Println("Input is empty, no Todo added.")
			}
		})

		toggleTodo := Function(c, "toggleTodo", func(event js.Value) {
			id := event.Get("target").Get("dataset").Get("id").Int()
			fmt.Printf("Toggling Todo with ID: %d\n", id)
			newTodos := make([]Todo, len(*todos))
			copy(newTodos, *todos)
			for i, todo := range newTodos {
				if todo.ID == id {
					newTodos[i].Completed = !newTodos[i].Completed
					fmt.Printf("Todo Toggled: %+v\n", newTodos[i])
					break
				}
			}
			//setTodos(newTodos)
		})

		removeTodo := Function(c, "removeTodo", func(event js.Value) {
			id := event.Get("target").Get("dataset").Get("id").Int()
			fmt.Printf("Removing Todo with ID: %d\n", id)
			newTodos := make([]Todo, 0, len(*todos)-1)
			for _, todo := range *todos {
				if todo.ID != id {
					newTodos = append(newTodos, todo)
				}
			}
			setTodos(newTodos)
			fmt.Println("Todo Removed.")
		})

		handleInputChange := Function(c, "handleInputChange", func(event js.Value) {
			fmt.Println("Input Changed.")
			newValue := event.Get("target").Get("value").String()
			fmt.Printf("Input Changed: %s\n", newValue)
			setInputValue(newValue)
		})

		// Compose the todo items list first
		var todoItems []NodeInterface
		for _, todo := range *todos {
			fmt.Printf("Rendering Todo: %+v\n", todo)
			todoItems = append(todoItems, Tag("li", map[string]string{"class": "flex items-center"},
				Tag("input", map[string]string{
					"type":     "checkbox",
					"checked":  fmt.Sprintf("%v", todo.Completed),
					"onchange": toggleTodo,
					"data-id":  fmt.Sprintf("%d", todo.ID),
					"class":    "mr-2",
				}),
				Tag("span", map[string]string{
					"class": fmt.Sprintf("flex-grow %s", map[bool]string{true: "line-through text-gray-500", false: ""}[todo.Completed]),
				}, Text(todo.Text)),
				Tag("button", map[string]string{
					"onclick": removeTodo,
					"data-id": fmt.Sprintf("%d", todo.ID),
					"class":   "ml-2 text-red-500 hover:text-red-700",
				}, Text("Remove")),
			))
		}

		// Convert []NodeInterface to []interface{}
		todoItemsInterface := make([]interface{}, len(todoItems))
		for i, item := range todoItems {
			todoItemsInterface[i] = item
		}

		// Compose the entire tree structure
		fmt.Println("Rendering the Todo App UI...")
		Render(c, Tag("div", map[string]string{"class": "min-h-screen bg-gray-100 py-6 flex flex-col justify-center sm:py-12"},
			Tag("div", map[string]string{"class": "relative py-3 sm:max-w-xl sm:mx-auto"},
				Tag("div", map[string]string{"class": "absolute inset-0 bg-gradient-to-r from-cyan-400 to-light-blue-500 shadow-lg transform -skew-y-6 sm:skew-y-0 sm:-rotate-6 sm:rounded-3xl"}),
				Tag("div", map[string]string{"class": "relative px-4 py-10 bg-white shadow-lg sm:rounded-3xl sm:p-20"},
					Tag("div", map[string]string{"class": "max-w-md mx-auto"},
						Tag("h1", map[string]string{"class": "text-2xl font-semibold mb-6 text-center"}, Text("Todo List")),
						Tag("div", map[string]string{"class": "flex mb-4"},
							Tag("input", map[string]string{
								"type":        "text",
								"placeholder": "Add a new todo",
								"value":       *inputValue,
								"onchange":    handleInputChange, // use "oninput" instead of "onchange" for real-time updates
								"class":       "flex-grow mr-2 p-2 border rounded",
							}),
							Tag("button", map[string]string{
								"onclick": addTodo,
								"class":   "bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded",
							}, Text("Add")),
						),
						Tag("ul", map[string]string{"class": "space-y-2"}, todoItemsInterface...), // Pass the composed todo items
					),
				),
			),
		))

		fmt.Println("Todo App UI Rendered.")
		return c
	})

	fmt.Println("Rendering Todo App to Body...")
	RenderToBody(todoApp(Props{}))
	fmt.Println("Todo App Rendered to Body.")
}

func preserveFocus(event js.Value, f func()) {
	doc := js.Global().Get("document")
	activeElement := doc.Get("activeElement")
	var activeID string

	// Only preserve focus if it's a change event
	if event.Get("type").String() == "change" {
		if !activeElement.IsUndefined() && !activeElement.IsNull() {
			activeID = activeElement.Get("id").String()
		}
	}

	f()

	if activeID != "" {
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			elem := doc.Call("getElementById", activeID)
			if !elem.IsUndefined() && !elem.IsNull() {
				elem.Call("focus")
			}
			return nil
		}), 0)
	}
}

func Example2() {
	// Log that the Todo App Component is being initialized
	fmt.Println("Initializing Todo App Component...")

	// Define the main component for the Todo App
	todoApp := CreateComponent(func(c *Component, _ Props, _ ...*Component) *Component {
		// Initialize state for todos, nextID, and inputValue
		todos, setTodos := AddState(c, "todos", []Todo{})          // List of todos
		nextID, setNextID := AddState(c, "nextID", 1)              // ID for the next todo item
		inputValue, setInputValue := AddState(c, "inputValue", "") // Current value of the input field

		// Define the function for adding a new todo item
		addTodo := Function(c, "addTodo", func(event js.Value) {
			// Check if the input value is not empty
			if *inputValue != "" {
				// Create a new todo with the current input value and the next ID
				newTodo := Todo{ID: *nextID, Text: *inputValue, Completed: false}
				fmt.Printf("Adding Todo: %+v\n", newTodo)

				// Update the todos list with the new todo
				setTodos(append(*todos, newTodo))

				// Increment the next ID and clear the input field
				setNextID(*nextID + 1)
				setInputValue("")
				fmt.Println("Todo Added and Input Cleared.")
			} else {
				fmt.Println("Input is empty, no Todo added.")
			}
		})

		// Define the function for toggling the completion status of a todo item
		toggleTodo := Function(c, "toggleTodo", func(event js.Value) {
			// Get the ID of the todo to toggle from the event's dataset
			idStr := event.Get("target").Get("dataset").Get("id").String()

			// Convert the string ID to an integer
			id, err := strconv.Atoi(idStr)
			if err != nil {
				fmt.Printf("Error converting id to integer: %s\n", err)
				return
			}

			// Log the ID of the todo being toggled
			fmt.Printf("Toggling Todo with ID: %d\n", id)

			// Create a new list of todos and toggle the completion status of the targeted todo
			newTodos := make([]Todo, len(*todos))
			copy(newTodos, *todos)
			for i, todo := range newTodos {
				if todo.ID == id {
					// Toggle the completed status
					newTodos[i].Completed = !newTodos[i].Completed
					fmt.Printf("Todo Toggled: %+v\n", newTodos[i])
					break
				}
			}
			// Update the state with the new list of todos
			setTodos(newTodos)
		})

		// Define the function for removing a todo item
		removeTodo := Function(c, "removeTodo", func(event js.Value) {
			// Get the ID of the todo to remove from the event's dataset
			idStr := event.Get("target").Get("dataset").Get("id").String()

			// Convert the string ID to an integer
			id, err := strconv.Atoi(idStr)
			if err != nil {
				fmt.Printf("Error converting id to integer: %s\n", err)
				return
			}

			// Log the ID of the todo being removed
			fmt.Printf("Removing Todo with ID: %d\n", id)

			// Create a new list of todos, excluding the one to be removed
			newTodos := make([]Todo, 0, len(*todos)-1)
			for _, todo := range *todos {
				if todo.ID != id {
					newTodos = append(newTodos, todo)
				}
			}
			// Update the state with the new list of todos
			setTodos(newTodos)
			fmt.Println("Todo Removed.")
		})

		// Define the function to handle input changes
		handleInputChange := Function(c, "handleInputChange", func(event js.Value) {
			// Preserve focus on the input field before re-rendering
			preserveFocus(event, func() {
				// Get the new value from the input field
				newValue := event.Get("target").Get("value").String()
				fmt.Printf("Input Changed: %s\n", newValue)

				// Update the inputValue state with the new value
				setInputValue(newValue)
			})
		})

		// Compose the todo items list based on the current state
		var todoItems []NodeInterface
		for _, todo := range *todos {
			// Log the todo item being rendered
			fmt.Printf("Rendering Todo: %+v\n", todo)

			// Create the base attributes for the checkbox
			checkboxAttrs := map[string]string{
				"id":       fmt.Sprintf("todo-checkbox-%d", todo.ID),
				"type":     "checkbox",
				"onchange": toggleTodo,
				"data-id":  fmt.Sprintf("%d", todo.ID),
				"class":    "mr-2",
			}

			// Add the "checked" attribute if the todo is completed
			if todo.Completed {
				checkboxAttrs["checked"] = ""
			}

			// Append the rendered todo item to the list
			todoItems = append(todoItems, Tag("li", map[string]string{"class": "flex items-center justify-between p-2 border-b border-gray-700"},
				Tag("input", checkboxAttrs), // Checkbox for toggling completion
				Tag("span", map[string]string{
					"class": fmt.Sprintf("flex-grow %s", map[bool]string{true: "line-through text-gray-500", false: ""}[todo.Completed]),
				}, Text(todo.Text)), // Text of the todo item
				Tag("button", map[string]string{
					"id":      fmt.Sprintf("remove-todo-%d", todo.ID),
					"onclick": removeTodo,
					"data-id": fmt.Sprintf("%d", todo.ID),
					"class":   "ml-2 text-red-500 hover:text-red-700",
				}, Text("Remove")), // Button to remove the todo item
			))
		}

		// Convert the slice of NodeInterface to a slice of interface{} for rendering
		todoItemsInterface := make([]interface{}, len(todoItems))
		for i, item := range todoItems {
			todoItemsInterface[i] = item
		}

		// Compose the entire tree structure with dark mode design
		fmt.Println("Adding the Nodes to the Component with updated closure...")
		Render(c, Tag("div", map[string]string{"class": "min-h-screen bg-gray-900 text-gray-100 p-4 flex flex-col items-center"},
			Tag("div", map[string]string{"class": "w-full max-w-md"},
				Tag("h1", map[string]string{"class": "text-2xl font-semibold mb-4 text-center text-gray-200"}, Text("Todo List")),
				Tag("div", map[string]string{"class": "flex mb-4"},
					Tag("input", map[string]string{
						"id":          "new-todo-input",
						"type":        "text",
						"placeholder": "Add a new todo",
						"value":       *inputValue,
						"onchange":    handleInputChange,
						"class":       "flex-grow mr-2 p-2 border rounded bg-gray-800 text-gray-100 border-gray-700",
					}), // Input field for new todos
					Tag("button", map[string]string{
						"id":      "add-todo-button",
						"onclick": addTodo,
						"class":   "bg-blue-600 hover:bg-blue-800 text-white font-bold py-2 px-4 rounded",
					}, Text("Add")), // Button to add a new todo
				),
				Tag("ul", map[string]string{"class": "space-y-2"}, todoItemsInterface...), // List of todos
			),
		))

		// Log that the UI has been rendered
		fmt.Println("Todo App UI Rendered.")
		return c
	})

	// Render the Todo App component to the body of the document
	fmt.Println("Rendering Todo App to Body...")
	RenderToBody(todoApp(Props{}))
	fmt.Println("Todo App Rendered to Body.")
}
