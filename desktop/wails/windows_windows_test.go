//go:build windows

package wails

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type parseOwnedWindow struct {
	application.Window
	parseID       uint
	parseVisible  bool
	parseClosed   int
	parseListener func(*application.WindowEvent)
}

// ID returns deterministic fake ownership.
func (parseWindow *parseOwnedWindow) ID() uint { return parseWindow.parseID }

// Width returns valid fake geometry.
func (*parseOwnedWindow) Width() int { return 640 }

// Height returns valid fake geometry.
func (*parseOwnedWindow) Height() int { return 480 }

// IsVisible reports the fake's current visibility.
func (parseWindow *parseOwnedWindow) IsVisible() bool { return parseWindow.parseVisible }

// Close records a requested close without accepting it.
func (parseWindow *parseOwnedWindow) Close() { parseWindow.parseClosed++ }

// Show marks the fake visible.
func (parseWindow *parseOwnedWindow) Show() application.Window {
	parseWindow.parseVisible = true
	return parseWindow
}

// Hide marks the fake hidden.
func (parseWindow *parseOwnedWindow) Hide() application.Window {
	parseWindow.parseVisible = false
	return parseWindow
}

// OnWindowEvent captures the accepted-close listener.
func (parseWindow *parseOwnedWindow) OnWindowEvent(_ events.WindowEventType, parseListener func(*application.WindowEvent)) func() {
	parseWindow.parseListener = parseListener
	return func() {}
}

// TestDefaultBackendDoesNotAdvertiseChildWindows verifies templates require host opt-in.
func TestDefaultBackendDoesNotAdvertiseChildWindows(parseTest *testing.T) {
	for _, parseFeature := range NewNativeBackend().Features() {
		if parseFeature == desktop.ChildWindows {
			parseTest.Fatal("default backend must not advertise child windows")
		}
	}
	parseBackend, parseErr := NewNativeBackendWithWindowTemplates(map[string]WindowTemplate{
		"counter": {Route: "/#/counter", Title: "Counter", Width: 720, Height: 560},
	})
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	for _, parseFeature := range parseBackend.Features() {
		if parseFeature == desktop.ChildWindows {
			return
		}
	}
	parseTest.Fatal("configured backend did not advertise child windows")
}

// TestWindowTemplateRejectsNonSameOriginRoutes verifies callers cannot configure remote navigation.
func TestWindowTemplateRejectsNonSameOriginRoutes(parseTest *testing.T) {
	for _, parseRoute := range []string{"https://example.com", "//example.com/path", `\\example.com\path`, "counter"} {
		_, parseErr := NewNativeBackendWithWindowTemplates(map[string]WindowTemplate{
			"counter": {Route: parseRoute, Title: "Counter", Width: 720, Height: 560},
		})
		if parseErr == nil {
			parseTest.Fatalf("route %q must be rejected", parseRoute)
		}
	}
}

// TestChildCloseRemainsOwnedWhilePending verifies a vetoable close is not reported complete.
func TestChildCloseRemainsOwnedWhilePending(parseTest *testing.T) {
	parseOwner := &parseOwnedWindow{parseID: 1, parseVisible: true}
	parseChild := &parseOwnedWindow{parseID: 2, parseVisible: true}
	parseEntry := &nativeWindowEntry{parseTemplateID: "counter", parseTitle: "Counter", parseWindow: parseChild}
	parseState := &nativeWindowState{parseTemplates: map[string]WindowTemplate{}, parseOwners: map[uint]map[string]*nativeWindowEntry{1: {"child": parseEntry}}, parseHooked: map[uint]bool{}, parseClosing: map[uint]bool{}}
	parseBackend := nativeBackend{parseWindows: parseState}
	parseContext := context.WithValue(context.Background(), application.WindowKey, application.Window(parseOwner))
	parseInfo, parseErr := parseBackend.ControlChildWindow(parseContext, desktop.ChildWindowControlRequest{ID: "child", Action: "close"})
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if !parseInfo.Closing || !parseInfo.Visible || parseChild.parseClosed != 1 {
		parseTest.Fatalf("pending close=%#v calls=%d", parseInfo, parseChild.parseClosed)
	}
	if _, parseErr = parseBackend.InspectChildWindow(parseContext, desktop.ChildWindowRequest{ID: "child"}); parseErr != nil {
		parseTest.Fatalf("pending child lost ownership: %v", parseErr)
	}
	removeOwnedWindow(parseState, 1, "child", parseChild)
	if _, parseErr = parseBackend.InspectChildWindow(parseContext, desktop.ChildWindowRequest{ID: "child"}); parseErr == nil {
		parseTest.Fatal("accepted close must release ownership")
	}
}

// TestChildWindowOwnerLimitCountsReservations verifies in-flight creates consume capacity.
func TestChildWindowOwnerLimitCountsReservations(parseTest *testing.T) {
	parseOwner := &parseOwnedWindow{parseID: 1, parseVisible: true}
	parseOwned := map[string]*nativeWindowEntry{}
	for parseIndex := 0; parseIndex < maximumOwnedWindows; parseIndex++ {
		parseOwned[string(rune('a'+parseIndex))] = nil
	}
	parseState := &nativeWindowState{parseTemplates: map[string]WindowTemplate{"counter": {Route: "/#/counter", Title: "Counter", Width: 640, Height: 480}}, parseOwners: map[uint]map[string]*nativeWindowEntry{1: parseOwned}, parseHooked: map[uint]bool{}, parseClosing: map[uint]bool{}}
	parseContext := context.WithValue(context.Background(), application.WindowKey, application.Window(parseOwner))
	_, parseErr := (nativeBackend{parseWindows: parseState}).CreateChildWindow(parseContext, desktop.ChildWindowCreateRequest{ID: "overflow", TemplateID: "counter"})
	if parseErr == nil || len(parseOwned) != maximumOwnedWindows {
		parseTest.Fatalf("owner limit not enforced: len=%d err=%v", len(parseOwned), parseErr)
	}
}

// TestOwnerCloseVetoDoesNotPoisonChildLifecycle verifies cleanup waits for accepted close listeners.
func TestOwnerCloseVetoDoesNotPoisonChildLifecycle(parseTest *testing.T) {
	parseOwner := &parseOwnedWindow{parseID: 1, parseVisible: true}
	parseState := &nativeWindowState{parseTemplates: map[string]WindowTemplate{"counter": {Route: "/#/counter", Title: "Counter", Width: 640, Height: 480}}, parseOwners: map[uint]map[string]*nativeWindowEntry{}, parseHooked: map[uint]bool{}, parseClosing: map[uint]bool{}}
	parseBackend := nativeBackend{parseWindows: parseState}
	parseContext := context.WithValue(context.Background(), application.WindowKey, application.Window(parseOwner))
	_, _ = parseBackend.CreateChildWindow(parseContext, desktop.ChildWindowCreateRequest{ID: "first", TemplateID: "counter"})
	if parseOwner.parseListener == nil {
		parseTest.Fatal("accepted-close listener was not registered")
	}
	// A host hook veto means Wails never invokes this listener. The failed
	// create above must have rolled its reservation back without closing owner state.
	if parseState.parseClosing[1] || len(parseState.parseOwners[1]) != 0 {
		parseTest.Fatalf("veto poisoned owner state: closing=%v owned=%d", parseState.parseClosing[1], len(parseState.parseOwners[1]))
	}
	_, parseErr := parseBackend.CreateChildWindow(parseContext, desktop.ChildWindowCreateRequest{ID: "second", TemplateID: "counter"})
	if parseErr == nil || parseState.parseClosing[1] {
		parseTest.Fatalf("future create was blocked as closing after veto: %v", parseErr)
	}
	// Invoking the listener models all host hooks accepting the close.
	parseOwner.parseListener(application.NewWindowEvent())
	if !parseState.parseClosing[1] {
		parseTest.Fatal("accepted close did not establish reservation barrier")
	}
}
