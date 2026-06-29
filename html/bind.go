package html

import "github.com/monstercameron/GoWebComponents/v4/ui"

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

// Binding is a two-way-bindable string value: any handle with Get() string and
// Set(string). It lets [BindTo] wire the V4 reactive handles — state.Signal[string],
// state.GlobalAtom[string], a state.Atom[string] handle — to a controlled input
// with the same one-call ergonomics as [Bind] for ui.State.
type Binding interface {
	Get() string
	Set(string)
}

// BindTo two-way binds a text input to any [Binding] (e.g. a state.Signal[string]):
// it sets the input value to target.Get() and registers an oninput handler that
// writes target.Set(value). Like [Bind], call it at a stable render position. A nil
// target is a no-op.
//
//	name := state.NewSignal("")
//	h.Input(h.Type("text"), h.BindTo(name))
func BindTo(parseTarget Binding) PropOption {
	if parseTarget == nil {
		return optionFunc(func(*Props) {})
	}
	return BindFunc(parseTarget.Get, parseTarget.Set)
}

// BindFunc is the general two-way binding: it sets the input value to get() and
// registers an oninput handler that calls set(value). Use it when the source is not
// a single handle (e.g. a struct field reached through a closure). A nil getter
// leaves the value unset; a nil setter makes input a no-op.
func BindFunc(parseGet func() string, parseSet func(string)) PropOption {
	return optionFunc(func(parseProps *Props) {
		if parseGet != nil {
			Value(parseGet()).apply(parseProps)
		}
		OnInput(bindInputHandler(parseSet)).apply(parseProps)
	})
}

// bindInputHandler returns the oninput callback used by BindTo/BindFunc: it writes
// the input's current value through set. Factored out so the write path is unit
// testable without dispatching a real DOM event.
func bindInputHandler(parseSet func(string)) func(string) {
	return func(parseValue string) {
		if parseSet != nil {
			parseSet(parseValue)
		}
	}
}
