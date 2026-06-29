//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
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

func buildSearchSnapshot(parseQuery string) searchSnapshot {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseQuery))
	parseResults := make([]searchRow, 0, 6)
	parseMatched := 0
	parseRowsScanned := len(directorySeed) * 18

	for parseCycle := 0; parseCycle < 18; parseCycle++ {
		for _, parseEntry := range directorySeed {
			parseSearchable := strings.ToLower(parseEntry.Name + " " + parseEntry.Team + " " + parseEntry.Region + " " + parseEntry.Status)
			if parseNormalized != "" && !strings.Contains(parseSearchable, parseNormalized) {
				continue
			}
			parseMatched++
			if len(parseResults) >= 6 {
				continue
			}
			parseResults = append(parseResults, searchRow{
				Title:   fmt.Sprintf("%s #%02d", parseEntry.Name, parseCycle+1),
				Meta:    fmt.Sprintf("%s - %s", parseEntry.Team, parseEntry.Region),
				Score:   fmt.Sprintf("status: %s", parseEntry.Status),
				Preview: fmt.Sprintf("Result rows stay responsive because the input value commits before the heavier filter pass finishes for %s.", parseEntry.Team),
			})
		}
	}

	if len(parseResults) == 0 {
		parseResults = append(parseResults, searchRow{
			Title:   "No matching operators",
			Meta:    "Try team names like Inventory, CX, or Warehouse.",
			Score:   "0 matches",
			Preview: "The urgent input still committed. Only the derived result set missed.",
		})
	}

	return searchSnapshot{
		Query:       parseQuery,
		RowsScanned: parseRowsScanned,
		Matched:     parseMatched,
		Results:     parseResults,
	}
}

func buildDashboardSnapshot(parseName string) dashboardSnapshot {
	if parseSnapshot, parseOk := dashboardSeeds[parseName]; parseOk {
		return parseSnapshot
	}
	return dashboardSeeds["Pipeline"]
}

func buildRouteSnapshot(parsePath string) routeSnapshot {
	if parseSnapshot, parseOk := routeSeeds[parsePath]; parseOk {
		return parseSnapshot
	}
	return routeSeeds["/orders"]
}

func transitionButton(parseLabel string, isActive bool, parseHandler ui.Handler) ui.Node {
	parseClassName := "rounded-full border px-4 py-2 text-sm font-semibold transition-colors"
	if isActive {
		parseClassName += " border-cyan-300/70 bg-cyan-300/15 text-cyan-100"
	} else {
		parseClassName += " border-white/10 bg-white/5 text-slate-200 hover:bg-white/10"
	}
	return html.Button(html.Props{Type: "button", OnClick: parseHandler, Class: parseClassName}, html.Text(parseLabel))
}

func transitionHooksExample() ui.Node {
	parseTransition := ui.UseTransition()

	parseQuery := ui.UseState("")
	parseSearch := ui.UseState(buildSearchSnapshot(""))
	parseRequestedTab := ui.UseState("Pipeline")
	parseDashboard := ui.UseState(buildDashboardSnapshot("Pipeline"))
	parseRequestedPath := ui.UseState("/orders")
	parseRouteView := ui.UseState(buildRouteSnapshot("/orders"))
	parseWorkLabel := ui.UseState("Idle")
	parseLastCommit := ui.UseState("Initial render committed")

	parseSwitchTab := func(parseName string) {
		parseRequestedTab.Set(parseName)
		parseWorkLabel.Set("Switching dashboard to " + parseName)
		parseTransition.Start(func() {
			parseDashboard.Set(buildDashboardSnapshot(parseName))
			parseLastCommit.Set("Tab commit: " + parseName)
			parseWorkLabel.Set(parseName + " dashboard ready")
		})
	}
	parseSwitchRoute := func(parsePath string) {
		parseRequestedPath.Set(parsePath)
		parseWorkLabel.Set("Preparing section " + parsePath)
		parseTransition.Start(func() {
			parseRouteView.Set(buildRouteSnapshot(parsePath))
			parseLastCommit.Set("Route commit: " + parsePath)
			parseWorkLabel.Set("Section ready at " + parsePath)
		})
	}

	parseUpdateSearch := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseNext := parseEvent.GetValue()
		parseQuery.Set(parseNext)
		parseWorkLabel.Set("Filtering " + fmt.Sprintf("%d", len(directorySeed)*18) + " directory rows")
		parseTransition.Start(func() {
			parseSnapshot := buildSearchSnapshot(parseNext)
			parseSearch.Set(parseSnapshot)
			parseLastCommit.Set(fmt.Sprintf("Search commit: %d matches", parseSnapshot.Matched))
			parseWorkLabel.Set("Filtered results ready")
		})
	})

	parseOpenPipeline := ui.UseEvent(func() { parseSwitchTab("Pipeline") })
	parseOpenCapacity := ui.UseEvent(func() { parseSwitchTab("Capacity") })
	parseOpenExperience := ui.UseEvent(func() { parseSwitchTab("Experience") })
	parseOpenOrders := ui.UseEvent(func() { parseSwitchRoute("/orders") })
	parseOpenInventory := ui.UseEvent(func() { parseSwitchRoute("/inventory") })
	parseOpenFulfillment := ui.UseEvent(func() { parseSwitchRoute("/fulfillment") })

	parseStatus := "Idle"
	if parseTransition.Pending() {
		parseStatus = "Transition pending"
	}
	parseStatusDetail := parseLastCommit.Get()
	if parseTransition.Pending() {
		parseStatusDetail = parseWorkLabel.Get()
	}

	parseResultCards := make([]ui.Node, 0, len(parseSearch.Get().Results))
	for _, parseResult := range parseSearch.Get().Results {
		parseResultCards = append(parseResultCards, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 p-4"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseResult.Title)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text(parseResult.Meta)),
			html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text(parseResult.Score)),
			html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-400"}, html.Text(parseResult.Preview)),
		))
	}

	parseMetricCards := make([]ui.Node, 0, len(parseDashboard.Get().Metrics))
	for _, parseMetric := range parseDashboard.Get().Metrics {
		parseMetricCards = append(parseMetricCards, html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 p-4"},
			html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseMetric.Label)),
			html.P(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(parseMetric.Value)),
		))
	}

	parseActivityItems := make([]ui.Node, 0, len(parseDashboard.Get().Activity))
	for _, parseItem := range parseDashboard.Get().Activity {
		parseActivityItems = append(parseActivityItems, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-3 text-sm leading-6 text-slate-300"}, html.Text(parseItem)))
	}

	parseRouteHighlights := make([]ui.Node, 0, len(parseRouteView.Get().Highlights))
	for _, parseItem2 := range parseRouteView.Get().Highlights {
		parseRouteHighlights = append(parseRouteHighlights, html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-3 text-sm leading-6 text-slate-300"}, html.Text(parseItem2)))
	}

	return shared.ExamplePage(
		"ui.StartTransition / ui.UseTransition",
		"Transition-style UX for search, tabs, and section swaps",
		"Use transitions when the user should see urgent intent commit first, while heavier derived results, dashboards, or route-sized sections finish in a separate non-urgent pass.",
		shared.ExamplePanel("Scheduler state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Scheduler", parseStatus),
				shared.ExampleStat("Requested tab", parseRequestedTab.Get()),
				shared.ExampleStat("Requested path", parseRequestedPath.Get()),
				shared.ExampleStat("Last commit", parseStatusDetail),
			),
			html.P(html.Props{Class: "mt-6 text-sm leading-7 text-slate-300"}, html.Text("This page keeps urgent controls responsive, then moves the larger result list, dashboard panel, and route-sized section swap into transition-marked work so each intent reads clearly.")),
		),
		shared.ExamplePanel("Typeahead filtering",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Typing updates the input immediately. The heavier directory filtering runs inside a transition so the query and pending state change before the result pane settles.")),
			html.Input(html.Props{
				Value:       parseQuery.Get(),
				OnInput:     parseUpdateSearch,
				Placeholder: "Search by name, team, region, or status",
				Class:       "mt-4 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500",
			}),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Rows scanned", fmt.Sprintf("%d", parseSearch.Get().RowsScanned)),
				shared.ExampleStat("Matches", fmt.Sprintf("%d", parseSearch.Get().Matched)),
				shared.ExampleStat("Current query", fallback(parseSearch.Get().Query, "All operators")),
			),
			html.Ul(html.Props{Class: "mt-6 grid gap-3 lg:grid-cols-2"}, parseResultCards...),
		),
		shared.ExamplePanel("Tab switches",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Tab clicks mark the next dashboard as requested right away. The metric cards and activity feed commit afterward through the transition lane.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				transitionButton("Pipeline", parseRequestedTab.Get() == "Pipeline", parseOpenPipeline),
				transitionButton("Capacity", parseRequestedTab.Get() == "Capacity", parseOpenCapacity),
				transitionButton("Experience", parseRequestedTab.Get() == "Experience", parseOpenExperience),
			),
			html.P(html.Props{Class: "mt-5 text-sm leading-7 text-slate-300"}, html.Text(parseDashboard.Get().Summary)),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"}, parseMetricCards...),
			html.Ul(html.Props{Class: "mt-6 grid gap-3"}, parseActivityItems...),
		),
		shared.ExamplePanel("Route-style section swaps",
			html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("These buttons simulate route-sized view changes. Requested navigation updates immediately, while the committed section content swaps after the non-urgent render path finishes.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				transitionButton("/orders", parseRequestedPath.Get() == "/orders", parseOpenOrders),
				transitionButton("/inventory", parseRequestedPath.Get() == "/inventory", parseOpenInventory),
				transitionButton("/fulfillment", parseRequestedPath.Get() == "/fulfillment", parseOpenFulfillment),
			),
			html.Div(html.Props{Class: "mt-6 rounded-[1.75rem] border border-cyan-300/15 bg-[linear-gradient(160deg,rgba(12,74,110,0.20),rgba(15,23,42,0.92))] p-6"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-cyan-300"}, html.Text(parseRouteView.Get().Path)),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(parseRouteView.Get().Title)),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-200"}, html.Text(parseRouteView.Get().Summary)),
				html.Ul(html.Props{Class: "mt-6 grid gap-3"}, parseRouteHighlights...),
			),
		),
	)
}

func fallback(parseValue string, parseEmpty string) string {
	if strings.TrimSpace(parseValue) == "" {
		return parseEmpty
	}
	return parseValue
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(transitionHooksExample))
	exampleboot.WaitExampleRuntime()
}
