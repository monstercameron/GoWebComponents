package interop

import (
	"fmt"
	"strconv"
)

// IntlNumberOptions configures a browser Intl.NumberFormat instance. Fields map
// directly to the JS options bag passed to the Intl.NumberFormat constructor.
// The zero value produces decimal formatting with grouping enabled.
type IntlNumberOptions struct {
	// Style is the formatting style: "decimal" (default), "currency", or
	// "percent".
	Style string
	// Currency is the ISO 4217 currency code (e.g. "USD"). Required when Style
	// is "currency".
	Currency string
	// MinimumFractionDigits sets the minimum number of fraction digits.
	MinimumFractionDigits int
	// MaximumFractionDigits sets the maximum number of fraction digits. A zero
	// value leaves the option unset (JS default applies).
	MaximumFractionDigits int
	// UseGrouping controls digit grouping separators (e.g. thousands commas).
	// Matches the JS useGrouping option.
	UseGrouping bool
}

// IntlDateOptions configures a browser Intl.DateTimeFormat instance. Fields map
// directly to the JS options bag passed to the Intl.DateTimeFormat constructor.
// Leaving DateStyle and TimeStyle empty lets the browser apply its defaults.
type IntlDateOptions struct {
	// DateStyle is the date display style: "full", "long", "medium", or
	// "short". Empty omits the option.
	DateStyle string
	// TimeStyle is the time display style: "full", "long", "medium", or
	// "short". Empty omits the option.
	TimeStyle string
	// TimeZone is the IANA time-zone name (e.g. "America/New_York"). Empty
	// omits the option, letting the browser use its local zone.
	TimeZone string
}

// numberOptionsKey returns a stable string key for the given locale and options
// struct. Distinct option combinations produce distinct keys; identical ones
// produce identical keys. This is the cache-correctness invariant for the WASM
// formatter cache.
func numberOptionsKey(parseLocale string, parseOpts IntlNumberOptions) string {
	return fmt.Sprintf("%s|style=%s|currency=%s|minFD=%s|maxFD=%s|grouping=%s",
		parseLocale,
		parseOpts.Style,
		parseOpts.Currency,
		strconv.Itoa(parseOpts.MinimumFractionDigits),
		strconv.Itoa(parseOpts.MaximumFractionDigits),
		strconv.FormatBool(parseOpts.UseGrouping),
	)
}

// dateOptionsKey returns a stable string key for the given locale and options
// struct. Distinct option combinations produce distinct keys; identical ones
// produce identical keys. This is the cache-correctness invariant for the WASM
// formatter cache.
func dateOptionsKey(parseLocale string, parseOpts IntlDateOptions) string {
	return fmt.Sprintf("%s|dateStyle=%s|timeStyle=%s|tz=%s",
		parseLocale,
		parseOpts.DateStyle,
		parseOpts.TimeStyle,
		parseOpts.TimeZone,
	)
}
