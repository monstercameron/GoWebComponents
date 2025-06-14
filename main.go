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

// main is the entry point of the program.
// It initializes a WaitGroup, prints a message, calls example functions,
// and waits for the WaitGroup to complete before exiting.
func main() {
	// Initialize a WaitGroup to simulate waiting for asynchronous tasks in the WebAssembly environment.
	var wg sync.WaitGroup

	// Add(1) indicates that we're waiting for 1 operation to complete.
	// In this case, it is just a placeholder for blocking the main function.
	wg.Add(1)

	// Print a message indicating the start of the program.
	fmt.Println("Main: Starting Go Web Components Examples")

	// Configure debug logging with namespaces
	// Use SetDebugNamespacesExclusive to have full control over individual namespaces
	// This disables global debug and only enables the namespaces you set to true
	fiber.SetDebugNamespacesExclusive(map[string]bool{
		"HOOKS":  false, // Disable hook debugging
		"RENDER": false, // Disable render debugging
		"MEMORY": false, // Disable memory debugging (can be noisy)
		"DOM":    false, // Disable DOM debugging
		"FETCH":  false, // Disable fetch debugging
		"EVENTS": false, // Disable event debugging
		"COMMIT": false, // Disable commit debugging
		"FIBER":  false, // Disable fiber debugging
	})

	// Or enable all debugging globally
	// fiber.EnableAllDebug()

	// Call the example functions from the examples package
	examples.SimpleStateExamplesDemo() // Simple GoUseState Examples
	// examples.ClickCounterExample()               // Click Counter Example
	// examples.ConcurrentDashboardExample()        // Concurrent Dashboard Example
	// examples.NetworkMonitoringDashboardExample() // Network Traffic Monitoring Dashboard

	// Show current debug status
	fmt.Println("Debug Status:", fiber.GetDebugStatus())

	// Print a message indicating the end of the main function logic.
	fmt.Println("Main: End of main function")

	// Wait() blocks the main function from exiting immediately.
	// In WebAssembly, this is used to keep the program alive for event handling and state management,
	// as WebAssembly is single-threaded and doesn't have native goroutines running in parallel.
	// Once WaitGroup's counter reaches zero (if manually done), it allows the program to exit.
	wg.Wait()
}
