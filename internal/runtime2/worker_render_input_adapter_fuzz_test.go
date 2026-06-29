package runtime2

import "testing"

// FuzzBuildWorkerRenderInputAndDiffPath fuzzes snapshot-to-render input adaptation and worker update flow for panic safety.
func FuzzBuildWorkerRenderInputAndDiffPath(parseF *testing.F) {
	parseF.Add("region-a", "title-a", "status", "healthy", uint64(1))
	parseF.Add("region-a", "", "status", "", uint64(2))
	parseF.Fuzz(func(parseT *testing.T, parseRegionID string, parseTitle string, parseSourceID string, parseSourceValue string, parseSourceVersion uint64) {
		if parseRegionID == "" {
			parseRegionID = "region-fuzz"
		}
		if parseSourceID == "" {
			parseSourceID = "status"
		}
		if parseSourceVersion == 0 {
			parseSourceVersion = 1
		}
		parseMountSnapshot, parseMountSnapshotErr := BuildSnapshotEnvelope(
			RegionInstanceID(parseRegionID),
			1,
			1,
			map[string]any{"title": parseTitle},
			[]string{parseSourceID},
			map[string]any{parseSourceID: parseSourceValue},
			map[string]uint64{parseSourceID: parseSourceVersion},
		)
		if parseMountSnapshotErr != nil {
			return
		}
		_, _ = BuildWorkerRenderInput(parseMountSnapshot)
		parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
		parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
			return map[string]any{
				"kind": "text",
				"text": parseTitle,
			}, nil
		})
		if parseRegisterErr != nil {
			return
		}
		_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
			RegionID:     parseRegionID,
			RendererID:   "dashboard.hot-panel",
			Epoch:        1,
			InputVersion: 1,
			Snapshot:     parseMountSnapshot,
		})
		if parseMountErr != nil {
			return
		}
		parseUpdateSnapshot, parseUpdateSnapshotErr := BuildSnapshotEnvelope(
			RegionInstanceID(parseRegionID),
			1,
			2,
			map[string]any{"title": parseTitle + "-next"},
			[]string{parseSourceID},
			map[string]any{parseSourceID: parseSourceValue + "-next"},
			map[string]uint64{parseSourceID: parseSourceVersion + 1},
		)
		if parseUpdateSnapshotErr != nil {
			return
		}
		_, _ = parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
			RegionID:     parseRegionID,
			RendererID:   "dashboard.hot-panel",
			Epoch:        1,
			InputVersion: 2,
			Snapshot:     parseUpdateSnapshot,
		})
	})
}
