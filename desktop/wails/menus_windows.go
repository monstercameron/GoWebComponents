//go:build windows

package wails

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type menuWindowState struct {
	parseCancel        func()
	parseMenuToken     *menuConfigToken
	parseInstalledMenu *application.Menu
	parseContexts      map[string]*menuContextState
	parseOperation     sync.Mutex
}

type menuContextState struct {
	parseName  string
	parseToken *menuConfigToken
	parseMenu  *application.ContextMenu
}

type menuConfigToken struct{ parseID uint64 }

var menuConfigSequence atomic.Uint64

// newMenuConfigToken returns a process-unique identity that is never reset on removal.
func newMenuConfigToken() *menuConfigToken {
	return &menuConfigToken{parseID: menuConfigSequence.Add(1)}
}

var menuWindows = struct {
	sync.Mutex
	parseStates map[uint]*menuWindowState
}{parseStates: map[uint]*menuWindowState{}}

// ReplaceMenu replaces the Wails application menu from a validated declarative tree.
func (nativeBackend) ReplaceMenu(parseContext context.Context, parseDefinition desktop.Menu) error {
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return parseErr
	}
	menuWindows.Lock()
	parseState := getMenuWindowStateLocked(parseWindow)
	menuWindows.Unlock()
	parseState.parseOperation.Lock()
	defer parseState.parseOperation.Unlock()
	parseToken := newMenuConfigToken()
	menuWindows.Lock()
	if menuWindows.parseStates[parseWindow.ID()] != parseState {
		menuWindows.Unlock()
		return unavailable("calling window closed")
	}
	parseState.parseMenuToken = parseToken
	parsePrevious := parseState.parseInstalledMenu
	menuWindows.Unlock()
	parseMenu := application.NewMenu()
	buildMenuItems(parseWindow, parseMenu, parseDefinition.Items, parseState, parseToken, "")
	parseWindow.SetMenu(parseMenu)
	menuWindows.Lock()
	if menuWindows.parseStates[parseWindow.ID()] == parseState && parseState.parseMenuToken == parseToken {
		parseState.parseInstalledMenu = parseMenu
	}
	menuWindows.Unlock()
	if parsePrevious != nil {
		parsePrevious.Destroy()
	}
	return nil
}

// InstallContextMenu installs a caller-window scoped Wails context menu.
func (nativeBackend) InstallContextMenu(parseContext context.Context, parseRequest desktop.ContextMenuRequest) error {
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return parseErr
	}
	parseApp := application.Get()
	if parseApp == nil {
		return unavailable("Wails application unavailable")
	}
	menuWindows.Lock()
	parseState := getMenuWindowStateLocked(parseWindow)
	menuWindows.Unlock()
	parseState.parseOperation.Lock()
	defer parseState.parseOperation.Unlock()
	menuWindows.Lock()
	if menuWindows.parseStates[parseWindow.ID()] != parseState {
		menuWindows.Unlock()
		return unavailable("calling window closed")
	}
	parsePrevious := parseState.parseContexts[parseRequest.ID]
	parseToken := newMenuConfigToken()
	parseName := fmt.Sprintf("gwc-%d-%s-%d", parseWindow.ID(), parseRequest.ID, parseToken.parseID)
	parseState.parseContexts[parseRequest.ID] = &menuContextState{parseName: parseName, parseToken: parseToken}
	menuWindows.Unlock()
	if parsePrevious != nil {
		parseApp.ContextMenu.Remove(parsePrevious.parseName)
	}
	parseMenu := application.NewDeferredContextMenu(parseName)
	buildMenuItems(parseWindow, parseMenu.Menu, parseRequest.Menu.Items, parseState, parseToken, parseRequest.ID)
	parseEntry := parseState.parseContexts[parseRequest.ID]
	parseEntry.parseMenu = parseMenu
	if parsePrevious != nil && parsePrevious.parseMenu != nil {
		parsePrevious.parseMenu.Destroy()
		parsePrevious.parseMenu.Menu.Destroy()
	}
	return nil
}

// ShowContextMenu shows only a context menu installed for the invoking window.
func (nativeBackend) ShowContextMenu(parseContext context.Context, parseRequest desktop.ContextMenuShowRequest) error {
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return parseErr
	}
	menuWindows.Lock()
	parseState := menuWindows.parseStates[parseWindow.ID()]
	var parseEntry *menuContextState
	if parseState != nil {
		parseEntry = parseState.parseContexts[parseRequest.ID]
	}
	menuWindows.Unlock()
	if parseEntry == nil {
		return invalid("context menu is not installed for caller window")
	}
	parseState.parseOperation.Lock()
	defer parseState.parseOperation.Unlock()
	menuWindows.Lock()
	isCurrent := menuWindows.parseStates[parseWindow.ID()] == parseState && parseState.parseContexts[parseRequest.ID] == parseEntry
	menuWindows.Unlock()
	if !isCurrent {
		return invalid("context menu is not installed for caller window")
	}
	parseWindow.OpenContextMenu(&application.ContextMenuData{Id: parseEntry.parseName, X: parseRequest.X, Y: parseRequest.Y, Data: parseRequest.Data})
	return nil
}

// RemoveContextMenu removes only a context menu installed for the invoking window.
func (nativeBackend) RemoveContextMenu(parseContext context.Context, parseID string) error {
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return parseErr
	}
	parseApp := application.Get()
	if parseApp == nil {
		return unavailable("Wails application unavailable")
	}
	menuWindows.Lock()
	parseState := menuWindows.parseStates[parseWindow.ID()]
	menuWindows.Unlock()
	if parseState == nil {
		return invalid("context menu is not installed for caller window")
	}
	parseState.parseOperation.Lock()
	defer parseState.parseOperation.Unlock()
	menuWindows.Lock()
	parseEntry := parseState.parseContexts[parseID]
	if menuWindows.parseStates[parseWindow.ID()] != parseState || parseEntry == nil {
		menuWindows.Unlock()
		return invalid("context menu is not installed for caller window")
	}
	delete(parseState.parseContexts, parseID)
	menuWindows.Unlock()
	parseApp.ContextMenu.Remove(parseEntry.parseName)
	if parseEntry.parseMenu != nil {
		parseEntry.parseMenu.Destroy()
		parseEntry.parseMenu.Menu.Destroy()
	}
	return nil
}

// getMenuWindowStateLocked returns lifecycle state while menuWindows is held.
func getMenuWindowStateLocked(parseWindow application.Window) *menuWindowState {
	parseWindowID := parseWindow.ID()
	if parseState := menuWindows.parseStates[parseWindowID]; parseState != nil {
		return parseState
	}
	parseState := &menuWindowState{parseContexts: map[string]*menuContextState{}}
	// Wails runs cancellable close hooks before WindowClosing listeners, so this
	// cleanup runs only after the close was accepted rather than on a vetoed attempt.
	parseState.parseCancel = parseWindow.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		parseState.parseOperation.Lock()
		defer parseState.parseOperation.Unlock()
		menuWindows.Lock()
		if menuWindows.parseStates[parseWindowID] != parseState {
			menuWindows.Unlock()
			return
		}
		delete(menuWindows.parseStates, parseWindowID)
		parseContexts := parseState.parseContexts
		parseInstalled := parseState.parseInstalledMenu
		menuWindows.Unlock()
		if parseApp := application.Get(); parseApp != nil {
			for _, parseEntry := range parseContexts {
				parseApp.ContextMenu.Remove(parseEntry.parseName)
				if parseEntry.parseMenu != nil {
					parseEntry.parseMenu.Destroy()
					parseEntry.parseMenu.Menu.Destroy()
				}
			}
		}
		parseWindow.SetMenu(application.NewMenu())
		if parseInstalled != nil {
			parseInstalled.Destroy()
		}
	})
	menuWindows.parseStates[parseWindowID] = parseState
	return parseState
}

// buildMenuItems converts portable menu nodes without exposing Wails types upstream.
func buildMenuItems(parseWindow application.Window, parseMenu *application.Menu, parseItems []desktop.MenuItem, parseState *menuWindowState, parseToken *menuConfigToken, parseMenuID string) {
	for _, parseDefinition := range parseItems {
		switch parseDefinition.Kind {
		case desktop.MenuItemSeparator:
			parseMenu.AddSeparator()
		case desktop.MenuItemSubmenu:
			parseItem := application.NewSubMenuItem(parseDefinition.Label).SetEnabled(parseDefinition.Enabled)
			configureMenuPresentation(parseItem, parseDefinition)
			parseMenu.Append(application.NewMenuFromItems(parseItem))
			buildMenuItems(parseWindow, parseItem.GetSubmenu(), parseDefinition.Items, parseState, parseToken, parseMenuID)
		case desktop.MenuItemCheckbox:
			parseItem := parseMenu.AddCheckbox(parseDefinition.Label, parseDefinition.Checked).SetEnabled(parseDefinition.Enabled)
			configureMenuItem(parseWindow, parseItem, parseDefinition, parseState, parseToken, parseMenuID)
		case desktop.MenuItemRadio:
			parseItem := parseMenu.AddRadio(parseDefinition.Label, parseDefinition.Checked).SetEnabled(parseDefinition.Enabled)
			configureMenuItem(parseWindow, parseItem, parseDefinition, parseState, parseToken, parseMenuID)
		default:
			parseItem := parseMenu.Add(parseDefinition.Label).SetEnabled(parseDefinition.Enabled)
			configureMenuItem(parseWindow, parseItem, parseDefinition, parseState, parseToken, parseMenuID)
		}
	}
}

// configureMenuItem applies optional presentation and emits only typed selection data.
func configureMenuItem(parseWindow application.Window, parseItem *application.MenuItem, parseDefinition desktop.MenuItem, parseState *menuWindowState, parseToken *menuConfigToken, parseMenuID string) {
	configureMenuPresentation(parseItem, parseDefinition)
	parseID := parseDefinition.ID
	parseKind := parseDefinition.Kind
	parseItem.OnClick(func(parseContext *application.Context) {
		menuWindows.Lock()
		isCurrent := isMenuConfigCurrentLocked(parseWindow.ID(), parseState, parseToken, parseMenuID)
		menuWindows.Unlock()
		if !isCurrent {
			return
		}
		parseSelection := desktop.MenuSelection{ID: parseID, MenuID: parseMenuID, Data: parseContext.ContextMenuData()}
		if parseKind == desktop.MenuItemCheckbox || parseKind == desktop.MenuItemRadio {
			isChecked := parseContext.IsChecked()
			parseSelection.Checked = &isChecked
		}
		parseWindow.EmitEvent(desktop.MenuSelectedTopic, parseSelection)
	})
}

// isMenuConfigCurrentLocked checks allocation identity while menuWindows is held.
func isMenuConfigCurrentLocked(parseWindowID uint, parseState *menuWindowState, parseToken *menuConfigToken, parseMenuID string) bool {
	if menuWindows.parseStates[parseWindowID] != parseState {
		return false
	}
	if parseMenuID == "" {
		return parseState.parseMenuToken == parseToken
	}
	parseContext := parseState.parseContexts[parseMenuID]
	return parseContext != nil && parseContext.parseToken == parseToken
}

// configureMenuPresentation applies Wails-supported declarative presentation fields.
func configureMenuPresentation(parseItem *application.MenuItem, parseDefinition desktop.MenuItem) {
	if parseDefinition.Shortcut != "" {
		parseItem.SetAccelerator(parseDefinition.Shortcut)
	}
	parseItem.SetHidden(parseDefinition.Hidden)
	if len(parseDefinition.Icon) != 0 {
		parseItem.SetBitmap(append([]byte(nil), parseDefinition.Icon...))
	}
}
