//go:build js && wasm
// +build js,wasm

package state

import "testing"

func BenchmarkSnapshotSelect(b *testing.B) {
	b.ReportAllocs()
	snapshot := Snapshot{
		"user:id":    "123",
		"user:name":  "Casey",
		"theme":      "dark",
		"locale":     "en-US",
		"feature:ai": true,
	}
	for b.Loop() {
		selected := snapshot.Select("user:id", "theme", "locale")
		if len(selected) != 3 {
			b.Fatalf("Snapshot.Select returned %d entries, want 3", len(selected))
		}
	}
}

func BenchmarkMarshalUnmarshalSnapshotJSON(b *testing.B) {
	b.ReportAllocs()
	snapshot := Snapshot{
		"user:id":    "123",
		"user:name":  "Casey",
		"theme":      "dark",
		"locale":     "en-US",
		"feature:ai": true,
	}
	for b.Loop() {
		data, err := MarshalSnapshotJSON(snapshot)
		if err != nil {
			b.Fatalf("MarshalSnapshotJSON: %v", err)
		}
		decoded, err := UnmarshalSnapshotJSON(data)
		if err != nil {
			b.Fatalf("UnmarshalSnapshotJSON: %v", err)
		}
		if len(decoded) == 0 {
			b.Fatal("decoded snapshot is empty")
		}
	}
}
