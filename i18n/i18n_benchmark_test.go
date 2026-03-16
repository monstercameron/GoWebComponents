package i18n

import "testing"

func BenchmarkBundleTranslate(b *testing.B) {
	bundle := buildTestBundle()
	args := Arguments{"name": "Cam", "count": 12}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bundle.Translate("fr-CA", "marketing", "cart", args, "en")
	}
}
