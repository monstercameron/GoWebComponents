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

// buildElement builds one virtual DOM element and stores the normalized children slice on the props map.
func buildElement(parseTyp interface{}, parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	if len(parseChildren) == 0 {
		parseChildren = emptyChildren
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
		Type:     parseTyp,
		Props:    parseProps,
		Children: parseChildren,
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
