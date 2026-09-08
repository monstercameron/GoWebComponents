package services

import (
	"fmt"
	"strconv"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v6/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// buildAPIContextItems defines real native menu controls without synthetic activation.
func buildAPIContextItems(parseMenu *application.Menu, parseService *APIService) {
	parseMenu.Add("Record context selection").OnClick(func(parseContext *application.Context) {
		parseService.appendAPIResult(contracts.APIResult{ID: "context-select", Outcome: "completed", Detail: "Native menu callback; context=" + parseContext.ContextMenuData()})
	})
	parseMenu.AddCheckbox("Checked option", false).OnClick(func(parseContext *application.Context) {
		parseService.appendAPIResult(contracts.APIResult{ID: "context-check", Outcome: "completed", Detail: fmt.Sprintf("checked=%t", parseContext.IsChecked())})
	})
	parseMenu.AddSeparator()
	for _, parseLabel := range []string{"Radio Alpha", "Radio Beta"} {
		parseMenu.AddRadio(parseLabel, parseLabel == "Radio Alpha").OnClick(func(parseContext *application.Context) {
			parseService.appendAPIResult(contracts.APIResult{ID: "context-radio", Outcome: "completed", Detail: parseContext.ClickedMenuItem().Label()})
		})
	}
	parseMenu.AddSeparator()
	parseMenu.AddSubmenu("More actions").Add("Record submenu selection").OnClick(func(parseContext *application.Context) {
		parseService.appendAPIResult(contracts.APIResult{ID: "context-submenu", Outcome: "completed", Detail: "Native submenu callback; context=" + parseContext.ContextMenuData()})
	})
	parseMenu.Add("Disabled action (must not run)").SetEnabled(false)
}

// InstallAPIMenus installs the context menu and returns the explicit Windows menu bar.
func InstallAPIMenus(parseApp *application.App, parseService *APIService) *application.Menu {
	parseContextMenu := parseApp.ContextMenu.New()
	buildAPIContextItems(parseContextMenu.Menu, parseService)
	parseApp.ContextMenu.Add("api-tester", parseContextMenu)
	parseMenu := application.NewMenu()
	parseTests := parseMenu.AddSubmenu("Tests")
	parseTests.Add("Record menu action").SetAccelerator("Ctrl+Shift+K").OnClick(func(*application.Context) {
		parseService.appendAPIResult(contracts.APIResult{ID: "menu-action", Outcome: "completed", Detail: "Native menu/accelerator callback (Ctrl+Shift+K)"})
	})
	parseTests.Add("Open second tester window").OnClick(func(*application.Context) {
		// A second window shares only the service report and saved counter, not UI state.
		parseApp.Window.NewWithOptions(application.WebviewWindowOptions{
			Title: "Windows API Lab — second window", Width: 1024, Height: 820,
			URL: "/#/tester", Windows: application.WindowsWindow{Menu: parseMenu},
			KeyBindings: APIKeyBindings(parseService),
		})
	})
	buildAPIEditMenu(parseMenu)
	return parseMenu
}

// APIKeyBindings supplies per-window keys without registering system-wide shortcuts.
func APIKeyBindings(parseService *APIService) map[string]func(application.Window) {
	parseBindings := map[string]func(application.Window){}
	if parseService == nil {
		return parseBindings
	}
	if parseService.Policy.Allows(desktop.NativeMenus) {
		parseBindings["F8"] = func(parseWindow application.Window) {
			parseService.appendAPIResult(contracts.APIResult{ID: "shortcut", Outcome: "completed", Detail: "Native F8 window keybinding", WindowID: strconv.FormatUint(uint64(parseWindow.ID()), 10)})
		}
	}
	// Exiting fullscreen belongs to window controls even when native menus are disabled.
	if parseService.Policy.Allows(desktop.WindowControls) {
		parseBindings["Escape"] = func(parseWindow application.Window) {
			if parseWindow.IsFullscreen() {
				parseWindow.UnFullscreen()
			}
		}
	}
	return parseBindings
}
