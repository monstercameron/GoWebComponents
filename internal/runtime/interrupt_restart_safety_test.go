package runtime

import "testing"

// v5 P2.5 / threat T12 — an interrupt must never blank the committed tree.
//
// The removed rebuildWIPRootForInterruptLocked reset the work-in-progress root
// mid-pass via acquireWorkInProgress(currentRoot). Because currentRoot.alternate
// IS the in-flight wipRoot in the two-fiber cycle, that zeroed the live root in
// place and cleared its child chain. Between slices it read as a clean restart;
// landing inside performUnitOfWork it was not, because the work loop writes its
// own return value back over nextUnitOfWork and discards the reset — after which
// commitRoot sees wipRoot.child == nil, commits nothing, and installs a
// childless currentRoot.
//
// These tests reproduce the conditions that made it reachable and assert the
// tree survives.

// TestInterruptDuringRender_DoesNotBlankCommittedTree fires a higher-lane
// update from inside a component's render function — which executes inside
// performUnitOfWork, the exact window that made T12 destructive.
//
// The tree must be MOUNTED first. During the very first render currentRoot is
// still nil, so scheduleUpdateWithLane returns early and the interrupt path is
// unreachable — an interrupt test that renders only once proves nothing.
func TestInterruptDuringRender_DoesNotBlankCommittedTree(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	parseContainer := parseAdapter.CreateElement("div")

	isArmed := false
	isInterruptObserved := false
	var setLabel func(any)

	parseLeaf := func() *Element {
		return CreateElement("span", map[string]any{}, "leaf")
	}
	parseApp := func() *Element {
		// A real state setter drives the second pass: ScheduleUpdate alone
		// reuses the committed elements, so everything memoizes and nothing
		// re-renders. The interrupt fires from THIS component because it is the
		// one the setter marks dirty — the leaf below memoizes (same type,
		// equal props and children) and never re-renders.
		//
		// By the time this body runs, the work loop has advanced past wipRoot
		// onto this fiber, so the pass is genuinely in flight — which is the
		// condition the interrupt path requires.
		_, parseSet := GoUseState(parseRt, "initial")
		setLabel = parseSet
		if isArmed && !isInterruptObserved {
			isInterruptObserved = true
			// Strictly higher than the input lane carrying this pass.
			parseRt.ScheduleUpdateWithLane(UpdateLaneSync)
		}
		return CreateElement("section", map[string]any{},
			CreateElement(parseLeaf, map[string]any{}),
		)
	}

	// Mount, so currentRoot exists and the interrupt path becomes reachable.
	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	if setLabel == nil {
		parseT.Fatal("expected the state setter to be captured")
	}

	// Second pass on the input lane; the leaf interrupts it mid-render.
	isArmed = true
	setLabel("updated")
	runScheduledTimeouts(parseScheduler)

	if !isInterruptObserved {
		parseT.Fatal("test did not exercise the interrupt path")
	}
	if parseRt.schedulerState.interruptedWork == 0 {
		parseT.Fatal("no interrupt was recorded; the test is not reaching the T12 window")
	}

	// The committed tree must still be there. Under the old in-place rebuild
	// this could commit nothing and leave a childless root.
	parseRoot := parseContainer.(*testDOMNode)
	if len(parseRoot.children) == 0 {
		parseT.Fatal("committed tree was blanked by a mid-render interrupt (T12)")
	}
	parseSection := parseRoot.children[0].(*testDOMNode)
	if len(parseSection.children) == 0 {
		parseT.Fatal("section lost its children to a mid-render interrupt (T12)")
	}

	// The runtime's own view must agree: a childless currentRoot is the
	// signature of the bug even when the DOM happens to look intact.
	if parseRt.currentRoot == nil || parseRt.currentRoot.child == nil {
		parseT.Fatal("currentRoot has no child after an interrupted pass (T12)")
	}
}

// TestInterruptDuringRender_HigherLaneStillRuns proves the deferral does not
// strand the interrupting update: it must land in a follow-up pass rather than
// being silently dropped, which would trade one bug for another.
func TestInterruptDuringRender_HigherLaneStillRuns(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	parseContainer := parseAdapter.CreateElement("div")

	parseRenderCount := 0
	isArmed := false
	isInterruptObserved := false
	var setLabel func(any)

	parseLeaf := func() *Element {
		return CreateElement("span", map[string]any{}, "leaf")
	}
	parseApp := func() *Element {
		// Counted here, not in the leaf: the leaf memoizes across updates
		// (same type, equal props and children), so it renders once ever and
		// would make this assertion vacuous.
		parseRenderCount++
		_, parseSet := GoUseState(parseRt, "initial")
		setLabel = parseSet
		if isArmed && !isInterruptObserved {
			isInterruptObserved = true
			parseRt.ScheduleUpdateWithLane(UpdateLaneSync)
		}
		return CreateElement("section", map[string]any{},
			CreateElement(parseLeaf, map[string]any{}),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	parseMountedRenders := parseRenderCount
	parseCommitsAfterMount := parseRt.profiling.commitCount

	isArmed = true
	setLabel("updated")
	runScheduledTimeouts(parseScheduler)

	if !isInterruptObserved {
		parseT.Fatal("test did not exercise the interrupt path")
	}
	if parseRenderCount <= parseMountedRenders {
		parseT.Fatalf("the interrupted pass itself never rendered: renders=%d", parseRenderCount)
	}

	// The observable is a follow-up PASS, not a follow-up component render:
	// ScheduleUpdateWithLane raises the lane without marking any fiber dirty,
	// so the deferred pass legitimately finds everything memoized. In real
	// usage the interrupting update marks its own fibers (ScheduleUpdateForFiber)
	// and that work lands in this pass. What must be true either way is that a
	// pass ran at all — a deferral that never fires would strand the update.
	if parseRt.profiling.commitCount < parseCommitsAfterMount+2 {
		parseT.Fatalf("deferred interrupt produced no follow-up commit: commits=%d, want >= %d",
			parseRt.profiling.commitCount, parseCommitsAfterMount+2)
	}
	if parseRt.pendingInterruptLane != 0 {
		parseT.Fatalf("pendingInterruptLane should be consumed by the follow-up commit, got %v", parseRt.pendingInterruptLane)
	}
}

// TestInterruptDeferral_DoesNotResetWorkCursor pins the mechanism itself: the
// interrupt records a lane and leaves the in-flight pass alone.
func TestInterruptDeferral_DoesNotResetWorkCursor(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT", props: map[string]any{"children": []any{}}, dirty: true}

	parseRt.ScheduleUpdateWithLane(UpdateLaneDefault)
	parseRt.schedulerState.currentLane = UpdateLaneDefault

	parseCursor := &Fiber{typeOf: "component", parent: parseRt.wipRoot}
	parseRt.nextUnitOfWork = parseCursor
	parseWipBefore := parseRt.wipRoot

	parseRt.ScheduleUpdateWithLane(UpdateLaneSync)

	if parseRt.nextUnitOfWork != parseCursor {
		parseT.Error("interrupt moved the work cursor; the in-flight pass must be left alone")
	}
	if parseRt.wipRoot != parseWipBefore {
		parseT.Error("interrupt replaced the work-in-progress root; that is the T12 failure")
	}
	if parseRt.wipRoot != nil && parseRt.wipRoot.typeOf == nil {
		parseT.Error("work-in-progress root was zeroed in place (T12)")
	}
	if parseRt.pendingInterruptLane != UpdateLaneSync {
		parseT.Errorf("expected the sync lane to be deferred, got %v", parseRt.pendingInterruptLane)
	}
}
