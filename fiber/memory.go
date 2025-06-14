//go:build js && wasm
// +build js,wasm

package fiber

import (
	"fmt"
	"runtime"
	"sync"
	"syscall/js"
)

// Object pools for reducing allocations
var (
	fiberPool = sync.Pool{
		New: func() interface{} {
			return &Fiber{}
		},
	}

	hooksPool = sync.Pool{
		New: func() interface{} {
			return &Hooks{
				state:        make([]interface{}, 0, 8),   // Larger pre-allocation
				deps:         make([][]interface{}, 0, 8), // Larger pre-allocation
				memos:        make([]memoizedValue, 0, 4), // Larger pre-allocation
				callOrder:    make([]HookCall, 0, 8),      // Pre-allocate hook order tracking
				prevOrder:    make([]HookCall, 0, 8),      // Pre-allocate previous order tracking
				orderChecked: false,
			}
		},
	}

	// Element pool for createElement optimization
	elementPool = sync.Pool{
		New: func() interface{} {
			return &Element{
				Props: make(map[string]interface{}, 8), // Pre-allocated map
			}
		},
	}

	// Props pool for map reuse
	propsPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]interface{}, 8)
		},
	}

	// Children slice pool for reducing allocations
	childrenPool = sync.Pool{
		New: func() interface{} {
			return make([]interface{}, 0, 4) // Common small capacity
		},
	}
)

// Global variables for memory management
var (
	eventCallbacks   []js.Func                            // Global slice to keep event callbacks alive
	rafCallbacks     []js.Func                            // Global slice to keep callbacks alive
	callbackRegistry map[string]js.Func                   // Track callbacks by ID for cleanup
	nextCallbackID   int                                  // Counter for unique callback IDs
	maxCallbacks     int                = 1000            // Maximum callbacks before cleanup
	maxPoolSize      int                = 100             // Maximum pool size before cleanup
	emptyChildren                       = []interface{}{} // Shared empty slice to avoid allocations
)

// Initialize memory management
func init() {
	callbackRegistry = make(map[string]js.Func)
}

// cleanupCallbacks removes unused callbacks to prevent memory leaks
func cleanupCallbacks() {
	fmt.Printf("🧹 [MEMORY_CLEANUP] Starting callback cleanup - current count: %d\n", len(eventCallbacks)+len(rafCallbacks))

	// Release all js.Func objects
	for _, callback := range eventCallbacks {
		callback.Release()
	}
	for _, callback := range rafCallbacks {
		callback.Release()
	}
	for id, callback := range callbackRegistry {
		callback.Release()
		delete(callbackRegistry, id)
	}

	// Clear or shrink slices to free backing arrays if they grew too big
	const shrinkCap = 128
	const defaultCap = 32

	if cap(eventCallbacks) > shrinkCap {
		eventCallbacks = make([]js.Func, 0, defaultCap)
	} else {
		eventCallbacks = eventCallbacks[:0]
	}

	if cap(rafCallbacks) > shrinkCap {
		rafCallbacks = make([]js.Func, 0, defaultCap)
	} else {
		rafCallbacks = rafCallbacks[:0]
	}

	fmt.Printf("🧹 [MEMORY_CLEANUP] Callback cleanup completed\n")
}

// cleanupPools prevents object pools from growing too large
func cleanupPools() {
	fmt.Printf("🧹 [MEMORY_CLEANUP] Pool cleanup via GC (avoids extra allocations)\n")
	// sync.Pool automatically drops cached items on GC.
	runtime.GC()
	runtime.GC()
	fmt.Printf("🧹 [MEMORY_CLEANUP] Pool cleanup completed (post-GC)\n")
}

// forceGarbageCollection triggers garbage collection
func forceGarbageCollection() {
	fmt.Printf("🧹 [MEMORY_CLEANUP] Triggering garbage collection\n")
	runtime.GC()
	runtime.GC() // Call twice to ensure cleanup
}

// performMemoryCleanup performs comprehensive memory cleanup
func performMemoryCleanup() {
	fmt.Printf("🧹 [MEMORY_CLEANUP] Starting comprehensive memory cleanup\n")
	cleanupCallbacks()
	cleanupPools()
	forceGarbageCollection()
	fmt.Printf("🧹 [MEMORY_CLEANUP] Comprehensive memory cleanup completed\n")
}

// checkMemoryPressure checks if we need to perform cleanup
func checkMemoryPressure() {
	totalCallbacks := len(eventCallbacks) + len(rafCallbacks) + len(callbackRegistry)
	if totalCallbacks > maxCallbacks {
		fmt.Printf("⚠️ [MEMORY_PRESSURE] High callback count detected: %d (max: %d) - triggering cleanup\n", totalCallbacks, maxCallbacks)
		performMemoryCleanup()
	}
}

// CleanupMemory provides a public API for manual memory cleanup
// This can be called by applications when they want to force cleanup
func CleanupMemory() {
	fmt.Printf("🧹 [PUBLIC_API] Manual memory cleanup requested\n")
	performMemoryCleanup()
}

// GetMemoryStats returns current memory usage statistics
func GetMemoryStats() map[string]int {
	return map[string]int{
		"eventCallbacks":   len(eventCallbacks),
		"rafCallbacks":     len(rafCallbacks),
		"callbackRegistry": len(callbackRegistry),
		"totalCallbacks":   len(eventCallbacks) + len(rafCallbacks) + len(callbackRegistry),
		"maxCallbacks":     maxCallbacks,
		"maxPoolSize":      maxPoolSize,
	}
}

// SetMemoryLimits allows applications to configure memory management thresholds
func SetMemoryLimits(maxCb, maxPool int) {
	fmt.Printf("🔧 [CONFIG] Memory limits updated - maxCallbacks: %d→%d, maxPoolSize: %d→%d\n",
		maxCallbacks, maxCb, maxPoolSize, maxPool)
	maxCallbacks = maxCb
	maxPoolSize = maxPool
}

// getHooksFromPool gets hooks from pool with reset
func getHooksFromPool() *Hooks {
	poolHooks := hooksPool.Get()
	hooks, ok := poolHooks.(*Hooks)
	if !ok {
		// This should never happen if pool is properly initialized, but handle gracefully
		fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] getHooksFromPool: hooksPool returned unexpected type %T, creating new Hooks\n", poolHooks)
		hooks = &Hooks{
			state:     []interface{}{},
			deps:      [][]interface{}{},
			memos:     []memoizedValue{},
			callOrder: []HookCall{},
			prevOrder: []HookCall{},
		}
	}
	hooks.index = 0
	// Reuse slices, just reset length
	hooks.state = hooks.state[:0]
	hooks.deps = hooks.deps[:0]
	hooks.memos = hooks.memos[:0]
	hooks.callOrder = hooks.callOrder[:0]
	// CRITICAL FIX: Clear prevOrder for new component - each component needs its own validation state
	hooks.prevOrder = hooks.prevOrder[:0]
	hooks.orderChecked = false
	return hooks
}

// releaseHooks returns hooks to pool
func releaseHooks(hooks *Hooks) {
	if hooks != nil {
		// Don't clear slices, just reset for reuse
		hooksPool.Put(hooks)
	}
}

// resetFiber resets fiber for pool reuse
func resetFiber(f *Fiber) {
	f.parent = nil
	f.alternate = nil
	f.child = nil
	f.sibling = nil
	if f.hooks != nil {
		releaseHooks(f.hooks)
		f.hooks = nil
	}
	f.typeOf = nil
	f.props = nil
	f.dom = js.Value{}
	f.effectTag = ""
	f.effects = f.effects[:0] // Reuse slice
}

// releaseElement releases element back to pool with optimized cleanup
func releaseElement(elem *Element) {
	if elem != nil {
		elem.Type = nil
		elem.Children = nil
		// Clear props map efficiently but keep it allocated for reuse
		if len(elem.Props) > 0 {
			for k := range elem.Props {
				delete(elem.Props, k)
			}
		}
		elementPool.Put(elem)
	}
}
