//go:build js && wasm
// +build js,wasm

package state_test

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/state"
)

func ExampleUseAtom() {
	// Note: UseAtom must be used inside a component

	// Access or create a global atom named "user" with default value "Guest"
	parseUser := state.UseAtom("user", "Guest")

	// Get current value
	fmt.Printf("Current user: %s\n", parseUser.Get())

	// Update value - this will trigger updates in all components using this atom
	parseUser.Set("Alice")
}
