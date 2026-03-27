package runtime2

import "fmt"

// HostWorkerRegionUpdateOrchestrationResult reports one end-to-end host-worker region update orchestration attempt.
type HostWorkerRegionUpdateOrchestrationResult struct {
	GetDispatchResult     HostRegionUpdateDispatchTransportResult
	GetWorkerUpdateResult WorkerPatchTransportResult
	HasPatchCommitted     bool
	GetPatchConsumeResult HostPatchConsumeResult
}

// HandleHostWorkerRegionUpdateOrchestration runs one end-to-end region update flow across host snapshot dispatch, worker update, patch transport, and host commit.
func HandleHostWorkerRegionUpdateOrchestration(
	parseHostRegionAdapter *HostRegionAdapter,
	parseWorkerRegionRuntime *WorkerRegionRuntime,
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseCapabilityReport CapabilityReport,
	parseSharedSnapshotPage *SharedSnapshotPage,
	parseSharedPatchPage *SharedPatchPage,
	parseDOMCommitter *DOMCommitter,
) (HostWorkerRegionUpdateOrchestrationResult, error) {
	if parseHostRegionAdapter == nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseWorkerRegionRuntime == nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, fmt.Errorf("runtime2: worker region runtime is nil")
	}
	parseDispatchResult, parseDispatchErr := parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		parseSpec,
		parseInputVersion,
		parseCapabilityReport,
		parseSharedSnapshotPage,
	)
	if parseDispatchErr != nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, parseDispatchErr
	}
	parseResult := HostWorkerRegionUpdateOrchestrationResult{
		GetDispatchResult: parseDispatchResult,
	}
	if !parseDispatchResult.GetDispatchResult.HasScheduled {
		return parseResult, nil
	}
	if !parseDispatchResult.HasSnapshotTransport {
		return HostWorkerRegionUpdateOrchestrationResult{}, fmt.Errorf("runtime2: snapshot transport result is required for scheduled dispatch")
	}
	parseSnapshotEnvelope, parseSnapshotErr := parseDecodeSnapshotForOrchestration(
		parseDispatchResult.GetSnapshotTransportResult,
		parseSharedSnapshotPage,
	)
	if parseSnapshotErr != nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, parseSnapshotErr
	}
	parseWorkerUpdateResult, parseWorkerUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		WorkerRegionUpdateSpec{
			RegionID:     string(parseSpec.RegionInstanceID),
			RendererID:   string(parseSpec.RendererID),
			Epoch:        parseSnapshotEnvelope.Epoch,
			InputVersion: parseSnapshotEnvelope.InputVersion,
			Snapshot:     parseSnapshotEnvelope,
		},
		parseCapabilityReport,
	)
	if parseWorkerUpdateErr != nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, parseWorkerUpdateErr
	}
	parseResult.GetWorkerUpdateResult = parseWorkerUpdateResult
	if !parseWorkerUpdateResult.HasPatchPayload {
		return parseResult, nil
	}
	parsePatchConsumeResult, parsePatchConsumeErr := parseHostRegionAdapter.HandleHostRegionPatchConsume(
		parseWorkerUpdateResult.GetTransportTier,
		parseWorkerUpdateResult.GetPatchPayload,
		parseSharedPatchPage,
		parseDOMCommitter,
	)
	if parsePatchConsumeErr != nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, parsePatchConsumeErr
	}
	parseResult.GetPatchConsumeResult = parsePatchConsumeResult
	parseResult.HasPatchCommitted = parsePatchConsumeResult.GetCommitResult.HasCommitted
	return parseResult, nil
}

// parseDecodeSnapshotForOrchestration decodes one dispatched snapshot transport payload into a validated snapshot envelope.
func parseDecodeSnapshotForOrchestration(
	parseTransportResult SharedSnapshotTransportResult,
	parseSharedSnapshotPage *SharedSnapshotPage,
) (SnapshotEnvelope, error) {
	switch parseTransportResult.GetTransportTier {
	case TransportTierStructuredClone:
		return ParseStructuredCloneSnapshotEnvelopeJSON(parseTransportResult.GetMessagePayload)
	case TransportTierBinary:
		return ParseBinarySnapshotEnvelope(parseTransportResult.GetMessagePayload)
	case TransportTierSharedBuffer:
		if parseSharedSnapshotPage == nil {
			return SnapshotEnvelope{}, fmt.Errorf("runtime2: shared snapshot page is required")
		}
		return parseSharedSnapshotPage.ParseSharedSnapshotEnvelope()
	default:
		return SnapshotEnvelope{}, fmt.Errorf(
			"runtime2: transport tier %q is unsupported for snapshot orchestration decode",
			parseTransportResult.GetTransportTier,
		)
	}
}
