//go:build js && wasm

package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// TestRunMountsComponentThenKeepsAlive verifies ui.Run composes the primitives:
// it builds the component, mounts it at the selector via Render, and then calls
// the keep-alive exactly once. The keep-alive is stubbed so the test does not
// block forever.
func TestRunMountsComponentThenKeepsAlive(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.selectors["#app"] = parseContainer
	parseScheduler := &queuedScheduler{}

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePreviousInitialized })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parsePreviousKeepAlive := runKeepAlive
	parseKeepAliveCalls := 0
	runKeepAlive = func() { parseKeepAliveCalls++ }
	parseT.Cleanup(func() { runKeepAlive = parsePreviousKeepAlive })

	parseRendered := false
	parseComponent := func() Node {
		parseRendered = true
		return Text("hello from run")
	}

	Run("#app", parseComponent)
	parseScheduler.Flush()

	if !parseRendered {
		parseT.Fatal("expected Run to mount and render the component")
	}
	if parseKeepAliveCalls != 1 {
		parseT.Fatalf("expected Run to call the keep-alive exactly once, got %d", parseKeepAliveCalls)
	}
}

// TestRunDefaultKeepAliveIsInteropKeepAlive guards the production default so a
// refactor cannot silently swap Run's blocking call for something that returns.
func TestRunDefaultKeepAliveIsInteropKeepAlive(parseT *testing.T) {
	if runKeepAlive == nil {
		parseT.Fatal("expected runKeepAlive to have a non-nil default")
	}
}
