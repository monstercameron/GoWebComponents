# Go WebAssembly in the Browser

This project demonstrates how to run a Go program compiled to WebAssembly (Wasm) in the browser.

## Prerequisites

- Go installed on your system. [Download Go](https://golang.org/dl/)
- Visual Studio Code with the Live Server extension.

## Setup Instructions

### 1. Set Up the Go Environment for WebAssembly

Set up your Go environment to support WebAssembly:

```sh
go env -w GOOS=js GOARCH=wasm
```

### 2. Write Your Go Code

Create a Go source file (`main.go`):

```go
package main

import (
    "fmt"
    "syscall/js"
)

func main() {
    fmt.Println("Hello, WebAssembly from Go!")

    // Example: Interact with JavaScript
    js.Global().Get("document").Call("write", "Hello, WebAssembly from Go!")
}
```

### 3. Compile Go Code to WebAssembly

Compile your Go program to a WebAssembly module:

```sh
GOOS=js GOARCH=wasm go build -o main.wasm
```

### 4. Get the WebAssembly JavaScript Support File

Copy the `wasm_exec.js` file from your Go installation to your project directory. You can find it in the following path:

```
$GOROOT/misc/wasm/wasm_exec.js
```

### 5. Create an HTML File to Load WebAssembly

Create an `index.html` file:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go WebAssembly Example</title>
    <script src="wasm_exec.js"></script>
    <script>
        const go = new Go();

        WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
            go.run(result.instance);
        }).catch(console.error);
    </script>
</head>
<body>
</body>
</html>
```

### 6. Serve the HTML File Using Live Server

- Open the project directory in Visual Studio Code.
- Right-click on `index.html` and select "Open with Live Server".

### 7. View in the Browser

- Your default browser should open automatically.
- Navigate to the provided local server URL (e.g., `http://127.0.0.1:5500`).
- You should see the output from your Go WebAssembly code in the browser.

### 8. Debugging Tips

- **Console Logs**: Use `fmt.Println` in Go or `console.log` in JavaScript to log output for debugging.
- **Browser Console**: Open the browser's Developer Tools to see any errors or logs.

## Conclusion

You have successfully run a Go WebAssembly program in the browser using Live Server in VS Code. You can expand this setup to build more complex web applications.

Happy coding!