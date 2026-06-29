package i18n

import "testing"

// TestLazyBundleLoadsOncePerLocale proves the lazy-locale-loading flow: a locale's catalog is
// fetched + registered on first EnsureLocale, translatable thereafter, and never fetched again.
func TestLazyBundleLoadsOncePerLocale(parseT *testing.T) {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	parseBundle.RegisterNamespace("en", "common", NamespaceCatalog{"greeting": {Text: "Hello"}})

	parseCalls := map[string]int{}
	parseLoad := func(parseLocale string) (Catalog, error) {
		parseCalls[parseLocale]++
		return Catalog{"common": NamespaceCatalog{"greeting": {Text: "Bonjour"}}}, nil
	}
	parseLazy := NewLazyBundle(parseBundle, parseLoad)

	// The pre-registered en locale is treated as loaded and never fetched.
	if !parseLazy.Loaded("en") {
		parseT.Fatal("pre-registered locale must count as loaded")
	}
	if parseLazy.Loaded("fr") {
		parseT.Fatal("fr must not be loaded before EnsureLocale")
	}

	if parseErr := parseLazy.EnsureLocale("fr"); parseErr != nil {
		parseT.Fatalf("EnsureLocale(fr): %v", parseErr)
	}
	if !parseLazy.Loaded("fr") {
		parseT.Fatal("fr must be loaded after EnsureLocale")
	}
	if parseGot := parseBundle.Translate("fr", "common", "greeting", nil, "en"); parseGot != "Bonjour" {
		parseT.Fatalf("expected lazily-loaded fr greeting, got %q", parseGot)
	}

	// Idempotent: a second EnsureLocale does not re-fetch.
	if parseErr := parseLazy.EnsureLocale("fr"); parseErr != nil {
		parseT.Fatalf("second EnsureLocale(fr): %v", parseErr)
	}
	if parseCalls["fr"] != 1 {
		parseT.Fatalf("loader must be called exactly once per locale, got %d for fr", parseCalls["fr"])
	}
}

// TestLazyBundleLoaderErrorLeavesLocaleUnloaded proves a failed load is retryable: the locale is
// not marked loaded, so a later EnsureLocale tries again.
func TestLazyBundleLoaderErrorLeavesLocaleUnloaded(parseT *testing.T) {
	parseBundle := NewBundle(BundleOptions{DefaultLocale: "en"})
	parseFail := true
	parseLoad := func(parseLocale string) (Catalog, error) {
		if parseFail {
			return nil, errLoad
		}
		return Catalog{"common": NamespaceCatalog{"k": {Text: "v"}}}, nil
	}
	parseLazy := NewLazyBundle(parseBundle, parseLoad)

	if parseErr := parseLazy.EnsureLocale("de"); parseErr == nil {
		parseT.Fatal("expected loader error to propagate")
	}
	if parseLazy.Loaded("de") {
		parseT.Fatal("a failed load must not mark the locale loaded")
	}
	parseFail = false
	if parseErr := parseLazy.EnsureLocale("de"); parseErr != nil {
		parseT.Fatalf("retry after fixable error must succeed: %v", parseErr)
	}
	if !parseLazy.Loaded("de") {
		parseT.Fatal("de must be loaded after a successful retry")
	}
}

type lazyLoadError struct{}

func (lazyLoadError) Error() string { return "load failed" }

var errLoad = lazyLoadError{}
