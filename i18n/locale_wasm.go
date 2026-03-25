//go:build js && wasm
// +build js,wasm

package i18n

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/ui"
)

// UseLocale creates a reactive LocaleState backed by component state, optionally detecting the browser locale.
func UseLocale(parseOptions LocaleOptions) LocaleState {
	parseSupported := normalizeLocales(parseOptions.SupportedLocales)
	parseFallback := fallbackString(parseOptions.FallbackLocale, firstLocale(parseSupported))
	parseInitial := chooseSupportedLocale(parseOptions.InitialLocale, parseSupported, parseFallback)
	if parseInitial == "" && parseOptions.DetectBrowser {
		parseInitial = chooseSupportedLocale(browserLocale(), parseSupported, parseFallback)
	}
	if parseInitial == "" {
		parseInitial = parseFallback
	}
	parseState := ui.UseState(parseInitial)
	parseLoaded := ui.UseRef(false)

	ui.UseEffect(func() func() {
		if parseLoaded.Get() {
			return nil
		}
		parseLoaded.Set(true)
		if parseOptions.PersistenceKey == "" {
			return nil
		}
		parseStored := readPersistedLocale(parseOptions.PersistenceKey)
		parseResolved := chooseSupportedLocale(parseStored, parseSupported, parseFallback)
		if parseResolved != "" && parseResolved != parseState.Get() {
			parseState.Set(parseResolved)
		}
		return nil
	}, parseOptions.PersistenceKey)

	ui.UseEffect(func() func() {
		parseCurrent := chooseSupportedLocale(parseState.Get(), parseSupported, parseFallback)
		if parseOptions.PersistenceKey != "" {
			writePersistedLocale(parseOptions.PersistenceKey, parseCurrent)
		}
		if parseOptions.OnChange != nil {
			parseOptions.OnChange(parseCurrent)
		}
		return nil
	}, parseState.Get(), parseOptions.PersistenceKey)

	return LocaleState{
		get: func() string {
			return chooseSupportedLocale(parseState.Get(), parseSupported, parseFallback)
		},
		set: func(parseNext string) {
			parseResolved2 := chooseSupportedLocale(parseNext, parseSupported, parseFallback)
			if parseResolved2 == "" {
				parseResolved2 = parseFallback
			}
			parseState.Set(parseResolved2)
		},
		direction: func() Direction {
			return DirectionForLocale(parseState.Get())
		},
		supported: func() []string {
			return append([]string(nil), parseSupported...)
		},
		fallback: func() string { return parseFallback },
	}
}

func readPersistedLocale(parseKey string) string {
	parseStorage := browserStorage()
	if !parseStorage.Truthy() {
		return ""
	}
	return strings.TrimSpace(parseStorage.Call("getItem", parseKey).String())
}

func writePersistedLocale(parseKey string, parseLocale string) {
	parseStorage := browserStorage()
	if !parseStorage.Truthy() {
		return
	}
	parseStorage.Call("setItem", parseKey, parseLocale)
}

func browserStorage() js.Value {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return js.Undefined()
	}
	return parseWindow.Get("localStorage")
}

func browserLocale() string {
	parseNavigator := js.Global().Get("navigator")
	if !parseNavigator.Truthy() {
		return ""
	}
	return strings.TrimSpace(parseNavigator.Get("language").String())
}

func firstLocale(parseLocales []string) string {
	if len(parseLocales) == 0 {
		return ""
	}
	return parseLocales[0]
}
