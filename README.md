# Go HTML Renderer

This project demonstrates a simple HTML rendering library and web server implemented in Go. It showcases how to create HTML structures programmatically and serve dynamic web pages without using traditional templating engines.

## Features

- Custom HTML rendering library with a focus on virtual DOM concepts
- Dynamic HTML generation using `NodeInterface` for flexible node management
- Simple web server with multiple routes
- Example of handling complex data structures with `ElementNode` and `TextNode`
- Advanced examples showcasing Go-specific string interpolation and dynamic content generation

## Project Structure

The project consists of two main parts:

1. HTML Rendering Library (`vdom/vdom.go`)
2. Web Server Implementation (`main.go`)

### HTML Rendering Library

The custom HTML rendering library provides a flexible way to create HTML structures in Go. Key features include:

- `NodeInterface` for representing HTML elements and text
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

1. **Virtual DOM Structure**: The library uses a `NodeInterface` to represent HTML elements, allowing for a more structured approach to building HTML. The `ElementNode` and `TextNode` types facilitate the creation of complex HTML structures.

2. **Dynamic HTML Generation**: The `HomePage` function in `example.go` showcases how to create a complete HTML document programmatically, including a header, main content, and footer, using the `Tag` and `Text` functions.

3. **Event Handling**: The library supports event handling through methods that can be added to nodes, allowing for interactive web pages.

## Example

Here's a simple example of how to use the HTML rendering library:

```go
package main

import (
    "fmt"
    "log"
    "yourmodule/vdom"
)

func main() {
    // Create a simple HTML structure
    doc := vdom.Tag("html", nil,
        vdom.Tag("head", nil,
            vdom.Tag("title", nil, vdom.Text("My Page")),
        ),
        vdom.Tag("body", nil,
            vdom.Tag("h1", nil, vdom.Text("Welcome")),
            vdom.Tag("p", map[string]string{"class": "content"}, vdom.Text("This is a paragraph.")),
        ),
    )

    // Render the HTML
    renderedHTML, err := doc.Render(0)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(renderedHTML)
}
```

This example demonstrates how to create a simple HTML document using the library and render it to a string.

## Conclusion

This project provides a flexible and type-safe way to generate HTML in Go without relying on string templates. It's particularly useful for creating dynamic web content or building custom static site generators, leveraging the virtual DOM concepts introduced in the `vdom` package.
