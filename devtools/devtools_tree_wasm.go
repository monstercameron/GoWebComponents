//go:build js && wasm && !production

package devtools

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

func selectedNodeSummary(parseSnapshot Snapshot, parseSelectedPath string) ui.Node {
	if parseSnapshot.Tree == nil {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No committed tree yet."))
	}
	parseNode := findNodeByPath(parseSnapshot.Tree, parseSelectedPath)
	if parseNode == nil {
		parseNode = parseSnapshot.Tree
	}
	parseRows := []ui.Node{
		metricRow("Node", emptyFallback(parseNode.Name, "unknown")),
		metricRow("Path", emptyFallback(parseNode.Path, "unknown")),
		metricRow("Kind", emptyFallback(parseNode.Kind, "unknown")),
		metricRow("Hooks", fmt.Sprintf("%d", parseNode.HookCount)),
		metricRow("Effects", fmt.Sprintf("%d", parseNode.EffectCount)),
		metricRow("Route", emptyFallback(parseSnapshot.Route.Path, "/")),
	}
	if len(parseSnapshot.Route.Stack) > 0 {
		parseRows = append(parseRows, metricRow("Route stack", emptyFallback(parseSnapshot.Route.Stack[len(parseSnapshot.Route.Stack)-1].Path, parseSnapshot.Route.Path)))
	}
	if strings.TrimSpace(parseNode.Signature) != "" {
		parseRows = append(parseRows, metricRow("Signature", parseNode.Signature))
	}
	if strings.TrimSpace(parseNode.UpdateOrigin) != "" || strings.TrimSpace(parseNode.ReactiveSource) != "" {
		parseMeta := []string{}
		if strings.TrimSpace(parseNode.UpdateOrigin) != "" {
			parseMeta = append(parseMeta, "origin="+parseNode.UpdateOrigin)
		}
		if strings.TrimSpace(parseNode.ReactiveSource) != "" {
			parseMeta = append(parseMeta, "source="+parseNode.ReactiveSource)
		}
		parseRows = append(parseRows, metricRow("Reactive", strings.Join(parseMeta, " | ")))
	}
	if len(parseNode.Hooks) > 0 {
		parseHooks := make([]string, 0, len(parseNode.Hooks))
		for _, parseHook := range parseNode.Hooks {
			parseDetail := parseHook.Kind + "#" + fmt.Sprintf("%d", parseHook.Slot) + "=" + parseHook.Value
			if strings.TrimSpace(parseHook.Status) != "" {
				parseDetail += " (" + parseHook.Status + ")"
			}
			parseHooks = append(parseHooks, parseDetail)
		}
		parseRows = append(parseRows, metricRow("State", strings.Join(parseHooks, " | ")))
	}
	if parseMatches := cacheEntriesForNode(parseNode, parseSnapshot.Cache); len(parseMatches) > 0 {
		parseSummaries := make([]string, 0, len(parseMatches))
		for _, parseEntry := range parseMatches {
			parseSummaries = append(parseSummaries, fmt.Sprintf("%s ready=%t stale=%t subscribers=%d", parseEntry.Key, parseEntry.Ready, parseEntry.Stale, parseEntry.SubscriberCount))
		}
		parseRows = append(parseRows, metricRow("Cache", strings.Join(parseSummaries, "; ")))
	}
	return html.Div(html.Props{}, parseRows...)
}

func treeSummary(parseNode *Node, parseDepth int, parseMaxDepth int, parseSelectedPath string, parseSelectNode func(string)) ui.Node {
	if parseNode == nil {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No committed tree yet."))
	}
	return renderNode(*parseNode, parseDepth, parseMaxDepth, parseSelectedPath, parseSelectNode)
}

func renderNode(parseNode Node, parseDepth int, parseMaxDepth int, parseSelectedPath string, parseSelectNode func(string)) ui.Node {
	parseChildren := []ui.Node{
		html.Div(html.Props{OnClick: ui.WrapHandler(func() {
			if parseSelectNode != nil {
				parseSelectNode(parseNode.Path)
			}
		}), Style: map[string]string{
			"display":       "flex",
			"flex-wrap":     "wrap",
			"gap":           "8px",
			"align-items":   "center",
			"cursor":        "pointer",
			"padding":       "4px 6px",
			"border-radius": "8px",
			"background": func() string {
				if strings.TrimSpace(parseNode.Path) == strings.TrimSpace(parseSelectedPath) {
					return "rgba(14,116,144,0.22)"
				}
				return "transparent"
			}(),
		}},
			html.Strong(html.Props{Style: map[string]string{"color": "#f8fafc"}}, html.Text(parseNode.Name)),
			html.Code(html.Props{Style: map[string]string{"color": "#67e8f9", "background": "rgba(15,23,42,0.6)", "padding": "2px 6px", "border-radius": "6px"}}, html.Text(parseNode.Kind)),
			html.Small(html.Props{Style: map[string]string{"color": "#94a3b8"}}, html.Text(fmt.Sprintf("hooks=%d effects=%d", parseNode.HookCount, parseNode.EffectCount))),
			func() ui.Node {
				if !parseNode.FineGrained {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"color": "#67e8f9"}}, html.Text("fine-grained"))
			}(),
			func() ui.Node {
				if parseNode.Dirty || parseNode.NeedsUpdate {
					return html.Small(html.Props{Style: map[string]string{"color": "#fde68a"}}, html.Text(fmt.Sprintf("dirty=%t update=%t", parseNode.Dirty, parseNode.NeedsUpdate)))
				}
				return nil
			}(),
			func() ui.Node {
				if parseNode.SubtreeDurationNs <= 0 {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"color": "#cbd5e1"}}, html.Text("subtree="+formatDurationNs(parseNode.SubtreeDurationNs)+" self="+formatDurationNs(parseNode.SelfDurationNs)+" render="+formatDurationNs(parseNode.RenderDurationNs)+" diff="+formatDurationNs(parseNode.DiffDurationNs)+" commit="+formatDurationNs(parseNode.CommitDurationNs)))
			}(),
		),
	}

	if strings.TrimSpace(parseNode.Signature) != "" {
		parseChildren = append(parseChildren, html.Small(html.Props{Style: map[string]string{
			"display":    "block",
			"margin-top": "4px",
			"color":      "#cbd5e1",
		}}, html.Text("signature: "+parseNode.Signature)))
	}

	if parseNode.FineGrained || strings.TrimSpace(parseNode.UpdateOrigin) != "" || strings.TrimSpace(parseNode.ReactiveSource) != "" {
		parseMeta := make([]string, 0, 3)
		if parseNode.FineGrained {
			parseMeta = append(parseMeta, "mode=fine-grained")
		}
		if strings.TrimSpace(parseNode.UpdateOrigin) != "" {
			parseMeta = append(parseMeta, "origin="+parseNode.UpdateOrigin)
		}
		if strings.TrimSpace(parseNode.ReactiveSource) != "" {
			parseMeta = append(parseMeta, "source="+parseNode.ReactiveSource)
		}
		parseChildren = append(parseChildren, html.Small(html.Props{Style: map[string]string{
			"display":    "block",
			"margin-top": "4px",
			"color":      "#67e8f9",
		}}, html.Text(strings.Join(parseMeta, " | "))))
	}

	if len(parseNode.Hooks) > 0 {
		parseHookNodes := make([]ui.Node, 0, len(parseNode.Hooks))
		for _, parseHook := range parseNode.Hooks {
			parseDetail := "#" + fmt.Sprintf("%d", parseHook.Slot) + ": " + parseHook.Value
			if strings.TrimSpace(parseHook.Dependencies) != "" {
				parseDetail += " | deps=" + parseHook.Dependencies
			}
			if strings.TrimSpace(parseHook.Status) != "" {
				parseDetail += " | " + parseHook.Status
			}
			parseHookNodes = append(parseHookNodes, html.Div(html.Props{Style: map[string]string{"margin-top": "6px", "color": "#cbd5e1"}},
				html.Code(html.Props{Style: map[string]string{"color": "#67e8f9"}}, html.Text(parseHook.Kind)),
				html.Text(": "+parseDetail),
			))
		}
		parseChildren = append(parseChildren, html.Div(html.Props{Style: map[string]string{"margin-top": "6px"}}, parseHookNodes...))
	}

	if parseDepth < parseMaxDepth && len(parseNode.Children) > 0 {
		parseChildNodes := make([]ui.Node, 0, len(parseNode.Children))
		for _, parseChild := range parseNode.Children {
			parseChildNodes = append(parseChildNodes, renderNode(parseChild, parseDepth+1, parseMaxDepth, parseSelectedPath, parseSelectNode))
		}
		parseChildren = append(parseChildren, html.Div(html.Props{Style: map[string]string{
			"margin-top":   "8px",
			"margin-left":  "14px",
			"padding-left": "10px",
			"border-left":  "1px solid rgba(51,65,85,0.7)",
		}}, parseChildNodes...))
	} else if parseDepth >= parseMaxDepth && len(parseNode.Children) > 0 {
		parseChildren = append(parseChildren, html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "8px", "color": "#94a3b8"}}, html.Text(fmt.Sprintf("%d child nodes hidden at max depth", len(parseNode.Children)))))
	}

	return html.Div(html.Props{Style: map[string]string{
		"padding":       "8px 0",
		"border-bottom": "1px solid rgba(30,41,59,0.8)",
	}}, parseChildren...)
}

func findNodeByPath(parseNode *Node, parsePath string) *Node {
	if parseNode == nil {
		return nil
	}
	if strings.TrimSpace(parseNode.Path) == strings.TrimSpace(parsePath) {
		return parseNode
	}
	for parseIndex := range parseNode.Children {
		if parseFound := findNodeByPath(&parseNode.Children[parseIndex], parsePath); parseFound != nil {
			return parseFound
		}
	}
	return nil
}

func cacheEntriesForNode(parseNode *Node, parseEntries []CacheEntry) []CacheEntry {
	if parseNode == nil || len(parseEntries) == 0 {
		return nil
	}
	parseMatches := make([]CacheEntry, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		for _, parseOwnerPath := range parseEntry.OwnerPaths {
			if strings.TrimSpace(parseOwnerPath) == strings.TrimSpace(parseNode.Path) {
				parseMatches = append(parseMatches, parseEntry)
				break
			}
		}
	}
	return parseMatches
}

func metricRow(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Style: map[string]string{
		"display":         "flex",
		"justify-content": "space-between",
		"gap":             "10px",
		"margin-bottom":   "6px",
	}},
		html.Small(html.Props{Style: map[string]string{"color": "#94a3b8"}}, html.Text(parseLabel)),
		html.Code(html.Props{Style: map[string]string{"color": "#f8fafc", "text-align": "right"}}, html.Text(parseValue)),
	)
}

func mapNode(parseNode *runtime.FiberSnapshot) *Node {
	if parseNode == nil {
		return nil
	}
	parseMapped := &Node{
		Name:              parseNode.Name,
		Path:              parseNode.Path,
		Kind:              parseNode.Kind,
		Dirty:             parseNode.Dirty,
		NeedsUpdate:       parseNode.NeedsUpdate,
		FineGrained:       parseNode.FineGrained,
		ReactiveSource:    parseNode.ReactiveSource,
		UpdateOrigin:      parseNode.UpdateOrigin,
		EffectCount:       parseNode.EffectCount,
		HookCount:         parseNode.HookCount,
		Signature:         "",
		RenderDurationNs:  parseNode.RenderDurationNs,
		DiffDurationNs:    parseNode.DiffDurationNs,
		CommitDurationNs:  parseNode.CommitDurationNs,
		EffectDurationNs:  parseNode.EffectDurationNs,
		CleanupDurationNs: parseNode.CleanupDurationNs,
		SelfDurationNs:    parseNode.SelfDurationNs,
		SubtreeDurationNs: parseNode.SubtreeDurationNs,
	}
	if parseNode.Signature != nil {
		parseMapped.Signature = parseNode.Signature.Summary()
	}
	for _, parseHook := range parseNode.Hooks {
		parseMapped.Hooks = append(parseMapped.Hooks, Hook{
			Slot:         parseHook.Slot,
			Kind:         parseHook.Kind,
			Value:        parseHook.Value,
			Dependencies: parseHook.Dependencies,
			Status:       parseHook.Status,
		})
	}
	for parseIndex := range parseNode.Children {
		parseMapped.Children = append(parseMapped.Children, *mapNode(&parseNode.Children[parseIndex]))
	}
	return parseMapped
}

func mapStats(parseStats runtime.InspectionStats) Stats {
	return Stats{
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

func mapProfiling(parseProfiling runtime.ProfilingSnapshot) Profiling {
	parseMapped := Profiling{
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
		PhaseTotals: ProfilingPhaseTotals{
			RenderDurationNs:  parseProfiling.PhaseTotals.RenderDurationNs,
			DiffDurationNs:    parseProfiling.PhaseTotals.DiffDurationNs,
			CommitDurationNs:  parseProfiling.PhaseTotals.CommitDurationNs,
			EffectDurationNs:  parseProfiling.PhaseTotals.EffectDurationNs,
			CleanupDurationNs: parseProfiling.PhaseTotals.CleanupDurationNs,
		},
		Startup: StartupProfiling{
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
	for _, parseBudget := range parseProfiling.Startup.RouteBudgets {
		parseMapped.Startup.RouteBudgets = append(parseMapped.Startup.RouteBudgets, RouteStartupBudget{
			RouteFamily:                       parseBudget.RouteFamily,
			LastRoutePath:                     parseBudget.LastRoutePath,
			SampleCount:                       parseBudget.SampleCount,
			AverageBootstrapReadDurationNs:    parseBudget.AverageBootstrapReadDurationNs,
			AverageWASMTransferBytes:          parseBudget.AverageWASMTransferBytes,
			AverageWASMDecodedBytes:           parseBudget.AverageWASMDecodedBytes,
			AverageBootstrapDecodedBytes:      parseBudget.AverageBootstrapDecodedBytes,
			AverageCacheWarmupDurationNs:      parseBudget.AverageCacheWarmupDurationNs,
			AverageServiceWorkerOverheadNs:    parseBudget.AverageServiceWorkerOverheadNs,
			AverageInitialRouteDataBytes:      parseBudget.AverageInitialRouteDataBytes,
			AverageHydrationDurationNs:        parseBudget.AverageHydrationDurationNs,
			AverageStartupCommitDurationNs:    parseBudget.AverageStartupCommitDurationNs,
			AverageFirstInteractionDurationNs: parseBudget.AverageFirstInteractionDurationNs,
		})
	}
	for _, parseEvent := range parseProfiling.RecentEvents {
		parseMapped.RecentEvents = append(parseMapped.RecentEvents, ProfilingEvent{
			Domain:        parseEvent.Domain,
			Name:          parseEvent.Name,
			Phase:         parseEvent.Phase,
			Target:        parseEvent.Target,
			CorrelationID: parseEvent.CorrelationID,
			DurationNs:    parseEvent.DurationNs,
			Timestamp:     parseEvent.Timestamp,
			Fields:        cloneStringMap(parseEvent.Fields),
		})
	}
	for _, parseTrace := range parseProfiling.ComponentRenders {
		parseMapped.ComponentRenders = append(parseMapped.ComponentRenders, ComponentRenderTrace{
			Name:                    parseTrace.Name,
			Path:                    parseTrace.Path,
			RenderCount:             parseTrace.RenderCount,
			RerenderCount:           parseTrace.RerenderCount,
			LastTrigger:             parseTrace.LastTrigger,
			LastRenderDurationNs:    parseTrace.LastRenderDurationNs,
			TotalRenderDurationNs:   parseTrace.TotalRenderDurationNs,
			AverageRenderDurationNs: parseTrace.AverageRenderDurationNs,
			LastRenderedAt:          parseTrace.LastRenderedAt,
			TriggerCounts:           cloneIntMap(parseTrace.TriggerCounts),
		})
	}
	for _, parseFrame := range parseProfiling.FlamegraphFrames {
		parseMapped.FlamegraphFrames = append(parseMapped.FlamegraphFrames, FlamegraphFrame{
			Name:              parseFrame.Name,
			Kind:              parseFrame.Kind,
			Path:              parseFrame.Path,
			Depth:             parseFrame.Depth,
			StartNs:           parseFrame.StartNs,
			DurationNs:        parseFrame.DurationNs,
			SelfDurationNs:    parseFrame.SelfDurationNs,
			RenderDurationNs:  parseFrame.RenderDurationNs,
			DiffDurationNs:    parseFrame.DiffDurationNs,
			CommitDurationNs:  parseFrame.CommitDurationNs,
			EffectDurationNs:  parseFrame.EffectDurationNs,
			CleanupDurationNs: parseFrame.CleanupDurationNs,
		})
	}
	for _, parseBranch := range parseProfiling.HotBranches {
		parseMapped.HotBranches = append(parseMapped.HotBranches, Branch{
			Name:              parseBranch.Name,
			Kind:              parseBranch.Kind,
			Path:              parseBranch.Path,
			RenderDurationNs:  parseBranch.RenderDurationNs,
			DiffDurationNs:    parseBranch.DiffDurationNs,
			CommitDurationNs:  parseBranch.CommitDurationNs,
			EffectDurationNs:  parseBranch.EffectDurationNs,
			CleanupDurationNs: parseBranch.CleanupDurationNs,
			SelfDurationNs:    parseBranch.SelfDurationNs,
			SubtreeDurationNs: parseBranch.SubtreeDurationNs,
		})
	}
	return parseMapped
}

func mapHydration(parseHydration runtime.HydrationDebugSnapshot) HydrationDebug {
	return HydrationDebug{
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

func cloneStringMap(parseInput map[string]string) map[string]string {
	if len(parseInput) == 0 {
		return nil
	}
	parseOut := make(map[string]string, len(parseInput))
	for parseKey, parseValue := range parseInput {
		parseOut[parseKey] = parseValue
	}
	return parseOut
}

func cloneIntMap(parseInput map[string]int) map[string]int {
	if len(parseInput) == 0 {
		return nil
	}
	parseOut := make(map[string]int, len(parseInput))
	for parseKey, parseValue := range parseInput {
		parseOut[parseKey] = parseValue
	}
	return parseOut
}

func cloneParams(parseParams map[string]string) map[string]string {
	if len(parseParams) == 0 {
		return nil
	}
	parseClone := make(map[string]string, len(parseParams))
	for parseKey, parseValue := range parseParams {
		parseClone[parseKey] = parseValue
	}
	return parseClone
}

func mapRouteStack(parseEntries []router.RouteStackInspection) []RouteStack {
	if len(parseEntries) == 0 {
		return nil
	}
	parseStack := make([]RouteStack, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseStack = append(parseStack, RouteStack{
			ID:             parseEntry.ID,
			Path:           parseEntry.Path,
			Params:         cloneParams(parseEntry.Params),
			HasLoader:      parseEntry.HasLoader,
			HasBeforeEnter: parseEntry.HasBeforeEnter,
			HasBeforeLeave: parseEntry.HasBeforeLeave,
			Metadata: RouteMetadata{
				Title:        parseEntry.Metadata.Title,
				Description:  parseEntry.Metadata.Description,
				CanonicalURL: parseEntry.Metadata.CanonicalURL,
			},
		})
	}
	return parseStack
}

func mapRouteLoaders(parseEntries []router.RouteLoaderInspection) []RouteLoader {
	if len(parseEntries) == 0 {
		return nil
	}
	parseLoaders := make([]RouteLoader, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseLoaders = append(parseLoaders, RouteLoader{
			Key:     parseEntry.Key,
			Path:    parseEntry.Path,
			Pending: parseEntry.Pending,
			HasData: parseEntry.HasData,
			Error:   parseEntry.Error,
		})
	}
	return parseLoaders
}

func formatQuery(parseQuery map[string][]string) string {
	if len(parseQuery) == 0 {
		return "none"
	}
	parseKeys := make([]string, 0, len(parseQuery))
	for parseKey := range parseQuery {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseParts := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseParts = append(parseParts, parseKey2+"="+strings.Join(parseQuery[parseKey2], ","))
	}
	return strings.Join(parseParts, " & ")
}

func formatParams(parseParams map[string]string) string {
	if len(parseParams) == 0 {
		return "none"
	}
	parseKeys := make([]string, 0, len(parseParams))
	for parseKey := range parseParams {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseParts := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseParts = append(parseParts, parseKey2+"="+parseParams[parseKey2])
	}
	return strings.Join(parseParts, " & ")
}

func formatStringMap(parseValues map[string]string) string {
	if len(parseValues) == 0 {
		return "none"
	}
	parseKeys := make([]string, 0, len(parseValues))
	for parseKey := range parseValues {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseParts := make([]string, 0, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		parseParts = append(parseParts, parseKey2+"="+parseValues[parseKey2])
	}
	return strings.Join(parseParts, " & ")
}

func emptyFallback(parseValue, parseFallback string) string {
	if strings.TrimSpace(parseValue) == "" {
		return parseFallback
	}
	return parseValue
}

func formatDurationNs(parseValue int64) string {
	if parseValue <= 0 {
		return "0ms"
	}
	return fmt.Sprintf("%.2fms", float64(parseValue)/1_000_000)
}

func formatByteCount(parseValue int64) string {
	if parseValue <= 0 {
		return "0 B"
	}
	if parseValue < 1024 {
		return fmt.Sprintf("%d B", parseValue)
	}
	if parseValue < 1024*1024 {
		return fmt.Sprintf("%.2f KiB", float64(parseValue)/1024.0)
	}
	return fmt.Sprintf("%.2f MiB", float64(parseValue)/(1024.0*1024.0))
}

func formatTime(parseValue time.Time) string {
	if parseValue.IsZero() {
		return "n/a"
	}
	return parseValue.Format("15:04:05")
}
