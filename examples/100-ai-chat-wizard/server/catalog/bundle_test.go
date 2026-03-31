package catalog

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"github.com/monstercameron/GoWebComponents/i18n"
)

// parseGetLocaleNamespaceKeys returns every key registered directly under locale+namespace
// in the bundle's private catalog map.  It uses reflect+unsafe because i18n.Bundle.catalogs
// is unexported; this is acceptable inside a _test.go file targeting the same package.
func parseGetLocaleNamespaceKeys(b *i18n.Bundle, locale, ns string) map[string]bool {
	rv := reflect.ValueOf(b).Elem()
	cf := rv.FieldByName("catalogs")
	ptr := reflect.NewAt(cf.Type(), unsafe.Pointer(cf.UnsafeAddr())).Elem()
	catalogs, _ := ptr.Interface().(map[string]i18n.Catalog)
	out := map[string]bool{}
	if lc, ok := catalogs[locale]; ok {
		if nc, ok := lc[ns]; ok {
			for k := range nc {
				out[k] = true
			}
		}
	}
	return out
}

// parseTranslateChat translates key from chatI18nNamespace for the given locale with no
// cross-locale fallback bleed.  Passing the same locale as explicit fallback limits the
// candidate chain to that locale only, preventing silent EN bleed on missing keys.
func parseTranslateChat(b *i18n.Bundle, locale, key string) string {
	return b.Translate(locale, chatI18nNamespace, key, nil, locale)
}

// TestBuildBundle asserts that BuildBundle returns a non-nil bundle.
func TestBuildBundle(t *testing.T) {
	b := BuildBundle()
	if b == nil {
		t.Fatal("BuildBundle() returned nil")
	}
}

// TestBuildBundleLocales asserts that exactly three locale codes are registered: en, es, fr.
func TestBuildBundleLocales(t *testing.T) {
	b := BuildBundle()
	got := b.Locales()
	sort.Strings(got)
	want := []string{"en", "es", "fr"}
	if len(got) != len(want) {
		t.Fatalf("Locales() = %v, want %v", got, want)
	}
	for i, l := range want {
		if got[i] != l {
			t.Errorf("Locales()[%d] = %q, want %q", i, got[i], l)
		}
	}
}

// TestBuildBundleChatKeyParity asserts that es and fr carry every chat namespace key
// that en carries; any missing key means a shipped surface is untranslated.
func TestBuildBundleChatKeyParity(t *testing.T) {
	b := BuildBundle()
	enKeys := parseGetLocaleNamespaceKeys(b, "en", chatI18nNamespace)
	if len(enKeys) == 0 {
		t.Fatal("en chat namespace has no keys — BuildBundle may have changed structure")
	}
	for _, locale := range []string{"es", "fr"} {
		localeKeys := parseGetLocaleNamespaceKeys(b, locale, chatI18nNamespace)
		for key := range enKeys {
			if !localeKeys[key] {
				t.Errorf("locale %q is missing chat key %q (present in en)", locale, key)
			}
		}
	}
}

// TestBuildBundleRequiredENKeys asserts that the EN chat namespace contains every key
// needed by the shipped billing, auth, modal, sidebar, and input surfaces.
func TestBuildBundleRequiredENKeys(t *testing.T) {
	b := BuildBundle()
	requiredKeys := []string{
		// footer / public-route entry points
		"auth.logIn", "auth.signUp", "auth.openApp", "auth.backToLogIn",
		// billing modal invoice lines
		"modal.billingTitle", "modal.billingHelp",
		"modal.billingPlatformFee", "modal.billingUsageCost",
		"modal.billingPremiumCost", "modal.billingTotalCost",
		"modal.billingPlanLabel", "modal.billingPlanValue",
		"modal.billingCoverageLabel", "modal.billingNoUsage",
		// settings modal
		"modal.settingsTitle", "modal.save", "modal.displayName",
		// memories surface
		"modal.memories", "modal.memoriesEmpty",
		// security surface
		"modal.securityTitle", "modal.securityLinked", "modal.securityNotLinked",
		// sidebar
		"sidebar.newChat", "sidebar.noConversations", "sidebar.editSettings",
		// empty state
		"empty.heading", "empty.body",
		// input
		"input.placeholder", "input.disclaimer",
	}
	for _, key := range requiredKeys {
		missingText := chatI18nNamespace + "." + key
		got := parseTranslateChat(b, "en", key)
		if got == missingText || got == "" {
			t.Errorf("EN chat key %q is missing or empty (got %q)", key, got)
		}
	}
}

// TestBuildBundlePricingVocabulary asserts that billing invoice-line copy uses the required
// vocabulary: "platform fee", "usage", and "service premium".  These terms must not be
// replaced with provider names, internal cost codes, or unmarked multipliers.
func TestBuildBundlePricingVocabulary(t *testing.T) {
	b := BuildBundle()
	cases := []struct {
		key      string
		contains []string
	}{
		// The billing help string must mention all three invoice-line concepts.
		{"modal.billingHelp", []string{"platform fee", "usage", "premium"}},
		// Each invoice line key must use its mandated term.
		{"modal.billingPlatformFee", []string{"platform fee"}},
		{"modal.billingUsageCost", []string{"usage"}},
		{"modal.billingPremiumCost", []string{"premium"}},
	}
	for _, tc := range cases {
		text := parseTranslateChat(b, "en", tc.key)
		lower := strings.ToLower(text)
		for _, want := range tc.contains {
			if !strings.Contains(lower, strings.ToLower(want)) {
				t.Errorf("EN %q = %q — missing required vocabulary %q", tc.key, text, want)
			}
		}
	}
}

// TestBuildBundleBillingPlanValue asserts that modal.billingPlanValue is "Pro" in every locale.
func TestBuildBundleBillingPlanValue(t *testing.T) {
	b := BuildBundle()
	for _, locale := range []string{"en", "es", "fr"} {
		got := parseTranslateChat(b, locale, "modal.billingPlanValue")
		if got != "Pro" {
			t.Errorf("locale %q modal.billingPlanValue = %q, want \"Pro\"", locale, got)
		}
	}
}

// TestBuildBundleBillingCopyIsProviderAgnostic asserts that billing invoice-line keys do not
// mention specific AI provider names.  Billing copy must remain provider-agnostic so that
// vendor changes do not silently corrupt customer-facing invoices.
func TestBuildBundleBillingCopyIsProviderAgnostic(t *testing.T) {
	b := BuildBundle()
	billingKeys := []string{
		"modal.billingTitle", "modal.billingHelp", "modal.billingNavSummary",
		"modal.billingPlanLabel", "modal.billingPlanValue",
		"modal.billingPlatformFee", "modal.billingUsageCost",
		"modal.billingPremiumCost", "modal.billingTotalCost",
		"modal.billingCoverageLabel", "modal.billingNoUsage",
	}
	// Provider names must not appear in billing copy regardless of case.
	forbiddenProviders := []string{"openai", "anthropic", "google", "deepseek", "mistral"}
	for _, key := range billingKeys {
		text := parseTranslateChat(b, "en", key)
		lower := strings.ToLower(text)
		for _, provider := range forbiddenProviders {
			if strings.Contains(lower, provider) {
				t.Errorf("EN billing key %q mentions provider %q — billing copy must be provider-agnostic: %q", key, provider, text)
			}
		}
	}
}

// TestBuildBundleBillingLocaleSpecificText spot-checks that es and fr provide their own
// translations for the core billing invoice keys rather than silently inheriting EN text.
func TestBuildBundleBillingLocaleSpecificText(t *testing.T) {
	b := BuildBundle()
	// Each entry: key, expected ES text, expected FR text.
	cases := []struct {
		key    string
		wantES string
		wantFR string
	}{
		{"modal.billingPlatformFee", "Tarifa de plataforma", "Frais de plateforme"},
		{"modal.billingUsageCost", "Costo de uso real", "Cout d'usage brut"},
		{"modal.billingPremiumCost", "Margen de servicio", "Prime de service"},
		{"modal.billingTotalCost", "Total", "Total"},
		{"modal.billingPlanLabel", "Plan actual", "Plan actuel"},
	}
	for _, tc := range cases {
		if gotES := parseTranslateChat(b, "es", tc.key); gotES != tc.wantES {
			t.Errorf("es %q = %q, want %q", tc.key, gotES, tc.wantES)
		}
		if gotFR := parseTranslateChat(b, "fr", tc.key); gotFR != tc.wantFR {
			t.Errorf("fr %q = %q, want %q", tc.key, gotFR, tc.wantFR)
		}
	}
}
