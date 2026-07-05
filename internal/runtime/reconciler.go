package runtime

import (
	"reflect"
	"strings"
	"sync"
)

type elementScratchPool struct {
	pool sync.Pool
}

type fiberScratchPool struct {
	pool sync.Pool
}

// get is an internal reconciler helper.
func (parseP *elementScratchPool) get() []any {
	return (*parseP.pool.Get().(*[]any))[:0]
}

// clear is an internal reconciler helper.
func (parseP *elementScratchPool) clear(parseElements []any) {
	clear(parseElements)
	parseP.pool.Put(&parseElements)
}

// get is an internal reconciler helper.
func (parseP *fiberScratchPool) get() []*Fiber {
	return (*parseP.pool.Get().(*[]*Fiber))[:0]
}

// clear is an internal reconciler helper.
func (parseP *fiberScratchPool) clear(parseFibers []*Fiber) {
	clear(parseFibers)
	parseP.pool.Put(&parseFibers)
}

// currentFiber tracks the fiber being processed (for hooks)
var (
	currentFiber                 *Fiber
	currentFiberOwnerGoroutineID uint64
	emptyChildren                = []any{}
	propMetaCache                = map[string]domPropMeta{
		"children":        {kind: propKindSkip},
		"style":           {kind: propKindStyle, attrName: "style"},
		"className":       {kind: propKindClass, attrName: "class"},
		"class":           {kind: propKindClass, attrName: "class"},
		"htmlFor":         {kind: propKindDefault, attrName: "for"},
		"value":           {kind: propKindSpecialProperty, attrName: "value", resetValue: "", shouldReset: true},
		"checked":         {kind: propKindSpecialProperty, attrName: "checked", resetValue: false, shouldReset: true},
		"selected":        {kind: propKindSpecialProperty, attrName: "selected", resetValue: false, shouldReset: true},
		"disabled":        {kind: propKindDefault, attrName: "disabled", resetValue: false, shouldReset: true},
		"required":        {kind: propKindDefault, attrName: "required", resetValue: false, shouldReset: true},
		"readOnly":        {kind: propKindDefault, attrName: "readOnly", resetValue: false, shouldReset: true},
		"hidden":          {kind: propKindDefault, attrName: "hidden", resetValue: false, shouldReset: true},
		"multiple":        {kind: propKindDefault, attrName: "multiple", resetValue: false, shouldReset: true},
		"autofocus":       {kind: propKindDefault, attrName: "autofocus", resetValue: false, shouldReset: true},
		"onclick":         {kind: propKindDefault, attrName: "onclick", resetValue: nil, shouldReset: true},
		"oninput":         {kind: propKindDefault, attrName: "oninput", resetValue: nil, shouldReset: true},
		"onchange":        {kind: propKindDefault, attrName: "onchange", resetValue: nil, shouldReset: true},
		"onsubmit":        {kind: propKindDefault, attrName: "onsubmit", resetValue: nil, shouldReset: true},
		"onkeydown":       {kind: propKindDefault, attrName: "onkeydown", resetValue: nil, shouldReset: true},
		"onkeyup":         {kind: propKindDefault, attrName: "onkeyup", resetValue: nil, shouldReset: true},
		"onmouseup":       {kind: propKindDefault, attrName: "onmouseup", resetValue: nil, shouldReset: true},
		"onmousedown":     {kind: propKindDefault, attrName: "onmousedown", resetValue: nil, shouldReset: true},
		"onmouseenter":    {kind: propKindDefault, attrName: "onmouseenter", resetValue: nil, shouldReset: true},
		"onmouseleave":    {kind: propKindDefault, attrName: "onmouseleave", resetValue: nil, shouldReset: true},
		"ondblclick":      {kind: propKindDefault, attrName: "ondblclick", resetValue: nil, shouldReset: true},
		"oncontextmenu":   {kind: propKindDefault, attrName: "oncontextmenu", resetValue: nil, shouldReset: true},
		"onwheel":         {kind: propKindDefault, attrName: "onwheel", resetValue: nil, shouldReset: true},
		"ontransitionend": {kind: propKindDefault, attrName: "ontransitionend", resetValue: nil, shouldReset: true},
		"onanimationend":  {kind: propKindDefault, attrName: "onanimationend", resetValue: nil, shouldReset: true},
		"onload":          {kind: propKindDefault, attrName: "onload", resetValue: nil, shouldReset: true},
		"onerror":         {kind: propKindDefault, attrName: "onerror", resetValue: nil, shouldReset: true},
		"onpointerdown":   {kind: propKindDefault, attrName: "onpointerdown", resetValue: nil, shouldReset: true},
		"onpointermove":   {kind: propKindDefault, attrName: "onpointermove", resetValue: nil, shouldReset: true},
		"onpointerup":     {kind: propKindDefault, attrName: "onpointerup", resetValue: nil, shouldReset: true},
		"ontouchstart":    {kind: propKindDefault, attrName: "ontouchstart", resetValue: nil, shouldReset: true},
		"ontouchmove":     {kind: propKindDefault, attrName: "ontouchmove", resetValue: nil, shouldReset: true},
		"ontouchend":      {kind: propKindDefault, attrName: "ontouchend", resetValue: nil, shouldReset: true},
		"ondragstart":     {kind: propKindDefault, attrName: "ondragstart", resetValue: nil, shouldReset: true},
		"ondragover":      {kind: propKindDefault, attrName: "ondragover", resetValue: nil, shouldReset: true},
		"ondrop":          {kind: propKindDefault, attrName: "ondrop", resetValue: nil, shouldReset: true},
		"ondragend":       {kind: propKindDefault, attrName: "ondragend", resetValue: nil, shouldReset: true},
		"onfocus":         {kind: propKindDefault, attrName: "onfocus", resetValue: nil, shouldReset: true},
		"onblur":          {kind: propKindDefault, attrName: "onblur", resetValue: nil, shouldReset: true},
		"onscroll":        {kind: propKindDefault, attrName: "onscroll", resetValue: nil, shouldReset: true},
	}
	slicePool = elementScratchPool{
		pool: sync.Pool{
			New: func() any {
				// Initial capacity 16 seems reasonable for children
				parseSlice := make([]any, 0, 16)
				return &parseSlice
			},
		},
	}
	keyedFiberMapPool = sync.Pool{
		New: func() any {
			return make(map[any]*Fiber, 16)
		},
	}
	getFiberIndexMapPool = sync.Pool{
		New: func() any {
			return make(map[*Fiber]int, 16)
		},
	}
	fiberScratchSlicePool = fiberScratchPool{
		pool: sync.Pool{
			New: func() any {
				parseSlice := make([]*Fiber, 0, 16)
				return &parseSlice
			},
		},
	}
)

type domPropKind uint8

const (
	propKindDefault domPropKind = iota
	propKindSkip
	propKindStyle
	propKindClass
	propKindSpecialProperty
)

type domPropMeta struct {
	kind        domPropKind
	attrName    string
	resetValue  any
	shouldReset bool
}

// getPropMeta is an internal reconciler helper.
func getPropMeta(parseName string) domPropMeta {
	if after, ok := strings.CutPrefix(parseName, "__gwc_prop__:"); ok {
		return domPropMeta{
			kind:        propKindSpecialProperty,
			attrName:    after,
			resetValue:  nil,
			shouldReset: true,
		}
	}
	if parseMeta, parseOk := propMetaCache[parseName]; parseOk {
		return parseMeta
	}
	return domPropMeta{kind: propKindDefault, attrName: parseName}
}

// acquireWorkInProgress is an internal reconciler helper.
func acquireWorkInProgress(parseOldFiber *Fiber) *Fiber {
	if parseOldFiber != nil && parseOldFiber.alternate != nil {
		parseReused := parseOldFiber.alternate
		*parseReused = Fiber{}
		return parseReused
	}
	return new(Fiber)
}

// ensureFineGrainedTwinLink is an internal reconciler helper.
//
// The back-link is set unconditionally (not just for fine-grained fibers):
// acquireWorkInProgress can only reuse the previous generation's fiber when
// parseOldFiber.alternate points at it, so an unconditional link is what makes
// double-buffering actually work — without it every child fiber was freshly
// allocated on every render.
func ensureFineGrainedTwinLink(parseOldFiber *Fiber, parseNewFiber *Fiber) {
	if parseOldFiber == nil || parseNewFiber == nil {
		return
	}
	parseOldFiber.alternate = parseNewFiber
}

// buildFiberNeedsWork separates self updates from child-only reconciliation work on one reused fiber.
func (parseRt *Runtime) buildFiberNeedsWork(parseOldFiber *Fiber, parseElem *Element) (bool, bool) {
	if parseOldFiber == nil || parseElem == nil {
		return true, false
	}

	if parseRt.isFiberDirty(parseOldFiber) || parseOldFiber.needsUpdate {
		return true, false
	}
	if parseOldFiber.needsChildReconcile {
		return false, true
	}
	parseElemProps := getElementFiberProps(parseElem)
	parseElemChildren := getElementChildren(parseElem)

	if parseT, parseOk := parseElem.Type.(string); parseOk && parseT == "TEXT_ELEMENT" {
		parseOldText := parseOldFiber.textContent
		if parseOldText == "" && parseOldFiber.props != nil {
			parseOldText, _ = parseOldFiber.props["nodeValue"].(string)
		}
		parseNewText := parseElem.TextContent
		if parseNewText == "" && parseElem.Props != nil {
			parseNewText, _ = parseElem.Props["nodeValue"].(string)
		}
		return parseOldText != parseNewText, false
	}

	// Typed fast lane: both sides carry deterministic compact attribute
	// slices and no props map, so the diff is a positional slice walk plus
	// the dedicated key field — no map iteration at all.
	if isFastLaneCompactFiber(parseOldFiber) && parseElem.isCompactHostProps && parseElemProps == nil {
		if parseOldFiber.hasDirectText != parseElem.hasDirectText {
			return true, true
		}
		if parseOldFiber.hasDirectText && parseOldFiber.textContent != parseElem.TextContent {
			return true, false
		}
		if parseOldFiber.key != parseElem.Key || !hostAttrsEqual(parseOldFiber.getHostAttrs, parseElem.getHostAttrs) {
			return true, false
		}
		if parseOldFiber.hasDirectText {
			return false, false
		}
		if childrenEqual(getFiberChildren(parseOldFiber), parseElemChildren) {
			return false, false
		}
		return false, true
	}

	// Mixed shapes (one side fast-lane, one side map-built) compare through
	// materialized map views so shape transitions diff correctly.
	parseOldView := fiberPropsView(parseOldFiber)
	parseNewView := parseElemProps
	if parseNewView == nil && parseElem.isCompactHostProps {
		parseNewView = fastLanePropsView(parseElem.getHostAttrs, parseElem.Key, parseElem.Children, parseElem.hasDirectText)
	}

	if parseOldFiber.hasDirectText || parseElem.hasDirectText {
		if parseOldFiber.hasDirectText != parseElem.hasDirectText {
			return true, true
		}
		if parseOldFiber.textContent != parseElem.TextContent {
			return true, false
		}
		if propsEqualIgnoringChildren(parseOldView, parseNewView) {
			return false, false
		}
		return true, false
	}

	if propsEqual(parseOldView, parseNewView) {
		if childrenEqual(getFiberChildren(parseOldFiber), parseElemChildren) {
			return false, false
		}
		return false, true
	}
	if propsEqualIgnoringChildren(parseOldView, parseNewView) {
		return false, true
	}

	return true, false
}

// childrenEqual compares one structural children slice using the same pointer-oriented semantics as propsEqual.
func childrenEqual(parseA []any, parseB []any) bool {
	if len(parseA) == 0 && len(parseB) == 0 {
		return true
	}
	if len(parseA) != len(parseB) {
		return false
	}
	if len(parseA) == 0 {
		return true
	}
	return &parseA[0] == &parseB[0]
}

// buildUpdatedFiber clones one reused fiber with the current element payload and update flags.
func (parseRt *Runtime) buildUpdatedFiber(parseWipFiber *Fiber, parseOldFiber *Fiber, parseElem *Element) *Fiber {
	if parseWipFiber == nil || parseOldFiber == nil || parseElem == nil {
		return nil
	}

	isParseNeedsUpdate, isParseNeedsChildReconcile := parseRt.buildFiberNeedsWork(parseOldFiber, parseElem)
	parseEffectTag := buildUpdateEffectTag(parseElem.Type, isParseNeedsUpdate)
	parseElemProps := getElementFiberProps(parseElem)
	parseNewFiber := acquireWorkInProgress(parseOldFiber)
	*parseNewFiber = Fiber{
		typeOf:              parseOldFiber.typeOf,
		props:               parseElemProps,
		children:            getElementChildren(parseElem),
		getHostAttrs:        parseElem.getHostAttrs,
		key:                 parseElem.Key,
		textContent:         parseElem.TextContent,
		dom:                 parseOldFiber.dom,
		parent:              parseWipFiber,
		alternate:           parseOldFiber,
		effectTag:           parseEffectTag,
		dirty:               isParseNeedsUpdate,
		subtreeDirty:        parseOldFiber.subtreeDirty,
		needsUpdate:         parseOldFiber.needsUpdate,
		needsChildReconcile: isParseNeedsChildReconcile,
		hooks:               parseOldFiber.hooks,
		eventCallbacks:      parseOldFiber.eventCallbacks,
		hydration:           parseWipFiber.childHydration,
		reactiveAtomID:      parseOldFiber.reactiveAtomID,
		reactiveSourceIDs:   parseOldFiber.reactiveSourceIDs,
		fineGrained:         parseOldFiber.fineGrained,
		hasDirectText:       parseElem.hasDirectText,
		isCompactHostProps:  parseElem.isCompactHostProps,
		updateOrigin:        parseOldFiber.updateOrigin,
		ownerRuntime:        parseRt,
	}
	ensureFineGrainedTwinLink(parseOldFiber, parseNewFiber)
	parseRt.handleClonedFiberSubscriptionMove(parseOldFiber, parseNewFiber)
	return parseNewFiber
}

// NOTE(experiment, 2026-07-04): a "relink clean fast-lane fibers instead of
// cloning" fast path was tried here (adopt the committed child fiber object
// directly when key/attrs/text/children are unchanged). It passed the full
// native suite and the runtime1 browser subject, but deterministically broke
// the runtime2 worker content flow (worker state arrived, re-render never
// happened — a stale fiber-reference interaction in the update targeting
// path), and its measured upside was flat because the fiber pool already
// makes clones allocation-free. Rejected; children of a dirty parent are
// always cloned.

// buildPlacementFiber builds one new placement fiber for an inserted or replaced element.
func buildPlacementFiber(parseWipFiber *Fiber, parseElem *Element, parseOldFiber *Fiber) *Fiber {
	if parseWipFiber == nil || parseElem == nil {
		return nil
	}

	parseElemProps := getElementFiberProps(parseElem)
	parseNewFiber := acquireWorkInProgress(nil)
	*parseNewFiber = Fiber{
		typeOf:             parseElem.Type,
		props:              parseElemProps,
		children:           getElementChildren(parseElem),
		getHostAttrs:       parseElem.getHostAttrs,
		key:                parseElem.Key,
		textContent:        parseElem.TextContent,
		parent:             parseWipFiber,
		effectTag:          effectTagPlacement,
		dirty:              true,
		hydration:          parseWipFiber.childHydration,
		fineGrained:        isFineGrainedType(parseElem.Type),
		hasDirectText:      parseElem.hasDirectText,
		isCompactHostProps: parseElem.isCompactHostProps,
		updateOrigin:       oldFiberUpdateOrigin(parseOldFiber, parseElem.Type),
		ownerRuntime:       runtimeForFiber(parseWipFiber),
	}
	return parseNewFiber
}

// buildUpdateEffectTag returns the effect tag used when one reused fiber keeps its existing DOM node.
func buildUpdateEffectTag(parseTypeOf any, isParseNeedsUpdate bool) effectTagKind {
	if !isParseNeedsUpdate {
		return effectTagNone
	}
	if parseT, parseOk := parseTypeOf.(string); parseOk && parseT == "FRAGMENT" {
		return effectTagNone
	}
	return effectTagUpdate
}

// hasElementFiberKeyMatch reports whether one element and one existing fiber carry the same keyed identity in the same position.
func hasElementFiberKeyMatch(parseElem *Element, parseFiber *Fiber) bool {
	if parseElem == nil || parseFiber == nil {
		return false
	}
	isParseElemKeyed := hasElementKey(parseElem)
	isParseFiberKeyed := hasFiberKey(parseFiber)
	if isParseElemKeyed != isParseFiberKeyed {
		return false
	}
	if !isParseElemKeyed {
		return true
	}
	// Typed fast-lane keys compare as plain strings; boxing them into `any`
	// for fastEqual allocated on every keyed row of every render.
	if parseElem.Key != "" && parseFiber.key != "" {
		return parseElem.Key == parseFiber.key
	}
	return fastEqual(elementKeyValue(parseElem), fiberKeyValue(parseFiber))
}

// elementKeyValue returns one element's reconciliation key, preferring the
// typed fast-lane field over the props map.
func elementKeyValue(parseElem *Element) any {
	if parseElem == nil {
		return nil
	}
	if parseElem.Key != "" {
		return parseElem.Key
	}
	if parseElem.Props == nil {
		return nil
	}
	return parseElem.Props["key"]
}

// fiberKeyValue returns one fiber's reconciliation key, preferring the typed
// fast-lane field over the props map.
func fiberKeyValue(parseFiber *Fiber) any {
	if parseFiber == nil {
		return nil
	}
	if parseFiber.key != "" {
		return parseFiber.key
	}
	if parseFiber.props == nil {
		return nil
	}
	return parseFiber.props["key"]
}

// tryReconcileKeyedChildrenInOrder fast-paths keyed lists that kept the same sibling order.
func (parseRt *Runtime) tryReconcileKeyedChildrenInOrder(parseWipFiber *Fiber, parseElements []any, parseOldFirst *Fiber) bool {
	if parseWipFiber == nil {
		return false
	}

	// Phase 1: validate the whole run is same-order/same-identity before any
	// fiber is built or relinked, so an abort can never leave half-mutated
	// chain links or moved atom subscriptions behind. Elements remaining
	// after the old chain is exhausted are trailing appends — the dominant
	// list-growth shape — and mount as placements in phase 2, so appends
	// never fall back to the key-boxing map path.
	parseOldFiber := parseOldFirst
	for _, parseElement := range parseElements {
		parseElem, parseOk := parseElement.(*Element)
		if !parseOk || parseElem == nil {
			return false
		}
		if parseOldFiber == nil {
			continue // trailing append candidate; only castability matters
		}
		if !sameFiberType(parseElem, parseOldFiber) || !hasElementFiberKeyMatch(parseElem, parseOldFiber) {
			return false
		}
		parseOldFiber = parseOldFiber.sibling
	}
	if parseOldFiber != nil {
		return false
	}

	// Phase 2: build the chain — updates over the matched prefix, placements
	// for the appended tail.
	parseOldFiber = parseOldFirst
	var parseFirstChild *Fiber
	var parsePrevSibling *Fiber
	for _, parseElement := range parseElements {
		parseElem := parseElement.(*Element)
		var parseNewFiber *Fiber
		if parseOldFiber != nil {
			parseNextOldFiber := parseOldFiber.sibling
			parseNewFiber = parseRt.buildUpdatedFiber(parseWipFiber, parseOldFiber, parseElem)
			parseOldFiber = parseNextOldFiber
		} else {
			parseNewFiber = buildPlacementFiber(parseWipFiber, parseElem, nil)
		}
		if parseFirstChild == nil {
			parseFirstChild = parseNewFiber
		} else if parsePrevSibling != nil {
			parsePrevSibling.sibling = parseNewFiber
		}
		parsePrevSibling = parseNewFiber
	}
	parseWipFiber.child = parseFirstChild
	return true
}

// GetCurrentFiber returns the fiber currently being processed
func (parseRt *Runtime) reconcileChildren(parseWipFiber *Fiber, parseElements []any) {
	parseWipFiber.needsChildOrder = false
	// Fast path: empty elements
	if len(parseElements) == 0 {
		if parseWipFiber.alternate != nil && parseWipFiber.alternate.child != nil {
			// Delete all old children
			parseOldFiber := parseWipFiber.alternate.child
			for parseOldFiber != nil {
				parseOldFiber.effectTag = effectTagDeletion
				parseRt.deletions = append(parseRt.deletions, parseOldFiber)
				parseOldFiber = parseOldFiber.sibling
			}
		}
		return
	}

	// Flatten any Fragment elements before reconciliation
	parseFlatElements, parseWasAllocated := flattenFragments(parseElements)
	if parseWasAllocated {
		defer func() {
			slicePool.clear(parseFlatElements)
		}()
	}
	parseElements = parseFlatElements
	reportMissingKeys(parseWipFiber, parseElements)

	if shouldUseKeyedReconciliation(parseElements, parseWipFiber) {
		parseRt.reconcileKeyedChildren(parseWipFiber, parseElements)
		return
	}

	parseIndex := 0
	var parseOldFiber2 *Fiber
	if parseWipFiber.alternate != nil {
		parseOldFiber2 = parseWipFiber.alternate.child
	}
	var parsePrevSibling *Fiber
	isParseFirstChildSet := false

	// Loop 1: Update/Replace (Both exist)
	// Pre-compute element count for better branch prediction
	parseElemCount := len(parseElements)
	for parseIndex < parseElemCount && parseOldFiber2 != nil {
		parseElement := parseElements[parseIndex]

		var parseNewFiber *Fiber

		if parseElement != nil {
			if parseElem, parseOk := parseElement.(*Element); parseOk && parseElem != nil {
				// Inline fast path for string type comparison (most common case)
				isParseSameType := false
				if parseS1, parseOk1 := parseElem.Type.(string); parseOk1 {
					if parseS2, parseOk2 := parseOldFiber2.typeOf.(string); parseOk2 {
						isParseSameType = parseS1 == parseS2
					}
				} else {
					isParseSameType = isSameType(parseElem.Type, parseOldFiber2.typeOf)
				}

				if isParseSameType {
					parseNewFiber = parseRt.buildUpdatedFiber(parseWipFiber, parseOldFiber2, parseElem)
					// Advance oldFiber
					parseOldFiber2 = parseOldFiber2.sibling
				} else {
					// REPLACE logic (Placement + Deletion)
					if parseOldFiber2.sibling != nil || parseIndex < parseElemCount-1 {
						parseWipFiber.needsChildOrder = true
					}
					parseNewFiber = buildPlacementFiber(parseWipFiber, parseElem, parseOldFiber2)

					// Mark old fiber for deletion
					parseOldFiber2.effectTag = effectTagDeletion
					parseRt.deletions = append(parseRt.deletions, parseOldFiber2)
					parseOldFiber2 = parseOldFiber2.sibling
				}
			}
		} else {
			parseOldFiber2.effectTag = effectTagDeletion
			parseRt.deletions = append(parseRt.deletions, parseOldFiber2)
			parseOldFiber2 = parseOldFiber2.sibling
		}

		// Link to parent
		if parseNewFiber != nil {
			if !isParseFirstChildSet {
				parseWipFiber.child = parseNewFiber
				isParseFirstChildSet = true
			} else if parsePrevSibling != nil {
				parsePrevSibling.sibling = parseNewFiber
			}
			parsePrevSibling = parseNewFiber
		}

		parseIndex++
	}

	// Loop 2: Placement (Remaining elements)
	for parseIndex < len(parseElements) {
		parseElement2 := parseElements[parseIndex]
		var parseNewFiber2 *Fiber

		if parseElement2 != nil {
			if parseElem2, parseOk4 := parseElement2.(*Element); parseOk4 && parseElem2 != nil {
				parseNewFiber2 = buildPlacementFiber(parseWipFiber, parseElem2, nil)
			}
		}

		if parseNewFiber2 != nil {
			if !isParseFirstChildSet {
				parseWipFiber.child = parseNewFiber2
				isParseFirstChildSet = true
			} else if parsePrevSibling != nil {
				parsePrevSibling.sibling = parseNewFiber2
			}
			parsePrevSibling = parseNewFiber2
		}
		parseIndex++
	}

	// Loop 3: Deletion (Remaining old fibers)
	for parseOldFiber2 != nil {
		parseOldFiber2.effectTag = effectTagDeletion
		parseRt.deletions = append(parseRt.deletions, parseOldFiber2)
		parseOldFiber2 = parseOldFiber2.sibling
	}
}

// shouldUseKeyedReconciliation is an internal reconciler helper.
func shouldUseKeyedReconciliation(parseElements []any, parseWipFiber *Fiber) bool {
	for _, parseElement := range parseElements {
		parseElem, parseOk := parseElement.(*Element)
		if !parseOk || parseElem == nil {
			continue
		}
		if hasElementKey(parseElem) {
			return true
		}
	}

	if parseWipFiber == nil || parseWipFiber.alternate == nil {
		return false
	}
	for parseOldFiber := parseWipFiber.alternate.child; parseOldFiber != nil; parseOldFiber = parseOldFiber.sibling {
		if hasFiberKey(parseOldFiber) {
			return true
		}
	}

	return false
}

// reconcileKeyedChildren is an internal reconciler helper.
func (parseRt *Runtime) reconcileKeyedChildren(parseWipFiber *Fiber, parseElements []any) {
	parseWipFiber.needsChildOrder = false
	parseOldByKey := keyedFiberMapPool.Get().(map[any]*Fiber)
	parseOldFallbackKeyed := fiberScratchSlicePool.get()
	parseOldUnkeyed := fiberScratchSlicePool.get()
	parseOldIndexByFiber := getFiberIndexMapPool.Get().(map[*Fiber]int)
	defer func() {
		clear(parseOldByKey)
		keyedFiberMapPool.Put(parseOldByKey)
		clear(parseOldIndexByFiber)
		getFiberIndexMapPool.Put(parseOldIndexByFiber)
		fiberScratchSlicePool.clear(parseOldFallbackKeyed)
		fiberScratchSlicePool.clear(parseOldUnkeyed)
	}()

	var parseOldFirst *Fiber
	if parseWipFiber.alternate != nil {
		parseOldFirst = parseWipFiber.alternate.child
	}
	if parseRt.tryReconcileKeyedChildrenInOrder(parseWipFiber, parseElements, parseOldFirst) {
		return
	}

	parseOldIndex := 0
	for parseOldFiber := parseOldFirst; parseOldFiber != nil; parseOldFiber = parseOldFiber.sibling {
		parseOldIndexByFiber[parseOldFiber] = parseOldIndex
		parseOldIndex++
		if parseKey, isComparable, hasKey := fiberReconcileKey(parseOldFiber); isComparable {
			if _, parseDup := parseOldByKey[parseKey]; parseDup {
				// Duplicate key: the keyed map holds only one fiber per key, so
				// route the collision to the positionally-matched fallback list.
				// This keeps every old fiber tracked in exactly one structure so
				// none leaks at cleanup — an overwritten map entry would otherwise
				// never be tagged for deletion, orphaning a stale DOM node.
				parseOldFallbackKeyed = append(parseOldFallbackKeyed, parseOldFiber)
			} else {
				parseOldByKey[parseKey] = parseOldFiber
			}
		} else if hasKey {
			parseOldFallbackKeyed = append(parseOldFallbackKeyed, parseOldFiber)
		} else {
			parseOldUnkeyed = append(parseOldUnkeyed, parseOldFiber)
		}
	}

	parseUnkeyedIndex := 0
	parseLastMatchedOldIndex := -1
	var parsePrevSibling *Fiber
	isParseFirstChildSet := false

	for _, parseElement := range parseElements {
		parseElem, parseOk2 := parseElement.(*Element)
		if !parseOk2 || parseElem == nil {
			continue
		}

		var parseMatchedOld *Fiber
		if parseKey2, isComparable2, hasKey2 := elementReconcileKey(parseElem); isComparable2 {
			parseMatchedOld = parseOldByKey[parseKey2]
			if parseMatchedOld != nil {
				delete(parseOldByKey, parseKey2)
			} else {
				// No primary keyed match: a duplicate-keyed old fiber routed to
				// the fallback list may still match this element by key.
				parseMatchedOld = takeMatchingFallbackKeyed(parseOldFallbackKeyed, parseElem)
			}
		} else if hasKey2 {
			parseMatchedOld = takeMatchingFallbackKeyed(parseOldFallbackKeyed, parseElem)
		} else if parseUnkeyedIndex < len(parseOldUnkeyed) {
			parseMatchedOld = parseOldUnkeyed[parseUnkeyedIndex]
			parseUnkeyedIndex++
		}

		var parseNewFiber *Fiber
		if parseMatchedOld != nil && sameFiberType(parseElem, parseMatchedOld) {
			if parseMatchedOldIndex, hasParseMatchedOldIndex := parseOldIndexByFiber[parseMatchedOld]; hasParseMatchedOldIndex {
				if parseMatchedOldIndex < parseLastMatchedOldIndex {
					parseWipFiber.needsChildOrder = true
				}
				if parseMatchedOldIndex > parseLastMatchedOldIndex {
					parseLastMatchedOldIndex = parseMatchedOldIndex
				}
			}

			parseNewFiber = parseRt.buildUpdatedFiber(parseWipFiber, parseMatchedOld, parseElem)
		} else {
			if len(parseOldByKey) > 0 || hasRemainingFallbackKeyed(parseOldFallbackKeyed) || parseUnkeyedIndex < len(parseOldUnkeyed) {
				parseWipFiber.needsChildOrder = true
			}
			if parseMatchedOld != nil {
				parseMatchedOld.effectTag = effectTagDeletion
				parseRt.deletions = append(parseRt.deletions, parseMatchedOld)
			}

			parseNewFiber = buildPlacementFiber(parseWipFiber, parseElem, nil)
		}

		if !isParseFirstChildSet {
			parseWipFiber.child = parseNewFiber
			isParseFirstChildSet = true
		} else if parsePrevSibling != nil {
			parsePrevSibling.sibling = parseNewFiber
		}
		parsePrevSibling = parseNewFiber
	}

	for parseOldFiber2 := parseOldFirst; parseOldFiber2 != nil; parseOldFiber2 = parseOldFiber2.sibling {
		parseKey, parseOk := fiberComparableKey(parseOldFiber2)
		if !parseOk {
			continue
		}
		if parseOldByKey[parseKey] != parseOldFiber2 {
			continue
		}
		delete(parseOldByKey, parseKey)
		parseOldFiber2.effectTag = effectTagDeletion
		parseRt.deletions = append(parseRt.deletions, parseOldFiber2)
	}
	for _, parseOldFiber3 := range parseOldFallbackKeyed {
		if parseOldFiber3 == nil {
			continue
		}
		parseOldFiber3.effectTag = effectTagDeletion
		parseRt.deletions = append(parseRt.deletions, parseOldFiber3)
	}
	for ; parseUnkeyedIndex < len(parseOldUnkeyed); parseUnkeyedIndex++ {
		parseOldFiber4 := parseOldUnkeyed[parseUnkeyedIndex]
		parseOldFiber4.effectTag = effectTagDeletion
		parseRt.deletions = append(parseRt.deletions, parseOldFiber4)
	}
}

// hasRemainingFallbackKeyed reports whether one keyed fallback slice still carries unmatched entries.
func hasRemainingFallbackKeyed(parseOldFibers []*Fiber) bool {
	for _, parseOldFiber := range parseOldFibers {
		if parseOldFiber != nil {
			return true
		}
	}
	return false
}

// sameFiberType is an internal reconciler helper.
func sameFiberType(parseElem *Element, parseOldFiber *Fiber) bool {
	if parseElem == nil || parseOldFiber == nil {
		return false
	}

	if parseS1, parseOk1 := parseElem.Type.(string); parseOk1 {
		if parseS2, parseOk2 := parseOldFiber.typeOf.(string); parseOk2 {
			return parseS1 == parseS2
		}
	}

	return isSameType(parseElem.Type, parseOldFiber.typeOf)
}

// hasElementKey is an internal reconciler helper.
func hasElementKey(parseElem *Element) bool {
	if parseElem == nil {
		return false
	}
	if parseElem.Key != "" {
		return true
	}
	if parseElem.Props == nil {
		return false
	}
	_, parseOk := parseElem.Props["key"]
	return parseOk
}

// hasFiberKey is an internal reconciler helper.
func hasFiberKey(parseFiber *Fiber) bool {
	if parseFiber == nil {
		return false
	}
	if parseFiber.key != "" {
		return true
	}
	if parseFiber.props == nil {
		return false
	}
	_, parseOk := parseFiber.props["key"]
	return parseOk
}

// elementComparableKey is an internal reconciler helper.
func elementComparableKey(parseElem *Element) (any, bool) {
	if parseElem == nil {
		return nil, false
	}
	if parseElem.Key != "" {
		return parseElem.Key, true
	}
	if parseElem.Props == nil {
		return nil, false
	}
	return propsComparableKey(parseElem.Props)
}

// fiberComparableKey is an internal reconciler helper.
func fiberComparableKey(parseFiber *Fiber) (any, bool) {
	if parseFiber == nil {
		return nil, false
	}
	if parseFiber.key != "" {
		return parseFiber.key, true
	}
	if parseFiber.props == nil {
		return nil, false
	}
	return propsComparableKey(parseFiber.props)
}

// propsComparableKey is an internal reconciler helper.
func propsComparableKey(parseProps map[string]any) (any, bool) {
	parseKey, parseOk := parseProps["key"]
	if !parseOk || parseKey == nil {
		return nil, false
	}
	return comparableKeyValue(parseKey)
}

// comparableKeyValue reports whether one key value can serve as a map key.
func comparableKeyValue(parseKey any) (any, bool) {
	switch parseTyped := parseKey.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, bool:
		return parseTyped, true
	case *Element, *Fiber:
		return parseTyped, true
	}
	parseT := reflect.TypeOf(parseKey)
	if parseT == nil || !parseT.Comparable() {
		return nil, false
	}
	return parseKey, true
}

// fiberReconcileKey classifies one fiber's key in a single probe: the
// map-usable key value (when comparable) plus whether any key exists at all,
// so keyed reconciliation avoids a second props lookup on the fallback path.
func fiberReconcileKey(parseFiber *Fiber) (any, bool, bool) {
	if parseFiber == nil {
		return nil, false, false
	}
	if parseFiber.key != "" {
		return parseFiber.key, true, true
	}
	if parseFiber.props == nil {
		return nil, false, false
	}
	parseKey, hasKey := parseFiber.props["key"]
	if !hasKey {
		return nil, false, false
	}
	if parseKey == nil {
		return nil, false, true
	}
	parseValue, isComparable := comparableKeyValue(parseKey)
	return parseValue, isComparable, true
}

// elementReconcileKey is the element-side twin of fiberReconcileKey.
func elementReconcileKey(parseElem *Element) (any, bool, bool) {
	if parseElem == nil {
		return nil, false, false
	}
	if parseElem.Key != "" {
		return parseElem.Key, true, true
	}
	if parseElem.Props == nil {
		return nil, false, false
	}
	parseKey, hasKey := parseElem.Props["key"]
	if !hasKey {
		return nil, false, false
	}
	if parseKey == nil {
		return nil, false, true
	}
	parseValue, isComparable := comparableKeyValue(parseKey)
	return parseValue, isComparable, true
}

// takeMatchingFallbackKeyed is an internal reconciler helper.
func takeMatchingFallbackKeyed(parseOldFibers []*Fiber, parseElem *Element) *Fiber {
	if parseElem == nil || (parseElem.Key == "" && parseElem.Props == nil) {
		return nil
	}
	parseKey := elementKeyValue(parseElem)
	for parseIndex, parseOldFiber := range parseOldFibers {
		if parseOldFiber == nil || (parseOldFiber.key == "" && parseOldFiber.props == nil) {
			continue
		}
		if fastEqual(fiberKeyValue(parseOldFiber), parseKey) {
			parseOldFibers[parseIndex] = nil
			return parseOldFiber
		}
	}
	return nil
}

// propsEqual compares two property maps for equality
func propsEqual(parseA, parseB map[string]any) bool {
	if len(parseA) == 0 && len(parseB) == 0 {
		return true
	}

	// Fast path: different lengths
	parseALen := len(parseA)
	if parseALen != len(parseB) {
		return false
	}

	for parseK, parseV1 := range parseA {
		parseV2, parseOk := parseB[parseK]
		if !parseOk {
			return false
		}

		if parseK == "children" {
			// Avoid deep comparison for children
			// Check if they are the same slice reference
			// If not, assume they are different to avoid O(Subtree) traversal
			if parseV1 == nil && parseV2 == nil {
				continue
			}
			if parseV1 == nil || parseV2 == nil {
				return false
			}

			// Fast path for the common children representation.
			if parseC1, parseOk1 := parseV1.([]any); parseOk1 {
				if parseC2, parseOk2 := parseV2.([]any); parseOk2 {
					if len(parseC1) == len(parseC2) {
						if len(parseC1) == 0 {
							continue
						}
						if &parseC1[0] == &parseC2[0] {
							continue
						}
					}
					return false
				}
			}

			// Fallback for non-standard slice types.
			parseT1 := reflect.TypeOf(parseV1)
			parseT2 := reflect.TypeOf(parseV2)
			if parseT1 != nil && parseT2 != nil && parseT1.Kind() == reflect.Slice && parseT2.Kind() == reflect.Slice {
				parseRv1 := reflect.ValueOf(parseV1)
				parseRv2 := reflect.ValueOf(parseV2)
				if parseRv1.Pointer() == parseRv2.Pointer() && parseRv1.Len() == parseRv2.Len() {
					continue
				}
			}

			// If pointers differ, assume different.
			// This skips DeepEqual.
			return false
		}

		if !fastEqual(parseV1, parseV2) {
			return false
		}
	}

	return true
}

// propsEqualIgnoringChildren compares two property maps while ignoring child slices.
func propsEqualIgnoringChildren(parseA, parseB map[string]any) bool {
	if len(parseA) == 0 && len(parseB) == 0 {
		return true
	}

	parseALen := len(parseA)
	parseBLen := len(parseB)
	if _, hasParseChildrenA := parseA["children"]; hasParseChildrenA {
		parseALen--
	}
	if _, hasParseChildrenB := parseB["children"]; hasParseChildrenB {
		parseBLen--
	}
	if parseALen != parseBLen {
		return false
	}

	// One pass suffices: the children-adjusted lengths already matched, and
	// every non-children key of A was found in B, so B cannot hold an extra
	// non-children key.
	for parseK, parseV1 := range parseA {
		if parseK == "children" {
			continue
		}
		parseV2, parseOk := parseB[parseK]
		if !parseOk {
			return false
		}
		if !fastEqual(parseV1, parseV2) {
			return false
		}
	}

	return true
}

// isSameType checks if two component types are the same
func isSameType(parseType1, parseType2 any) bool {
	// String types (HTML tags)
	if parseS1, parseOk1 := parseType1.(string); parseOk1 {
		if parseS2, parseOk2 := parseType2.(string); parseOk2 {
			return parseS1 == parseS2
		}
		return false
	}

	if parseComponent1, parseOk12 := parseType1.(*ComponentType); parseOk12 {
		parseComponent2, parseOk22 := parseType2.(*ComponentType)
		if !parseOk22 {
			return false
		}
		return parseComponent1.IdentityKey() != "" && parseComponent1.IdentityKey() == parseComponent2.IdentityKey()
	}

	parseV1 := reflect.ValueOf(parseType1)
	parseV2 := reflect.ValueOf(parseType2)

	if !parseV1.IsValid() || !parseV2.IsValid() {
		return false
	}

	if parseV1.Kind() == reflect.Func && parseV2.Kind() == reflect.Func {
		return sameFunctionIdentity(parseType1, parseType2)
	}

	if parseV1.Type().Comparable() {
		return parseType1 == parseType2
	}

	return reflect.DeepEqual(parseType1, parseType2)
}

// isFiberDirty checks if a fiber or any of its alternates are dirty
func (parseRt *Runtime) isFiberDirty(parseFiber *Fiber) bool {
	if parseFiber == nil {
		return false
	}

	if parseFiber.dirty {
		return true
	}

	// Child fibers are reused as current/work-in-progress pairs.
	// Avoid walking an alternate cycle indefinitely.
	parseAlternate := parseFiber.alternate
	return parseAlternate != nil && parseAlternate != parseFiber && parseAlternate.dirty
}

// clearFiberDirty clears update-consumed flags on a fiber and its alternate pair.
func (parseRt *Runtime) clearFiberDirty(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	parseFiber.dirty = false
	parseFiber.subtreeDirty = false
	parseFiber.needsUpdate = false
	parseFiber.needsChildReconcile = false

	parseAlternate := parseFiber.alternate
	if parseAlternate != nil && parseAlternate != parseFiber {
		parseAlternate.dirty = false
		parseAlternate.subtreeDirty = false
		parseAlternate.needsUpdate = false
		parseAlternate.needsChildReconcile = false
	}
}

// performUnitOfWork processes a single fiber
func (parseRt *Runtime) performUnitOfWork(parseFiber *Fiber) *Fiber {
	if parseFiber == nil {
		return nil
	}

	// Distinguish self work from descendant-only work so clean owners can forward updates without rerendering.
	isParseSelfDirty := parseRt.isFiberDirty(parseFiber) || parseFiber.needsChildReconcile
	isParseSubtreeOnly := !isParseSelfDirty && parseFiber.subtreeDirty

	// Reuse the committed child chain when neither the fiber nor any
	// descendant needs work. The bailout path skips diff timing entirely —
	// no render or reconcile ran, and the pair of clock reads was the
	// dominant cost of visiting a clean fiber.
	if !isParseSelfDirty && !isParseSubtreeOnly {
		parseFiber.renderDurationNs = 0
		parseFiber.diffDurationNs = 0
		if parseFiber.hooks != nil {
			parseFiber.hooks.owner = parseFiber
		}
		parseRt.reuseFiberChildSubtree(parseFiber)
		return parseRt.getNextSiblingUnitOfWork(parseFiber)
	}

	parseStart := commitTimingStart()
	parseFiber.renderDurationNs = 0
	parseFiber.diffDurationNs = 0
	parseFinalize := func(parseNext *Fiber) *Fiber {
		parseDiffDurationNs := max(commitTimingSinceNs(parseStart)-parseFiber.renderDurationNs, 0)
		parseFiber.diffDurationNs = parseDiffDurationNs
		parseRt.profiling.totalDiffDurationNs += parseDiffDurationNs
		return parseNext
	}

	// Clear dirty flags on fiber and alternates
	parseRt.clearFiberDirty(parseFiber)

	if isParseSubtreeOnly {
		if parseFiber.hooks != nil {
			parseFiber.hooks.owner = parseFiber
		}
		parseRt.cloneChildFibers(parseFiber)
		return parseFinalize(parseRt.getNextUnitOfWork(parseFiber))
	}

	if parseFiber.contextValues == nil && parseFiber.parent != nil {
		parseFiber.contextValues = parseFiber.parent.contextValues
	}
	if parseFiber.hydration == nil && parseFiber.parent != nil {
		parseFiber.hydration = parseFiber.parent.childHydration
	}

	if parseFiber.typeOf == nil || parseFiber.typeOf == "ROOT" {
		// Root fiber - reconcile children
		parseFiber.childHydration = parseFiber.hydration
		if parseChildren := getFiberChildren(parseFiber); parseChildren != nil {
			parseRt.reconcileChildren(parseFiber, parseChildren)
		}
	} else {
		switch parseTyped := parseFiber.typeOf.(type) {
		case string:
			// Host component (HTML element)
			if parseTyped == "FRAGMENT" {
				parseFiber.childHydration = parseFiber.hydration
			} else if IsDOMNodeNull(parseFiber.dom) {
				if parseHydratedDOM, parseOk2 := parseRt.claimHydrationNode(parseFiber); parseOk2 {
					parseFiber.dom = parseHydratedDOM
					parseFiber.hydrated = true
					parseFiber.effectTag = effectTagHydrate
					parseFiber.childHydration = newHydrationBoundary(parseHydratedDOM, parseRt.domAdapter.GetFirstChild(parseHydratedDOM))
				} else if parseRt.shouldDeferHostDomToCommit(parseFiber) {
					// Serialized-mount candidate: leave dom null so commit can
					// mount the whole subtree from one parsed HTML string; the
					// commit placement branch creates per-node DOM if the
					// subtree turns out ineligible.
					parseFiber.hydrated = false
					parseFiber.childHydration = nil
				} else {
					parseFiber.dom = parseRt.createDom(parseFiber)
					parseFiber.hydrated = false
					parseFiber.childHydration = nil
				}
			} else if parseTyped != "TEXT_ELEMENT" {
				parseFiber.childHydration = nil
			}

			if parseTyped == "TEXT_ELEMENT" {
				parseFiber.childHydration = nil
				parseRt.reconcileChildren(parseFiber, emptyChildren)
				break
			}
			if parseFiber.hasDirectText {
				parseFiber.childHydration = nil
				parseRt.reconcileChildren(parseFiber, emptyChildren)
				break
			}
			if parseChildren := getFiberChildren(parseFiber); parseChildren != nil {
				parseRt.reconcileChildren(parseFiber, parseChildren)
				break
			}
			parseRt.reconcileChildren(parseFiber, emptyChildren)

		case *ContextProviderType:
			parseValue := parseTyped.Descriptor.DefaultValue
			if parseFiber.props != nil {
				if parseProvided, parseOk4 := parseFiber.props["value"]; parseOk4 {
					parseValue = parseProvided
				}
			}

			var parseParentContextValues map[int64]any
			if parseFiber.parent != nil {
				parseParentContextValues = parseFiber.parent.contextValues
			}
			parseFiber.contextValues = deriveContextValues(parseParentContextValues, parseTyped.Descriptor.ID, parseValue)
			parseFiber.childHydration = parseFiber.hydration

			if parseFiber.alternate != nil && !fastEqual(resolveContextValue(parseFiber.alternate, parseTyped.Descriptor), parseValue) {
				markSubtreeNeedsUpdate(parseFiber.alternate.child, "context")
			}

			if parseChildren := getFiberChildren(parseFiber); parseChildren != nil {
				parseRt.reconcileChildren(parseFiber, parseChildren)
				break
			}
			parseRt.reconcileChildren(parseFiber, emptyChildren)

		case *PortalElementType:
			if parseChildren := getFiberChildren(parseFiber); parseChildren != nil {
				parseRt.reconcileChildren(parseFiber, parseChildren)
				break
			}
			parseRt.reconcileChildren(parseFiber, emptyChildren)

		case *ReactiveTextElementType:
			parseRt.syncReactiveTextSubscription(parseFiber)
			parseFiber.textContent = reactiveTextValue(parseFiber)
			if IsDOMNodeNull(parseFiber.dom) {
				if parseHydratedDOM2, parseOk7 := parseRt.claimHydrationNode(parseFiber); parseOk7 {
					parseFiber.dom = parseHydratedDOM2
					parseFiber.hydrated = true
					parseFiber.effectTag = effectTagHydrate
				} else {
					parseFiber.dom = parseRt.createDom(parseFiber)
					parseFiber.hydrated = false
				}
			}
			parseFiber.childHydration = nil

		case *ReactiveRegionElementType:
			parseRt.syncReactiveRegionSubscription(parseFiber)
			parseFiber.childHydration = parseFiber.hydration
			parseRendered := reactiveRegionValue(parseFiber)
			if parseRendered != nil {
				parseChildren2 := [1]any{parseRendered}
				parseRt.reconcileChildren(parseFiber, parseChildren2[:])
			} else {
				parseRt.reconcileChildren(parseFiber, emptyChildren)
			}

		case *ErrorBoundaryType:
			parseRt.renderBoundaryChildren(parseFiber)

		case *AsyncBoundaryElementType:
			parseRt.renderAsyncBoundaryChildren(parseFiber)

		default:
			parseFiber.childHydration = parseFiber.hydration
			parseElement, parseHandledPanic, parseNextFromBoundary := parseRt.renderFunctionComponent(parseFiber)
			if parseHandledPanic {
				return parseFinalize(parseNextFromBoundary)
			}
			if parseElement != nil {
				parseChildren3 := [1]any{parseElement}
				parseRt.reconcileChildren(parseFiber, parseChildren3[:])
			} else {
				// A component that now renders nothing must delete any previous subtree.
				parseRt.reconcileChildren(parseFiber, emptyChildren)
			}
		}
	}

	return parseFinalize(parseRt.getNextUnitOfWork(parseFiber))
}

// getNextUnitOfWork determines the next fiber to process
func (parseRt *Runtime) getNextUnitOfWork(parseFiber *Fiber) *Fiber {
	// Child first
	if parseFiber.child != nil {
		return parseFiber.child
	}

	// Then sibling
	parseNextFiber := parseFiber
	for parseNextFiber != nil {
		if parseNextFiber.sibling != nil {
			return parseNextFiber.sibling
		}
		parseNextFiber = parseNextFiber.parent
	}

	return nil
}

// getNextSiblingUnitOfWork determines the next unit of work without descending into the current fiber's children.
func (parseRt *Runtime) getNextSiblingUnitOfWork(parseFiber *Fiber) *Fiber {
	for parseNextFiber := parseFiber; parseNextFiber != nil; parseNextFiber = parseNextFiber.parent {
		if parseNextFiber.sibling != nil {
			return parseNextFiber.sibling
		}
	}
	return nil
}

// createDom creates a DOM node from a fiber
