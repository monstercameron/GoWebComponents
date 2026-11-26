package runtime

import (
	"fmt"
	"reflect"
	"sync/atomic"
)

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

// duplicateKeyWarned makes reportDuplicateKeys fire at most once per process,
// matching reportUnkeyedComponentAliasing: one educational nudge, then the scan
// short-circuits and costs nothing.
var duplicateKeyWarned atomic.Bool

// reportDuplicateKeys warns when two siblings in a keyed list share one key.
//
// Why a diagnostic rather than a fallback. The in-order fast path validates the
// matched prefix against the old chain but checks trailing APPENDS for
// castability only, so appending an element whose key already exists produces
// two same-key fibers with no deletion. Making that abort to the map path was
// the obvious fix and is the wrong one: trailing append is the dominant
// list-growth shape, the fast path exists precisely to keep it allocation-free,
// and the slow path does not actually repair the situation either — its
// duplicate handling routes the collision to a positional fallback list, which
// is a tolerance, not a resolution.
//
// The real defect is upstream, in the caller's key function, and it is
// invisible: the DOM stays correct while hook state follows the wrong logical
// item across the next reorder. So this reports it where it can be fixed and
// leaves the hot path alone. Dev builds only.
func reportDuplicateKeys(parseParent *Fiber, parseElements []any) {
	if !hookThreadingGuardEnabled || duplicateKeyWarned.Load() {
		return
	}
	// Small sibling lists scan a stack array instead of building a map.
	//
	// This check runs on EVERY keyed reconcile, and the map it used to allocate
	// unconditionally measured 63% of all allocations in
	// BenchmarkReconcileChildrenKeyedStableList16 — making the keyed path 4x the
	// cost of the unkeyed one (6315ns/6 allocs vs 1573ns/0 allocs) in exactly
	// the shape every real list has. It is compiled out of production builds, so
	// this never reached shipped code, but it is what developers profile against
	// and it made dev-mode measurements of keyed lists misleading.
	//
	// A linear scan is also genuinely faster at these sizes: comparing a handful
	// of interface values beats hashing them, and duplicate-key bugs are found
	// just as well either way. The map is kept for long lists so the quadratic
	// scan cannot become its own problem.
	const parseLinearScanLimit = 32
	var parseSeenSmall [parseLinearScanLimit]any
	parseSmallCount := 0
	var parseSeen map[any]struct{}
	for _, parseElement := range parseElements {
		parseElem, parseOk := parseElement.(*Element)
		if !parseOk || parseElem == nil {
			continue
		}
		parseKey, isComparable, hasKey := elementReconcileKey(parseElem)
		if !hasKey || !isComparable {
			continue
		}
		if parseSeen == nil && parseSmallCount < parseLinearScanLimit {
			parseIsDuplicate := false
			for parseIndex := range parseSmallCount {
				if parseSeenSmall[parseIndex] == parseKey {
					parseIsDuplicate = true
					break
				}
			}
			if !parseIsDuplicate {
				parseSeenSmall[parseSmallCount] = parseKey
				parseSmallCount++
				continue
			}
			reportDuplicateKeyOnce(parseParent, parseKey)
			return
		}
		if parseSeen == nil {
			// Crossed the linear-scan limit: promote what has been seen so far
			// into a map rather than restarting, so a long list still gets the
			// check without rescanning.
			parseSeen = make(map[any]struct{}, len(parseElements))
			for parseIndex := range parseSmallCount {
				parseSeen[parseSeenSmall[parseIndex]] = struct{}{}
			}
		}
		if _, isDuplicate := parseSeen[parseKey]; isDuplicate {
			reportDuplicateKeyOnce(parseParent, parseKey)
			return
		}
		parseSeen[parseKey] = struct{}{}
	}
}

// reportDuplicateKeyOnce emits the duplicate-sibling-key warning at most once per
// process. Extracted so the linear-scan and map paths in reportDuplicateKeys
// cannot drift into reporting the same defect two different ways.
func reportDuplicateKeyOnce(parseParent *Fiber, parseKey any) {
	if !duplicateKeyWarned.CompareAndSwap(false, true) {
		return
	}
	_, parseParentName := describeFiber(parseParent)
	ReportDiagnostic("runtime", DiagnosticWarning,
		fmt.Sprintf("two sibling elements under %s share the key %v; keys must be unique among siblings, "+
			"or hook state and DOM identity will follow list POSITION rather than the logical item across reorders",
			parseParentName, parseKey))
}

// tryReconcileKeyedChildrenInOrder fast-paths keyed lists that kept the same
// sibling order, including trailing append and trailing removal.
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
	// Phase 2: build the chain — updates over the matched prefix, placements
	// for the appended tail, or deletions for the old trailing tail.
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
			parseNewFiber = parseRt.buildPlacementFiber(parseWipFiber, parseElem, nil)
		}
		if parseFirstChild == nil {
			parseFirstChild = parseNewFiber
		} else if parsePrevSibling != nil {
			parsePrevSibling.sibling = parseNewFiber
		}
		parsePrevSibling = parseNewFiber
	}
	parseWipFiber.child = parseFirstChild
	for parseOldFiber != nil {
		parseNextOldFiber := parseOldFiber.sibling
		parseOldFiber.effectTag = effectTagDeletion
		parseRt.deletions = append(parseRt.deletions, parseOldFiber)
		parseOldFiber = parseNextOldFiber
	}
	return true
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
	var parseOldFirst *Fiber
	if parseWipFiber.alternate != nil {
		parseOldFirst = parseWipFiber.alternate.child
	}
	if parseRt.tryReconcileKeyedChildrenInOrder(parseWipFiber, parseElements, parseOldFirst) {
		return
	}

	// Everything below belongs to the reorder/fallback path. Acquire its maps,
	// scratch slices, and deferred cleanup only after the ordered path declines;
	// stable lists, appends, and suffix removals otherwise paid two allocations
	// plus four pool operations immediately before returning.
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
					markFiberCommitPath(parseWipFiber.parent)
				}
				if parseMatchedOldIndex > parseLastMatchedOldIndex {
					parseLastMatchedOldIndex = parseMatchedOldIndex
				}
			}

			parseNewFiber = parseRt.buildUpdatedFiber(parseWipFiber, parseMatchedOld, parseElem)
		} else {
			if len(parseOldByKey) > 0 || hasRemainingFallbackKeyed(parseOldFallbackKeyed) || parseUnkeyedIndex < len(parseOldUnkeyed) {
				parseWipFiber.needsChildOrder = true
				markFiberCommitPath(parseWipFiber.parent)
			}
			if parseMatchedOld != nil {
				parseMatchedOld.effectTag = effectTagDeletion
				parseRt.deletions = append(parseRt.deletions, parseMatchedOld)
			}

			parseNewFiber = parseRt.buildPlacementFiber(parseWipFiber, parseElem, nil)
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
