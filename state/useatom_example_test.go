package state_test

import (
	"github.com/monstercameron/GoWebComponents/v6/state"
)

// ExampleUseAtom shows shared atom state: components that bind the same atom id
// observe one value, and a Computed derives from it.
func ExampleUseAtom() {
	parseCount := state.UseAtom("counter", 0)
	parseDoubled := state.UseComputed(func() int { return parseCount.Get() * 2 })
	// Both update together when the atom changes.
	_ = parseDoubled
}
