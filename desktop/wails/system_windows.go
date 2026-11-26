//go:build windows

package wails

import (
	"context"
	"runtime"

	"github.com/monstercameron/GoWebComponents/v8/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SystemEnvironment returns typed Windows environment information from Wails.
func (nativeBackend) SystemEnvironment(ctx context.Context) (desktop.SystemEnvironmentInfo, error) {
	if err := validateCaller(ctx); err != nil {
		return desktop.SystemEnvironmentInfo{}, err
	}
	app := application.Get()
	if app == nil {
		return desktop.SystemEnvironmentInfo{}, unavailable("Wails application unavailable")
	}
	theme := "light"
	if app.Env.IsDarkMode() {
		theme = "dark"
	}
	info := desktop.SystemEnvironmentInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Theme:        theme,
		AccentColor:  app.Env.GetAccentColor(),
	}
	if envInfo := app.Env.Info(); envInfo.OSInfo != nil {
		info.OSName = envInfo.OSInfo.Name
		info.OSVersion = envInfo.OSInfo.Version
		info.OSBranding = envInfo.OSInfo.Branding
	}
	return info, nil
}

// OpenExternalURL opens one prevalidated web URL in the default browser.
func (nativeBackend) OpenExternalURL(ctx context.Context, req desktop.ExternalURLOpenRequest) error {
	if err := validateCaller(ctx); err != nil {
		return err
	}
	app := application.Get()
	if app == nil {
		return unavailable("Wails application unavailable")
	}
	return app.Browser.OpenURL(req.URL)
}

// RevealPath reveals one prevalidated existing local path in Windows File Explorer.
func (nativeBackend) RevealPath(ctx context.Context, req desktop.FileManagerRevealRequest) error {
	if err := validateCaller(ctx); err != nil {
		return err
	}
	app := application.Get()
	if app == nil {
		return unavailable("Wails application unavailable")
	}
	return app.Env.OpenFileManager(req.Path, req.SelectFile)
}

// AutostartStatus inspects the current application's Wails autostart registration.
func (nativeBackend) AutostartStatus(ctx context.Context) (desktop.AutostartStatus, error) {
	if err := validateCaller(ctx); err != nil {
		return desktop.AutostartStatus{}, err
	}
	app := application.Get()
	if app == nil || app.Autostart == nil {
		return desktop.AutostartStatus{}, unavailable("Wails autostart unavailable")
	}
	status, err := app.Autostart.Status()
	if err != nil {
		return desktop.AutostartStatus{}, err
	}
	return desktop.AutostartStatus{
		Enabled:  status.Enabled,
		Path:     status.Path,
		Strategy: string(status.Strategy),
	}, nil
}

// AutostartEnable explicitly enables Wails autostart with default options.
func (nativeBackend) AutostartEnable(ctx context.Context) error {
	if err := validateCaller(ctx); err != nil {
		return err
	}
	app := application.Get()
	if app == nil || app.Autostart == nil {
		return unavailable("Wails autostart unavailable")
	}
	return app.Autostart.Enable()
}

// AutostartDisable explicitly disables Wails autostart.
func (nativeBackend) AutostartDisable(ctx context.Context) error {
	if err := validateCaller(ctx); err != nil {
		return err
	}
	app := application.Get()
	if app == nil || app.Autostart == nil {
		return unavailable("Wails autostart unavailable")
	}
	return app.Autostart.Disable()
}
