package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// BenchmarkGetCapabilityReport benchmarks repeated package-level capability reads after initialization and in default mode.
func BenchmarkGetCapabilityReport(parseB *testing.B) {
	parseB.Run("initialized-lock-free-read", func(parseB *testing.B) {
		runtime2.ResetCapabilityReport()
		_, parseInitErr := runtime2.InitCapabilityReport(runtime2.CapabilitySource{
			HasWorkerSupport:                true,
			HasMessagePortSupport:           true,
			HasStructuredCloneSupport:       true,
			HasBinaryTransportSupport:       true,
			HasSharedBufferSupport:          true,
			HasSharedMemoryTransportSupport: true,
		})
		if parseInitErr != nil {
			parseB.Fatalf("InitCapabilityReport returned error: %v", parseInitErr)
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			_ = runtime2.GetCapabilityReport()
		}
	})

	parseB.Run("uninitialized-default-read", func(parseB *testing.B) {
		runtime2.ResetCapabilityReport()
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			_ = runtime2.GetCapabilityReport()
		}
	})
}
