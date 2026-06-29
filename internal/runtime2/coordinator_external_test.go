package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestBuildCoordinatorCreatesStandaloneCoordinator verifies the package compiles through its external surface.
func TestBuildCoordinatorCreatesStandaloneCoordinator(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseCoordinator == nil {
		parseT.Fatal("expected coordinator")
	}
	parseStatus := parseCoordinator.GetStatus()
	if parseStatus.IsConfigured {
		parseT.Fatal("expected new coordinator to be unconfigured")
	}
}

// TestGetStatusHandlesNilCoordinator verifies the zero-value guard path does not panic.
func TestGetStatusHandlesNilCoordinator(parseT *testing.T) {
	var parseCoordinator *runtime2.Coordinator
	parseStatus := parseCoordinator.GetStatus()
	if parseStatus.IsConfigured {
		parseT.Fatal("expected nil coordinator to report zero status")
	}
}
