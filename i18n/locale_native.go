//go:build !js || !wasm

package i18n

// UseLocale creates a non-reactive LocaleState using the given options.
func UseLocale(parseLocaleOptions LocaleOptions) LocaleState {
	parseLocaleSupported := normalizeLocales(parseLocaleOptions.SupportedLocales)
	parseLocaleFallback := fallbackString(parseLocaleOptions.FallbackLocale, firstLocale(parseLocaleSupported))
	parseLocaleCurrent := chooseSupportedLocale(parseLocaleOptions.InitialLocale, parseLocaleSupported, parseLocaleFallback)
	if parseLocaleCurrent == "" {
		parseLocaleCurrent = parseLocaleFallback
	}
	return LocaleState{
		get: func() string { return parseLocaleCurrent },
		set: func(parseLocaleNext string) {
			parseLocaleCurrent = chooseSupportedLocale(parseLocaleNext, parseLocaleSupported, parseLocaleFallback)
		},
		direction: func() Direction { return DirectionForLocale(parseLocaleCurrent) },
		supported: func() []string { return append([]string(nil), parseLocaleSupported...) },
		fallback:  func() string { return parseLocaleFallback },
	}
}

func firstLocale(parseLocaleList []string) string {
	if len(parseLocaleList) == 0 {
		return ""
	}
	return parseLocaleList[0]
}
