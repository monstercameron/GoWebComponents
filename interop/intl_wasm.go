//go:build js && wasm

package interop

import (
	"errors"
	"fmt"
	"sync"
	"syscall/js"
)

// intlFormatterCache caches Intl.NumberFormat and Intl.DateTimeFormat js.Value
// instances keyed by a stable locale+options signature. Formatter construction
// is the documented JS performance trap; reusing instances avoids it.
var intlFormatterCache = map[string]js.Value{}

// intlCacheMu guards intlFormatterCache against concurrent WASM goroutines.
var intlCacheMu sync.Mutex

// IntlAvailable reports whether the browser's Intl API is present and usable.
// It returns true when js.Global().Get("Intl") is truthy (not undefined/null).
func IntlAvailable() bool {
	parseIntl := js.Global().Get("Intl")
	return parseIntl.Truthy()
}

// IntlFormatNumber formats parseValue using the browser's Intl.NumberFormat for
// the given locale and optional options. Formatter instances are cached by
// locale+options signature to avoid the per-call construction overhead. Returns
// an unavailable error when the Intl API is absent, or a wrapped error when the
// locale/options are invalid (JS constructor throws).
func IntlFormatNumber(parseLocale string, parseValue float64, parseOptions ...IntlNumberOptions) (string, error) {
	if !IntlAvailable() {
		return "", unavailable("IntlFormatNumber", "Intl.NumberFormat")
	}
	parseOpts := IntlNumberOptions{}
	if len(parseOptions) > 0 {
		parseOpts = parseOptions[0]
	}
	parseCacheKey := numberOptionsKey(parseLocale, parseOpts)

	intlCacheMu.Lock()
	parseFmt, parseHit := intlFormatterCache[parseCacheKey]
	intlCacheMu.Unlock()

	if !parseHit {
		parseBuildErr := (error)(nil)
		parseFmt, parseBuildErr = intlBuildNumberFormatter(parseLocale, parseOpts)
		if parseBuildErr != nil {
			return "", parseBuildErr
		}
		intlCacheMu.Lock()
		intlFormatterCache[parseCacheKey] = parseFmt
		intlCacheMu.Unlock()
	}

	parseResult, parseCallErr := intlCallFormat(parseFmt, js.ValueOf(parseValue))
	if parseCallErr != nil {
		return "", wrapError("IntlFormatNumber", "Intl.NumberFormat.format", parseCallErr.Code, parseCallErr.Err)
	}
	return parseResult, nil
}

// IntlFormatDate formats parseUnixMillis using the browser's Intl.DateTimeFormat
// for the given locale and optional options. parseUnixMillis is milliseconds
// since the Unix epoch (compatible with JS Date(ms)). Formatter instances are
// cached by locale+options signature. Returns an unavailable error when the Intl
// API is absent, or a wrapped error when the locale/options are invalid.
func IntlFormatDate(parseLocale string, parseUnixMillis int64, parseOptions ...IntlDateOptions) (string, error) {
	if !IntlAvailable() {
		return "", unavailable("IntlFormatDate", "Intl.DateTimeFormat")
	}
	parseOpts := IntlDateOptions{}
	if len(parseOptions) > 0 {
		parseOpts = parseOptions[0]
	}
	parseCacheKey := dateOptionsKey(parseLocale, parseOpts)

	intlCacheMu.Lock()
	parseFmt, parseHit := intlFormatterCache[parseCacheKey]
	intlCacheMu.Unlock()

	if !parseHit {
		parseBuildErr := (error)(nil)
		parseFmt, parseBuildErr = intlBuildDateFormatter(parseLocale, parseOpts)
		if parseBuildErr != nil {
			return "", parseBuildErr
		}
		intlCacheMu.Lock()
		intlFormatterCache[parseCacheKey] = parseFmt
		intlCacheMu.Unlock()
	}

	parseDate := js.Global().Get("Date").New(parseUnixMillis)
	parseResult, parseCallErr := intlCallFormat(parseFmt, parseDate)
	if parseCallErr != nil {
		return "", wrapError("IntlFormatDate", "Intl.DateTimeFormat.format", parseCallErr.Code, parseCallErr.Err)
	}
	return parseResult, nil
}

// intlBuildNumberFormatter constructs a new Intl.NumberFormat js.Value from the
// given locale and options. Panics from an invalid locale or options bag are
// caught and returned as a structured error.
func intlBuildNumberFormatter(parseLocale string, parseOpts IntlNumberOptions) (parseFmt js.Value, parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseFmt = js.Undefined()
			parseErr = wrapError("IntlFormatNumber", "Intl.NumberFormat",
				CodeInvalid, fmt.Errorf("JS exception constructing formatter: %v", parseRecovered))
		}
	}()

	parseOptsObj := js.Global().Get("Object").New()
	if parseOpts.Style != "" {
		parseOptsObj.Set("style", parseOpts.Style)
	}
	if parseOpts.Currency != "" {
		parseOptsObj.Set("currency", parseOpts.Currency)
	}
	if parseOpts.MinimumFractionDigits != 0 {
		parseOptsObj.Set("minimumFractionDigits", parseOpts.MinimumFractionDigits)
	}
	if parseOpts.MaximumFractionDigits != 0 {
		parseOptsObj.Set("maximumFractionDigits", parseOpts.MaximumFractionDigits)
	}
	if parseOpts.UseGrouping != nil {
		parseOptsObj.Set("useGrouping", *parseOpts.UseGrouping)
	}

	parseFmt = js.Global().Get("Intl").Get("NumberFormat").New(parseLocale, parseOptsObj)
	return parseFmt, nil
}

// intlBuildDateFormatter constructs a new Intl.DateTimeFormat js.Value from the
// given locale and options. Panics from an invalid locale or options bag are
// caught and returned as a structured error.
func intlBuildDateFormatter(parseLocale string, parseOpts IntlDateOptions) (parseFmt js.Value, parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseFmt = js.Undefined()
			parseErr = wrapError("IntlFormatDate", "Intl.DateTimeFormat",
				CodeInvalid, fmt.Errorf("JS exception constructing formatter: %v", parseRecovered))
		}
	}()

	parseOptsObj := js.Global().Get("Object").New()
	if parseOpts.DateStyle != "" {
		parseOptsObj.Set("dateStyle", parseOpts.DateStyle)
	}
	if parseOpts.TimeStyle != "" {
		parseOptsObj.Set("timeStyle", parseOpts.TimeStyle)
	}
	if parseOpts.TimeZone != "" {
		parseOptsObj.Set("timeZone", parseOpts.TimeZone)
	}

	parseFmt = js.Global().Get("Intl").Get("DateTimeFormat").New(parseLocale, parseOptsObj)
	return parseFmt, nil
}

// intlCallFormat invokes the .format() method on a cached Intl formatter
// js.Value with parseArg as its sole argument. Panics from the JS call (e.g.
// an invalid Date) are caught and returned as a structured interop error.
func intlCallFormat(parseFmt js.Value, parseArg js.Value) (parseResult string, parseErr *Error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseResult = ""
			parseErr = &Error{
				Op:     "intlCallFormat",
				Target: "Intl.*.format",
				Code:   CodeInvalid,
				Err:    fmt.Errorf("JS exception calling format: %v", parseRecovered),
			}
		}
	}()
	parseStr := parseFmt.Call("format", parseArg)
	if parseStr.IsNull() || parseStr.IsUndefined() {
		return "", &Error{
			Op:     "intlCallFormat",
			Target: "Intl.*.format",
			Code:   CodeInvalid,
			Err:    errors.New("format returned null or undefined"),
		}
	}
	return parseStr.String(), nil
}
