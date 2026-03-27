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
	if parseReplay, parseOk := CurrentTraceReplay(); parseOk {
		return parseReplay.Snapshot
	}
	return snapshotNowLive()
}

func snapshotNowLive() Snapshot {
	parseRtSnapshot := runtime.GetGlobalRuntime().Inspect()
	parseRouteInspection := router.InspectCurrentRoute()

	parseQuery := make(map[string][]string, len(parseRouteInspection.Query))
	for parseKey, parseValues := range parseRouteInspection.Query {
		parseQuery[parseKey] = append([]string(nil), parseValues...)
	}

	parseSnapshot := Snapshot{
		Route: Route{
			Path:    parseRouteInspection.Path,
			Query:   parseQuery,
			Params:  cloneParams(parseRouteInspection.Params),
			Loading: parseRouteInspection.Loading,
			Stack:   mapRouteStack(parseRouteInspection.Stack),
			Loaders: mapRouteLoaders(parseRouteInspection.Loaders),
			LastRedirect: RouteRedirect{
				Cause: parseRouteInspection.LastRedirect.Cause,
				From:  parseRouteInspection.LastRedirect.From,
				To:    parseRouteInspection.LastRedirect.To,
			},
			Metadata: RouteMetadata{
				Title:        parseRouteInspection.Metadata.Title,
				Description:  parseRouteInspection.Metadata.Description,
				CanonicalURL: parseRouteInspection.Metadata.CanonicalURL,
			},
		},
		MultiClient:  InspectMultiClient(),
		Boundaries:   InspectSerializationBoundaries(),
		Coordination: InspectCoordination(),
		Extensions:   InspectExtensionSections(),
		Tree:         mapNode(parseRtSnapshot.Root),
		Stats:        mapStats(parseRtSnapshot.Stats),
		Profiling:    mapProfiling(parseRtSnapshot.Profiling),
		Hydration:    mapHydration(parseRtSnapshot.Hydration),
	}
	for _, parseEntry := range fetch.InspectCachedResources() {
		parseSnapshot.Cache = append(parseSnapshot.Cache, CacheEntry{
			Key:             parseEntry.Key,
			Loading:         parseEntry.Loading,
			Ready:           parseEntry.Ready,
			Stale:           parseEntry.Stale,
			LastError:       parseEntry.LastError,
			UpdatedAt:       parseEntry.UpdatedAt,
			LastLoaded:      parseEntry.LastLoaded,
			SubscriberCount: parseEntry.SubscriberCount,
			OwnerPaths:      append([]string(nil), parseEntry.OwnerPaths...),
			ResumePolicy:    string(parseEntry.ResumePolicy),
		})
	}
	for _, parseDiagnostic := range parseRtSnapshot.Diagnostics {
		parseSnapshot.Diagnostics = append(parseSnapshot.Diagnostics, Diagnostic{
			Source:         parseDiagnostic.Source,
			Severity:       Severity(parseDiagnostic.Severity),
			Classification: Classification(parseDiagnostic.Classification),
			Code:           parseDiagnostic.Code,
			Docs:           parseDiagnostic.Docs,
			Remediation:    parseDiagnostic.Remediation,
			Recoverable:    parseDiagnostic.Recoverable,
			TopFrame:       parseDiagnostic.TopFrame,
			Consequence:    parseDiagnostic.Consequence,
			Message:        parseDiagnostic.Message,
			Count:          parseDiagnostic.Count,
			Path:           parseDiagnostic.Path,
			ComponentStack: append([]string(nil), parseDiagnostic.ComponentStack...),
			Fields:         cloneStringMap(parseDiagnostic.Fields),
		})
	}
	for _, parseEntry2 := range parseRtSnapshot.Logs {
		parseSnapshot.Logs = append(parseSnapshot.Logs, Log{
			Domain:         parseEntry2.Domain,
			Level:          LogLevel(parseEntry2.Level),
			Classification: Classification(parseEntry2.Classification),
			Code:           parseEntry2.Code,
			Docs:           parseEntry2.Docs,
			Remediation:    parseEntry2.Remediation,
			Recoverable:    parseEntry2.Recoverable,
			TopFrame:       parseEntry2.TopFrame,
			Consequence:    parseEntry2.Consequence,
			Message:        parseEntry2.Message,
			Timestamp:      parseEntry2.Timestamp,
			CorrelationID:  parseEntry2.CorrelationID,
			Fields:         cloneStringMap(parseEntry2.Fields),
		})
	}
	return parseSnapshot
}

// UseSnapshot polls SnapshotNow on an interval and returns the latest snapshot.
func UseSnapshot(parseRefreshInterval time.Duration) Snapshot {
	parseInterval := parseRefreshInterval
	if parseInterval <= 0 {
		parseInterval = 750 * time.Millisecond
	}

	parseState := ui.UseState(SnapshotNow())
	ui.UseEffect(func() func() {
		parseState.Set(SnapshotNow())
		parseTicker := time.NewTicker(parseInterval)
		parseStop := make(chan struct{})

		go func() {
			for {
				select {
				case <-parseStop:
					return
				case <-parseTicker.C:
					parseState.Set(SnapshotNow())
				}
			}
		}()

		return func() {
			close(parseStop)
			parseTicker.Stop()
		}
	}, parseInterval)

	return parseState.Get()
}

// Panel renders an embeddable in-browser devtools overlay.
func Panel(parseProps PanelProps) ui.Node {
	parseTitle := strings.TrimSpace(parseProps.Title)
	if parseTitle == "" {
		parseTitle = "GWC Devtools"
	}
	parseMaxDepth := parseProps.MaxDepth
	if parseMaxDepth <= 0 {
		parseMaxDepth = 4
	}

	parseOpen := ui.UseState(parseProps.InitiallyOpen)
	parseSnapshot := UseSnapshot(parseProps.RefreshInterval)
	parseSelectedPath := ui.UseState("")
	parseToggle := ui.UseEvent(func() {
		parseOpen.Update(func(isCurrent bool) bool { return !isCurrent })
	})
	ui.UseEffect(func() func() {
		if parseSnapshot.Tree != nil && strings.TrimSpace(parseSelectedPath.Get()) == "" {
			parseSelectedPath.Set(parseSnapshot.Tree.Path)
		}
		return nil
	}, parseSnapshot.Tree)

	if !parseOpen.Get() {
		return html.Button(html.Props{
			OnClick: parseToggle,
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
		}, html.Text(parseTitle))
	}

	parseChildren := []ui.Node{
		header(parseTitle, parseToggle),
		section("Route", routeSummary(parseSnapshot.Route)),
		section("Cache", cacheSummary(parseSnapshot.Cache)),
		section("Multi-Client", multiClientSummary(parseSnapshot.MultiClient)),
		section("Boundaries", boundariesSummary(parseSnapshot.Boundaries)),
		section("Coordination", coordinationSummary(parseSnapshot.Coordination)),
		section("Runtime", statsSummary(parseSnapshot.Stats)),
		section("Inspector", selectedNodeSummary(parseSnapshot, parseSelectedPath.Get())),
		section("Profiling", profilingSummary(parseSnapshot.Profiling)),
		section("Hydration", hydrationSummary(parseSnapshot.Hydration)),
		section("Logs", logsSummary(parseSnapshot.Logs)),
		section("Diagnostics", diagnosticsSummary(parseSnapshot.Diagnostics)),
		section("Tree", treeSummary(parseSnapshot.Tree, 0, parseMaxDepth, parseSelectedPath.Get(), func(parsePath string) {
			parseSelectedPath.Set(parsePath)
		})),
	}
	parseChildren = append(parseChildren, renderExtensionSections(parseSnapshot.Extensions)...)

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
	}}, parseChildren...)
}

// ErrorOverlay renders a focused development overlay for current runtime failures.
func ErrorOverlay(parseProps ErrorOverlayProps) ui.Node {
	parseInterval := parseProps.RefreshInterval
	if parseInterval <= 0 {
		parseInterval = 750 * time.Millisecond
	}
	parseTitle := strings.TrimSpace(parseProps.Title)
	if parseTitle == "" {
		parseTitle = "GWC Error Overlay"
	}
	parseMaxItems := parseProps.MaxItems
	if parseMaxItems <= 0 {
		parseMaxItems = 4
	}

	parseSnapshot := UseSnapshot(parseInterval)
	parseIssues := collectOverlayIssues(parseSnapshot)
	parseActions := InspectErrorOverlayActions()
	parseDismissed := ui.UseState(false)
	parseSignature := overlayIssueFingerprint(parseIssues)
	ui.UseEffect(func() func() {
		parseDismissed.Set(false)
		return nil
	}, parseSignature)
	if len(parseIssues) == 0 {
		return nil
	}
	if len(parseIssues) > parseMaxItems {
		parseIssues = parseIssues[:parseMaxItems]
	}
	if parseDismissed.Get() {
		return nil
	}

	parseCloseOverlay := ui.UseEvent(func() {
		parseDismissed.Set(true)
	})

	parseItems := make([]ui.Node, 0, len(parseIssues)+1)
	parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
		"display":         "flex",
		"align-items":     "center",
		"justify-content": "space-between",
		"gap":             "12px",
		"margin-bottom":   "12px",
	}},
		html.Div(html.Props{},
			html.Strong(html.Props{Style: map[string]string{"display": "block", "color": "#fee2e2", "font-size": "14px", "letter-spacing": "0.04em", "text-transform": "uppercase"}}, html.Text(parseTitle)),
			html.Small(html.Props{Style: map[string]string{"color": "#fca5a5"}}, html.Text(fmt.Sprintf("%d active framework failure(s)", len(parseIssues)))),
		),
		html.Button(html.Props{OnClick: parseCloseOverlay, Style: map[string]string{
			"border":        "1px solid rgba(248,113,113,0.35)",
			"background":    "rgba(69,10,10,0.85)",
			"color":         "#fee2e2",
			"border-radius": "999px",
			"padding":       "8px 12px",
			"cursor":        "pointer",
		}}, html.Text("Dismiss")),
	))
	for _, parseIssue := range parseIssues {
		parseColor := "#fecaca"
		if parseIssue.Severity == SeverityWarning {
			parseColor = "#fde68a"
		}
		parseMatchedActions := matchingErrorOverlayActions(parseIssue, parseActions)
		parseItems = append(parseItems, html.Div(html.Props{Style: map[string]string{
			"padding":       "10px 12px",
			"border-radius": "12px",
			"border":        "1px solid rgba(248,113,113,0.22)",
			"background":    "rgba(127,29,29,0.32)",
			"margin-top":    "8px",
		}},
			html.Div(html.Props{Style: map[string]string{"font-size": "12px", "text-transform": "uppercase", "letter-spacing": "0.08em", "color": parseColor}}, html.Text(string(parseIssue.Severity)+" | "+emptyFallback(parseIssue.Source, "runtime")+" | "+emptyFallback(parseIssue.Code, "diagnostic"))),
			html.P(html.Props{Style: map[string]string{"margin": "6px 0 0 0", "color": "#fee2e2"}}, html.Text(parseIssue.Message)),
			func() ui.Node {
				if strings.TrimSpace(parseIssue.TopFrame) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "6px", "color": "#fecaca"}}, html.Text("where: "+parseIssue.TopFrame))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseIssue.Path) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#fecaca"}}, html.Text("path: "+parseIssue.Path))
			}(),
			func() ui.Node {
				if strings.TrimSpace(parseIssue.Docs) == "" {
					return nil
				}
				return html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#fca5a5"}}, html.Text("docs: "+parseIssue.Docs))
			}(),
			func() ui.Node {
				if len(parseMatchedActions) == 0 {
					return nil
				}
				parseButtons := make([]ui.Node, 0, len(parseMatchedActions))
				for _, parseAction := range parseMatchedActions {
					parseCurrent := parseAction
					parseButtons = append(parseButtons, html.Button(html.Props{OnClick: ui.WrapHandler(func() {
						if parseCurrent.Run != nil {
							parseCurrent.Run(ErrorOverlayActionContext{Snapshot: parseSnapshot, Issue: parseIssue})
						}
					}), Style: map[string]string{
						"border":        "1px solid rgba(248,113,113,0.28)",
						"background":    "rgba(69,10,10,0.72)",
						"color":         "#fee2e2",
						"border-radius": "999px",
						"padding":       "6px 10px",
						"cursor":        "pointer",
					}}, html.Text(parseAction.Label)))
				}
				return html.Div(html.Props{Style: map[string]string{
					"display":    "flex",
					"flex-wrap":  "wrap",
					"gap":        "8px",
					"margin-top": "8px",
				}}, parseButtons...)
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
	}}, parseItems...)
}

func header(parseTitle string, parseToggle ui.Handler) ui.Node {
	return html.Div(html.Props{Style: map[string]string{
		"display":         "flex",
		"align-items":     "center",
		"justify-content": "space-between",
		"gap":             "12px",
		"margin-bottom":   "14px",
	}},
		html.Div(html.Props{},
			html.Strong(html.Props{Style: map[string]string{"display": "block", "font-size": "14px", "letter-spacing": "0.08em", "text-transform": "uppercase", "color": "#67e8f9"}}, html.Text(parseTitle)),
			html.Small(html.Props{Style: map[string]string{"color": "#94a3b8"}}, html.Text("Component tree, hooks, routes, and diagnostics")),
		),
		html.Button(html.Props{OnClick: parseToggle, Style: map[string]string{
			"border":        "1px solid rgba(148,163,184,0.25)",
			"background":    "rgba(15,23,42,0.9)",
			"color":         "#e2e8f0",
			"border-radius": "999px",
			"padding":       "8px 12px",
			"cursor":        "pointer",
		}}, html.Text("Close")),
	)
}

func collectOverlayIssues(parseSnapshot Snapshot) []ErrorOverlayIssue {
	parseIssues := make([]ErrorOverlayIssue, 0, len(parseSnapshot.Diagnostics)+1)
	for _, parseDiagnostic := range parseSnapshot.Diagnostics {
		if !shouldShowOverlayDiagnostic(parseDiagnostic) {
			continue
		}
		parseIssues = append(parseIssues, ErrorOverlayIssue{
			Severity: parseDiagnostic.Severity,
			Source:   parseDiagnostic.Source,
			Code:     parseDiagnostic.Code,
			Message:  parseDiagnostic.Message,
			TopFrame: parseDiagnostic.TopFrame,
			Path:     parseDiagnostic.Path,
			Docs:     parseDiagnostic.Docs,
		})
	}
	if parseSnapshot.Hydration.Failed && !overlayHasHydrationIssue(parseIssues) {
		parseIssues = append(parseIssues, ErrorOverlayIssue{
			Severity: SeverityError,
			Source:   "hydration",
			Code:     "GWC-HYDRATION-FAILED",
			Message:  emptyFallback(parseSnapshot.Hydration.Failure, "hydration failed during client resume"),
			Docs:     "docs/ACTIONABLE_ERRORS.md#gwc-runtime-panic-hydration",
		})
	}
	return parseIssues
}

func shouldShowOverlayDiagnostic(parseDiagnostic Diagnostic) bool {
	if parseDiagnostic.Severity == SeverityError {
		return true
	}
	if strings.HasPrefix(strings.TrimSpace(parseDiagnostic.Code), "GWC-HYDRATION-") {
		return true
	}
	switch strings.TrimSpace(parseDiagnostic.Code) {
	case "GWC-ROUTER-LOADER-FAILED":
		return true
	}
	return false
}

func overlayHasHydrationIssue(parseIssues []ErrorOverlayIssue) bool {
	for _, parseIssue := range parseIssues {
		if strings.HasPrefix(strings.TrimSpace(parseIssue.Code), "GWC-HYDRATION-") {
			return true
		}
	}
	return false
}

func overlayIssueFingerprint(parseIssues []ErrorOverlayIssue) string {
	parseParts := make([]string, 0, len(parseIssues))
	for _, parseIssue := range parseIssues {
		parseParts = append(parseParts, parseIssue.Code+"|"+parseIssue.Message+"|"+parseIssue.TopFrame+"|"+parseIssue.Path)
	}
	return strings.Join(parseParts, "\n")
}

func matchingErrorOverlayActions(parseIssue ErrorOverlayIssue, parseActions []ErrorOverlayAction) []ErrorOverlayAction {
	if len(parseActions) == 0 {
		return nil
	}
	parseMatched := make([]ErrorOverlayAction, 0, len(parseActions))
	for _, parseAction := range parseActions {
		if !matchesErrorOverlayAction(parseIssue, parseAction) {
			continue
		}
		parseMatched = append(parseMatched, parseAction)
	}
	return parseMatched
}

func matchesErrorOverlayAction(parseIssue ErrorOverlayIssue, parseAction ErrorOverlayAction) bool {
	if len(parseAction.MatchCodes) == 0 && len(parseAction.MatchSources) == 0 {
		return true
	}
	if len(parseAction.MatchCodes) > 0 {
		isParseMatched := false
		for _, parseCode := range parseAction.MatchCodes {
			if strings.EqualFold(strings.TrimSpace(parseCode), strings.TrimSpace(parseIssue.Code)) {
				isParseMatched = true
				break
			}
		}
		if !isParseMatched {
			return false
		}
	}
	if len(parseAction.MatchSources) > 0 {
		isParseMatched2 := false
		for _, parseSource := range parseAction.MatchSources {
			if strings.EqualFold(strings.TrimSpace(parseSource), strings.TrimSpace(parseIssue.Source)) {
				isParseMatched2 = true
				break
			}
		}
		if !isParseMatched2 {
			return false
		}
	}
	return true
}

func section(parseTitle string, parseContent ui.Node) ui.Node {
	return html.Div(html.Props{Style: map[string]string{
		"margin-top":    "12px",
		"padding":       "12px",
		"border-radius": "14px",
		"background":    "rgba(15,23,42,0.88)",
		"border":        "1px solid rgba(51,65,85,0.7)",
	}},
		html.Div(html.Props{Style: map[string]string{"margin-bottom": "10px", "font-size": "12px", "text-transform": "uppercase", "letter-spacing": "0.08em", "color": "#67e8f9"}}, html.Text(parseTitle)),
		parseContent,
	)
}

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
			html.Div(html.Props{Style: map[string]string{"font-size": "12px", "text-transform": "uppercase", "letter-spacing": "0.08em", "color": parseColor}}, html.Text(string(parseDiagnostic.Severity)+" • "+parseDiagnostic.Source+" • count="+fmt.Sprintf("%d", parseDiagnostic.Count))),
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
