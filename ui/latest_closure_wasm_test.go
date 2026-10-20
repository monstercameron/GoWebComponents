//go:build js && wasm

package ui

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
)

// mountLatestClosureProbe mounts parseProbe on the wasm test runtime and returns a re-render func.
func mountLatestClosureProbe(parseT *testing.T, parseProbe func() Node) func() {
	parseT.Helper()
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}
	parsePrevInit := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePrevInit })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	if parseErr := RenderInto(CreateElement(parseProbe), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	parseScheduler.Flush()
	return parseScheduler.Flush
}

// TestUseWindowEventHandlerSeesLatestRender proves a global-event handler bound on the first
// render observes values captured by later renders, not the first render's closure.
func TestUseWindowEventHandlerSeesLatestRender(parseT *testing.T) {
	parsePrevBind := bindGlobalEvent
	parseT.Cleanup(func() { bindGlobalEvent = parsePrevBind })
	var parseBound func(Event)
	bindGlobalEvent = func(parseScope globalScope, parseEventType string, parseHandler func(Event)) func() {
		parseBound = parseHandler
		return func() {}
	}
	var parseSetCount func(int)
	parseSeen := -1
	parseFlush := mountLatestClosureProbe(parseT, func() Node {
		parseCount := UseState(0)
		parseSetCount = parseCount.Set
		parseValue := parseCount.Get()
		UseWindowEvent("resize", func(Event) { parseSeen = parseValue })
		return runtime.CreateElement("span", map[string]any{})
	})
	parseSetCount(5)
	parseFlush()
	parseBound(Event{})
	if parseSeen != 5 {
		parseT.Fatalf("handler saw count %d, want 5 (stale first-render closure)", parseSeen)
	}
}

// TestUseIntervalCallbackSeesLatestRender is the same latest-closure proof for timers.
func TestUseIntervalCallbackSeesLatestRender(parseT *testing.T) {
	parsePrev := scheduleTimer
	parseT.Cleanup(func() { scheduleTimer = parsePrev })
	var parseFire func()
	scheduleTimer = func(parseFn func(), parseDelay time.Duration, parseRepeat bool) func() {
		parseFire = parseFn
		return func() {}
	}
	var parseSetCount func(int)
	parseSeen := -1
	parseFlush := mountLatestClosureProbe(parseT, func() Node {
		parseCount := UseState(0)
		parseSetCount = parseCount.Set
		parseValue := parseCount.Get()
		UseInterval(func() { parseSeen = parseValue }, time.Second)
		return runtime.CreateElement("span", map[string]any{})
	})
	parseSetCount(5)
	parseFlush()
	parseFire()
	if parseSeen != 5 {
		parseT.Fatalf("interval callback saw count %d, want 5 (stale first-render closure)", parseSeen)
	}
}
