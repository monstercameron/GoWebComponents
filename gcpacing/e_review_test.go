package gcpacing

import "testing"

// TestApplyReportsNegativeMemoryLimitAsOff verifies a negative limit is
// reported as 0 (no limit), matching Settings.MemoryLimitBytes' contract.
func TestApplyReportsNegativeMemoryLimitAsOff(parseT *testing.T) {
	parseSettings, parsePrevious, parseErr := Apply(ProfileBalanced, -5)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	defer parsePrevious.Restore()
	if parseSettings.MemoryLimitBytes != 0 {
		parseT.Fatalf("expected MemoryLimitBytes 0 for a negative request, got %d", parseSettings.MemoryLimitBytes)
	}
}
