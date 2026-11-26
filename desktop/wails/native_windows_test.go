//go:build windows

package wails

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/monstercameron/GoWebComponents/v8/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// TestNativeBackendMissingCallerNeverUsesFocusedWindow verifies every adapter operation requires its caller.
func TestNativeBackendMissingCallerNeverUsesFocusedWindow(t *testing.T) {
	backend := NewNativeBackend()
	if _, err := backend.ClipboardRead(context.Background()); err == nil {
		t.Fatal("clipboard read accepted missing caller")
	}
	if err := backend.ClipboardWrite(context.Background(), desktop.ClipboardWriteRequest{Text: "x"}); err == nil {
		t.Fatal("clipboard write accepted missing caller")
	}
	if _, err := backend.ShowMessage(context.Background(), desktop.MessageRequest{Kind: "info"}); err == nil {
		t.Fatal("message accepted missing caller")
	}
	if _, err := backend.Window(context.Background(), desktop.WindowRequest{Action: "info"}); err == nil {
		t.Fatal("window accepted missing caller")
	}
	if _, err := backend.Screens(context.Background()); err == nil {
		t.Fatal("screens accepted missing caller")
	}
	systemBackend := backend.(desktop.SystemEnvironmentBackend)
	if _, err := systemBackend.SystemEnvironment(context.Background()); err == nil {
		t.Fatal("system environment accepted missing caller")
	}
	urlsBackend := backend.(desktop.ExternalURLBackend)
	if err := urlsBackend.OpenExternalURL(context.Background(), desktop.ExternalURLOpenRequest{URL: "https://example.com"}); err == nil {
		t.Fatal("external URL accepted missing caller")
	}
	autostartBackend := backend.(desktop.AutostartBackend)
	if _, err := autostartBackend.AutostartStatus(context.Background()); err == nil {
		t.Fatal("autostart status accepted missing caller")
	}
	if err := autostartBackend.AutostartEnable(context.Background()); err == nil {
		t.Fatal("autostart enable accepted missing caller")
	}
	if err := autostartBackend.AutostartDisable(context.Background()); err == nil {
		t.Fatal("autostart disable accepted missing caller")
	}
	fileManagerBackend := backend.(desktop.FileManagerBackend)
	if err := fileManagerBackend.RevealPath(context.Background(), desktop.FileManagerRevealRequest{Path: `C:\fixture`}); err == nil {
		t.Fatal("file manager accepted missing caller")
	}
}

// TestWindowsMessageRequestValidationMatchesV310Capabilities verifies MessageBox and TaskDialog rules.
func TestWindowsMessageRequestValidationMatchesV310Capabilities(t *testing.T) {
	for _, req := range []desktop.MessageRequest{
		{Kind: "info", Buttons: []string{"Ok"}, DefaultButton: "Ok", CancelButton: "Ok"},
		{Kind: "warning"},
		{Kind: "error"},
		{Kind: "question", DefaultButton: "No"},
		{Kind: "question", CancelButton: "No"},
		{Kind: "question", Buttons: []string{"Save", "Discard", "Cancel"}, DefaultButton: "Save", CancelButton: "Cancel"},
		{Kind: "question", Buttons: []string{"Confirm"}, DefaultButton: "Confirm"},
	} {
		if err := validateWindowsMessageRequest(req); err != nil {
			t.Errorf("accepted native request rejected: %+v: %v", req, err)
		}
	}
	for _, req := range []desktop.MessageRequest{
		{Kind: "info", Buttons: []string{"Continue"}},
		{Kind: "warning", DefaultButton: "Yes"},
		{Kind: "error", CancelButton: "No"},
		{Kind: "question", DefaultButton: "Ok"},
		{Kind: "question", CancelButton: "Missing"},
		{Kind: "question", Buttons: []string{"A", "B", "C", "D"}},
		{Kind: "question", Buttons: []string{"Save", "Save"}},
		{Kind: "question", Buttons: []string{"   "}},
		{Kind: "unknown"},
	} {
		if err := validateWindowsMessageRequest(req); err == nil {
			t.Errorf("unsupported native request accepted: %+v", req)
		}
	}
}

// TestWindowUsesCallerAndReportsRuntimeState verifies typed operations target only the context window.
func TestWindowUsesCallerAndReportsRuntimeState(t *testing.T) {
	win := &fakeWindow{
		id:        7,
		name:      "caller",
		x:         -20,
		y:         30,
		width:     640,
		height:    480,
		zoom:      1.25,
		focused:   true,
		visible:   true,
		resizable: true,
		screenID:  "display-1",
		borders:   &application.LRTB{Top: 32, Left: 8, Bottom: 8, Right: 8},
	}
	ctx := context.WithValue(context.Background(), application.WindowKey, application.Window(win))
	backend := nativeBackend{}

	info, err := backend.Window(ctx, desktop.WindowRequest{Action: "set-position", X: 12, Y: 34})
	if err != nil {
		t.Fatal(err)
	}
	if win.lastAction != "set-position" || win.x != 12 || win.y != 34 {
		t.Fatalf("operation not applied to caller: %+v", win)
	}
	if info.ID != "7" || info.Name != "caller" || info.X != 12 || info.Y != 34 || info.Width != 640 || info.Height != 480 || !info.Focused || !info.Visible || !info.Resizable || info.Zoom != 1.25 || info.ScreenID != "display-1" || info.BorderSizes == nil || info.BorderSizes.Top != 32 {
		t.Fatalf("unexpected window info: %+v", info)
	}

	// Sub-1.0 zoom is supported on Windows in Wails v3.1.0 (#6166).
	info, err = backend.Window(ctx, desktop.WindowRequest{Action: "set-zoom", Zoom: 0.5})
	if err != nil || win.lastAction != "set-zoom" || info.Zoom != 0.5 {
		t.Fatalf("sub-1.0 zoom not applied to caller: info=%+v window=%+v err=%v", info, win, err)
	}

	info, err = backend.Window(ctx, desktop.WindowRequest{Action: "set-ignore-mouse-events", Enabled: true})
	if err != nil || !info.IgnoreMouseEvents || !win.ignoreMouse {
		t.Fatalf("ignore-mouse not applied: info=%+v err=%v", info, err)
	}

	info, err = backend.Window(ctx, desktop.WindowRequest{Action: "set-enabled", Enabled: false})
	if err != nil || win.lastAction != "set-enabled" || win.enabled {
		t.Fatalf("set-enabled not applied: info=%+v window=%+v err=%v", info, win, err)
	}

	for _, action := range []string{"hide", "show", "reload", "snap-assist", "close"} {
		if _, err = backend.Window(ctx, desktop.WindowRequest{Action: action}); err != nil {
			t.Fatalf("action %q failed: %v", action, err)
		}
		if win.lastAction != action {
			t.Fatalf("action %q did not dispatch: last=%q", action, win.lastAction)
		}
	}
}

// TestWindowRejectsUnsupportedWindowsNoOps keeps Windows capability claims honest.
func TestWindowRejectsUnsupportedWindowsNoOps(t *testing.T) {
	win := &fakeWindow{id: 7, name: "caller", width: 640, height: 480, zoom: 1}
	ctx := context.WithValue(context.Background(), application.WindowKey, application.Window(win))
	for _, req := range []desktop.WindowRequest{
		{Action: "set-fullscreen-button-state", State: "disabled"},
		{Action: "force-reload"},
	} {
		_, err := (nativeBackend{}).Window(ctx, req)
		if !interop.IsCode(err, interop.CodeUnavailable) {
			t.Fatalf("Windows no-op %q reported wrong error: %v", req.Action, err)
		}
	}
}

// TestWindowRejectsOutOfBoundsZoom enforces portable zoom bounds before calling Wails.
func TestWindowRejectsOutOfBoundsZoom(t *testing.T) {
	win := &fakeWindow{id: 7, name: "caller", width: 640, height: 480, zoom: 1}
	ctx := context.WithValue(context.Background(), application.WindowKey, application.Window(win))
	for _, zoom := range []float64{0.1, 5.5} {
		_, err := (nativeBackend{}).Window(ctx, desktop.WindowRequest{Action: "set-zoom", Zoom: zoom})
		if !interop.IsCode(err, interop.CodeInvalid) || win.lastAction != "" || win.zoom != 1 {
			t.Fatalf("out-of-bounds zoom %v reached Wails: action=%q zoom=%v err=%v", zoom, win.lastAction, win.zoom, err)
		}
	}
}

type fakeWindow struct {
	application.Window
	lastAction                     string
	name                           string
	screenID                       string
	borders                        *application.LRTB
	id, x, y                       int
	width, height                  int
	zoom                           float64
	focused, minimised, maximised  bool
	fullscreen, visible, resizable bool
	ignoreMouse, enabled           bool
}

// ID returns deterministic fake identity.
func (w *fakeWindow) ID() uint { return uint(w.id) }

// Name returns deterministic fake name.
func (w *fakeWindow) Name() string { return w.name }

// Position returns deterministic fake position.
func (w *fakeWindow) Position() (int, int) { return w.x, w.y }

// RelativePosition returns deterministic fake client-relative position.
func (w *fakeWindow) RelativePosition() (int, int) { return w.x, w.y }

// SetPosition records a caller-owned move.
func (w *fakeWindow) SetPosition(x, y int) { w.lastAction, w.x, w.y = "set-position", x, y }

// Width returns deterministic fake width.
func (w *fakeWindow) Width() int { return w.width }

// Height returns deterministic fake height.
func (w *fakeWindow) Height() int { return w.height }

// GetZoom returns deterministic fake zoom.
func (w *fakeWindow) GetZoom() float64 { return w.zoom }

// SetZoom records a caller-owned bounded zoom change.
func (w *fakeWindow) SetZoom(zoom float64) application.Window {
	w.lastAction, w.zoom = "set-zoom", zoom
	return w
}

// IsFocused returns deterministic fake focus state.
func (w *fakeWindow) IsFocused() bool { return w.focused }

// IsMinimised returns deterministic fake minimized state.
func (w *fakeWindow) IsMinimised() bool { return w.minimised }

// IsMaximised returns deterministic fake maximized state.
func (w *fakeWindow) IsMaximised() bool { return w.maximised }

// IsFullscreen returns deterministic fake fullscreen state.
func (w *fakeWindow) IsFullscreen() bool { return w.fullscreen }

// IsVisible returns deterministic fake visibility state.
func (w *fakeWindow) IsVisible() bool { return w.visible }

// Resizable returns deterministic fake resizable state.
func (w *fakeWindow) Resizable() bool { return w.resizable }

// IsIgnoreMouseEvents returns deterministic fake mouse pass-through state.
func (w *fakeWindow) IsIgnoreMouseEvents() bool { return w.ignoreMouse }

// SetIgnoreMouseEvents records a mouse pass-through toggle.
func (w *fakeWindow) SetIgnoreMouseEvents(ignore bool) application.Window {
	w.lastAction, w.ignoreMouse = "set-ignore-mouse-events", ignore
	return w
}

// SetEnabled records an input-enabled toggle.
func (w *fakeWindow) SetEnabled(enabled bool) {
	w.lastAction, w.enabled = "set-enabled", enabled
}

// GetScreen returns deterministic fake screen metadata.
func (w *fakeWindow) GetScreen() (*application.Screen, error) {
	if w.screenID == "" {
		return nil, nil
	}
	return &application.Screen{ID: w.screenID}, nil
}

// GetBorderSizes returns deterministic fake border thicknesses.
func (w *fakeWindow) GetBorderSizes() *application.LRTB { return w.borders }

// Show records a visibility show request.
func (w *fakeWindow) Show() application.Window {
	w.lastAction, w.visible = "show", true
	return w
}

// Hide records a visibility hide request.
func (w *fakeWindow) Hide() application.Window {
	w.lastAction, w.visible = "hide", false
	return w
}

// Close records a caller-owned close request.
func (w *fakeWindow) Close() { w.lastAction = "close" }

// Reload records a caller-owned reload request.
func (w *fakeWindow) Reload() { w.lastAction = "reload" }

// SnapAssist records a caller-owned Snap Assist request.
func (w *fakeWindow) SnapAssist() { w.lastAction = "snap-assist" }
