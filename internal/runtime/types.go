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
	getHostAttrs       []HostAttr
	componentProps     *componentPropsBox
	isCompactHostProps bool
	// hasCompactSpecialProps means getHostAttrs carries every ordinary string
	// attribute while Props contains only event/property values plus children.
	hasCompactSpecialProps bool
	hasDirectText          bool
	hasFragmentChildren    bool
	fragmentHintValid      bool
	hasComponentProps      bool
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
	// Tree structure, traversal flags, typeOf, and hooks fit in Cache Line 0
	// (bytes 0..63) so sibling/ancestor walks touch one cache line per fiber.
	parent    *Fiber
	child     *Fiber
	sibling   *Fiber
	alternate *Fiber

	dirty               bool
	subtreeDirty        bool
	needsUpdate         bool
	portalUnresolved    bool // portal target selector did not resolve at commit; retry on the next commit
	needsChildReconcile bool
	needsChildOrder     bool
	// subtreeCommit reports that at least one descendant has DOM commit work.
	subtreeCommit bool
	// renderPhaseUpdate is set when a hook setter is called while this fiber is
	// rendering (a render-phase update). renderFunctionComponent re-runs the
	// component to converge on the new state instead of committing an output that
	// does not match the latest state.
	renderPhaseUpdate bool

	typeOf any
	hooks  *Hooks

	// Pointer-bearing component and host state (grouped before the scalar tail
	// so Go's GC pointer bitmap ends before the trailing scalar block).
	dom            DOMNode
	props          map[string]any
	children       []any
	getHostAttrs   []HostAttr
	contextValues  map[int64]any
	hydration      *hydrationBoundary
	childHydration *hydrationBoundary
	// key mirrors Element.Key for fast-lane elements; map-built fibers keep
	// their key in props and the key helpers consult this field first.
	key         string
	textContent string
	effects     []Effect
	// Cold profiling, boundary, event-callback, and fine-grained subscription
	// state kept out-of-line so ordinary host and text fibers stay compact.
	cold         *fiberCold
	updateOrigin string
	ownerRuntime *Runtime

	// Pointer-free scalar tail (14 bytes padded to 16; outside GC ptrdata).
	// slotGap counts the absent (nil) children that sat directly before this
	// fiber in its parent's child list. Absent children build no fiber, so
	// without it a fiber's position in the chain stops matching its position in
	// the element list and toggling an optional child shifts every sibling after
	// it onto the wrong old fiber. Zero is the common case.
	slotGap   uint16
	effectTag effectTagKind
	// updateLane records which priority lane marked this fiber dirty (v5 P2.2),
	// so a pass can tell whether the work belongs to it or should be deferred.
	// Zero means "no lane recorded" and always renders, keeping legacy callers
	// working unchanged.
	updateLane             UpdateLane
	hydrated               bool
	hasDirectText          bool
	isCompactHostProps     bool
	hasCompactSpecialProps bool
	hasFragmentChildren    bool
	fragmentHintValid      bool
	// underFineGrained caches whether an ancestor is a fine-grained region.
	// ancestorFlagsValid distinguishes a runtime-built false value from a
	// synthetic/test fiber whose ancestry must still be inspected.
	underFineGrained   bool
	ancestorFlagsValid bool
	// serializedUnbound marks a host/text node that already exists inside a
	// parsed serialized subtree but whose DOM handle has not been materialized
	// in Go yet. The parent binds one direct child level before reconciling it,
	// avoiding two Go/JS bridge crossings for every descendant on pure mount.
	serializedUnbound bool
	fineGrained       bool
}

// fiberCold holds profiling, boundary, event-callback, and fine-grained
// subscription fields that are empty on most host and text fibers. Moving them
// behind a lazily allocated pointer trims Fiber by 144 bytes (over two cache
// lines) and shrinks mount slabs by ~34%.
type fiberCold struct {
	eventCallbacks    []EventHandler
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
}

// ensureCold returns the fiber's cold storage block, allocating it lazily on first write.
func (f *Fiber) ensureCold() *fiberCold {
	if f.cold == nil {
		f.cold = new(fiberCold)
	}
	return f.cold
}

// getEventCallbacks returns the fiber's registered event callbacks, if any.
func (f *Fiber) getEventCallbacks() []EventHandler {
	if f == nil || f.cold == nil {
		return nil
	}
	return f.cold.eventCallbacks
}

// getReactiveSourceIDs returns the fiber's subscribed fine-grained atom IDs, if any.
func (f *Fiber) getReactiveSourceIDs() []string {
	if f == nil || f.cold == nil {
		return nil
	}
	return f.cold.reactiveSourceIDs
}

// getReactiveAtomID returns the fiber's joined reactive atom ID string, if any.
func (f *Fiber) getReactiveAtomID() string {
	if f == nil || f.cold == nil {
		return ""
	}
	return f.cold.reactiveAtomID
}

// getBoundaryError returns the fiber's captured error-boundary error, if any.
func (f *Fiber) getBoundaryError() error {
	if f == nil || f.cold == nil {
		return nil
	}
	return f.cold.boundaryError
}

// getBoundaryPhase returns the fiber's captured error-boundary phase, if any.
func (f *Fiber) getBoundaryPhase() string {
	if f == nil || f.cold == nil {
		return ""
	}
	return f.cold.boundaryPhase
}

// getAsyncSuspension returns the fiber's active async suspension, if any.
func (f *Fiber) getAsyncSuspension() *Suspension {
	if f == nil || f.cold == nil {
		return nil
	}
	return f.cold.asyncSuspension
}

// getAsyncWait returns the fiber's active async suspension wait channel, if any.
func (f *Fiber) getAsyncWait() <-chan struct{} {
	if f == nil || f.cold == nil {
		return nil
	}
	return f.cold.asyncWait
}

// getRenderDurationNs returns the recorded render duration in nanoseconds.
func (f *Fiber) getRenderDurationNs() int64 {
	if f == nil || f.cold == nil {
		return 0
	}
	return f.cold.renderDurationNs
}

// getDiffDurationNs returns the recorded diff duration in nanoseconds.
func (f *Fiber) getDiffDurationNs() int64 {
	if f == nil || f.cold == nil {
		return 0
	}
	return f.cold.diffDurationNs
}

// getCommitDurationNs returns the recorded commit duration in nanoseconds.
func (f *Fiber) getCommitDurationNs() int64 {
	if f == nil || f.cold == nil {
		return 0
	}
	return f.cold.commitDurationNs
}

// getEffectDurationNs returns the recorded effect duration in nanoseconds.
func (f *Fiber) getEffectDurationNs() int64 {
	if f == nil || f.cold == nil {
		return 0
	}
	return f.cold.effectDurationNs
}

// getCleanupDurationNs returns the recorded cleanup duration in nanoseconds.
func (f *Fiber) getCleanupDurationNs() int64 {
	if f == nil || f.cold == nil {
		return 0
	}
	return f.cold.cleanupDurationNs
}

// setRenderDurationNs records the fiber's render duration without allocating cold state for zero values.
func (f *Fiber) setRenderDurationNs(v int64) {
	if v == 0 && f.cold == nil {
		return
	}
	f.ensureCold().renderDurationNs = v
}

// setDiffDurationNs records the fiber's diff duration without allocating cold state for zero values.
func (f *Fiber) setDiffDurationNs(v int64) {
	if v == 0 && f.cold == nil {
		return
	}
	f.ensureCold().diffDurationNs = v
}

// addCommitDurationNs accumulates commit duration without allocating cold state for zero values.
func (f *Fiber) addCommitDurationNs(v int64) {
	if v == 0 {
		return
	}
	f.ensureCold().commitDurationNs += v
}

// setEffectDurationNs sets effect duration without allocating cold state for zero values.
func (f *Fiber) setEffectDurationNs(v int64) {
	if v == 0 && f.cold == nil {
		return
	}
	f.ensureCold().effectDurationNs = v
}

// addEffectDurationNs accumulates effect duration without allocating cold state for zero values.
func (f *Fiber) addEffectDurationNs(v int64) {
	if v == 0 {
		return
	}
	f.ensureCold().effectDurationNs += v
}

// addCleanupDurationNs accumulates cleanup duration without allocating cold state for zero values.
func (f *Fiber) addCleanupDurationNs(v int64) {
	if v == 0 {
		return
	}
	f.ensureCold().cleanupDurationNs += v
}

// resetRenderAndDiffDurationNs zeroes render and diff timings when cold state is present.
func (f *Fiber) resetRenderAndDiffDurationNs() {
	if f.cold != nil {
		f.cold.renderDurationNs = 0
		f.cold.diffDurationNs = 0
	}
}

// componentPropsBox lets typed component elements keep their props without a
// map while occupying the Element word previously used by the now-redundant
// host-props alias. Fibers retain the element itself as their typed component
// descriptor, so ordinary fibers do not grow for this optimization.
type componentPropsBox struct {
	value any
}

type hydrationBoundary struct {
	parent   DOMNode
	cursor   DOMNode
	active   bool
	fallback bool
}

// memoizedValue stores a memoized computation result with its dependencies
type memoizedValue struct {
	value        any
	deps         []any
	singleDep    any
	hasSingleDep bool
}

type singleEffectDependency struct {
	value any
	valid bool
}

// memoizedDependencies returns the generic dependency view used by cold
// inspection and hot-reload snapshot paths.
func memoizedDependencies(parseMemo memoizedValue) []any {
	if parseMemo.deps != nil {
		return parseMemo.deps
	}
	if parseMemo.hasSingleDep {
		return []any{parseMemo.singleDep}
	}
	return nil
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
	fiber *Fiber //nolint:unused // read by hooks_fetch.go, which is js/wasm-only.
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
	effectSingleDeps []singleEffectDependency
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
	order            *hookOrderTracker // dev-only previous-render hook sequence
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
