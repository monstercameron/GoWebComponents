package runtime

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
func (parseRt *Runtime) buildObservedChildNodes(parseDomParent DOMNode, parseCapacity int) []DOMNode {
	if parseRt == nil || parseRt.domAdapter == nil || IsDOMNodeNull(parseDomParent) {
		return nil
	}
	if parseCapacity < 4 {
		parseCapacity = 4
	}
	return parseRt.buildObservedChildNodesFrom(parseRt.domAdapter.GetFirstChild(parseDomParent), parseCapacity)
}

// buildObservedChildNodesFrom collects siblings starting at an already-read first child.
func (parseRt *Runtime) buildObservedChildNodesFrom(parseFirstNode DOMNode, parseCapacity int) []DOMNode {
	if parseCapacity < 4 {
		parseCapacity = 4
	}
	parseNodes := make([]DOMNode, 0, parseCapacity)
	for parseNode := parseFirstNode; !IsDOMNodeNull(parseNode); parseNode = parseRt.domAdapter.GetNextSibling(parseNode) {
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
	// Most commits preserve sibling order. Check that common case through the
	// adapter iterator so it does not allocate an observed-node slice. If the
	// children match an expected prefix and end, append the suffix directly.
	// Repeated references still use AppendChild, preserving DOM move semantics.
	parseFirstObservedNode := parseRt.domAdapter.GetFirstChild(parseDomParent)
	parseObservedNode := parseFirstObservedNode
	parseMatchedPrefix := 0
	for parseMatchedPrefix < len(parseExpected) {
		if IsDOMNodeNull(parseObservedNode) || !IsSameDOMNode(parseExpected[parseMatchedPrefix], parseObservedNode) {
			break
		}
		parseMatchedPrefix++
		parseObservedNode = parseRt.domAdapter.GetNextSibling(parseObservedNode)
	}
	if IsDOMNodeNull(parseObservedNode) {
		if parseMatchedPrefix == len(parseExpected) {
			return
		}
		for parseIndex := parseMatchedPrefix; parseIndex < len(parseExpected); parseIndex++ {
			parseRt.domAdapter.AppendChild(parseDomParent, parseExpected[parseIndex])
		}
		return
	}
	// A single adjacent inversion is common in keyed lists and needs only one
	// insertBefore. Confirm the complete suffix through the sibling iterator so
	// foreign children and repeated node identities retain the general repair.
	if repairCommittedChildOrderAdjacentSwap(parseRt, parseDomParent, parseExpected, parseMatchedPrefix, parseObservedNode) {
		return
	}
	parseObserved := parseRt.buildObservedChildNodesFrom(parseFirstObservedNode, len(parseExpected))

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

// repairCommittedChildOrderAdjacentSwap moves one inverted adjacent child after confirming the complete remaining order.
func repairCommittedChildOrderAdjacentSwap(parseRt *Runtime, parseDomParent DOMNode, parseExpected []DOMNode, parseMismatchIndex int, parseObservedNode DOMNode) bool {
	if parseMismatchIndex+1 >= len(parseExpected) || IsDOMNodeNull(parseObservedNode) || !IsSameDOMNode(parseObservedNode, parseExpected[parseMismatchIndex+1]) {
		return false
	}
	parseSwappedObservedNode := parseRt.domAdapter.GetNextSibling(parseObservedNode)
	if IsDOMNodeNull(parseSwappedObservedNode) || !IsSameDOMNode(parseSwappedObservedNode, parseExpected[parseMismatchIndex]) {
		return false
	}
	parseObservedNode = parseRt.domAdapter.GetNextSibling(parseSwappedObservedNode)
	for parseIndex := parseMismatchIndex + 2; parseIndex < len(parseExpected); parseIndex++ {
		if IsDOMNodeNull(parseObservedNode) || !IsSameDOMNode(parseObservedNode, parseExpected[parseIndex]) {
			return false
		}
		parseObservedNode = parseRt.domAdapter.GetNextSibling(parseObservedNode)
	}
	if !IsDOMNodeNull(parseObservedNode) {
		return false
	}
	parseRt.domAdapter.InsertBefore(parseDomParent, parseExpected[parseMismatchIndex], parseExpected[parseMismatchIndex+1])
	return true
}

// buildCommittedChildOrderMatch maps each observed child to its expected
// index; hasSameSet is false when the two lists are not the same node set.
//
// The scan resumes from the last match instead of restarting at zero, and that
// is the whole performance story here.
//
// This feeds an O(n log n) LIS, but the match that feeds it restarted its search
// at index 0 for every observed node, so the pair was quadratic and the LIS was
// free by comparison. Measured on the shapes a list actually produces: 500 rows
// 547µs, 1000 rows 1.57ms, 2000 rows 4.07ms, 5000 rows 22.5ms, against an LIS
// that never registered above 0.5ms. It runs inside commitRoot, which no frame
// budget bounds, and it triggers when a filtered list is re-populated — the
// exact shape M2's residual long frames were measured on.
//
// Resuming works because the two lists are permutations of the same set and DOM
// order is mostly preserved: a rotation, an insertion, or a handful of moved
// rows all advance the cursor monotonically, so each search stops on its first
// or second probe. The wrap-around second pass keeps it CORRECT for an arbitrary
// permutation rather than merely fast for a friendly one, and the used-set check
// keeps duplicate node values matching one-to-one exactly as before. Worst case
// is unchanged; the common case stops being the worst case.
//
// A hash index would make it unconditionally linear and cannot be built: DOMNode
// identity is Equals, not ==, and the browser adapter hands out a fresh wrapper
// per traversal call, so the same node is a different map key each time it is
// read. Removing the DOM read-back entirely — deriving the move set from fiber
// order, which IS pointer-comparable — is the real fix and a larger change to
// the most delicate part of commit.
func buildCommittedChildOrderMatch(parseExpected, parseObserved []DOMNode) ([]int, bool) {
	if len(parseExpected) != len(parseObserved) {
		return nil, false
	}
	parseMatch := make([]int, len(parseObserved))
	parseUsed := make([]bool, len(parseExpected))
	parseCursor := 0
	for parseIndex, parseObservedNode := range parseObserved {
		parseFound := -1
		// Forward from the cursor, then wrap: together these cover the whole
		// list exactly once.
		for parseOffset := range parseExpected {
			parseExpectedIndex := parseCursor + parseOffset
			if parseExpectedIndex >= len(parseExpected) {
				parseExpectedIndex -= len(parseExpected)
			}
			if parseUsed[parseExpectedIndex] {
				continue
			}
			if IsSameDOMNode(parseExpected[parseExpectedIndex], parseObservedNode) {
				parseFound = parseExpectedIndex
				break
			}
		}
		if parseFound < 0 {
			return nil, false
		}
		parseUsed[parseFound] = true
		parseMatch[parseIndex] = parseFound
		parseCursor = parseFound + 1
		if parseCursor >= len(parseExpected) {
			parseCursor = 0
		}
	}
	return parseMatch, true
}

// committedChildOrderLISState stores one LIS tail value and its source index.
type committedChildOrderLISState struct {
	parseValue int
	parseIndex int
}

// markCommittedChildOrderKept marks the expected indices forming the longest
// increasing subsequence of the observed→expected match — the children whose
// relative DOM order is already correct — using the O(n log n) patience
// algorithm. Everything unmarked must move.
func markCommittedChildOrderKept(parseMatch []int, parseExpectedLen int) ([]bool, int) {
	parseTails := make([]committedChildOrderLISState, 0, len(parseMatch))
	parsePrev := make([]int, len(parseMatch))
	for parseIndex, parseValue := range parseMatch {
		parseLo, parseHi := 0, len(parseTails)
		for parseLo < parseHi {
			parseMid := (parseLo + parseHi) / 2
			if parseTails[parseMid].parseValue < parseValue {
				parseLo = parseMid + 1
			} else {
				parseHi = parseMid
			}
		}
		if parseLo > 0 {
			parsePrev[parseIndex] = parseTails[parseLo-1].parseIndex
		} else {
			parsePrev[parseIndex] = -1
		}
		parseState := committedChildOrderLISState{parseValue: parseValue, parseIndex: parseIndex}
		if parseLo == len(parseTails) {
			parseTails = append(parseTails, parseState)
		} else {
			parseTails[parseLo] = parseState
		}
	}
	parseKept := make([]bool, parseExpectedLen)
	parseCount := len(parseTails)
	if parseCount > 0 {
		for parseCursor := parseTails[parseCount-1].parseIndex; parseCursor >= 0; parseCursor = parsePrev[parseCursor] {
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
