// ./main.go

//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"sync"

	. "github.com/monstercameron/GoWebComponents/example"
	// . "github.com/monstercameron/GoWebComponents/website"
)

// main is the entry point of the program
func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	fmt.Println("🚀 Go Web Components starting...")
	fmt.Println("📊 Loading GoUseAtom Example...")

	// Render the GoUseAtom example to the DOM
	RendertoDom("#app", App)

	// Alternative: Render the website (uncomment the website import above and comment out the example import)
	// RendertoDom("#app", App)

	fmt.Println("✅ GoUseAtom Example rendered successfully")

	// Keep the program alive for WebAssembly event handling
	wg.Wait()
}
