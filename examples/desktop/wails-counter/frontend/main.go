//go:build js && wasm

package main

import (
	"context"
	"fmt"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v6/css"
	"github.com/monstercameron/GoWebComponents/v6/desktop"
	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/kvstate"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderCounter renders local state and lifecycle-owned native tasks.
func renderCounter() ui.Node {
	parseCount := ui.UseState(0)
	parseProgress := ui.UseState(0)
	parseEventError := ui.UseState("")
	parseBackend := ui.UseMemo(func() kvstate.PersistenceBackend {
		parseClient, parseErr := desktop.Connect()
		if parseErr != nil || !parseClient.Supports(desktop.PersistentStorage) {
			return nil
		}
		return desktop.NewStorageBackend(parseClient)
	})
	parseShared := kvstate.UsePersistedState("shared-counter", 0, kvstate.Options{Name: "wails-counter", Backend: parseBackend, Strategy: kvstate.Immediate{}, ExternalInvalidation: true})
	ui.UseEffect(func() func() {
		parseClient, parseErr := desktop.Connect()
		if parseErr != nil || !parseClient.Supports(desktop.PersistentStorage) {
			if parseErr != nil {
				parseEventError.Set(parseErr.Error())
			}
			return nil
		}
		parseStop, parseErr := desktop.SubscribeStorage(context.Background(), parseClient, "wails-counter", func(parseErr error) { parseEventError.Set(parseErr.Error()) })
		if parseErr != nil {
			parseEventError.Set(parseErr.Error())
			return nil
		}
		return parseStop
	}, true)
	parseSaveClick := ui.UseEvent(func() { parseShared.Set(parseShared.Get() + 1) })
	parseBridge := ui.UseTask(func(parseContext context.Context) (desktop.Client, error) {
		return desktop.Connect()
	})
	ui.UseEffect(func() func() { parseBridge.Start(); return nil }, true)
	parseBridgeState := parseBridge.Get()
	parseNative := ui.UseTask(func(parseContext context.Context) (contracts.CounterState, error) {
		return desktop.Call[contracts.CounterState](parseContext, parseBridge.Get().Value, "counter.increment")
	})
	parseDialog := ui.UseTask(func(parseContext context.Context) (string, error) {
		parseSelection, parseErr := parseBridge.Get().Value.OpenFile(parseContext, desktop.FileDialogOptions{Title: "Choose a file"})
		if parseErr != nil || parseSelection.Cancelled {
			return "", parseErr
		}
		return parseSelection.Paths[0], nil
	})
	parseProgressTask := ui.UseTask(func(parseContext context.Context) (any, error) {
		return desktop.Call[any](parseContext, parseBridge.Get().Value, "counter.progress")
	})
	parseReject := ui.UseTask(func(parseContext context.Context) (any, error) {
		return desktop.Call[any](parseContext, parseBridge.Get().Value, "counter.reject")
	})
	parseWork := ui.UseTask(func(parseContext context.Context) (any, error) {
		return desktop.Call[any](parseContext, parseBridge.Get().Value, "counter.work")
	})
	ui.UseEffect(func() func() {
		parseClient, parseErr := desktop.Connect()
		if parseErr != nil {
			parseEventError.Set(parseErr.Error())
			return nil
		}
		parseStop, parseErr := desktop.Subscribe[contracts.Progress](context.Background(), parseClient, "counter.progress", func(parseEvent contracts.Progress, parseDecodeErr error) {
			if parseDecodeErr != nil {
				parseEventError.Set(parseDecodeErr.Error())
				return
			}
			parseProgress.Set(parseEvent.Percent)
		})
		if parseErr != nil {
			parseEventError.Set(parseErr.Error())
			return nil
		}
		return parseStop
	}, true)
	parseIncrement := ui.UseEvent(func() { parseCount.Update(func(parseValue int) int { return parseValue + 1 }) })
	parseNativeClick := ui.UseEvent(func() { parseNative.Start() })
	parseDialogClick := ui.UseEvent(func() { parseDialog.Start() })
	parseProgressClick := ui.UseEvent(func() { parseProgressTask.Start() })
	parseRejectClick := ui.UseEvent(func() { parseReject.Start() })
	parseWorkClick := ui.UseEvent(func() { parseWork.Start() })
	parseCancelClick := ui.UseEvent(func() { parseWork.Cancel() })
	parseMessage := "Native bridge loading…"
	if parseBridgeState.Ready {
		parseMessage = "Native bridge ready"
	}
	if parseBridgeState.Error != nil {
		parseMessage = "Native bridge unavailable: " + parseBridgeState.Error.Error()
	}
	parseNativeState := parseNative.Get()
	if parseNativeState.Ready {
		parseMessage = fmt.Sprintf("Native response: %d", parseNativeState.Value.Value)
	}
	if parseNativeState.Error != nil {
		parseMessage = "Native call failed: " + parseNativeState.Error.Error()
	}
	parseDialogState := parseDialog.Get()
	parseDialogMessage := "No file selected"
	if parseDialogState.Ready {
		parseDialogMessage = "Selected: " + parseDialogState.Value
		if parseDialogState.Value == "" {
			parseDialogMessage = "Dialog cancelled"
		}
	}
	if parseDialogState.Error != nil {
		parseDialogMessage = "Dialog failed: " + parseDialogState.Error.Error()
	}
	parseErrorMessage := parseEventError.Get()
	if parseProgressTask.Get().Error != nil {
		parseErrorMessage = parseProgressTask.Get().Error.Error()
	}
	if parseReject.Get().Error != nil {
		parseErrorMessage = parseReject.Get().Error.Error()
	}
	if parseShared.Err() != nil {
		parseErrorMessage = parseShared.Err().Error()
	}
	parsePanelClass := css.Class(css.Padding(css.Px(24)), css.FontSize(css.Px(16)))
	return Div(parsePanelClass, H1(ID("route-title"), Text("Wails Counter")),
		P(ID("local-count"), Textf("Count: %d", parseCount.Get())),
		Div(Button(ID("local-increment"), OnClick(parseIncrement), Text("Increment")),
			Button(ID("native-increment"), OnClick(parseNativeClick), Disabled(!parseBridgeState.Ready || parseNativeState.Running), Text("Native round trip")),
			Button(ID("native-open-file"), OnClick(parseDialogClick), Disabled(!parseBridgeState.Ready || !parseBridgeState.Value.Supports(desktop.FileDialogs) || parseDialogState.Running), Text("Open file")),
			Button(ID("native-progress"), OnClick(parseProgressClick), Disabled(!parseBridgeState.Ready || parseProgressTask.Get().Running), Text("Progress")),
			Button(ID("native-reject"), OnClick(parseRejectClick), Disabled(!parseBridgeState.Ready || parseReject.Get().Running), Text("Test service error"))),
		P(ID("native-status"), Text(parseMessage)), P(ID("dialog-status"), Text(parseDialogMessage)), P(ID("native-error"), Text(parseErrorMessage)),
		P(ID("shared-count"), Textf("Saved count: %d", parseShared.Get())),
		Button(ID("shared-increment"), OnClick(parseSaveClick), Disabled(parseShared.Loading()), Text("Increment shared saved count")),
		Div(Button(ID("native-work"), OnClick(parseWorkClick), Disabled(!parseBridgeState.Ready || parseWork.Get().Running), Text("Start 10-second work")),
			Button(ID("native-cancel"), OnClick(parseCancelClick), Disabled(!parseWork.Get().Running), Text("Cancel native work"))),
		P(ID("progress-status"), Textf("Native progress: %d%%", parseProgress.Get())),
		Div(ClassStr("progress"), Div(ID("progress-value"), Style(map[string]string{"width": fmt.Sprintf("%d%%", parseProgress.Get())}))),
		P(A(ID("about-link"), Href("#/about"), Text("About this desktop example"))),
		P(A(ID("tester-link"), Href("#/tester"), Text("Windows API Lab"))))
}

// renderAbout provides a separate route so navigation exercises unmount cleanup.
func renderAbout() ui.Node {
	return Div(H1(ID("route-title"), Text("About this desktop example")), P(Text("GWC renders the UI inside a native Wails WebView.")), A(ID("counter-link"), Href("#/counter"), Text("Back to counter")))
}

// main mounts the framework hash router, which renders through ui.Render.
func main() {
	parseRouter := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/counter"})
	parseRouter.Register("/counter", func(router.Attrs) *router.Element { return ui.CreateElement(renderCounter) })
	parseRouter.Register("/about", func(router.Attrs) *router.Element { return ui.CreateElement(renderAbout) })
	parseRouter.Register("/tester", func(router.Attrs) *router.Element { return ui.CreateElement(renderTester) })
	parseRouter.Mount("#app")
	select {}
}
