package runtime2_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestBinaryTransportEndToEndHostSnapshotToWorkerUpdate verifies one display-only region flows through host snapshot capture, binary transport, and worker update.
func TestBinaryTransportEndToEndHostSnapshotToWorkerUpdate(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}

	parseSourceValue := "healthy"
	parseSourceVersion := uint64(3)
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"status": parseSourceValue,
				},
				map[string]uint64{
					"status": parseSourceVersion,
				},
				nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}

	getSnapshotEnvelope, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"status"},
		},
		2,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot returned error: %v", parseErr)
	}

	getTransportTier, getSnapshotPayload, parseErr := runtime2.BuildSnapshotTransportPayloadWithFallback(
		getSnapshotEnvelope,
		runtime2.CapabilityReport{
			HasWorkerSupport:          true,
			HasStructuredCloneSupport: true,
			HasBinaryTransportSupport: true,
		},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildSnapshotTransportPayloadWithFallback returned error: %v", parseErr)
	}
	if getTransportTier != runtime2.TransportTierBinary {
		parseT.Fatalf("expected binary snapshot transport tier, got %q", getTransportTier)
	}
	getDecodedTransportTier, getDecodedSnapshotEnvelope, parseErr := runtime2.ParseSnapshotTransportPayloadWithFallback(getTransportTier, getSnapshotPayload)
	if parseErr != nil {
		parseT.Fatalf("ParseSnapshotTransportPayloadWithFallback returned error: %v", parseErr)
	}
	if getDecodedTransportTier != runtime2.TransportTierBinary {
		parseT.Fatalf("expected binary decode tier, got %q", getDecodedTransportTier)
	}
	if getDecodedSnapshotEnvelope.RegionInstanceID != runtime2.RegionInstanceID("region-1") {
		parseT.Fatalf("expected decoded snapshot region-1, got %#v", getDecodedSnapshotEnvelope)
	}

	getWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	parseErr = getWorkerRegionRuntime.RegisterWorkerRegionRenderer(
		"dashboard.hot-panel",
		func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
			return map[string]any{
				"kind": "host-element",
				"tag":  "div",
				"children": []any{
					map[string]any{
						"kind": "text",
						"text": fmt.Sprintf("version %d", parseMount.InputVersion),
					},
				},
			}, nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseErr)
	}

	getMountPayload, parseErr := runtime2.BuildBinaryMountEnvelope(runtime2.BinaryMountEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		SourceIDs:        []string{"status"},
		Snapshot:         getDecodedSnapshotEnvelope,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryMountEnvelope returned error: %v", parseErr)
	}
	getMountEnvelope, parseErr := runtime2.ParseBinaryMountEnvelope(getMountPayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinaryMountEnvelope returned error: %v", parseErr)
	}
	getWorkerRegionState, parseErr := getWorkerRegionRuntime.HandleWorkerRegionMount(runtime2.WorkerRegionMountSpec{
		RegionID:     string(getMountEnvelope.RegionInstanceID),
		RendererID:   string(getMountEnvelope.RendererID),
		Epoch:        getMountEnvelope.Snapshot.Epoch,
		InputVersion: getMountEnvelope.Snapshot.InputVersion,
	})
	if parseErr != nil {
		parseT.Fatalf("HandleWorkerRegionMount returned error: %v", parseErr)
	}
	if getWorkerRegionState.InputVersion != 2 {
		parseT.Fatalf("expected mounted worker input version 2, got %d", getWorkerRegionState.InputVersion)
	}

	parseSourceValue = "degraded"
	parseSourceVersion = 4
	getUpdateSnapshotEnvelope, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"status"},
		},
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(second) returned error: %v", parseErr)
	}

	getUpdatePayload, parseErr := runtime2.BuildBinaryUpdateEnvelope(runtime2.BinaryUpdateEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		InputVersion:     3,
		Snapshot:         getUpdateSnapshotEnvelope,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryUpdateEnvelope returned error: %v", parseErr)
	}
	getUpdateEnvelope, parseErr := runtime2.ParseBinaryUpdateEnvelope(getUpdatePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinaryUpdateEnvelope returned error: %v", parseErr)
	}
	getUpdateResult, parseErr := getWorkerRegionRuntime.HandleWorkerRegionUpdate(runtime2.WorkerRegionUpdateSpec{
		RegionID:     string(getUpdateEnvelope.RegionInstanceID),
		Epoch:        getUpdateEnvelope.Snapshot.Epoch,
		InputVersion: getUpdateEnvelope.InputVersion,
	})
	if parseErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdate returned error: %v", parseErr)
	}
	if !getUpdateResult.HasPatchReady {
		parseT.Fatalf("expected binary end-to-end worker update to produce patch-ready, got %+v", getUpdateResult)
	}
}
