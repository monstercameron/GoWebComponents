package scheduler

import "testing"

type queuedScheduler struct {
	idle    []func(Deadline)
	timeout []func()
}

func (parseS *queuedScheduler) RequestIdleCallback(parseCallback func(Deadline)) {
	parseS.idle = append(parseS.idle, parseCallback)
}

func (parseS *queuedScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	parseS.timeout = append(parseS.timeout, parseCallback)
}

func TestInstrumentedSchedulerRecordsScheduledAndExecutedWork(parseT *testing.T) {
	parseBase := &queuedScheduler{}
	parseScheduler := NewInstrumented(parseBase)
	parseScheduler.RequestIdleCallback(func(Deadline) {})
	parseScheduler.SetTimeout(func() {}, 25)

	parseBefore := parseScheduler.Snapshot()
	if parseBefore.IdleScheduled != 1 || parseBefore.TimeoutScheduled != 1 || parseBefore.TimeoutExecuted != 0 {
		parseT.Fatalf("unexpected pre-flush metrics: %#v", parseBefore)
	}

	parseBase.idle[0](inlineDeadline{})
	parseBase.timeout[0]()
	parseAfter := parseScheduler.Snapshot()
	if parseAfter.IdleExecuted != 1 || parseAfter.TimeoutExecuted != 1 || parseAfter.MaxDelayMs != 25 {
		parseT.Fatalf("unexpected post-flush metrics: %#v", parseAfter)
	}
	if parseScheduler.DevtoolsSection().Summary["timeoutExecuted"] != "1" {
		parseT.Fatalf("devtools section did not expose metrics: %#v", parseScheduler.DevtoolsSection())
	}
}

func TestInstrumentedSchedulerFallsBackInlineWhenNoSchedulerProvided(parseT *testing.T) {
	parseScheduler := NewInstrumented(nil)
	parseRan := false
	parseScheduler.SetTimeout(func() { parseRan = true }, 0)
	if !parseRan {
		parseT.Fatal("inline fallback did not run callback")
	}
	parseMetrics := parseScheduler.Snapshot()
	if parseMetrics.InlineFallbacks != 1 || parseMetrics.TimeoutExecuted != 1 {
		parseT.Fatalf("unexpected inline metrics: %#v", parseMetrics)
	}
}
