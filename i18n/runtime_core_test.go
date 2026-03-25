package i18n

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestRuntimeMethodsAndProvider(t *testing.T) {
	bundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	bundle.RegisterNamespace("en", "common", NamespaceCatalog{
		"greeting": {Text: "Hello {name}"},
	})
	bundle.RegisterNamespace("fr", "common", NamespaceCatalog{
		"greeting": {Text: "Bonjour {name}"},
	})

	var setTo string
	locale := "fr"
	state := LocaleState{
		get:       func() string { return locale },
		set:       func(next string) { setTo = next },
		direction: func() Direction { return DirectionRTL },
		fallback:  func() string { return "en" },
	}

	node := Provider(ProviderProps{
		Locale:   state,
		Bundle:   bundle,
		Child:    ui.Text("one"),
		Children: []ui.Node{ui.Text("two")},
	})
	if node == nil {
		t.Fatalf("expected provider node")
	}
	if len(node.Children) != 2 {
		t.Fatalf("expected provider to include child and children, got %d", len(node.Children))
	}
	rawRuntime, ok := node.Props["value"].(Runtime)
	if !ok {
		t.Fatalf("expected runtime value in provider props, got %T", node.Props["value"])
	}
	if got := rawRuntime.Locale(); got != "fr" {
		t.Fatalf("runtime locale = %q, want fr", got)
	}
	rawRuntime.SetLocale("ar")
	if setTo != "ar" {
		t.Fatalf("runtime set locale did not delegate: %q", setTo)
	}
	if got := rawRuntime.Direction(); got != DirectionRTL {
		t.Fatalf("runtime direction = %q, want rtl", got)
	}
	if got := rawRuntime.T("common", "greeting", Arguments{"name": "Cam"}); got != "Bonjour Cam" {
		t.Fatalf("runtime translation = %q, want Bonjour Cam", got)
	}
	if got := rawRuntime.FormatNumber(12.5); got == "" {
		t.Fatalf("expected formatted number")
	}
	if got := rawRuntime.FormatDate(time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC)); got == "" {
		t.Fatalf("expected formatted date")
	}
	if got := rawRuntime.PrefixPath("/pricing", RouteOptions{SupportedLocales: []string{"en", "fr"}, OmitDefaultPrefix: true}); got != "/fr/pricing" {
		t.Fatalf("runtime prefix path = %q, want /fr/pricing", got)
	}

	zero := Runtime{}
	if zero.Locale() != "" {
		t.Fatalf("zero runtime locale should be empty")
	}
	if zero.Direction() != DirectionLTR {
		t.Fatalf("zero runtime direction should default to ltr")
	}
	if got := zero.T("common", "missing"); got != "common.missing" {
		t.Fatalf("zero runtime missing text = %q, want common.missing", got)
	}
}

func TestUseI18nPanicsOnNative(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected UseI18n to panic in non-browser runtime")
		}
	}()
	_ = UseI18n()
}

func TestBundleAndLocaleStateNilSafety(t *testing.T) {
	var state LocaleState
	if got := state.SupportedLocales(); got != nil {
		t.Fatalf("zero locale state supported locales = %v, want nil", got)
	}
	state = LocaleState{supported: func() []string { return []string{"en", "fr"} }}
	if got := state.SupportedLocales(); len(got) != 2 || got[0] != "en" || got[1] != "fr" {
		t.Fatalf("supported locales = %v", got)
	}

	var nilBundle *Bundle
	if got := nilBundle.DefaultLocale(); got != "" {
		t.Fatalf("nil bundle default locale = %q, want empty", got)
	}
	if got := nilBundle.FallbackLocale(); got != "" {
		t.Fatalf("nil bundle fallback locale = %q, want empty", got)
	}
}

func TestFormatDateAdditionalLocaleBranches(t *testing.T) {
	date := time.Date(2026, time.March, 16, 10, 30, 0, 0, time.UTC)

	if got := FormatDate("de-DE", date, DateOptions{Style: DateStyleShort}); got != "16.03.2026" {
		t.Fatalf("expected german short date, got %q", got)
	}
	if got := FormatDate("de-DE", date, DateOptions{Style: DateStyleLong}); got != "16. March 2026" {
		t.Fatalf("expected german long date, got %q", got)
	}
	if got := FormatDate("de-DE", date); got != "16. Mar 2026" {
		t.Fatalf("expected german medium date, got %q", got)
	}
	if got := FormatDate("ar-EG", date, DateOptions{Style: DateStyleLong}); got != "16 Mar 2026" {
		t.Fatalf("expected arabic long date format, got %q", got)
	}
	if got := FormatDate("ar-EG", date); got != "16/03/2026" {
		t.Fatalf("expected arabic default date format, got %q", got)
	}
}

func TestNumericArgumentAndPluralCategoryCoverage(t *testing.T) {
	if numericArgument(int8(2)) != 2 || numericArgument(int16(3)) != 3 || numericArgument(int32(4)) != 4 || numericArgument(int64(5)) != 5 {
		t.Fatalf("expected signed integer conversions to succeed")
	}
	if numericArgument(uint(6)) != 6 || numericArgument(uint8(7)) != 7 || numericArgument(uint16(8)) != 8 || numericArgument(uint32(9)) != 9 || numericArgument(uint64(10)) != 10 {
		t.Fatalf("expected unsigned integer conversions to succeed")
	}
	if numericArgument(float32(11.5)) != 11.5 || numericArgument(float64(12.5)) != 12.5 {
		t.Fatalf("expected float conversions to succeed")
	}
	if numericArgument("13.25") != 13.25 {
		t.Fatalf("expected numeric string parsing to succeed")
	}
	if numericArgument("not-a-number") != 0 {
		t.Fatalf("expected invalid numeric string to return 0")
	}

	if got := pluralCategoryForLocale("ar", 0); got != PluralZero {
		t.Fatalf("arabic plural category for 0 = %q", got)
	}
	if got := pluralCategoryForLocale("ar", 1); got != PluralOne {
		t.Fatalf("arabic plural category for 1 = %q", got)
	}
	if got := pluralCategoryForLocale("ar", 2); got != PluralTwo {
		t.Fatalf("arabic plural category for 2 = %q", got)
	}
	if got := pluralCategoryForLocale("ar", 7); got != PluralFew {
		t.Fatalf("arabic plural category for 7 = %q", got)
	}
	if got := pluralCategoryForLocale("ar", 23); got != PluralMany {
		t.Fatalf("arabic plural category for 23 = %q", got)
	}
	if got := pluralCategoryForLocale("fr", 0); got != PluralOne {
		t.Fatalf("french plural category for 0 = %q", got)
	}
	if got := pluralCategoryForLocale("fr", 2); got != PluralOther {
		t.Fatalf("french plural category for 2 = %q", got)
	}
	if got := pluralCategoryForLocale("ru", 2); got != PluralFew {
		t.Fatalf("russian plural category for 2 = %q", got)
	}
	if got := pluralCategoryForLocale("ru", 5); got != PluralMany {
		t.Fatalf("russian plural category for 5 = %q", got)
	}
	if got := pluralCategoryForLocale("pl", 1); got != PluralOne {
		t.Fatalf("polish plural category for 1 = %q", got)
	}
	if got := pluralCategoryForLocale("pl", 3); got != PluralFew {
		t.Fatalf("polish plural category for 3 = %q", got)
	}
	if got := pluralCategoryForLocale("cs", 4); got != PluralFew {
		t.Fatalf("czech plural category for 4 = %q", got)
	}
	if got := pluralCategoryForLocale("ja", 10); got != PluralOther {
		t.Fatalf("japanese plural category for 10 = %q", got)
	}
	if got := pluralCategoryForLocale("en", 1); got != PluralOne {
		t.Fatalf("english plural category for 1 = %q", got)
	}
	if got := pluralCategoryForLocale("en", 9); got != PluralOther {
		t.Fatalf("english plural category for 9 = %q", got)
	}
}
