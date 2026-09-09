package desktop

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// RuntimeMenus identifies caller-owned runtime application menus.
const RuntimeMenus Feature = "runtime-menus"

// MenuReplaceMethod replaces the caller-owned native application menu.
const MenuReplaceMethod = "desktop.menu.replace"

// ContextMenuInstallMethod installs or replaces one caller-window context menu.
const ContextMenuInstallMethod = "desktop.menu.context.install"

// ContextMenuShowMethod shows one installed caller-window context menu.
const ContextMenuShowMethod = "desktop.menu.context.show"

// ContextMenuRemoveMethod removes one installed caller-window context menu.
const ContextMenuRemoveMethod = "desktop.menu.context.remove"

// MenuSelectedTopic carries typed native menu selections.
const MenuSelectedTopic = "desktop.menu.selected"

// MenuItemKind identifies one declarative menu node kind.
type MenuItemKind string

const (
	// MenuItemCommand identifies a selectable command.
	MenuItemCommand MenuItemKind = "command"
	// MenuItemCheckbox identifies an independently checked command.
	MenuItemCheckbox MenuItemKind = "checkbox"
	// MenuItemRadio identifies a radio command grouped with adjacent radio siblings.
	MenuItemRadio MenuItemKind = "radio"
	// MenuItemSubmenu identifies a labelled submenu.
	MenuItemSubmenu MenuItemKind = "submenu"
	// MenuItemSeparator identifies a visual separator.
	MenuItemSeparator MenuItemKind = "separator"
)

// Menu describes a complete caller-owned application menu.
type Menu struct {
	Items []MenuItem `json:"items"`
}

// MenuItem describes one bounded declarative native menu node.
type MenuItem struct {
	ID       string       `json:"id,omitempty"`
	Kind     MenuItemKind `json:"kind"`
	Label    string       `json:"label,omitempty"`
	Enabled  bool         `json:"enabled"`
	Hidden   bool         `json:"hidden,omitempty"`
	Checked  bool         `json:"checked,omitempty"`
	Shortcut string       `json:"shortcut,omitempty"`
	Icon     []byte       `json:"icon,omitempty"`
	Items    []MenuItem   `json:"items,omitempty"`
}

// MenuSelection reports the stable ID selected by the operator.
type MenuSelection struct {
	ID      string `json:"id"`
	MenuID  string `json:"menuId,omitempty"`
	Data    string `json:"data,omitempty"`
	Checked *bool  `json:"checked,omitempty"`
}

// ContextMenuRequest identifies a caller-window context menu and its tree.
type ContextMenuRequest struct {
	ID   string `json:"id"`
	Menu Menu   `json:"menu"`
}

// ContextMenuShowRequest positions an installed caller-window context menu.
type ContextMenuShowRequest struct {
	ID   string `json:"id"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Data string `json:"data,omitempty"`
}

// ContextMenuBackend is the optional native adapter contract for context menus.
type ContextMenuBackend interface {
	InstallContextMenu(context.Context, ContextMenuRequest) error
	ShowContextMenu(context.Context, ContextMenuShowRequest) error
	RemoveContextMenu(context.Context, string) error
}

// RuntimeMenuBackend is the optional native adapter contract for runtime menus.
type RuntimeMenuBackend interface {
	ReplaceMenu(context.Context, Menu) error
}

// InstallContextMenu validates and installs one caller-window context menu.
func (parseHost *NativeHost) InstallContextMenu(parseContext context.Context, parseRequest ContextMenuRequest) error {
	parseBackend, parseErr := parseHost.getContextMenuBackend(parseContext, ContextMenuInstallMethod)
	if parseErr != nil {
		return parseErr
	}
	if parseErr = validateContextMenuID(parseRequest.ID); parseErr != nil {
		return getError(ContextMenuInstallMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr = validateMenu(parseRequest.Menu); parseErr != nil {
		return getError(ContextMenuInstallMethod, interop.CodeInvalid, parseErr)
	}
	parseRequest.Menu.Items = cloneMenuItems(parseRequest.Menu.Items)
	return parseBackend.InstallContextMenu(parseContext, parseRequest)
}

// ShowContextMenu validates and shows one installed caller-window context menu.
func (parseHost *NativeHost) ShowContextMenu(parseContext context.Context, parseRequest ContextMenuShowRequest) error {
	parseBackend, parseErr := parseHost.getContextMenuBackend(parseContext, ContextMenuShowMethod)
	if parseErr != nil {
		return parseErr
	}
	if parseErr = validateContextMenuShow(parseRequest); parseErr != nil {
		return getError(ContextMenuShowMethod, interop.CodeInvalid, parseErr)
	}
	return parseBackend.ShowContextMenu(parseContext, parseRequest)
}

// RemoveContextMenu validates and removes one installed caller-window context menu.
func (parseHost *NativeHost) RemoveContextMenu(parseContext context.Context, parseID string) error {
	parseBackend, parseErr := parseHost.getContextMenuBackend(parseContext, ContextMenuRemoveMethod)
	if parseErr != nil {
		return parseErr
	}
	if parseErr = validateContextMenuID(parseID); parseErr != nil {
		return getError(ContextMenuRemoveMethod, interop.CodeInvalid, parseErr)
	}
	return parseBackend.RemoveContextMenu(parseContext, parseID)
}

// getContextMenuBackend applies policy, optional-interface, and context checks.
func (parseHost *NativeHost) getContextMenuBackend(parseContext context.Context, parseMethod string) (ContextMenuBackend, error) {
	if parseErr := parseHost.Require(RuntimeMenus); parseErr != nil {
		return nil, parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(ContextMenuBackend)
	if !parseOK {
		return nil, getError(parseMethod, interop.CodeUnavailable, errors.New("context menus are unavailable"))
	}
	if parseContext == nil {
		return nil, getError(parseMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return nil, getContextError(parseMethod, parseErr)
	}
	return parseBackend, nil
}

// ReplaceMenu replaces the caller-owned application menu with a declarative tree.
func (parseHost *NativeHost) ReplaceMenu(parseContext context.Context, parseMenu Menu) error {
	if parseErr := parseHost.Require(RuntimeMenus); parseErr != nil {
		return parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(RuntimeMenuBackend)
	if !parseOK {
		return getError(MenuReplaceMethod, interop.CodeUnavailable, errors.New("runtime menus are unavailable"))
	}
	if parseContext == nil {
		return getError(MenuReplaceMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return getContextError(MenuReplaceMethod, parseErr)
	}
	if parseErr := validateMenu(parseMenu); parseErr != nil {
		return getError(MenuReplaceMethod, interop.CodeInvalid, parseErr)
	}
	parseCopy := Menu{Items: cloneMenuItems(parseMenu.Items)}
	parseErr := parseBackend.ReplaceMenu(parseContext, parseCopy)
	if parseContext.Err() != nil {
		return getContextError(MenuReplaceMethod, parseContext.Err())
	}
	return parseErr
}

// ReplaceMenu replaces the connected host's caller-owned native application menu.
func (parseClient Client) ReplaceMenu(parseContext context.Context, parseMenu Menu) error {
	if parseErr := validateMenu(parseMenu); parseErr != nil {
		return getError(MenuReplaceMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(RuntimeMenus); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, MenuReplaceMethod, parseMenu)
	return parseErr
}

// InstallContextMenu installs or replaces one caller-window context menu.
func (parseClient Client) InstallContextMenu(parseContext context.Context, parseRequest ContextMenuRequest) error {
	if parseErr := validateContextMenuID(parseRequest.ID); parseErr != nil {
		return getError(ContextMenuInstallMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := validateMenu(parseRequest.Menu); parseErr != nil {
		return getError(ContextMenuInstallMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(RuntimeMenus); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, ContextMenuInstallMethod, parseRequest)
	return parseErr
}

// ShowContextMenu shows one installed caller-window context menu at client coordinates.
func (parseClient Client) ShowContextMenu(parseContext context.Context, parseRequest ContextMenuShowRequest) error {
	if parseErr := validateContextMenuShow(parseRequest); parseErr != nil {
		return getError(ContextMenuShowMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(RuntimeMenus); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, ContextMenuShowMethod, parseRequest)
	return parseErr
}

// RemoveContextMenu removes one installed caller-window context menu.
func (parseClient Client) RemoveContextMenu(parseContext context.Context, parseID string) error {
	if parseErr := validateContextMenuID(parseID); parseErr != nil {
		return getError(ContextMenuRemoveMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(RuntimeMenus); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, ContextMenuRemoveMethod, struct {
		ID string `json:"id"`
	}{ID: parseID})
	return parseErr
}

// SubscribeMenuSelections observes typed selections for the current runtime menu.
func (parseClient Client) SubscribeMenuSelections(parseContext context.Context, parseHandler func(MenuSelection, error)) (func(), error) {
	if parseHandler == nil {
		return nil, getError(MenuSelectedTopic, interop.CodeInvalid, errors.New("menu selection handler required"))
	}
	if parseErr := parseClient.Require(RuntimeMenus); parseErr != nil {
		return nil, parseErr
	}
	return Subscribe(parseContext, parseClient, MenuSelectedTopic, func(parseSelection MenuSelection, parseErr error) {
		if parseErr == nil {
			parseErr = validateMenuSelection(parseSelection)
			if parseErr != nil {
				parseErr = getError(MenuSelectedTopic, interop.CodeDecode, parseErr)
			}
		}
		parseHandler(parseSelection, parseErr)
	})
}

// validateMenuSelection rejects malformed native event payloads before application code handles them.
func validateMenuSelection(parseSelection MenuSelection) error {
	if parseSelection.ID == "" || !validateMenuText(parseSelection.ID, 128) {
		return errors.New("menu selection requires a valid ID")
	}
	if (parseSelection.MenuID != "" && !validateMenuText(parseSelection.MenuID, 128)) || !validateMenuText(parseSelection.Data, 4096) {
		return errors.New("menu selection contains invalid context data")
	}
	return nil
}

// validateContextMenuID checks a stable caller-scoped context menu identifier.
func validateContextMenuID(parseID string) error {
	if parseID == "" || !validateMenuText(parseID, 128) {
		return errors.New("context menu requires a valid ID")
	}
	return nil
}

// validateContextMenuShow bounds context-menu coordinates and opaque returned data.
func validateContextMenuShow(parseRequest ContextMenuShowRequest) error {
	if parseErr := validateContextMenuID(parseRequest.ID); parseErr != nil {
		return parseErr
	}
	if parseRequest.X < -100000 || parseRequest.X > 100000 || parseRequest.Y < -100000 || parseRequest.Y > 100000 || !validateMenuText(parseRequest.Data, 4096) {
		return errors.New("invalid context menu position or data")
	}
	return nil
}

// validateMenu rejects ambiguous or oversized trees before native work.
func validateMenu(parseMenu Menu) error {
	if len(parseMenu.Items) == 0 || len(parseMenu.Items) > 64 {
		return errors.New("menu must contain 1 to 64 top-level items")
	}
	parseIDs := map[string]bool{}
	parseCount := 0
	parseIconBytes := 0
	var validateItems func([]MenuItem, int) error
	validateItems = func(parseItems []MenuItem, parseDepth int) error {
		if parseDepth > 8 {
			return errors.New("menu nesting exceeds 8 levels")
		}
		parseRadioChecked := 0
		for _, parseItem := range parseItems {
			if parseItem.Kind != MenuItemRadio {
				parseRadioChecked = 0
			} else if parseItem.Checked {
				parseRadioChecked++
				if parseRadioChecked > 1 {
					return errors.New("adjacent radio group has multiple checked items")
				}
			}
			parseCount++
			if parseCount > 512 {
				return errors.New("menu exceeds 512 items")
			}
			if parseItem.Kind == MenuItemSeparator {
				if parseItem.ID != "" || parseItem.Label != "" || parseItem.Shortcut != "" || len(parseItem.Icon) != 0 || len(parseItem.Items) != 0 || parseItem.Checked || parseItem.Hidden {
					return errors.New("separator contains unsupported fields")
				}
				continue
			}
			if !validateMenuText(parseItem.ID, 128) || !validateMenuText(parseItem.Label, 256) || parseItem.ID == "" || parseItem.Label == "" {
				return errors.New("menu item requires valid ID and label")
			}
			if parseIDs[parseItem.ID] {
				return errors.New("menu item IDs must be unique")
			}
			parseIDs[parseItem.ID] = true
			if !validateMenuText(parseItem.Shortcut, 64) {
				return errors.New("invalid menu shortcut")
			}
			if len(parseItem.Icon) > 256*1024 {
				return errors.New("menu item icon exceeds 256 KiB")
			}
			parseIconBytes += len(parseItem.Icon)
			if parseIconBytes > 1024*1024 {
				return errors.New("menu icons exceed 1 MiB")
			}
			switch parseItem.Kind {
			case MenuItemCommand, MenuItemCheckbox, MenuItemRadio:
				if len(parseItem.Items) != 0 {
					return errors.New("selectable menu item cannot contain children")
				}
				if parseItem.Kind == MenuItemCommand && parseItem.Checked {
					return errors.New("command menu item cannot be checked")
				}
			case MenuItemSubmenu:
				if len(parseItem.Items) == 0 || parseItem.Checked || parseItem.Shortcut != "" {
					return errors.New("submenu requires children and cannot be checked or have a shortcut")
				}
				if parseErr := validateItems(parseItem.Items, parseDepth+1); parseErr != nil {
					return parseErr
				}
			default:
				return errors.New("unknown menu item kind")
			}
		}
		return nil
	}
	return validateItems(parseMenu.Items, 1)
}

// validateMenuText checks portable menu text bounds.
func validateMenuText(parseValue string, parseLimit int) bool {
	return len(parseValue) <= parseLimit && utf8.ValidString(parseValue) && !strings.ContainsRune(parseValue, 0)
}

// cloneMenuItems copies a menu tree before passing it to a backend.
func cloneMenuItems(parseItems []MenuItem) []MenuItem {
	parseCopy := make([]MenuItem, len(parseItems))
	for parseIndex := range parseItems {
		parseCopy[parseIndex] = parseItems[parseIndex]
		parseCopy[parseIndex].Icon = append([]byte(nil), parseItems[parseIndex].Icon...)
		parseCopy[parseIndex].Items = cloneMenuItems(parseItems[parseIndex].Items)
	}
	return parseCopy
}
