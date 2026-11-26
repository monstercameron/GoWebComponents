//go:build windows

package wails

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/monstercameron/GoWebComponents/v8/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type fakeTrayWindow struct {
	application.Window
	mu             sync.Mutex
	trayEvents     []desktop.TrayEvent
	shortcutEvents []desktop.ShortcutEvent
}

// ID returns a stable caller identity.
func (*fakeTrayWindow) ID() uint { return 77 }

// OnWindowEvent records lifecycle hooks without requiring a live Wails runtime.
func (*fakeTrayWindow) OnWindowEvent(events.WindowEventType, func(*application.WindowEvent)) func() {
	return func() {}
}

// EmitEvent records typed tray callbacks.
func (w *fakeTrayWindow) EmitEvent(_ string, data ...any) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(data) == 1 {
		if event, ok := data[0].(desktop.TrayEvent); ok {
			w.trayEvents = append(w.trayEvents, event)
		}
		if event, ok := data[0].(desktop.ShortcutEvent); ok {
			w.shortcutEvents = append(w.shortcutEvents, event)
		}
	}
	return true
}

// TestShortcutGenerationNeverRevivesStaleCallbacks verifies removed IDs retain monotonic tokens.
func TestShortcutGenerationNeverRevivesStaleCallbacks(t *testing.T) {
	win := &fakeTrayWindow{}
	owner := &nativeOwnerState{
		shortcuts:           map[string]string{"main": "Ctrl+K"},
		shortcutGenerations: map[string]uint64{"main": 4},
	}
	state := newNativeTrayShortcutState()
	state.owners[win.ID()] = owner
	state.emitShortcutEvent(win, "main", 3)
	state.emitShortcutEvent(win, "main", 4)
	delete(owner.shortcuts, "main")
	state.emitShortcutEvent(win, "main", 4)
	owner.shortcuts["main"] = "Ctrl+L"
	owner.shortcutGenerations["main"]++
	state.emitShortcutEvent(win, "main", 4)
	state.emitShortcutEvent(win, "main", 5)
	if len(win.shortcutEvents) != 2 {
		t.Fatalf("stale shortcut callbacks revived: %#v", win.shortcutEvents)
	}
}

// TestDefaultTrayIconIsValid verifies the synthetic fallback is a bounded decodable PNG.
func TestDefaultTrayIconIsValid(t *testing.T) {
	icon, err := base64.StdEncoding.DecodeString(defaultTrayIconPNG)
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(icon))
	if err != nil || config.Width != 16 || config.Height != 16 {
		t.Fatalf("invalid default tray icon: %+v: %v", config, err)
	}
}

// TestTrayShortcutBackendExtensionsRequireCaller verifies optional capabilities never fall back to focus.
func TestTrayShortcutBackendExtensionsRequireCaller(t *testing.T) {
	backend := NewNativeBackend()
	trayBackend, ok := backend.(desktop.TrayBackend)
	if !ok {
		t.Fatal("Windows backend does not implement tray extension")
	}
	if err := trayBackend.ConfigureTray(context.Background(), desktop.TrayRequest{Action: "upsert", ID: "main"}); err == nil {
		t.Fatal("tray accepted missing caller")
	}
	win := &fakeTrayWindow{}
	ctx := context.WithValue(context.Background(), application.WindowKey, application.Window(win))
	if err := trayBackend.ConfigureTray(ctx, desktop.TrayRequest{Action: "position-window", ID: "missing", Offset: 8}); !interop.IsCode(err, interop.CodeInvalid) {
		t.Fatalf("unregistered tray position-window error=%v", err)
	}
	shortcutBackend, ok := backend.(desktop.ShortcutBackend)
	if !ok {
		t.Fatal("Windows backend does not implement shortcut extension")
	}
	if err := shortcutBackend.ConfigureShortcut(context.Background(), desktop.ShortcutRequest{Action: "upsert", ID: "main", Accelerator: "Ctrl+Shift+K"}); err == nil {
		t.Fatal("shortcut accepted missing caller")
	}
}

// TestTrayGenerationNeverRevivesStaleCallbacks verifies remove and recreate keep monotonic identity.
func TestTrayGenerationNeverRevivesStaleCallbacks(t *testing.T) {
	win := &fakeTrayWindow{}
	owner := &nativeOwnerState{
		trays:           map[string]*application.SystemTray{"main": nil},
		trayGenerations: map[string]uint64{"main": 2},
	}
	state := newNativeTrayShortcutState()
	state.owners[win.ID()] = owner
	state.emitTrayEvent(win, "main", 1, "click", "")
	state.emitTrayEvent(win, "main", 2, "click", "")
	delete(owner.trays, "main")
	state.emitTrayEvent(win, "main", 2, "click", "")
	owner.trays["main"] = nil
	owner.trayGenerations["main"]++
	state.emitTrayEvent(win, "main", 2, "click", "")
	state.emitTrayEvent(win, "main", 3, "click", "")
	if len(win.trayEvents) != 2 {
		t.Fatalf("stale callbacks revived: %#v", win.trayEvents)
	}
}

// TestTrayCallbackDoesNotWaitForOperationLock verifies UI callbacks cannot deadlock native dispatch.
func TestTrayCallbackDoesNotWaitForOperationLock(t *testing.T) {
	win := &fakeTrayWindow{}
	state := newNativeTrayShortcutState()
	state.owners[win.ID()] = &nativeOwnerState{
		trays:           map[string]*application.SystemTray{"main": nil},
		trayGenerations: map[string]uint64{"main": 1},
	}
	state.opMu.Lock()
	done := make(chan struct{})
	go func() {
		state.emitTrayEvent(win, "main", 1, "click", "")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("tray callback waited for operation lock")
	}
	state.opMu.Unlock()
}

// TestTrayPublicationUsesSingleUIBoundary verifies complete configuration is wrapped as one UI action.
func TestTrayPublicationUsesSingleUIBoundary(t *testing.T) {
	state := newNativeTrayShortcutState()
	calls := 0
	isInsideUI := false
	state.invokeUI = func(action func()) {
		calls++
		isInsideUI = true
		action()
		isInsideUI = false
	}
	isConfigured := false
	state.runTrayUI(func() {
		if !isInsideUI {
			t.Fatal("tray configuration escaped UI boundary")
		}
		isConfigured = true
	})
	if calls != 1 || !isConfigured || isInsideUI {
		t.Fatalf("unexpected UI publication: calls=%d configured=%v inside=%v", calls, isConfigured, isInsideUI)
	}
}
