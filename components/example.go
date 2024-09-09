package components

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

// Example1 is a function that demonstrates creating a Tailwind-powered ToDo list using Go and WebAssembly.
// It creates a component with state management, event handling, and dynamic rendering.
func Example2() {
	fmt.Println("Starting Example 2: Tailwind-powered ToDo List")

	// Create the component using the MakeComponent function
	// This function defines the structure, state, and behavior of the ToDo component.
	component := MakeComponent(func(self *Component, props int, children ...*Component) *Component {
		// Setup state for todo list items
		// `todos` holds the current list of tasks, and `setTodos` is a setter function to update this state.
		todos, setTodos := AddState(self, "todos", []string{})

		// Local variable to hold the current input value for the new todo item
		newTodo := ""
		var target js.Value // Holds the reference to the input field in the DOM

		// Setup the component when it is mounted
		// This lifecycle function runs only once when the component is first added to the DOM.
		Setup(self, func() {
			fmt.Println("Setup: ToDo List component has been set up.")
			// Pre-populate the todo list with demo tasks.
			*todos = append(*todos, "Learn Go", "Build a Web App", "Deploy to Production")
			fmt.Println("Setup: Initial ToDo list:", *todos)
		})

		// Watch for changes to the `todos` state
		// The Watch function triggers a callback whenever the `todos` state changes.
		Watch(self, func() {
			fmt.Println("Watch: looks like todos has changed, or first render")
			fmt.Printf("Watch: Current todos: %v\n", *todos)
		}, "todos")

		// Function to handle adding a new todo
		// This is triggered when the "Add Task" button is clicked.
		handleAddTodo := Function(self, "handleAddTodo", func(event js.Value) {
			// Check if the new todo item is not empty
			if newTodo != "" {
				fmt.Println("Adding new todo:", newTodo)
				// Append the new todo to the current list and update the state
				*todos = append(*todos, newTodo)
				setTodos(*todos) // Update state to trigger re-render
				newTodo = ""     // Clear the input value after adding the task
			}
			// Reset the input field in the DOM by setting its value to an empty string
			target.Set("value", "")
		})

		// Function to handle removing a todo item by index
		// This is triggered when the "Remove" button for a todo item is clicked.
		Function(self, "handleRemoveTodo", func(event js.Value) {
			// The event passed from JavaScript contains the index of the todo to remove
			index := event.Int() // Convert the JS value to an integer
			fmt.Printf("Removing todo at index: %d\n", index)

			// Check if the index is valid and within the range of the todo list
			if index >= 0 && index < len(*todos) {
				// Remove the todo item at the given index
				*todos = append((*todos)[:index], (*todos)[index+1:]...)
				setTodos(*todos) // Update state to reflect the change
			} else {
				fmt.Printf("Invalid index: %d\n", index)
			}
		})

		// Input change handler for the new todo item
		// This is triggered when the user types into the input field.
		handleInputChange := Function(self, "handleInputChange", func(event js.Value) {
			// Get the current value from the input field
			newTodo = event.Get("target").Get("value").String()
			target = event.Get("target") // Store the target element to manipulate later
			fmt.Println("Input value changed:", newTodo)
		})

		// Render the component using Tailwind CSS for styling
		// This defines the HTML structure of the component, including input fields, buttons, and the todo list.
		RenderTemplate(self, Tag("div", Attributes{"class": "p-6 max-w-sm mx-auto bg-white shadow-lg rounded-lg"},
			// Header for the ToDo list
			Tag("h1", Attributes{"class": "text-2xl font-bold mb-4"}, Text("ToDo List")),

			// Input field for entering new todo tasks
			Tag("div", Attributes{"class": "mb-4"},
				Tag("input", Attributes{
					"type":        "text",                      // Input type is text
					"placeholder": "Enter a new task",          // Placeholder text
					"value":       newTodo,                     // Bind the value to the newTodo variable
					"class":       "border rounded w-full p-2", // Tailwind classes for styling
					"oninput":     handleInputChange,           // Handle input changes
				}),
			),

			// Button to add the new task to the todo list
			Tag("button", Attributes{
				"class":   "bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded", // Tailwind classes for button styling
				"onclick": handleAddTodo,                                                          // Handle click event to add the task
			}, Text("Add Task")),

			// Render the list of todo items dynamically
			Tag("ul", Attributes{"class": "mt-4 space-y-2"}, // Tailwind classes for list styling
				// Iterate over the todo items and generate HTML for each task
				Tag("div", Attributes{}, Text(func() string {
					todoItems := ""
					// Loop through the todos and create an <li> for each item
					for i, todo := range *todos {
						todoItems += fmt.Sprintf(`
							<li class="flex justify-between items-center p-2 border-b">
								<span>%s</span> <!-- Display the task text -->
								<button class="bg-red-500 hover:bg-red-700 text-white font-bold py-1 px-2 rounded" onclick="handleRemoveTodo(%d)">Remove</button> <!-- Remove button for each task -->
							</li>
						`, todo, i)
					}
					return todoItems
				}())),
			),
		))

		// Return the component after rendering
		return self
	})

	// Insert the component into the DOM at the specified root element
	InsertComponentIntoDOM(component(0)) // Call the component with the initial props
}

func Example1() {
	fmt.Println("Starting Example 2: Modern ToDo List with Editing and Persistence")

	// Helper function to return the button label based on completion status.
	ifCompleted := func(completed bool) string {
		if completed {
			return "Undo"
		}
		return "Complete"
	}

	// Converts the todos slice (which is a slice of map[string]interface{}) to a JSON string.
	TodosToJSONString := func(todos []map[string]interface{}) string {
		// Marshal the todos slice into a JSON string
		jsonData, err := json.Marshal(todos)
		if err != nil {
			fmt.Println("Error serializing todos to JSON:", err)
			return "[]"
		}
		return string(jsonData)
	}

	// Parses a JSON string into a slice of map[string]interface{} representing todos.
	ParseJSONStringToTodos := func(jsonStr string) []map[string]interface{} {
		var todos []map[string]interface{}
		// Unmarshal the JSON string into the todos slice
		err := json.Unmarshal([]byte(jsonStr), &todos)
		if err != nil {
			fmt.Println("Error parsing JSON string to todos:", err)
			return []map[string]interface{}{}
		}
		return todos
	}

	// Main component definition using MakeComponent.
	component := MakeComponent(func(self *Component, props int, children ...*Component) *Component {
		// Initialize state for todos and the newTodo input.
		todos, setTodos := AddState(self, "todos", []map[string]interface{}{})
		newTodo := ""
		var target js.Value

		// Setup lifecycle function to load todos from localStorage on component mount.
		Setup(self, func() {
			storedTodos := js.Global().Get("localStorage").Call("getItem", "todos")
			// If todos are stored in localStorage, load them.
			if storedTodos.Truthy() {
				todosFromStorage := ParseJSONStringToTodos(storedTodos.String())
				*todos = append(*todos, todosFromStorage...)
			} else {
				// Initialize default todos if none are stored.
				*todos = []map[string]interface{}{
					{"text": "Learn Go", "completed": false, "editing": false},
					{"text": "Build a Web App", "completed": false, "editing": false},
				}
			}
		})

		// Watch for changes in the todos state and store them in localStorage.
		Watch(self, func() {
			js.Global().Get("localStorage").Call("setItem", "todos", TodosToJSONString(*todos))
		}, "todos")

		// Function to handle adding a new todo when the "Add" button is clicked or Enter is pressed.
		handleAddTodo := Function(self, "handleAddTodo", func(event js.Value) {
			// Check if newTodo is not empty.
			if newTodo == "" || len(newTodo) == 0 {
				fmt.Println("Cannot add an empty task.")
				return
			}

			// Add the new todo to the list and update the state.
			*todos = append(*todos, map[string]interface{}{
				"text":      newTodo,
				"completed": false,
				"editing":   false,
			})
			setTodos(*todos)

			// Clear the input field after adding the todo.
			newTodo = ""

			// Ensure the target is valid before attempting to clear the input field.
			if !target.IsUndefined() && !target.IsNull() {
				target.Set("value", "")
			} else {
				fmt.Println("Target input field is undefined or null.")
			}
		})

		// Function to toggle the completion status of a todo item.
		Function(self, "handleToggleComplete", func(event js.Value) {
			index := event.Int()
			// Check if the index is valid and toggle the completion status.
			if index >= 0 && index < len(*todos) {
				(*todos)[index]["completed"] = !(*todos)[index]["completed"].(bool)
				setTodos(*todos)
			}
		})

		// Function to handle editing a todo.
		Function(self, "handleEditTodo", func(event js.Value) {
			index := event.Int()
			// Set the "editing" status to true for the selected todo.
			if index >= 0 && index < len(*todos) {
				(*todos)[index]["editing"] = true
				setTodos(*todos)
			}
		})

		// Global function to handle saving an edit on blur.
		js.Global().Set("handleSaveEdit", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// Expecting two arguments: event and indexValue.
			if len(args) != 2 {
				fmt.Println("handleSaveEdit expects 2 arguments: event and indexValue")
				return nil
			}

			event := args[0]      // The event object.
			indexValue := args[1] // The second argument should be the index.

			// Convert indexValue to an integer.
			index := indexValue.Int()
			// Check if the index is valid and save the updated todo.
			if index >= 0 && index < len(*todos) {
				(*todos)[index]["editing"] = false
				newValue := event.Get("target").Get("value").String()
				(*todos)[index]["text"] = newValue
				setTodos(*todos)
			} else {
				fmt.Printf("Invalid index: %d\n", index)
			}

			return nil
		}))

		// Function to remove a todo item.
		Function(self, "handleRemoveTodo", func(event js.Value) {
			index := event.Int()
			// Check if the index is valid and remove the todo.
			if index >= 0 && index < len(*todos) {
				*todos = append((*todos)[:index], (*todos)[index+1:]...)
				setTodos(*todos)
			}
		})

		// Function to handle the Enter key press for adding a new todo.
		handleEnterKey := Function(self, "handleEnterKey", func(event js.Value) {
			// Check if the key pressed is "Enter".
			if event.Get("key").String() == "Enter" {
				js.Global().Call("handleAddTodo", event)
			}
		})

		// Function to handle changes in the input field.
		handleInputChange := Function(self, "handleInputChange", func(event js.Value) {
			newTodo = event.Get("target").Get("value").String()
			target = event.Get("target")
		})

		// Render the ToDo List component.
		RenderTemplate(self, Tag("div", Attributes{
			"class": "min-h-screen bg-gradient-to-r from-blue-500 via-blue-600 to-purple-700 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8",
		},
			Tag("div", Attributes{"class": "max-w-lg w-full space-y-8 bg-white p-10 rounded-xl shadow-lg"}, // Increased max width to lg.
				Tag("h1", Attributes{"class": "text-4xl font-extrabold text-gray-900 text-center"}, Text("My Modern ToDo List")),
				Tag("div", Attributes{"class": "mb-4 flex"},
					Tag("input", Attributes{
						"type":        "text",
						"placeholder": "Enter a new task",
						"value":       newTodo,
						"class":       "flex-grow border rounded p-3 text-lg focus:outline-none focus:ring-2 focus:ring-purple-600",
						"oninput":     handleInputChange,
						"onkeypress":  handleEnterKey,
					}),
					Tag("button", Attributes{
						"class":   "ml-3 bg-purple-600 hover:bg-purple-800 text-white font-bold py-3 px-6 rounded transition-all ease-in-out duration-200 transform hover:scale-105",
						"onclick": handleAddTodo,
					}, Text("Add")),
				),
				Tag("ul", Attributes{"class": "space-y-4"},
					Tag("div", Attributes{}, Text(func() string {
						todoItems := ""
						for i, todo := range *todos {
							completed := ""
							if todo["completed"].(bool) {
								completed = "line-through text-gray-500"
							}
							if todo["editing"].(bool) {
								// Render the input field for editing.
								todoItems += fmt.Sprintf(`
									<li class="flex items-center justify-between bg-gray-100 p-4 rounded-lg shadow-md">
										<input type="text" value="%s" class="flex-grow border rounded p-2 text-lg" onblur="handleSaveEdit(event, %d)">
									</li>
								`, todo["text"], i)
							} else {
								// Render the todo item with action buttons.
								todoItems += fmt.Sprintf(`
									<li class="flex items-center justify-between bg-white p-4 rounded-lg shadow-md">
										<span class="flex-grow %s">%s</span>
										<div class="flex space-x-2"> <!-- Flex container for buttons with spacing -->
											<button class="bg-green-500 hover:bg-green-700 text-white font-bold py-1 px-3 rounded" onclick="handleToggleComplete(%d)">%s</button>
											<button class="bg-yellow-500 hover:bg-yellow-700 text-white font-bold py-1 px-3 rounded" onclick="handleEditTodo(%d)">Edit</button>
											<button class="bg-red-500 hover:bg-red-700 text-white font-bold py-1 px-3 rounded" onclick="handleRemoveTodo(%d)">Remove</button>
										</div>
									</li>
								`, completed, todo["text"], i, ifCompleted(todo["completed"].(bool)), i, i)
							}
						}
						return todoItems
					}())),
				),
			),
		))

		return self
	})

	// Insert the component into the DOM.
	InsertComponentIntoDOM(component(0))
}

