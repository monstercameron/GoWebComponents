//go:build js && wasm

package main

import (
	"context"
	"encoding/json"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type dashboardResult struct {
	Status string `json:"status"`
	Output string `json:"output"`
}

// renderDashboard owns all harness controls while the child remains the measured subject.
func renderDashboard() ui.Node {
	parseRun := ui.UseTask(func(parseContext context.Context) (dashboardResult, error) {
		parseGlobal, parseErr := interop.GetGlobalThis()
		if parseErr != nil {
			return dashboardResult{}, parseErr
		}
		parseValue, parseErr := parseGlobal.AwaitCall(parseContext, "__gwcRunDashboardComparison")
		if parseErr != nil {
			return dashboardResult{}, parseErr
		}
		var parseResult dashboardResult
		parseErr = json.Unmarshal([]byte(parseValue.String()), &parseResult)
		return parseResult, parseErr
	})
	parseClick := ui.UseEvent(func() { parseRun.Start() })
	parseState := parseRun.Get()
	parseStatus, parseOutput := "Ready.", "No run yet."
	if parseState.Running {
		parseStatus, parseOutput = "Running… (this takes about a minute)", "Running…"
	} else if parseState.Error != nil {
		parseStatus, parseOutput = "ERROR", parseState.Error.Error()
	} else if parseState.Ready {
		parseStatus, parseOutput = parseState.Value.Status, parseState.Value.Output
	}
	return html.Div(html.Props{ID: "harness-dashboard"},
		html.H1(html.Props{}, html.Text("GWC v6 Load Harness")),
		html.P(html.Props{Class: "muted"}, html.Text("Measures whether main-thread frame time is invariant to workload size. Background workloads run while a typing probe drives real input; only the probe is measured. A drifted run fails regardless of its numbers.")),
		html.Div(html.Props{Class: "bar"},
			html.Button(html.Props{ID: "run", Type: "button", Disabled: parseState.Running, OnClick: parseClick}, html.Text("Run harness")),
			html.Span(html.Props{ID: "status", Class: "muted", Role: "status"}, html.Text(parseStatus)),
		),
		html.Div(html.Props{ID: "app"}, ui.Component(renderApp)),
		html.Div(html.Props{ID: "out"}, html.Text(parseOutput)),
	)
}
