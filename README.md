# GoWebComponents – Build Dynamic UIs with Go and WebAssembly

Welcome to **GoWebComponents**, a fast, lightweight, and powerful framework for building dynamic web interfaces using **Go** and **WebAssembly**. With GoWebComponents, you can easily generate HTML, manage state, and handle events directly in Go, without the need for JavaScript frameworks. Experience the simplicity and performance of Go while creating fully interactive web applications.

## Why Go and WebAssembly?

- **Simplicity**: Go is famous for being easy to learn, read, and write. WebAssembly allows us to harness this simplicity and take it directly to the browser without introducing JavaScript complexity.
- **Type Safety Built-In**: Forget about adding yet another build step for type safety. Go has all the types and safety you need, right out of the box.
- **Lightweight**: No bloated dependency chains or gigantic node_modules folders. The entire application can be compiled down to a small and efficient WebAssembly binary.
- **Performance**: WebAssembly runs at near-native speeds, making your web applications fast and responsive.

## Project Structure

- **HTML Rendering Library (`components/html.go`)**: A lightweight, custom rendering engine built to create dynamic HTML in Go.
- **Example Application (`components/example.go`)**: A simple, interactive ToDo list application showcasing how to manage state and handle user events.

## How to Install

### Prerequisites

- Go installed on your system (version 1.18+ recommended).
- A modern browser that supports WebAssembly (most do).

### Installation Steps

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/monstercameron/go-html.git
   cd go-html
   ```

2. **Build the WebAssembly Target**:
   Run the provided `build.sh` script to compile the Go code into WebAssembly:
   ```bash
   ./build.sh
   ```

3. **Open the Application**:
   Simply open `wasm/index.html` in your browser to run the app. No need for a server, just double-click the file and watch your application run natively in the browser.

## Example: ToDo List in Go with Tailwind CSS

### Tutorial: Example1

In this example, we build a simple but powerful ToDo list application using Go and WebAssembly, powered by Tailwind CSS for styling. This demonstrates how Go can handle dynamic user input, update state, and manage the browser’s DOM directly.

#### Key Features:
- **State Management**: Add and remove tasks with state tracking directly in Go.
- **Event Handling**: Handle user input and button clicks via Go functions.
- **Dynamic Rendering**: The DOM updates dynamically as the ToDo list changes, all without page reloads or complex re-renders.
- **Tailwind CSS**: Beautiful, responsive UI styling without writing custom CSS.

Here’s the code:

```go
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
			fmt.Println("Setup: ToDo List component has been set up.")
			*todos = append(*todos, "Learn Go", "Build a Web App", "Deploy to Production")
			fmt.Println("Setup: Initial ToDo list:", *todos)
		})

		Watch(self, func() {
			fmt.Println("Watch: looks like todos has changed, or first render")
			fmt.Printf("Watch: Current todos: %v\n", *todos)
		}, "todos")

		// Function to handle adding a new todo
		handleAddTodo := Function(self, "handleAddTodo", func(event js.Value) {
			if newTodo != "" {
				fmt.Println("Adding new todo:", newTodo)
				*todos = append(*todos, newTodo)
				setTodos(*todos)
				newTodo = ""
			}
			target.Set("value", "") // Clear the input field
		})

		Function(self, "handleRemoveTodo", func(event js.Value) {
			index := event.Int() 
			fmt.Printf("Removing todo at index: %d\n", index)

			if index >= 0 && index < len(*todos) {
				*todos = append((*todos)[:index], (*todos)[index+1:]...)
				setTodos(*todos)
			} else {
				fmt.Printf("Invalid index: %d\n", index)
			}
		})

		handleInputChange := Function(self, "handleInputChange", func(event js.Value) {
			newTodo = event.Get("target").Get("value").String()
			target = event.Get("target")
			fmt.Println("Input value changed:", newTodo)
		})

		// Render the component
		RenderTemplate(self, Tag("div", Attributes{"class": "p-6 max-w-sm mx-auto bg-white shadow-lg rounded-lg"},
			Tag("h1", Attributes{"class": "text-2xl font-bold mb-4"}, Text("ToDo List")),
			Tag("div", Attributes{"class": "mb-4"},
				Tag("input", Attributes{
					"type":        "text",
					"placeholder": "Enter a new task",
					"value":       newTodo,
					"class":       "border rounded w-full p-2",
					"oninput":     handleInputChange,
				}),
			),
			Tag("button", Attributes{
				"class":   "bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded",
				"onclick": handleAddTodo,
			}, Text("Add Task")),
			Tag("ul", Attributes{"class": "mt-4 space-y-2"},
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
	InsertComponentIntoDOM(component(0)) 
}
```

### Important Notes:
- **State Management**: The `AddState` function manages the current state of the todo list items. It updates the state when a new task is added or removed.
- **Event Handling**: You handle user events like input changes, adding tasks, and removing tasks using Go functions. No JavaScript or React needed.
- **Tailwind CSS**: Tailwind CSS makes it easy to create a polished, responsive UI without writing custom styles.

## How to Build

Compile the Go code into WebAssembly with a single command:

```bash
GOOS=js GOARCH=wasm go build -o wasm/main.wasm
```

This builds the `main.wasm` file, which the browser runs as WebAssembly.

## How to Deploy

Simply **open `wasm/index.html`** in your browser. No servers, no complex setups – just open the file and see your Go-powered ToDo list come to life.

---

## Conclusion

**Go HTML Renderer** lets you build interactive web applications with Go and WebAssembly, without the overhead of JavaScript frameworks. It’s type-safe, fast, and minimal – just what you need to focus on building rather than debugging toolchains. Take advantage of WebAssembly's power and Go’s simplicity to create modern web experiences with ease.

