package main

import (
	"fmt"
	// html "github.com/monstercameron/GoWebComponents/components"
	"github.com/monstercameron/GoWebComponents/fiber"
	// testing "github.com/monstercameron/GoWebComponents/checkingsomething"
	"sync"
)

// main is the entry point of the program.
// It initializes a WaitGroup, prints a message, calls the Example1 function from the html package,
// and waits for the WaitGroup to complete before exiting.
func main() {
	// Initialize a WaitGroup to simulate waiting for asynchronous tasks in the WebAssembly environment.
	var wg sync.WaitGroup

	// Add(1) indicates that we're waiting for 1 operation to complete.
	// In this case, it is just a placeholder for blocking the main function.
	wg.Add(1)

	// Print a message indicating the start of the program.
	fmt.Println("Main: Starting fiber.Example1")

	// Call the Example1 function from the html package, which handles the HTML rendering.
	// html.Example4()
	fiber.Example1()

	// Print a message indicating the end of the main function logic.
	// At this point, the Example1 function has already executed.
	fmt.Println("Main: End of main function")

	// Wait() blocks the main function from exiting immediately.
	// In WebAssembly, this is used to keep the program alive for event handling and state management,
	// as WebAssembly is single-threaded and doesn't have native goroutines running in parallel.
	// Once WaitGroup's counter reaches zero (if manually done), it allows the program to exit.
	wg.Wait()
}
