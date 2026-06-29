package runtime2

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/pluginruntime"
)

// TestBuildRuntime2MetaServiceNormalizesCapabilityReport verifies runtime2 capability metadata is exposed through the plugin service.
func TestBuildRuntime2MetaServiceNormalizesCapabilityReport(parseT *testing.T) {
	parseT.Cleanup(ResetCapabilityReport)
	parseT.Cleanup(ResetRuntime2MetaProvider)
	if parseErr := SetCapabilityReportOverride(CapabilityReport{
		HasWorkerSupport:          true,
		HasStructuredCloneSupport: true,
		HasBinaryTransportSupport: true,
	}); parseErr != nil {
		parseT.Fatalf("SetCapabilityReportOverride() error = %v", parseErr)
	}
	getService := BuildRuntime2MetaService()
	getSnapshot, parseErr := getService.GetRuntime2MetaSnapshot(pluginruntime.QueryBudget{})
	if parseErr != nil {
		parseT.Fatalf("GetRuntime2MetaSnapshot() error = %v", parseErr)
	}
	if getSnapshot.Meta.BackendID != string(pluginruntime.BackendIDRuntime2) {
		parseT.Fatalf("expected runtime2 backend metadata, got %+v", getSnapshot.Meta)
	}
	if !getSnapshot.Capabilities["hasWorkerSupport"] || !getSnapshot.Capabilities["hasStructuredCloneSupport"] || !getSnapshot.Capabilities["hasBinaryTransportSupport"] {
		parseT.Fatalf("expected runtime2 capabilities to be preserved, got %+v", getSnapshot.Capabilities)
	}
}

// TestBuildRuntime2MetaServiceMergesProviderSnapshot verifies provider-owned region and diagnostic snapshots compose with capability metadata.
func TestBuildRuntime2MetaServiceMergesProviderSnapshot(parseT *testing.T) {
	parseT.Cleanup(ResetCapabilityReport)
	parseT.Cleanup(ResetRuntime2MetaProvider)
	if parseErr := SetCapabilityReportOverride(CapabilityReport{
		HasWorkerSupport:                true,
		HasStructuredCloneSupport:       true,
		HasSharedMemoryTransportSupport: true,
		HasSharedBufferSupport:          true,
	}); parseErr != nil {
		parseT.Fatalf("SetCapabilityReportOverride() error = %v", parseErr)
	}
	SetRuntime2MetaProvider(func(parseBudget pluginruntime.QueryBudget) (pluginruntime.Runtime2MetaSnapshot, error) {
		return pluginruntime.Runtime2MetaSnapshot{
			Meta: pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime2, true),
			Regions: []pluginruntime.Runtime2RegionSnapshot{{
				RegionInstanceID:            "region-1",
				RegionMode:                  "worker-attached",
				RendererID:                  "dashboard.hot-panel",
				TransportTier:               "shared-buffer",
				DiagnosticCount:             1,
				DispatchToCommitNs:          42,
				RepairTriggeredRemountCount: 2,
			}},
			Diagnostics: []pluginruntime.Runtime2Diagnostic{{
				RegionInstanceID: "region-1",
				Type:             string(DiagnosticEventKindPatchReady),
				Text:             "patch ready",
				TransportTier:    string(TransportTierSharedBuffer),
			}},
		}, nil
	})
	getService := BuildRuntime2MetaService()
	getSnapshot, parseErr := getService.GetRuntime2MetaSnapshot(pluginruntime.QueryBudget{MaxItems: 1})
	if parseErr != nil {
		parseT.Fatalf("GetRuntime2MetaSnapshot() error = %v", parseErr)
	}
	if !getSnapshot.Meta.Truncated {
		parseT.Fatalf("expected provider truncation bit to survive, got %+v", getSnapshot.Meta)
	}
	if len(getSnapshot.Regions) != 1 || getSnapshot.Regions[0].RegionInstanceID != "region-1" {
		parseT.Fatalf("expected provider region snapshot, got %+v", getSnapshot.Regions)
	}
	if len(getSnapshot.Diagnostics) != 1 || getSnapshot.Diagnostics[0].Type != string(DiagnosticEventKindPatchReady) {
		parseT.Fatalf("expected provider diagnostics snapshot, got %+v", getSnapshot.Diagnostics)
	}
	if !getSnapshot.Capabilities["hasWorkerSupport"] || !getSnapshot.Capabilities["hasSharedMemoryTransportSupport"] {
		parseT.Fatalf("expected provider snapshot to inherit capability metadata, got %+v", getSnapshot.Capabilities)
	}
}
