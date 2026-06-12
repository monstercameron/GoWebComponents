package runtime

import "testing"

func TestGoUseStateTransitionDefersUpdateUntilTimeout(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseFiber := &Fiber{typeOf: "test", props: map[string]any{}}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(parseRt, 1)
	parseRt.StartTransition(func() {
		set(5)
	})

	if parseGot := get(); parseGot != 1 {
		parseT.Fatalf("expected transition update to stay deferred before timeout, got %d", parseGot)
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one deferred transition timeout, got %d", len(parseScheduler.timeouts))
	}
	if parsePending, _ := parseRt.GetAtomValue(transitionPendingAtomID); parsePending != true {
		parseT.Fatalf("expected transition pending atom true before flush, got %#v", parsePending)
	}

	parseScheduler.timeouts[0]()

	if parseGot2 := get(); parseGot2 != 5 {
		parseT.Fatalf("expected deferred state update after timeout, got %d", parseGot2)
	}
	if parsePending2, _ := parseRt.GetAtomValue(transitionPendingAtomID); parsePending2 != false {
		parseT.Fatalf("expected transition pending atom false after flush, got %#v", parsePending2)
	}
}

func TestGoUseAtomTransitionDefersSharedUpdateUntilTimeout(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseFiber := &Fiber{typeOf: "test", props: map[string]any{}}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(parseRt, "shared-transition", 2)
	parseRt.StartTransition(func() {
		set(8)
	})

	if parseGot := get(); parseGot != 2 {
		parseT.Fatalf("expected atom update to stay deferred before timeout, got %d", parseGot)
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one deferred atom timeout, got %d", len(parseScheduler.timeouts))
	}

	parseScheduler.timeouts[0]()

	if parseGot2 := get(); parseGot2 != 8 {
		parseT.Fatalf("expected deferred atom update after timeout, got %d", parseGot2)
	}
}

func TestScheduleTransitionWithoutSchedulerRunsImmediately(parseT *testing.T) {
	parseRt := NewRuntime(Config{})
	isParseExecuted := false
	parseRt.ScheduleTransition(func() {
		isParseExecuted = true
	})
	if !isParseExecuted {
		parseT.Fatal("expected transition callback to run immediately without scheduler")
	}
}

func TestScheduleTransitionReportsProfilingTimelineEvents(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})

	parseRt.ScheduleTransition(func() {})

	if len(parseRt.profiling.events) == 0 {
		parseT.Fatal("expected scheduled transition profiling event")
	}
	parseScheduled := parseRt.profiling.events[0]
	if parseScheduled.Name != "transition" || parseScheduled.Phase != "scheduled" {
		parseT.Fatalf("expected scheduled transition profiling event, got %+v", parseScheduled)
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one transition timeout, got %d", len(parseScheduler.timeouts))
	}

	parseScheduler.timeouts[0]()

	isParseFoundRun := false
	for _, parseEvent := range parseRt.profiling.events {
		if parseEvent.Name == "transition" && parseEvent.Phase == "run" {
			isParseFoundRun = true
			if parseEvent.DurationNs < 0 {
				parseT.Fatalf("expected non-negative transition wait duration, got %+v", parseEvent)
			}
		}
	}
	if !isParseFoundRun {
		parseT.Fatalf("expected transition run profiling event, got %+v", parseRt.profiling.events)
	}
}
