package ui

import (
	"sort"

	"github.com/monstercameron/GoWebComponents/internal/pluginruntime"
	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

func init() {
	runtime2.SetRuntime2MetaProvider(buildUIRuntime2MetaSnapshot)
}

// buildUIRuntime2MetaSnapshot returns one normalized runtime2 metadata snapshot from tracked parallel regions.
func buildUIRuntime2MetaSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.Runtime2MetaSnapshot, error) {
	getHostAdapters := copyUIRuntime2HostAdapters()
	buildRegions := make([]pluginruntime.Runtime2RegionSnapshot, 0, len(getHostAdapters))
	buildDiagnostics := make([]pluginruntime.Runtime2Diagnostic, 0, len(getHostAdapters))
	for _, getHostAdapter := range getHostAdapters {
		if getHostAdapter == nil {
			continue
		}
		getRuntimeStatus, hasRuntimeStatus := getHostAdapter.GetHostRegionRuntimeStatus()
		if !hasRuntimeStatus {
			continue
		}
		getTiming := getHostAdapter.GetHostRegionRoundTripTiming()
		getDiagnosticsSnapshot, hasDiagnosticsSnapshot := getHostAdapter.GetHostRegionDiagnosticsSnapshot()
		buildRegions = append(buildRegions, buildUIRuntime2RegionSnapshot(getRuntimeStatus, getTiming, getDiagnosticsSnapshot, hasDiagnosticsSnapshot))
		if hasDiagnosticsSnapshot {
			buildDiagnostics = append(buildDiagnostics, buildUIRuntime2Diagnostics(getDiagnosticsSnapshot)...)
		}
	}
	sort.SliceStable(buildRegions, func(parseI int, parseJ int) bool {
		return buildRegions[parseI].RegionInstanceID < buildRegions[parseJ].RegionInstanceID
	})
	sort.SliceStable(buildDiagnostics, func(parseI int, parseJ int) bool {
		if buildDiagnostics[parseI].RegionInstanceID != buildDiagnostics[parseJ].RegionInstanceID {
			return buildDiagnostics[parseI].RegionInstanceID < buildDiagnostics[parseJ].RegionInstanceID
		}
		if buildDiagnostics[parseI].Type != buildDiagnostics[parseJ].Type {
			return buildDiagnostics[parseI].Type < buildDiagnostics[parseJ].Type
		}
		return buildDiagnostics[parseI].Text < buildDiagnostics[parseJ].Text
	})
	getLimitedRegions, isRegionsTruncated := applyUIRuntime2Budget(buildRegions, parseBudget)
	getLimitedDiagnostics, isDiagnosticsTruncated := applyUIRuntime2Budget(buildDiagnostics, parseBudget)
	return pluginruntime.Runtime2MetaSnapshot{
		Meta:        pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime2, isRegionsTruncated || isDiagnosticsTruncated),
		Regions:     getLimitedRegions,
		Diagnostics: getLimitedDiagnostics,
	}, nil
}

// copyUIRuntime2HostAdapters returns one cloned slice of tracked runtime2 host adapters.
func copyUIRuntime2HostAdapters() []*runtime2.HostRegionAdapter {
	storeParallelRegionAdapterMu.RLock()
	defer storeParallelRegionAdapterMu.RUnlock()
	buildAdapters := make([]*runtime2.HostRegionAdapter, 0, len(cacheParallelRegionAdapterByID))
	for _, getHostAdapter := range cacheParallelRegionAdapterByID {
		if getHostAdapter == nil {
			continue
		}
		buildAdapters = append(buildAdapters, getHostAdapter)
	}
	return buildAdapters
}

// buildUIRuntime2RegionSnapshot maps one runtime2 host-region status into the plugin service contract.
func buildUIRuntime2RegionSnapshot(
	parseRuntimeStatus runtime2.HostRegionRuntimeStatus,
	parseTiming runtime2.HostRegionRoundTripTiming,
	parseDiagnosticsSnapshot runtime2.HostRegionDiagnosticsSnapshot,
	hasParseDiagnosticsSnapshot bool,
) pluginruntime.Runtime2RegionSnapshot {
	buildSnapshot := pluginruntime.Runtime2RegionSnapshot{
		RegionInstanceID:            string(parseRuntimeStatus.GetRegionInstanceID),
		RegionMode:                  string(parseRuntimeStatus.GetRegionMode),
		AssignedWorkerShard:         parseRuntimeStatus.GetAssignedWorkerShard,
		RendererID:                  string(parseRuntimeStatus.GetRendererID),
		Epoch:                       parseRuntimeStatus.GetEpoch,
		IsHydrationComplete:         parseRuntimeStatus.GetIsHydrationComplete,
		HasHydratedShellAnchor:      parseRuntimeStatus.HasHydratedShellAnchor,
		HasPostHydrationAttached:    parseRuntimeStatus.HasPostHydrationAttached,
		LastSnapshotVersion:         parseRuntimeStatus.GetLastSnapshotVersion,
		LastDispatchedVersion:       parseRuntimeStatus.GetLastDispatchedVersion,
		LastCommittedVersion:        parseRuntimeStatus.GetLastCommittedVersion,
		TransportTier:               string(parseRuntimeStatus.GetTransportTier),
		HasSnapshotDowngrade:        parseRuntimeStatus.HasSnapshotDowngrade,
		SnapshotDowngradePath:       string(parseRuntimeStatus.GetSnapshotDowngradePath),
		SnapshotDowngradeReason:     parseRuntimeStatus.GetSnapshotDowngradeReason,
		HasPatchDowngrade:           parseRuntimeStatus.HasPatchDowngrade,
		PatchDowngradePath:          string(parseRuntimeStatus.GetPatchDowngradePath),
		PatchDowngradeReason:        parseRuntimeStatus.GetPatchDowngradeReason,
		DroppedStalePatchCount:      parseRuntimeStatus.GetDroppedStalePatchCount,
		IgnoredStaleDiagnosticCount: parseRuntimeStatus.GetIgnoredStaleDiagnosticCount,
		FallbackReason:              parseRuntimeStatus.GetFallbackReason,
		DispatchToPatchReadyNs:      parseTiming.GetDispatchToPatchReadyNS,
		DispatchToCommitNs:          parseTiming.GetDispatchToCommitNS,
		PatchReadyToCommitNs:        parseTiming.GetPatchReadyToCommitNS,
	}
	if hasParseDiagnosticsSnapshot {
		buildSnapshot.RepairTriggeredRemountCount = parseDiagnosticsSnapshot.GetRepairTriggeredRemountCount
		buildSnapshot.DiagnosticCount = len(parseDiagnosticsSnapshot.GetDiagnosticEvents)
	}
	return buildSnapshot
}

// buildUIRuntime2Diagnostics maps one redacted runtime2 diagnostics snapshot into plugin records.
func buildUIRuntime2Diagnostics(parseDiagnosticsSnapshot runtime2.HostRegionDiagnosticsSnapshot) []pluginruntime.Runtime2Diagnostic {
	buildDiagnostics := make([]pluginruntime.Runtime2Diagnostic, 0, len(parseDiagnosticsSnapshot.GetDiagnosticEvents))
	for _, getDiagnostic := range parseDiagnosticsSnapshot.GetDiagnosticEvents {
		buildCurrent := pluginruntime.Runtime2Diagnostic{
			RegionInstanceID: string(parseDiagnosticsSnapshot.GetRegionInstanceID),
			Type:             getDiagnostic.DiagnosticType,
			Text:             getDiagnostic.DiagnosticText,
			TransportTier:    string(getDiagnostic.TransportTier),
			ShardID:          string(getDiagnostic.DiagnosticShardID),
		}
		if getDiagnostic.DiagnosticFallback != nil {
			buildCurrent.FallbackDomain = getDiagnostic.DiagnosticFallback.Domain
			buildCurrent.FallbackReason = getDiagnostic.DiagnosticFallback.Reason
		}
		if getDiagnostic.DiagnosticDowngrade != nil {
			buildCurrent.DowngradePath = string(getDiagnostic.DiagnosticDowngrade.Path)
			buildCurrent.DowngradeReason = getDiagnostic.DiagnosticDowngrade.Reason
		}
		if getDiagnostic.DiagnosticTiming != nil {
			buildCurrent.QueueNs = getDiagnostic.DiagnosticTiming.QueueNanos
			buildCurrent.RenderNs = getDiagnostic.DiagnosticTiming.RenderNanos
		}
		if getDiagnostic.DiagnosticSize != nil {
			buildCurrent.SnapshotBytes = getDiagnostic.DiagnosticSize.SnapshotBytes
			buildCurrent.PatchBytes = getDiagnostic.DiagnosticSize.PatchBytes
		}
		buildDiagnostics = append(buildDiagnostics, buildCurrent)
	}
	return buildDiagnostics
}

// applyUIRuntime2Budget truncates one runtime2 slice to the current query budget.
func applyUIRuntime2Budget[T any](parseValues []T, parseBudget pluginruntime.QueryBudget) ([]T, bool) {
	if parseBudget.MaxItems <= 0 || len(parseValues) <= parseBudget.MaxItems {
		return parseValues, false
	}
	return append([]T(nil), parseValues[:parseBudget.MaxItems]...), true
}
