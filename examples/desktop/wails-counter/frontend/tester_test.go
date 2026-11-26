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
	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/monstercameron/GoWebComponents/v8/events"
	"github.com/monstercameron/GoWebComponents/v8/ui"
)

type testerFakeTransport struct {
	methods  []string
	starts   []string
	args     []json.RawMessage
	pollData json.RawMessage
}

// Capabilities advertises the test transport's configured methods and portable features.
func (f *testerFakeTransport) Capabilities() (desktop.Capabilities, error) {
	return desktop.Capabilities{Protocol: desktop.ProtocolVersion, Methods: f.methods}, nil
}

// Start records one dispatched request.
func (f *testerFakeTransport) Start(method string, args json.RawMessage) (string, error) {
	f.starts = append(f.starts, method)
	f.args = append(f.args, append(json.RawMessage(nil), args...))
	return "request-1", nil
}

// Poll completes tester requests with configured or default valid JSON data.
func (f *testerFakeTransport) Poll(string) (desktop.Reply, error) {
	data := f.pollData
	if len(data) == 0 {
		data = json.RawMessage(`{"version":1,"data":[]}`)
	}
	return desktop.Reply{Done: true, Data: data}, nil
}

// Cancel completes transport cleanup.
func (f *testerFakeTransport) Cancel(string) error { return nil }

// Listen is unused by tester action tests.
func (f *testerFakeTransport) Listen(string) (string, error) { return "", nil }

// Next is unused by tester action tests.
func (f *testerFakeTransport) Next(string) (desktop.Reply, error) {
	return desktop.Reply{}, nil
}

// Unlisten is unused by tester action tests.
func (f *testerFakeTransport) Unlisten(string) error { return nil }

// TestTesterInteractiveTimeouts limits longer waits to explicitly interactive cases.
func TestTesterInteractiveTimeouts(t *testing.T) {
	interactive := map[string]bool{
		"open-file": true, "open-files": true, "open-directory": true, "open-directories": true, "save-path": true,
		"message-info": true, "message-warning": true, "message-error": true, "message-question": true, "message-question-custom": true,
		"export-report": true, "window-print": true,
	}
	for _, c := range getTesterManualCases() {
		expected := 30 * time.Second
		if interactive[c.Action] {
			expected = 5 * time.Minute
		}
		if actual := getTesterTimeout(c.Action); actual != expected {
			t.Fatalf("%s timeout=%v want=%v", c.Action, actual, expected)
		}
	}
}

// TestTesterPortableActionDispatchesOnce prevents duplicate legacy and portable calls and publishes activity telemetry.
func TestTesterPortableActionDispatchesOnce(t *testing.T) {
	var seen []labActivityEvent
	stop := events.Subscribe(labActivityTopic, func(ev labActivityEvent) {
		seen = append(seen, ev)
	})
	defer stop()

	transport := &testerFakeTransport{methods: []string{desktop.ScreensMethod, "api.run"}}
	_, err := callTesterAction(context.Background(), desktop.NewClient(transport), "screens")
	if err != nil {
		t.Fatal(err)
	}
	if len(transport.starts) != 1 || transport.starts[0] != desktop.ScreensMethod {
		t.Fatalf("dispatches=%v want one %s call", transport.starts, desktop.ScreensMethod)
	}
	if len(seen) != 1 || seen[0].Action != "screens" || seen[0].Outcome != "completed" {
		t.Fatalf("activity events=%+v", seen)
	}
}

// TestTesterPortableControlsRequireTheirFeature prevents unsupported controls from appearing actionable.
func TestTesterPortableControlsRequireTheirFeature(t *testing.T) {
	client := desktop.NewClient(&testerFakeTransport{methods: []string{"api.run"}})
	for _, action := range []string{"open-directories", "window-info", "window-set-zoom-low", "menu-replace", "autostart-enable", "tray-position-window", "child-create-mica"} {
		if isTesterActionAvailable(client, action) {
			t.Fatalf("portable control %s enabled without its advertised feature", action)
		}
	}
	if !isTesterActionAvailable(client, "open-file") {
		t.Fatal("legacy APIService picker unexpectedly disabled")
	}
}

// TestTesterTrayUpsertIncludesPNGIconsAndLabel verifies Wails v3.1.0 tray icon, dark-mode icon, and label fallback.
func TestTesterTrayUpsertIncludesPNGIconsAndLabel(t *testing.T) {
	transport := &testerFakeTransport{methods: []string{desktop.TrayConfigureMethod}}
	req := buildTesterTrayUpsertRequest()
	if req.Label == "" || len(req.Icon) < 8 || len(req.DarkModeIcon) < 8 {
		t.Fatalf("tray upsert missing v3.1.0 fields: label=%q iconLen=%d darkLen=%d", req.Label, len(req.Icon), len(req.DarkModeIcon))
	}
	if err := desktop.NewClient(transport).ConfigureTray(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if len(transport.starts) != 1 || transport.starts[0] != desktop.TrayConfigureMethod {
		t.Fatalf("dispatches=%v", transport.starts)
	}
}

// TestTesterV310WindowsActionsDispatchThroughSDK verifies new v3.1.0 dialogs, zoom, tray position, and child controls.
func TestTesterV310WindowsActionsDispatchThroughSDK(t *testing.T) {
	t.Run("custom-taskdialog", func(t *testing.T) {
		transport := &testerFakeTransport{
			methods:  []string{desktop.MessageMethod},
			pollData: json.RawMessage(`{"version":1,"data":{"button":"Save"}}`),
		}
		res, err := callTesterAction(context.Background(), desktop.NewClient(transport), "message-question-custom")
		if err != nil || res.Detail != "Save" {
			t.Fatalf("custom taskdialog res=%+v err=%v", res, err)
		}
	})
	t.Run("open-directories", func(t *testing.T) {
		transport := &testerFakeTransport{
			methods:  []string{desktop.FileDialogMethod},
			pollData: json.RawMessage(`{"selection":{"paths":["C:\\alpha","C:\\beta"]}}`),
		}
		res, err := callTesterAction(context.Background(), desktop.NewClient(transport), "open-directories")
		if err != nil || len(res.Paths) != 2 {
			t.Fatalf("open-directories res=%+v err=%v", res, err)
		}
	})
	t.Run("tray-position-window", func(t *testing.T) {
		transport := &testerFakeTransport{
			methods:  []string{desktop.TrayConfigureMethod},
			pollData: json.RawMessage(`{"version":1,"data":{}}`),
		}
		if _, err := callTesterAction(context.Background(), desktop.NewClient(transport), "tray-position-window"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("child-controls", func(t *testing.T) {
		transport := &testerFakeTransport{
			methods:  []string{desktop.ChildWindowCreateMethod, desktop.ChildWindowListMethod, desktop.ChildWindowInspectMethod, desktop.ChildWindowControlMethod},
			pollData: json.RawMessage(`{"version":1,"data":{"id":"tester-observer","templateId":"counter","width":800,"height":600}}`),
		}
		client := desktop.NewClient(transport)
		for _, action := range []string{"child-create-mica", "child-focus", "child-center", "child-resize", "child-set-title", "child-minimize", "child-maximize", "child-restore"} {
			if _, err := callTesterAction(context.Background(), client, action); err != nil {
				t.Fatalf("%s failed: %v", action, err)
			}
		}
	})
}

// TestTesterChildWindowTemplatesAlignWithHost keeps fixtures aligned with the example host configuration.
func TestTesterChildWindowTemplatesAlignWithHost(t *testing.T) {
	counterReq := getTesterChildWindowRequest()
	if counterReq.TemplateID != "counter" || counterReq.ID != "tester-observer" {
		t.Fatalf("child request=%+v", counterReq)
	}
	micaReq := getTesterMicaChildWindowRequest()
	if micaReq.TemplateID != "mica" || micaReq.ID != "tester-mica" {
		t.Fatalf("mica child request=%+v", micaReq)
	}
}

// TestTesterPortableErrorsAreFailed prevents errors from being presented as completed actions.
func TestTesterPortableErrorsAreFailed(t *testing.T) {
	result, err := callTesterAction(context.Background(), desktop.Client{}, "child-create")
	if err == nil || result.Outcome != "failed" || result.Detail != "" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

// TestTesterWindowRequestsOnlyResizeWithDimensions keeps payloads valid for the SDK validator.
func TestTesterWindowRequestsOnlyResizeWithDimensions(t *testing.T) {
	for _, action := range []string{"info", "show", "snap-assist", "reload", "center", "enable-size-constraints", "disable-size-constraints", "minimize", "maximize", "restore", "fullscreen", "unfullscreen", "focus", "zoom-in", "zoom-out", "zoom-reset"} {
		req := getTesterWindowRequest(action)
		if req.Action != action || req.Width != 0 || req.Height != 0 {
			t.Fatalf("%s request=%+v", action, req)
		}
	}
	if lowZoom := getTesterWindowRequest("set-zoom-low"); lowZoom.Action != "set-zoom" || lowZoom.Zoom != 0.75 {
		t.Fatalf("low zoom request=%+v", lowZoom)
	}
	if mouseOff := getTesterWindowRequest("ignore-mouse-off"); mouseOff.Action != "set-ignore-mouse-events" || mouseOff.Enabled {
		t.Fatalf("ignore mouse off request=%+v", mouseOff)
	}
	resize := getTesterWindowRequest("resize")
	if resize.Width != 720 || resize.Height != 520 {
		t.Fatalf("resize request=%+v", resize)
	}
	pos := getTesterWindowRequest("set-position")
	if pos.X != 120 || pos.Y != 120 {
		t.Fatalf("position request=%+v", pos)
	}
	for _, action := range []string{"set-always-on-top", "set-resizable", "flash", "set-content-protection", "set-enabled"} {
		if req := getTesterWindowRequest(action); !req.Enabled {
			t.Fatalf("%s request=%+v", action, req)
		}
	}
}

// TestTesterFilterGroupsMatchesActionAndTitle verifies deferred search filtering across groups.
func TestTesterFilterGroupsMatchesActionAndTitle(t *testing.T) {
	if got := filterTesterGroups(testerCases, ""); len(got) != len(testerCases) {
		t.Fatalf("empty filter len=%d want=%d", len(got), len(testerCases))
	}
	mica := filterTesterGroups(testerCases, "mica")
	if len(mica) != 1 || len(mica[0].Items) != 1 || mica[0].Items[0].Action != "child-create-mica" {
		t.Fatalf("mica filter=%+v", mica)
	}
	if none := filterTesterGroups(testerCases, "no-such-api-xyz"); len(none) != 0 {
		t.Fatalf("unmatched filter=%+v", none)
	}
}

// TestTesterFormatWindowEventIncludesZoom verifies Wails v3.1.0 zoom event formatting.
func TestTesterFormatWindowEventIncludesZoom(t *testing.T) {
	text := formatWindowEventStatus(desktop.WindowEvent{Kind: desktop.WindowZoomChanged, WindowID: "1", Width: 800, Height: 600, Zoom: 0.75})
	if !strings.Contains(text, "zoom-changed") || !strings.Contains(text, "zoom=0.75") {
		t.Fatalf("formatted=%q", text)
	}
}

// TestTesterManualCasesCoverNativeActions verifies every button can receive a human verdict.
func TestTesterManualCasesCoverNativeActions(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range getTesterManualCases() {
		if c.Action == "" || c.Label == "" || seen[c.Action] {
			t.Fatalf("invalid/duplicate case: %+v", c)
		}
		seen[c.Action] = true
	}
	for _, group := range testerCases {
		for _, c := range group.Items {
			if !seen[c.Action] {
				t.Fatal(c.Action)
			}
		}
	}
	for _, action := range []string{"context-check", "context-radio", "context-disabled", "context-dismiss", "menu-action", "shortcut", "keyboard-focus", "keyboard-ime", "edit-menu", "second-window", "export-report"} {
		if !seen[action] {
			t.Fatalf("missing manual case %s", action)
		}
	}
}

// TestTesterTaskShowsActualOutcome keeps operation results, paths and errors visible.
func TestTesterTaskShowsActualOutcome(t *testing.T) {
	result := contracts.APIResult{Outcome: "cancelled", Detail: "picker dismissed", WindowID: "7", Paths: []string{"report.json"}}
	text := formatTesterTask(ui.TaskState[contracts.APIResult]{Value: result, Ready: true})
	for _, part := range []string{"cancelled", "picker dismissed", "window 7", "report.json"} {
		if !strings.Contains(text, part) {
			t.Fatal(text)
		}
	}
	if text = formatTesterTask(ui.TaskState[contracts.APIResult]{Error: errors.New("export refused")}); !strings.Contains(text, "export refused") {
		t.Fatal(text)
	}
	if text = formatTesterTask(ui.TaskState[contracts.APIResult]{Cancelled: true}); !strings.Contains(text, "OS dialog may still need dismissal") {
		t.Fatal(text)
	}
}

// TestTesterPollSuppressesLateState covers both successful and failed reads after unmount.
func TestTesterPollSuppressesLateState(t *testing.T) {
	for _, readErr := range []error{nil, errors.New("cancelled transport")} {
		ctx, cancel := context.WithCancel(context.Background())
		reads, writes := 0, 0
		pollTesterReport(ctx, func(context.Context) (contracts.APIReport, error) {
			reads++
			cancel()
			return contracts.APIReport{}, readErr
		}, func(contracts.APIReport) { writes++ }, func(error) { writes++ })
		if reads != 1 || writes != 0 {
			t.Fatalf("reads=%d late writes=%d", reads, writes)
		}
	}
}
