package logging

import "testing"

func BenchmarkCloneFieldsMicro(b *testing.B) {
	fields := Fields{
		"path":   "/orders/123",
		"status": "ok",
		"latency": map[string]interface{}{
			"ms": 12,
		},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cloneFields(fields)
	}
}
