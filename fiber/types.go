//go:build js && wasm
// +build js,wasm

package fiber

import (
	"syscall/js"
)

// Element represents a virtual DOM node.
type Element struct {
	Type     interface{}
	Props    map[string]interface{}
	Children []interface{}
}

// HookType represents the type of hook being called
type HookType int

const (
	HookTypeState HookType = iota
	HookTypeEffect
	HookTypeMemo
	HookTypeFunc
	HookTypeAtom
)

// HookCall represents a single hook call for order validation
type HookCall struct {
	Type     HookType
	Position int
}

// memoizedValue stores a memoized computation result with its dependencies
type memoizedValue struct {
	value interface{}
	deps  []interface{}
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

	// Smaller types grouped at end (64 bytes)
	effectTag      string    // 16 bytes
	effects        []func()  // 24 bytes
	eventCallbacks []js.Func // 24 bytes - event callbacks for cleanup
	dirty          bool      // 1 byte
}

// Attrs type is defined in html.go

// FetchState represents the state of a fetch operation
type FetchState struct {
	Data    interface{}
	Error   string
	Loading bool
}

// FetchOptions represents options for fetch operations
type FetchOptions struct {
	Method  string
	Headers map[string]interface{}
	Body    interface{}
}

// FetchResult represents the result of a fetch operation
type FetchResult struct {
	Data interface{}
	Err  error
}

// FastComparable interface for types that can perform fast equality checking
type FastComparable interface {
	FastEqual(other interface{}) bool
}

// GoEvent wraps JavaScript events with convenient Go methods
type GoEvent struct {
	jsEvent js.Value
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

// NewGoEvent creates a GoEvent from a JavaScript event
func NewGoEvent(jsEvent js.Value) GoEvent {
	return GoEvent{jsEvent: jsEvent}
}
