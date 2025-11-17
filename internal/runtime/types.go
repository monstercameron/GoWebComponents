package runtime

// Element represents a virtual DOM node
type Element struct {
	Type     interface{}
	Props    map[string]interface{}
	Children []interface{}
}

// Fiber represents a unit of work in the virtual DOM tree
type Fiber struct {
	// Tree structure
	parent    *Fiber
	child     *Fiber
	sibling   *Fiber
	alternate *Fiber

	// Component info
	typeOf interface{}
	props  map[string]interface{}

	// Platform-agnostic DOM reference
	dom DOMNode

	// Hooks and effects
	hooks          *Hooks
	effects        []func()
	eventCallbacks []EventHandler

	// Reconciliation metadata
	effectTag   string
	dirty       bool
	needsUpdate bool
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

// Hooks manages component hook state
type Hooks struct {
	index int

	state        []interface{} // Committed state
	pendingState []interface{} // Pending state updates (used during setState batching)
	deps         [][]interface{}
	memos        []memoizedValue

	callOrder    []HookCall
	prevOrder    []HookCall
	orderChecked bool
}

// Attrs is a convenience type for component props
type Attrs map[string]interface{}
