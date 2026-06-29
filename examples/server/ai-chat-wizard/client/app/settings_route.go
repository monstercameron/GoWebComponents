//go:build js && wasm

package app

import (
	"net/url"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/router"
)

const (
	settingsRoutePath           = "/app/settings"
	settingsPanelQueryKey       = "panel"
	settingsSectionProfile      = "settings-profile"
	settingsSectionTone         = "settings-tone"
	settingsSectionPrompt       = "settings-prompt"
	settingsSectionIntelligence = "settings-intelligence"
	settingsSectionSpeech       = "settings-speech"
	settingsSectionMemories     = "settings-memories"
	settingsSectionLanguage     = "settings-language"
	settingsSectionBilling      = "settings-billing"
	settingsSectionSecurity     = "settings-security"
	defaultSettingsSectionID    = settingsSectionProfile
)

func parseNormalizeSettingsSectionID(parseRaw string) string {
	parseTrimmed := strings.ToLower(strings.TrimSpace(parseRaw))
	parseTrimmed = strings.TrimPrefix(parseTrimmed, "#")
	switch parseTrimmed {
	case settingsSectionProfile, settingsSectionTone, settingsSectionPrompt, settingsSectionIntelligence, settingsSectionSpeech, settingsSectionMemories, settingsSectionLanguage, settingsSectionBilling, settingsSectionSecurity:
		return parseTrimmed
	default:
		return ""
	}
}

func isSettingsRoute(parsePath string) bool {
	return strings.TrimSpace(parsePath) == settingsRoutePath
}

func buildSettingsRoute(parseSection string) string {
	parseNormalized := parseNormalizeSettingsSectionID(parseSection)
	if parseNormalized == "" {
		parseNormalized = defaultSettingsSectionID
	}
	parseValues := url.Values{}
	parseValues.Set(settingsPanelQueryKey, parseNormalized)
	return settingsRoutePath + "?" + parseValues.Encode()
}

func parseCurrentSettingsPanelRouteID() string {
	return parseNormalizeSettingsSectionID(router.UseQuery().Get(settingsPanelQueryKey))
}

func parseCurrentLocationPathSearch() string {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return ""
	}
	parseLocation := parseWindow.Get("location")
	if !parseLocation.Truthy() {
		return ""
	}
	return strings.TrimSpace(parseLocation.Get("pathname").String()) + strings.TrimSpace(parseLocation.Get("search").String())
}

func buildSettingsReturnRoute(parsePath, parseSection string) string {
	parseBase := strings.TrimSpace(parsePath)
	if parseBase == "" {
		parseBase = chatRouteRoot
	}
	parseNormalized := parseNormalizeSettingsSectionID(parseSection)
	if parseNormalized == "" {
		return parseBase
	}
	return strings.TrimPrefix(parseBase, "#") + "#" + parseNormalized
}

func parseScrollSettingsSectionIntoView(parseSection string) {
	parseNormalized := parseNormalizeSettingsSectionID(parseSection)
	if parseNormalized == "" {
		return
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() || parseDocument.Get("getElementById").Type() != js.TypeFunction {
		return
	}
	parseTarget := parseDocument.Call("getElementById", parseNormalized)
	if !parseTarget.Truthy() || parseTarget.Get("scrollIntoView").Type() != js.TypeFunction {
		return
	}
	parseTarget.Call("scrollIntoView", map[string]interface{}{
		"behavior": "smooth",
		"block":    "start",
	})
}
