package runtime2

import "testing"

// TestHandleWorkerRegionUpdateTrustedRendererSkipsValidationWhenDisabled verifies trusted renderer updates can skip repeated output validation while mount remains strict.
func TestHandleWorkerRegionUpdateTrustedRendererSkipsValidationWhenDisabled(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		if parseMount.InputVersion == 1 {
			return map[string]any{
				"kind": "host-element",
				"tag":  "div",
				"text": "ready",
			}, nil
		}
		return map[string]any{
			"kind":   "host-element",
			"tag":    "div",
			"portal": "#app-root",
			"text":   "trusted-update",
		}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	if parseTrustErr := parseWorkerRegionRuntime.SetWorkerRegionRendererTrusted("dashboard.hot-panel", true); parseTrustErr != nil {
		parseTesting.Fatalf("SetWorkerRegionRendererTrusted returned error: %v", parseTrustErr)
	}
	parseWorkerRegionRuntime.SetWorkerRegionUpdateValidationEnabled(false)
	parseMountSnapshot, parseMountSnapshotErr := BuildSnapshotEnvelope(
		"region-trusted",
		1,
		1,
		map[string]any{"title": "A"},
		nil,
		nil,
		nil,
	)
	if parseMountSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(mount) returned error: %v", parseMountSnapshotErr)
	}
	if _, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-trusted",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseMountSnapshot,
	}); parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount returned error: %v", parseMountErr)
	}
	parseUpdateSnapshot, parseUpdateSnapshotErr := BuildSnapshotEnvelope(
		"region-trusted",
		1,
		2,
		map[string]any{"title": "B"},
		nil,
		nil,
		nil,
	)
	if parseUpdateSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(update) returned error: %v", parseUpdateSnapshotErr)
	}
	if _, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-trusted",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 2,
		Snapshot:     parseUpdateSnapshot,
	}); parseUpdateErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(trusted skip) returned error: %v", parseUpdateErr)
	}
	parseWorkerRegionRuntime.SetWorkerRegionUpdateValidationEnabled(true)
	if _, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-trusted",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 3,
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: RegionInstanceID("region-trusted"),
			Epoch:            1,
			InputVersion:     3,
			Props:            map[string]any{"title": "C"},
		},
	}); parseUpdateErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(validation enabled) expected validation error")
	}
}

// TestHandleWorkerRegionUpdateTrustedRendererRequiresDisplayMetadata verifies trusted update-skip only applies to renderers that explicitly advertise display-only metadata.
func TestHandleWorkerRegionUpdateTrustedRendererRequiresDisplayMetadata(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata(
		"dashboard.hot-panel",
		func(parseMount WorkerRegionMountSpec) (any, error) {
			if parseMount.InputVersion == 1 {
				return map[string]any{
					"kind": "host-element",
					"tag":  "div",
					"text": "ready",
				}, nil
			}
			return map[string]any{
				"kind":   "host-element",
				"tag":    "div",
				"portal": "#app-root",
				"text":   "trusted-update",
			}, nil
		},
		RendererMetadata{},
	)
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRendererWithMetadata returned error: %v", parseRegisterErr)
	}
	if parseTrustErr := parseWorkerRegionRuntime.SetWorkerRegionRendererTrusted("dashboard.hot-panel", true); parseTrustErr != nil {
		parseTesting.Fatalf("SetWorkerRegionRendererTrusted returned error: %v", parseTrustErr)
	}
	parseWorkerRegionRuntime.SetWorkerRegionUpdateValidationEnabled(false)
	parseMountSnapshot, parseMountSnapshotErr := BuildSnapshotEnvelope(
		"region-trusted",
		1,
		1,
		map[string]any{"title": "A"},
		nil,
		nil,
		nil,
	)
	if parseMountSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(mount) returned error: %v", parseMountSnapshotErr)
	}
	if _, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-trusted",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseMountSnapshot,
	}); parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount returned error: %v", parseMountErr)
	}
	parseUpdateSnapshot, parseUpdateSnapshotErr := BuildSnapshotEnvelope(
		"region-trusted",
		1,
		2,
		map[string]any{"title": "B"},
		nil,
		nil,
		nil,
	)
	if parseUpdateSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(update) returned error: %v", parseUpdateSnapshotErr)
	}
	if _, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-trusted",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 2,
		Snapshot:     parseUpdateSnapshot,
	}); parseUpdateErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(expected metadata-gated validation) error = nil, want validation error")
	}
}
