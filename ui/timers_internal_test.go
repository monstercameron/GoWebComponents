//go:build !(js && wasm)

package ui

import (
	"testing"
	"time"
)

func TestTimerDepsStableWhenNoneGiven(t *testing.T) {
	parseA := timerDeps(100*time.Millisecond, false, nil)
	if len(parseA) == 0 {
		t.Fatal("timer deps must be non-empty so the timer schedules once")
	}
	parseB := timerDeps(100*time.Millisecond, false, nil)
	if len(parseA) != len(parseB) {
		t.Fatal("timer dep length must be stable")
	}
	// Different delay/repeat => different identity (so two timers don't collide).
	if timerDeps(200*time.Millisecond, false, nil)[1] == parseA[1] {
		t.Fatal("distinct delays must differ in the dep set")
	}
	// Explicit deps pass through.
	parseOut := timerDeps(0, true, []any{"x"})
	if len(parseOut) != 1 || parseOut[0] != "x" {
		t.Fatalf("explicit deps must pass through, got %v", parseOut)
	}
}

func TestNativeScheduleTimerIsNoOp(t *testing.T) {
	if parseCancel := defaultScheduleTimer(func() {}, time.Second, false); parseCancel != nil {
		t.Fatal("native scheduleTimer should return nil")
	}
}

func TestScheduleTimerSeamSwappable(t *testing.T) {
	parsePrev := scheduleTimer
	defer func() { scheduleTimer = parsePrev }()
	var parseScheduled, parseCancelled int
	var parseGotRepeat bool
	scheduleTimer = func(parseFn func(), parseDelay time.Duration, parseRepeat bool) func() {
		parseScheduled++
		parseGotRepeat = parseRepeat
		return func() { parseCancelled++ }
	}
	parseCancel := scheduleTimer(func() {}, time.Second, true)
	if parseScheduled != 1 || !parseGotRepeat {
		t.Fatalf("seam not invoked correctly: scheduled=%d repeat=%v", parseScheduled, parseGotRepeat)
	}
	parseCancel()
	if parseCancelled != 1 {
		t.Fatalf("cancel contract not honored: %d", parseCancelled)
	}
}
