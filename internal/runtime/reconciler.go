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

// GetCurrentFiber returns the fiber currently being processed
func GetCurrentFiber() *Fiber {
	return currentFiber
}

// SetCurrentFiber sets the current fiber (used during component rendering)
func SetCurrentFiber(parseFiber *Fiber) {
	currentFiber = parseFiber
}

// IsCurrentFiberTransitionUpdate reports whether the current fiber render originated from deferred transition work.
func IsCurrentFiberTransitionUpdate() bool {
	for parseFiber := GetCurrentFiber(); parseFiber != nil; parseFiber = parseFiber.parent {
		if strings.HasPrefix(parseFiber.updateOrigin, "transition") {
			return true
		}
	}
	return false
}

// CreateElement creates a new virtual DOM element
func CreateElement(parseTyp interface{}, parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	if len(parseChildren) == 0 {
		parseChildren = emptyChildren
	}

	// Process children: wrap strings in TEXT_ELEMENT
	// We modify the children slice in-place to avoid allocation since it's a varargs slice
	for parseI, parseChild := range parseChildren {
		if parseStr, parseOk := parseChild.(string); parseOk {
			parseChildren[parseI] = &Element{
				Type:        "TEXT_ELEMENT",
				TextContent: parseStr,
				// Props:    nil, // No props map needed!
				Children: emptyChildren,
			}
		}
	}

	parsePropsLen := len(parseProps)
	parseElem := &Element{
		Type:     parseTyp,
		Props:    make(map[string]interface{}, parsePropsLen+1),
		Children: parseChildren,
	}

	if parsePropsLen > 0 {
		for parseK, parseV := range parseProps {
			parseElem.Props[parseK] = parseV
		}
	}
	parseElem.Props["children"] = parseChildren

	return parseElem
}

// flattenFragments is an internal reconciler helper.
func flattenFragments(parseElements []interface{}) ([]interface{}, bool) {
	isParseNeedsFlatten := false
	for _, parseElement := range parseElements {
		parseElem, parseOk := parseElement.(*Element)
		if !parseOk || parseElem == nil {
			continue
		}
		if parseT, parseOk2 := parseElem.Type.(string); parseOk2 && parseT == "FRAGMENT" {
			isParseNeedsFlatten = true
			break
		}
	}

	if !isParseNeedsFlatten {
		return parseElements, false
	}

	parseFlattened := slicePool.get()
	for _, parseElement2 := range parseElements {
		parseElem2, parseOk3 := parseElement2.(*Element)
		if !parseOk3 {
			if parseElement2 != nil {
				parseFlattened = append(parseFlattened, parseElement2)
			}
			continue
		}
		if parseElem2 == nil {
			continue
		}
		if parseT2, parseOk4 := parseElem2.Type.(string); parseOk4 && parseT2 == "FRAGMENT" {
			if parseChildren, parseOk5 := parseElem2.Props["children"].([]interface{}); parseOk5 {
				parseRes, parseAllocated := flattenFragments(parseChildren)
				parseFlattened = append(parseFlattened, parseRes...)
				if parseAllocated {
					slicePool.clear(parseRes)
				}
			}
			continue
		}
		parseFlattened = append(parseFlattened, parseElem2)
	}

	return parseFlattened, true
}

// cloneChildFibers clones the child fibers from the alternate to the current fiber
// This is used when skipping reconciliation for non-dirty fibers
func (parseRt *Runtime) cloneChildFibers(parseParent *Fiber) {
	if parseParent.alternate == nil || parseParent.alternate.child == nil {
		return
	}

	var parsePrevSibling *Fiber
	parseOldFiber := parseParent.alternate.child

	for parseOldFiber != nil {
		parseEffectTag := ""
		if parseOldFiber.dirty || parseOldFiber.needsUpdate {
			parseEffectTag = "UPDATE"
		}
		parseNewFiber := acquireWorkInProgress(parseOldFiber)
		*parseNewFiber = Fiber{
			typeOf:            parseOldFiber.typeOf,
			props:             parseOldFiber.props,
			textContent:       parseOldFiber.textContent,
			dom:               parseOldFiber.dom,
			parent:            parseParent,
			alternate:         parseOldFiber,
			effectTag:         parseEffectTag,
			dirty:             parseOldFiber.dirty,
			needsUpdate:       parseOldFiber.needsUpdate,
			hooks:             parseOldFiber.hooks, // Share hooks for non-updated components
			eventCallbacks:    parseOldFiber.eventCallbacks,
			contextValues:     parseOldFiber.contextValues,
			reactiveAtomID:    parseOldFiber.reactiveAtomID,
			reactiveSourceIDs: parseOldFiber.reactiveSourceIDs,
			fineGrained:       parseOldFiber.fineGrained,
			updateOrigin:      parseOldFiber.updateOrigin,
		}
		if parseNewFiber.hooks != nil {
			parseNewFiber.hooks.owner = parseNewFiber
		}
		ensureFineGrainedTwinLink(parseOldFiber, parseNewFiber)
		parseRt.handleClonedFiberSubscriptionMove(parseOldFiber, parseNewFiber)

		if parsePrevSibling == nil {
			parseParent.child = parseNewFiber
		} else {
			parsePrevSibling.sibling = parseNewFiber
		}
		parsePrevSibling = parseNewFiber
		parseOldFiber = parseOldFiber.sibling
	}
}

// handleClonedFiberSubscriptionMove moves atom subscriptions from one cloned fiber to its new current fiber.
func (parseRt *Runtime) handleClonedFiberSubscriptionMove(parseOldFiber *Fiber, parseNewFiber *Fiber) {
	if parseRt == nil || parseRt.atomRegistry == nil || parseOldFiber == nil || parseNewFiber == nil || parseOldFiber == parseNewFiber {
		return
	}
	getAtomIDs := buildClonedFiberSubscriptionAtomIDs(parseNewFiber)
	if len(getAtomIDs) == 0 {
		return
	}
	if parseRt.hydrating {
		for _, parseAtomID := range getAtomIDs {
			if parseAtomID == "" {
				continue
			}
			parseRt.queueHydrationSubscription(parseAtomID, parseOldFiber, false)
			parseRt.queueHydrationSubscription(parseAtomID, parseNewFiber, true)
		}
		return
	}
	parseRt.atomRegistry.MoveSubscriptions(getAtomIDs, parseOldFiber, parseNewFiber)
}

// buildClonedFiberSubscriptionAtomIDs returns one deduplicated atom ID list for cloned-fiber subscription transfer.
func buildClonedFiberSubscriptionAtomIDs(parseFiber *Fiber) []string {
	if parseFiber == nil {
		return nil
	}
	getCapacityHint := len(parseFiber.reactiveSourceIDs)
	if parseFiber.hooks != nil {
		getCapacityHint += len(parseFiber.hooks.atoms)
	}
	if getCapacityHint == 0 {
		return nil
	}
	getAtomIDs := make([]string, 0, getCapacityHint)
	hasAtomIDSeen := make(map[string]bool, getCapacityHint)
	storeAtomIDs := func(parseSourceIDs []string) {
		for _, parseAtomID := range parseSourceIDs {
			if parseAtomID == "" || hasAtomIDSeen[parseAtomID] {
				continue
			}
			hasAtomIDSeen[parseAtomID] = true
			getAtomIDs = append(getAtomIDs, parseAtomID)
		}
	}
	if parseFiber.hooks != nil && len(parseFiber.hooks.atoms) > 0 {
		storeAtomIDs(parseFiber.hooks.atoms)
	}
	if len(parseFiber.reactiveSourceIDs) > 0 {
		storeAtomIDs(parseFiber.reactiveSourceIDs)
	}
	return getAtomIDs
}

// reconcileChildren reconciles the children of a fiber
func (parseRt *Runtime) reconcileChildren(parseWipFiber *Fiber, parseElements []interface{}) {
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
					// UPDATE logic - optimized path
					// Check if this fiber or its subtree needs update
					isDirty := parseRt.isFiberDirty(parseOldFiber2)
					isParseNeedsUpdate := isDirty || parseOldFiber2.needsUpdate

					if !isParseNeedsUpdate {
						if parseT, parseOk3 := parseElem.Type.(string); parseOk3 && parseT == "TEXT_ELEMENT" {
							parseOldText := parseOldFiber2.textContent
							if parseOldText == "" && parseOldFiber2.props != nil {
								parseOldText, _ = parseOldFiber2.props["nodeValue"].(string)
							}
							parseNewText := parseElem.TextContent
							if parseNewText == "" && parseElem.Props != nil {
								parseNewText, _ = parseElem.Props["nodeValue"].(string)
							}
							isParseNeedsUpdate = parseOldText != parseNewText
						} else {
							isParseNeedsUpdate = !propsEqual(parseOldFiber2.props, parseElem.Props)
						}
					}

					parseEffectTag := "UPDATE"
					if !isParseNeedsUpdate {
						parseEffectTag = ""
					}

					// Get from pool and reset
					parseNewFiber = acquireWorkInProgress(parseOldFiber2)
					*parseNewFiber = Fiber{
						typeOf:            parseOldFiber2.typeOf,
						props:             parseElem.Props,
						textContent:       parseElem.TextContent,
						dom:               parseOldFiber2.dom,
						parent:            parseWipFiber,
						alternate:         parseOldFiber2,
						effectTag:         parseEffectTag,
						dirty:             isParseNeedsUpdate,
						needsUpdate:       parseOldFiber2.needsUpdate,
						hooks:             parseOldFiber2.hooks,
						eventCallbacks:    parseOldFiber2.eventCallbacks,
						hydration:         parseWipFiber.childHydration,
						reactiveAtomID:    parseOldFiber2.reactiveAtomID,
						reactiveSourceIDs: parseOldFiber2.reactiveSourceIDs,
						fineGrained:       parseOldFiber2.fineGrained,
						updateOrigin:      parseOldFiber2.updateOrigin,
					}
					ensureFineGrainedTwinLink(parseOldFiber2, parseNewFiber)
					parseRt.handleClonedFiberSubscriptionMove(parseOldFiber2, parseNewFiber)

					// Advance oldFiber
					parseOldFiber2 = parseOldFiber2.sibling
				} else {
					// REPLACE logic (Placement + Deletion)
					parseNewFiber = acquireWorkInProgress(parseOldFiber2)
					*parseNewFiber = Fiber{
						typeOf:       parseElem.Type,
						props:        parseElem.Props,
						textContent:  parseElem.TextContent,
						parent:       parseWipFiber,
						effectTag:    "PLACEMENT",
						dirty:        true,
						hydration:    parseWipFiber.childHydration,
						fineGrained:  isFineGrainedType(parseElem.Type),
						updateOrigin: oldFiberUpdateOrigin(parseOldFiber2, parseElem.Type),
					}

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
				parseNewFiber2 = acquireWorkInProgress(nil)
				*parseNewFiber2 = Fiber{
					typeOf:       parseElem2.Type,
					props:        parseElem2.Props,
					textContent:  parseElem2.TextContent,
					parent:       parseWipFiber,
					effectTag:    "PLACEMENT",
					dirty:        true,
					hydration:    parseWipFiber.childHydration,
					fineGrained:  isFineGrainedType(parseElem2.Type),
					updateOrigin: oldFiberUpdateOrigin(nil, parseElem2.Type),
				}
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
	parseOldByKey := keyedFiberMapPool.Get().(map[interface{}]*Fiber)
	parseOldFallbackKeyed := fiberScratchSlicePool.get()
	parseOldUnkeyed := fiberScratchSlicePool.get()
	defer func() {
		clear(parseOldByKey)
		keyedFiberMapPool.Put(parseOldByKey)
		fiberScratchSlicePool.clear(parseOldFallbackKeyed)
		fiberScratchSlicePool.clear(parseOldUnkeyed)
	}()

	var parseOldFirst *Fiber
	if parseWipFiber.alternate != nil {
		parseOldFirst = parseWipFiber.alternate.child
	}

	for parseOldFiber := parseOldFirst; parseOldFiber != nil; parseOldFiber = parseOldFiber.sibling {
		if parseKey, parseOk := fiberComparableKey(parseOldFiber); parseOk {
			parseOldByKey[parseKey] = parseOldFiber
		} else if hasFiberKey(parseOldFiber) {
			parseOldFallbackKeyed = append(parseOldFallbackKeyed, parseOldFiber)
		} else {
			parseOldUnkeyed = append(parseOldUnkeyed, parseOldFiber)
		}
	}

	parseUnkeyedIndex := 0
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
			isDirty := parseRt.isFiberDirty(parseMatchedOld)
			isParseNeedsUpdate := isDirty || parseMatchedOld.needsUpdate

			if !isParseNeedsUpdate {
				if parseT, parseOk3 := parseElem.Type.(string); parseOk3 && parseT == "TEXT_ELEMENT" {
					parseOldText := parseMatchedOld.textContent
					if parseOldText == "" && parseMatchedOld.props != nil {
						parseOldText, _ = parseMatchedOld.props["nodeValue"].(string)
					}
					parseNewText := parseElem.TextContent
					if parseNewText == "" && parseElem.Props != nil {
						parseNewText, _ = parseElem.Props["nodeValue"].(string)
					}
					isParseNeedsUpdate = parseOldText != parseNewText
				} else {
					isParseNeedsUpdate = !propsEqual(parseMatchedOld.props, parseElem.Props)
				}
			}

			parseEffectTag := "UPDATE"
			if !isParseNeedsUpdate {
				parseEffectTag = ""
			}

			parseNewFiber = acquireWorkInProgress(parseMatchedOld)
			*parseNewFiber = Fiber{
				typeOf:            parseMatchedOld.typeOf,
				props:             parseElem.Props,
				textContent:       parseElem.TextContent,
				dom:               parseMatchedOld.dom,
				parent:            parseWipFiber,
				alternate:         parseMatchedOld,
				effectTag:         parseEffectTag,
				dirty:             isParseNeedsUpdate,
				needsUpdate:       parseMatchedOld.needsUpdate,
				hooks:             parseMatchedOld.hooks,
				eventCallbacks:    parseMatchedOld.eventCallbacks,
				hydration:         parseWipFiber.childHydration,
				reactiveAtomID:    parseMatchedOld.reactiveAtomID,
				reactiveSourceIDs: parseMatchedOld.reactiveSourceIDs,
				fineGrained:       parseMatchedOld.fineGrained,
				updateOrigin:      parseMatchedOld.updateOrigin,
			}
			ensureFineGrainedTwinLink(parseMatchedOld, parseNewFiber)
			parseRt.handleClonedFiberSubscriptionMove(parseMatchedOld, parseNewFiber)
		} else {
			if parseMatchedOld != nil {
				parseMatchedOld.effectTag = "DELETION"
				parseRt.deletions = append(parseRt.deletions, parseMatchedOld)
			}

			parseNewFiber = acquireWorkInProgress(nil)
			*parseNewFiber = Fiber{
				typeOf:       parseElem.Type,
				props:        parseElem.Props,
				textContent:  parseElem.TextContent,
				parent:       parseWipFiber,
				effectTag:    "PLACEMENT",
				dirty:        true,
				hydration:    parseWipFiber.childHydration,
				fineGrained:  isFineGrainedType(parseElem.Type),
				updateOrigin: oldFiberUpdateOrigin(nil, parseElem.Type),
			}
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
			} else if parseFiber.dom == nil || parseFiber.dom.IsNull() {
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
			if parseFiber.dom == nil || parseFiber.dom.IsNull() {
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
func (parseRt *Runtime) createDom(parseFiber *Fiber) DOMNode {
	var parseDom DOMNode

	if parseT, parseOk := parseFiber.typeOf.(string); parseOk {
		switch parseT {
		case "TEXT_ELEMENT":
			parseText := parseFiber.textContent
			if parseText == "" && parseFiber.props != nil {
				parseText, _ = parseFiber.props["nodeValue"].(string)
			}
			parseDom = parseRt.domAdapter.CreateTextNode(parseText)
		case "FRAGMENT":
			// Fragments don't create DOM nodes - children are rendered directly
			return nil
		default:
			// Regular element (not TEXT_ELEMENT or FRAGMENT)
			parseDom = parseRt.domAdapter.CreateElement(parseT)
			// Apply properties only for non-text elements
			parseRt.updateDomProperties(parseDom, nil, parseFiber.props)
		}
	}
	if _, parseOk2 := parseFiber.typeOf.(*PortalElementType); parseOk2 {
		return nil
	}
	if _, parseOk3 := parseFiber.typeOf.(*ReactiveRegionElementType); parseOk3 {
		return nil
	}
	if _, parseOk4 := parseFiber.typeOf.(*ReactiveTextElementType); parseOk4 {
		return parseRt.domAdapter.CreateTextNode(reactiveTextValue(parseFiber))
	}
	// Function components don't have DOM nodes - they render their children

	return parseDom
}

// updateDomProperties updates DOM properties with optimized batching when available
func (parseRt *Runtime) updateDomProperties(parseDom DOMNode, parseOldProps, parseNewProps map[string]interface{}) {
	// Check if dom is nil (interface is nil) or if the concrete value is null
	if parseDom == nil || parseDom.IsNull() {
		return
	}

	// fmt.Printf("updateDomProperties: updating %d old props, %d new props\n", len(oldProps), len(newProps))

	// Check if adapter supports batching (only for WASM adapter)
	parseBatchAdapter, parseSupportsBatching := parseRt.domAdapter.(interface {
		BatchSetAttributes(DOMNode, map[string]string)
	})

	// Optimization: Fast path for initial render (no old props)
	if len(parseOldProps) == 0 && len(parseNewProps) > 0 {
		parseRt.applyInitialDomProps(parseDom, parseNewProps, parseSupportsBatching, parseBatchAdapter, false)
		return
	}

	// Remove old properties
	for parseName := range parseOldProps {
		parseMeta := getPropMeta(parseName)
		if parseMeta.kind == propKindSkip {
			continue
		}
		parseTargetName := parseMeta.attrName
		if parseTargetName == "" {
			parseTargetName = parseName
		}
		if _, parseExists := parseNewProps[parseName]; !parseExists {
			if parseMeta.shouldReset {
				parseRt.domAdapter.SetProperty(parseDom, parseTargetName, parseMeta.resetValue)
			} else {
				parseRt.domAdapter.RemoveAttribute(parseDom, parseMeta.attrName)
			}
		}
	}

	// Set new properties
	for parseName2, parseValue := range parseNewProps {
		parseMeta2 := getPropMeta(parseName2)
		if parseMeta2.kind == propKindSkip {
			continue
		}
		parseTargetName2 := parseMeta2.attrName
		if parseTargetName2 == "" {
			parseTargetName2 = parseName2
		}

		// Optimization: Skip if value hasn't changed
		if parseOldValue, parseExists2 := parseOldProps[parseName2]; parseExists2 && fastEqual(parseOldValue, parseValue) {
			continue
		}

		switch parseMeta2.kind {
		case propKindStyle:
			if parseStyles, parseOk := parseValue.(map[string]string); parseOk {
				parseRt.domAdapter.SetStyles(parseDom, parseStyles)
			} else if parseStr, parseOk2 := parseValue.(string); parseOk2 {
				// Allow inline style strings as attribute values
				parseRt.domAdapter.SetAttribute(parseDom, parseMeta2.attrName, parseStr)
			}
		case propKindClass:
			if parseStr2, parseOk3 := parseValue.(string); parseOk3 {
				parseRt.domAdapter.SetAttribute(parseDom, parseMeta2.attrName, parseStr2)
			}
		case propKindSpecialProperty:
			// Always set these as properties to ensure UI updates correctly
			parseRt.domAdapter.SetProperty(parseDom, parseTargetName2, parseValue)
		default:
			if parseStr3, parseOk4 := parseValue.(string); parseOk4 {
				parseRt.domAdapter.SetAttribute(parseDom, parseMeta2.attrName, parseStr3)
			} else {
				// Always update properties (especially event handlers which are closures)
				parseRt.domAdapter.SetProperty(parseDom, parseTargetName2, parseValue)
			}
		}
	}
}

// applyInitialDomProps is an internal reconciler helper.
func (parseRt *Runtime) applyInitialDomProps(parseDom DOMNode, parseNewProps map[string]interface{}, isSupportsBatching bool, parseBatchAdapter interface {
	BatchSetAttributes(DOMNode, map[string]string)
}, isPreserveHydrationState bool) {
	var parseAttrBatch map[string]string
	parseFlushAttrBatch := func() {
		if !isSupportsBatching || len(parseAttrBatch) == 0 {
			return
		}
		parseBatchAdapter.BatchSetAttributes(parseDom, parseAttrBatch)
		clear(parseAttrBatch)
	}

	for parseName, parseValue := range parseNewProps {
		parseMeta := getPropMeta(parseName)
		if parseMeta.kind == propKindSkip {
			continue
		}
		parseTargetName := parseMeta.attrName
		if parseTargetName == "" {
			parseTargetName = parseName
		}
		if isPreserveHydrationState && shouldPreserveHydrationInitialProperty(parseTargetName) {
			continue
		}

		switch parseMeta.kind {
		case propKindStyle:
			parseFlushAttrBatch()
			if parseStyles, parseOk := parseValue.(map[string]string); parseOk {
				parseRt.domAdapter.SetStyles(parseDom, parseStyles)
			} else if parseStr, parseOk2 := parseValue.(string); parseOk2 {
				if isSupportsBatching {
					if parseAttrBatch == nil {
						parseAttrBatch = make(map[string]string, len(parseNewProps))
					}
					parseAttrBatch[parseMeta.attrName] = parseStr
				} else {
					parseRt.domAdapter.SetAttribute(parseDom, parseMeta.attrName, parseStr)
				}
			}
		case propKindClass:
			if parseStr2, parseOk3 := parseValue.(string); parseOk3 {
				if isSupportsBatching {
					if parseAttrBatch == nil {
						parseAttrBatch = make(map[string]string, len(parseNewProps))
					}
					parseAttrBatch[parseMeta.attrName] = parseStr2
				} else {
					parseRt.domAdapter.SetAttribute(parseDom, parseMeta.attrName, parseStr2)
				}
			}
		case propKindSpecialProperty:
			parseFlushAttrBatch()
			parseRt.domAdapter.SetProperty(parseDom, parseTargetName, parseValue)
		default:
			if parseStr3, parseOk4 := parseValue.(string); parseOk4 {
				if isSupportsBatching {
					if parseAttrBatch == nil {
						parseAttrBatch = make(map[string]string, len(parseNewProps))
					}
					parseAttrBatch[parseMeta.attrName] = parseStr3
				} else {
					parseRt.domAdapter.SetAttribute(parseDom, parseMeta.attrName, parseStr3)
				}
			} else {
				parseFlushAttrBatch()
				parseRt.domAdapter.SetProperty(parseDom, parseTargetName, parseValue)
			}
		}
	}

	parseFlushAttrBatch()
}

// shouldPreserveHydrationInitialProperty is an internal reconciler helper.
func shouldPreserveHydrationInitialProperty(parseName string) bool {
	switch parseName {
	case "value", "checked", "selected", "autofocus":
		return true
	default:
		return false
	}
}

// commitRoot commits all changes to the DOM
func (parseRt *Runtime) commitRoot() {
	parseStart := time.Now()
	defer func() {
		parseDurationNs := time.Since(parseStart).Nanoseconds()
		parseRt.profiling.commitCount++
		parseRt.profiling.lastCommitDurationNs = parseDurationNs
		parseRt.profiling.totalCommitDurationNs += parseDurationNs
		parseRt.recordProfilingEventLocked(ProfilingEvent{
			Domain:     "runtime",
			Name:       "commit",
			Phase:      "finish",
			Target:     "root",
			DurationNs: parseDurationNs,
		})
		if !parseRt.profiling.startupStartedAt.IsZero() && parseRt.profiling.startupCommitDurationNs == 0 {
			parseRt.profiling.startupCommitDurationNs = parseDurationNs
			parseRt.recordProfilingEventLocked(ProfilingEvent{
				Domain:     "runtime",
				Name:       "startup.commit",
				Phase:      "finish",
				Target:     parseRt.profiling.startupMode,
				DurationNs: parseDurationNs,
			})
		}
	}()
	// Process deletions first
	for _, parseFiber := range parseRt.deletions {
		// Deletions need to find their parent DOM node
		// We can't pass a cached parent here easily because deletions can be anywhere
		parseRt.commitWork(parseFiber, nil)
	}
	// Clear deletions but keep capacity
	parseRt.deletions = parseRt.deletions[:0]

	// Commit the work
	parseCommittedRoot := parseRt.wipRoot
	if parseRt.wipRoot != nil && parseRt.wipRoot.child != nil {
		parseRt.finalizeHydrationBoundary(parseRt.wipRoot.childHydration, parseRt.wipRoot)
		// The root fiber's DOM node is the container
		parseRt.commitWork(parseRt.wipRoot.child, parseRt.wipRoot.dom)
	}

	parseRt.currentRoot = parseCommittedRoot
	parseRt.wipRoot = nil
	parseWasHydrating := parseRt.hydrating
	if parseWasHydrating {
		parseRt.hydrating = false
		parseRt.strictHydration = false
		parseRt.flushHydrationSubscriptions()
	}
	parseRt.updateScheduled = false

	// Run effects after the committed tree is current and hydration gates are lifted.
	parseRt.runEffects(parseCommittedRoot)

	if parseWasHydrating {
		parseRt.finishHydrationMetrics(false, "")
		parseRt.flushDeferredHydrationUpdates()
	}
	if parseRt.pendingBoundaryRecovery {
		parseRt.pendingBoundaryRecovery = false
		parseRt.ScheduleUpdate()
	}
}

// reportMissingKeys is an internal reconciler helper.
func reportMissingKeys(parseParent *Fiber, parseElements []interface{}) {
	parseRenderableCount := 0
	parseMissingKeyCount := 0
	hasKeyedSibling := false
	for _, parseElement := range parseElements {
		parseElem, parseOk := parseElement.(*Element)
		if !parseOk || parseElem == nil {
			continue
		}
		parseRenderableCount++
		if hasElementKey(parseElem) {
			hasKeyedSibling = true
			continue
		}
		parseMissingKeyCount++
	}

	if parseRenderableCount <= 1 || parseMissingKeyCount == 0 {
		return
	}

	if !hasKeyedSibling {
		for parseOldFiber := parentChild(parseParent); parseOldFiber != nil; parseOldFiber = parseOldFiber.sibling {
			if hasFiberKey(parseOldFiber) {
				hasKeyedSibling = true
				break
			}
		}
	}

	if !hasKeyedSibling {
		return
	}

	_, parseParentName := describeFiber(parseParent)
	ReportDiagnostic("runtime", DiagnosticWarning, "missing key on one or more sibling elements under "+parseParentName)
}

// parentChild is an internal reconciler helper.
func parentChild(parseParent *Fiber) *Fiber {
	if parseParent == nil || parseParent.alternate == nil {
		return nil
	}
	return parseParent.alternate.child
}

// commitWork commits a fiber's changes to the DOM with batch optimization
func (parseRt *Runtime) commitWork(parseFiber *Fiber, parseDomParent DOMNode) {
	if parseFiber == nil {
		return
	}

	// Fast path: most calls have valid domParent
	if parseDomParent == nil || parseDomParent.IsNull() {
		var parseDomParentFiber *Fiber = parseFiber.parent
		for parseDomParentFiber != nil && (parseDomParentFiber.dom == nil || parseDomParentFiber.dom.IsNull()) {
			parseDomParentFiber = parseDomParentFiber.parent
		}
		if parseDomParentFiber != nil {
			parseDomParent = parseDomParentFiber.dom
		}
	}

	isPortal := parseRt.isPortalFiber(parseFiber)
	parsePortalParent := parseDomParent
	isParsePortalTargetChanged := false
	if isPortal {
		parsePortalParent = parseRt.resolvePortalParent(parseFiber)
		isParsePortalTargetChanged = parseRt.portalTargetChanged(parseFiber)
	}

	if !isPortal && parseDomParent != nil && !parseDomParent.IsNull() {
		if parseFiber.effectTag == "PLACEMENT" && parseFiber.dom != nil && !parseFiber.dom.IsNull() {
			parseStart := time.Now()
			parseRt.domAdapter.AppendChild(parseDomParent, parseFiber.dom)
			parseFiber.commitDurationNs += time.Since(parseStart).Nanoseconds()
			if parseFiber.fineGrained {
				parseRt.profiling.fineGrainedCommits++
			} else {
				parseRt.recordFineGrainedDescendantCommit(parseFiber)
			}
		} else if parseFiber.effectTag == "HYDRATE" && parseFiber.dom != nil && !parseFiber.dom.IsNull() {
			if isTextLikeFiber(parseFiber) {
				parseNewValue := parseFiber.textContent
				if _, parseOk := parseFiber.typeOf.(*ReactiveTextElementType); parseOk {
					parseNewValue = reactiveTextValue(parseFiber)
				} else if parseNewValue == "" && parseFiber.props != nil {
					parseNewValue, _ = parseFiber.props["nodeValue"].(string)
				}
				parseStart2 := time.Now()
				parseRt.domAdapter.SetTextContent(parseFiber.dom, parseNewValue)
				parseFiber.commitDurationNs += time.Since(parseStart2).Nanoseconds()
				if parseFiber.fineGrained {
					parseRt.profiling.fineGrainedCommits++
				} else {
					parseRt.recordFineGrainedDescendantCommit(parseFiber)
				}
			} else {
				parseStart3 := time.Now()
				parseBatchAdapter, parseSupportsBatching := parseRt.domAdapter.(interface {
					BatchSetAttributes(DOMNode, map[string]string)
				})
				parseRt.applyInitialDomProps(parseFiber.dom, parseFiber.props, parseSupportsBatching, parseBatchAdapter, parseFiber.hydrated)
				parseFiber.commitDurationNs += time.Since(parseStart3).Nanoseconds()
				parseRt.recordFineGrainedDescendantCommit(parseFiber)
			}
		} else if parseFiber.effectTag == "UPDATE" && parseFiber.dom != nil && !parseFiber.dom.IsNull() {
			if parseFiber.alternate != nil {
				if isTextLikeFiber(parseFiber) {
					// Update text content
					parseOldValue := textLikeFiberValue(parseFiber.alternate)
					parseNewValue2 := textLikeFiberValue(parseFiber)

					if parseOldValue != parseNewValue2 {
						parseStart4 := time.Now()
						parseRt.domAdapter.SetTextContent(parseFiber.dom, parseNewValue2)
						parseFiber.commitDurationNs += time.Since(parseStart4).Nanoseconds()
						if parseFiber.fineGrained {
							parseRt.profiling.fineGrainedCommits++
						} else {
							parseRt.recordFineGrainedDescendantCommit(parseFiber)
						}
					}
				} else {
					// Regular element - update properties
					parseStart5 := time.Now()
					parseRt.updateDomProperties(parseFiber.dom, parseFiber.alternate.props, parseFiber.props)
					parseFiber.commitDurationNs += time.Since(parseStart5).Nanoseconds()
					parseRt.recordFineGrainedDescendantCommit(parseFiber)
				}
			}
		} else if parseFiber.effectTag == "DELETION" {
			// fmt.Printf("DEBUG: Committing deletion for %v\n", fiber.typeOf)
			parseRt.commitDeletion(parseFiber, parseDomParent)
			return
		}
	}

	if isPortal {
		if parseFiber.effectTag == "DELETION" {
			parseRt.commitDeletion(parseFiber, parsePortalParent)
			return
		}
		if isParsePortalTargetChanged {
			parseRt.movePortalSubtree(parseFiber.child, parseRt.resolvePortalParent(parseFiber.alternate), parsePortalParent)
		}
		if parsePortalParent == nil || parsePortalParent.IsNull() {
			if parseFiber.alternate != nil {
				parseRt.deleteFiberSubtree(parseFiber.child, parseRt.resolvePortalParent(parseFiber.alternate))
			}
			if parseFiber.sibling != nil {
				parseRt.commitWork(parseFiber.sibling, parseDomParent)
			}
			return
		}
	}

	// Determine the parent DOM node for children
	// If this fiber has a DOM node, it becomes the parent for its children
	parseChildDomParent := parseDomParent
	if isPortal {
		parseChildDomParent = parsePortalParent
	}
	if parseFiber.dom != nil && !parseFiber.dom.IsNull() {
		parseChildDomParent = parseFiber.dom
	}
	parseRt.finalizeHydrationBoundary(parseFiber.childHydration, parseFiber)

	// Recursively commit children and siblings
	if parseFiber.child != nil {
		parseRt.commitWork(parseFiber.child, parseChildDomParent)
	}
	if parseFiber.sibling != nil {
		parseRt.commitWork(parseFiber.sibling, parseDomParent)
	}
}

// oldFiberUpdateOrigin is an internal reconciler helper.
func oldFiberUpdateOrigin(parseOldFiber *Fiber, parseTypeOf interface{}) string {
	if parseOldFiber != nil && parseOldFiber.updateOrigin != "" {
		return parseOldFiber.updateOrigin
	}
	if isFineGrainedType(parseTypeOf) {
		return "fine-grained"
	}
	return ""
}

// isReactiveTextType is an internal reconciler helper.
func isReactiveTextType(parseTypeOf interface{}) bool {
	_, parseOk := parseTypeOf.(*ReactiveTextElementType)
	return parseOk
}

// isReactiveRegionType is an internal reconciler helper.
func isReactiveRegionType(parseTypeOf interface{}) bool {
	_, parseOk := parseTypeOf.(*ReactiveRegionElementType)
	return parseOk
}

// isFineGrainedType is an internal reconciler helper.
func isFineGrainedType(parseTypeOf interface{}) bool {
	return isReactiveTextType(parseTypeOf) || isReactiveRegionType(parseTypeOf)
}

// isHostFiber is an internal reconciler helper.
func isHostFiber(parseFiber *Fiber) bool {
	if parseFiber == nil {
		return false
	}
	parseTyp, parseOk := parseFiber.typeOf.(string)
	if !parseOk {
		return false
	}
	switch parseTyp {
	case "ROOT", "FRAGMENT", "TEXT_ELEMENT":
		return false
	default:
		return true
	}
}

// hasFineGrainedAncestor is an internal reconciler helper.
func hasFineGrainedAncestor(parseFiber *Fiber) bool {
	for parseParent := parseFiber.parent; parseParent != nil; parseParent = parseParent.parent {
		if parseParent.fineGrained {
			return true
		}
	}
	return false
}

// recordFineGrainedDescendantCommit is an internal reconciler helper.
func (parseRt *Runtime) recordFineGrainedDescendantCommit(parseFiber *Fiber) {
	if parseRt == nil || parseFiber == nil || parseFiber.fineGrained || !hasFineGrainedAncestor(parseFiber) {
		return
	}
	if isTextLikeFiber(parseFiber) {
		parseRt.profiling.fineGrainedDescendantTextCommits++
		return
	}
	if isHostFiber(parseFiber) {
		parseRt.profiling.fineGrainedDescendantHostCommits++
	}
}

// isTextLikeFiber is an internal reconciler helper.
func isTextLikeFiber(parseFiber *Fiber) bool {
	if parseFiber == nil {
		return false
	}
	if _, parseOk := parseFiber.typeOf.(*ReactiveTextElementType); parseOk {
		return true
	}
	parseTyp, parseOk2 := parseFiber.typeOf.(string)
	return parseOk2 && parseTyp == "TEXT_ELEMENT"
}

// textLikeFiberValue is an internal reconciler helper.
func textLikeFiberValue(parseFiber *Fiber) string {
	if parseFiber == nil {
		return ""
	}
	if _, parseOk := parseFiber.typeOf.(*ReactiveTextElementType); parseOk {
		return parseFiber.textContent
	}
	parseValue := parseFiber.textContent
	if parseValue == "" && parseFiber.props != nil {
		parseValue, _ = parseFiber.props["nodeValue"].(string)
	}
	return parseValue
}

const (
	reactiveTextAtomIDProp      = "__gwc_reactive_text_atom_id"
	reactiveTextGetterProp      = "__gwc_reactive_text_getter"
	reactiveRegionSourceIDsProp = "__gwc_reactive_region_source_ids"
	reactiveRegionRenderProp    = "__gwc_reactive_region_render"
)

// reactiveTextValue is an internal reconciler helper.
func reactiveTextValue(parseFiber *Fiber) string {
	if parseFiber == nil || parseFiber.props == nil {
		return ""
	}
	parseGetter, _ := parseFiber.props[reactiveTextGetterProp].(func() string)
	if parseGetter == nil {
		return ""
	}
	return parseGetter()
}

// reactiveRegionValue is an internal reconciler helper.
func reactiveRegionValue(parseFiber *Fiber) *Element {
	if parseFiber == nil || parseFiber.props == nil {
		return nil
	}
	render, _ := parseFiber.props[reactiveRegionRenderProp].(func() *Element)
	if render == nil {
		return nil
	}
	return render()
}

// reactiveRegionSourceIDs is an internal reconciler helper.
func reactiveRegionSourceIDs(parseFiber *Fiber) []string {
	if parseFiber == nil || parseFiber.props == nil {
		return nil
	}
	parseRaw, _ := parseFiber.props[reactiveRegionSourceIDsProp].([]string)
	if len(parseRaw) == 0 {
		return nil
	}
	if len(parseRaw) == 1 {
		if strings.TrimSpace(parseRaw[0]) == "" {
			return nil
		}
		return parseRaw
	}
	parseIds := make([]string, 0, len(parseRaw))
	parseSeen := make(map[string]struct{}, len(parseRaw))
	for _, parseId := range parseRaw {
		if strings.TrimSpace(parseId) == "" {
			continue
		}
		if _, parseExists := parseSeen[parseId]; parseExists {
			continue
		}
		parseSeen[parseId] = struct{}{}
		parseIds = append(parseIds, parseId)
	}
	if len(parseIds) == 0 {
		return nil
	}
	return parseIds
}

// sameReactiveSourceIDs is an internal reconciler helper.
func sameReactiveSourceIDs(parseLeft []string, parseRight []string) bool {
	if len(parseLeft) != len(parseRight) {
		return false
	}
	for parseIndex := range parseLeft {
		if parseLeft[parseIndex] != parseRight[parseIndex] {
			return false
		}
	}
	return true
}

// syncFineGrainedSubscriptions is an internal reconciler helper.
func (parseRt *Runtime) syncFineGrainedSubscriptions(parseFiber *Fiber, parseSourceIDs []string) {
	if parseRt == nil || parseRt.atomRegistry == nil || parseFiber == nil {
		return
	}
	parsePrevious := parseFiber.reactiveSourceIDs
	if parseFiber.alternate != nil && parseFiber.alternate != parseFiber && len(parseFiber.alternate.reactiveSourceIDs) > 0 {
		parsePrevious = parseFiber.alternate.reactiveSourceIDs
	}
	if sameReactiveSourceIDs(parsePrevious, parseSourceIDs) {
		parseFiber.reactiveSourceIDs = parsePrevious
		parseFiber.reactiveAtomID = ""
		if len(parsePrevious) > 0 {
			parseFiber.reactiveAtomID = strings.Join(parsePrevious, ",")
		}
		parseFiber.fineGrained = len(parsePrevious) > 0
		return
	}
	parsePreviousSet := make(map[string]struct{}, len(parsePrevious))
	for _, parseId := range parsePrevious {
		parsePreviousSet[parseId] = struct{}{}
	}
	parseNextSet := make(map[string]struct{}, len(parseSourceIDs))
	for _, parseId2 := range parseSourceIDs {
		parseNextSet[parseId2] = struct{}{}
	}
	for _, parseOldID := range parsePrevious {
		if _, parseKeep := parseNextSet[parseOldID]; parseKeep {
			continue
		}
		if parseRt.hydrating {
			parseRt.queueHydrationSubscription(parseOldID, parseFiber, false)
			if parseFiber.alternate != nil && parseFiber.alternate != parseFiber {
				parseRt.queueHydrationSubscription(parseOldID, parseFiber.alternate, false)
			}
		} else {
			parseRt.atomRegistry.Unsubscribe(parseOldID, parseFiber)
			if parseFiber.alternate != nil && parseFiber.alternate != parseFiber {
				parseRt.atomRegistry.Unsubscribe(parseOldID, parseFiber.alternate)
			}
		}
	}
	for _, parseNewID := range parseSourceIDs {
		if _, parseAlready := parsePreviousSet[parseNewID]; parseAlready {
			continue
		}
		if parseRt.hydrating {
			parseRt.queueHydrationSubscription(parseNewID, parseFiber, true)
		} else {
			parseRt.atomRegistry.Subscribe(parseNewID, parseFiber)
		}
	}
	parseFiber.reactiveSourceIDs = append([]string(nil), parseSourceIDs...)
	parseFiber.reactiveAtomID = ""
	if len(parseSourceIDs) > 0 {
		parseFiber.reactiveAtomID = strings.Join(parseSourceIDs, ",")
	}
	parseFiber.fineGrained = len(parseSourceIDs) > 0
}

// syncReactiveTextSubscription is an internal reconciler helper.
func (parseRt *Runtime) syncReactiveTextSubscription(parseFiber *Fiber) {
	if parseRt == nil || parseFiber == nil || parseFiber.props == nil {
		return
	}
	parseAtomID, _ := parseFiber.props[reactiveTextAtomIDProp].(string)
	parseSourceIDs := []string{}
	if parseAtomID != "" {
		parseSourceIDs = append(parseSourceIDs, parseAtomID)
	}
	parseRt.syncFineGrainedSubscriptions(parseFiber, parseSourceIDs)
	if parseAtomID != "" {
		parseFiber.reactiveAtomID = parseAtomID
	}
}

// syncReactiveRegionSubscription is an internal reconciler helper.
func (parseRt *Runtime) syncReactiveRegionSubscription(parseFiber *Fiber) {
	if parseRt == nil || parseFiber == nil {
		return
	}
	parseRt.syncFineGrainedSubscriptions(parseFiber, reactiveRegionSourceIDs(parseFiber))
}

// commitDeletion removes a fiber from the DOM and runs cleanup functions
func (parseRt *Runtime) commitDeletion(parseFiber *Fiber, parseDomParent DOMNode) {
	if parseFiber == nil {
		return
	}
	// fmt.Printf("DEBUG: commitDeletion for %v\n", fiber.typeOf)
	// Run all cleanup functions before removing from DOM
	parseRt.runCleanups(parseFiber)

	// Cleanup atom subscriptions for this fiber and subtree
	parseRt.cleanupAtomSubscriptionsSubtree(parseFiber)

	if parseRt.isPortalFiber(parseFiber) {
		parseRt.deleteFiberSubtree(parseFiber.child, parseRt.resolvePortalParent(parseFiber))
		return
	}

	if parseFiber.dom != nil && !parseFiber.dom.IsNull() {
		// This fiber has a DOM node, remove it
		parseRt.domAdapter.RemoveChild(parseDomParent, parseFiber.dom)
	} else {
		// Function component without DOM node - recursively delete all descendants
		// We need to find and remove all actual DOM nodes in the subtree
		parseRt.deleteFiberSubtree(parseFiber.child, parseDomParent)
	}
}

// cleanupAtomSubscriptionsSubtree is an internal reconciler helper.
func (parseRt *Runtime) cleanupAtomSubscriptionsSubtree(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}
	parseRt.CleanupAtomSubscriptions(parseFiber)
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseRt.cleanupAtomSubscriptionsSubtree(parseChild)
	}
	if parseFiber.alternate != nil && parseFiber.alternate != parseFiber {
		parseRt.CleanupAtomSubscriptions(parseFiber.alternate)
	}
}

// deleteFiberSubtree recursively removes all DOM nodes in a fiber's subtree
func (parseRt *Runtime) deleteFiberSubtree(parseFiber *Fiber, parseDomParent DOMNode) {
	if parseFiber == nil {
		return
	}

	// If this fiber has a DOM node, remove it (this stops the recursion down that branch)
	if parseFiber.dom != nil && !parseFiber.dom.IsNull() {
		parseRt.domAdapter.RemoveChild(parseDomParent, parseFiber.dom)
	} else {
		// No DOM node on this fiber - recurse to children until we find DOM nodes.
		parseRt.deleteFiberSubtree(parseFiber.child, parseDomParent)
	}

	// Continue across sibling branches so DOM-less parents remove their full subtree.
	parseRt.deleteFiberSubtree(parseFiber.sibling, parseDomParent)
}

// isPortalFiber is an internal reconciler helper.
func (parseRt *Runtime) isPortalFiber(parseFiber *Fiber) bool {
	if parseFiber == nil {
		return false
	}
	_, parseOk := parseFiber.typeOf.(*PortalElementType)
	return parseOk
}

// resolvePortalParent is an internal reconciler helper.
func (parseRt *Runtime) resolvePortalParent(parseFiber *Fiber) DOMNode {
	if parseFiber == nil || parseFiber.props == nil {
		return nil
	}

	if parseRawNode, parseOk := parseFiber.props["portalTargetNode"]; parseOk && parseRawNode != nil {
		if parseNode, parseOk2 := parseRawNode.(DOMNode); parseOk2 {
			return parseNode
		}
		if parseResolver, parseOk3 := parseRt.domAdapter.(interface{ ResolveNode(interface{}) DOMNode }); parseOk3 {
			return parseResolver.ResolveNode(parseRawNode)
		}
	}

	if parseSelector, parseOk4 := parseFiber.props["portalTargetSelector"].(string); parseOk4 && parseSelector != "" {
		return parseRt.queryContainer(parseSelector)
	}

	return nil
}

// portalTargetChanged is an internal reconciler helper.
func (parseRt *Runtime) portalTargetChanged(parseFiber *Fiber) bool {
	if parseFiber == nil || parseFiber.alternate == nil {
		return false
	}
	return !fastEqual(parseFiber.props["portalTargetSelector"], parseFiber.alternate.props["portalTargetSelector"]) || !fastEqual(parseFiber.props["portalTargetNode"], parseFiber.alternate.props["portalTargetNode"])
}

// movePortalSubtree is an internal reconciler helper.
func (parseRt *Runtime) movePortalSubtree(parseFiber *Fiber, parseOldParent, parseNewParent DOMNode) {
	if parseFiber == nil || parseOldParent == nil || parseOldParent.IsNull() || parseNewParent == nil || parseNewParent.IsNull() || parseOldParent.Equals(parseNewParent) {
		return
	}

	if parseRt.isPortalFiber(parseFiber) {
		return
	}

	if parseFiber.dom != nil && !parseFiber.dom.IsNull() {
		parseRt.domAdapter.RemoveChild(parseOldParent, parseFiber.dom)
		parseRt.domAdapter.AppendChild(parseNewParent, parseFiber.dom)
	} else {
		parseRt.movePortalSubtree(parseFiber.child, parseOldParent, parseNewParent)
	}

	parseRt.movePortalSubtree(parseFiber.sibling, parseOldParent, parseNewParent)
}

// runCleanups runs all cleanup functions for a fiber and its children
func (parseRt *Runtime) runCleanups(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	// Run this fiber's cleanups
	if parseFiber.hooks != nil {
		for parseIndex, parseCleanup := range parseFiber.hooks.cleanups {
			if parseCleanup != nil {
				parseStart := time.Now()
				var isHandled bool
				func() {
					defer func() {
						if parseRecovered := recover(); parseRecovered != nil {
							if panicPhaseMayRecoverWithBoundary(PanicPhaseCleanup) {
								_, isHandled = parseRt.recoverBoundaryError(parseFiber, parseRecovered, boundaryPhaseCleanup)
							}
							if !isHandled {
								panic(markUnhandledPanic(parseFiber, boundaryPhaseCleanup, parseRecovered))
							}
						}
					}()
					parseCleanup()
				}()
				parseDurationNs := time.Since(parseStart).Nanoseconds()
				parseFiber.cleanupDurationNs += parseDurationNs
				parseRt.profiling.cleanupExecutions++
				parseRt.profiling.lastCleanupDurationNs = parseDurationNs
				parseRt.profiling.totalCleanupDurationNs += parseDurationNs
				recordSlowOperationDiagnostic("cleanup", parseFiber, parseDurationNs)
				parseFiber.hooks.cleanups[parseIndex] = nil
			}
		}
	}

	// Recursively run cleanups for children and siblings
	if parseFiber.child != nil {
		parseRt.runCleanups(parseFiber.child)
	}
	if parseFiber.sibling != nil {
		parseRt.runCleanups(parseFiber.sibling)
	}
}

// RefreshEffectsForFiber forces a fiber subtree's effects to clean up and rerun on the next render.
func (parseRt *Runtime) RefreshEffectsForFiber(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	parseRt.runCleanups(parseFiber)
	bumpEffectEpochs(parseFiber)
}

// bumpEffectEpochs is an internal reconciler helper.
func bumpEffectEpochs(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	if parseFiber.hooks != nil {
		parseFiber.hooks.effectEpoch++
	}
	if parseFiber.child != nil {
		bumpEffectEpochs(parseFiber.child)
	}
	if parseFiber.sibling != nil {
		bumpEffectEpochs(parseFiber.sibling)
	}
}

// runEffects runs all effects for a fiber tree
func (parseRt *Runtime) runEffects(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	// Run this fiber's effects in batch
	parseEffects := parseFiber.effects
	parseEffectCount := len(parseEffects)
	parseFiber.effectDurationNs = 0

	// Unroll for common small effect counts
	if parseEffectCount == 1 {
		parseStart := time.Now()
		parseCleanup := func() func() {
			var parseCleanup2 func()
			var isHandled bool
			func() {
				defer func() {
					if parseRecovered := recover(); parseRecovered != nil {
						if panicPhaseMayRecoverWithBoundary(PanicPhaseEffect) {
							_, isHandled = parseRt.recoverBoundaryError(parseFiber, parseRecovered, boundaryPhaseEffect)
						}
						if !isHandled {
							panic(markUnhandledPanic(parseFiber, boundaryPhaseEffect, parseRecovered))
						}
					}
				}()
				parseCleanup2 = parseEffects[0].Fn()
			}()
			return parseCleanup2
		}()
		parseDurationNs := time.Since(parseStart).Nanoseconds()
		parseFiber.effectDurationNs += parseDurationNs
		parseRt.profiling.effectExecutions++
		parseRt.profiling.lastEffectDurationNs = parseDurationNs
		parseRt.profiling.totalEffectDurationNs += parseDurationNs
		recordSlowOperationDiagnostic("effect", parseFiber, parseDurationNs)
		if parseCleanup != nil {
			parseFiber.hooks.cleanups[parseEffects[0].CleanupIndex] = parseCleanup
		}
	} else {
		for parseI := 0; parseI < parseEffectCount; parseI++ {
			parseEffect := &parseEffects[parseI]
			parseStart2 := time.Now()
			parseCleanup3 := func() func() {
				var parseCleanup4 func()
				var isHandled2 bool
				func() {
					defer func() {
						if parseRecovered2 := recover(); parseRecovered2 != nil {
							if panicPhaseMayRecoverWithBoundary(PanicPhaseEffect) {
								_, isHandled2 = parseRt.recoverBoundaryError(parseFiber, parseRecovered2, boundaryPhaseEffect)
							}
							if !isHandled2 {
								panic(markUnhandledPanic(parseFiber, boundaryPhaseEffect, parseRecovered2))
							}
						}
					}()
					parseCleanup4 = parseEffect.Fn()
				}()
				return parseCleanup4
			}()
			parseDurationNs2 := time.Since(parseStart2).Nanoseconds()
			parseFiber.effectDurationNs += parseDurationNs2
			parseRt.profiling.effectExecutions++
			parseRt.profiling.lastEffectDurationNs = parseDurationNs2
			parseRt.profiling.totalEffectDurationNs += parseDurationNs2
			recordSlowOperationDiagnostic("effect", parseFiber, parseDurationNs2)
			if parseCleanup3 != nil {
				parseFiber.hooks.cleanups[parseEffect.CleanupIndex] = parseCleanup3
			}
		}
	}

	// Recursively run effects for children and siblings
	if parseFiber.child != nil {
		parseRt.runEffects(parseFiber.child)
	}
	if parseFiber.sibling != nil {
		parseRt.runEffects(parseFiber.sibling)
	}
}
