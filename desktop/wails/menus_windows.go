//go:build windows

package wails

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type menuWindowState struct {
	cancel        func()
	menuToken     *menuConfigToken
	installedMenu *application.Menu
	contexts      map[string]*menuContextState
	opMu          sync.Mutex
}

type menuContextState struct {
	name  string
	token *menuConfigToken
	menu  *application.ContextMenu
}

type menuConfigToken struct{ id uint64 }

var menuConfigSequence atomic.Uint64

// newMenuConfigToken returns a process-unique identity that is never reset on removal.
func newMenuConfigToken() *menuConfigToken {
	return &menuConfigToken{id: menuConfigSequence.Add(1)}
}

var menuWindows = struct {
	sync.Mutex
	states map[uint]*menuWindowState
}{states: map[uint]*menuWindowState{}}

// ReplaceMenu replaces the Wails application menu from a validated declarative tree.
func (nativeBackend) ReplaceMenu(ctx context.Context, definition desktop.Menu) error {
	win, err := getCaller(ctx)
	if err != nil {
		return err
	}
	menuWindows.Lock()
	state := getMenuWindowStateLocked(win)
	menuWindows.Unlock()
	state.opMu.Lock()
	defer state.opMu.Unlock()

	token := newMenuConfigToken()
	menuWindows.Lock()
	if menuWindows.states[win.ID()] != state {
		menuWindows.Unlock()
		return unavailable("calling window closed")
	}
	state.menuToken = token
	previous := state.installedMenu
	menuWindows.Unlock()

	menu := application.NewMenu()
	buildMenuItems(win, menu, definition.Items, state, token, "")
	win.SetMenu(menu)

	menuWindows.Lock()
	if menuWindows.states[win.ID()] == state && state.menuToken == token {
		state.installedMenu = menu
	}
	menuWindows.Unlock()
	if previous != nil {
		previous.Destroy()
	}
	return nil
}

// InstallContextMenu installs a caller-window scoped Wails context menu.
func (nativeBackend) InstallContextMenu(ctx context.Context, req desktop.ContextMenuRequest) error {
	win, err := getCaller(ctx)
	if err != nil {
		return err
	}
	app := application.Get()
	if app == nil {
		return unavailable("Wails application unavailable")
	}
	menuWindows.Lock()
	state := getMenuWindowStateLocked(win)
	menuWindows.Unlock()
	state.opMu.Lock()
	defer state.opMu.Unlock()

	menuWindows.Lock()
	if menuWindows.states[win.ID()] != state {
		menuWindows.Unlock()
		return unavailable("calling window closed")
	}
	previous := state.contexts[req.ID]
	token := newMenuConfigToken()
	name := fmt.Sprintf("gwc-%d-%s-%d", win.ID(), req.ID, token.id)
	entry := &menuContextState{name: name, token: token}
	state.contexts[req.ID] = entry
	menuWindows.Unlock()

	if previous != nil {
		app.ContextMenu.Remove(previous.name)
	}
	menu := application.NewDeferredContextMenu(name)
	buildMenuItems(win, menu.Menu, req.Menu.Items, state, token, req.ID)
	entry.menu = menu
	if previous != nil && previous.menu != nil {
		previous.menu.Destroy()
		previous.menu.Menu.Destroy()
	}
	return nil
}

// ShowContextMenu shows only a context menu installed for the invoking window.
func (nativeBackend) ShowContextMenu(ctx context.Context, req desktop.ContextMenuShowRequest) error {
	win, err := getCaller(ctx)
	if err != nil {
		return err
	}
	menuWindows.Lock()
	state := menuWindows.states[win.ID()]
	var entry *menuContextState
	if state != nil {
		entry = state.contexts[req.ID]
	}
	menuWindows.Unlock()
	if entry == nil {
		return invalid("context menu is not installed for caller window")
	}
	state.opMu.Lock()
	defer state.opMu.Unlock()
	menuWindows.Lock()
	isCurrent := menuWindows.states[win.ID()] == state && state.contexts[req.ID] == entry
	menuWindows.Unlock()
	if !isCurrent {
		return invalid("context menu is not installed for caller window")
	}
	win.OpenContextMenu(&application.ContextMenuData{Id: entry.name, X: req.X, Y: req.Y, Data: req.Data})
	return nil
}

// RemoveContextMenu removes only a context menu installed for the invoking window.
func (nativeBackend) RemoveContextMenu(ctx context.Context, id string) error {
	win, err := getCaller(ctx)
	if err != nil {
		return err
	}
	app := application.Get()
	if app == nil {
		return unavailable("Wails application unavailable")
	}
	menuWindows.Lock()
	state := menuWindows.states[win.ID()]
	menuWindows.Unlock()
	if state == nil {
		return invalid("context menu is not installed for caller window")
	}
	state.opMu.Lock()
	defer state.opMu.Unlock()
	menuWindows.Lock()
	entry := state.contexts[id]
	if menuWindows.states[win.ID()] != state || entry == nil {
		menuWindows.Unlock()
		return invalid("context menu is not installed for caller window")
	}
	delete(state.contexts, id)
	menuWindows.Unlock()
	app.ContextMenu.Remove(entry.name)
	if entry.menu != nil {
		entry.menu.Destroy()
		entry.menu.Menu.Destroy()
	}
	return nil
}

// getMenuWindowStateLocked returns lifecycle state while menuWindows is held.
func getMenuWindowStateLocked(win application.Window) *menuWindowState {
	windowID := win.ID()
	if state := menuWindows.states[windowID]; state != nil {
		return state
	}
	state := &menuWindowState{contexts: map[string]*menuContextState{}}
	// Wails runs cancellable close hooks before WindowClosing listeners, so this
	// cleanup runs only after the close was accepted rather than on a vetoed attempt.
	state.cancel = win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		state.opMu.Lock()
		defer state.opMu.Unlock()
		menuWindows.Lock()
		if menuWindows.states[windowID] != state {
			menuWindows.Unlock()
			return
		}
		delete(menuWindows.states, windowID)
		contexts := state.contexts
		installed := state.installedMenu
		menuWindows.Unlock()
		if app := application.Get(); app != nil {
			for _, entry := range contexts {
				app.ContextMenu.Remove(entry.name)
				if entry.menu != nil {
					entry.menu.Destroy()
					entry.menu.Menu.Destroy()
				}
			}
		}
		win.SetMenu(application.NewMenu())
		if installed != nil {
			installed.Destroy()
		}
	})
	menuWindows.states[windowID] = state
	return state
}

// buildMenuItems converts portable menu nodes without exposing Wails types upstream.
func buildMenuItems(win application.Window, menu *application.Menu, items []desktop.MenuItem, state *menuWindowState, token *menuConfigToken, menuID string) {
	for _, def := range items {
		switch def.Kind {
		case desktop.MenuItemSeparator:
			menu.AddSeparator()
		case desktop.MenuItemSubmenu:
			item := application.NewSubMenuItem(def.Label).SetEnabled(def.Enabled)
			applyMenuPresentation(item, def)
			menu.Append(application.NewMenuFromItems(item))
			buildMenuItems(win, item.GetSubmenu(), def.Items, state, token, menuID)
		case desktop.MenuItemCheckbox:
			item := menu.AddCheckbox(def.Label, def.Checked).SetEnabled(def.Enabled)
			configureMenuItem(win, item, def, state, token, menuID)
		case desktop.MenuItemRadio:
			item := menu.AddRadio(def.Label, def.Checked).SetEnabled(def.Enabled)
			configureMenuItem(win, item, def, state, token, menuID)
		default:
			item := menu.Add(def.Label).SetEnabled(def.Enabled)
			configureMenuItem(win, item, def, state, token, menuID)
		}
	}
}

// configureMenuItem applies optional presentation and emits only typed selection data.
func configureMenuItem(win application.Window, item *application.MenuItem, def desktop.MenuItem, state *menuWindowState, token *menuConfigToken, menuID string) {
	applyMenuPresentation(item, def)
	itemID := def.ID
	itemKind := def.Kind
	item.OnClick(func(menuCtx *application.Context) {
		menuWindows.Lock()
		isCurrent := isMenuConfigCurrentLocked(win.ID(), state, token, menuID)
		menuWindows.Unlock()
		if !isCurrent {
			return
		}
		selection := desktop.MenuSelection{ID: itemID, MenuID: menuID, Data: menuCtx.ContextMenuData()}
		if itemKind == desktop.MenuItemCheckbox || itemKind == desktop.MenuItemRadio {
			isChecked := menuCtx.IsChecked()
			selection.Checked = &isChecked
		}
		win.EmitEvent(desktop.MenuSelectedTopic, selection)
	})
}

// isMenuConfigCurrentLocked checks allocation identity while menuWindows is held.
func isMenuConfigCurrentLocked(windowID uint, state *menuWindowState, token *menuConfigToken, menuID string) bool {
	if menuWindows.states[windowID] != state {
		return false
	}
	if menuID == "" {
		return state.menuToken == token
	}
	entry := state.contexts[menuID]
	return entry != nil && entry.token == token
}

// applyMenuPresentation applies Wails-supported declarative presentation fields.
func applyMenuPresentation(item *application.MenuItem, def desktop.MenuItem) {
	if def.Shortcut != "" {
		item.SetAccelerator(def.Shortcut)
	}
	item.SetHidden(def.Hidden)
	if len(def.Icon) != 0 {
		item.SetBitmap(append([]byte(nil), def.Icon...))
	}
}
