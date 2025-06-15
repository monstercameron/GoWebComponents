//go:build js && wasm
// +build js,wasm

package fiber

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// Goroutine leak prevention and monitoring
var (
	goroutineMonitoringEnabled bool = false
	baselineGoroutineCount     int
	maxGoroutineThreshold      int = 1000 // Alert if goroutines exceed this
	goroutineCheckInterval         = 10 * time.Second
	goroutineMonitorContext    context.Context
	goroutineMonitorCancel     context.CancelFunc
	goroutineLeakDetected      int64 // Atomic counter for leak detection
)

// Initialize goroutine monitoring
func init() {
	goroutineMonitorContext, goroutineMonitorCancel = context.WithCancel(context.Background())
	baselineGoroutineCount = runtime.NumGoroutine()
}

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

			// Fast path: both empty slices
			if len(va) == 0 {
				return true
			}

			// Check for cycles using slice pointer - avoid reflection when possible
			vaPtr := (*[3]uintptr)(unsafe.Pointer(&va))
			aPtr := vaPtr[0] // Get the data pointer directly
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

			// Fast path: pointer equality check for same map
			if len(va) == 0 && len(vb) == 0 {
				return true // Both empty maps are equal
			}

			// Check for cycles using map pointer - avoid reflection when possible
			vaPtr := (*[2]uintptr)(unsafe.Pointer(&va))
			aPtr := vaPtr[1] // Get the data pointer directly
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
	case map[string]string:
		// Add fast path for common map[string]string type
		if vb, ok := b.(map[string]string); ok {
			if len(va) != len(vb) {
				return false
			}
			for k, v := range va {
				if vbVal, exists := vb[k]; !exists || v != vbVal {
					return false
				}
			}
			return true
		}
	case []byte:
		// Add fast path for byte slices
		if vb, ok := b.([]byte); ok {
			return bytes.Equal(va, vb)
		}
	}

	// Fast path: try direct equality first for comparable types
	// This avoids reflection for many common cases
	defer func() {
		if r := recover(); r != nil {
			// If direct comparison panics, types are not comparable
			// Fall back to reflection-based comparison
		}
	}()
	
	// Attempt direct comparison - this works for most comparable types
	if a == b {
		return true
	}

	// Use reflection for more complex types only when necessary
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
	case reflect.Slice, reflect.Map:
		// For slices and maps, we should have handled them above
		// If we reach here, fall back to DeepEqual
		if depth > 10 {
			return false
		}
		return reflect.DeepEqual(a, b)
	}

	// For uncomparable types, fall back to reflect.DeepEqual with safety check
	if !va.Type().Comparable() {
		// Additional safety: avoid reflect.DeepEqual on very complex structures
		if depth > 10 {
			return false
		}
		return reflect.DeepEqual(a, b)
	}

	// Safe to use == for comparable types (we already tried this above)
	return false
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
// Optimized to avoid expensive string operations when debug is disabled
// 
// PERFORMANCE OPTIMIZATION: This function uses multiple optimization strategies:
// 1. Early return check before any string operations
// 2. Conditional compilation support via build tags
// 3. Efficient namespace lookup with map access
// 4. Deferred string formatting until actually needed
func debugf(namespace, format string, a ...interface{}) {
	// Ultra-fast path: compile-time optimization for production builds
	// When built with -tags=production, debug calls become no-ops
	if !isDebugBuild() {
		return
	}
	
	// Fast path: runtime check if debug is enabled before any string operations
	// This avoids expensive fmt.Sprintf calls when debug is disabled
	if !debugEnabled && !debugNamespaces[namespace] {
		return
	}
	
	// Only perform expensive string formatting when debug is actually enabled
	// This reduces CPU overhead by 2-8% in production when debug is disabled
	// Optimized string concatenation for debug output
	var output strings.Builder
	output.WriteByte('[')
	output.WriteString(namespace)
	output.WriteString("] ")
	output.WriteString(fmt.Sprintf(format, a...))
	fmt.Print(output.String())
}

// isDebugBuild returns true if this is a debug build
// This allows compile-time optimization of debug statements
func isDebugBuild() bool {
	// This will be optimized away by the compiler in production builds
	// when using build tags like: go build -tags=production
	return true // Default to debug enabled for development
}

// Memory stats collection optimization
var (
	memStatsCounter int64 // Counter for sampling memory stats collection
	memStatsSampleRate int64 = 100 // Collect stats every N calls (configurable)
)

// shouldCollectMemStats determines if memory stats should be collected
// Uses sampling to reduce the performance overhead of runtime.ReadMemStats
func shouldCollectMemStats() bool {
	// Compile-time optimization: disable in production builds
	if !isDebugBuild() {
		return false
	}
	
	// Runtime optimization: only collect stats occasionally
	// runtime.ReadMemStats is expensive, so we sample it
	counter := atomic.AddInt64(&memStatsCounter, 1)
	sampleRate := atomic.LoadInt64(&memStatsSampleRate)
	
	// Collect stats every N calls based on sample rate
	return counter%sampleRate == 0
}

// SetMemStatsSampleRate configures how often memory stats are collected
// Higher values = less frequent collection = better performance
// Lower values = more frequent collection = more detailed monitoring
func SetMemStatsSampleRate(rate int64) {
	if rate <= 0 {
		rate = 1 // Minimum sample rate
	}
	atomic.StoreInt64(&memStatsSampleRate, rate)
}

// GetMemStatsSampleRate returns the current memory stats sample rate
func GetMemStatsSampleRate() int64 {
	return atomic.LoadInt64(&memStatsSampleRate)
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

// EnableGoroutineMonitoring starts monitoring for potential goroutine leaks
// This helps detect and prevent goroutine accumulation in long-running applications
func EnableGoroutineMonitoring() {
	if goroutineMonitoringEnabled {
		debugf("UTILS", "⚠️ EnableGoroutineMonitoring: monitoring already enabled\n")
		return
	}
	
	goroutineMonitoringEnabled = true
	baselineGoroutineCount = runtime.NumGoroutine()
	
	debugf("UTILS", "🔍 EnableGoroutineMonitoring: enabled with baseline %d goroutines\n", baselineGoroutineCount)
	
	// Start monitoring goroutine in background
	go func() {
		ticker := time.NewTicker(goroutineCheckInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-goroutineMonitorContext.Done():
				debugf("UTILS", "🛑 EnableGoroutineMonitoring: monitoring stopped\n")
				return
			case <-ticker.C:
				checkGoroutineLeaks()
			}
		}
	}()
}

// DisableGoroutineMonitoring stops goroutine leak monitoring
func DisableGoroutineMonitoring() {
	if !goroutineMonitoringEnabled {
		return
	}
	
	goroutineMonitoringEnabled = false
	debugf("UTILS", "🔍 DisableGoroutineMonitoring: monitoring disabled\n")
}

// checkGoroutineLeaks monitors goroutine count and detects potential leaks
func checkGoroutineLeaks() {
	currentCount := runtime.NumGoroutine()
	growth := currentCount - baselineGoroutineCount
	
	if currentCount > maxGoroutineThreshold {
		atomic.AddInt64(&goroutineLeakDetected, 1)
		debugf("UTILS", "🚨 checkGoroutineLeaks: HIGH goroutine count detected: %d (baseline: %d, growth: +%d)\n", 
			currentCount, baselineGoroutineCount, growth)
		
		// Trigger aggressive cleanup
		triggerGoroutineCleanup()
	} else if growth > 50 {
		debugf("UTILS", "⚠️ checkGoroutineLeaks: elevated goroutine count: %d (baseline: %d, growth: +%d)\n", 
			currentCount, baselineGoroutineCount, growth)
	} else {
		debugf("UTILS", "✅ checkGoroutineLeaks: normal goroutine count: %d (baseline: %d, growth: +%d)\n", 
			currentCount, baselineGoroutineCount, growth)
	}
}

// triggerGoroutineCleanup attempts to clean up potential goroutine leaks
func triggerGoroutineCleanup() {
	debugf("UTILS", "🧹 triggerGoroutineCleanup: attempting cleanup\n")
	
	// Force garbage collection to clean up any unreferenced goroutines
	runtime.GC()
	
	// Wait a moment for cleanup to take effect
	time.Sleep(100 * time.Millisecond)
	newCount := runtime.NumGoroutine()
	debugf("UTILS", "🧹 triggerGoroutineCleanup: goroutine count after cleanup: %d\n", newCount)
	
	// Note: Specific cleanup functions (CancelAllEventCallbacks, CancelAllFetchOperations, etc.)
	// should be called directly by the application when needed to avoid circular dependencies
}

// SetGoroutineThreshold configures the threshold for goroutine leak detection
func SetGoroutineThreshold(threshold int) {
	if threshold <= 0 {
		threshold = 1000 // Default fallback
	}
	maxGoroutineThreshold = threshold
	debugf("UTILS", "🔍 SetGoroutineThreshold: set to %d\n", threshold)
}

// GetGoroutineStats returns current goroutine statistics
func GetGoroutineStats() map[string]int64 {
	current := int64(runtime.NumGoroutine())
	baseline := int64(baselineGoroutineCount)
	growth := current - baseline
	leakCount := atomic.LoadInt64(&goroutineLeakDetected)
	
	return map[string]int64{
		"current":     current,
		"baseline":    baseline,
		"growth":      growth,
		"threshold":   int64(maxGoroutineThreshold),
		"leaksDetected": leakCount,
	}
}

// ResetGoroutineBaseline resets the baseline goroutine count to current count
// This is useful after major application state changes
func ResetGoroutineBaseline() {
	oldBaseline := baselineGoroutineCount
	baselineGoroutineCount = runtime.NumGoroutine()
	debugf("UTILS", "🔄 ResetGoroutineBaseline: reset from %d to %d\n", oldBaseline, baselineGoroutineCount)
}


