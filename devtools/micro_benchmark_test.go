package devtools

import "testing"

func BenchmarkSnapshotNowMicro(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = SnapshotNow()
	}
}

func BenchmarkCompareSnapshotsMicro(b *testing.B) {
	before := Snapshot{
		Route: Route{Path: "/home"},
		Stats: Stats{TotalFibers: 10},
	}
	after := Snapshot{
		Route: Route{Path: "/dashboard"},
		Stats: Stats{TotalFibers: 12},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := CompareSnapshots(before, after)
		if err != nil {
			b.Fatal(err)
		}
	}
}
