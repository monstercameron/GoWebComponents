package runtime2

import "testing"

// buildHostWorkerOrchestrationHarness constructs one mounted host and worker pair with seeded DOM state for orchestration helper tests.
func buildHostWorkerOrchestrationHarness(
	parseT testing.TB,
	parseRenderer WorkerRegionRenderer,
	parseInitialProps map[string]any,
	parseInitialRenderOutput any,
) (*HostRegionAdapter, *WorkerRegionRuntime) {
	parseT.Helper()

	buildHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}

	buildWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := buildWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", parseRenderer)
	if parseRegisterErr != nil {
		parseT.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}

	parseInitialSnapshot, parseInitialSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		parseInitialProps,
		nil,
		nil,
		nil,
	)
	if parseInitialSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope(initial) returned error: %v", parseInitialSnapshotErr)
	}
	parsePreviousIR := parseBuildCanonicalIRForHostPatchTest(parseT, parseInitialRenderOutput)
	parseSeedRegionDOMIndexFromCanonical(parseT, buildHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parsePreviousIR)
	_, parseWorkerMountErr := buildWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseInitialSnapshot,
	})
	if parseWorkerMountErr != nil {
		parseT.Fatalf("HandleWorkerRegionMount returned error: %v", parseWorkerMountErr)
	}
	return buildHostRegionAdapter, buildWorkerRegionRuntime
}

// buildHostWorkerOrchestrationCapabilityReport constructs one capability report that keeps orchestration on structured-clone transport.
func buildHostWorkerOrchestrationCapabilityReport() CapabilityReport {
	return CapabilityReport{
		HasWorkerSupport:          true,
		HasStructuredCloneSupport: true,
	}
}

// TestHandleHostWorkerRegionUpdateOrchestrationCommitsPatch verifies one helper chains host dispatch, worker update, transport encode/decode, and host commit.
func TestHandleHostWorkerRegionUpdateOrchestrationCommitsPatch(parseT *testing.T) {
	buildHostRegionAdapter, buildWorkerRegionRuntime := buildHostWorkerOrchestrationHarness(
		parseT,
		func(parseMount WorkerRegionMountSpec) (any, error) {
			parseProps, _ := parseMount.Snapshot.Props.(map[string]any)
			return map[string]any{
				"kind": "text",
				"text": parseProps["title"],
			}, nil
		},
		map[string]any{"title": "before"},
		map[string]any{"kind": "text", "text": "before"},
	)
	parseSpec := ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
		Props:            map[string]any{"title": "after"},
	}
	parseOrchestrationResult, parseOrchestrationErr := HandleHostWorkerRegionUpdateOrchestration(
		buildHostRegionAdapter,
		buildWorkerRegionRuntime,
		parseSpec,
		2,
		buildHostWorkerOrchestrationCapabilityReport(),
		nil,
		nil,
		nil,
	)
	if parseOrchestrationErr != nil {
		parseT.Fatalf("HandleHostWorkerRegionUpdateOrchestration returned error: %v", parseOrchestrationErr)
	}
	if !parseOrchestrationResult.HasPatchCommitted {
		parseT.Fatalf("expected orchestration helper to commit patch, got %+v", parseOrchestrationResult)
	}
	parseNextIR, parseNextIRErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "after"})
	if parseNextIRErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextIRErr)
	}
	parseNextTree, parseTreeErr := ParseCanonicalRenderTree(parseNextIR)
	if parseTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree(next) returned error: %v", parseTreeErr)
	}
	parseRootNode := parseNextTree.getNodeByID[parseNextTree.getRootNodeID]
	parseDOMNode, parseLookupErr := buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", parseRootNode.getNodeID)
	if parseLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseLookupErr)
	}
	if parseDOMNode.GetText != "after" {
		parseT.Fatalf("GetRegionDOMNode(root) text = %q, want %q", parseDOMNode.GetText, "after")
	}
}

// TestHandleHostWorkerRegionUpdateOrchestrationRejectsNilAndDispatchErrors verifies orchestration rejects missing handles and propagates host dispatch failures.
func TestHandleHostWorkerRegionUpdateOrchestrationRejectsNilAndDispatchErrors(parseT *testing.T) {
	if _, parseErr := HandleHostWorkerRegionUpdateOrchestration(
		nil,
		BuildWorkerRegionRuntime(),
		ParallelRegionSpec{},
		1,
		buildHostWorkerOrchestrationCapabilityReport(),
		nil,
		nil,
		nil,
	); parseErr == nil {
		parseT.Fatal("expected nil host region adapter to fail")
	}
	buildHostRegionAdapter, buildWorkerRegionRuntime := buildHostWorkerOrchestrationHarness(
		parseT,
		func(parseMount WorkerRegionMountSpec) (any, error) {
			return map[string]any{"kind": "text", "text": "ready"}, nil
		},
		map[string]any{"title": "ready"},
		map[string]any{"kind": "text", "text": "ready"},
	)
	if _, parseErr := HandleHostWorkerRegionUpdateOrchestration(
		buildHostRegionAdapter,
		nil,
		ParallelRegionSpec{
			RendererID:       RendererID("dashboard.hot-panel"),
			RegionInstanceID: RegionInstanceID("region-1"),
		},
		1,
		buildHostWorkerOrchestrationCapabilityReport(),
		nil,
		nil,
		nil,
	); parseErr == nil {
		parseT.Fatal("expected nil worker region runtime to fail")
	}
	if _, parseErr := HandleHostWorkerRegionUpdateOrchestration(
		buildHostRegionAdapter,
		buildWorkerRegionRuntime,
		ParallelRegionSpec{
			RendererID:       RendererID("dashboard.hot-panel"),
			RegionInstanceID: RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "ready"},
		},
		0,
		buildHostWorkerOrchestrationCapabilityReport(),
		nil,
		nil,
		nil,
	); parseErr == nil {
		parseT.Fatal("expected invalid input version to fail host dispatch")
	}
}

// TestHandleHostWorkerRegionUpdateOrchestrationReturnsEarlyForNoPatchAndNoChange verifies orchestration short-circuits when worker output is unchanged and when host dispatch detects no input change.
func TestHandleHostWorkerRegionUpdateOrchestrationReturnsEarlyForNoPatchAndNoChange(parseT *testing.T) {
	buildHostRegionAdapter, buildWorkerRegionRuntime := buildHostWorkerOrchestrationHarness(
		parseT,
		func(parseMount WorkerRegionMountSpec) (any, error) {
			return map[string]any{"kind": "text", "text": "stable"}, nil
		},
		map[string]any{"title": "stable"},
		map[string]any{"kind": "text", "text": "stable"},
	)
	parseSpec := ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
		Props:            map[string]any{"title": "stable"},
	}
	parseFirstResult, parseFirstErr := HandleHostWorkerRegionUpdateOrchestration(
		buildHostRegionAdapter,
		buildWorkerRegionRuntime,
		parseSpec,
		2,
		buildHostWorkerOrchestrationCapabilityReport(),
		nil,
		nil,
		nil,
	)
	if parseFirstErr != nil {
		parseT.Fatalf("HandleHostWorkerRegionUpdateOrchestration(first) returned error: %v", parseFirstErr)
	}
	if !parseFirstResult.GetDispatchResult.GetDispatchResult.HasScheduled {
		parseT.Fatalf("expected first orchestration dispatch to schedule, got %+v", parseFirstResult)
	}
	if parseFirstResult.GetWorkerUpdateResult.HasPatchPayload {
		parseT.Fatalf("expected unchanged worker render output to produce no patch payload, got %+v", parseFirstResult)
	}
	if parseFirstResult.HasPatchCommitted {
		parseT.Fatalf("expected unchanged worker render output to skip patch commit, got %+v", parseFirstResult)
	}

	parseSecondResult, parseSecondErr := HandleHostWorkerRegionUpdateOrchestration(
		buildHostRegionAdapter,
		buildWorkerRegionRuntime,
		parseSpec,
		3,
		buildHostWorkerOrchestrationCapabilityReport(),
		nil,
		nil,
		nil,
	)
	if parseSecondErr != nil {
		parseT.Fatalf("HandleHostWorkerRegionUpdateOrchestration(second) returned error: %v", parseSecondErr)
	}
	if !parseSecondResult.GetDispatchResult.GetDispatchResult.HasNoChange {
		parseT.Fatalf("expected repeated orchestration input to short-circuit as no-change, got %+v", parseSecondResult)
	}
	if parseSecondResult.GetWorkerUpdateResult.HasPatchPayload || parseSecondResult.HasPatchCommitted {
		parseT.Fatalf("expected no-change orchestration to return before worker patch handling, got %+v", parseSecondResult)
	}
}

// TestHandleHostWorkerRegionUpdateOrchestrationPropagatesWorkerErrors verifies orchestration surfaces worker update failures after host snapshot dispatch succeeds.
func TestHandleHostWorkerRegionUpdateOrchestrationPropagatesWorkerErrors(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	buildWorkerRegionRuntime := BuildWorkerRegionRuntime()
	if _, parseErr := HandleHostWorkerRegionUpdateOrchestration(
		buildHostRegionAdapter,
		buildWorkerRegionRuntime,
		ParallelRegionSpec{
			RendererID:       RendererID("dashboard.hot-panel"),
			RegionInstanceID: RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "missing-renderer"},
		},
		2,
		buildHostWorkerOrchestrationCapabilityReport(),
		nil,
		nil,
		nil,
	); parseErr == nil {
		parseT.Fatal("expected missing worker renderer to fail")
	}
}

// TestParseDecodeSnapshotForOrchestrationCoversTransportBranches verifies orchestration snapshot decode handles structured-clone, binary, shared-buffer, and unsupported transport branches.
func TestParseDecodeSnapshotForOrchestrationCoversTransportBranches(parseT *testing.T) {
	parseEnvelope, parseEnvelopeErr := BuildSnapshotEnvelope(
		"region-1",
		2,
		3,
		map[string]any{"title": "decode"},
		nil,
		nil,
		nil,
	)
	if parseEnvelopeErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseEnvelopeErr)
	}

	parseStructuredClonePayload, parseStructuredCloneErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope)
	if parseStructuredCloneErr != nil {
		parseT.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseStructuredCloneErr)
	}
	parseStructuredCloneEnvelope, parseStructuredCloneDecodeErr := parseDecodeSnapshotForOrchestration(
		SharedSnapshotTransportResult{
			GetTransportTier:  TransportTierStructuredClone,
			GetMessagePayload: parseStructuredClonePayload,
		},
		nil,
	)
	if parseStructuredCloneDecodeErr != nil {
		parseT.Fatalf("parseDecodeSnapshotForOrchestration(structured-clone) returned error: %v", parseStructuredCloneDecodeErr)
	}
	if parseStructuredCloneEnvelope.RegionInstanceID != parseEnvelope.RegionInstanceID {
		parseT.Fatalf("structured-clone decode region = %q, want %q", parseStructuredCloneEnvelope.RegionInstanceID, parseEnvelope.RegionInstanceID)
	}

	parseBinaryPayload, parseBinaryErr := BuildBinarySnapshotEnvelope(parseEnvelope)
	if parseBinaryErr != nil {
		parseT.Fatalf("BuildBinarySnapshotEnvelope returned error: %v", parseBinaryErr)
	}
	parseBinaryEnvelope, parseBinaryDecodeErr := parseDecodeSnapshotForOrchestration(
		SharedSnapshotTransportResult{
			GetTransportTier:  TransportTierBinary,
			GetMessagePayload: parseBinaryPayload,
		},
		nil,
	)
	if parseBinaryDecodeErr != nil {
		parseT.Fatalf("parseDecodeSnapshotForOrchestration(binary) returned error: %v", parseBinaryDecodeErr)
	}
	if parseBinaryEnvelope.InputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("binary decode input version = %d, want %d", parseBinaryEnvelope.InputVersion, parseEnvelope.InputVersion)
	}

	if _, parseSharedErr := parseDecodeSnapshotForOrchestration(
		SharedSnapshotTransportResult{GetTransportTier: TransportTierSharedBuffer},
		nil,
	); parseSharedErr == nil {
		parseT.Fatal("expected shared-buffer decode without page to fail")
	}

	parseSharedSnapshotPage, parseSharedPageErr := BuildSharedSnapshotPage(1024)
	if parseSharedPageErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseSharedPageErr)
	}
	if _, parsePublishErr := parseSharedSnapshotPage.HandleSharedSnapshotPublishPayload(parseStructuredClonePayload); parsePublishErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishPayload returned error: %v", parsePublishErr)
	}
	parseSharedEnvelope, parseSharedDecodeErr := parseDecodeSnapshotForOrchestration(
		SharedSnapshotTransportResult{GetTransportTier: TransportTierSharedBuffer},
		parseSharedSnapshotPage,
	)
	if parseSharedDecodeErr != nil {
		parseT.Fatalf("parseDecodeSnapshotForOrchestration(shared-buffer) returned error: %v", parseSharedDecodeErr)
	}
	if parseSharedEnvelope.Epoch != parseEnvelope.Epoch {
		parseT.Fatalf("shared-buffer decode epoch = %d, want %d", parseSharedEnvelope.Epoch, parseEnvelope.Epoch)
	}

	if _, parseUnsupportedErr := parseDecodeSnapshotForOrchestration(
		SharedSnapshotTransportResult{GetTransportTier: TransportTier("unsupported")},
		nil,
	); parseUnsupportedErr == nil {
		parseT.Fatal("expected unsupported transport tier to fail")
	}
}
