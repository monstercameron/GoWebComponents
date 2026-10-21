package runtime

import "testing"

// newP7Runtime builds a plain runtime with the test DOM and scheduler.
func newP7Runtime() (*Runtime, DOMNode, *testScheduler) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
	return parseRt, parseAdapter.CreateElement("div"), parseScheduler
}

// TestP7EffectCleanupPanicIsContained pins that a panicking cleanup is routed
// to the error boundary and does not abort the remaining effects.
func TestP7EffectCleanupPanicIsContained(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()
	parseRt, parseContainer, parseScheduler := newP7Runtime()
	parseBoundary := NewErrorBoundaryType()

	var parseSet func(any)
	parseSiblingRuns := 0
	parseLaterRuns := 0

	parseBad := func() *Element {
		parseGet, parseSetter := GoUseState(parseRt, 0)
		parseSet = parseSetter
		parseN := parseGet()
		GoUseEffect(func() func() {
			return func() { panic("cleanup boom") }
		}, parseN)
		GoUseEffect(func() func() {
			parseSiblingRuns++
			return nil
		}, parseN)
		return CreateElement("span", nil, "bad")
	}
	parseLater := func() *Element {
		GoUseEffect(func() func() {
			parseLaterRuns++
			return nil
		})
		return CreateElement("em", nil, "later")
	}

	parseRt.Render(CreateElement(parseBoundary, map[string]any{
		"errorFallback": func(parseErr error, reset func()) *Element {
			return CreateElement("p", nil, "fallback: "+parseErr.Error())
		},
	}, CreateElement(parseBad, nil), CreateElement(parseLater, nil)), parseContainer)
	flushScheduledWork(parseScheduler)
	parseSiblingBefore, parseLaterBefore := parseSiblingRuns, parseLaterRuns

	parseSet(1)
	flushScheduledWork(parseScheduler)
	for _, parseD := range GetDiagnostics() {
		parseT.Log(parseD.Message)
	}

	if parseSiblingRuns != parseSiblingBefore+1 {
		parseT.Errorf("sibling effect runs = %d, want %d; a cleanup panic aborted the remaining effects", parseSiblingRuns, parseSiblingBefore+1)
	}
	if parseLaterRuns < parseLaterBefore {
		parseT.Errorf("later fiber effect did not run")
	}
	if parseGot := textFromNode(parseContainer); parseGot != "fallback: cleanup boom" {
		parseT.Errorf("boundary text = %q, want fallback", parseGot)
	}
}

// TestP7SecondPanicInSameBoundaryIsHandled pins that two failing children under
// one boundary both route to it.
func TestP7SecondPanicInSameBoundaryIsHandled(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()
	parseRt, parseContainer, parseScheduler := newP7Runtime()
	parseBoundary := NewErrorBoundaryType()
	parseFailRuns := 0

	parseFail := func() *Element {
		GoUseEffect(func() func() { parseFailRuns++; panic("effect boom") })
		GoUseEffect(func() func() { parseFailRuns++; panic("effect boom 2") })
		return CreateElement("span", nil, "x")
	}
	parseRt.Render(CreateElement(parseBoundary, map[string]any{
		"errorFallback": func(parseErr error, reset func()) *Element {
			return CreateElement("p", nil, "fallback")
		},
	}, CreateElement(parseFail, nil)), parseContainer)
	flushScheduledWork(parseScheduler)

	if parseFailRuns != 2 {parseT.Errorf("effects run=%d want 2", parseFailRuns)}
	if parseGot := textFromNode(parseContainer); parseGot != "fallback" {
		parseT.Errorf("boundary text = %q, want fallback", parseGot)
	}
}

// TestP7TypedEffectCleanupNotRunDuringDiscardedRender pins that UseEffectOf
// leaves the outgoing cleanup to commit: a render interrupted by a higher lane
// is discarded, and the live effect must still be exactly one setup deep.
func TestP7TypedEffectCleanupNotRunDuringDiscardedRender(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newP7Runtime()
	parseSetups, parseCleanups := 0, 0
	var parseSet func(any)
	isArmed := false
	isFired := false
	parseComp := func() *Element {
		parseGet, parseSetter := GoUseState(parseRt, 0)
		parseSet = parseSetter
		GoUseEffectOf(func() func() {
			parseSetups++
			return func() { parseCleanups++ }
		}, parseGet())
		if isArmed && !isFired {
			isFired = true
			parseRt.ScheduleUpdateWithLane(UpdateLaneSync)
			if parseCleanups != 0 {
				parseT.Errorf("cleanup ran during render (cleanups=%d)", parseCleanups)
			}
		}
		return CreateElement("span", nil, "x")
	}
	parseRt.Render(CreateElement(parseComp, nil), parseContainer)
	flushScheduledWork(parseScheduler)

	isArmed = true
	parseSet(1)
	flushScheduledWork(parseScheduler)
	if !isFired {
		parseT.Fatal("render hook did not fire")
	}
	if parseSetups-parseCleanups != 1 {
		parseT.Errorf("setups=%d cleanups=%d; live effects must be exactly 1", parseSetups, parseCleanups)
	}
}

// TestP7RenderFlushesDeferredPassiveEffects pins that Render and Hydrate settle
// the previous commit's deferred passive effects before replacing the pass
// bookkeeping, as scheduleUpdateWithLane already does.
func TestP7RenderFlushesDeferredPassiveEffects(parseT *testing.T) {
	for _, parseName := range []string{"Render", "Hydrate"} {
		parseRt, parseContainer, parseScheduler := newPaintSplitRuntime(parseT)
		parseRuns := 0
		parseComp := func() *Element {
			GoUseEffect(func() func() { parseRuns++; return nil }, 1)
			return CreateElement("span", nil, "x")
		}
		parseRt.Render(CreateElement(parseComp, nil), parseContainer)
		// Drive to the commit, but stop before the deferred passive drain runs.
		for !parseRt.passiveDrainScheduled && parseScheduler.runOne() {
		}
		if !parseRt.passiveDrainScheduled || parseRuns != 0 {
			parseT.Fatalf("%s: setup failed: drainScheduled=%v runs=%d", parseName, parseRt.passiveDrainScheduled, parseRuns)
		}
		parseOther := func() *Element { return CreateElement("b", nil, "y") }
		if parseName == "Render" {
			parseRt.Render(CreateElement(parseOther, nil), parseContainer)
		} else {
			parseRt.Hydrate(CreateElement(parseOther, nil), parseContainer)
		}
		if parseRuns != 1 {
			parseT.Errorf("%s: first commit's passive effect ran %d times before the call returned, want 1", parseName, parseRuns)
		}
		parseScheduler.flush()
	}
}
// TestP7RestoreSnapshotRecomputesDerivedDependents pins that restoring a source
// atom updates derived atoms and schedules their subscribers.
func TestP7RestoreSnapshotRecomputesDerivedDependents(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{}
	parseRt.atomRegistry.InitAtom("src", 1)
	if parseErr := parseRt.RegisterDerivedAtom("doubled", []string{"src"}, func() any {
		parseV, _ := parseRt.GetAtomValue("src")
		return parseV.(int) * 2
	}); parseErr != nil {
		parseT.Fatalf("register derived: %v", parseErr)
	}
	parseFiber := newTestFiber("derived-subscriber")
	parseRt.atomRegistry.Subscribe("doubled", parseFiber)

	if parseErr := parseRt.RestoreAtomSnapshot(map[string]any{"src": 5}); parseErr != nil {
		parseT.Fatalf("restore: %v", parseErr)
	}
	if parseV, _ := parseRt.GetAtomValue("doubled"); parseV != 10 {
		parseT.Errorf("derived value = %v after restore, want 10", parseV)
	}
	if !parseFiber.needsUpdate {
		parseT.Error("derived subscriber was not scheduled after restore")
	}
}
// TestP7NilChildKeepsSiblingSlot pins positional stability: toggling an
// optional child between present and absent must not remount the siblings
// after it.
func TestP7NilChildKeepsSiblingSlot(parseT *testing.T) {
	for _, parseWithFragment := range []bool{false} {
		parseRt, parseContainer, parseScheduler := newP7Runtime()
		parseMounts := 0
		var parseSet func(any)
		parseA := func() *Element { return CreateElement("i", nil, "a") }
		parseX := func() *Element { return CreateElement("u", nil, "x") }
		parseB := func() *Element {
			GoUseEffect(func() func() { parseMounts++; return nil }, 1)
			return CreateElement("b", nil, "b")
		}
		parseApp := func() *Element {
			parseGet, parseSetter := GoUseState(parseRt, false)
			parseSet = parseSetter
			var parseOptional *Element
			if parseGet() {
				parseOptional = CreateElement(parseX, nil)
			}
			if parseWithFragment {
				return CreateElement("div", nil, CreateElement(parseA, nil), parseOptional, CreateElement(parseB, nil), CreateElement("FRAGMENT", nil, CreateElement("s", nil, "s")))
			}
			return CreateElement("div", nil, CreateElement(parseA, nil), parseOptional, CreateElement(parseB, nil))
		}
		parseRt.Render(CreateElement(parseApp, nil), parseContainer)
		flushScheduledWork(parseScheduler)
		for _, parseValue := range []bool{true, false, true} {
			parseSet(parseValue)
			flushScheduledWork(parseScheduler)
		}
		if parseMounts != 1 {
			parseT.Errorf("fragment=%v: B mounted %d times across toggles, want 1 (sibling remounted)", parseWithFragment, parseMounts)
		}
	}
}
// TestP7HydrationEmptyTextChildDoesNotConsumeNextNode pins that an empty text
// child, which the server never serializes, does not claim the next real node.
func TestP7HydrationEmptyTextChildDoesNotConsumeNextNode(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseServerP := parseAdapter.CreateElement("p")
	parseServerSpan := parseAdapter.CreateElement("span")
	parseAdapter.AppendChild(parseServerSpan, parseAdapter.CreateTextNode("x"))
	parseAdapter.AppendChild(parseServerP, parseServerSpan)
	parseAdapter.AppendChild(parseContainer, parseServerP)

	parseRt.Hydrate(CreateElement("p", nil, "", CreateElement("span", nil, "x")), parseContainer)
	for parseRt.updateScheduled && len(parseScheduler.timeouts) > 0 {
		runHydrationWork(parseT, parseScheduler)
	}
	parseKids := parseAdapter.GetChildren(parseAdapter.GetChildren(parseContainer)[0])
	hasServerSpan := false
	for _, parseKid := range parseKids {
		if parseKid == parseServerSpan {
			hasServerSpan = true
		}
	}
	if !hasServerSpan {
		parseT.Errorf("server span was discarded instead of adopted (children=%d)", len(parseKids))
	}
}