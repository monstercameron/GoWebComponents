//go:build js && wasm

package runtime

import (
	"syscall/js"
)

type jsEventTargetNode struct {
	value js.Value
}

// IsNull is a core package helper.
func (parseN *jsEventTargetNode) IsNull() bool {
	return parseN == nil || parseN.value.IsNull() || parseN.value.IsUndefined()
}

// Equals is a core package helper.
func (parseN *jsEventTargetNode) Equals(parseOther DOMNode) bool {
	if IsDOMNodeNull(parseN) {
		return IsDOMNodeNull(parseOther)
	}
	parseOtherNode, parseOk := parseOther.(*jsEventTargetNode)
	if !parseOk {
		return false
	}
	if IsDOMNodeNull(parseOtherNode) {
		return false
	}
	return parseN.value.Equal(parseOtherNode.value)
}

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

// targetValue is a core package helper.
func (parseE GoEvent) targetValue() js.Value {
	if parseE.jsValue.IsUndefined() || parseE.jsValue.IsNull() {
		return js.Undefined()
	}
	return parseE.jsValue.Get("target")
}

// NewGoEvent creates a GoEvent wrapper around a JavaScript event object.
func NewGoEvent(parseJsEvent js.Value) GoEvent {
	return GoEvent{jsValue: parseJsEvent}
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
func (parseE GoEvent) GetValue() string {
	parseTarget := parseE.targetValue()
	if parseTarget.IsUndefined() || parseTarget.IsNull() {
		return ""
	}
	parseValue := parseTarget.Get("value")
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return ""
	}
	return parseValue.String()
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
func (parseE GoEvent) IsChecked() bool {
	parseTarget := parseE.targetValue()
	if parseTarget.IsUndefined() || parseTarget.IsNull() {
		return false
	}
	parseChecked := parseTarget.Get("checked")
	if parseChecked.IsUndefined() || parseChecked.IsNull() {
		return false
	}
	return parseChecked.Truthy()
}

// GetTarget retrieves the event target as a DOMNode.
// It returns nil when the event has no target.
func (parseE GoEvent) GetTarget() DOMNode {
	parseTarget := parseE.targetValue()
	if parseTarget.IsUndefined() || parseTarget.IsNull() {
		return nil
	}
	return &jsEventTargetNode{value: parseTarget}
}

// GetKeyCode retrieves the legacy numeric keyCode for keyboard events.
// It returns 0 when the keyCode is unavailable.
func (parseE GoEvent) GetKeyCode() int {
	if parseE.jsValue.IsUndefined() || parseE.jsValue.IsNull() {
		return 0
	}
	parseKeyCode := parseE.jsValue.Get("keyCode")
	if parseKeyCode.IsUndefined() || parseKeyCode.IsNull() {
		return 0
	}
	return parseKeyCode.Int()
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
func (parseE GoEvent) GetKey() string {
	if parseE.jsValue.IsUndefined() || parseE.jsValue.IsNull() {
		return ""
	}
	parseKey := parseE.jsValue.Get("key")
	if parseKey.IsUndefined() || parseKey.IsNull() {
		return ""
	}
	return parseKey.String()
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
func (parseE GoEvent) PreventDefault() {
	if parseE.jsValue.IsUndefined() || parseE.jsValue.IsNull() {
		return
	}
	parseFn := parseE.jsValue.Get("preventDefault")
	if parseFn.IsUndefined() || parseFn.IsNull() {
		return
	}
	parseE.jsValue.Call("preventDefault")
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
func (parseE GoEvent) StopPropagation() {
	if parseE.jsValue.IsUndefined() || parseE.jsValue.IsNull() {
		return
	}
	parseFn := parseE.jsValue.Get("stopPropagation")
	if parseFn.IsUndefined() || parseFn.IsNull() {
		return
	}
	parseE.jsValue.Call("stopPropagation")
}

// JSValue returns the underlying JavaScript event object.
// This provides access to any properties or methods not covered by the GoEvent API.
//
// Example accessing custom properties:
//
//	handleCustomEvent := hooks.GoUseFunc(func(event dom.GoEvent) {
//	    customData := event.JSValue().Get("customProperty").String()
//	})
func (parseE GoEvent) JSValue() js.Value {
	return parseE.jsValue
}
