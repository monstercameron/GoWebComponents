package logging

import "testing"

func BenchmarkCloneFieldsMicro(parseB *testing.B) {
	parseFields := Fields{
		"path":   "/orders/123",
		"status": "ok",
		"latency": map[string]interface{}{
			"ms": 12,
		},
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = cloneFields(parseFields)
	}
}
