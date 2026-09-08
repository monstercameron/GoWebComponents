package i18n

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type testStringer string

func (parseS testStringer) String() string { return string(parseS) }

func TestRegisterAndTranslateMissingBranches(parseT *testing.T) {
	var parseNilBundle *Bundle
	parseNilBundle.Register("en", Catalog{
		"common": {"greeting": {Text: "Hello"}},
	})
	if parseGot := parseNilBundle.Translate("en", "common", "greeting", nil, "en"); parseGot != "common.greeting" {
		parseT.Fatalf("expected nil bundle to use missing-text fallback, got %q", parseGot)
	}

	parseBundle := &Bundle{}
	parseBundle.Register("   ", Catalog{"common": {"greeting": {Text: "ignored"}}})
	if len(parseBundle.catalogs) != 0 {
		parseT.Fatalf("expected blank locale registration to be ignored, got %+v", parseBundle.catalogs)
	}

	parseSourceCatalog := Catalog{
		"common": {"greeting": {Text: "Hello"}},
	}
	parseBundle.Register(" EN_us ", parseSourceCatalog)
	if parseBundle.defaultLocale != "en-US" || parseBundle.fallbackLocale != "en-US" {
		parseT.Fatalf("expected default/fallback locale to initialize from first register, got default=%q fallback=%q", parseBundle.defaultLocale, parseBundle.fallbackLocale)
	}

	// Ensure Register clones message values instead of aliasing caller maps.
	parseSourceCatalog["common"]["greeting"] = Message{Text: "Mutated"}
	if parseGot2 := parseBundle.Translate("en-US", "common", "greeting", nil, "en-US"); parseGot2 != "Hello" {
		parseT.Fatalf("expected registered message clone to remain stable, got %q", parseGot2)
	}

	parseBundle.onMissing = func(parseLocale, parseNamespace, parseKey string) string {
		return strings.ToUpper(parseLocale) + ":" + parseNamespace + "." + parseKey
	}
	if parseGot3 := parseBundle.Translate("fr", "common", "missing", nil, "en-US"); parseGot3 != "FR:common.missing" {
		parseT.Fatalf("expected custom missing handler response, got %q", parseGot3)
	}
}

func TestResolveTemplateAndInterpolationBranches(parseT *testing.T) {
	if parseGot := resolveTemplate("en", Message{
		SelectArg: "tone",
		Select:    map[string]string{"other": "Hello"},
	}, Arguments{}); parseGot != "Hello" {
		parseT.Fatalf("expected select branch to use other fallback, got %q", parseGot)
	}

	if parseGot2 := resolveTemplate("en", Message{
		SelectArg: "tone",
		Select:    map[string]string{"formal": "Welcome"},
		Default:   "Default",
	}, Arguments{"tone": "casual"}); parseGot2 != "Default" {
		parseT.Fatalf("expected select branch to use default fallback, got %q", parseGot2)
	}

	if parseGot3 := resolveTemplate("ar", Message{
		PluralArg: "count",
		Plural:    map[PluralCategory]string{PluralOther: "many"},
	}, Arguments{"count": 2}); parseGot3 != "many" {
		parseT.Fatalf("expected plural branch to use other fallback, got %q", parseGot3)
	}

	if parseGot4 := resolveTemplate("en", Message{
		PluralArg: "count",
		Plural:    map[PluralCategory]string{PluralOne: "one"},
		Default:   "plural-default",
	}, Arguments{"count": 3}); parseGot4 != "plural-default" {
		parseT.Fatalf("expected plural branch to use default fallback, got %q", parseGot4)
	}

	if parseGot5 := resolveTemplate("en", Message{Text: "Text", Default: "Default"}, nil); parseGot5 != "Text" {
		parseT.Fatalf("expected text fallback to win when present, got %q", parseGot5)
	}
	if parseGot6 := resolveTemplate("en", Message{Default: "OnlyDefault"}, nil); parseGot6 != "OnlyDefault" {
		parseT.Fatalf("expected default fallback when no other templates exist, got %q", parseGot6)
	}

	parseTs := time.Date(2026, time.March, 25, 12, 34, 56, 0, time.UTC)
	if parseGot7 := stringifyArgument(testStringer("cam")); parseGot7 != "cam" {
		parseT.Fatalf("expected fmt.Stringer branch, got %q", parseGot7)
	}
	if parseGot8 := stringifyArgument(parseTs); parseGot8 != parseTs.String() {
		parseT.Fatalf("expected time.Time to follow fmt.Stringer branch, got %q", parseGot8)
	}
	if parseGot9 := interpolateTemplate("Hello {name} at {when}", Arguments{"name": testStringer("Cam"), "when": parseTs}); !strings.Contains(parseGot9, "Cam") || !strings.Contains(parseGot9, "2026-03-25 12:34:56 +0000 UTC") {
		parseT.Fatalf("unexpected interpolated template output: %q", parseGot9)
	}
	if parseGot10 := interpolateTemplate("{name} and {name}", Arguments{"name": "Ada"}); parseGot10 != "Ada and Ada" {
		parseT.Fatalf("expected repeated placeholder replacement, got %q", parseGot10)
	}
}

func TestLocaleAndRouteHelperBranches(parseT *testing.T) {
	if parseGot := primaryLanguage(""); parseGot != "" {
		parseT.Fatalf("expected empty primary language for empty input, got %q", parseGot)
	}
	if parseGot2 := primaryLanguage("EN_us"); parseGot2 != "en" {
		parseT.Fatalf("expected normalized primary language en, got %q", parseGot2)
	}

	if parseGot3 := fallbackString("EN_us", "fr-fr"); parseGot3 != "en-US" {
		parseT.Fatalf("expected value branch of fallbackString, got %q", parseGot3)
	}
	if parseGot4 := fallbackString("", "fr-fr"); parseGot4 != "fr-FR" {
		parseT.Fatalf("expected fallback branch of fallbackString, got %q", parseGot4)
	}

	if parseGot5 := defaultMissingText("en", "", "headline"); parseGot5 != "headline" {
		parseT.Fatalf("expected namespace-empty missing text branch, got %q", parseGot5)
	}

	parseNamespace, parseKey := splitCombinedMessageKey("headline")
	if parseNamespace != "default" || parseKey != "headline" {
		parseT.Fatalf("expected splitCombinedMessageKey default namespace branch, got namespace=%q key=%q", parseNamespace, parseKey)
	}

	if parseBase, parseQuery := splitPathAndQuery("   "); parseBase != "/" || parseQuery != "" {
		parseT.Fatalf("expected blank splitPathAndQuery fallback, got base=%q query=%q", parseBase, parseQuery)
	}
	if parseBase2, parseQuery2 := splitPathAndQuery("docs/getting-started?tab=api"); parseBase2 != "/docs/getting-started" || parseQuery2 != "?tab=api" {
		parseT.Fatalf("expected splitPathAndQuery query branch, got base=%q query=%q", parseBase2, parseQuery2)
	}

	if parseGot6 := normalizeLeadingPath("docs/"); parseGot6 != "/docs" {
		parseT.Fatalf("expected normalizeLeadingPath trim branch, got %q", parseGot6)
	}
	if parseGot7 := normalizeLeadingPath(""); parseGot7 != "/" {
		parseT.Fatalf("expected normalizeLeadingPath empty branch, got %q", parseGot7)
	}

	parseOptions := normalizeRouteOptions(RouteOptions{
		SupportedLocales: []string{" fr ", "fr", "en"},
	})
	if parseOptions.DefaultLocale != "fr" || len(parseOptions.SupportedLocales) != 2 {
		parseT.Fatalf("unexpected normalized route options: %+v", parseOptions)
	}

	if !localeAllowed("fr-CA", []string{"en", "fr"}) {
		parseT.Fatal("expected localeAllowed to accept matching primary language in supported set")
	}
	if localeAllowed("", []string{"en"}) {
		parseT.Fatal("expected localeAllowed to reject empty locale")
	}
	if !localeAllowed("en", nil) {
		parseT.Fatal("expected localeAllowed to allow any non-empty locale when supported set is empty")
	}

	if parseGot8 := chooseSupportedLocale("fr-CA", []string{"en", "fr"}, "en"); parseGot8 != "fr-CA" {
		parseT.Fatalf("expected chooseSupportedLocale to keep normalized supported locale variant, got %q", parseGot8)
	}
	if parseGot9 := chooseSupportedLocale("pt-BR", []string{"en", "fr"}, "en"); parseGot9 != "en" {
		parseT.Fatalf("expected chooseSupportedLocale fallback branch, got %q", parseGot9)
	}

	parseCandidates := localeCandidates("fr-CA", "fr", "en")
	if len(parseCandidates) < 3 || parseCandidates[0] != "fr-CA" || parseCandidates[1] != "fr" || parseCandidates[len(parseCandidates)-1] != "en" {
		parseT.Fatalf("unexpected localeCandidates ordering/dedup behavior: %+v", parseCandidates)
	}
}

func TestSSRBootstrapNamespaceFilteringAndDefaultNamespaceRoundTrip(parseT *testing.T) {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	parseBundle.Register("en", Catalog{
		"common":  {"greeting": {Text: "Hello"}},
		"account": {"title": {Text: "Account"}},
	})
	parseBundle.Register("fr", Catalog{
		"common":  {"greeting": {Text: "Bonjour"}},
		"account": {"title": {Text: "Compte"}},
	})

	parsePayload := parseBundle.ToSSRBootstrap(SSRBootstrapOptions{
		Locale:            "fr-CA",
		FallbackLocale:    "en",
		IncludeLocales:    []string{"fr", "en"},
		IncludeNamespaces: []string{"common"},
		Direction:         DirectionAuto,
	})
	if parsePayload.Direction != string(DirectionAuto) {
		parseT.Fatalf("expected explicit direction override to be preserved, got %q", parsePayload.Direction)
	}
	if _, parseOk := parsePayload.Messages["fr"]["common.greeting"]; !parseOk {
		parseT.Fatalf("expected included namespace message in payload, got %+v", parsePayload.Messages)
	}
	if _, parseOk2 := parsePayload.Messages["fr"]["account.title"]; parseOk2 {
		parseT.Fatalf("expected filtered namespace to be excluded from payload, got %+v", parsePayload.Messages)
	}

	// Combined key without namespace should map into "default" namespace on restore.
	parseRestored := BundleFromSSRBootstrap(ui.SSRI18nBootstrap{
		Locale:         "en",
		FallbackLocale: "en",
		Messages: map[string]map[string]ui.SSRI18nMessage{
			"en": {
				"headline": {Text: "Welcome"},
			},
		},
	})
	if parseGot := parseRestored.Translate("en", "default", "headline", nil, "en"); parseGot != "Welcome" {
		parseT.Fatalf("expected default namespace round trip from combined key, got %q", parseGot)
	}
}

func TestFormattingAndRouteHelperEdgeBranches(parseT *testing.T) {
	parseDate := time.Date(2026, time.March, 16, 10, 30, 0, 0, time.UTC)

	if parseGot := FormatNumber("en-US", 42); parseGot != "42" {
		parseT.Fatalf("expected integer format without decimals, got %q", parseGot)
	}
	if parseGot2 := FormatNumber("en-US", 42.125, NumberOptions{MaximumFractionDigits: 1}); parseGot2 != "42.1" {
		parseT.Fatalf("expected explicit max fraction digits to apply, got %q", parseGot2)
	}

	if parseGot3 := FormatDate("fr-FR", parseDate, DateOptions{Style: DateStyleLong}); parseGot3 != "16 March 2026" {
		parseT.Fatalf("expected french long date branch, got %q", parseGot3)
	}
	if parseGot4 := FormatDate("fr-FR", parseDate); parseGot4 != "16 Mar 2026" {
		parseT.Fatalf("expected french medium date branch, got %q", parseGot4)
	}
	if parseGot5 := FormatDate("en-US", parseDate); parseGot5 != "Mar 16, 2026" {
		parseT.Fatalf("expected english medium date branch, got %q", parseGot5)
	}
	if parseGot6 := FormatDate("en-US", parseDate, DateOptions{Style: DateStyleShort}); parseGot6 != "03/16/2026" {
		parseT.Fatalf("expected english short date branch, got %q", parseGot6)
	}
	if parseGot7 := FormatDate("en-US", parseDate, DateOptions{Location: time.FixedZone("UTC+2", 2*60*60)}); parseGot7 != "Mar 16, 2026" {
		parseT.Fatalf("expected location-aware date branch to succeed, got %q", parseGot7)
	}

	parseOptions := RouteOptions{SupportedLocales: []string{"en", "fr"}, DefaultLocale: "en"}
	parseResolved := ResolvePath("/fr", parseOptions)
	if parseResolved.Locale != "fr" || parseResolved.BasePath != "/" || parseResolved.LocalizedPath != "/fr" || !parseResolved.PrefixPresent {
		parseT.Fatalf("expected locale-prefix root resolution, got %+v", parseResolved)
	}
	parseResolved = ResolvePath("/", RouteOptions{})
	if parseResolved.Locale != "" || parseResolved.BasePath != "/" || parseResolved.LocalizedPath != "/" || parseResolved.PrefixPresent {
		parseT.Fatalf("expected root path without locale defaults to stay empty, got %+v", parseResolved)
	}
	if parseGot8 := PrefixPath("en", "/", parseOptions); parseGot8 != "/en" {
		parseT.Fatalf("expected root prefix path branch, got %q", parseGot8)
	}

	if parseGot9 := stringifyArgument(7); parseGot9 != "7" {
		parseT.Fatalf("expected default stringifyArgument branch, got %q", parseGot9)
	}
	if parseGot10 := pluralCategoryForLocale("en", -1); parseGot10 != PluralOne {
		parseT.Fatalf("expected negative english value to normalize to one, got %q", parseGot10)
	}
	if parseGot10b := pluralCategoryForLocale("ru", -11); parseGot10b != PluralMany {
		parseT.Fatalf("expected negative russian value to normalize to many, got %q", parseGot10b)
	}
	if parseGot11 := pluralCategoryForLocale("ar", 100); parseGot11 != PluralOther {
		parseT.Fatalf("expected arabic default plural branch, got %q", parseGot11)
	}
	if parseGot12 := pluralCategoryForLocale("ru", 1); parseGot12 != PluralOne {
		parseT.Fatalf("expected russian one plural branch, got %q", parseGot12)
	}
	if parseGot13 := pluralCategoryForLocale("cs", 1); parseGot13 != PluralOne {
		parseT.Fatalf("expected czech one plural branch, got %q", parseGot13)
	}
	if parseGot14 := pluralCategoryForLocale("cs", 5); parseGot14 != PluralOther {
		parseT.Fatalf("expected czech default plural branch, got %q", parseGot14)
	}
	if parseBase, parseQuery := splitPathAndQuery("pricing"); parseBase != "/pricing" || parseQuery != "" {
		parseT.Fatalf("expected splitPathAndQuery without query to normalize base path, got base=%q query=%q", parseBase, parseQuery)
	}
	if parseGot15 := normalizeLeadingPath("///"); parseGot15 != "/" {
		parseT.Fatalf("expected normalizeLeadingPath to collapse slash-only input, got %q", parseGot15)
	}
	if parseGot16 := normalizeLocales([]string{" ", "en", "en", "fr "}); len(parseGot16) != 2 || parseGot16[0] != "en" || parseGot16[1] != "fr" {
		parseT.Fatalf("expected normalizeLocales to skip blanks and dedupe, got %v", parseGot16)
	}
	if parseGot17 := localeCandidates("", " ", "en"); len(parseGot17) != 1 || parseGot17[0] != "en" {
		parseT.Fatalf("expected localeCandidates to skip blank normalized values, got %v", parseGot17)
	}

	var parseNilBundle *Bundle
	if parsePayload := parseNilBundle.ToSSRBootstrap(SSRBootstrapOptions{}); parsePayload.Locale != "" || parsePayload.FallbackLocale != "" || parsePayload.Direction != "" || len(parsePayload.Messages) != 0 {
		parseT.Fatalf("expected nil bundle bootstrap payload to be empty, got %+v", parsePayload)
	}
}
