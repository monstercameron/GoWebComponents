package runtime2

import "fmt"

// HostRegionUpdateDispatchTransportResult reports one host update-dispatch result plus selected snapshot transport metadata.
type HostRegionUpdateDispatchTransportResult struct {
	GetDispatchResult          HostRegionUpdateDispatchResult
	HasSnapshotTransport       bool
	GetSnapshotTransportResult SharedSnapshotTransportResult
}

// HandleHostRegionUpdateDispatchWithTransport captures one update snapshot, dispatches host scheduling, and selects snapshot transport for scheduled updates.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateDispatchWithTransport(
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseCapabilityReport CapabilityReport,
	parseSharedSnapshotPage *SharedSnapshotPage,
) (HostRegionUpdateDispatchTransportResult, error) {
	return parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransportPriority(
		parseSpec,
		parseInputVersion,
		HostRegionDispatchPriorityUrgent,
		parseCapabilityReport,
		parseSharedSnapshotPage,
	)
}

// HandleHostRegionUpdateDispatchWithTransportPriority captures one update snapshot, dispatches with explicit priority, and selects snapshot transport for scheduled updates.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateDispatchWithTransportPriority(
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseDispatchPriority HostRegionDispatchPriority,
	parseCapabilityReport CapabilityReport,
	parseSharedSnapshotPage *SharedSnapshotPage,
) (HostRegionUpdateDispatchTransportResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionUpdateDispatchTransportResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	parseDispatchResult, parseDispatchErr := parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
		parseSpec,
		parseInputVersion,
		parseDispatchPriority,
	)
	if parseDispatchErr != nil {
		return HostRegionUpdateDispatchTransportResult{}, parseDispatchErr
	}
	parseResult := HostRegionUpdateDispatchTransportResult{
		GetDispatchResult: parseDispatchResult,
	}
	if !parseDispatchResult.HasScheduled {
		return parseResult, nil
	}
	parseTransportResult, parseTransportErr := BuildSharedSnapshotTransportResult(
		parseDispatchResult.GetSnapshotEnvelope,
		parseCapabilityReport,
		parseSharedSnapshotPage,
	)
	if parseTransportErr != nil {
		return HostRegionUpdateDispatchTransportResult{}, parseTransportErr
	}
	parseResult.HasSnapshotTransport = true
	parseResult.GetSnapshotTransportResult = parseTransportResult
	parseHostRegionAdapter.storeHostRegionSnapshotTier = parseTransportResult.GetTransportTier
	parseHostRegionAdapter.storeHostRegionTransportTier = parseTransportResult.GetTransportTier
	if parseTransportResult.GetDowngradeReason != "" {
		parseHostRegionAdapter.storeHostRegionSnapshotDowngrade = DiagnosticDowngradeReason{
			Path:   DiagnosticDowngradePathSharedMemory,
			Reason: string(parseTransportResult.GetDowngradeReason),
		}
		parseHostRegionAdapter.hasHostRegionSnapshotDowngrade = true
	}
	return parseResult, nil
}
