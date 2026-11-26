//go:build windows

package wails

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/monstercameron/GoWebComponents/v8/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// NewNativeBackend returns the pinned Wails implementation of portable native APIs.
func NewNativeBackend() desktop.NativeBackend {
	return &nativeBackend{trayShortcuts: newNativeTrayShortcutState()}
}

type nativeBackend struct {
	trayShortcuts *nativeTrayShortcutState
	windows       *nativeWindowState
}

// Features reports capabilities implemented by this Wails adapter.
func (b nativeBackend) Features() []desktop.Feature {
	features := []desktop.Feature{
		desktop.Clipboard,
		desktop.MessageDialogs,
		desktop.WindowControls,
		desktop.WindowPrinting,
		desktop.WindowEvents,
		desktop.Screens,
		desktop.ScreenGeometry,
		desktop.RuntimeMenus,
		desktop.SystemTray,
		desktop.GlobalShortcuts,
		desktop.SystemEnvironment,
		desktop.ExternalURLs,
		desktop.Autostart,
		desktop.FileManager,
	}
	if b.windows != nil {
		features = append(features, desktop.ChildWindows)
	}
	return features
}

// ClipboardWrite writes text only when explicitly requested by the caller.
func (nativeBackend) ClipboardWrite(ctx context.Context, req desktop.ClipboardWriteRequest) error {
	if err := validateCaller(ctx); err != nil {
		return err
	}
	app := application.Get()
	if app == nil || !app.Clipboard.SetText(req.Text) {
		return unavailable("clipboard write failed")
	}
	return nil
}

// ClipboardRead reads text only when explicitly requested by the caller.
func (nativeBackend) ClipboardRead(ctx context.Context) (string, error) {
	if err := validateCaller(ctx); err != nil {
		return "", err
	}
	app := application.Get()
	if app == nil {
		return "", unavailable("Wails application unavailable")
	}
	text, ok := app.Clipboard.Text()
	if !ok {
		return "", unavailable("clipboard read failed")
	}
	return text, nil
}

// ShowMessage displays a caller-owned Wails dialog and returns its exact button label.
func (nativeBackend) ShowMessage(ctx context.Context, req desktop.MessageRequest) (desktop.MessageReply, error) {
	if err := validateWindowsMessageRequest(req); err != nil {
		return desktop.MessageReply{}, err
	}
	win, err := getCaller(ctx)
	if err != nil {
		return desktop.MessageReply{}, err
	}
	app := application.Get()
	if app == nil {
		return desktop.MessageReply{}, unavailable("Wails application unavailable")
	}
	dialog := app.Dialog.Info()
	switch req.Kind {
	case "question":
		dialog = app.Dialog.Question()
	case "warning":
		dialog = app.Dialog.Warning()
	case "error":
		dialog = app.Dialog.Error()
	}
	chosen := make(chan string, 1)
	dialog.SetTitle(req.Title).SetMessage(req.Message).AttachToWindow(win)
	buttons := req.Buttons
	if len(buttons) == 0 {
		if req.Kind == "question" {
			buttons = []string{"Yes", "No"}
		} else {
			buttons = []string{"Ok"}
		}
	}
	added := make(map[string]*application.Button, len(buttons))
	for _, label := range buttons {
		labelCopy := label
		button := dialog.AddButton(label)
		button.OnClick(func() {
			select {
			case chosen <- labelCopy:
			default:
			}
		})
		added[label] = button
	}
	defaultLabel := req.DefaultButton
	if defaultLabel == "" {
		defaultLabel = buttons[0]
	}
	if btn := added[defaultLabel]; btn != nil {
		dialog.SetDefaultButton(btn)
	}
	cancelLabel := req.CancelButton
	if cancelLabel == "" && req.Kind != "question" {
		cancelLabel = buttons[len(buttons)-1]
	}
	if cancelLabel != "" {
		if btn := added[cancelLabel]; btn != nil {
			dialog.SetCancelButton(btn)
		}
	}
	dialog.Show()
	select {
	case button := <-chosen:
		return desktop.MessageReply{Button: button}, nil
	default:
		return desktop.MessageReply{}, errors.New("native dialog button unavailable")
	}
}

// validateWindowsMessageRequest enforces Windows MessageBox and TaskDialog button rules.
// Wails v3.1.0 uses TaskDialogIndirect for question dialogs (supporting 1-3 custom buttons
// plus arbitrary default/cancel roles) and MessageBox for info/warning/error dialogs (Ok only).
func validateWindowsMessageRequest(req desktop.MessageRequest) error {
	if len(req.Buttons) > 3 {
		return invalid("Windows message dialogs support at most 3 buttons")
	}
	seen := make(map[string]bool, len(req.Buttons))
	for _, label := range req.Buttons {
		if strings.TrimSpace(label) == "" || len(label) > 64 || !utf8.ValidString(label) || strings.ContainsRune(label, 0) {
			return invalid("invalid Windows message button label")
		}
		if seen[label] {
			return invalid("duplicate Windows message button label")
		}
		seen[label] = true
	}
	buttons := req.Buttons
	switch req.Kind {
	case "info", "warning", "error":
		if len(buttons) == 0 {
			buttons = []string{"Ok"}
		}
		if len(buttons) != 1 || buttons[0] != "Ok" {
			return invalid("Windows message dialogs support only the Ok button for this kind")
		}
		if req.DefaultButton != "" && req.DefaultButton != "Ok" {
			return invalid("only Ok can be the default for this Windows message dialog")
		}
		if req.CancelButton != "" && req.CancelButton != "Ok" {
			return invalid("only Ok can cancel this Windows message dialog")
		}
	case "question":
		if len(buttons) == 0 {
			buttons = []string{"Yes", "No"}
		}
		if req.DefaultButton != "" && !hasButtonLabel(buttons, req.DefaultButton) {
			return invalid("default message button is not present")
		}
		if req.CancelButton != "" && !hasButtonLabel(buttons, req.CancelButton) {
			return invalid("cancel message button is not present")
		}
	default:
		return invalid("unknown message kind")
	}
	return nil
}

// hasButtonLabel reports whether target exists in labels.
func hasButtonLabel(labels []string, target string) bool {
	for _, label := range labels {
		if label == target {
			return true
		}
	}
	return false
}

// Window performs a caller-owned window inspection or control.
func (nativeBackend) Window(ctx context.Context, req desktop.WindowRequest) (desktop.WindowInfo, error) {
	win, err := getCaller(ctx)
	if err != nil {
		return desktop.WindowInfo{}, err
	}
	switch req.Action {
	case "info":
	case "show":
		win.Show()
	case "hide":
		win.Hide()
	case "close":
		info := buildWindowInfo(win)
		win.Close()
		return info, nil
	case "reload":
		win.Reload()
	case "force-reload":
		return desktop.WindowInfo{}, unavailable("force-reload is unsupported on Windows")
	case "snap-assist":
		win.SnapAssist()
	case "set-title":
		win.SetTitle(req.Title)
	case "set-screen":
		app := application.Get()
		if app == nil {
			return desktop.WindowInfo{}, unavailable("Wails application unavailable")
		}
		var targetScreen *application.Screen
		for _, candidate := range app.Screen.GetAll() {
			if candidate.ID == req.ScreenID {
				targetScreen = candidate
				break
			}
		}
		if targetScreen == nil {
			return desktop.WindowInfo{}, invalid("unknown screen id")
		}
		win.SetScreen(targetScreen)
	case "set-position":
		win.SetPosition(req.X, req.Y)
	case "set-relative-position":
		win.SetRelativePosition(req.X, req.Y)
	case "set-bounds":
		win.SetBounds(application.Rect{X: req.X, Y: req.Y, Width: req.Width, Height: req.Height})
	case "center":
		win.Center()
	case "enable-size-constraints":
		win.EnableSizeConstraints()
	case "disable-size-constraints":
		win.DisableSizeConstraints()
	case "resize":
		if req.Width <= 0 || req.Height <= 0 {
			return desktop.WindowInfo{}, invalid("window dimensions must be positive")
		}
		win.SetSize(req.Width, req.Height)
	case "set-min-size":
		win.SetMinSize(req.Width, req.Height)
	case "set-max-size":
		win.SetMaxSize(req.Width, req.Height)
	case "minimize":
		win.Minimise()
	case "unminimize":
		win.UnMinimise()
	case "maximize":
		win.Maximise()
	case "unmaximize":
		win.UnMaximise()
	case "toggle-maximize":
		win.ToggleMaximise()
	case "restore":
		win.Restore()
	case "fullscreen":
		win.Fullscreen()
	case "unfullscreen":
		win.UnFullscreen()
	case "toggle-fullscreen":
		win.ToggleFullscreen()
	case "focus":
		win.Focus()
	case "set-always-on-top":
		win.SetAlwaysOnTop(req.Enabled)
	case "set-resizable":
		win.SetResizable(req.Enabled)
	case "set-frameless":
		win.SetFrameless(req.Enabled)
	case "toggle-frameless":
		win.ToggleFrameless()
	case "show-menu-bar":
		win.ShowMenuBar()
	case "hide-menu-bar":
		win.HideMenuBar()
	case "toggle-menu-bar":
		win.ToggleMenuBar()
	case "set-background-color":
		win.SetBackgroundColour(application.NewRGBA(uint8(req.Red), uint8(req.Green), uint8(req.Blue), uint8(req.Alpha)))
	case "set-minimize-button-state":
		win.SetMinimiseButtonState(getButtonState(req.State))
	case "set-maximize-button-state":
		win.SetMaximiseButtonState(getButtonState(req.State))
	case "set-close-button-state":
		win.SetCloseButtonState(getButtonState(req.State))
	case "set-fullscreen-button-state":
		return desktop.WindowInfo{}, unavailable("fullscreen caption button is unsupported on Windows")
	case "flash":
		win.Flash(req.Enabled)
	case "set-content-protection":
		win.SetContentProtection(req.Enabled)
	case "set-ignore-mouse-events":
		win.SetIgnoreMouseEvents(req.Enabled)
	case "set-enabled":
		win.SetEnabled(req.Enabled)
	case "zoom-in":
		win.ZoomIn()
	case "zoom-out":
		win.ZoomOut()
	case "set-zoom":
		if req.Zoom < 0.25 || req.Zoom > 5 {
			return desktop.WindowInfo{}, invalid("window zoom out of bounds")
		}
		win.SetZoom(req.Zoom)
	case "zoom-reset":
		win.ZoomReset()
	default:
		return desktop.WindowInfo{}, invalid("unknown window action")
	}
	return buildWindowInfo(win), nil
}

// PrintWindow opens Wails printing only after the separate portable feature gate succeeds.
func (nativeBackend) PrintWindow(ctx context.Context) error {
	win, err := getCaller(ctx)
	if err != nil {
		return err
	}
	return win.Print()
}

// getButtonState converts a validated portable caption-button state.
func getButtonState(state string) application.ButtonState {
	switch state {
	case "disabled":
		return application.ButtonDisabled
	case "hidden":
		return application.ButtonHidden
	default:
		return application.ButtonEnabled
	}
}

// Screens returns typed monitor metadata without exposing Wails screen objects.
func (nativeBackend) Screens(ctx context.Context) ([]desktop.ScreenInfo, error) {
	if err := validateCaller(ctx); err != nil {
		return nil, err
	}
	app := application.Get()
	if app == nil {
		return nil, unavailable("Wails application unavailable")
	}
	screens := app.Screen.GetAll()
	result := make([]desktop.ScreenInfo, 0, len(screens))
	for _, screen := range screens {
		result = append(result, desktop.ScreenInfo{
			ID:               screen.ID,
			Name:             screen.Name,
			Primary:          screen.IsPrimary,
			Scale:            screen.ScaleFactor,
			X:                screen.X,
			Y:                screen.Y,
			Width:            screen.Size.Width,
			Height:           screen.Size.Height,
			WorkArea:         getScreenBounds(screen.WorkArea),
			PhysicalBounds:   getScreenBounds(screen.PhysicalBounds),
			PhysicalWorkArea: getScreenBounds(screen.PhysicalWorkArea),
			Rotation:         screen.Rotation,
		})
	}
	return result, nil
}

// getScreenBounds converts Wails geometry without exporting backend types.
func getScreenBounds(bounds application.Rect) desktop.ScreenBounds {
	return desktop.ScreenBounds{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: bounds.Height}
}

// getCaller resolves the Wails RPC caller window and never falls back to focus.
func getCaller(ctx context.Context) (application.Window, error) {
	if ctx == nil {
		return nil, invalid("calling context required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	win, ok := ctx.Value(application.WindowKey).(application.Window)
	if !ok || win == nil {
		return nil, unavailable("calling window unavailable")
	}
	return win, nil
}

// validateCaller verifies a caller context before a non-window native operation.
func validateCaller(ctx context.Context) error {
	_, err := getCaller(ctx)
	return err
}

// buildWindowInfo converts a Wails window to the portable contract.
func buildWindowInfo(win application.Window) desktop.WindowInfo {
	x, y := win.Position()
	relX, relY := win.RelativePosition()
	info := desktop.WindowInfo{
		ID:                fmt.Sprint(win.ID()),
		Name:              win.Name(),
		X:                 x,
		Y:                 y,
		RelativeX:         relX,
		RelativeY:         relY,
		Width:             win.Width(),
		Height:            win.Height(),
		Focused:           win.IsFocused(),
		Minimised:         win.IsMinimised(),
		Maximised:         win.IsMaximised(),
		Fullscreen:        win.IsFullscreen(),
		Visible:           win.IsVisible(),
		Resizable:         win.Resizable(),
		Zoom:              win.GetZoom(),
		IgnoreMouseEvents: win.IsIgnoreMouseEvents(),
	}
	if screen, err := win.GetScreen(); err == nil && screen != nil && screen.ID != "" {
		info.ScreenID = screen.ID
	}
	if borders := win.GetBorderSizes(); borders != nil && borders.Top >= 0 && borders.Left >= 0 && borders.Bottom >= 0 && borders.Right >= 0 {
		info.BorderSizes = &desktop.WindowBorderSizes{
			Top:    borders.Top,
			Left:   borders.Left,
			Bottom: borders.Bottom,
			Right:  borders.Right,
		}
	}
	return info
}

// unavailable classifies an unavailable native operation.
func unavailable(message string) error {
	return &interop.Error{Op: "desktop", Code: interop.CodeUnavailable, Err: errors.New(message)}
}

// invalid classifies malformed native input.
func invalid(message string) error {
	return &interop.Error{Op: "desktop", Code: interop.CodeInvalid, Err: errors.New(message)}
}
