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
		},
		Tree:      mapNode(rtSnapshot.Root),
		Stats:     mapStats(rtSnapshot.Stats),
		Profiling: mapProfiling(rtSnapshot.Profiling),
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
	toggle := ui.UseEvent(func() {
		open.Update(func(current bool) bool { return !current })
	})

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
	}},
		header(title, toggle),
		section("Route", routeSummary(snapshot.Route)),
		section("Cache", cacheSummary(snapshot.Cache)),
		section("Runtime", statsSummary(snapshot.Stats)),
		section("Profiling", profilingSummary(snapshot.Profiling)),
		section("Logs", logsSummary(snapshot.Logs)),
		section("Diagnostics", diagnosticsSummary(snapshot.Diagnostics)),
		section("Tree", treeSummary(snapshot.Tree, 0, maxDepth)),
	)
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
			metricRow("Resume", emptyFallback(entry.ResumePolicy, "trust-once")),
			metricRow("Updated", formatTime(entry.UpdatedAt)),
			metricRow("Last load", formatTime(entry.LastLoaded)),
			metricRow("Error", emptyFallback(entry.LastError, "none")),
		))
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
	}
	if len(profiling.HotBranches) > 0 {
		children = append(children, hotBranchesSummary(profiling.HotBranches))
	}
	return html.Div(html.Props{}, children...)
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
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "4px", "color": "#cbd5e1"}}, html.Text("self="+formatDurationNs(branch.SelfDurationNs)+" commit="+formatDurationNs(branch.CommitDurationNs)+" effect="+formatDurationNs(branch.EffectDurationNs)+" cleanup="+formatDurationNs(branch.CleanupDurationNs))),
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

func treeSummary(node *Node, depth int, maxDepth int) ui.Node {
	if node == nil {
		return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#94a3b8"}}, html.Text("No committed tree yet."))
	}
	return renderNode(*node, depth, maxDepth)
}

func renderNode(node Node, depth int, maxDepth int) ui.Node {
	children := []ui.Node{
		html.Div(html.Props{Style: map[string]string{
			"display":     "flex",
			"flex-wrap":   "wrap",
			"gap":         "8px",
			"align-items": "center",
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
				return html.Small(html.Props{Style: map[string]string{"color": "#cbd5e1"}}, html.Text("subtree="+formatDurationNs(node.SubtreeDurationNs)+" self="+formatDurationNs(node.SelfDurationNs)))
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
			hookNodes = append(hookNodes, html.Div(html.Props{Style: map[string]string{"margin-top": "6px", "color": "#cbd5e1"}},
				html.Code(html.Props{Style: map[string]string{"color": "#67e8f9"}}, html.Text(hook.Kind)),
				html.Text(": "+hook.Value),
			))
		}
		children = append(children, html.Div(html.Props{Style: map[string]string{"margin-top": "6px"}}, hookNodes...))
	}

	if depth < maxDepth && len(node.Children) > 0 {
		childNodes := make([]ui.Node, 0, len(node.Children))
		for _, child := range node.Children {
			childNodes = append(childNodes, renderNode(child, depth+1, maxDepth))
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
		Kind:              node.Kind,
		Dirty:             node.Dirty,
		NeedsUpdate:       node.NeedsUpdate,
		FineGrained:       node.FineGrained,
		ReactiveSource:    node.ReactiveSource,
		UpdateOrigin:      node.UpdateOrigin,
		EffectCount:       node.EffectCount,
		HookCount:         node.HookCount,
		Signature:         "",
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
		mapped.Hooks = append(mapped.Hooks, Hook{Kind: hook.Kind, Value: hook.Value})
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
	}
	for _, branch := range profiling.HotBranches {
		mapped.HotBranches = append(mapped.HotBranches, Branch{
			Name:              branch.Name,
			Kind:              branch.Kind,
			Path:              branch.Path,
			CommitDurationNs:  branch.CommitDurationNs,
			EffectDurationNs:  branch.EffectDurationNs,
			CleanupDurationNs: branch.CleanupDurationNs,
			SelfDurationNs:    branch.SelfDurationNs,
			SubtreeDurationNs: branch.SubtreeDurationNs,
		})
	}
	return mapped
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

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "n/a"
	}
	return value.Format("15:04:05")
}
