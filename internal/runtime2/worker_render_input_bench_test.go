package runtime2

import (
	"fmt"
	"testing"
)

// BenchmarkHandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly compares snapshot-driven worker render invocation against metadata-only update specs.
func BenchmarkHandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly(parseB *testing.B) {
	parseB.Run("snapshot-driven", func(parseB *testing.B) {
		parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
		parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
			parseProps, _ := parseMount.RenderInput.GetProps.(map[string]any)
			return map[string]any{
				"kind": "text",
				"text": fmt.Sprintf("%v", parseProps["title"]),
			}, nil
		})
		if parseRegisterErr != nil {
			parseB.Fatalf("RegisterWorkerRegionRenderer(snapshot-driven) returned error: %v", parseRegisterErr)
		}
		if parseTrustErr := parseWorkerRegionRuntime.SetWorkerRegionRendererTrusted("dashboard.hot-panel", true); parseTrustErr != nil {
			parseB.Fatalf("SetWorkerRegionRendererTrusted(snapshot-driven) returned error: %v", parseTrustErr)
		}
		parseWorkerRegionRuntime.SetWorkerRegionUpdateValidationEnabled(false)
		parseMountSnapshot, parseMountSnapshotErr := BuildSnapshotEnvelope(
			"region-bench",
			1,
			1,
			map[string]any{"title": "0"},
			nil,
			nil,
			nil,
		)
		if parseMountSnapshotErr != nil {
			parseB.Fatalf("BuildSnapshotEnvelope(mount snapshot-driven) returned error: %v", parseMountSnapshotErr)
		}
		_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
			RegionID:     "region-bench",
			RendererID:   "dashboard.hot-panel",
			Epoch:        1,
			InputVersion: 1,
			Snapshot:     parseMountSnapshot,
		})
		if parseMountErr != nil {
			parseB.Fatalf("HandleWorkerRegionMount(snapshot-driven) returned error: %v", parseMountErr)
		}
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseSnapshot, parseSnapshotErr := BuildSnapshotEnvelope(
				"region-bench",
				1,
				uint64(parseIndex+2),
				map[string]any{"title": fmt.Sprintf("%d", parseIndex+1)},
				nil,
				nil,
				nil,
			)
			if parseSnapshotErr != nil {
				parseB.Fatalf("BuildSnapshotEnvelope(update snapshot-driven) returned error: %v", parseSnapshotErr)
			}
			if _, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
				RegionID:     "region-bench",
				RendererID:   "dashboard.hot-panel",
				Epoch:        1,
				InputVersion: uint64(parseIndex + 2),
				Snapshot:     parseSnapshot,
			}); parseUpdateErr != nil {
				parseB.Fatalf("HandleWorkerRegionUpdate(snapshot-driven) returned error: %v", parseUpdateErr)
			}
		}
	})
	parseB.Run("metadata-only", func(parseB *testing.B) {
		parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
		parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
			return map[string]any{
				"kind": "text",
				"text": fmt.Sprintf("%d", parseMount.InputVersion),
			}, nil
		})
		if parseRegisterErr != nil {
			parseB.Fatalf("RegisterWorkerRegionRenderer(metadata-only) returned error: %v", parseRegisterErr)
		}
		if parseTrustErr := parseWorkerRegionRuntime.SetWorkerRegionRendererTrusted("dashboard.hot-panel", true); parseTrustErr != nil {
			parseB.Fatalf("SetWorkerRegionRendererTrusted(metadata-only) returned error: %v", parseTrustErr)
		}
		parseWorkerRegionRuntime.SetWorkerRegionUpdateValidationEnabled(false)
		_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
			RegionID:     "region-bench",
			RendererID:   "dashboard.hot-panel",
			Epoch:        1,
			InputVersion: 1,
		})
		if parseMountErr != nil {
			parseB.Fatalf("HandleWorkerRegionMount(metadata-only) returned error: %v", parseMountErr)
		}
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
				RegionID:     "region-bench",
				RendererID:   "dashboard.hot-panel",
				Epoch:        1,
				InputVersion: uint64(parseIndex + 2),
			}); parseUpdateErr != nil {
				parseB.Fatalf("HandleWorkerRegionUpdate(metadata-only) returned error: %v", parseUpdateErr)
			}
		}
	})
}
