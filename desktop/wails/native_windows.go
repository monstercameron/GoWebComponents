//go:build windows

package wails

import (
	"context"
	"errors"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// NewNativeBackend returns the pinned Wails implementation of portable native APIs.
func NewNativeBackend() desktop.NativeBackend {
	return &nativeBackend{parseTrayShortcuts: newNativeTrayShortcutState()}
}

type nativeBackend struct {
	parseTrayShortcuts *nativeTrayShortcutState
	parseWindows       *nativeWindowState
}

// Features reports capabilities implemented by this Wails adapter.
func (parseBackend nativeBackend) Features() []desktop.Feature {
	parseFeatures := []desktop.Feature{desktop.Clipboard, desktop.MessageDialogs, desktop.WindowControls, desktop.WindowPrinting, desktop.WindowEvents, desktop.Screens, desktop.ScreenGeometry, desktop.RuntimeMenus, desktop.SystemTray, desktop.GlobalShortcuts, desktop.SystemEnvironment, desktop.ExternalURLs, desktop.Autostart, desktop.FileManager}
	if parseBackend.parseWindows != nil {
		parseFeatures = append(parseFeatures, desktop.ChildWindows)
	}
	return parseFeatures
}

// ClipboardWrite writes text only when explicitly requested by the caller.
func (nativeBackend) ClipboardWrite(parseContext context.Context, parseRequest desktop.ClipboardWriteRequest) error {
	if parseErr := parseCaller(parseContext); parseErr != nil {
		return parseErr
	}
	parseApp := application.Get()
	if parseApp == nil || !parseApp.Clipboard.SetText(parseRequest.Text) {
		return unavailable("clipboard write failed")
	}
	return nil
}

// ClipboardRead reads text only when explicitly requested by the caller.
func (nativeBackend) ClipboardRead(parseContext context.Context) (string, error) {
	if parseErr := parseCaller(parseContext); parseErr != nil {
		return "", parseErr
	}
	parseApp := application.Get()
	if parseApp == nil {
		return "", unavailable("Wails application unavailable")
	}
	parseText, parseOK := parseApp.Clipboard.Text()
	if !parseOK {
		return "", unavailable("clipboard read failed")
	}
	return parseText, nil
}

// ShowMessage displays a caller-owned Wails dialog and returns its exact button label.
func (nativeBackend) ShowMessage(parseContext context.Context, parseRequest desktop.MessageRequest) (desktop.MessageReply, error) {
	if parseErr := validateWindowsMessageRequest(parseRequest); parseErr != nil {
		return desktop.MessageReply{}, parseErr
	}
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return desktop.MessageReply{}, parseErr
	}
	parseApp := application.Get()
	if parseApp == nil {
		return desktop.MessageReply{}, unavailable("Wails application unavailable")
	}
	parseDialog := parseApp.Dialog.Info()
	switch parseRequest.Kind {
	case "question":
		parseDialog = parseApp.Dialog.Question()
	case "warning":
		parseDialog = parseApp.Dialog.Warning()
	case "error":
		parseDialog = parseApp.Dialog.Error()
	}
	parseChosen := make(chan string, 1)
	parseDialog.SetTitle(parseRequest.Title).SetMessage(parseRequest.Message).AttachToWindow(parseWindow)
	parseButtons := parseRequest.Buttons
	if len(parseButtons) == 0 {
		if parseRequest.Kind == "question" {
			parseButtons = []string{"Yes", "No"}
		} else {
			parseButtons = []string{"Ok"}
		}
	}
	parseAdded := make(map[string]*application.Button, len(parseButtons))
	for _, parseLabel := range parseButtons {
		parseLabelCopy := parseLabel
		parseButton := parseDialog.AddButton(parseLabel)
		parseButton.OnClick(func() { parseChosen <- parseLabelCopy })
		parseAdded[parseLabel] = parseButton
	}
	parseDefault := parseRequest.DefaultButton
	if parseDefault == "" {
		parseDefault = parseButtons[0]
	}
	parseCancel := parseRequest.CancelButton
	if parseCancel == "" {
		parseCancel = parseButtons[len(parseButtons)-1]
	}
	if parseButton := parseAdded[parseDefault]; parseButton != nil {
		parseDialog.SetDefaultButton(parseButton)
	}
	if parseButton := parseAdded[parseCancel]; parseButton != nil {
		parseDialog.SetCancelButton(parseButton)
	}
	parseDialog.Show()
	select {
	case parseButton := <-parseChosen:
		return desktop.MessageReply{Button: parseButton}, nil
	default:
		return desktop.MessageReply{}, errors.New("native dialog button unavailable")
	}
}

// validateWindowsMessageRequest rejects controls that Windows MessageBox cannot render.
func validateWindowsMessageRequest(parseRequest desktop.MessageRequest) error {
	parseButtons := parseRequest.Buttons
	if len(parseButtons) == 0 {
		if parseRequest.Kind == "question" {
			parseButtons = []string{"Yes", "No"}
		} else {
			parseButtons = []string{"Ok"}
		}
	}
	switch parseRequest.Kind {
	case "info", "warning", "error":
		if len(parseButtons) != 1 || parseButtons[0] != "Ok" {
			return invalid("Windows message dialogs support only the Ok button for this kind")
		}
	case "question":
		if len(parseButtons) != 2 || parseButtons[0] != "Yes" || parseButtons[1] != "No" {
			return invalid("Windows question dialogs support only Yes and No buttons")
		}
	default:
		return invalid("unknown message kind")
	}
	if parseRequest.DefaultButton != "" && parseRequest.DefaultButton != "Ok" && parseRequest.DefaultButton != "Yes" && parseRequest.DefaultButton != "No" {
		return invalid("unsupported Windows default message button")
	}
	if parseRequest.Kind != "question" && parseRequest.DefaultButton != "" && parseRequest.DefaultButton != "Ok" {
		return invalid("only Ok can be the default for this Windows message dialog")
	}
	if parseRequest.Kind == "question" && parseRequest.DefaultButton != "" && parseRequest.DefaultButton != "Yes" && parseRequest.DefaultButton != "No" {
		return invalid("only Yes or No can be the default for a Windows question dialog")
	}
	if parseRequest.CancelButton != "" && parseRequest.CancelButton != "Ok" && parseRequest.CancelButton != "No" {
		return invalid("unsupported Windows cancel message button")
	}
	if parseRequest.Kind != "question" && parseRequest.CancelButton != "" && parseRequest.CancelButton != "Ok" {
		return invalid("only Ok can cancel this Windows message dialog")
	}
	if parseRequest.Kind == "question" && parseRequest.CancelButton != "" {
		return invalid("Windows question dialogs do not support a cancel button")
	}
	return nil
}

// Window performs a caller-owned window inspection or control.
func (nativeBackend) Window(parseContext context.Context, parseRequest desktop.WindowRequest) (desktop.WindowInfo, error) {
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return desktop.WindowInfo{}, parseErr
	}
	switch parseRequest.Action {
	case "info":
	case "set-title":
		parseWindow.SetTitle(parseRequest.Title)
	case "set-screen":
		parseApp := application.Get()
		if parseApp == nil {
			return desktop.WindowInfo{}, unavailable("Wails application unavailable")
		}
		var parseScreen *application.Screen
		for _, parseCandidate := range parseApp.Screen.GetAll() {
			if parseCandidate.ID == parseRequest.ScreenID {
				parseScreen = parseCandidate
				break
			}
		}
		if parseScreen == nil {
			return desktop.WindowInfo{}, invalid("unknown screen id")
		}
		parseWindow.SetScreen(parseScreen)
	case "set-position":
		parseWindow.SetPosition(parseRequest.X, parseRequest.Y)
	case "set-relative-position":
		parseWindow.SetRelativePosition(parseRequest.X, parseRequest.Y)
	case "set-bounds":
		parseWindow.SetBounds(application.Rect{X: parseRequest.X, Y: parseRequest.Y, Width: parseRequest.Width, Height: parseRequest.Height})
	case "center":
		parseWindow.Center()
	case "enable-size-constraints":
		parseWindow.EnableSizeConstraints()
	case "disable-size-constraints":
		parseWindow.DisableSizeConstraints()
	case "resize":
		if parseRequest.Width <= 0 || parseRequest.Height <= 0 {
			return desktop.WindowInfo{}, invalid("window dimensions must be positive")
		}
		parseWindow.SetSize(parseRequest.Width, parseRequest.Height)
	case "set-min-size":
		parseWindow.SetMinSize(parseRequest.Width, parseRequest.Height)
	case "set-max-size":
		parseWindow.SetMaxSize(parseRequest.Width, parseRequest.Height)
	case "minimize":
		parseWindow.Minimise()
	case "unminimize":
		parseWindow.UnMinimise()
	case "maximize":
		parseWindow.Maximise()
	case "unmaximize":
		parseWindow.UnMaximise()
	case "toggle-maximize":
		parseWindow.ToggleMaximise()
	case "restore":
		parseWindow.Restore()
	case "fullscreen":
		parseWindow.Fullscreen()
	case "unfullscreen":
		parseWindow.UnFullscreen()
	case "toggle-fullscreen":
		parseWindow.ToggleFullscreen()
	case "focus":
		parseWindow.Focus()
	case "set-always-on-top":
		parseWindow.SetAlwaysOnTop(parseRequest.Enabled)
	case "set-resizable":
		parseWindow.SetResizable(parseRequest.Enabled)
	case "set-frameless":
		parseWindow.SetFrameless(parseRequest.Enabled)
	case "toggle-frameless":
		parseWindow.ToggleFrameless()
	case "show-menu-bar":
		parseWindow.ShowMenuBar()
	case "hide-menu-bar":
		parseWindow.HideMenuBar()
	case "toggle-menu-bar":
		parseWindow.ToggleMenuBar()
	case "set-background-color":
		parseWindow.SetBackgroundColour(application.NewRGBA(uint8(parseRequest.Red), uint8(parseRequest.Green), uint8(parseRequest.Blue), uint8(parseRequest.Alpha)))
	case "set-minimize-button-state":
		parseWindow.SetMinimiseButtonState(getButtonState(parseRequest.State))
	case "set-maximize-button-state":
		parseWindow.SetMaximiseButtonState(getButtonState(parseRequest.State))
	case "set-close-button-state":
		parseWindow.SetCloseButtonState(getButtonState(parseRequest.State))
	case "set-fullscreen-button-state":
		return desktop.WindowInfo{}, unavailable("fullscreen caption button is unsupported on Windows")
	case "flash":
		parseWindow.Flash(parseRequest.Enabled)
	case "set-content-protection":
		parseWindow.SetContentProtection(parseRequest.Enabled)
	case "zoom-in":
		parseWindow.ZoomIn()
	case "zoom-out":
		parseWindow.ZoomOut()
	case "set-zoom":
		if parseRequest.Zoom < 1 {
			return desktop.WindowInfo{}, invalid("Windows window zoom must be at least 1")
		}
		parseWindow.SetZoom(parseRequest.Zoom)
	case "zoom-reset":
		parseWindow.ZoomReset()
	default:
		return desktop.WindowInfo{}, invalid("unknown window action")
	}
	return windowInfo(parseWindow), nil
}

// PrintWindow opens Wails printing only after the separate portable feature gate succeeds.
func (nativeBackend) PrintWindow(parseContext context.Context) error {
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return parseErr
	}
	return parseWindow.Print()
}

// getButtonState converts a validated portable caption-button state.
func getButtonState(parseState string) application.ButtonState {
	switch parseState {
	case "disabled":
		return application.ButtonDisabled
	case "hidden":
		return application.ButtonHidden
	default:
		return application.ButtonEnabled
	}
}

// Screens returns typed monitor metadata without exposing Wails screen objects.
func (nativeBackend) Screens(parseContext context.Context) ([]desktop.ScreenInfo, error) {
	if parseErr := parseCaller(parseContext); parseErr != nil {
		return nil, parseErr
	}
	parseApp := application.Get()
	if parseApp == nil {
		return nil, unavailable("Wails application unavailable")
	}
	parseScreens := parseApp.Screen.GetAll()
	parseResult := make([]desktop.ScreenInfo, 0, len(parseScreens))
	for _, parseScreen := range parseScreens {
		parseResult = append(parseResult, desktop.ScreenInfo{ID: parseScreen.ID, Name: parseScreen.Name, Primary: parseScreen.IsPrimary, Scale: parseScreen.ScaleFactor, X: parseScreen.X, Y: parseScreen.Y, Width: parseScreen.Size.Width, Height: parseScreen.Size.Height, WorkArea: getScreenBounds(parseScreen.WorkArea), PhysicalBounds: getScreenBounds(parseScreen.PhysicalBounds), PhysicalWorkArea: getScreenBounds(parseScreen.PhysicalWorkArea), Rotation: parseScreen.Rotation})
	}
	return parseResult, nil
}

// getScreenBounds converts Wails geometry without exporting backend types.
func getScreenBounds(parseBounds application.Rect) desktop.ScreenBounds {
	return desktop.ScreenBounds{X: parseBounds.X, Y: parseBounds.Y, Width: parseBounds.Width, Height: parseBounds.Height}
}

// getCaller resolves the Wails RPC caller window and never falls back to focus.
func getCaller(parseContext context.Context) (application.Window, error) {
	if parseContext == nil {
		return nil, invalid("calling context required")
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return nil, parseErr
	}
	parseWindow, parseOK := parseContext.Value(application.WindowKey).(application.Window)
	if !parseOK || parseWindow == nil {
		return nil, unavailable("calling window unavailable")
	}
	return parseWindow, nil
}

// parseCaller validates a context before a non-window native operation.
func parseCaller(parseContext context.Context) error {
	_, parseErr := getCaller(parseContext)
	return parseErr
}

// windowInfo converts a Wails window to the portable contract.
func windowInfo(parseWindow application.Window) desktop.WindowInfo {
	parseX, parseY := parseWindow.Position()
	parseRelativeX, parseRelativeY := parseWindow.RelativePosition()
	return desktop.WindowInfo{ID: fmt.Sprint(parseWindow.ID()), Name: parseWindow.Name(), X: parseX, Y: parseY, RelativeX: parseRelativeX, RelativeY: parseRelativeY, Width: parseWindow.Width(), Height: parseWindow.Height(), Focused: parseWindow.IsFocused(), Minimised: parseWindow.IsMinimised(), Maximised: parseWindow.IsMaximised(), Fullscreen: parseWindow.IsFullscreen(), Visible: parseWindow.IsVisible(), Resizable: parseWindow.Resizable(), Zoom: parseWindow.GetZoom()}
}

// unavailable classifies an unavailable native operation.
func unavailable(parseMessage string) error {
	return &interop.Error{Op: "desktop", Code: interop.CodeUnavailable, Err: errors.New(parseMessage)}
}

// invalid classifies malformed native input.
func invalid(parseMessage string) error {
	return &interop.Error{Op: "desktop", Code: interop.CodeInvalid, Err: errors.New(parseMessage)}
}
