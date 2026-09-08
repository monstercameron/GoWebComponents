package i18n

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestRuntimeMethodsAndProvider(parseT *testing.T) {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	parseBundle.RegisterNamespace("en", "common", NamespaceCatalog{
		"greeting": {Text: "Hello {name}"},
	})
	parseBundle.RegisterNamespace("fr", "common", NamespaceCatalog{
		"greeting": {Text: "Bonjour {name}"},
	})

	var setTo string
	parseLocale := "fr"
	parseState := LocaleState{
		get:       func() string { return parseLocale },
		set:       func(parseNext string) { setTo = parseNext },
		direction: func() Direction { return DirectionRTL },
		fallback:  func() string { return "en" },
	}

	parseNode := Provider(ProviderProps{
		Locale:   parseState,
		Bundle:   parseBundle,
		Child:    ui.Text("one"),
		Children: []ui.Node{ui.Text("two")},
	})
	if parseNode == nil {
		parseT.Fatalf("expected provider node")
	}
	if len(parseNode.Children) != 2 {
		parseT.Fatalf("expected provider to include child and children, got %d", len(parseNode.Children))
	}
	parseRawRuntime, parseOk := parseNode.Props["value"].(Runtime)
	if !parseOk {
		parseT.Fatalf("expected runtime value in provider props, got %T", parseNode.Props["value"])
	}
	if parseGot := parseRawRuntime.Locale(); parseGot != "fr" {
		parseT.Fatalf("runtime locale = %q, want fr", parseGot)
	}
	parseRawRuntime.SetLocale("ar")
	if setTo != "ar" {
		parseT.Fatalf("runtime set locale did not delegate: %q", setTo)
	}
	if parseGot2 := parseRawRuntime.Direction(); parseGot2 != DirectionRTL {
		parseT.Fatalf("runtime direction = %q, want rtl", parseGot2)
	}
	if parseGot3 := parseRawRuntime.T("common", "greeting", Arguments{"name": "Cam"}); parseGot3 != "Bonjour Cam" {
		parseT.Fatalf("runtime translation = %q, want Bonjour Cam", parseGot3)
	}
	if parseGot4 := parseRawRuntime.FormatNumber(12.5); parseGot4 == "" {
		parseT.Fatalf("expected formatted number")
	}
	if parseGot5 := parseRawRuntime.FormatDate(time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC)); parseGot5 == "" {
		parseT.Fatalf("expected formatted date")
	}
	if parseGot6 := parseRawRuntime.PrefixPath("/pricing", RouteOptions{SupportedLocales: []string{"en", "fr"}, OmitDefaultPrefix: true}); parseGot6 != "/fr/pricing" {
		parseT.Fatalf("runtime prefix path = %q, want /fr/pricing", parseGot6)
	}

	parseZero := Runtime{}
	if parseZero.Locale() != "" {
		parseT.Fatalf("zero runtime locale should be empty")
	}
	if parseZero.Direction() != DirectionLTR {
		parseT.Fatalf("zero runtime direction should default to ltr")
	}
	if parseGot7 := parseZero.T("common", "missing"); parseGot7 != "common.missing" {
		parseT.Fatalf("zero runtime missing text = %q, want common.missing", parseGot7)
	}
}

func TestUseI18nPanicsOnNative(parseT *testing.T) {
	defer func() {
		if recover() == nil {
			parseT.Fatalf("expected UseI18n to panic in non-browser runtime")
		}
	}()
	_ = UseI18n()
}

func TestBundleAndLocaleStateNilSafety(parseT *testing.T) {
	var parseState LocaleState
	if parseGot := parseState.SupportedLocales(); parseGot != nil {
		parseT.Fatalf("zero locale state supported locales = %v, want nil", parseGot)
	}
	parseState = LocaleState{supported: func() []string { return []string{"en", "fr"} }}
	if parseGot2 := parseState.SupportedLocales(); len(parseGot2) != 2 || parseGot2[0] != "en" || parseGot2[1] != "fr" {
		parseT.Fatalf("supported locales = %v", parseGot2)
	}

	var parseNilBundle *Bundle
	if parseGot3 := parseNilBundle.DefaultLocale(); parseGot3 != "" {
		parseT.Fatalf("nil bundle default locale = %q, want empty", parseGot3)
	}
	if parseGot4 := parseNilBundle.FallbackLocale(); parseGot4 != "" {
		parseT.Fatalf("nil bundle fallback locale = %q, want empty", parseGot4)
	}
}

func TestFormatDateAdditionalLocaleBranches(parseT *testing.T) {
	parseDate := time.Date(2026, time.March, 16, 10, 30, 0, 0, time.UTC)

	if parseGot := FormatDate("de-DE", parseDate, DateOptions{Style: DateStyleShort}); parseGot != "16.03.2026" {
		parseT.Fatalf("expected german short date, got %q", parseGot)
	}
	if parseGot2 := FormatDate("de-DE", parseDate, DateOptions{Style: DateStyleLong}); parseGot2 != "16. March 2026" {
		parseT.Fatalf("expected german long date, got %q", parseGot2)
	}
	if parseGot3 := FormatDate("de-DE", parseDate); parseGot3 != "16. Mar 2026" {
		parseT.Fatalf("expected german medium date, got %q", parseGot3)
	}
	if parseGot4 := FormatDate("ar-EG", parseDate, DateOptions{Style: DateStyleLong}); parseGot4 != "16 Mar 2026" {
		parseT.Fatalf("expected arabic long date format, got %q", parseGot4)
	}
	if parseGot5 := FormatDate("ar-EG", parseDate); parseGot5 != "16/03/2026" {
		parseT.Fatalf("expected arabic default date format, got %q", parseGot5)
	}
}

func TestNumericArgumentAndPluralCategoryCoverage(parseT *testing.T) {
	if numericArgument(int8(2)) != 2 || numericArgument(int16(3)) != 3 || numericArgument(int32(4)) != 4 || numericArgument(int64(5)) != 5 {
		parseT.Fatalf("expected signed integer conversions to succeed")
	}
	if numericArgument(uint(6)) != 6 || numericArgument(uint8(7)) != 7 || numericArgument(uint16(8)) != 8 || numericArgument(uint32(9)) != 9 || numericArgument(uint64(10)) != 10 {
		parseT.Fatalf("expected unsigned integer conversions to succeed")
	}
	if numericArgument(float32(11.5)) != 11.5 || numericArgument(float64(12.5)) != 12.5 {
		parseT.Fatalf("expected float conversions to succeed")
	}
	if numericArgument("13.25") != 13.25 {
		parseT.Fatalf("expected numeric string parsing to succeed")
	}
	if numericArgument("not-a-number") != 0 {
		parseT.Fatalf("expected invalid numeric string to return 0")
	}

	if parseGot := pluralCategoryForLocale("ar", 0); parseGot != PluralZero {
		parseT.Fatalf("arabic plural category for 0 = %q", parseGot)
	}
	if parseGot2 := pluralCategoryForLocale("ar", 1); parseGot2 != PluralOne {
		parseT.Fatalf("arabic plural category for 1 = %q", parseGot2)
	}
	if parseGot3 := pluralCategoryForLocale("ar", 2); parseGot3 != PluralTwo {
		parseT.Fatalf("arabic plural category for 2 = %q", parseGot3)
	}
	if parseGot4 := pluralCategoryForLocale("ar", 7); parseGot4 != PluralFew {
		parseT.Fatalf("arabic plural category for 7 = %q", parseGot4)
	}
	if parseGot5 := pluralCategoryForLocale("ar", 23); parseGot5 != PluralMany {
		parseT.Fatalf("arabic plural category for 23 = %q", parseGot5)
	}
	if parseGot6 := pluralCategoryForLocale("fr", 0); parseGot6 != PluralOne {
		parseT.Fatalf("french plural category for 0 = %q", parseGot6)
	}
	if parseGot7 := pluralCategoryForLocale("fr", 2); parseGot7 != PluralOther {
		parseT.Fatalf("french plural category for 2 = %q", parseGot7)
	}
	if parseGot8 := pluralCategoryForLocale("ru", 2); parseGot8 != PluralFew {
		parseT.Fatalf("russian plural category for 2 = %q", parseGot8)
	}
	if parseGot9 := pluralCategoryForLocale("ru", 5); parseGot9 != PluralMany {
		parseT.Fatalf("russian plural category for 5 = %q", parseGot9)
	}
	if parseGot10 := pluralCategoryForLocale("pl", 1); parseGot10 != PluralOne {
		parseT.Fatalf("polish plural category for 1 = %q", parseGot10)
	}
	if parseGot11 := pluralCategoryForLocale("pl", 3); parseGot11 != PluralFew {
		parseT.Fatalf("polish plural category for 3 = %q", parseGot11)
	}
	if parseGot12 := pluralCategoryForLocale("cs", 4); parseGot12 != PluralFew {
		parseT.Fatalf("czech plural category for 4 = %q", parseGot12)
	}
	if parseGot13 := pluralCategoryForLocale("ja", 10); parseGot13 != PluralOther {
		parseT.Fatalf("japanese plural category for 10 = %q", parseGot13)
	}
	if parseGot14 := pluralCategoryForLocale("en", 1); parseGot14 != PluralOne {
		parseT.Fatalf("english plural category for 1 = %q", parseGot14)
	}
	if parseGot15 := pluralCategoryForLocale("en", 9); parseGot15 != PluralOther {
		parseT.Fatalf("english plural category for 9 = %q", parseGot15)
	}
}
