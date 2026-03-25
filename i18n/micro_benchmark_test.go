package i18n

import "testing"

func BenchmarkNormalizeLocaleMicro(b *testing.B) {
	locales := []string{
		"en-us",
		"EN_us",
		" pt-BR ",
		"zh_hans_cn",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NormalizeLocale(locales[i%len(locales)])
	}
}
