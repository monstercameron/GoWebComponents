package logging

import "testing"

func BenchmarkCloneFieldsSmall(b *testing.B) {
	b.ReportAllocs()
	fields := Fields{
		"component": "router",
		"code":      "route_blocked",
		"path":      "/app/thread/1",
		"attempt":   3,
	}
	for b.Loop() {
		cloned := cloneFields(fields)
		if len(cloned) != len(fields) {
			b.Fatalf("cloneFields length mismatch: got %d want %d", len(cloned), len(fields))
		}
	}
}

func BenchmarkLoggerScope(b *testing.B) {
	b.ReportAllocs()
	logger := New("router")
	for b.Loop() {
		if logger.Scope() != "router" {
			b.Fatal("Scope mismatch")
		}
	}
}
