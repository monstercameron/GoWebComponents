package runtime

import (
	"maps"

	"github.com/monstercameron/GoWebComponents/v6/internal/pluginruntime"
)

func init() {
	_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
		Key:   pluginruntime.ServiceKeyRuntime,
		Value: BuildRuntimeInspectionService(),
	})
	_ = pluginruntime.RegisterBuiltinService(pluginruntime.ServiceRegistration{
		Key:   pluginruntime.ServiceKeyDiagnostics,
		Value: BuildDiagnosticsService(),
	})
}

type buildRuntimeInspectionService struct{}

type buildDiagnosticsService struct{}

// BuildRuntimeInspectionService returns one runtime1-backed inspection service.
func BuildRuntimeInspectionService() pluginruntime.RuntimeInspectionService {
	return buildRuntimeInspectionService{}
}

// BuildDiagnosticsService returns one runtime1-backed diagnostics facade.
func BuildDiagnosticsService() pluginruntime.DiagnosticsService {
	return buildDiagnosticsService{}
}

// GetRuntimeSnapshot returns one normalized runtime inspection snapshot.
func (buildRuntimeInspectionService) GetRuntimeSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.RuntimeSnapshot, error) {
	getSnapshot := GetGlobalRuntime().Inspect()
	return pluginruntime.RuntimeSnapshot{
		Meta:        pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, false),
		Root:        mapRuntimeFiberSnapshot(getSnapshot.Root),
		Stats:       mapRuntimeStats(getSnapshot.Stats),
		Profiling:   mapRuntimeProfiling(getSnapshot.Profiling),
		Hydration:   mapRuntimeHydration(getSnapshot.Hydration),
		Diagnostics: mapRuntimeDiagnostics(getSnapshot.Diagnostics, parseBudget),
		Logs:        mapRuntimeLogs(getSnapshot.Logs, parseBudget),
	}, nil
}

// GetDiagnosticsSnapshot returns one aggregated diagnostics snapshot.
func (buildDiagnosticsService) GetDiagnosticsSnapshot(parseBudget pluginruntime.QueryBudget) (pluginruntime.DiagnosticsSnapshot, error) {
	getRuntime := BuildRuntimeInspectionService()
	getRuntimeSnapshot, parseErr := getRuntime.GetRuntimeSnapshot(parseBudget)
	if parseErr != nil {
		return pluginruntime.DiagnosticsSnapshot{}, parseErr
	}
	buildSnapshot := pluginruntime.DiagnosticsSnapshot{
		Meta:             pluginruntime.BuildSnapshotMeta(pluginruntime.BackendIDRuntime1, false),
		Runtime:          append([]pluginruntime.RuntimeDiagnostic(nil), getRuntimeSnapshot.Diagnostics...),
		KernelAPIVersion: "",
	}
	if getKernel := pluginruntime.GetGlobalKernel(); getKernel != nil {
		buildSnapshot.PluginReports = getKernel.HealthReports()
		buildSnapshot.PluginEvents = getKernel.Diagnostics()
		buildSnapshot.KernelAPIVersion = getKernel.Info().APIVersion
	}
	return buildSnapshot, nil
}

// mapRuntimeFiberSnapshot normalizes one runtime fiber snapshot.
func mapRuntimeFiberSnapshot(parseSnapshot *FiberSnapshot) *pluginruntime.RuntimeNode {
	if parseSnapshot == nil {
		return nil
	}
	buildNode := &pluginruntime.RuntimeNode{
		Name:              parseSnapshot.Name,
		Path:              parseSnapshot.Path,
		Kind:              parseSnapshot.Kind,
		Dirty:             parseSnapshot.Dirty,
		NeedsUpdate:       parseSnapshot.NeedsUpdate,
		FineGrained:       parseSnapshot.FineGrained,
		ReactiveSource:    parseSnapshot.ReactiveSource,
		UpdateOrigin:      parseSnapshot.UpdateOrigin,
		EffectCount:       parseSnapshot.EffectCount,
		HookCount:         parseSnapshot.HookCount,
		RenderDurationNs:  parseSnapshot.RenderDurationNs,
		DiffDurationNs:    parseSnapshot.DiffDurationNs,
		CommitDurationNs:  parseSnapshot.CommitDurationNs,
		EffectDurationNs:  parseSnapshot.EffectDurationNs,
		CleanupDurationNs: parseSnapshot.CleanupDurationNs,
		SelfDurationNs:    parseSnapshot.SelfDurationNs,
		SubtreeDurationNs: parseSnapshot.SubtreeDurationNs,
	}
	if parseSnapshot.Signature != nil {
		buildNode.Signature = parseSnapshot.Signature.Summary()
	}
	for _, getHook := range parseSnapshot.Hooks {
		buildNode.Hooks = append(buildNode.Hooks, pluginruntime.RuntimeHook{
			Slot:         getHook.Slot,
			Kind:         getHook.Kind,
			Value:        getHook.Value,
			Dependencies: getHook.Dependencies,
			Status:       getHook.Status,
		})
	}
	for _, getChild := range parseSnapshot.Children {
		getChildSnapshot := getChild
		buildMapped := mapRuntimeFiberSnapshot(&getChildSnapshot)
		if buildMapped != nil {
			buildNode.Children = append(buildNode.Children, *buildMapped)
		}
	}
	return buildNode
}

// mapRuntimeStats normalizes runtime stats.
func mapRuntimeStats(parseStats InspectionStats) pluginruntime.RuntimeStats {
	return pluginruntime.RuntimeStats{
		TotalFibers:       parseStats.TotalFibers,
		DirtyFibers:       parseStats.DirtyFibers,
		ComponentFibers:   parseStats.ComponentFibers,
		HostFibers:        parseStats.HostFibers,
		TextFibers:        parseStats.TextFibers,
		FineGrainedFibers: parseStats.FineGrainedFibers,
		HookEntries:       parseStats.HookEntries,
		Effects:           parseStats.Effects,
	}
}

// mapRuntimeProfiling normalizes runtime profiling.
func mapRuntimeProfiling(parseProfiling ProfilingSnapshot) pluginruntime.RuntimeProfiling {
	buildProfiling := pluginruntime.RuntimeProfiling{
		RenderCalls:                      parseProfiling.RenderCalls,
		ScheduledRootUpdates:             parseProfiling.ScheduledRootUpdates,
		ScheduledFiberMarks:              parseProfiling.ScheduledFiberMarks,
		ScheduledGranularMarks:           parseProfiling.ScheduledGranularMarks,
		WorkLoopPasses:                   parseProfiling.WorkLoopPasses,
		ProcessedUnits:                   parseProfiling.ProcessedUnits,
		CommitCount:                      parseProfiling.CommitCount,
		FineGrainedCommits:               parseProfiling.FineGrainedCommits,
		FineGrainedDescendantHostCommits: parseProfiling.FineGrainedDescendantHostCommits,
		FineGrainedDescendantTextCommits: parseProfiling.FineGrainedDescendantTextCommits,
		EffectExecutions:                 parseProfiling.EffectExecutions,
		CleanupExecutions:                parseProfiling.CleanupExecutions,
		LastRenderDurationNs:             parseProfiling.LastRenderDurationNs,
		LastCommitDurationNs:             parseProfiling.LastCommitDurationNs,
		LastEffectDurationNs:             parseProfiling.LastEffectDurationNs,
		LastCleanupDurationNs:            parseProfiling.LastCleanupDurationNs,
		PhaseTotals: pluginruntime.RuntimeProfilingPhaseTotals{
			RenderDurationNs:  parseProfiling.PhaseTotals.RenderDurationNs,
			DiffDurationNs:    parseProfiling.PhaseTotals.DiffDurationNs,
			CommitDurationNs:  parseProfiling.PhaseTotals.CommitDurationNs,
			EffectDurationNs:  parseProfiling.PhaseTotals.EffectDurationNs,
			CleanupDurationNs: parseProfiling.PhaseTotals.CleanupDurationNs,
		},
		Startup: pluginruntime.RuntimeStartupProfiling{
			Mode:                       parseProfiling.Startup.Mode,
			StartedAt:                  parseProfiling.Startup.StartedAt,
			BootstrapReadDurationNs:    parseProfiling.Startup.BootstrapReadDurationNs,
			WASMTransferBytes:          parseProfiling.Startup.WASMTransferBytes,
			WASMDecodedBytes:           parseProfiling.Startup.WASMDecodedBytes,
			BootstrapDecodedBytes:      parseProfiling.Startup.BootstrapDecodedBytes,
			CacheWarmupDurationNs:      parseProfiling.Startup.CacheWarmupDurationNs,
			ServiceWorkerOverheadNs:    parseProfiling.Startup.ServiceWorkerOverheadNs,
			InitialRouteDataBytes:      parseProfiling.Startup.InitialRouteDataBytes,
			HydrationDurationNs:        parseProfiling.Startup.HydrationDurationNs,
			StartupCommitDurationNs:    parseProfiling.Startup.StartupCommitDurationNs,
			FirstInteractionDurationNs: parseProfiling.Startup.FirstInteractionDurationNs,
			FirstInteractionCaptured:   parseProfiling.Startup.FirstInteractionCaptured,
			FirstInteractionEvent:      parseProfiling.Startup.FirstInteractionEvent,
		},
	}
	for _, getTrace := range parseProfiling.ComponentRenders {
		buildProfiling.ComponentRenders = append(buildProfiling.ComponentRenders, pluginruntime.RuntimeComponentRenderTrace{
			Name:                    getTrace.Name,
			Path:                    getTrace.Path,
			RenderCount:             getTrace.RenderCount,
			RerenderCount:           getTrace.RerenderCount,
			LastTrigger:             getTrace.LastTrigger,
			LastRenderDurationNs:    getTrace.LastRenderDurationNs,
			TotalRenderDurationNs:   getTrace.TotalRenderDurationNs,
			AverageRenderDurationNs: getTrace.AverageRenderDurationNs,
			LastRenderedAt:          getTrace.LastRenderedAt,
			TriggerCounts:           cloneRuntimeIntMap(getTrace.TriggerCounts),
		})
	}
	for _, getEvent := range parseProfiling.RecentEvents {
		buildProfiling.RecentEvents = append(buildProfiling.RecentEvents, pluginruntime.RuntimeProfilingEvent{
			Domain:        getEvent.Domain,
			Name:          getEvent.Name,
			Phase:         getEvent.Phase,
			Target:        getEvent.Target,
			CorrelationID: getEvent.CorrelationID,
			DurationNs:    getEvent.DurationNs,
			Timestamp:     getEvent.Timestamp,
			Fields:        cloneRuntimeStringMap(getEvent.Fields),
		})
	}
	for _, getFrame := range parseProfiling.FlamegraphFrames {
		buildProfiling.FlamegraphFrames = append(buildProfiling.FlamegraphFrames, pluginruntime.RuntimeFlamegraphFrame{
			Name:              getFrame.Name,
			Kind:              getFrame.Kind,
			Path:              getFrame.Path,
			Depth:             getFrame.Depth,
			StartNs:           getFrame.StartNs,
			DurationNs:        getFrame.DurationNs,
			SelfDurationNs:    getFrame.SelfDurationNs,
			RenderDurationNs:  getFrame.RenderDurationNs,
			DiffDurationNs:    getFrame.DiffDurationNs,
			CommitDurationNs:  getFrame.CommitDurationNs,
			EffectDurationNs:  getFrame.EffectDurationNs,
			CleanupDurationNs: getFrame.CleanupDurationNs,
		})
	}
	for _, getBudget := range parseProfiling.Startup.RouteBudgets {
		buildProfiling.Startup.RouteBudgets = append(buildProfiling.Startup.RouteBudgets, pluginruntime.RuntimeRouteStartupBudget{
			RouteFamily:                       getBudget.RouteFamily,
			LastRoutePath:                     getBudget.LastRoutePath,
			SampleCount:                       getBudget.SampleCount,
			AverageBootstrapReadDurationNs:    getBudget.AverageBootstrapReadDurationNs,
			AverageWASMTransferBytes:          getBudget.AverageWASMTransferBytes,
			AverageWASMDecodedBytes:           getBudget.AverageWASMDecodedBytes,
			AverageBootstrapDecodedBytes:      getBudget.AverageBootstrapDecodedBytes,
			AverageCacheWarmupDurationNs:      getBudget.AverageCacheWarmupDurationNs,
			AverageServiceWorkerOverheadNs:    getBudget.AverageServiceWorkerOverheadNs,
			AverageInitialRouteDataBytes:      getBudget.AverageInitialRouteDataBytes,
			AverageHydrationDurationNs:        getBudget.AverageHydrationDurationNs,
			AverageStartupCommitDurationNs:    getBudget.AverageStartupCommitDurationNs,
			AverageFirstInteractionDurationNs: getBudget.AverageFirstInteractionDurationNs,
		})
	}
	for _, getBranch := range parseProfiling.HotBranches {
		buildProfiling.HotBranches = append(buildProfiling.HotBranches, pluginruntime.RuntimeBranch{
			Name:              getBranch.Name,
			Kind:              getBranch.Kind,
			Path:              getBranch.Path,
			RenderDurationNs:  getBranch.RenderDurationNs,
			DiffDurationNs:    getBranch.DiffDurationNs,
			CommitDurationNs:  getBranch.CommitDurationNs,
			EffectDurationNs:  getBranch.EffectDurationNs,
			CleanupDurationNs: getBranch.CleanupDurationNs,
			SelfDurationNs:    getBranch.SelfDurationNs,
			SubtreeDurationNs: getBranch.SubtreeDurationNs,
		})
	}
	return buildProfiling
}

// mapRuntimeHydration normalizes hydration debug data.
func mapRuntimeHydration(parseHydration HydrationDebugSnapshot) pluginruntime.RuntimeHydration {
	return pluginruntime.RuntimeHydration{
		CorrelationID:        parseHydration.CorrelationID,
		StartedAt:            parseHydration.StartedAt,
		FinishedAt:           parseHydration.FinishedAt,
		DurationNs:           parseHydration.DurationNs,
		ExistingDOMNodeCount: parseHydration.ExistingDOMNodeCount,
		FallbackCount:        parseHydration.FallbackCount,
		MismatchCount:        parseHydration.MismatchCount,
		DiscardedNodeCount:   parseHydration.DiscardedNodeCount,
		Strict:               parseHydration.Strict,
		Failed:               parseHydration.Failed,
		Failure:              parseHydration.Failure,
		RecentMessages:       append([]string(nil), parseHydration.RecentMessages...),
	}
}

// mapRuntimeDiagnostics normalizes runtime diagnostics.
func mapRuntimeDiagnostics(parseDiagnostics []Diagnostic, parseBudget pluginruntime.QueryBudget) []pluginruntime.RuntimeDiagnostic {
	buildDiagnostics := make([]pluginruntime.RuntimeDiagnostic, 0, len(parseDiagnostics))
	for _, getDiagnostic := range parseDiagnostics {
		buildDiagnostics = append(buildDiagnostics, pluginruntime.RuntimeDiagnostic{
			Source:         getDiagnostic.Source,
			Severity:       string(getDiagnostic.Severity),
			Classification: string(getDiagnostic.Classification),
			Code:           getDiagnostic.Code,
			Docs:           getDiagnostic.Docs,
			Remediation:    getDiagnostic.Remediation,
			Recoverable:    getDiagnostic.Recoverable,
			TopFrame:       getDiagnostic.TopFrame,
			Consequence:    getDiagnostic.Consequence,
			Message:        getDiagnostic.Message,
			Count:          getDiagnostic.Count,
			Path:           getDiagnostic.Path,
			ComponentStack: append([]string(nil), getDiagnostic.ComponentStack...),
			Fields:         cloneRuntimeStringMap(getDiagnostic.Fields),
		})
	}
	return applyRuntimeBudget(buildDiagnostics, parseBudget)
}

// mapRuntimeLogs normalizes runtime logs.
func mapRuntimeLogs(parseLogs []LogEntry, parseBudget pluginruntime.QueryBudget) []pluginruntime.RuntimeLog {
	buildLogs := make([]pluginruntime.RuntimeLog, 0, len(parseLogs))
	for _, getLog := range parseLogs {
		buildLogs = append(buildLogs, pluginruntime.RuntimeLog{
			Domain:         getLog.Domain,
			Level:          string(getLog.Level),
			Classification: string(getLog.Classification),
			Code:           getLog.Code,
			Docs:           getLog.Docs,
			Remediation:    getLog.Remediation,
			Recoverable:    getLog.Recoverable,
			TopFrame:       getLog.TopFrame,
			Consequence:    getLog.Consequence,
			Message:        getLog.Message,
			Timestamp:      getLog.Timestamp,
			CorrelationID:  getLog.CorrelationID,
			Fields:         cloneRuntimeStringMap(getLog.Fields),
		})
	}
	return applyRuntimeBudget(buildLogs, parseBudget)
}

// cloneRuntimeIntMap clones one string-int map.
func cloneRuntimeIntMap(parseValues map[string]int) map[string]int {
	if len(parseValues) == 0 {
		return nil
	}
	buildClone := make(map[string]int, len(parseValues))
	maps.Copy(buildClone, parseValues)
	return buildClone
}

// cloneRuntimeStringMap clones one string-string map.
func cloneRuntimeStringMap(parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return nil
	}
	buildClone := make(map[string]string, len(parseValues))
	maps.Copy(buildClone, parseValues)
	return buildClone
}

// applyRuntimeBudget truncates one slice to the requested query budget.
func applyRuntimeBudget[T any](parseValues []T, parseBudget pluginruntime.QueryBudget) []T {
	if parseBudget.MaxItems <= 0 || len(parseValues) <= parseBudget.MaxItems {
		return parseValues
	}
	return append([]T(nil), parseValues[:parseBudget.MaxItems]...)
}
