//go:build js && wasm
// +build js,wasm

package fiber

import (
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
	debugf("MEMORY", "🧹 Starting callback cleanup - current count: %d\n", len(eventCallbacks)+len(rafCallbacks))

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

	debugf("MEMORY", "🧹 Callback cleanup completed\n")
}

// cleanupPools resets object pools to free up memory
func cleanupPools() {
	debugf("MEMORY", "🧹 Pool cleanup via GC (avoids extra allocations)\n")
	// Let Go's GC handle pool cleanup naturally
	runtime.GC()
	debugf("MEMORY", "🧹 Pool cleanup completed (post-GC)\n")
}

func forceGarbageCollection() {
	debugf("MEMORY", "🧹 Triggering garbage collection\n")
	runtime.GC()
	runtime.GC() // Double GC to ensure full cleanup
}

// CleanupMemory performs comprehensive memory cleanup
func CleanupMemory() {
	debugf("MEMORY", "🧹 Starting comprehensive memory cleanup\n")
	cleanupCallbacks()
	cleanupPools()
	forceGarbageCollection()
	debugf("MEMORY", "🧹 Comprehensive memory cleanup completed\n")
}

func checkMemoryPressure() {
	totalCallbacks := len(eventCallbacks) + len(rafCallbacks) + len(callbackRegistry)
	if totalCallbacks > maxCallbacks {
		debugf("MEMORY", "⚠️ High callback count detected: %d (max: %d) - triggering cleanup\n", totalCallbacks, maxCallbacks)
		CleanupMemory()
	}
}

// SetMemoryLimits allows configuration of memory management thresholds (exported for user control)
func SetMemoryLimits(maxCalls, maxPool int) {
	debugf("MEMORY", "🧹 Manual memory cleanup requested\n")
	CleanupMemory()
}

// ConfigureMemoryLimits allows runtime configuration of memory limits
func ConfigureMemoryLimits(maxCalls, maxPool int) {
	oldMaxCallbacks := maxCallbacks
	oldMaxPoolSize := maxPoolSize

	maxCallbacks = maxCalls
	maxPoolSize = maxPool

	debugf("MEMORY", "🔧 Memory limits updated - maxCallbacks: %d→%d, maxPoolSize: %d→%d\n",
		oldMaxCallbacks, maxCallbacks, oldMaxPoolSize, maxPoolSize)
}

// getHooksFromPool retrieves a Hooks instance from the pool or creates a new one
func getHooksFromPool() *Hooks {
	poolHooks := hooksPool.Get()
	if hooks, ok := poolHooks.(*Hooks); ok {
		// Reset the hooks for reuse
		hooks.index = 0
		hooks.state = hooks.state[:0]
		hooks.deps = hooks.deps[:0]
		hooks.memos = hooks.memos[:0]
		hooks.prevOrder = hooks.prevOrder[:0]
		hooks.callOrder = hooks.callOrder[:0]
		hooks.orderChecked = false
		return hooks
	}
	// This should never happen if pool is properly initialized, but handle gracefully
	debugf("MEMORY", "🚨 getHooksFromPool: hooksPool returned unexpected type %T, creating new Hooks\n", poolHooks)
	return &Hooks{
		state:        []interface{}{},
		deps:         [][]interface{}{},
		memos:        []memoizedValue{},
		prevOrder:    []HookCall{},
		callOrder:    []HookCall{},
		orderChecked: false,
	}
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
