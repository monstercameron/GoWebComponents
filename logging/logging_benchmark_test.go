package logging

import "testing"

func BenchmarkCloneFieldsSmall(parseB *testing.B) {
	parseB.ReportAllocs()
	parseFields := Fields{
		"component": "router",
		"code":      "route_blocked",
		"path":      "/app/thread/1",
		"attempt":   3,
	}
	for parseB.Loop() {
		parseCloned := cloneFields(parseFields)
		if len(parseCloned) != len(parseFields) {
			parseB.Fatalf("cloneFields length mismatch: got %d want %d", len(parseCloned), len(parseFields))
		}
	}
}

func BenchmarkLoggerScope(parseB *testing.B) {
	parseB.ReportAllocs()
	parseLogger := New("router")
	for parseB.Loop() {
		if parseLogger.Scope() != "router" {
			parseB.Fatal("Scope mismatch")
		}
	}
}
