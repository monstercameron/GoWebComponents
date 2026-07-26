package runtime

import (
	"strings"
	"testing"
	"time"
)

// Regressions found reviewing the v5 renderer. Each test names the defect it
// pins rather than the function it calls, because the function is not the point.

// A handler whose prop name is not in propMetaCache must still come off the node.
//
// The removal path asked getPropMeta — a NAME-keyed table — how to unset a prop,
// and for anything outside that hand-written table it answered "remove the
// attribute" while the value had been installed as a property. Every event below
// survived its own removal before domPropUnsetsAsProperty; the three in the
// table did not, which is why the bug stayed invisible.
func TestUnlistedEventHandlerIsRemovedFromTheNode(parseT *testing.T) {
	parseUnlisted := []string{
		"onmouseover", "onmouseout", "onpointerenter", "onpointerleave",
		"onpointercancel", "onkeypress", "onpaste", "oncopy", "ontoggle",
		"onfocusin", "onfocusout", "onauxclick", "onbeforeinput",
		"ondragenter", "ondragleave", "ontouchcancel", "onanimationstart",
		"ontransitionstart", "onselect", "oninvalid",
	}
	parseListed := []string{"onclick", "oninput", "onblur"}

	isStillAttached := func(parseName string) bool {
		parseT.Helper()
		parseAdapter := newTestDOMAdapter()
		parseScheduler := newTestScheduler()
		parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
		parseApp := parseAdapter.CreateElement("div")

		parseHandler := func() {}
		renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
			CreateElement("section", nil,
				CreateElement("button", map[string]any{"id": "b", parseName: parseHandler}, "x"),
			))
		parseButton := findNodeByID(parseApp, "b")
		if parseButton == nil || parseButton.properties[parseName] == nil {
			parseT.Fatalf("setup: %s was not installed as a property", parseName)
		}

		renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
			CreateElement("section", nil,
				CreateElement("button", map[string]any{"id": "b"}, "x"),
			))
		return parseButton.properties[parseName] != nil
	}

	for _, parseName := range parseListed {
		if isStillAttached(parseName) {
			parseT.Errorf("%s (in propMetaCache) survived removal", parseName)
		}
	}
	parseSurvived := []string{}
	for _, parseName := range parseUnlisted {
		if isStillAttached(parseName) {
			parseSurvived = append(parseSurvived, parseName)
		}
	}
	if len(parseSurvived) > 0 {
		parseT.Errorf("%d/%d handlers outside propMetaCache survived removal: %v",
			len(parseSurvived), len(parseUnlisted), parseSurvived)
	}
}

// A string-valued attribute must still be removed as an ATTRIBUTE. The fix above
// keys off the value's type, so this is the other half of the same contract:
// getting it wrong the other way would leave stale attributes in the DOM.
func TestStringAttributeIsStillRemovedAsAnAttribute(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseApp := parseAdapter.CreateElement("div")

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
		CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "d", "data-tag": "keep", "title": "hello"}, "x"),
		))
	parseNode := findNodeByID(parseApp, "d")
	if parseNode == nil || parseNode.attributes["data-tag"] != "keep" {
		parseT.Fatal("setup: expected data-tag to be applied as an attribute")
	}

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
		CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "d"}, "x"),
		))

	if parseValue, hasValue := parseNode.attributes["data-tag"]; hasValue && parseValue != "" {
		parseT.Errorf("data-tag=%q survived removal; string props must still be removed as attributes", parseValue)
	}
	if parseValue, hasValue := parseNode.attributes["title"]; hasValue && parseValue != "" {
		parseT.Errorf("title=%q survived removal", parseValue)
	}
}

// P2.2 must defer through the REAL scheduling path, not only when a test writes
// fiber.updateLane by hand.
//
// The existing accept test set the lane on a current-tree fiber itself and then
// asserted only that the work eventually rendered — true whether or not anything
// was deferred. Driving it through ScheduleUpdateForFiberWithOrigin exposed that
// the WIP clone dropped updateLane, so laneAdmitsFiber saw "no recorded lane"
// and admitted every fiber: nothing was ever deferred on any application path.
func TestLanes_BackgroundWorkIsDeferredThroughRealScheduling(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	parseContainer := parseAdapter.CreateElement("div")

	parseBackgroundRenders := 0
	parseBackgroundChild := func() *Element {
		parseBackgroundRenders++
		return CreateElement("span", map[string]any{}, "bg")
	}
	parseInputRenders := 0
	parseInputChild := func() *Element {
		parseInputRenders++
		return CreateElement("em", map[string]any{}, "in")
	}
	parseApp := func() *Element {
		return CreateElement("section", map[string]any{},
			CreateElement(parseBackgroundChild, map[string]any{}),
			CreateElement(parseInputChild, map[string]any{}),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	parseAfterMount := parseBackgroundRenders
	parseInputAfterMount := parseInputRenders

	parseSection := parseRt.currentRoot.child.child
	parseBackgroundFiber := parseSection.child
	parseInputFiber := parseBackgroundFiber.sibling
	if parseBackgroundFiber == nil || parseInputFiber == nil {
		parseT.Fatal("expected two child fibers")
	}

	// Background work arrives first, input work second, so the coalesced pass
	// runs at input priority with a background-marked fiber inside it.
	parseRt.ScheduleUpdateForFiberWithOrigin(parseBackgroundFiber, "background")
	parseRt.ScheduleUpdateForFiberWithOrigin(parseInputFiber, "input")

	if len(parseScheduler.timeouts) == 0 {
		parseT.Fatal("expected a scheduled pass")
	}
	parseFirstPass := parseScheduler.timeouts[0]
	parseScheduler.timeouts = parseScheduler.timeouts[1:]
	parseFirstPass()

	if parseInputRenders <= parseInputAfterMount {
		parseT.Fatal("setup: the input-lane child did not render in the input pass")
	}
	if parseBackgroundRenders > parseAfterMount {
		parseT.Errorf("background-lane child rendered inside an input-lane pass (%d -> %d): deferral did not fire",
			parseAfterMount, parseBackgroundRenders)
	}
	if !parseRt.schedulerState.lanes.pending[UpdateLaneBackground] {
		parseT.Error("the deferred lane was not recorded as pending, so no follow-up pass is owed")
	}

	// Deferred is not dropped: the follow-up pass must land it.
	runScheduledTimeouts(parseScheduler)
	if parseBackgroundRenders <= parseAfterMount {
		parseT.Error("deferred background work never rendered; deferral must not strand it")
	}
}

// The lane must survive BOTH clone paths — the update clone and the bailout
// clone — since either can be how a marked fiber reaches the work loop.
func TestLanes_CloneCarriesTheUpdateLane(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Reset: true})

	parseParent := &Fiber{typeOf: "div"}
	parseOld := &Fiber{typeOf: "span", updateLane: UpdateLaneBackground, dirty: true}
	parseParent.alternate = &Fiber{typeOf: "div", child: parseOld}

	parseUpdated := parseRt.buildUpdatedFiber(parseParent, parseOld, CreateElement("span", nil))
	if parseUpdated.updateLane != UpdateLaneBackground {
		parseT.Errorf("buildUpdatedFiber dropped the lane: got %v", parseUpdated.updateLane)
	}

	parseBailoutParent := &Fiber{typeOf: "div", alternate: &Fiber{typeOf: "div", child: parseOld}}
	parseRt.cloneChildFibers(parseBailoutParent)
	if parseBailoutParent.child == nil || parseBailoutParent.child.updateLane != UpdateLaneBackground {
		parseT.Errorf("cloneChildFibers dropped the lane: got %v", parseBailoutParent.child.updateLane)
	}
}

// Duplicate sibling keys are reported. The in-order fast path cannot see them
// (trailing appends are validated for castability only), and the slow path
// tolerates rather than resolves them, so the only real fix is upstream — which
// means the framework has to say so.
func TestDuplicateSiblingKeysAreReported(parseT *testing.T) {
	if !hookThreadingGuardEnabled {
		parseT.Skip("dev-only diagnostic; stripped from production builds")
	}
	duplicateKeyWarned.Store(false)
	defer duplicateKeyWarned.Store(false)
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := &Runtime{}
	parseParent := &Fiber{typeOf: "ul"}
	parseRt.reconcileChildren(parseParent, []any{
		CreateElement("li", map[string]any{"key": "a"}),
		CreateElement("li", map[string]any{"key": "b"}),
		CreateElement("li", map[string]any{"key": "a"}),
	})

	parseFound := false
	for _, parseDiagnostic := range GetDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, "share the key") {
			parseFound = true
		}
	}
	if !parseFound {
		parseT.Error("expected a duplicate-key diagnostic")
	}
}

// ...and a correctly keyed list must stay silent, or the warning is noise.
func TestUniqueSiblingKeysAreNotReported(parseT *testing.T) {
	if !hookThreadingGuardEnabled {
		parseT.Skip("dev-only diagnostic; stripped from production builds")
	}
	duplicateKeyWarned.Store(false)
	defer duplicateKeyWarned.Store(false)
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := &Runtime{}
	parseParent := &Fiber{typeOf: "ul"}
	parseRt.reconcileChildren(parseParent, []any{
		CreateElement("li", map[string]any{"key": "a"}),
		CreateElement("li", map[string]any{"key": "b"}),
	})

	for _, parseDiagnostic := range GetDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, "share the key") {
			parseT.Error("a uniquely keyed list must not produce a duplicate-key diagnostic")
		}
	}
}

// ui.PostAsync must not pick its target runtime by asking which one is
// rendering. The caller is off-loop by construction, so currentFiber describes
// someone else's stack; resolving through it made the same call site land on
// different runtimes depending on timing, and read render-goroutine-owned
// package state from a producer goroutine while doing it.
func TestPostAsyncGlobalTargetsTheGlobalRuntimeNotTheRenderingOne(parseT *testing.T) {
	parseGlobal := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler(), Reset: true})
	InitGlobalRuntime(Config{Reset: false})
	globalRuntimeMu.Lock()
	globalRuntime = parseGlobal
	globalRuntimeMu.Unlock()

	parseOther := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})

	// Stand in for "a second runtime is mid-render": currentFiber points into a
	// tree owned by parseOther, which is what ResolveRuntime followed.
	parseRenderingFiber := &Fiber{typeOf: "div", ownerRuntime: parseOther}
	SetCurrentFiber(parseRenderingFiber)
	defer SetCurrentFiber(nil)

	if parseResolved := ResolveRuntime(); parseResolved != parseOther {
		parseT.Fatal("setup: ResolveRuntime should follow currentFiber to the other runtime")
	}

	postAsyncGlobal(func() {})

	if parseOther.AsyncInboxDepth() != 0 {
		parseT.Error("the post landed on the runtime that happened to be rendering")
	}
	if parseGlobal.AsyncInboxDepth() != 1 {
		parseT.Errorf("expected the post on the global runtime's inbox, depth = %d", parseGlobal.AsyncInboxDepth())
	}
}

// An abandoned pass must not leave the committed tree pointing into it.
//
// Hypothesis under test: reuseFiberChildSubtree has the WIP parent adopt the
// SAME child objects as the committed tree, and sanitizeFiberSubtree then
// repoints those children's .parent at the WIP fiber. recoverWorkLoopState nils
// wipRoot and nothing else, so after an abandoned pass the committed tree's
// descendants can point at a fiber that was thrown away — and isFiberInCurrentTree
// walks .parent to decide whether an update is deliverable at all.
func TestAbandonedPassLeavesTheCommittedTreeReachable(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	parseContainer := parseAdapter.CreateElement("div")

	var parseSet func(any)
	parseLeaf := func() *Element {
		parseValue, parseSetter := GoUseState(parseRt, "before")
		parseSet = parseSetter
		return CreateElement("span", map[string]any{"id": "leaf"}, parseValue())
	}
	parseShell := func() *Element {
		return CreateElement("div", map[string]any{"id": "shell"},
			CreateElement(parseLeaf, map[string]any{}))
	}

	parseRt.Render(CreateElement(parseShell, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseCommittedRoot := parseRt.currentRoot

	// A root-only update: every descendant is clean, so the first child bails out
	// through reuseFiberChildSubtree and the repointing happens.
	parseRt.ScheduleUpdate()
	if len(parseScheduler.timeouts) == 0 {
		parseT.Fatal("expected a scheduled pass")
	}
	parseScheduler.timeouts = parseScheduler.timeouts[1:] // drop it: the pass never runs to completion

	// Drive exactly the part of the pass that performs the bailout, then abandon
	// it the way an unhandled panic would.
	for parseSteps := 0; parseSteps < 4 && parseRt.nextUnitOfWork != nil; parseSteps++ {
		parseRt.nextUnitOfWork = parseRt.performUnitOfWork(parseRt.nextUnitOfWork)
	}
	parseRt.recoverWorkLoopState()

	if parseRt.currentRoot != parseCommittedRoot {
		parseT.Fatal("setup: the committed root should be untouched by an abandoned pass")
	}

	// Every fiber still in the committed tree must still be reachable from it,
	// or updates targeting it are silently undeliverable.
	var parseWalk func(*Fiber, string)
	parseOrphans := 0
	parseWalk = func(parseFiber *Fiber, parseLabel string) {
		if parseFiber == nil {
			return
		}
		if !parseRt.isFiberInCurrentTree(parseFiber) {
			parseOrphans++
			parseT.Errorf("%s is in the committed tree but does not resolve back to currentRoot", parseLabel)
		}
		parseWalk(parseFiber.child, parseLabel+"/child")
		parseWalk(parseFiber.sibling, parseLabel+"/sibling")
	}
	parseWalk(parseCommittedRoot.child, "root/child")

	// The behavioural consequence: a state write after an abandoned pass must
	// still reach the DOM.
	parseSet("after")
	runScheduledTimeouts(parseScheduler)
	if parseNode := findNodeByID(parseContainer, "leaf"); parseNode == nil || collectFiberText(parseNode) != "after" {
		parseT.Errorf("state write after an abandoned pass never reached the DOM (orphaned fibers: %d)", parseOrphans)
	}
}

func collectFiberText(parseNode *testDOMNode) string {
	if parseNode == nil {
		return ""
	}
	if parseNode.text != "" {
		return parseNode.text
	}
	for _, parseChild := range parseNode.children {
		if parseTyped, parseOk := parseChild.(*testDOMNode); parseOk {
			if parseText := collectFiberText(parseTyped); parseText != "" {
				return parseText
			}
		}
	}
	return ""
}

// A portal nested inside a deleted subtree must take its DOM with it.
//
// commitDeletion handled the case where the deleted fiber IS a portal and
// nothing handled a portal below it — which is where every real portal lives,
// since a modal or tooltip is rendered by a component that gets conditionally
// unmounted. Both leak shapes are covered here: an ancestor that owns a DOM node
// (removed, never descended past) and a DOM-less component ancestor (descended,
// but carrying the wrong DOM parent, so the browser's parentNode check makes the
// removal a silent no-op). Effect cleanups run either way, so the leak is
// orphaned DOM wired to torn-down state.
func TestPortalInsideDeletedSubtreeIsRemoved(parseT *testing.T) {
	parseCases := []struct {
		name  string
		owner func(parseOverlay DOMNode, parseID string) *Element
	}{
		{
			name: "ancestor owns a DOM node",
			owner: func(parseOverlay DOMNode, parseID string) *Element {
				return CreateElement("div", map[string]any{"id": "wrapper"},
					CreateElement(PortalNodeType, map[string]any{"portalTargetNode": parseOverlay},
						CreateElement("div", map[string]any{"id": parseID}, "overlay"),
					))
			},
		},
		{
			name: "ancestor is a DOM-less component",
			owner: func(parseOverlay DOMNode, parseID string) *Element {
				parseComponent := func() *Element {
					return CreateElement(PortalNodeType, map[string]any{"portalTargetNode": parseOverlay},
						CreateElement("div", map[string]any{"id": parseID}, "overlay"),
					)
				}
				return CreateElement(parseComponent, map[string]any{})
			},
		},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parseAdapter := newTestDOMAdapter()
			parseScheduler := newTestScheduler()
			parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
			parseApp := parseAdapter.CreateElement("div")
			parseOverlay := parseAdapter.CreateElement("div")

			renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
				CreateElement("section", nil, parseCase.owner(parseOverlay, "portaled")))
			if findNodeByID(parseOverlay, "portaled") == nil {
				parseT.Fatal("setup: the portal child should be mounted in the overlay")
			}

			renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
				CreateElement("section", nil,
					CreateElement("p", map[string]any{"id": "replacement"}, "gone")))

			if findNodeByID(parseOverlay, "portaled") != nil {
				parseT.Error("portal content survived deletion of its owning subtree")
			}
		})
	}
}

// ...and a portal that is still mounted must be left alone, or the fix above
// would be a different bug: deleting a SIBLING must not empty the overlay.
func TestPortalSurvivesDeletionOfASibling(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseApp := parseAdapter.CreateElement("div")
	parseOverlay := parseAdapter.CreateElement("div")

	parsePortal := func() *Element {
		return CreateElement(PortalNodeType, map[string]any{"portalTargetNode": parseOverlay},
			CreateElement("div", map[string]any{"id": "kept"}, "overlay"))
	}

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
		CreateElement("section", nil,
			CreateElement("div", map[string]any{"id": "doomed"}, "bye"),
			parsePortal(),
		))
	if findNodeByID(parseOverlay, "kept") == nil {
		parseT.Fatal("setup: the portal child should be mounted in the overlay")
	}

	renderAndDrain(parseT, parseRt, parseScheduler, parseApp,
		CreateElement("section", nil, parsePortal()))

	if findNodeByID(parseApp, "doomed") != nil {
		parseT.Fatal("setup: the sibling should have been removed")
	}
	if findNodeByID(parseOverlay, "kept") == nil {
		parseT.Error("a still-mounted portal lost its content when a sibling was deleted")
	}
}

type orderProbeNode struct{ id int }

func (parseN *orderProbeNode) IsNull() bool { return parseN == nil }
func (parseN *orderProbeNode) Equals(parseOther DOMNode) bool {
	parseTyped, parseOk := parseOther.(*orderProbeNode)
	return parseOk && parseTyped != nil && parseTyped.id == parseN.id
}

// The child-order match must not be quadratic on the shapes lists actually
// produce. It feeds an O(n log n) LIS and used to restart its scan at index 0
// for every node, so the pair cost 22.5ms at 5000 rows with the LIS itself at
// ~0 — inside commitRoot, which no frame budget bounds.
//
// Asserted as a scaling ratio rather than a wall-clock bound so the test means
// the same thing on a slower machine: 4x the rows must not cost anything like
// 16x the time.
func TestChildOrderMatchScalesLinearly(parseT *testing.T) {
	build := func(parseN int) ([]DOMNode, []DOMNode) {
		parseExpected := make([]DOMNode, parseN)
		for parseI := range parseExpected {
			parseExpected[parseI] = &orderProbeNode{id: parseI}
		}
		// Rotation by half: the shape a filter being cleared produces, where the
		// re-placed rows land ahead of the retained ones.
		parseObserved := make([]DOMNode, parseN)
		for parseI := range parseObserved {
			parseObserved[parseI] = parseExpected[(parseI+parseN/2)%parseN]
		}
		return parseExpected, parseObserved
	}

	measure := func(parseN int) time.Duration {
		parseExpected, parseObserved := build(parseN)
		parseBest := time.Hour
		for parseRun := 0; parseRun < 5; parseRun++ {
			parseStart := time.Now()
			parseMatch, parseOk := buildCommittedChildOrderMatch(parseExpected, parseObserved)
			parseElapsed := time.Since(parseStart)
			if !parseOk || len(parseMatch) != parseN {
				parseT.Fatalf("n=%d: the two lists are the same set and must match", parseN)
			}
			if parseElapsed < parseBest {
				parseBest = parseElapsed
			}
		}
		return parseBest
	}

	parseSmall := measure(500)
	parseLarge := measure(2000)
	parseT.Logf("match cost: n=500 %v, n=2000 %v", parseSmall, parseLarge)

	// Quadratic would be ~16x for 4x the rows. Linear is ~4x. 8x leaves generous
	// room for timer noise while still failing a return to the nested scan.
	if parseSmall > 0 && parseLarge > parseSmall*8 {
		parseT.Errorf("4x the rows cost %.1fx the time (%v -> %v); the match is superlinear again",
			float64(parseLarge)/float64(parseSmall), parseSmall, parseLarge)
	}
}

// Correctness of the resumed scan on an arbitrary permutation, including one
// that forces the wrap-around, and on duplicate node values which must still
// match one-to-one.
func TestChildOrderMatchHandlesArbitraryPermutations(parseT *testing.T) {
	parseExpected := []DOMNode{
		&orderProbeNode{id: 0}, &orderProbeNode{id: 1}, &orderProbeNode{id: 2},
		&orderProbeNode{id: 3}, &orderProbeNode{id: 4},
	}
	parseObserved := []DOMNode{
		parseExpected[4], parseExpected[0], parseExpected[3],
		parseExpected[1], parseExpected[2],
	}
	parseMatch, parseOk := buildCommittedChildOrderMatch(parseExpected, parseObserved)
	if !parseOk {
		parseT.Fatal("same set must match")
	}
	parseWant := []int{4, 0, 3, 1, 2}
	for parseI := range parseWant {
		if parseMatch[parseI] != parseWant[parseI] {
			parseT.Errorf("match[%d] = %d, want %d (full: %v)", parseI, parseMatch[parseI], parseWant[parseI], parseMatch)
		}
	}

	// A node value appearing twice must consume two distinct expected slots.
	parseDup := &orderProbeNode{id: 9}
	parseDupExpected := []DOMNode{parseDup, &orderProbeNode{id: 1}, parseDup}
	parseDupObserved := []DOMNode{parseDup, parseDup, &orderProbeNode{id: 1}}
	parseDupMatch, parseDupOk := buildCommittedChildOrderMatch(parseDupExpected, parseDupObserved)
	if !parseDupOk {
		parseT.Fatal("duplicate values are still the same set")
	}
	parseSeen := map[int]bool{}
	for _, parseValue := range parseDupMatch {
		if parseSeen[parseValue] {
			parseT.Errorf("expected index %d matched twice: %v", parseValue, parseDupMatch)
		}
		parseSeen[parseValue] = true
	}

	// A genuinely different set must still be rejected.
	if _, parseOk := buildCommittedChildOrderMatch(
		[]DOMNode{&orderProbeNode{id: 0}},
		[]DOMNode{&orderProbeNode{id: 7}},
	); parseOk {
		parseT.Error("a different node set must not report a match")
	}
}
