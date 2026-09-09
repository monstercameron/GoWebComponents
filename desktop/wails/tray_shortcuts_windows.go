//go:build windows

package wails

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const parseTrayIconPNG = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAHUlEQVR42mNkYPj/n4ECwESJ5lEDRg0YNWAwGQAAN8cCHXKDgfsAAAAASUVORK5CYII="

type nativeTrayShortcutState struct {
	parseOperationMutex sync.Mutex
	parseMutex          sync.Mutex
	parseOwners         map[uint]*nativeOwnerState
	invokeUI            func(func())
}

type nativeOwnerState struct {
	parseWindow              application.Window
	parseTrays               map[string]*application.SystemTray
	parseTrayMenus           map[string]*application.Menu
	parseTrayGenerations     map[string]uint64
	parseShortcuts           map[string]string
	parseShortcutGenerations map[string]uint64
}

func init() {
	application.RegisterEvent[desktop.TrayEvent](desktop.TrayEventTopic)
	application.RegisterEvent[desktop.ShortcutEvent](desktop.ShortcutEventTopic)
}

// newNativeTrayShortcutState constructs isolated caller ownership state.
func newNativeTrayShortcutState() *nativeTrayShortcutState {
	return &nativeTrayShortcutState{parseOwners: make(map[uint]*nativeOwnerState), invokeUI: application.InvokeSync}
}

// runTrayUI publishes one complete tray mutation on the Wails UI thread.
func (parseState *nativeTrayShortcutState) runTrayUI(parseAction func()) {
	parseState.invokeUI(parseAction)
}

// ConfigureTray creates, atomically replaces, or removes a caller-owned tray icon.
func (parseBackend *nativeBackend) ConfigureTray(parseContext context.Context, parseRequest desktop.TrayRequest) error {
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return parseErr
	}
	parseState := parseBackend.parseTrayShortcuts
	if parseState == nil {
		return unavailable("tray state unavailable")
	}
	var parseApp *application.App
	var parseIcon []byte
	if parseRequest.Action == "upsert" {
		parseApp = application.Get()
		if parseApp == nil || parseApp.SystemTray == nil {
			return unavailable("Wails system tray unavailable")
		}
		parseIcon, parseErr = base64.StdEncoding.DecodeString(parseTrayIconPNG)
		if parseErr != nil {
			return unavailable("embedded tray icon unavailable")
		}
		if len(parseRequest.Icon) != 0 {
			parseIcon = append([]byte(nil), parseRequest.Icon...)
		}
	}
	parseState.parseOperationMutex.Lock()
	defer parseState.parseOperationMutex.Unlock()
	parseState.parseMutex.Lock()
	parseOwner := parseState.getOwnerLocked(parseWindow)
	parseExisting, parseExists := parseOwner.parseTrays[parseRequest.ID]
	parseExistingMenu := parseOwner.parseTrayMenus[parseRequest.ID]
	if parseRequest.Action == "show" || parseRequest.Action == "hide" {
		if !parseExists {
			parseState.parseMutex.Unlock()
			return invalid("tray identifier is not registered")
		}
		parseState.parseMutex.Unlock()
		if parseRequest.Action == "show" {
			parseExisting.Show()
		} else {
			parseExisting.Hide()
		}
		return nil
	}
	if parseRequest.Action == "remove" {
		if !parseExists {
			parseState.parseMutex.Unlock()
			return invalid("tray identifier is not registered")
		}
		delete(parseOwner.parseTrays, parseRequest.ID)
		delete(parseOwner.parseTrayMenus, parseRequest.ID)
		parseState.parseMutex.Unlock()
		destroyTrayResources(parseExisting, parseExistingMenu)
		return nil
	}
	if !parseExists && len(parseOwner.parseTrays) >= 32 {
		parseState.parseMutex.Unlock()
		return invalid("caller tray limit reached")
	}
	parseID := parseRequest.ID
	parseGeneration := parseOwner.parseTrayGenerations[parseID] + 1
	parseOwner.parseTrayGenerations[parseID] = parseGeneration
	parseState.parseMutex.Unlock()
	var parseMenu *application.Menu
	var parseTray *application.SystemTray
	// The Windows tray becomes visible during New. Publish its complete callback
	// state in one UI turn so WndProc cannot observe a partially configured tray.
	parseState.runTrayUI(func() {
		if parseRequest.Menu != nil {
			parseMenu = application.NewMenu()
			buildTrayMenuItems(parseState, parseWindow, parseMenu, parseID, parseGeneration, parseRequest.Menu.Items)
		}
		parseTray = parseApp.SystemTray.New().SetIcon(parseIcon)
		parseTray.SetTooltip(parseRequest.Tooltip)
		parseTray.OnClick(func() { parseState.emitTrayEvent(parseWindow, parseID, parseGeneration, "click", "") })
		parseTray.OnRightClick(func() {
			if parseState.emitTrayEvent(parseWindow, parseID, parseGeneration, "right-click", "") && parseMenu != nil {
				parseTray.OpenMenu()
			}
		})
		parseTray.OnDoubleClick(func() {
			parseState.emitTrayEvent(parseWindow, parseID, parseGeneration, "double-click", "")
		})
		if parseMenu != nil {
			parseTray.SetMenu(parseMenu)
		}
	})
	parseState.parseMutex.Lock()
	parseOwner.parseTrays[parseID] = parseTray
	parseOwner.parseTrayMenus[parseID] = parseMenu
	parseState.parseMutex.Unlock()
	if parseExists {
		destroyTrayResources(parseExisting, parseExistingMenu)
	}
	return nil
}

// destroyTrayResources releases tray and menu native objects on the Wails UI thread.
func destroyTrayResources(parseTray *application.SystemTray, parseMenu *application.Menu) {
	application.InvokeSync(func() {
		if parseTray != nil {
			parseTray.Destroy()
		}
		if parseMenu != nil {
			parseMenu.Destroy()
		}
	})
}

// buildTrayMenuItems converts a validated portable menu to typed tray callbacks.
func buildTrayMenuItems(parseState *nativeTrayShortcutState, parseWindow application.Window, parseMenu *application.Menu, parseTrayID string, parseGeneration uint64, parseItems []desktop.MenuItem) {
	for _, parseDefinition := range parseItems {
		switch parseDefinition.Kind {
		case desktop.MenuItemSeparator:
			parseMenu.AddSeparator()
		case desktop.MenuItemSubmenu:
			parseItem := application.NewSubMenuItem(parseDefinition.Label).SetEnabled(parseDefinition.Enabled)
			parseMenu.Append(application.NewMenuFromItems(parseItem))
			buildTrayMenuItems(parseState, parseWindow, parseItem.GetSubmenu(), parseTrayID, parseGeneration, parseDefinition.Items)
		default:
			var parseItem *application.MenuItem
			if parseDefinition.Kind == desktop.MenuItemCheckbox {
				parseItem = parseMenu.AddCheckbox(parseDefinition.Label, parseDefinition.Checked)
			} else if parseDefinition.Kind == desktop.MenuItemRadio {
				parseItem = parseMenu.AddRadio(parseDefinition.Label, parseDefinition.Checked)
			} else {
				parseItem = parseMenu.Add(parseDefinition.Label)
			}
			parseItem.SetEnabled(parseDefinition.Enabled)
			parseItemID := parseDefinition.ID
			parseItem.OnClick(func(*application.Context) {
				parseState.emitTrayEvent(parseWindow, parseTrayID, parseGeneration, "menu", parseItemID)
			})
		}
	}
}

// emitTrayEvent suppresses callbacks from removed or replaced tray generations.
func (parseState *nativeTrayShortcutState) emitTrayEvent(parseWindow application.Window, parseID string, parseGeneration uint64, parseKind, parseItemID string) bool {
	parseState.parseMutex.Lock()
	parseOwner := parseState.parseOwners[parseWindow.ID()]
	isCurrent := false
	if parseOwner != nil {
		_, parseExists := parseOwner.parseTrays[parseID]
		isCurrent = parseExists && parseOwner.parseTrayGenerations[parseID] == parseGeneration
	}
	parseState.parseMutex.Unlock()
	if isCurrent {
		parseWindow.EmitEvent(desktop.TrayEventTopic, desktop.TrayEvent{ID: parseID, Kind: parseKind, ItemID: parseItemID})
	}
	return isCurrent
}

// ConfigureShortcut creates, atomically replaces, or removes a caller-owned global shortcut.
func (parseBackend *nativeBackend) ConfigureShortcut(parseContext context.Context, parseRequest desktop.ShortcutRequest) error {
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return parseErr
	}
	parseState := parseBackend.parseTrayShortcuts
	if parseState == nil {
		return unavailable("shortcut state unavailable")
	}
	parseApp := application.Get()
	if parseApp == nil || parseApp.GlobalShortcut == nil {
		return unavailable("Wails global shortcuts unavailable")
	}
	parseState.parseOperationMutex.Lock()
	defer parseState.parseOperationMutex.Unlock()
	parseState.parseMutex.Lock()
	parseOwner := parseState.getOwnerLocked(parseWindow)
	parseExisting, parseExists := parseOwner.parseShortcuts[parseRequest.ID]
	if parseRequest.Action == "remove" {
		if !parseExists {
			parseState.parseMutex.Unlock()
			return invalid("shortcut identifier is not registered")
		}
		parseState.parseMutex.Unlock()
		if parseErr = parseApp.GlobalShortcut.Unregister(parseExisting); parseErr != nil {
			return fmt.Errorf("unregister global shortcut: %w", parseErr)
		}
		parseState.parseMutex.Lock()
		delete(parseOwner.parseShortcuts, parseRequest.ID)
		parseState.parseMutex.Unlock()
		return nil
	}
	if parseExists && parseExisting == parseRequest.Accelerator {
		parseState.parseMutex.Unlock()
		return nil
	}
	if !parseExists && len(parseOwner.parseShortcuts) >= 32 {
		parseState.parseMutex.Unlock()
		return invalid("caller shortcut limit reached")
	}
	parseID := parseRequest.ID
	parseGeneration := parseOwner.parseShortcutGenerations[parseID] + 1
	parseState.parseMutex.Unlock()
	if parseErr = parseApp.GlobalShortcut.Register(parseRequest.Accelerator, func() {
		parseState.emitShortcutEvent(parseWindow, parseID, parseGeneration)
	}); parseErr != nil {
		return fmt.Errorf("register global shortcut: %w", parseErr)
	}
	if parseExists {
		if parseErr = parseApp.GlobalShortcut.Unregister(parseExisting); parseErr != nil {
			_ = parseApp.GlobalShortcut.Unregister(parseRequest.Accelerator)
			return fmt.Errorf("replace global shortcut: %w", parseErr)
		}
	}
	parseState.parseMutex.Lock()
	parseOwner.parseShortcuts[parseID] = parseRequest.Accelerator
	parseOwner.parseShortcutGenerations[parseID] = parseGeneration
	parseState.parseMutex.Unlock()
	return nil
}

// emitShortcutEvent suppresses callbacks from removed or replaced shortcut generations.
func (parseState *nativeTrayShortcutState) emitShortcutEvent(parseWindow application.Window, parseID string, parseGeneration uint64) {
	parseState.parseMutex.Lock()
	parseOwner := parseState.parseOwners[parseWindow.ID()]
	isCurrent := false
	if parseOwner != nil {
		_, parseExists := parseOwner.parseShortcuts[parseID]
		isCurrent = parseExists && parseOwner.parseShortcutGenerations[parseID] == parseGeneration
	}
	parseState.parseMutex.Unlock()
	if isCurrent {
		parseWindow.EmitEvent(desktop.ShortcutEventTopic, desktop.ShortcutEvent{ID: parseID})
	}
}

// getOwnerLocked returns caller state and installs its one-time closing cleanup.
func (parseState *nativeTrayShortcutState) getOwnerLocked(parseWindow application.Window) *nativeOwnerState {
	parseOwner := parseState.parseOwners[parseWindow.ID()]
	if parseOwner != nil {
		return parseOwner
	}
	parseOwner = &nativeOwnerState{parseWindow: parseWindow, parseTrays: make(map[string]*application.SystemTray), parseTrayMenus: make(map[string]*application.Menu), parseTrayGenerations: make(map[string]uint64), parseShortcuts: make(map[string]string), parseShortcutGenerations: make(map[string]uint64)}
	parseState.parseOwners[parseWindow.ID()] = parseOwner
	parseWindow.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		parseState.cleanupOwner(parseWindow.ID())
	})
	return parseOwner
}

// cleanupOwner releases every operating-system resource owned by one closing window.
func (parseState *nativeTrayShortcutState) cleanupOwner(parseWindowID uint) {
	parseState.parseOperationMutex.Lock()
	defer parseState.parseOperationMutex.Unlock()
	parseState.parseMutex.Lock()
	parseOwner := parseState.parseOwners[parseWindowID]
	if parseOwner == nil {
		parseState.parseMutex.Unlock()
		return
	}
	delete(parseState.parseOwners, parseWindowID)
	parseState.parseMutex.Unlock()
	for parseID, parseTray := range parseOwner.parseTrays {
		destroyTrayResources(parseTray, parseOwner.parseTrayMenus[parseID])
	}
	parseApp := application.Get()
	if parseApp == nil || parseApp.GlobalShortcut == nil {
		return
	}
	for _, parseAccelerator := range parseOwner.parseShortcuts {
		if parseErr := parseApp.GlobalShortcut.Unregister(parseAccelerator); parseErr != nil && !errors.Is(parseErr, context.Canceled) {
			// Window teardown is best effort; Wails performs an application-wide final cleanup.
			continue
		}
	}
}
