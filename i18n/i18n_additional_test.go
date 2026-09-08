//go:build !js || !wasm

package i18n

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestNamespaceHandleBindsNamespace proves Runtime.NS(ns) binds the namespace so t.T(key) equals
// Runtime.T(ns, key), removing the forgot-the-namespace footgun of the positional form.
func TestNamespaceHandleBindsNamespace(parseT *testing.T) {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	parseBundle.RegisterNamespace("en", "checkout", NamespaceCatalog{
		"title": {Text: "Checkout"},
		"total": {Text: "Total: {amount}"},
	})
	parseNode := Provider(ProviderProps{Bundle: parseBundle, Children: []ui.Node{ui.Text("c")}})
	parseRuntime, parseOk := parseNode.Props["value"].(Runtime)
	if !parseOk {
		parseT.Fatalf("expected runtime in provider props, got %T", parseNode.Props["value"])
	}

	parseNS := parseRuntime.NS("checkout")
	if parseNS.Name() != "checkout" {
		parseT.Fatalf("Name() = %q, want checkout", parseNS.Name())
	}
	if parseGot, parseWant := parseNS.T("title"), parseRuntime.T("checkout", "title"); parseGot != parseWant || parseGot != "Checkout" {
		parseT.Fatalf("NS.T(title) = %q, want %q (== %q)", parseGot, "Checkout", parseWant)
	}
	if parseGot := parseNS.T("total", Arguments{"amount": "$9"}); parseGot != "Total: $9" {
		parseT.Fatalf("NS.T(total,args) = %q, want \"Total: $9\"", parseGot)
	}
}

func TestLocaleStateAndBundleWrapperHelpers(parseT *testing.T) {
	parseState := LocaleState{}
	if parseState.Get() != "" || parseState.Direction() != DirectionLTR || parseState.FallbackLocale() != "" || parseState.SupportedLocales() != nil {
		parseT.Fatalf("zero LocaleState should return safe defaults, got locale=%q direction=%q fallback=%q supported=%v", parseState.Get(), parseState.Direction(), parseState.FallbackLocale(), parseState.SupportedLocales())
	}

	handle := UseLocale(LocaleOptions{
		InitialLocale:    "fr-CA",
		SupportedLocales: []string{"en", "fr"},
		FallbackLocale:   "en",
	})
	if parseGot := handle.Get(); parseGot != "fr-CA" {
		parseT.Fatalf("UseLocale().Get() = %q, want fr-CA", parseGot)
	}
	if parseGot2 := handle.Direction(); parseGot2 != DirectionLTR {
		parseT.Fatalf("UseLocale().Direction() = %q, want ltr", parseGot2)
	}
	if parseGot3 := handle.FallbackLocale(); parseGot3 != "en" {
		parseT.Fatalf("UseLocale().FallbackLocale() = %q, want en", parseGot3)
	}
	handle.Set("unknown")
	if parseGot4 := handle.Get(); parseGot4 != "en" {
		parseT.Fatalf("UseLocale().Set(unknown) should fall back to en, got %q", parseGot4)
	}
	if parseGot5 := firstLocale([]string{"fr", "en"}); parseGot5 != "fr" {
		parseT.Fatalf("firstLocale() = %q, want fr", parseGot5)
	}
	if parseGot6 := firstLocale(nil); parseGot6 != "" {
		parseT.Fatalf("firstLocale(nil) = %q, want empty", parseGot6)
	}

	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "fr"})
	parseBundle.RegisterNamespace("en", "common", NamespaceCatalog{"greeting": {Text: "Hello"}})
	parseBundle.RegisterNamespace("fr", "common", NamespaceCatalog{"greeting": {Text: "Bonjour"}})
	if parseGot7 := parseBundle.DefaultLocale(); parseGot7 != "en" {
		parseT.Fatalf("DefaultLocale() = %q, want en", parseGot7)
	}
	if parseGot8 := parseBundle.FallbackLocale(); parseGot8 != "fr" {
		parseT.Fatalf("FallbackLocale() = %q, want fr", parseGot8)
	}
	parseLocales := parseBundle.Locales()
	if len(parseLocales) != 2 || parseLocales[0] != "en" || parseLocales[1] != "fr" {
		parseT.Fatalf("Locales() = %v, want [en fr]", parseLocales)
	}
	if parseTranslated := parseBundle.Translate("fr", "common", "greeting", nil, "en"); parseTranslated != "Bonjour" {
		parseT.Fatalf("Translate() = %q, want Bonjour", parseTranslated)
	}
	if parseGot9 := defaultMissingText("fr", "common", "headline"); parseGot9 != "common.headline" {
		parseT.Fatalf("defaultMissingText() = %q, want common.headline", parseGot9)
	}
}

func TestNilBundleLocalesProviderDefaultsAndNativeUseLocaleFallback(parseT *testing.T) {
	var parseNilBundle *Bundle
	if parseGot := parseNilBundle.Locales(); parseGot != nil {
		parseT.Fatalf("nil bundle Locales() = %v, want nil", parseGot)
	}

	parseBundle := NewBundle()
	parseBundle.RegisterNamespace("ar", "common", NamespaceCatalog{
		"greeting": {Text: "مرحبا"},
	})
	parseNode := Provider(ProviderProps{
		Bundle:   parseBundle,
		Children: []ui.Node{ui.Text("child")},
	})
	if parseNode == nil {
		parseT.Fatal("expected provider node")
	}
	parseRawRuntime, parseOk := parseNode.Props["value"].(Runtime)
	if !parseOk {
		parseT.Fatalf("expected runtime value in provider props, got %T", parseNode.Props["value"])
	}
	if parseGot2 := parseRawRuntime.Locale(); parseGot2 != "ar" {
		parseT.Fatalf("runtime locale = %q, want ar from bundle default", parseGot2)
	}
	parseRawRuntime.SetLocale("fr")
	if parseGot3 := parseRawRuntime.Direction(); parseGot3 != DirectionRTL {
		parseT.Fatalf("runtime direction = %q, want rtl from current locale fallback", parseGot3)
	}
	if parseGot4 := parseRawRuntime.T("common", "greeting"); parseGot4 != "مرحبا" {
		parseT.Fatalf("runtime translation = %q, want Arabic greeting", parseGot4)
	}

	handle := UseLocale(LocaleOptions{
		InitialLocale:    "",
		SupportedLocales: []string{"fr", "en"},
		FallbackLocale:   "fr",
	})
	if parseGot5 := handle.Get(); parseGot5 != "fr" {
		parseT.Fatalf("UseLocale() fallback current = %q, want fr", parseGot5)
	}
	parseSupported := handle.SupportedLocales()
	parseSupported[0] = "mutated"
	if parseGot6 := handle.SupportedLocales()[0]; parseGot6 != "fr" {
		parseT.Fatalf("expected SupportedLocales() to return a clone, got %v", handle.SupportedLocales())
	}
}

func TestTranslateMissingDefaultLookupMissAndEmptyNativeLocaleConfig(parseT *testing.T) {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	parseBundle.RegisterNamespace("en", "common", NamespaceCatalog{
		"greeting": {Text: "Hello"},
	})

	if parseGot := parseBundle.Translate("en", "common", "missing", nil, "en"); parseGot != "common.missing" {
		parseT.Fatalf("Translate() missing default = %q, want common.missing", parseGot)
	}
	if _, parseOk := parseBundle.lookup("en", "missing", "greeting", "en"); parseOk {
		parseT.Fatal("lookup() should report missing namespace as not found")
	}

	parseNode := Provider(ProviderProps{
		FallbackLocale: "fr",
		Children:       []ui.Node{ui.Text("child")},
	})
	parseRawRuntime, parseOk2 := parseNode.Props["value"].(Runtime)
	if !parseOk2 {
		parseT.Fatalf("expected runtime value in provider props, got %T", parseNode.Props["value"])
	}
	if parseGot2 := parseRawRuntime.Locale(); parseGot2 != "" {
		parseT.Fatalf("runtime locale = %q, want empty when bundle and locale handle provide none", parseGot2)
	}
	parseRawRuntime.SetLocale("fr")
	if parseGot3 := parseRawRuntime.PrefixPath("/pricing"); parseGot3 != "/fr/pricing" {
		parseT.Fatalf("runtime PrefixPath() = %q, want /fr/pricing from explicit fallback locale", parseGot3)
	}

	parseEmptyHandle := UseLocale(LocaleOptions{})
	if parseGot4 := parseEmptyHandle.Get(); parseGot4 != "" {
		parseT.Fatalf("UseLocale() empty config current = %q, want empty", parseGot4)
	}
	if parseGot5 := parseEmptyHandle.FallbackLocale(); parseGot5 != "" {
		parseT.Fatalf("UseLocale() empty config fallback = %q, want empty", parseGot5)
	}
	if parseGot6 := parseEmptyHandle.Direction(); parseGot6 != DirectionLTR {
		parseT.Fatalf("UseLocale() empty config direction = %q, want ltr", parseGot6)
	}
}
