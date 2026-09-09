//go:build windows

package wails

import (
	"context"
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var parseWindowMenus = map[*parseWindowFake]*application.Menu{}
var parseWindowCloseHandlers = map[*parseWindowFake]func(*application.WindowEvent){}
var parseWindowMenuBlockers = map[*parseWindowFake]<-chan struct{}{}
var parseWindowMenuEntered = map[*parseWindowFake]chan<- struct{}{}

// SetMenu records the menu installed on this exact fake window.
func (parseWindow *parseWindowFake) SetMenu(parseMenu *application.Menu) {
	if parseBlock := parseWindowMenuBlockers[parseWindow]; parseBlock != nil {
		if parseEntered := parseWindowMenuEntered[parseWindow]; parseEntered != nil {
			close(parseEntered)
			delete(parseWindowMenuEntered, parseWindow)
		}
		<-parseBlock
		delete(parseWindowMenuBlockers, parseWindow)
	}
	parseWindowMenus[parseWindow] = parseMenu
}

// OnWindowEvent records caller-window cleanup and returns an idempotent cancellation function.
func (parseWindow *parseWindowFake) OnWindowEvent(_ events.WindowEventType, parseHandler func(*application.WindowEvent)) func() {
	parseWindowCloseHandlers[parseWindow] = parseHandler
	return func() { delete(parseWindowCloseHandlers, parseWindow) }
}

// TestRuntimeMenuSerializesConcurrentReplacement verifies an older build cannot install after a newer request.
func TestRuntimeMenuSerializesConcurrentReplacement(parseTest *testing.T) {
	parseWindow := &parseWindowFake{parseID: 303}
	parseContext := context.WithValue(context.Background(), application.WindowKey, application.Window(parseWindow))
	parseRelease := make(chan struct{})
	parseEntered := make(chan struct{})
	parseWindowMenuBlockers[parseWindow] = parseRelease
	parseWindowMenuEntered[parseWindow] = parseEntered
	parseBackend := nativeBackend{}
	parseFirst := desktop.Menu{Items: []desktop.MenuItem{{ID: "first", Kind: desktop.MenuItemCommand, Label: "First", Enabled: true}}}
	parseSecond := desktop.Menu{Items: []desktop.MenuItem{{ID: "second", Kind: desktop.MenuItemCommand, Label: "Second", Enabled: true}}}
	var parseWait sync.WaitGroup
	parseWait.Add(2)
	go func() { defer parseWait.Done(); _ = parseBackend.ReplaceMenu(parseContext, parseFirst) }()
	<-parseEntered
	parseSecondStarted := make(chan struct{})
	go func() {
		defer parseWait.Done()
		close(parseSecondStarted)
		_ = parseBackend.ReplaceMenu(parseContext, parseSecond)
	}()
	<-parseSecondStarted
	close(parseRelease)
	parseWait.Wait()
	parseInstalled := parseWindowMenus[parseWindow]
	if parseInstalled == nil || parseInstalled.ItemAt(0) == nil || parseInstalled.ItemAt(0).Label() != "Second" {
		parseTest.Fatal("concurrent replacement installed stale menu last")
	}
}

// TestContextMenuReinstallNeverRevivesRemovedCallbacks verifies config identities are not reusable counters.
func TestContextMenuReinstallNeverRevivesRemovedCallbacks(parseTest *testing.T) {
	parseWindowID := uint(404)
	parseOldToken := &menuConfigToken{}
	parseState := &menuWindowState{parseContexts: map[string]*menuContextState{"row": {parseToken: parseOldToken}}}
	menuWindows.Lock()
	menuWindows.parseStates[parseWindowID] = parseState
	delete(parseState.parseContexts, "row")
	parseState.parseContexts["row"] = &menuContextState{parseToken: &menuConfigToken{}}
	isOldCurrent := isMenuConfigCurrentLocked(parseWindowID, parseState, parseOldToken, "row")
	delete(menuWindows.parseStates, parseWindowID)
	menuWindows.Unlock()
	if isOldCurrent {
		parseTest.Fatal("reinstalling a removed context menu revived its old callback token")
	}
}

// EmitEvent records a typed event without broadcasting it to another fake window.
func (parseWindow *parseWindowFake) EmitEvent(string, ...any) bool { return true }

// TestRuntimeMenuTargetsOnlyCallerWindow verifies per-window ownership and close cleanup.
func TestRuntimeMenuTargetsOnlyCallerWindow(parseTest *testing.T) {
	parseFirst := &parseWindowFake{parseID: 101}
	parseSecond := &parseWindowFake{parseID: 202}
	parseBackend := nativeBackend{}
	parseMenu := desktop.Menu{Items: []desktop.MenuItem{{ID: "file", Kind: desktop.MenuItemSubmenu, Label: "File", Enabled: false, Hidden: true, Items: []desktop.MenuItem{{ID: "open", Kind: desktop.MenuItemCommand, Label: "Open", Enabled: true}}}}}
	parseFirstContext := context.WithValue(context.Background(), application.WindowKey, application.Window(parseFirst))
	if parseErr := parseBackend.ReplaceMenu(parseFirstContext, parseMenu); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseWindowMenus[parseFirst] == nil || parseWindowMenus[parseSecond] != nil {
		parseTest.Fatal("menu replacement escaped caller window")
	}
	parseTop := parseWindowMenus[parseFirst].ItemAt(0)
	if parseTop == nil || parseTop.Enabled() || !parseTop.Hidden() || !parseTop.IsSubmenu() {
		parseTest.Fatal("disabled submenu was not preserved")
	}
	parseClose := parseWindowCloseHandlers[parseFirst]
	if parseClose == nil {
		parseTest.Fatal("caller-window close cleanup was not registered")
	}
	parseClose(application.NewWindowEvent())
	if parseWindowMenus[parseFirst] == nil || parseWindowMenus[parseFirst].ItemAt(0) != nil {
		parseTest.Fatal("caller-window menu was not cleared on close")
	}
}
