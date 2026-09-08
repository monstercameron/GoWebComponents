package i18n

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func buildTestBundle() *Bundle {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	parseBundle.Register("en", Catalog{
		"marketing": {
			"headline": Message{Text: "Hello {name}"},
			"cart": Message{PluralArg: "count", Plural: map[PluralCategory]string{
				PluralOne:   "{count} item",
				PluralOther: "{count} items",
			}},
			"tone": Message{SelectArg: "tone", Select: map[string]string{
				"formal": "Welcome back",
				"casual": "Good to see you",
				"other":  "Hello again",
			}},
		},
	})
	parseBundle.Register("fr", Catalog{
		"marketing": {
			"headline": Message{Text: "Bonjour {name}"},
			"cart": Message{PluralArg: "count", Plural: map[PluralCategory]string{
				PluralOne:   "{count} article",
				PluralOther: "{count} articles",
			}},
		},
	})
	parseBundle.Register("ar", Catalog{
		"marketing": {
			"headline": Message{Text: "مرحبا {name}"},
		},
	})
	return parseBundle
}

func TestBundleTranslateUsesLocaleFallbacks(parseT *testing.T) {
	parseBundle := buildTestBundle()

	parseTranslated := parseBundle.Translate("fr-CA", "marketing", "headline", Arguments{"name": "Cam"}, "en")
	if parseTranslated != "Bonjour Cam" {
		parseT.Fatalf("expected locale base fallback translation, got %q", parseTranslated)
	}

	parseFallback := parseBundle.Translate("es", "marketing", "headline", Arguments{"name": "Cam"}, "en")
	if parseFallback != "Hello Cam" {
		parseT.Fatalf("expected fallback locale translation, got %q", parseFallback)
	}
}

func TestBundleTranslateSupportsPluralAndSelect(parseT *testing.T) {
	parseBundle := buildTestBundle()

	parseOne := parseBundle.Translate("en", "marketing", "cart", Arguments{"count": 1}, "en")
	if parseOne != "1 item" {
		parseT.Fatalf("expected singular translation, got %q", parseOne)
	}

	parseMany := parseBundle.Translate("en", "marketing", "cart", Arguments{"count": 4}, "en")
	if parseMany != "4 items" {
		parseT.Fatalf("expected plural translation, got %q", parseMany)
	}

	parseFormal := parseBundle.Translate("en", "marketing", "tone", Arguments{"tone": "formal"}, "en")
	if parseFormal != "Welcome back" {
		parseT.Fatalf("expected select translation, got %q", parseFormal)
	}
}

func TestFormatHelpers(parseT *testing.T) {
	parseFormattedEN := FormatNumber("en-US", 1234.5)
	parseFormattedFR := FormatNumber("fr-FR", 1234.5)
	if parseFormattedEN == "" || parseFormattedFR == "" {
		parseT.Fatal("expected formatted numbers")
	}
	if parseFormattedEN == parseFormattedFR {
		parseT.Fatalf("expected locale-aware number formatting to differ, got %q and %q", parseFormattedEN, parseFormattedFR)
	}

	parseDate := time.Date(2026, time.March, 16, 10, 30, 0, 0, time.UTC)
	if parseGot := FormatDate("en-US", parseDate, DateOptions{Style: DateStyleLong}); parseGot != "March 16, 2026" {
		parseT.Fatalf("expected english long date, got %q", parseGot)
	}
	if parseGot2 := FormatDate("fr-FR", parseDate, DateOptions{Style: DateStyleShort}); parseGot2 != "16/03/2026" {
		parseT.Fatalf("expected french short date, got %q", parseGot2)
	}
	if parseGot3 := DirectionForLocale("ar-EG"); parseGot3 != DirectionRTL {
		parseT.Fatalf("expected arabic locale to resolve rtl direction, got %q", parseGot3)
	}
}

func TestRouteHelpers(parseT *testing.T) {
	parseOptions := RouteOptions{SupportedLocales: []string{"en", "fr", "ar"}, DefaultLocale: "en", OmitDefaultPrefix: true}

	if parseGot := PrefixPath("fr", "/pricing", parseOptions); parseGot != "/fr/pricing" {
		parseT.Fatalf("expected prefixed path, got %q", parseGot)
	}
	if parseGot2 := PrefixPath("en", "/pricing", parseOptions); parseGot2 != "/pricing" {
		parseT.Fatalf("expected default locale path without prefix, got %q", parseGot2)
	}

	parseResolved := ResolvePath("/fr/pricing?plan=team", parseOptions)
	if parseResolved.Locale != "fr" || parseResolved.BasePath != "/pricing" || parseResolved.LocalizedPath != "/fr/pricing?plan=team" || !parseResolved.PrefixPresent {
		parseT.Fatalf("unexpected resolved path: %+v", parseResolved)
	}

	parseResolved = ResolvePath("/en-US/dashboard", RouteOptions{SupportedLocales: []string{"en", "fr"}, DefaultLocale: "en"})
	if parseResolved.Locale != "en" || parseResolved.BasePath != "/dashboard" || !parseResolved.PrefixPresent {
		parseT.Fatalf("unexpected BCP-47 resolved path: %+v", parseResolved)
	}
}

func TestSSRBootstrapRoundTrip(parseT *testing.T) {
	parseBundle := buildTestBundle()
	parsePayload := parseBundle.ToSSRBootstrap(SSRBootstrapOptions{
		Locale:         "fr",
		FallbackLocale: "en",
	})
	if parsePayload.Locale != "fr" || parsePayload.FallbackLocale != "en" {
		parseT.Fatalf("unexpected bootstrap locale payload: %+v", parsePayload)
	}
	if parsePayload.Messages["fr"]["marketing.headline"].Text != "Bonjour {name}" {
		parseT.Fatalf("expected french message payload, got %+v", parsePayload.Messages)
	}

	parseRestored := BundleFromSSRBootstrap(parsePayload)
	parseTranslated := parseRestored.Translate("fr", "marketing", "headline", Arguments{"name": "Cam"}, parsePayload.FallbackLocale)
	if parseTranslated != "Bonjour Cam" {
		parseT.Fatalf("expected bootstrap-restored translation, got %q", parseTranslated)
	}

	parseWrapped := ui.SSRBootstrap{I18n: parsePayload}
	if parseWrapped.I18n.Direction == "" {
		parseT.Fatalf("expected direction to be set in payload: %+v", parseWrapped)
	}
}
