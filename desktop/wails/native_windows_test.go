//go:build windows

package wails

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// TestNativeBackendMissingCallerNeverUsesFocusedWindow verifies every adapter operation requires its caller.
func TestNativeBackendMissingCallerNeverUsesFocusedWindow(parseTest *testing.T) {
	parseBackend := NewNativeBackend()
	if _, parseErr := parseBackend.ClipboardRead(context.Background()); parseErr == nil {
		parseTest.Fatal("clipboard read accepted missing caller")
	}
	if parseErr := parseBackend.ClipboardWrite(context.Background(), desktop.ClipboardWriteRequest{Text: "x"}); parseErr == nil {
		parseTest.Fatal("clipboard write accepted missing caller")
	}
	if _, parseErr := parseBackend.ShowMessage(context.Background(), desktop.MessageRequest{Kind: "info"}); parseErr == nil {
		parseTest.Fatal("message accepted missing caller")
	}
	if _, parseErr := parseBackend.Window(context.Background(), desktop.WindowRequest{Action: "info"}); parseErr == nil {
		parseTest.Fatal("window accepted missing caller")
	}
	if _, parseErr := parseBackend.Screens(context.Background()); parseErr == nil {
		parseTest.Fatal("screens accepted missing caller")
	}
	parseSystem := parseBackend.(desktop.SystemEnvironmentBackend)
	if _, parseErr := parseSystem.SystemEnvironment(context.Background()); parseErr == nil {
		parseTest.Fatal("system environment accepted missing caller")
	}
	parseURLs := parseBackend.(desktop.ExternalURLBackend)
	if parseErr := parseURLs.OpenExternalURL(context.Background(), desktop.ExternalURLOpenRequest{URL: "https://example.com"}); parseErr == nil {
		parseTest.Fatal("external URL accepted missing caller")
	}
	parseAutostart := parseBackend.(desktop.AutostartBackend)
	if _, parseErr := parseAutostart.AutostartStatus(context.Background()); parseErr == nil {
		parseTest.Fatal("autostart status accepted missing caller")
	}
	if parseErr := parseAutostart.AutostartEnable(context.Background()); parseErr == nil {
		parseTest.Fatal("autostart enable accepted missing caller")
	}
	if parseErr := parseAutostart.AutostartDisable(context.Background()); parseErr == nil {
		parseTest.Fatal("autostart disable accepted missing caller")
	}
	parseFileManager := parseBackend.(desktop.FileManagerBackend)
	if parseErr := parseFileManager.RevealPath(context.Background(), desktop.FileManagerRevealRequest{Path: `C:\fixture`}); parseErr == nil {
		parseTest.Fatal("file manager accepted missing caller")
	}
}

// TestWindowsMessageRequestValidationMatchesMessageBox verifies only native button families are accepted.
func TestWindowsMessageRequestValidationMatchesMessageBox(parseTest *testing.T) {
	for _, parseRequest := range []desktop.MessageRequest{
		{Kind: "info", Buttons: []string{"Ok"}, DefaultButton: "Ok", CancelButton: "Ok"},
		{Kind: "warning"},
		{Kind: "error"},
		{Kind: "question", DefaultButton: "No"},
	} {
		if parseErr := validateWindowsMessageRequest(parseRequest); parseErr != nil {
			parseTest.Errorf("accepted native request rejected: %+v: %v", parseRequest, parseErr)
		}
	}
	for _, parseRequest := range []desktop.MessageRequest{
		{Kind: "info", Buttons: []string{"Continue"}},
		{Kind: "question", Buttons: []string{"Yes", "Maybe"}},
		{Kind: "question", DefaultButton: "Ok"},
		{Kind: "question", CancelButton: "No"},
	} {
		if parseErr := validateWindowsMessageRequest(parseRequest); parseErr == nil {
			parseTest.Errorf("unsupported native request accepted: %+v", parseRequest)
		}
	}
}

// TestWindowUsesCallerAndReportsRuntimeState verifies typed operations target only the context window.
func TestWindowUsesCallerAndReportsRuntimeState(parseTest *testing.T) {
	parseWindow := &parseWindowFake{parseID: 7, parseName: "caller", parseX: -20, parseY: 30, parseWidth: 640, parseHeight: 480, parseZoom: 1.25, isFocused: true, isVisible: true, isResizable: true}
	parseContext := context.WithValue(context.Background(), application.WindowKey, application.Window(parseWindow))
	parseBackend := nativeBackend{}
	parseInfo, parseErr := parseBackend.Window(parseContext, desktop.WindowRequest{Action: "set-position", X: 12, Y: 34})
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseWindow.parseAction != "set-position" || parseWindow.parseX != 12 || parseWindow.parseY != 34 {
		parseTest.Fatalf("operation not applied to caller: %+v", parseWindow)
	}
	if parseInfo.ID != "7" || parseInfo.Name != "caller" || parseInfo.X != 12 || parseInfo.Y != 34 || parseInfo.Width != 640 || parseInfo.Height != 480 || !parseInfo.Focused || !parseInfo.Visible || !parseInfo.Resizable || parseInfo.Zoom != 1.25 {
		parseTest.Fatalf("unexpected window info: %+v", parseInfo)
	}
	parseInfo, parseErr = parseBackend.Window(parseContext, desktop.WindowRequest{Action: "set-zoom", Zoom: 1.5})
	if parseErr != nil || parseWindow.parseAction != "set-zoom" || parseInfo.Zoom != 1.5 {
		parseTest.Fatalf("bounded zoom not applied to caller: info=%+v window=%+v err=%v", parseInfo, parseWindow, parseErr)
	}
}

// TestWindowRejectsNoOpFullscreenCaptionButton keeps Windows capability claims honest.
func TestWindowRejectsNoOpFullscreenCaptionButton(parseTest *testing.T) {
	parseWindow := &parseWindowFake{parseID: 7, parseName: "caller", parseWidth: 640, parseHeight: 480, parseZoom: 1}
	parseContext := context.WithValue(context.Background(), application.WindowKey, application.Window(parseWindow))
	_, parseErr := (nativeBackend{}).Window(parseContext, desktop.WindowRequest{Action: "set-fullscreen-button-state", State: "disabled"})
	if !interop.IsCode(parseErr, interop.CodeUnavailable) {
		parseTest.Fatalf("Windows no-op reported success: %v", parseErr)
	}
}

// TestWindowRejectsZoomBelowWindowsFloor prevents Wails from silently clamping a request.
func TestWindowRejectsZoomBelowWindowsFloor(parseTest *testing.T) {
	parseWindow := &parseWindowFake{parseID: 7, parseName: "caller", parseWidth: 640, parseHeight: 480, parseZoom: 1}
	parseContext := context.WithValue(context.Background(), application.WindowKey, application.Window(parseWindow))
	_, parseErr := (nativeBackend{}).Window(parseContext, desktop.WindowRequest{Action: "set-zoom", Zoom: 0.5})
	if !interop.IsCode(parseErr, interop.CodeInvalid) || parseWindow.parseAction != "" || parseWindow.parseZoom != 1 {
		parseTest.Fatalf("sub-floor zoom reached Wails: action=%q zoom=%v err=%v", parseWindow.parseAction, parseWindow.parseZoom, parseErr)
	}
}

type parseWindowFake struct {
	application.Window
	parseAction                          string
	parseName                            string
	parseID, parseX, parseY              int
	parseWidth, parseHeight              int
	parseZoom                            float64
	isFocused, isMinimised, isMaximised  bool
	isFullscreen, isVisible, isResizable bool
}

// ID returns deterministic fake identity.
func (parseWindow *parseWindowFake) ID() uint { return uint(parseWindow.parseID) }

// Name returns deterministic fake name.
func (parseWindow *parseWindowFake) Name() string { return parseWindow.parseName }

// Position returns deterministic fake position.
func (parseWindow *parseWindowFake) Position() (int, int) {
	return parseWindow.parseX, parseWindow.parseY
}

// RelativePosition returns deterministic fake client-relative position.
func (parseWindow *parseWindowFake) RelativePosition() (int, int) {
	return parseWindow.parseX, parseWindow.parseY
}

// SetPosition records a caller-owned move.
func (parseWindow *parseWindowFake) SetPosition(parseX, parseY int) {
	parseWindow.parseAction, parseWindow.parseX, parseWindow.parseY = "set-position", parseX, parseY
}

// Width returns deterministic fake width.
func (parseWindow *parseWindowFake) Width() int { return parseWindow.parseWidth }

// Height returns deterministic fake height.
func (parseWindow *parseWindowFake) Height() int { return parseWindow.parseHeight }

// GetZoom returns deterministic fake zoom.
func (parseWindow *parseWindowFake) GetZoom() float64 { return parseWindow.parseZoom }

// SetZoom records a caller-owned bounded zoom change.
func (parseWindow *parseWindowFake) SetZoom(parseZoom float64) application.Window {
	parseWindow.parseAction, parseWindow.parseZoom = "set-zoom", parseZoom
	return parseWindow
}

// IsFocused returns deterministic fake focus state.
func (parseWindow *parseWindowFake) IsFocused() bool { return parseWindow.isFocused }

// IsMinimised returns deterministic fake minimized state.
func (parseWindow *parseWindowFake) IsMinimised() bool { return parseWindow.isMinimised }

// IsMaximised returns deterministic fake maximized state.
func (parseWindow *parseWindowFake) IsMaximised() bool { return parseWindow.isMaximised }

// IsFullscreen returns deterministic fake fullscreen state.
func (parseWindow *parseWindowFake) IsFullscreen() bool { return parseWindow.isFullscreen }

// IsVisible returns deterministic fake visibility state.
func (parseWindow *parseWindowFake) IsVisible() bool { return parseWindow.isVisible }

// Resizable returns deterministic fake resizable state.
func (parseWindow *parseWindowFake) Resizable() bool { return parseWindow.isResizable }
