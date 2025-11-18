// ./main.go

//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"sync"
)

// main is the entry point for the WASM module
func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	fmt.Println("🚀 Go Web Components Blog Landing Page starting...")
	fmt.Println("📊 Loading Blog Landing Page...")

	// Render the blog landing page to the DOM
	BlogLandingPage()

	fmt.Println("✅ Blog Landing Page rendered successfully")

	// Keep the program alive for WebAssembly event handling
	wg.Wait()
}
