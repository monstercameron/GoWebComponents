//go:build windows

package wails

import (
	"context"
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var (
	testMenuMu            sync.Mutex
	testWindowMenus       = map[*fakeWindow]*application.Menu{}
	testWindowCloseHooks  = map[*fakeWindow]func(*application.WindowEvent){}
	testWindowMenuBlocker = map[*fakeWindow]<-chan struct{}{}
	testWindowMenuEntered = map[*fakeWindow]chan<- struct{}{}
)

// SetMenu records the menu installed on this exact fake window.
func (w *fakeWindow) SetMenu(menu *application.Menu) {
	testMenuMu.Lock()
	blocker := testWindowMenuBlocker[w]
	entered := testWindowMenuEntered[w]
	if blocker != nil {
		delete(testWindowMenuBlocker, w)
		delete(testWindowMenuEntered, w)
	}
	testMenuMu.Unlock()
	if blocker != nil {
		if entered != nil {
			close(entered)
		}
		<-blocker
	}
	testMenuMu.Lock()
	testWindowMenus[w] = menu
	testMenuMu.Unlock()
}

// OnWindowEvent records caller-window cleanup and returns an idempotent cancellation function.
func (w *fakeWindow) OnWindowEvent(_ events.WindowEventType, handler func(*application.WindowEvent)) func() {
	testMenuMu.Lock()
	testWindowCloseHooks[w] = handler
	testMenuMu.Unlock()
	return func() {
		testMenuMu.Lock()
		delete(testWindowCloseHooks, w)
		testMenuMu.Unlock()
	}
}

// EmitEvent records a typed event without broadcasting it to another fake window.
func (*fakeWindow) EmitEvent(string, ...any) bool { return true }

// TestRuntimeMenuSerializesConcurrentReplacement verifies an older build cannot install after a newer request.
func TestRuntimeMenuSerializesConcurrentReplacement(t *testing.T) {
	win := &fakeWindow{id: 303}
	ctx := context.WithValue(context.Background(), application.WindowKey, application.Window(win))
	release := make(chan struct{})
	entered := make(chan struct{})
	testMenuMu.Lock()
	testWindowMenuBlocker[win] = release
	testWindowMenuEntered[win] = entered
	testMenuMu.Unlock()

	backend := nativeBackend{}
	first := desktop.Menu{Items: []desktop.MenuItem{{ID: "first", Kind: desktop.MenuItemCommand, Label: "First", Enabled: true}}}
	second := desktop.Menu{Items: []desktop.MenuItem{{ID: "second", Kind: desktop.MenuItemCommand, Label: "Second", Enabled: true}}}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _ = backend.ReplaceMenu(ctx, first) }()
	<-entered
	secondStarted := make(chan struct{})
	go func() {
		defer wg.Done()
		close(secondStarted)
		_ = backend.ReplaceMenu(ctx, second)
	}()
	<-secondStarted
	close(release)
	wg.Wait()

	testMenuMu.Lock()
	installed := testWindowMenus[win]
	testMenuMu.Unlock()
	if installed == nil || installed.ItemAt(0) == nil || installed.ItemAt(0).Label() != "Second" {
		t.Fatal("concurrent replacement installed stale menu last")
	}
}

// TestContextMenuReinstallNeverRevivesRemovedCallbacks verifies config identities are not reusable counters.
func TestContextMenuReinstallNeverRevivesRemovedCallbacks(t *testing.T) {
	windowID := uint(404)
	oldToken := &menuConfigToken{}
	state := &menuWindowState{contexts: map[string]*menuContextState{"row": {token: oldToken}}}
	menuWindows.Lock()
	menuWindows.states[windowID] = state
	delete(state.contexts, "row")
	state.contexts["row"] = &menuContextState{token: &menuConfigToken{}}
	isOldCurrent := isMenuConfigCurrentLocked(windowID, state, oldToken, "row")
	delete(menuWindows.states, windowID)
	menuWindows.Unlock()
	if isOldCurrent {
		t.Fatal("reinstalling a removed context menu revived its old callback token")
	}
}

// TestRuntimeMenuTargetsOnlyCallerWindow verifies per-window ownership and close cleanup.
func TestRuntimeMenuTargetsOnlyCallerWindow(t *testing.T) {
	first := &fakeWindow{id: 101}
	second := &fakeWindow{id: 202}
	backend := nativeBackend{}
	menu := desktop.Menu{Items: []desktop.MenuItem{{
		ID:      "file",
		Kind:    desktop.MenuItemSubmenu,
		Label:   "File",
		Enabled: false,
		Hidden:  true,
		Items:   []desktop.MenuItem{{ID: "open", Kind: desktop.MenuItemCommand, Label: "Open", Enabled: true}},
	}}}
	firstCtx := context.WithValue(context.Background(), application.WindowKey, application.Window(first))
	if err := backend.ReplaceMenu(firstCtx, menu); err != nil {
		t.Fatal(err)
	}
	testMenuMu.Lock()
	firstMenu := testWindowMenus[first]
	secondMenu := testWindowMenus[second]
	closeHook := testWindowCloseHooks[first]
	testMenuMu.Unlock()
	if firstMenu == nil || secondMenu != nil {
		t.Fatal("menu replacement escaped caller window")
	}
	top := firstMenu.ItemAt(0)
	if top == nil || top.Enabled() || !top.Hidden() || !top.IsSubmenu() {
		t.Fatal("disabled submenu was not preserved")
	}
	if closeHook == nil {
		t.Fatal("caller-window close cleanup was not registered")
	}
	closeHook(application.NewWindowEvent())
	testMenuMu.Lock()
	clearedMenu := testWindowMenus[first]
	testMenuMu.Unlock()
	if clearedMenu == nil || clearedMenu.ItemAt(0) != nil {
		t.Fatal("caller-window menu was not cleared on close")
	}
}
