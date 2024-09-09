package components

import (
	"fmt"
	"syscall/js"
)

func Example2() {
	fmt.Println("Starting the example 1")

	// Create the component using MakeComponent
	component := MakeComponent(func(self *Component, props int, children ...*Component) *Component {
		// Setup state for the counter, initializing with the props value
		counter, setCounter := AddState(self, "counter", props)

		// Setup the component when it is mounted
		Setup(self, func() {
			fmt.Println("Component has been set up.")
			fmt.Println("Initial counter value:", *counter)
		})

		// Define the click handler using the Function helper
		handleClick := Function(self, "handleClick", func(event js.Value) {
			// Update the counter state
			fmt.Println("handleClick called")
			setCounter(*counter + 1)
			fmt.Println("Counter updated to:", *counter)
		})

		RenderTemplate(self, Tag("div", Attributes{},
			Tag("p", Attributes{}, Text(fmt.Sprintf("Counter: %d", *counter))),
			Tag("button", Attributes{
				"onclick": handleClick,
				"class":   "btn",
			}, Text("Click me to increment")),
		))

		return self
	})

	// Use the InsertComponentIntoDOM function to render and insert the component into the DOM
	InsertComponentIntoDOM(component(0)) // Initial counter value starts at 0
}

func Example1() {
	fmt.Println("Starting Example 2: Tailwind-powered ToDo List")

	// Create the component using MakeComponent
	component := MakeComponent(func(self *Component, props int, children ...*Component) *Component {
		// Setup state for todo list items
		todos, setTodos := AddState(self, "todos", []string{})

		// Local variable to hold the current input value
		newTodo := ""
		var target js.Value

		// Setup the component when it is mounted
		Setup(self, func() {
			fmt.Println("ToDo List component has been set up.")
			fmt.Println("Initial ToDo list:", *todos)
		})

		Watch(self, func() {
			fmt.Println("Looks like the state has changed!")
		}, "todos")
		
		// Function to handle adding a new todo
		handleAddTodo := Function(self, "handleAddTodo", func(event js.Value) {
			if newTodo != "" {
				fmt.Println("Adding new todo:", newTodo)
				*todos = append(*todos, newTodo) // Update the todo list
				setTodos(*todos)                 // Update state
				newTodo = ""                     // Reset input
			}
			target.Set("value", "") // Clear the input field
		})

		handleRemoveTodo := Function(self, "handleRemoveTodo", func(event js.Value) {
			// handleRemoveTodo(1) on the JS side, where event is just an integer
			index := event.Int() // Convert the JS value to an integer
			fmt.Printf("Removing todo at index: %d\n", index)

			// Remove the todo item at the given index
			if index >= 0 && index < len(*todos) {
				*todos = append((*todos)[:index], (*todos)[index+1:]...)
				setTodos(*todos) // Update state
			} else {
				fmt.Printf("Invalid index: %d\n", index)
			}
		})

		fmt.Println("handleRemoveTodo", handleRemoveTodo)

		// Input change handler
		handleInputChange := Function(self, "handleInputChange", func(event js.Value) {
			newTodo = event.Get("target").Get("value").String()
			target = event.Get("target")
			fmt.Println("Input value changed:", newTodo)
		})

		// Render the component
		RenderTemplate(self, Tag("div", Attributes{"class": "p-6 max-w-sm mx-auto bg-white shadow-lg rounded-lg"},
			Tag("h1", Attributes{"class": "text-2xl font-bold mb-4"}, Text("ToDo List")),

			// Input field for new todo
			Tag("div", Attributes{"class": "mb-4"},
				Tag("input", Attributes{
					"type":        "text",
					"placeholder": "Enter a new task",
					"value":       newTodo,
					"class":       "border rounded w-full p-2",
					"oninput":     handleInputChange,
				}),
			),

			// Add button
			Tag("button", Attributes{
				"class":   "bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded",
				"onclick": handleAddTodo,
			}, Text("Add Task")),

			// Render the todo list
			Tag("ul", Attributes{"class": "mt-4 space-y-2"},
				// Iterate over the todo items
				Tag("div", Attributes{}, Text(func() string {
					todoItems := ""
					for i, todo := range *todos {
						todoItems += fmt.Sprintf(`
							<li class="flex justify-between items-center p-2 border-b">
								<span>%s</span>
								<button class="bg-red-500 hover:bg-red-700 text-white font-bold py-1 px-2 rounded" onclick="handleRemoveTodo(%d)">Remove</button>
							</li>
						`, todo, i)
					}
					return todoItems
				}())),
			),
		))

		return self
	})

	// Use the InsertComponentIntoDOM function to render and insert the component into the DOM
	InsertComponentIntoDOM(component(0)) // Initial call
}
