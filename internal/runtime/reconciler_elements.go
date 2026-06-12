package runtime

import (
	"maps"
	"strconv"
	"strings"
)

func GetCurrentFiber() *Fiber {
	return currentFiber
}

// SetCurrentFiber sets the current fiber (used during component rendering)
func SetCurrentFiber(parseFiber *Fiber) {
	currentFiber = parseFiber
	if parseFiber == nil || !hookThreadingGuardEnabled {
		currentFiberOwnerGoroutineID = 0
		return
	}
	currentFiberOwnerGoroutineID = currentHookGoroutineID()
}

// runtimeForFiber returns the owning runtime for a fiber subtree, falling back
// to the global runtime for legacy tests that install a bare current fiber.
func runtimeForFiber(parseFiber *Fiber) *Runtime {
	for parseCursor := parseFiber; parseCursor != nil; parseCursor = parseCursor.parent {
		if parseCursor.ownerRuntime != nil {
			return parseCursor.ownerRuntime
		}
	}
	if parseFiber != nil && parseFiber.alternate != nil {
		for parseCursor := parseFiber.alternate; parseCursor != nil; parseCursor = parseCursor.parent {
			if parseCursor.ownerRuntime != nil {
				return parseCursor.ownerRuntime
			}
		}
	}
	return GetGlobalRuntime()
}

// requireCurrentHookFiber returns the current render fiber or panics with the
// hook-specific development diagnostic.
func requireCurrentHookFiber(parseName string) *Fiber {
	parseFiber := GetCurrentFiber()
	if parseFiber == nil {
		panic(actionableHookUsagePanic(parseName))
	}
	if !isCurrentHookGoroutineOwner() {
		reportHookThreadingViolation(parseName, parseFiber)
		panic(actionableHookThreadingPanic(parseName, parseFiber))
	}
	return parseFiber
}

// isCurrentHookGoroutineOwner reports whether a hook call is running on the
// goroutine that claimed the current render fiber.
func isCurrentHookGoroutineOwner() bool {
	if !hookThreadingGuardEnabled || currentFiberOwnerGoroutineID == 0 {
		return true
	}
	parseCurrentID := currentHookGoroutineID()
	return parseCurrentID == 0 || parseCurrentID == currentFiberOwnerGoroutineID
}

// reportHookThreadingViolation records a structured diagnostic before the hook
// panic is raised. The panic report includes its own stack detail.
func reportHookThreadingViolation(parseName string, parseFiber *Fiber) {
	parseFields := map[string]string{
		"hook":            strings.TrimSpace(parseName),
		"renderGoroutine": strconv.FormatUint(currentFiberOwnerGoroutineID, 10),
		"callGoroutine":   strconv.FormatUint(currentHookGoroutineID(), 10),
	}
	reportDiagnosticWithContextDetails(
		"runtime",
		DiagnosticError,
		hookThreadingViolationMessage(parseName),
		diagnosticPathForFiber(parseFiber),
		diagnosticComponentStack(parseFiber),
		"",
		"hook state is render-goroutine-owned; continuing would corrupt component state",
		parseFields,
	)
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
func CreateElement(parseTyp any, parseProps map[string]any, parseChildren ...any) *Element {
	return buildElement(parseTyp, cloneElementProps(parseProps), parseChildren...)
}

// CreateElementOwned creates a new virtual DOM element and takes ownership of the provided props map.
func CreateElementOwned(parseTyp any, parseProps map[string]any, parseChildren ...any) *Element {
	return buildElement(parseTyp, parseProps, parseChildren...)
}

// CreateElementCompactHostOwned creates one host element from an owned props map
// and a caller-normalized compact string-attribute view.
func CreateElementCompactHostOwned(parseTag string, parseProps map[string]any, parseAttrs []HostAttr, parseChildren ...any) *Element {
	if parseTag == "TEXT_ELEMENT" || parseTag == "FRAGMENT" {
		return buildElement(parseTag, parseProps, parseChildren...)
	}
	return buildElementWithHostProps(parseTag, parseProps, parseProps, parseAttrs, true, parseChildren...)
}

// buildElementHostProps creates one host-only props map and optional compact string attrs for one public element payload.
func buildElementHostProps(parseTyp any, parseProps map[string]any) (map[string]any, []HostAttr, bool) {
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
func cloneElementProps(parseProps map[string]any) map[string]any {
	if len(parseProps) == 0 {
		return nil
	}

	getProps := make(map[string]any, len(parseProps)+1)
	maps.Copy(getProps, parseProps)
	return getProps
}

// canStoreElementDirectText reports whether one host element can carry its only string child directly on the host fiber.
func canStoreElementDirectText(parseTyp any, parseChildren []any) (bool, string) {
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
func getElementChildren(parseElem *Element) []any {
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
			if parseChildren, parseOk := parseElem.Props["children"].([]any); parseOk && len(parseChildren) > 0 {
				return parseChildren
			}
		}
		return parseElem.Children
	}
	if parseChildren, parseOk := parseElem.Props["children"].([]any); parseOk {
		return parseChildren
	}
	return nil
}

// getFiberChildren returns one fiber's structural child slice while tolerating legacy props-backed child storage.
func getFiberChildren(parseFiber *Fiber) []any {
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
			if parseChildren, parseOk := parseFiber.props["children"].([]any); parseOk && len(parseChildren) > 0 {
				return parseChildren
			}
		}
		return parseFiber.children
	}
	if parseChildren, parseOk := parseFiber.props["children"].([]any); parseOk {
		return parseChildren
	}
	return nil
}

// getElementFiberProps resolves one element's internal working props bag.
func getElementFiberProps(parseElem *Element) map[string]any {
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
func buildElement(parseTyp any, parseProps map[string]any, parseChildren ...any) *Element {
	getHostProps, getHostAttrs, isCompactHostProps := buildElementHostProps(parseTyp, parseProps)
	return buildElementWithHostProps(parseTyp, parseProps, getHostProps, getHostAttrs, isCompactHostProps, parseChildren...)
}

// buildElementWithHostProps builds one virtual DOM element from an already-normalized host-prop view.
func buildElementWithHostProps(parseTyp any, parseProps map[string]any, getHostProps map[string]any, getHostAttrs []HostAttr, isCompactHostProps bool, parseChildren ...any) *Element {
	if len(parseChildren) == 0 {
		parseChildren = emptyChildren
	}

	if isParseDirectText, parseDirectText := canStoreElementDirectText(parseTyp, parseChildren); isParseDirectText {
		if parseProps == nil {
			parseProps = make(map[string]any, 1)
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

	// Normalize string children once so downstream reconciliation sees only
	// Elements.  All text elements for one parent share a single backing array
	// so N string children cost one allocation instead of N.
	parseTextCount := 0
	for _, parseChild := range parseChildren {
		if _, hasParseText := parseChild.(string); hasParseText {
			parseTextCount++
		}
	}
	if parseTextCount > 0 {
		parseTextElems := make([]Element, parseTextCount)
		parseTextIdx := 0
		for parseIndex, parseChild := range parseChildren {
			if parseText, hasParseText := parseChild.(string); hasParseText {
				parseTextElems[parseTextIdx] = Element{
					Type:        "TEXT_ELEMENT",
					TextContent: parseText,
					Children:    emptyChildren,
				}
				parseChildren[parseIndex] = &parseTextElems[parseTextIdx]
				parseTextIdx++
			}
		}
	}

	if parseProps == nil {
		parseProps = make(map[string]any, 1)
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
func flattenFragments(parseElements []any) ([]any, bool) {
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
		parseEffectTag := effectTagNone
		if parseOldFiber.dirty || parseOldFiber.needsUpdate {
			parseEffectTag = effectTagUpdate
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
			ownerRuntime:        parseRt,
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
		parseCurrent.effectTag = effectTagNone
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
