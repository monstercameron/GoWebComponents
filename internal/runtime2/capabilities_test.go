package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestGetCapabilityReportDefaultsWithoutWorkerSupport verifies the package-level report stays disabled by default.
func TestGetCapabilityReportDefaultsWithoutWorkerSupport(parseT *testing.T) {
	parseReport := runtime2.GetCapabilityReport()
	if parseReport.HasWorkerSupport {
		parseT.Fatal("expected worker support to default to false")
	}
	if parseReport.HasStructuredCloneSupport {
		parseT.Fatal("expected structured-clone support to default to false")
	}
	if parseReport.HasSharedMemoryTransportSupport {
		parseT.Fatal("expected shared-memory transport support to default to false")
	}
}

// TestBuildCapabilityReportKeepsStructuredCloneIndependentFromSharedMemory verifies the report does not collapse transport tiers together.
func TestBuildCapabilityReportKeepsStructuredCloneIndependentFromSharedMemory(parseT *testing.T) {
	parseReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:          true,
		HasStructuredCloneSupport: true,
	})
	if !parseReport.HasWorkerSupport {
		parseT.Fatal("expected worker support")
	}
	if !parseReport.HasStructuredCloneSupport {
		parseT.Fatal("expected structured-clone support")
	}
	if parseReport.HasSharedMemoryTransportSupport {
		parseT.Fatal("expected shared-memory transport support to remain false")
	}
}

// TestBuildCapabilityReportDisablesDependentFeaturesWithoutWorkerSupport verifies impossible combinations normalize safely.
func TestBuildCapabilityReportDisablesDependentFeaturesWithoutWorkerSupport(parseT *testing.T) {
	parseReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	if parseReport.HasMessagePortSupport {
		parseT.Fatal("expected message-port support to stay disabled without workers")
	}
	if parseReport.HasStructuredCloneSupport {
		parseT.Fatal("expected structured-clone support to stay disabled without workers")
	}
	if parseReport.HasSharedBufferSupport {
		parseT.Fatal("expected shared-buffer support to stay disabled without workers")
	}
}
