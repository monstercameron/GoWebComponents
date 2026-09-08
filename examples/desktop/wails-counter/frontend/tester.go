//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v5/desktop"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

type testerCase struct{ ID, Label, Action string }
type testerGroup struct {
	ID, Title string
	Items     []testerCase
}
type testerActionProps struct {
	Client      desktop.Client
	Case        testerCase
	IsAvailable bool
	Refresh     func(context.Context) error
}
type testerNavProps struct{ Target, Label string }
type testerObservation struct{ ID, Outcome, Detail string }

var testerCases = []testerGroup{
	{"tester-dialogs", "01 / Dialogs & paths", []testerCase{{"api-open-file", "Open file", "open-file"}, {"api-open-files", "Open multiple files", "open-files"}, {"api-open-directory", "Choose directory", "open-directory"}, {"api-save-path", "Choose save path", "save-path"}}},
	{"tester-messages", "02 / Messages & clipboard", []testerCase{{"api-message-info", "Show information", "message-info"}, {"api-message-question", "Ask a question", "message-question"}, {"api-clipboard-write", "Write fixture to clipboard", "clipboard-write"}, {"api-clipboard-read", "Inspect clipboard fixture", "clipboard-read"}}},
	{"tester-window", "03 / Calling window", []testerCase{{"api-window-info", "Inspect window", "window-info"}, {"api-window-resize", "Resize to 720 × 520", "window-resize"}, {"api-window-maximize", "Maximize window", "window-maximize"}, {"api-window-restore", "Restore window", "window-restore"}, {"api-window-fullscreen", "Enter fullscreen", "window-fullscreen"}, {"api-window-unfullscreen", "Exit fullscreen", "window-unfullscreen"}}},
	{"tester-inspection", "04 / Screens", []testerCase{{"api-screens", "Inspect screens & DPI", "screens"}}},
}

// getTesterManualCases includes every native action and every human-only observation.
func getTesterManualCases() []testerCase {
	parseCases := []testerCase{}
	for _, parseGroup := range testerCases {
		parseCases = append(parseCases, parseGroup.Items...)
	}
	return append(parseCases,
		testerCase{"", "Context menu selection", "context-select"}, testerCase{"", "Context checkbox", "context-check"},
		testerCase{"", "Context radio group", "context-radio"}, testerCase{"", "Context submenu", "context-submenu"},
		testerCase{"", "Disabled menu action", "context-disabled"}, testerCase{"", "Escape dismisses menu", "context-dismiss"},
		testerCase{"", "Menu / Ctrl+Shift+K", "menu-action"}, testerCase{"", "F8 window shortcut", "shortcut"},
		testerCase{"", "Keyboard focus / Tab", "keyboard-focus"}, testerCase{"", "IME / Unicode typing", "keyboard-ime"},
		testerCase{"", "Native edit menu", "edit-menu"}, testerCase{"", "Physical resize / DPI", "resize-dpi"},
		testerCase{"", "Second window ownership", "second-window"}, testerCase{"", "Title-bar close", "window-close"},
		testerCase{"", "Report export", "export-report"})
}

// formatTesterResult shows the returned operation, not a synthetic success claim.
func formatTesterResult(parseResult contracts.APIResult) string {
	parseParts := []string{parseResult.Outcome}
	if parseResult.Detail != "" {
		parseParts = append(parseParts, parseResult.Detail)
	}
	if parseResult.WindowID != "" {
		parseParts = append(parseParts, "window "+parseResult.WindowID)
	}
	if len(parseResult.Paths) != 0 {
		parseParts = append(parseParts, strings.Join(parseResult.Paths, ", "))
	}
	return strings.Join(parseParts, " · ")
}

// formatTesterTask distinguishes cancellation of the caller's wait from OS dismissal.
func formatTesterTask(parseState ui.TaskState[contracts.APIResult]) string {
	if parseState.Running {
		return "Waiting for native result…"
	}
	if parseState.Cancelled {
		return "Wait cancelled. An open OS dialog may still need dismissal."
	}
	parseDetail := formatTesterResult(parseState.Value)
	if parseState.Error != nil {
		return strings.TrimSpace(parseDetail + " " + parseState.Error.Error())
	}
	if parseState.Ready {
		return parseDetail
	}
	return "Not run"
}

// getTesterTimeout allows deliberate dialog inspection without extending routine calls.
func getTesterTimeout(parseAction string) time.Duration {
	switch parseAction {
	case "open-file", "open-files", "open-directory", "save-path", "message-info", "message-question", "export-report":
		return 5 * time.Minute
	default:
		return desktop.RequestTimeout
	}
}

// callTesterNativeAction uses the portable desktop SDK for native APIs; picker actions remain APIService probes.
func callTesterNativeAction(parseContext context.Context, parseClient desktop.Client, parseAction string) (contracts.APIResult, error) {
	parseResult := contracts.APIResult{ID: parseAction, Outcome: "completed"}
	switch parseAction {
	case "clipboard-write":
		parseErr := parseClient.WriteClipboard(parseContext, "GoWebComponents API clipboard fixture")
		parseResult.Detail = "synthetic text written (replaces clipboard)"
		return parseResult, parseErr
	case "clipboard-read":
		parseValue, parseErr := parseClient.ReadClipboard(parseContext)
		if parseErr == nil {
			parseResult.Detail = fmt.Sprintf("matches=%t chars=%d", parseValue == "GoWebComponents API clipboard fixture", len([]rune(parseValue)))
		}
		return parseResult, parseErr
	case "message-info", "message-question":
		parseKind := "info"
		if parseAction == "message-question" {
			parseKind = "question"
		}
		parseReply, parseErr := parseClient.ShowMessage(parseContext, desktop.MessageRequest{Kind: parseKind, Title: "API Lab", Message: "Portable desktop SDK message test"})
		parseResult.Detail = parseReply.Button
		return parseResult, parseErr
	case "window-info", "window-resize", "window-maximize", "window-restore", "window-fullscreen", "window-unfullscreen":
		parseWindowAction := parseAction[len("window-"):]
		parseInfo, parseErr := parseClient.ControlWindow(parseContext, desktop.WindowRequest{Action: parseWindowAction, Width: 720, Height: 520})
		if parseErr == nil {
			parseResult.WindowID = parseInfo.ID
			parseResult.Detail = fmt.Sprintf("%s size=%dx%d", parseWindowAction, parseInfo.Width, parseInfo.Height)
		}
		return parseResult, parseErr
	case "screens":
		parseScreens, parseErr := parseClient.ListScreens(parseContext)
		if parseErr == nil {
			parseJSON, parseMarshalErr := json.Marshal(parseScreens)
			if parseMarshalErr != nil {
				return parseResult, parseMarshalErr
			}
			parseResult.Detail = string(parseJSON)
		}
		return parseResult, parseErr
	default:
		return desktop.Call[contracts.APIResult](parseContext, parseClient, "api.run", parseAction)
	}
}

// renderTesterCase owns one action and refreshes evidence within that task's lifetime.
func renderTesterCase(parseProps testerActionProps) ui.Node {
	parseTask := ui.UseTask(func(parseContext context.Context) (contracts.APIResult, error) {
		parseResult, parseErr := desktop.CallWithTimeout[contracts.APIResult](parseContext, parseProps.Client, getTesterTimeout(parseProps.Case.Action), "api.run", parseProps.Case.Action)
		if parseProps.Case.Action == "clipboard-write" || parseProps.Case.Action == "clipboard-read" || strings.HasPrefix(parseProps.Case.Action, "message-") || strings.HasPrefix(parseProps.Case.Action, "window-") || parseProps.Case.Action == "screens" {
			parseResult, parseErr = callTesterNativeAction(parseContext, parseProps.Client, parseProps.Case.Action)
		}
		if parseErr == nil {
			parseErr = parseProps.Refresh(parseContext)
		}
		return parseResult, parseErr
	})
	parseRun := ui.UseEvent(func() { parseTask.Start() })
	parseCancel := ui.UseEvent(func() { parseTask.Cancel() })
	parseState := parseTask.Get()
	var parseCancelNode ui.Node
	if parseState.Running {
		parseCancelNode = Button(ClassStr("tester-cancel"), OnClick(parseCancel), Text("Cancel wait"))
	}
	return Div(ClassStr("tester-case"),
		Button(ID(parseProps.Case.ID), OnClick(parseRun), Disabled(!parseProps.IsAvailable || parseState.Running), Aria("describedby", parseProps.Case.ID+"-result"), Text(parseProps.Case.Label)),
		parseCancelNode,
		P(ID(parseProps.Case.ID+"-result"), ClassStr("tester-case__hint"), Role("status"), Text(formatTesterTask(parseState))))
}

// renderTesterNav scrolls without changing the hash router's active route.
func renderTesterNav(parseProps testerNavProps) ui.Node {
	parseError := ui.UseState("")
	parseJump := ui.UseEvent(func() {
		parseDocument, parseErr := interop.GetDocument()
		if parseErr == nil {
			parseElement, isFound, parseFindErr := parseDocument.ElementByID(parseProps.Target)
			parseErr = parseFindErr
			if parseErr == nil && isFound {
				parseErr = parseElement.ScrollIntoView(interop.ScrollIntoViewOptions{Block: "start"})
			}
		}
		if parseErr != nil {
			parseError.Set(parseErr.Error())
		}
	})
	var parseErrorNode ui.Node
	if parseError.Get() != "" {
		parseErrorNode = P(Role("status"), Text(parseError.Get()))
	}
	return Div(Button(OnClick(parseJump), Text(parseProps.Label)), parseErrorNode)
}

// pollTesterReport suppresses results/errors returned after the route lifetime ends.
func pollTesterReport(parseContext context.Context, parseRead func(context.Context) (contracts.APIReport, error), parsePublish func(contracts.APIReport), parseFail func(error)) {
	defer interop.RecoverContainedPanic("API report polling")
	parseTimer := time.NewTicker(time.Second)
	defer parseTimer.Stop()
	for {
		if parseContext.Err() != nil {
			return
		}
		parseReport, parseErr := parseRead(parseContext)
		if parseContext.Err() != nil {
			return
		}
		if parseErr != nil {
			parseFail(parseErr)
		} else {
			parsePublish(parseReport)
		}
		select {
		case <-parseContext.Done():
			return
		case <-parseTimer.C:
		}
	}
}

// renderTester renders explicit operations and separately recorded human verdicts.
func renderTester() ui.Node {
	parseClient, parseConnectErr := desktop.Connect()
	parseReport := ui.UseState(contracts.APIReport{})
	parseInitialStatus := "Session polling active; no visual tests are auto-passed."
	if parseConnectErr != nil {
		parseInitialStatus = "Native report unavailable: " + parseConnectErr.Error()
	}
	parseStatus := ui.UseState(parseInitialStatus)
	parseNote := ui.UseState("")
	parseSelected := ui.UseState("open-file")
	parseObservation := ui.UseRef(testerObservation{})
	parseRead := func(parseContext context.Context) (contracts.APIReport, error) {
		return desktop.Call[contracts.APIReport](parseContext, parseClient, "api.report")
	}
	parseRefreshReport := func(parseContext context.Context) error {
		parseNext, parseErr := parseRead(parseContext)
		if parseContext.Err() != nil {
			return parseContext.Err()
		}
		if parseErr != nil {
			return fmt.Errorf("report refresh: %w", parseErr)
		}
		parseReport.Set(parseNext)
		return nil
	}
	parseRefreshTask := ui.UseTask(func(parseContext context.Context) (contracts.APIResult, error) {
		if parseErr := parseRefreshReport(parseContext); parseErr != nil {
			return contracts.APIResult{}, parseErr
		}
		return contracts.APIResult{Outcome: "completed", Detail: "Report refreshed"}, nil
	})
	parseExportTask := ui.UseTask(func(parseContext context.Context) (contracts.APIResult, error) {
		parseResult, parseErr := desktop.CallWithTimeout[contracts.APIResult](parseContext, parseClient, getTesterTimeout("export-report"), "api.export")
		if parseErr == nil {
			parseErr = parseRefreshReport(parseContext)
		}
		return parseResult, parseErr
	})
	parseObserveTask := ui.UseTask(func(parseContext context.Context) (contracts.APIResult, error) {
		parseRequest := parseObservation.Get()
		parseResult, parseErr := desktop.Call[contracts.APIResult](parseContext, parseClient, "api.observe", parseRequest.ID, parseRequest.Outcome, parseRequest.Detail)
		if parseErr == nil {
			parseErr = parseRefreshReport(parseContext)
		}
		return parseResult, parseErr
	})
	ui.UseEffect(func() func() {
		parseContext, parseCancel := context.WithCancel(context.Background())
		if parseConnectErr == nil {
			go pollTesterReport(parseContext, parseRead, parseReport.Set, func(parseErr error) { parseStatus.Set(parseErr.Error()) })
		}
		return parseCancel
	}, true)
	parseRefresh := ui.UseEvent(func() { parseRefreshTask.Start() })
	parseExport := ui.UseEvent(func() { parseExportTask.Start() })
	parseNoteInput := ui.UseEvent(func(parseEvent ui.InputEvent) { parseNote.Set(parseEvent.GetValue()) })
	parseCaseInput := ui.UseEvent(func(parseEvent ui.InputEvent) { parseSelected.Set(parseEvent.GetValue()) })
	parseRecord := func(parseOutcome string) {
		parseObservation.Set(testerObservation{ID: parseSelected.Get(), Outcome: parseOutcome, Detail: parseNote.Get()})
		parseObserveTask.Start()
	}
	parsePass := ui.UseEvent(func() { parseRecord("observed-pass") })
	parseFail := ui.UseEvent(func() { parseRecord("observed-fail") })
	parseNotTested := ui.UseEvent(func() { parseRecord("not-tested") })
	parseGroups := []any{}
	parseNavigation := []any{Aria("label", "Test sections")}
	for _, parseGroup := range testerCases {
		parseButtons := []any{ClassStr("tester-grid")}
		for _, parseCase := range parseGroup.Items {
			parseButtons = append(parseButtons, ui.CreateElement(renderTesterCase, testerActionProps{Client: parseClient, Case: parseCase, IsAvailable: parseConnectErr == nil, Refresh: parseRefreshReport}))
		}
		parseGroups = append(parseGroups, Section(ID(parseGroup.ID), ClassStr("tester-panel"), H2(Text(parseGroup.Title)), Div(parseButtons...)))
		parseNavigation = append(parseNavigation, ui.CreateElement(renderTesterNav, testerNavProps{Target: parseGroup.ID, Label: parseGroup.Title}))
	}
	parseNavigation = append(parseNavigation, ui.CreateElement(renderTesterNav, testerNavProps{Target: "tester-evidence", Label: "05 / Evidence log"}))
	parseOptions := []any{ID("api-case"), OnChange(parseCaseInput), Value(parseSelected.Get())}
	for _, parseCase := range getTesterManualCases() {
		parseOptions = append(parseOptions, Option(Value(parseCase.Action), Text(parseCase.Label)))
	}
	parseJSON, parseJSONErr := json.MarshalIndent(parseReport.Get(), "", "  ")
	if parseJSONErr != nil {
		parseJSON = []byte(parseJSONErr.Error())
	}
	parseConnection := "Native bridge connected"
	if parseConnectErr != nil {
		parseConnection = "Unavailable: " + parseConnectErr.Error()
	}
	parseIsObserving := parseObserveTask.Get().Running || parseConnectErr != nil
	parseContextLabel := "RIGHT-CLICK TEST SURFACE"
	parseContextHelp := "Select · checkbox · radio · submenu · disabled action · Escape"
	if parseConnectErr != nil || !parseClient.Supports(desktop.NativeMenus) {
		parseContextLabel = "RIGHT-CLICK TEST SURFACE (native menu unavailable)"
		parseContextHelp = "Native menus are disabled by the host feature policy."
	}
	parseMain := []any{ClassStr("tester-main"),
		Div(ID("api-context-target"), ClassStr("tester-context-target"), TabIndex(0), Text(parseContextLabel), P(Text(parseContextHelp))),
		Div(ClassStr("tester-editor"), Label(For("api-editable"), Text("Keyboard / IME / native edit menu")), Input(ID("api-editable"), Placeholder("Type Unicode text, select, copy and paste…"))),
		P(ClassStr("tester-warning"), Text("Clipboard warning: Write fixture replaces your current clipboard. Read is explicit; use the synthetic fixture, not private data. Selected paths and manual notes may appear in the shared session report.")),
	}
	parseMain = append(parseMain, parseGroups...)
	parseMain = append(parseMain, Section(ID("tester-evidence"), ClassStr("tester-report"), H2(Text("05 / Evidence log")),
		P(ID("api-status"), Role("status"), Text(parseStatus.Get())),
		Div(ClassStr("tester-observation"), Label(For("api-case"), Text("Observed case")), Select(parseOptions...), Label(For("api-note"), Text("Human observation (up to 2,048 bytes)")), Input(ID("api-note"), OnInput(parseNoteInput), Value(parseNote.Get()), Placeholder("What did you actually see?"), Attr("maxlength", 1024))),
		Div(ClassStr("tester-controls"), Button(ID("api-pass"), OnClick(parsePass), Disabled(parseIsObserving), Text("Observed pass")), Button(ID("api-fail"), OnClick(parseFail), Disabled(parseIsObserving), Text("Observed fail")), Button(ID("api-not-tested"), OnClick(parseNotTested), Disabled(parseIsObserving), Text("Not tested"))),
		P(ID("api-observe-result"), Role("status"), Text(formatTesterTask(parseObserveTask.Get()))),
		Div(ClassStr("tester-controls"), Button(ID("api-refresh"), OnClick(parseRefresh), Disabled(parseConnectErr != nil || parseRefreshTask.Get().Running), Text("Refresh report")), Button(ID("api-export"), OnClick(parseExport), Disabled(parseConnectErr != nil || parseExportTask.Get().Running), Text("Export report…"))),
		P(ID("api-refresh-result"), Role("status"), Text(formatTesterTask(parseRefreshTask.Get()))), P(ID("api-export-result"), Role("status"), Text(formatTesterTask(parseExportTask.Get()))),
		Pre(ID("api-report-log"), TabIndex(0), Aria("label", "Session report JSON"), Text(string(parseJSON)))))
	return Div(ClassStr("tester-shell"),
		Header(ClassStr("tester-topbar"), Div(Span(ClassStr("tester-kicker"), Text("GWC / WAILS · WINDOWS")), H1(Text("Native API laboratory")), P(Text("Native outcomes are not visual verdicts."))), Div(ClassStr("tester-links"), A(Href("#/counter"), Text("Counter")), A(Href("#/tester"), Text("Tester")))),
		Div(ClassStr("tester-layout"), Aside(ClassStr("tester-sidebar"), H2(Text("Runbook")), P(Text(parseConnection)), Nav(parseNavigation...), P(Text("Use Tests / Ctrl+Shift+K and F8. Escape leaves fullscreen. Use the native Edit menu in the text field. Record a verdict only after observing the result."))), Main(parseMain...)),
		Footer(ClassStr("tester-footer"), Text("Manual Windows verification · local session only · export is explicit · cancellation cannot force an OS picker closed")))
}
