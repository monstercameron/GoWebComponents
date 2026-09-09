package desktop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// TestWindowRequestAllowlist covers every typed caller-owned runtime operation.
func TestWindowRequestAllowlist(parseTest *testing.T) {
	parseValid := []WindowRequest{
		{Action: "info"}, {Action: "set-title", Title: "Workspace"},
		{Action: "set-screen", ScreenID: "display-1"},
		{Action: "set-position", X: -10, Y: 20}, {Action: "center"},
		{Action: "set-relative-position", X: 10, Y: 20}, {Action: "set-bounds", X: 10, Y: 20, Width: 800, Height: 600},
		{Action: "enable-size-constraints"}, {Action: "disable-size-constraints"},
		{Action: "resize", Width: 800, Height: 600}, {Action: "set-min-size", Width: 320, Height: 240},
		{Action: "set-max-size", Width: 1920, Height: 1080}, {Action: "minimize"},
		{Action: "unminimize"}, {Action: "maximize"}, {Action: "unmaximize"}, {Action: "toggle-maximize"},
		{Action: "restore"}, {Action: "fullscreen"}, {Action: "unfullscreen"}, {Action: "toggle-fullscreen"},
		{Action: "focus"}, {Action: "set-always-on-top", Enabled: true},
		{Action: "set-resizable", Enabled: true}, {Action: "flash", Enabled: true},
		{Action: "set-frameless", Enabled: true}, {Action: "toggle-frameless"},
		{Action: "show-menu-bar"}, {Action: "hide-menu-bar"}, {Action: "toggle-menu-bar"},
		{Action: "set-background-color", Red: 1, Green: 2, Blue: 3, Alpha: 255},
		{Action: "set-minimize-button-state", State: "disabled"}, {Action: "set-maximize-button-state", State: "hidden"},
		{Action: "set-close-button-state", State: "enabled"}, {Action: "set-fullscreen-button-state", State: "disabled"},
		{Action: "set-content-protection", Enabled: true}, {Action: "zoom-in"},
		{Action: "zoom-out"}, {Action: "set-zoom", Zoom: 1.25}, {Action: "zoom-reset"},
	}
	for _, parseRequest := range parseValid {
		if parseErr := validateWindowRequest(parseRequest); parseErr != nil {
			parseTest.Errorf("%s rejected: %v", parseRequest.Action, parseErr)
		}
	}
}

// TestWindowRequestRejectsCrossActionArguments prevents ambiguous native commands.
func TestWindowRequestRejectsCrossActionArguments(parseTest *testing.T) {
	parseInvalid := []WindowRequest{
		{Action: "info", Enabled: true},
		{Action: "set-title", Title: "bad\x00title"},
		{Action: "set-screen"},
		{Action: "set-position", X: 100001},
		{Action: "resize", Width: 800, Height: 600, X: 1},
		{Action: "set-min-size", Width: 0, Height: 240},
		{Action: "set-always-on-top", Width: 1},
		{Action: "set-background-color", Red: 256},
		{Action: "set-background-color", Red: 1, Zoom: 1},
		{Action: "set-close-button-state", State: "broken"},
		{Action: "set-close-button-state", State: "enabled", Blue: 1},
		{Action: "set-zoom", Zoom: 10},
		{Action: "set-zoom", Zoom: 1, Alpha: 255},
		{Action: "set-zoom", Zoom: 1, State: "enabled"},
		{Action: "close"}, {Action: "hide"}, {Action: "reload"}, {Action: "exec-js"},
	}
	for _, parseRequest := range parseInvalid {
		if parseErr := validateWindowRequest(parseRequest); parseErr == nil {
			parseTest.Errorf("%+v accepted", parseRequest)
		}
	}
}

// TestControlWindowValidatesBeforeTransport keeps malformed operations out of the host.
func TestControlWindowValidatesBeforeTransport(parseTest *testing.T) {
	parseTransport := &parseRejectWindowTransport{}
	parseClient := Client{parseTransport: parseTransport}
	_, parseErr := parseClient.ControlWindow(context.Background(), WindowRequest{Action: "hide"})
	if !interop.IsCode(parseErr, interop.CodeInvalid) || parseTransport.isCalled {
		parseTest.Fatalf("invalid request reached transport: called=%v err=%v", parseTransport.isCalled, parseErr)
	}
}

// TestWindowPrintingRequiresItsSeparateFeature verifies printing is not implied by window controls.
func TestWindowPrintingRequiresItsSeparateFeature(parseTest *testing.T) {
	parseBackend := &parsePrintingBackend{parseNativeFake: &parseNativeFake{}}
	parsePolicy, parseErr := ParseFeaturePolicy(string(WindowControls))
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseErr = NewNativeHost(parseBackend, parsePolicy).PrintWindow(context.Background()); !interop.IsCode(parseErr, interop.CodeUnavailable) || parseBackend.isPrinted {
		parseTest.Fatalf("printing bypassed separate gate: printed=%v err=%v", parseBackend.isPrinted, parseErr)
	}
	parsePolicy, parseErr = ParseFeaturePolicy(string(WindowPrinting))
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseErr = NewNativeHost(parseBackend, parsePolicy).PrintWindow(context.Background()); parseErr != nil || !parseBackend.isPrinted {
		parseTest.Fatalf("enabled printing failed: printed=%v err=%v", parseBackend.isPrinted, parseErr)
	}
}

type parsePrintingBackend struct {
	*parseNativeFake
	isPrinted bool
}

// Features advertises both controls to prove their policies remain independent.
func (*parsePrintingBackend) Features() []Feature { return []Feature{WindowControls, WindowPrinting} }

// PrintWindow records an explicit gated printing request.
func (parseBackend *parsePrintingBackend) PrintWindow(context.Context) error {
	parseBackend.isPrinted = true
	return nil
}

type parseRejectWindowTransport struct{ isCalled bool }

// Capabilities advertises the window method for validation-order testing.
func (parseTransport *parseRejectWindowTransport) Capabilities() (Capabilities, error) {
	return Capabilities{Methods: []string{WindowMethod}}, nil
}

// Start records an unexpected transport invocation.
func (parseTransport *parseRejectWindowTransport) Start(string, json.RawMessage) (string, error) {
	parseTransport.isCalled = true
	return "", nil
}

// Poll is unused by validation-order testing.
func (*parseRejectWindowTransport) Poll(string) (Reply, error) { return Reply{}, nil }

// Cancel is unused by validation-order testing.
func (*parseRejectWindowTransport) Cancel(string) error { return nil }

// Listen is unused by validation-order testing.
func (*parseRejectWindowTransport) Listen(string) (string, error) { return "", nil }

// Next is unused by validation-order testing.
func (*parseRejectWindowTransport) Next(string) (Reply, error) { return Reply{}, nil }

// Unlisten is unused by validation-order testing.
func (*parseRejectWindowTransport) Unlisten(string) error { return nil }
