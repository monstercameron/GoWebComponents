//go:build js && wasm
// +build js,wasm

package fiber

import (
	"runtime"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"
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

	// Fetch result channel pool for reducing channel allocations during fetch operations
	// PERFORMANCE OPTIMIZATION: Optimized buffer size for fetch operations
	fetchChannelPool = sync.Pool{
		New: func() interface{} {
			// Buffer size of 2 allows for result + potential error without blocking
			// This handles the common case of response + json parsing or error
			return make(chan FetchResult, 2) // Optimized buffer size for fetch operations
		},
	}
)

// Global variables for memory management with dynamic pool sizing
var (
	eventCallbacks   []js.Func                            // Global slice to keep event callbacks alive
	rafCallbacks     []js.Func                            // Global slice to keep callbacks alive
	maxCallbacks     int                = 1000            // Maximum callbacks before cleanup
	emptyChildren                       = []interface{}{} // Shared empty slice to avoid allocations
	
	// Dynamic pool sizing configuration
	poolConfig = struct {
		// Base pool sizes (minimum)
		basePoolSize     int32 // Base size for all pools
		maxPoolSize      int32 // Maximum size for all pools
		
		// Pool-specific configurations
		fiberPoolSize    int32 // Current fiber pool target size
		hooksPoolSize    int32 // Current hooks pool target size
		elementPoolSize  int32 // Current element pool target size
		
		// Adaptive sizing parameters
		growthThreshold  float64 // Utilization % to trigger growth
		shrinkThreshold  float64 // Utilization % to trigger shrinking
		growthFactor     float64 // Multiplier for pool growth
		shrinkFactor     float64 // Multiplier for pool shrinking
		
		// Monitoring
		adaptiveEnabled  bool    // Enable adaptive pool sizing
		lastOptimization int64   // Timestamp of last optimization
		optimizationInterval int64 // Minimum interval between optimizations (ms)
	}{
		basePoolSize:     32,   // Minimum pool size
		maxPoolSize:      512,  // Maximum pool size to prevent excessive memory usage
		fiberPoolSize:    64,   // Start with reasonable fiber pool size
		hooksPoolSize:    48,   // Hooks are used frequently
		elementPoolSize:  96,   // Elements are created/destroyed often
		growthThreshold:  0.8,  // Grow when 80% utilized
		shrinkThreshold:  0.3,  // Shrink when below 30% utilized
		growthFactor:     1.5,  // Grow by 50%
		shrinkFactor:     0.7,  // Shrink by 30%
		adaptiveEnabled:  true, // Enable adaptive sizing by default
		optimizationInterval: 5000, // Optimize at most every 5 seconds
	}
	
	// Pool size tracking for enforcement and monitoring
	poolSizes = struct {
		fiber        int32
		hooks        int32
		element      int32
		props        int32
		children     int32
		effectFibers int32
		fetchChannel int32
	}{}
	
	// Pool utilization tracking for adaptive sizing
	poolUtilization = struct {
		fiberHits    int64 // Successful pool retrievals
		fiberMisses  int64 // Pool was empty, created new
		hooksHits    int64
		hooksMisses  int64
		elementHits  int64
		elementMisses int64
		
		// Performance metrics
		totalAllocations int64 // Total objects allocated
		totalPoolHits    int64 // Total successful pool retrievals
		poolHitRate      float64 // Cache hit rate percentage
	}{}
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
	oldMaxPoolSize := atomic.LoadInt32(&poolConfig.maxPoolSize)

	maxCallbacks = maxCalls
	atomic.StoreInt32(&poolConfig.maxPoolSize, int32(maxPool))
	
	// Update all pool target sizes to not exceed new maximum
	if atomic.LoadInt32(&poolConfig.fiberPoolSize) > int32(maxPool) {
		atomic.StoreInt32(&poolConfig.fiberPoolSize, int32(maxPool))
	}
	if atomic.LoadInt32(&poolConfig.hooksPoolSize) > int32(maxPool) {
		atomic.StoreInt32(&poolConfig.hooksPoolSize, int32(maxPool))
	}
	if atomic.LoadInt32(&poolConfig.elementPoolSize) > int32(maxPool) {
		atomic.StoreInt32(&poolConfig.elementPoolSize, int32(maxPool))
	}

	debugf("MEMORY", "🔧 Memory limits updated - maxCallbacks: %d→%d, maxPoolSize: %d→%d\n",
		oldMaxCallbacks, maxCallbacks, oldMaxPoolSize, int32(maxPool))
}

// GetPoolSizes returns current pool sizes for monitoring
func GetPoolSizes() map[string]int32 {
	return map[string]int32{
		"fiber":        atomic.LoadInt32(&poolSizes.fiber),
		"hooks":        atomic.LoadInt32(&poolSizes.hooks),
		"element":      atomic.LoadInt32(&poolSizes.element),
		"props":        atomic.LoadInt32(&poolSizes.props),
		"children":     atomic.LoadInt32(&poolSizes.children),
		"effectFibers": atomic.LoadInt32(&poolSizes.effectFibers),
		"fetchChannel": atomic.LoadInt32(&poolSizes.fetchChannel),
	}
}

// getHooksFromPool retrieves a Hooks instance from the pool or creates a new one
func getHooksFromPool() *Hooks {
	debugf("MEMORY", "♻️ getHooksFromPool: retrieving hooks from pool\n")
	atomic.AddInt64(&poolUtilization.totalAllocations, 1)
	
	poolHooks := hooksPool.Get()
	if hooks, ok := poolHooks.(*Hooks); ok {
		// Pool hit - track metrics
		atomic.AddInt64(&poolUtilization.hooksHits, 1)
		atomic.AddInt64(&poolUtilization.totalPoolHits, 1)
		atomic.AddInt32(&poolSizes.hooks, -1)
		debugf("MEMORY", "✅ getHooksFromPool: reusing hooks %p from pool (hit)\n", hooks)
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
	// Pool miss - track metrics and create new
	atomic.AddInt64(&poolUtilization.hooksMisses, 1)
	debugf("MEMORY", "🚨 getHooksFromPool: hooksPool returned unexpected type %T, creating new Hooks (miss)\n", poolHooks)
	newHooks := &Hooks{
		state:        []interface{}{},
		deps:         [][]interface{}{},
		memos:        []memoizedValue{},
		prevOrder:    []HookCall{},
		callOrder:    []HookCall{},
		orderChecked: false,
	}
	debugf("MEMORY", "🆕 getHooksFromPool: created new hooks %p (miss)\n", newHooks)
	
	// Trigger pool optimization periodically
	optimizePoolSizes()
	
	return newHooks
}

// releaseHooks returns hooks to pool with adaptive size enforcement
func releaseHooks(hooks *Hooks) {
	if hooks != nil {
		debugf("MEMORY", "♻️ releaseHooks: returning hooks %p to pool (state: %d, deps: %d, memos: %d)\n",
			hooks, len(hooks.state), len(hooks.deps), len(hooks.memos))
		
		// Enforce adaptive pool size limits
		currentSize := atomic.LoadInt32(&poolSizes.hooks)
		targetSize := atomic.LoadInt32(&poolConfig.hooksPoolSize)
		if currentSize < targetSize {
			atomic.AddInt32(&poolSizes.hooks, 1)
			hooksPool.Put(hooks)
			debugf("MEMORY", "✅ releaseHooks: hooks returned to pool (pool size: %d/%d)\n", currentSize+1, targetSize)
		} else {
			debugf("MEMORY", "🚨 releaseHooks: pool size limit reached (%d), discarding hooks to prevent overflow\n", targetSize)
			// Let GC handle the discarded hooks
		}
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

// releaseElement releases element back to pool with optimized cleanup and adaptive size enforcement
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

		// Enforce adaptive pool size limits
		currentSize := atomic.LoadInt32(&poolSizes.element)
		targetSize := atomic.LoadInt32(&poolConfig.elementPoolSize)
		if currentSize < targetSize {
			atomic.AddInt32(&poolSizes.element, 1)
			elementPool.Put(elem)
			debugf("MEMORY", "✅ releaseElement: element returned to pool (pool size: %d/%d)\n", currentSize+1, targetSize)
		} else {
			debugf("MEMORY", "🚨 releaseElement: pool size limit reached (%d), discarding element to prevent overflow\n", targetSize)
			// Let GC handle the discarded element
		}
	} else {
		debugf("MEMORY", "🚨 releaseElement: attempted to release nil element\n")
	}
}

// getEffectFibersSlice retrieves a pooled slice for effect fibers collection
func getEffectFibersSlice() *[]*Fiber {
	debugf("MEMORY", "♻️ getEffectFibersSlice: retrieving slice from pool\n")
	poolSlice := effectFibersPool.Get()
	if slice, ok := poolSlice.(*[]*Fiber); ok {
		// Decrement pool size counter when retrieving from pool
		atomic.AddInt32(&poolSizes.effectFibers, -1)
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

// returnEffectFibersSlice returns a slice to the pool with smart capacity management and size enforcement
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

	// Enforce adaptive pool size limits (use base pool size for effect fibers)
	currentSize := atomic.LoadInt32(&poolSizes.effectFibers)
	targetSize := atomic.LoadInt32(&poolConfig.basePoolSize) // Effect fibers use base size
	if currentSize < targetSize {
		atomic.AddInt32(&poolSizes.effectFibers, 1)
		// Clear the slice and return to pool
		*slice = (*slice)[:0]
		effectFibersPool.Put(slice)
		debugf("MEMORY", "✅ returnEffectFibersSlice: slice returned to pool (pool size: %d/%d)\n", currentSize+1, targetSize)
	} else {
		debugf("MEMORY", "🚨 returnEffectFibersSlice: pool size limit reached (%d), discarding slice to prevent overflow\n", targetSize)
		// Let GC handle the discarded slice
	}
}

// getFetchChannel retrieves a pooled channel for fetch operations
func getFetchChannel() chan FetchResult {
	debugf("MEMORY", "♻️ getFetchChannel: retrieving channel from pool\n")
	poolChan := fetchChannelPool.Get()
	if ch, ok := poolChan.(chan FetchResult); ok {
		// Decrement pool size counter when retrieving from pool
		atomic.AddInt32(&poolSizes.fetchChannel, -1)
		// Drain any leftover data from previous use
		select {
		case <-ch:
			debugf("MEMORY", "🧹 getFetchChannel: drained leftover data from reused channel\n")
		default:
			// Channel is empty, ready for use
		}
		debugf("MEMORY", "✅ getFetchChannel: reused channel from pool\n")
		return ch
	}
	// This should never happen if pool is properly initialized
	debugf("MEMORY", "🚨 getFetchChannel: pool returned unexpected type %T, creating new channel\n", poolChan)
	// Use optimized buffer size for fallback channel
	return make(chan FetchResult, 2)
}

// returnFetchChannel returns a channel to the pool with size enforcement
func returnFetchChannel(ch chan FetchResult) {
	if ch == nil {
		debugf("MEMORY", "🚨 returnFetchChannel: attempted to return nil channel\n")
		return
	}

	debugf("MEMORY", "♻️ returnFetchChannel: returning channel to pool\n")

	// Drain any remaining data to clean the channel for reuse
	select {
	case <-ch:
		debugf("MEMORY", "🧹 returnFetchChannel: drained remaining data from channel\n")
	default:
		// Channel is empty
	}

	// Enforce adaptive pool size limits (use base pool size for fetch channels)
	currentSize := atomic.LoadInt32(&poolSizes.fetchChannel)
	targetSize := atomic.LoadInt32(&poolConfig.basePoolSize) // Fetch channels use base size
	if currentSize < targetSize {
		atomic.AddInt32(&poolSizes.fetchChannel, 1)
		fetchChannelPool.Put(ch)
		debugf("MEMORY", "✅ returnFetchChannel: channel returned to pool (pool size: %d/%d)\n", currentSize+1, targetSize)
	} else {
		debugf("MEMORY", "🚨 returnFetchChannel: pool size limit reached (%d), discarding channel to prevent overflow\n", targetSize)
		// Let GC handle the discarded channel
	}
}

// optimizePoolSizes analyzes usage patterns and adjusts pool sizes dynamically
func optimizePoolSizes() {
	if !poolConfig.adaptiveEnabled {
		return
	}
	
	// Rate limiting: don't optimize too frequently
	now := time.Now().UnixMilli()
	if now-poolConfig.lastOptimization < poolConfig.optimizationInterval {
		return
	}
	poolConfig.lastOptimization = now
	
	debugf("MEMORY", "📊 optimizePoolSizes: analyzing pool utilization patterns\n")
	
	// Calculate hit rates for each pool type
	fiberHitRate := calculateHitRate(poolUtilization.fiberHits, poolUtilization.fiberMisses)
	hooksHitRate := calculateHitRate(poolUtilization.hooksHits, poolUtilization.hooksMisses)
	elementHitRate := calculateHitRate(poolUtilization.elementHits, poolUtilization.elementMisses)
	
	// Update global hit rate
	totalHits := poolUtilization.fiberHits + poolUtilization.hooksHits + poolUtilization.elementHits
	totalMisses := poolUtilization.fiberMisses + poolUtilization.hooksMisses + poolUtilization.elementMisses
	poolUtilization.poolHitRate = calculateHitRate(totalHits, totalMisses)
	
	debugf("MEMORY", "📊 Pool hit rates - Fiber: %.1f%%, Hooks: %.1f%%, Element: %.1f%%, Overall: %.1f%%\n",
		fiberHitRate*100, hooksHitRate*100, elementHitRate*100, poolUtilization.poolHitRate*100)
	
	// Optimize individual pool sizes based on hit rates
	optimizeIndividualPool("fiber", &poolConfig.fiberPoolSize, fiberHitRate)
	optimizeIndividualPool("hooks", &poolConfig.hooksPoolSize, hooksHitRate)
	optimizeIndividualPool("element", &poolConfig.elementPoolSize, elementHitRate)
}

// calculateHitRate computes cache hit rate from hits and misses
func calculateHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 1.0 // Perfect hit rate when no activity
	}
	return float64(hits) / float64(total)
}

// optimizeIndividualPool adjusts a single pool's target size based on hit rate
func optimizeIndividualPool(poolName string, targetSize *int32, hitRate float64) {
	currentSize := atomic.LoadInt32(targetSize)
	newSize := currentSize
	
	// Grow pool if hit rate is too low (many misses)
	if hitRate < poolConfig.shrinkThreshold {
		newSize = int32(float64(currentSize) * poolConfig.growthFactor)
		if newSize > poolConfig.maxPoolSize {
			newSize = poolConfig.maxPoolSize
		}
		if newSize != currentSize {
			atomic.StoreInt32(targetSize, newSize)
			debugf("MEMORY", "📈 optimizeIndividualPool: grew %s pool %d→%d (hit rate: %.1f%%)\n",
				poolName, currentSize, newSize, hitRate*100)
		}
	}
	
	// Shrink pool if hit rate is very high (pool may be oversized)
	if hitRate > poolConfig.growthThreshold && currentSize > poolConfig.basePoolSize {
		newSize = int32(float64(currentSize) * poolConfig.shrinkFactor)
		if newSize < poolConfig.basePoolSize {
			newSize = poolConfig.basePoolSize
		}
		if newSize != currentSize {
			atomic.StoreInt32(targetSize, newSize)
			debugf("MEMORY", "📉 optimizeIndividualPool: shrunk %s pool %d→%d (hit rate: %.1f%%)\n",
				poolName, currentSize, newSize, hitRate*100)
		}
	}
}

// SetPoolSizingConfig configures the adaptive pool sizing parameters
func SetPoolSizingConfig(baseSize, maxSize int32, growthThreshold, shrinkThreshold float64) {
	if baseSize < 8 {
		baseSize = 8 // Absolute minimum
	}
	if maxSize < baseSize {
		maxSize = baseSize * 4
	}
	if maxSize > 2048 {
		maxSize = 2048 // Absolute maximum to prevent excessive memory usage
	}
	if growthThreshold <= 0 || growthThreshold >= 1 {
		growthThreshold = 0.8 // Default
	}
	if shrinkThreshold <= 0 || shrinkThreshold >= growthThreshold {
		shrinkThreshold = 0.3 // Default
	}
	
	oldBase, oldMax := poolConfig.basePoolSize, poolConfig.maxPoolSize
	poolConfig.basePoolSize = baseSize
	poolConfig.maxPoolSize = maxSize
	poolConfig.growthThreshold = growthThreshold
	poolConfig.shrinkThreshold = shrinkThreshold
	
	debugf("MEMORY", "🔧 SetPoolSizingConfig: base=%d→%d, max=%d→%d, growth=%.1f%%, shrink=%.1f%%\n",
		oldBase, baseSize, oldMax, maxSize, growthThreshold*100, shrinkThreshold*100)
}

// EnableAdaptivePoolSizing enables or disables adaptive pool sizing
func EnableAdaptivePoolSizing(enabled bool) {
	poolConfig.adaptiveEnabled = enabled
	debugf("MEMORY", "🔧 EnableAdaptivePoolSizing: %v\n", enabled)
}

// GetPoolUtilizationStats returns comprehensive pool utilization statistics
func GetPoolUtilizationStats() map[string]interface{} {
	return map[string]interface{}{
		"hitRates": map[string]float64{
			"fiber":   calculateHitRate(poolUtilization.fiberHits, poolUtilization.fiberMisses),
			"hooks":   calculateHitRate(poolUtilization.hooksHits, poolUtilization.hooksMisses),
			"element": calculateHitRate(poolUtilization.elementHits, poolUtilization.elementMisses),
			"overall": poolUtilization.poolHitRate,
		},
		"targetSizes": map[string]int32{
			"fiber":   atomic.LoadInt32(&poolConfig.fiberPoolSize),
			"hooks":   atomic.LoadInt32(&poolConfig.hooksPoolSize),
			"element": atomic.LoadInt32(&poolConfig.elementPoolSize),
		},
		"currentSizes": map[string]int32{
			"fiber":        atomic.LoadInt32(&poolSizes.fiber),
			"hooks":        atomic.LoadInt32(&poolSizes.hooks),
			"element":      atomic.LoadInt32(&poolSizes.element),
			"props":        atomic.LoadInt32(&poolSizes.props),
			"children":     atomic.LoadInt32(&poolSizes.children),
			"effectFibers": atomic.LoadInt32(&poolSizes.effectFibers),
			"fetchChannel": atomic.LoadInt32(&poolSizes.fetchChannel),
		},
		"metrics": map[string]int64{
			"totalAllocations": atomic.LoadInt64(&poolUtilization.totalAllocations),
			"totalPoolHits":    atomic.LoadInt64(&poolUtilization.totalPoolHits),
			"fiberHits":        atomic.LoadInt64(&poolUtilization.fiberHits),
			"fiberMisses":      atomic.LoadInt64(&poolUtilization.fiberMisses),
			"hooksHits":        atomic.LoadInt64(&poolUtilization.hooksHits),
			"hooksMisses":      atomic.LoadInt64(&poolUtilization.hooksMisses),
			"elementHits":      atomic.LoadInt64(&poolUtilization.elementHits),
			"elementMisses":    atomic.LoadInt64(&poolUtilization.elementMisses),
		},
		"config": map[string]interface{}{
			"adaptiveEnabled":  poolConfig.adaptiveEnabled,
			"basePoolSize":     poolConfig.basePoolSize,
			"maxPoolSize":      poolConfig.maxPoolSize,
			"growthThreshold":  poolConfig.growthThreshold,
			"shrinkThreshold":  poolConfig.shrinkThreshold,
		},
	}
}

// ResetPoolUtilizationStats resets all pool utilization counters
func ResetPoolUtilizationStats() {
	atomic.StoreInt64(&poolUtilization.fiberHits, 0)
	atomic.StoreInt64(&poolUtilization.fiberMisses, 0)
	atomic.StoreInt64(&poolUtilization.hooksHits, 0)
	atomic.StoreInt64(&poolUtilization.hooksMisses, 0)
	atomic.StoreInt64(&poolUtilization.elementHits, 0)
	atomic.StoreInt64(&poolUtilization.elementMisses, 0)
	atomic.StoreInt64(&poolUtilization.totalAllocations, 0)
	atomic.StoreInt64(&poolUtilization.totalPoolHits, 0)
	poolUtilization.poolHitRate = 0.0
	debugf("MEMORY", "🔄 ResetPoolUtilizationStats: all counters reset\n")
}
