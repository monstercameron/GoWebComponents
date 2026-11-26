package runtime

import (
	"sync/atomic"
)

// unkeyedComponentAliasWarned makes reportUnkeyedComponentAliasing fire at most
// once per process — one educational nudge, not a per-render spam, and after it
// fires the whole scan short-circuits so the hot path pays nothing.
var unkeyedComponentAliasWarned atomic.Bool

const getCommittedChildReplaceThreshold = 8

// attrUpdateBatchDOMAdapter is the optional capability for buffering a
// commit's attribute writes, text writes, and removals into a small number of
// bridge calls (browser adapter only). While a batch is open, reads through the
// adapter may observe stale values — commitRoot flushes before anything that
// reads the DOM runs.
type attrUpdateBatchDOMAdapter interface {
	BeginAttrUpdateBatch()
	EndAttrUpdateBatch()
}

// commitRoot commits all changes to the DOM
func (parseRt *Runtime) commitRoot() {
	parseStart := commitTimingStart()
	defer func() {
		parseDurationNs := commitTimingSinceNs(parseStart)
		parseRt.AdvanceAgentStateVersion()
		// The profiling counters and the event ring are guarded by schedulerMu
		// everywhere else (recordComponentRenderTrace takes it before calling its
		// ...Locked half); this tail wrote them bare while calling a function
		// named for the lock. Taken here rather than inside
		// recordProfilingEventLocked so the *Locked convention keeps meaning what
		// it says at every other call site.
		schedulerMu.Lock()
		defer schedulerMu.Unlock()
		parseRt.profiling.commitCount++
		parseRt.profiling.lastCommitDurationNs = parseDurationNs
		parseRt.profiling.totalCommitDurationNs += parseDurationNs
		if runtimeHotPathProfilingEnabled {
			parseRt.recordProfilingEventLocked(ProfilingEvent{
				Domain:     "runtime",
				Name:       "commit",
				Phase:      "finish",
				Target:     "root",
				DurationNs: parseDurationNs,
			})
		}
		if runtimeHotPathProfilingEnabled && !parseRt.profiling.startupStartedAt.IsZero() && parseRt.profiling.startupCommitDurationNs == 0 {
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
	// Cross-node mutation batching: buffer every attribute write, text write,
	// and removal in the commit walk. Nothing in the walk reads those values
	// back; the deferred End covers panic unwinds and the explicit flush below
	// runs before effects, which may read the DOM.
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
	// Clear deletions but keep capacity.
	//
	// clear() before the reslice, because [:0] alone keeps every *Fiber in the
	// backing array reachable. The slice is unbounded, so one teardown of a
	// large list left thousands of deleted fibers — each pinning its detached
	// subtree, its hooks, and its DOM handles — resident until some later pass
	// happened to overwrite those slots.
	clear(parseRt.deletions)
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

	// Flush buffered mutations before effects and subscribers run —
	// they may read the DOM. The deferred End above then no-ops.
	if hasAttrBatch {
		parseAttrBatchAdapter.EndAttrUpdateBatch()
	}

	// Under schedulerMu, because every field written here is read under it by
	// scheduleUpdateWithLane and coalesceScheduledUpdateLocked. commitRoot held
	// no lock at all and still called finishScheduledLocked — a name whose whole
	// job is to tell the reader the caller holds it. Single-threaded in wasm and
	// therefore invisible where the runtime usually runs; genuinely concurrent in
	// SSR and in the native suite, which is what the -race job now watches.
	schedulerMu.Lock()
	parseRt.currentRoot = parseCommittedRoot
	parseRt.schedulerState.finishScheduledLocked()
	parseRt.wipRoot = nil
	schedulerMu.Unlock()

	parseWasHydrating := parseRt.hydrating
	if parseWasHydrating {
		parseRt.hydrating = false
		parseRt.strictHydration = false
		// Deliberately OUTSIDE the lock: a flushed subscription can schedule an
		// update, which takes schedulerMu, so holding it across this region
		// would deadlock. Two short critical sections rather than one long one.
		parseRt.flushHydrationSubscriptions()
	}

	schedulerMu.Lock()
	parseRt.updateScheduled = false
	schedulerMu.Unlock()

	// Run effects after the committed tree is current and hydration gates are lifted.
	//
	// v5 P1.1: when passiveEffectsAfterPaint is enabled, only LAYOUT effects run
	// here — synchronously, in the same task as the DOM writes, because that is
	// the whole reason the tier exists. Passive effects are deferred past the
	// paint boundary so a slow UseEffect can no longer hold the frame.
	if parseRt.passiveEffectsAfterPaint {
		parseRt.commitLayoutEffectsAndDeferPassive(parseCommittedRoot)
	} else {
		if parseRt.tracksPendingEffects && !parseRt.pendingEffectOverflow {
			parseRt.runPendingEffects()
		} else {
			parseRt.runEffects(parseCommittedRoot)
		}
		parseRt.tracksPendingEffects = false
		parseRt.pendingEffectOverflow = false
	}

	if parseWasHydrating {
		parseRt.finishHydrationMetrics(false, "")
		parseRt.flushDeferredHydrationUpdates()
	}
	if parseRt.pendingBoundaryRecovery {
		parseRt.pendingBoundaryRecovery = false
		parseRt.ScheduleUpdate()
	}
	parseRt.ageEffectErroredBoundaries()

	// v5 P2.5 (T12): a higher-priority lane arrived mid-pass. The pass was
	// allowed to finish and commit rather than being torn down in place; start
	// the higher lane now that the tree is consistent.
	if parseInterruptLane := parseRt.pendingInterruptLane; parseInterruptLane != 0 {
		parseRt.pendingInterruptLane = 0
		parseRt.scheduleUpdateWithLane(parseInterruptLane, false)
	}

	// v5 P2.2: this pass declined to render work belonging to lower lanes.
	// Start a follow-up pass at the highest lane still pending, so deferral
	// never becomes a stranded update.
	parseRt.scheduleDeferredLaneWork()
	parseRt.scheduleFollowUpForInFlightUpdate()

	// G16: signal "app ready" once the first commit (initial paint + effects) is
	// done. No-op on every later commit.
	parseRt.fireFirstCommitHooks()
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

// reportUnkeyedComponentAliasing warns once when a parent has two or more UNKEYED
// COMPONENT children that resolve to the same identity (the classic inline-closure-
// per-list-item case, where every closure shares one code-pointer identity). Unlike
// reportMissingKeys — which only flags a MIXED keyed/unkeyed list — this catches the
// fully-unkeyed stateful-component list, where the reconciler associates hook state
// with slot position rather than logical item, so insert/delete/reorder shows the
// wrong item's state. Dev builds only; warn-once so it costs nothing after firing.
func reportUnkeyedComponentAliasing(parseParent *Fiber, parseElements []any) {
	if !hookThreadingGuardEnabled || unkeyedComponentAliasWarned.Load() {
		return
	}
	// Cheap pre-count before any reflection or allocation.
	//
	// Two unkeyed component siblings are required for this to report anything,
	// and the scan below reaches describeCallableIdentity — which reflects over
	// a function value — plus a map allocation, on the FIRST candidate. Most
	// child lists are host elements or a single component, so the expensive part
	// ran constantly on lists that could never produce a warning. This pass is
	// type assertions only and stops at two.
	parseCandidates := 0
	for _, parseElement := range parseElements {
		parseElem, parseOk := parseElement.(*Element)
		if !parseOk || parseElem == nil || parseElem.Type == nil {
			continue
		}
		if _, isParseHost := parseElem.Type.(string); isParseHost {
			continue
		}
		if hasElementKey(parseElem) {
			continue
		}
		parseCandidates++
		if parseCandidates >= 2 {
			break
		}
	}
	if parseCandidates < 2 {
		return
	}

	var parseIdentityCounts map[string]int
	for _, parseElement := range parseElements {
		parseElem, parseOk := parseElement.(*Element)
		if !parseOk || parseElem == nil || parseElem.Type == nil {
			continue
		}
		if _, isParseHost := parseElem.Type.(string); isParseHost {
			continue // host element (e.g. a <div>): unkeyed host children carry no hook state
		}
		if hasElementKey(parseElem) {
			continue // explicitly keyed → correctly identified across reorders
		}
		_, parseIdentity := describeCallableIdentity(parseElem.Type)
		if parseIdentity == "" {
			continue
		}
		if parseIdentityCounts == nil {
			parseIdentityCounts = map[string]int{}
		}
		parseIdentityCounts[parseIdentity]++
		if parseIdentityCounts[parseIdentity] >= 2 {
			if unkeyedComponentAliasWarned.CompareAndSwap(false, true) {
				_, parseParentName := describeFiber(parseParent)
				ReportDiagnostic("runtime", DiagnosticWarning,
					"two or more sibling components without keys under "+parseParentName+
						" share one identity ("+parseIdentity+"); their hook state will associate with list POSITION, not the logical item, so insert/delete/reorder can render the wrong item's state — pass an explicit key (e.g. WithKey) to each list item")
			}
			return
		}
	}
}

// parentChild is an internal reconciler helper.
func parentChild(parseParent *Fiber) *Fiber {
	if parseParent == nil || parseParent.alternate == nil {
		return nil
	}
	return parseParent.alternate.child
}

// commitWork commits a fiber's changes to the DOM with batch optimization.
//
// Siblings are walked in a loop and only child depth recurses, so a list of
// tens of thousands of siblings costs one stack frame instead of one per
// sibling (the per-sibling recursion overflowed the JS stack near 5,000 rows).
func (parseRt *Runtime) commitWork(parseFiber *Fiber, parseDomParent DOMNode) {
	for parseFiber != nil {
		parseFiber = parseRt.commitWorkFiber(parseFiber, parseDomParent)
	}
}

// commitWorkFiber commits one fiber and its children and returns the sibling
// that commitWork should continue with; a nil fiber ends the sibling walk. The
// DOM parent is unchanged while walking siblings, so returning it alongside
// the fiber only adds a Wasm multi-value return on the hottest commit path.
func (parseRt *Runtime) commitWorkFiber(parseFiber *Fiber, parseDomParent DOMNode) *Fiber {
	// The render/reconcile pass marks every ancestor of a placement, update,
	// hydration claim, unresolved portal retry, or child-order repair. Outside
	// hydration, a fiber with neither direct nor descendant work can skip its
	// whole child subtree; its sibling remains an independent branch.
	if parseRt.wipRoot != nil && !parseRt.hydrating && parseFiber.effectTag == effectTagNone &&
		!parseFiber.subtreeCommit && !parseFiber.needsChildOrder && !parseFiber.portalUnresolved {
		return parseFiber.sibling
	}

	// Fast path: most calls have valid domParent
	if IsDOMNodeNull(parseDomParent) {
		parseDomParentFiber := parseFiber.parent
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
				if parseFiber.isCompactHostProps {
					// Typed fast-lane fibers carry no events or controlled
					// values; adopting a hydrated node only needs its
					// attributes reasserted.
					parseRt.applyCompactHostAttrs(parseFiber.dom, parseFiber.getHostAttrs)
					if parseFiber.hasCompactSpecialProps {
						parseRt.applyInitialDomProps(parseFiber.dom, parseFiber.props, parseFiber.hydrated)
					}
				} else {
					parseRt.applyInitialDomProps(parseFiber.dom, parseFiber.props, parseFiber.hydrated)
				}
				if parseFiber.hasDirectText && parseRt.domNodeText(parseFiber.dom) != parseFiber.textContent {
					parseRt.domAdapter.SetTextContent(parseFiber.dom, parseFiber.textContent)
				}
				// A hydrated node is adopted, never placed, so the placement-time
				// ref publish does not run for it.
				parseRt.publishDOMRef(parseFiber, parseFiber.dom)
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
					if parseFiber.isCompactHostProps && parseFiber.alternate.isCompactHostProps &&
						parseFiber.hasCompactSpecialProps == parseFiber.alternate.hasCompactSpecialProps {
						// Typed fast lane: diff the deterministic attribute
						// slices directly and touch only changed attributes.
						if parseRt.updateCompactHostAttrs(parseFiber.dom, parseFiber.alternate.getHostAttrs, parseFiber.getHostAttrs) {
							isParseCommitted = true
						}
						if parseFiber.hasCompactSpecialProps && !propsEqualIgnoringChildren(parseFiber.alternate.props, parseFiber.props) {
							parseRt.updateDomProperties(parseFiber.dom, parseFiber.alternate.props, parseFiber.props)
							isParseCommitted = true
						}
					} else if parseOldView, parseNewView := fiberPropsView(parseFiber.alternate), fiberPropsView(parseFiber); !propsEqualIgnoringChildren(parseOldView, parseNewView) {
						parseRt.updateDomProperties(parseFiber.dom, parseOldView, parseNewView)
						isParseCommitted = true
					}
					if fiberMayCarryDOMRef(parseFiber) || fiberMayCarryDOMRef(parseFiber.alternate) {
						parseRt.retargetDOMRef(parseFiber)
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
			return nil
		}
	}

	if isPortal {
		if parseFiber.effectTag == effectTagDeletion {
			parseRt.commitDeletion(parseFiber, parsePortalParent)
			return nil
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
			return parseFiber.sibling
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
			parseRt.prepareSerializedSiblingRuns(parseFiber, parseChildDomParent)
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
	return parseFiber.sibling
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
		parseRt.releaseDeletedSubtree(parseFiber)
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
	parseRt.releaseDeletedSubtree(parseFiber)
}

// releaseDeletedSubtree drops the references a fully deleted fiber still holds,
// after its cleanups have run and its DOM is detached.
//
// Mount fibers are slab-allocated 128 at a time (acquireMountFiber), and a slab
// stays alive while ANY fiber in it is reachable. The slab comment bounds the
// cost at the 424-byte slot on the premise that "a fiber is zeroed when it is
// reused or dropped" — true of the REUSE path, which zeroes through
// acquireWorkInProgress, but nothing zeroed a DELETED fiber. So unmounting 127
// rows of a slab whose 128th fiber survives kept all 127 rows' props maps,
// hooks (state values, memo results, refs), children slices, and DOM handles
// reachable through the slab, indefinitely.
//
// Sibling contract matches runCleanups: the argument fiber's own sibling chain
// is NOT traversed, because a DELETION-tagged fiber's old-tree siblings may
// still be mounted. Children are traversed with their siblings, as they belong
// to the deleted subtree.
//
// typeOf and ownerRuntime are deliberately left in place: they are word-sized,
// retain nothing of consequence, and a late async write that reaches a deleted
// fiber (a resolved suspension, an in-flight validator) still finds a coherent
// object to no-op against rather than a stripped one.
func (parseRt *Runtime) releaseDeletedSubtree(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}
	for parseChild := parseFiber.child; parseChild != nil; {
		// Read the sibling link before the child releases it.
		parseNext := parseChild.sibling
		parseRt.releaseDeletedSubtree(parseChild)
		parseChild = parseNext
	}

	// Only plain host/text fibers are safe to recycle. Component fibers may be
	// retained by hook setters, async completions, or user callbacks; reusing
	// one would let a stale handle mutate an unrelated newly-mounted component.
	// A deleted host fiber has no such public identity once refs, events, and
	// subscriptions have been detached by teardownDeletedSubtree.
	parseRecyclableHost := false
	if parseType, parseOk := parseFiber.typeOf.(string); parseOk &&
		parseType != "ROOT" && parseType != "FRAGMENT" &&
		parseFiber.hooks == nil && len(parseFiber.eventCallbacks) == 0 &&
		parseFiber.asyncSuspension == nil && parseFiber.asyncWait == nil &&
		parseFiber.reactiveAtomID == "" && len(parseFiber.reactiveSourceIDs) == 0 &&
		!parseFiber.fineGrained {
		parseRecyclableHost = true
	}

	parseFiber.parent = nil
	parseFiber.child = nil
	parseFiber.sibling = nil
	parseFiber.alternate = nil
	parseFiber.dom = nil
	parseFiber.hooks = nil
	parseFiber.props = nil
	parseFiber.children = nil
	parseFiber.getHostAttrs = nil
	parseFiber.contextValues = nil
	parseFiber.hydration = nil
	parseFiber.childHydration = nil
	parseFiber.effects = nil
	parseFiber.eventCallbacks = nil
	parseFiber.reactiveSourceIDs = nil
	parseFiber.asyncSuspension = nil
	parseFiber.asyncWait = nil
	parseFiber.boundaryError = nil

	// Retain a bounded high-water pool. This converts repeated mount/clear
	// cycles into pointer reuse without pinning arbitrarily large one-off trees.
	const parseMaxRecycledHostFibers = 4096
	if parseRecyclableHost && parseRt != nil && len(parseRt.recycledHostFibers) < parseMaxRecycledHostFibers {
		*parseFiber = Fiber{}
		parseRt.recycledHostFibers = append(parseRt.recycledHostFibers, parseFiber)
	}
}

// teardownDeletedSubtree unsubscribes atoms, releases DOM refs, and detaches
// portal-hosted DOM across one deleted subtree in a single recursive walk (see
// commitDeletion).
func (parseRt *Runtime) teardownDeletedSubtree(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}
	parseRt.CleanupAtomSubscriptions(parseFiber)
	if parseFiber.alternate != nil && parseFiber.alternate != parseFiber {
		parseRt.CleanupAtomSubscriptions(parseFiber.alternate)
	}
	parseRt.publishDOMRef(parseFiber, nil)
	parseRt.detachDeletedPortalDOM(parseFiber)
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseRt.teardownDeletedSubtree(parseChild)
	}
}

// detachDeletedPortalDOM removes the DOM a portal placed in its OWN target when
// that portal is deleted as part of a larger subtree.
//
// commitDeletion handles the case where the deleted fiber IS a portal, and
// nothing handled a portal nested BELOW it — which is where every portal in a
// real application lives, since a modal or tooltip is rendered by a component
// that gets conditionally unmounted. Both shapes leaked:
//
//   - the deleted ancestor owns a DOM node, so commitDeletion removes that node
//     and returns without ever descending to the portal
//   - the deleted ancestor is a DOM-less component, so deleteFiberSubtree
//     descends with the ancestor's OWN dom parent, and the portal's children are
//     not under it; the browser adapter's RemoveChild checks parentNode before
//     calling remove(), so the mismatch is a silent no-op rather than an error
//
// Either way the portal's nodes stay in the document forever, wired to state
// whose cleanups have already run — runCleanups walks the deleted subtree, so
// the effects are torn down while their DOM lives on.
//
// Removal uses the portal's own resolved target rather than the inherited DOM
// parent, which is the whole point: a portal's children were never under the
// parent the deletion walk is carrying.
func (parseRt *Runtime) detachDeletedPortalDOM(parseFiber *Fiber) {
	if parseRt == nil || !parseRt.isPortalFiber(parseFiber) {
		return
	}
	parsePortalParent := parseRt.resolvePortalParent(parseFiber)
	if IsDOMNodeNull(parsePortalParent) {
		return
	}
	parseRt.deleteFiberSubtree(parseFiber.child, parsePortalParent)
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
	// Looping instead of recursing keeps wide lists off the call stack.
	for parseSibling := parseFiber.sibling; parseSibling != nil; parseSibling = parseSibling.sibling {
		if !IsDOMNodeNull(parseSibling.dom) {
			parseRt.domAdapter.RemoveChild(parseDomParent, parseSibling.dom)
		} else {
			parseRt.deleteFiberSubtree(parseSibling.child, parseDomParent)
		}
	}
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

	// Siblings are moved in a loop so wide lists do not recurse per sibling.
	for parseSibling := parseFiber.sibling; parseSibling != nil && !parseRt.isPortalFiber(parseSibling); parseSibling = parseSibling.sibling {
		if !IsDOMNodeNull(parseSibling.dom) {
			parseRt.domAdapter.RemoveChild(parseOldParent, parseSibling.dom)
			parseRt.domAdapter.AppendChild(parseNewParent, parseSibling.dom)
		} else {
			parseRt.movePortalSubtree(parseSibling.child, parseOldParent, parseNewParent)
		}
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
