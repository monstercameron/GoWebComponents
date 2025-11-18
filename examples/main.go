// ./main.go

//go:build js && wasm
// +build js,wasm

package example

import (
	"fmt"
	"sync"

	"github.com/monstercameron/GoWebComponents/render"
)

// Main is the exported entry point for the WASM module
func Main() {
	var wg sync.WaitGroup
	wg.Add(1)

	fmt.Println("🚀 Go Web Components starting...")
	fmt.Println("📊 Loading Counter Example...")

	// Render the counter example to the DOM
	render.To(CounterExample(nil), "#app")

	fmt.Println("✅ Counter Example rendered successfully")

	// Keep the program alive for WebAssembly event handling
	wg.Wait()
}

func main() {
	Main()
}
