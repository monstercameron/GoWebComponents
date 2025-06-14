//go:build js && wasm
// +build js,wasm

package fiber

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
)

// fastEqual performs equality checking with cycle detection
func fastEqual(a, b interface{}) bool {
	return fastEqualWithDepth(a, b, make(map[uintptr]bool), 0)
}

// fastEqualWithDepth performs equality checking with cycle detection and depth limiting
func fastEqualWithDepth(a, b interface{}, visited map[uintptr]bool, depth int) bool {
	// Prevent infinite recursion - limit depth to reasonable level
	const maxDepth = 100
	if depth > maxDepth {
		return false
	}

	// Fast path: nil checks first
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Fast path: Check if types implement FastComparable
	if fc, ok := a.(FastComparable); ok {
		return fc.FastEqual(b)
	}

	// Fast path: Common primitive types (avoid reflection)
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return va == vb
		}
	case int:
		if vb, ok := b.(int); ok {
			return va == vb
		}
	case int8:
		if vb, ok := b.(int8); ok {
			return va == vb
		}
	case int16:
		if vb, ok := b.(int16); ok {
			return va == vb
		}
	case int32:
		if vb, ok := b.(int32); ok {
			return va == vb
		}
	case int64:
		if vb, ok := b.(int64); ok {
			return va == vb
		}
	case uint:
		if vb, ok := b.(uint); ok {
			return va == vb
		}
	case uint8:
		if vb, ok := b.(uint8); ok {
			return va == vb
		}
	case uint16:
		if vb, ok := b.(uint16); ok {
			return va == vb
		}
	case uint32:
		if vb, ok := b.(uint32); ok {
			return va == vb
		}
	case uint64:
		if vb, ok := b.(uint64); ok {
			return va == vb
		}
	case float32:
		if vb, ok := b.(float32); ok {
			return va == vb
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return va == vb
		}
	case bool:
		if vb, ok := b.(bool); ok {
			return va == vb
		}
	case []string:
		if vb, ok := b.([]string); ok {
			if len(va) != len(vb) {
				return false
			}
			for i := range va {
				if va[i] != vb[i] {
					return false
				}
			}
			return true
		}
	case []interface{}:
		if vb, ok := b.([]interface{}); ok {
			if len(va) != len(vb) {
				return false
			}

			// Check for cycles using slice pointer
			aPtr := reflect.ValueOf(va).Pointer()
			if visited[aPtr] {
				return true // Assume equal if we've seen this before (cycle detected)
			}
			visited[aPtr] = true
			defer delete(visited, aPtr)

			for i := range va {
				if !fastEqualWithDepth(va[i], vb[i], visited, depth+1) {
					return false
				}
			}
			return true
		}
	case []int:
		if vb, ok := b.([]int); ok {
			if len(va) != len(vb) {
				return false
			}
			for i := range va {
				if va[i] != vb[i] {
					return false
				}
			}
			return true
		}
	case map[string]interface{}:
		if vb, ok := b.(map[string]interface{}); ok {
			if len(va) != len(vb) {
				return false
			}

			// Check for cycles using map pointer
			aPtr := reflect.ValueOf(va).Pointer()
			if visited[aPtr] {
				return true // Assume equal if we've seen this before (cycle detected)
			}
			visited[aPtr] = true
			defer delete(visited, aPtr)

			for k, v := range va {
				if vbVal, exists := vb[k]; !exists || !fastEqualWithDepth(v, vbVal, visited, depth+1) {
					return false
				}
			}
			return true
		}
	}

	// Use reflection for more complex types
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	// Check if types are the same
	if va.Type() != vb.Type() {
		return false
	}

	// Handle special cases that could cause issues
	switch va.Kind() {
	case reflect.Func:
		// Functions are not comparable in a meaningful way
		return va.Pointer() == vb.Pointer()
	case reflect.Chan:
		// Channels are comparable by identity
		return va.Pointer() == vb.Pointer()
	case reflect.UnsafePointer:
		// Unsafe pointers are comparable by value
		return va.Pointer() == vb.Pointer()
	}

	// For uncomparable types, fall back to reflect.DeepEqual with safety check
	if !va.Type().Comparable() {
		// Additional safety: avoid reflect.DeepEqual on very complex structures
		if depth > 10 {
			return false
		}
		return reflect.DeepEqual(a, b)
	}

	// Safe to use == for comparable types
	return a == b
}

// getFunctionName extracts a readable name from a function interface
func getFunctionName(i interface{}) string {
	if i == nil {
		return "nil"
	}
	// Use reflection to get the function's name
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer())
	if fn == nil {
		// Fallback for anonymous functions or other cases
		return reflect.TypeOf(i).String()
	}
	name := fn.Name()
	// Clean up the name to be more readable
	parts := strings.Split(name, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return name
}

// enqueueUI schedules fn to run on the main JS/event thread
func enqueueUI(fn func()) {
	select {
	case uiQueue <- fn:
	default:
		fn()
	}

	// If we're currently outside scheduler, ensure an idle callback exists to process the queue
	if atomic.LoadInt32(&schedulerActive) == 0 && !updateScheduled {
		requestIdleCallback(workLoop)
	}
}

// processUIQueue drains queued UI operations; should be called from main thread
func processUIQueue() {
	startSchedulerSection()
	defer endSchedulerSection()
	for {
		select {
		case fn := <-uiQueue:
			if fn != nil {
				fn()
			}
		default:
			return
		}
	}
}

// startSchedulerSection marks that we're in a scheduler section
func startSchedulerSection() {
	atomic.StoreInt32(&schedulerActive, 1)
}

// endSchedulerSection marks that we're done with scheduler section
func endSchedulerSection() {
	atomic.StoreInt32(&schedulerActive, 0)
}

// Debug namespace control - map of namespace to enabled status
var debugNamespaces = make(map[string]bool)

// SetDebug enables or disables verbose debug logs at runtime
func SetDebug(enabled bool) {
	debugEnabled = enabled
}

// SetDebugNamespace enables or disables debug logs for a specific namespace
func SetDebugNamespace(namespace string, enabled bool) {
	debugNamespaces[namespace] = enabled
}

// SetDebugNamespaces enables multiple namespaces at once
func SetDebugNamespaces(namespaces map[string]bool) {
	for ns, enabled := range namespaces {
		debugNamespaces[ns] = enabled
	}
}

// debugf prints debug messages if debug is enabled globally or for the specific namespace
func debugf(namespace, format string, a ...interface{}) {
	if debugEnabled || debugNamespaces[namespace] {
		fmt.Printf("[%s] %s", namespace, fmt.Sprintf(format, a...))
	}
}

// EnableAllDebug enables all debug logging globally
func EnableAllDebug() {
	SetDebug(true)
}

// DisableAllDebug disables all debug logging globally
func DisableAllDebug() {
	SetDebug(false)
	// Clear namespace-specific settings
	debugNamespaces = make(map[string]bool)
}

// GetDebugStatus returns current debug settings
func GetDebugStatus() map[string]bool {
	status := make(map[string]bool)
	status["global"] = debugEnabled
	for ns, enabled := range debugNamespaces {
		status[ns] = enabled
	}
	return status
}
