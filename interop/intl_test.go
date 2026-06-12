package interop

import (
	"testing"
)

// groupingPtr returns a *bool for the UseGrouping option in tests.
func groupingPtr(parseValue bool) *bool { return &parseValue }

// TestIntlNumberOptionsKeyDistinct asserts that distinct IntlNumberOptions
// configurations produce distinct cache keys. This is the core cache-correctness
// invariant: formatters with different options must never share a cache entry.
func TestIntlNumberOptionsKeyDistinct(parseT *testing.T) {
	parseLocale := "en-US"
	parseCases := []struct {
		name string
		opts IntlNumberOptions
	}{
		{name: "zero", opts: IntlNumberOptions{}},
		{name: "decimal-no-group", opts: IntlNumberOptions{Style: "decimal", UseGrouping: groupingPtr(false)}},
		{name: "decimal-group", opts: IntlNumberOptions{Style: "decimal", UseGrouping: groupingPtr(true)}},
		{name: "currency-USD", opts: IntlNumberOptions{Style: "currency", Currency: "USD", UseGrouping: groupingPtr(true)}},
		{name: "currency-EUR", opts: IntlNumberOptions{Style: "currency", Currency: "EUR", UseGrouping: groupingPtr(true)}},
		{name: "percent", opts: IntlNumberOptions{Style: "percent"}},
		{name: "minFD-2", opts: IntlNumberOptions{MinimumFractionDigits: 2}},
		{name: "maxFD-4", opts: IntlNumberOptions{MaximumFractionDigits: 4}},
		{name: "minFD-2-maxFD-4", opts: IntlNumberOptions{MinimumFractionDigits: 2, MaximumFractionDigits: 4}},
	}

	parseSeen := map[string]string{}
	for _, parseCase := range parseCases {
		parseKey := numberOptionsKey(parseLocale, parseCase.opts)
		if parseKey == "" {
			parseT.Errorf("case %q: numberOptionsKey returned empty string", parseCase.name)
			continue
		}
		if parseConflict, parseExists := parseSeen[parseKey]; parseExists {
			parseT.Errorf("cache key collision: case %q and %q share key %q", parseCase.name, parseConflict, parseKey)
		}
		parseSeen[parseKey] = parseCase.name
	}
}

// TestIntlNumberOptionsKeyIdentical asserts that identical IntlNumberOptions
// values produce identical cache keys. This ensures the cache is actually hit on
// repeated calls with the same parameters.
func TestIntlNumberOptionsKeyIdentical(parseT *testing.T) {
	parseLocale := "de-DE"
	parseOptsA := IntlNumberOptions{Style: "currency", Currency: "EUR", MinimumFractionDigits: 2, MaximumFractionDigits: 2, UseGrouping: groupingPtr(true)}
	parseOptsB := IntlNumberOptions{Style: "currency", Currency: "EUR", MinimumFractionDigits: 2, MaximumFractionDigits: 2, UseGrouping: groupingPtr(true)}
	parseKeyA := numberOptionsKey(parseLocale, parseOptsA)
	parseKeyB := numberOptionsKey(parseLocale, parseOptsB)
	if parseKeyA != parseKeyB {
		parseT.Fatalf("identical options produced different keys: %q vs %q", parseKeyA, parseKeyB)
	}
}

// TestIntlNumberOptionsKeyLocaleDistinct asserts that the same options under
// different locales produce distinct keys.
func TestIntlNumberOptionsKeyLocaleDistinct(parseT *testing.T) {
	parseOpts := IntlNumberOptions{Style: "decimal", UseGrouping: groupingPtr(true)}
	parseKeyEN := numberOptionsKey("en-US", parseOpts)
	parseKeyDE := numberOptionsKey("de-DE", parseOpts)
	if parseKeyEN == parseKeyDE {
		parseT.Fatalf("different locales produced identical keys: %q", parseKeyEN)
	}
}

// TestIntlDateOptionsKeyDistinct asserts that distinct IntlDateOptions
// configurations produce distinct cache keys.
func TestIntlDateOptionsKeyDistinct(parseT *testing.T) {
	parseLocale := "en-US"
	parseCases := []struct {
		name string
		opts IntlDateOptions
	}{
		{name: "zero", opts: IntlDateOptions{}},
		{name: "dateStyle-short", opts: IntlDateOptions{DateStyle: "short"}},
		{name: "dateStyle-full", opts: IntlDateOptions{DateStyle: "full"}},
		{name: "timeStyle-short", opts: IntlDateOptions{TimeStyle: "short"}},
		{name: "dateStyle-long-timeStyle-long", opts: IntlDateOptions{DateStyle: "long", TimeStyle: "long"}},
		{name: "tz-NYC", opts: IntlDateOptions{TimeZone: "America/New_York"}},
		{name: "tz-Tokyo", opts: IntlDateOptions{TimeZone: "Asia/Tokyo"}},
		{name: "dateStyle-medium-tz-UTC", opts: IntlDateOptions{DateStyle: "medium", TimeZone: "UTC"}},
	}

	parseSeen := map[string]string{}
	for _, parseCase := range parseCases {
		parseKey := dateOptionsKey(parseLocale, parseCase.opts)
		if parseKey == "" {
			parseT.Errorf("case %q: dateOptionsKey returned empty string", parseCase.name)
			continue
		}
		if parseConflict, parseExists := parseSeen[parseKey]; parseExists {
			parseT.Errorf("cache key collision: case %q and %q share key %q", parseCase.name, parseConflict, parseKey)
		}
		parseSeen[parseKey] = parseCase.name
	}
}

// TestIntlDateOptionsKeyIdentical asserts that identical IntlDateOptions values
// produce identical cache keys.
func TestIntlDateOptionsKeyIdentical(parseT *testing.T) {
	parseLocale := "ja-JP"
	parseOptsA := IntlDateOptions{DateStyle: "full", TimeStyle: "long", TimeZone: "Asia/Tokyo"}
	parseOptsB := IntlDateOptions{DateStyle: "full", TimeStyle: "long", TimeZone: "Asia/Tokyo"}
	parseKeyA := dateOptionsKey(parseLocale, parseOptsA)
	parseKeyB := dateOptionsKey(parseLocale, parseOptsB)
	if parseKeyA != parseKeyB {
		parseT.Fatalf("identical options produced different keys: %q vs %q", parseKeyA, parseKeyB)
	}
}

// TestIntlDateOptionsKeyLocaleDistinct asserts that the same IntlDateOptions
// under different locales produce distinct keys.
func TestIntlDateOptionsKeyLocaleDistinct(parseT *testing.T) {
	parseOpts := IntlDateOptions{DateStyle: "short"}
	parseKeyEN := dateOptionsKey("en-US", parseOpts)
	parseKeyFR := dateOptionsKey("fr-FR", parseOpts)
	if parseKeyEN == parseKeyFR {
		parseT.Fatalf("different locales produced identical date keys: %q", parseKeyEN)
	}
}
