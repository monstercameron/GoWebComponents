package devtools

import "testing"

func BenchmarkSnapshotNowMicro(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = SnapshotNow()
	}
}

func BenchmarkCompareSnapshotsMicro(parseB *testing.B) {
	parseBefore := Snapshot{
		Route: Route{Path: "/home"},
		Stats: Stats{TotalFibers: 10},
	}
	parseAfter := Snapshot{
		Route: Route{Path: "/dashboard"},
		Stats: Stats{TotalFibers: 12},
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_, parseErr := CompareSnapshots(parseBefore, parseAfter)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
	}
}
