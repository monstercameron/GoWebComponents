//go:build !js || !wasm
// +build !js !wasm

package i18n

// UseLocale creates a non-reactive LocaleState using the given options.
func UseLocale(parseOptions LocaleOptions) LocaleState {
	parseSupported := normalizeLocales(parseOptions.SupportedLocales)
	parseFallback := fallbackString(parseOptions.FallbackLocale, firstLocale(parseSupported))
	parseCurrent := chooseSupportedLocale(parseOptions.InitialLocale, parseSupported, parseFallback)
	if parseCurrent == "" {
		parseCurrent = parseFallback
	}
	return LocaleState{
		get:       func() string { return parseCurrent },
		set:       func(parseNext string) { parseCurrent = chooseSupportedLocale(parseNext, parseSupported, parseFallback) },
		direction: func() Direction { return DirectionForLocale(parseCurrent) },
		supported: func() []string { return append([]string(nil), parseSupported...) },
		fallback:  func() string { return parseFallback },
	}
}

func firstLocale(parseLocales []string) string {
	if len(parseLocales) == 0 {
		return ""
	}
	return parseLocales[0]
}
