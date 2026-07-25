package runtime

import (
	"testing"
)

// v5 P4.2 — workload budgets, threat T9.
//
// The criterion is that every T9 cliff has a DEFINED behaviour, not merely a
// diagnostic, plus a devtools signal — and that the diagnostics themselves stay
// inside R6's allocation budget.
//
// The tests are therefore about two things a log line cannot provide: a
// structured reading a panel or a CI gate can consume, and a bound on what
// producing that reading costs.

func TestBudgetsReportEveryCliffWithAName(parseT *testing.T) {
	parseRt := NewRuntime(Config{})
	parseBudgets := parseRt.Budgets()

	parseSignals := parseBudgets.Signals()
	if len(parseSignals) != 2 {
		parseT.Fatalf("signals = %d, want both T9 cliffs represented", len(parseSignals))
	}

	parseSeen := map[string]bool{}
	for _, parseSignal := range parseSignals {
		if parseSignal.Name == "" {
			parseT.Error("a cliff without a stable name cannot be keyed on by devtools or a test")
		}
		if parseSeen[parseSignal.Name] {
			parseT.Errorf("cliff name %q is not distinct", parseSignal.Name)
		}
		parseSeen[parseSignal.Name] = true

		if parseSignal.Limit <= 0 {
			parseT.Errorf("cliff %q has no limit; T9 is about bounded queues", parseSignal.Name)
		}
	}
	if !parseSeen[BudgetPendingEffects] || !parseSeen[BudgetQueuedUpdates] {
		parseT.Errorf("signals = %v, want both %q and %q", parseSeen, BudgetPendingEffects, BudgetQueuedUpdates)
	}
}

// TestEachCliffDeclaresWhetherItLosesWork answers §11-Q12 in a form a caller can
// branch on.
//
// Both cliffs preserve work — one by degrading to a more expensive path, the
// other by merging idempotent re-render requests. Neither rejects. A signal that
// did not say so would leave every reader to guess, and "coalesce" reads like a
// loss when it is not.
func TestEachCliffDeclaresWhetherItLosesWork(parseT *testing.T) {
	parseBudgets := NewRuntime(Config{}).Budgets()

	if parseBudgets.PendingEffects.Behavior != BudgetDegrade {
		parseT.Errorf("pending effects behaviour = %s, want degrade", parseBudgets.PendingEffects.Behavior)
	}
	if parseBudgets.QueuedUpdates.Behavior != BudgetCoalesce {
		parseT.Errorf("queued updates behaviour = %s, want coalesce", parseBudgets.QueuedUpdates.Behavior)
	}

	for _, parseSignal := range parseBudgets.Signals() {
		if parseSignal.Behavior.LosesWork() {
			parseT.Errorf("cliff %q claims to lose work; neither T9 cliff does", parseSignal.Name)
		}
	}
	// The vocabulary still has to be able to express a losing cliff, or a future
	// one will be described as something it is not.
	if !BudgetReject.LosesWork() {
		parseT.Error("reject must be describable as losing work")
	}
	if BudgetBlock.LosesWork() {
		parseT.Error("blocking stalls a producer; it does not lose work")
	}
}

// TestHeadroomDistinguishesNearFromOver is why the signal carries an occupancy
// rather than only a tripped flag: a workload sitting at 95% is a warning, and
// one that never trips is not a bug.
func TestHeadroomDistinguishesNearFromOver(parseT *testing.T) {
	parseSignal := BudgetSignal{Name: "x", Limit: 100, Observed: 95}
	if parseSignal.AtCapacity() {
		parseT.Error("95 of 100 is not at capacity")
	}
	if parseSignal.Headroom() != 5 {
		parseT.Errorf("headroom = %d, want 5", parseSignal.Headroom())
	}

	parseFull := BudgetSignal{Name: "x", Limit: 100, Observed: 100}
	if !parseFull.AtCapacity() || parseFull.Headroom() != 0 {
		parseT.Errorf("at the limit: atCapacity=%v headroom=%d", parseFull.AtCapacity(), parseFull.Headroom())
	}

	parseUnbounded := BudgetSignal{Name: "x", Limit: 0, Observed: 1000}
	if parseUnbounded.AtCapacity() {
		parseT.Error("an unbounded cliff is never at capacity")
	}
	if parseUnbounded.Headroom() != -1 {
		parseT.Errorf("headroom = %d, want -1 for unbounded", parseUnbounded.Headroom())
	}
}

// TestBudgetsReflectConfiguredLimits: a signal reporting the default while the
// runtime enforces something else would be worse than no signal.
func TestBudgetsReflectConfiguredLimits(parseT *testing.T) {
	parseRt := NewRuntime(Config{Limits: RuntimeLimits{
		MaxPendingEffectFibers: 7,
		MaxQueuedUpdates:       9,
	}})
	parseBudgets := parseRt.Budgets()

	if parseBudgets.PendingEffects.Limit != 7 {
		parseT.Errorf("pending effects limit = %d, want the configured 7", parseBudgets.PendingEffects.Limit)
	}
	if parseBudgets.QueuedUpdates.Limit != 9 {
		parseT.Errorf("queued updates limit = %d, want the configured 9", parseBudgets.QueuedUpdates.Limit)
	}
}

// TestEffectOverflowIsCountedNotJustFlagged: a boolean cannot distinguish
// "tripped once during a bulk import" from "trips every frame", which are
// different problems with different fixes.
func TestEffectOverflowIsCountedNotJustFlagged(parseT *testing.T) {
	parseRt := NewRuntime(Config{Limits: RuntimeLimits{MaxPendingEffectFibers: 2}})

	if parseRt.Budgets().PendingEffects.Triggered != 0 {
		parseT.Fatal("a fresh runtime has tripped nothing")
	}

	// Three fibers with effects against a limit of two forces the cliff.
	for range 3 {
		parseRt.queuePendingEffectFiber(&Fiber{effects: []Effect{{Fn: func() func() { return nil }}}})
	}

	parseBudgets := parseRt.Budgets()
	if parseBudgets.PendingEffects.Triggered == 0 {
		parseT.Error("the overflow was not counted")
	}
	if !parseBudgets.PendingEffects.Active {
		parseT.Error("the cliff must report itself active while degraded")
	}
	if !parseBudgets.Degraded() {
		parseT.Error("a runtime with an active cliff must report degraded")
	}
}

// TestRepeatedOverflowCostsNothingExtraIsTheR6Fix.
//
// Emptying the queue on overflow means the next N fibers refill it and overflow
// AGAIN within the same pass, so a large tree hits this path repeatedly. Each
// visit used to evaluate diagnosticPathForFiber and diagnosticComponentStack —
// building a path string and walking the component stack — on the render
// thread, during the pass already large enough to have overflowed.
//
// The diagnostics layer dedups by message, but dedup happens AFTER the call
// site, so it never saved the allocation. That is why this measures marginal
// ALLOCATION rather than counting diagnostic entries: entries were already
// deduped to one, and the cost R6 cares about was being paid anyway.
func TestRepeatedOverflowCostsNothingExtraIsTheR6Fix(parseT *testing.T) {
	// Marginal cost of one further overflow on a runtime that has already
	// degraded. With the report inside the transition check this is just the
	// slice reset; with it on every overflow it is a path string and a stack
	// walk per event.
	measureMarginalAllocs := func(parseOverflows int) float64 {
		return testing.AllocsPerRun(20, func() {
			parseRt := NewRuntime(Config{Limits: RuntimeLimits{MaxPendingEffectFibers: 1}})
			for range parseOverflows {
				parseRt.queuePendingEffectFiber(&Fiber{effects: []Effect{{Fn: func() func() { return nil }}}})
			}
		})
	}

	parseFew := measureMarginalAllocs(4)
	parseMany := measureMarginalAllocs(24)
	parseMarginal := (parseMany - parseFew) / 20

	parseT.Logf("R6: %.1f allocs for 4 overflows, %.1f for 24 — %.2f per additional overflow",
		parseFew, parseMany, parseMarginal)

	// One allocation per extra overflow is the fiber itself, which the test
	// creates. The path string and stack slice would push this well past 2.
	if parseMarginal > 2.0 {
		parseT.Errorf("each additional overflow costs %.2f allocations; R6 forbids per-event diagnostic cost on the render thread", parseMarginal)
	}

	// The signal must still carry the repetition the warnings used to.
	parseRt := NewRuntime(Config{Limits: RuntimeLimits{MaxPendingEffectFibers: 1}})
	for range 10 {
		parseRt.queuePendingEffectFiber(&Fiber{effects: []Effect{{Fn: func() func() { return nil }}}})
	}
	if parseRt.Budgets().PendingEffects.Triggered < 2 {
		parseT.Errorf("Triggered = %d, want the repeated overflows visible as a count",
			parseRt.Budgets().PendingEffects.Triggered)
	}
}

// TestBudgetsAllocateNothing is R6 stated as a measurement rather than an
// intention. A devtools panel polls this every frame.
func TestBudgetsAllocateNothing(parseT *testing.T) {
	parseRt := NewRuntime(Config{})

	parseAllocs := testing.AllocsPerRun(100, func() {
		parseBudgets := parseRt.Budgets()
		for _, parseSignal := range parseBudgets.Signals() {
			if parseSignal.Headroom() == -2 {
				parseT.Fatal("impossible headroom")
			}
		}
	})
	if parseAllocs > 0 {
		parseT.Errorf("reading budgets allocated %.1f times per call; R6 requires a per-frame read to be free", parseAllocs)
	}
}

func TestNilRuntimeBudgetsAreSafe(parseT *testing.T) {
	var parseRt *Runtime
	parseBudgets := parseRt.Budgets()

	// A devtools panel polls on a timer and outlives what it was watching.
	if parseBudgets.Degraded() {
		parseT.Error("a nil runtime is not degraded")
	}
	for _, parseSignal := range parseBudgets.Signals() {
		if parseSignal.Name == "" {
			parseT.Error("even a nil runtime must name its cliffs, or a panel renders blanks")
		}
	}
}

func TestBudgetBehaviorLabelsAreDistinct(parseT *testing.T) {
	parseSeen := map[string]bool{}
	for _, parseBehavior := range []BudgetBehavior{BudgetDegrade, BudgetCoalesce, BudgetReject, BudgetBlock} {
		if parseSeen[parseBehavior.String()] {
			parseT.Errorf("behaviour label %q is not distinct", parseBehavior)
		}
		parseSeen[parseBehavior.String()] = true
	}
	if BudgetBehavior(99).String() != "unknown" {
		parseT.Error("an unknown behaviour must still render for diagnostics")
	}
}
