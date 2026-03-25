//go:build !js || !wasm
// +build !js !wasm

package i18n

// UseLocale creates a non-reactive LocaleState using the given options.
func UseLocale(options LocaleOptions) LocaleState {
	supported := normalizeLocales(options.SupportedLocales)
	fallback := fallbackString(options.FallbackLocale, firstLocale(supported))
	current := chooseSupportedLocale(options.InitialLocale, supported, fallback)
	if current == "" {
		current = fallback
	}
	return LocaleState{
		get:       func() string { return current },
		set:       func(next string) { current = chooseSupportedLocale(next, supported, fallback) },
		direction: func() Direction { return DirectionForLocale(current) },
		supported: func() []string { return append([]string(nil), supported...) },
		fallback:  func() string { return fallback },
	}
}

func firstLocale(locales []string) string {
	if len(locales) == 0 {
		return ""
	}
	return locales[0]
}
