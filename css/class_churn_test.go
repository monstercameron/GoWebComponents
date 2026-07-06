package css

import (
	"strconv"
	"testing"
)

// TestClassRegistryChurnWarnsOnceOverThreshold pins the class-churn watchdog: when
// the class registry grows past the churn threshold (the signature of classes being
// minted from runtime values instead of routed through the Dynamic escape valve),
// exactly one diagnostic is emitted — not one per subsequent class.
func TestClassRegistryChurnWarnsOnceOverThreshold(parseT *testing.T) {
	Reset()
	parsePrevThreshold := classRegistryChurnThreshold
	parsePrevReport := reportClassRegistryChurn
	parseWarnCount := 0
	classRegistryChurnThreshold = 3
	reportClassRegistryChurn = func(int) { parseWarnCount++ }
	defer func() {
		classRegistryChurnThreshold = parsePrevThreshold
		reportClassRegistryChurn = parsePrevReport
		Reset()
	}()

	// Register distinct classes past the threshold.
	for parseI := 0; parseI < 10; parseI++ {
		registerAndEmit("c-churn-"+strconv.Itoa(parseI), ".c-churn-"+strconv.Itoa(parseI)+"{}")
	}

	if parseWarnCount != 1 {
		parseT.Fatalf("expected exactly one churn warning, got %d", parseWarnCount)
	}
}

// TestClassRegistryUnderThresholdDoesNotWarn pins that ordinary usage (well below
// the threshold) never warns.
func TestClassRegistryUnderThresholdDoesNotWarn(parseT *testing.T) {
	Reset()
	parsePrevThreshold := classRegistryChurnThreshold
	parsePrevReport := reportClassRegistryChurn
	parseWarnCount := 0
	classRegistryChurnThreshold = 100
	reportClassRegistryChurn = func(int) { parseWarnCount++ }
	defer func() {
		classRegistryChurnThreshold = parsePrevThreshold
		reportClassRegistryChurn = parsePrevReport
		Reset()
	}()

	for parseI := 0; parseI < 20; parseI++ {
		registerAndEmit("c-ok-"+strconv.Itoa(parseI), ".c-ok-"+strconv.Itoa(parseI)+"{}")
	}
	if parseWarnCount != 0 {
		parseT.Fatalf("expected no churn warning under threshold, got %d", parseWarnCount)
	}
}

// TestClassRegistryChurnResetClearsWarnFlag pins that Reset re-arms the one-time
// warning (so a fresh process/test can warn again).
func TestClassRegistryChurnResetClearsWarnFlag(parseT *testing.T) {
	Reset()
	if classChurnWarned {
		parseT.Fatal("Reset must clear the churn-warned flag")
	}
}
