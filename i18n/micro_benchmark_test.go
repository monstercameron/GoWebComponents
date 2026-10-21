package i18n

import "testing"

func BenchmarkNormalizeLocaleMicro(parseB *testing.B) {
	parseLocales := []string{
		"en-us",
		"EN_us",
		" pt-BR ",
		"zh_hans_cn",
	}

	parseB.ReportAllocs()
	parseI := 0
	for parseB.Loop() {
		_ = NormalizeLocale(parseLocales[parseI%len(parseLocales)])
		parseI++
	}
}
