//go:build js && wasm
// +build js,wasm

package state

import "testing"

func BenchmarkSnapshotSelect(parseB *testing.B) {
	parseB.ReportAllocs()
	parseSnapshot := Snapshot{
		"user:id":    "123",
		"user:name":  "Casey",
		"theme":      "dark",
		"locale":     "en-US",
		"feature:ai": true,
	}
	for parseB.Loop() {
		parseSelected := parseSnapshot.Select("user:id", "theme", "locale")
		if len(parseSelected) != 3 {
			parseB.Fatalf("Snapshot.Select returned %d entries, want 3", len(parseSelected))
		}
	}
}

func BenchmarkMarshalUnmarshalSnapshotJSON(parseB *testing.B) {
	parseB.ReportAllocs()
	parseSnapshot := Snapshot{
		"user:id":    "123",
		"user:name":  "Casey",
		"theme":      "dark",
		"locale":     "en-US",
		"feature:ai": true,
	}
	for parseB.Loop() {
		parseData, parseErr := MarshalSnapshotJSON(parseSnapshot)
		if parseErr != nil {
			parseB.Fatalf("MarshalSnapshotJSON: %v", parseErr)
		}
		parseDecoded, parseErr := UnmarshalSnapshotJSON(parseData)
		if parseErr != nil {
			parseB.Fatalf("UnmarshalSnapshotJSON: %v", parseErr)
		}
		if len(parseDecoded) == 0 {
			parseB.Fatal("decoded snapshot is empty")
		}
	}
}
