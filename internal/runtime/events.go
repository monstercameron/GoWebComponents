//go:build js && wasm
// +build js,wasm

package runtime

import (
	"syscall/js"
)

// GoEvent is a Go-friendly wrapper around the JavaScript event object.
// It provides a safe, convenient API for working with DOM events in Go.
//
// GoEvent is automatically provided to event handlers registered with GoUseFunc
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
type GoEvent struct {
	jsValue js.Value
}

// NewGoEvent creates a GoEvent wrapper around a JavaScript event object.
// This is typically called internally by the hooks system.
func NewGoEvent(jsEvent js.Value) GoEvent {
	return GoEvent{jsValue: jsEvent}
}

// GetValue retrieves the value of an input element from the event target.
// This is commonly used with oninput, onchange, and other input events.
//
// Example:
//
//	handleInput := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    inputValue := event.GetValue()
//	    setSearchTerm(inputValue)
//	})
//
// Returns empty string if the target has no value property.
func (e GoEvent) GetValue() string {
	target := e.jsValue.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return ""
	}
	value := target.Get("value")
	if value.IsUndefined() || value.IsNull() {
		return ""
	}
	return value.String()
}

// IsChecked retrieves the checked state of a checkbox or radio button from the event target.
// This is commonly used with onchange events on checkboxes and radio buttons.
//
// Example:
//
//	handleToggle := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    isChecked := event.IsChecked()
//	    setEnabled(isChecked)
//	})
//
// Returns false if the target has no checked property.
func (e GoEvent) IsChecked() bool {
	target := e.jsValue.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return false
	}
	checked := target.Get("checked")
	if checked.IsUndefined() || checked.IsNull() {
		return false
	}
	return checked.Truthy()
}

// GetKey retrieves the key name from keyboard events (onkeydown, onkeyup, onkeypress).
// This allows you to detect which key was pressed.
//
// Example:
//
//	handleKeyDown := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    if event.GetKey() == "Enter" {
//	        handleSubmit()
//	    }
//	})
//
// Common key values:
//   - "Enter", "Escape", "Tab", "Backspace", "Delete"
//   - "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"
//   - " " for Space
//   - Alphanumeric characters for letter/number keys
//
// Returns empty string if this is not a keyboard event or key is not available.
func (e GoEvent) GetKey() string {
	key := e.jsValue.Get("key")
	if key.IsUndefined() || key.IsNull() {
		return ""
	}
	return key.String()
}

// PreventDefault prevents the browser's default action for the event.
// This is useful for form submissions, link navigation, and other browser defaults.
//
// Example preventing form submission:
//
//	handleSubmit := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    event.PreventDefault()
//	    // Handle form submission in Go
//	    submitForm()
//	})
//
// Example preventing link navigation:
//
//	handleLinkClick := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    event.PreventDefault()
//	    // Navigate using router instead
//	    router.Navigate("/page")
//	})
func (e GoEvent) PreventDefault() {
	e.jsValue.Call("preventDefault")
}

// StopPropagation prevents the event from bubbling up to parent elements.
// This is useful when you want to handle an event without it affecting parent elements.
//
// Example preventing click bubbling:
//
//	handleModalClick := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    event.StopPropagation()
//	    // Handle modal click without closing
//	})
//
// Example in a nested list:
//
//	handleItemClick := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    event.StopPropagation()
//	    selectItem()
//	})
func (e GoEvent) StopPropagation() {
	e.jsValue.Call("stopPropagation")
}

// JSValue returns the underlying JavaScript event object.
// This provides access to any properties or methods not covered by the GoEvent API.
//
// Example accessing custom properties:
//
//	handleCustomEvent := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    customData := event.JSValue().Get("customProperty").String()
//	})
func (e GoEvent) JSValue() js.Value {
	return e.jsValue
}
