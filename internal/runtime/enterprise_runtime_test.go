package runtime

import (
	"strings"
	"testing"
)

func TestEnterpriseMultiInstanceRuntimeIsolation(parseT *testing.T) {
	parseFirst := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})
	parseSecond := NewRuntime(Config{DOMAdapter: newTestDOMAdapter()})

	if parseErr := parseFirst.SetAtomValue("shared", "first"); parseErr != nil {
		parseT.Fatalf("expected first atom set to succeed: %v", parseErr)
	}
	if parseErr := parseSecond.SetAtomValue("shared", "second"); parseErr != nil {
		parseT.Fatalf("expected second atom set to succeed: %v", parseErr)
	}
	if parseValue, _ := parseFirst.GetAtomValue("shared"); parseValue != "first" {
		parseT.Fatalf("expected first runtime atom registry to stay isolated, got %#v", parseValue)
	}
	if parseValue, _ := parseSecond.GetAtomValue("shared"); parseValue != "second" {
		parseT.Fatalf("expected second runtime atom registry to stay isolated, got %#v", parseValue)
	}

	parseFiber := &Fiber{typeOf: "component", ownerRuntime: parseFirst}
	SetCurrentFiber(parseFiber)
	parseID := GoUseId()
	SetCurrentFiber(nil)
	if !strings.HasPrefix(parseID, "gwc-") || parseFirst.idCounter != 1 || parseSecond.idCounter != 0 {
		parseT.Fatalf("expected GoUseId to use the owning runtime, id=%q first=%d second=%d", parseID, parseFirst.idCounter, parseSecond.idCounter)
	}
}

func TestEnterpriseSchedulerLanesCoalescePromoteAndBackpressure(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRoot := &Fiber{typeOf: "ROOT", props: map[string]any{"children": []any{}}, dirty: true}
	parseRt := NewRuntime(Config{
		Scheduler: parseScheduler,
		Limits:    RuntimeLimits{MaxQueuedUpdates: 1},
	})
	parseRt.currentRoot = parseRoot

	parseRt.ScheduleUpdateWithLane(UpdateLaneBackground)
	parseRt.schedulerState.currentLane = UpdateLaneBackground

	// v5 P2.5: InterruptedWork counts passes whose IN-FLIGHT work was
	// interrupted. A pass that is merely scheduled still has
	// nextUnitOfWork == wipRoot and has visited nothing, so an update arriving
	// in that window is ordinary coalescing, not an interrupt. Advance the
	// cursor off the root to model a pass that has actually started consuming.
	parseRt.nextUnitOfWork = &Fiber{typeOf: "component", parent: parseRt.wipRoot}

	parseRt.ScheduleUpdateWithLane(UpdateLaneInput)
	parseRt.ScheduleUpdateWithLane(UpdateLaneTransition)

	parseSnapshot := parseRt.SchedulerSnapshot()
	if parseSnapshot.PendingLane != "input" {
		parseT.Fatalf("expected pending lane to promote to input, got %#v", parseSnapshot)
	}
	if parseSnapshot.CoalescedUpdates != 1 || parseSnapshot.InterruptedWork != 1 || !parseSnapshot.Backpressure || parseSnapshot.DroppedUpdates != 1 {
		parseT.Fatalf("expected coalescing, interrupt, and backpressure counters, got %#v", parseSnapshot)
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one scheduled timeout after coalescing, got %d", len(parseScheduler.timeouts))
	}
	// The interrupt must be deferred, never applied in place: the pass keeps
	// its cursor and the higher lane is recorded for after the commit.
	if parseRt.pendingInterruptLane != UpdateLaneInput {
		parseT.Fatalf("expected the higher lane to be deferred to after commit, got %v", parseRt.pendingInterruptLane)
	}
	if parseRt.nextUnitOfWork == parseRt.wipRoot {
		parseT.Fatal("interrupt must not reset the work cursor mid-pass (v5 T12)")
	}
}

// TestSchedulerLanePromotionBeforePassStartsIsNotAnInterrupt pins the other
// half of the P2.5 distinction: a higher lane arriving while the pass is only
// scheduled needs no interrupt at all, because the pass has visited nothing and
// will pick the new work up naturally.
func TestSchedulerLanePromotionBeforePassStartsIsNotAnInterrupt(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRoot := &Fiber{typeOf: "ROOT", props: map[string]any{"children": []any{}}, dirty: true}
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = parseRoot

	parseRt.ScheduleUpdateWithLane(UpdateLaneBackground)
	parseRt.schedulerState.currentLane = UpdateLaneBackground
	// Deliberately leave nextUnitOfWork == wipRoot: scheduled, not started.
	parseRt.ScheduleUpdateWithLane(UpdateLaneInput)

	if parseSnapshot := parseRt.SchedulerSnapshot(); parseSnapshot.InterruptedWork != 0 {
		parseT.Fatalf("a not-yet-started pass cannot be interrupted, got InterruptedWork=%d", parseSnapshot.InterruptedWork)
	}
	if parseRt.pendingInterruptLane != 0 {
		parseT.Fatalf("no deferred interrupt should be recorded, got %v", parseRt.pendingInterruptLane)
	}
	// The lane still promotes — coalescing is unaffected.
	if parseSnapshot := parseRt.SchedulerSnapshot(); parseSnapshot.PendingLane != "input" {
		parseT.Fatalf("expected lane promotion to still apply, got %#v", parseSnapshot)
	}
}

func TestEnterpriseBoundedInternalStateFallsBackForEffectOverflow(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := NewRuntime(Config{Limits: RuntimeLimits{MaxPendingEffectFibers: 1}})
	parseFirst := &Fiber{typeOf: "first", effects: []Effect{{Fn: func() func() { return nil }}}}
	parseSecond := &Fiber{typeOf: "second", effects: []Effect{{Fn: func() func() { return nil }}}}

	parseRt.queuePendingEffectFiber(parseFirst)
	parseRt.queuePendingEffectFiber(parseSecond)

	if !parseRt.pendingEffectOverflow || parseRt.tracksPendingEffects || len(parseRt.pendingEffectFibers) != 0 {
		parseT.Fatalf("expected bounded pending-effect queue to fall back cleanly, overflow=%t tracks=%t len=%d", parseRt.pendingEffectOverflow, parseRt.tracksPendingEffects, len(parseRt.pendingEffectFibers))
	}
	if len(GetDiagnostics()) == 0 {
		parseT.Fatal("expected overflow diagnostic")
	}
}

func TestEnterpriseStrictModeDoubleRenderAndWarnings(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := NewRuntime(Config{StrictMode: StrictModeOptions{Enabled: true}})
	parseRenderCount := 0
	parseFiber := &Fiber{
		typeOf: func() *Element {
			parseRenderCount++
			_, parseSet := GoUseState(parseRt, 0)
			if parseRenderCount == 1 {
				parseSet(1)
			}
			return &Element{Type: "div"}
		},
		ownerRuntime: parseRt,
	}

	parseElement, parseHandled, _ := parseRt.renderFunctionComponent(parseFiber)
	if parseHandled || parseElement == nil {
		parseT.Fatalf("expected strict render to complete, handled=%t element=%#v", parseHandled, parseElement)
	}
	if parseRenderCount != 2 {
		parseT.Fatalf("expected strict mode to double-invoke render, got %d calls", parseRenderCount)
	}
	if !hasDiagnosticContaining("state update during render") {
		parseT.Fatalf("expected setState-during-render diagnostic, got %#v", GetDiagnostics())
	}

	parseRt.checkStrictEffectCleanupSymmetry(parseFiber, 0, true)
	parseRt.checkStrictEffectCleanupSymmetry(parseFiber, 0, false)
	if !hasDiagnosticContaining("effect cleanup contract changing") {
		parseT.Fatalf("expected asymmetric cleanup diagnostic, got %#v", GetDiagnostics())
	}
}

func TestEnterpriseDeterministicReplaySchedulesCapturedUpdates(parseT *testing.T) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseChild := &Fiber{typeOf: "child", parent: parseRoot}
	parseRoot.child = parseChild
	parseRt := NewRuntime(Config{})
	parseRt.currentRoot = parseRoot

	parseRt.BeginReplayCapture()
	parseRt.ScheduleGranularUpdateForFiberWithOrigin(parseChild, "atom")
	parseEvents := parseRt.EndReplayCapture()
	if len(parseEvents) != 1 || parseEvents[0].Kind != string(replayUpdateKindGranular) || len(parseEvents[0].Path) != 1 || parseEvents[0].Path[0] != 0 {
		parseT.Fatalf("expected one granular replay event with child path, got %#v", parseEvents)
	}

	parseReplayRoot := &Fiber{typeOf: "ROOT"}
	parseReplayChild := &Fiber{typeOf: "child", parent: parseReplayRoot}
	parseReplayRoot.child = parseReplayChild
	parseReplayRuntime := NewRuntime(Config{})
	parseReplayRuntime.currentRoot = parseReplayRoot

	parseReplayRuntime.ReplayUpdates(parseEvents)
	if !parseReplayChild.dirty || !parseReplayChild.needsUpdate || parseReplayChild.updateOrigin != "atom" {
		parseT.Fatalf("expected replay to reschedule child update, child=%#v", parseReplayChild)
	}
}

func TestEnterpriseLongSessionMemoryHygieneSnapshotAndDiagnostics(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := NewRuntime(Config{})
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseChild := &Fiber{typeOf: "child", parent: parseRoot}
	parseRoot.child = parseChild
	parseRt.currentRoot = parseRoot
	parseRt.atomRegistry.InitAtom("a", 1)
	parseRt.atomRegistry.Subscribe("a", parseChild)

	parseState := parseRt.InternalStateSnapshot()
	if parseState.FiberCount != 2 || parseState.AtomCount != 1 || parseState.AtomSubscriberCount != 1 {
		parseT.Fatalf("expected internal state counters, got %#v", parseState)
	}

	parseMemory := parseRt.CheckMemoryHygiene(MemoryHygieneOptions{MaxFiberCount: 1, MaxAtomCount: 0})
	if len(parseMemory.Diagnostics) == 0 || !hasDiagnosticContaining("fiber count threshold") {
		parseT.Fatalf("expected memory hygiene diagnostic, snapshot=%#v diagnostics=%#v", parseMemory, GetDiagnostics())
	}
}

func hasDiagnosticContaining(parseNeedle string) bool {
	for _, parseDiagnostic := range GetDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, parseNeedle) {
			return true
		}
	}
	return false
}
