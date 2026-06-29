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
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = NormalizeLocale(parseLocales[parseI%len(parseLocales)])
	}
}
