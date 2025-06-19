// ./main.go

//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"sync"

	"github.com/monstercameron/GoWebComponents/website"

	"github.com/monstercameron/GoWebComponents/fiber"
)

// Global app reference
var app *fiber.Element

// main is the entry point of the program
func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	fmt.Println("🚀 Go Web Components starting...")

	// Configure debug logging
	fiber.SetDebugNamespacesExclusive(map[string]bool{
		"HOOKS":  false,
		"RENDER": false,
		"MEMORY": false,
		"DOM":    false,
		"FETCH":  false,
		"EVENTS": false,
		"COMMIT": false,
		"FIBER":  false,
	})

	// Enable hot reload for development
	fiber.EnableHotReload(true)

	// Create and render the main app component
	// Hot Reload Test Component - Perfect for testing partial reloading!
	// Uncomment this line to test hot reload vs full reload classification
	// appComponent := examples.StateTest()

	// Other available examples (comment out the line above and uncomment one below)
	// appComponent := examples.GetSimpleStateExamplesApp()
	// examples.ClickCounterExample()
	// examples.ConcurrentDashboardExample()
	// examples.NetworkMonitoringDashboardExample()

	// Render the app to the DOM with router
	fiber.RenderTo("#app", website.AppRouter)

	fmt.Println("✅ App rendered successfully")

	// Keep the program alive for WebAssembly event handling
	wg.Wait()
}
