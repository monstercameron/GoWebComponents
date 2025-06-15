//go:build js && wasm
// +build js,wasm

package fiber

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// fastEqual performs equality checking with cycle detection
func fastEqual(a, b interface{}) bool {
	debugf("UTILS", "🔍 fastEqual: comparing %T vs %T\n", a, b)
	result := fastEqualWithDepth(a, b, make(map[uintptr]bool), 0)
	debugf("UTILS", "✅ fastEqual: result = %v\n", result)
	return result
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

// enqueueUI schedules fn to run on the main JS/event thread with overflow monitoring
func enqueueUI(fn func()) {
	debugf("UTILS", "📋 enqueueUI: attempting to enqueue UI task\n")
	
	// Track current queue size for monitoring
	currentSize := int64(len(uiQueue))
	if currentSize > atomic.LoadInt64(&uiQueueMaxSize) {
		atomic.StoreInt64(&uiQueueMaxSize, currentSize)
		debugf("UTILS", "📊 enqueueUI: new max queue size: %d\n", currentSize)
	}
	
	select {
	case uiQueue <- fn:
		debugf("UTILS", "✅ enqueueUI: task enqueued successfully (queue size: %d/1024)\n", currentSize+1)
	default:
		// Queue is full - increment overflow counter and handle gracefully
		atomic.AddInt64(&uiQueueOverflows, 1)
		overflowCount := atomic.LoadInt64(&uiQueueOverflows)
		debugf("UTILS", "⚠️ enqueueUI: queue overflow #%d, executing immediately\n", overflowCount)
		
		// Log warning for frequent overflows
		if overflowCount%100 == 0 {
			debugf("UTILS", "🚨 enqueueUI: WARNING - %d queue overflows detected, consider increasing queue size\n", overflowCount)
		}
		
		// Execute immediately as fallback
		fn()
	}

	// If we're currently outside scheduler, ensure an idle callback exists to process the queue
	if atomic.LoadInt32(&schedulerActive) == 0 && !updateScheduled {
		debugf("UTILS", "🚀 enqueueUI: scheduler inactive, requesting idle callback\n")
		requestIdleCallback(workLoop)
	}
}

// processUIQueue drains queued UI operations; should be called from main thread
func processUIQueue() {
	debugf("UTILS", "📋 processUIQueue: starting UI queue processing\n")
	queueStartTime := time.Now()
	tasksProcessed := 0
	initialQueueSize := len(uiQueue)

	startSchedulerSection()
	defer endSchedulerSection()
	
	// Process with a reasonable limit to prevent blocking too long
	const maxTasksPerBatch = 100
	
	for tasksProcessed < maxTasksPerBatch {
		select {
		case fn := <-uiQueue:
			if fn != nil {
				debugf("UTILS", "⚡ processUIQueue: executing task %d/%d\n", tasksProcessed+1, initialQueueSize)
				taskStartTime := time.Now()
				fn()
				taskDuration := time.Since(taskStartTime)
				debugf("UTILS", "✅ processUIQueue: task completed in %v\n", taskDuration)
				tasksProcessed++
			} else {
				debugf("UTILS", "🚨 processUIQueue: received nil task\n")
				tasksProcessed++ // Count nil tasks to avoid infinite loop
			}
		default:
			// No more tasks available
			queueDuration := time.Since(queueStartTime)
			debugf("UTILS", "✅ processUIQueue: completed %d/%d tasks in %v\n", tasksProcessed, initialQueueSize, queueDuration)
			return
		}
	}
	
	// If we hit the batch limit, schedule another processing cycle
	if len(uiQueue) > 0 {
		queueDuration := time.Since(queueStartTime)
		debugf("UTILS", "⏳ processUIQueue: batch limit reached (%d tasks), %d remaining - scheduling next cycle\n", 
			maxTasksPerBatch, len(uiQueue))
		debugf("UTILS", "📊 processUIQueue: batch completed in %v\n", queueDuration)
		
		// Schedule another processing cycle
		if atomic.LoadInt32(&schedulerActive) == 0 && !updateScheduled {
			requestIdleCallback(workLoop)
		}
	} else {
		queueDuration := time.Since(queueStartTime)
		debugf("UTILS", "✅ processUIQueue: all tasks completed (%d total) in %v\n", tasksProcessed, queueDuration)
	}
}

// startSchedulerSection marks that we're in a scheduler section
func startSchedulerSection() {
	debugf("UTILS", "🔒 startSchedulerSection: entering scheduler section\n")
	atomic.StoreInt32(&schedulerActive, 1)
}

// endSchedulerSection marks that we're done with scheduler section
func endSchedulerSection() {
	debugf("UTILS", "🔓 endSchedulerSection: exiting scheduler section\n")
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

// SetDebugNamespacesExclusive disables global debug and only enables specified namespaces
func SetDebugNamespacesExclusive(namespaces map[string]bool) {
	// Disable global debug first
	debugEnabled = false
	// Clear existing namespace settings efficiently without creating new map
	for k := range debugNamespaces {
		delete(debugNamespaces, k)
	}
	// Set only the specified namespaces
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
	// Clear namespace-specific settings efficiently without creating new map
	for k := range debugNamespaces {
		delete(debugNamespaces, k)
	}
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

// GetUIQueueStats returns UI queue statistics for monitoring
func GetUIQueueStats() map[string]int64 {
	return map[string]int64{
		"currentSize":    int64(len(uiQueue)),
		"capacity":       1024,
		"overflowCount":  atomic.LoadInt64(&uiQueueOverflows),
		"maxSizeReached": atomic.LoadInt64(&uiQueueMaxSize),
	}
}

// ResetUIQueueStats resets UI queue statistics counters
func ResetUIQueueStats() {
	atomic.StoreInt64(&uiQueueOverflows, 0)
	atomic.StoreInt64(&uiQueueMaxSize, 0)
	debugf("UTILS", "🔄 ResetUIQueueStats: statistics counters reset\n")
}
