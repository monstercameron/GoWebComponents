package i18n

import (
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPluralCategoryFractionsAndEdgeValues pins CLDR behaviour for values the integer-only rules
// got wrong: a fractional count is never "one" in English/Russian/Polish, is "many" in Czech, and
// non-finite or huge values must not wrap through int conversion.
func TestPluralCategoryFractionsAndEdgeValues(parseT *testing.T) {
	parseCases := []struct {
		locale string
		value  float64
		want   PluralCategory
	}{
		{"en", 1, PluralOne},
		{"en", -1, PluralOne},
		{"en", 1.5, PluralOther},
		{"en", 0, PluralOther},
		{"en", 0.5, PluralOther},
		{"en", 2, PluralOther},
		{"en", math.NaN(), PluralOther},
		{"en", math.Inf(1), PluralOther},
		{"en", math.Inf(-1), PluralOther},
		{"en", 1e19, PluralOther},
		{"ru", 1.5, PluralOther},
		{"ru", 21, PluralOne},
		{"ru", 22, PluralFew},
		{"ru", 25, PluralMany},
		{"pl", 1.5, PluralOther},
		{"pl", 22, PluralFew},
		{"cs", 1.5, PluralMany},
		{"cs", 3, PluralFew},
		{"ar", 1.5, PluralOther},
		{"ar", 0, PluralZero},
		{"ar", 102, PluralOther},
		{"ar", 111, PluralMany},
		{"fr", 1.5, PluralOne},
		{"fr", 2, PluralOther},
		{"tr", 1, PluralOne},
		{"tr", 2, PluralOther},
		{"tr", 1.5, PluralOther},
		{"ja", 1, PluralOther},
	}
	for _, parseCase := range parseCases {
		if parseGot := pluralCategoryForLocale(parseCase.locale, parseCase.value); parseGot != parseCase.want {
			parseT.Errorf("%s %v = %s, want %s", parseCase.locale, parseCase.value, parseGot, parseCase.want)
		}
	}
}

// TestFormatNumberLargeAndNonFinite pins that integral values beyond int64 do not gain a bogus
// ".00" and that NaN/Inf do not panic.
func TestFormatNumberLargeAndNonFinite(parseT *testing.T) {
	if parseGot := FormatNumber("en", 1e19); strings.Contains(parseGot, ".") {
		parseT.Errorf("FormatNumber(1e19) = %q, want no fraction digits", parseGot)
	}
	for _, parseValue := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_ = FormatNumber("en", parseValue)
	}
	if parseGot := FormatNumber("en", 1234.5); parseGot != "1,234.50" && parseGot != "1,234.5" {
		parseT.Errorf("FormatNumber(1234.5) = %q", parseGot)
	}
}

// TestFormatRelativeTimeBoundariesAndZeroTime pins unit promotion at rounding boundaries (59.7s
// is "1 minute", not "60 seconds") and that the zero time.Time -- whose Sub saturates to
// MinInt64 -- does not render a negative count.
func TestFormatRelativeTimeBoundariesAndZeroTime(parseT *testing.T) {
	parseBase := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	parseCases := []struct {
		delta time.Duration
		want  string
	}{
		{-59700 * time.Millisecond, "1 minute ago"},
		{-59*time.Minute - 40*time.Second, "1 hour ago"},
		{-23*time.Hour - 40*time.Minute, "1 day ago"},
		{-6*24*time.Hour - 14*time.Hour, "1 week ago"},
		{-350 * 24 * time.Hour, "1 year ago"},
		{90 * time.Second, "in 2 minutes"},
		{-30 * time.Second, "30 seconds ago"},
	}
	for _, parseCase := range parseCases {
		if parseGot := FormatRelativeTime("en", parseBase.Add(parseCase.delta), parseBase); parseGot != parseCase.want {
			parseT.Errorf("delta %v = %q, want %q", parseCase.delta, parseGot, parseCase.want)
		}
	}
	parseZero := FormatRelativeTime("en", time.Time{}, parseBase)
	if strings.Contains(parseZero, "-") || !strings.HasSuffix(parseZero, "years ago") {
		parseT.Errorf("zero time rendered %q, want a positive count of years", parseZero)
	}
}

// TestPrefixPathCannotProduceProtocolRelativeURL pins that a path beginning with "//" (or the
// browser-equivalent "/\") is collapsed to a single leading slash. With OmitDefaultPrefix the
// default locale returns the path unchanged, so "//evil.example" came back as a protocol-relative
// URL that an open-redirect sink would follow off-site.
func TestPrefixPathCannotProduceProtocolRelativeURL(parseT *testing.T) {
	parseOptions := RouteOptions{SupportedLocales: []string{"en", "fr"}, DefaultLocale: "en", OmitDefaultPrefix: true}
	for _, parseInput := range []string{"//evil.example/x", `/\evil.example`, "///evil.example", `\\evil.example`} {
		parseGot := PrefixPath("en", parseInput, parseOptions)
		if strings.HasPrefix(parseGot, "//") || strings.HasPrefix(parseGot, `/\`) {
			parseT.Errorf("PrefixPath(%q) = %q is protocol-relative", parseInput, parseGot)
		}
		parseResolved := ResolvePath(parseInput, parseOptions)
		if strings.HasPrefix(parseResolved.LocalizedPath, "//") || strings.HasPrefix(parseResolved.BasePath, "//") {
			parseT.Errorf("ResolvePath(%q) = %#v is protocol-relative", parseInput, parseResolved)
		}
	}
}

// TestPrefixPathKeepsQueryTrailingSlash pins that trailing slashes belonging to the query string
// or fragment (a "next=/" redirect target) are not trimmed as if they were path slashes.
func TestPrefixPathKeepsQueryTrailingSlash(parseT *testing.T) {
	parseOptions := RouteOptions{SupportedLocales: []string{"en", "fr"}, DefaultLocale: "en"}
	if parseGot := PrefixPath("fr", "/login?next=/", parseOptions); parseGot != "/fr/login?next=/" {
		parseT.Errorf("PrefixPath = %q, want /fr/login?next=/", parseGot)
	}
	if parseGot := ResolvePath("/fr/login?next=/", parseOptions).LocalizedPath; parseGot != "/fr/login?next=/" {
		parseT.Errorf("ResolvePath localized = %q, want /fr/login?next=/", parseGot)
	}
}

// TestLazyBundleLoadsEachLocaleOnceUnderConcurrency pins the LocaleLoader contract ("called at
// most once per locale"): concurrent first callers used to each invoke the loader.
func TestLazyBundleLoadsEachLocaleOnceUnderConcurrency(parseT *testing.T) {
	var parseCalls atomic.Int32
	parseRelease := make(chan struct{})
	parseLazy := NewLazyBundle(NewBundle(), func(parseLocale string) (Catalog, error) {
		parseCalls.Add(1)
		<-parseRelease
		return Catalog{"app": NamespaceCatalog{"hi": Message{Text: "bonjour"}}}, nil
	})

	var parseGroup sync.WaitGroup
	for range 8 {
		parseGroup.Add(1)
		go func() {
			defer parseGroup.Done()
			if parseErr := parseLazy.EnsureLocale("fr"); parseErr != nil {
				parseT.Error(parseErr)
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(parseRelease)
	parseGroup.Wait()

	if parseCalls.Load() != 1 {
		parseT.Fatalf("loader invoked %d times, want 1", parseCalls.Load())
	}
	if parseGot := parseLazy.Bundle().Translate("fr", "app", "hi", nil, ""); parseGot != "bonjour" {
		parseT.Fatalf("translation = %q", parseGot)
	}
}

// TestLazyBundleFailedLoadIsRetriedAndEmptyLocaleIsNoop covers the error path and a blank locale.
func TestLazyBundleFailedLoadIsRetriedAndEmptyLocaleIsNoop(parseT *testing.T) {
	var parseCalls atomic.Int32
	parseLazy := NewLazyBundle(NewBundle(), func(string) (Catalog, error) {
		if parseCalls.Add(1) == 1 {
			return nil, errTestLoad
		}
		return Catalog{"a": NamespaceCatalog{"k": Message{Text: "v"}}}, nil
	})
	if parseErr := parseLazy.EnsureLocale("de"); parseErr == nil {
		parseT.Fatal("first load should fail")
	}
	if parseLazy.Loaded("de") {
		parseT.Fatal("failed locale marked loaded")
	}
	if parseErr := parseLazy.EnsureLocale("de"); parseErr != nil {
		parseT.Fatalf("retry: %v", parseErr)
	}
	before := parseCalls.Load()
	if parseErr := parseLazy.EnsureLocale("  "); parseErr != nil || parseCalls.Load() != before {
		parseT.Fatalf("blank locale should be a no-op (err=%v, loader calls %d->%d)", parseErr, before, parseCalls.Load())
	}
}

type testLoadError string

func (parseE testLoadError) Error() string { return string(parseE) }

const errTestLoad = testLoadError("load failed")

// TestLocaleFallbackWalksScriptSubtags pins that zh-Hant-TW falls back to zh-Hant before zh, so a
// Traditional-script catalog is not skipped in favour of Simplified.
func TestLocaleFallbackWalksScriptSubtags(parseT *testing.T) {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en"})
	parseBundle.RegisterNamespace("zh", "a", NamespaceCatalog{"k": Message{Text: "simplified"}})
	parseBundle.RegisterNamespace("zh-Hant", "a", NamespaceCatalog{"k": Message{Text: "traditional"}})
	if parseGot := parseBundle.Translate("zh-Hant-TW", "a", "k", nil, ""); parseGot != "traditional" {
		parseT.Errorf("zh-Hant-TW resolved to %q, want traditional", parseGot)
	}
	if parseGot := parseBundle.Translate("zh-CN", "a", "k", nil, ""); parseGot != "simplified" {
		parseT.Errorf("zh-CN resolved to %q, want simplified", parseGot)
	}
}

// reviewNilStringer dereferences its receiver in String.
type reviewNilStringer struct{ text string }

func (parseS *reviewNilStringer) String() string { return parseS.text }

// TestInterpolateTypedNilStringerDoesNotPanic pins that a typed-nil fmt.Stringer argument renders
// like fmt would ("<nil>") instead of panicking during translation.
func TestInterpolateTypedNilStringerDoesNotPanic(parseT *testing.T) {
	var parseNil *reviewNilStringer
	parseGot := interpolateTemplate("hi {who}", Arguments{"who": parseNil})
	if !strings.HasPrefix(parseGot, "hi ") {
		parseT.Fatalf("got %q", parseGot)
	}
}

// TestDirectionForLocaleUsesScript pins script-aware direction: Latin-script Kurdish is LTR,
// Sorani (ckb) and Arabic-script variants are RTL, as are Yiddish and Dhivehi.
func TestDirectionForLocaleUsesScript(parseT *testing.T) {
	for parseLocale, parseWant := range map[string]Direction{
		"ar": DirectionRTL, "ar-EG": DirectionRTL, "he": DirectionRTL, "fa": DirectionRTL, "ur-PK": DirectionRTL,
		"ckb": DirectionRTL, "yi": DirectionRTL, "dv": DirectionRTL, "az-Arab": DirectionRTL, "ku-Arab": DirectionRTL,
		"ku": DirectionLTR, "ku-Latn": DirectionLTR, "en": DirectionLTR, "az": DirectionLTR, "zh-Hant": DirectionLTR,
		"": DirectionLTR,
	} {
		if parseGot := DirectionForLocale(parseLocale); parseGot != parseWant {
			parseT.Errorf("DirectionForLocale(%q) = %s, want %s", parseLocale, parseGot, parseWant)
		}
	}
}
