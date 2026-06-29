package i18n

import "testing"

func BenchmarkBundleTranslate(parseB *testing.B) {
	parseBundle := buildTestBundle()
	parseArgs := Arguments{"name": "Cam", "count": 12}
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseBundle.Translate("fr-CA", "marketing", "cart", parseArgs, "en")
	}
}
