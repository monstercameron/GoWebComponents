package runtime

// DOM element refs (G2).
//
// A ref is carried through props under the reserved key DOMRefKey, holding a
// DOMRefSink. The key is registered as propKindSkip so it is NEVER applied to the
// DOM as an attribute/property; the commit phase instead publishes the live node
// into the sink on placement and clears it (nil) on deletion. Because commitRoot
// processes deletions before placements, a ref reassigned across a key-change
// remount detaches (nil) from the old element before attaching to the new one.

// DOMRefKey is the reserved props key that carries a DOMRefSink. It is namespaced
// like the other internal keys (e.g. the custom-element property prefix) so it can
// never collide with a real attribute name.
const DOMRefKey = "__gwc_dom_ref__"

// DOMRefSink receives the live DOM node when its element mounts and nil when the
// element unmounts. ui.UseDOMRef returns a handle implementing this; the runtime
// only ever calls SetDOMNode from the (single-threaded) commit phase.
type DOMRefSink interface {
	SetDOMNode(node DOMNode)
}

// Focuser is the optional capability a DOMNode may implement to move keyboard
// focus to itself. The wasm node implements it (element.focus()); ui.DOMRef.Focus
// and ui.UseAutoFocus route through it so they work cross-build (no-op when the
// node — or the build — does not support focusing).
type Focuser interface {
	Focus()
}

// Blurrer / Clicker / ScrollIntoViewer are the optional capabilities a DOMNode may implement for
// the remaining common imperative-handle operations, mirroring Focuser. The wasm node implements
// all three; ui.DOMRef.Blur/Click/ScrollIntoView route through them so they work cross-build
// (no-op when the node — or the build — does not support the operation).
type Blurrer interface {
	Blur()
}

type Clicker interface {
	Click()
}

type ScrollIntoViewer interface {
	ScrollIntoView(behavior string)
}

// init registers the ref key so the DOM property differ skips it entirely — the
// sink is plumbing, never markup. (Mutating propMetaCache here is safe: it runs
// during package init, before any render goroutine exists.)
func init() {
	propMetaCache[DOMRefKey] = domPropMeta{kind: propKindSkip}
}

// publishDOMRef sets (or clears, when node is nil) the ref carried by a fiber's
// props, if any. No-op for fibers without a ref or with a malformed sink.
func (parseRt *Runtime) publishDOMRef(parseFiber *Fiber, parseNode DOMNode) {
	if parseFiber == nil || parseFiber.props == nil {
		return
	}
	parseRaw, parseOk := parseFiber.props[DOMRefKey]
	if !parseOk {
		return
	}
	if parseSink, parseIsSink := parseRaw.(DOMRefSink); parseIsSink && parseSink != nil {
		parseSink.SetDOMNode(parseNode)
	}
}

// releaseDOMRefsSubtree clears every ref in a fiber subtree that is being deleted,
// so a ref held across an unmount observes nil. It mirrors
// cleanupAtomSubscriptionsSubtree's shape: the deleted element may be nested below
// DOM-less function-component fibers, so the whole subtree must be swept.
func (parseRt *Runtime) releaseDOMRefsSubtree(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}
	parseRt.publishDOMRef(parseFiber, nil)
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseRt.releaseDOMRefsSubtree(parseChild)
	}
}
