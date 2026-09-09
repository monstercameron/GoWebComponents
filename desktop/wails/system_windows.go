//go:build windows

package wails

import (
	"context"
	"runtime"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SystemEnvironment returns typed Windows environment information from Wails.
func (nativeBackend) SystemEnvironment(parseContext context.Context) (desktop.SystemEnvironmentInfo, error) {
	if parseErr := parseCaller(parseContext); parseErr != nil {
		return desktop.SystemEnvironmentInfo{}, parseErr
	}
	parseApp := application.Get()
	if parseApp == nil {
		return desktop.SystemEnvironmentInfo{}, unavailable("Wails application unavailable")
	}
	parseTheme := "light"
	if parseApp.Env.IsDarkMode() {
		parseTheme = "dark"
	}
	return desktop.SystemEnvironmentInfo{OS: runtime.GOOS, Architecture: runtime.GOARCH, Theme: parseTheme, AccentColor: parseApp.Env.GetAccentColor()}, nil
}

// OpenExternalURL opens one prevalidated web URL in the default browser.
func (nativeBackend) OpenExternalURL(parseContext context.Context, parseRequest desktop.ExternalURLOpenRequest) error {
	if parseErr := parseCaller(parseContext); parseErr != nil {
		return parseErr
	}
	parseApp := application.Get()
	if parseApp == nil {
		return unavailable("Wails application unavailable")
	}
	return parseApp.Browser.OpenURL(parseRequest.URL)
}

// RevealPath reveals one prevalidated existing local path in Windows File Explorer.
func (nativeBackend) RevealPath(parseContext context.Context, parseRequest desktop.FileManagerRevealRequest) error {
	if parseErr := parseCaller(parseContext); parseErr != nil {
		return parseErr
	}
	parseApp := application.Get()
	if parseApp == nil {
		return unavailable("Wails application unavailable")
	}
	return parseApp.Env.OpenFileManager(parseRequest.Path, parseRequest.SelectFile)
}

// AutostartStatus inspects the current application's Wails autostart registration.
func (nativeBackend) AutostartStatus(parseContext context.Context) (desktop.AutostartStatus, error) {
	if parseErr := parseCaller(parseContext); parseErr != nil {
		return desktop.AutostartStatus{}, parseErr
	}
	parseApp := application.Get()
	if parseApp == nil || parseApp.Autostart == nil {
		return desktop.AutostartStatus{}, unavailable("Wails autostart unavailable")
	}
	parseStatus, parseErr := parseApp.Autostart.Status()
	if parseErr != nil {
		return desktop.AutostartStatus{}, parseErr
	}
	return desktop.AutostartStatus{Enabled: parseStatus.Enabled, Path: parseStatus.Path, Strategy: string(parseStatus.Strategy)}, nil
}

// AutostartEnable explicitly enables Wails autostart with default options.
func (nativeBackend) AutostartEnable(parseContext context.Context) error {
	if parseErr := parseCaller(parseContext); parseErr != nil {
		return parseErr
	}
	parseApp := application.Get()
	if parseApp == nil || parseApp.Autostart == nil {
		return unavailable("Wails autostart unavailable")
	}
	return parseApp.Autostart.Enable()
}

// AutostartDisable explicitly disables Wails autostart.
func (nativeBackend) AutostartDisable(parseContext context.Context) error {
	if parseErr := parseCaller(parseContext); parseErr != nil {
		return parseErr
	}
	parseApp := application.Get()
	if parseApp == nil || parseApp.Autostart == nil {
		return unavailable("Wails autostart unavailable")
	}
	return parseApp.Autostart.Disable()
}
