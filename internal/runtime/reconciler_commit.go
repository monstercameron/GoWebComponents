package runtime

import (
	"slices"
	"strings"
)

const getCommittedChildReplaceThreshold = 8

type preparedHostDOMAdapter interface {
	CreatePreparedElement(tag string, attrs []HostAttr, text string) DOMNode
}

type passiveEventDOMAdapter interface {
	AddPassiveEventListener(node DOMNode, eventType string, handler any)
	RemovePassiveEventListener(node DOMNode, eventType string, handler any)
}

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
			if parseFiber.isCompactHostProps {
				if parsePreparedAdapter, parsePreparedOk := parseRt.domAdapter.(preparedHostDOMAdapter); parsePreparedOk {
					parsePreparedText := ""
					if parseFiber.hasDirectText {
						parsePreparedText = parseFiber.textContent
					}
					parseDom = parsePreparedAdapter.CreatePreparedElement(parseT, parseFiber.getHostAttrs, parsePreparedText)
				}
			}
			if IsDOMNodeNull(parseDom) {
				// Regular element (not TEXT_ELEMENT or FRAGMENT)
				parseDom = parseRt.domAdapter.CreateElement(parseT)
				if parseFiber.isCompactHostProps {
					parseRt.applyCompactHostAttrs(parseDom, parseFiber.getHostAttrs)
				} else {
					// Apply properties only for non-text elements
					parseRt.updateDomProperties(parseDom, nil, parseFiber.props)
				}
				if parseFiber.hasDirectText {
					parseRt.domAdapter.SetTextContent(parseDom, parseFiber.textContent)
				}
			}
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

// applyCompactHostAttrs applies one pre-normalized string-attr slice without the generic prop meta walk.
func (parseRt *Runtime) applyCompactHostAttrs(parseDom DOMNode, parseAttrs []HostAttr) {
	if parseRt == nil || IsDOMNodeNull(parseDom) || len(parseAttrs) == 0 {
		return
	}
	if parseBatchAdapter, parseBatchOk := parseRt.domAdapter.(interface {
		BatchSetAttributes(DOMNode, map[string]string)
	}); parseBatchOk && len(parseAttrs) >= 3 {
		parseAttrBatch := make(map[string]string, len(parseAttrs))
		for _, parseAttr := range parseAttrs {
			parseAttrBatch[parseAttr.Name] = parseAttr.Value
		}
		parseBatchAdapter.BatchSetAttributes(parseDom, parseAttrBatch)
		return
	}
	for _, parseAttr := range parseAttrs {
		parseRt.domAdapter.SetAttribute(parseDom, parseAttr.Name, parseAttr.Value)
	}
}

// updateCompactHostAttrs diffs two deterministic compact attribute slices and
// applies only the changed attributes; it reports whether any DOM write
// happened. Slices are tiny (a handful of attributes), so the containment
// scans are linear rather than map-backed.
func (parseRt *Runtime) updateCompactHostAttrs(parseDom DOMNode, parseOldAttrs, parseNewAttrs []HostAttr) bool {
	if hostAttrsEqual(parseOldAttrs, parseNewAttrs) {
		return false
	}
	isChanged := false
	for _, parseNew := range parseNewAttrs {
		parseOldValue, hasOld := lookupHostAttr(parseOldAttrs, parseNew.Name)
		if !hasOld || parseOldValue != parseNew.Value {
			parseRt.domAdapter.SetAttribute(parseDom, parseNew.Name, parseNew.Value)
			isChanged = true
		}
	}
	for _, parseOld := range parseOldAttrs {
		if _, hasNew := lookupHostAttr(parseNewAttrs, parseOld.Name); !hasNew {
			parseRt.domAdapter.RemoveAttribute(parseDom, parseOld.Name)
			isChanged = true
		}
	}
	return isChanged
}

// lookupHostAttr finds one attribute by name in a compact attribute slice.
func lookupHostAttr(parseAttrs []HostAttr, parseName string) (string, bool) {
	for _, parseAttr := range parseAttrs {
		if parseAttr.Name == parseName {
			return parseAttr.Value, true
		}
	}
	return "", false
}

// updateDomProperties updates DOM properties with optimized batching when available
func (parseRt *Runtime) updateDomProperties(parseDom DOMNode, parseOldProps, parseNewProps map[string]any) {
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
			if parseHandler, parseOk := passiveEventHandler(parseOldProps[parseName]); parseOk {
				parseRt.removePassiveEventHandler(parseDom, parseTargetName, parseHandler)
				continue
			}
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
		parseHadOldNonPassiveHandler := false
		if parseOldValue, parseExists3 := parseOldProps[parseName2]; parseExists3 {
			if parseOldHandler, parseOldPassive := passiveEventHandler(parseOldValue); parseOldPassive {
				parseRt.removePassiveEventHandler(parseDom, parseTargetName2, parseOldHandler)
			} else if _, parseEventOk := eventTypeFromProp(parseTargetName2); parseEventOk {
				parseHadOldNonPassiveHandler = true
			}
		}
		if parseHandler, parseOk := passiveEventHandler(parseValue); parseOk {
			if parseHadOldNonPassiveHandler {
				parseRt.domAdapter.SetProperty(parseDom, parseTargetName2, nil)
			}
			parseRt.addPassiveEventHandler(parseDom, parseTargetName2, parseHandler)
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
func (parseRt *Runtime) applyInitialDomProps(parseDom DOMNode, parseNewProps map[string]any, isSupportsBatching bool, parseBatchAdapter interface {
	BatchSetAttributes(DOMNode, map[string]string)
}, isPreserveHydrationState bool) {
	var (
		parseAttrBatch                            map[string]string
		parseFirstAttrName, parseFirstAttrValue   string
		parseSecondAttrName, parseSecondAttrValue string
		parseAttrCount                            int
	)
	parseFlushAttrBatch := func() {
		if parseAttrCount == 0 {
			return
		}
		if parseAttrBatch != nil {
			parseBatchAdapter.BatchSetAttributes(parseDom, parseAttrBatch)
			clear(parseAttrBatch)
			parseAttrBatch = nil
		} else {
			parseRt.domAdapter.SetAttribute(parseDom, parseFirstAttrName, parseFirstAttrValue)
			if parseAttrCount > 1 {
				parseRt.domAdapter.SetAttribute(parseDom, parseSecondAttrName, parseSecondAttrValue)
			}
		}
		parseFirstAttrName = ""
		parseFirstAttrValue = ""
		parseSecondAttrName = ""
		parseSecondAttrValue = ""
		parseAttrCount = 0
	}
	parseStoreAttr := func(parseName string, parseValue string) {
		if !isSupportsBatching {
			parseRt.domAdapter.SetAttribute(parseDom, parseName, parseValue)
			return
		}

		// Avoid allocating the batch map for the common one- and two-attribute mount paths.
		switch parseAttrCount {
		case 0:
			parseFirstAttrName = parseName
			parseFirstAttrValue = parseValue
		case 1:
			parseSecondAttrName = parseName
			parseSecondAttrValue = parseValue
		case 2:
			parseAttrBatch = make(map[string]string, len(parseNewProps))
			parseAttrBatch[parseFirstAttrName] = parseFirstAttrValue
			parseAttrBatch[parseSecondAttrName] = parseSecondAttrValue
			parseAttrBatch[parseName] = parseValue
		default:
			parseAttrBatch[parseName] = parseValue
		}
		parseAttrCount++
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
		if parseHandler, parseOk := passiveEventHandler(parseValue); parseOk {
			parseFlushAttrBatch()
			parseRt.addPassiveEventHandler(parseDom, parseTargetName, parseHandler)
			continue
		}

		switch parseMeta.kind {
		case propKindStyle:
			if parseStyles, parseOk := parseValue.(map[string]string); parseOk {
				parseFlushAttrBatch()
				parseRt.domAdapter.SetStyles(parseDom, parseStyles)
			} else if parseStr, parseOk2 := parseValue.(string); parseOk2 {
				parseStoreAttr(parseMeta.attrName, parseStr)
			}
		case propKindClass:
			if parseStr2, parseOk3 := parseValue.(string); parseOk3 {
				parseStoreAttr(parseMeta.attrName, parseStr2)
			}
		case propKindSpecialProperty:
			parseFlushAttrBatch()
			parseRt.domAdapter.SetProperty(parseDom, parseTargetName, parseValue)
		default:
			if parseStr3, parseOk4 := parseValue.(string); parseOk4 {
				parseStoreAttr(parseMeta.attrName, parseStr3)
			} else {
				parseFlushAttrBatch()
				parseRt.domAdapter.SetProperty(parseDom, parseTargetName, parseValue)
			}
		}
	}

	parseFlushAttrBatch()
}

func passiveEventHandler(parseValue any) (any, bool) {
	parsePassive, parseOk := parseValue.(PassiveEventHandler)
	if !parseOk || parsePassive.Handler == nil {
		return nil, false
	}
	return parsePassive.Handler, true
}

func eventTypeFromProp(parseName string) (string, bool) {
	if len(parseName) <= 2 || !strings.HasPrefix(parseName, "on") {
		return "", false
	}
	return parseName[2:], true
}

func (parseRt *Runtime) addPassiveEventHandler(parseDom DOMNode, parseName string, parseHandler any) {
	parseAdapter, parseOk := parseRt.domAdapter.(passiveEventDOMAdapter)
	parseEventType, parseEventOk := eventTypeFromProp(parseName)
	if !parseOk || !parseEventOk {
		parseRt.domAdapter.SetProperty(parseDom, parseName, parseHandler)
		return
	}
	parseAdapter.AddPassiveEventListener(parseDom, parseEventType, parseHandler)
}

func (parseRt *Runtime) removePassiveEventHandler(parseDom DOMNode, parseName string, parseHandler any) {
	parseAdapter, parseOk := parseRt.domAdapter.(passiveEventDOMAdapter)
	parseEventType, parseEventOk := eventTypeFromProp(parseName)
	if !parseOk || !parseEventOk {
		parseRt.domAdapter.SetProperty(parseDom, parseName, nil)
		return
	}
	parseAdapter.RemovePassiveEventListener(parseDom, parseEventType, parseHandler)
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

// attrUpdateBatchDOMAdapter is the optional capability for buffering a
// commit's attribute writes into one bridge call (browser adapter only).
// While a batch is open, attribute reads through the adapter may observe
// stale values — commitRoot flushes before anything that reads the DOM runs.
type attrUpdateBatchDOMAdapter interface {
	BeginAttrUpdateBatch()
	EndAttrUpdateBatch()
}

// commitRoot commits all changes to the DOM
func (parseRt *Runtime) commitRoot() {
	parseStart := commitTimingStart()
	defer func() {
		parseDurationNs := commitTimingSinceNs(parseStart)
		parseRt.profiling.commitCount++
		parseRt.AdvanceAgentStateVersion()
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
	// Cross-node attribute batching: buffer every attribute write in the
	// commit walk into one bridge call (one setAttribute hop per attribute
	// otherwise dominates attribute-heavy commits). Nothing in the walk
	// reads attributes back; the deferred End covers panic unwinds and the
	// explicit flush below runs before effects, which may read the DOM.
	parseAttrBatchAdapter, hasAttrBatch := parseRt.domAdapter.(attrUpdateBatchDOMAdapter)
	if hasAttrBatch {
		parseAttrBatchAdapter.BeginAttrUpdateBatch()
		defer parseAttrBatchAdapter.EndAttrUpdateBatch()
	}

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
		parseBatchAdapter, parseSupportsBatching := parseRt.domAdapter.(interface {
			BeginBatch(DOMNode)
			EndBatch()
		})
		isParseBatching := parseSupportsBatching && parseRt.shouldBatchCommittedPlacements(parseRt.wipRoot, parseRt.wipRoot.dom)
		if isParseBatching {
			parseBatchAdapter.BeginBatch(parseRt.wipRoot.dom)
		}
		parseRt.commitWork(parseRt.wipRoot.child, parseRt.wipRoot.dom)
		if isParseBatching {
			parseBatchAdapter.EndBatch()
		}
		if parseRt.shouldRepairCommittedChildOrder(parseRt.wipRoot) {
			parseRt.applyCommittedChildOrder(parseRt.wipRoot.dom, parseRt.buildCommittedChildNodes(parseRt.wipRoot.child, nil))
		}
	}

	// Flush buffered attribute writes before effects and subscribers run —
	// they may read the DOM. The deferred End above then no-ops.
	if hasAttrBatch {
		parseAttrBatchAdapter.EndAttrUpdateBatch()
	}

	parseRt.currentRoot = parseCommittedRoot
	parseRt.schedulerState.finishScheduledLocked()
	parseRt.wipRoot = nil
	parseWasHydrating := parseRt.hydrating
	if parseWasHydrating {
		parseRt.hydrating = false
		parseRt.strictHydration = false
		parseRt.flushHydrationSubscriptions()
	}
	parseRt.updateScheduled = false

	// Run effects after the committed tree is current and hydration gates are lifted.
	if parseRt.tracksPendingEffects && !parseRt.pendingEffectOverflow {
		parseRt.runPendingEffects()
	} else {
		parseRt.runEffects(parseCommittedRoot)
	}
	parseRt.tracksPendingEffects = false
	parseRt.pendingEffectOverflow = false

	if parseWasHydrating {
		parseRt.finishHydrationMetrics(false, "")
		parseRt.flushDeferredHydrationUpdates()
	}
	if parseRt.pendingBoundaryRecovery {
		parseRt.pendingBoundaryRecovery = false
		parseRt.ScheduleUpdate()
	}

	// G16: signal "app ready" once the first commit (initial paint + effects) is
	// done. No-op on every later commit.
	fireFirstCommitHooks()
}

// reportMissingKeys is an internal reconciler helper. It exists only to emit
// the missing-key development warning, but it walks the ENTIRE child slice on
// every reconcileChildren call — an unamortized O(N) pass per list per render
// (the keyed-dispatch check it duplicates short-circuits on the first keyed
// element). Gated behind the same dev/production split as the commit timers
// and the hook threading guard.
func reportMissingKeys(parseParent *Fiber, parseElements []any) {
	if !hookThreadingGuardEnabled {
		// production build: diagnostics stripped, skip the O(N) scan.
		return
	}
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
		if parseFiber.effectTag == effectTagPlacement {
			parseStart := commitTimingStart()
			if IsDOMNodeNull(parseFiber.dom) && !parseRt.tryCommitSerializedSubtree(parseFiber) {
				parseFiber.dom = parseRt.createDom(parseFiber)
			}
			if !IsDOMNodeNull(parseFiber.dom) {
				parseRt.domAdapter.AppendChild(parseDomParent, parseFiber.dom)
			}
			// G2: publish the freshly-mounted node into any DOM ref carried in props.
			parseRt.publishDOMRef(parseFiber, parseFiber.dom)
			parseFiber.commitDurationNs += commitTimingSinceNs(parseStart)
			if parseFiber.fineGrained {
				parseRt.profiling.fineGrainedCommits++
			} else {
				parseRt.recordFineGrainedDescendantCommit(parseFiber)
			}
		} else if parseFiber.effectTag == effectTagHydrate && !IsDOMNodeNull(parseFiber.dom) {
			if isTextLikeFiber(parseFiber) {
				parseNewValue := parseFiber.textContent
				if _, parseOk := parseFiber.typeOf.(*ReactiveTextElementType); parseOk {
					parseNewValue = reactiveTextValue(parseFiber)
				} else if parseNewValue == "" && parseFiber.props != nil {
					parseNewValue, _ = parseFiber.props["nodeValue"].(string)
				}
				parseStart2 := commitTimingStart()
				parseRt.domAdapter.SetTextContent(parseFiber.dom, parseNewValue)
				parseFiber.commitDurationNs += commitTimingSinceNs(parseStart2)
				if parseFiber.fineGrained {
					parseRt.profiling.fineGrainedCommits++
				} else {
					parseRt.recordFineGrainedDescendantCommit(parseFiber)
				}
			} else {
				parseStart3 := commitTimingStart()
				if isFastLaneCompactFiber(parseFiber) {
					// Typed fast-lane fibers carry no events or controlled
					// values; adopting a hydrated node only needs its
					// attributes reasserted.
					parseRt.applyCompactHostAttrs(parseFiber.dom, parseFiber.getHostAttrs)
				} else {
					parseBatchAdapter, parseSupportsBatching := parseRt.domAdapter.(interface {
						BatchSetAttributes(DOMNode, map[string]string)
					})
					parseRt.applyInitialDomProps(parseFiber.dom, parseFiber.props, parseSupportsBatching, parseBatchAdapter, parseFiber.hydrated)
				}
				if parseFiber.hasDirectText && parseRt.domNodeText(parseFiber.dom) != parseFiber.textContent {
					parseRt.domAdapter.SetTextContent(parseFiber.dom, parseFiber.textContent)
				}
				parseFiber.commitDurationNs += commitTimingSinceNs(parseStart3)
				parseRt.recordFineGrainedDescendantCommit(parseFiber)
			}
		} else if parseFiber.effectTag == effectTagUpdate && !IsDOMNodeNull(parseFiber.dom) {
			if parseFiber.alternate != nil {
				if isTextLikeFiber(parseFiber) {
					// Update text content
					parseOldValue := textLikeFiberValue(parseFiber.alternate)
					parseNewValue2 := textLikeFiberValue(parseFiber)

					if parseOldValue != parseNewValue2 {
						parseStart4 := commitTimingStart()
						parseRt.domAdapter.SetTextContent(parseFiber.dom, parseNewValue2)
						parseFiber.commitDurationNs += commitTimingSinceNs(parseStart4)
						if parseFiber.fineGrained {
							parseRt.profiling.fineGrainedCommits++
						} else {
							parseRt.recordFineGrainedDescendantCommit(parseFiber)
						}
					}
				} else {
					// Regular element - update properties
					isParseCommitted := false
					parseStart5 := commitTimingStart()
					if parseFiber.alternate.hasDirectText != parseFiber.hasDirectText || (parseFiber.hasDirectText && parseFiber.alternate.textContent != parseFiber.textContent) {
						if parseFiber.hasDirectText {
							parseRt.domAdapter.SetTextContent(parseFiber.dom, parseFiber.textContent)
						} else {
							parseRt.domAdapter.SetTextContent(parseFiber.dom, "")
						}
						isParseCommitted = true
					}
					if isFastLaneCompactFiber(parseFiber) && isFastLaneCompactFiber(parseFiber.alternate) {
						// Typed fast lane: diff the deterministic attribute
						// slices directly and touch only changed attributes.
						if parseRt.updateCompactHostAttrs(parseFiber.dom, parseFiber.alternate.getHostAttrs, parseFiber.getHostAttrs) {
							isParseCommitted = true
						}
					} else if parseOldView, parseNewView := fiberPropsView(parseFiber.alternate), fiberPropsView(parseFiber); !propsEqualIgnoringChildren(parseOldView, parseNewView) {
						parseRt.updateDomProperties(parseFiber.dom, parseOldView, parseNewView)
						isParseCommitted = true
					}
					if isParseCommitted {
						parseFiber.commitDurationNs += commitTimingSinceNs(parseStart5)
						parseRt.recordFineGrainedDescendantCommit(parseFiber)
					}
				}
			}
		} else if parseFiber.effectTag == effectTagDeletion {
			// fmt.Printf("DEBUG: Committing deletion for %v\n", fiber.typeOf)
			parseRt.commitDeletion(parseFiber, parseDomParent)
			return
		}
	}

	if isPortal {
		if parseFiber.effectTag == effectTagDeletion {
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
			// Remember the unresolved target so the next commit retries the
			// placement once the target element exists, instead of leaving
			// the portal permanently empty.
			parseFiber.portalUnresolved = true
			if parseFiber.sibling != nil {
				parseRt.commitWork(parseFiber.sibling, parseDomParent)
			}
			return
		}
		if parseFiber.alternate != nil && parseFiber.alternate.portalUnresolved {
			// The target failed to resolve on a previous commit; the subtree
			// was never created, so force it through the placement path now.
			markPortalSubtreeForPlacement(parseFiber.child)
		}
		parseFiber.portalUnresolved = false
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
			// Multiple placements are landing under this parent: pre-mount
			// runs of eligible siblings from one combined parse before the
			// per-child walk creates them one template at a time.
			parseRt.prepareSerializedSiblingRuns(parseFiber)
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

// hasCommittedPlacementChildrenAtLeast reports whether one subtree contains at least the requested number of DOM-bearing placements.
func (parseRt *Runtime) hasCommittedPlacementChildrenAtLeast(parseFiber *Fiber, parseLimit int) bool {
	if parseRt == nil || parseLimit <= 0 {
		return true
	}
	return parseRt.countCommittedPlacementChildrenUntil(parseFiber, parseLimit) >= parseLimit
}

// countCommittedPlacementChildrenUntil counts DOM-bearing placements until the requested threshold is satisfied.
func (parseRt *Runtime) countCommittedPlacementChildrenUntil(parseFiber *Fiber, parseLimit int) int {
	if parseRt == nil || parseLimit <= 0 {
		return 0
	}
	parseCount := 0
	for parseFiber != nil {
		if parseRt.isPortalFiber(parseFiber) {
			parseFiber = parseFiber.sibling
			continue
		}
		if !IsDOMNodeNull(parseFiber.dom) || isDeferredCommitHostFiber(parseFiber) {
			if parseFiber.effectTag == effectTagPlacement {
				parseCount++
				if parseCount >= parseLimit {
					return parseCount
				}
			}
			parseFiber = parseFiber.sibling
			continue
		}
		if parseFiber.child != nil {
			parseCount += parseRt.countCommittedPlacementChildrenUntil(parseFiber.child, parseLimit-parseCount)
			if parseCount >= parseLimit {
				return parseCount
			}
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
	if _, parseOk := parseRt.domAdapter.(interface {
		BeginBatch(DOMNode)
		EndBatch()
	}); !parseOk {
		return false
	}
	return parseRt.hasCommittedPlacementChildrenAtLeast(parseFiber.child, 2)
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

	// Same node set, order differs: keep the longest run of children already
	// in relative order and move only the rest. A single displaced row in a
	// long list costs one insertBefore instead of a wholesale rebuild; a
	// heavily permuted long list (moves > half) still collapses into one
	// replaceChildren call when the adapter supports it.
	if parseMatch, hasSameSet := buildCommittedChildOrderMatch(parseExpected, parseObserved); hasSameSet {
		parseKept, parseKeptCount := markCommittedChildOrderKept(parseMatch, len(parseExpected))
		parseMoveCount := len(parseExpected) - parseKeptCount
		if parseReplaceAdapter, parseOk := parseRt.domAdapter.(interface {
			ReplaceChildren(DOMNode, []DOMNode)
		}); parseOk && len(parseExpected) >= getCommittedChildReplaceThreshold && parseMoveCount*2 > len(parseExpected) {
			parseReplaceAdapter.ReplaceChildren(parseDomParent, parseExpected)
			return
		}
		var parseAnchor DOMNode
		for parseIndex := len(parseExpected) - 1; parseIndex >= 0; parseIndex-- {
			parseNode := parseExpected[parseIndex]
			if parseKept[parseIndex] {
				parseAnchor = parseNode
				continue
			}
			if IsDOMNodeNull(parseAnchor) {
				parseRt.domAdapter.AppendChild(parseDomParent, parseNode)
			} else {
				parseRt.domAdapter.InsertBefore(parseDomParent, parseNode, parseAnchor)
			}
			parseAnchor = parseNode
		}
		return
	}

	// Set mismatch (detached expected nodes or foreign observed nodes):
	// greedy positional repair handles inserts and removals.
	for parseIndex, parseExpectedNode := range parseExpected {
		if parseIndex < len(parseObserved) && IsSameDOMNode(parseObserved[parseIndex], parseExpectedNode) {
			continue
		}

		getObservedIndex := parseFindObservedChildNodeIndex(parseObserved, parseExpectedNode, parseIndex)
		isParseAttached := getObservedIndex >= 0

		if isParseAttached {
			parseObserved = parseRemoveObservedChildNode(parseObserved, getObservedIndex)
		}

		if parseIndex < len(parseObserved) {
			parseRt.domAdapter.InsertBefore(parseDomParent, parseExpectedNode, parseObserved[parseIndex])
			parseObserved = parseInsertObservedChildNode(parseObserved, parseIndex, parseExpectedNode)
			continue
		}
		parseRt.domAdapter.AppendChild(parseDomParent, parseExpectedNode)
		parseObserved = append(parseObserved, parseExpectedNode)
	}
}

// buildCommittedChildOrderMatch maps each observed child to its expected
// index; hasSameSet is false when the two lists are not the same node set.
func buildCommittedChildOrderMatch(parseExpected, parseObserved []DOMNode) ([]int, bool) {
	if len(parseExpected) != len(parseObserved) {
		return nil, false
	}
	parseMatch := make([]int, len(parseObserved))
	parseUsed := make([]bool, len(parseExpected))
	for parseIndex, parseObservedNode := range parseObserved {
		parseFound := -1
		for parseExpectedIndex, parseExpectedNode := range parseExpected {
			if !parseUsed[parseExpectedIndex] && IsSameDOMNode(parseExpectedNode, parseObservedNode) {
				parseFound = parseExpectedIndex
				break
			}
		}
		if parseFound < 0 {
			return nil, false
		}
		parseUsed[parseFound] = true
		parseMatch[parseIndex] = parseFound
	}
	return parseMatch, true
}

// markCommittedChildOrderKept marks the expected indices forming the longest
// increasing subsequence of the observed→expected match — the children whose
// relative DOM order is already correct — using the O(n log n) patience
// algorithm. Everything unmarked must move.
func markCommittedChildOrderKept(parseMatch []int, parseExpectedLen int) ([]bool, int) {
	parseTails := make([]int, 0, len(parseMatch))
	parsePrev := make([]int, len(parseMatch))
	parseTailIndexes := make([]int, 0, len(parseMatch))
	for parseIndex, parseValue := range parseMatch {
		parseLo, parseHi := 0, len(parseTails)
		for parseLo < parseHi {
			parseMid := (parseLo + parseHi) / 2
			if parseTails[parseMid] < parseValue {
				parseLo = parseMid + 1
			} else {
				parseHi = parseMid
			}
		}
		if parseLo > 0 {
			parsePrev[parseIndex] = parseTailIndexes[parseLo-1]
		} else {
			parsePrev[parseIndex] = -1
		}
		if parseLo == len(parseTails) {
			parseTails = append(parseTails, parseValue)
			parseTailIndexes = append(parseTailIndexes, parseIndex)
		} else {
			parseTails[parseLo] = parseValue
			parseTailIndexes[parseLo] = parseIndex
		}
	}
	parseKept := make([]bool, parseExpectedLen)
	parseCount := len(parseTails)
	if parseCount > 0 {
		for parseCursor := parseTailIndexes[parseCount-1]; parseCursor >= 0; parseCursor = parsePrev[parseCursor] {
			parseKept[parseMatch[parseCursor]] = true
		}
	}
	return parseKept, parseCount
}

// parseFindObservedChildNodeIndex reports one node index inside the observed child order, starting at parseFromIndex.
func parseFindObservedChildNodeIndex(parseObserved []DOMNode, parseNode DOMNode, parseFromIndex int) int {
	if parseFromIndex < 0 {
		parseFromIndex = 0
	}
	if parseFromIndex >= len(parseObserved) {
		return -1
	}
	for parseIndex := parseFromIndex; parseIndex < len(parseObserved); parseIndex++ {
		if IsSameDOMNode(parseObserved[parseIndex], parseNode) {
			return parseIndex
		}
	}
	return -1
}

// parseRemoveObservedChildNode removes one observed child entry in place so child-order repair can avoid rebuilding DOM observation state.
func parseRemoveObservedChildNode(parseObserved []DOMNode, parseIndex int) []DOMNode {
	if parseIndex < 0 || parseIndex >= len(parseObserved) {
		return parseObserved
	}
	copy(parseObserved[parseIndex:], parseObserved[parseIndex+1:])
	parseObserved[len(parseObserved)-1] = nil
	return parseObserved[:len(parseObserved)-1]
}

// parseInsertObservedChildNode inserts one observed child entry in place so child-order repair can keep one synchronized local view of DOM order.
func parseInsertObservedChildNode(parseObserved []DOMNode, parseIndex int, parseNode DOMNode) []DOMNode {
	if parseIndex < 0 {
		parseIndex = 0
	}
	if parseIndex >= len(parseObserved) {
		return append(parseObserved, parseNode)
	}
	parseObserved = append(parseObserved, nil)
	copy(parseObserved[parseIndex+1:], parseObserved[parseIndex:])
	parseObserved[parseIndex] = parseNode
	return parseObserved
}

// oldFiberUpdateOrigin is an internal reconciler helper.
func oldFiberUpdateOrigin(parseOldFiber *Fiber, parseTypeOf any) string {
	if parseOldFiber != nil && parseOldFiber.updateOrigin != "" {
		return parseOldFiber.updateOrigin
	}
	if isFineGrainedType(parseTypeOf) {
		return "fine-grained"
	}
	return ""
}

// isReactiveTextType is an internal reconciler helper.
func isReactiveTextType(parseTypeOf any) bool {
	_, parseOk := parseTypeOf.(*ReactiveTextElementType)
	return parseOk
}

// isReactiveRegionType is an internal reconciler helper.
func isReactiveRegionType(parseTypeOf any) bool {
	_, parseOk := parseTypeOf.(*ReactiveRegionElementType)
	return parseOk
}

// isFineGrainedType is an internal reconciler helper.
func isFineGrainedType(parseTypeOf any) bool {
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
	if parseFiber.hasDirectText {
		parseRt.profiling.fineGrainedDescendantTextCommits++
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
	// For small subscription sets use inline linear search to avoid map allocation.
	if len(parsePrevious) <= 4 && len(parseSourceIDs) <= 4 {
		for _, parseOldID := range parsePrevious {
			parseKeep := slices.Contains(parseSourceIDs, parseOldID)
			if parseKeep {
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
			parseAlready := slices.Contains(parsePrevious, parseNewID)
			if parseAlready {
				continue
			}
			if parseRt.hydrating {
				parseRt.queueHydrationSubscription(parseNewID, parseFiber, true)
			} else {
				parseRt.atomRegistry.Subscribe(parseNewID, parseFiber)
			}
		}
	} else {
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
	// The atom-id prop may carry several comma-joined source ids (a multi-source computed),
	// so the text node subscribes to every dependency, not only the first.
	parseSourceIDs := []string{}
	for _, parseID := range strings.Split(parseAtomID, ",") {
		if strings.TrimSpace(parseID) != "" {
			parseSourceIDs = append(parseSourceIDs, parseID)
		}
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

	// Cleanup atom subscriptions and detach DOM refs (G2) for the fiber and
	// its subtree in ONE walk — these are order-independent runtime
	// bookkeeping phases; walking the deleted subtree separately for each
	// tripled the traversal cost of bulk removals. runCleanups above stays a
	// separate first pass: user cleanup callbacks may read descendant refs.
	parseRt.teardownDeletedSubtree(parseFiber)

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

// teardownDeletedSubtree unsubscribes atoms and releases DOM refs across one
// deleted subtree in a single recursive walk (see commitDeletion).
func (parseRt *Runtime) teardownDeletedSubtree(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}
	parseRt.CleanupAtomSubscriptions(parseFiber)
	if parseFiber.alternate != nil && parseFiber.alternate != parseFiber {
		parseRt.CleanupAtomSubscriptions(parseFiber.alternate)
	}
	parseRt.publishDOMRef(parseFiber, nil)
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseRt.teardownDeletedSubtree(parseChild)
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
		if parseResolver, parseOk3 := parseRt.domAdapter.(interface{ ResolveNode(any) DOMNode }); parseOk3 {
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

// runCleanups runs all cleanup functions for a fiber and its child subtree.
// It deliberately does NOT traverse the argument fiber's own sibling chain:
// commitDeletion passes a DELETION-tagged fiber whose old-tree siblings may
// survive (their Hooks objects are shared with their live alternates), so
// cleaning the sibling chain would tear down effects and release event-handler
// wrappers on components that are still mounted.
func (parseRt *Runtime) runCleanups(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}
	parseRt.runFiberCleanups(parseFiber)
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseRt.runCleanups(parseChild)
	}
}

// runFiberCleanups runs the cleanups and releases the event-handler wrappers
// for a single fiber, without traversing children or siblings.
func (parseRt *Runtime) runFiberCleanups(parseFiber *Fiber) {

	// Run this fiber's cleanups
	if parseFiber.hooks != nil {
		for parseIndex, parseCleanup := range parseFiber.hooks.cleanups {
			if parseCleanup != nil {
				parseStart := commitTimingStart()
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
				parseDurationNs := commitTimingSinceNs(parseStart)
				parseFiber.cleanupDurationNs += parseDurationNs
				parseRt.profiling.cleanupExecutions++
				parseRt.profiling.lastCleanupDurationNs = parseDurationNs
				parseRt.profiling.totalCleanupDurationNs += parseDurationNs
				recordSlowOperationDiagnostic("cleanup", parseFiber, parseDurationNs)
				parseFiber.hooks.cleanups[parseIndex] = nil
			}
		}
		// Release any wrapped event-handler funcs (e.g. js.Func in WASM) so the
		// JS GC bridge can free them.  The wrapper slot is zeroed to prevent a
		// double-release if runCleanups is called again (e.g. hot-reload path).
		for parseIdx := range parseFiber.hooks.funcs {
			releaseFuncHandlerWrapper(parseFiber.hooks.funcs[parseIdx].wrapper)
			parseFiber.hooks.funcs[parseIdx] = funcHandlerValue{}
		}
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
	parseLimit := parseRt.limits.withDefaults().MaxPendingEffectFibers
	if parseLimit > 0 && len(parseRt.pendingEffectFibers) >= parseLimit {
		parseRt.pendingEffectFibers = parseRt.pendingEffectFibers[:0]
		parseRt.pendingEffectOverflow = true
		parseRt.tracksPendingEffects = false
		ReportDiagnosticWithContext("runtime", DiagnosticWarning, "pending effect queue exceeded its bounded capacity; falling back to full-tree effect scan", diagnosticPathForFiber(parseFiber), diagnosticComponentStack(parseFiber))
		return
	}
	parseRt.tracksPendingEffects = true
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
//
// Layout effects (UseLayoutEffect, G36) run before passive effects (UseEffect)
// within the fiber, so measure/focus/scroll work observes the committed DOM
// before any passive effect mutates further. When the fiber has no layout
// effects (the common case) the effects run in their queued order in a single
// pass.
func (parseRt *Runtime) runFiberEffects(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	parseEffects := parseFiber.effects
	parseEffectCount := len(parseEffects)
	parseFiber.effectDurationNs = 0
	if parseEffectCount == 0 {
		return
	}
	// Consume the queue: effects run exactly once. A bailout-reused fiber keeps
	// its Fiber object across commits, and the full-tree runEffects fallback
	// (taken whenever a pass queues no effects) would otherwise re-execute every
	// stale queued effect on it — re-firing mount effects on unrelated updates
	// and overwriting their cleanups without calling them.
	defer func() { parseFiber.effects = parseFiber.effects[:0] }()

	parseHasLayout := false
	for parseI := range parseEffects {
		if parseEffects[parseI].Layout {
			parseHasLayout = true
			break
		}
	}

	if !parseHasLayout {
		for parseI := range parseEffects {
			parseRt.runOneEffect(parseFiber, &parseEffects[parseI])
		}
		return
	}

	// Two passes: layout effects first, then passive effects.
	for parseI := range parseEffects {
		if parseEffects[parseI].Layout {
			parseRt.runOneEffect(parseFiber, &parseEffects[parseI])
		}
	}
	for parseI := range parseEffects {
		if !parseEffects[parseI].Layout {
			parseRt.runOneEffect(parseFiber, &parseEffects[parseI])
		}
	}
}

// runOneEffect runs a single effect with effect-boundary panic containment and
// profiling, storing any returned cleanup at the effect's cleanup index.
func (parseRt *Runtime) runOneEffect(parseFiber *Fiber, parseEffect *Effect) {
	parseStart := commitTimingStart()
	parseCleanup := func() func() {
		var parseCleanupFn func()
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
			parseCleanupFn = parseEffect.Fn()
		}()
		return parseCleanupFn
	}()
	parseDurationNs := commitTimingSinceNs(parseStart)
	parseFiber.effectDurationNs += parseDurationNs
	parseRt.profiling.effectExecutions++
	parseRt.profiling.lastEffectDurationNs = parseDurationNs
	parseRt.profiling.totalEffectDurationNs += parseDurationNs
	recordSlowOperationDiagnostic("effect", parseFiber, parseDurationNs)
	if parseCleanup != nil {
		parseFiber.hooks.cleanups[parseEffect.CleanupIndex] = parseCleanup
	}
	parseRt.checkStrictEffectCleanupSymmetry(parseFiber, parseEffect.CleanupIndex, parseCleanup != nil)
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

// markPortalSubtreeForPlacement re-tags a portal subtree whose target failed
// to resolve on a previous commit: the fibers exist but never created DOM, so
// they must run the placement path once the target element appears.
func markPortalSubtreeForPlacement(parseFiber *Fiber) {
	for parseCursor := parseFiber; parseCursor != nil; parseCursor = parseCursor.sibling {
		// DOM nodes may already exist (created during render, never attached
		// because the first commit had no target); the placement path appends
		// existing nodes, so tag unconditionally.
		parseCursor.effectTag = effectTagPlacement
		markPortalSubtreeForPlacement(parseCursor.child)
	}
}
