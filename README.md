# Go HTML Renderer

This project demonstrates a modern HTML rendering library and web server implemented in Go. It showcases how to create HTML structures programmatically and serve dynamic web pages without relying on traditional templating engines.

## Features

- Custom HTML rendering library inspired by virtual DOM concepts
- Dynamic HTML generation using `NodeInterface` for flexible node management
- Event handling capabilities for interactive web pages
- Example of a Todo List application with a clean, modern interface and dark mode
- Simple web server implementation to demonstrate rendering and event handling

## Project Structure

The project is divided into two primary components:

1. **HTML Rendering Library (`vdom/vdom.go`)**
2. **Web Server Implementation (`main.go`)**

### HTML Rendering Library

The custom HTML rendering library provides a powerful and flexible way to create HTML structures in Go. Key features include:

- **`NodeInterface`**: A core interface representing HTML elements and text nodes.
- **`ElementNode` and `TextNode`**: Structs for creating and managing HTML elements and text content.
- **Rendering Methods**: Convert node structures into HTML strings, with support for HTML escaping and validation.
- **Void Element Handling**: Proper management of self-closing tags.
- **Helper Functions**: Simplify the creation of HTML elements (`Tag`) and text nodes (`Text`).

### Web Server Implementation

The `main.go` file contains a simple web server that demonstrates the usage of the HTML rendering library, including dynamic content generation and event handling.

- **Multiple Routes**: Each route demonstrates a different aspect of the rendering library.
- **Dynamic Content**: Examples of generating HTML content dynamically using Go.
- **Interactive Elements**: Showcase of event handling capabilities within the library.

## Routes

- `/`: Home page
- `/simple`: Simple HTML rendering example
- `/complex`: Complex HTML rendering example with dynamic data
- `/todo`: Todo List application with event handling and state management
- `/advanced`: Advanced examples featuring Go-specific string interpolation and dynamic content generation

## Usage

To run the project:

1. Ensure Go is installed on your system.
2. Clone the repository.
3. Navigate to the project directory.
4. Build the WebAssembly target using the following command:

   ```sh
   GOOS=js GOARCH=wasm go build -o wasm/main.wasm
   ```

5. Start the web server:

   ```sh
   go run main.go
   ```

6. Open your web browser and visit `http://localhost:8080`.

## Example: Todo List Application

One of the standout examples in this project is a Todo List application that demonstrates the library's ability to manage state and handle events within a dynamic HTML structure.

### Key Components:

1. **State Management**: Using `AddState`, the application manages the state of the todo list, including the list of todos, the next ID, and the input value.
   
2. **Event Handling**: The application defines event handlers for adding, toggling, and removing todos using the `Function` method, showcasing the integration of JavaScript events within Go.
   
3. **Dynamic Rendering**: The entire UI, including the list of todos, is dynamically generated and updated based on the application state.

### Example Code (From `Example2`):

```go
func Example2() {
    fmt.Println("Initializing Todo App Component...")

    todoApp := CreateComponent(func(c *Component, _ Props, _ ...*Component) *Component {
        todos, setTodos := AddState(c, "todos", []Todo{})
        nextID, setNextID := AddState(c, "nextID", 1)
        inputValue, setInputValue := AddState(c, "inputValue", "")

        addTodo := Function(c, "addTodo", func(_ js.Value) {
            if *inputValue != "" {
                newTodo := Todo{ID: *nextID, Text: *inputValue, Completed: false}
                setTodos(append(*todos, newTodo))
                setNextID(*nextID + 1)
                setInputValue("")
            }
        })

        toggleTodo := Function(c, "toggleTodo", func(event js.Value) {
            idStr := event.Get("target").Get("dataset").Get("id").String()
            id, err := strconv.Atoi(idStr)
            if err != nil {
                fmt.Printf("Error converting id to integer: %s\n", err)
                return
            }
            newTodos := make([]Todo, len(*todos))
            copy(newTodos, *todos)
            for i, todo := range newTodos {
                if todo.ID == id {
                    newTodos[i].Completed = !newTodos[i].Completed
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
            newTodos := make([]Todo, 0, len(*todos)-1)
            for _, todo := range *todos {
                if todo.ID != id {
                    newTodos = append(newTodos, todo)
                }
            }
            setTodos(newTodos)
        })

        handleInputChange := Function(c, "handleInputChange", func(event js.Value) {
            newValue := event.Get("target").Get("value").String()
            setInputValue(newValue)
        })

        var todoItems []NodeInterface
        for _, todo := range *todos {
            todoItems = append(todoItems, Tag("li", map[string]string{"class": "flex items-center justify-between p-2 border-b border-gray-700"},
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

        todoItemsInterface := make([]interface{}, len(todoItems))
        for i, item := range todoItems {
            todoItemsInterface[i] = item
        }

        Render(c, Tag("div", map[string]string{"class": "min-h-screen bg-gray-900 text-gray-100 p-4 flex flex-col items-center"},
            Tag("div", map[string]string{"class": "w-full max-w-md"},
                Tag("h1", map[string]string{"class": "text-2xl font-semibold mb-4 text-center text-gray-200"}, Text("Todo List")),
                Tag("div", map[string]string{"class": "flex mb-4"},
                    Tag("input", map[string]string{
                        "type":        "text",
                        "placeholder": "Add a new todo",
                        "value":       *inputValue,
                        "oninput":     handleInputChange,
                        "class":       "flex-grow mr-2 p-2 border rounded bg-gray-800 text-gray-100 border-gray-700",
                    }),
                    Tag("button", map[string]string{
                        "onclick": addTodo,
                        "class":   "bg-blue-600 hover:bg-blue-800 text-white font-bold py-2 px-4 rounded",
                    }, Text("Add")),
                ),
                Tag("ul", map[string]string{"class": "space-y-2"}, todoItemsInterface...),
            ),
        ))

        return c
    })

    RenderToBody(todoApp(Props{}))
}
```

### Explanation:

- **State Management**: The example demonstrates how to manage state in a Go-based web application, allowing the user to add, toggle, and remove todos.
- **Dynamic UI Rendering**: The UI is dynamically generated based on the state, showcasing the power of the custom rendering library.
- **Event Handling**: Events like clicking and typing are handled directly within Go, providing a seamless development experience.

## Conclusion

This project provides a robust, type-safe way to generate HTML and manage dynamic web content in Go without the need for traditional templating engines. It’s particularly well-suited for creating interactive web applications or custom static site generators using modern web development practices.