//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type testerFakeTransport struct {
	parseMethods []string
	parseStarts  []string
	parseArgs    []json.RawMessage
}

// Capabilities advertises the test transport's configured methods and portable features.
func (parseFake *testerFakeTransport) Capabilities() (desktop.Capabilities, error) {
	return desktop.Capabilities{Protocol: desktop.ProtocolVersion, Methods: parseFake.parseMethods}, nil
}

// Start records one dispatched request.
func (parseFake *testerFakeTransport) Start(parseMethod string, parseArgs json.RawMessage) (string, error) {
	parseFake.parseStarts = append(parseFake.parseStarts, parseMethod)
	parseFake.parseArgs = append(parseFake.parseArgs, append(json.RawMessage(nil), parseArgs...))
	return "request-1", nil
}

// Poll completes tester requests with a valid empty screen list.
func (parseFake *testerFakeTransport) Poll(string) (desktop.Reply, error) {
	return desktop.Reply{Done: true, Data: json.RawMessage(`{"version":1,"data":[]}`)}, nil
}

// Cancel completes transport cleanup.
func (parseFake *testerFakeTransport) Cancel(string) error { return nil }

// Listen is unused by tester action tests.
func (parseFake *testerFakeTransport) Listen(string) (string, error) { return "", nil }

// Next is unused by tester action tests.
func (parseFake *testerFakeTransport) Next(string) (desktop.Reply, error) {
	return desktop.Reply{}, nil
}

// Unlisten is unused by tester action tests.
func (parseFake *testerFakeTransport) Unlisten(string) error { return nil }

// TestTesterInteractiveTimeouts limits longer waits to explicitly interactive cases.
func TestTesterInteractiveTimeouts(parseTest *testing.T) {
	parseInteractive := map[string]bool{"open-file": true, "open-files": true, "open-directory": true, "save-path": true, "message-info": true, "message-warning": true, "message-error": true, "message-question": true, "export-report": true, "window-print": true}
	for _, parseCase := range getTesterManualCases() {
		parseExpected := 30 * time.Second
		if parseInteractive[parseCase.Action] {
			parseExpected = 5 * time.Minute
		}
		if parseActual := getTesterTimeout(parseCase.Action); parseActual != parseExpected {
			parseTest.Fatalf("%s timeout=%v want=%v", parseCase.Action, parseActual, parseExpected)
		}
	}
}

// TestTesterPortableActionDispatchesOnce prevents duplicate legacy and portable calls.
func TestTesterPortableActionDispatchesOnce(parseTest *testing.T) {
	parseTransport := &testerFakeTransport{parseMethods: []string{desktop.ScreensMethod, "api.run"}}
	_, parseErr := callTesterAction(context.Background(), desktop.NewClient(parseTransport), "screens")
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if len(parseTransport.parseStarts) != 1 || parseTransport.parseStarts[0] != desktop.ScreensMethod {
		parseTest.Fatalf("dispatches=%v want one %s call", parseTransport.parseStarts, desktop.ScreensMethod)
	}
}

// TestTesterPortableControlsRequireTheirFeature prevents unsupported controls from appearing actionable.
func TestTesterPortableControlsRequireTheirFeature(parseTest *testing.T) {
	parseClient := desktop.NewClient(&testerFakeTransport{parseMethods: []string{"api.run"}})
	if isTesterActionAvailable(parseClient, "window-info") || isTesterActionAvailable(parseClient, "menu-replace") || isTesterActionAvailable(parseClient, "autostart-enable") {
		parseTest.Fatal("portable control enabled without its advertised feature")
	}
	if !isTesterActionAvailable(parseClient, "open-file") {
		parseTest.Fatal("legacy APIService picker unexpectedly disabled")
	}
}

// TestTesterTrayMenuUsesSupportedPresentations keeps the interactive fixture valid on Windows.
func TestTesterTrayMenuUsesSupportedPresentations(parseTest *testing.T) {
	parseTransport := &testerFakeTransport{parseMethods: []string{desktop.TrayConfigureMethod}}
	parseMenu := getTesterTrayMenu()
	parseErr := desktop.NewClient(parseTransport).ConfigureTray(context.Background(), desktop.TrayRequest{Action: "upsert", ID: "tester-tray", Tooltip: "tester", Menu: &parseMenu})
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if len(parseTransport.parseStarts) != 1 || parseTransport.parseStarts[0] != desktop.TrayConfigureMethod {
		parseTest.Fatalf("dispatches=%v", parseTransport.parseStarts)
	}
}

// TestTesterChildWindowUsesRegisteredTemplate keeps the fixture aligned with the example host configuration.
func TestTesterChildWindowUsesRegisteredTemplate(parseTest *testing.T) {
	parseRequest := getTesterChildWindowRequest()
	if parseRequest.TemplateID != "counter" || parseRequest.ID != "tester-observer" {
		parseTest.Fatalf("child request=%+v", parseRequest)
	}
}

// TestTesterPortableErrorsAreFailed prevents errors from being presented as completed actions.
func TestTesterPortableErrorsAreFailed(parseTest *testing.T) {
	parseResult, parseErr := callTesterAction(context.Background(), desktop.Client{}, "child-create")
	if parseErr == nil || parseResult.Outcome != "failed" || parseResult.Detail != "" {
		parseTest.Fatalf("result=%+v error=%v", parseResult, parseErr)
	}
}

// TestTesterWindowRequestsOnlyResizeWithDimensions keeps payloads valid for the SDK validator.
func TestTesterWindowRequestsOnlyResizeWithDimensions(parseTest *testing.T) {
	for _, parseAction := range []string{"info", "center", "enable-size-constraints", "disable-size-constraints", "minimize", "maximize", "restore", "fullscreen", "unfullscreen", "focus", "zoom-in", "zoom-out", "zoom-reset"} {
		parseRequest := getTesterWindowRequest(parseAction)
		if parseRequest.Action != parseAction || parseRequest.Width != 0 || parseRequest.Height != 0 {
			parseTest.Fatalf("%s request=%+v", parseAction, parseRequest)
		}
	}
	parseResize := getTesterWindowRequest("resize")
	if parseResize.Width != 720 || parseResize.Height != 520 {
		parseTest.Fatalf("resize request=%+v", parseResize)
	}
	parsePosition := getTesterWindowRequest("set-position")
	if parsePosition.X != 120 || parsePosition.Y != 120 {
		parseTest.Fatalf("position request=%+v", parsePosition)
	}
	for _, parseAction := range []string{"set-always-on-top", "set-resizable", "flash", "set-content-protection"} {
		if parseRequest := getTesterWindowRequest(parseAction); !parseRequest.Enabled {
			parseTest.Fatalf("%s request=%+v", parseAction, parseRequest)
		}
	}
}

// TestTesterManualCasesCoverNativeActions verifies every button can receive a human verdict.
func TestTesterManualCasesCoverNativeActions(parseTest *testing.T) {
	parseSeen := map[string]bool{}
	for _, parseCase := range getTesterManualCases() {
		if parseCase.Action == "" || parseCase.Label == "" || parseSeen[parseCase.Action] {
			parseTest.Fatalf("invalid/duplicate case: %+v", parseCase)
		}
		parseSeen[parseCase.Action] = true
	}
	for _, parseGroup := range testerCases {
		for _, parseCase := range parseGroup.Items {
			if !parseSeen[parseCase.Action] {
				parseTest.Fatal(parseCase.Action)
			}
		}
	}
	for _, parseAction := range []string{"context-check", "context-radio", "context-disabled", "context-dismiss", "menu-action", "shortcut", "keyboard-focus", "keyboard-ime", "edit-menu", "second-window", "export-report"} {
		if !parseSeen[parseAction] {
			parseTest.Fatalf("missing manual case %s", parseAction)
		}
	}
}

// TestTesterTaskShowsActualOutcome keeps operation results, paths and errors visible.
func TestTesterTaskShowsActualOutcome(parseTest *testing.T) {
	parseResult := contracts.APIResult{Outcome: "cancelled", Detail: "picker dismissed", WindowID: "7", Paths: []string{"report.json"}}
	parseText := formatTesterTask(ui.TaskState[contracts.APIResult]{Value: parseResult, Ready: true})
	for _, parsePart := range []string{"cancelled", "picker dismissed", "window 7", "report.json"} {
		if !strings.Contains(parseText, parsePart) {
			parseTest.Fatal(parseText)
		}
	}
	if parseText = formatTesterTask(ui.TaskState[contracts.APIResult]{Error: errors.New("export refused")}); !strings.Contains(parseText, "export refused") {
		parseTest.Fatal(parseText)
	}
	if parseText = formatTesterTask(ui.TaskState[contracts.APIResult]{Cancelled: true}); !strings.Contains(parseText, "OS dialog may still need dismissal") {
		parseTest.Fatal(parseText)
	}
}

// TestTesterPollSuppressesLateState covers both successful and failed reads after unmount.
func TestTesterPollSuppressesLateState(parseTest *testing.T) {
	for _, parseReadErr := range []error{nil, errors.New("cancelled transport")} {
		parseContext, parseCancel := context.WithCancel(context.Background())
		parseReads, parseWrites := 0, 0
		pollTesterReport(parseContext, func(context.Context) (contracts.APIReport, error) {
			parseReads++
			parseCancel()
			return contracts.APIReport{}, parseReadErr
		}, func(contracts.APIReport) { parseWrites++ }, func(error) { parseWrites++ })
		if parseReads != 1 || parseWrites != 0 {
			parseTest.Fatalf("reads=%d late writes=%d", parseReads, parseWrites)
		}
	}
}
