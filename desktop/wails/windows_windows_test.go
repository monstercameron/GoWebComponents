//go:build windows

package wails

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type fakeOwnedWindow struct {
	application.Window
	id         uint
	x, y       int
	width      int
	height     int
	title      string
	visible    bool
	focused    bool
	minimised  bool
	maximised  bool
	closed     int
	lastAction string
	listener   func(*application.WindowEvent)
}

// ID returns deterministic fake ownership.
func (w *fakeOwnedWindow) ID() uint { return w.id }

// Position returns deterministic fake position.
func (w *fakeOwnedWindow) Position() (int, int) { return w.x, w.y }

// Width returns valid fake geometry.
func (w *fakeOwnedWindow) Width() int {
	if w.width == 0 {
		return 640
	}
	return w.width
}

// Height returns valid fake geometry.
func (w *fakeOwnedWindow) Height() int {
	if w.height == 0 {
		return 480
	}
	return w.height
}

// IsVisible reports the fake's current visibility.
func (w *fakeOwnedWindow) IsVisible() bool { return w.visible }

// IsFocused reports the fake's current focus state.
func (w *fakeOwnedWindow) IsFocused() bool { return w.focused }

// IsMinimised reports the fake's current minimized state.
func (w *fakeOwnedWindow) IsMinimised() bool { return w.minimised }

// IsMaximised reports the fake's current maximized state.
func (w *fakeOwnedWindow) IsMaximised() bool { return w.maximised }

// Close records a requested close without accepting it.
func (w *fakeOwnedWindow) Close() { w.closed++; w.lastAction = "close" }

// Show marks the fake visible.
func (w *fakeOwnedWindow) Show() application.Window {
	w.visible = true
	w.lastAction = "show"
	return w
}

// Hide marks the fake hidden.
func (w *fakeOwnedWindow) Hide() application.Window {
	w.visible = false
	w.lastAction = "hide"
	return w
}

// Focus marks the fake focused.
func (w *fakeOwnedWindow) Focus() { w.focused = true; w.lastAction = "focus" }

// Minimise marks the fake minimized.
func (w *fakeOwnedWindow) Minimise() application.Window {
	w.minimised = true
	w.lastAction = "minimize"
	return w
}

// UnMinimise clears minimized state.
func (w *fakeOwnedWindow) UnMinimise() { w.minimised = false; w.lastAction = "unminimize" }

// Maximise marks the fake maximized.
func (w *fakeOwnedWindow) Maximise() application.Window {
	w.maximised = true
	w.lastAction = "maximize"
	return w
}

// UnMaximise clears maximized state.
func (w *fakeOwnedWindow) UnMaximise() { w.maximised = false; w.lastAction = "unmaximize" }

// Restore clears minimized and maximized states.
func (w *fakeOwnedWindow) Restore() {
	w.minimised = false
	w.maximised = false
	w.lastAction = "restore"
}

// Center records a center request.
func (w *fakeOwnedWindow) Center() { w.lastAction = "center" }

// SetSize records resized dimensions.
func (w *fakeOwnedWindow) SetSize(width, height int) application.Window {
	w.width, w.height = width, height
	w.lastAction = "resize"
	return w
}

// SetTitle records an updated title.
func (w *fakeOwnedWindow) SetTitle(title string) application.Window {
	w.title = title
	w.lastAction = "set-title"
	return w
}

// SetBounds records updated position and size.
func (w *fakeOwnedWindow) SetBounds(bounds application.Rect) {
	w.x, w.y, w.width, w.height = bounds.X, bounds.Y, bounds.Width, bounds.Height
	w.lastAction = "set-bounds"
}

// OnWindowEvent captures the accepted-close listener.
func (w *fakeOwnedWindow) OnWindowEvent(_ events.WindowEventType, listener func(*application.WindowEvent)) func() {
	w.listener = listener
	return func() {}
}

// TestDefaultBackendDoesNotAdvertiseChildWindows verifies templates require host opt-in.
func TestDefaultBackendDoesNotAdvertiseChildWindows(t *testing.T) {
	for _, feature := range NewNativeBackend().Features() {
		if feature == desktop.ChildWindows {
			t.Fatal("default backend must not advertise child windows")
		}
	}
	backend, err := NewNativeBackendWithWindowTemplates(map[string]WindowTemplate{
		"counter": {Route: "/#/counter", Title: "Counter", Width: 720, Height: 560, MinWidth: 320, MinHeight: 240, MaxWidth: 1600, MaxHeight: 1200, Backdrop: WindowBackdropMica},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, feature := range backend.Features() {
		if feature == desktop.ChildWindows {
			return
		}
	}
	t.Fatal("configured backend did not advertise child windows")
}

// TestWindowTemplateRejectsNonSameOriginRoutesAndInvalidConstraints verifies template validation.
func TestWindowTemplateRejectsNonSameOriginRoutesAndInvalidConstraints(t *testing.T) {
	for _, route := range []string{"https://example.com", "//example.com/path", `\\example.com\path`, "counter"} {
		_, err := NewNativeBackendWithWindowTemplates(map[string]WindowTemplate{
			"counter": {Route: route, Title: "Counter", Width: 720, Height: 560},
		})
		if err == nil {
			t.Fatalf("route %q must be rejected", route)
		}
	}
	for _, tmpl := range []WindowTemplate{
		{Route: "/#/c", Title: "C", Width: 720, Height: 560, MinWidth: 800, MaxWidth: 600},
		{Route: "/#/c", Title: "C", Width: 720, Height: 560, Backdrop: WindowBackdrop("glass")},
	} {
		if _, err := NewNativeBackendWithWindowTemplates(map[string]WindowTemplate{"c": tmpl}); err == nil {
			t.Fatalf("invalid template %+v must be rejected", tmpl)
		}
	}
}

// TestChildWindowExtendedControls verifies bounded child operations update state safely.
func TestChildWindowExtendedControls(t *testing.T) {
	owner := &fakeOwnedWindow{id: 1, visible: true}
	child := &fakeOwnedWindow{id: 2, width: 640, height: 480, visible: true}
	entry := &nativeWindowEntry{templateID: "counter", title: "Counter", window: child}
	state := &nativeWindowState{
		templates: map[string]WindowTemplate{},
		owners:    map[uint]map[string]*nativeWindowEntry{1: {"child": entry}},
		hooked:    map[uint]bool{},
		closing:   map[uint]bool{},
	}
	backend := nativeBackend{windows: state}
	ctx := context.WithValue(context.Background(), application.WindowKey, application.Window(owner))

	info, err := backend.ControlChildWindow(ctx, desktop.ChildWindowControlRequest{ID: "child", Action: "set-title", Title: "Inspector"})
	if err != nil || info.Title != "Inspector" || child.title != "Inspector" {
		t.Fatalf("set-title failed: info=%+v err=%v", info, err)
	}
	info, err = backend.ControlChildWindow(ctx, desktop.ChildWindowControlRequest{ID: "child", Action: "set-bounds", X: 40, Y: 50, Width: 800, Height: 600})
	if err != nil || info.X != 40 || info.Y != 50 || info.Width != 800 || info.Height != 600 {
		t.Fatalf("set-bounds failed: info=%+v err=%v", info, err)
	}
	info, err = backend.ControlChildWindow(ctx, desktop.ChildWindowControlRequest{ID: "child", Action: "resize", Width: 900, Height: 700})
	if err != nil || info.Width != 900 || info.Height != 700 {
		t.Fatalf("resize failed: info=%+v err=%v", info, err)
	}
	for _, action := range []string{"focus", "minimize", "unminimize", "maximize", "unmaximize", "restore", "center"} {
		if _, err = backend.ControlChildWindow(ctx, desktop.ChildWindowControlRequest{ID: "child", Action: action}); err != nil {
			t.Fatalf("action %q failed: %v", action, err)
		}
		if child.lastAction != action {
			t.Fatalf("action %q did not apply: last=%q", action, child.lastAction)
		}
	}
}

// TestChildCloseRemainsOwnedWhilePending verifies a vetoable close is not reported complete.
func TestChildCloseRemainsOwnedWhilePending(t *testing.T) {
	owner := &fakeOwnedWindow{id: 1, visible: true}
	child := &fakeOwnedWindow{id: 2, visible: true}
	entry := &nativeWindowEntry{templateID: "counter", title: "Counter", window: child}
	state := &nativeWindowState{
		templates: map[string]WindowTemplate{},
		owners:    map[uint]map[string]*nativeWindowEntry{1: {"child": entry}},
		hooked:    map[uint]bool{},
		closing:   map[uint]bool{},
	}
	backend := nativeBackend{windows: state}
	ctx := context.WithValue(context.Background(), application.WindowKey, application.Window(owner))
	info, err := backend.ControlChildWindow(ctx, desktop.ChildWindowControlRequest{ID: "child", Action: "close"})
	if err != nil {
		t.Fatal(err)
	}
	if !info.Closing || !info.Visible || child.closed != 1 {
		t.Fatalf("pending close=%#v calls=%d", info, child.closed)
	}
	if _, err = backend.InspectChildWindow(ctx, desktop.ChildWindowRequest{ID: "child"}); err != nil {
		t.Fatalf("pending child lost ownership: %v", err)
	}
	removeOwnedWindow(state, 1, "child", child)
	if _, err = backend.InspectChildWindow(ctx, desktop.ChildWindowRequest{ID: "child"}); err == nil {
		t.Fatal("accepted close must release ownership")
	}
}

// TestChildWindowOwnerLimitCountsReservations verifies in-flight creates consume capacity.
func TestChildWindowOwnerLimitCountsReservations(t *testing.T) {
	owner := &fakeOwnedWindow{id: 1, visible: true}
	owned := map[string]*nativeWindowEntry{}
	for i := 0; i < maximumOwnedWindows; i++ {
		owned[string(rune('a'+i))] = nil
	}
	state := &nativeWindowState{
		templates: map[string]WindowTemplate{"counter": {Route: "/#/counter", Title: "Counter", Width: 640, Height: 480}},
		owners:    map[uint]map[string]*nativeWindowEntry{1: owned},
		hooked:    map[uint]bool{},
		closing:   map[uint]bool{},
	}
	ctx := context.WithValue(context.Background(), application.WindowKey, application.Window(owner))
	_, err := (nativeBackend{windows: state}).CreateChildWindow(ctx, desktop.ChildWindowCreateRequest{ID: "overflow", TemplateID: "counter"})
	if err == nil || len(owned) != maximumOwnedWindows {
		t.Fatalf("owner limit not enforced: len=%d err=%v", len(owned), err)
	}
}

// TestOwnerCloseVetoDoesNotPoisonChildLifecycle verifies cleanup waits for accepted close listeners.
func TestOwnerCloseVetoDoesNotPoisonChildLifecycle(t *testing.T) {
	owner := &fakeOwnedWindow{id: 1, visible: true}
	state := &nativeWindowState{
		templates: map[string]WindowTemplate{"counter": {Route: "/#/counter", Title: "Counter", Width: 640, Height: 480}},
		owners:    map[uint]map[string]*nativeWindowEntry{},
		hooked:    map[uint]bool{},
		closing:   map[uint]bool{},
	}
	backend := nativeBackend{windows: state}
	ctx := context.WithValue(context.Background(), application.WindowKey, application.Window(owner))
	_, _ = backend.CreateChildWindow(ctx, desktop.ChildWindowCreateRequest{ID: "first", TemplateID: "counter"})
	if owner.listener == nil {
		t.Fatal("accepted-close listener was not registered")
	}
	if state.closing[1] || len(state.owners[1]) != 0 {
		t.Fatalf("veto poisoned owner state: closing=%v owned=%d", state.closing[1], len(state.owners[1]))
	}
	_, err := backend.CreateChildWindow(ctx, desktop.ChildWindowCreateRequest{ID: "second", TemplateID: "counter"})
	if err == nil || state.closing[1] {
		t.Fatalf("future create was blocked as closing after veto: %v", err)
	}
	owner.listener(application.NewWindowEvent())
	if !state.closing[1] {
		t.Fatal("accepted close did not establish reservation barrier")
	}
}
