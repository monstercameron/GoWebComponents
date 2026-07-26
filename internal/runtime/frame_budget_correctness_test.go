package runtime

import (
	"strconv"
	"testing"
)

// Time-slicing must not lose the tree.
//
// Turning FrameBudgetMs on by default broke the Example 201 core-* and
// enterprise-* scenarios outright: the tree never rendered and the browser
// harness timed out waiting for it, in both development and production builds.
// That is a correctness defect in a shipped option, not a tuning question — an
// application that sets FrameBudgetMs today gets a blank subtree.
//
// P2.5 was supposed to make interrupted passes safe, and interruptRestartIsSafe
// asserts it did. These tests exist because the browser says otherwise, and they
// are native so the failure can be debugged in seconds rather than in a
// 20-second browser round trip.
//
// The budget is set absurdly small so the work loop yields at every opportunity,
// which is the same condition a real budget reaches on a large tree — just
// reached sooner.

// buildSlicedListTree returns a keyed list of the requested size.
func buildSlicedListTree(parseCount int) *Element {
	parseChildren := make([]any, 0, parseCount)
	for parseIndex := range parseCount {
		parseChildren = append(parseChildren, CreateElement("li", map[string]any{
			"key":   "row-" + strconv.Itoa(parseIndex),
			"class": "row",
		}, "item-"+strconv.Itoa(parseIndex)))
	}
	return CreateElement("ul", map[string]any{"class": "list"}, parseChildren...)
}

// countDescendants walks the committed mock tree.
func countDescendants(parseAdapter *testDOMAdapter, parseNode DOMNode) int {
	parseTotal := 0
	for parseChild := parseAdapter.GetFirstChild(parseNode); !IsDOMNodeNull(parseChild); parseChild = parseAdapter.GetNextSibling(parseChild) {
		parseTotal += 1 + countDescendants(parseAdapter, parseChild)
	}
	return parseTotal
}

// TestFrameBudget_SlicedMountCommitsTheWholeTree is the defect, stated directly.
func TestFrameBudget_SlicedMountCommitsTheWholeTree(parseT *testing.T) {
	const parseRows = 200

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter:    parseAdapter,
		Scheduler:     parseScheduler,
		FrameBudgetMs: 0.0001, // yields at the first check, every slice
		Reset:         true,
	})
	parseContainer := parseAdapter.CreateElement("div")

	parseRt.Render(buildSlicedListTree(parseRows), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseCommitted := countDescendants(parseAdapter, parseContainer)
	// ul + 200 li + 200 text nodes, though the exact total depends on direct-text
	// storage; what matters is that it is not a fraction of the tree.
	if parseCommitted < parseRows {
		parseT.Errorf("a sliced mount committed %d nodes for a %d-row list; the pass yielded and never finished, which is what makes a browser scenario time out with an empty tree",
			parseCommitted, parseRows)
	}
}

// TestFrameBudget_SlicedUpdateCommitsTheWholeTree covers the update path, which
// is where the failing browser scenarios actually live: core-render and
// enterprise-subtree-update both re-render an existing tree.
func TestFrameBudget_SlicedUpdateCommitsTheWholeTree(parseT *testing.T) {
	const parseRows = 150

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter:    parseAdapter,
		Scheduler:     parseScheduler,
		FrameBudgetMs: 0.0001,
		Reset:         true,
	})
	parseContainer := parseAdapter.CreateElement("div")

	parseSize := parseRows
	parseComponent := func() *Element {
		return buildSlicedListTree(parseSize)
	}
	parseRt.Render(CreateElement(parseComponent, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	parseAfterMount := countDescendants(parseAdapter, parseContainer)
	if parseAfterMount < parseRows {
		parseT.Fatalf("mount committed %d nodes for %d rows; the update case cannot be judged from a broken mount",
			parseAfterMount, parseRows)
	}

	// Clear, then repopulate — the shape core-render drives.
	parseSize = 0
	parseRt.ScheduleUpdate()
	runScheduledTimeouts(parseScheduler)

	parseSize = parseRows
	parseRt.ScheduleUpdate()
	runScheduledTimeouts(parseScheduler)

	parseAfterRefill := countDescendants(parseAdapter, parseContainer)
	if parseAfterRefill < parseRows {
		parseT.Errorf("after clear and refill a sliced pass committed %d nodes for %d rows; this is the core-render shape that renders nothing in the browser",
			parseAfterRefill, parseRows)
	}
}

// NOT TESTED NATIVELY: the mid-pass lost update.
//
// The defect that made time-slicing freeze the Example 201 core-* scenarios is
// an update scheduled while a pass is ALREADY WALKING the fibers it touches.
// coalesceScheduledUpdateLocked folded such an update into pendingLane and
// acted only when the new work was more urgent than the running pass, so a
// same-lane update was recorded and never run. The fix is
// scheduleFollowUpForInFlightUpdate, called from commitRoot.
//
// A native reproduction was attempted and DELETED rather than kept. It passed
// with the fix removed, because arranging the pass to have already visited the
// specific fiber before the update arrives is not something the test scheduler
// can do deterministically — a state setter marks the fiber dirty, and if the
// pass has not reached it yet the running pass renders it anyway. A test that
// passes without the fix reports coverage it does not have, which is worse than
// no test.
//
// The reproduction that works is the browser: with the budget on and the fix
// absent, core-render times out with zero items and the runtime's own counters
// freeze — passes 8, units 108, commits 2, items 0 — showing the work loop stop
// rather than spin. TestExample201SlicingHang samples exactly that, and
// TestExample201PhaseTotalsProbe fails outright without the fix.
