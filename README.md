# Go HTML Renderer

This project demonstrates a simple HTML rendering library and web server implemented in Go. It showcases how to create HTML structures programmatically and serve dynamic web pages without using traditional templating engines.

## Features

- Custom HTML rendering library
- Dynamic HTML generation
- Simple web server with multiple routes
- Example of handling complex data structures

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

### Web Server Implementation

The web server demonstrates the usage of the HTML rendering library. It includes:

- A main function that sets up routes and starts the server
- Multiple handler functions for different pages (Home, Simple Example, Complex Example)
- Dynamic generation of navigation links
- Example of rendering complex structures (e.g., select dropdowns)

## Usage

To run the project:

1. Ensure you have Go installed on your system
2. Clone the repository
3. Navigate to the project directory
4. Run the following command:

   ```go run main.go```

5. Open a web browser and visit `http://localhost:8080`

## Routes

- `/`: Home page
- `/simple`: Simple HTML rendering example
- `/complex`: Complex HTML rendering example with dynamic data