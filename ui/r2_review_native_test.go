//go:build !js || !wasm

package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/anim"
	"github.com/monstercameron/GoWebComponents/v6/query"
)

// TestNativeStateAnyUpdateAppliesUpdater guards against Update storing the updater func itself
// when T is an interface type that the func satisfies.
func TestNativeStateAnyUpdateAppliesUpdater(parseT *testing.T) {
	parseState := UseState[any](1)
	parseState.Update(func(parsePrev any) any { return parsePrev.(int) + 1 })
	if parseGot, parseOk := parseState.Get().(int); !parseOk || parseGot != 2 {
		parseT.Fatalf("Update on State[any] = %#v, want 2", parseState.Get())
	}
	parseState.Set(nil)
	if parseState.Get() != nil {
		parseT.Fatalf("Set(nil) on State[any] = %#v, want nil", parseState.Get())
	}
}

// TestNativeUseIDIsCSSSafe asserts UseId output can be used in a bare #id selector.
func TestNativeUseIDIsCSSSafe(parseT *testing.T) {
	parseID := UseId()
	if strings.ContainsAny(parseID, ":.") {
		parseT.Fatalf("UseId() = %q contains selector-special characters", parseID)
	}
}

// runSpringToRest drives a spring-start func with 16ms synthetic frames and returns the last position.
func runSpringToRest(parseT *testing.T, parseStart func(parseSet func(float64), parseSchedule springFrameScheduler) func()) float64 {
	parseT.Helper()
	var parsePending func(float64)
	parseLast := -999.0
	parseSchedule := func(parseFrame func(float64)) func() {
		parsePending = parseFrame
		return func() {}
	}
	parseStart(func(parsePos float64) { parseLast = parsePos }, parseSchedule)
	for parseIndex := 1; parsePending != nil && parseIndex < 5000; parseIndex++ {
		parseNext := parsePending
		parsePending = nil
		parseNext(float64(parseIndex) * 16)
	}
	if parsePending != nil {
		parseT.Fatal("spring never settled")
	}
	return parseLast
}

// TestSpringSettlesExactlyOnTarget proves a settled spring publishes its target, including
// when the whole travel is smaller than the settle epsilon (opacity 0 -> 1, scale 1 -> 1.05).
func TestSpringSettlesExactlyOnTarget(parseT *testing.T) {
	for _, parseCase := range []struct{ from, to float64 }{{0, 1}, {1, 1.05}, {0, 300}} {
		parseSpring := anim.NewSpring(anim.GentleSpring(), parseCase.from)
		parseSpring.SetTarget(parseCase.to)
		parseGot := runSpringToRest(parseT, func(parseSet func(float64), parseSchedule springFrameScheduler) func() {
			return startSpringAnimationTo(parseSpring, parseCase.to, parseSet, parseSchedule)
		})
		if parseGot != parseCase.to {
			parseT.Errorf("spring %v -> %v ended at %v", parseCase.from, parseCase.to, parseGot)
		}
	}
}

// TestUseAsyncMutationFnPanicRollsBack proves a panicking background action is contained and
// treated as a failed mutation (rolled back) instead of crashing the process.
func TestUseAsyncMutationFnPanicRollsBack(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("likes", 10)
	parseSettled := make(chan struct{}, 1)
	parseRun := UseAsyncMutation[int](parseCache, "likes")
	parseRun(11, func() (int, error) {
		defer func() { parseSettled <- struct{}{} }()
		panic("boom")
	})
	select {
	case <-parseSettled:
	case <-time.After(2 * time.Second):
		parseT.Fatal("action never ran")
	}
	time.Sleep(50 * time.Millisecond)
	if parseGot, _ := parseCache.Peek("likes"); parseGot != 10 {
		parseT.Fatalf("after a panicking action the cache holds %v, want rollback to 10", parseGot)
	}
}

// TestFormRunnerPanicSettlesState proves a panicking submit or validator callback is contained AND
// leaves the form settled (not stuck Submitting/Validating, which would lock the submit button).
func TestFormRunnerPanicSettlesState(parseT *testing.T) {
	parseForm := UseForm(sampleFormModel{Name: "x"})
	parseForm.Submit(func(sampleFormModel) error { panic("submit boom") })
	waitForCondition(parseT, time.Second, func() bool { return !parseForm.Submitting() })
	if parseForm.Submitted() || parseForm.SubmitError() == nil {
		parseT.Fatalf("panicked submit should report a failure, submitted=%t err=%v", parseForm.Submitted(), parseForm.SubmitError())
	}

	parseForm.SubmitWithIntent("publish", func(sampleFormModel, string) error { panic("intent boom") })
	waitForCondition(parseT, time.Second, func() bool { return !parseForm.Submitting() })
	if parseForm.SubmitError() == nil {
		parseT.Fatal("panicked intent submit should report a failure")
	}

	parseForm.ValidateAsync(func(sampleFormModel) (FieldErrors, string) { panic("validate boom") }, nil)
	waitForCondition(parseT, time.Second, func() bool { return !parseForm.Validating() })
	if parseForm.FormError() == "" {
		parseT.Fatal("panicked validator should surface a form error")
	}
}
