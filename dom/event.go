//go:build js && wasm
// +build js,wasm

package dom

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// GoEvent is a Go-friendly wrapper around the JavaScript event object.
// It provides a safe, convenient API for working with DOM events in Go.
//
// GoEvent is automatically provided to event handlers registered with hooks.GoUseFunc
// when using the func(GoEvent) signature.
//
// Example:
//
//	handleClick := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    event.PreventDefault()
//	    println("Clicked!")
//	})
//
//	Button(Attrs{"onclick": handleClick}, "Click Me")
//
// Example with form input:
//
//	handleChange := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    value := event.GetValue()
//	    println("New value:", value)
//	})
//
//	Input(Attrs{"oninput": handleChange}, nil)
//
// Methods:
//   - GetValue(): Retrieves input element value
//   - IsChecked(): Gets checkbox/radio button state
//   - GetKey(): Gets the key from keyboard events
//   - PreventDefault(): Prevents browser default action
//   - StopPropagation(): Stops event bubbling
//   - JSValue(): Gets the underlying js.Value
type GoEvent = runtime.GoEvent
