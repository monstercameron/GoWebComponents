package main

import (
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type serverInteractiveState struct {
	Version          int
	ActiveUsers      int
	PendingApprovals int
	IncidentsToday   int
	RecentEvents     []string
	UpdatedAt        time.Time
}

type serverInteractiveAction struct {
	Action string `json:"action"`
}

type serverInteractiveSnapshot struct {
	HTML      string                 `json:"html"`
	Version   int                    `json:"version"`
	UpdatedAt string                 `json:"updatedAt"`
	State     serverInteractiveState `json:"state"`
}

type serverInteractiveView struct {
	State   serverInteractiveState
	Status  string
	IsBusy  bool
	Actions map[string]ui.Handler
}

// renderServerInteractiveView builds every visible dashboard element through GWC.
func renderServerInteractiveView(parseView serverInteractiveView) ui.Node {
	parseActions := []ui.Node{}
	for _, parseAction := range []struct{ ID, Label string }{
		{"add-user", "Add User"}, {"resolve-approval", "Resolve Approval"},
		{"add-incident", "Add Incident"}, {"clear-incidents", "Clear Incidents"},
	} {
		parseActions = append(parseActions, html.Button(html.Props{
			Type: "button", Disabled: parseView.IsBusy,
			Data: map[string]string{"action": parseAction.ID}, OnClick: parseView.Actions[parseAction.ID],
		}, html.Text(parseAction.Label)))
	}
	parseEvents := make([]ui.Node, 0, len(parseView.State.RecentEvents))
	for _, parseEvent := range parseView.State.RecentEvents {
		parseEvents = append(parseEvents, html.Li(html.Props{}, html.Text(parseEvent)))
	}
	return html.Main(html.Props{},
		html.H1(html.Props{}, html.Text("Server-Interactive Dashboard POC")),
		html.P(html.Props{Class: "subtitle"}, html.Text("The server owns state and streams snapshots over SSE. GWC owns every dashboard element and reconciles each snapshot without replacing raw HTML.")),
		html.Div(html.Props{Class: "layout"},
			html.Section(html.Props{Class: "panel-grid"},
				renderServerInteractiveMetricNode("Active Users", parseView.State.ActiveUsers),
				renderServerInteractiveMetricNode("Pending Approvals", parseView.State.PendingApprovals),
				renderServerInteractiveMetricNode("Incidents Today", parseView.State.IncidentsToday)),
			html.Section(html.Props{Class: "action-panel"}, html.H2(html.Props{}, html.Text("Operator Actions")), html.Div(html.Props{Class: "actions"}, parseActions...)),
			html.Section(html.Props{Class: "events-panel"},
				html.H2(html.Props{}, html.Text("Recent Server Events")), html.Ul(html.Props{}, parseEvents...),
				html.P(html.Props{Class: "meta"}, html.Text(fmt.Sprintf("Version %d · Updated %s UTC", parseView.State.Version, parseView.State.UpdatedAt.Format("15:04:05")))))),
		html.P(html.Props{ID: "status", Class: "status", Role: "status"}, html.Text(parseView.Status)),
	)
}

// renderServerInteractiveMetricNode builds one text-safe metric tile.
func renderServerInteractiveMetricNode(parseLabel string, parseValue int) ui.Node {
	return html.Article(html.Props{Class: "metric"},
		html.P(html.Props{Class: "metric-label"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "metric-value"}, html.Text(fmt.Sprint(parseValue))))
}
