package desktop

import (
	"bytes"
	"context"
	"errors"
	"image/png"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// SystemTray identifies caller-owned Windows notification-area icons.
const SystemTray Feature = "system-tray"

// GlobalShortcuts identifies caller-owned operating-system keyboard shortcuts.
const GlobalShortcuts Feature = "global-shortcuts"

const (
	TrayConfigureMethod     = "desktop.tray.configure"
	ShortcutConfigureMethod = "desktop.shortcut.configure"
	TrayEventTopic          = "desktop.tray.event"
	ShortcutEventTopic      = "desktop.shortcut.event"
)

// TrayRequest creates, replaces, or removes one caller-owned tray icon.
type TrayRequest struct {
	Action  string `json:"action"`
	ID      string `json:"id"`
	Tooltip string `json:"tooltip,omitempty"`
	Icon    []byte `json:"icon,omitempty"`
	Menu    *Menu  `json:"menu,omitempty"`
}

// ShortcutRequest creates, replaces, or removes one caller-owned global shortcut.
type ShortcutRequest struct {
	Action      string `json:"action"`
	ID          string `json:"id"`
	Accelerator string `json:"accelerator,omitempty"`
}

// TrayEvent is the complete typed payload emitted for tray interaction.
type TrayEvent struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	ItemID string `json:"itemId,omitempty"`
}

// ShortcutEvent is the complete typed payload emitted when a shortcut fires.
type ShortcutEvent struct {
	ID string `json:"id"`
}

// TrayBackend is an optional NativeBackend extension for system tray support.
type TrayBackend interface {
	ConfigureTray(context.Context, TrayRequest) error
}

// ShortcutBackend is an optional NativeBackend extension for global shortcut support.
type ShortcutBackend interface {
	ConfigureShortcut(context.Context, ShortcutRequest) error
}

// TrayShortcutTopics returns the exact typed topics hosts should advertise with these features.
func TrayShortcutTopics() []string {
	return []string{TrayEventTopic, ShortcutEventTopic}
}

// ConfigureTray invokes one typed tray operation through this client.
func (parseClient Client) ConfigureTray(parseContext context.Context, parseRequest TrayRequest) error {
	if parseErr := validateTrayRequest(parseRequest); parseErr != nil {
		return getError(TrayConfigureMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(SystemTray); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, TrayConfigureMethod, parseRequest)
	return parseErr
}

// ConfigureShortcut invokes one typed global shortcut operation through this client.
func (parseClient Client) ConfigureShortcut(parseContext context.Context, parseRequest ShortcutRequest) error {
	if parseErr := validateShortcutRequest(parseRequest); parseErr != nil {
		return getError(ShortcutConfigureMethod, interop.CodeInvalid, parseErr)
	}
	if parseErr := parseClient.Require(GlobalShortcuts); parseErr != nil {
		return parseErr
	}
	_, parseErr := parseClient.getNativeReply(parseContext, ShortcutConfigureMethod, parseRequest)
	return parseErr
}

// SubscribeTrayEvents observes validated interactions with caller-owned tray icons.
func (parseClient Client) SubscribeTrayEvents(parseContext context.Context, parseHandler func(TrayEvent, error)) (func(), error) {
	if parseHandler == nil {
		return nil, getError(TrayEventTopic, interop.CodeInvalid, errors.New("tray event handler required"))
	}
	if parseErr := parseClient.Require(SystemTray); parseErr != nil {
		return nil, parseErr
	}
	return Subscribe(parseContext, parseClient, TrayEventTopic, func(parseEvent TrayEvent, parseErr error) {
		if parseErr == nil {
			if parseErr = validateTrayEvent(parseEvent); parseErr != nil {
				parseErr = getError(TrayEventTopic, interop.CodeDecode, parseErr)
			}
		}
		parseHandler(parseEvent, parseErr)
	})
}

// SubscribeShortcutEvents observes validated activations of caller-owned shortcuts.
func (parseClient Client) SubscribeShortcutEvents(parseContext context.Context, parseHandler func(ShortcutEvent, error)) (func(), error) {
	if parseHandler == nil {
		return nil, getError(ShortcutEventTopic, interop.CodeInvalid, errors.New("shortcut event handler required"))
	}
	if parseErr := parseClient.Require(GlobalShortcuts); parseErr != nil {
		return nil, parseErr
	}
	return Subscribe(parseContext, parseClient, ShortcutEventTopic, func(parseEvent ShortcutEvent, parseErr error) {
		if parseErr == nil {
			if parseErr = validateNativeID(parseEvent.ID); parseErr != nil {
				parseErr = getError(ShortcutEventTopic, interop.CodeDecode, parseErr)
			}
		}
		parseHandler(parseEvent, parseErr)
	})
}

// ConfigureTray applies one validated caller-owned tray operation.
func (parseHost *NativeHost) ConfigureTray(parseContext context.Context, parseRequest TrayRequest) error {
	if parseErr := parseHost.Require(SystemTray); parseErr != nil {
		return parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(TrayBackend)
	if !parseOK {
		return getError(TrayConfigureMethod, interop.CodeUnavailable, errors.New("tray backend unavailable"))
	}
	if parseContext == nil {
		return getError(TrayConfigureMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return getContextError(TrayConfigureMethod, parseErr)
	}
	if parseErr := validateTrayRequest(parseRequest); parseErr != nil {
		return getError(TrayConfigureMethod, interop.CodeInvalid, parseErr)
	}
	parseErr := parseBackend.ConfigureTray(parseContext, parseRequest)
	if parseContext.Err() != nil {
		return getContextError(TrayConfigureMethod, parseContext.Err())
	}
	return parseErr
}

// ConfigureShortcut applies one validated caller-owned global shortcut operation.
func (parseHost *NativeHost) ConfigureShortcut(parseContext context.Context, parseRequest ShortcutRequest) error {
	if parseErr := parseHost.Require(GlobalShortcuts); parseErr != nil {
		return parseErr
	}
	parseBackend, parseOK := parseHost.parseBackend.(ShortcutBackend)
	if !parseOK {
		return getError(ShortcutConfigureMethod, interop.CodeUnavailable, errors.New("shortcut backend unavailable"))
	}
	if parseContext == nil {
		return getError(ShortcutConfigureMethod, interop.CodeInvalid, errors.New("calling context required"))
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return getContextError(ShortcutConfigureMethod, parseErr)
	}
	if parseErr := validateShortcutRequest(parseRequest); parseErr != nil {
		return getError(ShortcutConfigureMethod, interop.CodeInvalid, parseErr)
	}
	parseErr := parseBackend.ConfigureShortcut(parseContext, parseRequest)
	if parseContext.Err() != nil {
		return getContextError(ShortcutConfigureMethod, parseContext.Err())
	}
	return parseErr
}

// validateTrayRequest bounds tray identity and presentation data.
func validateTrayRequest(parseRequest TrayRequest) error {
	if parseErr := validateNativeID(parseRequest.ID); parseErr != nil {
		return parseErr
	}
	switch parseRequest.Action {
	case "upsert":
		if len(parseRequest.Tooltip) > 256 {
			return errors.New("tray tooltip exceeds 256 bytes")
		}
		if parseErr := validateNativeText(parseRequest.Tooltip, "tray tooltip"); parseErr != nil {
			return parseErr
		}
		if len(parseRequest.Icon) > 256*1024 {
			return errors.New("tray icon exceeds 256 KiB")
		}
		if len(parseRequest.Icon) != 0 {
			parseConfig, parseErr := png.DecodeConfig(bytes.NewReader(parseRequest.Icon))
			if parseErr != nil || parseConfig.Width < 1 || parseConfig.Height < 1 || parseConfig.Width > 256 || parseConfig.Height > 256 {
				return errors.New("tray icon must be valid PNG data up to 256 by 256 pixels")
			}
		}
		if parseRequest.Menu != nil {
			if parseErr := validateMenu(*parseRequest.Menu); parseErr != nil {
				return parseErr
			}
			if parseErr := validateTrayMenuItems(parseRequest.Menu.Items); parseErr != nil {
				return parseErr
			}
		}
	case "remove", "show", "hide":
		if parseRequest.Tooltip != "" || len(parseRequest.Icon) != 0 || parseRequest.Menu != nil {
			return errors.New("tray lifecycle action does not accept presentation data")
		}
	default:
		return errors.New("unknown tray action")
	}
	return nil
}

// validateTrayMenuItems rejects presentation Wails cannot preserve and aligns emitted IDs.
func validateTrayMenuItems(parseItems []MenuItem) error {
	for _, parseItem := range parseItems {
		if parseItem.Hidden || len(parseItem.Icon) != 0 || parseItem.Shortcut != "" {
			return errors.New("tray menus do not support hidden items, item icons, or accelerators")
		}
		if parseItem.Kind != MenuItemSeparator {
			if parseErr := validateNativeID(parseItem.ID); parseErr != nil {
				return errors.New("tray menu item has invalid identifier")
			}
		}
		if parseErr := validateTrayMenuItems(parseItem.Items); parseErr != nil {
			return parseErr
		}
	}
	return nil
}

// validateShortcutRequest bounds shortcut identity and accelerator data.
func validateShortcutRequest(parseRequest ShortcutRequest) error {
	if parseErr := validateNativeID(parseRequest.ID); parseErr != nil {
		return parseErr
	}
	switch parseRequest.Action {
	case "upsert":
		if parseRequest.Accelerator == "" || len(parseRequest.Accelerator) > 128 {
			return errors.New("invalid shortcut accelerator")
		}
		if parseErr := validateNativeText(parseRequest.Accelerator, "shortcut accelerator"); parseErr != nil {
			return parseErr
		}
	case "remove":
		if parseRequest.Accelerator != "" {
			return errors.New("remove does not accept an accelerator")
		}
	default:
		return errors.New("unknown shortcut action")
	}
	return nil
}

// validateNativeID accepts a small portable identifier alphabet.
func validateNativeID(parseID string) error {
	if len(parseID) < 1 || len(parseID) > 64 {
		return errors.New("native identifier must contain 1 to 64 characters")
	}
	for _, parseRune := range parseID {
		if !((parseRune >= 'a' && parseRune <= 'z') || (parseRune >= 'A' && parseRune <= 'Z') || (parseRune >= '0' && parseRune <= '9') || parseRune == '-' || parseRune == '_' || parseRune == '.') {
			return errors.New("native identifier contains an invalid character")
		}
	}
	return nil
}

// validateTrayEvent rejects unknown native callback kinds and identifiers.
func validateTrayEvent(parseEvent TrayEvent) error {
	if parseErr := validateNativeID(parseEvent.ID); parseErr != nil {
		return parseErr
	}
	switch parseEvent.Kind {
	case "click", "right-click", "double-click":
		if parseEvent.ItemID != "" {
			return errors.New("pointer tray event contains menu item")
		}
		return nil
	case "menu":
		if parseErr := validateNativeID(parseEvent.ItemID); parseErr != nil {
			return parseErr
		}
		return nil
	default:
		return errors.New("unknown tray event kind")
	}
}
