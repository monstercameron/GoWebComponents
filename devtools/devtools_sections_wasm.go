//go:build js && wasm
// +build js,wasm

package devtools

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

func routeSummary(parseRoute Route) ui.Node {
	parseRows := []ui.Node{
		metricRow("Path", emptyFallback(parseRoute.Path, "/")),
		metricRow("Loading", fmt.Sprintf("%t", parseRoute.Loading)),
	}
	parseRows = append(parseRows, metricRow("Query", formatQuery(parseRoute.Query)))
	parseRows = append(parseRows, metricRow("Params", formatParams(parseRoute.Params)))
	if len(parseRoute.Stack) > 0 {
		parseStack := make([]string, 0, len(parseRoute.Stack))
		for _, parseEntry := range parseRoute.Stack {
			parseMeta := make([]string, 0, 3)
			if parseEntry.HasLoader {
				parseMeta = append(parseMeta, "loader")
			}
			if parseEntry.HasBeforeEnter {
				parseMeta = append(parseMeta, "before-enter")
			}
			if parseEntry.HasBeforeLeave {
				parseMeta = append(parseMeta, "before-leave")
			}
			parseStack = append(parseStack, fmt.Sprintf("%s[%s]", emptyFallback(parseEntry.Path, parseEntry.ID), strings.Join(parseMeta, ",")))
		}
		parseRows = append(parseRows, metricRow("Stack", strings.Join(parseStack, " -> ")))
	}
	if len(parseRoute.Loaders) > 0 {
		parseLoaders := make([]string, 0, len(parseRoute.Loaders))
		for _, parseLoader := range parseRoute.Loaders {
			parseLoaders = append(parseLoaders, fmt.Sprintf("%s pending=%t data=%t error=%s", emptyFallback(parseLoader.Path, parseLoader.Key), parseLoader.Pending, parseLoader.HasData, emptyFallback(parseLoader.Error, "none")))
		}
		parseRows = append(parseRows, metricRow("Loaders", strings.Join(parseLoaders, "; ")))
	}
	if strings.TrimSpace(parseRoute.LastRedirect.To) != "" {
		parseRows = append(parseRows, metricRow("Last redirect", emptyFallback(parseRoute.LastRedirect.Cause, "redirect")+": "+emptyFallback(parseRoute.LastRedirect.From, "unknown")+" -> "+parseRoute.LastRedirect.To))
	}
	if strings.TrimSpace(parseRoute.Metadata.Title) != "" || strings.TrimSpace(parseRoute.Metadata.Description) != "" || strings.TrimSpace(parseRoute.Metadata.CanonicalURL) != "" {
		parseRows = append(parseRows, metricRow("Metadata", strings.Join([]string{
			"title=" + emptyFallback(parseRoute.Metadata.Title, "none"),
			"description=" + emptyFallback(parseRoute.Metadata.Description, "none"),
			"canonical=" + emptyFallback(parseRoute.Metadata.CanonicalURL, "none"),
		}, " | ")))
	}
	return html.Div(html.Props{}, parseRows...)
}

func statsSummary(parseStats Stats) ui.Node {
	return html.Div(html.Props{},
		metricRow("Fibers", fmt.Sprintf("%d", parseStats.TotalFibers)),
		metricRow("Dirty", fmt.Sprintf("%d", parseStats.DirtyFibers)),
		metricRow("Components", fmt.Sprintf("%d", parseStats.ComponentFibers)),
		metricRow("Host nodes", fmt.Sprintf("%d", parseStats.HostFibers)),
		metricRow("Text nodes", fmt.Sprintf("%d", parseStats.TextFibers)),
		metricRow("Fine-grained", fmt.Sprintf("%d", parseStats.FineGrainedFibers)),
		metricRow("Hook entries", fmt.Sprintf("%d", parseStats.HookEntries)),
		metricRow("Effects", fmt.Sprintf("%d", parseStats.Effects)),
	)
}

func cacheSummary(parseEntries []CacheEntry) ui.Node {
	if len(parseEntries) == 0 {
		return html.Div(html.Props{}, metricRow("Entries", "0"))
	}

	parseRows := make([]ui.Node, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseRows = append(parseRows, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 0",
			"border-bottom": "1px solid rgba(30,41,59,0.8)",
		}},
			metricRow("Key", parseEntry.Key),
			metricRow("Ready", fmt.Sprintf("%t", parseEntry.Ready)),
			metricRow("Loading", fmt.Sprintf("%t", parseEntry.Loading)),
			metricRow("Stale", fmt.Sprintf("%t", parseEntry.Stale)),
			metricRow("Subscribers", fmt.Sprintf("%d", parseEntry.SubscriberCount)),
			func() ui.Node {
				if len(parseEntry.OwnerPaths) == 0 {
					return nil
				}
				return metricRow("Owners", strings.Join(parseEntry.OwnerPaths, " | "))
			}(),
			metricRow("Resume", emptyFallback(parseEntry.ResumePolicy, "trust-once")),
			metricRow("Updated", formatTime(parseEntry.UpdatedAt)),
			metricRow("Last load", formatTime(parseEntry.LastLoaded)),
			metricRow("Error", emptyFallback(parseEntry.LastError, "none")),
		))
	}
	return html.Div(html.Props{}, parseRows...)
}

func multiClientSummary(parseState MultiClient) ui.Node {
	if !parseState.Enabled && len(parseState.Peers) == 0 && len(parseState.RecentTraffic) == 0 && len(parseState.FailedPublishes) == 0 {
		return html.Div(html.Props{}, metricRow("State", "none"))
	}
	parseRows := []ui.Node{
		metricRow("Enabled", fmt.Sprintf("%t", parseState.Enabled)),
		metricRow("Local Peer", emptyFallback(parseState.LocalPeerID, "unknown")),
		metricRow("Transport", emptyFallback(parseState.ResolvedTransport, "unknown")),
		metricRow("Peers", fmt.Sprintf("%d", len(parseState.Peers))),
		metricRow("Traffic", fmt.Sprintf("%d", len(parseState.RecentTraffic))),
		metricRow("Failed Publishes", fmt.Sprintf("%d", len(parseState.FailedPublishes))),
	}
	if len(parseState.AuthorityView) > 0 {
		parseRows = append(parseRows, metricRow("Authority", formatStringMap(parseState.AuthorityView)))
	}
	if len(parseState.Peers) > 0 {
		parsePeerSummaries := make([]string, 0, len(parseState.Peers))
		for _, parsePeer := range parseState.Peers {
			parseLease := "n/a"
			if !parsePeer.LeaseDeadline.IsZero() {
				parseLease = parsePeer.LeaseDeadline.UTC().Format(time.RFC3339)
			}
			parsePeerSummaries = append(parsePeerSummaries, fmt.Sprintf("%s(%s/%s state=%s compatible=%t lease=%s)", emptyFallback(parsePeer.ID, "unknown"), emptyFallback(parsePeer.Surface, "unknown"), emptyFallback(parsePeer.Role, "none"), emptyFallback(parsePeer.State, "unknown"), parsePeer.Compatible, parseLease))
		}
		parseRows = append(parseRows, metricRow("Peer Registry", strings.Join(parsePeerSummaries, "; ")))
	}
	if len(parseState.RecentTraffic) > 0 {
		parseTrafficSummaries := make([]string, 0, len(parseState.RecentTraffic))
		for _, parseEntry := range parseState.RecentTraffic {
			parseTrafficSummaries = append(parseTrafficSummaries, fmt.Sprintf("%s %s %s peer=%s id=%s latency=%dms failed=%t", emptyFallback(parseEntry.Direction, "unknown"), emptyFallback(parseEntry.Kind, "unknown"), emptyFallback(parseEntry.Topic, "unknown"), emptyFallback(parseEntry.PeerID, "unknown"), emptyFallback(parseEntry.CorrelationID, "-"), parseEntry.LatencyMs, parseEntry.Failed))
		}
		parseRows = append(parseRows, metricRow("Recent Traffic", strings.Join(parseTrafficSummaries, "; ")))
	}
	if len(parseState.FailedPublishes) > 0 {
		parseFailureSummaries := make([]string, 0, len(parseState.FailedPublishes))
		for _, parseFailure := range parseState.FailedPublishes {
			parseFailureSummaries = append(parseFailureSummaries, fmt.Sprintf("%s topic=%s target=%s code=%s", emptyFallback(parseFailure.Op, "unknown"), emptyFallback(parseFailure.Topic, "unknown"), emptyFallback(parseFailure.Target, "-"), emptyFallback(parseFailure.Code, "unknown")))
		}
		parseRows = append(parseRows, metricRow("Failures", strings.Join(parseFailureSummaries, "; ")))
	}
	return html.Div(html.Props{}, parseRows...)
}

func boundariesSummary(parseState BoundaryInspection) ui.Node {
	if len(parseState.Entries) == 0 {
		return html.Div(html.Props{}, metricRow("Entries", "0"))
	}

	parseRows := make([]ui.Node, 0, len(parseState.Entries))
	for _, parseEntry := range parseState.Entries {
		parseMeta := []string{
			"kind=" + emptyFallback(parseEntry.Kind, "unknown"),
			"direction=" + emptyFallback(parseEntry.Direction, "unknown"),
			"status=" + emptyFallback(parseEntry.Status, "observed"),
			"encoding=" + emptyFallback(parseEntry.Encoding, "n/a"),
			"transport=" + emptyFallback(parseEntry.Transport, "n/a"),
		}
		if strings.TrimSpace(parseEntry.Scope) != "" {
			parseMeta = append(parseMeta, "scope="+parseEntry.Scope)
		}
		if strings.TrimSpace(parseEntry.Target) != "" {
			parseMeta = append(parseMeta, "target="+parseEntry.Target)
		}
		parseRows = append(parseRows, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 0",
			"border-bottom": "1px solid rgba(30,41,59,0.8)",
		}},
			metricRow("Name", emptyFallback(parseEntry.Name, "boundary")),
			metricRow("Size", fmt.Sprintf("%d bytes", parseEntry.SizeBytes)),
			func() ui.Node {
				if parseEntry.InlineBytes <= 0 && parseEntry.BinaryBytes <= 0 {
					return nil
				}
				return metricRow("Variants", fmt.Sprintf("inline=%d binary=%d", parseEntry.InlineBytes, parseEntry.BinaryBytes))
			}(),
			metricRow("Meta", strings.Join(parseMeta, " | ")),
			func() ui.Node {
				if strings.TrimSpace(parseEntry.CorrelationID) == "" {
					return nil
				}
				return metricRow("Correlation", parseEntry.CorrelationID)
			}(),
			func() ui.Node {
				if len(parseEntry.Redacted) == 0 {
					return nil
				}
				return metricRow("Redacted", strings.Join(parseEntry.Redacted, ", "))
			}(),
			func() ui.Node {
				if len(parseEntry.Downgraded) == 0 {
					return nil
				}
				return metricRow("Downgraded", strings.Join(parseEntry.Downgraded, ", "))
			}(),
			func() ui.Node {
				if len(parseEntry.Rejected) == 0 {
					return nil
				}
				return metricRow("Rejected", strings.Join(parseEntry.Rejected, " | "))
			}(),
			func() ui.Node {
				if len(parseEntry.Notes) == 0 {
					return nil
				}
				return metricRow("Notes", strings.Join(parseEntry.Notes, " | "))
			}(),
		))
	}
	return html.Div(html.Props{}, parseRows...)
}

func coordinationSummary(parseState Coordination) ui.Node {
	hasReconnect := strings.TrimSpace(parseState.Reconnect.State) != "" || strings.TrimSpace(parseState.Reconnect.Transport) != "" || parseState.Reconnect.Attempts > 0 || parseState.Reconnect.MaxAttempts > 0 || !parseState.Reconnect.NextRetryAt.IsZero() || !parseState.Reconnect.LastChange.IsZero() || parseState.Reconnect.IsConnected
	hasConflict := strings.TrimSpace(parseState.Conflict.Entity) != "" || strings.TrimSpace(parseState.Conflict.Status) != "" || strings.TrimSpace(parseState.Conflict.LastError) != "" || !parseState.Conflict.DetectedAt.IsZero()
	if len(parseState.Workers) == 0 && len(parseState.SyncEvents) == 0 && len(parseState.Replay) == 0 && len(parseState.QueueEntries) == 0 && len(parseState.SyncHealth) == 0 && !hasReconnect && !hasConflict && strings.TrimSpace(parseState.LastReplayError) == "" {
		return html.Div(html.Props{}, metricRow("State", "none"))
	}

	parseRows := []ui.Node{
		metricRow("Workers", fmt.Sprintf("%d", len(parseState.Workers))),
		metricRow("Sync events", fmt.Sprintf("%d", len(parseState.SyncEvents))),
		metricRow("Replay entries", fmt.Sprintf("%d", len(parseState.Replay))),
		metricRow("Queue entries", fmt.Sprintf("%d", len(parseState.QueueEntries))),
		metricRow("Sync health", fmt.Sprintf("%d", len(parseState.SyncHealth))),
	}
	if len(parseState.Workers) > 0 {
		parseSummaries := make([]string, 0, len(parseState.Workers))
		for _, parseWorker := range parseState.Workers {
			parseSummaries = append(parseSummaries, fmt.Sprintf("%s status=%s running=%t ready=%t cancelled=%t error=%s", emptyFallback(parseWorker.Name, "worker"), emptyFallback(parseWorker.Status, "unknown"), parseWorker.Running, parseWorker.Ready, parseWorker.Cancelled, emptyFallback(parseWorker.Error, "none")))
		}
		parseRows = append(parseRows, metricRow("Worker jobs", strings.Join(parseSummaries, "; ")))
	}
	if len(parseState.SyncEvents) > 0 {
		parseSummaries2 := make([]string, 0, len(parseState.SyncEvents))
		for _, parseEvent := range parseState.SyncEvents {
			parseSummaries2 = append(parseSummaries2, fmt.Sprintf("%s %s topic=%s target=%s status=%s", emptyFallback(parseEvent.Transport, "sync"), emptyFallback(parseEvent.Direction, "unknown"), emptyFallback(parseEvent.Topic, emptyFallback(parseEvent.Channel, "unknown")), emptyFallback(parseEvent.Target, "-"), emptyFallback(parseEvent.Status, "observed")))
		}
		parseRows = append(parseRows, metricRow("Sync", strings.Join(parseSummaries2, "; ")))
	}
	if len(parseState.Replay) > 0 {
		parseSummaries3 := make([]string, 0, len(parseState.Replay))
		for _, parseEntry := range parseState.Replay {
			parseSummaries3 = append(parseSummaries3, fmt.Sprintf("%s owner=%s state=%s attempts=%d/%d next=%s error=%s", emptyFallback(parseEntry.Kind, parseEntry.ID), emptyFallback(parseEntry.Owner, "n/a"), emptyFallback(parseEntry.State, "queued"), parseEntry.Attempts, parseEntry.MaxAttempts, formatTime(parseEntry.NextAttemptAt), emptyFallback(parseEntry.LastError, "none")))
		}
		parseRows = append(parseRows, metricRow("Replay", strings.Join(parseSummaries3, "; ")))
	}
	if len(parseState.QueueEntries) > 0 {
		parseSummaries4 := make([]string, 0, len(parseState.QueueEntries))
		for _, parseEntry2 := range parseState.QueueEntries {
			parseSummaries4 = append(parseSummaries4, fmt.Sprintf("%s op=%s owner=%s state=%s attempts=%d/%d queued=%s error=%s", emptyFallback(parseEntry2.Entity, parseEntry2.ID), emptyFallback(parseEntry2.Operation, "mutation"), emptyFallback(parseEntry2.Owner, "n/a"), emptyFallback(parseEntry2.State, "queued"), parseEntry2.Attempts, parseEntry2.MaxAttempts, formatTime(parseEntry2.QueuedAt), emptyFallback(parseEntry2.LastError, "none")))
		}
		parseRows = append(parseRows, metricRow("Queue", strings.Join(parseSummaries4, "; ")))
	}
	if len(parseState.SyncHealth) > 0 {
		parseSummaries5 := make([]string, 0, len(parseState.SyncHealth))
		for _, parseEntry3 := range parseState.SyncHealth {
			parseSummaries5 = append(parseSummaries5, fmt.Sprintf("%s owner=%s status=%s pending=%d version=%s synced=%s error=%s", emptyFallback(parseEntry3.Entity, "entity"), emptyFallback(parseEntry3.Owner, "n/a"), emptyFallback(parseEntry3.Status, "unknown"), parseEntry3.PendingOps, emptyFallback(parseEntry3.Version, "n/a"), formatTime(parseEntry3.LastSyncAt), emptyFallback(parseEntry3.LastError, "none")))
		}
		parseRows = append(parseRows, metricRow("Health", strings.Join(parseSummaries5, "; ")))
	}
	if hasReconnect {
		parseRows = append(parseRows, metricRow("Reconnect", fmt.Sprintf("%s transport=%s connected=%t attempts=%d/%d next=%s since=%s", emptyFallback(parseState.Reconnect.State, "unknown"), emptyFallback(parseState.Reconnect.Transport, "n/a"), parseState.Reconnect.IsConnected, parseState.Reconnect.Attempts, parseState.Reconnect.MaxAttempts, formatTime(parseState.Reconnect.NextRetryAt), formatTime(parseState.Reconnect.LastChange))))
	}
	if hasConflict {
		parseRows = append(parseRows, metricRow("Conflict", fmt.Sprintf("%s owner=%s status=%s strategy=%s since=%s error=%s", emptyFallback(parseState.Conflict.Entity, "n/a"), emptyFallback(parseState.Conflict.Owner, "n/a"), emptyFallback(parseState.Conflict.Status, "none"), emptyFallback(parseState.Conflict.Strategy, "n/a"), formatTime(parseState.Conflict.DetectedAt), emptyFallback(parseState.Conflict.LastError, "none"))))
	}
	if strings.TrimSpace(parseState.LastReplayError) != "" {
		parseRows = append(parseRows, metricRow("Last replay error", parseState.LastReplayError))
	}
	return html.Div(html.Props{}, parseRows...)
}

func renderExtensionSections(parseSections []ExtensionSection) []ui.Node {
	if len(parseSections) == 0 {
		return nil
	}
	parseNodes := make([]ui.Node, 0, len(parseSections))
	for _, parseSectionState := range parseSections {
		parseCurrent := parseSectionState
		parseNodes = append(parseNodes, section(emptyFallback(parseCurrent.Name, "Extension"), extensionSectionSummary(parseCurrent)))
	}
	return parseNodes
}

func extensionSectionSummary(parseSectionState ExtensionSection) ui.Node {
	parseRows := make([]ui.Node, 0, len(parseSectionState.Summary)+len(parseSectionState.Lines))
	if len(parseSectionState.Summary) > 0 {
		parseKeys := make([]string, 0, len(parseSectionState.Summary))
		for parseKey := range parseSectionState.Summary {
			parseKeys = append(parseKeys, parseKey)
		}
		sort.Strings(parseKeys)
		for _, parseKey2 := range parseKeys {
			parseRows = append(parseRows, metricRow(parseKey2, parseSectionState.Summary[parseKey2]))
		}
	}
	for _, parseLine := range parseSectionState.Lines {
		if strings.TrimSpace(parseLine) == "" {
			continue
		}
		parseRows = append(parseRows, html.Small(html.Props{Style: map[string]string{
			"display":    "block",
			"margin-top": "4px",
			"color":      "#cbd5e1",
		}}, html.Text(parseLine)))
	}
	if len(parseRows) == 0 {
		return html.Div(html.Props{}, metricRow("State", "empty"))
	}
	return html.Div(html.Props{}, parseRows...)
}

func profilingSummary(parseProfiling Profiling) ui.Node {
	parseChildren := []ui.Node{
		metricRow("Render calls", fmt.Sprintf("%d", parseProfiling.RenderCalls)),
		metricRow("Root updates", fmt.Sprintf("%d", parseProfiling.ScheduledRootUpdates)),
		metricRow("Fiber marks", fmt.Sprintf("%d", parseProfiling.ScheduledFiberMarks)),
		metricRow("Granular marks", fmt.Sprintf("%d", parseProfiling.ScheduledGranularMarks)),
		metricRow("Work loops", fmt.Sprintf("%d", parseProfiling.WorkLoopPasses)),
		metricRow("Units processed", fmt.Sprintf("%d", parseProfiling.ProcessedUnits)),
		metricRow("Commits", fmt.Sprintf("%d", parseProfiling.CommitCount)),
		metricRow("Granular commits", fmt.Sprintf("%d", parseProfiling.FineGrainedCommits)),
		metricRow("Region host commits", fmt.Sprintf("%d", parseProfiling.FineGrainedDescendantHostCommits)),
		metricRow("Region text commits", fmt.Sprintf("%d", parseProfiling.FineGrainedDescendantTextCommits)),
		metricRow("Effects run", fmt.Sprintf("%d", parseProfiling.EffectExecutions)),
		metricRow("Cleanups run", fmt.Sprintf("%d", parseProfiling.CleanupExecutions)),
		metricRow("Last render", formatDurationNs(parseProfiling.LastRenderDurationNs)),
		metricRow("Last commit", formatDurationNs(parseProfiling.LastCommitDurationNs)),
		metricRow("Last effect", formatDurationNs(parseProfiling.LastEffectDurationNs)),
		metricRow("Last cleanup", formatDurationNs(parseProfiling.LastCleanupDurationNs)),
		metricRow("Total render", formatDurationNs(parseProfiling.PhaseTotals.RenderDurationNs)),
		metricRow("Total diff", formatDurationNs(parseProfiling.PhaseTotals.DiffDurationNs)),
		metricRow("Total commit", formatDurationNs(parseProfiling.PhaseTotals.CommitDurationNs)),
		metricRow("Total effect", formatDurationNs(parseProfiling.PhaseTotals.EffectDurationNs)),
		metricRow("Total cleanup", formatDurationNs(parseProfiling.PhaseTotals.CleanupDurationNs)),
	}
	if len(parseProfiling.RecentEvents) > 0 {
		parseChildren = append(parseChildren, profilingEventsSummary(parseProfiling.RecentEvents))
	}
	if strings.TrimSpace(parseProfiling.Startup.Mode) != "" || parseProfiling.Startup.FirstInteractionCaptured || parseProfiling.Startup.BootstrapReadDurationNs > 0 || parseProfiling.Startup.HydrationDurationNs > 0 {
		parseChildren = append(parseChildren, startupProfilingSummary(parseProfiling.Startup))
	}
	if len(parseProfiling.ComponentRenders) > 0 {
		parseChildren = append(parseChildren, componentRenderSummary(parseProfiling.ComponentRenders))
	}
	if len(parseProfiling.FlamegraphFrames) > 0 {
		parseChildren = append(parseChildren, flamegraphSummary(parseProfiling.FlamegraphFrames))
	}
	if len(parseProfiling.HotBranches) > 0 {
		parseChildren = append(parseChildren, hotBranchesSummary(parseProfiling.HotBranches))
	}
	return html.Div(html.Props{}, parseChildren...)
}

func hydrationSummary(parseHydration HydrationDebug) ui.Node {
	if strings.TrimSpace(parseHydration.CorrelationID) == "" &&
		strings.TrimSpace(parseHydration.StartedAt) == "" &&
		parseHydration.DurationNs <= 0 &&
		parseHydration.ExistingDOMNodeCount == 0 &&
		parseHydration.FallbackCount == 0 &&
		parseHydration.MismatchCount == 0 &&
		parseHydration.DiscardedNodeCount == 0 &&
		!parseHydration.Strict &&
		!parseHydration.Failed &&
		len(parseHydration.RecentMessages) == 0 {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No hydration activity captured yet."))
	}

	parseRows := []ui.Node{
		metricRow("Correlation", emptyFallback(parseHydration.CorrelationID, "n/a")),
		metricRow("Started", emptyFallback(parseHydration.StartedAt, "n/a")),
		metricRow("Finished", emptyFallback(parseHydration.FinishedAt, "n/a")),
		metricRow("Duration", formatDurationNs(parseHydration.DurationNs)),
		metricRow("Existing DOM", fmt.Sprintf("%d", parseHydration.ExistingDOMNodeCount)),
		metricRow("Fallbacks", fmt.Sprintf("%d", parseHydration.FallbackCount)),
		metricRow("Mismatches", fmt.Sprintf("%d", parseHydration.MismatchCount)),
		metricRow("Discarded", fmt.Sprintf("%d", parseHydration.DiscardedNodeCount)),
		metricRow("Strict", fmt.Sprintf("%t", parseHydration.Strict)),
		metricRow("Failed", fmt.Sprintf("%t", parseHydration.Failed)),
	}
	if strings.TrimSpace(parseHydration.Failure) != "" {
		parseRows = append(parseRows, metricRow("Failure", parseHydration.Failure))
	}
	if len(parseHydration.RecentMessages) > 0 {
		parseRows = append(parseRows, metricRow("Recent", strings.Join(parseHydration.RecentMessages, " | ")))
	}
	return html.Div(html.Props{}, parseRows...)
}

func profilingEventsSummary(parseEvents []ProfilingEvent) ui.Node {
	parseItems := make([]ui.Node, 0, 4)
	parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
		"margin-top":     "10px",
		"margin-bottom":  "8px",
		"font-size":      "12px",
		"text-transform": "uppercase",
		"letter-spacing": "0.08em",
		"color":          "#67e8f9",
	}}, html.Text("Recent events")))
	parseStart := 0
	if len(parseEvents) > 3 {
		parseStart = len(parseEvents) - 3
	}
	for parseIndex := len(parseEvents) - 1; parseIndex >= parseStart; parseIndex-- {
		parseEvent := parseEvents[parseIndex]
		parseLabel := strings.TrimSpace(parseEvent.Domain) + "." + strings.TrimSpace(parseEvent.Name) + ":" + strings.TrimSpace(parseEvent.Phase)
		parseTarget := emptyFallback(parseEvent.Target, "-")
		parseDuration := formatDurationNs(parseEvent.DurationNs)
		parseItems = append(parseItems, html.Small(html.Props{Style: map[string]string{
			"display":    "block",
			"margin-top": "4px",
			"color":      "#cbd5e1",
		}}, html.Text(parseLabel+" target="+parseTarget+" duration="+parseDuration)))
	}
	return html.Div(html.Props{}, parseItems...)
}

func startupProfilingSummary(parseStartup StartupProfiling) ui.Node {
	parseItems := []ui.Node{
		html.Div(html.Props{Style: map[string]string{
			"margin-top":     "10px",
			"margin-bottom":  "8px",
			"font-size":      "12px",
			"text-transform": "uppercase",
			"letter-spacing": "0.08em",
			"color":          "#67e8f9",
		}}, html.Text("Startup workflow")),
		metricRow("Mode", emptyFallback(parseStartup.Mode, "n/a")),
		metricRow("Started", emptyFallback(parseStartup.StartedAt, "n/a")),
		metricRow("Bootstrap read", formatDurationNs(parseStartup.BootstrapReadDurationNs)),
		metricRow("WASM transfer", formatByteCount(parseStartup.WASMTransferBytes)),
		metricRow("WASM decoded", formatByteCount(parseStartup.WASMDecodedBytes)),
		metricRow("Bootstrap decoded", formatByteCount(parseStartup.BootstrapDecodedBytes)),
		metricRow("Cache warmup", formatDurationNs(parseStartup.CacheWarmupDurationNs)),
		metricRow("Service worker", formatDurationNs(parseStartup.ServiceWorkerOverheadNs)),
		metricRow("Initial route data", formatByteCount(parseStartup.InitialRouteDataBytes)),
		metricRow("Hydration", formatDurationNs(parseStartup.HydrationDurationNs)),
		metricRow("First commit", formatDurationNs(parseStartup.StartupCommitDurationNs)),
		metricRow("First interaction", formatDurationNs(parseStartup.FirstInteractionDurationNs)),
		metricRow("Interaction captured", fmt.Sprintf("%t", parseStartup.FirstInteractionCaptured)),
	}
	if strings.TrimSpace(parseStartup.FirstInteractionEvent) != "" {
		parseItems = append(parseItems, metricRow("Interaction event", parseStartup.FirstInteractionEvent))
	}
	if len(parseStartup.RouteBudgets) > 0 {
		parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
			"margin-top":     "10px",
			"margin-bottom":  "8px",
			"font-size":      "12px",
			"text-transform": "uppercase",
			"letter-spacing": "0.08em",
			"color":          "#67e8f9",
		}}, html.Text("Route startup budgets")))
		parseLimit := len(parseStartup.RouteBudgets)
		if parseLimit > 5 {
			parseLimit = 5
		}
		for parseIndex := 0; parseIndex < parseLimit; parseIndex++ {
			parseBudget := parseStartup.RouteBudgets[parseIndex]
			parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
				"padding":       "8px 10px",
				"border-radius": "10px",
				"border":        "1px solid rgba(51,65,85,0.7)",
				"margin-bottom": "8px",
			}},
				html.Small(html.Props{Style: map[string]string{"display": "block", "color": "#94a3b8"}}, html.Text("family="+emptyFallback(parseBudget.RouteFamily, "n/a")+" sample="+fmt.Sprintf("%d", parseBudget.SampleCount))),
				html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("path="+emptyFallback(parseBudget.LastRoutePath, "n/a"))),
				html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("bootstrap="+formatDurationNs(parseBudget.AverageBootstrapReadDurationNs)+" hydration="+formatDurationNs(parseBudget.AverageHydrationDurationNs)+" commit="+formatDurationNs(parseBudget.AverageStartupCommitDurationNs)+" first-interaction="+formatDurationNs(parseBudget.AverageFirstInteractionDurationNs))),
				html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("wasm="+formatByteCount(parseBudget.AverageWASMTransferBytes)+" decoded="+formatByteCount(parseBudget.AverageWASMDecodedBytes)+" bootstrap="+formatByteCount(parseBudget.AverageBootstrapDecodedBytes)+" route-data="+formatByteCount(parseBudget.AverageInitialRouteDataBytes))),
				html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("cache="+formatDurationNs(parseBudget.AverageCacheWarmupDurationNs)+" service-worker="+formatDurationNs(parseBudget.AverageServiceWorkerOverheadNs))),
			))
		}
	}
	return html.Div(html.Props{}, parseItems...)
}

func componentRenderSummary(parseTraces []ComponentRenderTrace) ui.Node {
	parseItems := make([]ui.Node, 0, 6)
	parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
		"margin-top":     "10px",
		"margin-bottom":  "8px",
		"font-size":      "12px",
		"text-transform": "uppercase",
		"letter-spacing": "0.08em",
		"color":          "#67e8f9",
	}}, html.Text("Component rerenders")))
	parseLimit := len(parseTraces)
	if parseLimit > 5 {
		parseLimit = 5
	}
	for parseIndex := 0; parseIndex < parseLimit; parseIndex++ {
		parseTrace := parseTraces[parseIndex]
		parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 10px",
			"border-radius": "10px",
			"border":        "1px solid rgba(51,65,85,0.7)",
			"margin-bottom": "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "gap": "8px", "align-items": "baseline"}},
				html.Strong(html.Props{Style: map[string]string{"color": "#f8fafc"}}, html.Text(emptyFallback(parseTrace.Name, "Component"))),
				html.Code(html.Props{Style: map[string]string{"color": "#67e8f9"}}, html.Text(fmt.Sprintf("renders=%d rerenders=%d", parseTrace.RenderCount, parseTrace.RerenderCount))),
			),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text(emptyFallback(parseTrace.Path, "path unavailable"))),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("trigger="+emptyFallback(parseTrace.LastTrigger, "unknown")+" avg="+formatDurationNs(parseTrace.AverageRenderDurationNs)+" last="+formatDurationNs(parseTrace.LastRenderDurationNs))),
		))
	}
	return html.Div(html.Props{}, parseItems...)
}

func flamegraphSummary(parseFrames []FlamegraphFrame) ui.Node {
	if len(parseFrames) == 0 {
		return nil
	}
	parseItems := make([]ui.Node, 0, 14)
	parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
		"margin-top":     "10px",
		"margin-bottom":  "8px",
		"font-size":      "12px",
		"text-transform": "uppercase",
		"letter-spacing": "0.08em",
		"color":          "#67e8f9",
	}}, html.Text("Flamegraph capture")))
	parseMaxEnd := int64(0)
	for _, parseFrame := range parseFrames {
		parseEnd := parseFrame.StartNs + parseFrame.DurationNs
		if parseEnd > parseMaxEnd {
			parseMaxEnd = parseEnd
		}
	}
	if parseMaxEnd <= 0 {
		parseMaxEnd = 1
	}
	parseLimit := len(parseFrames)
	if parseLimit > 12 {
		parseLimit = 12
	}
	for parseIndex := 0; parseIndex < parseLimit; parseIndex++ {
		parseFrame2 := parseFrames[parseIndex]
		parseLeftPct := (float64(parseFrame2.StartNs) / float64(parseMaxEnd)) * 100
		parseWidthPct := (float64(parseFrame2.DurationNs) / float64(parseMaxEnd)) * 100
		if parseWidthPct < 3 {
			parseWidthPct = 3
		}
		parseColor := "#38bdf8"
		switch parseFrame2.Depth % 3 {
		case 1:
			parseColor = "#22d3ee"
		case 2:
			parseColor = "#34d399"
		}
		parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
			"position":      "relative",
			"height":        "26px",
			"margin-bottom": "6px",
			"border-radius": "8px",
			"background":    "rgba(15,23,42,0.6)",
			"overflow":      "hidden",
		}},
			html.Div(html.Props{Style: map[string]string{
				"position":        "absolute",
				"left":            fmt.Sprintf("%.2f%%", parseLeftPct),
				"width":           fmt.Sprintf("%.2f%%", parseWidthPct),
				"height":          "100%",
				"background":      parseColor,
				"opacity":         "0.35",
				"border":          "1px solid rgba(125,211,252,0.35)",
				"border-radius":   "8px",
				"display":         "flex",
				"align-items":     "center",
				"justify-content": "space-between",
				"gap":             "6px",
				"padding":         "0 8px",
			}},
				html.Small(html.Props{Style: map[string]string{"color": "#f8fafc", "white-space": "nowrap", "overflow": "hidden", "text-overflow": "ellipsis"}}, html.Text(emptyFallback(parseFrame2.Name, "node"))),
				html.Small(html.Props{Style: map[string]string{"color": "#e2e8f0", "white-space": "nowrap"}}, html.Text(formatDurationNs(parseFrame2.DurationNs))),
			),
		))
	}
	return html.Div(html.Props{}, parseItems...)
}

func hotBranchesSummary(parseBranches []Branch) ui.Node {
	parseItems := make([]ui.Node, 0, len(parseBranches)+1)
	parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
		"margin-top":     "10px",
		"margin-bottom":  "8px",
		"font-size":      "12px",
		"text-transform": "uppercase",
		"letter-spacing": "0.08em",
		"color":          "#67e8f9",
	}}, html.Text("Hot branches")))
	for _, parseBranch := range parseBranches {
		parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 10px",
			"border-radius": "10px",
			"border":        "1px solid rgba(51,65,85,0.7)",
			"margin-bottom": "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "gap": "8px", "align-items": "baseline"}},
				html.Strong(html.Props{Style: map[string]string{"color": "#f8fafc"}}, html.Text(parseBranch.Name)),
				html.Code(html.Props{Style: map[string]string{"color": "#67e8f9"}}, html.Text(formatDurationNs(parseBranch.SubtreeDurationNs))),
			),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text(parseBranch.Path)),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("self="+formatDurationNs(parseBranch.SelfDurationNs)+" render="+formatDurationNs(parseBranch.RenderDurationNs)+" diff="+formatDurationNs(parseBranch.DiffDurationNs)+" commit="+formatDurationNs(parseBranch.CommitDurationNs)+" effect="+formatDurationNs(parseBranch.EffectDurationNs)+" cleanup="+formatDurationNs(parseBranch.CleanupDurationNs))),
		))
	}
	return html.Div(html.Props{}, parseItems...)
}

func diagnosticsSummary(parseDiagnostics []Diagnostic) ui.Node {
	if len(parseDiagnostics) == 0 {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No diagnostics reported."))
	}

	parseItems := make([]ui.Node, 0, len(parseDiagnostics))
	for _, parseDiagnostic := range parseDiagnostics {
		parseColor := "#f8fafc"
		switch parseDiagnostic.Severity {
		case SeverityWarning:
			parseColor = "#fde68a"
		case SeverityError:
			parseColor = "#fda4af"
		}
		parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 10px",
			"border-radius": "10px",
			"border":        "1px solid rgba(51,65,85,0.7)",
			"margin-bottom": "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"font-size": "12px", "text-transform": "uppercase", "letter-spacing": "0.08em", "color": parseColor}}, html.Text(string(parseDiagnostic.Severity)+" â€¢ "+parseDiagnostic.Source+" â€¢ count="+fmt.Sprintf("%d", parseDiagnostic.Count))),
			func() ui.Node {
				if strings.TrimSpace(parseDiagnostic.Code) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#67e8f9"}}, html.Text("code: "+parseDiagnostic.Code))
			}(),
			html.P(html.Props{Style: map[string]string{"margin": "6px 0 0 0", "color": "#cbd5e1"}}, html.Text(parseDiagnostic.Message)),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text("class: "+string(parseDiagnostic.Classification)+" | recoverable="+fmt.Sprintf("%t", parseDiagnostic.Recoverable))),
			func() ui.Node {
				if strings.TrimSpace(parseDiagnostic.TopFrame) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "6px", "color": "#cbd5e1"}}, html.Text("where: "+parseDiagnostic.TopFrame))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseDiagnostic.Path) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "6px", "color": "#94a3b8"}}, html.Text("path: "+parseDiagnostic.Path))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseDiagnostic.Consequence) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("runtime: "+parseDiagnostic.Consequence))
			}(),
			func() ui.Node {
				if len(parseDiagnostic.ComponentStack) == 0 {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("stack: "+strings.Join(parseDiagnostic.ComponentStack, " > ")))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseDiagnostic.Remediation) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "6px", "color": "#cbd5e1"}}, html.Text("next step: "+parseDiagnostic.Remediation))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseDiagnostic.Docs) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text("docs: "+parseDiagnostic.Docs))
			}(),
		))
	}
	return html.Div(html.Props{}, parseItems...)
}

func logsSummary(parseEntries []Log) ui.Node {
	if len(parseEntries) == 0 {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No framework logs buffered."))
	}

	parseItems := make([]ui.Node, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseColor := "#cbd5e1"
		switch parseEntry.Level {
		case LogLevel(runtime.LogWarn):
			parseColor = "#fde68a"
		case LogLevel(runtime.LogError):
			parseColor = "#fda4af"
		case LogLevel(runtime.LogInfo):
			parseColor = "#67e8f9"
		}
		parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 10px",
			"border-radius": "10px",
			"border":        "1px solid rgba(51,65,85,0.7)",
			"margin-bottom": "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"font-size": "12px", "text-transform": "uppercase", "letter-spacing": "0.08em", "color": parseColor}}, html.Text(string(parseEntry.Level)+" | "+parseEntry.Domain+" | "+string(parseEntry.Classification))),
			func() ui.Node {
				if strings.TrimSpace(parseEntry.Code) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#67e8f9"}}, html.Text("code: "+parseEntry.Code+" | recoverable="+fmt.Sprintf("%t", parseEntry.Recoverable)))
			}(),
			html.P(html.Props{Style: map[string]string{"margin": "6px 0 0 0", "color": "#e2e8f0"}}, html.Text(parseEntry.Message)),
			func() ui.Node {
				parseMeta := []string{}
				if strings.TrimSpace(parseEntry.Timestamp) != "" {
					parseMeta = append(parseMeta, parseEntry.Timestamp)
				}
				if strings.TrimSpace(parseEntry.CorrelationID) != "" {
					parseMeta = append(parseMeta, "corr="+parseEntry.CorrelationID)
				}
				if len(parseMeta) == 0 {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text(strings.Join(parseMeta, " | ")))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseEntry.TopFrame) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("where: "+parseEntry.TopFrame))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseEntry.Consequence) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("runtime: "+parseEntry.Consequence))
			}(),
			func() ui.Node {
				if len(parseEntry.Fields) == 0 {
					return nil
				}
				parseKeys := make([]string, 0, len(parseEntry.Fields))
				for parseKey := range parseEntry.Fields {
					parseKeys = append(parseKeys, parseKey)
				}
				sort.Strings(parseKeys)
				parseParts := make([]string, 0, len(parseKeys))
				for _, parseKey2 := range parseKeys {
					parseParts = append(parseParts, parseKey2+"="+parseEntry.Fields[parseKey2])
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text(strings.Join(parseParts, " | ")))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseEntry.Remediation) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("next step: "+parseEntry.Remediation))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseEntry.Docs) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text("docs: "+parseEntry.Docs))
			}(),
		))
	}
	return html.Div(html.Props{}, parseItems...)
}
