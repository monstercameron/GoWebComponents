//go:build windows

package wails

import (
	"context"
	"errors"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/desktop"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// NewNativeBackend returns the pinned Wails implementation of portable native APIs.
func NewNativeBackend() desktop.NativeBackend { return nativeBackend{} }

type nativeBackend struct{}

// Features reports capabilities implemented by this Wails adapter.
func (nativeBackend) Features() []desktop.Feature {
	return []desktop.Feature{desktop.Clipboard, desktop.MessageDialogs, desktop.WindowControls, desktop.Screens}
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
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return desktop.MessageReply{}, parseErr
	}
	parseApp := application.Get()
	if parseApp == nil {
		return desktop.MessageReply{}, unavailable("Wails application unavailable")
	}
	parseDialog := parseApp.Dialog.Info()
	if parseRequest.Kind == "question" {
		parseDialog = parseApp.Dialog.Question()
	}
	parseChosen := make(chan string, 1)
	parseDialog.SetTitle(parseRequest.Title).SetMessage(parseRequest.Message).AttachToWindow(parseWindow)
	if parseRequest.Kind == "info" {
		parseOK := parseDialog.AddButton("Ok")
		parseOK.OnClick(func() { parseChosen <- "Ok" })
		parseDialog.SetDefaultButton(parseOK).SetCancelButton(parseOK)
	} else {
		parseYes := parseDialog.AddButton("Yes")
		parseNo := parseDialog.AddButton("No")
		parseYes.OnClick(func() { parseChosen <- "Yes" })
		parseNo.OnClick(func() { parseChosen <- "No" })
		parseDialog.SetDefaultButton(parseYes).SetCancelButton(parseNo)
	}
	parseDialog.Show()
	select {
	case parseButton := <-parseChosen:
		return desktop.MessageReply{Button: parseButton}, nil
	default:
		return desktop.MessageReply{}, errors.New("native dialog button unavailable")
	}
}

// Window performs a caller-owned window inspection or control.
func (nativeBackend) Window(parseContext context.Context, parseRequest desktop.WindowRequest) (desktop.WindowInfo, error) {
	parseWindow, parseErr := getCaller(parseContext)
	if parseErr != nil {
		return desktop.WindowInfo{}, parseErr
	}
	switch parseRequest.Action {
	case "info":
	case "resize":
		if parseRequest.Width <= 0 || parseRequest.Height <= 0 {
			return desktop.WindowInfo{}, invalid("window dimensions must be positive")
		}
		parseWindow.SetSize(parseRequest.Width, parseRequest.Height)
	case "maximize":
		parseWindow.Maximise()
	case "restore":
		parseWindow.Restore()
	case "fullscreen":
		parseWindow.Fullscreen()
	case "unfullscreen":
		parseWindow.UnFullscreen()
	default:
		return desktop.WindowInfo{}, invalid("unknown window action")
	}
	return windowInfo(parseWindow), nil
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
		parseResult = append(parseResult, desktop.ScreenInfo{ID: parseScreen.ID, Name: parseScreen.Name, Primary: parseScreen.IsPrimary, Scale: parseScreen.ScaleFactor, X: parseScreen.X, Y: parseScreen.Y, Width: parseScreen.Size.Width, Height: parseScreen.Size.Height})
	}
	return parseResult, nil
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
	return desktop.WindowInfo{ID: fmt.Sprint(parseWindow.ID()), Name: parseWindow.Name(), Width: parseWindow.Width(), Height: parseWindow.Height(), Maximised: parseWindow.IsMaximised(), Fullscreen: parseWindow.IsFullscreen()}
}

// unavailable classifies an unavailable native operation.
func unavailable(parseMessage string) error {
	return &interop.Error{Op: "desktop", Code: interop.CodeUnavailable, Err: errors.New(parseMessage)}
}

// invalid classifies malformed native input.
func invalid(parseMessage string) error {
	return &interop.Error{Op: "desktop", Code: interop.CodeInvalid, Err: errors.New(parseMessage)}
}
