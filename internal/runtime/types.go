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

	// Component unique ID counter for useId
	componentIdCounter int

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
	HookTypeCallback
	HookTypeRef
	HookTypeFunc
	HookTypeAtom
	HookTypeId
	HookTypeFetch
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

// callbackValue stores a memoized callback function with its dependencies
type callbackValue struct {
	fn   interface{}
	deps []interface{}
}

// fetchValue stores fetch state and URL for a UseFetch hook call
type fetchValue struct {
	state FetchState
	url   string
	// channel for ongoing fetch (can be nil if not fetching)
	fetchChannel <-chan interface{}
	// fiber stores the fiber that owns this fetch, updated on every render
	fiber *Fiber
}

// funcHandlerValue stores a wrapped event handler function
type funcHandlerValue struct {
	fn      interface{} // The user's function (func(), func(string), func(js.Value), etc.)
	wrapper interface{} // The wrapped js.Func (or equivalent)
}

// RefValue represents a reference object that persists across renders
// It has a single .current property that can hold any value
type RefValue struct {
	Current interface{}
}

// FetchState represents the state of a fetch operation
type FetchState struct {
	Data    interface{} // The fetched data
	Error   string      // Error message if fetch failed
	Loading bool        // Whether currently fetching
}

// Hooks manages component hook state
type Hooks struct {
	index int

	state        []interface{} // Committed state
	pendingState []interface{} // Pending state updates (used during setState batching)
	deps         [][]interface{}
	memos        []memoizedValue
	callbacks    []callbackValue
	refs         []*RefValue        // Store refs separately to persist across renders
	ids          []string           // Store generated IDs that persist across renders
	fetches      []fetchValue       // Store fetch states for manual fetch hooks
	funcs        []funcHandlerValue // Store wrapped event handler functions
	cleanups     []func()           // Cleanup functions from UseEffect

	callOrder    []HookCall
	prevOrder    []HookCall
	orderChecked bool
}

// Attrs is a convenience type for component props
type Attrs map[string]interface{}
