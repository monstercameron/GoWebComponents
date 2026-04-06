//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
	"github.com/monstercameron/GoWebComponents/virtualization"
)

type feedItem struct {
	ID       string
	Title    string
	Severity string
	Owner    string
	Summary  string
}

var demoFeed = buildDemoFeed()

func buildDemoFeed() []feedItem {
	parseItems := make([]feedItem, 0, 600)
	parseSeverities := []string{"Info", "Warn", "Critical"}
	parseOwners := []string{"Atlas Ops", "Warehouse Sync", "Billing Edge", "Route Loader"}
	for parseI := 1; parseI <= 600; parseI++ {
		parseSeverity := parseSeverities[parseI%len(parseSeverities)]
		parseOwner := parseOwners[parseI%len(parseOwners)]
		parseItems = append(parseItems, feedItem{
			ID:       fmt.Sprintf("evt-%03d", parseI),
			Title:    fmt.Sprintf("%s signal %03d", parseSeverity, parseI),
			Severity: parseSeverity,
			Owner:    parseOwner,
			Summary:  fmt.Sprintf("This event row exercises fixed-height virtualization with stable keys, row selection, and live rendered-range diagnostics for item %03d.", parseI),
		})
	}
	return parseItems
}

func virtualizedFeedExample() ui.Node {
	parseSelectedID := ui.UseState(demoFeed[0].ID)
	parseDiagnostics := ui.UseState(virtualization.ViewportDiagnostics{})
	parsePreservedFlags := ui.UseState(map[string]bool{})
	parseRestorationItems := ui.UseState(cloneFeedItems(demoFeed[:180]))
	parseRestorationHeight := ui.UseState(280.0)
	parseRestorationInsertCount := ui.UseState(0)
	parseRestorationDiagnostics := ui.UseState(virtualization.ViewportDiagnostics{})
	parseSelected := selectedFeedItem(parseSelectedID.Get())
	parseRestorationAnchor := currentVisibleItem(parseRestorationItems.Get(), parseRestorationDiagnostics.Get())

	return shared.ExamplePage(
		"virtualization.List",
		"Fixed-height virtualized feed with live viewport diagnostics",
		"This example proves the first virtualization contract with a long event feed, explicit stable row keys, row selection that survives scrolling, and a diagnostics panel that reports visible range, overscan, and row churn.",
		shared.ExamplePanel("Virtualized feed",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Scroll the feed, select different rows, and watch the diagnostics update as the visible window shifts. The selected item state lives outside the row shell, so it stays stable even when the row unmounts.")),
			html.Div(html.Props{Class: "mt-6 grid gap-6 xl:grid-cols-[minmax(0,1.6fr)_minmax(18rem,0.9fr)]"},
				virtualization.List(virtualization.ListProps[feedItem]{
					ID:         "virtualized-feed-demo",
					Items:      demoFeed,
					Height:     420,
					RowHeight:  56,
					Overscan:   3,
					Class:      "rounded-[1.75rem] border border-cyan-300/15 bg-slate-950/45 p-2 shadow-[0_18px_48px_rgba(2,6,23,0.32)]",
					InnerClass: "rounded-[1.2rem]",
					ItemKey: func(parseItem feedItem) string {
						return parseItem.ID
					},
					OnViewportChange: parseDiagnostics.Set,
					RenderRow: func(parseRow virtualization.RowRenderProps[feedItem]) ui.Node {
						isParseActive := parseRow.Item.ID == parseSelectedID.Get()
						return html.Button(html.Props{
							Type: "button",
							Class: strings.Join([]string{
								"flex h-full w-full items-center justify-between rounded-[1.1rem] border px-4 text-left transition-colors",
								"border-white/8 bg-[rgba(15,23,42,0.55)] hover:border-cyan-300/35 hover:bg-[rgba(14,116,144,0.18)]",
								when(isParseActive, "border-cyan-300/55 bg-[rgba(8,145,178,0.22)] text-white"),
							}, " "),
							OnClick: ui.UseEvent(func() { parseSelectedID.Set(parseRow.Item.ID) }),
						},
							html.Div(html.Props{Class: "min-w-0"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300/80"}, html.Text(parseRow.Item.Severity+" • "+parseRow.Item.Owner)),
								html.P(html.Props{Class: "mt-2 truncate text-sm font-semibold text-white"}, html.Text(parseRow.Item.Title)),
								html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text(parseRow.Item.Summary)),
							),
							html.Div(html.Props{Class: "ml-4 text-right"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-500"}, html.Text("Index")),
								html.P(html.Props{Class: "mt-2 text-sm font-semibold text-slate-200"}, html.Text(fmt.Sprintf("%d", parseRow.Index))),
							),
						)
					},
				}),
				html.Div(html.Props{Class: "grid gap-4"},
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Selected item")),
						html.H3(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(parseSelected.Title)),
						html.P(html.Props{Class: "mt-2 text-sm text-slate-400"}, html.Text(parseSelected.ID+" • "+parseSelected.Owner+" • "+parseSelected.Severity)),
						html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text(parseSelected.Summary)),
					),
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Viewport diagnostics")),
						html.Div(html.Props{Class: "mt-4 grid gap-3 sm:grid-cols-2"},
							shared.ExampleStat("Visible range", fmt.Sprintf("%d-%d", parseDiagnostics.Get().VisibleStart, parseDiagnostics.Get().VisibleEnd)),
							shared.ExampleStat("Rendered range", fmt.Sprintf("%d-%d", parseDiagnostics.Get().RenderedStart, parseDiagnostics.Get().RenderedEnd)),
							shared.ExampleStat("Rendered rows", fmt.Sprintf("%d", parseDiagnostics.Get().RenderedCount)),
							shared.ExampleStat("Overscan", fmt.Sprintf("%d / %d", parseDiagnostics.Get().OverscanBeforeCount, parseDiagnostics.Get().OverscanAfterCount)),
							shared.ExampleStat("Row mounts", fmt.Sprintf("%d", parseDiagnostics.Get().RowMountCount)),
							shared.ExampleStat("Row unmounts", fmt.Sprintf("%d", parseDiagnostics.Get().RowUnmountCount)),
							shared.ExampleStat("Measurement churn", fmt.Sprintf("%d / %d", parseDiagnostics.Get().MeasurementCount, parseDiagnostics.Get().InvalidationCount)),
							shared.ExampleStat("Scroll correction", fmt.Sprintf("%d", parseDiagnostics.Get().ScrollCorrectionCount)),
							shared.ExampleStat("Scroll jank signal", buildFeedScrollJankSignal(parseDiagnostics.Get())),
						),
					),
				),
			),
		),
		shared.ExamplePanel("Why this matters",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The feed keeps 600 rows in the dataset while only rendering a narrow fixed-height window. That demonstrates the core value of virtualization: cut mounted row count first, then use the diagnostics to verify the window remains bounded and the churn stays proportional to scrolling.")),
			shared.ExampleCode(
				"virtualization.List(virtualization.ListProps[feedItem]{",
				"    ID: \"virtualized-feed-demo\",",
				"    Items: demoFeed,",
				"    Height: 420,",
				"    RowHeight: 56,",
				"    Overscan: 3,",
				"    ItemKey: func(item feedItem) string { return item.ID },",
				"    OnViewportChange: diagnostics.Set,",
				"    RenderRow: func(row virtualization.RowRenderProps[feedItem]) ui.Node { ... },",
				"})",
			),
		),
		shared.ExamplePanel("Row state ownership",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The left list keeps checkbox state inside each row component, so scrolling rows out of the window can wipe that local state. The right list stores the same flag in parent state keyed by item ID, so it survives virtualization unmounts.")),
			html.Div(html.Props{Class: "mt-6 grid gap-6 xl:grid-cols-2"},
				html.Div(html.Props{Class: "grid gap-4"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-rose-300"}, html.Text("Pitfall: row-local state")),
					virtualization.List(virtualization.ListProps[feedItem]{
						ID:        "virtualized-feed-pitfall",
						Items:     demoFeed[:180],
						Height:    280,
						RowHeight: 54,
						Overscan:  2,
						Class:     "rounded-[1.5rem] border border-rose-300/15 bg-slate-950/45 p-2",
						ItemKey: func(parseItem2 feedItem) string {
							return parseItem2.ID
						},
						RenderRow: func(parseRow2 virtualization.RowRenderProps[feedItem]) ui.Node {
							return ui.CreateElement(func() ui.Node {
								parseLocalFlag := ui.UseState(false)
								return html.Label(html.Props{
									Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
								},
									html.Div(html.Props{Class: "min-w-0"},
										html.P(html.Props{Class: "font-semibold text-white"}, html.Text(parseRow2.Item.Title)),
										html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text("Toggle this, then scroll it far away and back.")),
									),
									html.Input(html.Props{
										Type:    "checkbox",
										Checked: parseLocalFlag.Get(),
										OnChange: ui.UseEvent(func() {
											parseLocalFlag.Set(!parseLocalFlag.Get())
										}),
									}),
								)
							})
						},
					}),
				),
				html.Div(html.Props{Class: "grid gap-4"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-emerald-300"}, html.Text("Recommended: externalized by key")),
					virtualization.List(virtualization.ListProps[feedItem]{
						ID:        "virtualized-feed-preserved",
						Items:     demoFeed[:180],
						Height:    280,
						RowHeight: 54,
						Overscan:  2,
						Class:     "rounded-[1.5rem] border border-emerald-300/15 bg-slate-950/45 p-2",
						ItemKey: func(parseItem3 feedItem) string {
							return parseItem3.ID
						},
						RenderRow: func(parseRow3 virtualization.RowRenderProps[feedItem]) ui.Node {
							parseFlagged := parsePreservedFlags.Get()[parseRow3.Item.ID]
							return html.Label(html.Props{
								Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
							},
								html.Div(html.Props{Class: "min-w-0"},
									html.P(html.Props{Class: "font-semibold text-white"}, html.Text(parseRow3.Item.Title)),
									html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text("This flag lives in parent state keyed by item ID.")),
								),
								html.Input(html.Props{
									Type:    "checkbox",
									Checked: parseFlagged,
									OnChange: ui.UseEvent(func() {
										parseNext := cloneFlags(parsePreservedFlags.Get())
										parseNext[parseRow3.Item.ID] = !parseFlagged
										parsePreservedFlags.Set(parseNext)
									}),
								}),
							)
						},
					}),
				),
			),
		),
		shared.ExamplePanel("Scroll restoration and anchors",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("This surface exercises the documented first-pass restoration contract. Ordinary rerenders keep pixel offset stable, prepending new rows restores from the first visible stable item key, and reloading the page restores the prior anchor from session state instead of snapping back to the top.")),
			html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"},
				html.Button(html.Props{
					ID:    "restoration-prepend-button",
					Type:  "button",
					Class: "rounded-full border border-cyan-300/30 bg-cyan-400/10 px-4 py-2 text-xs font-semibold uppercase tracking-[0.22em] text-cyan-100 transition-colors hover:bg-cyan-400/20",
					OnClick: ui.UseEvent(func() {
						parseRestorationItems.Set(prependSyntheticEvents(parseRestorationItems.Get(), parseRestorationInsertCount, 3))
					}),
				}, html.Text("Prepend 3 events")),
				html.Button(html.Props{
					ID:      "restoration-height-button",
					Type:    "button",
					Class:   "rounded-full border border-white/12 bg-white/5 px-4 py-2 text-xs font-semibold uppercase tracking-[0.22em] text-slate-100 transition-colors hover:bg-white/10",
					OnClick: ui.UseEvent(func() { parseRestorationHeight.Set(toggleRestorationHeight(parseRestorationHeight.Get())) }),
				}, html.Text("Toggle height")),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.9fr)]"},
				renderRestorationSurface(parseRestorationItems.Get(), parseRestorationHeight.Get(), parseRestorationDiagnostics.Set),
				html.Div(html.Props{Class: "grid gap-4"},
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Restoration anchor")),
						html.H3(html.Props{ID: "restoration-anchor-title", Class: "mt-3 text-2xl font-bold text-white"}, html.Text(parseRestorationAnchor.Title)),
						html.P(html.Props{ID: "restoration-anchor-id", Class: "mt-2 text-sm text-slate-400"}, html.Text(parseRestorationAnchor.ID+" - "+parseRestorationAnchor.Owner+" - "+parseRestorationAnchor.Severity)),
						html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("Scroll the list down, prepend new rows, resize the viewport, and reload the page. The first visible stable row should remain anchored instead of snapping back to the start.")),
					),
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Restoration diagnostics")),
						html.Div(html.Props{Class: "mt-4 grid gap-3 sm:grid-cols-2"},
							shared.ExampleStat("Visible range", fmt.Sprintf("%d-%d", parseRestorationDiagnostics.Get().VisibleStart, parseRestorationDiagnostics.Get().VisibleEnd)),
							shared.ExampleStat("Rendered range", fmt.Sprintf("%d-%d", parseRestorationDiagnostics.Get().RenderedStart, parseRestorationDiagnostics.Get().RenderedEnd)),
							shared.ExampleStat("Current height", fmt.Sprintf("%.0fpx", parseRestorationHeight.Get())),
							shared.ExampleStat("Total rows", fmt.Sprintf("%d", len(parseRestorationItems.Get()))),
						),
					),
				),
			),
		),
		shared.ExamplePanel("Benchmark comparison surfaces",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("These two surfaces carry the same dataset and visual row treatment. The left uses virtualization to keep the mounted row count bounded. The right renders the entire feed at once so the browser benchmark lane can compare the scroll workload directly.")),
			html.Div(html.Props{Class: "mt-6 grid gap-6 xl:grid-cols-2"},
				html.Div(html.Props{Class: "grid gap-4"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Virtualized benchmark surface")),
					virtualization.List(virtualization.ListProps[feedItem]{
						ID:        "virtualized-benchmark-feed",
						Items:     demoFeed,
						Height:    360,
						RowHeight: 48,
						Overscan:  3,
						Class:     "rounded-[1.5rem] border border-cyan-300/15 bg-slate-950/45 p-2",
						ItemKey: func(parseItem4 feedItem) string {
							return parseItem4.ID
						},
						RenderRow: func(parseRow4 virtualization.RowRenderProps[feedItem]) ui.Node {
							return benchmarkRow(parseRow4.Item)
						},
					}),
				),
				html.Div(html.Props{Class: "grid gap-4"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-amber-300"}, html.Text("Full render benchmark surface")),
					html.Div(html.Props{
						ID:    "full-benchmark-feed",
						Class: "rounded-[1.5rem] border border-amber-300/15 bg-slate-950/45 p-2 overflow-y-auto",
						Style: map[string]string{"height": "360px"},
					}, renderFullBenchmarkRows(demoFeed)...),
				),
			),
		),
	)
}

func selectedFeedItem(parseId string) feedItem {
	for _, parseItem := range demoFeed {
		if parseItem.ID == parseId {
			return parseItem
		}
	}
	return demoFeed[0]
}

func when(isCondition bool, parseClassName string) string {
	if isCondition {
		return parseClassName
	}
	return ""
}

func benchmarkRow(parseItem feedItem) ui.Node {
	return html.Div(html.Props{
		Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
	},
		html.Div(html.Props{Class: "min-w-0"},
			html.P(html.Props{Class: "font-semibold text-white"}, html.Text(parseItem.Title)),
			html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text(parseItem.Owner+" • "+parseItem.Severity)),
		),
		html.P(html.Props{Class: "ml-4 text-xs uppercase tracking-[0.2em] text-slate-500"}, html.Text(parseItem.ID)),
	)
}

func renderFullBenchmarkRows(parseItems []feedItem) []ui.Node {
	parseRows := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseRows = append(parseRows, html.Div(html.Props{
			Key:   parseItem.ID,
			Style: map[string]string{"height": "48px", "boxSizing": "border-box"},
		}, benchmarkRow(parseItem)))
	}
	return parseRows
}

func cloneFlags(parseCurrent map[string]bool) map[string]bool {
	parseNext := make(map[string]bool, len(parseCurrent))
	for parseKey, parseValue := range parseCurrent {
		parseNext[parseKey] = parseValue
	}
	return parseNext
}

func cloneFeedItems(parseItems []feedItem) []feedItem {
	parseNext := make([]feedItem, len(parseItems))
	copy(parseNext, parseItems)
	return parseNext
}

func initialHydrationDiagnostics(parseTotalItems int, parseRowHeight float64, parseOverscan int, parseViewportHeight float64) virtualization.ViewportDiagnostics {
	parseState, parseErr := virtualization.ComputeViewportState(virtualization.ViewportConfig{
		TotalItems: parseTotalItems,
		RowHeight:  parseRowHeight,
		Overscan:   parseOverscan,
	}, 0, parseViewportHeight)
	if parseErr != nil {
		panic(parseErr)
	}
	return parseState.Diagnostics()
}

func prependSyntheticEvents(parseItems []feedItem, parseCounter ui.State[int], parseCount int) []feedItem {
	parseNextCounter := parseCounter.Get()
	parseInserted := make([]feedItem, 0, parseCount)
	for parseOffset := 0; parseOffset < parseCount; parseOffset++ {
		parseNextCounter++
		parseInserted = append(parseInserted, feedItem{
			ID:       fmt.Sprintf("prep-%03d", parseNextCounter),
			Title:    fmt.Sprintf("Prepended anchor %03d", parseNextCounter),
			Severity: "Warn",
			Owner:    "Restoration Lane",
			Summary:  fmt.Sprintf("Synthetic prepended event %03d used to verify stable anchor restoration.", parseNextCounter),
		})
	}
	parseCounter.Set(parseNextCounter)
	return append(parseInserted, cloneFeedItems(parseItems)...)
}

func toggleRestorationHeight(parseCurrent float64) float64 {
	if parseCurrent <= 280 {
		return 360
	}
	return 280
}

func currentVisibleItem(parseItems []feedItem, parseDiagnostics virtualization.ViewportDiagnostics) feedItem {
	if len(parseItems) == 0 {
		return feedItem{Title: "No rows", ID: "n/a", Owner: "n/a", Severity: "n/a"}
	}
	parseIndex := parseDiagnostics.VisibleStart
	if parseIndex < 0 {
		parseIndex = 0
	}
	if parseIndex >= len(parseItems) {
		parseIndex = len(parseItems) - 1
	}
	return parseItems[parseIndex]
}

func buildFeedScrollJankSignal(parseDiagnostics virtualization.ViewportDiagnostics) string {
	if parseDiagnostics.MeasurementCount > 0 || parseDiagnostics.InvalidationCount > 0 || parseDiagnostics.ScrollCorrectionCount > 0 {
		return "watch runtime churn"
	}
	parseExpectedRows := parseDiagnostics.VisibleCount + parseDiagnostics.OverscanBeforeCount + parseDiagnostics.OverscanAfterCount
	if parseDiagnostics.RenderedCount > parseExpectedRows+2 {
		return "watch rendered window"
	}
	return "steady"
}

func virtualizedHydrationExample() ui.Node {
	parseItems := demoFeed[:180]
	parseDiagnostics := ui.UseState(initialHydrationDiagnostics(len(parseItems), 54, 2, 280))
	parseAnchor := currentVisibleItem(parseItems, parseDiagnostics.Get())

	return html.Div(html.Props{Class: "mx-auto max-w-[72rem] px-6 py-16"},
		html.P(html.Props{Class: "text-sm uppercase tracking-[0.32em] text-cyan-300/80"}, html.Text("Virtualized hydration contract")),
		html.H1(html.Props{Class: "mt-4 text-4xl font-semibold tracking-[-0.03em] text-white"}, html.Text("virtualization.List hydration")),
		html.P(html.Props{Class: "mt-4 max-w-3xl leading-7 text-slate-300"}, html.Text("This page ships prerendered fixed-height list HTML and then hydrates the same initial window. After hydration, the list can switch to real scroll-driven window math without losing the overall spacer geometry.")),
		html.Div(html.Props{Class: "mt-8 grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.9fr)]"},
			virtualization.List(virtualization.ListProps[feedItem]{
				ID:        "virtualized-feed-hydration",
				Items:     parseItems,
				Height:    280,
				RowHeight: 54,
				Overscan:  2,
				Class:     "rounded-[1.5rem] border border-cyan-300/15 bg-slate-950/45 p-2",
				ItemKey: func(parseItem feedItem) string {
					return parseItem.ID
				},
				OnViewportChange: parseDiagnostics.Set,
				RenderRow: func(parseRow virtualization.RowRenderProps[feedItem]) ui.Node {
					return html.Div(html.Props{
						Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
					},
						html.Div(html.Props{Class: "min-w-0"},
							html.P(html.Props{Class: "font-semibold text-white"}, html.Text(parseRow.Item.Title)),
							html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text(parseRow.Item.Owner+" • "+parseRow.Item.Severity)),
						),
						html.P(html.Props{Class: "ml-4 text-xs uppercase tracking-[0.2em] text-slate-500"}, html.Text(parseRow.Item.ID)),
					)
				},
			}),
			html.Div(html.Props{Class: "grid gap-4"},
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Hydration anchor")),
					html.H3(html.Props{ID: "hydration-anchor-title", Class: "mt-3 text-2xl font-bold text-white"}, html.Text(parseAnchor.Title)),
					html.P(html.Props{ID: "hydration-anchor-id", Class: "mt-2 text-sm text-slate-400"}, html.Text(parseAnchor.ID+" • "+parseAnchor.Owner+" • "+parseAnchor.Severity)),
					html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("The prerendered HTML should start with the same initial rows the hydrated list expects, then continue scrolling correctly once the browser owns the viewport math.")),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Hydration diagnostics")),
					html.Div(html.Props{Class: "mt-4 grid gap-3 sm:grid-cols-2"},
						hydrationStat("hydration-visible-range", "Visible range", fmt.Sprintf("%d-%d", parseDiagnostics.Get().VisibleStart, parseDiagnostics.Get().VisibleEnd)),
						hydrationStat("hydration-rendered-range", "Rendered range", fmt.Sprintf("%d-%d", parseDiagnostics.Get().RenderedStart, parseDiagnostics.Get().RenderedEnd)),
						hydrationStat("hydration-rendered-rows", "Rendered rows", fmt.Sprintf("%d", parseDiagnostics.Get().RenderedCount)),
						hydrationStat("hydration-total-rows", "Total rows", fmt.Sprintf("%d", len(parseItems))),
					),
				),
			),
		),
	)
}

func hydrationStat(parseId, parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 p-4"},
		html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{ID: parseId, Class: "mt-3 text-3xl font-black text-white"}, html.Text(parseValue)),
	)
}

func renderRestorationSurface(parseItems []feedItem, parseHeight float64, setDiagnostics func(virtualization.ViewportDiagnostics)) ui.Node {
	return virtualization.List(virtualization.ListProps[feedItem]{
		ID:        "virtualized-feed-restoration",
		Items:     parseItems,
		Height:    parseHeight,
		RowHeight: 54,
		Overscan:  2,
		Class:     "rounded-[1.5rem] border border-cyan-300/15 bg-slate-950/45 p-2",
		ItemKey: func(parseItem feedItem) string {
			return parseItem.ID
		},
		OnViewportChange: setDiagnostics,
		RenderRow: func(parseRow virtualization.RowRenderProps[feedItem]) ui.Node {
			return html.Div(html.Props{
				Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
			},
				html.Div(html.Props{Class: "min-w-0"},
					html.P(html.Props{Class: "font-semibold text-white"}, html.Text(parseRow.Item.Title)),
					html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text(parseRow.Item.Owner+" • "+parseRow.Item.Severity)),
				),
				html.P(html.Props{Class: "ml-4 text-xs uppercase tracking-[0.2em] text-slate-500"}, html.Text(parseRow.Item.ID)),
			)
		},
	})
}

func main() {
	utils.DisableAllDebug()
	if _, parseErr := ui.Hydrate(ui.CreateElement(virtualizedHydrationExample), "#hydration-app"); parseErr == nil {
		select {}
	}
	ui.Render(ui.CreateElement(virtualizedFeedExample), "#app")
	select {}
}
