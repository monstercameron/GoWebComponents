//go:build js && wasm

package ui

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v7/query"
)

// TestUseRevalidateOnFocusRefetchesMountedQuery proves the whole focus loop: a mounted UseQuery
// whose data is fresh must refetch after the window focus signal fires, not merely be marked stale.
func TestUseRevalidateOnFocusRefetchesMountedQuery(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePreviousInitialized })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Capture the focus handler instead of binding a real window listener.
	parsePrevBind := bindGlobalEvent
	parseT.Cleanup(func() { bindGlobalEvent = parsePrevBind })
	var parseFocus func(Event)
	bindGlobalEvent = func(parseScope globalScope, parseEventType string, parseHandler func(Event)) func() {
		if parseEventType == "focus" {
			parseFocus = parseHandler
		}
		return func() {}
	}

	parseCache := query.New(query.WithStaleTime(time.Hour))
	var parseCalls atomic.Int32
	parseProbe := func() Node {
		UseRevalidateOnFocus(parseCache)
		UseQuery(parseCache, "k", func() (int, error) {
			return int(parseCalls.Add(1)), nil
		})
		return runtime.CreateElement("span", map[string]any{})
	}
	if parseErr := RenderInto(CreateElement(parseProbe), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	parseScheduler.Flush()
	waitQueryCalls(parseT, parseScheduler, &parseCalls, 1)

	if parseFocus == nil {
		parseT.Fatal("focus handler was never bound")
	}
	parseFocus(Event{})
	parseScheduler.Flush()
	waitQueryCalls(parseT, parseScheduler, &parseCalls, 2)
}

// waitQueryCalls pumps the scheduler until the fetcher has run parseWant times or a deadline passes.
func waitQueryCalls(parseT *testing.T, parseScheduler *queuedScheduler, parseCalls *atomic.Int32, parseWant int32) {
	parseT.Helper()
	parseDeadline := time.Now().Add(2 * time.Second)
	for parseCalls.Load() < parseWant && time.Now().Before(parseDeadline) {
		time.Sleep(5 * time.Millisecond)
		parseScheduler.Flush()
	}
	if parseGot := parseCalls.Load(); parseGot != parseWant {
		parseT.Fatalf("fetcher calls = %d, want %d", parseGot, parseWant)
	}
}
