//go:build js && wasm
// +build js,wasm

package hooks_test

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/hooks"
)

func ExampleUseState() {
	// Note: Hooks can only be used inside a component function
	// This example demonstrates the syntax

	// Initialize state with 0
	count, setCount := hooks.UseState(0)

	// Get current value
	fmt.Printf("Initial count: %d\n", count())

	// Update state
	setCount(1)

	// In a real component, accessing count() again would return the new value
	// after a re-render.
}

func ExampleUseEffect() {
	// Note: Hooks can only be used inside a component function

	// Run an effect on every render
	hooks.UseEffect(func() func() {
		fmt.Println("Component rendered")
		return nil
	})

	// Run an effect only on mount (empty dependencies)
	hooks.UseEffect(func() func() {
		fmt.Println("Component mounted")

		// Return cleanup function
		return func() {
			fmt.Println("Component unmounted")
		}
	}, []interface{}{})
}

func ExampleUseMemo() {
	// Note: Hooks can only be used inside a component function

	// Assume we have some props
	count := 5

	// Memoize an expensive calculation
	// The calculation will only run when 'count' changes
	doubled := hooks.UseMemo(func() interface{} {
		// Simulate expensive work
		return count * 2
	}, count)

	fmt.Printf("Doubled: %v\n", doubled)
}
