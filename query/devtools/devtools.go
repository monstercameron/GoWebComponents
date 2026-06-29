// Package devtools renders the query cache's observability panel as a GoWebComponents
// component (D2). It is the visual view over Cache.Inspect: a table of cached keys with
// their freshness and in-flight status, so a developer can see the data layer's state in
// the app itself. The framework observes its own runtime.
package devtools

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/query"
	"github.com/monstercameron/GoWebComponents/ui"
)

// CachePanel renders a query cache's current entries: one row per key with its state
// (fetching / stale / fresh / empty). Restyle via the gwc-query-devtools classes.
func CachePanel(parseCache *query.Cache) ui.Node {
	if parseCache == nil {
		return html.Div(html.Props{Class: "gwc-query-devtools"}, html.Text("no cache"))
	}

	parseEntries := parseCache.Inspect()
	parseRows := make([]ui.Node, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		parseRows = append(parseRows, html.Tag("li", html.Props{
			Class: "gwc-query-devtools-row",
			Raw:   map[string]any{"data-key": parseEntry.Key},
		},
			html.Tag("span", html.Props{Class: "gwc-query-devtools-key"}, html.Text(parseEntry.Key)),
			html.Tag("span", html.Props{Class: "gwc-query-devtools-status"}, html.Text(entryStatus(parseEntry))),
		))
	}

	return html.Div(html.Props{Class: "gwc-query-devtools", Role: "table"},
		html.Tag("h3", html.Props{Class: "gwc-query-devtools-title"}, html.Text("Query cache")),
		html.Tag("ul", html.Props{Class: "gwc-query-devtools-rows"}, parseRows...),
	)
}

// entryStatus renders a single human-readable status label for one entry.
func entryStatus(parseEntry query.EntryInfo) string {
	switch {
	case parseEntry.Fetching:
		return "fetching"
	case !parseEntry.HasData:
		return "empty"
	case parseEntry.Stale:
		return "stale"
	default:
		return "fresh"
	}
}
