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
	defaultSettingsSectionID    = settingsSectionProfile
)

func normalizeSettingsSectionID(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	trimmed = strings.TrimPrefix(trimmed, "#")
	switch trimmed {
	case settingsSectionProfile, settingsSectionTone, settingsSectionPrompt, settingsSectionIntelligence, settingsSectionSpeech, settingsSectionMemories, settingsSectionLanguage:
		return trimmed
	default:
		return ""
	}
}

func isSettingsRoute(path string) bool {
	return strings.TrimSpace(path) == settingsRoutePath
}

func buildSettingsRoute(section string) string {
	normalized := normalizeSettingsSectionID(section)
	if normalized == "" {
		normalized = defaultSettingsSectionID
	}
	values := url.Values{}
	values.Set(settingsPanelQueryKey, normalized)
	return settingsRoutePath + "?" + values.Encode()
}

func currentSettingsPanelRouteID() string {
	return normalizeSettingsSectionID(router.UseQuery().Get(settingsPanelQueryKey))
}

func currentLocationPathSearch() string {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return ""
	}
	location := window.Get("location")
	if !location.Truthy() {
		return ""
	}
	return strings.TrimSpace(location.Get("pathname").String()) + strings.TrimSpace(location.Get("search").String())
}

func buildSettingsReturnRoute(path, section string) string {
	base := strings.TrimSpace(path)
	if base == "" {
		base = chatRouteRoot
	}
	normalized := normalizeSettingsSectionID(section)
	if normalized == "" {
		return base
	}
	return strings.TrimPrefix(base, "#") + "#" + normalized
}

func replaceSettingsSectionHash(section string) {
	normalized := normalizeSettingsSectionID(section)
	if normalized == "" {
		return
	}
	window := js.Global().Get("window")
	if !window.Truthy() {
		return
	}
	location := window.Get("location")
	if !location.Truthy() {
		return
	}
	history := window.Get("history")
	url := buildSettingsReturnRoute(currentLocationPathSearch(), normalized)
	if history.Truthy() && history.Get("replaceState").Type() == js.TypeFunction {
		history.Call("replaceState", nil, "", url)
		return
	}
	location.Set("hash", normalized)
}

func scrollSettingsSectionIntoView(section string) {
	normalized := normalizeSettingsSectionID(section)
	if normalized == "" {
		return
	}
	document := js.Global().Get("document")
	if !document.Truthy() || document.Get("getElementById").Type() != js.TypeFunction {
		return
	}
	target := document.Call("getElementById", normalized)
	if !target.Truthy() || target.Get("scrollIntoView").Type() != js.TypeFunction {
		return
	}
	target.Call("scrollIntoView", map[string]interface{}{
		"behavior": "smooth",
		"block":    "start",
	})
}
