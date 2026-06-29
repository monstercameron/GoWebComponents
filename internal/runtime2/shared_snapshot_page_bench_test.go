package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// BenchmarkHandleSharedSnapshotPublishAndRead benchmarks shared-page publish and read operations.
func BenchmarkHandleSharedSnapshotPublishAndRead(parseB *testing.B) {
	parseSharedSnapshotPage, parseErr := runtime2.BuildSharedSnapshotPage(4096)
	if parseErr != nil {
		parseB.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parsePayload := []byte(`{"region_instance_id":"region-1","epoch":1,"input_version":1,"props":{"status":"ok"}}`)

	parseB.Run("publish", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := parseSharedSnapshotPage.HandleSharedSnapshotPublishPayload(parsePayload); parseErr != nil {
				parseB.Fatalf("HandleSharedSnapshotPublishPayload returned error: %v", parseErr)
			}
		}
	})

	if _, parseErr := parseSharedSnapshotPage.HandleSharedSnapshotPublishPayload(parsePayload); parseErr != nil {
		parseB.Fatalf("HandleSharedSnapshotPublishPayload warmup returned error: %v", parseErr)
	}
	parseB.Run("read", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := parseSharedSnapshotPage.GetSharedSnapshotReadPayload(); parseErr != nil {
				parseB.Fatalf("GetSharedSnapshotReadPayload returned error: %v", parseErr)
			}
		}
	})
}
