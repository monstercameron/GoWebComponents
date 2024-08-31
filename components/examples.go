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
	fmt.Println("Initializing Todo App Component...")

	todoApp := CreateComponent(func(c *Component, _ Props, _ ...*Component) *Component {
		// Initialize state
		todos, setTodos := AddState(c, "todos", []Todo{})
		nextID, setNextID := AddState(c, "nextID", 1)
		inputValue, setInputValue := AddState(c, "inputValue", "")

		// Define functions for adding, toggling, and removing todos
		addTodo := Function(c, "addTodo", func(event js.Value) {
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
			idStr := event.Get("target").Get("dataset").Get("id").String()

			id, err := strconv.Atoi(idStr)
			if err != nil {
				fmt.Printf("Error converting id to integer: %s\n", err)
				return
			}

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
			setTodos(newTodos)
		})

		removeTodo := Function(c, "removeTodo", func(event js.Value) {
			idStr := event.Get("target").Get("dataset").Get("id").String()

			id, err := strconv.Atoi(idStr)
			if err != nil {
				fmt.Printf("Error converting id to integer: %s\n", err)
				return
			}

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
			preserveFocus(event, func() {
				newValue := event.Get("target").Get("value").String()
				fmt.Printf("Input Changed: %s\n", newValue)
				setInputValue(newValue)
			})
		})

		// Compose the todo items list
		var todoItems []NodeInterface
		for _, todo := range *todos {
			fmt.Printf("Rendering Todo: %+v\n", todo)

			// Create the base attributes map
			checkboxAttrs := map[string]string{
				"id":       fmt.Sprintf("todo-checkbox-%d", todo.ID),
				"type":     "checkbox",
				"onchange": toggleTodo,
				"data-id":  fmt.Sprintf("%d", todo.ID),
				"class":    "mr-2",
			}

			// If the todo is completed, add the "checked" attribute
			if todo.Completed {
				checkboxAttrs["checked"] = ""
			}

			todoItems = append(todoItems, Tag("li", map[string]string{"class": "flex items-center justify-between p-2 border-b border-gray-700"},
				Tag("input", checkboxAttrs),
				Tag("span", map[string]string{
					"class": fmt.Sprintf("flex-grow %s", map[bool]string{true: "line-through text-gray-500", false: ""}[todo.Completed]),
				}, Text(todo.Text)),
				Tag("button", map[string]string{
					"id":      fmt.Sprintf("remove-todo-%d", todo.ID),
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
					}),
					Tag("button", map[string]string{
						"id":      "add-todo-button",
						"onclick": addTodo,
						"class":   "bg-blue-600 hover:bg-blue-800 text-white font-bold py-2 px-4 rounded",
					}, Text("Add")),
				),
				Tag("ul", map[string]string{"class": "space-y-2"}, todoItemsInterface...), // Pass the composed todo items
			),
		))

		fmt.Println("Todo App UI Rendered.")
		return c
	})

	fmt.Println("Rendering Todo App to Body...")
	RenderToBody(todoApp(Props{}))
	fmt.Println("Todo App Rendered to Body.")
}
