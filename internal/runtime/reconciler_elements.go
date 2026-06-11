package runtime

import (
	"strings"
)

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

// CreateElement creates a new virtual DOM element.
func CreateElement(parseTyp interface{}, parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return buildElement(parseTyp, cloneElementProps(parseProps), parseChildren...)
}

// CreateElementOwned creates a new virtual DOM element and takes ownership of the provided props map.
func CreateElementOwned(parseTyp interface{}, parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return buildElement(parseTyp, parseProps, parseChildren...)
}

// buildElementHostProps creates one host-only props map and optional compact string attrs for one public element payload.
func buildElementHostProps(parseTyp interface{}, parseProps map[string]interface{}) (map[string]interface{}, []HostAttr, bool) {
	parseTag, parseOk := parseTyp.(string)
	if !parseOk || parseTag == "TEXT_ELEMENT" || parseTag == "FRAGMENT" {
		return nil, nil, false
	}
	if len(parseProps) == 0 {
		return nil, nil, true
	}

	// Host fibers alias the element's props map directly instead of building a
	// separate host-only copy: every entry point clones or owns the map before
	// reaching here, and the DOM differ skips propKindSkip entries (including
	// "children"), so the copy only added one map allocation per host element
	// per render — the single largest allocation site in component updates.
	getHostAttrs := make([]HostAttr, 0, len(parseProps))
	isCompactHostProps := true
	for parseName, parseValue := range parseProps {
		if parseName == "children" {
			continue
		}
		if parseName == "key" || parseValue == nil {
			continue
		}
		parseMeta := getPropMeta(parseName)
		if parseMeta.kind == propKindSkip {
			continue
		}
		parseAttrName := parseMeta.attrName
		if parseAttrName == "" {
			parseAttrName = parseName
		}
		switch parseMeta.kind {
		case propKindSpecialProperty:
			isCompactHostProps = false
		case propKindStyle:
			parseTextValue, parseTextOk := parseValue.(string)
			if !parseTextOk {
				isCompactHostProps = false
				continue
			}
			getHostAttrs = append(getHostAttrs, HostAttr{Name: parseAttrName, Value: parseTextValue})
		default:
			parseTextValue, parseTextOk := parseValue.(string)
			if !parseTextOk {
				isCompactHostProps = false
				continue
			}
			getHostAttrs = append(getHostAttrs, HostAttr{Name: parseAttrName, Value: parseTextValue})
		}
	}
	if !isCompactHostProps {
		getHostAttrs = nil
	}
	return parseProps, getHostAttrs, isCompactHostProps
}

// cloneElementProps clones one props map so callers can safely retain and reuse their original input.
func cloneElementProps(parseProps map[string]interface{}) map[string]interface{} {
	if len(parseProps) == 0 {
		return nil
	}

	getProps := make(map[string]interface{}, len(parseProps)+1)
	for parseKey, parseValue := range parseProps {
		getProps[parseKey] = parseValue
	}
	return getProps
}

// canStoreElementDirectText reports whether one host element can carry its only string child directly on the host fiber.
func canStoreElementDirectText(parseTyp interface{}, parseChildren []interface{}) (bool, string) {
	if len(parseChildren) != 1 {
		return false, ""
	}
	parseTag, parseOk := parseTyp.(string)
	if !parseOk || parseTag == "TEXT_ELEMENT" || parseTag == "FRAGMENT" {
		return false, ""
	}
	parseText, hasParseText := parseChildren[0].(string)
	if !hasParseText {
		return false, ""
	}
	return true, parseText
}

// getElementChildren returns one element's structural children while tolerating legacy props-backed child storage.
func getElementChildren(parseElem *Element) []interface{} {
	if parseElem == nil {
		return nil
	}
	if parseElem.hasDirectText {
		if parseElem.Children != nil {
			return parseElem.Children
		}
		return emptyChildren
	}
	if parseElem.Children != nil {
		if len(parseElem.Children) == 0 {
			if parseChildren, parseOk := parseElem.Props["children"].([]interface{}); parseOk && len(parseChildren) > 0 {
				return parseChildren
			}
		}
		return parseElem.Children
	}
	if parseChildren, parseOk := parseElem.Props["children"].([]interface{}); parseOk {
		return parseChildren
	}
	return nil
}

// getFiberChildren returns one fiber's structural child slice while tolerating legacy props-backed child storage.
func getFiberChildren(parseFiber *Fiber) []interface{} {
	if parseFiber == nil {
		return nil
	}
	if parseFiber.hasDirectText {
		if parseFiber.children != nil {
			return parseFiber.children
		}
		return emptyChildren
	}
	if parseFiber.children != nil {
		if len(parseFiber.children) == 0 {
			if parseChildren, parseOk := parseFiber.props["children"].([]interface{}); parseOk && len(parseChildren) > 0 {
				return parseChildren
			}
		}
		return parseFiber.children
	}
	if parseChildren, parseOk := parseFiber.props["children"].([]interface{}); parseOk {
		return parseChildren
	}
	return nil
}

// getElementFiberProps resolves one element's internal working props bag.
func getElementFiberProps(parseElem *Element) map[string]interface{} {
	if parseElem == nil {
		return nil
	}
	if parseElem.getHostProps != nil || parseElem.isCompactHostProps {
		return parseElem.getHostProps
	}
	return parseElem.Props
}

// RefreshElementHostProps rebuilds one element's cached host-prop view after post-creation prop mutation.
func RefreshElementHostProps(parseElem *Element) {
	if parseElem == nil {
		return
	}
	parseElem.getHostProps, parseElem.getHostAttrs, parseElem.isCompactHostProps = buildElementHostProps(parseElem.Type, parseElem.Props)
}

// buildElement builds one virtual DOM element and stores the normalized children slice on the props map.
func buildElement(parseTyp interface{}, parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	if len(parseChildren) == 0 {
		parseChildren = emptyChildren
	}

	getHostProps, getHostAttrs, isCompactHostProps := buildElementHostProps(parseTyp, parseProps)

	if isParseDirectText, parseDirectText := canStoreElementDirectText(parseTyp, parseChildren); isParseDirectText {
		if parseProps == nil {
			parseProps = make(map[string]interface{}, 1)
		}
		parseProps["children"] = parseChildren
		return &Element{
			Type:               parseTyp,
			Props:              parseProps,
			Children:           emptyChildren,
			TextContent:        parseDirectText,
			getHostProps:       getHostProps,
			getHostAttrs:       getHostAttrs,
			isCompactHostProps: isCompactHostProps,
			hasDirectText:      true,
		}
	}

	// Normalize string children once so downstream reconciliation sees only Elements.
	for parseIndex, parseChild := range parseChildren {
		if parseText, hasParseText := parseChild.(string); hasParseText {
			parseChildren[parseIndex] = &Element{
				Type:        "TEXT_ELEMENT",
				TextContent: parseText,
				Children:    emptyChildren,
			}
		}
	}

	if parseProps == nil {
		parseProps = make(map[string]interface{}, 1)
	}
	parseProps["children"] = parseChildren

	return &Element{
		Type:               parseTyp,
		Props:              parseProps,
		Children:           parseChildren,
		getHostProps:       getHostProps,
		getHostAttrs:       getHostAttrs,
		isCompactHostProps: isCompactHostProps,
	}
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
			if parseChildren := getElementChildren(parseElem2); parseChildren != nil {
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
			typeOf:              parseOldFiber.typeOf,
			props:               parseOldFiber.props,
			children:            parseOldFiber.children,
			getHostAttrs:        parseOldFiber.getHostAttrs,
			textContent:         parseOldFiber.textContent,
			dom:                 parseOldFiber.dom,
			parent:              parseParent,
			alternate:           parseOldFiber,
			effectTag:           parseEffectTag,
			dirty:               parseOldFiber.dirty,
			subtreeDirty:        parseOldFiber.subtreeDirty,
			needsUpdate:         parseOldFiber.needsUpdate,
			needsChildReconcile: parseOldFiber.needsChildReconcile,
			hooks:               parseOldFiber.hooks, // Share hooks for non-updated components
			eventCallbacks:      parseOldFiber.eventCallbacks,
			contextValues:       parseOldFiber.contextValues,
			reactiveAtomID:      parseOldFiber.reactiveAtomID,
			reactiveSourceIDs:   parseOldFiber.reactiveSourceIDs,
			fineGrained:         parseOldFiber.fineGrained,
			hasDirectText:       parseOldFiber.hasDirectText,
			isCompactHostProps:  parseOldFiber.isCompactHostProps,
			updateOrigin:        parseOldFiber.updateOrigin,
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

// reuseFiberChildSubtree relinks one committed child chain under the current fiber without cloning descendants.
func (parseRt *Runtime) reuseFiberChildSubtree(parseParent *Fiber) {
	if parseParent == nil || parseParent.alternate == nil {
		return
	}
	parseParent.child = parseParent.alternate.child
	if parseParent.child != nil {
		parseRt.sanitizeFiberSubtree(parseParent.child, parseParent)
	}
}

// sanitizeFiberSubtree relinks one reused committed subtree and clears stale work flags before commit traversal.
func (parseRt *Runtime) sanitizeFiberSubtree(parseFiber *Fiber, parseParent *Fiber) {
	for parseCurrent := parseFiber; parseCurrent != nil; parseCurrent = parseCurrent.sibling {
		parseCurrent.parent = parseParent
		parseCurrent.effectTag = ""
		parseCurrent.dirty = false
		parseCurrent.subtreeDirty = false
		parseCurrent.needsUpdate = false
		parseCurrent.needsChildReconcile = false
		parseCurrent.needsChildOrder = false
		if parseCurrent.hooks != nil {
			parseCurrent.hooks.owner = parseCurrent
		}
		if parseCurrent.child != nil {
			parseRt.sanitizeFiberSubtree(parseCurrent.child, parseCurrent)
		}
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
