package i18n

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
)

func buildTestBundle() *Bundle {
	bundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	bundle.Register("en", Catalog{
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
	bundle.Register("fr", Catalog{
		"marketing": {
			"headline": Message{Text: "Bonjour {name}"},
			"cart": Message{PluralArg: "count", Plural: map[PluralCategory]string{
				PluralOne:   "{count} article",
				PluralOther: "{count} articles",
			}},
		},
	})
	bundle.Register("ar", Catalog{
		"marketing": {
			"headline": Message{Text: "مرحبا {name}"},
		},
	})
	return bundle
}

func TestBundleTranslateUsesLocaleFallbacks(t *testing.T) {
	bundle := buildTestBundle()

	translated := bundle.Translate("fr-CA", "marketing", "headline", Arguments{"name": "Cam"}, "en")
	if translated != "Bonjour Cam" {
		t.Fatalf("expected locale base fallback translation, got %q", translated)
	}

	fallback := bundle.Translate("es", "marketing", "headline", Arguments{"name": "Cam"}, "en")
	if fallback != "Hello Cam" {
		t.Fatalf("expected fallback locale translation, got %q", fallback)
	}
}

func TestBundleTranslateSupportsPluralAndSelect(t *testing.T) {
	bundle := buildTestBundle()

	one := bundle.Translate("en", "marketing", "cart", Arguments{"count": 1}, "en")
	if one != "1 item" {
		t.Fatalf("expected singular translation, got %q", one)
	}

	many := bundle.Translate("en", "marketing", "cart", Arguments{"count": 4}, "en")
	if many != "4 items" {
		t.Fatalf("expected plural translation, got %q", many)
	}

	formal := bundle.Translate("en", "marketing", "tone", Arguments{"tone": "formal"}, "en")
	if formal != "Welcome back" {
		t.Fatalf("expected select translation, got %q", formal)
	}
}

func TestFormatHelpers(t *testing.T) {
	formattedEN := FormatNumber("en-US", 1234.5)
	formattedFR := FormatNumber("fr-FR", 1234.5)
	if formattedEN == "" || formattedFR == "" {
		t.Fatal("expected formatted numbers")
	}
	if formattedEN == formattedFR {
		t.Fatalf("expected locale-aware number formatting to differ, got %q and %q", formattedEN, formattedFR)
	}

	date := time.Date(2026, time.March, 16, 10, 30, 0, 0, time.UTC)
	if got := FormatDate("en-US", date, DateOptions{Style: DateStyleLong}); got != "March 16, 2026" {
		t.Fatalf("expected english long date, got %q", got)
	}
	if got := FormatDate("fr-FR", date, DateOptions{Style: DateStyleShort}); got != "16/03/2026" {
		t.Fatalf("expected french short date, got %q", got)
	}
	if got := DirectionForLocale("ar-EG"); got != DirectionRTL {
		t.Fatalf("expected arabic locale to resolve rtl direction, got %q", got)
	}
}

func TestRouteHelpers(t *testing.T) {
	options := RouteOptions{SupportedLocales: []string{"en", "fr", "ar"}, DefaultLocale: "en", OmitDefaultPrefix: true}

	if got := PrefixPath("fr", "/pricing", options); got != "/fr/pricing" {
		t.Fatalf("expected prefixed path, got %q", got)
	}
	if got := PrefixPath("en", "/pricing", options); got != "/pricing" {
		t.Fatalf("expected default locale path without prefix, got %q", got)
	}

	resolved := ResolvePath("/fr/pricing?plan=team", options)
	if resolved.Locale != "fr" || resolved.BasePath != "/pricing" || resolved.LocalizedPath != "/fr/pricing?plan=team" || !resolved.PrefixPresent {
		t.Fatalf("unexpected resolved path: %+v", resolved)
	}
}

func TestSSRBootstrapRoundTrip(t *testing.T) {
	bundle := buildTestBundle()
	payload := bundle.ToSSRBootstrap(SSRBootstrapOptions{
		Locale:         "fr",
		FallbackLocale: "en",
	})
	if payload.Locale != "fr" || payload.FallbackLocale != "en" {
		t.Fatalf("unexpected bootstrap locale payload: %+v", payload)
	}
	if payload.Messages["fr"]["marketing.headline"].Text != "Bonjour {name}" {
		t.Fatalf("expected french message payload, got %+v", payload.Messages)
	}

	restored := BundleFromSSRBootstrap(payload)
	translated := restored.Translate("fr", "marketing", "headline", Arguments{"name": "Cam"}, payload.FallbackLocale)
	if translated != "Bonjour Cam" {
		t.Fatalf("expected bootstrap-restored translation, got %q", translated)
	}

	wrapped := ui.SSRBootstrap{I18n: payload}
	if wrapped.I18n.Direction == "" {
		t.Fatalf("expected direction to be set in payload: %+v", wrapped)
	}
}
