package runtime

// Element represents a virtual DOM node.
type Element struct {
	Type        interface{}
	Props       map[string]interface{}
	Children    []interface{}
	TextContent string // Optimization for TEXT_ELEMENT to avoid map allocation
}

// PortalElementType marks a subtree whose committed DOM children should render
// into a separate target container.
type PortalElementType struct{}

// PortalNodeType marks subtrees that should commit into a separate target container.
var PortalNodeType = &PortalElementType{}

// Effect represents a side effect to be run after render.
type Effect struct {
	Fn           func() func()
	CleanupIndex int
}

// Fiber represents a unit of work in the virtual DOM tree.
type Fiber struct {
	// Tree structure - grouped for traversal locality (Cache Line 0)
	parent    *Fiber
	child     *Fiber
	sibling   *Fiber
	alternate *Fiber

	// Flags - grouped with tree pointers for fast traversal checks (Cache Line 0)
	dirty       bool
	needsUpdate bool
	// 6 bytes padding here to align next 8-byte field

	// Component info
	hooks          *Hooks
	props          map[string]interface{}
	contextValues  map[int64]interface{}
	hydration      *hydrationBoundary
	childHydration *hydrationBoundary
	hydrated       bool

	// Interfaces and Strings (16 bytes each)
	typeOf      interface{}
	dom         DOMNode
	textContent string
	effectTag   string

	// Slices (24 bytes each)
	effects        []Effect
	eventCallbacks []EventHandler

	// Counters
	componentIdCounter int
	commitDurationNs   int64
	effectDurationNs   int64
	cleanupDurationNs  int64
	boundaryError      error
	boundaryPhase      string
}

type hydrationBoundary struct {
	parent   DOMNode
	cursor   DOMNode
	active   bool
	fallback bool
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

type atomAccessorValue struct {
	getter interface{}
	setter interface{}
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

// RefValue represents a reference object that persists across renders.
type RefValue struct {
	Current interface{}
}

// FetchState represents the state of a fetch operation.
type FetchState struct {
	Data    interface{} // The fetched data
	Error   string      // Error message if fetch failed
	Loading bool        // Whether currently fetching
}

// Hooks manages component hook state for a fiber.
type Hooks struct {
	owner *Fiber

	index int
	// 4 bytes padding (on 32-bit) or 0 on 64-bit if int is 64-bit.
	// Actually int is 64-bit on 64-bit arch.

	// Indices for packed storage
	stateIndex    int
	depIndex      int
	memoIndex     int
	callbackIndex int
	refIndex      int
	idIndex       int
	fetchIndex    int
	funcIndex     int
	atomIndex     int
	cleanupIndex  int

	states    []interface{} // Interleaved: state, pending, state, pending...
	deps      [][]interface{}
	memos     []memoizedValue
	callbacks []callbackValue
	refs      []*RefValue        // Store refs separately to persist across renders
	ids       []string           // Store generated IDs that persist across renders
	fetches   []fetchValue       // Store fetch states for manual fetch hooks
	funcs     []funcHandlerValue // Store wrapped event handler functions
	cleanups  []func()           // Cleanup functions from UseEffect
	atoms     []string           // Store subscribed atom IDs for efficient cleanup
	atomFuncs []atomAccessorValue
}

// Attrs is a convenience type for component props.
type Attrs map[string]interface{}
