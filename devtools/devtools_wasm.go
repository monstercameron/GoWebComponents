//go:build js && wasm && !production

package devtools

import (
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/fetch"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/router"
	"github.com/monstercameron/GoWebComponents/v5/ui"
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
		Kernel:       snapshotKernelState(),
		Extensions:   InspectComposedExtensionSections(),
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
		parseCallback := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseState.Set(SnapshotNow())
			return nil
		})
		parseTimerID := js.Global().Call("setInterval", parseCallback, parseInterval.Milliseconds())

		return func() {
			js.Global().Call("clearInterval", parseTimerID)
			parseCallback.Release()
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
	parseActions := InspectComposedErrorOverlayActions()
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
			Docs:     "docs/REFERENCE_MANUAL/12-devtools-testing-and-observability.md#actionable-errors-and-troubleshooting",
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
