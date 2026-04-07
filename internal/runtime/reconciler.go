package runtime

import (
	"reflect"
	"strings"
	"sync"
	"time"
)

type elementScratchPool struct {
	pool sync.Pool
}

type fiberScratchPool struct {
	pool sync.Pool
}

// get is an internal reconciler helper.
func (parseP *elementScratchPool) get() []interface{} {
	return (*parseP.pool.Get().(*[]interface{}))[:0]
}

// clear is an internal reconciler helper.
func (parseP *elementScratchPool) clear(parseElements []interface{}) {
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
	currentFiber  *Fiber
	emptyChildren = []interface{}{}
	propMetaCache = map[string]domPropMeta{
		"children":  {kind: propKindSkip},
		"style":     {kind: propKindStyle, attrName: "style"},
		"className": {kind: propKindClass, attrName: "class"},
		"class":     {kind: propKindClass, attrName: "class"},
		"htmlFor":   {kind: propKindDefault, attrName: "for"},
		"value":     {kind: propKindSpecialProperty, attrName: "value", resetValue: "", shouldReset: true},
		"checked":   {kind: propKindSpecialProperty, attrName: "checked", resetValue: false, shouldReset: true},
		"selected":  {kind: propKindSpecialProperty, attrName: "selected", resetValue: false, shouldReset: true},
		"disabled":  {kind: propKindDefault, attrName: "disabled", resetValue: false, shouldReset: true},
		"required":  {kind: propKindDefault, attrName: "required", resetValue: false, shouldReset: true},
		"readOnly":  {kind: propKindDefault, attrName: "readOnly", resetValue: false, shouldReset: true},
		"hidden":    {kind: propKindDefault, attrName: "hidden", resetValue: false, shouldReset: true},
		"multiple":  {kind: propKindDefault, attrName: "multiple", resetValue: false, shouldReset: true},
		"autofocus": {kind: propKindDefault, attrName: "autofocus", resetValue: false, shouldReset: true},
		"onclick":   {kind: propKindDefault, attrName: "onclick", resetValue: nil, shouldReset: true},
		"oninput":   {kind: propKindDefault, attrName: "oninput", resetValue: nil, shouldReset: true},
		"onchange":  {kind: propKindDefault, attrName: "onchange", resetValue: nil, shouldReset: true},
		"onsubmit":  {kind: propKindDefault, attrName: "onsubmit", resetValue: nil, shouldReset: true},
		"onkeydown": {kind: propKindDefault, attrName: "onkeydown", resetValue: nil, shouldReset: true},
		"onkeyup":   {kind: propKindDefault, attrName: "onkeyup", resetValue: nil, shouldReset: true},
		"onfocus":   {kind: propKindDefault, attrName: "onfocus", resetValue: nil, shouldReset: true},
		"onblur":    {kind: propKindDefault, attrName: "onblur", resetValue: nil, shouldReset: true},
	}
	slicePool = elementScratchPool{
		pool: sync.Pool{
			New: func() interface{} {
				// Initial capacity 16 seems reasonable for children
				parseSlice := make([]interface{}, 0, 16)
				return &parseSlice
			},
		},
	}
	// Fiber pool to reduce allocations
	fiberPool = sync.Pool{
		New: func() interface{} {
			return &Fiber{}
		},
	}
	keyedFiberMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[interface{}]*Fiber, 16)
		},
	}
	getFiberIndexMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[*Fiber]int, 16)
		},
	}
	fiberScratchSlicePool = fiberScratchPool{
		pool: sync.Pool{
			New: func() interface{} {
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
	resetValue  interface{}
	shouldReset bool
}

// getPropMeta is an internal reconciler helper.
func getPropMeta(parseName string) domPropMeta {
	if strings.HasPrefix(parseName, "__gwc_prop__:") {
		return domPropMeta{
			kind:        propKindSpecialProperty,
			attrName:    strings.TrimPrefix(parseName, "__gwc_prop__:"),
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
	parseReused2 := fiberPool.Get().(*Fiber)
	*parseReused2 = Fiber{}
	return parseReused2
}

// ensureFineGrainedTwinLink is an internal reconciler helper.
func ensureFineGrainedTwinLink(parseOldFiber *Fiber, parseNewFiber *Fiber) {
	if parseOldFiber == nil || parseNewFiber == nil {
		return
	}
	if parseOldFiber.fineGrained || parseNewFiber.fineGrained {
		parseOldFiber.alternate = parseNewFiber
	}
}

// buildFiberNeedsUpdate reports whether one reused fiber should keep its subtree on the update path.
func (parseRt *Runtime) buildFiberNeedsUpdate(parseOldFiber *Fiber, parseElem *Element) bool {
	if parseOldFiber == nil || parseElem == nil {
		return true
	}

	isDirty := parseRt.isFiberDirty(parseOldFiber)
	if isDirty || parseOldFiber.needsUpdate {
		return true
	}

	if parseT, parseOk := parseElem.Type.(string); parseOk && parseT == "TEXT_ELEMENT" {
		parseOldText := parseOldFiber.textContent
		if parseOldText == "" && parseOldFiber.props != nil {
			parseOldText, _ = parseOldFiber.props["nodeValue"].(string)
		}
		parseNewText := parseElem.TextContent
		if parseNewText == "" && parseElem.Props != nil {
			parseNewText, _ = parseElem.Props["nodeValue"].(string)
		}
		return parseOldText != parseNewText
	}

	return !propsEqual(parseOldFiber.props, parseElem.Props)
}

// buildUpdatedFiber clones one reused fiber with the current element payload and update flags.
func (parseRt *Runtime) buildUpdatedFiber(parseWipFiber *Fiber, parseOldFiber *Fiber, parseElem *Element) *Fiber {
	if parseWipFiber == nil || parseOldFiber == nil || parseElem == nil {
		return nil
	}

	parseEffectTag := buildUpdateEffectTag(parseElem.Type)
	parseNewFiber := acquireWorkInProgress(parseOldFiber)
	*parseNewFiber = Fiber{
		typeOf:            parseOldFiber.typeOf,
		props:             parseElem.Props,
		textContent:       parseElem.TextContent,
		dom:               parseOldFiber.dom,
		parent:            parseWipFiber,
		alternate:         parseOldFiber,
		effectTag:         parseEffectTag,
		dirty:             parseRt.buildFiberNeedsUpdate(parseOldFiber, parseElem),
		needsUpdate:       parseOldFiber.needsUpdate,
		hooks:             parseOldFiber.hooks,
		eventCallbacks:    parseOldFiber.eventCallbacks,
		hydration:         parseWipFiber.childHydration,
		reactiveAtomID:    parseOldFiber.reactiveAtomID,
		reactiveSourceIDs: parseOldFiber.reactiveSourceIDs,
		fineGrained:       parseOldFiber.fineGrained,
		updateOrigin:      parseOldFiber.updateOrigin,
	}
	ensureFineGrainedTwinLink(parseOldFiber, parseNewFiber)
	parseRt.handleClonedFiberSubscriptionMove(parseOldFiber, parseNewFiber)
	return parseNewFiber
}

// buildPlacementFiber builds one new placement fiber for an inserted or replaced element.
func buildPlacementFiber(parseWipFiber *Fiber, parseElem *Element, parseOldFiber *Fiber) *Fiber {
	if parseWipFiber == nil || parseElem == nil {
		return nil
	}

	parseNewFiber := acquireWorkInProgress(nil)
	*parseNewFiber = Fiber{
		typeOf:       parseElem.Type,
		props:        parseElem.Props,
		textContent:  parseElem.TextContent,
		parent:       parseWipFiber,
		effectTag:    "PLACEMENT",
		dirty:        true,
		hydration:    parseWipFiber.childHydration,
		fineGrained:  isFineGrainedType(parseElem.Type),
		updateOrigin: oldFiberUpdateOrigin(parseOldFiber, parseElem.Type),
	}
	return parseNewFiber
}

// buildUpdateEffectTag returns the effect tag used when one reused fiber keeps its existing DOM node.
func buildUpdateEffectTag(parseTypeOf interface{}) string {
	if parseT, parseOk := parseTypeOf.(string); parseOk && parseT == "FRAGMENT" {
		return ""
	}
	return "UPDATE"
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
	return fastEqual(parseElem.Props["key"], parseFiber.props["key"])
}

// tryReconcileKeyedChildrenInOrder fast-paths keyed lists that kept the same sibling order.
func (parseRt *Runtime) tryReconcileKeyedChildrenInOrder(parseWipFiber *Fiber, parseElements []interface{}, parseOldFirst *Fiber) bool {
	if parseWipFiber == nil {
		return false
	}

	parseOldFiber := parseOldFirst
	var parseFirstChild *Fiber
	var parsePrevSibling *Fiber

	for _, parseElement := range parseElements {
		parseElem, parseOk := parseElement.(*Element)
		if !parseOk || parseElem == nil {
			return false
		}
		if parseOldFiber == nil {
			return false
		}
		if !sameFiberType(parseElem, parseOldFiber) || !hasElementFiberKeyMatch(parseElem, parseOldFiber) {
			return false
		}

		parseNewFiber := parseRt.buildUpdatedFiber(parseWipFiber, parseOldFiber, parseElem)
		if parseFirstChild == nil {
			parseFirstChild = parseNewFiber
		} else if parsePrevSibling != nil {
			parsePrevSibling.sibling = parseNewFiber
		}
		parsePrevSibling = parseNewFiber
		parseOldFiber = parseOldFiber.sibling
	}

	if parseOldFiber != nil {
		return false
	}
	parseWipFiber.child = parseFirstChild
	return true
}

// GetCurrentFiber returns the fiber currently being processed
func (parseRt *Runtime) reconcileChildren(parseWipFiber *Fiber, parseElements []interface{}) {
	parseWipFiber.needsChildOrder = false
	// Fast path: empty elements
	if len(parseElements) == 0 {
		if parseWipFiber.alternate != nil && parseWipFiber.alternate.child != nil {
			// Delete all old children
			parseOldFiber := parseWipFiber.alternate.child
			for parseOldFiber != nil {
				parseOldFiber.effectTag = "DELETION"
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
					parseOldFiber2.effectTag = "DELETION"
					parseRt.deletions = append(parseRt.deletions, parseOldFiber2)
					parseOldFiber2 = parseOldFiber2.sibling
				}
			}
		} else {
			parseOldFiber2.effectTag = "DELETION"
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
		parseOldFiber2.effectTag = "DELETION"
		parseRt.deletions = append(parseRt.deletions, parseOldFiber2)
		parseOldFiber2 = parseOldFiber2.sibling
	}
}

// shouldUseKeyedReconciliation is an internal reconciler helper.
func shouldUseKeyedReconciliation(parseElements []interface{}, parseWipFiber *Fiber) bool {
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
func (parseRt *Runtime) reconcileKeyedChildren(parseWipFiber *Fiber, parseElements []interface{}) {
	parseWipFiber.needsChildOrder = false
	parseOldByKey := keyedFiberMapPool.Get().(map[interface{}]*Fiber)
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
		if parseKey, parseOk := fiberComparableKey(parseOldFiber); parseOk {
			parseOldByKey[parseKey] = parseOldFiber
		} else if hasFiberKey(parseOldFiber) {
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
		if parseKey2, hasKey := elementComparableKey(parseElem); hasKey {
			parseMatchedOld = parseOldByKey[parseKey2]
			if parseMatchedOld != nil {
				delete(parseOldByKey, parseKey2)
			}
		} else if hasElementKey(parseElem) {
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
				parseMatchedOld.effectTag = "DELETION"
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

	for _, parseOldFiber2 := range parseOldByKey {
		parseOldFiber2.effectTag = "DELETION"
		parseRt.deletions = append(parseRt.deletions, parseOldFiber2)
	}
	for _, parseOldFiber3 := range parseOldFallbackKeyed {
		if parseOldFiber3 == nil {
			continue
		}
		parseOldFiber3.effectTag = "DELETION"
		parseRt.deletions = append(parseRt.deletions, parseOldFiber3)
	}
	for ; parseUnkeyedIndex < len(parseOldUnkeyed); parseUnkeyedIndex++ {
		parseOldFiber4 := parseOldUnkeyed[parseUnkeyedIndex]
		parseOldFiber4.effectTag = "DELETION"
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
	if parseElem == nil || parseElem.Props == nil {
		return false
	}
	_, parseOk := parseElem.Props["key"]
	return parseOk
}

// hasFiberKey is an internal reconciler helper.
func hasFiberKey(parseFiber *Fiber) bool {
	if parseFiber == nil || parseFiber.props == nil {
		return false
	}
	_, parseOk := parseFiber.props["key"]
	return parseOk
}

// elementComparableKey is an internal reconciler helper.
func elementComparableKey(parseElem *Element) (interface{}, bool) {
	if parseElem == nil || parseElem.Props == nil {
		return nil, false
	}
	return propsComparableKey(parseElem.Props)
}

// fiberComparableKey is an internal reconciler helper.
func fiberComparableKey(parseFiber *Fiber) (interface{}, bool) {
	if parseFiber == nil || parseFiber.props == nil {
		return nil, false
	}
	return propsComparableKey(parseFiber.props)
}

// propsComparableKey is an internal reconciler helper.
func propsComparableKey(parseProps map[string]interface{}) (interface{}, bool) {
	parseKey, parseOk := parseProps["key"]
	if !parseOk || parseKey == nil {
		return nil, false
	}
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

// takeMatchingFallbackKeyed is an internal reconciler helper.
func takeMatchingFallbackKeyed(parseOldFibers []*Fiber, parseElem *Element) *Fiber {
	if parseElem == nil || parseElem.Props == nil {
		return nil
	}
	parseKey := parseElem.Props["key"]
	for parseIndex, parseOldFiber := range parseOldFibers {
		if parseOldFiber == nil || parseOldFiber.props == nil {
			continue
		}
		if fastEqual(parseOldFiber.props["key"], parseKey) {
			parseOldFibers[parseIndex] = nil
			return parseOldFiber
		}
	}
	return nil
}

// propsEqual compares two property maps for equality
func propsEqual(parseA, parseB map[string]interface{}) bool {
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
			if parseC1, parseOk1 := parseV1.([]interface{}); parseOk1 {
				if parseC2, parseOk2 := parseV2.([]interface{}); parseOk2 {
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
func propsEqualIgnoringChildren(parseA, parseB map[string]interface{}) bool {
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

	for parseK := range parseB {
		if parseK == "children" {
			continue
		}
		if _, parseOk := parseA[parseK]; !parseOk {
			return false
		}
	}

	return true
}

// isSameType checks if two component types are the same
func isSameType(parseType1, parseType2 interface{}) bool {
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
	parseFiber.needsUpdate = false

	parseAlternate := parseFiber.alternate
	if parseAlternate != nil && parseAlternate != parseFiber {
		parseAlternate.dirty = false
		parseAlternate.needsUpdate = false
	}
}

// performUnitOfWork processes a single fiber
func (parseRt *Runtime) performUnitOfWork(parseFiber *Fiber) *Fiber {
	if parseFiber == nil {
		return nil
	}
	parseStart := time.Now()
	parseFiber.renderDurationNs = 0
	parseFiber.diffDurationNs = 0
	parseFinalize := func(parseNext *Fiber) *Fiber {
		parseDiffDurationNs := time.Since(parseStart).Nanoseconds() - parseFiber.renderDurationNs
		if parseDiffDurationNs < 0 {
			parseDiffDurationNs = 0
		}
		parseFiber.diffDurationNs = parseDiffDurationNs
		parseRt.profiling.totalDiffDurationNs += parseDiffDurationNs
		return parseNext
	}

	// Check if fiber or any alternate is dirty
	isDirty := parseRt.isFiberDirty(parseFiber)

	// Skip non-dirty fibers (optimization)
	if !isDirty {
		if parseFiber.hooks != nil {
			parseFiber.hooks.owner = parseFiber
		}
		parseRt.cloneChildFibers(parseFiber)
		return parseFinalize(parseRt.getNextUnitOfWork(parseFiber))
	}

	// Clear dirty flags on fiber and alternates
	parseRt.clearFiberDirty(parseFiber)

	if parseFiber.contextValues == nil && parseFiber.parent != nil {
		parseFiber.contextValues = parseFiber.parent.contextValues
	}
	if parseFiber.hydration == nil && parseFiber.parent != nil {
		parseFiber.hydration = parseFiber.parent.childHydration
	}

	if parseFiber.typeOf == nil || parseFiber.typeOf == "ROOT" {
		// Root fiber - reconcile children
		parseFiber.childHydration = parseFiber.hydration
		if parseChildren, parseOk := parseFiber.props["children"].([]interface{}); parseOk {
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
					parseFiber.effectTag = "HYDRATE"
					parseFiber.childHydration = newHydrationBoundary(parseHydratedDOM, parseRt.domAdapter.GetFirstChild(parseHydratedDOM))
				} else {
					parseFiber.dom = parseRt.createDom(parseFiber)
					parseFiber.hydrated = false
					parseFiber.childHydration = nil
				}
			} else if parseTyped != "TEXT_ELEMENT" {
				parseFiber.childHydration = nil
			}

			if parsePropsChildren, parseOk3 := parseFiber.props["children"]; parseOk3 {
				if parseElements, parseElementsOk := parsePropsChildren.([]interface{}); parseElementsOk {
					parseRt.reconcileChildren(parseFiber, parseElements)
				}
			}

		case *ContextProviderType:
			parseValue := parseTyped.Descriptor.DefaultValue
			if parseFiber.props != nil {
				if parseProvided, parseOk4 := parseFiber.props["value"]; parseOk4 {
					parseValue = parseProvided
				}
			}

			var parseParentContextValues map[int64]interface{}
			if parseFiber.parent != nil {
				parseParentContextValues = parseFiber.parent.contextValues
			}
			parseFiber.contextValues = deriveContextValues(parseParentContextValues, parseTyped.Descriptor.ID, parseValue)
			parseFiber.childHydration = parseFiber.hydration

			if parseFiber.alternate != nil && !fastEqual(resolveContextValue(parseFiber.alternate, parseTyped.Descriptor), parseValue) {
				markSubtreeNeedsUpdate(parseFiber.alternate.child, "context")
			}

			if parsePropsChildren2, parseOk5 := parseFiber.props["children"]; parseOk5 {
				if parseElements2, parseElementsOk2 := parsePropsChildren2.([]interface{}); parseElementsOk2 {
					parseRt.reconcileChildren(parseFiber, parseElements2)
					break
				}
			}
			parseRt.reconcileChildren(parseFiber, emptyChildren)

		case *PortalElementType:
			if parsePropsChildren3, parseOk6 := parseFiber.props["children"]; parseOk6 {
				if parseElements3, parseElementsOk3 := parsePropsChildren3.([]interface{}); parseElementsOk3 {
					parseRt.reconcileChildren(parseFiber, parseElements3)
					break
				}
			}
			parseRt.reconcileChildren(parseFiber, emptyChildren)

		case *ReactiveTextElementType:
			parseRt.syncReactiveTextSubscription(parseFiber)
			parseFiber.textContent = reactiveTextValue(parseFiber)
			if IsDOMNodeNull(parseFiber.dom) {
				if parseHydratedDOM2, parseOk7 := parseRt.claimHydrationNode(parseFiber); parseOk7 {
					parseFiber.dom = parseHydratedDOM2
					parseFiber.hydrated = true
					parseFiber.effectTag = "HYDRATE"
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
				parseChildren2 := [1]interface{}{parseRendered}
				parseRt.reconcileChildren(parseFiber, parseChildren2[:])
			} else {
				parseRt.reconcileChildren(parseFiber, emptyChildren)
			}

		case *ErrorBoundaryType:
			parseRt.renderBoundaryChildren(parseFiber)

		default:
			parseFiber.childHydration = parseFiber.hydration
			parseElement, parseHandledPanic, parseNextFromBoundary := parseRt.renderFunctionComponent(parseFiber)
			if parseHandledPanic {
				return parseFinalize(parseNextFromBoundary)
			}
			if parseElement != nil {
				parseChildren3 := [1]interface{}{parseElement}
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

// createDom creates a DOM node from a fiber
