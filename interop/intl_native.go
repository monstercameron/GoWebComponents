//go:build !js || !wasm
// +build !js !wasm

package interop

// IntlAvailable reports whether the browser Intl API is available. On native
// and SSR builds this always returns false; callers should fall back to the Go
// i18n formatters (e.g. i18n.FormatNumber / i18n.FormatDate). This bridge is
// browser-only by design.
func IntlAvailable() bool { return false }

// IntlFormatNumber is a non-browser stub that always returns an unavailable
// error. On native builds callers should fall back to the Go i18n formatters.
func IntlFormatNumber(parseLocale string, parseValue float64, parseOptions ...IntlNumberOptions) (string, error) {
	_ = parseLocale
	_ = parseValue
	return "", unavailable("IntlFormatNumber", "Intl.NumberFormat")
}

// IntlFormatDate is a non-browser stub that always returns an unavailable
// error. On native builds callers should fall back to the Go i18n formatters.
func IntlFormatDate(parseLocale string, parseUnixMillis int64, parseOptions ...IntlDateOptions) (string, error) {
	_ = parseLocale
	_ = parseUnixMillis
	return "", unavailable("IntlFormatDate", "Intl.DateTimeFormat")
}
