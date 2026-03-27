package runtime2

import (
	"fmt"
	"testing"
)

// TestHandleWorkerRegionUpdateRespondsToSnapshotInputChanges verifies snapshot props, source values, and source version each affect rendered output.
func TestHandleWorkerRegionUpdateRespondsToSnapshotInputChanges(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		parseProps, _ := parseMount.RenderInput.GetProps.(map[string]any)
		parseStatus := ""
		for _, parseSourceEntry := range parseMount.RenderInput.GetSourceEntries {
			if parseSourceEntry.GetSourceID == "status" {
				parseStatus = fmt.Sprintf("%v", parseSourceEntry.GetSourceValue)
				break
			}
		}
		return map[string]any{
			"kind": "text",
			"text": fmt.Sprintf("%v|%s|%d", parseProps["title"], parseStatus, parseMount.RenderInput.GetSourceVersion),
		}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	parseMountSnapshot, parseMountSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		1,
		map[string]any{"title": "A"},
		[]string{"status"},
		map[string]any{"status": "healthy"},
		map[string]uint64{"status": 7},
	)
	if parseMountSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(mount) returned error: %v", parseMountSnapshotErr)
	}
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseMountSnapshot,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount returned error: %v", parseMountErr)
	}
	parsePropsChangeSnapshot, parsePropsChangeSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		2,
		map[string]any{"title": "B"},
		[]string{"status"},
		map[string]any{"status": "healthy"},
		map[string]uint64{"status": 7},
	)
	if parsePropsChangeSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(props change) returned error: %v", parsePropsChangeSnapshotErr)
	}
	parsePropsChangeResult, parsePropsChangeErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 2,
		Snapshot:     parsePropsChangeSnapshot,
	})
	if parsePropsChangeErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(props change) returned error: %v", parsePropsChangeErr)
	}
	if !parsePropsChangeResult.HasPatchReady {
		parseTesting.Fatalf("expected patch-ready for props change, got %+v", parsePropsChangeResult)
	}
	parseSourceChangeSnapshot, parseSourceChangeSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		3,
		map[string]any{"title": "B"},
		[]string{"status"},
		map[string]any{"status": "degraded"},
		map[string]uint64{"status": 7},
	)
	if parseSourceChangeSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(source change) returned error: %v", parseSourceChangeSnapshotErr)
	}
	parseSourceChangeResult, parseSourceChangeErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 3,
		Snapshot:     parseSourceChangeSnapshot,
	})
	if parseSourceChangeErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(source change) returned error: %v", parseSourceChangeErr)
	}
	if !parseSourceChangeResult.HasPatchReady {
		parseTesting.Fatalf("expected patch-ready for source value change, got %+v", parseSourceChangeResult)
	}
	parseSourceVersionChangeSnapshot, parseSourceVersionChangeSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		4,
		map[string]any{"title": "B"},
		[]string{"status"},
		map[string]any{"status": "degraded"},
		map[string]uint64{"status": 8},
	)
	if parseSourceVersionChangeSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(source version change) returned error: %v", parseSourceVersionChangeSnapshotErr)
	}
	parseSourceVersionChangeResult, parseSourceVersionChangeErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 4,
		Snapshot:     parseSourceVersionChangeSnapshot,
	})
	if parseSourceVersionChangeErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(source version change) returned error: %v", parseSourceVersionChangeErr)
	}
	if !parseSourceVersionChangeResult.HasPatchReady {
		parseTesting.Fatalf("expected patch-ready for source version change, got %+v", parseSourceVersionChangeResult)
	}
}
