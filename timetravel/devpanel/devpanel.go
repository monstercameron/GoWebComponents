// Package devpanel renders the time-travel devtools UI as a GoWebComponents component — the
// framework dogfooding its own engine. It is the visual consumer of the pure
// timetravel.History engine (C4/FB6): a scrubber timeline plus step-back/step-forward
// controls. The engine stays free of any UI dependency; this package is the thin view over
// it.
package devpanel

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// Model is the read view a time-travel panel needs — satisfied by *timetravel.History[T] for
// any T, so the panel renders any history without being generic itself.
type Model interface {
	Labels() []string
	Cursor() int
}

// Props configures a Panel: the history to display and the navigation callbacks (each wired
// to the matching timetravel.History method by the host, which also triggers its own
// re-render after navigating).
type Props struct {
	Model   Model
	OnUndo  func()
	OnRedo  func()
	OnScrub func(parseIndex int)
}

// Panel renders the time-travel timeline: a row of step-back/step-forward controls and a
// list of snapshot labels with the current one marked (aria-current) and each clickable to
// scrub to that point. Drop it into a devtools surface; restyle via the gwc-timetravel
// classes.
func Panel(parseProps Props) ui.Node {
	parseLabels := []string{}
	parseCursor := 0
	if parseProps.Model != nil {
		parseLabels = parseProps.Model.Labels()
		parseCursor = parseProps.Model.Cursor()
	}

	parseItems := make([]ui.Node, 0, len(parseLabels))
	for parseIndex, parseLabel := range parseLabels {
		parseIsCurrent := parseIndex == parseCursor
		parseClass := "gwc-timetravel-item"
		parseAria := map[string]string{}
		if parseIsCurrent {
			parseClass += " gwc-timetravel-current"
			parseAria["current"] = "true"
		}
		parseIndexCopy := parseIndex
		parseItems = append(parseItems, html.Tag("li", html.Props{
			Class:   parseClass,
			Role:    "button",
			Aria:    parseAria,
			Raw:     map[string]any{"tabindex": "0", "data-index": strconv.Itoa(parseIndex)},
			OnClick: ui.UseEvent(func() { scrub(parseProps, parseIndexCopy) }),
		}, html.Text(parseLabel)))
	}

	return html.Div(html.Props{Class: "gwc-timetravel", Role: "group"},
		html.Div(html.Props{Class: "gwc-timetravel-controls"},
			html.Button(html.Props{
				Class:   "gwc-timetravel-undo",
				Type:    "button",
				OnClick: ui.UseEvent(func() { invoke(parseProps.OnUndo) }),
				Aria:    map[string]string{"label": "Step back"},
			}, html.Text("◀ Back")),
			html.Button(html.Props{
				Class:   "gwc-timetravel-redo",
				Type:    "button",
				OnClick: ui.UseEvent(func() { invoke(parseProps.OnRedo) }),
				Aria:    map[string]string{"label": "Step forward"},
			}, html.Text("Forward ▶")),
		),
		html.Tag("ol", html.Props{Class: "gwc-timetravel-timeline"}, parseItems...),
	)
}

// scrub invokes the scrub callback for an index when present.
func scrub(parseProps Props, parseIndex int) {
	if parseProps.OnScrub != nil {
		parseProps.OnScrub(parseIndex)
	}
}

// invoke calls a callback when present.
func invoke(parseFn func()) {
	if parseFn != nil {
		parseFn()
	}
}
