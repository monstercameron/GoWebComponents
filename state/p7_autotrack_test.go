package state

import "testing"

// TestP7AutoComputedPanicUnwindsTracking pins that a panicking compute does not
// leave auto-tracking switched on or its collector on the stack.
func TestP7AutoComputedPanicUnwindsTracking(parseT *testing.T) {
	func() {
		defer func() { _ = recover() }()
		NewAutoComputed(func() int { panic("compute boom") })
	}()
	if parseActive := trackActive.Load(); parseActive != 0 {
		parseT.Errorf("trackActive = %d after panicking compute, want 0", parseActive)
	}
	trackMu.Lock()
	parseDepth := len(trackStack)
	trackMu.Unlock()
	if parseDepth != 0 {
		parseT.Errorf("trackStack depth = %d after panicking compute, want 0", parseDepth)
	}
}
