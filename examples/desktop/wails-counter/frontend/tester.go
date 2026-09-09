//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v6/desktop"
	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type testerCase struct{ ID, Label, Action string }
type testerGroup struct {
	ID, Title string
	Items     []testerCase
}
type testerActionProps struct {
	Client      desktop.Client
	Case        testerCase
	IsAvailable bool
	Refresh     func(context.Context) error
}
type testerNavProps struct{ Target, Label string }
type testerObservation struct{ ID, Outcome, Detail string }

const testerChildWindowTemplateID = "counter"

var testerCases = []testerGroup{
	{"tester-dialogs", "01 / Dialogs & paths", []testerCase{{"api-open-file", "Open file", "open-file"}, {"api-open-files", "Open multiple files", "open-files"}, {"api-open-directory", "Choose directory", "open-directory"}, {"api-save-path", "Choose save path", "save-path"}}},
	{"tester-messages", "02 / Messages & clipboard", []testerCase{{"api-message-info", "Show information", "message-info"}, {"api-message-warning", "Show warning", "message-warning"}, {"api-message-error", "Show error", "message-error"}, {"api-message-question", "Ask question (No default)", "message-question"}, {"api-clipboard-write", "Write fixture to clipboard", "clipboard-write"}, {"api-clipboard-read", "Inspect clipboard fixture", "clipboard-read"}}},
	{"tester-window", "03 / Calling window", []testerCase{
		{"api-window-info", "Inspect window", "window-info"}, {"api-window-title", "Set fixture title", "window-set-title"}, {"api-window-position", "Move to 120 × 120", "window-set-position"}, {"api-window-center", "Center window", "window-center"}, {"api-window-screen", "Move to primary screen", "window-set-screen"},
		{"api-window-relative-position", "Move relative to screen", "window-set-relative-position"}, {"api-window-bounds", "Set bounds 120,120 · 720×520", "window-set-bounds"},
		{"api-window-resize", "Resize to 720 × 520", "window-resize"}, {"api-window-min-size", "Set minimum 480 × 360", "window-set-min-size"}, {"api-window-max-size", "Set maximum 1440 × 1000", "window-set-max-size"}, {"api-window-enable-constraints", "Enable size constraints", "window-enable-size-constraints"}, {"api-window-disable-constraints", "Disable size constraints", "window-disable-size-constraints"},
		{"api-window-minimize", "Minimize window", "window-minimize"}, {"api-window-unminimize", "Unminimize window", "window-unminimize"}, {"api-window-maximize", "Maximize window", "window-maximize"}, {"api-window-unmaximize", "Unmaximize window", "window-unmaximize"}, {"api-window-toggle-maximize", "Toggle maximize", "window-toggle-maximize"}, {"api-window-restore", "Restore window", "window-restore"}, {"api-window-fullscreen", "Enter fullscreen", "window-fullscreen"}, {"api-window-unfullscreen", "Exit fullscreen", "window-unfullscreen"}, {"api-window-toggle-fullscreen", "Toggle fullscreen", "window-toggle-fullscreen"}, {"api-window-focus", "Focus window", "window-focus"},
		{"api-window-pin", "Enable always on top", "window-set-always-on-top"}, {"api-window-resizable", "Enable resizing", "window-set-resizable"}, {"api-window-frameless", "Enable frameless", "window-set-frameless"}, {"api-window-toggle-frameless", "Toggle frameless", "window-toggle-frameless"}, {"api-window-menu-show", "Show menu bar", "window-show-menu-bar"}, {"api-window-menu-hide", "Hide menu bar", "window-hide-menu-bar"}, {"api-window-menu-toggle", "Toggle menu bar", "window-toggle-menu-bar"}, {"api-window-flash", "Flash taskbar", "window-flash"}, {"api-window-protect", "Enable content protection", "window-set-content-protection"}, {"api-window-background", "Set fixture background", "window-set-background-color"},
		{"api-window-minimize-button", "Disable minimize button", "window-set-minimize-button-state"}, {"api-window-maximize-button", "Disable maximize button", "window-set-maximize-button-state"}, {"api-window-close-button", "Disable close button", "window-set-close-button-state"},
		{"api-window-set-zoom", "Set zoom to 1.25", "window-set-zoom"}, {"api-window-zoom-in", "Zoom in", "window-zoom-in"}, {"api-window-zoom-out", "Zoom out", "window-zoom-out"}, {"api-window-zoom-reset", "Reset zoom", "window-zoom-reset"},
		{"api-window-print", "Open native print workflow", "window-print"},
	}},
	{"tester-inspection", "04 / Screens", []testerCase{{"api-screens", "Inspect screens & DPI", "screens"}, {"api-screen-dip-point", "DIP → physical point", "screen-dip-to-physical-point"}, {"api-screen-physical-point", "Physical → DIP point", "screen-physical-to-dip-point"}, {"api-screen-dip-rect", "DIP → physical rectangle", "screen-dip-to-physical-rect"}, {"api-screen-physical-rect", "Physical → DIP rectangle", "screen-physical-to-dip-rect"}, {"api-screen-nearest-dip-point", "Nearest screen to DIP point", "screen-nearest-dip-point"}, {"api-screen-nearest-physical-point", "Nearest screen to physical point", "screen-nearest-physical-point"}, {"api-screen-nearest-dip-rect", "Nearest screen to DIP rectangle", "screen-nearest-dip-rect"}, {"api-screen-nearest-physical-rect", "Nearest screen to physical rectangle", "screen-nearest-physical-rect"}}},
	{"tester-system", "05 / System & integration", []testerCase{{"api-system-environment", "Inspect system environment", "system-environment"}, {"api-external-url", "Open HTTPS fixture", "external-url"}, {"api-autostart-status", "Inspect autostart", "autostart-status"}, {"api-autostart-enable", "Enable autostart", "autostart-enable"}, {"api-autostart-disable", "Disable autostart", "autostart-disable"}, {"api-menu-replace", "Install runtime test menu", "menu-replace"}, {"api-context-install", "Install context menu", "context-install"}, {"api-context-show", "Show context menu at 160 × 160", "context-show"}, {"api-context-remove", "Remove context menu", "context-remove"}, {"api-tray-upsert", "Install tray fixture", "tray-upsert"}, {"api-tray-show", "Show tray fixture", "tray-show"}, {"api-tray-hide", "Hide tray fixture", "tray-hide"}, {"api-tray-remove", "Remove tray fixture", "tray-remove"}, {"api-shortcut-upsert", "Register Ctrl+Alt+Shift+G", "shortcut-upsert"}, {"api-shortcut-remove", "Remove global shortcut", "shortcut-remove"}, {"api-child-create", "Create observer child", "child-create"}, {"api-child-list", "List child windows", "child-list"}, {"api-child-inspect", "Inspect observer child", "child-inspect"}, {"api-child-show", "Show observer child", "child-show"}, {"api-child-hide", "Hide observer child", "child-hide"}, {"api-child-close", "Close observer child", "child-close"}}},
	{"tester-file-manager", "06 / File manager", []testerCase{{"api-reveal-folder", "Reveal fixture folder", "reveal-folder"}, {"api-reveal-file", "Reveal sample file", "reveal-file"}}},
}

// getTesterManualCases includes every native action and every human-only observation.
func getTesterManualCases() []testerCase {
	parseCases := []testerCase{}
	for _, parseGroup := range testerCases {
		parseCases = append(parseCases, parseGroup.Items...)
	}
	return append(parseCases,
		testerCase{"", "Context menu selection", "context-select"}, testerCase{"", "Context checkbox", "context-check"},
		testerCase{"", "Context radio group", "context-radio"}, testerCase{"", "Context submenu", "context-submenu"},
		testerCase{"", "Disabled menu action", "context-disabled"}, testerCase{"", "Escape dismisses menu", "context-dismiss"},
		testerCase{"", "Menu / Ctrl+Shift+K", "menu-action"}, testerCase{"", "F8 window shortcut", "shortcut"},
		testerCase{"", "Keyboard focus / Tab", "keyboard-focus"}, testerCase{"", "IME / Unicode typing", "keyboard-ime"},
		testerCase{"", "Native edit menu", "edit-menu"}, testerCase{"", "Physical resize / DPI", "resize-dpi"},
		testerCase{"", "Second window ownership", "second-window"}, testerCase{"", "Title-bar close", "window-close"},
		testerCase{"", "Report export", "export-report"})
}

// formatTesterResult shows the returned operation, not a synthetic success claim.
func formatTesterResult(parseResult contracts.APIResult) string {
	parseParts := []string{parseResult.Outcome}
	if parseResult.Detail != "" {
		parseParts = append(parseParts, parseResult.Detail)
	}
	if parseResult.WindowID != "" {
		parseParts = append(parseParts, "window "+parseResult.WindowID)
	}
	if len(parseResult.Paths) != 0 {
		parseParts = append(parseParts, strings.Join(parseResult.Paths, ", "))
	}
	return strings.Join(parseParts, " · ")
}

// formatTesterTask distinguishes cancellation of the caller's wait from OS dismissal.
func formatTesterTask(parseState ui.TaskState[contracts.APIResult]) string {
	if parseState.Running {
		return "Waiting for native result…"
	}
	if parseState.Cancelled {
		return "Wait cancelled. An open OS dialog may still need dismissal."
	}
	parseDetail := formatTesterResult(parseState.Value)
	if parseState.Error != nil {
		return strings.TrimSpace(parseDetail + " " + parseState.Error.Error())
	}
	if parseState.Ready {
		return parseDetail
	}
	return "Not run"
}

// getTesterTimeout allows deliberate dialog inspection without extending routine calls.
func getTesterTimeout(parseAction string) time.Duration {
	switch parseAction {
	case "open-file", "open-files", "open-directory", "save-path", "message-info", "message-warning", "message-error", "message-question", "export-report", "window-print":
		return 5 * time.Minute
	default:
		return desktop.RequestTimeout
	}
}

// getTesterWindowRequest builds only the fields accepted for each window operation.
func getTesterWindowRequest(parseAction string) desktop.WindowRequest {
	parseRequest := desktop.WindowRequest{Action: parseAction}
	switch parseAction {
	case "set-title":
		parseRequest.Title = "GWC Windows API Lab · portable title"
	case "set-position", "set-relative-position":
		parseRequest.X, parseRequest.Y = 120, 120
	case "resize", "set-bounds":
		if parseAction == "set-bounds" {
			parseRequest.X, parseRequest.Y = 120, 120
		}
		parseRequest.Width = 720
		parseRequest.Height = 520
	case "set-min-size":
		parseRequest.Width, parseRequest.Height = 480, 360
	case "set-max-size":
		parseRequest.Width, parseRequest.Height = 1440, 1000
	case "set-always-on-top", "set-resizable", "set-frameless", "flash", "set-content-protection":
		parseRequest.Enabled = true
	case "set-background-color":
		parseRequest.Red, parseRequest.Green, parseRequest.Blue, parseRequest.Alpha = 20, 24, 32, 255
	case "set-zoom":
		parseRequest.Zoom = 1.25
	case "set-minimize-button-state", "set-maximize-button-state", "set-close-button-state", "set-fullscreen-button-state":
		parseRequest.State = "disabled"
	}
	return parseRequest
}

// isTesterPortableAction reports whether an action is dispatched through the portable desktop SDK.
func isTesterPortableAction(parseAction string) bool {
	return parseAction == "clipboard-write" || parseAction == "clipboard-read" || strings.HasPrefix(parseAction, "message-") || strings.HasPrefix(parseAction, "window-") || parseAction == "screens" || strings.HasPrefix(parseAction, "screen-") || strings.HasPrefix(parseAction, "system-") || parseAction == "external-url" || strings.HasPrefix(parseAction, "reveal-") || strings.HasPrefix(parseAction, "autostart-") || parseAction == "menu-replace" || strings.HasPrefix(parseAction, "context-") || strings.HasPrefix(parseAction, "tray-") || strings.HasPrefix(parseAction, "shortcut-") || strings.HasPrefix(parseAction, "child-")
}

// isTesterActionAvailable gates each portable control on its exact advertised host feature.
func isTesterActionAvailable(parseClient desktop.Client, parseAction string) bool {
	var parseFeature desktop.Feature
	switch {
	case strings.HasPrefix(parseAction, "clipboard-"):
		parseFeature = desktop.Clipboard
	case strings.HasPrefix(parseAction, "message-"):
		parseFeature = desktop.MessageDialogs
	case parseAction == "window-print":
		parseFeature = desktop.WindowPrinting
	case strings.HasPrefix(parseAction, "window-"):
		parseFeature = desktop.WindowControls
	case parseAction == "screens":
		parseFeature = desktop.Screens
	case strings.HasPrefix(parseAction, "screen-"):
		parseFeature = desktop.ScreenGeometry
	case strings.HasPrefix(parseAction, "system-"):
		parseFeature = desktop.SystemEnvironment
	case parseAction == "external-url":
		parseFeature = desktop.ExternalURLs
	case strings.HasPrefix(parseAction, "reveal-"):
		parseFeature = desktop.FileManager
	case strings.HasPrefix(parseAction, "autostart-"):
		parseFeature = desktop.Autostart
	case parseAction == "menu-replace" || strings.HasPrefix(parseAction, "context-"):
		parseFeature = desktop.RuntimeMenus
	case strings.HasPrefix(parseAction, "tray-"):
		parseFeature = desktop.SystemTray
	case strings.HasPrefix(parseAction, "shortcut-"):
		parseFeature = desktop.GlobalShortcuts
	case strings.HasPrefix(parseAction, "child-"):
		parseFeature = desktop.ChildWindows
	default:
		return true
	}
	return parseClient.Supports(parseFeature)
}

// getTesterRuntimeMenu returns a bounded menu covering command, checkbox, radio, submenu, separator, shortcut, and disabled states.
func getTesterRuntimeMenu() desktop.Menu {
	return desktop.Menu{Items: []desktop.MenuItem{{ID: "tester-command", Kind: desktop.MenuItemCommand, Label: "Tester command", Enabled: true, Shortcut: "Ctrl+Shift+K"}, {Kind: desktop.MenuItemSeparator}, {ID: "tester-check", Kind: desktop.MenuItemCheckbox, Label: "Checked fixture", Enabled: true, Checked: true}, {ID: "tester-radio-a", Kind: desktop.MenuItemRadio, Label: "Radio A", Enabled: true, Checked: true}, {ID: "tester-radio-b", Kind: desktop.MenuItemRadio, Label: "Radio B", Enabled: true}, {ID: "tester-disabled", Kind: desktop.MenuItemCommand, Label: "Disabled fixture", Enabled: false}, {ID: "tester-submenu", Kind: desktop.MenuItemSubmenu, Label: "Nested fixtures", Enabled: true, Items: []desktop.MenuItem{{ID: "tester-nested", Kind: desktop.MenuItemCommand, Label: "Nested command", Enabled: true}}}}}
}

// getTesterContextMenu returns a separate fixture so menu item IDs remain unambiguous.
func getTesterContextMenu() desktop.Menu {
	return desktop.Menu{Items: []desktop.MenuItem{{ID: "tester-context-command", Kind: desktop.MenuItemCommand, Label: "Context command", Enabled: true}, {ID: "tester-context-check", Kind: desktop.MenuItemCheckbox, Label: "Context checkbox", Enabled: true}, {ID: "tester-context-submenu", Kind: desktop.MenuItemSubmenu, Label: "Context submenu", Enabled: true, Items: []desktop.MenuItem{{ID: "tester-context-nested", Kind: desktop.MenuItemCommand, Label: "Nested context command", Enabled: true}}}}}
}

// getTesterTrayMenu covers the menu presentations supported by the Windows tray adapter.
func getTesterTrayMenu() desktop.Menu {
	return desktop.Menu{Items: []desktop.MenuItem{
		{ID: "tester-tray-command", Kind: desktop.MenuItemCommand, Label: "Tray command", Enabled: true},
		{ID: "tester-tray-check", Kind: desktop.MenuItemCheckbox, Label: "Tray checkbox", Enabled: true, Checked: true},
		{ID: "tester-tray-radio-a", Kind: desktop.MenuItemRadio, Label: "Tray radio A", Enabled: true, Checked: true},
		{ID: "tester-tray-radio-b", Kind: desktop.MenuItemRadio, Label: "Tray radio B", Enabled: true},
		{Kind: desktop.MenuItemSeparator},
		{ID: "tester-tray-disabled", Kind: desktop.MenuItemCommand, Label: "Disabled tray command", Enabled: false},
		{ID: "tester-tray-submenu", Kind: desktop.MenuItemSubmenu, Label: "Tray submenu", Enabled: true, Items: []desktop.MenuItem{{ID: "tester-tray-nested", Kind: desktop.MenuItemCommand, Label: "Nested tray command", Enabled: true}}},
	}}
}

// getTesterChildWindowRequest targets the immutable template registered by the example host.
func getTesterChildWindowRequest() desktop.ChildWindowCreateRequest {
	return desktop.ChildWindowCreateRequest{ID: "tester-observer", TemplateID: testerChildWindowTemplateID, Title: "Windows API Lab — portable child", Width: 720, Height: 520}
}

// callTesterNativeAction uses the portable desktop SDK for native APIs; picker actions remain APIService probes.
func callTesterNativeAction(parseContext context.Context, parseClient desktop.Client, parseAction string) (contracts.APIResult, error) {
	parseResult := contracts.APIResult{ID: parseAction, Outcome: "completed"}
	switch parseAction {
	case "clipboard-write":
		parseErr := parseClient.WriteClipboard(parseContext, "GoWebComponents API clipboard fixture")
		parseResult.Detail = "synthetic text written (replaces clipboard)"
		return parseResult, parseErr
	case "clipboard-read":
		parseValue, parseErr := parseClient.ReadClipboard(parseContext)
		if parseErr == nil {
			parseResult.Detail = fmt.Sprintf("matches=%t chars=%d", parseValue == "GoWebComponents API clipboard fixture", len([]rune(parseValue)))
		}
		return parseResult, parseErr
	case "message-info", "message-warning", "message-error", "message-question":
		parseKind := "info"
		parseRequest := desktop.MessageRequest{Kind: parseKind, Title: "API Lab", Message: "Portable desktop SDK message test"}
		switch parseAction {
		case "message-warning":
			parseRequest.Kind = "warning"
		case "message-error":
			parseRequest.Kind = "error"
		case "message-question":
			parseRequest.Kind = "question"
			parseRequest.Buttons = []string{"Yes", "No"}
			parseRequest.DefaultButton = "No"
		}
		parseReply, parseErr := parseClient.ShowMessage(parseContext, parseRequest)
		parseResult.Detail = parseReply.Button
		return parseResult, parseErr
	case "window-info", "window-set-title", "window-set-position", "window-set-relative-position", "window-set-bounds", "window-center", "window-set-screen", "window-enable-size-constraints", "window-disable-size-constraints", "window-resize", "window-set-min-size", "window-set-max-size", "window-minimize", "window-unminimize", "window-maximize", "window-unmaximize", "window-toggle-maximize", "window-restore", "window-fullscreen", "window-unfullscreen", "window-toggle-fullscreen", "window-focus", "window-set-always-on-top", "window-set-resizable", "window-set-frameless", "window-toggle-frameless", "window-show-menu-bar", "window-hide-menu-bar", "window-toggle-menu-bar", "window-flash", "window-set-content-protection", "window-set-background-color", "window-set-minimize-button-state", "window-set-maximize-button-state", "window-set-close-button-state", "window-set-fullscreen-button-state", "window-set-zoom", "window-zoom-in", "window-zoom-out", "window-zoom-reset":
		parseWindowAction := parseAction[len("window-"):]
		parseRequest := getTesterWindowRequest(parseWindowAction)
		if parseWindowAction == "set-screen" {
			parseScreens, parseScreensErr := parseClient.ListScreens(parseContext)
			if parseScreensErr != nil {
				return parseResult, parseScreensErr
			}
			for _, parseScreen := range parseScreens {
				if parseScreen.Primary {
					parseRequest.ScreenID = parseScreen.ID
					break
				}
			}
			if parseRequest.ScreenID == "" && len(parseScreens) != 0 {
				parseRequest.ScreenID = parseScreens[0].ID
			}
			if parseRequest.ScreenID == "" {
				return parseResult, errors.New("no screen is available")
			}
		}
		parseInfo, parseErr := parseClient.ControlWindow(parseContext, parseRequest)
		if parseErr == nil {
			parseResult.WindowID = parseInfo.ID
			parseResult.Detail = fmt.Sprintf("%s size=%dx%d", parseWindowAction, parseInfo.Width, parseInfo.Height)
		}
		return parseResult, parseErr
	case "window-print":
		parseResult.Detail = "native print workflow opened"
		return parseResult, parseClient.PrintWindow(parseContext)
	case "screens":
		parseScreens, parseErr := parseClient.ListScreens(parseContext)
		if parseErr == nil {
			parseJSON, parseMarshalErr := json.Marshal(parseScreens)
			if parseMarshalErr != nil {
				return parseResult, parseMarshalErr
			}
			parseResult.Detail = string(parseJSON)
		}
		return parseResult, parseErr
	case "screen-dip-to-physical-point", "screen-physical-to-dip-point", "screen-nearest-dip-point", "screen-nearest-physical-point", "screen-dip-to-physical-rect", "screen-physical-to-dip-rect", "screen-nearest-dip-rect", "screen-nearest-physical-rect":
		parseOperation := desktop.ScreenGeometryOperation(strings.TrimPrefix(parseAction, "screen-"))
		parseRequest := desktop.ScreenGeometryRequest{Operation: parseOperation}
		if strings.HasSuffix(parseAction, "point") {
			parseRequest.Point = desktop.ScreenPoint{X: 120, Y: 120}
		} else {
			parseRequest.Rect = desktop.ScreenRect{X: 120, Y: 120, Width: 720, Height: 520}
		}
		parseReply, parseErr := parseClient.ScreenGeometry(parseContext, parseRequest)
		if parseErr == nil {
			parseJSON, parseMarshalErr := json.Marshal(parseReply)
			if parseMarshalErr != nil {
				return parseResult, parseMarshalErr
			}
			parseResult.Detail = string(parseJSON)
		}
		return parseResult, parseErr
	case "system-environment":
		parseInfo, parseErr := parseClient.InspectSystemEnvironment(parseContext)
		if parseErr == nil {
			parseJSON, parseMarshalErr := json.Marshal(parseInfo)
			if parseMarshalErr != nil {
				return parseResult, parseMarshalErr
			}
			parseResult.Detail = string(parseJSON)
		}
		return parseResult, parseErr
	case "external-url":
		parseErr := parseClient.OpenExternalURL(parseContext, "https://example.com/")
		parseResult.Detail = "requested https://example.com/"
		return parseResult, parseErr
	case "reveal-folder", "reveal-file":
		parseReport, parseErr := desktop.Call[contracts.APIReport](parseContext, parseClient, "api.report")
		if parseErr != nil {
			return parseResult, parseErr
		}
		parsePath := parseReport.FixtureDir
		parseSelectFile := parseAction == "reveal-file"
		if parseSelectFile {
			parsePath = filepath.Join(parsePath, "sample-a.txt")
		}
		parseResult.Detail = "requested reveal of trusted fixture path " + parsePath
		return parseResult, parseClient.RevealPath(parseContext, parsePath, parseSelectFile)
	case "autostart-status":
		parseStatus, parseErr := parseClient.GetAutostartStatus(parseContext)
		if parseErr == nil {
			parseJSON, parseMarshalErr := json.Marshal(parseStatus)
			if parseMarshalErr != nil {
				return parseResult, parseMarshalErr
			}
			parseResult.Detail = string(parseJSON)
		}
		return parseResult, parseErr
	case "autostart-enable":
		parseResult.Detail = "autostart enabled"
		return parseResult, parseClient.EnableAutostart(parseContext)
	case "autostart-disable":
		parseResult.Detail = "autostart disabled"
		return parseResult, parseClient.DisableAutostart(parseContext)
	case "menu-replace":
		parseResult.Detail = "runtime test menu installed; select an item to verify its event"
		return parseResult, parseClient.ReplaceMenu(parseContext, getTesterRuntimeMenu())
	case "context-install":
		parseResult.Detail = "context menu fixture installed"
		return parseResult, parseClient.InstallContextMenu(parseContext, desktop.ContextMenuRequest{ID: "tester-context", Menu: getTesterContextMenu()})
	case "context-show":
		parseResult.Detail = "context menu shown with scoped fixture data"
		return parseResult, parseClient.ShowContextMenu(parseContext, desktop.ContextMenuShowRequest{ID: "tester-context", X: 160, Y: 160, Data: "tester-context-scope"})
	case "context-remove":
		parseResult.Detail = "context menu fixture removed"
		return parseResult, parseClient.RemoveContextMenu(parseContext, "tester-context")
	case "tray-upsert":
		parseResult.Detail = "tray fixture installed; click it to verify its event"
		parseMenu := getTesterTrayMenu()
		return parseResult, parseClient.ConfigureTray(parseContext, desktop.TrayRequest{Action: "upsert", ID: "tester-tray", Tooltip: "GoWebComponents Windows API Lab", Menu: &parseMenu})
	case "tray-remove":
		parseResult.Detail = "tray fixture removed"
		return parseResult, parseClient.ConfigureTray(parseContext, desktop.TrayRequest{Action: "remove", ID: "tester-tray"})
	case "tray-show", "tray-hide":
		parseTrayAction := strings.TrimPrefix(parseAction, "tray-")
		parseResult.Detail = "tray fixture " + parseTrayAction
		return parseResult, parseClient.ConfigureTray(parseContext, desktop.TrayRequest{Action: parseTrayAction, ID: "tester-tray"})
	case "shortcut-upsert":
		parseResult.Detail = "global shortcut fixture registered; press Ctrl+Alt+Shift+G to verify its event"
		return parseResult, parseClient.ConfigureShortcut(parseContext, desktop.ShortcutRequest{Action: "upsert", ID: "tester-shortcut", Accelerator: "Ctrl+Alt+Shift+G"})
	case "shortcut-remove":
		parseResult.Detail = "global shortcut fixture removed"
		return parseResult, parseClient.ConfigureShortcut(parseContext, desktop.ShortcutRequest{Action: "remove", ID: "tester-shortcut"})
	case "child-create":
		parseChild, parseErr := parseClient.CreateChildWindow(parseContext, getTesterChildWindowRequest())
		if parseErr == nil {
			parseJSON, parseMarshalErr := json.Marshal(parseChild)
			if parseMarshalErr != nil {
				return parseResult, parseMarshalErr
			}
			parseResult.Detail = string(parseJSON)
		}
		return parseResult, parseErr
	case "child-list":
		parseChildren, parseErr := parseClient.ListChildWindows(parseContext)
		if parseErr == nil {
			parseJSON, parseMarshalErr := json.Marshal(parseChildren)
			if parseMarshalErr != nil {
				return parseResult, parseMarshalErr
			}
			parseResult.Detail = string(parseJSON)
		}
		return parseResult, parseErr
	case "child-inspect":
		parseChild, parseErr := parseClient.InspectChildWindow(parseContext, "tester-observer")
		if parseErr == nil {
			parseJSON, parseMarshalErr := json.Marshal(parseChild)
			if parseMarshalErr != nil {
				return parseResult, parseMarshalErr
			}
			parseResult.Detail = string(parseJSON)
		}
		return parseResult, parseErr
	case "child-show", "child-hide", "child-close":
		parseChildAction := strings.TrimPrefix(parseAction, "child-")
		parseChild, parseErr := parseClient.ControlChildWindow(parseContext, desktop.ChildWindowControlRequest{ID: "tester-observer", Action: parseChildAction})
		if parseErr == nil {
			parseJSON, parseMarshalErr := json.Marshal(parseChild)
			if parseMarshalErr != nil {
				return parseResult, parseMarshalErr
			}
			parseResult.Detail = string(parseJSON)
		}
		return parseResult, parseErr
	default:
		return desktop.Call[contracts.APIResult](parseContext, parseClient, "api.run", parseAction)
	}
}

// callTesterAction dispatches each tester action exactly once through its owning API.
func callTesterAction(parseContext context.Context, parseClient desktop.Client, parseAction string) (contracts.APIResult, error) {
	if isTesterPortableAction(parseAction) {
		parseResult, parseErr := callTesterNativeAction(parseContext, parseClient, parseAction)
		if parseErr != nil {
			parseResult.Outcome = "failed"
			parseResult.Detail = ""
		}
		return parseResult, parseErr
	}
	return desktop.CallWithTimeout[contracts.APIResult](parseContext, parseClient, getTesterTimeout(parseAction), "api.run", parseAction)
}

// renderTesterCase owns one action and refreshes evidence within that task's lifetime.
func renderTesterCase(parseProps testerActionProps) ui.Node {
	parseTask := ui.UseTask(func(parseContext context.Context) (contracts.APIResult, error) {
		parseResult, parseErr := callTesterAction(parseContext, parseProps.Client, parseProps.Case.Action)
		if parseErr == nil {
			parseErr = parseProps.Refresh(parseContext)
		}
		return parseResult, parseErr
	})
	parseRun := ui.UseEvent(func() { parseTask.Start() })
	parseCancel := ui.UseEvent(func() { parseTask.Cancel() })
	parseState := parseTask.Get()
	var parseCancelNode ui.Node
	if parseState.Running {
		parseCancelNode = Button(ClassStr("tester-cancel"), OnClick(parseCancel), Text("Cancel wait"))
	}
	return Div(ClassStr("tester-case"),
		Button(ID(parseProps.Case.ID), OnClick(parseRun), Disabled(!parseProps.IsAvailable || parseState.Running), Aria("describedby", parseProps.Case.ID+"-result"), Text(parseProps.Case.Label)),
		parseCancelNode,
		P(ID(parseProps.Case.ID+"-result"), ClassStr("tester-case__hint"), Role("status"), Text(formatTesterTask(parseState))))
}

// renderTesterNav scrolls without changing the hash router's active route.
func renderTesterNav(parseProps testerNavProps) ui.Node {
	parseError := ui.UseState("")
	parseJump := ui.UseEvent(func() {
		parseDocument, parseErr := interop.GetDocument()
		if parseErr == nil {
			parseElement, isFound, parseFindErr := parseDocument.ElementByID(parseProps.Target)
			parseErr = parseFindErr
			if parseErr == nil && isFound {
				parseErr = parseElement.ScrollIntoView(interop.ScrollIntoViewOptions{Block: "start"})
			}
		}
		if parseErr != nil {
			parseError.Set(parseErr.Error())
		}
	})
	var parseErrorNode ui.Node
	if parseError.Get() != "" {
		parseErrorNode = P(Role("status"), Text(parseError.Get()))
	}
	return Div(Button(OnClick(parseJump), Text(parseProps.Label)), parseErrorNode)
}

// pollTesterReport suppresses results/errors returned after the route lifetime ends.
func pollTesterReport(parseContext context.Context, parseRead func(context.Context) (contracts.APIReport, error), parsePublish func(contracts.APIReport), parseFail func(error)) {
	defer interop.RecoverContainedPanic("API report polling")
	parseTimer := time.NewTicker(time.Second)
	defer parseTimer.Stop()
	for {
		if parseContext.Err() != nil {
			return
		}
		parseReport, parseErr := parseRead(parseContext)
		if parseContext.Err() != nil {
			return
		}
		if parseErr != nil {
			parseFail(parseErr)
		} else {
			parsePublish(parseReport)
		}
		select {
		case <-parseContext.Done():
			return
		case <-parseTimer.C:
		}
	}
}

// renderTester renders explicit operations and separately recorded human verdicts.
func renderTester() ui.Node {
	parseClient, parseConnectErr := desktop.Connect()
	parseReport := ui.UseState(contracts.APIReport{})
	parseInitialStatus := "Session polling active; no visual tests are auto-passed."
	if parseConnectErr != nil {
		parseInitialStatus = "Native report unavailable: " + parseConnectErr.Error()
	}
	parseStatus := ui.UseState(parseInitialStatus)
	parseMenuStatus := ui.UseState("No runtime menu selection observed.")
	parseIntegrationStatus := ui.UseState("No tray or global shortcut event observed.")
	parseWindowEventStatus := ui.UseState("No portable window event observed.")
	parseNote := ui.UseState("")
	parseSelected := ui.UseState("open-file")
	parseObservation := ui.UseRef(testerObservation{})
	parseRead := func(parseContext context.Context) (contracts.APIReport, error) {
		return desktop.Call[contracts.APIReport](parseContext, parseClient, "api.report")
	}
	parseRefreshReport := func(parseContext context.Context) error {
		parseNext, parseErr := parseRead(parseContext)
		if parseContext.Err() != nil {
			return parseContext.Err()
		}
		if parseErr != nil {
			return fmt.Errorf("report refresh: %w", parseErr)
		}
		parseReport.Set(parseNext)
		return nil
	}
	parseRefreshTask := ui.UseTask(func(parseContext context.Context) (contracts.APIResult, error) {
		if parseErr := parseRefreshReport(parseContext); parseErr != nil {
			return contracts.APIResult{}, parseErr
		}
		return contracts.APIResult{Outcome: "completed", Detail: "Report refreshed"}, nil
	})
	parseExportTask := ui.UseTask(func(parseContext context.Context) (contracts.APIResult, error) {
		parseResult, parseErr := desktop.CallWithTimeout[contracts.APIResult](parseContext, parseClient, getTesterTimeout("export-report"), "api.export")
		if parseErr == nil {
			parseErr = parseRefreshReport(parseContext)
		}
		return parseResult, parseErr
	})
	parseObserveTask := ui.UseTask(func(parseContext context.Context) (contracts.APIResult, error) {
		parseRequest := parseObservation.Get()
		parseResult, parseErr := desktop.Call[contracts.APIResult](parseContext, parseClient, "api.observe", parseRequest.ID, parseRequest.Outcome, parseRequest.Detail)
		if parseErr == nil {
			parseErr = parseRefreshReport(parseContext)
		}
		return parseResult, parseErr
	})
	ui.UseEffect(func() func() {
		parseContext, parseCancel := context.WithCancel(context.Background())
		parseUnsubscribe := func() {}
		if parseConnectErr == nil {
			go pollTesterReport(parseContext, parseRead, parseReport.Set, func(parseErr error) { parseStatus.Set(parseErr.Error()) })
			if parseClient.Supports(desktop.RuntimeMenus) {
				parseStop, parseSubscribeErr := parseClient.SubscribeMenuSelections(parseContext, func(parseSelection desktop.MenuSelection, parseErr error) {
					if parseContext.Err() != nil {
						return
					}
					if parseErr != nil {
						parseMenuStatus.Set("Runtime menu event: " + parseErr.Error())
						return
					}
					parseChecked := ""
					if parseSelection.Checked != nil {
						parseChecked = fmt.Sprintf(" checked=%t", *parseSelection.Checked)
					}
					parseMenuStatus.Set("Runtime menu selected: " + parseSelection.ID + " menu=" + parseSelection.MenuID + " data=" + parseSelection.Data + parseChecked)
				})
				if parseSubscribeErr != nil {
					parseMenuStatus.Set("Runtime menu events unavailable: " + parseSubscribeErr.Error())
				} else {
					parseUnsubscribe = parseStop
				}
			}
			if parseClient.Supports(desktop.SystemTray) {
				parseStop, parseSubscribeErr := parseClient.SubscribeTrayEvents(parseContext, func(parseEvent desktop.TrayEvent, parseErr error) {
					if parseContext.Err() == nil && parseErr != nil {
						parseIntegrationStatus.Set("Tray event: " + parseErr.Error())
					} else if parseContext.Err() == nil {
						parseIntegrationStatus.Set("Tray event: " + parseEvent.ID + " " + parseEvent.Kind + " item=" + parseEvent.ItemID)
					}
				})
				if parseSubscribeErr == nil {
					parsePrevious := parseUnsubscribe
					parseUnsubscribe = func() { parsePrevious(); parseStop() }
				}
			}
			if parseClient.Supports(desktop.GlobalShortcuts) {
				parseStop, parseSubscribeErr := parseClient.SubscribeShortcutEvents(parseContext, func(parseEvent desktop.ShortcutEvent, parseErr error) {
					if parseContext.Err() == nil && parseErr != nil {
						parseIntegrationStatus.Set("Shortcut event: " + parseErr.Error())
					} else if parseContext.Err() == nil {
						parseIntegrationStatus.Set("Shortcut event: " + parseEvent.ID)
					}
				})
				if parseSubscribeErr == nil {
					parsePrevious := parseUnsubscribe
					parseUnsubscribe = func() { parsePrevious(); parseStop() }
				}
			}
			if parseClient.Supports(desktop.WindowEvents) {
				parseHandleWindowEvent := func(parseEvent desktop.WindowEvent, parseErr error) {
					if parseContext.Err() == nil && parseErr != nil {
						parseWindowEventStatus.Set("Window event: " + parseErr.Error())
					} else if parseContext.Err() == nil {
						parseWindowEventStatus.Set(fmt.Sprintf("Window event: %s window=%s position=%d,%d size=%dx%d files=%d", parseEvent.Kind, parseEvent.WindowID, parseEvent.X, parseEvent.Y, parseEvent.Width, parseEvent.Height, len(parseEvent.Files)))
					}
				}
				parseStop, parseSubscribeErr := parseClient.SubscribeWindowEvents(parseContext, parseHandleWindowEvent)
				if parseSubscribeErr == nil {
					parsePrevious := parseUnsubscribe
					parseUnsubscribe = func() { parsePrevious(); parseStop() }
				}
				parseDropStop, parseDropErr := parseClient.SubscribeWindowDrops(parseContext, parseHandleWindowEvent)
				if parseDropErr == nil {
					parsePrevious := parseUnsubscribe
					parseUnsubscribe = func() { parsePrevious(); parseDropStop() }
				}
			}
		}
		return func() {
			parseCancel()
			parseUnsubscribe()
		}
	}, true)
	parseRefresh := ui.UseEvent(func() { parseRefreshTask.Start() })
	parseExport := ui.UseEvent(func() { parseExportTask.Start() })
	parseNoteInput := ui.UseEvent(func(parseEvent ui.InputEvent) { parseNote.Set(parseEvent.GetValue()) })
	parseCaseInput := ui.UseEvent(func(parseEvent ui.InputEvent) { parseSelected.Set(parseEvent.GetValue()) })
	parseRecord := func(parseOutcome string) {
		parseObservation.Set(testerObservation{ID: parseSelected.Get(), Outcome: parseOutcome, Detail: parseNote.Get()})
		parseObserveTask.Start()
	}
	parsePass := ui.UseEvent(func() { parseRecord("observed-pass") })
	parseFail := ui.UseEvent(func() { parseRecord("observed-fail") })
	parseNotTested := ui.UseEvent(func() { parseRecord("not-tested") })
	parseGroups := []any{}
	parseNavigation := []any{Aria("label", "Test sections")}
	for _, parseGroup := range testerCases {
		parseButtons := []any{ClassStr("tester-grid")}
		for _, parseCase := range parseGroup.Items {
			parseButtons = append(parseButtons, ui.CreateElement(renderTesterCase, testerActionProps{Client: parseClient, Case: parseCase, IsAvailable: parseConnectErr == nil && isTesterActionAvailable(parseClient, parseCase.Action), Refresh: parseRefreshReport}))
		}
		parseGroups = append(parseGroups, Section(ID(parseGroup.ID), ClassStr("tester-panel"), H2(Text(parseGroup.Title)), Div(parseButtons...)))
		parseNavigation = append(parseNavigation, ui.CreateElement(renderTesterNav, testerNavProps{Target: parseGroup.ID, Label: parseGroup.Title}))
	}
	parseNavigation = append(parseNavigation, ui.CreateElement(renderTesterNav, testerNavProps{Target: "tester-evidence", Label: "07 / Evidence log"}))
	parseOptions := []any{ID("api-case"), OnChange(parseCaseInput), Value(parseSelected.Get())}
	for _, parseCase := range getTesterManualCases() {
		parseOptions = append(parseOptions, Option(Value(parseCase.Action), Text(parseCase.Label)))
	}
	parseJSON, parseJSONErr := json.MarshalIndent(parseReport.Get(), "", "  ")
	if parseJSONErr != nil {
		parseJSON = []byte(parseJSONErr.Error())
	}
	parseConnection := "Native bridge connected"
	if parseConnectErr != nil {
		parseConnection = "Unavailable: " + parseConnectErr.Error()
	}
	parseIsObserving := parseObserveTask.Get().Running || parseConnectErr != nil
	parseContextLabel := "RIGHT-CLICK TEST SURFACE"
	parseContextHelp := "Select · checkbox · radio · submenu · disabled action · Escape"
	if parseConnectErr != nil || !parseClient.Supports(desktop.NativeMenus) {
		parseContextLabel = "RIGHT-CLICK TEST SURFACE (native menu unavailable)"
		parseContextHelp = "Native menus are disabled by the host feature policy."
	}
	parseMain := []any{ClassStr("tester-main"),
		Div(ID("api-context-target"), ClassStr("tester-context-target"), TabIndex(0), Text(parseContextLabel), P(Text(parseContextHelp))),
		Div(ClassStr("tester-editor"), Label(For("api-editable"), Text("Keyboard / IME / native edit menu")), Input(ID("api-editable"), Placeholder("Type Unicode text, select, copy and paste…"))),
		P(ClassStr("tester-warning"), Text("Clipboard warning: Write fixture replaces your current clipboard. Read is explicit; use the synthetic fixture, not private data. Selected paths and manual notes may appear in the shared session report.")),
		P(ClassStr("tester-warning"), Text("System warning: Autostart changes Windows login startup, global shortcuts reserve a system key chord, and print opens the native print workflow. Use the paired disable/remove controls to restore state.")),
	}
	parseMain = append(parseMain, parseGroups...)
	parseMain = append(parseMain, Section(ID("tester-evidence"), ClassStr("tester-report"), H2(Text("07 / Evidence log")),
		P(ID("api-status"), Role("status"), Text(parseStatus.Get())),
		P(ID("api-menu-selection"), Role("status"), Text(parseMenuStatus.Get())),
		P(ID("api-integration-event"), Role("status"), Text(parseIntegrationStatus.Get())),
		P(ID("api-window-event"), Role("status"), Text(parseWindowEventStatus.Get())),
		Div(ClassStr("tester-observation"), Label(For("api-case"), Text("Observed case")), Select(parseOptions...), Label(For("api-note"), Text("Human observation (up to 2,048 bytes)")), Input(ID("api-note"), OnInput(parseNoteInput), Value(parseNote.Get()), Placeholder("What did you actually see?"), Attr("maxlength", 1024))),
		Div(ClassStr("tester-controls"), Button(ID("api-pass"), OnClick(parsePass), Disabled(parseIsObserving), Text("Observed pass")), Button(ID("api-fail"), OnClick(parseFail), Disabled(parseIsObserving), Text("Observed fail")), Button(ID("api-not-tested"), OnClick(parseNotTested), Disabled(parseIsObserving), Text("Not tested"))),
		P(ID("api-observe-result"), Role("status"), Text(formatTesterTask(parseObserveTask.Get()))),
		Div(ClassStr("tester-controls"), Button(ID("api-refresh"), OnClick(parseRefresh), Disabled(parseConnectErr != nil || parseRefreshTask.Get().Running), Text("Refresh report")), Button(ID("api-export"), OnClick(parseExport), Disabled(parseConnectErr != nil || parseExportTask.Get().Running), Text("Export report…"))),
		P(ID("api-refresh-result"), Role("status"), Text(formatTesterTask(parseRefreshTask.Get()))), P(ID("api-export-result"), Role("status"), Text(formatTesterTask(parseExportTask.Get()))),
		Pre(ID("api-report-log"), TabIndex(0), Aria("label", "Session report JSON"), Text(string(parseJSON)))))
	return Div(ClassStr("tester-shell"),
		Header(ClassStr("tester-topbar"), Div(Span(ClassStr("tester-kicker"), Text("GWC / WAILS · WINDOWS")), H1(Text("Native API laboratory")), P(Text("Native outcomes are not visual verdicts."))), Div(ClassStr("tester-links"), A(Href("#/counter"), Text("Counter")), A(Href("#/tester"), Text("Tester")))),
		Div(ClassStr("tester-layout"), Aside(ClassStr("tester-sidebar"), H2(Text("Runbook")), P(Text(parseConnection)), Nav(parseNavigation...), P(Text("Use Tests / Ctrl+Shift+K and F8. Escape leaves fullscreen. Use the native Edit menu in the text field. Record a verdict only after observing the result."))), Main(parseMain...)),
		Footer(ClassStr("tester-footer"), Text("Manual Windows verification · local session only · export is explicit · cancellation cannot force an OS picker closed")))
}
