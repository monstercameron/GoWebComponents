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
)

type directoryEntry struct {
	Name   string
	Team   string
	Region string
	Status string
}

type searchRow struct {
	Title   string
	Meta    string
	Score   string
	Preview string
}

type searchSnapshot struct {
	Query       string
	RowsScanned int
	Matched     int
	Results     []searchRow
}

type dashboardMetric struct {
	Label string
	Value string
}

type dashboardSnapshot struct {
	Name     string
	Summary  string
	Metrics  []dashboardMetric
	Activity []string
}

type routeSnapshot struct {
	Path       string
	Title      string
	Summary    string
	Highlights []string
}

var directorySeed = []directoryEntry{
	{Name: "Avery Cole", Team: "Operations", Region: "Boston", Status: "Ready"},
	{Name: "Bianca Soto", Team: "Inventory", Region: "Phoenix", Status: "Review"},
	{Name: "Callum Hart", Team: "Fulfillment", Region: "Austin", Status: "Ready"},
	{Name: "Daria Lin", Team: "Support", Region: "Chicago", Status: "Escalated"},
	{Name: "Emmett Hale", Team: "Routing", Region: "Seattle", Status: "Ready"},
	{Name: "Farah Noor", Team: "Planning", Region: "Atlanta", Status: "Review"},
	{Name: "Gavin Price", Team: "Analytics", Region: "Denver", Status: "Ready"},
	{Name: "Hana Park", Team: "CX", Region: "Miami", Status: "Pilot"},
	{Name: "Iris Webb", Team: "Warehouse", Region: "Dallas", Status: "Ready"},
	{Name: "Jonah Moss", Team: "Returns", Region: "Portland", Status: "Review"},
	{Name: "Kira Shaw", Team: "Demand", Region: "New York", Status: "Ready"},
	{Name: "Leo Grant", Team: "Ops Systems", Region: "San Diego", Status: "Pilot"},
}

var dashboardSeeds = map[string]dashboardSnapshot{
	"Pipeline": {
		Name:    "Pipeline",
		Summary: "Track incoming work, aging, and SLA pressure without blocking urgent controls.",
		Metrics: []dashboardMetric{
			{Label: "Queued reviews", Value: "128"},
			{Label: "Breached SLA", Value: "6"},
			{Label: "Today throughput", Value: "412"},
		},
		Activity: []string{
			"Rebuilt review queue ordering for priority-first dispatch.",
			"Grouped expiring requests into one visible sweep.",
			"Collapsed duplicate alerts before repainting the lane.",
		},
	},
	"Capacity": {
		Name:    "Capacity",
		Summary: "Switch to allocation-heavy summaries while the current tab selection still responds immediately.",
		Metrics: []dashboardMetric{
			{Label: "Available labor", Value: "84%"},
			{Label: "Open shifts", Value: "19"},
			{Label: "Overflow risk", Value: "Moderate"},
		},
		Activity: []string{
			"Recomputed station coverage by region and shift.",
			"Folded training reserves into the capacity forecast.",
			"Updated downstream constraints for late inbound stock.",
		},
	},
	"Experience": {
		Name:    "Experience",
		Summary: "Model customer-facing impact as a non-urgent dashboard swap instead of tying it to every click.",
		Metrics: []dashboardMetric{
			{Label: "Active incidents", Value: "4"},
			{Label: "First reply", Value: "11m"},
			{Label: "Satisfaction", Value: "96.2%"},
		},
		Activity: []string{
			"Ranked accounts by impact radius and response lag.",
			"Merged duplicate complaint clusters into one card.",
			"Projected overnight risk for delayed warehouse routes.",
		},
	},
}

var routeSeeds = map[string]routeSnapshot{
	"/orders": {
		Path:    "/orders",
		Title:   "Orders workspace",
		Summary: "A route transition can commit the next screen after non-urgent preparation finishes, instead of blocking the whole shell.",
		Highlights: []string{
			"Reuse the current shell while the next section tree prepares.",
			"Keep the requested path visible before the heavier view commits.",
			"Move layout-heavy reflow behind the transition boundary.",
		},
	},
	"/inventory": {
		Path:    "/inventory",
		Title:   "Inventory workspace",
		Summary: "Large tables, filters, and summaries are typical candidates for transition-marked route work.",
		Highlights: []string{
			"Refresh stock cards after the urgent nav intent is already acknowledged.",
			"Preserve focus and global controls while the panel swaps.",
			"Treat loaders and boundaries as separate pending surfaces from transition state.",
		},
	},
	"/fulfillment": {
		Path:    "/fulfillment",
		Title:   "Fulfillment workspace",
		Summary: "Dispatch-heavy views often change a lot of chrome at once, which is where explicit non-urgent scheduling becomes easier to reason about.",
		Highlights: []string{
			"Defer route-sized repaint work instead of every tap feeling blocking.",
			"Keep the nav rail and current intent readable while content catches up.",
			"Use the same pattern for section-level swaps that do not need a real router yet.",
		},
	},
}

func buildSearchSnapshot(query string) searchSnapshot {
	normalized := strings.ToLower(strings.TrimSpace(query))
	results := make([]searchRow, 0, 6)
	matched := 0
	rowsScanned := len(directorySeed) * 18

	for cycle := 0; cycle < 18; cycle++ {
		for _, entry := range directorySeed {
			searchable := strings.ToLower(entry.Name + " " + entry.Team + " " + entry.Region + " " + entry.Status)
			if normalized != "" && !strings.Contains(searchable, normalized) {
				continue
			}
			matched++
			if len(results) >= 6 {
				continue
			}
			results = append(results, searchRow{
				Title:   fmt.Sprintf("%s #%02d", entry.Name, cycle+1),
				Meta:    fmt.Sprintf("%s - %s", entry.Team, entry.Region),
				Score:   fmt.Sprintf("status: %s", entry.Status),
				Preview: fmt.Sprintf("Result rows stay responsive because the input value commits before the heavier filter pass finishes for %s.", entry.Team),
			})
		}
	}

	if len(results) == 0 {
		results = append(results, searchRow{
			Title:   "No matching operators",
			Meta:    "Try team names like Inventory, CX, or Warehouse.",
			Score:   "0 matches",
			Preview: "The urgent input still committed. Only the derived result set missed.",
		})
	}

	return searchSnapshot{
		Query:       query,
		RowsScanned: rowsScanned,
		Matched:     matched,
		Results:     results,
	}
}

func buildDashboardSnapshot(name string) dashboardSnapshot {
	if snapshot, ok := dashboardSeeds[name]; ok {
		return snapshot
	}
	return dashboardSeeds["Pipeline"]
}

func buildRouteSnapshot(path string) routeSnapshot {
	if snapshot, ok := routeSeeds[path]; ok {
		return snapshot
	}
	return routeSeeds["/orders"]
}

func transitionButton(label string, active bool, handler ui.Handler) ui.Node {
	className := "rounded-full border px-4 py-2 text-sm font-semibold transition-colors"
	if active {
		className += " border-cyan-300/70 bg-cyan-300/15 text-cyan-100"
	} else {
		className += " border-white/10 bg-white/5 text-slate-200 hover:bg-white/10"
	}
	return html.Button(html.Props{Type: "button", OnClick: handler, Class: className}, html.Text(label))
}

func transitionHooksExample() ui.Node {
	transition := ui.UseTransition()

	query := ui.UseState("")
	search := ui.UseState(buildSearchSnapshot(""))
	requestedTab := ui.UseState("Pipeline")
	dashboard := ui.UseState(buildDashboardSnapshot("Pipeline"))
	requestedPath := ui.UseState("/orders")
	routeView := ui.UseState(buildRouteSnapshot("/orders"))
	workLabel := ui.UseState("Idle")
	lastCommit := ui.UseState("Initial render committed")

	switchTab := func(name string) {
		requestedTab.Set(name)
		workLabel.Set("Switching dashboard to " + name)
		transition.Start(func() {
			dashboard.Set(buildDashboardSnapshot(name))
			lastCommit.Set("Tab commit: " + name)
			workLabel.Set(name + " dashboard ready")
		})
	}
	switchRoute := func(path string) {
		requestedPath.Set(path)
		workLabel.Set("Preparing section " + path)
		transition.Start(func() {
			routeView.Set(buildRouteSnapshot(path))
			lastCommit.Set("Route commit: " + path)
			workLabel.Set("Section ready at " + path)
		})
	}

	updateSearch := ui.UseEvent(func(event ui.InputEvent) {
		next := event.GetValue()
		query.Set(next)
		workLabel.Set("Filtering " + fmt.Sprintf("%d", len(directorySeed)*18) + " directory rows")
		transition.Start(func() {
			snapshot := buildSearchSnapshot(next)
			search.Set(snapshot)
			lastCommit.Set(fmt.Sprintf("Search commit: %d matches", snapshot.Matched))
			workLabel.Set("Filtered results ready")
		})
	})

	openPipeline := ui.UseEvent(func() { switchTab("Pipeline") })
	openCapacity := ui.UseEvent(func() { switchTab("Capacity") })
	openExperience := ui.UseEvent(func() { switchTab("Experience") })
	openOrders := ui.UseEvent(func() { switchRoute("/orders") })
	openInventory := ui.UseEvent(func() { switchRoute("/inventory") })
	openFulfillment := ui.UseEvent(func() { switchRoute("/fulfillment") })

	status := "Idle"
	if transition.Pending() {
		status = "Transition pending"
	}
	statusDetail := lastCommit.Get()
	if transition.Pending() {
		statusDetail = workLabel.Get()
	}

	resultCards := make([]ui.Node, 0, len(search.Get().Results))
	for _, result := range search.Get().Results {
		resultCards = append(resultCards, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 p-4"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(result.Title)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(result.Meta)),
			html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(result.Score)),
			html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-400"}, html.Text(result.Preview)),
		))
	}

	metricCards := make([]ui.Node, 0, len(dashboard.Get().Metrics))
	for _, metric := range dashboard.Get().Metrics {
		metricCards = append(metricCards, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 p-4"},
			html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(metric.Label)),
			html.P(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(metric.Value)),
		))
	}

	activityItems := make([]ui.Node, 0, len(dashboard.Get().Activity))
	for _, item := range dashboard.Get().Activity {
		activityItems = append(activityItems, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-3 text-sm leading-6 text-slate-300"}, html.Text(item)))
	}

	routeHighlights := make([]ui.Node, 0, len(routeView.Get().Highlights))
	for _, item := range routeView.Get().Highlights {
		routeHighlights = append(routeHighlights, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-3 text-sm leading-6 text-slate-300"}, html.Text(item)))
	}

	return shared.ExamplePage(
		"ui.StartTransition / ui.UseTransition",
		"Transition-style UX for search, tabs, and section swaps",
		"Use transitions when the user should see urgent intent commit first, while heavier derived results, dashboards, or route-sized sections finish in a separate non-urgent pass.",
		shared.ExamplePanel("Scheduler state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Scheduler", status),
				shared.ExampleStat("Requested tab", requestedTab.Get()),
				shared.ExampleStat("Requested path", requestedPath.Get()),
				shared.ExampleStat("Last commit", statusDetail),
			),
			html.P(html.Props{Class: "mt-6 text-sm leading-7 text-slate-300"}, html.Text("This page keeps urgent controls responsive, then moves the larger result list, dashboard panel, and route-sized section swap into transition-marked work so each intent reads clearly.")),
		),
		shared.ExamplePanel("Typeahead filtering",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Typing updates the input immediately. The heavier directory filtering runs inside a transition so the query and pending state change before the result pane settles.")),
			html.Input(html.Props{
				Value:       query.Get(),
				OnInput:     updateSearch,
				Placeholder: "Search by name, team, region, or status",
				Class:       "mt-4 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500",
			}),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Rows scanned", fmt.Sprintf("%d", search.Get().RowsScanned)),
				shared.ExampleStat("Matches", fmt.Sprintf("%d", search.Get().Matched)),
				shared.ExampleStat("Current query", fallback(search.Get().Query, "All operators")),
			),
			html.Ul(html.Props{Class: "mt-6 grid gap-3 lg:grid-cols-2"}, resultCards...),
		),
		shared.ExamplePanel("Tab switches",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Tab clicks mark the next dashboard as requested right away. The metric cards and activity feed commit afterward through the transition lane.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				transitionButton("Pipeline", requestedTab.Get() == "Pipeline", openPipeline),
				transitionButton("Capacity", requestedTab.Get() == "Capacity", openCapacity),
				transitionButton("Experience", requestedTab.Get() == "Experience", openExperience),
			),
			html.P(html.Props{Class: "mt-5 text-sm leading-7 text-slate-300"}, html.Text(dashboard.Get().Summary)),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"}, metricCards...),
			html.Ul(html.Props{Class: "mt-6 grid gap-3"}, activityItems...),
		),
		shared.ExamplePanel("Route-style section swaps",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("These buttons simulate route-sized view changes. Requested navigation updates immediately, while the committed section content swaps after the non-urgent render path finishes.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				transitionButton("/orders", requestedPath.Get() == "/orders", openOrders),
				transitionButton("/inventory", requestedPath.Get() == "/inventory", openInventory),
				transitionButton("/fulfillment", requestedPath.Get() == "/fulfillment", openFulfillment),
			),
			html.Div(html.Props{Class: "mt-6 rounded-[1.75rem] border border-cyan-300/15 bg-[linear-gradient(160deg,rgba(12,74,110,0.20),rgba(15,23,42,0.92))] p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-cyan-300"}, html.Text(routeView.Get().Path)),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(routeView.Get().Title)),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-200"}, html.Text(routeView.Get().Summary)),
				html.Ul(html.Props{Class: "mt-6 grid gap-3"}, routeHighlights...),
			),
		),
	)
}

func fallback(value string, empty string) string {
	if strings.TrimSpace(value) == "" {
		return empty
	}
	return value
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(transitionHooksExample), "#app")
	select {}
}
