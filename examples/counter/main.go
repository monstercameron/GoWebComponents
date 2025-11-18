//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	fmt.Println("🚀 Go Web Components Click Counter starting...")
	fmt.Println("📊 Loading Click Counter...")

	// Render the click counter app
	ClickCounterExample()

	fmt.Println("✅ Click Counter rendered successfully")

	// Keep the program alive for WebAssembly event handling
	wg.Wait()
}
