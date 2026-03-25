package i18n

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
)

type testStringer string

func (s testStringer) String() string { return string(s) }

func TestRegisterAndTranslateMissingBranches(t *testing.T) {
	var nilBundle *Bundle
	nilBundle.Register("en", Catalog{
		"common": {"greeting": {Text: "Hello"}},
	})
	if got := nilBundle.Translate("en", "common", "greeting", nil, "en"); got != "common.greeting" {
		t.Fatalf("expected nil bundle to use missing-text fallback, got %q", got)
	}

	bundle := &Bundle{}
	bundle.Register("   ", Catalog{"common": {"greeting": {Text: "ignored"}}})
	if len(bundle.catalogs) != 0 {
		t.Fatalf("expected blank locale registration to be ignored, got %+v", bundle.catalogs)
	}

	sourceCatalog := Catalog{
		"common": {"greeting": {Text: "Hello"}},
	}
	bundle.Register(" EN_us ", sourceCatalog)
	if bundle.defaultLocale != "en-US" || bundle.fallbackLocale != "en-US" {
		t.Fatalf("expected default/fallback locale to initialize from first register, got default=%q fallback=%q", bundle.defaultLocale, bundle.fallbackLocale)
	}

	// Ensure Register clones message values instead of aliasing caller maps.
	sourceCatalog["common"]["greeting"] = Message{Text: "Mutated"}
	if got := bundle.Translate("en-US", "common", "greeting", nil, "en-US"); got != "Hello" {
		t.Fatalf("expected registered message clone to remain stable, got %q", got)
	}

	bundle.onMissing = func(locale, namespace, key string) string {
		return strings.ToUpper(locale) + ":" + namespace + "." + key
	}
	if got := bundle.Translate("fr", "common", "missing", nil, "en-US"); got != "FR:common.missing" {
		t.Fatalf("expected custom missing handler response, got %q", got)
	}
}

func TestResolveTemplateAndInterpolationBranches(t *testing.T) {
	if got := resolveTemplate("en", Message{
		SelectArg: "tone",
		Select:    map[string]string{"other": "Hello"},
	}, Arguments{}); got != "Hello" {
		t.Fatalf("expected select branch to use other fallback, got %q", got)
	}

	if got := resolveTemplate("en", Message{
		SelectArg: "tone",
		Select:    map[string]string{"formal": "Welcome"},
		Default:   "Default",
	}, Arguments{"tone": "casual"}); got != "Default" {
		t.Fatalf("expected select branch to use default fallback, got %q", got)
	}

	if got := resolveTemplate("ar", Message{
		PluralArg: "count",
		Plural:    map[PluralCategory]string{PluralOther: "many"},
	}, Arguments{"count": 2}); got != "many" {
		t.Fatalf("expected plural branch to use other fallback, got %q", got)
	}

	if got := resolveTemplate("en", Message{
		PluralArg: "count",
		Plural:    map[PluralCategory]string{PluralOne: "one"},
		Default:   "plural-default",
	}, Arguments{"count": 3}); got != "plural-default" {
		t.Fatalf("expected plural branch to use default fallback, got %q", got)
	}

	if got := resolveTemplate("en", Message{Text: "Text", Default: "Default"}, nil); got != "Text" {
		t.Fatalf("expected text fallback to win when present, got %q", got)
	}
	if got := resolveTemplate("en", Message{Default: "OnlyDefault"}, nil); got != "OnlyDefault" {
		t.Fatalf("expected default fallback when no other templates exist, got %q", got)
	}

	ts := time.Date(2026, time.March, 25, 12, 34, 56, 0, time.UTC)
	if got := stringifyArgument(testStringer("cam")); got != "cam" {
		t.Fatalf("expected fmt.Stringer branch, got %q", got)
	}
	if got := stringifyArgument(ts); got != ts.String() {
		t.Fatalf("expected time.Time to follow fmt.Stringer branch, got %q", got)
	}
	if got := interpolateTemplate("Hello {name} at {when}", Arguments{"name": testStringer("Cam"), "when": ts}); !strings.Contains(got, "Cam") || !strings.Contains(got, "2026-03-25 12:34:56 +0000 UTC") {
		t.Fatalf("unexpected interpolated template output: %q", got)
	}
}

func TestLocaleAndRouteHelperBranches(t *testing.T) {
	if got := primaryLanguage(""); got != "" {
		t.Fatalf("expected empty primary language for empty input, got %q", got)
	}
	if got := primaryLanguage("EN_us"); got != "en" {
		t.Fatalf("expected normalized primary language en, got %q", got)
	}

	if got := fallbackString("EN_us", "fr-fr"); got != "en-US" {
		t.Fatalf("expected value branch of fallbackString, got %q", got)
	}
	if got := fallbackString("", "fr-fr"); got != "fr-FR" {
		t.Fatalf("expected fallback branch of fallbackString, got %q", got)
	}

	if got := defaultMissingText("en", "", "headline"); got != "headline" {
		t.Fatalf("expected namespace-empty missing text branch, got %q", got)
	}

	namespace, key := splitCombinedMessageKey("headline")
	if namespace != "default" || key != "headline" {
		t.Fatalf("expected splitCombinedMessageKey default namespace branch, got namespace=%q key=%q", namespace, key)
	}

	if base, query := splitPathAndQuery("   "); base != "/" || query != "" {
		t.Fatalf("expected blank splitPathAndQuery fallback, got base=%q query=%q", base, query)
	}
	if base, query := splitPathAndQuery("docs/getting-started?tab=api"); base != "/docs/getting-started" || query != "?tab=api" {
		t.Fatalf("expected splitPathAndQuery query branch, got base=%q query=%q", base, query)
	}

	if got := normalizeLeadingPath("docs/"); got != "/docs" {
		t.Fatalf("expected normalizeLeadingPath trim branch, got %q", got)
	}
	if got := normalizeLeadingPath(""); got != "/" {
		t.Fatalf("expected normalizeLeadingPath empty branch, got %q", got)
	}

	options := normalizeRouteOptions(RouteOptions{
		SupportedLocales: []string{" fr ", "fr", "en"},
	})
	if options.DefaultLocale != "fr" || len(options.SupportedLocales) != 2 {
		t.Fatalf("unexpected normalized route options: %+v", options)
	}

	if !localeAllowed("fr-CA", []string{"en", "fr"}) {
		t.Fatal("expected localeAllowed to accept matching primary language in supported set")
	}
	if localeAllowed("", []string{"en"}) {
		t.Fatal("expected localeAllowed to reject empty locale")
	}
	if !localeAllowed("en", nil) {
		t.Fatal("expected localeAllowed to allow any non-empty locale when supported set is empty")
	}

	if got := chooseSupportedLocale("fr-CA", []string{"en", "fr"}, "en"); got != "fr-CA" {
		t.Fatalf("expected chooseSupportedLocale to keep normalized supported locale variant, got %q", got)
	}
	if got := chooseSupportedLocale("pt-BR", []string{"en", "fr"}, "en"); got != "en" {
		t.Fatalf("expected chooseSupportedLocale fallback branch, got %q", got)
	}

	candidates := localeCandidates("fr-CA", "fr", "en")
	if len(candidates) < 3 || candidates[0] != "fr-CA" || candidates[1] != "fr" || candidates[len(candidates)-1] != "en" {
		t.Fatalf("unexpected localeCandidates ordering/dedup behavior: %+v", candidates)
	}
}

func TestSSRBootstrapNamespaceFilteringAndDefaultNamespaceRoundTrip(t *testing.T) {
	bundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	bundle.Register("en", Catalog{
		"common":  {"greeting": {Text: "Hello"}},
		"account": {"title": {Text: "Account"}},
	})
	bundle.Register("fr", Catalog{
		"common":  {"greeting": {Text: "Bonjour"}},
		"account": {"title": {Text: "Compte"}},
	})

	payload := bundle.ToSSRBootstrap(SSRBootstrapOptions{
		Locale:            "fr-CA",
		FallbackLocale:    "en",
		IncludeLocales:    []string{"fr", "en"},
		IncludeNamespaces: []string{"common"},
		Direction:         DirectionAuto,
	})
	if payload.Direction != string(DirectionAuto) {
		t.Fatalf("expected explicit direction override to be preserved, got %q", payload.Direction)
	}
	if _, ok := payload.Messages["fr"]["common.greeting"]; !ok {
		t.Fatalf("expected included namespace message in payload, got %+v", payload.Messages)
	}
	if _, ok := payload.Messages["fr"]["account.title"]; ok {
		t.Fatalf("expected filtered namespace to be excluded from payload, got %+v", payload.Messages)
	}

	// Combined key without namespace should map into "default" namespace on restore.
	restored := BundleFromSSRBootstrap(ui.SSRI18nBootstrap{
		Locale:         "en",
		FallbackLocale: "en",
		Messages: map[string]map[string]ui.SSRI18nMessage{
			"en": {
				"headline": {Text: "Welcome"},
			},
		},
	})
	if got := restored.Translate("en", "default", "headline", nil, "en"); got != "Welcome" {
		t.Fatalf("expected default namespace round trip from combined key, got %q", got)
	}
}

func TestFormattingAndRouteHelperEdgeBranches(t *testing.T) {
	date := time.Date(2026, time.March, 16, 10, 30, 0, 0, time.UTC)

	if got := FormatNumber("en-US", 42); got != "42" {
		t.Fatalf("expected integer format without decimals, got %q", got)
	}
	if got := FormatNumber("en-US", 42.125, NumberOptions{MaximumFractionDigits: 1}); got != "42.1" {
		t.Fatalf("expected explicit max fraction digits to apply, got %q", got)
	}

	if got := FormatDate("fr-FR", date, DateOptions{Style: DateStyleLong}); got != "16 March 2026" {
		t.Fatalf("expected french long date branch, got %q", got)
	}
	if got := FormatDate("fr-FR", date); got != "16 Mar 2026" {
		t.Fatalf("expected french medium date branch, got %q", got)
	}
	if got := FormatDate("en-US", date); got != "Mar 16, 2026" {
		t.Fatalf("expected english medium date branch, got %q", got)
	}
	if got := FormatDate("en-US", date, DateOptions{Style: DateStyleShort}); got != "03/16/2026" {
		t.Fatalf("expected english short date branch, got %q", got)
	}
	if got := FormatDate("en-US", date, DateOptions{Location: time.FixedZone("UTC+2", 2*60*60)}); got != "Mar 16, 2026" {
		t.Fatalf("expected location-aware date branch to succeed, got %q", got)
	}

	options := RouteOptions{SupportedLocales: []string{"en", "fr"}, DefaultLocale: "en"}
	resolved := ResolvePath("/fr", options)
	if resolved.Locale != "fr" || resolved.BasePath != "/" || resolved.LocalizedPath != "/fr" || !resolved.PrefixPresent {
		t.Fatalf("expected locale-prefix root resolution, got %+v", resolved)
	}
	resolved = ResolvePath("/", RouteOptions{})
	if resolved.Locale != "" || resolved.BasePath != "/" || resolved.LocalizedPath != "/" || resolved.PrefixPresent {
		t.Fatalf("expected root path without locale defaults to stay empty, got %+v", resolved)
	}
	if got := PrefixPath("en", "/", options); got != "/en" {
		t.Fatalf("expected root prefix path branch, got %q", got)
	}

	if got := stringifyArgument(7); got != "7" {
		t.Fatalf("expected default stringifyArgument branch, got %q", got)
	}
	if got := pluralCategoryForLocale("en", -1); got != PluralOne {
		t.Fatalf("expected negative english value to normalize to one, got %q", got)
	}
	if got := pluralCategoryForLocale("ar", 100); got != PluralOther {
		t.Fatalf("expected arabic default plural branch, got %q", got)
	}
	if got := pluralCategoryForLocale("ru", 1); got != PluralOne {
		t.Fatalf("expected russian one plural branch, got %q", got)
	}
	if got := pluralCategoryForLocale("cs", 1); got != PluralOne {
		t.Fatalf("expected czech one plural branch, got %q", got)
	}
	if got := pluralCategoryForLocale("cs", 5); got != PluralOther {
		t.Fatalf("expected czech default plural branch, got %q", got)
	}
	if base, query := splitPathAndQuery("pricing"); base != "/pricing" || query != "" {
		t.Fatalf("expected splitPathAndQuery without query to normalize base path, got base=%q query=%q", base, query)
	}
	if got := normalizeLeadingPath("///"); got != "/" {
		t.Fatalf("expected normalizeLeadingPath to collapse slash-only input, got %q", got)
	}
	if got := normalizeLocales([]string{" ", "en", "en", "fr "}); len(got) != 2 || got[0] != "en" || got[1] != "fr" {
		t.Fatalf("expected normalizeLocales to skip blanks and dedupe, got %v", got)
	}
	if got := localeCandidates("", " ", "en"); len(got) != 1 || got[0] != "en" {
		t.Fatalf("expected localeCandidates to skip blank normalized values, got %v", got)
	}

	var nilBundle *Bundle
	if payload := nilBundle.ToSSRBootstrap(SSRBootstrapOptions{}); payload.Locale != "" || payload.FallbackLocale != "" || payload.Direction != "" || len(payload.Messages) != 0 {
		t.Fatalf("expected nil bundle bootstrap payload to be empty, got %+v", payload)
	}
}
