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

	// Effect fibers slice pool for reducing allocations during effect execution
	effectFibersPool = sync.Pool{
		New: func() interface{} {
			slice := make([]*Fiber, 0, 16) // Reasonable initial capacity for most apps
			return &slice
		},
	}
)

// Global variables for memory management
var (
	eventCallbacks   []js.Func                            // Global slice to keep event callbacks alive
	rafCallbacks     []js.Func                            // Global slice to keep callbacks alive
	maxCallbacks     int                = 1000            // Maximum callbacks before cleanup
	maxPoolSize      int                = 100             // Maximum pool size before cleanup
	emptyChildren                       = []interface{}{} // Shared empty slice to avoid allocations
)

// Initialize memory management
func init() {
	// Removed unused callbackRegistry initialization
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
	totalCallbacks := len(eventCallbacks) + len(rafCallbacks)
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
	debugf("MEMORY", "♻️ getHooksFromPool: retrieving hooks from pool\n")
	poolHooks := hooksPool.Get()
	if hooks, ok := poolHooks.(*Hooks); ok {
		debugf("MEMORY", "✅ getHooksFromPool: reusing hooks %p from pool\n", hooks)
		// Reset the hooks for reuse with smart slice management
		debugf("MEMORY", "🔄 getHooksFromPool: resetting hooks - state: %d, deps: %d, memos: %d\n",
			len(hooks.state), len(hooks.deps), len(hooks.memos))
		hooks.index = 0
		
		// Smart slice management: shrink backing arrays if they grew too large
		const maxHookSliceCapacity = 32 // Reasonable upper bound for most components
		const defaultHookSliceCapacity = 8 // Default capacity for new slices
		
		// Reset state slice with capacity management
		if cap(hooks.state) > maxHookSliceCapacity {
			oldCap := cap(hooks.state)
			hooks.state = make([]interface{}, 0, defaultHookSliceCapacity)
			debugf("MEMORY", "🧹 getHooksFromPool: shrunk state slice capacity %d→%d\n", oldCap, defaultHookSliceCapacity)
		} else {
			hooks.state = hooks.state[:0]
		}
		
		// Reset deps slice with capacity management
		if cap(hooks.deps) > maxHookSliceCapacity {
			oldCap := cap(hooks.deps)
			hooks.deps = make([][]interface{}, 0, defaultHookSliceCapacity)
			debugf("MEMORY", "🧹 getHooksFromPool: shrunk deps slice capacity %d→%d\n", oldCap, defaultHookSliceCapacity)
		} else {
			hooks.deps = hooks.deps[:0]
		}
		
		// Reset memos slice with capacity management
		if cap(hooks.memos) > maxHookSliceCapacity {
			oldCap := cap(hooks.memos)
			hooks.memos = make([]memoizedValue, 0, defaultHookSliceCapacity)
			debugf("MEMORY", "🧹 getHooksFromPool: shrunk memos slice capacity %d→%d\n", oldCap, defaultHookSliceCapacity)
		} else {
			hooks.memos = hooks.memos[:0]
		}
		
		// Reset order tracking slices (these typically stay small)
		hooks.prevOrder = hooks.prevOrder[:0]
		hooks.callOrder = hooks.callOrder[:0]
		hooks.orderChecked = false
		debugf("MEMORY", "✅ getHooksFromPool: hooks reset complete\n")
		return hooks
	}
	// This should never happen if pool is properly initialized, but handle gracefully
	debugf("MEMORY", "🚨 getHooksFromPool: hooksPool returned unexpected type %T, creating new Hooks\n", poolHooks)
	newHooks := &Hooks{
		state:        []interface{}{},
		deps:         [][]interface{}{},
		memos:        []memoizedValue{},
		prevOrder:    []HookCall{},
		callOrder:    []HookCall{},
		orderChecked: false,
	}
	debugf("MEMORY", "🆕 getHooksFromPool: created new hooks %p\n", newHooks)
	return newHooks
}

// releaseHooks returns hooks to pool
func releaseHooks(hooks *Hooks) {
	if hooks != nil {
		debugf("MEMORY", "♻️ releaseHooks: returning hooks %p to pool (state: %d, deps: %d, memos: %d)\n",
			hooks, len(hooks.state), len(hooks.deps), len(hooks.memos))
		// Don't clear slices, just reset for reuse
		hooksPool.Put(hooks)
		debugf("MEMORY", "✅ releaseHooks: hooks returned to pool\n")
	} else {
		debugf("MEMORY", "🚨 releaseHooks: attempted to release nil hooks\n")
	}
}

// resetFiber resets fiber for pool reuse
func resetFiber(f *Fiber) {
	debugf("MEMORY", "🔄 resetFiber: resetting fiber %p (type: %v, effects: %d)\n",
		f, f.typeOf, len(f.effects))

	f.parent = nil
	f.alternate = nil
	f.child = nil
	f.sibling = nil
	if f.hooks != nil {
		debugf("MEMORY", "♻️ resetFiber: releasing hooks from fiber\n")
		releaseHooks(f.hooks)
		f.hooks = nil
	}
	f.typeOf = nil
	f.props = nil
	f.dom = js.Value{}
	f.effectTag = ""
	f.effects = f.effects[:0]        // Reuse slice
	f.eventCallbacks = f.eventCallbacks[:0] // Reuse slice

	debugf("MEMORY", "✅ resetFiber: fiber reset complete\n")
}

// releaseElement releases element back to pool with optimized cleanup
func releaseElement(elem *Element) {
	if elem != nil {
		debugf("MEMORY", "♻️ releaseElement: returning element %p to pool (type: %v, props: %d, children: %d)\n",
			elem, elem.Type, len(elem.Props), len(elem.Children))

		elem.Type = nil
		elem.Children = nil
		// Clear props map efficiently but keep it allocated for reuse
		propsCleared := 0
		if len(elem.Props) > 0 {
			for k := range elem.Props {
				delete(elem.Props, k)
				propsCleared++
			}
		}
		debugf("MEMORY", "🧹 releaseElement: cleared %d props from element\n", propsCleared)

		elementPool.Put(elem)
		debugf("MEMORY", "✅ releaseElement: element returned to pool\n")
	} else {
		debugf("MEMORY", "🚨 releaseElement: attempted to release nil element\n")
	}
}

// getEffectFibersSlice retrieves a pooled slice for effect fibers collection
func getEffectFibersSlice() *[]*Fiber {
	debugf("MEMORY", "♻️ getEffectFibersSlice: retrieving slice from pool\n")
	poolSlice := effectFibersPool.Get()
	if slice, ok := poolSlice.(*[]*Fiber); ok {
		// Reset slice length but keep capacity
		*slice = (*slice)[:0]
		debugf("MEMORY", "✅ getEffectFibersSlice: reused slice with capacity %d\n", cap(*slice))
		return slice
	}
	// This should never happen if pool is properly initialized
	debugf("MEMORY", "🚨 getEffectFibersSlice: pool returned unexpected type %T, creating new slice\n", poolSlice)
	newSlice := make([]*Fiber, 0, 16)
	return &newSlice
}

// returnEffectFibersSlice returns a slice to the pool with smart capacity management
func returnEffectFibersSlice(slice *[]*Fiber) {
	if slice == nil {
		debugf("MEMORY", "🚨 returnEffectFibersSlice: attempted to return nil slice\n")
		return
	}

	debugf("MEMORY", "♻️ returnEffectFibersSlice: returning slice to pool (len: %d, cap: %d)\n",
		len(*slice), cap(*slice))

	// Smart capacity management: don't return overly large slices to pool
	const maxEffectFibersCapacity = 64 // Reasonable upper bound
	if cap(*slice) > maxEffectFibersCapacity {
		debugf("MEMORY", "🧹 returnEffectFibersSlice: slice capacity %d too large, not returning to pool\n", cap(*slice))
		// Let GC handle the oversized slice, don't pollute the pool
		return
	}

	// Clear the slice and return to pool
	*slice = (*slice)[:0]
	effectFibersPool.Put(slice)
	debugf("MEMORY", "✅ returnEffectFibersSlice: slice returned to pool\n")
}
