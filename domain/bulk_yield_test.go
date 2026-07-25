package domain

import (
	"strings"
	"testing"
)

// Bulk cancellation in a single-threaded message loop (P3.4).
//
// The existing cancellation test calls Cancel from inside the running step. That
// proves the flag is read every item, but it cannot fail the way production
// does, because it presupposes exactly the thing production denies: that
// something else gets to run while the loop is running.
//
// A domain worker receives a command, runs it, and only then reads its next
// message. A 50,000-item run that never returns to that loop cannot be cancelled
// by a message — the cancel sits in a queue the worker will not read until the
// run it is meant to stop has already finished. Checking a flag on every item is
// necessary and useless on its own, because with no yield nothing can set it.
//
// These tests model that: the ONLY code that can cancel runs inside the yielder,
// standing in for the worker's message pump. A run that does not yield can never
// see it.

// pumpedRuntime returns a runtime whose yielder plays the part of a worker's
// message pump: each turn delivers at most one queued message.
func pumpedRuntime(parseT *testing.T) (*Runtime, func(func())) {
	parseT.Helper()
	parseRuntime := NewRuntime(nil)

	parseMessages := []func(){}
	parseEnqueue := func(parseMessage func()) {
		parseMessages = append(parseMessages, parseMessage)
	}
	parseRuntime.SetYield(func() {
		if len(parseMessages) == 0 {
			return
		}
		parseNext := parseMessages[0]
		parseMessages = parseMessages[1:]
		parseNext()
	})
	return parseRuntime, parseEnqueue
}

// TestExecuteBulk_CancelArrivesOnlyThroughAYield is the criterion, stated the
// way the worker actually behaves.
func TestExecuteBulk_CancelArrivesOnlyThroughAYield(parseT *testing.T) {
	parseRuntime, parseEnqueue := pumpedRuntime(parseT)
	const parseCommandID CommandID = "bulk-cancel"

	// The cancel is already waiting when the run starts, exactly as a message
	// posted while the worker was busy would be.
	parseEnqueue(func() { parseRuntime.Cancel(parseCommandID) })

	parseProcessed := 0
	parseResult, parseErr := parseRuntime.ExecuteBulk(
		BulkCommand{ID: parseCommandID, Total: 10000, YieldEvery: 100},
		func(int) error {
			parseProcessed++
			return nil
		})
	if parseErr != nil {
		parseT.Fatalf("ExecuteBulk: %v", parseErr)
	}

	if parseResult.Outcome != OutcomeCancelled {
		parseT.Errorf("outcome = %v, want %v; the queued cancel never reached the run",
			parseResult.Outcome, OutcomeCancelled)
	}
	if parseResult.Completed {
		parseT.Error("a cancelled run reported completion")
	}
	// It must stop promptly, not merely eventually. One yield interval plus the
	// items before the first yield is the bound.
	if parseProcessed > 300 {
		parseT.Errorf("processed %d of 10000 items before stopping; the cancel was seen far later than the yield interval",
			parseProcessed)
	}
	parseT.Logf("stopped after %d of 10000 items", parseProcessed)
}

// TestExecuteBulk_WithoutYieldingTheCancelIsNeverSeen states the old behaviour
// as a fact rather than leaving it implied, so the yield's purpose stays legible
// and a later "simplification" that drops it fails here.
func TestExecuteBulk_WithoutYieldingTheCancelIsNeverSeen(parseT *testing.T) {
	parseRuntime, parseEnqueue := pumpedRuntime(parseT)
	const parseCommandID CommandID = "bulk-no-yield"

	parseEnqueue(func() { parseRuntime.Cancel(parseCommandID) })

	parseProcessed := 0
	parseResult, parseErr := parseRuntime.ExecuteBulk(
		BulkCommand{ID: parseCommandID, Total: 500}, // YieldEvery unset
		func(int) error {
			parseProcessed++
			return nil
		})
	if parseErr != nil {
		parseT.Fatalf("ExecuteBulk: %v", parseErr)
	}

	if parseResult.Outcome != OutcomeApplied || parseProcessed != 500 {
		parseT.Errorf("outcome=%v processed=%d; a non-yielding run is expected to complete, because the pump never gets a turn",
			parseResult.Outcome, parseProcessed)
	}
}

// TestExecuteBulk_YieldingWithoutAYielderIsRefused pins the loud failure.
//
// Running anyway would produce the worst outcome available: a command that
// checks the cancel flag on every item, and so looks cancellable, while being
// structurally incapable of ever receiving a cancel.
func TestExecuteBulk_YieldingWithoutAYielderIsRefused(parseT *testing.T) {
	parseRuntime := NewRuntime(nil)

	parseResult, parseErr := parseRuntime.ExecuteBulk(
		BulkCommand{ID: "bulk-unyielding", Total: 10, YieldEvery: 2},
		func(int) error { return nil })

	if parseErr == nil {
		parseT.Fatal("a yield interval with no yielder installed was accepted")
	}
	if !strings.Contains(parseErr.Error(), "SetYield") {
		parseT.Errorf("error %q does not say how to fix the configuration", parseErr)
	}
	if parseResult.Outcome != OutcomeFailed {
		parseT.Errorf("outcome = %v, want %v", parseResult.Outcome, OutcomeFailed)
	}
}

// TestExecuteBulk_YieldingRunStillCompletesAndCheckpoints keeps the yield from
// being bought at the cost of the guarantees around it.
func TestExecuteBulk_YieldingRunStillCompletesAndCheckpoints(parseT *testing.T) {
	parseStore := NewMemoryCheckpointStore()
	parseRuntime := NewRuntime(parseStore)
	parseYields := 0
	parseRuntime.SetYield(func() { parseYields++ })

	parseSeen := make([]bool, 1000)
	parseResult, parseErr := parseRuntime.ExecuteBulk(
		BulkCommand{ID: "bulk-complete", Total: 1000, YieldEvery: 50, CheckpointEvery: 100},
		func(parseIndex int) error {
			if parseSeen[parseIndex] {
				parseT.Errorf("item %d ran twice", parseIndex)
			}
			parseSeen[parseIndex] = true
			return nil
		})
	if parseErr != nil {
		parseT.Fatalf("ExecuteBulk: %v", parseErr)
	}

	if !parseResult.Completed || parseResult.Outcome != OutcomeApplied {
		parseT.Errorf("completed=%v outcome=%v, want a fully applied run", parseResult.Completed, parseResult.Outcome)
	}
	if parseResult.Processed != 1000 {
		parseT.Errorf("processed %d items, want 1000", parseResult.Processed)
	}
	if parseYields == 0 {
		parseT.Error("the run never yielded, so this asserts nothing about a yielding run")
	}
	for parseIndex, hasRun := range parseSeen {
		if !hasRun {
			parseT.Fatalf("item %d never ran", parseIndex)
		}
	}
	parseT.Logf("1000 items, %d yields", parseYields)
}
