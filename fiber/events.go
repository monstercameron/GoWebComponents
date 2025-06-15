//go:build js && wasm
// +build js,wasm

package fiber

import (
	"reflect"
	"syscall/js"
)

// PreventDefault prevents the default action of the event
func (e GoEvent) PreventDefault() {
	e.jsEvent.Call("preventDefault")
}

// StopPropagation stops the event from bubbling up the DOM tree
func (e GoEvent) StopPropagation() {
	e.jsEvent.Call("stopPropagation")
}

// GetValue gets the value from the event target (useful for input elements)
func (e GoEvent) GetValue() string {
	target := e.jsEvent.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return ""
	}
	value := target.Get("value")
	if value.IsUndefined() || value.IsNull() {
		return ""
	}
	return value.String()
}

// GetInnerText gets the innerText from the event target
func (e GoEvent) GetInnerText() string {
	target := e.jsEvent.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return ""
	}
	innerText := target.Get("innerText")
	if innerText.IsUndefined() || innerText.IsNull() {
		return ""
	}
	return innerText.String()
}

// GetAttribute gets an attribute from the event target
func (e GoEvent) GetAttribute(name string) string {
	target := e.jsEvent.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return ""
	}
	attr := target.Call("getAttribute", name)
	if attr.IsUndefined() || attr.IsNull() {
		return ""
	}
	return attr.String()
}

// GetFormData extracts all form data when the target is within a form
func (e GoEvent) GetFormData() map[string]string {
	target := e.jsEvent.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return make(map[string]string)
	}

	// Find the closest form element
	form := target
	for !form.IsNull() && !form.IsUndefined() && form.Get("tagName").String() != "FORM" {
		form = form.Get("parentElement")
	}

	if form.IsNull() || form.IsUndefined() {
		return make(map[string]string)
	}

	formData := make(map[string]string)
	elements := form.Get("elements")
	length := elements.Get("length").Int()

	for i := 0; i < length; i++ {
		element := elements.Index(i)
		name := element.Get("name").String()
		if name != "" {
			value := element.Get("value").String()
			formData[name] = value
		}
	}

	return formData
}

// GetKeyCode gets the key code for keyboard events
func (e GoEvent) GetKeyCode() int {
	keyCode := e.jsEvent.Get("keyCode")
	if keyCode.IsUndefined() || keyCode.IsNull() {
		// Try 'which' for older browsers or 'key' for modern ones
		which := e.jsEvent.Get("which")
		if !which.IsUndefined() && !which.IsNull() {
			return which.Int()
		}
		return 0
	}
	return keyCode.Int()
}

// GetKey gets the key string for keyboard events (modern browsers)
func (e GoEvent) GetKey() string {
	key := e.jsEvent.Get("key")
	if key.IsUndefined() || key.IsNull() {
		return ""
	}
	return key.String()
}

// GetMousePosition gets the mouse position for mouse events
func (e GoEvent) GetMousePosition() (x, y int) {
	clientX := e.jsEvent.Get("clientX")
	clientY := e.jsEvent.Get("clientY")

	if !clientX.IsUndefined() && !clientX.IsNull() {
		x = clientX.Int()
	}
	if !clientY.IsUndefined() && !clientY.IsNull() {
		y = clientY.Int()
	}

	return x, y
}

// IsChecked gets the checked state for checkbox/radio elements
func (e GoEvent) IsChecked() bool {
	target := e.jsEvent.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return false
	}
	checked := target.Get("checked")
	if checked.IsUndefined() || checked.IsNull() {
		return false
	}
	return checked.Bool()
}

// Raw provides access to the underlying JavaScript event for advanced use cases
func (e GoEvent) Raw() js.Value {
	return e.jsEvent
}

// GoUseFunc creates an event handler with a more convenient Go-friendly interface
// The callback receives a GoEvent as the first parameter, followed by any additional parameters
func GoUseFunc(callback interface{}) js.Func {
	// Check memory pressure before creating new callback
	checkMemoryPressure()

	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Create GoEvent from the first argument (the JS event)
		var goEvent GoEvent
		if len(args) > 0 {
			goEvent = NewGoEvent(args[0])
		}

		// Use reflection to call the callback with appropriate parameters
		callbackValue := reflect.ValueOf(callback)
		callbackType := callbackValue.Type()

		if callbackType.Kind() != reflect.Func {
			debugf("EVENTS", "🚨 GoUseFunc: callback must be a function\n")
			return nil
		}

		// Prepare arguments for the callback
		var callArgs []reflect.Value

		// First argument is always the GoEvent
		if callbackType.NumIn() > 0 {
			callArgs = append(callArgs, reflect.ValueOf(goEvent))
		}

		// Additional arguments from the JS event (if any)
		for i := 1; i < len(args) && len(callArgs) < callbackType.NumIn(); i++ {
			// Convert js.Value to interface{} for additional parameters
			callArgs = append(callArgs, reflect.ValueOf(args[i]))
		}

		// Call the callback function
		if len(callArgs) <= callbackType.NumIn() {
			callbackValue.Call(callArgs)
		}

		return nil
	})

	// Associate callback with current fiber for proper cleanup
	currentFiber := getCurrentFiber()
	if currentFiber != nil {
		if currentFiber.eventCallbacks == nil {
			currentFiber.eventCallbacks = make([]js.Func, 0, 4)
		}
		currentFiber.eventCallbacks = append(currentFiber.eventCallbacks, cb)
		debugf("EVENTS", "🔗 GoUseFunc callback associated with fiber %p - fiber callbacks: %d\n", 
			currentFiber, len(currentFiber.eventCallbacks))
	} else {
		// Fallback to global tracking if no current fiber (shouldn't happen in normal usage)
		eventCallbacks = append(eventCallbacks, cb)
		debugf("EVENTS", "⚠️ GoUseFunc callback added to global pool (no current fiber) - total: %d\n", 
			len(eventCallbacks))
	}

	return cb
}
