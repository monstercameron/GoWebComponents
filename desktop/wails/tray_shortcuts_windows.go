//go:build windows

package wails

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const defaultTrayIconPNG = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAHUlEQVR42mNkYPj/n4ECwESJ5lEDRg0YNWAwGQAAN8cCHXKDgfsAAAAASUVORK5CYII="

type nativeTrayShortcutState struct {
	opMu     sync.Mutex
	mu       sync.Mutex
	owners   map[uint]*nativeOwnerState
	invokeUI func(func())
}

type nativeOwnerState struct {
	window              application.Window
	trays               map[string]*application.SystemTray
	trayMenus           map[string]*application.Menu
	trayGenerations     map[string]uint64
	shortcuts           map[string]string
	shortcutGenerations map[string]uint64
}

func init() {
	application.RegisterEvent[desktop.TrayEvent](desktop.TrayEventTopic)
	application.RegisterEvent[desktop.ShortcutEvent](desktop.ShortcutEventTopic)
}

// newNativeTrayShortcutState constructs isolated caller ownership state.
func newNativeTrayShortcutState() *nativeTrayShortcutState {
	return &nativeTrayShortcutState{
		owners:   make(map[uint]*nativeOwnerState),
		invokeUI: application.InvokeSync,
	}
}

// runTrayUI publishes one complete tray mutation on the Wails UI thread.
func (s *nativeTrayShortcutState) runTrayUI(action func()) {
	s.invokeUI(action)
}

// ConfigureTray creates, updates in-place, positions relative to, or removes a caller-owned tray icon.
func (b *nativeBackend) ConfigureTray(ctx context.Context, req desktop.TrayRequest) error {
	win, err := getCaller(ctx)
	if err != nil {
		return err
	}
	state := b.trayShortcuts
	if state == nil {
		return unavailable("tray state unavailable")
	}
	var app *application.App
	var icon []byte
	var darkIcon []byte
	if req.Action == "upsert" {
		app = application.Get()
		if app == nil || app.SystemTray == nil {
			return unavailable("Wails system tray unavailable")
		}
		icon, err = base64.StdEncoding.DecodeString(defaultTrayIconPNG)
		if err != nil {
			return unavailable("embedded tray icon unavailable")
		}
		if len(req.Icon) != 0 {
			icon = append([]byte(nil), req.Icon...)
		}
		if len(req.DarkModeIcon) != 0 {
			darkIcon = append([]byte(nil), req.DarkModeIcon...)
		}
	}

	state.opMu.Lock()
	defer state.opMu.Unlock()
	state.mu.Lock()
	owner := state.getOwnerLocked(win)
	existing, exists := owner.trays[req.ID]
	existingMenu := owner.trayMenus[req.ID]

	switch req.Action {
	case "show", "hide":
		if !exists {
			state.mu.Unlock()
			return invalid("tray identifier is not registered")
		}
		state.mu.Unlock()
		if req.Action == "show" {
			existing.Show()
		} else {
			existing.Hide()
		}
		return nil
	case "position-window":
		if !exists {
			state.mu.Unlock()
			return invalid("tray identifier is not registered")
		}
		state.mu.Unlock()
		return existing.PositionWindow(win, req.Offset)
	case "remove":
		if !exists {
			state.mu.Unlock()
			return invalid("tray identifier is not registered")
		}
		delete(owner.trays, req.ID)
		delete(owner.trayMenus, req.ID)
		state.mu.Unlock()
		state.destroyTrayResources(existing, existingMenu)
		return nil
	}

	if !exists && len(owner.trays) >= 32 {
		state.mu.Unlock()
		return invalid("caller tray limit reached")
	}
	trayID := req.ID
	generation := owner.trayGenerations[trayID] + 1
	owner.trayGenerations[trayID] = generation
	state.mu.Unlock()

	var menu *application.Menu
	var tray *application.SystemTray
	// Publish the complete tray state in one UI turn so WndProc never observes a
	// partially configured tray. Reuse an existing tray handle in place to avoid
	// taskbar icon flicker on updates.
	state.runTrayUI(func() {
		if req.Menu != nil {
			menu = application.NewMenu()
			buildTrayMenuItems(state, win, menu, trayID, generation, req.Menu.Items)
		}
		if exists && existing != nil {
			tray = existing
			tray.SetIcon(icon)
			if len(darkIcon) != 0 {
				tray.SetDarkModeIcon(darkIcon)
			}
			tray.SetLabel(req.Label)
			tray.SetTooltip(req.Tooltip)
			tray.OnClick(func() { state.emitTrayEvent(win, trayID, generation, "click", "") })
			tray.OnRightClick(func() {
				if state.emitTrayEvent(win, trayID, generation, "right-click", "") && menu != nil {
					tray.OpenMenu()
				}
			})
			tray.OnDoubleClick(func() {
				state.emitTrayEvent(win, trayID, generation, "double-click", "")
			})
			tray.SetMenu(menu)
			if existingMenu != nil {
				existingMenu.Destroy()
			}
			return
		}
		tray = app.SystemTray.New().SetIcon(icon)
		if len(darkIcon) != 0 {
			tray.SetDarkModeIcon(darkIcon)
		}
		if req.Label != "" {
			tray.SetLabel(req.Label)
		}
		tray.SetTooltip(req.Tooltip)
		tray.OnClick(func() { state.emitTrayEvent(win, trayID, generation, "click", "") })
		tray.OnRightClick(func() {
			if state.emitTrayEvent(win, trayID, generation, "right-click", "") && menu != nil {
				tray.OpenMenu()
			}
		})
		tray.OnDoubleClick(func() {
			state.emitTrayEvent(win, trayID, generation, "double-click", "")
		})
		if menu != nil {
			tray.SetMenu(menu)
		}
	})

	state.mu.Lock()
	owner.trays[trayID] = tray
	owner.trayMenus[trayID] = menu
	state.mu.Unlock()
	return nil
}

// destroyTrayResources releases tray and menu native objects on the Wails UI thread.
func (s *nativeTrayShortcutState) destroyTrayResources(tray *application.SystemTray, menu *application.Menu) {
	s.runTrayUI(func() {
		if tray != nil {
			tray.Destroy()
		}
		if menu != nil {
			menu.Destroy()
		}
	})
}

// buildTrayMenuItems converts a validated portable menu to typed tray callbacks.
func buildTrayMenuItems(state *nativeTrayShortcutState, win application.Window, menu *application.Menu, trayID string, generation uint64, items []desktop.MenuItem) {
	for _, def := range items {
		switch def.Kind {
		case desktop.MenuItemSeparator:
			menu.AddSeparator()
		case desktop.MenuItemSubmenu:
			item := application.NewSubMenuItem(def.Label).SetEnabled(def.Enabled)
			menu.Append(application.NewMenuFromItems(item))
			buildTrayMenuItems(state, win, item.GetSubmenu(), trayID, generation, def.Items)
		default:
			var item *application.MenuItem
			switch def.Kind {
			case desktop.MenuItemCheckbox:
				item = menu.AddCheckbox(def.Label, def.Checked)
			case desktop.MenuItemRadio:
				item = menu.AddRadio(def.Label, def.Checked)
			default:
				item = menu.Add(def.Label)
			}
			item.SetEnabled(def.Enabled)
			itemID := def.ID
			item.OnClick(func(*application.Context) {
				state.emitTrayEvent(win, trayID, generation, "menu", itemID)
			})
		}
	}
}

// emitTrayEvent suppresses callbacks from removed or replaced tray generations.
func (s *nativeTrayShortcutState) emitTrayEvent(win application.Window, id string, generation uint64, kind, itemID string) bool {
	s.mu.Lock()
	owner := s.owners[win.ID()]
	isCurrent := false
	if owner != nil {
		_, exists := owner.trays[id]
		isCurrent = exists && owner.trayGenerations[id] == generation
	}
	s.mu.Unlock()
	if isCurrent {
		win.EmitEvent(desktop.TrayEventTopic, desktop.TrayEvent{ID: id, Kind: kind, ItemID: itemID})
	}
	return isCurrent
}

// ConfigureShortcut creates, atomically replaces, or removes a caller-owned global shortcut.
func (b *nativeBackend) ConfigureShortcut(ctx context.Context, req desktop.ShortcutRequest) error {
	win, err := getCaller(ctx)
	if err != nil {
		return err
	}
	state := b.trayShortcuts
	if state == nil {
		return unavailable("shortcut state unavailable")
	}
	app := application.Get()
	if app == nil || app.GlobalShortcut == nil {
		return unavailable("Wails global shortcuts unavailable")
	}
	state.opMu.Lock()
	defer state.opMu.Unlock()
	state.mu.Lock()
	owner := state.getOwnerLocked(win)
	existing, exists := owner.shortcuts[req.ID]
	if req.Action == "remove" {
		if !exists {
			state.mu.Unlock()
			return invalid("shortcut identifier is not registered")
		}
		state.mu.Unlock()
		if err = app.GlobalShortcut.Unregister(existing); err != nil {
			return fmt.Errorf("unregister global shortcut: %w", err)
		}
		state.mu.Lock()
		delete(owner.shortcuts, req.ID)
		state.mu.Unlock()
		return nil
	}
	if exists && existing == req.Accelerator {
		state.mu.Unlock()
		return nil
	}
	if !exists && len(owner.shortcuts) >= 32 {
		state.mu.Unlock()
		return invalid("caller shortcut limit reached")
	}
	shortcutID := req.ID
	generation := owner.shortcutGenerations[shortcutID] + 1
	state.mu.Unlock()
	if err = app.GlobalShortcut.Register(req.Accelerator, func() {
		state.emitShortcutEvent(win, shortcutID, generation)
	}); err != nil {
		return fmt.Errorf("register global shortcut: %w", err)
	}
	if exists {
		if err = app.GlobalShortcut.Unregister(existing); err != nil {
			_ = app.GlobalShortcut.Unregister(req.Accelerator)
			return fmt.Errorf("replace global shortcut: %w", err)
		}
	}
	state.mu.Lock()
	owner.shortcuts[shortcutID] = req.Accelerator
	owner.shortcutGenerations[shortcutID] = generation
	state.mu.Unlock()
	return nil
}

// emitShortcutEvent suppresses callbacks from removed or replaced shortcut generations.
func (s *nativeTrayShortcutState) emitShortcutEvent(win application.Window, id string, generation uint64) {
	s.mu.Lock()
	owner := s.owners[win.ID()]
	isCurrent := false
	if owner != nil {
		_, exists := owner.shortcuts[id]
		isCurrent = exists && owner.shortcutGenerations[id] == generation
	}
	s.mu.Unlock()
	if isCurrent {
		win.EmitEvent(desktop.ShortcutEventTopic, desktop.ShortcutEvent{ID: id})
	}
}

// getOwnerLocked returns caller state and installs its one-time closing cleanup.
func (s *nativeTrayShortcutState) getOwnerLocked(win application.Window) *nativeOwnerState {
	owner := s.owners[win.ID()]
	if owner != nil {
		return owner
	}
	owner = &nativeOwnerState{
		window:              win,
		trays:               make(map[string]*application.SystemTray),
		trayMenus:           make(map[string]*application.Menu),
		trayGenerations:     make(map[string]uint64),
		shortcuts:           make(map[string]string),
		shortcutGenerations: make(map[string]uint64),
	}
	s.owners[win.ID()] = owner
	win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		s.cleanupOwner(win.ID())
	})
	return owner
}

// cleanupOwner releases every operating-system resource owned by one closing window.
func (s *nativeTrayShortcutState) cleanupOwner(windowID uint) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	owner := s.owners[windowID]
	if owner == nil {
		s.mu.Unlock()
		return
	}
	delete(s.owners, windowID)
	s.mu.Unlock()
	for id, tray := range owner.trays {
		s.destroyTrayResources(tray, owner.trayMenus[id])
	}
	app := application.Get()
	if app == nil || app.GlobalShortcut == nil {
		return
	}
	for _, accelerator := range owner.shortcuts {
		if err := app.GlobalShortcut.Unregister(accelerator); err != nil && !errors.Is(err, context.Canceled) {
			// Window teardown is best effort; Wails performs an application-wide final cleanup.
			continue
		}
	}
}
