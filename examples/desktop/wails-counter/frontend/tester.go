//go:build js && wasm

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/monstercameron/GoWebComponents/v8/events"
	. "github.com/monstercameron/GoWebComponents/v8/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v8/interop"
	"github.com/monstercameron/GoWebComponents/v8/state"
	"github.com/monstercameron/GoWebComponents/v8/ui"
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

const (
	testerChildWindowTemplateID     = "counter"
	testerChildMicaWindowTemplateID = "mica"
)

var (
	testerTrayIconsOnce sync.Once
	testerTrayLightPNG  []byte
	testerTrayDarkPNG   []byte
)

var testerCases = []testerGroup{
	{"tester-dialogs", "01 / Dialogs & paths", []testerCase{
		{"api-open-file", "Open file", "open-file"},
		{"api-open-files", "Open multiple files", "open-files"},
		{"api-open-directory", "Choose directory", "open-directory"},
		{"api-open-directories", "Choose multiple directories (v3.1.0)", "open-directories"},
		{"api-save-path", "Choose save path", "save-path"},
	}},
	{"tester-messages", "02 / Messages & clipboard", []testerCase{
		{"api-message-info", "Show information", "message-info"},
		{"api-message-warning", "Show warning", "message-warning"},
		{"api-message-error", "Show error", "message-error"},
		{"api-message-question", "Ask question (No default)", "message-question"},
		{"api-message-question-custom", "Ask 3-button TaskDialog (v3.1.0)", "message-question-custom"},
		{"api-clipboard-write", "Write fixture to clipboard", "clipboard-write"},
		{"api-clipboard-read", "Inspect clipboard fixture", "clipboard-read"},
	}},
	{"tester-window", "03 / Calling window", []testerCase{
		{"api-window-info", "Inspect window", "window-info"},
		{"api-window-title", "Set fixture title", "window-set-title"},
		{"api-window-position", "Move to 120 × 120", "window-set-position"},
		{"api-window-center", "Center window", "window-center"},
		{"api-window-screen", "Move to primary screen", "window-set-screen"},
		{"api-window-relative-position", "Move relative to screen", "window-set-relative-position"},
		{"api-window-bounds", "Set bounds 120,120 · 720×520", "window-set-bounds"},
		{"api-window-resize", "Resize to 720 × 520", "window-resize"},
		{"api-window-min-size", "Set minimum 480 × 360", "window-set-min-size"},
		{"api-window-max-size", "Set maximum 1440 × 1000", "window-set-max-size"},
		{"api-window-enable-constraints", "Enable size constraints", "window-enable-size-constraints"},
		{"api-window-disable-constraints", "Disable size constraints", "window-disable-size-constraints"},
		{"api-window-minimize", "Minimize window", "window-minimize"},
		{"api-window-unminimize", "Unminimize window", "window-unminimize"},
		{"api-window-maximize", "Maximize window", "window-maximize"},
		{"api-window-unmaximize", "Unmaximize window", "window-unmaximize"},
		{"api-window-toggle-maximize", "Toggle maximize", "window-toggle-maximize"},
		{"api-window-restore", "Restore window", "window-restore"},
		{"api-window-fullscreen", "Enter fullscreen", "window-fullscreen"},
		{"api-window-unfullscreen", "Exit fullscreen", "window-unfullscreen"},
		{"api-window-toggle-fullscreen", "Toggle fullscreen", "window-toggle-fullscreen"},
		{"api-window-focus", "Focus window", "window-focus"},
		{"api-window-show", "Show window", "window-show"},
		{"api-window-snap-assist", "Open Snap Assist (Win11)", "window-snap-assist"},
		{"api-window-reload", "Reload WebView", "window-reload"},
		{"api-window-pin", "Enable always on top", "window-set-always-on-top"},
		{"api-window-resizable", "Enable resizing", "window-set-resizable"},
		{"api-window-frameless", "Enable frameless", "window-set-frameless"},
		{"api-window-toggle-frameless", "Toggle frameless", "window-toggle-frameless"},
		{"api-window-menu-show", "Show menu bar", "window-show-menu-bar"},
		{"api-window-menu-hide", "Hide menu bar", "window-hide-menu-bar"},
		{"api-window-menu-toggle", "Toggle menu bar", "window-toggle-menu-bar"},
		{"api-window-flash", "Flash taskbar", "window-flash"},
		{"api-window-protect", "Enable content protection", "window-set-content-protection"},
		{"api-window-ignore-mouse-off", "Accept mouse events", "window-ignore-mouse-off"},
		{"api-window-set-enabled", "Enable window input", "window-set-enabled"},
		{"api-window-background", "Set fixture background", "window-set-background-color"},
		{"api-window-minimize-button", "Disable minimize button", "window-set-minimize-button-state"},
		{"api-window-maximize-button", "Disable maximize button", "window-set-maximize-button-state"},
		{"api-window-close-button", "Disable close button", "window-set-close-button-state"},
		{"api-window-set-zoom", "Set zoom to 1.25", "window-set-zoom"},
		{"api-window-set-zoom-low", "Set zoom to 0.75 (v3.1.0)", "window-set-zoom-low"},
		{"api-window-zoom-in", "Zoom in", "window-zoom-in"},
		{"api-window-zoom-out", "Zoom out", "window-zoom-out"},
		{"api-window-zoom-reset", "Reset zoom", "window-zoom-reset"},
		{"api-window-print", "Open native print workflow", "window-print"},
	}},
	{"tester-inspection", "04 / Screens", []testerCase{
		{"api-screens", "Inspect screens & DPI", "screens"},
		{"api-screen-dip-point", "DIP → physical point", "screen-dip-to-physical-point"},
		{"api-screen-physical-point", "Physical → DIP point", "screen-physical-to-dip-point"},
		{"api-screen-dip-rect", "DIP → physical rectangle", "screen-dip-to-physical-rect"},
		{"api-screen-physical-rect", "Physical → DIP rectangle", "screen-physical-to-dip-rect"},
		{"api-screen-nearest-dip-point", "Nearest screen to DIP point", "screen-nearest-dip-point"},
		{"api-screen-nearest-physical-point", "Nearest screen to physical point", "screen-nearest-physical-point"},
		{"api-screen-nearest-dip-rect", "Nearest screen to DIP rectangle", "screen-nearest-dip-rect"},
		{"api-screen-nearest-physical-rect", "Nearest screen to physical rectangle", "screen-nearest-physical-rect"},
	}},
	{"tester-system", "05 / System & integration", []testerCase{
		{"api-system-environment", "Inspect system environment", "system-environment"},
		{"api-external-url", "Open HTTPS fixture", "external-url"},
		{"api-autostart-status", "Inspect autostart", "autostart-status"},
		{"api-autostart-enable", "Enable autostart", "autostart-enable"},
		{"api-autostart-disable", "Disable autostart", "autostart-disable"},
		{"api-menu-replace", "Install runtime test menu", "menu-replace"},
		{"api-context-install", "Install context menu", "context-install"},
		{"api-context-show", "Show context menu at 160 × 160", "context-show"},
		{"api-context-remove", "Remove context menu", "context-remove"},
		{"api-tray-upsert", "Install tray fixture (PNG + dark icon)", "tray-upsert"},
		{"api-tray-position-window", "Position window at tray", "tray-position-window"},
		{"api-tray-show", "Show tray fixture", "tray-show"},
		{"api-tray-hide", "Hide tray fixture", "tray-hide"},
		{"api-tray-remove", "Remove tray fixture", "tray-remove"},
		{"api-shortcut-upsert", "Register Ctrl+Alt+Shift+G", "shortcut-upsert"},
		{"api-shortcut-remove", "Remove global shortcut", "shortcut-remove"},
		{"api-child-create", "Create observer child", "child-create"},
		{"api-child-create-mica", "Create Mica backdrop child (v3.1.0)", "child-create-mica"},
		{"api-child-list", "List child windows", "child-list"},
		{"api-child-inspect", "Inspect observer child", "child-inspect"},
		{"api-child-show", "Show observer child", "child-show"},
		{"api-child-focus", "Focus observer child", "child-focus"},
		{"api-child-center", "Center observer child", "child-center"},
		{"api-child-resize", "Resize child to 800 × 600", "child-resize"},
		{"api-child-set-title", "Retitle observer child", "child-set-title"},
		{"api-child-minimize", "Minimize observer child", "child-minimize"},
		{"api-child-maximize", "Maximize observer child", "child-maximize"},
		{"api-child-restore", "Restore observer child", "child-restore"},
		{"api-child-hide", "Hide observer child", "child-hide"},
		{"api-child-close", "Close observer child", "child-close"},
	}},
	{"tester-file-manager", "06 / File manager", []testerCase{
		{"api-reveal-folder", "Reveal fixture folder", "reveal-folder"},
		{"api-reveal-file", "Reveal sample file", "reveal-file"},
	}},
}

// buildTesterTrayIcons lazily generates compact valid 16x16 PNG icons for light and dark Windows taskbars.
func buildTesterTrayIcons() ([]byte, []byte) {
	testerTrayIconsOnce.Do(func() {
		testerTrayLightPNG = encodeTesterBadgePNG(color.NRGBA{R: 7, G: 108, B: 101, A: 255}, color.NRGBA{R: 242, G: 240, B: 233, A: 255})
		testerTrayDarkPNG = encodeTesterBadgePNG(color.NRGBA{R: 224, G: 138, B: 60, A: 255}, color.NRGBA{R: 18, G: 32, B: 34, A: 255})
	})
	return testerTrayLightPNG, testerTrayDarkPNG
}

// encodeTesterBadgePNG encodes a 16x16 crisp framed square PNG icon.
func encodeTesterBadgePNG(fill, border color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			c := fill
			if x == 0 || y == 0 || x == 15 || y == 15 || (x >= 5 && x <= 10 && y >= 5 && y <= 10) {
				c = border
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// getTesterManualCases includes every native action and every human-only observation.
func getTesterManualCases() []testerCase {
	cases := []testerCase{}
	for _, group := range testerCases {
		cases = append(cases, group.Items...)
	}
	return append(cases,
		testerCase{"", "Context menu selection", "context-select"},
		testerCase{"", "Context checkbox", "context-check"},
		testerCase{"", "Context radio group", "context-radio"},
		testerCase{"", "Context submenu", "context-submenu"},
		testerCase{"", "Disabled menu action", "context-disabled"},
		testerCase{"", "Escape dismisses menu", "context-dismiss"},
		testerCase{"", "Menu / Ctrl+Shift+K", "menu-action"},
		testerCase{"", "F8 window shortcut", "shortcut"},
		testerCase{"", "Keyboard focus / Tab", "keyboard-focus"},
		testerCase{"", "IME / Unicode typing", "keyboard-ime"},
		testerCase{"", "Native edit menu", "edit-menu"},
		testerCase{"", "Physical resize / DPI", "resize-dpi"},
		testerCase{"", "Second window ownership", "second-window"},
		testerCase{"", "Title-bar close", "window-close"},
		testerCase{"", "Report export", "export-report"},
	)
}

// formatTesterResult shows the returned operation, not a synthetic success claim.
func formatTesterResult(result contracts.APIResult) string {
	parts := []string{result.Outcome}
	if result.Detail != "" {
		parts = append(parts, result.Detail)
	}
	if result.WindowID != "" {
		parts = append(parts, "window "+result.WindowID)
	}
	if len(result.Paths) != 0 {
		parts = append(parts, strings.Join(result.Paths, ", "))
	}
	return strings.Join(parts, " · ")
}

// formatTesterTask distinguishes cancellation of the caller's wait from OS dismissal.
func formatTesterTask(taskState ui.TaskState[contracts.APIResult]) string {
	if taskState.Running {
		return "Waiting for native result…"
	}
	if taskState.Cancelled {
		return "Wait cancelled. An open OS dialog may still need dismissal."
	}
	detail := formatTesterResult(taskState.Value)
	if taskState.Error != nil {
		return strings.TrimSpace(detail + " " + taskState.Error.Error())
	}
	if taskState.Ready {
		return detail
	}
	return "Not run"
}

// getTesterTimeout allows deliberate dialog inspection without extending routine calls.
func getTesterTimeout(action string) time.Duration {
	switch action {
	case "open-file", "open-files", "open-directory", "open-directories", "save-path",
		"message-info", "message-warning", "message-error", "message-question", "message-question-custom",
		"export-report", "window-print":
		return 5 * time.Minute
	default:
		return desktop.RequestTimeout
	}
}

// getTesterWindowRequest builds only the fields accepted for each window operation.
func getTesterWindowRequest(action string) desktop.WindowRequest {
	req := desktop.WindowRequest{Action: action}
	switch action {
	case "set-title":
		req.Title = "GWC Windows API Lab · portable title"
	case "set-position", "set-relative-position":
		req.X, req.Y = 120, 120
	case "resize", "set-bounds":
		if action == "set-bounds" {
			req.X, req.Y = 120, 120
		}
		req.Width = 720
		req.Height = 520
	case "set-min-size":
		req.Width, req.Height = 480, 360
	case "set-max-size":
		req.Width, req.Height = 1440, 1000
	case "set-always-on-top", "set-resizable", "set-frameless", "flash", "set-content-protection", "set-enabled":
		req.Enabled = true
	case "ignore-mouse-off":
		req.Action = "set-ignore-mouse-events"
		req.Enabled = false
	case "set-background-color":
		req.Red, req.Green, req.Blue, req.Alpha = 20, 24, 32, 255
	case "set-zoom":
		req.Zoom = 1.25
	case "set-zoom-low":
		req.Action = "set-zoom"
		req.Zoom = 0.75
	case "set-minimize-button-state", "set-maximize-button-state", "set-close-button-state", "set-fullscreen-button-state":
		req.State = "disabled"
	}
	return req
}

// isTesterPortableAction reports whether an action is dispatched through the portable desktop SDK.
func isTesterPortableAction(action string) bool {
	return action == "open-directories" ||
		action == "clipboard-write" ||
		action == "clipboard-read" ||
		strings.HasPrefix(action, "message-") ||
		strings.HasPrefix(action, "window-") ||
		action == "screens" ||
		strings.HasPrefix(action, "screen-") ||
		strings.HasPrefix(action, "system-") ||
		action == "external-url" ||
		strings.HasPrefix(action, "reveal-") ||
		strings.HasPrefix(action, "autostart-") ||
		action == "menu-replace" ||
		strings.HasPrefix(action, "context-") ||
		strings.HasPrefix(action, "tray-") ||
		strings.HasPrefix(action, "shortcut-") ||
		strings.HasPrefix(action, "child-")
}

// isTesterActionAvailable gates each portable control on its exact advertised host feature.
func isTesterActionAvailable(client desktop.Client, action string) bool {
	var feature desktop.Feature
	switch {
	case action == "open-directories":
		feature = desktop.FileDialogs
	case strings.HasPrefix(action, "clipboard-"):
		feature = desktop.Clipboard
	case strings.HasPrefix(action, "message-"):
		feature = desktop.MessageDialogs
	case action == "window-print":
		feature = desktop.WindowPrinting
	case strings.HasPrefix(action, "window-"):
		feature = desktop.WindowControls
	case action == "screens":
		feature = desktop.Screens
	case strings.HasPrefix(action, "screen-"):
		feature = desktop.ScreenGeometry
	case strings.HasPrefix(action, "system-"):
		feature = desktop.SystemEnvironment
	case action == "external-url":
		feature = desktop.ExternalURLs
	case strings.HasPrefix(action, "reveal-"):
		feature = desktop.FileManager
	case strings.HasPrefix(action, "autostart-"):
		feature = desktop.Autostart
	case action == "menu-replace" || strings.HasPrefix(action, "context-"):
		feature = desktop.RuntimeMenus
	case strings.HasPrefix(action, "tray-"):
		feature = desktop.SystemTray
	case strings.HasPrefix(action, "shortcut-"):
		feature = desktop.GlobalShortcuts
	case strings.HasPrefix(action, "child-"):
		feature = desktop.ChildWindows
	default:
		return true
	}
	return client.Supports(feature)
}

// getTesterRuntimeMenu returns a bounded menu covering command, checkbox, radio, submenu, separator, shortcut, and disabled states.
func getTesterRuntimeMenu() desktop.Menu {
	return desktop.Menu{Items: []desktop.MenuItem{
		{ID: "tester-command", Kind: desktop.MenuItemCommand, Label: "Tester command", Enabled: true, Shortcut: "Ctrl+Shift+K"},
		{Kind: desktop.MenuItemSeparator},
		{ID: "tester-check", Kind: desktop.MenuItemCheckbox, Label: "Checked fixture", Enabled: true, Checked: true},
		{ID: "tester-radio-a", Kind: desktop.MenuItemRadio, Label: "Radio A", Enabled: true, Checked: true},
		{ID: "tester-radio-b", Kind: desktop.MenuItemRadio, Label: "Radio B", Enabled: true},
		{ID: "tester-disabled", Kind: desktop.MenuItemCommand, Label: "Disabled fixture", Enabled: false},
		{ID: "tester-submenu", Kind: desktop.MenuItemSubmenu, Label: "Nested fixtures", Enabled: true, Items: []desktop.MenuItem{
			{ID: "tester-nested", Kind: desktop.MenuItemCommand, Label: "Nested command", Enabled: true},
		}},
	}}
}

// getTesterContextMenu returns a separate fixture so menu item IDs remain unambiguous.
func getTesterContextMenu() desktop.Menu {
	return desktop.Menu{Items: []desktop.MenuItem{
		{ID: "tester-context-command", Kind: desktop.MenuItemCommand, Label: "Context command", Enabled: true},
		{ID: "tester-context-check", Kind: desktop.MenuItemCheckbox, Label: "Context checkbox", Enabled: true},
		{ID: "tester-context-submenu", Kind: desktop.MenuItemSubmenu, Label: "Context submenu", Enabled: true, Items: []desktop.MenuItem{
			{ID: "tester-context-nested", Kind: desktop.MenuItemCommand, Label: "Nested context command", Enabled: true},
		}},
	}}
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
		{ID: "tester-tray-submenu", Kind: desktop.MenuItemSubmenu, Label: "Tray submenu", Enabled: true, Items: []desktop.MenuItem{
			{ID: "tester-tray-nested", Kind: desktop.MenuItemCommand, Label: "Nested tray command", Enabled: true},
		}},
	}}
}

// getTesterChildWindowRequest targets the immutable counter template registered by the example host.
func getTesterChildWindowRequest() desktop.ChildWindowCreateRequest {
	return desktop.ChildWindowCreateRequest{
		ID:         "tester-observer",
		TemplateID: testerChildWindowTemplateID,
		Title:      "Windows API Lab — portable child",
		Width:      720,
		Height:     520,
	}
}

// getTesterMicaChildWindowRequest targets the immutable Windows Mica backdrop template.
func getTesterMicaChildWindowRequest() desktop.ChildWindowCreateRequest {
	return desktop.ChildWindowCreateRequest{
		ID:         "tester-mica",
		TemplateID: testerChildMicaWindowTemplateID,
		Title:      "Windows API Lab — Mica child",
		Width:      760,
		Height:     580,
	}
}

// buildTesterTrayUpsertRequest builds an in-place tray configuration with light/dark PNG icons and label fallback.
func buildTesterTrayUpsertRequest() desktop.TrayRequest {
	menu := getTesterTrayMenu()
	lightIcon, darkIcon := buildTesterTrayIcons()
	return desktop.TrayRequest{
		Action:       "upsert",
		ID:           "tester-tray",
		Label:        "GWC Lab",
		Tooltip:      "GoWebComponents Windows API Lab (Wails v3.1.0)",
		Icon:         lightIcon,
		DarkModeIcon: darkIcon,
		Menu:         &menu,
	}
}

// callTesterNativeAction uses the portable desktop SDK for native APIs; legacy pickers remain APIService probes.
func callTesterNativeAction(ctx context.Context, client desktop.Client, action string) (contracts.APIResult, error) {
	result := contracts.APIResult{ID: action, Outcome: "completed"}
	switch action {
	case "open-directories":
		selection, err := client.OpenDirectories(ctx, desktop.FileDialogOptions{Title: "API Lab — Choose multiple directories"})
		if err != nil {
			return result, err
		}
		if selection.Cancelled {
			result.Outcome = "cancelled"
			return result, nil
		}
		result.Paths = selection.Paths
		result.Detail = fmt.Sprintf("selected %d directories", len(selection.Paths))
		return result, nil
	case "clipboard-write":
		err := client.WriteClipboard(ctx, "GoWebComponents API clipboard fixture")
		result.Detail = "synthetic text written (replaces clipboard)"
		return result, err
	case "clipboard-read":
		val, err := client.ReadClipboard(ctx)
		if err == nil {
			result.Detail = fmt.Sprintf("matches=%t chars=%d", val == "GoWebComponents API clipboard fixture", len([]rune(val)))
		}
		return result, err
	case "message-info", "message-warning", "message-error", "message-question", "message-question-custom":
		req := desktop.MessageRequest{Kind: "info", Title: "API Lab", Message: "Portable desktop SDK message test"}
		switch action {
		case "message-warning":
			req.Kind = "warning"
		case "message-error":
			req.Kind = "error"
		case "message-question":
			req.Kind = "question"
			req.Buttons = []string{"Yes", "No"}
			req.DefaultButton = "No"
		case "message-question-custom":
			req.Kind = "question"
			req.Title = "API Lab — TaskDialogIndirect"
			req.Message = "Choose a custom Wails v3.1.0 TaskDialog action:"
			req.Buttons = []string{"Save", "Discard", "Cancel"}
			req.DefaultButton = "Save"
			req.CancelButton = "Cancel"
		}
		reply, err := client.ShowMessage(ctx, req)
		result.Detail = reply.Button
		return result, err
	case "window-info", "window-set-title", "window-set-position", "window-set-relative-position", "window-set-bounds",
		"window-center", "window-set-screen", "window-enable-size-constraints", "window-disable-size-constraints",
		"window-resize", "window-set-min-size", "window-set-max-size", "window-minimize", "window-unminimize",
		"window-maximize", "window-unmaximize", "window-toggle-maximize", "window-restore", "window-fullscreen",
		"window-unfullscreen", "window-toggle-fullscreen", "window-focus", "window-show", "window-snap-assist",
		"window-reload", "window-set-always-on-top", "window-set-resizable", "window-set-frameless",
		"window-toggle-frameless", "window-show-menu-bar", "window-hide-menu-bar", "window-toggle-menu-bar",
		"window-flash", "window-set-content-protection", "window-ignore-mouse-off", "window-set-enabled",
		"window-set-background-color", "window-set-minimize-button-state", "window-set-maximize-button-state",
		"window-set-close-button-state", "window-set-fullscreen-button-state", "window-set-zoom",
		"window-set-zoom-low", "window-zoom-in", "window-zoom-out", "window-zoom-reset":
		windowAction := strings.TrimPrefix(action, "window-")
		req := getTesterWindowRequest(windowAction)
		if windowAction == "set-screen" {
			screens, screensErr := client.ListScreens(ctx)
			if screensErr != nil {
				return result, screensErr
			}
			for _, screen := range screens {
				if screen.Primary {
					req.ScreenID = screen.ID
					break
				}
			}
			if req.ScreenID == "" && len(screens) != 0 {
				req.ScreenID = screens[0].ID
			}
			if req.ScreenID == "" {
				return result, errors.New("no screen is available")
			}
		}
		info, err := client.ControlWindow(ctx, req)
		if err == nil {
			result.WindowID = info.ID
			detail := fmt.Sprintf("%s size=%dx%d zoom=%.2f", windowAction, info.Width, info.Height, info.Zoom)
			if info.ScreenID != "" {
				detail += " screen=" + info.ScreenID
			}
			if info.BorderSizes != nil {
				detail += fmt.Sprintf(" borders=%d,%d,%d,%d", info.BorderSizes.Top, info.BorderSizes.Left, info.BorderSizes.Bottom, info.BorderSizes.Right)
			}
			if info.IgnoreMouseEvents {
				detail += " ignoreMouse=true"
			}
			result.Detail = detail
		}
		return result, err
	case "window-print":
		result.Detail = "native print workflow opened"
		return result, client.PrintWindow(ctx)
	case "screens":
		screens, err := client.ListScreens(ctx)
		if err == nil {
			encoded, marshalErr := json.Marshal(screens)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	case "screen-dip-to-physical-point", "screen-physical-to-dip-point", "screen-nearest-dip-point", "screen-nearest-physical-point",
		"screen-dip-to-physical-rect", "screen-physical-to-dip-rect", "screen-nearest-dip-rect", "screen-nearest-physical-rect":
		op := desktop.ScreenGeometryOperation(strings.TrimPrefix(action, "screen-"))
		req := desktop.ScreenGeometryRequest{Operation: op}
		if strings.HasSuffix(action, "point") {
			req.Point = desktop.ScreenPoint{X: 120, Y: 120}
		} else {
			req.Rect = desktop.ScreenRect{X: 120, Y: 120, Width: 720, Height: 520}
		}
		reply, err := client.ScreenGeometry(ctx, req)
		if err == nil {
			encoded, marshalErr := json.Marshal(reply)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	case "system-environment":
		info, err := client.InspectSystemEnvironment(ctx)
		if err == nil {
			encoded, marshalErr := json.Marshal(info)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	case "external-url":
		err := client.OpenExternalURL(ctx, "https://example.com/")
		result.Detail = "requested https://example.com/"
		return result, err
	case "reveal-folder", "reveal-file":
		report, err := desktop.Call[contracts.APIReport](ctx, client, "api.report")
		if err != nil {
			return result, err
		}
		target := report.FixtureDir
		selectFile := action == "reveal-file"
		if selectFile {
			target = filepath.Join(target, "sample-a.txt")
		}
		result.Detail = "requested reveal of trusted fixture path " + target
		return result, client.RevealPath(ctx, target, selectFile)
	case "autostart-status":
		status, err := client.GetAutostartStatus(ctx)
		if err == nil {
			encoded, marshalErr := json.Marshal(status)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	case "autostart-enable":
		result.Detail = "autostart enabled"
		return result, client.EnableAutostart(ctx)
	case "autostart-disable":
		result.Detail = "autostart disabled"
		return result, client.DisableAutostart(ctx)
	case "menu-replace":
		result.Detail = "runtime test menu installed; select an item to verify its event"
		return result, client.ReplaceMenu(ctx, getTesterRuntimeMenu())
	case "context-install":
		result.Detail = "context menu fixture installed"
		return result, client.InstallContextMenu(ctx, desktop.ContextMenuRequest{ID: "tester-context", Menu: getTesterContextMenu()})
	case "context-show":
		result.Detail = "context menu shown with scoped fixture data"
		return result, client.ShowContextMenu(ctx, desktop.ContextMenuShowRequest{ID: "tester-context", X: 160, Y: 160, Data: "tester-context-scope"})
	case "context-remove":
		result.Detail = "context menu fixture removed"
		return result, client.RemoveContextMenu(ctx, "tester-context")
	case "tray-upsert":
		result.Detail = "tray fixture installed/updated in-place with PNG + dark-mode icon"
		return result, client.ConfigureTray(ctx, buildTesterTrayUpsertRequest())
	case "tray-position-window":
		result.Detail = "window positioned relative to tray icon"
		return result, client.ConfigureTray(ctx, desktop.TrayRequest{Action: "position-window", ID: "tester-tray", Offset: 8})
	case "tray-remove":
		result.Detail = "tray fixture removed"
		return result, client.ConfigureTray(ctx, desktop.TrayRequest{Action: "remove", ID: "tester-tray"})
	case "tray-show", "tray-hide":
		trayAction := strings.TrimPrefix(action, "tray-")
		result.Detail = "tray fixture " + trayAction
		return result, client.ConfigureTray(ctx, desktop.TrayRequest{Action: trayAction, ID: "tester-tray"})
	case "shortcut-upsert":
		result.Detail = "global shortcut fixture registered; press Ctrl+Alt+Shift+G to verify its event"
		return result, client.ConfigureShortcut(ctx, desktop.ShortcutRequest{Action: "upsert", ID: "tester-shortcut", Accelerator: "Ctrl+Alt+Shift+G"})
	case "shortcut-remove":
		result.Detail = "global shortcut fixture removed"
		return result, client.ConfigureShortcut(ctx, desktop.ShortcutRequest{Action: "remove", ID: "tester-shortcut"})
	case "child-create", "child-create-mica":
		req := getTesterChildWindowRequest()
		if action == "child-create-mica" {
			req = getTesterMicaChildWindowRequest()
		}
		child, err := client.CreateChildWindow(ctx, req)
		if err == nil {
			encoded, marshalErr := json.Marshal(child)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	case "child-list":
		children, err := client.ListChildWindows(ctx)
		if err == nil {
			encoded, marshalErr := json.Marshal(children)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	case "child-inspect":
		child, err := client.InspectChildWindow(ctx, "tester-observer")
		if err == nil {
			encoded, marshalErr := json.Marshal(child)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	case "child-show", "child-hide", "child-close", "child-focus", "child-center", "child-minimize", "child-maximize", "child-restore":
		childAction := strings.TrimPrefix(action, "child-")
		child, err := client.ControlChildWindow(ctx, desktop.ChildWindowControlRequest{ID: "tester-observer", Action: childAction})
		if err == nil {
			encoded, marshalErr := json.Marshal(child)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	case "child-resize":
		child, err := client.ResizeChildWindow(ctx, "tester-observer", 800, 600)
		if err == nil {
			encoded, marshalErr := json.Marshal(child)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	case "child-set-title":
		child, err := client.SetChildWindowTitle(ctx, "tester-observer", "Windows API Lab — retitled child")
		if err == nil {
			encoded, marshalErr := json.Marshal(child)
			if marshalErr != nil {
				return result, marshalErr
			}
			result.Detail = string(encoded)
		}
		return result, err
	default:
		return desktop.Call[contracts.APIResult](ctx, client, "api.run", action)
	}
}

// callTesterAction dispatches each tester action exactly once through its owning API.
func callTesterAction(ctx context.Context, client desktop.Client, action string) (contracts.APIResult, error) {
	if isTesterPortableAction(action) {
		result, err := callTesterNativeAction(ctx, client, action)
		if err != nil {
			result.Outcome = "failed"
			result.Detail = ""
			publishLabActivity("sdk", action, "failed", err.Error())
		} else {
			publishLabActivity("sdk", action, result.Outcome, result.Detail)
		}
		return result, err
	}
	result, err := desktop.CallWithTimeout[contracts.APIResult](ctx, client, getTesterTimeout(action), "api.run", action)
	if err != nil {
		publishLabActivity("service", action, "failed", err.Error())
	} else {
		publishLabActivity("service", action, result.Outcome, result.Detail)
	}
	return result, err
}

// filterTesterGroups filters capability groups by label, action ID, or group title.
func filterTesterGroups(groups []testerGroup, query string) []testerGroup {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return groups
	}
	filtered := make([]testerGroup, 0, len(groups))
	for _, group := range groups {
		if strings.Contains(strings.ToLower(group.Title), needle) {
			filtered = append(filtered, group)
			continue
		}
		matched := make([]testerCase, 0, len(group.Items))
		for _, item := range group.Items {
			if strings.Contains(strings.ToLower(item.Label), needle) || strings.Contains(strings.ToLower(item.Action), needle) {
				matched = append(matched, item)
			}
		}
		if len(matched) != 0 {
			filtered = append(filtered, testerGroup{ID: group.ID, Title: group.Title, Items: matched})
		}
	}
	return filtered
}

// renderTesterCase owns one action and refreshes evidence within that task's lifetime.
func renderTesterCase(props testerActionProps) ui.Node {
	task := ui.UseTask(func(ctx context.Context) (contracts.APIResult, error) {
		result, err := callTesterAction(ctx, props.Client, props.Case.Action)
		if err == nil {
			err = props.Refresh(ctx)
		}
		return result, err
	})
	onRun := ui.UseEvent(func() { task.Start() })
	onCancel := ui.UseEvent(func() { task.Cancel() })
	taskState := task.Get()
	cardClass := ClassNames(
		"tester-case",
		When(taskState.Running, "is-running"),
		When(taskState.Error != nil, "is-error"),
		When(taskState.Ready && taskState.Error == nil, "is-ready"),
	)
	return Div(ClassStr(cardClass),
		Button(ID(props.Case.ID), OnClick(onRun), Disabled(!props.IsAvailable || taskState.Running), Aria("describedby", props.Case.ID+"-result"), props.Case.Label),
		If(taskState.Running, Button(ClassStr("tester-cancel"), OnClick(onCancel), "Cancel wait")),
		P(ID(props.Case.ID+"-result"), ClassStr("tester-case__hint"), Role("status"), formatTesterTask(taskState)),
	)
}

// renderTesterNav scrolls without changing the hash router's active route.
func renderTesterNav(props testerNavProps) ui.Node {
	errMsg := ui.UseState("")
	onJump := ui.UseEvent(func() {
		doc, err := interop.GetDocument()
		if err == nil {
			el, found, findErr := doc.ElementByID(props.Target)
			err = findErr
			if err == nil && found {
				err = el.ScrollIntoView(interop.ScrollIntoViewOptions{Block: "start"})
			}
		}
		if err != nil {
			errMsg.Set(err.Error())
		}
	})
	return Div(
		Button(OnClick(onJump), props.Label),
		If(errMsg.Get() != "", P(Role("status"), errMsg.Get())),
	)
}

// renderLabTelemetry subscribes to typed in-app + native activity events via events.UseTopic.
func renderLabTelemetry() ui.Node {
	feed := ui.UseState([]labActivityEvent(nil))
	lastActivity := state.UseAtom(labLastActivityAtom, labActivityEvent{})
	events.UseTopic(labActivityTopic, func(ev labActivityEvent) {
		lastActivity.Set(ev)
		feed.Update(func(prev []labActivityEvent) []labActivityEvent {
			next := make([]labActivityEvent, 0, min(len(prev)+1, labMaxTelemetryItems))
			next = append(next, ev)
			for i := 0; i < len(prev) && len(next) < labMaxTelemetryItems; i++ {
				if prev[i].Seq != ev.Seq {
					next = append(next, prev[i])
				}
			}
			return next
		})
	}, events.WithReplayLast())
	items := feed.Get()
	return Section(ClassStr("tester-telemetry"), Aria("label", "Live event telemetry"),
		H2("Live event bus"),
		IfElse(len(items) == 0,
			P(ClassStr("tester-case__hint"), "Waiting for in-app or native desktop events…"),
			Ul(ClassStr("tester-telemetry-list"),
				MapKeyed(items, func(ev labActivityEvent) any { return ev.Seq }, func(ev labActivityEvent) ui.Node {
					return Li(ClassStr("tester-telemetry-item"),
						Span(ClassStr("tester-telemetry-tag"), ev.Source),
						Span(ClassStr("tester-telemetry-action"), ev.Action),
						Span(ClassStr("tester-telemetry-time"), ev.At),
					)
				}),
			),
		),
	)
}

// pollTesterReport suppresses results/errors returned after the route lifetime ends.
func pollTesterReport(ctx context.Context, read func(context.Context) (contracts.APIReport, error), publish func(contracts.APIReport), fail func(error)) {
	defer interop.RecoverContainedPanic("API report polling")
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		report, err := read(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			fail(err)
		} else {
			publish(report)
		}
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}

// formatWindowEventStatus formats portable Wails v3.1.0 window events including zoom and dropped file counts.
func formatWindowEventStatus(ev desktop.WindowEvent) string {
	msg := fmt.Sprintf("Window event: %s window=%s position=%d,%d size=%dx%d files=%d", ev.Kind, ev.WindowID, ev.X, ev.Y, ev.Width, ev.Height, len(ev.Files))
	if ev.Zoom > 0 {
		msg += fmt.Sprintf(" zoom=%.2f", ev.Zoom)
	}
	return msg
}

// renderTester renders explicit operations, live deferred filtering, typed telemetry, and human verdicts.
func renderTester() ui.Node {
	client, connectErr := desktop.Connect()
	report := ui.UseState(contracts.APIReport{})
	initialStatus := "Session polling active; no visual tests are auto-passed."
	if connectErr != nil {
		initialStatus = "Native report unavailable: " + connectErr.Error()
	}
	status := ui.UseState(initialStatus)
	menuStatus := ui.UseState("No runtime menu selection observed.")
	integrationStatus := ui.UseState("No tray or global shortcut event observed.")
	windowEventStatus := ui.UseState("No portable window event observed.")
	filterQuery := ui.UseState("")
	deferredQuery := ui.UseDeferredValue(filterQuery.Get())
	filterID := ui.UseId()
	note := ui.UseState("")
	selected := ui.UseState("open-file")
	observation := ui.UseRef(testerObservation{})

	ui.UseGlobalKey(func(ev ui.KeyboardEvent) {
		if ev.GetKey() == "Escape" && filterQuery.Get() != "" {
			filterQuery.Set("")
		}
	}, true)

	readReport := func(ctx context.Context) (contracts.APIReport, error) {
		return desktop.Call[contracts.APIReport](ctx, client, "api.report")
	}
	refreshReport := func(ctx context.Context) error {
		next, err := readReport(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return fmt.Errorf("report refresh: %w", err)
		}
		report.Set(next)
		return nil
	}
	refreshTask := ui.UseTask(func(ctx context.Context) (contracts.APIResult, error) {
		if err := refreshReport(ctx); err != nil {
			return contracts.APIResult{}, err
		}
		return contracts.APIResult{Outcome: "completed", Detail: "Report refreshed"}, nil
	})
	exportTask := ui.UseTask(func(ctx context.Context) (contracts.APIResult, error) {
		result, err := desktop.CallWithTimeout[contracts.APIResult](ctx, client, getTesterTimeout("export-report"), "api.export")
		if err == nil {
			err = refreshReport(ctx)
		}
		return result, err
	})
	observeTask := ui.UseTask(func(ctx context.Context) (contracts.APIResult, error) {
		req := observation.Get()
		result, err := desktop.Call[contracts.APIResult](ctx, client, "api.observe", req.ID, req.Outcome, req.Detail)
		if err == nil {
			err = refreshReport(ctx)
			publishLabActivity("verdict", req.ID, req.Outcome, req.Detail)
		}
		return result, err
	})

	ui.UseEffect(func() func() {
		ctx, cancel := context.WithCancel(context.Background())
		unsubscribe := func() {}
		addCleanup := func(stop func()) {
			prev := unsubscribe
			unsubscribe = func() { prev(); stop() }
		}
		if connectErr == nil {
			go pollTesterReport(ctx, readReport, report.Set, func(err error) { status.Set(err.Error()) })
			if client.Supports(desktop.RuntimeMenus) {
				stop, subErr := client.SubscribeMenuSelections(ctx, func(sel desktop.MenuSelection, err error) {
					if ctx.Err() != nil {
						return
					}
					if err != nil {
						menuStatus.Set("Runtime menu event: " + err.Error())
						return
					}
					checked := ""
					if sel.Checked != nil {
						checked = fmt.Sprintf(" checked=%t", *sel.Checked)
					}
					detail := sel.ID + " menu=" + sel.MenuID + " data=" + sel.Data + checked
					menuStatus.Set("Runtime menu selected: " + detail)
					publishLabActivity("menu", sel.ID, "selected", detail)
				})
				if subErr != nil {
					menuStatus.Set("Runtime menu events unavailable: " + subErr.Error())
				} else {
					addCleanup(stop)
				}
			}
			if client.Supports(desktop.SystemTray) {
				stop, subErr := client.SubscribeTrayEvents(ctx, func(ev desktop.TrayEvent, err error) {
					if ctx.Err() != nil {
						return
					}
					if err != nil {
						integrationStatus.Set("Tray event: " + err.Error())
						return
					}
					detail := ev.ID + " " + ev.Kind + " item=" + ev.ItemID
					integrationStatus.Set("Tray event: " + detail)
					publishLabActivity("tray", ev.Kind, "received", detail)
				})
				if subErr == nil {
					addCleanup(stop)
				}
			}
			if client.Supports(desktop.GlobalShortcuts) {
				stop, subErr := client.SubscribeShortcutEvents(ctx, func(ev desktop.ShortcutEvent, err error) {
					if ctx.Err() != nil {
						return
					}
					if err != nil {
						integrationStatus.Set("Shortcut event: " + err.Error())
						return
					}
					integrationStatus.Set("Shortcut event: " + ev.ID)
					publishLabActivity("shortcut", ev.ID, "triggered", ev.ID)
				})
				if subErr == nil {
					addCleanup(stop)
				}
			}
			if client.Supports(desktop.WindowEvents) {
				handleWindowEvent := func(ev desktop.WindowEvent, err error) {
					if ctx.Err() != nil {
						return
					}
					if err != nil {
						windowEventStatus.Set("Window event: " + err.Error())
						return
					}
					formatted := formatWindowEventStatus(ev)
					windowEventStatus.Set(formatted)
					publishLabActivity("window", string(ev.Kind), "observed", ev.WindowID)
				}
				if stop, subErr := client.SubscribeWindowEvents(ctx, handleWindowEvent); subErr == nil {
					addCleanup(stop)
				}
				if stop, subErr := client.SubscribeWindowDrops(ctx, handleWindowEvent); subErr == nil {
					addCleanup(stop)
				}
			}
		}
		return func() {
			cancel()
			unsubscribe()
		}
	}, true)

	onRefresh := ui.UseEvent(func() { refreshTask.Start() })
	onExport := ui.UseEvent(func() { exportTask.Start() })
	onFilterInput := ui.UseEvent(func(ev ui.InputEvent) { filterQuery.Set(ev.GetValue()) })
	onClearFilter := ui.UseEvent(func() { filterQuery.Set("") })
	onNoteInput := ui.UseEvent(func(ev ui.InputEvent) { note.Set(ev.GetValue()) })
	onCaseInput := ui.UseEvent(func(ev ui.InputEvent) { selected.Set(ev.GetValue()) })
	recordVerdict := func(outcome string) {
		observation.Set(testerObservation{ID: selected.Get(), Outcome: outcome, Detail: note.Get()})
		observeTask.Start()
	}
	onPass := ui.UseEvent(func() { recordVerdict("observed-pass") })
	onFail := ui.UseEvent(func() { recordVerdict("observed-fail") })
	onNotTested := ui.UseEvent(func() { recordVerdict("not-tested") })

	visibleGroups := filterTesterGroups(testerCases, deferredQuery)
	totalVisibleCases := 0
	for _, group := range visibleGroups {
		totalVisibleCases += len(group.Items)
	}

	reportJSON, jsonErr := json.MarshalIndent(report.Get(), "", "  ")
	if jsonErr != nil {
		reportJSON = []byte(jsonErr.Error())
	}
	connection := "Native bridge connected"
	if connectErr != nil {
		connection = "Unavailable: " + connectErr.Error()
	}
	isObserving := observeTask.Get().Running || connectErr != nil
	contextLabel := "RIGHT-CLICK TEST SURFACE"
	contextHelp := "Select · checkbox · radio · submenu · disabled action · Escape"
	if connectErr != nil || !client.Supports(desktop.NativeMenus) {
		contextLabel = "RIGHT-CLICK TEST SURFACE (native menu unavailable)"
		contextHelp = "Native menus are disabled by the host feature policy."
	}

	return Div(ClassStr("tester-shell"),
		Header(ClassStr("tester-topbar"),
			Div(
				Span(ClassStr("tester-kicker"), "GWC v8 · WAILS v3.1.0 · WINDOWS"),
				H1("Native API laboratory"),
				P("Native outcomes are not visual verdicts."),
			),
			Div(ClassStr("tester-topbar-right"),
				ui.CreateElement(renderThemeToggle),
				Div(ClassStr("tester-links"),
					A(Href("#/counter"), "Counter"),
					A(Href("#/tester"), "Tester"),
				),
			),
		),
		Div(ClassStr("tester-layout"),
			Aside(ClassStr("tester-sidebar"),
				H2("Runbook"),
				P(ClassStr(ClassNames("tester-conn", When(connectErr == nil, "is-connected"))), connection),
				Nav(Aria("label", "Test sections"),
					MapKeyed(testerCases, func(group testerGroup) any { return group.ID }, func(group testerGroup) ui.Node {
						return ui.CreateElement(renderTesterNav, testerNavProps{Target: group.ID, Label: group.Title})
					}),
					ui.CreateElement(renderTesterNav, testerNavProps{Target: "tester-evidence", Label: "07 / Evidence log"}),
				),
				P("Use Tests / Ctrl+Shift+K and F8. Escape clears active filter or leaves fullscreen. Use the native Edit menu in the text field. Record a verdict only after observing the result."),
				ui.CreateElement(renderLabTelemetry),
			),
			Main(ClassStr("tester-main"),
				Div(ID("api-context-target"), ClassStr("tester-context-target"), TabIndex(0),
					contextLabel,
					P(contextHelp),
				),
				Div(ClassStr("tester-toolbar"),
					Div(ClassStr("tester-editor"),
						Label(For("api-editable"), "Keyboard / IME / native edit menu"),
						Input(ID("api-editable"), Placeholder("Type Unicode text, select, copy and paste…")),
					),
					Div(ClassStr("tester-editor"),
						Label(For(filterID), Textf("Filter API cases (%d shown · Escape clears)", totalVisibleCases)),
						Div(ClassStr("tester-filter-row"),
							Input(ID(filterID), Value(filterQuery.Get()), OnInput(onFilterInput), Placeholder("Filter e.g. zoom, mica, tray, TaskDialog…")),
							If(filterQuery.Get() != "", Button(ClassStr("tester-cancel"), OnClick(onClearFilter), "Clear")),
						),
					),
				),
				P(ClassStr("tester-warning"), "Clipboard warning: Write fixture replaces your current clipboard. Read is explicit; use the synthetic fixture, not private data. Selected paths and manual notes may appear in the shared session report."),
				P(ClassStr("tester-warning"), "System warning: Autostart changes Windows login startup, global shortcuts reserve a system key chord, and print opens the native print workflow. Use the paired disable/remove controls to restore state."),
				IfElse(len(visibleGroups) == 0,
					Section(ClassStr("tester-panel"),
						H2("No matching API cases"),
						P(ClassStr("tester-case__hint"), Textf("No controls matched %q. Press Escape or click Clear to reset.", deferredQuery)),
					),
					Fragment(
						MapKeyed(visibleGroups, func(group testerGroup) any { return group.ID }, func(group testerGroup) ui.Node {
							return Section(ID(group.ID), ClassStr("tester-panel"),
								H2(group.Title),
								Div(ClassStr("tester-grid"),
									MapKeyed(group.Items, func(item testerCase) any { return item.ID }, func(item testerCase) ui.Node {
										return ui.CreateElement(renderTesterCase, testerActionProps{
											Client:      client,
											Case:        item,
											IsAvailable: connectErr == nil && isTesterActionAvailable(client, item.Action),
											Refresh:     refreshReport,
										})
									}),
								),
							)
						}),
					),
				),
				Section(ID("tester-evidence"), ClassStr("tester-report"),
					H2("07 / Evidence log"),
					P(ID("api-status"), Role("status"), status.Get()),
					P(ID("api-menu-selection"), Role("status"), menuStatus.Get()),
					P(ID("api-integration-event"), Role("status"), integrationStatus.Get()),
					P(ID("api-window-event"), Role("status"), windowEventStatus.Get()),
					Div(ClassStr("tester-observation"),
						Label(For("api-case"), "Observed case"),
						Select(ID("api-case"), OnChange(onCaseInput), Value(selected.Get()),
							MapKeyed(getTesterManualCases(), func(item testerCase) any { return item.Action }, func(item testerCase) ui.Node {
								return Option(Value(item.Action), item.Label)
							}),
						),
						Label(For("api-note"), "Human observation (up to 2,048 bytes)"),
						Input(ID("api-note"), OnInput(onNoteInput), Value(note.Get()), Placeholder("What did you actually see?"), Attr("maxlength", 1024)),
					),
					Div(ClassStr("tester-controls"),
						Button(ID("api-pass"), OnClick(onPass), Disabled(isObserving), "Observed pass"),
						Button(ID("api-fail"), OnClick(onFail), Disabled(isObserving), "Observed fail"),
						Button(ID("api-not-tested"), OnClick(onNotTested), Disabled(isObserving), "Not tested"),
					),
					P(ID("api-observe-result"), Role("status"), formatTesterTask(observeTask.Get())),
					Div(ClassStr("tester-controls"),
						Button(ID("api-refresh"), OnClick(onRefresh), Disabled(connectErr != nil || refreshTask.Get().Running), "Refresh report"),
						Button(ID("api-export"), OnClick(onExport), Disabled(connectErr != nil || exportTask.Get().Running), "Export report…"),
					),
					P(ID("api-refresh-result"), Role("status"), formatTesterTask(refreshTask.Get())),
					P(ID("api-export-result"), Role("status"), formatTesterTask(exportTask.Get())),
					Pre(ID("api-report-log"), TabIndex(0), Aria("label", "Session report JSON"), string(reportJSON)),
				),
			),
		),
		Footer(ClassStr("tester-footer"), "Manual Windows verification · local session only · export is explicit · cancellation cannot force an OS picker closed"),
	)
}
