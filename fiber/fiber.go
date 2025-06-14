// ./fiber/fiber.go

//go:build js && wasm
// +build js,wasm

package fiber

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall/js"
	"unsafe"
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

// Global variables for tracking the current fiber and root.
var (
	wipRoot         *Fiber
	currentRoot     *Fiber
	nextUnitOfWork  *Fiber
	deletions       []*Fiber
	wipFiber        *Fiber
	eventCallbacks  []js.Func // Global slice to keep event callbacks alive
	rafCallbacks    []js.Func // Global slice to keep callbacks alive
	updateScheduled bool      // Flag to prevent multiple update scheduling

	// Shared empty slice to avoid allocations
	emptyChildren = []interface{}{}

	// Memory management tracking
	callbackRegistry map[string]js.Func        // Track callbacks by ID for cleanup
	nextCallbackID   int                       // Counter for unique callback IDs
	maxCallbacks     int                = 1000 // Maximum callbacks before cleanup
	maxPoolSize      int                = 100  // Maximum pool size before cleanup
)

// Initialize memory management
func init() {
	callbackRegistry = make(map[string]js.Func)
}

// Memory management and cleanup functions

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
	fmt.Printf("🧹 [MEMORY_CLEANUP] Starting pool cleanup\n")

	// Clean up element pool if it's too large
	poolSize := 0
	for {
		elem := elementPool.Get()
		if elem == nil {
			break
		}
		poolSize++
		if poolSize > maxPoolSize {
			// Don't put it back - let it be garbage collected
			fmt.Printf("🧹 [MEMORY_CLEANUP] Discarding excess element from pool\n")
		} else {
			elementPool.Put(elem)
		}
	}

	// Clean up hooks pool if it's too large
	poolSize = 0
	for {
		hooks := hooksPool.Get()
		if hooks == nil {
			break
		}
		poolSize++
		if poolSize > maxPoolSize {
			// Don't put it back - let it be garbage collected
			fmt.Printf("🧹 [MEMORY_CLEANUP] Discarding excess hooks from pool\n")
		} else {
			hooksPool.Put(hooks)
		}
	}

	// Clean up fiber pool if it's too large
	poolSize = 0
	for {
		fiber := fiberPool.Get()
		if fiber == nil {
			break
		}
		poolSize++
		if poolSize > maxPoolSize {
			// Don't put it back - let it be garbage collected
			fmt.Printf("🧹 [MEMORY_CLEANUP] Discarding excess fiber from pool\n")
		} else {
			fiberPool.Put(fiber)
		}
	}

	fmt.Printf("🧹 [MEMORY_CLEANUP] Pool cleanup completed\n")
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

// Element represents a virtual DOM node.
type Element struct {
	Type     interface{}
	Props    map[string]interface{}
	Children []interface{}
}

// NEW: Fast equality checking without reflection
type FastComparable interface {
	FastEqual(other interface{}) bool
}

// NEW: Common primitive type fast equality with cycle detection
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

// createElement constructs an Element with optimized allocations
func createElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	// Get element from pool with safe type assertion
	poolElem := elementPool.Get()
	elem, ok := poolElem.(*Element)
	if !ok {
		// This should never happen if pool is properly initialized, but handle gracefully
		fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] createElement: elementPool returned unexpected type %T, creating new Element\n", poolElem)
		elem = &Element{
			Props: make(map[string]interface{}),
		}
	}

	// Reset the element
	elem.Type = typ

	// Process children to support both component references and return values
	processedChildren := make([]interface{}, 0, len(children))

	for _, child := range children {
		if child == nil {
			continue
		}

		// Check if child is a component function reference (map[string]interface{} signature)
		if componentFunc, ok := child.(func(map[string]interface{}) *Element); ok {
			// Call the component function with nil props
			result := componentFunc(nil)
			if result != nil {
				processedChildren = append(processedChildren, result)
			}
		} else if componentFunc, ok := child.(func(Attrs) *Element); ok {
			// Check if child is a component function reference (Attrs signature)
			// Call the component function with nil props
			result := componentFunc(nil)
			if result != nil {
				processedChildren = append(processedChildren, result)
			}
		} else {
			// Child is already processed (Element, Text, etc.)
			processedChildren = append(processedChildren, child)
		}
	}

	elem.Children = processedChildren

	// Handle props efficiently
	if props != nil {
		// Clear existing props map efficiently
		for k := range elem.Props {
			delete(elem.Props, k)
		}
		// Copy props (map is already allocated)
		for k, v := range props {
			elem.Props[k] = v
		}
	} else {
		// Clear props if none provided
		for k := range elem.Props {
			delete(elem.Props, k)
		}
	}

	// Set children in props
	if len(processedChildren) > 0 {
		elem.Props["children"] = processedChildren
	} else {
		// Use a shared empty slice to avoid allocations while maintaining type safety
		elem.Props["children"] = emptyChildren
	}

	return elem
}

// Release element back to pool with optimized cleanup
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

// Text creates a text node.
func Text(content string) *Element {
	return createElement("TEXT_ELEMENT", map[string]interface{}{
		"nodeValue": content,
	})
}

// GoUseState manages state in a component with optimized equality checking
// This is the Go-branded version of useState for GoWebComponents
func GoUseState[T any](initialValue T) (func() T, func(T)) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeState, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] GoUseState: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	if len(currentFiber.hooks.state) > position {
		// Existing state
	} else {
		// Initial state - grow slice efficiently
		if cap(currentFiber.hooks.state) <= position {
			// Double capacity when needed
			newCap := max(8, len(currentFiber.hooks.state)*2)
			newState := make([]interface{}, len(currentFiber.hooks.state), newCap)
			copy(newState, currentFiber.hooks.state)
			currentFiber.hooks.state = newState
		}
		currentFiber.hooks.state = append(currentFiber.hooks.state, initialValue)
	}

	// Capture hooks and position
	hooks := currentFiber.hooks
	idx := position

	getter := func() T {
		// Bounds check to prevent index out of range panic
		if idx >= len(hooks.state) {
			var zero T
			return zero
		}
		if value, ok := hooks.state[idx].(T); ok {
			return value
		}
		// Return zero value if type assertion fails
		var zero T
		return zero
	}

	setter := func(newValue T) {
		// Bounds check to prevent index out of range panic
		if idx >= len(hooks.state) {
			// This shouldn't happen if the getter/setter are used correctly,
			// but we'll handle it gracefully by expanding the state array
			for len(hooks.state) <= idx {
				hooks.state = append(hooks.state, nil)
			}
		}
		// Use fast equality check instead of reflect.DeepEqual
		if hooks.state[idx] == nil || !fastEqual(hooks.state[idx], newValue) {
			hooks.state[idx] = newValue
			scheduleUpdateAtRoot()
		}
	}

	return getter, setter
}

// validateHookOrder checks that hooks are called in the same order as previous render
func validateHookOrder(hooks *Hooks, hookType HookType, position int) error {
	// Debug: Log that validation is being called with detailed info
	fmt.Printf("🔍 [HOOK_DEBUG] validateHookOrder called: type=%d, position=%d, prevOrder=%d, callOrder=%d\n",
		hookType, position, len(hooks.prevOrder), len(hooks.callOrder))

	// Record this hook call
	hookCall := HookCall{Type: hookType, Position: position}
	hooks.callOrder = append(hooks.callOrder, hookCall)

	// Skip validation on first render (no previous order to compare)
	if len(hooks.prevOrder) == 0 {
		fmt.Printf("🔍 [HOOK_DEBUG] Skipping validation - first render (no prevOrder)\n")
		return nil
	}

	// Debug: Show what we're comparing
	fmt.Printf("🔍 [HOOK_DEBUG] Validating: current hook type=%d pos=%d vs previous hook type=%d pos=%d\n",
		hookType, position,
		func() int {
			if position < len(hooks.prevOrder) {
				return int(hooks.prevOrder[position].Type)
			} else {
				return -1
			}
		}(),
		func() int {
			if position < len(hooks.prevOrder) {
				return hooks.prevOrder[position].Position
			} else {
				return -1
			}
		}())

	// Check if we have a corresponding hook call from previous render
	if position >= len(hooks.prevOrder) {
		return fmt.Errorf("hook order violation: more hooks called than previous render (position %d >= prevOrder length %d)", position, len(hooks.prevOrder))
	}

	prevCall := hooks.prevOrder[position]
	if prevCall.Type != hookType {
		return fmt.Errorf("hook order violation: hook type mismatch at position %d (expected %d, got %d)",
			position, prevCall.Type, hookType)
	}

	if prevCall.Position != position {
		return fmt.Errorf("hook order violation: position mismatch at index %d (expected %d, got %d)",
			position, prevCall.Position, position)
	}

	fmt.Printf("✅ [HOOK_DEBUG] Hook order validation passed for position %d\n", position)
	return nil
}

// finalizeHookOrder completes hook order validation after all hooks have been called
func finalizeHookOrder(hooks *Hooks) error {
	// Debug: Log finalization
	fmt.Printf("🔍 [HOOK_DEBUG] finalizeHookOrder called: callOrder=%d, prevOrder=%d\n",
		len(hooks.callOrder), len(hooks.prevOrder))

	// Check if fewer hooks were called than previous render
	if len(hooks.prevOrder) > 0 && len(hooks.callOrder) < len(hooks.prevOrder) {
		return fmt.Errorf("hook order violation: fewer hooks called than previous render (%d vs %d)",
			len(hooks.callOrder), len(hooks.prevOrder))
	}

	// Move current order to previous order for next render
	hooks.prevOrder = make([]HookCall, len(hooks.callOrder))
	copy(hooks.prevOrder, hooks.callOrder)

	// Debug: Log the transfer
	fmt.Printf("🔍 [HOOK_DEBUG] Transferred %d hook calls to prevOrder for next render\n", len(hooks.prevOrder))

	// Reset current order for next render
	hooks.callOrder = hooks.callOrder[:0]
	hooks.orderChecked = true

	return nil
}

// NEW: Get hooks from pool with reset
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

// NEW: Return hooks to pool
func releaseHooks(hooks *Hooks) {
	if hooks != nil {
		// Don't clear slices, just reset for reuse
		hooksPool.Put(hooks)
	}
}

func scheduleUpdateAtRoot() {
	if currentRoot == nil || updateScheduled {
		return
	}
	updateScheduled = true

	// Reuse fiber instead of allocating new one
	if wipRoot == nil {
		poolFiber := fiberPool.Get()
		if fiber, ok := poolFiber.(*Fiber); ok {
			wipRoot = fiber
		} else {
			// This should never happen if pool is properly initialized, but handle gracefully
			fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] scheduleUpdateAtRoot: fiberPool returned unexpected type %T, creating new Fiber\n", poolFiber)
			wipRoot = &Fiber{
				props: make(map[string]interface{}),
			}
		}
	}

	wipRoot.typeOf = currentRoot.typeOf
	wipRoot.dom = currentRoot.dom
	wipRoot.props = currentRoot.props
	wipRoot.alternate = currentRoot
	wipRoot.effectTag = ""
	wipRoot.parent = nil
	wipRoot.child = nil
	wipRoot.sibling = nil

	nextUnitOfWork = wipRoot
	deletions = deletions[:0] // Reuse slice
	requestIdleCallback(workLoop)
}

type memoizedValue struct {
	value interface{}
	deps  []interface{}
}

// HookType represents the type of hook being called
type HookType int

const (
	HookTypeState HookType = iota
	HookTypeEffect
	HookTypeMemo
	HookTypeFunc
)

// HookCall represents a single hook call for order validation
type HookCall struct {
	Type     HookType
	Position int
}

// Hooks struct optimized for memory alignment and access patterns with order validation
type Hooks struct {
	// Hot path data - accessed most frequently
	index int // 8 bytes (padded)

	// Slice headers grouped together (each is 24 bytes)
	state []interface{}   // 24 bytes
	deps  [][]interface{} // 24 bytes
	memos []memoizedValue // 24 bytes

	// Hook order validation - tracks the sequence of hook calls
	callOrder    []HookCall // 24 bytes - current render's hook call sequence
	prevOrder    []HookCall // 24 bytes - previous render's hook call sequence
	orderChecked bool       // 1 byte - whether order has been validated this render
	// Total: 129 bytes (padded to 136 bytes for alignment)
}

func useEffect(effect func(), deps ...interface{}) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeEffect, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] useEffect: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Grow deps slice efficiently
	for len(currentFiber.hooks.deps) <= position {
		currentFiber.hooks.deps = append(currentFiber.hooks.deps, nil)
	}

	if currentFiber.hooks.deps[position] == nil {
		// First time this effect is used
		currentFiber.hooks.deps[position] = deps
		// Grow effects slice efficiently
		if cap(currentFiber.effects) <= len(currentFiber.effects) {
			newCap := max(4, cap(currentFiber.effects)*2)
			newEffects := make([]func(), len(currentFiber.effects), newCap)
			copy(newEffects, currentFiber.effects)
			currentFiber.effects = newEffects
		}
		currentFiber.effects = append(currentFiber.effects, effect)
	} else {
		prevDeps := currentFiber.hooks.deps[position]
		shouldRun := len(deps) == 0 || !areDepsEqual(prevDeps, deps)
		if shouldRun {
			// Dependencies have changed or no dependencies provided
			currentFiber.hooks.deps[position] = deps
			currentFiber.effects = append(currentFiber.effects, effect)
		}
	}
}

// GoUseEffect runs side effects in a component with dependency tracking
// This is the Go-branded version of useEffect for GoWebComponents
func GoUseEffect(effect func(), deps ...interface{}) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeEffect, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] GoUseEffect: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Grow deps slice efficiently
	for len(currentFiber.hooks.deps) <= position {
		currentFiber.hooks.deps = append(currentFiber.hooks.deps, nil)
	}

	if currentFiber.hooks.deps[position] == nil {
		// First time this effect is used
		currentFiber.hooks.deps[position] = deps
		// Grow effects slice efficiently
		if cap(currentFiber.effects) <= len(currentFiber.effects) {
			newCap := max(4, cap(currentFiber.effects)*2)
			newEffects := make([]func(), len(currentFiber.effects), newCap)
			copy(newEffects, currentFiber.effects)
			currentFiber.effects = newEffects
		}
		currentFiber.effects = append(currentFiber.effects, effect)
	} else {
		prevDeps := currentFiber.hooks.deps[position]
		shouldRun := len(deps) == 0 || !areDepsEqual(prevDeps, deps)
		if shouldRun {
			// Dependencies have changed or no dependencies provided
			currentFiber.hooks.deps[position] = deps
			currentFiber.effects = append(currentFiber.effects, effect)
		}
	}
}

// Optimized dependency comparison
func areDepsEqual(prevDeps, newDeps []interface{}) bool {
	if len(prevDeps) != len(newDeps) {
		return false
	}

	// Fast path for empty deps
	if len(prevDeps) == 0 {
		return true
	}

	// Use fast equality check
	for i := range prevDeps {
		if !fastEqual(prevDeps[i], newDeps[i]) {
			return false
		}
	}
	return true
}

func useMemo(compute func() interface{}, deps ...interface{}) interface{} {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeMemo, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] useMemo: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Grow memos slice efficiently
	for len(currentFiber.hooks.memos) <= position {
		currentFiber.hooks.memos = append(currentFiber.hooks.memos, memoizedValue{})
	}

	memo := &currentFiber.hooks.memos[position]

	if memo.value == nil {
		// First time this memo is used
		value := compute() // Remove goroutine overhead for simple computations
		memo.value = value
		memo.deps = deps
		return value
	}

	shouldCompute := len(deps) == 0 || !areDepsEqual(memo.deps, deps)
	if shouldCompute {
		value := compute() // Direct call, no goroutine
		memo.value = value
		memo.deps = deps
		return value
	}

	// Dependencies haven't changed, return the memoized value
	return memo.value
}

// GoUseMemo memoizes expensive computations with dependency tracking
// This is the Go-branded version of useMemo for GoWebComponents
func GoUseMemo(compute func() interface{}, deps ...interface{}) interface{} {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeMemo, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] GoUseMemo: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Grow memos slice efficiently
	for len(currentFiber.hooks.memos) <= position {
		currentFiber.hooks.memos = append(currentFiber.hooks.memos, memoizedValue{})
	}

	memo := &currentFiber.hooks.memos[position]

	if memo.value == nil {
		// First time this memo is used
		value := compute() // Remove goroutine overhead for simple computations
		memo.value = value
		memo.deps = deps
		return value
	}

	shouldCompute := len(deps) == 0 || !areDepsEqual(memo.deps, deps)
	if shouldCompute {
		value := compute() // Direct call, no goroutine
		memo.value = value
		memo.deps = deps
		return value
	}

	// Dependencies haven't changed, return the memoized value
	return memo.value
}

// NEW: Utility function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Fiber represents a unit of work in the virtual DOM tree.
// Fiber struct optimized for memory alignment and cache efficiency
type Fiber struct {
	// Group pointers together for better cache locality (40 bytes)
	parent    *Fiber // 8 bytes
	alternate *Fiber // 8 bytes
	child     *Fiber // 8 bytes
	sibling   *Fiber // 8 bytes
	hooks     *Hooks // 8 bytes

	// Group interface and map together (48 bytes)
	typeOf interface{}            // 16 bytes
	props  map[string]interface{} // 8 bytes
	dom    js.Value               // 24 bytes

	// Smaller types grouped at end (40 bytes)
	effectTag string   // 16 bytes
	effects   []func() // 24 bytes
}

// NEW: Reset fiber for pool reuse
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

// getCurrentFiber retrieves the current working fiber.
func getCurrentFiber() *Fiber {
	return wipFiber
}

// scheduleUpdate triggers a re-render of the component.
func scheduleUpdate(fiber *Fiber) {
	// fmt.Println("scheduleUpdate: Scheduling update")
	wipRoot = &Fiber{
		typeOf:    "ROOT",
		dom:       currentRoot.dom,
		props:     currentRoot.props,
		alternate: currentRoot,
	}
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	// fmt.Println("scheduleUpdate: wipRoot set and workLoop scheduled")
	requestIdleCallback(workLoop)
}

// render starts the rendering process.
func render(element *Element, container js.Value) {
	// fmt.Println("render: Starting rendering process.")
	wipRoot = &Fiber{
		typeOf:    "ROOT", // Assign a type to the root fiber
		dom:       container,
		props:     map[string]interface{}{"children": []interface{}{element}},
		alternate: currentRoot,
	}
	fmt.Println("render: Root fiber created.")
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	// fmt.Println("render: Scheduling work loop.")
	requestIdleCallback(workLoop)
}

// Render is the exported version of render - starts the rendering process
func Render(element *Element, container js.Value) {
	render(element, container)
}

// CreateElement is the exported version of createElement
func CreateElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	return createElement(typ, props, children...)
}

// workLoop performs work until there is no more work left or the deadline expires.
func workLoop(deadline js.Value) {
	// fmt.Println("workLoop: Starting work loop.")
	var shouldYield bool = false
	for nextUnitOfWork != nil && !shouldYield {
		// fmt.Println("workLoop: Performing a unit of work.")
		nextUnitOfWork = performUnitOfWork(nextUnitOfWork)
		shouldYield = deadline.Call("timeRemaining").Float() < 1
		// fmt.Printf("workLoop: timeRemaining=%f, shouldYield=%v\n", deadline.Call("timeRemaining").Float(), shouldYield)
	}

	if wipRoot != nil && nextUnitOfWork == nil {
		// fmt.Println("workLoop: No more units of work. Committing root.")
		commitRoot()
	}

	if nextUnitOfWork != nil {
		// fmt.Println("workLoop: Work remains. Scheduling next work loop.")
		requestIdleCallback(workLoop)
	} else {
		// fmt.Println("workLoop: All work completed.")
	}
}

// performUnitOfWork performs a single unit of work.
func performUnitOfWork(fiber *Fiber) *Fiber {
	if fiber == nil {
		// fmt.Println("performUnitOfWork: Fiber is nil.")
		return nil
	}

	// fmt.Printf("performUnitOfWork: Processing fiber of type %v.\n", fiber.typeOf)

	if fiber.typeOf == nil || fiber.typeOf == "ROOT" {
		// fmt.Println("performUnitOfWork: Fiber has typeOf nil or ROOT, reconciling children.")
		if children, ok := fiber.props["children"].([]interface{}); ok {
			reconcileChildren(fiber, children)
		} else {
			// Handle case where children is not the expected type
			fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] performUnitOfWork: fiber.props[\"children\"] is not []interface{}, got %T\n", fiber.props["children"])
			// Try to reconcile with empty children to avoid crash
			emptyChildren := make([]interface{}, 0)
			reconcileChildren(fiber, emptyChildren)
		}
	} else {
		switch fiber.typeOf.(type) {
		case func(map[string]interface{}) *Element:
			// Function component with map[string]interface{} props
			componentFunc, ok := fiber.typeOf.(func(map[string]interface{}) *Element)
			if !ok {
				fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] performUnitOfWork: fiber.typeOf is not func(map[string]interface{}) *Element, got %T\n", fiber.typeOf)
				return nil
			}
			wipFiber = fiber

			// Preserve hooks from alternate fiber
			var oldHooks *Hooks
			if fiber.alternate != nil {
				oldHooks = fiber.alternate.hooks
			}

			if oldHooks != nil {
				// Reuse existing Hooks instance and reset per-render state
				wipFiber.hooks = oldHooks
				wipFiber.hooks.index = 0
				wipFiber.hooks.callOrder = wipFiber.hooks.callOrder[:0]
				wipFiber.hooks.orderChecked = false
				// prevOrder already contains the last render's sequence
			} else {
				// First render – allocate a fresh Hooks container
				wipFiber.hooks = &Hooks{
					state:        []interface{}{},
					deps:         [][]interface{}{},
					memos:        []memoizedValue{},
					prevOrder:    []HookCall{},
					callOrder:    []HookCall{},
					orderChecked: false,
				}
			}

			// Initialize effects
			wipFiber.effects = []func(){}

			element := componentFunc(fiber.props)

			// Finalize hook order validation after component execution
			if wipFiber.hooks != nil && !wipFiber.hooks.orderChecked {
				if err := finalizeHookOrder(wipFiber.hooks); err != nil {
					componentName := getFunctionName(fiber.typeOf)
					fmt.Printf("🚨 [HOOK_ORDER_ERROR] Component '%s': %v\n", componentName, err)
				}
			}

			if element == nil {
				return nil
			}

			reconcileChildren(fiber, []interface{}{element})
		case func(Attrs) *Element:
			// Function component with Attrs props
			componentFunc, ok := fiber.typeOf.(func(Attrs) *Element)
			if !ok {
				fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] performUnitOfWork: fiber.typeOf is not func(Attrs) *Element, got %T\n", fiber.typeOf)
				return nil
			}
			wipFiber = fiber

			// Preserve hooks from alternate fiber
			var oldHooks *Hooks
			if fiber.alternate != nil {
				oldHooks = fiber.alternate.hooks
			}

			if oldHooks != nil {
				// Reuse existing Hooks instance and reset per-render state
				wipFiber.hooks = oldHooks
				wipFiber.hooks.index = 0
				wipFiber.hooks.callOrder = wipFiber.hooks.callOrder[:0]
				wipFiber.hooks.orderChecked = false
				// prevOrder already contains the last render's sequence
			} else {
				// First render – allocate a fresh Hooks container
				wipFiber.hooks = &Hooks{
					state:        []interface{}{},
					deps:         [][]interface{}{},
					memos:        []memoizedValue{},
					prevOrder:    []HookCall{},
					callOrder:    []HookCall{},
					orderChecked: false,
				}
			}

			// Initialize effects
			wipFiber.effects = []func(){}

			// Convert map[string]interface{} to Attrs
			var attrs Attrs
			if fiber.props != nil {
				attrs = Attrs(fiber.props)
			}

			element := componentFunc(attrs)

			// Finalize hook order validation after component execution
			if wipFiber.hooks != nil && !wipFiber.hooks.orderChecked {
				if err := finalizeHookOrder(wipFiber.hooks); err != nil {
					componentName := getFunctionName(fiber.typeOf)
					fmt.Printf("🚨 [HOOK_ORDER_ERROR] Component '%s': %v\n", componentName, err)
				}
			}

			if element == nil {
				return nil
			}

			reconcileChildren(fiber, []interface{}{element})
		case string:
			// Host component (HTML element)
			// fmt.Printf("performUnitOfWork: Handling host component of type '%s'.\n", fiber.typeOf.(string))
			if fiber.dom.IsUndefined() || fiber.dom.IsNull() {
				// fmt.Println("performUnitOfWork: Creating DOM node for host component.")
				fiber.dom = createDom(fiber)
				// fmt.Println("performUnitOfWork: DOM node created.")
			}

			if fiber.props == nil {
				// fmt.Println("performUnitOfWork: Fiber props are nil. Skipping children reconciliation.")
				return nil
			}

			if propsChildren, ok := fiber.props["children"]; ok {
				// fmt.Println("performUnitOfWork: Reconciling children of host component.")
				if elements, elementsOk := propsChildren.([]interface{}); elementsOk {
					reconcileChildren(fiber, elements)
				} else {
					fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] performUnitOfWork: fiber.props[\"children\"] is not []interface{}, got %T\n", propsChildren)
					// Try to reconcile with empty children to avoid crash
					emptyChildren := make([]interface{}, 0)
					reconcileChildren(fiber, emptyChildren)
				}
			}
		default:
			// fmt.Printf("performUnitOfWork: Unhandled fiber type %T.\n", fiber.typeOf)
		}
	}

	// fmt.Printf("performUnitOfWork: Completed processing fiber of type %v.\n", fiber.typeOf)

	// Traverse to child fibers
	if fiber.child != nil {
		// fmt.Printf("performUnitOfWork: Moving to child fiber of type %v.\n", fiber.child.typeOf)
		return fiber.child
	}

	nextFiber := fiber
	for nextFiber != nil {
		if nextFiber.sibling != nil {
			// fmt.Printf("performUnitOfWork: Moving to sibling fiber of type %v.\n", nextFiber.sibling.typeOf)
			return nextFiber.sibling
		}
		// fmt.Println("performUnitOfWork: Moving up to parent fiber.")
		nextFiber = nextFiber.parent
	}
	// fmt.Println("performUnitOfWork: No more fibers to process.")
	return nil
}

// createDom creates a DOM node from a fiber.
func createDom(fiber *Fiber) js.Value {
	// fmt.Printf("createDom: Creating DOM for fiber type %v\n", fiber.typeOf)
	var dom js.Value
	switch t := fiber.typeOf.(type) {
	case string:
		if t == "TEXT_ELEMENT" {
			dom = js.Global().Get("document").Call("createTextNode", fiber.props["nodeValue"])
		} else {
			dom = js.Global().Get("document").Call("createElement", t)
		}
	default:
		// Function components do not create DOM nodes here
		// fmt.Println("createDom: Function component, no DOM node created")
		return js.Value{}
	}

	// Add event listeners and properties
	for name, value := range fiber.props {
		if name == "children" {
			continue
		}
		if name == "dangerouslySetInnerHTML" {
			// Set innerHTML directly
			if htmlMap, ok := value.(map[string]string); ok {
				if htmlContent, htmlOk := htmlMap["__html"]; htmlOk {
					// fmt.Println("createDom: Setting innerHTML")
					dom.Set("innerHTML", htmlContent)
				} else {
					fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] createDom: dangerouslySetInnerHTML missing __html key\n")
				}
			} else {
				fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] createDom: dangerouslySetInnerHTML is not map[string]string, got %T\n", value)
			}
			continue
		}
		if len(name) > 2 && name[:2] == "on" {
			// Event handlers
			eventType := strings.ToLower(name[2:]) // Convert event type to lowercase
			// fmt.Printf("createDom: Adding event listener for %s\n", eventType)

			// Ensure the value is of the correct function type
			eventHandler, ok := value.(js.Func)
			if !ok {
				// fmt.Printf("createDom: Event handler for %s is not a js.Func\n", eventType)
				continue
			}

			dom.Call("addEventListener", eventType, eventHandler)
			continue
		}
		if name == "class" {
			// Handle 'class' attribute using setAttribute
			// fmt.Printf("createDom: Setting attribute 'class' to '%v'\n", value)
			dom.Call("setAttribute", "class", value)
			continue
		}
		// Set other properties directly
		// fmt.Printf("createDom: Setting property '%s' to '%v'\n", name, value)
		dom.Set(name, value)
	}
	return dom
}

// reconcileChildren reconciles the children of a fiber.
func reconcileChildren(wipFiber *Fiber, elements []interface{}) {
	// fmt.Printf("reconcileChildren: Reconciling %d children for fiber type %v\n", len(elements), wipFiber.typeOf)
	index := 0
	var oldFiber *Fiber
	if wipFiber.alternate != nil {
		oldFiber = wipFiber.alternate.child
	}
	var prevSibling *Fiber

	for index < len(elements) || oldFiber != nil {
		var element interface{}
		if index < len(elements) {
			element = elements[index]
		}

		var newFiber *Fiber

		sameType := false
		if oldFiber != nil && element != nil {
			if elem, ok := element.(*Element); ok {
				switch elemType := elem.Type.(type) {
				case func(map[string]interface{}) *Element:
					// Function component with map[string]interface{} props: Compare function pointers using reflect
					funcPtrNew := reflect.ValueOf(elemType).Pointer()
					funcPtrOld, ok := oldFiber.typeOf.(func(map[string]interface{}) *Element)
					if ok {
						funcPtrOldValue := reflect.ValueOf(funcPtrOld).Pointer()
						if funcPtrNew == funcPtrOldValue {
							sameType = true
						}
					}
				case func(Attrs) *Element:
					// Function component with Attrs props: Compare function pointers using reflect
					funcPtrNew := reflect.ValueOf(elemType).Pointer()
					funcPtrOld, ok := oldFiber.typeOf.(func(Attrs) *Element)
					if ok {
						funcPtrOldValue := reflect.ValueOf(funcPtrOld).Pointer()
						if funcPtrNew == funcPtrOldValue {
							sameType = true
						}
					}
				case string:
					// Host component: Use reflect.DeepEqual for string comparison
					if reflect.DeepEqual(elem.Type, oldFiber.typeOf) {
						sameType = true
					}
				default:
					// Other types: Use reflect.DeepEqual
					if reflect.DeepEqual(elem.Type, oldFiber.typeOf) {
						sameType = true
					}
				}
			} else {
				fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] reconcileChildren: element is not *Element, got %T\n", element)
			}
		}

		if sameType {
			// Reuse the existing fiber
			// fmt.Printf("reconcileChildren: Reusing existing fiber of type %v\n", oldFiber.typeOf)
			if elem, ok := element.(*Element); ok {
				newFiber = &Fiber{
					typeOf:    oldFiber.typeOf,
					props:     elem.Props,
					dom:       oldFiber.dom,
					parent:    wipFiber,
					alternate: oldFiber,
					effectTag: "UPDATE",
				}
			} else {
				fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] reconcileChildren: element is not *Element for reuse, got %T\n", element)
			}
		} else if element != nil {
			// Create a new fiber
			// fmt.Printf("reconcileChildren: Creating new fiber of type %v\n", element.(*Element).Type)
			if elem, ok := element.(*Element); ok {
				newFiber = &Fiber{
					typeOf:    elem.Type,
					props:     elem.Props,
					dom:       js.Value{},
					parent:    wipFiber,
					effectTag: "PLACEMENT",
				}
			} else {
				fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] reconcileChildren: element is not *Element for creation, got %T\n", element)
			}
		}

		if oldFiber != nil && !sameType {
			// Mark the old fiber for deletion
			// fmt.Printf("reconcileChildren: Deleting fiber of type %v\n", oldFiber.typeOf)
			oldFiber.effectTag = "DELETION"
			deletions = append(deletions, oldFiber)
		}

		if oldFiber != nil {
			oldFiber = oldFiber.sibling
		}

		if index == 0 {
			wipFiber.child = newFiber
			// fmt.Println("reconcileChildren: Setting first child fiber")
		} else if element != nil && prevSibling != nil {
			prevSibling.sibling = newFiber
			// fmt.Printf("reconcileChildren: Linking sibling fiber of type %v\n", newFiber.typeOf)
		}

		prevSibling = newFiber
		index++
	}

	// fmt.Printf("reconcileChildren: Completed reconciliation for fiber type %v\n", wipFiber.typeOf)
}

// commitRoot commits the changes to the DOM.
func commitRoot() {
	// fmt.Println("commitRoot: Starting to commit changes to DOM")
	for _, deletion := range deletions {
		// fmt.Printf("commitRoot: Processing deletion for fiber type %v\n", deletion.typeOf)
		commitWork(deletion)
	}
	if wipRoot.child != nil {
		// fmt.Printf("commitRoot: Committing child fiber of type %v\n", wipRoot.child.typeOf)
		commitWork(wipRoot.child)
	}
	currentRoot = wipRoot
	wipRoot = nil
	deletions = nil
	updateScheduled = false // Reset flag after commit
	// fmt.Println("commitRoot: Finished committing changes to DOM")

	// Execute effects after committing
	executeEffects()

	// Check memory pressure after each commit cycle
	checkMemoryPressure()
}

func executeEffects() {
	if currentRoot == nil {
		return
	}

	var effectFibers []*Fiber
	var collectEffects func(fiber *Fiber)
	collectEffects = func(fiber *Fiber) {
		if fiber == nil {
			return
		}
		if len(fiber.effects) > 0 {
			effectFibers = append(effectFibers, fiber)
		}
		collectEffects(fiber.child)
		collectEffects(fiber.sibling)
	}

	// Collect fibers with effects starting from the root
	collectEffects(currentRoot.child)

	// Execute effects sequentially to avoid race conditions and ensure predictable order
	for _, fiber := range effectFibers {
		for _, effect := range fiber.effects {
			if effect != nil {
				effect()
			}
		}
		// Clear the effects after executing them to prevent accumulation
		fiber.effects = nil // Set to nil instead of empty slice to release memory
	}

	// Clear the slice to prevent memory leaks
	effectFibers = nil
}

// commitWork recursively commits work to the DOM.
func commitWork(fiber *Fiber) {
	if fiber == nil {
		return
	}
	var domParentFiber = fiber.parent
	for domParentFiber != nil && (domParentFiber.dom.IsUndefined() || domParentFiber.dom.IsNull()) {
		domParentFiber = domParentFiber.parent
	}
	if domParentFiber == nil {
		// fmt.Println("commitWork: No valid parent DOM fiber found")
		return
	}
	domParent := domParentFiber.dom

	switch fiber.effectTag {
	case "PLACEMENT":
		if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
			// fmt.Printf("commitWork: Appending child %v to parent %v\n", fiber.dom, domParent)
			domParent.Call("appendChild", fiber.dom)
		} else {
			// fmt.Println("commitWork: Fiber has no DOM node, committing its children")
			commitWork(fiber.child)
			return
		}
	case "UPDATE":
		if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
			// fmt.Printf("commitWork: Updating DOM node for fiber type %v\n", fiber.typeOf)
			updateDom(fiber.dom, fiber.alternate.props, fiber.props)
		}
	case "DELETION":
		// fmt.Println("commitWork: Deleting DOM node")
		commitDeletion(fiber, domParent)
		return
	}

	// Commit children and siblings
	commitWork(fiber.child)
	commitWork(fiber.sibling)
}

func commitDeletion(fiber *Fiber, domParent js.Value) {
	// Enhanced cleanup for deleted components
	if fiber.hooks != nil {
		fmt.Printf("🧹 [COMPONENT_CLEANUP] Cleaning up hooks for deleted component\n")

		// Release event callbacks associated with this fiber
		for _, state := range fiber.hooks.state {
			if fn, ok := state.(js.Func); ok {
				fmt.Printf("🧹 [COMPONENT_CLEANUP] Releasing js.Func from state\n")
				fn.Release()
			}
		}

		// Release hooks back to pool
		releaseHooks(fiber.hooks)
		fiber.hooks = nil
	}

	// Clear effects to prevent memory leaks
	if len(fiber.effects) > 0 {
		fmt.Printf("🧹 [COMPONENT_CLEANUP] Clearing %d effects for deleted component\n", len(fiber.effects))
		fiber.effects = nil
	}

	if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
		// fmt.Printf("commitDeletion: Removing child %v from parent %v\n", fiber.dom, domParent)
		domParent.Call("removeChild", fiber.dom)
	} else if fiber.child != nil {
		// fmt.Println("commitDeletion: Deleting child fibers recursively")
		commitDeletion(fiber.child, domParent)
	}

	// Recursively cleanup children and siblings
	if fiber.child != nil {
		commitDeletion(fiber.child, domParent)
	}
	if fiber.sibling != nil {
		commitDeletion(fiber.sibling, domParent)
	}
}

func updateDom(dom js.Value, oldProps, newProps map[string]interface{}) {
	// Fast path: check if maps are equal first
	if len(oldProps) == 0 && len(newProps) == 0 {
		return
	}

	// Ultra-fast path: pointer equality check
	if unsafe.Pointer(&oldProps) == unsafe.Pointer(&newProps) {
		return
	}

	// 1. Remove old or changed event listeners (optimized)
	for name, oldValue := range oldProps {
		// Branch optimization: check first character before string operations
		if len(name) > 2 && name[0] == 'o' && name[1] == 'n' {
			// Only remove if not in new props or value changed
			if newValue, exists := newProps[name]; !exists || !fastEqual(oldValue, newValue) {
				eventType := strings.ToLower(name[2:])
				if oldHandler, ok := oldValue.(js.Func); ok {
					dom.Call("removeEventListener", eventType, oldHandler)
				} else {
					fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] updateDom: old event handler %s is not js.Func, got %T\n", name, oldValue)
				}
			}
		} else if newProps[name] == nil && name != "children" {
			// Remove properties that no longer exist, excluding event listeners and children
			dom.Set(name, js.Undefined())
		}
	}

	// 2. Add new or changed properties and event listeners (optimized)
	for name, value := range newProps {
		// Skip common exclusions first (most frequent check)
		if name == "children" {
			continue
		}

		// Skip if value hasn't changed
		if oldValue, exists := oldProps[name]; exists && fastEqual(oldValue, value) {
			continue
		}

		// Branch optimization: inline checks for most common patterns
		switch name {
		case "class":
			dom.Call("setAttribute", "class", value)
		case "style":
			dom.Set("style", value)
		case "id":
			dom.Set("id", value)
		case "value":
			dom.Set("value", value)
		case "dangerouslySetInnerHTML":
			if htmlMap, ok := value.(map[string]string); ok {
				if htmlContent, htmlOk := htmlMap["__html"]; htmlOk {
					dom.Set("innerHTML", htmlContent)
				} else {
					fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] updateDom: dangerouslySetInnerHTML missing __html key\n")
				}
			} else {
				fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] updateDom: dangerouslySetInnerHTML is not map[string]string, got %T\n", value)
			}
		default:
			// Check for event handlers (less common)
			if len(name) > 2 && name[0] == 'o' && name[1] == 'n' {
				eventType := strings.ToLower(name[2:])
				if eventHandler, ok := value.(js.Func); ok {
					dom.Call("addEventListener", eventType, eventHandler)
				} else {
					fmt.Printf("🚨 [TYPE_ASSERTION_ERROR] updateDom: event handler %s is not js.Func, got %T\n", name, value)
				}
			} else {
				dom.Set(name, value)
			}
		}
	}
}

// requestIdleCallback schedules work during idle periods.
func requestIdleCallback(callback func(js.Value)) {
	// Check memory pressure before creating new callback
	checkMemoryPressure()

	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback(args[0])
		return nil
	})
	rafCallbacks = append(rafCallbacks, cb) // Keep the function alive

	fmt.Printf("🔗 [CALLBACK_CREATED] requestIdleCallback created - total callbacks: %d\n", len(eventCallbacks)+len(rafCallbacks))
	js.Global().Call("requestIdleCallback", cb)
}

func useFunc(callback func(js.Value, []js.Value) interface{}) js.Func {
	// Check memory pressure before creating new callback
	checkMemoryPressure()

	cb := js.FuncOf(callback)
	eventCallbacks = append(eventCallbacks, cb) // Keep callback alive

	fmt.Printf("🔗 [CALLBACK_CREATED] useFunc callback created - total callbacks: %d\n", len(eventCallbacks)+len(rafCallbacks))
	return cb
}

// GoEvent wraps JavaScript events with convenient Go methods
type GoEvent struct {
	jsEvent js.Value
}

// NewGoEvent creates a GoEvent from a JavaScript event
func NewGoEvent(jsEvent js.Value) GoEvent {
	return GoEvent{jsEvent: jsEvent}
}

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
			fmt.Println("GoUseFunc: callback must be a function")
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

	eventCallbacks = append(eventCallbacks, cb) // Keep callback alive

	fmt.Printf("🔗 [CALLBACK_CREATED] GoUseFunc callback created - total callbacks: %d\n", len(eventCallbacks)+len(rafCallbacks))
	return cb
}

type FetchState struct {
	Data    interface{}
	Error   string
	Loading bool
}

// Implement FastComparable for FetchState to ensure proper state updates
func (fs FetchState) FastEqual(other interface{}) bool {
	otherFS, ok := other.(FetchState)
	if !ok {
		return false
	}

	// Compare Loading and Error first (simple comparisons)
	if fs.Loading != otherFS.Loading || fs.Error != otherFS.Error {
		return false
	}

	// For Data field, use simple pointer/nil comparison to avoid deep comparison issues
	// This ensures that any data change triggers an update
	if (fs.Data == nil) != (otherFS.Data == nil) {
		return false
	}

	// If both are nil, they're equal
	if fs.Data == nil && otherFS.Data == nil {
		return true
	}

	// If both are non-nil, consider them different to force updates
	// This is safe because fetch operations should produce new data objects
	return false
}

type FetchOptions struct {
	Method  string
	Headers map[string]interface{}
	Body    interface{}
}

type FetchResult struct {
	Data interface{}
	Err  error
}

// GoUseFetch is a hook that simplifies data fetching within a component.
// It manages loading, error, and data states automatically.
// It returns a getter for the current FetchState and a function to trigger a refetch.
func GoUseFetch(url string, options ...FetchOptions) (func() FetchState, func()) {
	getState, setState := GoUseState(FetchState{Loading: true})

	var opts FetchOptions
	if len(options) > 0 {
		opts = options[0]
	}

	fetchData := func() {
		fmt.Println("useFetch: Fetching data from", url)

		// Set loading state
		setState(FetchState{Loading: true})

		// Create fetch options
		fetchOptions := js.Global().Get("Object").New()
		if opts.Method != "" {
			fetchOptions.Set("method", opts.Method)
		}
		if len(opts.Headers) > 0 {
			headers := js.Global().Get("Object").New()
			for key, value := range opts.Headers {
				headers.Set(key, value)
			}
			fetchOptions.Set("headers", headers)
		}
		if opts.Body != nil {
			switch v := opts.Body.(type) {
			case string:
				fetchOptions.Set("body", v)
			default:
				bodyJSON, err := json.Marshal(v)
				if err != nil {
					setState(FetchState{Error: "Error encoding request body: " + err.Error(), Loading: false})
					return
				}
				fetchOptions.Set("body", string(bodyJSON))
			}
		}

		fetchPromise := js.Global().Call("fetch", url, fetchOptions)
		fetchPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			response := args[0]
			if !response.Get("ok").Bool() {
				errorMsg := fmt.Sprintf("HTTP error! status: %s", response.Get("status").String())
				fmt.Println("useFetch:", errorMsg)
				setState(FetchState{Error: errorMsg, Loading: false})
				return nil
			}
			response.Call("json").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				data := args[0]
				jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
				var parsedData interface{}
				err := json.Unmarshal([]byte(jsonStr), &parsedData)
				if err != nil {
					fmt.Println("Error parsing data:", err)
					setState(FetchState{Error: err.Error(), Loading: false})
				} else {
					fmt.Println("useFetch: Successfully fetched data")
					setState(FetchState{Data: parsedData, Loading: false})
				}
				return nil
			}))
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			err := args[0]
			errorMsg := fmt.Sprintf("Fetch error: %s", err.Get("message").String())
			fmt.Println(errorMsg)
			setState(FetchState{Error: errorMsg, Loading: false})
			return nil
		}))
	}

	GoUseEffect(func() {
		fetchData()
	}, []interface{}{url})

	return getState, fetchData
}

// GoFetch performs an asynchronous fetch operation and returns a channel for the result.
// This is a utility function for imperative fetching and requires manual state management.
func GoFetch(url string, options FetchOptions) <-chan FetchResult {
	resultChan := make(chan FetchResult, 1) // Buffered channel to avoid goroutine leak

	go func() {
		defer close(resultChan)

		fetchOptions := js.Global().Get("Object").New()
		setFetchOptions(fetchOptions, options)

		promiseResultChan := make(chan FetchResult, 1)
		performFetch(url, fetchOptions, promiseResultChan)

		result := <-promiseResultChan
		resultChan <- result
	}()

	return resultChan
}

func setFetchOptions(fetchOptions js.Value, options FetchOptions) {
	if options.Method != "" {
		fetchOptions.Set("method", options.Method)
	}

	if len(options.Headers) > 0 {
		headers := js.Global().Get("Object").New()
		for key, value := range options.Headers {
			headers.Set(key, value)
		}
		fetchOptions.Set("headers", headers)
	}

	if options.Body != nil {
		switch v := options.Body.(type) {
		case string:
			fetchOptions.Set("body", v)
		default:
			bodyJSON, err := json.Marshal(v)
			if err != nil {
				fetchOptions.Set("body", fmt.Sprintf("error encoding body: %v", err))
			} else {
				fetchOptions.Set("body", string(bodyJSON))
			}
		}
	}
}

func performFetch(url string, fetchOptions js.Value, resultChan chan<- FetchResult) {
	promise := js.Global().Call("fetch", url, fetchOptions)
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		if !response.Get("ok").Bool() {
			resultChan <- FetchResult{Err: fmt.Errorf("HTTP error! status: %s", response.Get("status").String())}
			return nil
		}

		response.Call("json").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			data := args[0]
			jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
			var parsedData interface{}
			err := json.Unmarshal([]byte(jsonStr), &parsedData)
			if err != nil {
				resultChan <- FetchResult{Err: fmt.Errorf("error parsing response: %w", err)}
			} else {
				resultChan <- FetchResult{Data: parsedData}
			}
			return nil
		}))
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err := args[0]
		resultChan <- FetchResult{Err: fmt.Errorf("fetch error: %s", err.Get("message").String())}
		return nil
	}))
}

// getFunctionName extracts a readable name from a function interface.
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
