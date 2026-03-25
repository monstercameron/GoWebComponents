//go:build js && wasm
// +build js,wasm

package i18n

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/ui"
)

// UseLocale creates a reactive LocaleState backed by component state, optionally detecting the browser locale.
func UseLocale(options LocaleOptions) LocaleState {
	supported := normalizeLocales(options.SupportedLocales)
	fallback := fallbackString(options.FallbackLocale, firstLocale(supported))
	initial := chooseSupportedLocale(options.InitialLocale, supported, fallback)
	if initial == "" && options.DetectBrowser {
		initial = chooseSupportedLocale(browserLocale(), supported, fallback)
	}
	if initial == "" {
		initial = fallback
	}
	state := ui.UseState(initial)
	loaded := ui.UseRef(false)

	ui.UseEffect(func() func() {
		if loaded.Get() {
			return nil
		}
		loaded.Set(true)
		if options.PersistenceKey == "" {
			return nil
		}
		stored := readPersistedLocale(options.PersistenceKey)
		resolved := chooseSupportedLocale(stored, supported, fallback)
		if resolved != "" && resolved != state.Get() {
			state.Set(resolved)
		}
		return nil
	}, options.PersistenceKey)

	ui.UseEffect(func() func() {
		current := chooseSupportedLocale(state.Get(), supported, fallback)
		if options.PersistenceKey != "" {
			writePersistedLocale(options.PersistenceKey, current)
		}
		if options.OnChange != nil {
			options.OnChange(current)
		}
		return nil
	}, state.Get(), options.PersistenceKey)

	return LocaleState{
		get: func() string {
			return chooseSupportedLocale(state.Get(), supported, fallback)
		},
		set: func(next string) {
			resolved := chooseSupportedLocale(next, supported, fallback)
			if resolved == "" {
				resolved = fallback
			}
			state.Set(resolved)
		},
		direction: func() Direction {
			return DirectionForLocale(state.Get())
		},
		supported: func() []string {
			return append([]string(nil), supported...)
		},
		fallback: func() string { return fallback },
	}
}

func readPersistedLocale(key string) string {
	storage := browserStorage()
	if !storage.Truthy() {
		return ""
	}
	return strings.TrimSpace(storage.Call("getItem", key).String())
}

func writePersistedLocale(key string, locale string) {
	storage := browserStorage()
	if !storage.Truthy() {
		return
	}
	storage.Call("setItem", key, locale)
}

func browserStorage() js.Value {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return js.Undefined()
	}
	return window.Get("localStorage")
}

func browserLocale() string {
	navigator := js.Global().Get("navigator")
	if !navigator.Truthy() {
		return ""
	}
	return strings.TrimSpace(navigator.Get("language").String())
}

func firstLocale(locales []string) string {
	if len(locales) == 0 {
		return ""
	}
	return locales[0]
}
