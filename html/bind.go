package html

import "github.com/monstercameron/GoWebComponents/ui"

// Bind wires a controlled input to a string State in one call (U6): it sets the
// element's value from the state and updates the state on input. Replaces the
// manual Value(s.Get()) + OnInput(func(v){ s.Set(v) }) pairing.
//
//	name := ui.UseState("")
//	html.Input(html.Bind(name))
func Bind(parseState ui.State[string]) PropOption {
	return optionFunc(func(parseProps *Props) {
		Value(parseState.Get()).apply(parseProps)
		OnInput(func(parseValue string) { parseState.Set(parseValue) }).apply(parseProps)
	})
}
