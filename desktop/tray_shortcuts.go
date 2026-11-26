package desktop

import (
	"bytes"
	"context"
	"errors"
	"image/png"

	"github.com/monstercameron/GoWebComponents/v8/interop"
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

// TrayRequest creates, replaces, positions relative to, or removes one caller-owned tray icon.
type TrayRequest struct {
	Action       string `json:"action"`
	ID           string `json:"id"`
	Label        string `json:"label,omitempty"`
	Tooltip      string `json:"tooltip,omitempty"`
	Icon         []byte `json:"icon,omitempty"`
	DarkModeIcon []byte `json:"darkModeIcon,omitempty"`
	Menu         *Menu  `json:"menu,omitempty"`
	Offset       int    `json:"offset,omitempty"`
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
func validateTrayRequest(request TrayRequest) error {
	if err := validateNativeID(request.ID); err != nil {
		return err
	}
	switch request.Action {
	case "upsert":
		if request.Offset != 0 {
			return errors.New("tray upsert does not accept a window offset")
		}
		if len(request.Label) > 256 {
			return errors.New("tray label exceeds 256 bytes")
		}
		if err := validateNativeText(request.Label, "tray label"); err != nil {
			return err
		}
		if len(request.Tooltip) > 256 {
			return errors.New("tray tooltip exceeds 256 bytes")
		}
		if err := validateNativeText(request.Tooltip, "tray tooltip"); err != nil {
			return err
		}
		if err := validateTrayIconBytes(request.Icon, "tray icon"); err != nil {
			return err
		}
		if err := validateTrayIconBytes(request.DarkModeIcon, "tray dark mode icon"); err != nil {
			return err
		}
		if request.Menu != nil {
			if err := validateMenu(*request.Menu); err != nil {
				return err
			}
			if err := validateTrayMenuItems(request.Menu.Items); err != nil {
				return err
			}
		}
	case "position-window":
		if request.Label != "" || request.Tooltip != "" || len(request.Icon) != 0 || len(request.DarkModeIcon) != 0 || request.Menu != nil {
			return errors.New("tray position-window does not accept presentation data")
		}
		if request.Offset < -4096 || request.Offset > 4096 {
			return errors.New("tray window offset out of bounds")
		}
	case "remove", "show", "hide":
		if request.Label != "" || request.Tooltip != "" || len(request.Icon) != 0 || len(request.DarkModeIcon) != 0 || request.Menu != nil || request.Offset != 0 {
			return errors.New("tray lifecycle action does not accept presentation data")
		}
	default:
		return errors.New("unknown tray action")
	}
	return nil
}

// validateTrayIconBytes accepts bounded PNG (up to 256x256) or Windows ICO payloads.
func validateTrayIconBytes(data []byte, label string) error {
	if len(data) == 0 {
		return nil
	}
	if len(data) > 256*1024 {
		return errors.New(label + " exceeds 256 KiB")
	}
	if config, err := png.DecodeConfig(bytes.NewReader(data)); err == nil {
		if config.Width >= 1 && config.Height >= 1 && config.Width <= 256 && config.Height <= 256 {
			return nil
		}
		return errors.New(label + " PNG dimensions must be between 1 and 256 pixels")
	}
	// Accept valid Windows ICO containers (reserved=0, type=1, count>=1, >= 22-byte header+entry).
	if len(data) >= 22 && data[0] == 0 && data[1] == 0 && data[2] == 1 && data[3] == 0 {
		imageCount := int(data[4]) | int(data[5])<<8
		if imageCount >= 1 && imageCount <= 32 && len(data) >= 6+imageCount*16 {
			return nil
		}
	}
	return errors.New(label + " must be valid PNG (up to 256 by 256 pixels) or ICO data")
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
		if (parseRune < 'a' || parseRune > 'z') && (parseRune < 'A' || parseRune > 'Z') && (parseRune < '0' || parseRune > '9') && parseRune != '-' && parseRune != '_' && parseRune != '.' {
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
