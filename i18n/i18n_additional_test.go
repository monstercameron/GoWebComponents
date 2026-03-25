package i18n

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestLocaleStateAndBundleWrapperHelpers(t *testing.T) {
	state := LocaleState{}
	if state.Get() != "" || state.Direction() != DirectionLTR || state.FallbackLocale() != "" || state.SupportedLocales() != nil {
		t.Fatalf("zero LocaleState should return safe defaults, got locale=%q direction=%q fallback=%q supported=%v", state.Get(), state.Direction(), state.FallbackLocale(), state.SupportedLocales())
	}

	handle := UseLocale(LocaleOptions{
		InitialLocale:    "fr-CA",
		SupportedLocales: []string{"en", "fr"},
		FallbackLocale:   "en",
	})
	if got := handle.Get(); got != "fr-CA" {
		t.Fatalf("UseLocale().Get() = %q, want fr-CA", got)
	}
	if got := handle.Direction(); got != DirectionLTR {
		t.Fatalf("UseLocale().Direction() = %q, want ltr", got)
	}
	if got := handle.FallbackLocale(); got != "en" {
		t.Fatalf("UseLocale().FallbackLocale() = %q, want en", got)
	}
	handle.Set("unknown")
	if got := handle.Get(); got != "en" {
		t.Fatalf("UseLocale().Set(unknown) should fall back to en, got %q", got)
	}
	if got := firstLocale([]string{"fr", "en"}); got != "fr" {
		t.Fatalf("firstLocale() = %q, want fr", got)
	}
	if got := firstLocale(nil); got != "" {
		t.Fatalf("firstLocale(nil) = %q, want empty", got)
	}

	bundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "fr"})
	bundle.RegisterNamespace("en", "common", NamespaceCatalog{"greeting": {Text: "Hello"}})
	bundle.RegisterNamespace("fr", "common", NamespaceCatalog{"greeting": {Text: "Bonjour"}})
	if got := bundle.DefaultLocale(); got != "en" {
		t.Fatalf("DefaultLocale() = %q, want en", got)
	}
	if got := bundle.FallbackLocale(); got != "fr" {
		t.Fatalf("FallbackLocale() = %q, want fr", got)
	}
	locales := bundle.Locales()
	if len(locales) != 2 || locales[0] != "en" || locales[1] != "fr" {
		t.Fatalf("Locales() = %v, want [en fr]", locales)
	}
	if translated := bundle.Translate("fr", "common", "greeting", nil, "en"); translated != "Bonjour" {
		t.Fatalf("Translate() = %q, want Bonjour", translated)
	}
	if got := defaultMissingText("fr", "common", "headline"); got != "common.headline" {
		t.Fatalf("defaultMissingText() = %q, want common.headline", got)
	}
}

func TestNilBundleLocalesProviderDefaultsAndNativeUseLocaleFallback(t *testing.T) {
	var nilBundle *Bundle
	if got := nilBundle.Locales(); got != nil {
		t.Fatalf("nil bundle Locales() = %v, want nil", got)
	}

	bundle := NewBundle()
	bundle.RegisterNamespace("ar", "common", NamespaceCatalog{
		"greeting": {Text: "مرحبا"},
	})
	node := Provider(ProviderProps{
		Bundle:   bundle,
		Children: []ui.Node{ui.Text("child")},
	})
	if node == nil {
		t.Fatal("expected provider node")
	}
	rawRuntime, ok := node.Props["value"].(Runtime)
	if !ok {
		t.Fatalf("expected runtime value in provider props, got %T", node.Props["value"])
	}
	if got := rawRuntime.Locale(); got != "ar" {
		t.Fatalf("runtime locale = %q, want ar from bundle default", got)
	}
	rawRuntime.SetLocale("fr")
	if got := rawRuntime.Direction(); got != DirectionRTL {
		t.Fatalf("runtime direction = %q, want rtl from current locale fallback", got)
	}
	if got := rawRuntime.T("common", "greeting"); got != "مرحبا" {
		t.Fatalf("runtime translation = %q, want Arabic greeting", got)
	}

	handle := UseLocale(LocaleOptions{
		InitialLocale:    "",
		SupportedLocales: []string{"fr", "en"},
		FallbackLocale:   "fr",
	})
	if got := handle.Get(); got != "fr" {
		t.Fatalf("UseLocale() fallback current = %q, want fr", got)
	}
	supported := handle.SupportedLocales()
	supported[0] = "mutated"
	if got := handle.SupportedLocales()[0]; got != "fr" {
		t.Fatalf("expected SupportedLocales() to return a clone, got %v", handle.SupportedLocales())
	}
}

func TestTranslateMissingDefaultLookupMissAndEmptyNativeLocaleConfig(t *testing.T) {
	bundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	bundle.RegisterNamespace("en", "common", NamespaceCatalog{
		"greeting": {Text: "Hello"},
	})

	if got := bundle.Translate("en", "common", "missing", nil, "en"); got != "common.missing" {
		t.Fatalf("Translate() missing default = %q, want common.missing", got)
	}
	if _, ok := bundle.lookup("en", "missing", "greeting", "en"); ok {
		t.Fatal("lookup() should report missing namespace as not found")
	}

	node := Provider(ProviderProps{
		FallbackLocale: "fr",
		Children:       []ui.Node{ui.Text("child")},
	})
	rawRuntime, ok := node.Props["value"].(Runtime)
	if !ok {
		t.Fatalf("expected runtime value in provider props, got %T", node.Props["value"])
	}
	if got := rawRuntime.Locale(); got != "" {
		t.Fatalf("runtime locale = %q, want empty when bundle and locale handle provide none", got)
	}
	rawRuntime.SetLocale("fr")
	if got := rawRuntime.PrefixPath("/pricing"); got != "/fr/pricing" {
		t.Fatalf("runtime PrefixPath() = %q, want /fr/pricing from explicit fallback locale", got)
	}

	emptyHandle := UseLocale(LocaleOptions{})
	if got := emptyHandle.Get(); got != "" {
		t.Fatalf("UseLocale() empty config current = %q, want empty", got)
	}
	if got := emptyHandle.FallbackLocale(); got != "" {
		t.Fatalf("UseLocale() empty config fallback = %q, want empty", got)
	}
	if got := emptyHandle.Direction(); got != DirectionLTR {
		t.Fatalf("UseLocale() empty config direction = %q, want ltr", got)
	}
}
