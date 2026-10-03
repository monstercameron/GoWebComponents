package i18n

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestFormatDateLocaleTables pins month names and field order per language
// and style; en must stay byte-identical to the original layouts.
func TestFormatDateLocaleTables(parseT *testing.T) {
	parseDate := time.Date(2026, time.March, 5, 10, 30, 0, 0, time.UTC)
	parseCases := []struct {
		parseLocale string
		parseShort  string
		parseMedium string
		parseLong   string
	}{
		{"en", "03/05/2026", "Mar 5, 2026", "March 5, 2026"},
		{"en-GB", "03/05/2026", "Mar 5, 2026", "March 5, 2026"},
		{"xx", "03/05/2026", "Mar 5, 2026", "March 5, 2026"},
		{"fr", "05/03/2026", "5 mars 2026", "5 mars 2026"},
		{"de", "05.03.2026", "5. März 2026", "5. März 2026"},
		{"es", "05/03/2026", "5 mar 2026", "5 de marzo de 2026"},
		{"pt-BR", "05/03/2026", "5 de mar. de 2026", "5 de março de 2026"},
		{"it", "05/03/2026", "5 mar 2026", "5 marzo 2026"},
		{"nl", "05-03-2026", "5 mrt 2026", "5 maart 2026"},
		{"ru", "05.03.2026", "5 мар. 2026 г.", "5 марта 2026 г."},
		{"pl", "05.03.2026", "5 mar 2026", "5 marca 2026"},
		{"cs", "05.03.2026", "5. 3. 2026", "5. března 2026"},
		{"tr", "05.03.2026", "5 Mar 2026", "5 Mart 2026"},
		{"ja", "2026/03/05", "2026年3月5日", "2026年3月5日"},
		{"zh-CN", "2026/03/05", "2026年3月5日", "2026年3月5日"},
		{"ko", "2026. 03. 05.", "2026. 3. 5.", "2026년 3월 5일"},
		{"ar-EG", "05/03/2026", "05/03/2026", "5 مارس 2026"},
	}
	for _, parseCase := range parseCases {
		if parseGot := FormatDate(parseCase.parseLocale, parseDate, DateOptions{Style: DateStyleShort}); parseGot != parseCase.parseShort {
			parseT.Errorf("%s short = %q, want %q", parseCase.parseLocale, parseGot, parseCase.parseShort)
		}
		if parseGot := FormatDate(parseCase.parseLocale, parseDate); parseGot != parseCase.parseMedium {
			parseT.Errorf("%s medium = %q, want %q", parseCase.parseLocale, parseGot, parseCase.parseMedium)
		}
		if parseGot := FormatDate(parseCase.parseLocale, parseDate, DateOptions{Style: DateStyleLong}); parseGot != parseCase.parseLong {
			parseT.Errorf("%s long = %q, want %q", parseCase.parseLocale, parseGot, parseCase.parseLong)
		}
	}
}

// TestFormatDateLocaleAllMonths guards every month slot of every table.
func TestFormatDateLocaleAllMonths(parseT *testing.T) {
	for parseFamily, parseData := range dateLocales {
		if parseData.monthsLong[0] == "" {
			continue // numeric-only languages (ja, zh, ko)
		}
		for parseMonth := time.January; parseMonth <= time.December; parseMonth++ {
			if parseData.monthsLong[parseMonth-1] == "" || parseData.monthsShort[parseMonth-1] == "" {
				parseT.Errorf("%s month %d has an empty name", parseFamily, parseMonth)
			}
		}
	}
}

// TestFormatListLocales pins conjunctions for two and three items.
func TestFormatListLocales(parseT *testing.T) {
	parseCases := []struct {
		parseLocale string
		parsePair   string
		parseTriple string
	}{
		{"en", "a and b", "a, b, and c"},
		{"fr", "a et b", "a, b et c"},
		{"de", "a und b", "a, b und c"},
		{"es", "a y b", "a, b y c"},
		{"pt", "a e b", "a, b e c"},
		{"it", "a e b", "a, b e c"},
		{"nl", "a en b", "a, b en c"},
		{"ru", "a и b", "a, b и c"},
		{"ja", "a、b", "a、b、c"},
		{"zh", "a和b", "a、b和c"},
		{"ko", "a 및 b", "a, b 및 c"},
		{"ar", "a و b", "a، b، و c"},
		{"pl", "a i b", "a, b i c"},
		{"cs", "a a b", "a, b a c"},
		{"tr", "a ve b", "a, b ve c"},
	}
	for _, parseCase := range parseCases {
		if parseGot := FormatList(parseCase.parseLocale, []string{"a", "b"}); parseGot != parseCase.parsePair {
			parseT.Errorf("%s pair = %q, want %q", parseCase.parseLocale, parseGot, parseCase.parsePair)
		}
		if parseGot := FormatList(parseCase.parseLocale, []string{"a", "b", "c"}); parseGot != parseCase.parseTriple {
			parseT.Errorf("%s triple = %q, want %q", parseCase.parseLocale, parseGot, parseCase.parseTriple)
		}
	}
}

// TestSSRBootstrapNamespaceWithDotRoundTrips pins that dotted and backslash
// namespaces survive ToSSRBootstrap -> BundleFromSSRBootstrap, while dot-free
// namespaces keep the plain "namespace.key" encoding.
func TestSSRBootstrapNamespaceWithDotRoundTrips(parseT *testing.T) {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	parseBundle.Register("en", Catalog{
		"plain":      {"title": {Text: "Plain"}},
		"app.shell":  {"title": {Text: "Dotted"}, "a.b": {Text: "DottedKey"}},
		`back\slash`: {"x": {Text: "Slash"}},
	})
	parsePayload := parseBundle.ToSSRBootstrap(SSRBootstrapOptions{Locale: "en"})
	if _, parseOk := parsePayload.Messages["en"]["plain.title"]; !parseOk {
		parseT.Fatalf("dot-free namespace key changed: %v", parsePayload.Messages["en"])
	}
	parseRestored := BundleFromSSRBootstrap(parsePayload)
	for _, parseCase := range []struct{ parseNamespace, parseKey, parseWant string }{
		{"plain", "title", "Plain"},
		{"app.shell", "title", "Dotted"},
		{"app.shell", "a.b", "DottedKey"},
		{`back\slash`, "x", "Slash"},
	} {
		if parseGot := parseRestored.Translate("en", parseCase.parseNamespace, parseCase.parseKey, nil, "en"); parseGot != parseCase.parseWant {
			parseT.Errorf("%q/%q = %q, want %q", parseCase.parseNamespace, parseCase.parseKey, parseGot, parseCase.parseWant)
		}
	}
	// Legacy payloads: an unescaped first dot still splits namespace from key.
	parseLegacy := BundleFromSSRBootstrap(ui.SSRI18nBootstrap{Locale: "en", FallbackLocale: "en", Messages: map[string]map[string]ui.SSRI18nMessage{
		"en": {"nav.home": {Text: "Home"}},
	}})
	if parseGot := parseLegacy.Translate("en", "nav", "home", nil, "en"); parseGot != "Home" {
		parseT.Errorf("legacy key = %q", parseGot)
	}
}

// TestResolvePathWithoutSupportedLocales pins that only well-formed locale
// shapes are treated as a prefix when no SupportedLocales list is set.
func TestResolvePathWithoutSupportedLocales(parseT *testing.T) {
	parseCases := []struct {
		parsePath   string
		parseLocale string
		parseBase   string
		parsePrefix bool
	}{
		{"/about", "", "/about", false},
		{"/about/x", "", "/about/x", false},
		{"/api/users", "", "/api/users", false},
		{"/app", "", "/app", false},
		{"/js/app.js", "", "/js/app.js", false},
		{"/en", "en", "/", true},
		{"/en-US/docs", "en-US", "/docs", true},
		{"/zh-Hant-TW/a", "zh-Hant-TW", "/a", true},
		{"/fil-PH", "fil-PH", "/", true},
	}
	for _, parseCase := range parseCases {
		parseGot := ResolvePath(parseCase.parsePath, RouteOptions{})
		if parseGot.Locale != parseCase.parseLocale || parseGot.BasePath != parseCase.parseBase || parseGot.PrefixPresent != parseCase.parsePrefix {
			parseT.Errorf("ResolvePath(%q) = %+v, want locale=%q base=%q prefix=%v", parseCase.parsePath, parseGot, parseCase.parseLocale, parseCase.parseBase, parseCase.parsePrefix)
		}
	}
}
