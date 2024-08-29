# Go HTML Renderer

This project demonstrates a simple HTML rendering library and web server implemented in Go. It showcases how to create HTML structures programmatically and serve dynamic web pages without using traditional templating engines.

## Features

- Custom HTML rendering library
- Dynamic HTML generation
- Simple web server with multiple routes
- Example of handling complex data structures
- Advanced examples showcasing Go-specific string interpolation and dynamic content generation

## Project Structure

The project consists of two main parts:

1. HTML Rendering Library (`html/html.go`)
2. Web Server Implementation (`main.go`)

### HTML Rendering Library

The custom HTML rendering library provides a flexible way to create HTML structures in Go. Key features include:

- `Node` interface for representing HTML elements and text
- `ElementNode` and `TextNode` structs for creating HTML elements and text content
- Methods for rendering nodes to HTML strings
- HTML escaping and validation of tag and attribute names
- Safe handling of void elements (self-closing tags)
- Helper functions for creating HTML elements and text nodes

### Web Server Implementation

The `main.go` file implements a simple web server that demonstrates the usage of the HTML rendering library. It includes:

- Multiple routes showcasing different aspects of the library
- Dynamic content generation
- Examples of string interpolation and advanced Go features

## Routes

- `/`: Home page
- `/simple`: Simple HTML rendering example
- `/complex`: Complex HTML rendering example with dynamic data
- `/advanced`: Advanced examples showcasing string interpolation and dynamic content generation

## Usage

To run the project:

1. Ensure you have Go installed on your system.
2. Clone the repository.
3. Navigate to the project directory.
4. Build the WebAssembly target using the following command:

   ```
   GOOS=js GOARCH=wasm go build -o wasm/main.wasm
   ```

5. Run the following command to start the web server:

   ```
   go run main.go
   ```

6. Open a web browser and visit `http://localhost:8080`.

## Advanced Examples

The `/advanced` route demonstrates several Go-specific features and string interpolation techniques:

1. Adding an event listener for a click event:
   ```go
   func GenerateClicker() html.Node {
       // Initialize state with an initial count of 0.
       count, setCount, countId := html.UseState(0)

       // Define the JavaScript functions that will be exposed to the global scope.
       html.WasmFunc("increment", func() {
           // Increment the count value and update the display.
           setCount(*count + 1)
       })

       // Construct and return the HTML structure for the clicker component.
       return html.HTML("div", map[string]string{"class": "bg-white p-8 rounded-lg shadow-md text-center"},
           html.HTML("h1", map[string]string{"class": "text-3xl font-bold mb-4"},
               html.Text("Clicker"),
           ),
           html.HTML("p", map[string]string{"class": "text-xl mb-4"},
               html.Text("Count: "),
               html.HTML("span", map[string]string{"id": "count",
                   "data-state": countId, "class": "font-bold"},
                   html.Text(fmt.Sprint(*count)), // Render the initial count value.
               ),
           ),
           html.HTML("button", map[string]string{
               "onclick": "increment()",
               "class":   "bg-green-500 hover:bg-green-600 text-white font-bold py-2 px-4 rounded mr-2 focus:outline-none focus:shadow-outline",
           },
               html.Text("+"),
           ),
       )
   }
   ```

2. Using Go's string formatting for dynamic attributes:
   ```go
   func DynamicAttributes() html.Node {
       // Generate a random number and use it in an attribute.
       randomNumber := rand.Intn(100)
       randomAttr := map[string]string{"data-random": fmt.Sprintf("random-%d", randomNumber)}

       return html.HTML("div", randomAttr,
           html.Text(fmt.Sprintf("Random number is %d", randomNumber)),
       )
   }
   ```

These examples showcase how to use Go's string formatting and event handling to create dynamic HTML content.

## Example

Here's a simple example of how to use the HTML rendering library:

```go
package main

import (
    "fmt"
    "log"
    "yourmodule/html"
)

func main() {
    // Create a simple HTML structure
    doc := html.HTML("html", nil,
        html.HTML("head", nil,
            html.HTML("title", nil, html.Text("My Page")),
        ),
        html.HTML("body", nil,
            html.HTML("h1", nil, html.Text("Welcome")),
            html.HTML("p", map[string]string{"class": "content"}, html.Text("This is a paragraph.")),
        ),
    )

    // Render the HTML
    renderedHTML, err := doc.Render()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(renderedHTML)
}
```

This example demonstrates how to create a simple HTML document using the library and render it to a string.

## Conclusion

This project provides a flexible and type-safe way to generate HTML in Go without relying on string templates. It's particularly useful for creating dynamic web content or building custom static site generators.
