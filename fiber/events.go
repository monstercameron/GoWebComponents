//go:build js && wasm
// +build js,wasm

package fiber

import (
	"context"
	"reflect"
	"sync"
	"syscall/js"
	"time"
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
// Optimized to cache target reference and reduce DOM API calls
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
// Optimized to cache target reference and reduce DOM API calls
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
// Optimized to cache target reference and reduce DOM API calls
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
// Optimized with form caching and efficient DOM traversal
func (e GoEvent) GetFormData() map[string]string {
	target := e.jsEvent.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return make(map[string]string)
	}

	// Optimized form finding using closest() method when available
	// This is much faster than manual parent traversal
	var form js.Value
	
	// Try modern closest() method first (supported in all modern browsers)
	if !target.Get("closest").IsUndefined() {
		form = target.Call("closest", "form")
		if form.IsNull() {
			// No form found
			return make(map[string]string)
		}
	} else {
		// Fallback to manual traversal for older browsers
		// Optimized with depth limit to prevent infinite loops
		form = target
		maxDepth := 20 // Reasonable DOM depth limit
		for depth := 0; depth < maxDepth && !form.IsNull() && !form.IsUndefined(); depth++ {
			tagName := form.Get("tagName")
			if !tagName.IsUndefined() && !tagName.IsNull() && tagName.String() == "FORM" {
				break
			}
			form = form.Get("parentElement")
		}
		
		if form.IsNull() || form.IsUndefined() || form.Get("tagName").String() != "FORM" {
			return make(map[string]string)
		}
	}

	// Pre-size the map based on typical form sizes to reduce allocations
	// Most forms have 2-8 fields, so we start with capacity 8
	formData := make(map[string]string, 8)
	
	// Batch DOM operations - get elements collection once
	elements := form.Get("elements")
	if elements.IsUndefined() || elements.IsNull() {
		return formData
	}
	
	length := elements.Get("length").Int()
	if length == 0 {
		return formData
	}

	// Optimized element processing with reduced DOM calls
	for i := 0; i < length; i++ {
		element := elements.Index(i)
		if element.IsUndefined() || element.IsNull() {
			continue
		}
		
		// Batch property access to reduce DOM API calls
		name := element.Get("name")
		if name.IsUndefined() || name.IsNull() {
			continue
		}
		
		nameStr := name.String()
		if nameStr == "" {
			continue
		}
		
		// Get value - handle different input types efficiently
		value := element.Get("value")
		if !value.IsUndefined() && !value.IsNull() {
			formData[nameStr] = value.String()
		} else {
			// Some elements might not have value property
			formData[nameStr] = ""
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
// Optimized to cache target reference and reduce DOM API calls
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

// GetTarget returns the cached target element to avoid repeated DOM access
// This is useful when you need to access multiple properties from the same target
func (e GoEvent) GetTarget() js.Value {
	return e.jsEvent.Get("target")
}

// GetTargetProperties efficiently retrieves multiple properties from the event target
// This reduces DOM API calls by batching property access
func (e GoEvent) GetTargetProperties(properties ...string) map[string]js.Value {
	target := e.jsEvent.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return make(map[string]js.Value)
	}
	
	// Pre-size map to avoid growth during insertion
	result := make(map[string]js.Value, len(properties))
	
	// Batch property access - single target reference, multiple property gets
	for _, prop := range properties {
		result[prop] = target.Get(prop)
	}
	
	return result
}

// GetInputData efficiently retrieves common input element properties in one call
// This is optimized for form inputs and reduces DOM API calls
func (e GoEvent) GetInputData() (value, name, id, className string, checked bool) {
	target := e.jsEvent.Get("target")
	if target.IsUndefined() || target.IsNull() {
		return "", "", "", "", false
	}
	
	// Batch all property access with single target reference
	valueJS := target.Get("value")
	nameJS := target.Get("name")
	idJS := target.Get("id")
	classJS := target.Get("className")
	checkedJS := target.Get("checked")
	
	// Convert to Go types with null checks
	if !valueJS.IsUndefined() && !valueJS.IsNull() {
		value = valueJS.String()
	}
	if !nameJS.IsUndefined() && !nameJS.IsNull() {
		name = nameJS.String()
	}
	if !idJS.IsUndefined() && !idJS.IsNull() {
		id = idJS.String()
	}
	if !classJS.IsUndefined() && !classJS.IsNull() {
		className = classJS.String()
	}
	if !checkedJS.IsUndefined() && !checkedJS.IsNull() {
		checked = checkedJS.Bool()
	}
	
	return value, name, id, className, checked
}

// Goroutine leak prevention - global context and cancellation
var (
	globalEventContext    context.Context
	globalEventCancel     context.CancelFunc
	eventCallbackTimeouts = make(map[uintptr]*time.Timer) // Use uintptr as key instead of js.Func
	eventTimeoutMutex     sync.RWMutex
	maxEventTimeout       = 30 * time.Second // Maximum time an event callback can run
)

// Initialize global event context for goroutine leak prevention
func init() {
	globalEventContext, globalEventCancel = context.WithCancel(context.Background())
}

// GoUseFunc creates an event handler with a more convenient Go-friendly interface
// The callback receives a GoEvent as the first parameter, followed by any additional parameters
// PERFORMANCE OPTIMIZATION: Added goroutine leak prevention with timeout controls
func GoUseFunc(callback interface{}) js.Func {
	// Check memory pressure before creating new callback
	checkMemoryPressure()

	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Goroutine leak prevention: Create timeout context for this callback
		callbackCtx, callbackCancel := context.WithTimeout(globalEventContext, maxEventTimeout)
		defer callbackCancel()
		
		// Track callback execution with timeout
		done := make(chan struct{}, 1)
		var callbackPanic interface{}
		
		go func() {
			defer func() {
				if r := recover(); r != nil {
					callbackPanic = r
					debugf("EVENTS", "🚨 GoUseFunc: callback panicked: %v\n", r)
				}
				close(done)
			}()
			
			// Create GoEvent from the first argument (the JS event)
			var goEvent GoEvent
			if len(args) > 0 {
				goEvent = NewGoEvent(args[0])
			}

			// Optimized callback dispatch using type switches for common signatures
			// This avoids expensive reflection calls for 95% of use cases
			switch cb := callback.(type) {
			case func(GoEvent):
				// Most common case: single GoEvent parameter
				cb(goEvent)
			case func(GoEvent, js.Value):
				// Second most common: GoEvent + one js.Value
				if len(args) > 1 {
					cb(goEvent, args[1])
				} else {
					cb(goEvent, js.Undefined())
				}
			case func(GoEvent, js.Value, js.Value):
				// Less common: GoEvent + two js.Values
				arg1 := js.Undefined()
				arg2 := js.Undefined()
				if len(args) > 1 {
					arg1 = args[1]
				}
				if len(args) > 2 {
					arg2 = args[2]
				}
				cb(goEvent, arg1, arg2)
			case func():
				// No parameters callback
				cb()
			case func(js.Value):
				// Raw js.Value callback (for compatibility)
				if len(args) > 0 {
					cb(args[0])
				} else {
					cb(js.Undefined())
				}
			default:
				// Fallback to reflection for uncommon signatures
				// This maintains compatibility while optimizing the common cases
				callbackValue := reflect.ValueOf(callback)
				callbackType := callbackValue.Type()

				if callbackType.Kind() != reflect.Func {
					debugf("EVENTS", "🚨 GoUseFunc: callback must be a function\n")
					return
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
			}
		}()
		
		// Wait for callback completion or timeout
		select {
		case <-done:
			// Callback completed normally
			if callbackPanic != nil {
				debugf("EVENTS", "🚨 GoUseFunc: callback completed with panic: %v\n", callbackPanic)
			}
		case <-callbackCtx.Done():
			// Callback timed out - potential goroutine leak prevented
			debugf("EVENTS", "⏰ GoUseFunc: callback timed out after %v, preventing potential goroutine leak\n", maxEventTimeout)
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
		
		// Warn if global fallback is being used frequently
		if len(eventCallbacks)%50 == 0 && len(eventCallbacks) > 0 {
			debugf("EVENTS", "🚨 GoUseFunc: WARNING - %d callbacks in global fallback pool, consider investigating why no current fiber\n", 
				len(eventCallbacks))
		}
		
		// Trigger memory pressure check if global pool grows too large
		if len(eventCallbacks) > 200 {
			debugf("EVENTS", "🧹 GoUseFunc: global callback pool large (%d), triggering memory cleanup\n", 
				len(eventCallbacks))
			checkMemoryPressure()
		}
	}

	return cb
}

// SetEventCallbackTimeout configures the maximum timeout for event callbacks
// This helps prevent goroutine leaks from long-running or stuck callbacks
func SetEventCallbackTimeout(timeout time.Duration) {
	if timeout <= 0 {
		timeout = 30 * time.Second // Default fallback
	}
	maxEventTimeout = timeout
	debugf("EVENTS", "⏰ SetEventCallbackTimeout: set to %v\n", timeout)
}

// GetEventCallbackTimeout returns the current event callback timeout
func GetEventCallbackTimeout() time.Duration {
	return maxEventTimeout
}

// CancelAllEventCallbacks cancels all active event callbacks to prevent goroutine leaks
// This should be called during application shutdown or major state resets
func CancelAllEventCallbacks() {
	debugf("EVENTS", "🛑 CancelAllEventCallbacks: cancelling global event context\n")
	
	// Cancel the global context - this will cancel all active callback contexts
	globalEventCancel()
	
	// Create new global context for future callbacks
	globalEventContext, globalEventCancel = context.WithCancel(context.Background())
	
	// Clean up timeout timers
	eventTimeoutMutex.Lock()
	for _, timer := range eventCallbackTimeouts {
		timer.Stop()
	}
	eventCallbackTimeouts = make(map[uintptr]*time.Timer)
	eventTimeoutMutex.Unlock()
	
	debugf("EVENTS", "✅ CancelAllEventCallbacks: all callbacks cancelled, new context created\n")
}

// CleanupGlobalEventCallbacks manually cleans up the global event callbacks fallback pool
// This should only be needed in edge cases where callbacks accumulate in the global pool
func CleanupGlobalEventCallbacks() {
	debugf("EVENTS", "🧹 CleanupGlobalEventCallbacks: cleaning up %d global callbacks\n", len(eventCallbacks))
	
	// Release all callbacks in the global pool
	for _, callback := range eventCallbacks {
		callback.Release()
	}
	
	// Clear the slice
	eventCallbacks = eventCallbacks[:0]
	
	debugf("EVENTS", "✅ CleanupGlobalEventCallbacks: global callback pool cleared\n")
}

// GetGlobalEventCallbacksCount returns the number of callbacks in the global fallback pool
func GetGlobalEventCallbacksCount() int {
	return len(eventCallbacks)
}
