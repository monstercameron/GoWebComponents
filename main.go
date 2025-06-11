// ./main.go

package main

import (
	"fmt"
	"sync"

	"github.com/monstercameron/GoWebComponents/fiber"
)

// main is the entry point of the program.
// It initializes a WaitGroup, prints a message, calls the Example2 function from the fiber package,
// and waits for the WaitGroup to complete before exiting.
func main() {
	// Initialize a WaitGroup to simulate waiting for asynchronous tasks in the WebAssembly environment.
	var wg sync.WaitGroup

	// Add(1) indicates that we're waiting for 1 operation to complete.
	// In this case, it is just a placeholder for blocking the main function.
	wg.Add(1)

	// Print a message indicating the start of the program.
	fmt.Println("Main: Starting fiber.Example2")

	// Call the Example2 function from the fiber package, which handles the optimized click counter.
	// fiber.Example1()
	// fiber.Example2()
	// fiber.Example3()
	// fiber.Example4()
	fiber.Example5()

	// Print a message indicating the end of the main function logic.
	// At this point, the Example2 function has already executed.
	fmt.Println("Main: End of main function")

	// Wait() blocks the main function from exiting immediately.
	// In WebAssembly, this is used to keep the program alive for event handling and state management,
	// as WebAssembly is single-threaded and doesn't have native goroutines running in parallel.
	// Once WaitGroup's counter reaches zero (if manually done), it allows the program to exit.
	wg.Wait()
}
