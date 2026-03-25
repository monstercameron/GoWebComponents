//go:build js && wasm
// +build js,wasm

package devtools

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// SnapshotNow captures the current runtime, route, and diagnostic inspection state.
func SnapshotNow() Snapshot {
	if replay, ok := CurrentTraceReplay(); ok {
		return replay.Snapshot
	}
	return snapshotNowLive()
}

func snapshotNowLive() Snapshot {
	rtSnapshot := runtime.GetGlobalRuntime().Inspect()
	routeInspection := router.InspectCurrentRoute()

	query := make(map[string][]string, len(routeInspection.Query))
	for key, values := range routeInspection.Query {
		query[key] = append([]string(nil), values...)
	}

	snapshot := Snapshot{
		Route: Route{
			Path:    routeInspection.Path,
			Query:   query,
			Params:  cloneParams(routeInspection.Params),
			Loading: routeInspection.Loading,
			Stack:   mapRouteStack(routeInspection.Stack),
			Loaders: mapRouteLoaders(routeInspection.Loaders),
			LastRedirect: RouteRedirect{
				Cause: routeInspection.LastRedirect.Cause,
				From:  routeInspection.LastRedirect.From,
				To:    routeInspection.LastRedirect.To,
			},
			Metadata: RouteMetadata{
				Title:        routeInspection.Metadata.Title,
				Description:  routeInspection.Metadata.Description,
				CanonicalURL: routeInspection.Metadata.CanonicalURL,
			},
		},
		MultiClient:  InspectMultiClient(),
		Boundaries:   InspectSerializationBoundaries(),
		Coordination: InspectCoordination(),
		Extensions:   InspectExtensionSections(),
		Tree:         mapNode(rtSnapshot.Root),
		Stats:        mapStats(rtSnapshot.Stats),
		Profiling:    mapProfiling(rtSnapshot.Profiling),
		Hydration:    mapHydration(rtSnapshot.Hydration),
	}
	for _, entry := range fetch.InspectCachedResources() {
		snapshot.Cache = append(snapshot.Cache, CacheEntry{
			Key:             entry.Key,
			Loading:         entry.Loading,
			Ready:           entry.Ready,
			Stale:           entry.Stale,
			LastError:       entry.LastError,
			UpdatedAt:       entry.UpdatedAt,
			LastLoaded:      entry.LastLoaded,
			SubscriberCount: entry.SubscriberCount,
			OwnerPaths:      append([]string(nil), entry.OwnerPaths...),
			ResumePolicy:    string(entry.ResumePolicy),
		})
	}
	for _, diagnostic := range rtSnapshot.Diagnostics {
		snapshot.Diagnostics = append(snapshot.Diagnostics, Diagnostic{
			Source:         diagnostic.Source,
			Severity:       Severity(diagnostic.Severity),
			Classification: Classification(diagnostic.Classification),
			Code:           diagnostic.Code,
			Docs:           diagnostic.Docs,
			Remediation:    diagnostic.Remediation,
			Recoverable:    diagnostic.Recoverable,
			TopFrame:       diagnostic.TopFrame,
			Consequence:    diagnostic.Consequence,
			Message:        diagnostic.Message,
			Count:          diagnostic.Count,
			Path:           diagnostic.Path,
			ComponentStack: append([]string(nil), diagnostic.ComponentStack...),
			Fields:         cloneStringMap(diagnostic.Fields),
		})
	}
	for _, entry := range rtSnapshot.Logs {
		snapshot.Logs = append(snapshot.Logs, Log{
			Domain:         entry.Domain,
			Level:          LogLevel(entry.Level),
			Classification: Classification(entry.Classification),
			Code:           entry.Code,
			Docs:           entry.Docs,
			Remediation:    entry.Remediation,
			Recoverable:    entry.Recoverable,
			TopFrame:       entry.TopFrame,
			Consequence:    entry.Consequence,
			Message:        entry.Message,
			Timestamp:      entry.Timestamp,
			CorrelationID:  entry.CorrelationID,
			Fields:         cloneStringMap(entry.Fields),
		})
	}
	return snapshot
}

// UseSnapshot polls SnapshotNow on an interval and returns the latest snapshot.
func UseSnapshot(refreshInterval time.Duration) Snapshot {
	interval := refreshInterval
	if interval <= 0 {
		interval = 750 * time.Millisecond
	}

	state := ui.UseState(SnapshotNow())
	ui.UseEffect(func() func() {
		state.Set(SnapshotNow())
		ticker := time.NewTicker(interval)
		stop := make(chan struct{})

		go func() {
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					state.Set(SnapshotNow())
				}
			}
		}()

		return func() {
			close(stop)
			ticker.Stop()
		}
	}, interval)

	return state.Get()
}

// Panel renders an embeddable in-browser devtools overlay.
func Panel(props PanelProps) ui.Node {
	title := strings.TrimSpace(props.Title)
	if title == "" {
		title = "GWC Devtools"
	}
	maxDepth := props.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 4
	}

	open := ui.UseState(props.InitiallyOpen)
	snapshot := UseSnapshot(props.RefreshInterval)
	selectedPath := ui.UseState("")
	toggle := ui.UseEvent(func() {
		open.Update(func(current bool) bool { return !current })
	})
	ui.UseEffect(func() func() {
		if snapshot.Tree != nil && strings.TrimSpace(selectedPath.Get()) == "" {
			selectedPath.Set(snapshot.Tree.Path)
		}
		return nil
	}, snapshot.Tree)

	if !open.Get() {
		return html.Button(html.Props{
			OnClick: toggle,
			Style: map[string]string{
				"position":      "fixed",
				"right":         "16px",
				"bottom":        "16px",
				"z-index":       "9999",
				"padding":       "10px 14px",
				"border-radius": "999px",
				"border":        "1px solid rgba(56,189,248,0.45)",
				"background":    "rgba(8,47,73,0.95)",
				"color":         "#e0f2fe",
				"font-weight":   "700",
				"cursor":        "pointer",
				"box-shadow":    "0 12px 40px rgba(2,6,23,0.45)",
			},
		}, html.Text(title))
	}

	children := []ui.Node{
		header(title, toggle),
		section("Route", routeSummary(snapshot.Route)),
		section("Cache", cacheSummary(snapshot.Cache)),
		section("Multi-Client", multiClientSummary(snapshot.MultiClient)),
		section("Boundaries", boundariesSummary(snapshot.Boundaries)),
		section("Coordination", coordinationSummary(snapshot.Coordination)),
		section("Runtime", statsSummary(snapshot.Stats)),
		section("Inspector", selectedNodeSummary(snapshot, selectedPath.Get())),
		section("Profiling", profilingSummary(snapshot.Profiling)),
		section("Hydration", hydrationSummary(snapshot.Hydration)),
		section("Logs", logsSummary(snapshot.Logs)),
		section("Diagnostics", diagnosticsSummary(snapshot.Diagnostics)),
		section("Tree", treeSummary(snapshot.Tree, 0, maxDepth, selectedPath.Get(), func(path string) {
			selectedPath.Set(path)
		})),
	}
	children = append(children, renderExtensionSections(snapshot.Extensions)...)

	return html.Div(html.Props{Style: map[string]string{
		"position":      "fixed",
		"right":         "16px",
		"bottom":        "16px",
		"z-index":       "9999",
		"width":         "420px",
		"max-width":     "calc(100vw - 32px)",
		"max-height":    "calc(100vh - 32px)",
		"overflow":      "auto",
		"border-radius": "20px",
		"border":        "1px solid rgba(148,163,184,0.25)",
		"background":    "rgba(2,6,23,0.95)",
		"color":         "#e2e8f0",
		"padding":       "16px",
		"font-family":   "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
		"box-shadow":    "0 24px 64px rgba(2,6,23,0.55)",
	}}, children...)
}

// ErrorOverlay renders a focused development overlay for current runtime failures.
func ErrorOverlay(props ErrorOverlayProps) ui.Node {
	interval := props.RefreshInterval
	if interval <= 0 {
		interval = 750 * time.Millisecond
	}
	title := strings.TrimSpace(props.Title)
	if title == "" {
		title = "GWC Error Overlay"
	}
	maxItems := props.MaxItems
	if maxItems <= 0 {
		maxItems = 4
	}

	snapshot := UseSnapshot(interval)
	issues := collectOverlayIssues(snapshot)
	actions := InspectErrorOverlayActions()
	dismissed := ui.UseState(false)
	signature := overlayIssueFingerprint(issues)
	ui.UseEffect(func() func() {
		dismissed.Set(false)
		return nil
	}, signature)
	if len(issues) == 0 {
		return nil
	}
	if len(issues) > maxItems {
		issues = issues[:maxItems]
	}
	if dismissed.Get() {
		return nil
	}

	closeOverlay := ui.UseEvent(func() {
		dismissed.Set(true)
	})

	items := make([]ui.Node, 0, len(issues)+1)
	items = append(items, html.Div(html.Props{Style: map[string]string{
		"display":         "flex",
		"align-items":     "center",
		"justify-content": "space-between",
		"gap":             "12px",
		"margin-bottom":   "12px",
	}},
		html.Div(html.Props{},
			html.Strong(html.Props{Style: map[string]string{"display": "block", "color": "#fee2e2", "font-size": "14px", "letter-spacing": "0.04em", "text-transform": "uppercase"}}, html.Text(title)),
			html.Small(html.Props{Style: map[string]string{"color": "#fca5a5"}}, html.Text(fmt.Sprintf("%d active framework failure(s)", len(issues)))),
		),
		html.Button(html.Props{OnClick: closeOverlay, Style: map[string]string{
			"border":        "1px solid rgba(248,113,113,0.35)",
			"background":    "rgba(69,10,10,0.85)",
			"color":         "#fee2e2",
			"border-radius": "999px",
			"padding":       "8px 12px",
			"cursor":        "pointer",
		}}, html.Text("Dismiss")),
	))
	for _, issue := range issues {
		color := "#fecaca"
		if issue.Severity == SeverityWarning {
			color = "#fde68a"
		}
		matchedActions := matchingErrorOverlayActions(issue, actions)
		items = append(items, html.Div(html.Props{Style: map[string]string{
			"padding":       "10px 12px",
			"border-radius": "12px",
			"border":        "1px solid rgba(248,113,113,0.22)",
			"background":    "rgba(127,29,29,0.32)",
			"margin-top":    "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"font-size": "12px", "text-transform": "uppercase", "letter-spacing": "0.08em", "color": color}}, html.Text(string(issue.Severity)+" | "+emptyFallback(issue.Source, "runtime")+" | "+emptyFallback(issue.Code, "diagnostic"))),
			html.P(html.Props{Style: map[string]string{"margin": "6px 0 0 0", "color": "#fee2e2"}}, html.Text(issue.Message)),
			func() ui.Node {
				if strings.TrimSpace(issue.TopFrame) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "6px", "color": "#fecaca"}}, html.Text("where: "+issue.TopFrame))
			}(),
			func() ui.Node {
				if strings.TrimSpace(issue.Path) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#fecaca"}}, html.Text("path: "+issue.Path))
			}(),
			func() ui.Node {
				if strings.TrimSpace(issue.Docs) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#fca5a5"}}, html.Text("docs: "+issue.Docs))
			}(),
			func() ui.Node {
				if len(matchedActions) == 0 {
					return nil
				}
				buttons := make([]ui.Node, 0, len(matchedActions))
				for _, action := range matchedActions {
					current := action
					buttons = append(buttons, html.Button(html.Props{OnClick: ui.WrapHandler(func() {
						if current.Run != nil {
							current.Run(ErrorOverlayActionContext{Snapshot: snapshot, Issue: issue})
						}
					}), Style: map[string]string{
						"border":        "1px solid rgba(248,113,113,0.28)",
						"background":    "rgba(69,10,10,0.72)",
						"color":         "#fee2e2",
						"border-radius": "999px",
						"padding":       "6px 10px",
						"cursor":        "pointer",
					}}, html.Text(action.Label)))
				}
				return html.Div(html.Props{Style: map[string]string{
					"display":    "flex",
					"flex-wrap":  "wrap",
					"gap":        "8px",
					"margin-top": "8px",
				}}, buttons...)
			}(),
		))
	}

	return html.Div(html.Props{Style: map[string]string{
		"position":      "fixed",
		"top":           "16px",
		"right":         "16px",
		"z-index":       "10000",
		"width":         "460px",
		"max-width":     "calc(100vw - 32px)",
		"max-height":    "calc(100vh - 32px)",
		"overflow":      "auto",
		"border-radius": "18px",
		"border":        "1px solid rgba(248,113,113,0.28)",
		"background":    "rgba(24,24,27,0.97)",
		"padding":       "16px",
		"box-shadow":    "0 24px 64px rgba(0,0,0,0.45)",
		"font-family":   "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
	}}, items...)
}

func header(title string, toggle ui.Handler) ui.Node {
	return html.Div(html.Props{Style: map[string]string{
		"display":         "flex",
		"align-items":     "center",
		"justify-content": "space-between",
		"gap":             "12px",
		"margin-bottom":   "14px",
	}},
		html.Div(html.Props{},
			html.Strong(html.Props{Style: map[string]string{"display": "block", "font-size": "14px", "letter-spacing": "0.08em", "text-transform": "uppercase", "color": "#67e8f9"}}, html.Text(title)),
			html.Small(html.Props{Style: map[string]string{"color": "#94a3b8"}}, html.Text("Component tree, hooks, routes, and diagnostics")),
		),
		html.Button(html.Props{OnClick: toggle, Style: map[string]string{
			"border":        "1px solid rgba(148,163,184,0.25)",
			"background":    "rgba(15,23,42,0.9)",
			"color":         "#e2e8f0",
			"border-radius": "999px",
			"padding":       "8px 12px",
			"cursor":        "pointer",
		}}, html.Text("Close")),
	)
}

func collectOverlayIssues(snapshot Snapshot) []ErrorOverlayIssue {
	issues := make([]ErrorOverlayIssue, 0, len(snapshot.Diagnostics)+1)
	for _, diagnostic := range snapshot.Diagnostics {
		if !shouldShowOverlayDiagnostic(diagnostic) {
			continue
		}
		issues = append(issues, ErrorOverlayIssue{
			Severity: diagnostic.Severity,
			Source:   diagnostic.Source,
			Code:     diagnostic.Code,
			Message:  diagnostic.Message,
			TopFrame: diagnostic.TopFrame,
			Path:     diagnostic.Path,
			Docs:     diagnostic.Docs,
		})
	}
	if snapshot.Hydration.Failed && !overlayHasHydrationIssue(issues) {
		issues = append(issues, ErrorOverlayIssue{
			Severity: SeverityError,
			Source:   "hydration",
			Code:     "GWC-HYDRATION-FAILED",
			Message:  emptyFallback(snapshot.Hydration.Failure, "hydration failed during client resume"),
			Docs:     "docs/ACTIONABLE_ERRORS.md#gwc-runtime-panic-hydration",
		})
	}
	return issues
}

func shouldShowOverlayDiagnostic(diagnostic Diagnostic) bool {
	if diagnostic.Severity == SeverityError {
		return true
	}
	if strings.HasPrefix(strings.TrimSpace(diagnostic.Code), "GWC-HYDRATION-") {
		return true
	}
	switch strings.TrimSpace(diagnostic.Code) {
	case "GWC-ROUTER-LOADER-FAILED":
		return true
	}
	return false
}

func overlayHasHydrationIssue(issues []ErrorOverlayIssue) bool {
	for _, issue := range issues {
		if strings.HasPrefix(strings.TrimSpace(issue.Code), "GWC-HYDRATION-") {
			return true
		}
	}
	return false
}

func overlayIssueFingerprint(issues []ErrorOverlayIssue) string {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, issue.Code+"|"+issue.Message+"|"+issue.TopFrame+"|"+issue.Path)
	}
	return strings.Join(parts, "\n")
}

func matchingErrorOverlayActions(issue ErrorOverlayIssue, actions []ErrorOverlayAction) []ErrorOverlayAction {
	if len(actions) == 0 {
		return nil
	}
	matched := make([]ErrorOverlayAction, 0, len(actions))
	for _, action := range actions {
		if !matchesErrorOverlayAction(issue, action) {
			continue
		}
		matched = append(matched, action)
	}
	return matched
}

func matchesErrorOverlayAction(issue ErrorOverlayIssue, action ErrorOverlayAction) bool {
	if len(action.MatchCodes) == 0 && len(action.MatchSources) == 0 {
		return true
	}
	if len(action.MatchCodes) > 0 {
		matched := false
		for _, code := range action.MatchCodes {
			if strings.EqualFold(strings.TrimSpace(code), strings.TrimSpace(issue.Code)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(action.MatchSources) > 0 {
		matched := false
		for _, source := range action.MatchSources {
			if strings.EqualFold(strings.TrimSpace(source), strings.TrimSpace(issue.Source)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func section(title string, content ui.Node) ui.Node {
	return html.Div(html.Props{Style: map[string]string{
		"margin-top":    "12px",
		"padding":       "12px",
		"border-radius": "14px",
		"background":    "rgba(15,23,42,0.88)",
		"border":        "1px solid rgba(51,65,85,0.7)",
	}},
		html.Div(html.Props{Style: map[string]string{"margin-bottom": "10px", "font-size": "12px", "text-transform": "uppercase", "letter-spacing": "0.08em", "color": "#67e8f9"}}, html.Text(title)),
		content,
	)
}

func routeSummary(route Route) ui.Node {
	rows := []ui.Node{
		metricRow("Path", emptyFallback(route.Path, "/")),
		metricRow("Loading", fmt.Sprintf("%t", route.Loading)),
	}
	rows = append(rows, metricRow("Query", formatQuery(route.Query)))
	rows = append(rows, metricRow("Params", formatParams(route.Params)))
	if len(route.Stack) > 0 {
		stack := make([]string, 0, len(route.Stack))
		for _, entry := range route.Stack {
			meta := make([]string, 0, 3)
			if entry.HasLoader {
				meta = append(meta, "loader")
			}
			if entry.HasBeforeEnter {
				meta = append(meta, "before-enter")
			}
			if entry.HasBeforeLeave {
				meta = append(meta, "before-leave")
			}
			stack = append(stack, fmt.Sprintf("%s[%s]", emptyFallback(entry.Path, entry.ID), strings.Join(meta, ",")))
		}
		rows = append(rows, metricRow("Stack", strings.Join(stack, " -> ")))
	}
	if len(route.Loaders) > 0 {
		loaders := make([]string, 0, len(route.Loaders))
		for _, loader := range route.Loaders {
			loaders = append(loaders, fmt.Sprintf("%s pending=%t data=%t error=%s", emptyFallback(loader.Path, loader.Key), loader.Pending, loader.HasData, emptyFallback(loader.Error, "none")))
		}
		rows = append(rows, metricRow("Loaders", strings.Join(loaders, "; ")))
	}
	if strings.TrimSpace(route.LastRedirect.To) != "" {
		rows = append(rows, metricRow("Last redirect", emptyFallback(route.LastRedirect.Cause, "redirect")+": "+emptyFallback(route.LastRedirect.From, "unknown")+" -> "+route.LastRedirect.To))
	}
	if strings.TrimSpace(route.Metadata.Title) != "" || strings.TrimSpace(route.Metadata.Description) != "" || strings.TrimSpace(route.Metadata.CanonicalURL) != "" {
		rows = append(rows, metricRow("Metadata", strings.Join([]string{
			"title=" + emptyFallback(route.Metadata.Title, "none"),
			"description=" + emptyFallback(route.Metadata.Description, "none"),
			"canonical=" + emptyFallback(route.Metadata.CanonicalURL, "none"),
		}, " | ")))
	}
	return html.Div(html.Props{}, rows...)
}

func statsSummary(stats Stats) ui.Node {
	return html.Div(html.Props{},
		metricRow("Fibers", fmt.Sprintf("%d", stats.TotalFibers)),
		metricRow("Dirty", fmt.Sprintf("%d", stats.DirtyFibers)),
		metricRow("Components", fmt.Sprintf("%d", stats.ComponentFibers)),
		metricRow("Host nodes", fmt.Sprintf("%d", stats.HostFibers)),
		metricRow("Text nodes", fmt.Sprintf("%d", stats.TextFibers)),
		metricRow("Fine-grained", fmt.Sprintf("%d", stats.FineGrainedFibers)),
		metricRow("Hook entries", fmt.Sprintf("%d", stats.HookEntries)),
		metricRow("Effects", fmt.Sprintf("%d", stats.Effects)),
	)
}

func cacheSummary(entries []CacheEntry) ui.Node {
	if len(entries) == 0 {
		return html.Div(html.Props{}, metricRow("Entries", "0"))
	}

	rows := make([]ui.Node, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 0",
			"border-bottom": "1px solid rgba(30,41,59,0.8)",
		}},
			metricRow("Key", entry.Key),
			metricRow("Ready", fmt.Sprintf("%t", entry.Ready)),
			metricRow("Loading", fmt.Sprintf("%t", entry.Loading)),
			metricRow("Stale", fmt.Sprintf("%t", entry.Stale)),
			metricRow("Subscribers", fmt.Sprintf("%d", entry.SubscriberCount)),
			func() ui.Node {
				if len(entry.OwnerPaths) == 0 {
					return nil
				}
				return metricRow("Owners", strings.Join(entry.OwnerPaths, " | "))
			}(),
			metricRow("Resume", emptyFallback(entry.ResumePolicy, "trust-once")),
			metricRow("Updated", formatTime(entry.UpdatedAt)),
			metricRow("Last load", formatTime(entry.LastLoaded)),
			metricRow("Error", emptyFallback(entry.LastError, "none")),
		))
	}
	return html.Div(html.Props{}, rows...)
}

func multiClientSummary(state MultiClient) ui.Node {
	if !state.Enabled && len(state.Peers) == 0 && len(state.RecentTraffic) == 0 && len(state.FailedPublishes) == 0 {
		return html.Div(html.Props{}, metricRow("State", "none"))
	}
	rows := []ui.Node{
		metricRow("Enabled", fmt.Sprintf("%t", state.Enabled)),
		metricRow("Local Peer", emptyFallback(state.LocalPeerID, "unknown")),
		metricRow("Transport", emptyFallback(state.ResolvedTransport, "unknown")),
		metricRow("Peers", fmt.Sprintf("%d", len(state.Peers))),
		metricRow("Traffic", fmt.Sprintf("%d", len(state.RecentTraffic))),
		metricRow("Failed Publishes", fmt.Sprintf("%d", len(state.FailedPublishes))),
	}
	if len(state.AuthorityView) > 0 {
		rows = append(rows, metricRow("Authority", formatStringMap(state.AuthorityView)))
	}
	if len(state.Peers) > 0 {
		peerSummaries := make([]string, 0, len(state.Peers))
		for _, peer := range state.Peers {
			lease := "n/a"
			if !peer.LeaseDeadline.IsZero() {
				lease = peer.LeaseDeadline.UTC().Format(time.RFC3339)
			}
			peerSummaries = append(peerSummaries, fmt.Sprintf("%s(%s/%s state=%s compatible=%t lease=%s)", emptyFallback(peer.ID, "unknown"), emptyFallback(peer.Surface, "unknown"), emptyFallback(peer.Role, "none"), emptyFallback(peer.State, "unknown"), peer.Compatible, lease))
		}
		rows = append(rows, metricRow("Peer Registry", strings.Join(peerSummaries, "; ")))
	}
	if len(state.RecentTraffic) > 0 {
		trafficSummaries := make([]string, 0, len(state.RecentTraffic))
		for _, entry := range state.RecentTraffic {
			trafficSummaries = append(trafficSummaries, fmt.Sprintf("%s %s %s peer=%s id=%s latency=%dms failed=%t", emptyFallback(entry.Direction, "unknown"), emptyFallback(entry.Kind, "unknown"), emptyFallback(entry.Topic, "unknown"), emptyFallback(entry.PeerID, "unknown"), emptyFallback(entry.CorrelationID, "-"), entry.LatencyMs, entry.Failed))
		}
		rows = append(rows, metricRow("Recent Traffic", strings.Join(trafficSummaries, "; ")))
	}
	if len(state.FailedPublishes) > 0 {
		failureSummaries := make([]string, 0, len(state.FailedPublishes))
		for _, failure := range state.FailedPublishes {
			failureSummaries = append(failureSummaries, fmt.Sprintf("%s topic=%s target=%s code=%s", emptyFallback(failure.Op, "unknown"), emptyFallback(failure.Topic, "unknown"), emptyFallback(failure.Target, "-"), emptyFallback(failure.Code, "unknown")))
		}
		rows = append(rows, metricRow("Failures", strings.Join(failureSummaries, "; ")))
	}
	return html.Div(html.Props{}, rows...)
}

func boundariesSummary(state BoundaryInspection) ui.Node {
	if len(state.Entries) == 0 {
		return html.Div(html.Props{}, metricRow("Entries", "0"))
	}

	rows := make([]ui.Node, 0, len(state.Entries))
	for _, entry := range state.Entries {
		meta := []string{
			"kind=" + emptyFallback(entry.Kind, "unknown"),
			"direction=" + emptyFallback(entry.Direction, "unknown"),
			"status=" + emptyFallback(entry.Status, "observed"),
			"encoding=" + emptyFallback(entry.Encoding, "n/a"),
			"transport=" + emptyFallback(entry.Transport, "n/a"),
		}
		if strings.TrimSpace(entry.Scope) != "" {
			meta = append(meta, "scope="+entry.Scope)
		}
		if strings.TrimSpace(entry.Target) != "" {
			meta = append(meta, "target="+entry.Target)
		}
		rows = append(rows, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 0",
			"border-bottom": "1px solid rgba(30,41,59,0.8)",
		}},
			metricRow("Name", emptyFallback(entry.Name, "boundary")),
			metricRow("Size", fmt.Sprintf("%d bytes", entry.SizeBytes)),
			func() ui.Node {
				if entry.InlineBytes <= 0 && entry.BinaryBytes <= 0 {
					return nil
				}
				return metricRow("Variants", fmt.Sprintf("inline=%d binary=%d", entry.InlineBytes, entry.BinaryBytes))
			}(),
			metricRow("Meta", strings.Join(meta, " | ")),
			func() ui.Node {
				if strings.TrimSpace(entry.CorrelationID) == "" {
					return nil
				}
				return metricRow("Correlation", entry.CorrelationID)
			}(),
			func() ui.Node {
				if len(entry.Redacted) == 0 {
					return nil
				}
				return metricRow("Redacted", strings.Join(entry.Redacted, ", "))
			}(),
			func() ui.Node {
				if len(entry.Downgraded) == 0 {
					return nil
				}
				return metricRow("Downgraded", strings.Join(entry.Downgraded, ", "))
			}(),
			func() ui.Node {
				if len(entry.Rejected) == 0 {
					return nil
				}
				return metricRow("Rejected", strings.Join(entry.Rejected, " | "))
			}(),
			func() ui.Node {
				if len(entry.Notes) == 0 {
					return nil
				}
				return metricRow("Notes", strings.Join(entry.Notes, " | "))
			}(),
		))
	}
	return html.Div(html.Props{}, rows...)
}

func coordinationSummary(state Coordination) ui.Node {
	hasReconnect := strings.TrimSpace(state.Reconnect.State) != "" || strings.TrimSpace(state.Reconnect.Transport) != "" || state.Reconnect.Attempts > 0 || state.Reconnect.MaxAttempts > 0 || !state.Reconnect.NextRetryAt.IsZero() || !state.Reconnect.LastChange.IsZero() || state.Reconnect.IsConnected
	hasConflict := strings.TrimSpace(state.Conflict.Entity) != "" || strings.TrimSpace(state.Conflict.Status) != "" || strings.TrimSpace(state.Conflict.LastError) != "" || !state.Conflict.DetectedAt.IsZero()
	if len(state.Workers) == 0 && len(state.SyncEvents) == 0 && len(state.Replay) == 0 && len(state.QueueEntries) == 0 && len(state.SyncHealth) == 0 && !hasReconnect && !hasConflict && strings.TrimSpace(state.LastReplayError) == "" {
		return html.Div(html.Props{}, metricRow("State", "none"))
	}

	rows := []ui.Node{
		metricRow("Workers", fmt.Sprintf("%d", len(state.Workers))),
		metricRow("Sync events", fmt.Sprintf("%d", len(state.SyncEvents))),
		metricRow("Replay entries", fmt.Sprintf("%d", len(state.Replay))),
		metricRow("Queue entries", fmt.Sprintf("%d", len(state.QueueEntries))),
		metricRow("Sync health", fmt.Sprintf("%d", len(state.SyncHealth))),
	}
	if len(state.Workers) > 0 {
		summaries := make([]string, 0, len(state.Workers))
		for _, worker := range state.Workers {
			summaries = append(summaries, fmt.Sprintf("%s status=%s running=%t ready=%t cancelled=%t error=%s", emptyFallback(worker.Name, "worker"), emptyFallback(worker.Status, "unknown"), worker.Running, worker.Ready, worker.Cancelled, emptyFallback(worker.Error, "none")))
		}
		rows = append(rows, metricRow("Worker jobs", strings.Join(summaries, "; ")))
	}
	if len(state.SyncEvents) > 0 {
		summaries := make([]string, 0, len(state.SyncEvents))
		for _, event := range state.SyncEvents {
			summaries = append(summaries, fmt.Sprintf("%s %s topic=%s target=%s status=%s", emptyFallback(event.Transport, "sync"), emptyFallback(event.Direction, "unknown"), emptyFallback(event.Topic, emptyFallback(event.Channel, "unknown")), emptyFallback(event.Target, "-"), emptyFallback(event.Status, "observed")))
		}
		rows = append(rows, metricRow("Sync", strings.Join(summaries, "; ")))
	}
	if len(state.Replay) > 0 {
		summaries := make([]string, 0, len(state.Replay))
		for _, entry := range state.Replay {
			summaries = append(summaries, fmt.Sprintf("%s owner=%s state=%s attempts=%d/%d next=%s error=%s", emptyFallback(entry.Kind, entry.ID), emptyFallback(entry.Owner, "n/a"), emptyFallback(entry.State, "queued"), entry.Attempts, entry.MaxAttempts, formatTime(entry.NextAttemptAt), emptyFallback(entry.LastError, "none")))
		}
		rows = append(rows, metricRow("Replay", strings.Join(summaries, "; ")))
	}
	if len(state.QueueEntries) > 0 {
		summaries := make([]string, 0, len(state.QueueEntries))
		for _, entry := range state.QueueEntries {
			summaries = append(summaries, fmt.Sprintf("%s op=%s owner=%s state=%s attempts=%d/%d queued=%s error=%s", emptyFallback(entry.Entity, entry.ID), emptyFallback(entry.Operation, "mutation"), emptyFallback(entry.Owner, "n/a"), emptyFallback(entry.State, "queued"), entry.Attempts, entry.MaxAttempts, formatTime(entry.QueuedAt), emptyFallback(entry.LastError, "none")))
		}
		rows = append(rows, metricRow("Queue", strings.Join(summaries, "; ")))
	}
	if len(state.SyncHealth) > 0 {
		summaries := make([]string, 0, len(state.SyncHealth))
		for _, entry := range state.SyncHealth {
			summaries = append(summaries, fmt.Sprintf("%s owner=%s status=%s pending=%d version=%s synced=%s error=%s", emptyFallback(entry.Entity, "entity"), emptyFallback(entry.Owner, "n/a"), emptyFallback(entry.Status, "unknown"), entry.PendingOps, emptyFallback(entry.Version, "n/a"), formatTime(entry.LastSyncAt), emptyFallback(entry.LastError, "none")))
		}
		rows = append(rows, metricRow("Health", strings.Join(summaries, "; ")))
	}
	if hasReconnect {
		rows = append(rows, metricRow("Reconnect", fmt.Sprintf("%s transport=%s connected=%t attempts=%d/%d next=%s since=%s", emptyFallback(state.Reconnect.State, "unknown"), emptyFallback(state.Reconnect.Transport, "n/a"), state.Reconnect.IsConnected, state.Reconnect.Attempts, state.Reconnect.MaxAttempts, formatTime(state.Reconnect.NextRetryAt), formatTime(state.Reconnect.LastChange))))
	}
	if hasConflict {
		rows = append(rows, metricRow("Conflict", fmt.Sprintf("%s owner=%s status=%s strategy=%s since=%s error=%s", emptyFallback(state.Conflict.Entity, "n/a"), emptyFallback(state.Conflict.Owner, "n/a"), emptyFallback(state.Conflict.Status, "none"), emptyFallback(state.Conflict.Strategy, "n/a"), formatTime(state.Conflict.DetectedAt), emptyFallback(state.Conflict.LastError, "none"))))
	}
	if strings.TrimSpace(state.LastReplayError) != "" {
		rows = append(rows, metricRow("Last replay error", state.LastReplayError))
	}
	return html.Div(html.Props{}, rows...)
}

func renderExtensionSections(sections []ExtensionSection) []ui.Node {
	if len(sections) == 0 {
		return nil
	}
	nodes := make([]ui.Node, 0, len(sections))
	for _, sectionState := range sections {
		current := sectionState
		nodes = append(nodes, section(emptyFallback(current.Name, "Extension"), extensionSectionSummary(current)))
	}
	return nodes
}

func extensionSectionSummary(sectionState ExtensionSection) ui.Node {
	rows := make([]ui.Node, 0, len(sectionState.Summary)+len(sectionState.Lines))
	if len(sectionState.Summary) > 0 {
		keys := make([]string, 0, len(sectionState.Summary))
		for key := range sectionState.Summary {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			rows = append(rows, metricRow(key, sectionState.Summary[key]))
		}
	}
	for _, line := range sectionState.Lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rows = append(rows, html.Small(html.Props{Style: map[string]string{
			"display":    "block",
			"margin-top": "4px",
			"color":      "#cbd5e1",
		}}, html.Text(line)))
	}
	if len(rows) == 0 {
		return html.Div(html.Props{}, metricRow("State", "empty"))
	}
	return html.Div(html.Props{}, rows...)
}

func profilingSummary(profiling Profiling) ui.Node {
	children := []ui.Node{
		metricRow("Render calls", fmt.Sprintf("%d", profiling.RenderCalls)),
		metricRow("Root updates", fmt.Sprintf("%d", profiling.ScheduledRootUpdates)),
		metricRow("Fiber marks", fmt.Sprintf("%d", profiling.ScheduledFiberMarks)),
		metricRow("Granular marks", fmt.Sprintf("%d", profiling.ScheduledGranularMarks)),
		metricRow("Work loops", fmt.Sprintf("%d", profiling.WorkLoopPasses)),
		metricRow("Units processed", fmt.Sprintf("%d", profiling.ProcessedUnits)),
		metricRow("Commits", fmt.Sprintf("%d", profiling.CommitCount)),
		metricRow("Granular commits", fmt.Sprintf("%d", profiling.FineGrainedCommits)),
		metricRow("Region host commits", fmt.Sprintf("%d", profiling.FineGrainedDescendantHostCommits)),
		metricRow("Region text commits", fmt.Sprintf("%d", profiling.FineGrainedDescendantTextCommits)),
		metricRow("Effects run", fmt.Sprintf("%d", profiling.EffectExecutions)),
		metricRow("Cleanups run", fmt.Sprintf("%d", profiling.CleanupExecutions)),
		metricRow("Last render", formatDurationNs(profiling.LastRenderDurationNs)),
		metricRow("Last commit", formatDurationNs(profiling.LastCommitDurationNs)),
		metricRow("Last effect", formatDurationNs(profiling.LastEffectDurationNs)),
		metricRow("Last cleanup", formatDurationNs(profiling.LastCleanupDurationNs)),
		metricRow("Total render", formatDurationNs(profiling.PhaseTotals.RenderDurationNs)),
		metricRow("Total diff", formatDurationNs(profiling.PhaseTotals.DiffDurationNs)),
		metricRow("Total commit", formatDurationNs(profiling.PhaseTotals.CommitDurationNs)),
		metricRow("Total effect", formatDurationNs(profiling.PhaseTotals.EffectDurationNs)),
		metricRow("Total cleanup", formatDurationNs(profiling.PhaseTotals.CleanupDurationNs)),
	}
	if len(profiling.RecentEvents) > 0 {
		children = append(children, profilingEventsSummary(profiling.RecentEvents))
	}
	if strings.TrimSpace(profiling.Startup.Mode) != "" || profiling.Startup.FirstInteractionCaptured || profiling.Startup.BootstrapReadDurationNs > 0 || profiling.Startup.HydrationDurationNs > 0 {
		children = append(children, startupProfilingSummary(profiling.Startup))
	}
	if len(profiling.ComponentRenders) > 0 {
		children = append(children, componentRenderSummary(profiling.ComponentRenders))
	}
	if len(profiling.FlamegraphFrames) > 0 {
		children = append(children, flamegraphSummary(profiling.FlamegraphFrames))
	}
	if len(profiling.HotBranches) > 0 {
		children = append(children, hotBranchesSummary(profiling.HotBranches))
	}
	return html.Div(html.Props{}, children...)
}

func hydrationSummary(hydration HydrationDebug) ui.Node {
	if strings.TrimSpace(hydration.CorrelationID) == "" &&
		strings.TrimSpace(hydration.StartedAt) == "" &&
		hydration.DurationNs <= 0 &&
		hydration.ExistingDOMNodeCount == 0 &&
		hydration.FallbackCount == 0 &&
		hydration.MismatchCount == 0 &&
		hydration.DiscardedNodeCount == 0 &&
		!hydration.Strict &&
		!hydration.Failed &&
		len(hydration.RecentMessages) == 0 {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No hydration activity captured yet."))
	}

	rows := []ui.Node{
		metricRow("Correlation", emptyFallback(hydration.CorrelationID, "n/a")),
		metricRow("Started", emptyFallback(hydration.StartedAt, "n/a")),
		metricRow("Finished", emptyFallback(hydration.FinishedAt, "n/a")),
		metricRow("Duration", formatDurationNs(hydration.DurationNs)),
		metricRow("Existing DOM", fmt.Sprintf("%d", hydration.ExistingDOMNodeCount)),
		metricRow("Fallbacks", fmt.Sprintf("%d", hydration.FallbackCount)),
		metricRow("Mismatches", fmt.Sprintf("%d", hydration.MismatchCount)),
		metricRow("Discarded", fmt.Sprintf("%d", hydration.DiscardedNodeCount)),
		metricRow("Strict", fmt.Sprintf("%t", hydration.Strict)),
		metricRow("Failed", fmt.Sprintf("%t", hydration.Failed)),
	}
	if strings.TrimSpace(hydration.Failure) != "" {
		rows = append(rows, metricRow("Failure", hydration.Failure))
	}
	if len(hydration.RecentMessages) > 0 {
		rows = append(rows, metricRow("Recent", strings.Join(hydration.RecentMessages, " | ")))
	}
	return html.Div(html.Props{}, rows...)
}

func profilingEventsSummary(events []ProfilingEvent) ui.Node {
	items := make([]ui.Node, 0, 4)
	items = append(items, html.Div(html.Props{Style: map[string]string{
		"margin-top":     "10px",
		"margin-bottom":  "8px",
		"font-size":      "12px",
		"text-transform": "uppercase",
		"letter-spacing": "0.08em",
		"color":          "#67e8f9",
	}}, html.Text("Recent events")))
	start := 0
	if len(events) > 3 {
		start = len(events) - 3
	}
	for index := len(events) - 1; index >= start; index-- {
		event := events[index]
		label := strings.TrimSpace(event.Domain) + "." + strings.TrimSpace(event.Name) + ":" + strings.TrimSpace(event.Phase)
		target := emptyFallback(event.Target, "-")
		duration := formatDurationNs(event.DurationNs)
		items = append(items, html.Small(html.Props{Style: map[string]string{
			"display":    "block",
			"margin-top": "4px",
			"color":      "#cbd5e1",
		}}, html.Text(label+" target="+target+" duration="+duration)))
	}
	return html.Div(html.Props{}, items...)
}

func startupProfilingSummary(startup StartupProfiling) ui.Node {
	items := []ui.Node{
		html.Div(html.Props{Style: map[string]string{
			"margin-top":     "10px",
			"margin-bottom":  "8px",
			"font-size":      "12px",
			"text-transform": "uppercase",
			"letter-spacing": "0.08em",
			"color":          "#67e8f9",
		}}, html.Text("Startup workflow")),
		metricRow("Mode", emptyFallback(startup.Mode, "n/a")),
		metricRow("Started", emptyFallback(startup.StartedAt, "n/a")),
		metricRow("Bootstrap read", formatDurationNs(startup.BootstrapReadDurationNs)),
		metricRow("WASM transfer", formatByteCount(startup.WASMTransferBytes)),
		metricRow("WASM decoded", formatByteCount(startup.WASMDecodedBytes)),
		metricRow("Bootstrap decoded", formatByteCount(startup.BootstrapDecodedBytes)),
		metricRow("Cache warmup", formatDurationNs(startup.CacheWarmupDurationNs)),
		metricRow("Service worker", formatDurationNs(startup.ServiceWorkerOverheadNs)),
		metricRow("Initial route data", formatByteCount(startup.InitialRouteDataBytes)),
		metricRow("Hydration", formatDurationNs(startup.HydrationDurationNs)),
		metricRow("First commit", formatDurationNs(startup.StartupCommitDurationNs)),
		metricRow("First interaction", formatDurationNs(startup.FirstInteractionDurationNs)),
		metricRow("Interaction captured", fmt.Sprintf("%t", startup.FirstInteractionCaptured)),
	}
	if strings.TrimSpace(startup.FirstInteractionEvent) != "" {
		items = append(items, metricRow("Interaction event", startup.FirstInteractionEvent))
	}
	if len(startup.RouteBudgets) > 0 {
		items = append(items, html.Div(html.Props{Style: map[string]string{
			"margin-top":     "10px",
			"margin-bottom":  "8px",
			"font-size":      "12px",
			"text-transform": "uppercase",
			"letter-spacing": "0.08em",
			"color":          "#67e8f9",
		}}, html.Text("Route startup budgets")))
		limit := len(startup.RouteBudgets)
		if limit > 5 {
			limit = 5
		}
		for index := 0; index < limit; index++ {
			budget := startup.RouteBudgets[index]
			items = append(items, html.Div(html.Props{Style: map[string]string{
				"padding":       "8px 10px",
				"border-radius": "10px",
				"border":        "1px solid rgba(51,65,85,0.7)",
				"margin-bottom": "8px",
			}},
				html.Small(html.Props{Style: map[string]string{"display": "block", "color": "#94a3b8"}}, html.Text("family="+emptyFallback(budget.RouteFamily, "n/a")+" sample="+fmt.Sprintf("%d", budget.SampleCount))),
				html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("path="+emptyFallback(budget.LastRoutePath, "n/a"))),
				html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("bootstrap="+formatDurationNs(budget.AverageBootstrapReadDurationNs)+" hydration="+formatDurationNs(budget.AverageHydrationDurationNs)+" commit="+formatDurationNs(budget.AverageStartupCommitDurationNs)+" first-interaction="+formatDurationNs(budget.AverageFirstInteractionDurationNs))),
				html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("wasm="+formatByteCount(budget.AverageWASMTransferBytes)+" decoded="+formatByteCount(budget.AverageWASMDecodedBytes)+" bootstrap="+formatByteCount(budget.AverageBootstrapDecodedBytes)+" route-data="+formatByteCount(budget.AverageInitialRouteDataBytes))),
				html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("cache="+formatDurationNs(budget.AverageCacheWarmupDurationNs)+" service-worker="+formatDurationNs(budget.AverageServiceWorkerOverheadNs))),
			))
		}
	}
	return html.Div(html.Props{}, items...)
}

func componentRenderSummary(traces []ComponentRenderTrace) ui.Node {
	items := make([]ui.Node, 0, 6)
	items = append(items, html.Div(html.Props{Style: map[string]string{
		"margin-top":     "10px",
		"margin-bottom":  "8px",
		"font-size":      "12px",
		"text-transform": "uppercase",
		"letter-spacing": "0.08em",
		"color":          "#67e8f9",
	}}, html.Text("Component rerenders")))
	limit := len(traces)
	if limit > 5 {
		limit = 5
	}
	for index := 0; index < limit; index++ {
		trace := traces[index]
		items = append(items, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 10px",
			"border-radius": "10px",
			"border":        "1px solid rgba(51,65,85,0.7)",
			"margin-bottom": "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "gap": "8px", "align-items": "baseline"}},
				html.Strong(html.Props{Style: map[string]string{"color": "#f8fafc"}}, html.Text(emptyFallback(trace.Name, "Component"))),
				html.Code(html.Props{Style: map[string]string{"color": "#67e8f9"}}, html.Text(fmt.Sprintf("renders=%d rerenders=%d", trace.RenderCount, trace.RerenderCount))),
			),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text(emptyFallback(trace.Path, "path unavailable"))),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("trigger="+emptyFallback(trace.LastTrigger, "unknown")+" avg="+formatDurationNs(trace.AverageRenderDurationNs)+" last="+formatDurationNs(trace.LastRenderDurationNs))),
		))
	}
	return html.Div(html.Props{}, items...)
}

func flamegraphSummary(frames []FlamegraphFrame) ui.Node {
	if len(frames) == 0 {
		return nil
	}
	items := make([]ui.Node, 0, 14)
	items = append(items, html.Div(html.Props{Style: map[string]string{
		"margin-top":     "10px",
		"margin-bottom":  "8px",
		"font-size":      "12px",
		"text-transform": "uppercase",
		"letter-spacing": "0.08em",
		"color":          "#67e8f9",
	}}, html.Text("Flamegraph capture")))
	maxEnd := int64(0)
	for _, frame := range frames {
		end := frame.StartNs + frame.DurationNs
		if end > maxEnd {
			maxEnd = end
		}
	}
	if maxEnd <= 0 {
		maxEnd = 1
	}
	limit := len(frames)
	if limit > 12 {
		limit = 12
	}
	for index := 0; index < limit; index++ {
		frame := frames[index]
		leftPct := (float64(frame.StartNs) / float64(maxEnd)) * 100
		widthPct := (float64(frame.DurationNs) / float64(maxEnd)) * 100
		if widthPct < 3 {
			widthPct = 3
		}
		color := "#38bdf8"
		if frame.Depth%3 == 1 {
			color = "#22d3ee"
		} else if frame.Depth%3 == 2 {
			color = "#34d399"
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{
			"position":      "relative",
			"height":        "26px",
			"margin-bottom": "6px",
			"border-radius": "8px",
			"background":    "rgba(15,23,42,0.6)",
			"overflow":      "hidden",
		}},
			html.Div(html.Props{Style: map[string]string{
				"position":        "absolute",
				"left":            fmt.Sprintf("%.2f%%", leftPct),
				"width":           fmt.Sprintf("%.2f%%", widthPct),
				"height":          "100%",
				"background":      color,
				"opacity":         "0.35",
				"border":          "1px solid rgba(125,211,252,0.35)",
				"border-radius":   "8px",
				"display":         "flex",
				"align-items":     "center",
				"justify-content": "space-between",
				"gap":             "6px",
				"padding":         "0 8px",
			}},
				html.Small(html.Props{Style: map[string]string{"color": "#f8fafc", "white-space": "nowrap", "overflow": "hidden", "text-overflow": "ellipsis"}}, html.Text(emptyFallback(frame.Name, "node"))),
				html.Small(html.Props{Style: map[string]string{"color": "#e2e8f0", "white-space": "nowrap"}}, html.Text(formatDurationNs(frame.DurationNs))),
			),
		))
	}
	return html.Div(html.Props{}, items...)
}

func hotBranchesSummary(branches []Branch) ui.Node {
	items := make([]ui.Node, 0, len(branches)+1)
	items = append(items, html.Div(html.Props{Style: map[string]string{
		"margin-top":     "10px",
		"margin-bottom":  "8px",
		"font-size":      "12px",
		"text-transform": "uppercase",
		"letter-spacing": "0.08em",
		"color":          "#67e8f9",
	}}, html.Text("Hot branches")))
	for _, branch := range branches {
		items = append(items, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 10px",
			"border-radius": "10px",
			"border":        "1px solid rgba(51,65,85,0.7)",
			"margin-bottom": "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "gap": "8px", "align-items": "baseline"}},
				html.Strong(html.Props{Style: map[string]string{"color": "#f8fafc"}}, html.Text(branch.Name)),
				html.Code(html.Props{Style: map[string]string{"color": "#67e8f9"}}, html.Text(formatDurationNs(branch.SubtreeDurationNs))),
			),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text(branch.Path)),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("self="+formatDurationNs(branch.SelfDurationNs)+" render="+formatDurationNs(branch.RenderDurationNs)+" diff="+formatDurationNs(branch.DiffDurationNs)+" commit="+formatDurationNs(branch.CommitDurationNs)+" effect="+formatDurationNs(branch.EffectDurationNs)+" cleanup="+formatDurationNs(branch.CleanupDurationNs))),
		))
	}
	return html.Div(html.Props{}, items...)
}

func diagnosticsSummary(diagnostics []Diagnostic) ui.Node {
	if len(diagnostics) == 0 {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No diagnostics reported."))
	}

	items := make([]ui.Node, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		color := "#f8fafc"
		switch diagnostic.Severity {
		case SeverityWarning:
			color = "#fde68a"
		case SeverityError:
			color = "#fda4af"
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 10px",
			"border-radius": "10px",
			"border":        "1px solid rgba(51,65,85,0.7)",
			"margin-bottom": "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"font-size": "12px", "text-transform": "uppercase", "letter-spacing": "0.08em", "color": color}}, html.Text(string(diagnostic.Severity)+" • "+diagnostic.Source+" • count="+fmt.Sprintf("%d", diagnostic.Count))),
			func() ui.Node {
				if strings.TrimSpace(diagnostic.Code) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#67e8f9"}}, html.Text("code: "+diagnostic.Code))
			}(),
			html.P(html.Props{Style: map[string]string{"margin": "6px 0 0 0", "color": "#cbd5e1"}}, html.Text(diagnostic.Message)),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text("class: "+string(diagnostic.Classification)+" | recoverable="+fmt.Sprintf("%t", diagnostic.Recoverable))),
			func() ui.Node {
				if strings.TrimSpace(diagnostic.TopFrame) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "6px", "color": "#cbd5e1"}}, html.Text("where: "+diagnostic.TopFrame))
			}(),
			func() ui.Node {
				if strings.TrimSpace(diagnostic.Path) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "6px", "color": "#94a3b8"}}, html.Text("path: "+diagnostic.Path))
			}(),
			func() ui.Node {
				if strings.TrimSpace(diagnostic.Consequence) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("runtime: "+diagnostic.Consequence))
			}(),
			func() ui.Node {
				if len(diagnostic.ComponentStack) == 0 {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("stack: "+strings.Join(diagnostic.ComponentStack, " > ")))
			}(),
			func() ui.Node {
				if strings.TrimSpace(diagnostic.Remediation) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "6px", "color": "#cbd5e1"}}, html.Text("next step: "+diagnostic.Remediation))
			}(),
			func() ui.Node {
				if strings.TrimSpace(diagnostic.Docs) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text("docs: "+diagnostic.Docs))
			}(),
		))
	}
	return html.Div(html.Props{}, items...)
}

func logsSummary(entries []Log) ui.Node {
	if len(entries) == 0 {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No framework logs buffered."))
	}

	items := make([]ui.Node, 0, len(entries))
	for _, entry := range entries {
		color := "#cbd5e1"
		switch entry.Level {
		case LogLevel(runtime.LogWarn):
			color = "#fde68a"
		case LogLevel(runtime.LogError):
			color = "#fda4af"
		case LogLevel(runtime.LogInfo):
			color = "#67e8f9"
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{
			"padding":       "8px 10px",
			"border-radius": "10px",
			"border":        "1px solid rgba(51,65,85,0.7)",
			"margin-bottom": "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"font-size": "12px", "text-transform": "uppercase", "letter-spacing": "0.08em", "color": color}}, html.Text(string(entry.Level)+" | "+entry.Domain+" | "+string(entry.Classification))),
			func() ui.Node {
				if strings.TrimSpace(entry.Code) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#67e8f9"}}, html.Text("code: "+entry.Code+" | recoverable="+fmt.Sprintf("%t", entry.Recoverable)))
			}(),
			html.P(html.Props{Style: map[string]string{"margin": "6px 0 0 0", "color": "#e2e8f0"}}, html.Text(entry.Message)),
			func() ui.Node {
				meta := []string{}
				if strings.TrimSpace(entry.Timestamp) != "" {
					meta = append(meta, entry.Timestamp)
				}
				if strings.TrimSpace(entry.CorrelationID) != "" {
					meta = append(meta, "corr="+entry.CorrelationID)
				}
				if len(meta) == 0 {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text(strings.Join(meta, " | ")))
			}(),
			func() ui.Node {
				if strings.TrimSpace(entry.TopFrame) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("where: "+entry.TopFrame))
			}(),
			func() ui.Node {
				if strings.TrimSpace(entry.Consequence) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("runtime: "+entry.Consequence))
			}(),
			func() ui.Node {
				if len(entry.Fields) == 0 {
					return nil
				}
				keys := make([]string, 0, len(entry.Fields))
				for key := range entry.Fields {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				parts := make([]string, 0, len(keys))
				for _, key := range keys {
					parts = append(parts, key+"="+entry.Fields[key])
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text(strings.Join(parts, " | ")))
			}(),
			func() ui.Node {
				if strings.TrimSpace(entry.Remediation) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("next step: "+entry.Remediation))
			}(),
			func() ui.Node {
				if strings.TrimSpace(entry.Docs) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#94a3b8"}}, html.Text("docs: "+entry.Docs))
			}(),
		))
	}
	return html.Div(html.Props{}, items...)
}

func selectedNodeSummary(snapshot Snapshot, selectedPath string) ui.Node {
	if snapshot.Tree == nil {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No committed tree yet."))
	}
	node := findNodeByPath(snapshot.Tree, selectedPath)
	if node == nil {
		node = snapshot.Tree
	}
	rows := []ui.Node{
		metricRow("Node", emptyFallback(node.Name, "unknown")),
		metricRow("Path", emptyFallback(node.Path, "unknown")),
		metricRow("Kind", emptyFallback(node.Kind, "unknown")),
		metricRow("Hooks", fmt.Sprintf("%d", node.HookCount)),
		metricRow("Effects", fmt.Sprintf("%d", node.EffectCount)),
		metricRow("Route", emptyFallback(snapshot.Route.Path, "/")),
	}
	if len(snapshot.Route.Stack) > 0 {
		rows = append(rows, metricRow("Route stack", emptyFallback(snapshot.Route.Stack[len(snapshot.Route.Stack)-1].Path, snapshot.Route.Path)))
	}
	if strings.TrimSpace(node.Signature) != "" {
		rows = append(rows, metricRow("Signature", node.Signature))
	}
	if strings.TrimSpace(node.UpdateOrigin) != "" || strings.TrimSpace(node.ReactiveSource) != "" {
		meta := []string{}
		if strings.TrimSpace(node.UpdateOrigin) != "" {
			meta = append(meta, "origin="+node.UpdateOrigin)
		}
		if strings.TrimSpace(node.ReactiveSource) != "" {
			meta = append(meta, "source="+node.ReactiveSource)
		}
		rows = append(rows, metricRow("Reactive", strings.Join(meta, " | ")))
	}
	if len(node.Hooks) > 0 {
		hooks := make([]string, 0, len(node.Hooks))
		for _, hook := range node.Hooks {
			detail := hook.Kind + "#" + fmt.Sprintf("%d", hook.Slot) + "=" + hook.Value
			if strings.TrimSpace(hook.Status) != "" {
				detail += " (" + hook.Status + ")"
			}
			hooks = append(hooks, detail)
		}
		rows = append(rows, metricRow("State", strings.Join(hooks, " | ")))
	}
	if matches := cacheEntriesForNode(node, snapshot.Cache); len(matches) > 0 {
		summaries := make([]string, 0, len(matches))
		for _, entry := range matches {
			summaries = append(summaries, fmt.Sprintf("%s ready=%t stale=%t subscribers=%d", entry.Key, entry.Ready, entry.Stale, entry.SubscriberCount))
		}
		rows = append(rows, metricRow("Cache", strings.Join(summaries, "; ")))
	}
	return html.Div(html.Props{}, rows...)
}

func treeSummary(node *Node, depth int, maxDepth int, selectedPath string, selectNode func(string)) ui.Node {
	if node == nil {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No committed tree yet."))
	}
	return renderNode(*node, depth, maxDepth, selectedPath, selectNode)
}

func renderNode(node Node, depth int, maxDepth int, selectedPath string, selectNode func(string)) ui.Node {
	children := []ui.Node{
		html.Div(html.Props{OnClick: ui.WrapHandler(func() {
			if selectNode != nil {
				selectNode(node.Path)
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
				if strings.TrimSpace(node.Path) == strings.TrimSpace(selectedPath) {
					return "rgba(14,116,144,0.22)"
				}
				return "transparent"
			}(),
		}},
			html.Strong(html.Props{Style: map[string]string{"color": "#f8fafc"}}, html.Text(node.Name)),
			html.Code(html.Props{Style: map[string]string{"color": "#67e8f9", "background": "rgba(15,23,42,0.6)", "padding": "2px 6px", "border-radius": "6px"}}, html.Text(node.Kind)),
			html.Small(html.Props{Style: map[string]string{"color": "#94a3b8"}}, html.Text(fmt.Sprintf("hooks=%d effects=%d", node.HookCount, node.EffectCount))),
			func() ui.Node {
				if !node.FineGrained {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"color": "#67e8f9"}}, html.Text("fine-grained"))
			}(),
			func() ui.Node {
				if node.Dirty || node.NeedsUpdate {
					return html.Small(html.Props{Style: map[string]string{"color": "#fde68a"}}, html.Text(fmt.Sprintf("dirty=%t update=%t", node.Dirty, node.NeedsUpdate)))
				}
				return nil
			}(),
			func() ui.Node {
				if node.SubtreeDurationNs <= 0 {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"color": "#cbd5e1"}}, html.Text("subtree="+formatDurationNs(node.SubtreeDurationNs)+" self="+formatDurationNs(node.SelfDurationNs)+" render="+formatDurationNs(node.RenderDurationNs)+" diff="+formatDurationNs(node.DiffDurationNs)+" commit="+formatDurationNs(node.CommitDurationNs)))
			}(),
		),
	}

	if strings.TrimSpace(node.Signature) != "" {
		children = append(children, html.Small(html.Props{Style: map[string]string{
			"display":    "block",
			"margin-top": "4px",
			"color":      "#cbd5e1",
		}}, html.Text("signature: "+node.Signature)))
	}

	if node.FineGrained || strings.TrimSpace(node.UpdateOrigin) != "" || strings.TrimSpace(node.ReactiveSource) != "" {
		meta := make([]string, 0, 3)
		if node.FineGrained {
			meta = append(meta, "mode=fine-grained")
		}
		if strings.TrimSpace(node.UpdateOrigin) != "" {
			meta = append(meta, "origin="+node.UpdateOrigin)
		}
		if strings.TrimSpace(node.ReactiveSource) != "" {
			meta = append(meta, "source="+node.ReactiveSource)
		}
		children = append(children, html.Small(html.Props{Style: map[string]string{
			"display":    "block",
			"margin-top": "4px",
			"color":      "#67e8f9",
		}}, html.Text(strings.Join(meta, " | "))))
	}

	if len(node.Hooks) > 0 {
		hookNodes := make([]ui.Node, 0, len(node.Hooks))
		for _, hook := range node.Hooks {
			detail := "#" + fmt.Sprintf("%d", hook.Slot) + ": " + hook.Value
			if strings.TrimSpace(hook.Dependencies) != "" {
				detail += " | deps=" + hook.Dependencies
			}
			if strings.TrimSpace(hook.Status) != "" {
				detail += " | " + hook.Status
			}
			hookNodes = append(hookNodes, html.Div(html.Props{Style: map[string]string{"margin-top": "6px", "color": "#cbd5e1"}},
				html.Code(html.Props{Style: map[string]string{"color": "#67e8f9"}}, html.Text(hook.Kind)),
				html.Text(": "+detail),
			))
		}
		children = append(children, html.Div(html.Props{Style: map[string]string{"margin-top": "6px"}}, hookNodes...))
	}

	if depth < maxDepth && len(node.Children) > 0 {
		childNodes := make([]ui.Node, 0, len(node.Children))
		for _, child := range node.Children {
			childNodes = append(childNodes, renderNode(child, depth+1, maxDepth, selectedPath, selectNode))
		}
		children = append(children, html.Div(html.Props{Style: map[string]string{
			"margin-top":   "8px",
			"margin-left":  "14px",
			"padding-left": "10px",
			"border-left":  "1px solid rgba(51,65,85,0.7)",
		}}, childNodes...))
	} else if depth >= maxDepth && len(node.Children) > 0 {
		children = append(children, html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "8px", "color": "#94a3b8"}}, html.Text(fmt.Sprintf("%d child nodes hidden at max depth", len(node.Children)))))
	}

	return html.Div(html.Props{Style: map[string]string{
		"padding":       "8px 0",
		"border-bottom": "1px solid rgba(30,41,59,0.8)",
	}}, children...)
}

func findNodeByPath(node *Node, path string) *Node {
	if node == nil {
		return nil
	}
	if strings.TrimSpace(node.Path) == strings.TrimSpace(path) {
		return node
	}
	for index := range node.Children {
		if found := findNodeByPath(&node.Children[index], path); found != nil {
			return found
		}
	}
	return nil
}

func cacheEntriesForNode(node *Node, entries []CacheEntry) []CacheEntry {
	if node == nil || len(entries) == 0 {
		return nil
	}
	matches := make([]CacheEntry, 0, len(entries))
	for _, entry := range entries {
		for _, ownerPath := range entry.OwnerPaths {
			if strings.TrimSpace(ownerPath) == strings.TrimSpace(node.Path) {
				matches = append(matches, entry)
				break
			}
		}
	}
	return matches
}

func metricRow(label, value string) ui.Node {
	return html.Div(html.Props{Style: map[string]string{
		"display":         "flex",
		"justify-content": "space-between",
		"gap":             "10px",
		"margin-bottom":   "6px",
	}},
		html.Small(html.Props{Style: map[string]string{"color": "#94a3b8"}}, html.Text(label)),
		html.Code(html.Props{Style: map[string]string{"color": "#f8fafc", "text-align": "right"}}, html.Text(value)),
	)
}

func mapNode(node *runtime.FiberSnapshot) *Node {
	if node == nil {
		return nil
	}
	mapped := &Node{
		Name:              node.Name,
		Path:              node.Path,
		Kind:              node.Kind,
		Dirty:             node.Dirty,
		NeedsUpdate:       node.NeedsUpdate,
		FineGrained:       node.FineGrained,
		ReactiveSource:    node.ReactiveSource,
		UpdateOrigin:      node.UpdateOrigin,
		EffectCount:       node.EffectCount,
		HookCount:         node.HookCount,
		Signature:         "",
		RenderDurationNs:  node.RenderDurationNs,
		DiffDurationNs:    node.DiffDurationNs,
		CommitDurationNs:  node.CommitDurationNs,
		EffectDurationNs:  node.EffectDurationNs,
		CleanupDurationNs: node.CleanupDurationNs,
		SelfDurationNs:    node.SelfDurationNs,
		SubtreeDurationNs: node.SubtreeDurationNs,
	}
	if node.Signature != nil {
		mapped.Signature = node.Signature.Summary()
	}
	for _, hook := range node.Hooks {
		mapped.Hooks = append(mapped.Hooks, Hook{
			Slot:         hook.Slot,
			Kind:         hook.Kind,
			Value:        hook.Value,
			Dependencies: hook.Dependencies,
			Status:       hook.Status,
		})
	}
	for index := range node.Children {
		mapped.Children = append(mapped.Children, *mapNode(&node.Children[index]))
	}
	return mapped
}

func mapStats(stats runtime.InspectionStats) Stats {
	return Stats{
		TotalFibers:       stats.TotalFibers,
		DirtyFibers:       stats.DirtyFibers,
		ComponentFibers:   stats.ComponentFibers,
		HostFibers:        stats.HostFibers,
		TextFibers:        stats.TextFibers,
		FineGrainedFibers: stats.FineGrainedFibers,
		HookEntries:       stats.HookEntries,
		Effects:           stats.Effects,
	}
}

func mapProfiling(profiling runtime.ProfilingSnapshot) Profiling {
	mapped := Profiling{
		RenderCalls:                      profiling.RenderCalls,
		ScheduledRootUpdates:             profiling.ScheduledRootUpdates,
		ScheduledFiberMarks:              profiling.ScheduledFiberMarks,
		ScheduledGranularMarks:           profiling.ScheduledGranularMarks,
		WorkLoopPasses:                   profiling.WorkLoopPasses,
		ProcessedUnits:                   profiling.ProcessedUnits,
		CommitCount:                      profiling.CommitCount,
		FineGrainedCommits:               profiling.FineGrainedCommits,
		FineGrainedDescendantHostCommits: profiling.FineGrainedDescendantHostCommits,
		FineGrainedDescendantTextCommits: profiling.FineGrainedDescendantTextCommits,
		EffectExecutions:                 profiling.EffectExecutions,
		CleanupExecutions:                profiling.CleanupExecutions,
		LastRenderDurationNs:             profiling.LastRenderDurationNs,
		LastCommitDurationNs:             profiling.LastCommitDurationNs,
		LastEffectDurationNs:             profiling.LastEffectDurationNs,
		LastCleanupDurationNs:            profiling.LastCleanupDurationNs,
		PhaseTotals: ProfilingPhaseTotals{
			RenderDurationNs:  profiling.PhaseTotals.RenderDurationNs,
			DiffDurationNs:    profiling.PhaseTotals.DiffDurationNs,
			CommitDurationNs:  profiling.PhaseTotals.CommitDurationNs,
			EffectDurationNs:  profiling.PhaseTotals.EffectDurationNs,
			CleanupDurationNs: profiling.PhaseTotals.CleanupDurationNs,
		},
		Startup: StartupProfiling{
			Mode:                       profiling.Startup.Mode,
			StartedAt:                  profiling.Startup.StartedAt,
			BootstrapReadDurationNs:    profiling.Startup.BootstrapReadDurationNs,
			WASMTransferBytes:          profiling.Startup.WASMTransferBytes,
			WASMDecodedBytes:           profiling.Startup.WASMDecodedBytes,
			BootstrapDecodedBytes:      profiling.Startup.BootstrapDecodedBytes,
			CacheWarmupDurationNs:      profiling.Startup.CacheWarmupDurationNs,
			ServiceWorkerOverheadNs:    profiling.Startup.ServiceWorkerOverheadNs,
			InitialRouteDataBytes:      profiling.Startup.InitialRouteDataBytes,
			HydrationDurationNs:        profiling.Startup.HydrationDurationNs,
			StartupCommitDurationNs:    profiling.Startup.StartupCommitDurationNs,
			FirstInteractionDurationNs: profiling.Startup.FirstInteractionDurationNs,
			FirstInteractionCaptured:   profiling.Startup.FirstInteractionCaptured,
			FirstInteractionEvent:      profiling.Startup.FirstInteractionEvent,
		},
	}
	for _, budget := range profiling.Startup.RouteBudgets {
		mapped.Startup.RouteBudgets = append(mapped.Startup.RouteBudgets, RouteStartupBudget{
			RouteFamily:                       budget.RouteFamily,
			LastRoutePath:                     budget.LastRoutePath,
			SampleCount:                       budget.SampleCount,
			AverageBootstrapReadDurationNs:    budget.AverageBootstrapReadDurationNs,
			AverageWASMTransferBytes:          budget.AverageWASMTransferBytes,
			AverageWASMDecodedBytes:           budget.AverageWASMDecodedBytes,
			AverageBootstrapDecodedBytes:      budget.AverageBootstrapDecodedBytes,
			AverageCacheWarmupDurationNs:      budget.AverageCacheWarmupDurationNs,
			AverageServiceWorkerOverheadNs:    budget.AverageServiceWorkerOverheadNs,
			AverageInitialRouteDataBytes:      budget.AverageInitialRouteDataBytes,
			AverageHydrationDurationNs:        budget.AverageHydrationDurationNs,
			AverageStartupCommitDurationNs:    budget.AverageStartupCommitDurationNs,
			AverageFirstInteractionDurationNs: budget.AverageFirstInteractionDurationNs,
		})
	}
	for _, event := range profiling.RecentEvents {
		mapped.RecentEvents = append(mapped.RecentEvents, ProfilingEvent{
			Domain:        event.Domain,
			Name:          event.Name,
			Phase:         event.Phase,
			Target:        event.Target,
			CorrelationID: event.CorrelationID,
			DurationNs:    event.DurationNs,
			Timestamp:     event.Timestamp,
			Fields:        cloneStringMap(event.Fields),
		})
	}
	for _, trace := range profiling.ComponentRenders {
		mapped.ComponentRenders = append(mapped.ComponentRenders, ComponentRenderTrace{
			Name:                    trace.Name,
			Path:                    trace.Path,
			RenderCount:             trace.RenderCount,
			RerenderCount:           trace.RerenderCount,
			LastTrigger:             trace.LastTrigger,
			LastRenderDurationNs:    trace.LastRenderDurationNs,
			TotalRenderDurationNs:   trace.TotalRenderDurationNs,
			AverageRenderDurationNs: trace.AverageRenderDurationNs,
			LastRenderedAt:          trace.LastRenderedAt,
			TriggerCounts:           cloneIntMap(trace.TriggerCounts),
		})
	}
	for _, frame := range profiling.FlamegraphFrames {
		mapped.FlamegraphFrames = append(mapped.FlamegraphFrames, FlamegraphFrame{
			Name:              frame.Name,
			Kind:              frame.Kind,
			Path:              frame.Path,
			Depth:             frame.Depth,
			StartNs:           frame.StartNs,
			DurationNs:        frame.DurationNs,
			SelfDurationNs:    frame.SelfDurationNs,
			RenderDurationNs:  frame.RenderDurationNs,
			DiffDurationNs:    frame.DiffDurationNs,
			CommitDurationNs:  frame.CommitDurationNs,
			EffectDurationNs:  frame.EffectDurationNs,
			CleanupDurationNs: frame.CleanupDurationNs,
		})
	}
	for _, branch := range profiling.HotBranches {
		mapped.HotBranches = append(mapped.HotBranches, Branch{
			Name:              branch.Name,
			Kind:              branch.Kind,
			Path:              branch.Path,
			RenderDurationNs:  branch.RenderDurationNs,
			DiffDurationNs:    branch.DiffDurationNs,
			CommitDurationNs:  branch.CommitDurationNs,
			EffectDurationNs:  branch.EffectDurationNs,
			CleanupDurationNs: branch.CleanupDurationNs,
			SelfDurationNs:    branch.SelfDurationNs,
			SubtreeDurationNs: branch.SubtreeDurationNs,
		})
	}
	return mapped
}

func mapHydration(hydration runtime.HydrationDebugSnapshot) HydrationDebug {
	return HydrationDebug{
		CorrelationID:        hydration.CorrelationID,
		StartedAt:            hydration.StartedAt,
		FinishedAt:           hydration.FinishedAt,
		DurationNs:           hydration.DurationNs,
		ExistingDOMNodeCount: hydration.ExistingDOMNodeCount,
		FallbackCount:        hydration.FallbackCount,
		MismatchCount:        hydration.MismatchCount,
		DiscardedNodeCount:   hydration.DiscardedNodeCount,
		Strict:               hydration.Strict,
		Failed:               hydration.Failed,
		Failure:              hydration.Failure,
		RecentMessages:       append([]string(nil), hydration.RecentMessages...),
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneIntMap(input map[string]int) map[string]int {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]int, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneParams(params map[string]string) map[string]string {
	if len(params) == 0 {
		return nil
	}
	clone := make(map[string]string, len(params))
	for key, value := range params {
		clone[key] = value
	}
	return clone
}

func mapRouteStack(entries []router.RouteStackInspection) []RouteStack {
	if len(entries) == 0 {
		return nil
	}
	stack := make([]RouteStack, 0, len(entries))
	for _, entry := range entries {
		stack = append(stack, RouteStack{
			ID:             entry.ID,
			Path:           entry.Path,
			Params:         cloneParams(entry.Params),
			HasLoader:      entry.HasLoader,
			HasBeforeEnter: entry.HasBeforeEnter,
			HasBeforeLeave: entry.HasBeforeLeave,
			Metadata: RouteMetadata{
				Title:        entry.Metadata.Title,
				Description:  entry.Metadata.Description,
				CanonicalURL: entry.Metadata.CanonicalURL,
			},
		})
	}
	return stack
}

func mapRouteLoaders(entries []router.RouteLoaderInspection) []RouteLoader {
	if len(entries) == 0 {
		return nil
	}
	loaders := make([]RouteLoader, 0, len(entries))
	for _, entry := range entries {
		loaders = append(loaders, RouteLoader{
			Key:     entry.Key,
			Path:    entry.Path,
			Pending: entry.Pending,
			HasData: entry.HasData,
			Error:   entry.Error,
		})
	}
	return loaders
}

func formatQuery(query map[string][]string) string {
	if len(query) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strings.Join(query[key], ","))
	}
	return strings.Join(parts, " & ")
}

func formatParams(params map[string]string) string {
	if len(params) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, " & ")
}

func formatStringMap(values map[string]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key])
	}
	return strings.Join(parts, " & ")
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func formatDurationNs(value int64) string {
	if value <= 0 {
		return "0ms"
	}
	return fmt.Sprintf("%.2fms", float64(value)/1_000_000)
}

func formatByteCount(value int64) string {
	if value <= 0 {
		return "0 B"
	}
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	if value < 1024*1024 {
		return fmt.Sprintf("%.2f KiB", float64(value)/1024.0)
	}
	return fmt.Sprintf("%.2f MiB", float64(value)/(1024.0*1024.0))
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "n/a"
	}
	return value.Format("15:04:05")
}
