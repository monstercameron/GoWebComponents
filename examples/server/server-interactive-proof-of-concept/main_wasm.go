//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/fetch"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderServerInteractiveClient reconciles bounded SSE state using public GWC hooks.
func renderServerInteractiveClient() ui.Node {
	parseStream := fetch.UseEventSource("/events", fetch.EventSourceOptions{MaxMessages: 1})
	parseAction := ui.UseRef("")
	parseTask := ui.UseTask(func(parseContext context.Context) (bool, error) {
		parseBody, parseErr := json.Marshal(serverInteractiveAction{Action: parseAction.Get()})
		if parseErr != nil {
			return false, parseErr
		}
		parseRequest, parseErr := http.NewRequestWithContext(parseContext, http.MethodPost, "/action", strings.NewReader(string(parseBody)))
		if parseErr != nil {
			return false, parseErr
		}
		parseRequest.Header.Set("Content-Type", "application/json")
		parseResponse, parseErr := http.DefaultClient.Do(parseRequest)
		if parseErr != nil {
			return false, parseErr
		}
		defer parseResponse.Body.Close()
		if parseResponse.StatusCode != http.StatusAccepted {
			return false, fmt.Errorf("action rejected: HTTP %d", parseResponse.StatusCode)
		}
		return true, nil
	})
	parseAddUser := ui.UseEvent(func() { parseAction.Set("add-user"); parseTask.Start() })
	parseResolve := ui.UseEvent(func() { parseAction.Set("resolve-approval"); parseTask.Start() })
	parseIncident := ui.UseEvent(func() { parseAction.Set("add-incident"); parseTask.Start() })
	parseClear := ui.UseEvent(func() { parseAction.Set("clear-incidents"); parseTask.Start() })
	parseConnection := parseStream.Get()
	parseTaskState := parseTask.Get()
	parseSnapshot := serverInteractiveSnapshot{}
	parseStatus := "Connecting to server stream..."
	if parseConnection.LastMessage.Data != "" {
		if parseErr := json.Unmarshal([]byte(parseConnection.LastMessage.Data), &parseSnapshot); parseErr != nil {
			parseStatus = "Invalid server snapshot: " + parseErr.Error()
		} else {
			parseStatus = fmt.Sprintf("Snapshot v%d applied at %s.", parseSnapshot.Version, parseSnapshot.UpdatedAt)
		}
	}
	if parseConnection.Error != nil {
		parseStatus = "Connection error: " + parseConnection.Error.Error()
	}
	if parseTaskState.Error != nil {
		parseStatus = "Action failed: " + parseTaskState.Error.Error()
	}
	return renderServerInteractiveView(serverInteractiveView{
		State: parseSnapshot.State, Status: parseStatus, IsBusy: parseTaskState.Running || !parseConnection.Open,
		Actions: map[string]ui.Handler{"add-user": parseAddUser, "resolve-approval": parseResolve, "add-incident": parseIncident, "clear-incidents": parseClear},
	})
}

// main hydrates the server-rendered GWC view while the native server retains all mutable business state.
func main() {
	if _, parseErr := ui.Hydrate(ui.Component(renderServerInteractiveClient), "#app"); parseErr != nil {
		panic(parseErr)
	}
	select {}
}
