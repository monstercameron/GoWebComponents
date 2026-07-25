package runtime

import "reflect"

// Element represents a virtual DOM node.
type Element struct {
	Type     any
	Props    map[string]any
	Children []any
	// Key carries the reconciliation key for elements built through the typed
	// fast lane (html.Props). Map-built elements keep their key in Props; the
	// key helpers consult this field first.
	Key                string
	TextContent        string // Optimization for TEXT_ELEMENT to avoid map allocation
	getHostProps       map[string]any
	getHostAttrs       []HostAttr
	isCompactHostProps bool
	hasDirectText      bool
}

// HostAttr stores one normalized string attribute for one compact host mount.
type HostAttr struct {
	Name  string
	Value string
}

// HostMountSpec stores one simple host mount description for one batched DOM append path.
type HostMountSpec struct {
	Tag   string
	Attrs []HostAttr
	Text  string
}

// PortalElementType marks a subtree whose committed DOM children should render
// into a separate target container.
type PortalElementType struct{}

// PortalNodeType marks subtrees that should commit into a separate target container.
var PortalNodeType = &PortalElementType{}

// ReactiveTextElementType marks a text node that can update from an explicit
// atom-backed subscription without rerendering the owning component.
type ReactiveTextElementType struct{}

// ReactiveTextNodeType marks text-like fine-grained update regions.
var ReactiveTextNodeType = &ReactiveTextElementType{}

// ReactiveRegionElementType marks a subscribed fine-grained region that can
// rerender its own child subtree without rerendering the owning component.
type ReactiveRegionElementType struct{}

// ReactiveRegionNodeType marks anchored fine-grained update regions.
var ReactiveRegionNodeType = &ReactiveRegionElementType{}

// AsyncBoundaryElementType marks a subtree that can recover render-time
// suspensions by rendering fallback content until the suspended work resolves.
type AsyncBoundaryElementType struct{}

// AsyncBoundaryNodeType marks async fallback boundaries.
var AsyncBoundaryNodeType = &AsyncBoundaryElementType{}

// Effect represents a side effect to be run after render.
type Effect struct {
	Fn           func() func()
	CleanupIndex int
	// Layout marks an effect that must run synchronously after DOM mutation but
	// before the browser paints, and before this fiber's passive effects (G36 /
	// UseLayoutEffect). Passive effects (UseEffect) leave it false.
	Layout bool
}

// Fiber represents a unit of work in the virtual DOM tree.
type Fiber struct {
	// Tree structure - grouped for traversal locality (Cache Line 0)
	parent    *Fiber
	child     *Fiber
	sibling   *Fiber
	alternate *Fiber

	// Flags - grouped with tree pointers for fast traversal checks (Cache Line 0)
	dirty               bool
	subtreeDirty        bool
	needsUpdate         bool
	portalUnresolved    bool // portal target selector did not resolve at commit; retry on the next commit
	needsChildReconcile bool
	needsChildOrder     bool
	// renderPhaseUpdate is set when a hook setter is called while this fiber is
	// rendering (a render-phase update). renderFunctionComponent re-runs the
	// component to converge on the new state instead of committing an output that
	// does not match the latest state.
	renderPhaseUpdate bool

	// Component info
	hooks              *Hooks
	props              map[string]any
	children           []any
	getHostAttrs       []HostAttr
	contextValues      map[int64]any
	hydration          *hydrationBoundary
	childHydration     *hydrationBoundary
	hydrated           bool
	hasDirectText      bool
	isCompactHostProps bool

	// Interfaces and Strings (16 bytes each)
	typeOf any
	dom    DOMNode
	// key mirrors Element.Key for fast-lane elements; map-built fibers keep
	// their key in props and the key helpers consult this field first.
	key         string
	textContent string
	effectTag   effectTagKind

	// Slices (24 bytes each)
	effects        []Effect
	eventCallbacks []EventHandler

	// Counters
	renderDurationNs  int64
	diffDurationNs    int64
	commitDurationNs  int64
	effectDurationNs  int64
	cleanupDurationNs int64
	boundaryError     error
	boundaryPhase     string
	asyncSuspension   *Suspension
	asyncWait         <-chan struct{}
	reactiveAtomID    string
	reactiveSourceIDs []string
	fineGrained       bool
	updateOrigin      string
	// updateLane records which priority lane marked this fiber dirty (v5 P2.2),
	// so a pass can tell whether the work belongs to it or should be deferred.
	// Zero means "no lane recorded" and always renders, keeping legacy callers
	// working unchanged.
	updateLane   UpdateLane
	ownerRuntime *Runtime
}

type hydrationBoundary struct {
	parent   DOMNode
	cursor   DOMNode
	active   bool
	fallback bool
}

// memoizedValue stores a memoized computation result with its dependencies
type memoizedValue struct {
	value any
	deps  []any
}

// HotReloadMemoSnapshot stores a memoized computation and its dependencies.
type HotReloadMemoSnapshot struct {
	Value any   `json:"value"`
	Deps  []any `json:"deps,omitempty"`
}

// callbackValue stores a memoized callback function with its dependencies
type callbackValue struct {
	fn   any
	deps []any
}

type atomAccessorValue struct {
	getter any
	setter any
}

// fetchValue stores fetch state and URL for a UseFetch hook call
type fetchValue struct {
	state FetchState
	url   string
	// fiber stores the fiber that owns this fetch, updated on every render
	fiber *Fiber
}

// funcHandlerValue stores a wrapped event handler function
type funcHandlerValue struct {
	fn      any // The user's function (func(), func(string), func(js.Value), etc.)
	wrapper any // The wrapped js.Func (or equivalent)
	cell    *funcHandlerCell
}

type funcHandlerCell struct {
	owner  *Fiber
	fn     any
	fnVal  reflect.Value // cached reflect.Value of fn; updated whenever fn changes
	fnType reflect.Type  // cached reflect.Type of fn; updated whenever fn changes
}

// RefValue represents a reference object that persists across renders.
type RefValue struct {
	Current any
}

// FetchState represents the state of a fetch operation.
type FetchState struct {
	Data    any    // The fetched data
	Error   string // Error message if fetch failed
	Loading bool   // Whether currently fetching
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
	effectEpoch   int

	states           []any // Interleaved: state, pending, state, pending...
	stateAccessors   []stateAccessor
	deps             [][]any
	memos            []memoizedValue
	callbacks        []callbackValue
	refs             []*RefValue        // Store refs separately to persist across renders
	ids              []string           // Store generated IDs that persist across renders
	fetches          []fetchValue       // Store fetch states for manual fetch hooks
	funcs            []funcHandlerValue // Store wrapped event handler functions
	cleanups         []func()           // Cleanup functions from UseEffect
	effectEpochs     []int
	atoms            []string // Store subscribed atom IDs for efficient cleanup
	atomFuncs        []atomAccessorValue
	signature        []string
	hotReloadRestore *HotReloadComponentSnapshot
	effectHadCleanup []bool
	effectSeen       []bool

	// Cached render-trace identity (see recordComponentRenderTraceLocked).
	// Hooks survive exactly the updates that keep a component at the same
	// tree position, so kind/name/path stay valid for the store's lifetime.
	traceKind string
	traceName string
	tracePath string
}

// Attrs is a convenience type for component props.
type Attrs map[string]any

// effectTagKind classifies the commit-phase work a fiber needs.  A one-byte
// enum instead of a string: the tag is compared in the hottest commit branches
// and stored on every fiber, where the string cost 16 bytes plus a string
// comparison per check.
type effectTagKind uint8

const (
	effectTagNone effectTagKind = iota
	effectTagPlacement
	effectTagUpdate
	effectTagDeletion
	effectTagHydrate
)

func (parseTag effectTagKind) String() string {
	switch parseTag {
	case effectTagPlacement:
		return "PLACEMENT"
	case effectTagUpdate:
		return "UPDATE"
	case effectTagDeletion:
		return "DELETION"
	case effectTagHydrate:
		return "HYDRATE"
	default:
		return ""
	}
}
