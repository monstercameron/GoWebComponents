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
	items := make([]feedItem, 0, 600)
	severities := []string{"Info", "Warn", "Critical"}
	owners := []string{"Atlas Ops", "Warehouse Sync", "Billing Edge", "Route Loader"}
	for i := 1; i <= 600; i++ {
		severity := severities[i%len(severities)]
		owner := owners[i%len(owners)]
		items = append(items, feedItem{
			ID:       fmt.Sprintf("evt-%03d", i),
			Title:    fmt.Sprintf("%s signal %03d", severity, i),
			Severity: severity,
			Owner:    owner,
			Summary:  fmt.Sprintf("This event row exercises fixed-height virtualization with stable keys, row selection, and live rendered-range diagnostics for item %03d.", i),
		})
	}
	return items
}

func virtualizedFeedExample() ui.Node {
	selectedID := ui.UseState(demoFeed[0].ID)
	diagnostics := ui.UseState(virtualization.ViewportDiagnostics{})
	preservedFlags := ui.UseState(map[string]bool{})
	restorationItems := ui.UseState(cloneFeedItems(demoFeed[:180]))
	restorationHeight := ui.UseState(280.0)
	restorationInsertCount := ui.UseState(0)
	restorationDiagnostics := ui.UseState(virtualization.ViewportDiagnostics{})
	selected := selectedFeedItem(selectedID.Get())
	restorationAnchor := currentVisibleItem(restorationItems.Get(), restorationDiagnostics.Get())

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
					ItemKey: func(item feedItem) string {
						return item.ID
					},
					OnViewportChange: diagnostics.Set,
					RenderRow: func(row virtualization.RowRenderProps[feedItem]) ui.Node {
						active := row.Item.ID == selectedID.Get()
						return html.Button(html.Props{
							Type: "button",
							Class: strings.Join([]string{
								"flex h-full w-full items-center justify-between rounded-[1.1rem] border px-4 text-left transition-colors",
								"border-white/8 bg-[rgba(15,23,42,0.55)] hover:border-cyan-300/35 hover:bg-[rgba(14,116,144,0.18)]",
								when(active, "border-cyan-300/55 bg-[rgba(8,145,178,0.22)] text-white"),
							}, " "),
							OnClick: ui.UseEvent(func() { selectedID.Set(row.Item.ID) }),
						},
							html.Div(html.Props{Class: "min-w-0"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300/80"}, html.Text(row.Item.Severity+" • "+row.Item.Owner)),
								html.P(html.Props{Class: "mt-2 truncate text-sm font-semibold text-white"}, html.Text(row.Item.Title)),
								html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text(row.Item.Summary)),
							),
							html.Div(html.Props{Class: "ml-4 text-right"},
								html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-500"}, html.Text("Index")),
								html.P(html.Props{Class: "mt-2 text-sm font-semibold text-slate-200"}, html.Text(fmt.Sprintf("%d", row.Index))),
							),
						)
					},
				}),
				html.Div(html.Props{Class: "grid gap-4"},
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Selected item")),
						html.H3(html.Props{Class: "mt-3 text-2xl font-bold text-white"}, html.Text(selected.Title)),
						html.P(html.Props{Class: "mt-2 text-sm text-slate-400"}, html.Text(selected.ID+" • "+selected.Owner+" • "+selected.Severity)),
						html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text(selected.Summary)),
					),
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Viewport diagnostics")),
						html.Div(html.Props{Class: "mt-4 grid gap-3 sm:grid-cols-2"},
							shared.ExampleStat("Visible range", fmt.Sprintf("%d-%d", diagnostics.Get().VisibleStart, diagnostics.Get().VisibleEnd)),
							shared.ExampleStat("Rendered range", fmt.Sprintf("%d-%d", diagnostics.Get().RenderedStart, diagnostics.Get().RenderedEnd)),
							shared.ExampleStat("Rendered rows", fmt.Sprintf("%d", diagnostics.Get().RenderedCount)),
							shared.ExampleStat("Overscan", fmt.Sprintf("%d / %d", diagnostics.Get().OverscanBeforeCount, diagnostics.Get().OverscanAfterCount)),
							shared.ExampleStat("Row mounts", fmt.Sprintf("%d", diagnostics.Get().RowMountCount)),
							shared.ExampleStat("Row unmounts", fmt.Sprintf("%d", diagnostics.Get().RowUnmountCount)),
							shared.ExampleStat("Measurement churn", fmt.Sprintf("%d / %d", diagnostics.Get().MeasurementCount, diagnostics.Get().InvalidationCount)),
							shared.ExampleStat("Scroll correction", fmt.Sprintf("%d", diagnostics.Get().ScrollCorrectionCount)),
							shared.ExampleStat("Scroll jank signal", buildFeedScrollJankSignal(diagnostics.Get())),
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
						ItemKey: func(item feedItem) string {
							return item.ID
						},
						RenderRow: func(row virtualization.RowRenderProps[feedItem]) ui.Node {
							return ui.CreateElement(func() ui.Node {
								localFlag := ui.UseState(false)
								return html.Label(html.Props{
									Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
								},
									html.Div(html.Props{Class: "min-w-0"},
										html.P(html.Props{Class: "font-semibold text-white"}, html.Text(row.Item.Title)),
										html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text("Toggle this, then scroll it far away and back.")),
									),
									html.Input(html.Props{
										Type:    "checkbox",
										Checked: localFlag.Get(),
										OnChange: ui.UseEvent(func() {
											localFlag.Set(!localFlag.Get())
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
						ItemKey: func(item feedItem) string {
							return item.ID
						},
						RenderRow: func(row virtualization.RowRenderProps[feedItem]) ui.Node {
							flagged := preservedFlags.Get()[row.Item.ID]
							return html.Label(html.Props{
								Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
							},
								html.Div(html.Props{Class: "min-w-0"},
									html.P(html.Props{Class: "font-semibold text-white"}, html.Text(row.Item.Title)),
									html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text("This flag lives in parent state keyed by item ID.")),
								),
								html.Input(html.Props{
									Type:    "checkbox",
									Checked: flagged,
									OnChange: ui.UseEvent(func() {
										next := cloneFlags(preservedFlags.Get())
										next[row.Item.ID] = !flagged
										preservedFlags.Set(next)
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
						restorationItems.Set(prependSyntheticEvents(restorationItems.Get(), restorationInsertCount, 3))
					}),
				}, html.Text("Prepend 3 events")),
				html.Button(html.Props{
					ID:      "restoration-height-button",
					Type:    "button",
					Class:   "rounded-full border border-white/12 bg-white/5 px-4 py-2 text-xs font-semibold uppercase tracking-[0.22em] text-slate-100 transition-colors hover:bg-white/10",
					OnClick: ui.UseEvent(func() { restorationHeight.Set(toggleRestorationHeight(restorationHeight.Get())) }),
				}, html.Text("Toggle height")),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.9fr)]"},
				renderRestorationSurface(restorationItems.Get(), restorationHeight.Get(), restorationDiagnostics.Set),
				html.Div(html.Props{Class: "grid gap-4"},
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Restoration anchor")),
						html.H3(html.Props{ID: "restoration-anchor-title", Class: "mt-3 text-2xl font-bold text-white"}, html.Text(restorationAnchor.Title)),
						html.P(html.Props{ID: "restoration-anchor-id", Class: "mt-2 text-sm text-slate-400"}, html.Text(restorationAnchor.ID+" â€¢ "+restorationAnchor.Owner+" â€¢ "+restorationAnchor.Severity)),
						html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("Scroll the list down, prepend new rows, resize the viewport, and reload the page. The first visible stable row should remain anchored instead of snapping back to the start.")),
					),
					html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Restoration diagnostics")),
						html.Div(html.Props{Class: "mt-4 grid gap-3 sm:grid-cols-2"},
							shared.ExampleStat("Visible range", fmt.Sprintf("%d-%d", restorationDiagnostics.Get().VisibleStart, restorationDiagnostics.Get().VisibleEnd)),
							shared.ExampleStat("Rendered range", fmt.Sprintf("%d-%d", restorationDiagnostics.Get().RenderedStart, restorationDiagnostics.Get().RenderedEnd)),
							shared.ExampleStat("Current height", fmt.Sprintf("%.0fpx", restorationHeight.Get())),
							shared.ExampleStat("Total rows", fmt.Sprintf("%d", len(restorationItems.Get()))),
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
						ItemKey: func(item feedItem) string {
							return item.ID
						},
						RenderRow: func(row virtualization.RowRenderProps[feedItem]) ui.Node {
							return benchmarkRow(row.Item)
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

func selectedFeedItem(id string) feedItem {
	for _, item := range demoFeed {
		if item.ID == id {
			return item
		}
	}
	return demoFeed[0]
}

func when(condition bool, className string) string {
	if condition {
		return className
	}
	return ""
}

func benchmarkRow(item feedItem) ui.Node {
	return html.Div(html.Props{
		Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
	},
		html.Div(html.Props{Class: "min-w-0"},
			html.P(html.Props{Class: "font-semibold text-white"}, html.Text(item.Title)),
			html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text(item.Owner+" • "+item.Severity)),
		),
		html.P(html.Props{Class: "ml-4 text-xs uppercase tracking-[0.2em] text-slate-500"}, html.Text(item.ID)),
	)
}

func renderFullBenchmarkRows(items []feedItem) []ui.Node {
	rows := make([]ui.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, html.Div(html.Props{
			Key:   item.ID,
			Style: map[string]string{"height": "48px", "boxSizing": "border-box"},
		}, benchmarkRow(item)))
	}
	return rows
}

func cloneFlags(current map[string]bool) map[string]bool {
	next := make(map[string]bool, len(current))
	for key, value := range current {
		next[key] = value
	}
	return next
}

func cloneFeedItems(items []feedItem) []feedItem {
	next := make([]feedItem, len(items))
	copy(next, items)
	return next
}

func initialHydrationDiagnostics(totalItems int, rowHeight float64, overscan int, viewportHeight float64) virtualization.ViewportDiagnostics {
	state, err := virtualization.ComputeViewportState(virtualization.ViewportConfig{
		TotalItems: totalItems,
		RowHeight:  rowHeight,
		Overscan:   overscan,
	}, 0, viewportHeight)
	if err != nil {
		panic(err)
	}
	return state.Diagnostics()
}

func prependSyntheticEvents(items []feedItem, counter ui.State[int], count int) []feedItem {
	nextCounter := counter.Get()
	inserted := make([]feedItem, 0, count)
	for offset := 0; offset < count; offset++ {
		nextCounter++
		inserted = append(inserted, feedItem{
			ID:       fmt.Sprintf("prep-%03d", nextCounter),
			Title:    fmt.Sprintf("Prepended anchor %03d", nextCounter),
			Severity: "Warn",
			Owner:    "Restoration Lane",
			Summary:  fmt.Sprintf("Synthetic prepended event %03d used to verify stable anchor restoration.", nextCounter),
		})
	}
	counter.Set(nextCounter)
	return append(inserted, cloneFeedItems(items)...)
}

func toggleRestorationHeight(current float64) float64 {
	if current <= 280 {
		return 360
	}
	return 280
}

func currentVisibleItem(items []feedItem, diagnostics virtualization.ViewportDiagnostics) feedItem {
	if len(items) == 0 {
		return feedItem{Title: "No rows", ID: "n/a", Owner: "n/a", Severity: "n/a"}
	}
	index := diagnostics.VisibleStart
	if index < 0 {
		index = 0
	}
	if index >= len(items) {
		index = len(items) - 1
	}
	return items[index]
}

func buildFeedScrollJankSignal(diagnostics virtualization.ViewportDiagnostics) string {
	if diagnostics.MeasurementCount > 0 || diagnostics.InvalidationCount > 0 || diagnostics.ScrollCorrectionCount > 0 {
		return "watch runtime churn"
	}
	expectedRows := diagnostics.VisibleCount + diagnostics.OverscanBeforeCount + diagnostics.OverscanAfterCount
	if diagnostics.RenderedCount > expectedRows+2 {
		return "watch rendered window"
	}
	return "steady"
}

func virtualizedHydrationExample() ui.Node {
	items := demoFeed[:180]
	diagnostics := ui.UseState(initialHydrationDiagnostics(len(items), 54, 2, 280))
	anchor := currentVisibleItem(items, diagnostics.Get())

	return html.Div(html.Props{Class: "mx-auto max-w-[72rem] px-6 py-16"},
		html.P(html.Props{Class: "text-sm uppercase tracking-[0.32em] text-cyan-300/80"}, html.Text("Virtualized hydration contract")),
		html.H1(html.Props{Class: "mt-4 text-4xl font-semibold tracking-[-0.03em] text-white"}, html.Text("virtualization.List hydration")),
		html.P(html.Props{Class: "mt-4 max-w-3xl leading-7 text-slate-300"}, html.Text("This page ships prerendered fixed-height list HTML and then hydrates the same initial window. After hydration, the list can switch to real scroll-driven window math without losing the overall spacer geometry.")),
		html.Div(html.Props{Class: "mt-8 grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.9fr)]"},
			virtualization.List(virtualization.ListProps[feedItem]{
				ID:        "virtualized-feed-hydration",
				Items:     items,
				Height:    280,
				RowHeight: 54,
				Overscan:  2,
				Class:     "rounded-[1.5rem] border border-cyan-300/15 bg-slate-950/45 p-2",
				ItemKey: func(item feedItem) string {
					return item.ID
				},
				OnViewportChange: diagnostics.Set,
				RenderRow: func(row virtualization.RowRenderProps[feedItem]) ui.Node {
					return html.Div(html.Props{
						Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
					},
						html.Div(html.Props{Class: "min-w-0"},
							html.P(html.Props{Class: "font-semibold text-white"}, html.Text(row.Item.Title)),
							html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text(row.Item.Owner+" • "+row.Item.Severity)),
						),
						html.P(html.Props{Class: "ml-4 text-xs uppercase tracking-[0.2em] text-slate-500"}, html.Text(row.Item.ID)),
					)
				},
			}),
			html.Div(html.Props{Class: "grid gap-4"},
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Hydration anchor")),
					html.H3(html.Props{ID: "hydration-anchor-title", Class: "mt-3 text-2xl font-bold text-white"}, html.Text(anchor.Title)),
					html.P(html.Props{ID: "hydration-anchor-id", Class: "mt-2 text-sm text-slate-400"}, html.Text(anchor.ID+" • "+anchor.Owner+" • "+anchor.Severity)),
					html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("The prerendered HTML should start with the same initial rows the hydrated list expects, then continue scrolling correctly once the browser owns the viewport math.")),
				),
				html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Hydration diagnostics")),
					html.Div(html.Props{Class: "mt-4 grid gap-3 sm:grid-cols-2"},
						hydrationStat("hydration-visible-range", "Visible range", fmt.Sprintf("%d-%d", diagnostics.Get().VisibleStart, diagnostics.Get().VisibleEnd)),
						hydrationStat("hydration-rendered-range", "Rendered range", fmt.Sprintf("%d-%d", diagnostics.Get().RenderedStart, diagnostics.Get().RenderedEnd)),
						hydrationStat("hydration-rendered-rows", "Rendered rows", fmt.Sprintf("%d", diagnostics.Get().RenderedCount)),
						hydrationStat("hydration-total-rows", "Total rows", fmt.Sprintf("%d", len(items))),
					),
				),
			),
		),
	)
}

func hydrationStat(id, label, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 p-4"},
		html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
		html.P(html.Props{ID: id, Class: "mt-3 text-3xl font-black text-white"}, html.Text(value)),
	)
}

func renderRestorationSurface(items []feedItem, height float64, setDiagnostics func(virtualization.ViewportDiagnostics)) ui.Node {
	return virtualization.List(virtualization.ListProps[feedItem]{
		ID:        "virtualized-feed-restoration",
		Items:     items,
		Height:    height,
		RowHeight: 54,
		Overscan:  2,
		Class:     "rounded-[1.5rem] border border-cyan-300/15 bg-slate-950/45 p-2",
		ItemKey: func(item feedItem) string {
			return item.ID
		},
		OnViewportChange: setDiagnostics,
		RenderRow: func(row virtualization.RowRenderProps[feedItem]) ui.Node {
			return html.Div(html.Props{
				Class: "flex h-full w-full items-center justify-between rounded-[1rem] border border-white/8 bg-[rgba(15,23,42,0.55)] px-4 text-sm text-slate-200",
			},
				html.Div(html.Props{Class: "min-w-0"},
					html.P(html.Props{Class: "font-semibold text-white"}, html.Text(row.Item.Title)),
					html.P(html.Props{Class: "mt-1 truncate text-xs text-slate-400"}, html.Text(row.Item.Owner+" • "+row.Item.Severity)),
				),
				html.P(html.Props{Class: "ml-4 text-xs uppercase tracking-[0.2em] text-slate-500"}, html.Text(row.Item.ID)),
			)
		},
	})
}

func main() {
	utils.DisableAllDebug()
	if _, err := ui.Hydrate(ui.CreateElement(virtualizedHydrationExample), "#hydration-app"); err == nil {
		select {}
	}
	ui.Render(ui.CreateElement(virtualizedFeedExample), "#app")
	select {}
}
