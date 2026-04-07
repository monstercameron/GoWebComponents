package runtime

import (
	"strings"
	"time"
)

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
	if IsDOMNodeNull(parseDom) {
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
		if parseRt.shouldRepairCommittedChildOrder(parseRt.wipRoot) {
			parseRt.applyCommittedChildOrder(parseRt.wipRoot.dom, parseRt.buildCommittedChildNodes(parseRt.wipRoot.child, nil))
		}
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
	if parseRt.tracksPendingEffects {
		parseRt.runPendingEffects()
		parseRt.tracksPendingEffects = false
	} else {
		parseRt.runEffects(parseCommittedRoot)
	}

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
	if IsDOMNodeNull(parseDomParent) {
		var parseDomParentFiber *Fiber = parseFiber.parent
		for parseDomParentFiber != nil && IsDOMNodeNull(parseDomParentFiber.dom) {
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

	if !isPortal && !IsDOMNodeNull(parseDomParent) {
		if parseFiber.effectTag == "PLACEMENT" && !IsDOMNodeNull(parseFiber.dom) {
			parseStart := time.Now()
			parseRt.domAdapter.AppendChild(parseDomParent, parseFiber.dom)
			parseFiber.commitDurationNs += time.Since(parseStart).Nanoseconds()
			if parseFiber.fineGrained {
				parseRt.profiling.fineGrainedCommits++
			} else {
				parseRt.recordFineGrainedDescendantCommit(parseFiber)
			}
		} else if parseFiber.effectTag == "HYDRATE" && !IsDOMNodeNull(parseFiber.dom) {
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
		} else if parseFiber.effectTag == "UPDATE" && !IsDOMNodeNull(parseFiber.dom) {
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
					if !propsEqualIgnoringChildren(parseFiber.alternate.props, parseFiber.props) {
						parseStart5 := time.Now()
						parseRt.updateDomProperties(parseFiber.dom, parseFiber.alternate.props, parseFiber.props)
						parseFiber.commitDurationNs += time.Since(parseStart5).Nanoseconds()
						parseRt.recordFineGrainedDescendantCommit(parseFiber)
					}
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
		if IsDOMNodeNull(parsePortalParent) {
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
	if !IsDOMNodeNull(parseFiber.dom) {
		parseChildDomParent = parseFiber.dom
	}
	parseRt.finalizeHydrationBoundary(parseFiber.childHydration, parseFiber)

	// Recursively commit children and siblings
	if parseFiber.child != nil {
		parseBatchAdapter, parseSupportsBatching := parseRt.domAdapter.(interface {
			BeginBatch(DOMNode)
			EndBatch()
		})
		isParseBatching := parseSupportsBatching && parseRt.shouldBatchCommittedPlacements(parseFiber, parseChildDomParent)
		if isParseBatching {
			parseBatchAdapter.BeginBatch(parseChildDomParent)
		}
		parseRt.commitWork(parseFiber.child, parseChildDomParent)
		if isParseBatching {
			parseBatchAdapter.EndBatch()
		}
		if parseRt.shouldRepairCommittedChildOrder(parseFiber) && !IsDOMNodeNull(parseChildDomParent) && (!IsDOMNodeNull(parseFiber.dom) || isPortal) {
			parseRt.applyCommittedChildOrder(parseChildDomParent, parseRt.buildCommittedChildNodes(parseFiber.child, nil))
		}
	}
	if parseFiber.sibling != nil {
		parseRt.commitWork(parseFiber.sibling, parseDomParent)
	}
}

// shouldRepairCommittedChildOrder reports whether one fiber subtree needs a post-commit DOM order repair pass.
func (parseRt *Runtime) shouldRepairCommittedChildOrder(parseFiber *Fiber) bool {
	return parseFiber != nil && parseFiber.needsChildOrder
}

// countCommittedPlacementChildren counts DOM-bearing placements that append directly into the current parent.
func (parseRt *Runtime) countCommittedPlacementChildren(parseFiber *Fiber) int {
	parseCount := 0
	for parseFiber != nil {
		if parseRt.isPortalFiber(parseFiber) {
			parseFiber = parseFiber.sibling
			continue
		}
		if !IsDOMNodeNull(parseFiber.dom) {
			if parseFiber.effectTag == "PLACEMENT" {
				parseCount++
			}
			parseFiber = parseFiber.sibling
			continue
		}
		if parseFiber.child != nil {
			parseCount += parseRt.countCommittedPlacementChildren(parseFiber.child)
		}
		parseFiber = parseFiber.sibling
	}
	return parseCount
}

// shouldBatchCommittedPlacements reports whether one child subtree can safely batch appends into one parent.
func (parseRt *Runtime) shouldBatchCommittedPlacements(parseFiber *Fiber, parseDomParent DOMNode) bool {
	if parseRt == nil || parseRt.domAdapter == nil || parseFiber == nil || IsDOMNodeNull(parseDomParent) {
		return false
	}
	if parseRt.shouldRepairCommittedChildOrder(parseFiber) {
		return false
	}
	if _, parseOk := parseRt.domAdapter.(interface {
		BeginBatch(DOMNode)
		EndBatch()
	}); !parseOk {
		return false
	}
	return parseRt.countCommittedPlacementChildren(parseFiber.child) >= 2
}

// buildCommittedChildNodes is an internal reconciler helper.
func (parseRt *Runtime) buildCommittedChildNodes(parseFiber *Fiber, parseNodes []DOMNode) []DOMNode {
	for parseFiber != nil {
		if parseRt.isPortalFiber(parseFiber) {
			parseFiber = parseFiber.sibling
			continue
		}
		if !IsDOMNodeNull(parseFiber.dom) {
			parseNodes = append(parseNodes, parseFiber.dom)
		} else if parseFiber.child != nil {
			parseNodes = parseRt.buildCommittedChildNodes(parseFiber.child, parseNodes)
		}
		parseFiber = parseFiber.sibling
	}
	return parseNodes
}

// buildObservedChildNodes is an internal reconciler helper.
func (parseRt *Runtime) buildObservedChildNodes(parseDomParent DOMNode) []DOMNode {
	if parseRt == nil || parseRt.domAdapter == nil || IsDOMNodeNull(parseDomParent) {
		return nil
	}
	parseNodes := make([]DOMNode, 0, 4)
	for parseNode := parseRt.domAdapter.GetFirstChild(parseDomParent); !IsDOMNodeNull(parseNode); parseNode = parseRt.domAdapter.GetNextSibling(parseNode) {
		parseNodes = append(parseNodes, parseNode)
	}
	return parseNodes
}

// isCommittedChildOrderStable is an internal reconciler helper.
func isCommittedChildOrderStable(parseExpected, parseObserved []DOMNode) bool {
	if len(parseExpected) != len(parseObserved) {
		return false
	}
	for parseIndex, parseExpectedNode := range parseExpected {
		if !IsSameDOMNode(parseExpectedNode, parseObserved[parseIndex]) {
			return false
		}
	}
	return true
}

// applyCommittedChildOrder is an internal reconciler helper.
func (parseRt *Runtime) applyCommittedChildOrder(parseDomParent DOMNode, parseExpected []DOMNode) {
	if parseRt == nil || parseRt.domAdapter == nil || IsDOMNodeNull(parseDomParent) || len(parseExpected) == 0 {
		return
	}
	parseObserved := parseRt.buildObservedChildNodes(parseDomParent)
	if isCommittedChildOrderStable(parseExpected, parseObserved) {
		return
	}

	for parseIndex, parseExpectedNode := range parseExpected {
		parseObserved = parseRt.buildObservedChildNodes(parseDomParent)
		if parseIndex < len(parseObserved) && IsSameDOMNode(parseObserved[parseIndex], parseExpectedNode) {
			continue
		}

		isParseAttached := false
		for _, parseObservedNode := range parseObserved {
			if IsSameDOMNode(parseObservedNode, parseExpectedNode) {
				isParseAttached = true
				break
			}
		}

		// Remove first so the test adapters and browser adapters both model a DOM move as one relocation, not one duplicate append.
		if isParseAttached {
			parseRt.domAdapter.RemoveChild(parseDomParent, parseExpectedNode)
			parseObserved = parseRt.buildObservedChildNodes(parseDomParent)
		}

		if parseIndex < len(parseObserved) {
			parseRt.domAdapter.InsertBefore(parseDomParent, parseExpectedNode, parseObserved[parseIndex])
			continue
		}
		parseRt.domAdapter.AppendChild(parseDomParent, parseExpectedNode)
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

	if !IsDOMNodeNull(parseFiber.dom) {
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
	if !IsDOMNodeNull(parseFiber.dom) {
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
	if parseFiber == nil || IsDOMNodeNull(parseOldParent) || IsDOMNodeNull(parseNewParent) || IsSameDOMNode(parseOldParent, parseNewParent) {
		return
	}

	if parseRt.isPortalFiber(parseFiber) {
		return
	}

	if !IsDOMNodeNull(parseFiber.dom) {
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

// queuePendingEffectFiber records one fiber that scheduled post-commit effects during the current render pass.
func (parseRt *Runtime) queuePendingEffectFiber(parseFiber *Fiber) {
	if parseRt == nil || parseFiber == nil || len(parseFiber.effects) == 0 {
		return
	}
	parseRt.pendingEffectFibers = append(parseRt.pendingEffectFibers, parseFiber)
}

// runPendingEffects runs only the fibers that scheduled effects during the current render pass.
func (parseRt *Runtime) runPendingEffects() {
	if parseRt == nil || len(parseRt.pendingEffectFibers) == 0 {
		return
	}
	for _, parseFiber := range parseRt.pendingEffectFibers {
		parseRt.runFiberEffects(parseFiber)
	}
	parseRt.pendingEffectFibers = parseRt.pendingEffectFibers[:0]
}

// runFiberEffects runs one fiber's queued effects without traversing descendants.
func (parseRt *Runtime) runFiberEffects(parseFiber *Fiber) {
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
}

// runEffects runs all effects for a fiber tree
func (parseRt *Runtime) runEffects(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	parseRt.runFiberEffects(parseFiber)

	// Recursively run effects for children and siblings
	if parseFiber.child != nil {
		parseRt.runEffects(parseFiber.child)
	}
	if parseFiber.sibling != nil {
		parseRt.runEffects(parseFiber.sibling)
	}
}
