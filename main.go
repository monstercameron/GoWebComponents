// ./main.go

//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"sync"

	"github.com/monstercameron/GoWebComponents/examples"
	"github.com/monstercameron/GoWebComponents/fiber"
)

// Global app reference
var app *fiber.Element

// main is the entry point of the program
func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	fmt.Println("Main: Starting Go Web Components Examples")

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
	fmt.Println("Main: Creating and rendering app component")
	appComponent := examples.GetSimpleStateExamplesApp()
	app = fiber.CreateElement(appComponent, nil)

	// Render the app to the DOM
	fiber.RenderTo("#app", appComponent)

	//

	// Other available examples (commented out)
	// examples.ClickCounterExample()
	// examples.ConcurrentDashboardExample()
	// examples.NetworkMonitoringDashboardExample()

	fmt.Println("Main: App component rendered and reference saved")

	// Show current debug status
	fmt.Println("Debug Status:", fiber.GetDebugStatus())

	fmt.Println("Main: End of main function")

	// Keep the program alive for WebAssembly event handling
	wg.Wait()
}
