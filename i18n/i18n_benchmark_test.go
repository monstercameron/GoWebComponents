package i18n

import "testing"

func BenchmarkBundleTranslate(parseB *testing.B) {
	parseBundle := buildTestBundle()
	parseArgs := Arguments{"name": "Cam", "count": 12}
	for parseB.Loop() {
		_ = parseBundle.Translate("fr-CA", "marketing", "cart", parseArgs, "en")
	}
}
