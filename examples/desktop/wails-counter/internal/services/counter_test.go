package services

import "testing"

// TestCounterServiceIncrementRoundTrip verifies the typed state advances exactly once per call.
func TestCounterServiceIncrementRoundTrip(parseT *testing.T) {
	parseService := &CounterService{}
	if parseState := parseService.Increment(); parseState.Value != 1 {
		parseT.Fatalf("first increment = %d, want 1", parseState.Value)
	}
	if parseState := parseService.Increment(); parseState.Value != 2 {
		parseT.Fatalf("second increment = %d, want 2", parseState.Value)
	}
}
