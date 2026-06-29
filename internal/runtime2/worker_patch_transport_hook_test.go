package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// buildWorkerRuntimeForPatchTransportTests creates one worker runtime with one renderer used for patch-transport tests.
func buildWorkerRuntimeForPatchTransportTests(parseT *testing.T, parseIsConstantRender bool) *runtime2.WorkerRegionRuntime {
	parseT.Helper()
	parseWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
		parseRenderVersion := parseMount.InputVersion
		if parseIsConstantRender {
			parseRenderVersion = 1
		}
		return map[string]any{
			"version": parseRenderVersion,
		}, nil
	})
	if parseRegisterErr != nil {
		parseT.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(runtime2.WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
	})
	if parseMountErr != nil {
		parseT.Fatalf("HandleWorkerRegionMount returned error: %v", parseMountErr)
	}
	return parseWorkerRegionRuntime
}

// buildPatchTransportCapabilityReport creates one capability report that supports structured-clone patch transport.
func buildPatchTransportCapabilityReport() runtime2.CapabilityReport {
	return runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	})
}

// TestHandleWorkerRegionUpdateWithPatchTransportBuildsPatchReadyPayload verifies changed worker updates select patch transport and emit patch-ready envelopes.
func TestHandleWorkerRegionUpdateWithPatchTransportBuildsPatchReadyPayload(parseT *testing.T) {
	parseWorkerRegionRuntime := buildWorkerRuntimeForPatchTransportTests(parseT, false)
	parseResult, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		runtime2.WorkerRegionUpdateSpec{
			RegionID:     "region-1",
			Epoch:        1,
			InputVersion: 2,
		},
		buildPatchTransportCapabilityReport(),
	)
	if parseErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdateWithPatchTransport returned error: %v", parseErr)
	}
	if !parseResult.GetUpdateResult.HasPatchReady {
		parseT.Fatal("expected changed update to produce patch-ready result")
	}
	if !parseResult.HasPatchPayload {
		parseT.Fatal("expected patch payload for patch-ready result")
	}
	if parseResult.GetTransportTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone patch transport tier, got %q", parseResult.GetTransportTier)
	}
	if !parseResult.HasPatchReadyEnvelope {
		parseT.Fatal("expected patch-ready control envelope output")
	}
	if parseResult.GetPatchReadyEnvelope.Kind != runtime2.ControlKindPatchReady {
		parseT.Fatalf("expected patch-ready control kind, got %q", parseResult.GetPatchReadyEnvelope.Kind)
	}
	if parseResult.GetPatchReadyEnvelope.PatchVersion != 2 {
		parseT.Fatalf("expected patch-ready version 2, got %d", parseResult.GetPatchReadyEnvelope.PatchVersion)
	}
}

// TestHandleWorkerRegionUpdateWithPatchTransportSkipsNoOp verifies no-op worker updates do not emit patch transport payloads.
func TestHandleWorkerRegionUpdateWithPatchTransportSkipsNoOp(parseT *testing.T) {
	parseWorkerRegionRuntime := buildWorkerRuntimeForPatchTransportTests(parseT, true)
	parseResult, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		runtime2.WorkerRegionUpdateSpec{
			RegionID:     "region-1",
			Epoch:        1,
			InputVersion: 2,
		},
		buildPatchTransportCapabilityReport(),
	)
	if parseErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdateWithPatchTransport returned error: %v", parseErr)
	}
	if !parseResult.GetUpdateResult.IsNoOp {
		parseT.Fatal("expected no-op worker update result")
	}
	if parseResult.HasPatchPayload {
		parseT.Fatal("expected no patch payload for no-op worker update")
	}
}

// TestHandleWorkerRegionUpdateWithPatchTransportRejectsUnsupportedCapability verifies patch-ready updates fail when no supported patch transport is available.
func TestHandleWorkerRegionUpdateWithPatchTransportRejectsUnsupportedCapability(parseT *testing.T) {
	parseWorkerRegionRuntime := buildWorkerRuntimeForPatchTransportTests(parseT, false)
	_, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		runtime2.WorkerRegionUpdateSpec{
			RegionID:     "region-1",
			Epoch:        1,
			InputVersion: 2,
		},
		runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
			HasWorkerSupport: true,
		}),
	)
	if parseErr == nil {
		parseT.Fatal("expected unsupported patch transport capability to fail")
	}
}

// TestHandleWorkerRegionUpdateWithPatchTransportCarriesPatchVersionAfterNoOpGap verifies patch-ready envelopes carry patch-stream patch versions when no-op updates create input-version gaps.
func TestHandleWorkerRegionUpdateWithPatchTransportCarriesPatchVersionAfterNoOpGap(parseT *testing.T) {
	parseWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
		parseRenderText := "stable"
		if parseMount.InputVersion >= 3 {
			parseRenderText = "changed"
		}
		return map[string]any{
			"kind": "text",
			"text": parseRenderText,
		}, nil
	})
	if parseRegisterErr != nil {
		parseT.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(runtime2.WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
	})
	if parseMountErr != nil {
		parseT.Fatalf("HandleWorkerRegionMount returned error: %v", parseMountErr)
	}
	parseNoOpResult, parseNoOpErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		runtime2.WorkerRegionUpdateSpec{
			RegionID:     "region-1",
			Epoch:        1,
			InputVersion: 2,
		},
		buildPatchTransportCapabilityReport(),
	)
	if parseNoOpErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdateWithPatchTransport(no-op) returned error: %v", parseNoOpErr)
	}
	if !parseNoOpResult.GetUpdateResult.IsNoOp {
		parseT.Fatalf("expected no-op update at input version 2, got %+v", parseNoOpResult.GetUpdateResult)
	}
	parsePatchResult, parsePatchErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		runtime2.WorkerRegionUpdateSpec{
			RegionID:     "region-1",
			Epoch:        1,
			InputVersion: 3,
		},
		buildPatchTransportCapabilityReport(),
	)
	if parsePatchErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdateWithPatchTransport(patch) returned error: %v", parsePatchErr)
	}
	if !parsePatchResult.GetUpdateResult.HasPatchReady || !parsePatchResult.HasPatchReadyEnvelope {
		parseT.Fatalf("expected patch-ready envelope after no-op gap, got %+v", parsePatchResult)
	}
	if parsePatchResult.GetPatchReadyEnvelope.PatchVersion != parsePatchResult.GetUpdateResult.PatchIR.GetHeader.PatchVersion {
		parseT.Fatalf(
			"expected patch-ready patch version %d to match patch stream header, got %d",
			parsePatchResult.GetUpdateResult.PatchIR.GetHeader.PatchVersion,
			parsePatchResult.GetPatchReadyEnvelope.PatchVersion,
		)
	}
	if parsePatchResult.GetPatchReadyEnvelope.InputVersion != 3 {
		parseT.Fatalf("expected patch-ready input version 3, got %d", parsePatchResult.GetPatchReadyEnvelope.InputVersion)
	}
	if parsePatchResult.GetPatchReadyEnvelope.PatchVersion == parsePatchResult.GetPatchReadyEnvelope.InputVersion {
		parseT.Fatalf(
			"expected patch-ready patch and input versions to diverge after no-op gap, got patch=%d input=%d",
			parsePatchResult.GetPatchReadyEnvelope.PatchVersion,
			parsePatchResult.GetPatchReadyEnvelope.InputVersion,
		)
	}
}
